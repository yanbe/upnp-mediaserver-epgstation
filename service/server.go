package service

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"upnp-mediaserver/bufferpool"
	"upnp-mediaserver/epgstation"
	"upnp-mediaserver/service/contentdirectory"
	"upnp-mediaserver/soap"

	"github.com/google/uuid"
)

var URLBase string

func serveXMLFileHandler(tmplFile string, vars map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		if len(vars) == 0 {
			r, err := os.Open(tmplFile)
			if err != nil {
				log.Fatal("error on open file: ", err)
			}
			fi, err := r.Stat()
			if err != nil {
				log.Fatal("error on stat file: ", err)
			}
			w.Header().Set("Content-Length", strconv.Itoa(int(fi.Size())))
			io.Copy(w, r)
		} else {
			buf := bufferpool.NewBytesBuffer()
			defer bufferpool.PutBytesBuffer(buf)
			template.Must(template.ParseFiles(tmplFile)).Execute(buf, vars)
			w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
			w.Write(buf.Bytes())
		}
	}
}

func serviceContentDirectoryControlHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", `text/xml; charset="utf-8"`)
	w.Header().Set("Server", "Linux/i686 UPnP/1.0 go-upnp-playground/0.0.1")
	buf := bufferpool.NewBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	buf.WriteString(xml.Header)
	buf.Write(soap.HandleAction(r))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}

func parseTimeSeekHeader(header string) (time.Duration, string) {
	var h, m, s, ms int64
	fmt.Sscanf(header, "npt=%d:%02d:%02d.%03d-", &h, &m, &s, &ms)
	formatted := fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
	duration := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second + time.Duration(ms)*time.Millisecond
	return duration, formatted
}

func recordedVideoStreamHandler(w http.ResponseWriter, r *http.Request) {
	videoFileId := r.URL.Query().Get("videoFileId")
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/videos/%s", epgstation.ServerAPIRoot, videoFileId), nil)
	if err != nil {
		log.Fatal(err)
	}
	for k, vs := range r.Header {
		req.Header.Set(k, vs[0])
	}
	timeSeekReqHeader := r.Header.Get("Timeseekrange.dlna.org")
	if timeSeekReqHeader != "" {
		startDuration, startStr := parseTimeSeekHeader(timeSeekReqHeader)
		resource := contentdirectory.GetObject(videoFileId).(*contentdirectory.Res)
		elapsedRatio := float64(startDuration) / float64(resource.DurationNS)
		startByte := int(elapsedRatio * float64(resource.Size))
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, resource.Size-1))
		w.Header().Set("Timeseekrange.dlna.org", fmt.Sprintf("npt=%s-%s/%s", startStr, resource.Duration, resource.Duration))
	}
	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	for k, vs := range res.Header {
		if k == "Content-Type" && vs[0] == "video/mp2t" {
			vs[0] = "video/mpeg"
		}
		w.Header().Set(k, vs[0])
	}
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
}

// A Server defines parameters for running an HTTPU server.
type Server struct {
	deviceUUID uuid.UUID
	hostIP     net.IP
	listener   *net.TCPListener
}

func (s *Server) Listen() {
	var err error
	s.listener, err = net.ListenTCP("tcp", &net.TCPAddr{
		IP:   s.hostIP,
		Port: 0,
	}) // start listen arbitorary port
	if err != nil {
		log.Fatal(err)
	}
	listenAddr := s.listener.Addr().(*net.TCPAddr)
	URLBase = fmt.Sprintf("http://%s:%d/", listenAddr.IP, listenAddr.Port)
}

func (s *Server) Setup() {
	epgstation.Setup(net.TCPAddr{
		IP:   s.hostIP,
		Port: 8888,
	})
	contentdirectory.Setup(URLBase)

	http.HandleFunc("/", serveXMLFileHandler("tmpl/device.xml", map[string]interface{}{
		"uuid":    s.deviceUUID,
		"URLBase": URLBase,
	}))
	http.HandleFunc("/ContentDirectory/scpd.xml", serveXMLFileHandler("file/ContentDirectory1.xml", nil))
	http.HandleFunc("/ConnectionManager/scpd.xml", serveXMLFileHandler("file/ConnectionManager1.xml", nil))

	http.HandleFunc("/ContentDirectory/control.xml", serviceContentDirectoryControlHandler)
	http.HandleFunc("/ConnectionManager/control.xml", serviceContentDirectoryControlHandler)

	http.HandleFunc("/videos/recorded", recordedVideoStreamHandler)
}

func (s *Server) Serve() error {
	return http.Serve(s.listener, nil)
}

func NewServer(deviceUUID uuid.UUID, hostIP net.IP) *Server {
	return &Server{
		deviceUUID: deviceUUID,
		hostIP:     hostIP,
		listener:   nil,
	}
}
