//go:build !linux

package mdns

import (
	"fmt"
	"net"
	"time"
)

type udpPacketConn struct{ conn *net.UDPConn }

func listenMulticast(iface *net.Interface) (packetConn, error) {
	v4, v4Err := listenMulticastIPv4(iface)
	v6, v6Err := listenMulticastIPv6(iface)
	if v4 == nil && v6 == nil {
		return nil, fmt.Errorf("IPv4 multicast: %v; IPv6 multicast: %v", v4Err, v6Err)
	}
	return newDualPacketConn(v4, v6), nil
}

func listenMulticastIPv4(iface *net.Interface) (packetConn, error) {
	conn, err := net.ListenMulticastUDP("udp4", iface, &net.UDPAddr{IP: net.ParseIP(MulticastIPv4), Port: MulticastPort})
	if err != nil {
		return nil, err
	}
	return &udpPacketConn{conn: conn}, nil
}

func listenMulticastIPv6(iface *net.Interface) (packetConn, error) {
	conn, err := net.ListenMulticastUDP("udp6", iface, &net.UDPAddr{IP: net.ParseIP(MulticastIPv6), Port: MulticastPort})
	if err != nil {
		return nil, err
	}
	return &udpPacketConn{conn: conn}, nil
}

func (c *udpPacketConn) ReadFrom(payload []byte) (int, net.Addr, error) {
	return c.conn.ReadFrom(payload)
}

func (c *udpPacketConn) WriteTo(payload []byte, addr net.Addr) (int, error) {
	return c.conn.WriteTo(payload, addr)
}

func (c *udpPacketConn) SetReadDeadline(deadline time.Time) error {
	return c.conn.SetReadDeadline(deadline)
}
func (c *udpPacketConn) Close() error { return c.conn.Close() }
