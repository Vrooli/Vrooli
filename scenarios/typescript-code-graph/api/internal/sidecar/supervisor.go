package sidecar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Config configures a Supervisor.
type Config struct {
	// DistPath is the absolute path to the bundled sidecar entrypoint
	// (e.g. scenarios/typescript-code-graph/sidecar/dist/index.js).
	DistPath string

	// NodeBin is the node executable. Defaults to "node" on PATH.
	NodeBin string

	// HeartbeatInterval is the time between heartbeats. Defaults to 10s.
	HeartbeatInterval time.Duration

	// HeartbeatTimeout is the deadline for a single heartbeat reply
	// before the child is killed. Defaults to 5s.
	HeartbeatTimeout time.Duration

	// HandshakeTimeout is the deadline for the initial handshake. Defaults
	// to 5s.
	HandshakeTimeout time.Duration

	// StderrSink, if non-nil, receives raw stderr bytes from the child.
	// Production typically wires os.Stderr; tests may capture a buffer.
	StderrSink io.Writer
}

func (c Config) withDefaults() Config {
	if c.NodeBin == "" {
		c.NodeBin = "node"
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = defaultHeartbeatInterval
	}
	if c.HeartbeatTimeout <= 0 {
		c.HeartbeatTimeout = defaultHeartbeatTimeout
	}
	if c.HandshakeTimeout <= 0 {
		c.HandshakeTimeout = 5 * time.Second
	}
	if c.StderrSink == nil {
		c.StderrSink = io.Discard
	}
	return c
}

// childExtraEnv lets tests inject env vars for the fake sidecar.
// Kept package-private; production never sets this.
type childExtraEnv []string

// pending holds the bookkeeping for an outstanding request.
type pending struct {
	ch chan rawResponse
	//nolint:unused // reserved for drainPending crash-unblock wiring; retained as documented intent.
	cancel func() // called by drainPending to unblock waiters on crash
}

// rawResponse carries an undecoded payload to the request waiter.
type rawResponse struct {
	raw  json.RawMessage
	kind string // top-level "type" field
}

// Supervisor owns a single Node child and the IPC bookkeeping around
// it. The supervisor goroutine respawns on crash with bounded backoff.
type Supervisor struct {
	cfg      Config
	extraEnv childExtraEnv

	// rootCtx scopes the entire supervisor lifetime; Shutdown cancels
	// it to stop the supervisor goroutine.
	rootCtx    context.Context
	rootCancel context.CancelFunc

	// mu guards everything below.
	mu       sync.Mutex
	status   Status
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	writer   *frameWriter
	pendings map[string]*pending
	ledger   restartLedger
	// generation increments per successful start; tests can use it for
	// crash detection.
	generation int

	// supDone closes when the supervisor goroutine returns (after
	// Shutdown or permanent failure).
	supDone chan struct{}

	// started is true after Start completes successfully; prevents
	// double-Start.
	started bool
}

// NewSupervisor constructs an unstarted supervisor.
func NewSupervisor(cfg Config) *Supervisor {
	return &Supervisor{
		cfg:      cfg.withDefaults(),
		status:   StatusUnhealthy,
		pendings: make(map[string]*pending),
		supDone:  make(chan struct{}),
	}
}

// Start spawns the child, performs the handshake, and launches the
// supervisor and heartbeat goroutines.
//
// The provided ctx scopes ONLY the synchronous startup window (the
// initial handshake): cancelling it before Start returns aborts startup
// and tears the child down. Once Start returns, the caller's ctx no
// longer governs anything — the child process lifetime is owned by the
// supervisor's internal context (rooted at context.Background()) and is
// torn down exclusively via Shutdown. This means a caller may safely
// pass a request-scoped or otherwise short-lived ctx to Start without
// risking a SIGKILL of the running child when that ctx later cancels.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("sidecar: supervisor already started")
	}
	s.started = true
	s.mu.Unlock()

	if _, err := os.Stat(s.cfg.DistPath); err != nil {
		return fmt.Errorf("sidecar: dist path not found: %w", err)
	}

	// The child's lifetime is owned by the supervisor, NOT the caller.
	// rootCtx is rooted at Background and cancelled only by Shutdown (or
	// permanent failure below) — never by the caller's startup ctx.
	s.rootCtx, s.rootCancel = context.WithCancel(context.Background())

	if err := s.spawn(ctx); err != nil {
		// supervisor goroutine never started; mark permanent so callers
		// don't wait forever on supDone.
		s.mu.Lock()
		s.status = StatusPermanentlyUnhealthy
		s.mu.Unlock()
		close(s.supDone)
		s.rootCancel()
		return err
	}

	go s.supervise()
	go s.heartbeatLoop()
	return nil
}

