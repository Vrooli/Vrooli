package activityedge

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestControlSocketDrainsAndResumesLiveTracker(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "ae-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "edge.sock")
	h := &Handlers{}
	srv, listener, err := serveControlSocket(path, h)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = srv.Close()
		_ = listener.Close()
		_ = os.Remove(path)
	}()
	tr := &tracker{drained: make(chan struct{})}
	close(tr.drained)
	h.tracker = tr
	go func() { _ = srv.Serve(listener) }()
	if err := ControlSocketReady(context.Background(), path); err != nil {
		t.Fatalf("control socket readiness: %v", err)
	}

	if err := ControlSocketRequest(context.Background(), path, "drain"); err != nil {
		t.Fatalf("drain through lifecycle socket: %v", err)
	}
	if tr.begin() {
		t.Fatal("request entered after lifecycle drain")
	}
	if err := ControlSocketRequest(context.Background(), path, "resume"); err != nil {
		t.Fatalf("resume through lifecycle socket: %v", err)
	}
	if !tr.begin() {
		t.Fatal("request remained blocked after lifecycle resume")
	}
	tr.end()
}

func TestDrainWaitsForSequencedIdleReport(t *testing.T) {
	reported := make(chan string, 2)
	tr := &tracker{drained: make(chan struct{}), markActive: func() { reported <- activityActive }, markIdle: func() { reported <- activityIdle }}
	close(tr.drained)
	if !tr.begin() {
		t.Fatal("initial request was rejected")
	}
	tr.end()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := tr.drain(ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if first, second := <-reported, <-reported; first != activityActive || second != activityIdle {
		t.Fatalf("reports = %q then %q, want active then idle", first, second)
	}
	_, active := tr.snapshot()
	if active {
		t.Fatal("tracker remained active after drain")
	}
}

func TestControlSocketRefusesRegularFile(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "ae-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "edge.sock")
	if err := os.WriteFile(path, []byte("operator file"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := serveControlSocket(path, &Handlers{}); err == nil {
		t.Fatal("regular file at control path was accepted")
	}
}

func TestControlSocketRequestHonorsCancellationWhileReadingResponse(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "ae-cancel-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "edge.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	release := make(chan struct{})
	serverDone := make(chan struct{})
	defer close(release)
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = http.ReadRequest(bufio.NewReader(conn))
		close(accepted)
		// Simulate a stalled owner response; the client context must still bound it.
		<-release
	}()
	ctx, cancel := context.WithCancel(context.Background())
	requestDone := make(chan error, 1)
	go func() { requestDone <- ControlSocketRequest(ctx, path, "drain") }()
	select {
	case <-accepted:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("control request was not accepted")
	}
	started := time.Now()
	cancel()
	select {
	case err := <-requestDone:
		if err == nil {
			t.Fatal("canceled control request succeeded")
		}
		if time.Since(started) > 500*time.Millisecond {
			t.Fatalf("canceled control request took too long: %s", time.Since(started))
		}
	case <-time.After(time.Second):
		t.Fatal("control request ignored context cancellation")
	}
}
