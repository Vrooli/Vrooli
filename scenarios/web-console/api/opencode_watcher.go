package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"web-console/backends/opencode"
	"web-console/internal/sessionstore"
)

var errOpenCodeServeTimeout = errors.New("opencode serve did not report a listening URL in time")

const (
	opencodeSource            = "opencode_api"
	opencodeReconcileInterval = 15 * time.Second
	opencodeReconnectBackoff  = 2 * time.Second
	// attribution slack tolerates a small clock/timing gap between web-console
	// recording the pane and opencode stamping the session it spawns.
	opencodeAttributionSlack = 10 * time.Second
)

// OpenCodeWatcher captures OpenCode conversations through the `opencode serve`
// HTTP API. It owns a single loopback server instance, subscribes to its SSE
// event stream, and reconciles affected sessions through the full-history
// `/session/{id}/message` endpoint rather than trusting deltas. A per-session
// high-water-mark cursor (persisted in agent_transcript_checkpoints) makes
// restart and SSE reconnect idempotent.
//
// Attribution is by directory + creation time + mutual uniqueness against live
// web-console panes that were launched as opencode. Ambiguous matches are
// skipped, never guessed, so one project's sessions never bleed into another
// pane.
//
// DOC: docs/guides/CONVERSATION_TRACKING.md
// DOC: docs/internal/TEMPORAL-FLOWS.md
type OpenCodeWatcher struct {
	server      *Server
	checkpoints AgentTranscriptCheckpointStore

	// Seams for tests: startServer yields a base URL + stop func; newClient
	// builds the API client for that URL.
	startServer func() (string, func(), error)
	newClient   func(baseURL string) opencode.Client

	reconcileInterval time.Duration
	reconnectBackoff  time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup

	mu      sync.Mutex
	client  opencode.Client
	claimed map[string]string // opencode session id -> web-console session id
}

func NewOpenCodeWatcher(server *Server) *OpenCodeWatcher {
	return &OpenCodeWatcher{
		server:            server,
		checkpoints:       server.agentCheckpointStore,
		startServer:       spawnOpenCodeServe,
		newClient:         func(baseURL string) opencode.Client { return opencode.NewHTTPClient(baseURL) },
		reconcileInterval: opencodeReconcileInterval,
		reconnectBackoff:  opencodeReconnectBackoff,
		stopCh:            make(chan struct{}),
		claimed:           make(map[string]string),
	}
}

func (w *OpenCodeWatcher) Start() {
	w.wg.Add(1)
	go w.run()
}

func (w *OpenCodeWatcher) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

func (w *OpenCodeWatcher) run() {
	defer w.wg.Done()

	baseURL, stop, err := w.startServer()
	if err != nil {
		log.Printf("opencode-watcher: server unavailable, capture disabled: %v", err)
		return
	}
	if stop != nil {
		defer stop()
	}
	log.Printf("opencode-watcher: managed server at %s", baseURL)

	client := w.newClient(baseURL)
	w.mu.Lock()
	w.client = client
	w.mu.Unlock()

	// Re-establish claims for already-attributed panes so a restart reconciles
	// (idempotently, via the persisted cursor) instead of orphaning them.
	w.loadExistingClaims()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-w.stopCh
		cancel()
	}()

	// Debounced reconcile worker fed by SSE events and a periodic safety-net
	// ticker. Coalescing avoids a full reconcile per message.part.updated burst.
	trigger := make(chan struct{}, 1)
	w.wg.Add(1)
	go w.reconcileWorker(ctx, client, trigger)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// One reconcile on (re)connect catches anything the stream missed while
		// we were not subscribed.
		signal(trigger)
		err := client.Events(ctx, func(opencode.Event) { signal(trigger) })
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("opencode-watcher: event stream ended (%v); reconnecting", err)
		}
		backoff := w.reconnectBackoff
		if backoff <= 0 {
			backoff = opencodeReconnectBackoff
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

func signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (w *OpenCodeWatcher) reconcileWorker(ctx context.Context, client opencode.Client, trigger <-chan struct{}) {
	defer w.wg.Done()
	interval := w.reconcileInterval
	if interval <= 0 {
		interval = opencodeReconcileInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.reconcileAll(ctx, client)
		case <-trigger:
			w.reconcileAll(ctx, client)
		}
	}
}

