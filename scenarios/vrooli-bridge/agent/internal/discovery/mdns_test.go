package discovery

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeBrowser is the test seam substitute for MDNSBrowser: it returns a canned
// set of instances (or an error) and records whether it was invoked.
//
// [REQ:BRG-P1-006]
type fakeBrowser struct {
	instances []ServiceInstance
	err       error
	called    bool
}

func (f *fakeBrowser) Browse(_ context.Context, _ string, _ time.Duration) ([]ServiceInstance, error) {
	f.called = true
	return f.instances, f.err
}

// compile-time assertion the fake satisfies the discovery seam, mirroring the
// real MDNSBrowser assertion in mdns.go.
var _ Browser = (*fakeBrowser)(nil)

// [REQ:BRG-P1-006] mDNS discovery locates an advertised control plane on the LAN
// and resolves it to the advertised dial-out URL.
func TestResolve_DiscoversAdvertisedControlPlane(t *testing.T) {
	br := &fakeBrowser{instances: []ServiceInstance{
		{Host: "10.0.0.9", Port: 8443, URL: "https://10.0.0.9:8443"},
	}}

	got, err := Resolve(context.Background(), "", br, time.Second)
	require.NoError(t, err)
	require.True(t, br.called, "with no manual URL the browser must be consulted")
	require.True(t, got.Found())
	require.Equal(t, SourceDiscovered, got.Source)
	require.Equal(t, "https://10.0.0.9:8443", got.URL)
	require.Equal(t, "10.0.0.9", got.Instance.Host)
	require.Equal(t, 8443, got.Instance.Port)
}

// [REQ:BRG-P1-006] When mDNS finds nothing, Resolve falls back cleanly to the
// manual URL path with no error spam.
func TestResolve_FallsBackCleanlyWhenNothingAdvertised(t *testing.T) {
	br := &fakeBrowser{instances: nil}

	got, err := Resolve(context.Background(), "", br, time.Second)
	require.NoError(t, err, "an empty browse is a normal outcome, not an error")
	require.True(t, br.called)
	require.False(t, got.Found())
	require.Equal(t, SourceNone, got.Source)
	require.Empty(t, got.URL)
}

// [REQ:BRG-P1-006] A transport-level browse failure is non-fatal: Resolve
// surfaces it for optional logging but still returns a clean SourceNone result
// so the manual URL path remains the fallback.
func TestResolve_TransportErrorIsNonFatal(t *testing.T) {
	br := &fakeBrowser{err: context.DeadlineExceeded}

	got, err := Resolve(context.Background(), "", br, time.Second)
	require.Error(t, err)
	require.False(t, got.Found())
	require.Equal(t, SourceNone, got.Source)
}

// [REQ:BRG-P1-006] An instance advertised without a usable URL is skipped, and
// the first instance carrying a URL wins.
func TestResolve_SkipsInstancesWithoutURL(t *testing.T) {
	br := &fakeBrowser{instances: []ServiceInstance{
		{Host: "h1", Port: 1},
		{Host: "h2", Port: 2, URL: "https://h2:2"},
	}}

	got, err := Resolve(context.Background(), "", br, time.Second)
	require.NoError(t, err)
	require.Equal(t, "https://h2:2", got.URL)
}

// [REQ:BRG-P1-006] With no browser wired (discovery disabled) and no manual URL,
// Resolve returns a clean SourceNone without panicking.
func TestResolve_NilBrowserIsCleanNone(t *testing.T) {
	got, err := Resolve(context.Background(), "", nil, time.Second)
	require.NoError(t, err)
	require.Equal(t, SourceNone, got.Source)
}

