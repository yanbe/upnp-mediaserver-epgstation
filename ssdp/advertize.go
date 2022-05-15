package ssdp

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

const (
	methodNotify = "NOTIFY"
	ssdpUDP4Addr = "239.255.255.250:1900"
	ntsAlive     = `ssdp:alive`
	ntsByebye    = `ssdp:byebye`
	ntsUpdate    = `ssdp:update`
	serverName   = "Linux/i686 UPnP/1.0 go-upnp-playground/0.0.1"
	maxAge       = 1800
)

type SSDPAdvertiser struct {
	deviceUUID uuid.UUID
	urlBase    string
}

func (s *SSDPAdvertiser) RoundTrip(req *http.Request) (*http.Response, error) {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	req.Write(&buf)
	destAddr, err := net.ResolveUDPAddr("udp", req.Host)
	if err != nil {
		return nil, err
	}
	conn.WriteTo(buf.Bytes(), destAddr)
	return &http.Response{}, nil
}

func NewSSDPAdvertiser(deviceUUID uuid.UUID, urlBase string) SSDPAdvertiser {
	return SSDPAdvertiser{
		deviceUUID: deviceUUID,
		urlBase:    urlBase,
	}
}

func (s *SSDPAdvertiser) ntAndUSN(target string) (string, string) {
	var NT string
	var USN string
	switch target {
	case "":
		NT = fmt.Sprintf("uuid:%s", s.deviceUUID)
		USN = NT
	default:
		NT = target
		USN = fmt.Sprintf("uuid:%s::%s", s.deviceUUID, NT)
	}
	return NT, USN
}

func (s *SSDPAdvertiser) notifyTarget(target string) {
	NT, USN := s.ntAndUSN(target)
	req := http.Request{
		Method: methodNotify,
		// TODO: Support both IPv4 and IPv6.
		Host: ssdpUDP4Addr,
		URL:  &url.URL{Opaque: "*"},
		Header: http.Header{
			// Putting headers in here avoids them being title-cased.
			// (The UPnP discovery protocol uses case-sensitive headers)
			"Cache-Control": {fmt.Sprintf("max-age=%d", maxAge)},
			"Location":      {s.urlBase},
			"Server":        {serverName},
			"NT":            {NT},
			"NTS":           {ntsAlive},
			"USN":           {USN},
		},
	}
	client := http.Client{Transport: &UDPRoundTripper{}}
	client.Do(&req)
}

func (s *SSDPAdvertiser) NotifyAlive() {
	for i := 0; i < 2; i++ {
		s.notifyTarget("")
		s.notifyTarget(upnpMediaServer)
		s.notifyTarget(upnpContentDirectory)
		s.notifyTarget(upnpConnectionManager)
		s.notifyTarget(upnpRootDevice)
	}
}

func (s *SSDPAdvertiser) notifyByebye(target string) {
	NT, USN := s.ntAndUSN(target)
	req := http.Request{
		Method: methodNotify,
		// TODO: Support both IPv4 and IPv6.
		Host: ssdpUDP4Addr,
		URL:  &url.URL{Opaque: "*"},
		Header: http.Header{
			// Putting headers in here avoids them being title-cased.
			// (The UPnP discovery protocol uses case-sensitive headers)
			"NT":  []string{NT},
			"NTS": []string{ntsByebye},
			"USN": []string{USN},
		},
	}
	client := http.Client{Transport: s}
	client.Do(&req)
}

func (s *SSDPAdvertiser) NotifyByebye() {
	for i := 0; i < 2; i++ {
		s.notifyByebye("")
		s.notifyByebye(upnpMediaServer)
		s.notifyByebye(upnpContentDirectory)
		s.notifyByebye(upnpConnectionManager)
		s.notifyByebye(upnpRootDevice)
	}
}

func (s *SSDPAdvertiser) Serve() error {
	// Devices should wait a random interval less than 100 milliseconds before sending an initial set of advertisements in order to
	// reduce the likelihood of network storms
	waitRandomMillis(100)
	for {
		s.NotifyAlive()
		waitRandomMillis((maxAge / 2) * 1000)
	}
}
