// Responsibility: forward a sealed one-shot kernel's owner calls without exposing bearer proofs to Python.
package programs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

type peerContextKey struct{}

// ErrAuthorityEffectUncertain means a forwarded owner effect may have reached its
// owner without an observed completion. Reconcile owner state before retrying.
var ErrAuthorityEffectUncertain = errors.New("owner effect is uncertain; inspect owner state before retry")

type limitedListener struct {
	net.Listener
	slots chan struct{}
	done  chan struct{}
	once  sync.Once
}

type limitedConn struct {
	net.Conn
	release func()
	once    sync.Once
}

func (c *limitedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

func (l *limitedListener) Accept() (net.Conn, error) {
	select {
	case l.slots <- struct{}{}:
	case <-l.done:
		return nil, net.ErrClosed
	}
	c, err := l.Listener.Accept()
	if err != nil {
		<-l.slots
		return nil, err
	}
	return &limitedConn{Conn: c, release: func() { <-l.slots }}, nil
}

func (l *limitedListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return l.Listener.Close()
}

type authorityRelay struct {
	listener     net.Listener
	server       *http.Server
	dir          string
	path         string
	target       *url.URL
	owner        context.Context
	pid          uint32
	processFD    int
	processMu    sync.RWMutex
	closeOnce    sync.Once
	sessionID    string
	programID    string
	flightMu     sync.Mutex
	closing      bool
	inFlight     int
	drained      chan struct{}
	uncertain    bool
	drainTimeout time.Duration
}

func strictObject(data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	open, err := decoder.Token()
	if err != nil || open != json.Delim('{') {
		return nil, errors.New("object required")
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, errors.New("invalid field")
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, errors.New("duplicate field")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[name] = value
	}
	if close, err := decoder.Token(); err != nil || close != json.Delim('}') {
		return nil, errors.New("object not closed")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("trailing value")
	}
	return fields, nil
}

func newAuthorityRelay(ctx context.Context, bridgeURL, sessionID, programID string) (*authorityRelay, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("verified one-shot authority relay unavailable on this platform")
	}
	target, err := url.Parse(bridgeURL)
	if err != nil || target.Scheme != "http" || target.User != nil || net.ParseIP(target.Hostname()) == nil || !net.ParseIP(target.Hostname()).IsLoopback() || target.Port() == "" {
		return nil, errors.New("verified one-shot authority relay requires a configured loopback owner")
	}
	dir, err := os.MkdirTemp("", "prt-relay-")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "owner.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	if err = os.Chmod(path, 0o600); err != nil {
		_ = listener.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return &authorityRelay{listener: &limitedListener{Listener: listener, slots: make(chan struct{}, 16), done: make(chan struct{})}, dir: dir, path: path, target: target, owner: ctx, sessionID: sessionID, programID: programID, processFD: -1, drained: make(chan struct{}), drainTimeout: 5 * time.Second}, nil
}

func (a *authorityRelay) beginFlight() bool {
	a.flightMu.Lock()
	defer a.flightMu.Unlock()
	if a.closing {
		return false
	}
	a.inFlight++
	return true
}

func (a *authorityRelay) endFlight(uncertain bool) {
	a.flightMu.Lock()
	defer a.flightMu.Unlock()
	a.uncertain = a.uncertain || uncertain
	a.inFlight--
	if a.closing && a.inFlight == 0 {
		close(a.drained)
	}
}

func (a *authorityRelay) start(pid int) error {
	if pid <= 0 {
		return errors.New("kernel process identity unavailable")
	}
	a.pid = uint32(pid)
	fd, err := openKernelProcess(pid)
	if err != nil {
		return errors.New("live kernel process identity unavailable")
	}
	a.processFD = fd
	a.server = &http.Server{ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, MaxHeaderBytes: 8192, Handler: http.HandlerFunc(a.serve), ConnContext: func(ctx context.Context, c net.Conn) context.Context {
		limited, ok := c.(*limitedConn)
		if !ok {
			return ctx
		}
		uc, ok := limited.Conn.(*net.UnixConn)
		if !ok {
			return ctx
		}
		pid, err := localprincipal.PeerPID(uc)
		if err != nil {
			return ctx
		}
		return context.WithValue(ctx, peerContextKey{}, pid)
	}}
	go func() { _ = a.server.Serve(a.listener) }()
	return nil
}

