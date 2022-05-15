package ssdp

import (
	"bufio"
	"bytes"
	"fmt"
	"go-upnp-playground/bufferpool"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var (
	crlf = []byte("\r\n")
)

// Handler is the interface by which received HTTPU messages are passed to
// handling code.
type Handler interface {
	// ServeMessage is called for each HTTPU message received. peerAddr contains
	// the address that the message was received from.
	ServeMessage(w http.ResponseWriter, r *http.Request)
}

// A Server defines parameters for running an HTTPU server.
type SSDPDiscoveryResponder struct {
	urlBase    string         // TCP address to listen on
	Multicast  bool           // Should listen for multicast?
	Interface  *net.Interface // Network interface to listen on for multicast, nil for default multicast interface
	Handler    Handler        // handler to invoke
	deviceUUID uuid.UUID
}

// ListenAndServe listens on the UDP network address srv.Addr. If srv.Multicast
// is true, then a multicast UDP listener will be used on srv.Interface (or
// default interface if nil).
func (s *SSDPDiscoveryResponder) ListenAndServe() error {
	var err error

	var listenAddr *net.UDPAddr
	if listenAddr, err = net.ResolveUDPAddr("udp", ssdpUDP4Addr); err != nil {
		log.Fatal(err)
	}

	var conn net.PacketConn
	if s.Multicast {
		if conn, err = net.ListenMulticastUDP("udp", s.Interface, listenAddr); err != nil {
			return err
		}
	} else {
		if conn, err = net.ListenUDP("udp", listenAddr); err != nil {
			return err
		}
	}

	return s.Serve(conn)
}

type UDPResponseWriter struct {
	conn         net.PacketConn
	addr         net.Addr
	req          *http.Request
	res          *response
	header       *http.Header
	calledHeader bool
	wroteHeader  bool
	status       int
	statusBuf    [3]byte
	dateBuf      [len(http.TimeFormat)]byte
	bufw         *bufio.Writer
}

type response struct {
	rw *UDPResponseWriter
}

func (r *response) Write(data []byte) (int, error) {
	return r.rw.conn.WriteTo(data, r.rw.addr)
}

func (w *UDPResponseWriter) Header() http.Header {
	if !w.calledHeader {
		w.header = &http.Header{}
	}
	w.calledHeader = true
	return *w.header
}

// TODO: http.Server compatible implementation
func (w *UDPResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.bufw.Write(body)
}

func checkWriteHeaderCode(code int) {
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", code))
	}
}

func writeStatusLine(bw *bufio.Writer, is11 bool, code int, scratch []byte) {
	if is11 {
		bw.WriteString("HTTP/1.1 ")
	} else {
		bw.WriteString("HTTP/1.0 ")
	}
	text := http.StatusText(code)
	if text != "" {
		bw.Write(strconv.AppendInt(scratch[:0], int64(code), 10))
		bw.WriteByte(' ')
		bw.WriteString(text)
		bw.WriteString("\r\n")
	} else {
		// don't worry about performance
		fmt.Fprintf(bw, "%03d status code %d\r\n", code, code)
	}
}

// appendTime is a non-allocating version of []byte(t.UTC().Format(TimeFormat))
func appendTime(b []byte, t time.Time) []byte {
	const days = "SunMonTueWedThuFriSat"
	const months = "JanFebMarAprMayJunJulAugSepOctNovDec"

	t = t.UTC()
	yy, mm, dd := t.Date()
	hh, mn, ss := t.Clock()
	day := days[3*t.Weekday():]
	mon := months[3*(mm-1):]

	return append(b,
		day[0], day[1], day[2], ',', ' ',
		byte('0'+dd/10), byte('0'+dd%10), ' ',
		mon[0], mon[1], mon[2], ' ',
		byte('0'+yy/1000), byte('0'+(yy/100)%10), byte('0'+(yy/10)%10), byte('0'+yy%10), ' ',
		byte('0'+hh/10), byte('0'+hh%10), ':',
		byte('0'+mn/10), byte('0'+mn%10), ':',
		byte('0'+ss/10), byte('0'+ss%10), ' ',
		'G', 'M', 'T')
}

func (w *UDPResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		log.Default().Fatal("httpu: superfluous response.WriteHeader")
	}
	if !w.calledHeader {
		w.Header() // instantiate
	}
	checkWriteHeaderCode(code)
	w.wroteHeader = true
	w.status = code
	writeStatusLine(w.bufw, w.req.ProtoAtLeast(1, 1), code, w.statusBuf[:])
	if w.Header().Get("Date") != "" {
		w.Header().Set("Date", string(appendTime(w.dateBuf[:0], time.Now())))
	}
	w.header.WriteSubset(w.bufw, nil)
	w.bufw.Write(crlf)
}

func (w *UDPResponseWriter) finishRequest() {
	defer bufferpool.PutBufioWriter(w.bufw)
	if !w.wroteHeader {
		if !w.calledHeader {
			return // unsupported or invalid request
		}
		w.WriteHeader(http.StatusOK)
	}
	w.bufw.Flush()
}

// Serve messages received on the given packet listener to the srv.Handler.
func (s *SSDPDiscoveryResponder) Serve(l net.PacketConn) error {
	for {
		buf := bufferpool.NewBytesBuf()
		n, addr, err := l.ReadFrom(buf)
		if err != nil {
			return err
		}

		go func(buf []byte, n int, addr net.Addr) {
			r := io.LimitReader(bytes.NewReader(buf), int64(n))
			br := bufferpool.NewBufioReader(r)
			req, err := http.ReadRequest(br)
			bufferpool.PutBytesBuf(buf)
			bufferpool.PutBufioReader(br)
			if err != nil {
				log.Printf("httpu: Failed to parse request: %v", err)
				return
			}
			req.RemoteAddr = addr.String()
			rw := &UDPResponseWriter{
				conn: l,
				addr: addr,
				req:  req,
			}
			rw.res = &response{
				rw,
			}
			rw.bufw = bufferpool.NewBufioWriterSize(rw.res, 2<<10)
			s.ServeMessage(rw, req)
			rw.finishRequest()
		}(buf, n, addr)
	}
}

type UDPRoundTripper struct {
}

func (t *UDPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}
	buf := bufferpool.NewBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	err = req.Write(buf)
	if err != nil {
		return nil, err
	}
	destAddr, err := net.ResolveUDPAddr("udp", req.Host)
	if err != nil {
		return nil, err
	}
	conn.WriteTo(buf.Bytes(), destAddr)
	return nil, nil
}
