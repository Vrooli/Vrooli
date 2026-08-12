// mdns.go is the node-agent's minimal, dependency-free mDNS/DNS-SD querier
// (OT-P1-006). It speaks just enough of RFC 6762 (multicast DNS) and RFC 6763
// (DNS-SD) to find a control plane that advertises `_vrooli-bridge._tcp.local`
// on a trusted LAN, using ONLY the Go stdlib `net` package — adding a real mDNS
// library would route through dependency governance (SDA), which is out of scope
// for this convenience path, and the agent must cross-compile CGO_ENABLED=0
// across the whole matrix.
//
// MDNSBrowser sends a single PTR query to the mDNS multicast group
// (224.0.0.251:5353), reads the answers for a short window, and decodes the
// PTR → SRV/A/TXT chain into discovery.ServiceInstance values. The DNS message
// encode/decode is a small self-contained codec (encodeQuery / parseMessage),
// kept exported-to-package and pure so its wire logic is unit-testable against a
// hand-built packet with no network. Any real send/recv failure (no route, no
// multicast) degrades to an empty result so discovery.Resolve falls back
// cleanly to the manual URL path.
package discovery

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// mdnsGroupAddr is the IPv4 mDNS multicast endpoint (RFC 6762 §3).
const mdnsGroupAddr = "224.0.0.251:5353"

// DNS resource-record type codes (RFC 1035 / RFC 2782) the querier understands.
const (
	dnsTypeA   = 1
	dnsTypePTR = 12
	dnsTypeTXT = 16
	dnsTypeSRV = 33

	dnsClassIN = 1
)

// MDNSBrowser is the production Browser: it queries the LAN over real UDP
// multicast. Group defaults to the standard mDNS endpoint; tests of the codec
// use the pure helpers directly rather than this type.
type MDNSBrowser struct {
	// Group is the multicast endpoint to query; empty uses mdnsGroupAddr.
	Group string
}

// compile-time assertion that the real querier satisfies the discovery seam.
var _ Browser = (*MDNSBrowser)(nil)

// Browse sends one PTR query for service to the mDNS group and collects answers
// until timeout elapses, decoding them into ServiceInstance values. A transport
// failure (no multicast route, unreadable socket) returns a wrapped error and an
// empty slice; finding nothing within the window returns an empty slice and a
// nil error.
func (b *MDNSBrowser) Browse(ctx context.Context, service string, timeout time.Duration) ([]ServiceInstance, error) {
	group := b.Group
	if group == "" {
		group = mdnsGroupAddr
	}

	groupAddr, err := net.ResolveUDPAddr("udp4", group)
	if err != nil {
		return nil, fmt.Errorf("resolve mdns group %q: %w", group, err)
	}

	// Bind an ephemeral UDP socket. We send to the multicast group and read the
	// unicast/multicast responses back on the same socket.
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("open mdns socket: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set mdns deadline: %w", err)
	}

	query, err := encodeQuery(service, dnsTypePTR)
	if err != nil {
		return nil, fmt.Errorf("encode mdns query: %w", err)
	}
	if _, err := conn.WriteToUDP(query, groupAddr); err != nil {
		return nil, fmt.Errorf("send mdns query: %w", err)
	}

	// Accumulate records across every response packet until the deadline, then
	// stitch the PTR → SRV/A/TXT chain into instances. Reading past the deadline
	// returns a timeout error, which is the normal terminal condition — not a
	// failure to report.
	agg := newRecordSet()
	buf := make([]byte, 9000) // jumbo-frame headroom; mDNS packets are far smaller
	for {
		if ctx.Err() != nil {
			break
		}
		n, _, rerr := conn.ReadFromUDP(buf)
		if rerr != nil {
			if isTimeout(rerr) {
				break
			}
			// A mid-stream read error means we stop collecting but still return
			// whatever we decoded so far (best-effort).
			break
		}
		msg, perr := parseMessage(buf[:n])
		if perr != nil {
			continue // ignore malformed packets from noisy LANs
		}
		agg.absorb(msg)
	}

	return agg.instances(), nil
}

func isTimeout(err error) bool {
	var nerr net.Error
	return errors.As(err, &nerr) && nerr.Timeout()
}

// dnsRecord is a decoded resource record carrying only the fields the DNS-SD
// chain needs. The raw RDATA is retained so a test can assert the codec parsed
// the wire bytes, and so SRV/A targets can be cross-referenced by name.
type dnsRecord struct {
	name  string
	rtype uint16
	// PTR target (the instance name); SRV target host + port; A address; TXT
	// key/value pairs — only the field matching rtype is populated.
	ptrTarget string
	srvTarget string
	srvPort   uint16
	aAddr     net.IP
	txt       []string
}

