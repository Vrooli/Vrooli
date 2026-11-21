package activityedge

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func serveControlSocket(path string, h *Handlers) (*http.Server, net.Listener, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, nil, err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, nil, fmt.Errorf("control path exists and is not a socket")
		}
		return nil, nil, fmt.Errorf("control socket already exists")
	} else if !os.IsNotExist(err) {
		return nil, nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
			return
		}
		h.trackerMu.Lock()
		ready := h.tracker != nil
		h.trackerMu.Unlock()
		if !ready {
			http.Error(w, "tracker unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/drain", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		if err := h.Drain(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusRequestTimeout)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		h.Resume()
		w.WriteHeader(http.StatusNoContent)
	})
	return &http.Server{Handler: mux}, listener, nil
}

// ControlSocketRequest sends one lifecycle command over the private local
// activity-edge control socket. It is exported for the resource owner and
// intentionally accepts only the two fixed lifecycle actions.
func ControlSocketRequest(ctx context.Context, path, action string) error {
	if action != "drain" && action != "resume" {
		return fmt.Errorf("unsupported activity-edge control action %q", action)
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err != nil {
		return fmt.Errorf("connect activity-edge control socket: %w", err)
	}
	defer conn.Close()
	stopCancel := applyControlContext(ctx, conn)
	defer stopCancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://activity-edge/"+action, nil)
	if err != nil {
		return err
	}
	if err := request.Write(conn); err != nil {
		return fmt.Errorf("write activity-edge control request: %w", err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		return fmt.Errorf("read activity-edge control response: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("activity-edge %s returned %s", action, response.Status)
	}
	return nil
}

func ControlSocketReady(ctx context.Context, path string) error {
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err != nil {
		return err
	}
	defer conn.Close()
	stopCancel := applyControlContext(ctx, conn)
	defer stopCancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://activity-edge/ready", nil)
	if err != nil {
		return err
	}
	if err := request.Write(conn); err != nil {
		return err
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("activity-edge control readiness returned %s", response.Status)
	}
	return nil
}

func applyControlContext(ctx context.Context, conn net.Conn) func() bool {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	return context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
}
