package mdns

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func TestLiveBrowseWhenRequested(t *testing.T) {
	if os.Getenv("MDNS_LIVE_CHECK") != "1" {
		t.Skip("set MDNS_LIVE_CHECK=1 for an operator-network witness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	var lanInterface *net.Interface
	for i := range interfaces {
		if interfaces[i].Flags&net.FlagUp != 0 && interfaces[i].Flags&net.FlagMulticast != 0 && interfaces[i].Flags&net.FlagLoopback == 0 {
			lanInterface = &interfaces[i]
			break
		}
	}
	if lanInterface == nil {
		t.Fatal("no non-loopback multicast interface")
	}
	browser := &Browser{Interfaces: []*net.Interface{lanInterface}}
	for _, service := range []string{"_androidtvremote2._tcp", "_googlecast._tcp"} {
		instances, err := browser.Browse(ctx, service)
		if err != nil {
			t.Fatalf("browse %s: %v", service, err)
		}
		t.Logf("live %s instances: %#v", service, instances)
		if len(instances) == 0 {
			t.Fatalf("browse %s returned no instances", service)
		}
	}
}

type fakeTimeout struct{}

func (fakeTimeout) Error() string   { return "timeout" }
func (fakeTimeout) Timeout() bool   { return true }
func (fakeTimeout) Temporary() bool { return true }

type fakePacketConn struct {
	packet []byte
	reads  int
}

func (c *fakePacketConn) ReadFrom(buffer []byte) (int, net.Addr, error) {
	if c.reads == 0 {
		c.reads++
		return copy(buffer, c.packet), &net.UDPAddr{}, nil
	}
	return 0, nil, fakeTimeout{}
}
func (c *fakePacketConn) WriteTo(payload []byte, _ net.Addr) (int, error) { return len(payload), nil }
func (c *fakePacketConn) Close() error                                    { return nil }
func (c *fakePacketConn) SetReadDeadline(time.Time) error                 { return nil }

type cancelAfterReadPacketConn struct {
	packet []byte
	cancel context.CancelFunc
	read   bool
}

func (c *cancelAfterReadPacketConn) ReadFrom(buffer []byte) (int, net.Addr, error) {
	if !c.read {
		c.read = true
		n := copy(buffer, c.packet)
		c.cancel()
		return n, &net.UDPAddr{}, nil
	}
	return 0, nil, fakeTimeout{}
}

func (c *cancelAfterReadPacketConn) WriteTo(payload []byte, _ net.Addr) (int, error) {
	return len(payload), nil
}
func (c *cancelAfterReadPacketConn) Close() error                    { return nil }
func (c *cancelAfterReadPacketConn) SetReadDeadline(time.Time) error { return nil }

func TestBrowserUsesMulticastInterfaceAndResolvesPTRResponse(t *testing.T) {
	service := "_googlecast._tcp.local"
	instance := "Living Room._googlecast._tcp.local"
	host := "living-room.local"
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true})
	_ = builder.StartQuestions()
	_ = builder.StartAnswers()
	_ = builder.PTRResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(service + "."), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET}, dnsmessage.PTRResource{PTR: dnsmessage.MustNewName(instance + ".")})
	_ = builder.SRVResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(instance + "."), Type: dnsmessage.TypeSRV, Class: dnsmessage.ClassINET}, dnsmessage.SRVResource{Port: 8009, Target: dnsmessage.MustNewName(host + ".")})
	_ = builder.TXTResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(instance + "."), Type: dnsmessage.TypeTXT, Class: dnsmessage.ClassINET}, dnsmessage.TXTResource{TXT: []string{"id=cast-1"}})
	_ = builder.StartAdditionals()
	_ = builder.AResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(host + "."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}, dnsmessage.AResource{A: [4]byte{192, 168, 1, 42}})
	packet, err := builder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	iface := &net.Interface{Name: "fixture0", Flags: net.FlagUp | net.FlagMulticast}
	browser := &Browser{Window: 5 * time.Millisecond, Interfaces: []*net.Interface{iface}, Listen: func(got *net.Interface) (packetConn, error) {
		if got != iface {
			t.Fatalf("browser used unexpected interface: %#v", got)
		}
		return &fakePacketConn{packet: packet}, nil
	}}
	instances, err := browser.Browse(context.Background(), service)
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 || instances[0].Instance != "living room._googlecast._tcp.local" || instances[0].Port != 8009 || instances[0].TXT["id"] != "cast-1" || len(instances[0].Addrs) != 1 {
		t.Fatalf("unexpected browse result: %#v", instances)
	}
}

