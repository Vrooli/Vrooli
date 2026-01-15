package preflight

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	bundleruntime "scenario-to-desktop-runtime"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// SupervisorFactory creates runtime supervisors for preflight sessions.
type SupervisorFactory func(manifest *bundlemanifest.Manifest, bundleRoot, appData string) (*bundleruntime.Supervisor, error)

// InMemorySessionStore is the default in-memory implementation of SessionStore.
type InMemorySessionStore struct {
	sessions          map[string]*Session
	mux               sync.Mutex
	createSupervisor  SupervisorFactory
	readFileWithRetry func(path string, timeout time.Duration) ([]byte, error)
	readPortFile      func(path string, timeout time.Duration) (int, error)
	waitForHealth     func(client *http.Client, baseURL string, timeout time.Duration) error
}

// InMemorySessionStoreOption configures an InMemorySessionStore.
type InMemorySessionStoreOption func(*InMemorySessionStore)

// WithSupervisorFactory sets a custom supervisor factory.
func WithSupervisorFactory(factory SupervisorFactory) InMemorySessionStoreOption {
	return func(s *InMemorySessionStore) {
		s.createSupervisor = factory
	}
}

// WithFileReader sets a custom file reader for testing.
func WithFileReader(reader func(path string, timeout time.Duration) ([]byte, error)) InMemorySessionStoreOption {
	return func(s *InMemorySessionStore) {
		s.readFileWithRetry = reader
	}
}

// WithPortReader sets a custom port reader for testing.
func WithPortReader(reader func(path string, timeout time.Duration) (int, error)) InMemorySessionStoreOption {
	return func(s *InMemorySessionStore) {
		s.readPortFile = reader
	}
}

// WithHealthWaiter sets a custom health waiter for testing.
func WithHealthWaiter(waiter func(client *http.Client, baseURL string, timeout time.Duration) error) InMemorySessionStoreOption {
	return func(s *InMemorySessionStore) {
		s.waitForHealth = waiter
	}
}

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore(opts ...InMemorySessionStoreOption) *InMemorySessionStore {
	s := &InMemorySessionStore{
		sessions:          make(map[string]*Session),
		createSupervisor:  defaultCreateSupervisor,
		readFileWithRetry: defaultReadFileWithRetry,
		readPortFile:      defaultReadPortFile,
		waitForHealth:     defaultWaitForHealth,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func defaultCreateSupervisor(manifest *bundlemanifest.Manifest, bundleRoot, appData string) (*bundleruntime.Supervisor, error) {
	return bundleruntime.NewSupervisor(bundleruntime.Options{
		Manifest:   manifest,
		BundlePath: bundleRoot,
		AppDataDir: appData,
		DryRun:     false,
	})
}

func defaultReadFileWithRetry(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

func defaultReadPortFile(path string, timeout time.Duration) (int, error) {
	data, err := defaultReadFileWithRetry(path, timeout)
	if err != nil {
		return 0, err
	}
	var port int
	_, err = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &port)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func defaultWaitForHealth(client *http.Client, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("runtime control API not responding within %s", timeout)
}

// Create creates a new preflight session with the given manifest and bundle root.
func (s *InMemorySessionStore) Create(manifest *bundlemanifest.Manifest, bundleRoot string, ttlSeconds int) (*Session, error) {
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	if ttlSeconds > 900 {
		ttlSeconds = 900
	}

	appData, err := os.MkdirTemp("", "s2d-preflight-*")
	if err != nil {
		return nil, err
	}

	supervisor, err := s.createSupervisor(manifest, bundleRoot, appData)
	if err != nil {
		_ = os.RemoveAll(appData)
		return nil, err
	}

	ctx := context.Background()
	if err := supervisor.Start(ctx); err != nil {
		_ = os.RemoveAll(appData)
		return nil, err
	}

	fileTimeout := 5 * time.Second
	tokenPath := bundlemanifest.ResolvePath(appData, manifest.IPC.AuthTokenRel)
	tokenBytes, err := s.readFileWithRetry(tokenPath, fileTimeout)
	if err != nil {
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("read auth token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	portPath := filepath.Join(appData, "runtime", "ipc_port")
	port, err := s.readPortFile(portPath, fileTimeout)
	if err != nil {
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("read ipc_port: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%d", manifest.IPC.Host, port)
	client := &http.Client{Timeout: 2 * time.Second}
	if err := s.waitForHealth(client, baseURL, 10*time.Second); err != nil {
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, err
	}

	session := &Session{
		ID:         uuid.NewString(),
		Manifest:   manifest,
		BundleDir:  bundleRoot,
		AppData:    appData,
		Supervisor: supervisor,
		BaseURL:    baseURL,
		Token:      token,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}

	s.mux.Lock()
	s.sessions[session.ID] = session
	s.mux.Unlock()

	return session, nil
}

// Get retrieves a session by ID, checking expiry.
func (s *InMemorySessionStore) Get(id string) (*Session, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, id)
		go shutdownSession(session)
		return nil, false
	}
	return session, true
}

// Refresh extends the TTL of an existing session.
func (s *InMemorySessionStore) Refresh(session *Session, ttlSeconds int) {
	if ttlSeconds <= 0 {
		return
	}
	maxTTL := 900
	if ttlSeconds > maxTTL {
		ttlSeconds = maxTTL
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	session.ExpiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
}

// Stop stops and cleans up a preflight session.
func (s *InMemorySessionStore) Stop(id string) bool {
	s.mux.Lock()
	session, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mux.Unlock()
	if ok {
		shutdownSession(session)
	}
	return ok
}

// Cleanup removes expired sessions.
func (s *InMemorySessionStore) Cleanup() {
	var expired []*Session
	now := time.Now()
	s.mux.Lock()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			expired = append(expired, session)
			delete(s.sessions, id)
		}
	}
	s.mux.Unlock()
	for _, session := range expired {
		shutdownSession(session)
	}
}

// shutdownSession cleans up session resources.
func shutdownSession(session *Session) {
	if session == nil {
		return
	}
	if session.Supervisor != nil {
		_ = session.Supervisor.Shutdown(context.Background())
	}
	if session.AppData != "" {
		_ = os.RemoveAll(session.AppData)
	}
}