func (a *authorityRelay) serve(w http.ResponseWriter, r *http.Request) {
	pid, ok := r.Context().Value(peerContextKey{}).(uint32)
	a.processMu.RLock()
	alive := kernelProcessAlive(a.processFD)
	a.processMu.RUnlock()
	if !ok || pid != a.pid || !alive {
		http.Error(w, "kernel peer identity denied", http.StatusForbidden)
		return
	}
	path := r.URL.Path
	allowed := map[string]bool{
		"/internal/program-runtime/bindings/execute": true,
		"/internal/program-runtime/tasks/resolve":    true,
		"/internal/program-runtime/tasks/begin":      true,
		"/internal/program-runtime/tasks/start":      true,
		"/internal/program-runtime/tasks/complete":   true,
		"/internal/program-runtime/agent/start":      true,
		"/internal/program-runtime/agent/collect":    true,
	}
	if r.Method != http.MethodPost || !allowed[path] || r.URL.RawQuery != "" || r.URL.RawPath != "" {
		http.Error(w, "owner route not allowed", http.StatusForbidden)
		return
	}
	ownerCtx, cancel := context.WithCancel(a.owner)
	stop := context.AfterFunc(r.Context(), cancel)
	defer func() { stop(); cancel() }()
	target := *a.target
	target.Path, target.RawPath, target.RawQuery = path, "", r.URL.RawQuery
	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024+1))
	if err != nil || len(body) > 2*1024*1024 {
		http.Error(w, "owner request exceeds limit", http.StatusRequestEntityTooLarge)
		return
	}
	if r.Method == http.MethodPost {
		fields, err := strictObject(body)
		if err != nil {
			http.Error(w, "invalid owner request", http.StatusBadRequest)
			return
		}
		for name := range fields {
			if (strings.EqualFold(name, "session_id") && name != "session_id") || (strings.EqualFold(name, "program_id") && name != "program_id") || (strings.EqualFold(name, "binding_id") && name != "binding_id") {
				http.Error(w, "owner identity aliases forbidden", http.StatusBadRequest)
				return
			}
		}
		for name, expected := range map[string]string{"session_id": a.sessionID, "program_id": a.programID} {
			if value, exists := fields[name]; exists {
				var actual string
				if json.Unmarshal(value, &actual) != nil || actual != expected {
					http.Error(w, "owner identity mismatch", http.StatusForbidden)
					return
				}
			} else {
				http.Error(w, "owner identity missing", http.StatusForbidden)
				return
			}
		}
		if path == "/internal/program-runtime/bindings/execute" {
			var binding string
			if json.Unmarshal(fields["binding_id"], &binding) != nil || (binding != "web-console/session/native-thread-state" && binding != "web-console/session/native-usage") {
				http.Error(w, "binding is not allowed by this relay", http.StatusForbidden)
				return
			}
		}
	}
	request, err := http.NewRequestWithContext(ownerCtx, r.Method, target.String(), bytes.NewReader(body))
	if err != nil {
		http.Error(w, "invalid owner request", http.StatusBadRequest)
		return
	}
	if !a.beginFlight() {
		http.Error(w, "checkpoint relay is closing", http.StatusServiceUnavailable)
		return
	}
	uncertain := false
	readOnly := path == "/internal/program-runtime/bindings/execute"
	defer func() { a.endFlight(uncertain) }()
	request.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	if database.IsTestMode(a.owner) {
		request.Header.Set("X-Vrooli-Test-Mode", "1")
	}
	if token := provenance.ForwardedIdentityToken(a.owner); token != "" {
		request.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	}
	if strings.HasPrefix(path, "/internal/program-runtime/agent/") {
		if token := provenance.ForwardedOperationToken(a.owner); token != "" {
			request.Header.Set(cliutil.HeaderAgentOperationToken, token)
		}
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}, Timeout: 6 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		uncertain = !readOnly
		http.Error(w, "owner request unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if contentType := response.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 16*1024*1024+1))
	if err != nil || len(data) > 16*1024*1024 {
		uncertain = !readOnly
		http.Error(w, "owner response unavailable or too large", http.StatusBadGateway)
		return
	}
	if !readOnly && (response.StatusCode < 200 || response.StatusCode >= 300) {
		// Only this exact owner-produced resolve disposition is proven before
		// any task or declaration write. All other errors remain uncertain.
		var noEffect struct {
			Reason string `json:"reason"`
			Effect string `json:"effect"`
		}
		provenAbsent := path == "/internal/program-runtime/tasks/resolve" && response.StatusCode == http.StatusConflict && json.Unmarshal(data, &noEffect) == nil && noEffect.Reason == "operation_not_registered" && noEffect.Effect == "none"
		uncertain = !provenAbsent
	}
	w.WriteHeader(response.StatusCode)
	if _, err := w.Write(data); err != nil || r.Context().Err() != nil {
		uncertain = !readOnly
	}
}

func (a *authorityRelay) close() error {
	a.closeOnce.Do(func() {
		a.flightMu.Lock()
		a.closing = true
		if a.inFlight == 0 {
			close(a.drained)
		}
		a.flightMu.Unlock()
		_ = a.listener.Close()
		timer := time.NewTimer(a.drainTimeout)
		select {
		case <-a.drained:
		case <-timer.C:
			a.flightMu.Lock()
			a.uncertain = true
			a.flightMu.Unlock()
		}
		timer.Stop()
		if a.server != nil {
			_ = a.server.Close()
		}
		a.processMu.Lock()
		closeKernelProcess(a.processFD)
		a.processFD = -1
		a.processMu.Unlock()
		_ = os.RemoveAll(a.dir)
	})
	a.flightMu.Lock()
	defer a.flightMu.Unlock()
	if a.uncertain {
		return ErrAuthorityEffectUncertain
	}
	return nil
}