func TestBrowseReturnsInstancesCollectedBeforeCancellation(t *testing.T) {
	service := "_googlecast._tcp.local"
	instance := "Living Room._googlecast._tcp.local"
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true})
	if err := builder.StartQuestions(); err != nil {
		t.Fatal(err)
	}
	if err := builder.StartAnswers(); err != nil {
		t.Fatal(err)
	}
	if err := builder.PTRResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(service + "."), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET}, dnsmessage.PTRResource{PTR: dnsmessage.MustNewName(instance + ".")}); err != nil {
		t.Fatal(err)
	}
	packet, err := builder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	iface := &net.Interface{Name: "fixture0", Flags: net.FlagUp | net.FlagMulticast}
	ctx, cancel := context.WithCancel(context.Background())
	browser := &Browser{Window: time.Second, Interfaces: []*net.Interface{iface}, Listen: func(*net.Interface) (packetConn, error) {
		return &cancelAfterReadPacketConn{packet: packet, cancel: cancel}, nil
	}}
	instances, err := browser.Browse(ctx, service)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if len(instances) != 1 || instances[0].Instance != strings.ToLower(instance) {
		t.Fatalf("expected the collected instance, got %#v", instances)
	}
}

func TestParsePacketRejectsTruncatedDNS(t *testing.T) {
	if _, err := parsePacket([]byte{0x01}, "_googlecast._tcp.local"); err == nil || errors.Is(err, dnsmessage.ErrSectionDone) {
		t.Fatal("expected a malformed packet error")
	}
}

func TestDNSQueryUsesCanonicalWireName(t *testing.T) {
	packet := dnsQuery("_googlecast._tcp.local", dnsmessage.TypePTR)
	if len(packet) == 0 {
		t.Fatal("expected a DNS query")
	}
	var parser dnsmessage.Parser
	if _, err := parser.Start(packet); err != nil {
		t.Fatal(err)
	}
	question, err := parser.Question()
	if err != nil {
		t.Fatal(err)
	}
	if question.Name.String() != "_googlecast._tcp.local." {
		t.Fatalf("unexpected question name %q", question.Name.String())
	}
	if question.Type != dnsmessage.TypePTR || question.Class != dnsmessage.ClassINET {
		t.Fatalf("unexpected question: %#v", question)
	}
}

func TestParsePacketResolvesCompressedDNSServiceRecords(t *testing.T) {
	service := "_googlecast._tcp.local"
	instance := "Living Room._googlecast._tcp.local"
	host := "living-room.local"
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true})
	if err := builder.StartQuestions(); err != nil {
		t.Fatal(err)
	}
	if err := builder.StartAnswers(); err != nil {
		t.Fatal(err)
	}
	if err := builder.PTRResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(service + "."), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET}, dnsmessage.PTRResource{PTR: dnsmessage.MustNewName(instance + ".")}); err != nil {
		t.Fatal(err)
	}
	if err := builder.SRVResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(instance + "."), Type: dnsmessage.TypeSRV, Class: dnsmessage.ClassINET}, dnsmessage.SRVResource{Priority: 0, Weight: 0, Port: 8009, Target: dnsmessage.MustNewName(host + ".")}); err != nil {
		t.Fatal(err)
	}
	if err := builder.TXTResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(instance + "."), Type: dnsmessage.TypeTXT, Class: dnsmessage.ClassINET}, dnsmessage.TXTResource{TXT: []string{"id=cast-1", "md=Google TV", "fn=Living Room"}}); err != nil {
		t.Fatal(err)
	}
	unrelated := "Spotify desktop._spotify-connect._tcp.local"
	if err := builder.SRVResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(unrelated + "."), Type: dnsmessage.TypeSRV, Class: dnsmessage.ClassINET}, dnsmessage.SRVResource{Port: 62342, Target: dnsmessage.MustNewName("host.local.")}); err != nil {
		t.Fatal(err)
	}
	if err := builder.TXTResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(unrelated + "."), Type: dnsmessage.TypeTXT, Class: dnsmessage.ClassINET}, dnsmessage.TXTResource{TXT: []string{"CPath=/zc"}}); err != nil {
		t.Fatal(err)
	}
	if err := builder.StartAdditionals(); err != nil {
		t.Fatal(err)
	}
	if err := builder.AResource(dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(host + "."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}, dnsmessage.AResource{A: [4]byte{192, 168, 1, 42}}); err != nil {
		t.Fatal(err)
	}
	packet, err := builder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	items, err := parsePacket(packet, service)
	if err != nil {
		t.Fatal(err)
	}
	item, ok := items["living room._googlecast._tcp.local"]
	if !ok {
		t.Fatal("instance missing")
	}
	if item.Port != 8009 || item.Instance != "living room._googlecast._tcp.local" || item.Host != "living-room.local" || item.TXT["id"] != "cast-1" || len(item.Addrs) != 1 || item.Addrs[0].String() != "192.168.1.42" {
		t.Fatalf("unexpected parsed instance: %#v", item)
	}
	if _, ok := items[strings.ToLower(unrelated)]; ok {
		t.Fatal("unrelated DNS-SD instance was attributed to the requested service")
	}
}
