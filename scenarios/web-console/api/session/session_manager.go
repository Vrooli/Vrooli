package session

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"web-console/internal/backend"
	"web-console/internal/config"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/policy"
	"web-console/internal/pty"
	"web-console/internal/sessionstore"
	"web-console/terminal"

	"github.com/google/uuid"
)

// Sentinel errors for session operations. Handlers use these to select the
// correct HTTP status code and user-facing message.
var (
	// ErrSessionLimitReached is returned when MaxSessions is configured and
	// the limit has been reached. Maps to HTTP 429.
	ErrSessionLimitReached = errors.New("session limit reached")

	// ErrPTYSpawnFailed wraps PTY creation failures. Maps to HTTP 500.
	ErrPTYSpawnFailed = errors.New("PTY spawn failed")

	// ErrBackendUnavailable is returned when the requested backend is not available.
	ErrBackendUnavailable = errors.New("backend unavailable")

	// ErrBackendUnknown is returned when the requested backend is not registered.
	ErrBackendUnknown = errors.New("unknown backend")
)

// Manager tracks all active terminal sessions.
// [REQ:P0-002a] PTY Session Backend
type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	ptyFactory   pty.Factory
	cfgMu        sync.RWMutex // protects cfg from concurrent read/write (session-defaults handler vs Create)
	cfg          config.Config
	registry     *backend.Registry
	store        sessionstore.Store
	shuttingDown bool // set by Shutdown(); prevents auto-remove from deleting persistent session metadata

	// Seams for testability: injectable tmux operations. After Session moves
	// to its own sub-package these break the dependency on tmux helpers that
	// live with the persistent backend.
	tmuxAttachFunc       TmuxAttachFunc
	tmuxDiscoverFunc     TmuxDiscoverFunc
	applyTmuxOptionsFunc func(sessionName string)
	killTmuxSessionFunc  func(sessionName string)
	tmuxSessionPrefix    string

	// uploadDirFunc resolves the per-session upload root. Defaults to
	// resolveUploadDir; injectable so internal/session/ doesn't import the
	// upload handler.
	uploadDirFunc func() string

	// envForSession returns the per-session environment variables to inject
	// into the spawned PTY (CODEX_HOME etc). Set by package main; the
	// session package does not know about backend-specific env keys.
	envForSession func(sessionID string) map[string]string

	// Observability: optional metrics and event logger for session lifecycle.
	metrics *metrics.Metrics
	events  *events.Logger

	// reattachStopCh signals the periodic re-attach watchdog to stop.
	reattachStopCh chan struct{}

	// lifecycleCtx is owned by the manager rather than an HTTP request. It gives
	// cleanup and watchdog goroutines a bounded lifetime without smuggling a
	// request-scoped context into work that outlives the request.
	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc

	// recoveryMu guards recovery, the live progress of startup session
	// recovery. Recovery runs asynchronously (so the HTTP listener comes up
	// without waiting on reattaching N tmux sessions); this snapshot is what
	// the API exposes so the UI can honestly show "sessions still recovering".
	recoveryMu sync.RWMutex
	recovery   RecoveryProgress
}

// RecoveryProgress is a snapshot of startup session-recovery progress, surfaced
// through the API so clients opening the app mid-recovery see an honest state
// rather than an apparently-empty session list.
type RecoveryProgress struct {
	// InProgress is true from the moment recovery is scheduled until it finishes.
	InProgress bool
	// StartedAt / CompletedAt bound the recovery window (CompletedAt is zero
	// while InProgress).
	StartedAt   time.Time
	CompletedAt time.Time
	// Total is the number of persisted (detached) metadata rows recovery will
	// attempt to reattach. Adopted orphan sessions are extra and not counted here.
	Total int
	// Recovered / AwaitingRecovery / Adopted accumulate as recovery proceeds.
	Recovered        int
	AwaitingRecovery int
	Adopted          int
}

// NewManager returns a Manager with the configured PTY factory and
// configuration wired in plus nil-safe no-op defaults for the optional
// integration hooks (tmux, upload-dir, per-session env). Production callers
// (package main) overwrite the defaults via the Set* methods; tests can use
// the bare manager directly because the defaults won't panic.
func NewManager(factory pty.Factory, cfg config.Config) *Manager {
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	return &Manager{
		sessions:             make(map[string]*Session),
		ptyFactory:           factory,
		cfg:                  cfg,
		lifecycleCtx:         lifecycleCtx,
		lifecycleCancel:      lifecycleCancel,
		uploadDirFunc:        func() string { return "" },
		envForSession:        func(string) map[string]string { return nil },
		tmuxDiscoverFunc:     func() ([]string, error) { return nil, nil },
		tmuxAttachFunc:       func(string) (pty.PTY, error) { return nil, nil },
		applyTmuxOptionsFunc: func(string) {},
		killTmuxSessionFunc:  func(string) {},
	}
}

