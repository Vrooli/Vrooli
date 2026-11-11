package health

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestRunChecksHTTPHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result, err := RunChecks(context.Background(), []manifestpkg.ResourceHealthCheck{{
		Type:   "http",
		Target: srv.URL,
	}}, Config{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("RunChecks: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("expected healthy result: %+v", result)
	}
}

func TestRunCheckOpenRGBSDKV6NegotiatesWithoutDiscovery(t *testing.T) {
	// Captured from the pinned OpenRGB 1.1 release in the hardware-free
	// release fixture. The listener ACKs after the version and name replies.
	result, err := runTestOpenRGBHandshake(t,
		testOpenRGBFrame{packetID: 40, body: []byte{6, 0, 0, 0}},
		testOpenRGBFrame{packetID: 51, body: []byte("OpenRGB 1.1\x00")},
		testOpenRGBFrame{packetID: 10, body: []byte{40, 0, 0, 0, 0, 0, 0, 0}},
	)
	if err != nil || !result.Healthy {
		t.Fatalf("OpenRGB protocol health = %#v, %v", result, err)
	}
}

func TestRunCheckOpenRGBSDKV6RequiresCorrelatedNegotiationAck(t *testing.T) {
	version := testOpenRGBFrame{packetID: 40, body: []byte{6, 0, 0, 0}}
	serverName := testOpenRGBFrame{packetID: 51, body: []byte("OpenRGB 1.1\x00")}
	ack := testOpenRGBFrame{packetID: 10, body: []byte{40, 0, 0, 0, 0, 0, 0, 0}}
	for _, tt := range []struct {
		name   string
		frames []testOpenRGBFrame
		want   string
	}{
		{"missing ACK", []testOpenRGBFrame{version, serverName}, "protocol negotiation failed"},
		{"premature ACK", []testOpenRGBFrame{version, ack, serverName}, "ACK arrived before both replies"},
		{"truncated ACK body", []testOpenRGBFrame{version, serverName, {packetID: 10, body: ack.body[:4]}}, "ACK is malformed"},
		{"wrong device", []testOpenRGBFrame{version, serverName, {packetID: 10, deviceID: 1, body: ack.body}}, "ACK is malformed"},
		{"wrong packet", []testOpenRGBFrame{version, serverName, {packetID: 10, body: make([]byte, 8)}}, "ACK is malformed"},
		{"nonzero status", []testOpenRGBFrame{version, serverName, {packetID: 10, body: []byte{40, 0, 0, 0, 2, 0, 0, 0}}}, "ACK returned status 2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := runTestOpenRGBHandshake(t, tt.frames...)
			if err != nil || result.Healthy || !strings.Contains(result.Message, tt.want) {
				t.Fatalf("invalid ACK health = %#v, %v; want %q", result, err, tt.want)
			}
		})
	}
}

func TestRunCheckOpenRGBSDKV6AllowsInterleavedNotification(t *testing.T) {
	result, err := runTestOpenRGBHandshake(t,
		testOpenRGBFrame{packetID: 40, body: []byte{6, 0, 0, 0}},
		testOpenRGBFrame{packetID: 100}, // device-list update, unsolicited
		testOpenRGBFrame{packetID: 1150, deviceID: 17, body: make([]byte, openRGBSDKMaxHandshakeBody+1)},
		testOpenRGBFrame{packetID: 51, body: []byte("OpenRGB 1.1\x00")},
		testOpenRGBFrame{packetID: 10, body: []byte{40, 0, 0, 0, 0, 0, 0, 0}},
	)
	if err != nil || !result.Healthy {
		t.Fatalf("interleaved SDK notification health = %#v, %v", result, err)
	}
}

func TestReadOpenRGBHandshakeFrameRejectsOversizedBodies(t *testing.T) {
	for _, tt := range []struct {
		packetID uint32
		size     uint32
	}{
		{40, openRGBSDKMaxHandshakeBody + 1},
		{1150, openRGBSDKMaxUpdateBody + 1},
	} {
		var frame bytes.Buffer
		if err := writeTestOpenRGBFrame(&frame, tt.packetID, nil); err != nil {
			t.Fatal(err)
		}
		header := frame.Bytes()
		binary.LittleEndian.PutUint32(header[12:16], tt.size)
		if _, _, _, err := readOpenRGBSDKHandshakeFrame(bytes.NewReader(header)); err == nil || !strings.Contains(err.Error(), "bound") {
			t.Fatalf("packet %d size %d = %v, want size bound", tt.packetID, tt.size, err)
		}
	}
}

