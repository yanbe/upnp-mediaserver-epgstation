package ssdp

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	// upnpRootDevice is a value for searchTarget that searches for all root devices.
	upnpRootDevice        = "upnp:rootdevice"
	upnpMediaServer       = "urn:schemas-upnp-org:device:MediaServer:1"
	upnpContentDirectory  = "urn:schemas-upnp-org:service:ContentDirectory:1"
	upnpConnectionManager = "urn:schemas-upnp-org:service:ConnectionManager:1"
	vendor                = "Linux/i686 UPnP/1.0 go-upnp-playground/0.0.1"
)

func NewSSDPDiscoveryResponder(deviceUUID uuid.UUID, urlBase string) SSDPDiscoveryResponder {
	return SSDPDiscoveryResponder{
		Multicast:  true,
		deviceUUID: deviceUUID,
		urlBase:    urlBase,
	}
}

func (s *SSDPDiscoveryResponder) stAndUSN(target string) (ST string, USN string, err error) {
	deviceTarget := fmt.Sprintf("uuid:%s", s.deviceUUID)
	switch target {
	case deviceTarget:
		ST = deviceTarget
		USN = ST
	case upnpRootDevice:
		fallthrough
	case upnpContentDirectory:
		fallthrough
	case upnpConnectionManager:
		ST = target
		USN = fmt.Sprintf("%s::%s", deviceTarget, ST)
	default:
		err = errors.New(fmt.Sprint("unsupported search target: ", target))
	}
	return
}

func waitRandomMillis(mx int64) {
	rand.Seed(time.Now().UnixNano())
	randSleepMilliSeconds := rand.Intn(int(mx))
	time.Sleep(time.Duration(randSleepMilliSeconds) * time.Millisecond)
}

func (srv *SSDPDiscoveryResponder) ServeMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "M-SEARCH" {
		return
	}
	ST, USN, err := srv.stAndUSN(r.Header.Get("ST"))
	if err != nil {
		return
	}
	mxHeader := r.Header.Get("MX")
	mx, err := strconv.ParseInt(mxHeader, 10, 8)
	if err != nil {
		return
	}
	if mx > 120 {
		mx = 120
	}
	waitRandomMillis(mx * 1000)
	h := w.Header()
	h.Set("Cache-Control", "max-age=1800")
	h.Set("Location", srv.urlBase)
	h.Set("Server", vendor)
	h.Set("EXT", "")
	h.Set("USN", USN)
	h.Set("ST", ST)
}
