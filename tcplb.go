package tcplb

import (
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"sync/atomic"
)

// Target represent a backend.
type Target struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	ActiveConn int64  `json:"active_conn"`
	Errors     int64  `json:"errors"`
}

func (t Target) String() string {
	return fmt.Sprintf("%s:%d", t.Host, t.Port)
}

// Targets is a list of Target.
type Targets []*Target

// Len implements sort.Interface.
func (t Targets) Len() int {
	return len(t)
}

// Less implements sort.Interface.
func (t Targets) Less(i, j int) bool {
	return t[i].ActiveConn < t[j].ActiveConn
}

// Swap implements sort.Interface.
func (t Targets) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// ipHash hashes the given ip modulo the given length.
func ipHash(addr net.Addr, length int) int {
	// Extract the IP from the CIDR.
	ip, _, err := net.ParseCIDR(addr.String())
	if err != nil {
		panic(fmt.Errorf("Error parsing remote IP: %s", err))
	}

	// Sum up the bytes.
	total := 0
	for _, elem := range ip {
		total += int(elem)
	}

	// Modulo requested length.
	return int(total) % length
}

// Server holds the load balancer state.
type Server struct {
	Laddr           string
	Targets         Targets
	LBMode          LBMode
	roundRobinIndex int64
}

// LBMode is the Load Balancer mode enum type.
type LBMode int

// Load Balancer mode enum values.
const (
	LBLeastConn LBMode = iota << 1
	LBRoundRobin
	LBIpHash
)

// LoadBalance returns a target based on the requested mode.
func (s *Server) LoadBalance(conn net.Conn) *Target {
	// No available target, error out.
	if len(s.Targets) == 0 {
		panic(fmt.Errorf("no target to balance on"))
	}

	// Only one target available, use it.
	if len(s.Targets) == 1 {
		return s.Targets[0]
	}

	// Multiple targets available, load balance based on s.LBMode.
	var ret *Target

	switch s.LBMode {
	case LBRoundRobin:
		ret = s.Targets[int(atomic.AddInt64(&s.roundRobinIndex, 1)-1)%len(s.Targets)]
	case LBIpHash:
		ret = s.Targets[ipHash(conn.RemoteAddr(), len(s.Targets))]
	case LBLeastConn:
		sort.Sort(s.Targets)
		ret = s.Targets[0]
	default:
		panic(fmt.Errorf("Unkown LB mode: %d", s.LBMode))
	}

	return ret
}

// isClosedConnection checks if the given error is net.errClosing. Not exposed so we need that helper.
func isClosedConnection(err error) bool {
	if e1, ok := err.(*net.OpError); ok {
		if e2, ok := e1.Err.(*net.OpError); ok {
			return e2.Err.Error() == "use of closed network connection"
		}
	}
	return false
}

// Run starts the load balancer.
func (s *Server) Run() {
	ln, err := net.Listen("tcp", s.Laddr)
	if err != nil {
		log.Fatalf("Error listenning to laddr: %s", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("Error accepting connection: %s", err)
		}
		go func() {
			defer conn.Close()

			target := s.LoadBalance(conn)

			rConn, err := net.Dial("tcp", target.String())
			if err != nil {
				atomic.AddInt64(&target.Errors, 1)
				log.Printf("Error connecting to remote target %s (%s)", target, err)
				return
			}
			atomic.AddInt64(&target.ActiveConn, 1)
			go func() {
				defer rConn.Close()

				if _, err := io.Copy(rConn, conn); err != nil {
					// If error not a disconnect, display.
					if !isClosedConnection(err) {
						log.Printf("Connection error with remote target %s: %s", target, err)
					}
					atomic.AddInt64(&target.Errors, 1)
				}
			}()
			// Check errors from remote.
			if _, err := io.Copy(conn, rConn); err != nil {
				// If error not a disconnect, display.
				if !isClosedConnection(err) {
					log.Printf("Connection error with client: %s", err)
				}
			}
			atomic.AddInt64(&target.ActiveConn, -1)
		}()
	}
}