// loadExistingClaims seeds the claim map from session metadata so previously
// attributed opencode panes keep reconciling across a restart.
func (w *OpenCodeWatcher) loadExistingClaims() {
	if w.server == nil || w.server.sessionStore == nil {
		return
	}
	metas, err := w.server.sessionStore.List()
	if err != nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, m := range metas {
		if m.AgentType == sessionstore.AgentOpenCode && m.AgentSessionID != "" {
			w.claimed[m.AgentSessionID] = m.ID
		}
	}
}

func (w *OpenCodeWatcher) reconcileAll(ctx context.Context, client opencode.Client) {
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		log.Printf("opencode-watcher: list sessions: %v", err)
		return
	}
	w.attribute(sessions)

	w.mu.Lock()
	pairs := make(map[string]string, len(w.claimed))
	for ocID, wcID := range w.claimed {
		pairs[ocID] = wcID
	}
	w.mu.Unlock()

	for ocID, wcID := range pairs {
		w.reconcileSession(ctx, client, ocID, wcID)
	}
}

// attribute binds unclaimed opencode sessions to live web-console panes that
// were launched as opencode but have no agent session id yet. A pair is claimed
// only when it is mutually unique — the session matches exactly one candidate
// pane and that pane matches exactly one session — so concurrent same-directory
// sessions are skipped rather than misrouted.
func (w *OpenCodeWatcher) attribute(sessions []opencode.Session) {
	if w.server == nil || w.server.sessionStore == nil {
		return
	}
	metas, err := w.server.sessionStore.List()
	if err != nil {
		return
	}

	w.mu.Lock()
	claimedSession := make(map[string]bool, len(w.claimed))
	claimedPane := make(map[string]bool, len(w.claimed))
	for ocID, wcID := range w.claimed {
		claimedSession[ocID] = true
		claimedPane[wcID] = true
	}
	w.mu.Unlock()

	var panes []sessionstore.Metadata
	for _, m := range metas {
		if m.AgentType == sessionstore.AgentOpenCode && m.AgentSessionID == "" && !claimedPane[m.ID] {
			panes = append(panes, m)
		}
	}
	if len(panes) == 0 {
		return
	}

	// Candidate edges between panes and unclaimed sessions.
	paneMatches := make(map[string][]string)    // pane id -> session ids
	sessionMatches := make(map[string][]string) // session id -> pane ids
	sessionByID := make(map[string]opencode.Session)
	for _, s := range sessions {
		if claimedSession[s.ID] {
			continue
		}
		sessionByID[s.ID] = s
		for _, p := range panes {
			if !opencodeSessionMatchesPane(s, p) {
				continue
			}
			paneMatches[p.ID] = append(paneMatches[p.ID], s.ID)
			sessionMatches[s.ID] = append(sessionMatches[s.ID], p.ID)
		}
	}

	for paneID, sessionIDs := range paneMatches {
		if len(sessionIDs) != 1 {
			continue // pane is ambiguous across sessions
		}
		sID := sessionIDs[0]
		if len(sessionMatches[sID]) != 1 {
			continue // session is ambiguous across panes
		}
		w.claim(paneID, sessionByID[sID])
	}
}

// opencodeSessionMatchesPane is the per-edge attribution predicate: a session
// created at/after the pane (minus slack) and, when the pane's cwd is known,
// in the same directory.
func opencodeSessionMatchesPane(s opencode.Session, p sessionstore.Metadata) bool {
	if !p.Created.IsZero() {
		earliest := p.Created.Add(-opencodeAttributionSlack).UnixMilli()
		if s.Time.Created < earliest {
			return false
		}
	}
	if p.CWD != "" && s.Directory != p.CWD {
		return false
	}
	return true
}

