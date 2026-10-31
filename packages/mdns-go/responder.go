package mdns

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ResponderConfig describes one DNS-SD service instance. URL is advertised as
// a TXT value so consumers can use an endpoint that is not reconstructible
// from the SRV host (for example a tunnel or HTTPS proxy).
type ResponderConfig struct {
	Service   string
	Instance  string
	Host      string
	Port      int
	Address   net.IP
	URL       string
	TXT       map[string]string
	Interface *net.Interface
}

// Responder is a small, service-agnostic mDNS responder. It answers PTR
// queries for one service and includes the SRV, TXT and address records needed
// by a DNS-SD browser to construct an endpoint.
type Responder struct {
	config ResponderConfig
	mu     sync.Mutex
	conns  []packetConn
}

const (
	responderReadDeadline = 250 * time.Millisecond
	dnsHeaderSize         = 12
	dnsResponseFlags      = 0x8400
	dnsRecordCount        = 3
	dnsClassIN            = 1
	dnsPTRType            = 12
	dnsTXTType            = 16
	dnsAType              = 1
	dnsSRVType            = 33
	dnsRecordTTL          = 120
	dnsSRVDataSize        = 6
	dnsMaxTXTEntrySize    = 255
)

func NewResponder(config ResponderConfig) *Responder { return &Responder{config: config} }

// Start begins serving and returns after the socket is ready. A failed or
// unavailable multicast route is returned to the caller, which should log and
// continue with manual discovery.
func (r *Responder) Start(ctx context.Context) error {
	service := normalizeDNSName(r.config.Service)
	instance := normalizeInstanceName(r.config.Instance, service)
	host := normalizeDNSName(r.config.Host)
	if service == "" || instance == "" || host == "" || r.config.Port <= 0 || r.config.Port > 65535 {
		return errors.New("mDNS responder requires service, instance, host, and valid port")
	}
	iface := r.config.Interface
	if iface == nil {
		interfaces, err := net.Interfaces()
		if err != nil {
			return fmt.Errorf("enumerate multicast interfaces: %w", err)
		}
		for i := range interfaces {
			candidate := &interfaces[i]
			if candidate.Flags&net.FlagUp != 0 && candidate.Flags&net.FlagMulticast != 0 {
				iface = candidate
				break
			}
		}
	}
	if iface == nil {
		return errors.New("mDNS responder has no usable multicast interface")
	}
	conn, err := listenMulticast(iface)
	if err != nil {
		return fmt.Errorf("open mDNS responder: %w", err)
	}
	r.mu.Lock()
	r.conns = append(r.conns, conn)
	r.mu.Unlock()
	go r.serve(ctx, conn, service, instance, host)
	return nil
}

func (r *Responder) serve(ctx context.Context, conn packetConn, service, instance, host string) {
	defer r.remove(conn)
	defer conn.Close()
	packet := make([]byte, maxPacketSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, from, err := readResponder(conn, packet)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return
			}
			continue
		}
		if !isDNSQueryFor(packet[:n], service) {
			continue
		}
		response := encodeResponse(service, instance, host, r.config.Port, r.config.Address, r.config.URL, r.config.TXT)
		_, _ = conn.WriteTo(response, from)
	}
}

func readResponder(conn packetConn, packet []byte) (int, net.Addr, error) {
	// UDP deadlines are installed by the concrete socket through this helper's
	// optional interface. The short deadline makes context cancellation prompt.
	if deadlineConn, ok := conn.(interface{ SetReadDeadline(time.Time) error }); ok {
		_ = deadlineConn.SetReadDeadline(time.Now().Add(responderReadDeadline))
	}
	n, from, err := conn.ReadFrom(packet)
	return n, from, err
}

func (r *Responder) remove(conn packetConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, current := range r.conns {
		if current == conn {
			r.conns = append(r.conns[:i], r.conns[i+1:]...)
			return
		}
	}
}

