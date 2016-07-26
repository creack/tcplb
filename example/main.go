package main

import (
	"log"

	"github.com/creack/tcplb"
)

func main() {
	lb := &tcplb.Server{
		Laddr: "0.0.0.0:7001",
		Targets: tcplb.Targets{
			0: {
				Host: "192.168.99.100",
				Port: 8001,
			},
			1: {
				Host: "192.168.99.100",
				Port: 8002,
			},
		},
		LBMode: tcplb.LBRoundRobin,
	}
	if err := lb.Run(2); err != nil {
		log.Fatalf("Error starting the laod balancer: %s", err)
	}
	defer func() { _ = lb.Close() }()

	<-make(chan struct{})
}
