package main

import (
	"errors"
	"fmt"
	"upnp-mediaserver/service"
	"upnp-mediaserver/ssdp"
	"log"
	"net"

	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
)

func localIP() (net.IP, error) {
	ifAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}
	for _, ifAddr := range ifAddrs {
		netIP, ok := ifAddr.(*net.IPNet)
		if ok && !netIP.IP.IsLoopback() && netIP.IP.To4() != nil {
			return netIP.IP, nil
		}
	}
	return nil, errors.New("could not get local IP address")
}

func main() {
	deviceUUID := uuid.New()
	localIP, err := localIP()
	server := service.NewServer(deviceUUID, localIP)
	if err != nil {
		log.Fatal(err)
	}

	server.Listen()
	log.Println("Listening: ", service.URLBase)
	server.Setup()

	errSrv := make(chan error)
	go func() {
		errSrv <- server.Serve()
	}()

	ssdpadv := ssdp.NewSSDPAdvertiser(deviceUUID, service.URLBase)
	ssdpres := ssdp.NewSSDPDiscoveryResponder(deviceUUID, service.URLBase)

	errSsdpRes := make(chan error)
	errSsdpAdvRes := make(chan error)

	go func() {
		errSsdpRes <- ssdpres.ListenAndServe()
	}()
	go func() {
		errSsdpAdvRes <- ssdpadv.Serve()
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		ssdpadv.NotifyByebye()
		os.Exit(1)
	}()

	msgSrv := <-errSrv
	fmt.Println(msgSrv)
	msgSsdpRes := <-errSsdpRes
	fmt.Println(msgSsdpRes)
	msgSsdpAdvRes := <-errSsdpAdvRes
	fmt.Println(msgSsdpAdvRes)
}
