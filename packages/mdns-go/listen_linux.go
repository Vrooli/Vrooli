//go:build linux

package mdns

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
)

type multicastPacketConn struct{ conn *ipv4.PacketConn }

func listenMulticast(iface *net.Interface) (packetConn, error) {
	v4, v4Err := listenMulticastIPv4(iface)
	v6, v6Err := listenMulticastIPv6(iface)
	if v4 == nil && v6 == nil {
		return nil, fmt.Errorf("IPv4 multicast: %v; IPv6 multicast: %v", v4Err, v6Err)
	}
	return newDualPacketConn(v4, v6), nil
}

func listenMulticastIPv4(iface *net.Interface) (packetConn, error) {
	config := net.ListenConfig{Control: func(_ string, _ string, raw syscall.RawConn) error {
		var controlErr error
		if err := raw.Control(func(fd uintptr) {
			fileDescriptor := int(fd)
			// Multicast listeners commonly share 5353 with Avahi or another
			// DNS-SD daemon. REUSEADDR is the multicast fan-out contract;
			// REUSEPORT would hash multicast packets to one listener instead of
			// delivering the browse response to every joined consumer.
			if err := unix.SetsockoptInt(fileDescriptor, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
				controlErr = err
				return
			}
		}); err != nil {
			return err
		}
		return controlErr
	}}
	packet, err := config.ListenPacket(context.Background(), "udp4", ":5353")
	if err != nil {
		// A host daemon may not permit a second 5353 listener. The ephemeral
		// fallback remains useful for responders that honor QU questions.
		packet, err = config.ListenPacket(context.Background(), "udp4", ":0")
		if err != nil {
			return nil, err
		}
	}
	conn := ipv4.NewPacketConn(packet)
	if err := conn.JoinGroup(iface, &net.UDPAddr{IP: net.ParseIP(MulticastIPv4)}); err != nil {
		_ = packet.Close()
		return nil, err
	}
	if err := conn.SetMulticastInterface(iface); err != nil {
		_ = packet.Close()
		return nil, err
	}
	return &multicastPacketConn{conn: conn}, nil
}

type multicastIPv6PacketConn struct{ conn *ipv6.PacketConn }

func listenMulticastIPv6(iface *net.Interface) (packetConn, error) {
	config := net.ListenConfig{Control: func(_ string, _ string, raw syscall.RawConn) error {
		var controlErr error
		if err := raw.Control(func(fd uintptr) {
			if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
				controlErr = err
			}
		}); err != nil {
			return err
		}
		return controlErr
	}}
	packet, err := config.ListenPacket(context.Background(), "udp6", ":5353")
	if err != nil {
		packet, err = config.ListenPacket(context.Background(), "udp6", ":0")
		if err != nil {
			return nil, err
		}
	}
	conn := ipv6.NewPacketConn(packet)
	if err := conn.JoinGroup(iface, &net.UDPAddr{IP: net.ParseIP(MulticastIPv6), Port: MulticastPort}); err != nil {
		_ = packet.Close()
		return nil, err
	}
	if err := conn.SetMulticastInterface(iface); err != nil {
		_ = packet.Close()
		return nil, err
	}
	return &multicastIPv6PacketConn{conn: conn}, nil
}

func (c *multicastPacketConn) ReadFrom(payload []byte) (int, net.Addr, error) {
	n, _, addr, err := c.conn.ReadFrom(payload)
	return n, addr, err
}

func (c *multicastPacketConn) WriteTo(payload []byte, addr net.Addr) (int, error) {
	return c.conn.WriteTo(payload, nil, addr)
}

func (c *multicastPacketConn) SetReadDeadline(deadline time.Time) error {
	return c.conn.SetReadDeadline(deadline)
}

func (c *multicastPacketConn) Close() error { return c.conn.Close() }

func (c *multicastIPv6PacketConn) ReadFrom(payload []byte) (int, net.Addr, error) {
	n, _, addr, err := c.conn.ReadFrom(payload)
	return n, addr, err
}

func (c *multicastIPv6PacketConn) WriteTo(payload []byte, addr net.Addr) (int, error) {
	return c.conn.WriteTo(payload, nil, addr)
}

func (c *multicastIPv6PacketConn) SetReadDeadline(deadline time.Time) error {
	return c.conn.SetReadDeadline(deadline)
}

func (c *multicastIPv6PacketConn) Close() error { return c.conn.Close() }