// NewManagerWithFactory is a convenience for tests: returns a bare Manager
// with the given factory and config.Default() applied. Tmux hooks remain nil;
// callers that exercise tmux paths must call SetTmuxHooks.
func NewManagerWithFactory(factory pty.Factory) *Manager {
	return NewManager(factory, config.Default())
}

// SetTmuxHooks installs the tmux integration callbacks. Called by package
// main during server startup; tests substitute fakes here when exercising
// recovery or re-attach paths.
func (sm *Manager) SetTmuxHooks(
	attach TmuxAttachFunc,
	discover TmuxDiscoverFunc,
	applyOptions func(sessionName string),
	killSession func(sessionName string),
	prefix string,
) {
	sm.tmuxAttachFunc = attach
	sm.tmuxDiscoverFunc = discover
	sm.applyTmuxOptionsFunc = applyOptions
	sm.killTmuxSessionFunc = killSession
	sm.tmuxSessionPrefix = prefix
}

// SetUploadDirFunc registers the per-session upload-root resolver. Required
// before Create or Recover; nil defeats upload cleanup on session exit.
func (sm *Manager) SetUploadDirFunc(fn func() string) { sm.uploadDirFunc = fn }

// SetEnvForSessionFunc registers the per-session environment-variable builder
// (CODEX_HOME, etc.). Called once at startup by package main.
func (sm *Manager) SetEnvForSessionFunc(fn func(sessionID string) map[string]string) {
	sm.envForSession = fn
}

// SetRegistry sets the backend registry for backend-aware session creation.
func (sm *Manager) SetRegistry(reg *backend.Registry) {
	sm.registry = reg
}

// SetStore sets the session metadata store for persistence.
func (sm *Manager) SetStore(store sessionstore.Store) {
	sm.store = store
}

// SetMetrics sets the metrics collector for session lifecycle counters.
func (sm *Manager) SetMetrics(m *metrics.Metrics) {
	sm.metrics = m
}

// SetEvents sets the event logger for structured session lifecycle events.
func (sm *Manager) SetEvents(el *events.Logger) {
	sm.events = el
}

// GetConfig returns a snapshot of the current configuration. Thread-safe.
func (sm *Manager) GetConfig() config.Config {
	sm.cfgMu.RLock()
	defer sm.cfgMu.RUnlock()
	return sm.cfg
}

// SetConfigField updates a mutable config field under the write lock.
// Only use for fields that can change at runtime (default backend/policy).
func (sm *Manager) SetConfigField(fn func(cfg *config.Config)) {
	sm.cfgMu.Lock()
	defer sm.cfgMu.Unlock()
	fn(&sm.cfg)
}

// applySessionDefaults fills in zero-valued parameters with configured defaults.
// The convention is: zero/empty from the caller means "use server default".
func (sm *Manager) applySessionDefaults(shell string, cols, rows uint16) (string, uint16, uint16) {
	sm.cfgMu.RLock()
	defer sm.cfgMu.RUnlock()
	if shell == "" {
		shell = sm.cfg.DefaultShell
	}
	if cols == 0 {
		cols = sm.cfg.DefaultCols
	}
	if rows == 0 {
		rows = sm.cfg.DefaultRows
	}
	return shell, cols, rows
}

// isSessionLimitReached decides whether a new session should be rejected based
// on the configured MaxSessions cap. A cap of 0 means unlimited.
func (sm *Manager) isSessionLimitReached() bool {
	if sm.cfg.MaxSessions <= 0 {
		return false
	}
	sm.mu.RLock()
	count := len(sm.sessions)
	sm.mu.RUnlock()
	return count >= sm.cfg.MaxSessions
}

// Create starts a new shell session with a PTY.
// [REQ:P0-002a] PTY Session Backend
func (sm *Manager) Create(ctx context.Context, shell string, cols, rows uint16, bid backend.ID, pol *policy.Policy) (*Session, error) {
	return sm.createWithOptions(ctx, shell, cols, rows, bid, pol, "", false)
}

// CreateWithWorkingDir starts a session in workingDir when non-empty. It is
// intentionally separate from Create so existing create paths retain their
// configured-default behavior.
func (sm *Manager) CreateWithWorkingDir(ctx context.Context, shell string, cols, rows uint16, bid backend.ID, pol *policy.Policy, workingDir string) (*Session, error) {
	return sm.createWithOptions(ctx, shell, cols, rows, bid, pol, workingDir, false)
}