type testOpenRGBFrame struct {
	packetID uint32
	deviceID uint32
	body     []byte
}

func runTestOpenRGBHandshake(t *testing.T, frames ...testOpenRGBFrame) (Result, error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		var request [20]byte
		if _, err := io.ReadFull(conn, request[:]); err != nil {
			serverErr <- err
			return
		}
		if !bytes.Equal(request[:], []byte("ORGB\x00\x00\x00\x00\x28\x00\x00\x00\x04\x00\x00\x00\x06\x00\x00\x00")) {
			serverErr <- errors.New("health probe sent a packet other than protocol negotiation")
			return
		}
		var replies bytes.Buffer
		for _, frame := range frames {
			if err := writeTestOpenRGBFrameWithDevice(&replies, frame.packetID, frame.deviceID, frame.body); err != nil {
				serverErr <- err
				return
			}
		}
		if _, err := replies.WriteTo(conn); err != nil {
			serverErr <- err
			return
		}
		if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
			serverErr <- err
			return
		}
		var extra [1]byte
		if n, err := conn.Read(extra[:]); n != 0 || (!errors.Is(err, io.EOF) && !errors.Is(err, syscall.ECONNRESET)) {
			serverErr <- errors.New("health probe did not close the connection after negotiation")
			return
		}
		serverErr <- nil
	}()

	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{Type: "openrgb-sdk-v6", Target: listener.Addr().String(), TimeoutSeconds: 2}, Config{})
	if fixtureErr := <-serverErr; fixtureErr != nil {
		t.Fatalf("fake OpenRGB SDK server: %v", fixtureErr)
	}
	return result, err
}

func TestRunCheckOpenRGBSDKV6RejectsOldProtocol(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var request [openRGBSDKHeaderSize + 4]byte
		if _, err := io.ReadFull(conn, request[:]); err != nil {
			return
		}
		_ = writeTestOpenRGBFrame(conn, openRGBProtocolRequest, uint32Bytes(5))
		_ = writeTestOpenRGBFrame(conn, openRGBServerNamePacket, []byte("OpenRGB old fixture\x00"))
	}()

	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{Type: "openrgb-sdk-v6", Target: listener.Addr().String()}, Config{})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Healthy || !strings.Contains(result.Message, "below required version 6") {
		t.Fatalf("old protocol result = %#v, want unhealthy protocol mismatch", result)
	}
}

func TestRunCheckOpenRGBSDKV6RefusesNonLoopbackTarget(t *testing.T) {
	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{Type: "openrgb-sdk-v6", Target: "192.0.2.1:6742"}, Config{})
	if err == nil || result.Healthy {
		t.Fatalf("non-loopback SDK health target result=%#v err=%v, want rejection", result, err)
	}
}

func TestRunCheckOpenRGBSDKV6RejectsMalformedServerName(t *testing.T) {
	for name, tc := range map[string]struct {
		device uint32
		body   []byte
	}{
		"empty name":          {body: []byte{0}},
		"missing terminator":  {body: []byte("OpenRGB")},
		"embedded terminator": {body: []byte("Open\x00RGB\x00")},
		"invalid utf8":        {body: []byte{0xff, 0}},
		"unexpected device":   {device: 1, body: []byte("OpenRGB\x00")},
	} {
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			serverErr := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					serverErr <- err
					return
				}
				defer conn.Close()
				var request [openRGBSDKHeaderSize + 4]byte
				if _, err := io.ReadFull(conn, request[:]); err != nil {
					serverErr <- err
					return
				}
				if err := writeTestOpenRGBFrame(conn, openRGBProtocolRequest, uint32Bytes(openRGBProtocolVersion)); err != nil {
					serverErr <- err
					return
				}
				serverErr <- writeTestOpenRGBFrameWithDevice(conn, openRGBServerNamePacket, tc.device, tc.body)
			}()
			result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{Type: "openrgb-sdk-v6", Target: listener.Addr().String()}, Config{})
			if err != nil || result.Healthy || !strings.Contains(result.Message, "server-name reply is malformed") {
				t.Fatalf("malformed server-name result=%#v err=%v", result, err)
			}
			if err := <-serverErr; err != nil {
				t.Fatalf("fake OpenRGB SDK server: %v", err)
			}
		})
	}
}

func uint32Bytes(value uint32) []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)
	return data
}

func writeTestOpenRGBFrame(w io.Writer, packetID uint32, body []byte) error {
	return writeTestOpenRGBFrameWithDevice(w, packetID, 0, body)
}

