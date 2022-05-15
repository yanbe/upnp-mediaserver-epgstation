//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --package=epgstation --generate types,client -o client.gen.go http://$EPGSTATION/api/docs
// NOTE: To to re-generate EPGStation client code, you have to pass EPGStation's IP:Port as $EPGSTATION environment variable
// $ EPGSTATION=192.168.10.10:8888 go generate
package epgstation

import (
	"fmt"
	"log"
	"net"
)

var EPGStation *ClientWithResponses
var ServerAPIRoot string

func Setup(epgstationAddr net.TCPAddr) {
	var err error
	ServerAPIRoot = fmt.Sprintf("http://%s:%d/api", epgstationAddr.IP, epgstationAddr.Port)
	EPGStation, err = NewClientWithResponses(ServerAPIRoot)
	if err != nil {
		log.Fatalf("epgstation client init error: %s", err)
	}
}