// CreateWithOptions starts a session with creation-time backend options. The
// tmux mouse choice is carried in the typed launch spec so it cannot leak
// through an untyped environment variable or a global setting.
func (sm *Manager) CreateWithOptions(ctx context.Context, shell string, cols, rows uint16, bid backend.ID, pol *policy.Policy, workingDir string, tmuxMouseMode bool) (*Session, error) {
	return sm.createWithOptions(ctx, shell, cols, rows, bid, pol, workingDir, tmuxMouseMode)
}

// RemoteLaunch carries server-side Bridge credentials into the typed backend
// factory. It is never serialized into a browser response.
type RemoteLaunch struct {
	BaseURL              string
	NodeID               string
	OwnerToken           string
	ReauthToken          string
	Shell                string
	WorkingDir           string
	Cols                 uint16
	Rows                 uint16
	LaunchCommand        string
	ExecuteLaunchCommand bool
}

// CreateRemote creates a node-agent session through the same Session object
// and websocket handler used by local PTYs.
func (sm *Manager) CreateRemote(ctx context.Context, launch RemoteLaunch, pol *policy.Policy) (*Session, error) {
	return sm.createWithRemote(ctx, launch.Shell, launch.Cols, launch.Rows, backend.Remote, pol, launch.WorkingDir, false, &launch)
}

func (sm *Manager) createWithOptions(ctx context.Context, shell string, cols, rows uint16, bid backend.ID, pol *policy.Policy, workingDir string, tmuxMouseMode bool) (*Session, error) {
	return sm.createWithRemote(ctx, shell, cols, rows, bid, pol, workingDir, tmuxMouseMode, nil)
}

func (sm *Manager) createWithRemote(ctx context.Context, shell string, cols, rows uint16, bid backend.ID, pol *policy.Policy, workingDir string, tmuxMouseMode bool, remote *RemoteLaunch) (*Session, error) {
	shell, cols, rows = sm.applySessionDefaults(shell, cols, rows)

	// Resolve backend (read default under lock to avoid data race with settings handler)
	if bid == "" || bid == "auto" {
		sm.cfgMu.RLock()
		bid = backend.ID(sm.cfg.DefaultBackend)
		sm.cfgMu.RUnlock()
	}

	// Look up factory from registry if available, otherwise use injected factory
	var factory pty.Factory
	if sm.registry != nil {
		// Resolve "auto" via registry when it wasn't resolved at startup (e.g. tests)
		if bid == "auto" {
			bid = sm.registry.ResolveAuto()
		}
		desc, ok := sm.registry.Get(bid)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrBackendUnknown, bid)
		}
		if !desc.Available {
			return nil, fmt.Errorf("%w: %s — %s", ErrBackendUnavailable, bid, desc.Reason)
		}
		factory, _ = sm.registry.Factory(bid)
	} else {
		// No registry (test path) — use injected factory, clear backend ID
		if bid == "auto" {
			bid = ""
		}
		factory = sm.ptyFactory
	}

	if sm.isSessionLimitReached() {
		return nil, fmt.Errorf("%w (%d)", ErrSessionLimitReached, sm.cfg.MaxSessions)
	}

	sessionID := uuid.New().String()
	// Resolve the launch directory here rather than leaving it empty for the
	// PTY layer to default. The directory is not just a spawn argument: it is
	// half of the key that locates an agent's transcript on disk, and a session
	// that never records it can never have its messages captured. Resolving
	// once means the spawned process and the persisted row cannot disagree.
	launchDir := strings.TrimSpace(workingDir)
	if launchDir == "" {
		launchDir = config.ResolveWorkingDir()
	}
	spec := pty.LaunchSpec{
		SessionID:     sessionID,
		Shell:         shell,
		Cols:          cols,
		Rows:          rows,
		WorkingDir:    launchDir,
		Env:           sm.envForSession(sessionID),
		TmuxMouseMode: tmuxMouseMode,
	}
	if remote != nil {
		spec.RemoteURL = remote.BaseURL
		spec.RemoteNodeID = remote.NodeID
		spec.RemoteOwnerToken = remote.OwnerToken
		spec.RemoteReauthToken = remote.ReauthToken
		spec.LaunchCommand = remote.LaunchCommand
		spec.ExecuteLaunchCommand = remote.ExecuteLaunchCommand
	}

	p, err := factory(spec)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPTYSpawnFailed, err)
	}
	uploadRoot := sm.uploadDirFunc()

	// Resolve policy (read defaults under lock to avoid data race with settings handler)
	var sessionPolicy policy.Policy
	if pol != nil {
		sessionPolicy = *pol
	} else {
		sm.cfgMu.RLock()
		mode := sm.cfg.DefaultPolicyMode
		dur := sm.cfg.DefaultPolicyDuration
		sm.cfgMu.RUnlock()
		if mode != "" {
			sessionPolicy = policy.Policy{
				Mode:     policy.Mode(mode),
				Duration: dur,
			}
		} else {
			sessionPolicy = policy.Default()
		}
	}

	sess := &Session{
		ID:                      sessionID,
		Shell:                   shell,
		CreatedAt:               time.Now(),
		Cols:                    cols,
		Rows:                    rows,
		Backend:                 bid,
		pty:                     p,
		policy:                  sessionPolicy,
		uploadRoot:              uploadRoot,
		clients:                 make(map[chan []byte]*ClientInfo),
		inputQueue:              make(chan queuedInput, sm.cfg.InputQueueSize),
		inputStopCh:             make(chan struct{}),
		exitCh:                  make(chan struct{}),
		emu:                     terminal.New(terminal.Options{Cols: int(cols), Rows: int(rows), ScrollbackLines: sm.cfg.TerminalScrollbackLines}),
		ptyReadBuffer:           sm.cfg.PTYReadBuffer,
		clientChannelBuffer:     sm.cfg.ClientChannelBuffer,
		coalesceNotifyThreshold: sm.cfg.CoalesceNotifyThreshold,
		reattachFunc:            sm.tmuxAttachFunc,
		sessionPrefix:           sm.tmuxSessionPrefix,
		metrics:                 sm.metrics,
	}
	sess.RefreshEchoState(true)
	sess.startInputWriter()

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	// Persist metadata if store is configured
	if sm.store != nil {
		detached := bid == backend.Persistent
		_ = sm.store.Save(ctx, sessionstore.Metadata{
			ID:       sess.ID,
			Backend:  bid,
			Shell:    shell,
			Cols:     cols,
			Rows:     rows,
			Policy:   sessionPolicy,
			Created:  sess.CreatedAt,
			Detached: detached,
			CWD:      launchDir,
		})
	}

	// Wire server-side ANSI responder before readLoop starts so the
	// ControlEvent channel is in place when the first query arrives.
	sess.startAnsiResponder()

	// Start the PTY output reader; it will close exitCh when the process exits.
	go sess.readLoop()

	// Auto-remove: when the PTY exits, clean up the session map entry and
	// any upload temp directory so List()/Get() no longer return a terminated session.
	go func() {
		<-sess.Done()
		log.Printf("session %s: process exited (backend=%s)", sess.ID, bid)
		sm.mu.Lock()
		delete(sm.sessions, sess.ID)
		sm.mu.Unlock()
		// Persistent sessions: ALWAYS preserve metadata so recovery can
		// re-attach on the next startup. The tmux session survives in its
		// own systemd scope even when the attach process dies. Deleting
		// metadata here would orphan the tmux session, causing recovery to
		// kill it — permanently destroying a recoverable session.
		//
		// Standard sessions: always delete metadata (they cannot survive).
		if sm.store != nil && bid != backend.Persistent {
			_ = sm.store.Delete(sm.lifecycleCtx, sess.ID)
		}
		// Clean up session upload directory
		uploadDir := filepath.Join(sess.uploadRoot, sess.ID)
		if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
			log.Printf("session %s: failed to clean up upload dir: %v", sess.ID, err)
		}
	}()

	return sess, nil
}