// spawn launches a fresh child and performs the handshake. Caller must
// NOT hold s.mu. The child process is bound to the supervisor-owned
// rootCtx (so it outlives any per-call ctx); startupCtx scopes only the
// handshake wait, letting an initial Start abort cleanly if its caller
// cancels mid-startup. On respawn the supervisor passes rootCtx as the
// startupCtx. On success the supervisor's status is READY and
// cmd/stdin/writer fields are populated.
func (s *Supervisor) spawn(startupCtx context.Context) error {
	s.mu.Lock()
	s.status = StatusRestarting
	s.mu.Unlock()

	cmd := exec.CommandContext(s.rootCtx, s.cfg.NodeBin, s.cfg.DistPath)
	if len(s.extraEnv) > 0 {
		cmd.Env = append(os.Environ(), s.extraEnv...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("sidecar: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("sidecar: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("sidecar: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("sidecar: start: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.stdin = stdin
	s.writer = newFrameWriter(stdin)
	s.generation++
	gen := s.generation
	s.mu.Unlock()

	// Stderr pump: copy raw bytes to the configured sink. Exits when
	// the child closes stderr.
	go func() {
		_, _ = io.Copy(s.cfg.StderrSink, stderr)
	}()

	// Reader pump: demux stdout frames into per-request channels.
	readerDone := make(chan struct{})
	go s.readLoop(stdout, gen, readerDone)

	// Synchronous handshake before declaring ready.
	if err := s.doHandshake(startupCtx); err != nil {
		// Force the child down so the supervisor goroutine observes a
		// crash and applies backoff.
		_ = cmd.Process.Kill()
		<-readerDone
		_ = cmd.Wait()
		return fmt.Errorf("sidecar: handshake: %w", err)
	}

	s.mu.Lock()
	s.status = StatusReady
	s.mu.Unlock()
	return nil
}

// doHandshake performs the initial handshake. Must be called after
// spawn so the writer / pending registry are wired. startupCtx scopes
// the handshake wait only: cancelling it aborts startup but does not
// (by itself) kill the already-spawned child — spawn does that
// explicitly on the error path.
func (s *Supervisor) doHandshake(startupCtx context.Context) error {
	reqID := uuid.NewString()
	ch := s.registerPending(reqID)
	defer s.unregisterPending(reqID)

	req := handshakeRequest{
		Type:            "handshake",
		RequestID:       reqID,
		ProtocolVersion: ProtocolVersion,
	}
	if err := s.writer.Write(req); err != nil {
		return err
	}

	select {
	case resp := <-ch:
		if resp.kind != "handshake" {
			return fmt.Errorf("unexpected handshake reply kind %q", resp.kind)
		}
		var hr handshakeResponse
		if err := json.Unmarshal(resp.raw, &hr); err != nil {
			return fmt.Errorf("decode handshake: %w", err)
		}
		if hr.ProtocolVersion != ProtocolVersion {
			return &SidecarVersionMismatch{Want: ProtocolVersion, Got: hr.ProtocolVersion}
		}
		return nil
	case <-time.After(s.cfg.HandshakeTimeout):
		return ErrSidecarTimeout
	case <-startupCtx.Done():
		return startupCtx.Err()
	case <-s.rootCtx.Done():
		return s.rootCtx.Err()
	}
}

// readLoop scans stdout, decodes each line as a JSON object, and
// routes it by request_id. Exits when stdout closes (i.e. the child
// died) or on unrecoverable decode error. Closes done before
// returning so spawn() can synchronize on it during failed handshake.
func (s *Supervisor) readLoop(stdout io.ReadCloser, gen int, done chan struct{}) {
	defer close(done)
	defer stdout.Close()

	scanner := newFrameScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Minimal envelope decode: peel type + request_id only.
		var env struct {
			Type      string `json:"type"`
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal(line, &env); err != nil {
			// Malformed line — log via stderr sink and continue. We
			// can't recover the request_id, so we drop.
			fmt.Fprintf(s.cfg.StderrSink, "sidecar: malformed frame: %v\n", err)
			continue
		}
		// Copy the bytes because scanner.Bytes() is reused on the next Scan.
		raw := make(json.RawMessage, len(line))
		copy(raw, line)

		s.mu.Lock()
		// If the generation has rolled (i.e. a respawn happened), drop
		// any stragglers from the prior child.
		if gen != s.generation {
			s.mu.Unlock()
			continue
		}
		p, ok := s.pendings[env.RequestID]
		s.mu.Unlock()
		if !ok {
			// Likely a cancel-after-response or an orphan from a
			// caller that already gave up. Drop with note.
			fmt.Fprintf(s.cfg.StderrSink, "sidecar: orphan response request_id=%s type=%s\n", env.RequestID, env.Type)
			continue
		}
		select {
		case p.ch <- rawResponse{raw: raw, kind: env.Type}:
		default:
			// Pending channel buffered=1; if full, the waiter already
			// took a response. Drop.
		}
	}
	// scanner.Err() may be non-nil on broken pipe — that's expected on
	// child death.
}

// registerPending allocates a buffered channel for a request. Buffer
// size 1 ensures the reader goroutine never blocks on a slow waiter.
func (s *Supervisor) registerPending(reqID string) chan rawResponse {
	ch := make(chan rawResponse, 1)
	s.mu.Lock()
	s.pendings[reqID] = &pending{ch: ch}
	s.mu.Unlock()
	return ch
}

func (s *Supervisor) unregisterPending(reqID string) {
	s.mu.Lock()
	delete(s.pendings, reqID)
	s.mu.Unlock()
}

// drainPending closes all in-flight request channels with the given
// error sentinel (delivered as a synthetic error envelope). Called
// after a crash so blocked callers unblock with SidecarUnavailable.
func (s *Supervisor) drainPending() {
	s.mu.Lock()
	pendings := s.pendings
	s.pendings = make(map[string]*pending)
	s.mu.Unlock()

	// Synthetic envelope tagged "__drained__" so request methods know
	// to translate to ErrSidecarUnavailable.
	for _, p := range pendings {
		select {
		case p.ch <- rawResponse{kind: "__drained__"}:
		default:
		}
	}
}

// supervise watches cmd.Wait and respawns with backoff. Exits when
// the root context is cancelled (Shutdown) or the restart budget is
// exhausted.
func (s *Supervisor) supervise() {
	defer close(s.supDone)
	attempt := 0
	for {
		// Wait for current child to exit.
		s.mu.Lock()
		cmd := s.cmd
		s.mu.Unlock()
		if cmd == nil {
			return
		}
		_ = cmd.Wait()

		// Mark unhealthy + drain waiters.
		s.mu.Lock()
		s.status = StatusUnhealthy
		s.mu.Unlock()
		s.drainPending()

		// Stop if shutdown was requested.
		select {
		case <-s.rootCtx.Done():
			return
		default:
		}

		// Record this restart and consult the budget.
		s.mu.Lock()
		s.ledger.record(time.Now())
		exhausted := s.ledger.exhausted()
		s.mu.Unlock()
		if exhausted {
			s.mu.Lock()
			s.status = StatusPermanentlyUnhealthy
			s.mu.Unlock()
			return
		}

		// Backoff before respawn.
		wait := backoffSchedule(attempt)
		attempt++
		select {
		case <-time.After(wait):
		case <-s.rootCtx.Done():
			return
		}

		// Try to respawn. If spawn itself fails (e.g. dist gone), loop
		// will re-record and eventually exhaust budget. The respawn's
		// handshake is scoped to rootCtx (no external caller ctx exists
		// here) so a Shutdown mid-respawn aborts it.
		if err := s.spawn(s.rootCtx); err != nil {
			fmt.Fprintf(s.cfg.StderrSink, "sidecar: respawn failed: %v\n", err)
			// Force a zero-wait child so cmd.Wait above returns
			// immediately on the next iteration: use a dummy by leaving
			// s.cmd nil-checked. Simpler: continue, which will loop on
			// the now-stale s.cmd, observe immediate exit, and apply
			// backoff again until budget exhausts.
			continue
		}
		// Reset attempt counter when a respawn fully succeeds (status READY).
		s.mu.Lock()
		if s.status == StatusReady {
			attempt = 0
		}
		s.mu.Unlock()
	}
}

// heartbeatLoop pings the child every HeartbeatInterval. On timeout,
// kills the process so the supervisor goroutine respawns. Exits when
// the root context is cancelled.
func (s *Supervisor) heartbeatLoop() {
	t := time.NewTicker(s.cfg.HeartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-s.rootCtx.Done():
			return
		case <-t.C:
			if !s.heartbeatOnce() {
				// Heartbeat failed — kill child; supervisor goroutine
				// will respawn.
				s.mu.Lock()
				cmd := s.cmd
				s.mu.Unlock()
				if cmd != nil && cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			}
		}
	}
}

// heartbeatOnce sends a single heartbeat and waits up to HeartbeatTimeout.
// Returns true on success, false on timeout or transport error.
func (s *Supervisor) heartbeatOnce() bool {
	s.mu.Lock()
	w := s.writer
	st := s.status
	s.mu.Unlock()
	if w == nil || st != StatusReady {
		return true // skip; nothing to do until READY
	}

	reqID := uuid.NewString()
	ch := s.registerPending(reqID)
	defer s.unregisterPending(reqID)

	if err := w.Write(heartbeatRequest{Type: "heartbeat", RequestID: reqID}); err != nil {
		return false
	}
	select {
	case resp := <-ch:
		return resp.kind == "heartbeat"
	case <-time.After(s.cfg.HeartbeatTimeout):
		return false
	case <-s.rootCtx.Done():
		return true // shutdown in progress; don't kill
	}
}

// Status reports the current supervisor status.
func (s *Supervisor) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Shutdown sends a graceful shutdown IPC and waits for the supervisor
// goroutine to exit. If ctx expires first, the child is killed.
func (s *Supervisor) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	w := s.writer
	cmd := s.cmd
	s.mu.Unlock()

	// Cancel root context first so the supervisor goroutine does not
	// respawn after the imminent exit.
	if s.rootCancel != nil {
		s.rootCancel()
	}

	if w != nil {
		// Best-effort graceful message; ignore write errors.
		_ = w.Write(map[string]string{"type": "shutdown"})
	}

	// Wait for supervisor exit or ctx expiry.
	select {
	case <-s.supDone:
		return nil
	case <-ctx.Done():
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-s.supDone
		return ctx.Err()
	}
}

// ---------- request methods ----------

// extractRequest / extractResponse mirror plan §8.4 wire shapes.
type extractRequest struct {
	Type         string `json:"type"`
	RequestID    string `json:"request_id"`
	ScenarioPath string `json:"scenario_path"`
}

type extractResponse struct {
	Graph        RawGraph  `json:"graph"`
	Warnings     []Warning `json:"warnings"`
	ExtractionMS int64     `json:"extraction_ms"`
	GraphHash    string    `json:"graph_hash"`
}

// rewriteApplyRequest / rewriteApplyResponse mirror plan §8.4 wire shapes.
type rewriteApplyRequest struct {
	Type         string      `json:"type"`
	RequestID    string      `json:"request_id"`
	ScenarioPath string      `json:"scenario_path"`
	Operations   []Operation `json:"operations"`
}

type rewriteApplyResponse struct {
	Results []OperationResult `json:"results"`
}

// cancelRequest mirrors plan §8.4 wire shape.
type cancelRequest struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
}

// Extract implements SidecarClient.
func (s *Supervisor) Extract(ctx context.Context, scenarioPath string) (ExtractResult, error) {
	if st := s.Status(); st != StatusReady {
		return ExtractResult{}, s.statusErr(st)
	}
	reqID := uuid.NewString()
	ch := s.registerPending(reqID)
	// unregister deferred so a late response after ctx-cancel is dropped
	// on the next read-loop dispatch (channel goes out of the map).
	defer s.unregisterPending(reqID)

	s.mu.Lock()
	w := s.writer
	s.mu.Unlock()
	if w == nil {
		return ExtractResult{}, ErrSidecarUnavailable
	}
	if err := w.Write(extractRequest{Type: "extract", RequestID: reqID, ScenarioPath: scenarioPath}); err != nil {
		return ExtractResult{}, fmt.Errorf("%w: %v", ErrSidecarUnavailable, err)
	}

	select {
	case resp := <-ch:
		return s.decodeExtractResponse(reqID, resp)
	case <-ctx.Done():
		// Best-effort cancel IPC; resolve locally regardless. The reqID
		// still travels back so callers can correlate the cancelled work.
		_ = w.Write(cancelRequest{Type: "cancel", RequestID: reqID})
		return ExtractResult{RequestID: reqID}, ctx.Err()
	}
}

func (s *Supervisor) decodeExtractResponse(reqID string, resp rawResponse) (ExtractResult, error) {
	switch resp.kind {
	case "extract":
		var er extractResponse
		if err := json.Unmarshal(resp.raw, &er); err != nil {
			return ExtractResult{RequestID: reqID}, fmt.Errorf("decode extract: %w", err)
		}
		return ExtractResult{Graph: er.Graph, Warnings: er.Warnings, RequestID: reqID}, nil
	case "error":
		var ee errorEnvelope
		if err := json.Unmarshal(resp.raw, &ee); err != nil {
			return ExtractResult{RequestID: reqID}, fmt.Errorf("decode error envelope: %w", err)
		}
		return ExtractResult{RequestID: reqID}, ee.toExtractError()
	case "__drained__":
		return ExtractResult{RequestID: reqID}, ErrSidecarUnavailable
	default:
		return ExtractResult{RequestID: reqID}, fmt.Errorf("unexpected response kind %q", resp.kind)
	}
}

// RewriteApply implements SidecarClient.
func (s *Supervisor) RewriteApply(ctx context.Context, scenarioPath string, ops []Operation) ([]OperationResult, error) {
	if st := s.Status(); st != StatusReady {
		return nil, s.statusErr(st)
	}
	reqID := uuid.NewString()
	ch := s.registerPending(reqID)
	defer s.unregisterPending(reqID)

	s.mu.Lock()
	w := s.writer
	s.mu.Unlock()
	if w == nil {
		return nil, ErrSidecarUnavailable
	}
	if err := w.Write(rewriteApplyRequest{
		Type:         "rewrite_apply",
		RequestID:    reqID,
		ScenarioPath: scenarioPath,
		Operations:   ops,
	}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSidecarUnavailable, err)
	}

	select {
	case resp := <-ch:
		return s.decodeRewriteResponse(resp)
	case <-ctx.Done():
		_ = w.Write(cancelRequest{Type: "cancel", RequestID: reqID})
		return nil, ctx.Err()
	}
}

func (s *Supervisor) decodeRewriteResponse(resp rawResponse) ([]OperationResult, error) {
	switch resp.kind {
	case "rewrite_apply":
		var rr rewriteApplyResponse
		if err := json.Unmarshal(resp.raw, &rr); err != nil {
			return nil, fmt.Errorf("decode rewrite_apply: %w", err)
		}
		return rr.Results, nil
	case "error":
		var ee errorEnvelope
		if err := json.Unmarshal(resp.raw, &ee); err != nil {
			return nil, fmt.Errorf("decode error envelope: %w", err)
		}
		return nil, ee.toRewriteError()
	case "__drained__":
		return nil, ErrSidecarUnavailable
	default:
		return nil, fmt.Errorf("unexpected response kind %q", resp.kind)
	}
}

// statusErr maps a non-READY status to the canonical sentinel.
func (s *Supervisor) statusErr(st Status) error {
	if st == StatusPermanentlyUnhealthy {
		return ErrSidecarPermanentlyUnhealthy
	}
	return ErrSidecarUnavailable
}