// dnsMessage is a decoded mDNS response: the records from the answer + additional
// sections (mDNS responders pack the SRV/A/TXT into additionals).
type dnsMessage struct {
	records []dnsRecord
}

// encodeQuery builds a single-question DNS query packet for name/qtype with the
// mDNS conventions (id 0, no flags, QU bit left clear). It is pure and exported
// to the package so the round-trip codec is testable without a socket.
func encodeQuery(name string, qtype uint16) ([]byte, error) {
	labels, err := encodeName(name)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 0, 12+len(labels)+4)
	header := make([]byte, 12)
	// id=0, flags=0 (standard query), qdcount=1, the rest zero.
	binary.BigEndian.PutUint16(header[4:6], 1)
	msg = append(msg, header...)
	msg = append(msg, labels...)
	qt := make([]byte, 4)
	binary.BigEndian.PutUint16(qt[0:2], qtype)
	binary.BigEndian.PutUint16(qt[2:4], dnsClassIN)
	msg = append(msg, qt...)
	return msg, nil
}

// encodeName encodes a dotted DNS name into length-prefixed labels terminated by
// a zero byte. It rejects empty or over-long (>63 byte) labels.
func encodeName(name string) ([]byte, error) {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	if name == "" {
		return nil, fmt.Errorf("empty dns name")
	}
	var out []byte
	for _, label := range strings.Split(name, ".") {
		if label == "" {
			return nil, fmt.Errorf("empty label in dns name %q", name)
		}
		if len(label) > 63 {
			return nil, fmt.Errorf("label %q exceeds 63 bytes", label)
		}
		out = append(out, byte(len(label))) // #nosec G115 -- the preceding 63-byte label bound is the DNS wire maximum.
		out = append(out, label...)
	}
	return append(out, 0), nil
}

// parseMessage decodes a DNS message, returning the records from the answer and
// additional sections. It tolerates compression pointers (RFC 1035 §4.1.4),
// which mDNS responders use heavily. It is pure and exported to the package so
// the codec has real coverage against a hand-built packet.
func parseMessage(data []byte) (dnsMessage, error) {
	if len(data) < 12 {
		return dnsMessage{}, fmt.Errorf("dns message too short: %d bytes", len(data))
	}
	qd := int(binary.BigEndian.Uint16(data[4:6]))
	an := int(binary.BigEndian.Uint16(data[6:8]))
	ns := int(binary.BigEndian.Uint16(data[8:10]))
	ar := int(binary.BigEndian.Uint16(data[10:12]))

	off := 12
	var err error
	// Skip the question section.
	for i := 0; i < qd; i++ {
		if _, off, err = readName(data, off); err != nil {
			return dnsMessage{}, fmt.Errorf("parse question %d: %w", i, err)
		}
		if off+4 > len(data) {
			return dnsMessage{}, fmt.Errorf("truncated question %d", i)
		}
		off += 4 // qtype + qclass
	}

	total := an + ns + ar
	msg := dnsMessage{records: make([]dnsRecord, 0, total)}
	for i := 0; i < total; i++ {
		var rec dnsRecord
		if rec, off, err = readRecord(data, off); err != nil {
			return dnsMessage{}, fmt.Errorf("parse record %d: %w", i, err)
		}
		msg.records = append(msg.records, rec)
	}
	return msg, nil
}

// readRecord decodes one resource record starting at off and returns it plus the
// next offset.
func readRecord(data []byte, off int) (dnsRecord, int, error) {
	name, off, err := readName(data, off)
	if err != nil {
		return dnsRecord{}, off, err
	}
	if off+10 > len(data) {
		return dnsRecord{}, off, fmt.Errorf("truncated record header")
	}
	rtype := binary.BigEndian.Uint16(data[off : off+2])
	rdlen := int(binary.BigEndian.Uint16(data[off+8 : off+10]))
	off += 10
	if off+rdlen > len(data) {
		return dnsRecord{}, off, fmt.Errorf("truncated rdata: need %d, have %d", rdlen, len(data)-off)
	}
	rec := dnsRecord{name: name, rtype: rtype}
	rdata := data[off : off+rdlen]

	switch rtype {
	case dnsTypePTR:
		// PTR rdata is a (possibly compressed) name pointing back into the full
		// message, so decode against data, not the rdata slice.
		if rec.ptrTarget, _, err = readName(data, off); err != nil {
			return dnsRecord{}, off, fmt.Errorf("ptr target: %w", err)
		}
	case dnsTypeSRV:
		if len(rdata) < 6 {
			return dnsRecord{}, off, fmt.Errorf("srv rdata too short")
		}
		rec.srvPort = binary.BigEndian.Uint16(rdata[4:6])
		if rec.srvTarget, _, err = readName(data, off+6); err != nil {
			return dnsRecord{}, off, fmt.Errorf("srv target: %w", err)
		}
	case dnsTypeA:
		if len(rdata) != 4 {
			return dnsRecord{}, off, fmt.Errorf("a rdata must be 4 bytes, got %d", len(rdata))
		}
		rec.aAddr = net.IPv4(rdata[0], rdata[1], rdata[2], rdata[3])
	case dnsTypeTXT:
		rec.txt = parseTXT(rdata)
	}

	return rec, off + rdlen, nil
}