func writeTestOpenRGBFrameWithDevice(w io.Writer, packetID, deviceID uint32, body []byte) error {
	frame := make([]byte, openRGBSDKHeaderSize+len(body))
	copy(frame[:4], "ORGB")
	binary.LittleEndian.PutUint32(frame[4:8], deviceID)
	binary.LittleEndian.PutUint32(frame[8:12], packetID)
	binary.LittleEndian.PutUint32(frame[12:16], uint32(len(body)))
	copy(frame[openRGBSDKHeaderSize:], body)
	return writeOpenRGBSDKBytes(w, frame)
}

// Scenario: a failing liveness check degrades the resource without downing it.
//
// The control plane executes liveness checks. A failing one means the resource
// answers requests but is not meeting its contract — running below its declared
// accelerator backend is the canonical case — so serving stays true while
// healthy becomes false.
func TestRunChecksDegradeOnFailingLiveness(t *testing.T) {
	// Given a resource whose readiness passes and whose liveness fails
	result, err := RunChecks(context.Background(), []manifestpkg.ResourceHealthCheck{
		{Kind: "readiness", Type: "command", Command: []string{"true"}},
		{Kind: "liveness", Type: "command", Command: []string{"not-a-real-command"}},
	}, Config{})
	// When the combined verdict is read
	if err != nil {
		t.Fatalf("RunChecks: %v", err)
	}

	// Then it is not healthy, because the contract is not met
	if result.Healthy {
		t.Fatalf("a failing liveness check must make the result unhealthy: %+v", result)
	}
	// And it is still serving, because the resource answers requests
	if !result.Serving {
		t.Fatalf("a failing liveness check must not make a serving resource look stopped: %+v", result)
	}
	// And the message names the check that failed
	if result.LivenessFailed == "" {
		t.Fatalf("LivenessFailed is empty; the operator must be told which contract was not met: %+v", result)
	}
}

func TestRunCheckHTTPRendersEnvironmentTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{Type: "http", Target: "${HEALTH_ENDPOINT}", ExpectedStatus: []int{http.StatusNoContent}}, Config{Env: []string{"HEALTH_ENDPOINT=" + srv.URL}})
	if err != nil || !result.Healthy {
		t.Fatalf("rendered HTTP health = %#v, %v", result, err)
	}
}

func TestRunCheckHTTPRendersHeadersFromEnvironment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{
		Type:           "http",
		Target:         srv.URL,
		Headers:        map[string]string{"Authorization": "Bearer ${API_KEY}"},
		ExpectedStatus: []int{http.StatusNoContent},
	}, Config{Env: []string{"API_KEY=test-key"}})
	if err != nil || !result.Healthy {
		t.Fatalf("authenticated HTTP health = %#v, %v", result, err)
	}
}

func TestRunCheckCommandFailureReturnsUnhealthy(t *testing.T) {
	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{
		Type:    "command",
		Command: []string{"fake", "cmd"},
	}, Config{
		Runner: func(context.Context, *exec.Cmd) ([]byte, error) {
			return nil, errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Healthy {
		t.Fatalf("expected unhealthy result: %+v", result)
	}
}

func TestRunCheckCommandLeavesCombinedOutputUnset(t *testing.T) {
	var captured *exec.Cmd

	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{
		Type:    "command",
		Command: []string{"claude", "--version"},
	}, Config{
		Runner: func(_ context.Context, cmd *exec.Cmd) ([]byte, error) {
			captured = cmd
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("expected healthy result: %+v", result)
	}
	if captured == nil {
		t.Fatal("expected runner to receive command")
	}
	if captured.Stdout != nil || captured.Stderr != nil {
		t.Fatalf("expected command health check to leave stdout/stderr unset, got stdout=%v stderr=%v", captured.Stdout, captured.Stderr)
	}
}

func TestCommandHealthUsesCheckContextForExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := false
	result, err := RunCheck(ctx, manifestpkg.ResourceHealthCheck{Type: "command", Command: []string{"test-only-command"}}, Config{Runner: func(checkCtx context.Context, cmd *exec.Cmd) ([]byte, error) {
		called = true
		if cmd.Cancel == nil {
			t.Error("command has no context-owned cancellation")
		}
		cancel()
		if !errors.Is(checkCtx.Err(), context.Canceled) {
			t.Error("caller cancellation detached from check")
		}
		return nil, checkCtx.Err()
	}})
	if !called || err != nil || result.Healthy {
		t.Fatalf("health verdict changed: called=%v result=%+v err=%v", called, result, err)
	}
}