// Close stops all sockets. It is safe to call more than once.
func (r *Responder) Close() error {
	r.mu.Lock()
	conns := append([]packetConn(nil), r.conns...)
	r.conns = nil
	r.mu.Unlock()
	var first error
	for _, conn := range conns {
		if err := conn.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func normalizeDNSName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.TrimSuffix(value, ".") + "."
}

// normalizeInstanceName accepts the human-readable instance label commonly
// supplied by callers and expands it to the fully qualified DNS-SD instance
// name used by PTR, SRV, and TXT records. Fully qualified names remain
// unchanged so callers that already own the service suffix are supported.
func normalizeInstanceName(instance, service string) string {
	instance = normalizeDNSName(instance)
	service = normalizeDNSName(service)
	if instance == "" || service == "" {
		return instance
	}
	if strings.HasSuffix(strings.ToLower(instance), strings.ToLower(service)) {
		return instance
	}
	return strings.TrimSuffix(instance, ".") + "." + service
}

func isDNSQueryFor(packet []byte, service string) bool {
	if len(packet) < dnsHeaderSize {
		return false
	}
	name, _, ok := readDNSName(packet, dnsHeaderSize)
	return ok && strings.EqualFold(name, service)
}

func readDNSName(packet []byte, offset int) (string, int, bool) {
	var labels []string
	for offset < len(packet) {
		length := int(packet[offset])
		offset++
		if length == 0 {
			return strings.Join(labels, ".") + ".", offset, true
		}
		if length > 63 || offset+length > len(packet) {
			return "", offset, false
		}
		labels = append(labels, string(packet[offset:offset+length]))
		offset += length
	}
	return "", offset, false
}

func encodeResponse(service, instance, host string, port int, address net.IP, advertisedURL string, values map[string]string) []byte {
	answers := make([]byte, dnsHeaderSize)
	binary.BigEndian.PutUint16(answers[2:4], dnsResponseFlags)
	binary.BigEndian.PutUint16(answers[6:8], dnsClassIN)
	binary.BigEndian.PutUint16(answers[10:12], dnsRecordCount)
	answers = appendRecord(answers, service, dnsPTRType, dnsRecordTTL, encodeName(instance))
	srv := make([]byte, dnsSRVDataSize)
	binary.BigEndian.PutUint16(srv[4:], uint16(port))
	srv = append(srv, encodeName(host)...)
	answers = appendRecord(answers, instance, dnsSRVType, dnsRecordTTL, srv)
	textValues := make([]byte, 0)
	if advertisedURL != "" {
		values = copyTXT(values)
		values["url"] = advertisedURL
	}
	for key, value := range values {
		entry := []byte(key + "=" + value)
		if len(entry) > dnsMaxTXTEntrySize {
			entry = entry[:dnsMaxTXTEntrySize]
		}
		textValues = append(textValues, byte(len(entry)))
		textValues = append(textValues, entry...)
	}
	answers = appendRecord(answers, instance, dnsTXTType, dnsRecordTTL, textValues)
	if ip := address.To4(); ip != nil {
		answers = appendRecord(answers, host, dnsAType, dnsRecordTTL, ip)
	}
	return answers
}

func copyTXT(values map[string]string) map[string]string {
	copy := make(map[string]string, len(values)+1)
	for key, value := range values {
		copy[key] = value
	}
	return copy
}

func appendRecord(packet []byte, name string, recordType uint16, ttl uint32, data []byte) []byte {
	packet = append(packet, encodeName(name)...)
	var header [10]byte
	binary.BigEndian.PutUint16(header[0:2], recordType)
	binary.BigEndian.PutUint16(header[2:4], 1)
	binary.BigEndian.PutUint32(header[4:8], ttl)
	binary.BigEndian.PutUint16(header[8:10], uint16(len(data)))
	packet = append(packet, header[:]...)
	return append(packet, data...)
}

func encodeName(name string) []byte {
	var out []byte
	for _, label := range strings.Split(strings.TrimSuffix(name, "."), ".") {
		if label == "" || len(label) > 63 {
			return nil
		}
		out = append(out, byte(len(label)))
		out = append(out, label...)
	}
	return append(out, 0)
}
