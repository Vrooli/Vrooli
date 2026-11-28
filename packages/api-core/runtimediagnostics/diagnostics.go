// Package runtimediagnostics provides bounded CPU profiling over an operator-
// private Unix socket. It never mounts a diagnostic route on the application API.
package runtimediagnostics

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	DirectoryEnv    = "VROOLI_RUNTIME_DIAGNOSTICS_DIR"
	MaxSeconds      = 30
	MaxProfileBytes = 16 << 20
	protocolVersion = "v1"
)

// Identity binds a socket and every collection request to one component launch.
type Identity struct {
	Scenario          string `json:"scenario"`
	Variant           string `json:"variant"`
	RuntimeInstanceID string `json:"runtime_instance_id"`
	Generation        string `json:"generation"`
	PID               int    `json:"pid"`
}

func (i Identity) Validate() error {
	if i.Scenario == "" || i.RuntimeInstanceID == "" || i.Generation == "" || i.PID <= 0 {
		return errors.New("runtime diagnostics requires a complete component launch identity")
	}
	return nil
}

// SocketPath is opaque to callers; PID reuse cannot address a different launch.
func SocketPath(directory string, identity Identity) string {
	sum := sha256.Sum256([]byte(identity.RuntimeInstanceID + "\x00" + identity.Generation))
	return filepath.Join(directory, strconv.Itoa(identity.PID)+"-"+hex.EncodeToString(sum[:6])+".sock")
}

type Capabilities struct {
	Version    string   `json:"version"`
	Identity   Identity `json:"identity"`
	MaxSeconds int      `json:"max_seconds"`
}

type profileRequest struct {
	Identity Identity `json:"identity"`
	Seconds  int      `json:"seconds"`
}

var profiling atomic.Bool

// StartFromEnv is disabled unless the control plane injects DirectoryEnv.
// Unsupported platforms and incomplete identity fail explicitly when enabled.
func StartFromEnv(ctx context.Context, getenv func(string) string) (func(), error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	dir := strings.TrimSpace(getenv(DirectoryEnv))
	if dir == "" {
		return func() {}, nil
	}
	return Listen(ctx, dir, Identity{
		Scenario: getenv("VROOLI_SCENARIO"), Variant: getenv("VROOLI_VARIANT"),
		RuntimeInstanceID: getenv("VROOLI_RUNTIME_INSTANCE_ID"),
		Generation:        getenv("VROOLI_COMPONENT_GENERATION"), PID: os.Getpid(),
	})
}

// Listen serves CPU profiles only. Collection follows the requesting connection;
// cancellation or runtime shutdown stops profiling and discards partial output.
func Listen(ctx context.Context, directory string, identity Identity) (func(), error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	if identity.PID != os.Getpid() {
		return nil, errors.New("runtime diagnostics can only identify its own process")
	}
	if !filepath.IsAbs(directory) {
		return nil, errors.New("runtime diagnostics directory must be absolute")
	}
	if err := ensurePrivateDirectory(directory); err != nil {
		return nil, err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: SocketPath(directory, identity), Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("listen for runtime diagnostics: %w", err)
	}
	if err := os.Chmod(listener.Addr().String(), 0o600); err != nil {
		_ = listener.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Capabilities{Version: protocolVersion, Identity: identity, MaxSeconds: MaxSeconds})
	})
	mux.HandleFunc("POST /v1/cpu", cpuHandler(identity))
	server := &http.Server{
		Handler: mux, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second,
		WriteTimeout: (MaxSeconds + 5) * time.Second, IdleTimeout: 2 * time.Second, MaxHeaderBytes: 4096,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve(listener) }()
	return func() {
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
		_ = server.Close()
		<-done
	}, nil
}

func cpuHandler(identity Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request profileRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid profile request", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(w, "invalid profile request", http.StatusBadRequest)
			return
		}
		if request.Identity != identity {
			http.Error(w, "component launch identity changed", http.StatusConflict)
			return
		}
		if request.Seconds < 1 || request.Seconds > MaxSeconds {
			http.Error(w, "profile duration must be between 1 and 30 seconds", http.StatusBadRequest)
			return
		}
		if !profiling.CompareAndSwap(false, true) {
			http.Error(w, "CPU profiling is already active", http.StatusConflict)
			return
		}
		defer profiling.Store(false)
		var output boundedProfile
		if err := pprof.StartCPUProfile(&output); err != nil {
			http.Error(w, "CPU profiling is already active", http.StatusConflict)
			return
		}
		timer := time.NewTimer(time.Duration(request.Seconds) * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-r.Context().Done():
		}
		pprof.StopCPUProfile()
		if r.Context().Err() != nil {
			return
		}
		if output.overflow {
			http.Error(w, "CPU profile exceeded size limit", http.StatusInsufficientStorage)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(output.Len()))
		_, _ = w.Write(output.Bytes())
	}
}

type boundedProfile struct {
	bytes.Buffer
	overflow bool
}

func (b *boundedProfile) Write(p []byte) (int, error) {
	if b.overflow || b.Len()+len(p) > MaxProfileBytes {
		b.overflow = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

// Client pins all HTTP calls to the specified local socket, without proxies.
type Client struct {
	http      *http.Client
	transport *http.Transport
}

func NewClient(socket string) *Client {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", socket)
	}}
	return &Client{http: &http.Client{Transport: transport, Timeout: (MaxSeconds + 5) * time.Second}, transport: transport}
}
func (c *Client) Close() { c.transport.CloseIdleConnections() }
func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	var capabilities Capabilities
	response, err := c.do(ctx, http.MethodGet, "/v1/capabilities", nil)
	if err != nil {
		return capabilities, err
	}
	defer response.Body.Close()
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&capabilities); err != nil {
		return capabilities, err
	}
	if capabilities.Version != protocolVersion || capabilities.MaxSeconds != MaxSeconds {
		return capabilities, errors.New("unsupported runtime diagnostics protocol")
	}
	return capabilities, capabilities.Identity.Validate()
}

func (c *Client) CPU(ctx context.Context, identity Identity, seconds int) ([]byte, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	if seconds < 1 || seconds > MaxSeconds {
		return nil, errors.New("profile duration must be between 1 and 30 seconds")
	}
	body, _ := json.Marshal(profileRequest{Identity: identity, Seconds: seconds})
	response, err := c.do(ctx, http.MethodPost, "/v1/cpu", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "application/octet-stream" {
		return nil, errors.New("invalid CPU profile response type")
	}
	profile, err := io.ReadAll(io.LimitReader(response.Body, MaxProfileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(profile) == 0 || len(profile) > MaxProfileBytes {
		return nil, errors.New("invalid CPU profile response size")
	}
	return profile, nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, "http://runtime-diagnostics"+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		message, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("runtime diagnostics HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	return response, nil
}