// Get returns a session by ID.
func (sm *Manager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sess, ok := sm.sessions[id]
	return sess, ok
}

// List returns all active sessions.
// [REQ:P0-003a] Session Persistence Store
func (sm *Manager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		result = append(result, s)
	}
	return result
}

// Delete terminates a session and cleans up resources.
func (sm *Manager) Delete(ctx context.Context, id string) error {
	return sm.terminate(ctx, id, false)
}

// Archive terminates a session while preserving its persisted metadata. The
// archive service owns the archived_at transition and all transcript state.
func (sm *Manager) Archive(ctx context.Context, id string) error {
	return sm.terminate(ctx, id, true)
}

func (sm *Manager) terminate(ctx context.Context, id string, preserveMetadata bool) error {
	sm.mu.Lock()
	sess, ok := sm.sessions[id]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("session %s not found", id)
	}
	delete(sm.sessions, id)
	sm.mu.Unlock()

	p := sess.currentPTY()
	_ = p.Kill()
	sess.stopInputWriter()
	_ = p.Close()
	// Clean up persisted metadata
	if sm.store != nil && !preserveMetadata {
		_ = sm.store.Delete(ctx, id)
	}
	// Clean up session upload directory
	uploadRoot := sess.uploadRoot
	if uploadRoot == "" {
		uploadRoot = sm.uploadDirFunc()
	}
	uploadDir := filepath.Join(uploadRoot, id)
	if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
		log.Printf("session %s: failed to clean up upload dir on delete: %v", id, err)
	}
	return nil
}
