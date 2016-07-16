package tcplb_test

import "github.com/creack/tcplb"

func ExampleServer() {
	(&tcplb.Server{
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
	}).Run()
}