func (w *OpenCodeWatcher) claim(paneID string, s opencode.Session) {
	w.mu.Lock()
	if _, exists := w.claimed[s.ID]; exists {
		w.mu.Unlock()
		return
	}
	w.claimed[s.ID] = paneID
	w.mu.Unlock()

	if w.server.sessionStore != nil {
		_ = w.server.sessionStore.UpdateAgentInfo(paneID, sessionstore.AgentInfo{
			AgentType:      sessionstore.AgentOpenCode,
			AgentSessionID: s.ID,
			CWD:            s.Directory,
			LastActivityAt: time.Now(),
		})
	}
	log.Printf("opencode-watcher: attributed session %s -> pane %s", s.ID, paneID)
}

func (w *OpenCodeWatcher) reconcileSession(ctx context.Context, client opencode.Client, ocID, wcID string) {
	messages, err := client.SessionMessages(ctx, ocID)
	if err != nil {
		log.Printf("opencode-watcher: messages for %s: %v", ocID, err)
		return
	}
	cur := w.loadCursor(ocID)
	emissions, next := opencode.Normalize(messages, cur)
	for _, e := range emissions {
		switch e.Role {
		case "user":
			w.server.AppendUser(e.Text, wcID, opencodeSource)
		case "assistant":
			w.server.AppendAssistant(e.Text, wcID, opencodeSource)
		}
	}
	if next != cur {
		w.saveCursor(ocID, wcID, next)
	}
}

func (w *OpenCodeWatcher) loadCursor(ocID string) opencode.Cursor {
	if w.checkpoints == nil {
		return opencode.Cursor{}
	}
	cp, ok, err := w.checkpoints.Get(opencodeSource, ocID)
	if err != nil || !ok || cp.Cursor == "" {
		return opencode.Cursor{}
	}
	var cur opencode.Cursor
	if err := json.Unmarshal([]byte(cp.Cursor), &cur); err != nil {
		return opencode.Cursor{}
	}
	return cur
}

func (w *OpenCodeWatcher) saveCursor(ocID, wcID string, cur opencode.Cursor) {
	if w.checkpoints == nil {
		return
	}
	data, err := json.Marshal(cur)
	if err != nil {
		return
	}
	if err := w.checkpoints.Save(AgentTranscriptCheckpoint{
		Source:    opencodeSource,
		SourceKey: ocID,
		SessionID: wcID,
		Cursor:    string(data),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		log.Printf("opencode-watcher: save cursor for %s: %v", ocID, err)
	}
}

var opencodeListeningRE = regexp.MustCompile(`listening on (http://\S+)`)

// spawnOpenCodeServe starts a loopback-only `opencode serve` and returns its
// base URL plus a stop func. Binds 127.0.0.1 with an OS-chosen port; never
// exposed beyond loopback. Returns an error (capture stays disabled) when the
// binary is absent or never prints a listening URL.
func spawnOpenCodeServe() (string, func(), error) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return "", nil, err
	}
	cmd := exec.Command(path, "serve", "--hostname", "127.0.0.1", "--port", "0")
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", nil, err
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return "", nil, err
	}
	pw.Close() // parent keeps only the read end

	urlCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			if m := opencodeListeningRE.FindStringSubmatch(scanner.Text()); m != nil {
				select {
				case urlCh <- m[1]:
				default:
				}
				break
			}
		}
		// Drain the rest so the child never blocks on a full pipe.
		_, _ = io.Copy(io.Discard, pr)
	}()

	stop := func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		pr.Close()
	}

	select {
	case url := <-urlCh:
		return url, stop, nil
	case <-time.After(15 * time.Second):
		stop()
		return "", nil, errOpenCodeServeTimeout
	}
}
