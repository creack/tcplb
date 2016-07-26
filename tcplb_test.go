package tcplb_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/creack/tcplb"
)

func extractHost(url string) (string, int) {
	url = strings.TrimPrefix(url, "http://")
	tmp := strings.Split(url, ":")
	host := tmp[0]
	port, err := strconv.ParseInt(tmp[1], 10, 64)
	if err != nil {
		panic(fmt.Errorf("Error parsing url port: %s (%s)", err, url))
	}
	return host, int(port)
}

func TestRoundRobin(t *testing.T) {
	var (
		remote1Count int
		remote2Count int
	)
	remote1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		remote1Count++
		w.Header().Add("Connection", "Close")
	}))
	defer remote1.Close()

	remote2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		remote2Count++
		w.Header().Add("Connection", "Close")
	}))
	defer remote2.Close()

	remote1Host, remote1Port := extractHost(remote1.URL)
	remote2Host, remote2Port := extractHost(remote2.URL)

	srv := &tcplb.Server{
		Laddr: "127.0.0.1:0",
		Targets: tcplb.Targets{
			0: {
				Host: remote1Host,
				Port: remote1Port,
			},
			1: {
				Host: remote2Host,
				Port: remote2Port,
			},
		},
		LBMode: tcplb.LBRoundRobin,
	}
	if err := srv.Run(1); err != nil {
		t.Fatalf("Error starting the load balancer: %s", err)
	}
	defer func() { _ = srv.Close() }()

	callLB := func() {
		resp, err := http.Get("http://" + srv.Laddr)
		if err != nil {
			t.Fatalf("Error requesting http server via LB: %s", err)
		}
		_ = resp.Body.Close()
	}

	callLB()
	if remote1Count != 1 || remote2Count != 0 {
		t.Fatalf("Unexpected counters. Expect 1 for target1, 0 for target2. Got: %d, %d", remote1Count, remote2Count)
	}
	callLB()
	if remote1Count != 1 || remote2Count != 1 {
		t.Fatalf("Unexpected counters. Expect 1 for target1, 1 for target2. Got: %d, %d", remote1Count, remote2Count)
	}
	callLB()
	if remote1Count != 2 || remote2Count != 1 {
		t.Fatalf("Unexpected counters. Expect 2 for target1, 1 for target2. Got: %d, %d", remote1Count, remote2Count)
	}
	callLB()
	if remote1Count != 2 || remote2Count != 2 {
		t.Fatalf("Unexpected counters. Expect 2 for target1, 2 for target2. Got: %d, %d", remote1Count, remote2Count)
	}
}
