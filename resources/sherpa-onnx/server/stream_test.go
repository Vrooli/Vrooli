package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
)

type contractStreaming struct{}

func (contractStreaming) NewStream() (STTStream, error) { return contractStream{}, nil }
func (contractStreaming) Close()                        {}

type contractStream struct{}

func (contractStream) AcceptPCM([]byte) ([]STTEvent, error) {
	return []STTEvent{{Text: "hello"}}, nil
}
func (contractStream) Finish() ([]STTEvent, error) {
	return []STTEvent{{Text: "hello world", Final: true, EndSample: 16000}}, nil
}
func (contractStream) Close() {}

func TestStreamingWebSocketContract(t *testing.T) {
	server := httptest.NewServer(newHandlerWithEncoderAndStreaming(fakeTTS{}, fakeEncoder{}, contractStreaming{}))
	defer server.Close()
	addr := strings.TrimPrefix(server.URL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	fmt.Fprintf(conn, "GET /v1/stream HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n", addr)
	handshake, err := readUntilHeaderEnd(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(handshake, "HTTP/1.1 101") {
		t.Fatalf("handshake = %q", handshake)
	}
	writeMaskedFrame(t, conn, 0x1, []byte(`{"type":"start","sample_rate":16000,"language":"en"}`))
	if got := readJSONFrame(t, reader); !strings.Contains(got, `"type":"ready"`) {
		t.Fatalf("ready = %s", got)
	}
	writeMaskedFrame(t, conn, 0x2, []byte{0, 0})
	if got := readJSONFrame(t, reader); !strings.Contains(got, `"type":"partial"`) {
		t.Fatalf("partial = %s", got)
	}
	if got := readJSONFrame(t, reader); !strings.Contains(got, `"type":"processed"`) {
		t.Fatalf("processed = %s", got)
	}
	writeMaskedFrame(t, conn, 0x1, []byte(`{"type":"end"}`))
	if got := readJSONFrame(t, reader); !strings.Contains(got, `"type":"segment"`) {
		t.Fatalf("segment = %s", got)
	}
	if got := readJSONFrame(t, reader); !strings.Contains(got, `"type":"done"`) {
		t.Fatalf("done = %s", got)
	}
}

func readUntilHeaderEnd(r *bufio.Reader) (string, error) {
	var data []byte
	for {
		line, err := r.ReadBytes('\n')
		data = append(data, line...)
		if err != nil {
			return "", err
		}
		if len(data) >= 4 && string(data[len(data)-4:]) == "\r\n\r\n" {
			return string(data), nil
		}
	}
}

func writeMaskedFrame(t *testing.T, conn net.Conn, opcode byte, payload []byte) {
	t.Helper()
	mask := [4]byte{1, 2, 3, 4}
	frame := []byte{0x80 | opcode}
	switch {
	case len(payload) < 126:
		frame = append(frame, byte(0x80|len(payload)))
	case len(payload) <= 0xffff:
		frame = append(frame, 0x80|126, byte(len(payload)>>8), byte(len(payload)))
	default:
		t.Fatal("test payload is too large")
	}
	frame = append(frame, mask[:]...)
	for i, value := range payload {
		frame = append(frame, value^mask[i%4])
	}
	if _, err := conn.Write(frame); err != nil {
		t.Fatal(err)
	}
}

func readJSONFrame(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	first, err := r.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	if first&0x0f != 0x1 || second&0x80 != 0 {
		t.Fatalf("unexpected server frame: %#x %#x", first, second)
	}
	length := int(second & 0x7f)
	if length == 126 {
		var extended [2]byte
		_, err = io.ReadFull(r, extended[:])
		length = int(binary.BigEndian.Uint16(extended[:]))
	}
	if err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		t.Fatal(err)
	}
	return string(payload)
}
