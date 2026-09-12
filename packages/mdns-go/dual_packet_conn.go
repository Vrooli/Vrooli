package mdns

import (
	"net"
	"sync"
	"time"
)

// dualPacketConn presents IPv4 and IPv6 multicast sockets as one bounded
// reader. Short alternating reads avoid dedicating a goroutine to each socket
// and still let either address family deliver a browse response promptly.
type dualPacketConn struct {
	v4       packetConn
	v6       packetConn
	mu       sync.RWMutex
	deadline time.Time
}

func newDualPacketConn(v4, v6 packetConn) packetConn {
	if v4 == nil {
		return v6
	}
	if v6 == nil {
		return v4
	}
	return &dualPacketConn{v4: v4, v6: v6}
}

func (c *dualPacketConn) ReadFrom(payload []byte) (int, net.Addr, error) {
	deadline := c.readDeadline()
	short := time.Now().Add(10 * time.Millisecond)
	if deadline.IsZero() || short.Before(deadline) {
		deadline = short
	}
	_ = setReadDeadline(c.v4, deadline)
	if n, addr, err := c.v4.ReadFrom(payload); err == nil || !isTimeout(err) {
		return n, addr, err
	}
	deadline = time.Now().Add(10 * time.Millisecond)
	callerDeadline := c.readDeadline()
	if !callerDeadline.IsZero() && callerDeadline.Before(deadline) {
		deadline = callerDeadline
	}
	_ = setReadDeadline(c.v6, deadline)
	if n, addr, err := c.v6.ReadFrom(payload); err == nil || !isTimeout(err) {
		return n, addr, err
	}
	return 0, nil, dualTimeoutError{}
}

func (c *dualPacketConn) WriteTo(payload []byte, addr net.Addr) (int, error) {
	udp, ok := addr.(*net.UDPAddr)
	if ok && udp.IP.To4() == nil {
		return c.v6.WriteTo(payload, addr)
	}
	return c.v4.WriteTo(payload, addr)
}

func (c *dualPacketConn) SetReadDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.deadline = deadline
	c.mu.Unlock()
	return nil
}

func (c *dualPacketConn) readDeadline() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.deadline
}

func (c *dualPacketConn) Close() error {
	err4 := c.v4.Close()
	err6 := c.v6.Close()
	if err4 != nil {
		return err4
	}
	return err6
}

type dualTimeoutError struct{}

func (dualTimeoutError) Error() string   { return "mDNS read timeout" }
func (dualTimeoutError) Timeout() bool   { return true }
func (dualTimeoutError) Temporary() bool { return true }