// parseTXT splits TXT rdata into its length-prefixed character-strings.
func parseTXT(rdata []byte) []string {
	var out []string
	for i := 0; i < len(rdata); {
		n := int(rdata[i])
		i++
		if i+n > len(rdata) {
			break
		}
		out = append(out, string(rdata[i:i+n]))
		i += n
	}
	return out
}

// readName decodes a DNS name at off, following compression pointers, and
// returns the dotted name plus the offset just past the name in the ORIGINAL
// stream (pointers do not advance the outer offset beyond their two bytes).
func readName(data []byte, off int) (string, int, error) {
	var labels []string
	jumped := false
	next := off
	// Guard against pointer loops.
	for budget := len(data); budget > 0; budget-- {
		if off >= len(data) {
			return "", next, fmt.Errorf("name runs past end of message")
		}
		b := int(data[off])
		switch {
		case b == 0:
			off++
			if !jumped {
				next = off
			}
			return strings.Join(labels, "."), next, nil
		case b&0xC0 == 0xC0:
			if off+1 >= len(data) {
				return "", next, fmt.Errorf("truncated compression pointer")
			}
			ptr := (b&0x3F)<<8 | int(data[off+1])
			if !jumped {
				next = off + 2
				jumped = true
			}
			off = ptr
		default:
			if off+1+b > len(data) {
				return "", next, fmt.Errorf("label runs past end of message")
			}
			labels = append(labels, string(data[off+1:off+1+b]))
			off += 1 + b
		}
	}
	return "", next, fmt.Errorf("dns name compression loop")
}

// recordSet accumulates records across response packets and stitches the
// PTR → SRV/A/TXT chain into ServiceInstances. mDNS spreads these across
// packets and sections, so collecting first and resolving last avoids
// ordering assumptions.
type recordSet struct {
	ptrTargets []string             // instance names from PTR answers
	srv        map[string]dnsRecord // instance name → SRV
	addrs      map[string]net.IP    // host name → A
}

func newRecordSet() *recordSet {
	return &recordSet{
		srv:   map[string]dnsRecord{},
		addrs: map[string]net.IP{},
	}
}

func (rs *recordSet) absorb(msg dnsMessage) {
	for _, rec := range msg.records {
		switch rec.rtype {
		case dnsTypePTR:
			if rec.ptrTarget != "" {
				rs.ptrTargets = append(rs.ptrTargets, rec.ptrTarget)
			}
		case dnsTypeSRV:
			rs.srv[strings.ToLower(rec.name)] = rec
		case dnsTypeA:
			if rec.aAddr != nil {
				rs.addrs[strings.ToLower(rec.name)] = rec.aAddr
			}
		}
	}
}

// instances resolves the collected records into ServiceInstances. Each PTR
// target names an SRV record (host+port); the SRV target names an A record
// (address). The dial-out URL is built from the resolved host (A address when
// known, else the SRV target host) and the SRV port.
func (rs *recordSet) instances() []ServiceInstance {
	seen := map[string]bool{}
	var out []ServiceInstance
	for _, target := range rs.ptrTargets {
		srv, ok := rs.srv[strings.ToLower(target)]
		if !ok {
			continue
		}
		host := strings.TrimSuffix(srv.srvTarget, ".")
		if ip, ok := rs.addrs[strings.ToLower(srv.srvTarget)]; ok {
			host = ip.String()
		}
		if host == "" || srv.srvPort == 0 {
			continue
		}
		inst := ServiceInstance{
			Host: host,
			Port: int(srv.srvPort),
			URL:  fmt.Sprintf("https://%s:%d", host, srv.srvPort),
		}
		if seen[inst.URL] {
			continue
		}
		seen[inst.URL] = true
		out = append(out, inst)
	}
	return out
}
