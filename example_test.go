package tcplb_test

import (
	"log"

	"github.com/creack/tcplb"
)

func ExampleServer() {
	if err := (&tcplb.Server{
		Laddr: "0.0.0.0:7000",
		Targets: tcplb.Targets{
			{
				Host: "127.0.0.1",
				Port: 8001,
			},
			{
				Host: "127.0.0.1",
				Port: 8002,
			},
		},
		LBMode: tcplb.LBRoundRobin,
	}).Run(1); err != nil {
		log.Fatalf("Error starting the load balancer: %s", err)
	}
	<-make(chan struct{})
}
