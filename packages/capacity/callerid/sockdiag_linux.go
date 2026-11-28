//go:build linux

package callerid

import (
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

// The kernel's socket-diagnostics interface answers "which socket is this
// connection" with one exact lookup, where /proc/net/tcp has to be read and
// parsed whole: on a busy host the table holds thousands of sockets and
// parsing it took longer than a short-lived caller lives. Most callers of a
// local model are resource CLIs that send one request and exit, so speed
// decides whether they are named at all.

const (
	nlmsgHeaderLen   = 16
	inetDiagReqLen   = 56
	inetDiagMsgLen   = 72
	inetDiagNoCookie = 0xFFFFFFFF
	sockDiagTimeout  = 250 * time.Millisecond
)

// diagSocketInode asks the kernel for the socket whose local end is client
// and remote end is server, returning its inode and owner.
func diagSocketInode(client, server *net.TCPAddr) (string, uint32, bool) {
	family, src, dst, ok := diagAddrs(client.IP, server.IP)
	if !ok {
		return "", 0, false
	}
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, unix.NETLINK_SOCK_DIAG)
	if err != nil {
		return "", 0, false
	}
	defer unix.Close(fd)
	timeout := unix.NsecToTimeval(int64(sockDiagTimeout))
	_ = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &timeout)
	if err := unix.Sendto(fd, diagRequest(family, client.Port, server.Port, src, dst), 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return "", 0, false
	}
	buf := make([]byte, 8192)
	n, _, err := unix.Recvfrom(fd, buf, 0)
	if err != nil {
		return "", 0, false
	}
	return parseDiagResponse(buf[:n])
}

// diagAddrs encodes both ends for one address family. A v4-mapped address is
// an IPv4 connection.
func diagAddrs(clientIP, serverIP net.IP) (uint8, [16]byte, [16]byte, bool) {
	var src, dst [16]byte
	if c4, s4 := clientIP.To4(), serverIP.To4(); c4 != nil && s4 != nil {
		copy(src[:4], c4)
		copy(dst[:4], s4)
		return unix.AF_INET, src, dst, true
	}
	c16, s16 := clientIP.To16(), serverIP.To16()
	if c16 == nil || s16 == nil {
		return 0, src, dst, false
	}
	copy(src[:], c16)
	copy(dst[:], s16)
	return unix.AF_INET6, src, dst, true
}

// diagRequest is a netlink SOCK_DIAG_BY_FAMILY request for exactly one TCP
// socket (struct inet_diag_req_v2 with a fully specified inet_diag_sockid).
func diagRequest(family uint8, sport, dport int, src, dst [16]byte) []byte {
	msg := make([]byte, nlmsgHeaderLen+inetDiagReqLen)
	binary.NativeEndian.PutUint32(msg[0:], uint32(len(msg)))
	binary.NativeEndian.PutUint16(msg[4:], unix.SOCK_DIAG_BY_FAMILY)
	binary.NativeEndian.PutUint16(msg[6:], unix.NLM_F_REQUEST)
	binary.NativeEndian.PutUint32(msg[8:], 1)
	req := msg[nlmsgHeaderLen:]
	req[0], req[1] = family, unix.IPPROTO_TCP
	binary.NativeEndian.PutUint32(req[4:], ^uint32(0)) // every TCP state
	id := req[8:]
	binary.BigEndian.PutUint16(id[0:], uint16(sport))
	binary.BigEndian.PutUint16(id[2:], uint16(dport))
	copy(id[4:20], src[:])
	copy(id[20:36], dst[:])
	binary.NativeEndian.PutUint32(id[40:], inetDiagNoCookie)
	binary.NativeEndian.PutUint32(id[44:], inetDiagNoCookie)
	return msg
}

// parseDiagResponse reads the inode and uid from the one inet_diag_msg the
// kernel returns, or reports a miss for an error reply.
func parseDiagResponse(buf []byte) (string, uint32, bool) {
	if len(buf) < nlmsgHeaderLen {
		return "", 0, false
	}
	length := binary.NativeEndian.Uint32(buf[0:])
	kind := binary.NativeEndian.Uint16(buf[4:])
	if kind != unix.SOCK_DIAG_BY_FAMILY || int(length) > len(buf) || length < nlmsgHeaderLen+inetDiagMsgLen {
		return "", 0, false
	}
	msg := buf[nlmsgHeaderLen:]
	uid := binary.NativeEndian.Uint32(msg[64:])
	inode := binary.NativeEndian.Uint32(msg[68:])
	if inode == 0 {
		return "", 0, false
	}
	return strconv.FormatUint(uint64(inode), 10), uid, true
}