// [REQ:BRG-P1-006] encodeQuery/parseMessage round-trip: the minimal DNS codec is
// verified against a HAND-BUILT mDNS response packet (no network), so the wire
// encode/decode of the PTR → SRV/A chain has real coverage.
func TestDNSCodec_RoundTripsHandBuiltPacket(t *testing.T) {
	// The query side: encode a PTR question and decode it back to confirm the
	// header counts and name labels are written correctly.
	query, err := encodeQuery("_vrooli-bridge._tcp.local", dnsTypePTR)
	require.NoError(t, err)
	require.Equal(t, uint16(1), binary.BigEndian.Uint16(query[4:6]), "qdcount must be 1")
	// parseMessage must skip the question section without error even though there
	// are no answers.
	qmsg, err := parseMessage(query)
	require.NoError(t, err)
	require.Empty(t, qmsg.records)

	// The response side: a hand-built mDNS answer carrying the PTR → SRV → A
	// chain for one advertised control plane.
	const (
		service  = "_vrooli-bridge._tcp.local"
		instance = "cp1._vrooli-bridge._tcp.local"
		host     = "cp1.local"
		port     = 8443
	)
	packet := buildResponse(t, service, instance, host, port, []byte{192, 168, 1, 50})

	msg, err := parseMessage(packet)
	require.NoError(t, err)

	rs := newRecordSet()
	rs.absorb(msg)
	insts := rs.instances()
	require.Len(t, insts, 1)
	require.Equal(t, "192.168.1.50", insts[0].Host)
	require.Equal(t, port, insts[0].Port)
	require.Equal(t, "https://192.168.1.50:8443", insts[0].URL)
}

// [REQ:BRG-P1-006] parseMessage rejects a truncated message rather than panicking
// on a noisy LAN.
func TestParseMessage_RejectsTruncated(t *testing.T) {
	_, err := parseMessage([]byte{0, 0, 0})
	require.Error(t, err)
}

// [REQ:BRG-P1-006] encodeName rejects empty and over-long labels.
func TestEncodeName_RejectsInvalidLabels(t *testing.T) {
	_, err := encodeName("")
	require.Error(t, err)
	_, err = encodeName("a..b")
	require.Error(t, err)
}

// buildResponse assembles a minimal but valid mDNS response packet carrying a PTR
// answer plus SRV/TXT/A additional records for one advertised instance. It uses
// only the package codec's wire conventions (length-prefixed labels, no
// compression) so the test exercises parseMessage against realistic bytes.
//
// [REQ:BRG-P1-006]
func buildResponse(t *testing.T, service, instance, host string, port int, addr []byte) []byte {
	t.Helper()

	name := func(n string) []byte {
		b, err := encodeName(n)
		require.NoError(t, err)
		return b
	}

	rr := func(owner []byte, rtype uint16, rdata []byte) []byte {
		out := append([]byte{}, owner...)
		hdr := make([]byte, 10)
		binary.BigEndian.PutUint16(hdr[0:2], rtype)
		binary.BigEndian.PutUint16(hdr[2:4], dnsClassIN)
		// ttl hdr[4:8] left zero
		binary.BigEndian.PutUint16(hdr[8:10], uint16(len(rdata)))
		out = append(out, hdr...)
		return append(out, rdata...)
	}

	// PTR rdata = the instance name.
	ptr := rr(name(service), dnsTypePTR, name(instance))

	// SRV rdata = priority(2) weight(2) port(2) target-name.
	srvRdata := make([]byte, 6)
	binary.BigEndian.PutUint16(srvRdata[4:6], uint16(port))
	srvRdata = append(srvRdata, name(host)...)
	srv := rr(name(instance), dnsTypeSRV, srvRdata)

	// TXT rdata = one length-prefixed key=value string.
	txtVal := "path=/api"
	txtRdata := append([]byte{byte(len(txtVal))}, txtVal...)
	txt := rr(name(instance), dnsTypeTXT, txtRdata)

	// A rdata = the 4-byte address.
	a := rr(name(host), dnsTypeA, addr)

	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[2:4], 0x8400) // QR=1, AA=1 (mDNS response)
	binary.BigEndian.PutUint16(header[6:8], 1)      // ancount: the PTR answer
	binary.BigEndian.PutUint16(header[10:12], 3)    // arcount: SRV, TXT, A

	packet := append([]byte{}, header...)
	packet = append(packet, ptr...)
	packet = append(packet, srv...)
	packet = append(packet, txt...)
	packet = append(packet, a...)
	return packet
}
