package mdns

import (
	"encoding/binary"
	"net"
	"testing"
)

func dnsQuestion(name string, questionType uint16) []byte {
	packet := make([]byte, 12)
	binary.BigEndian.PutUint16(packet[4:6], 1)
	packet = append(packet, encodeName(name)...)
	var question [4]byte
	binary.BigEndian.PutUint16(question[:2], questionType)
	binary.BigEndian.PutUint16(question[2:], 1)
	return append(packet, question[:]...)
}

func TestIsDNSQueryForService(t *testing.T) {
	service := "_vrooli-bridge._tcp.local."
	if !isDNSQueryFor(dnsQuestion(service, 12), service) {
		t.Fatal("expected PTR query for advertised service to match")
	}
	if isDNSQueryFor(dnsQuestion("_other._tcp.local.", 12), service) {
		t.Fatal("unexpected service query match")
	}
	if isDNSQueryFor([]byte{0, 1, 2}, service) {
		t.Fatal("truncated packet matched")
	}
}

func TestNormalizeInstanceNameExpandsShortLabel(t *testing.T) {
	if got := normalizeInstanceName("vrooli-bridge", "_vrooli-bridge._tcp.local"); got != "vrooli-bridge._vrooli-bridge._tcp.local." {
		t.Fatalf("short instance normalized to %q", got)
	}
	if got := normalizeInstanceName("vrooli-bridge._vrooli-bridge._tcp.local.", "_vrooli-bridge._tcp.local."); got != "vrooli-bridge._vrooli-bridge._tcp.local." {
		t.Fatalf("qualified instance changed to %q", got)
	}
}

func TestEncodeResponseIncludesURLAndAddress(t *testing.T) {
	packet := encodeResponse(
		"_vrooli-bridge._tcp.local.",
		"vrooli-bridge._vrooli-bridge._tcp.local.",
		"vrooli-bridge.local.",
		18767,
		net.ParseIP("192.0.2.7"),
		"https://bridge.example.test:18767",
		map[string]string{"version": "v1"},
	)
	if len(packet) < 12 || binary.BigEndian.Uint16(packet[2:4]) != 0x8400 {
		t.Fatalf("not an authoritative DNS response: %x", packet[:min(len(packet), 12)])
	}
	for _, want := range []string{"_vrooli-bridge", "vrooli-bridge", "https://bridge.example.test:18767", "version=v1"} {
		if !containsBytes(packet, []byte(want)) {
			t.Fatalf("response did not contain %q", want)
		}
	}
	if !containsBytes(packet, net.ParseIP("192.0.2.7").To4()) {
		t.Fatal("response did not contain the advertised IPv4 address")
	}
}

func containsBytes(haystack, needle []byte) bool {
	for start := 0; start+len(needle) <= len(haystack); start++ {
		match := true
		for i := range needle {
			if haystack[start+i] != needle[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
