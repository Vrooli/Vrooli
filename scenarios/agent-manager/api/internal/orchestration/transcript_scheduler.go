// Transcript scheduling owns the periodic, idempotent adoption of external
// harness transcripts into the agent-manager evidence corpus.
package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
)

// TranscriptImportSummary describes one scheduled corpus sweep.
type TranscriptImportSummary struct {
	Scanned  int `json:"scanned"`
	Imported int `json:"imported"`
	Existing int `json:"existing"`
	Skipped  int `json:"skipped"`
	// SkipReasons breaks the Skipped count down by cause so a sweep that is
	// quietly dropping sessions is observable instead of a bare count. Keyed by
	// a stable reason label (see the transcriptSkip* constants). Previously a
	// skipped session only incremented Skipped with no "why", so a format drift
	// that stopped every session from yielding a session_id looked identical to
	// an empty corpus.
	SkipReasons map[string]int `json:"skip_reasons,omitempty"`
	Failures    []string       `json:"failures,omitempty"`
	// Roots reports what each scanned root contributed so "why isn't this
	// conversation here" is answerable from the sweep output alone: a configured
	// root that does not exist, or exists but yielded zero sessions, is visible
	// instead of silently folded into the aggregate counts.
	Roots []TranscriptRootSummary `json:"roots,omitempty"`
}

// TranscriptRootSummary is one scanned root's contribution to a sweep. Present
// (false) with a non-empty Error distinguishes a root that is simply absent (no
// activity yet) from one that could not be read (a permissions or path problem
// worth an operator's attention).
type TranscriptRootSummary struct {
	Path                string `json:"path"`
	Kind                string `json:"kind"`
	WebConsoleSessionID string `json:"web_console_session_id,omitempty"`
	Present             bool   `json:"present"`
	Scanned             int    `json:"scanned"`
	Imported            int    `json:"imported"`
	Existing            int    `json:"existing"`
	Skipped             int    `json:"skipped"`
	Error               string `json:"error,omitempty"`
}

// Stable skip-reason labels surfaced in import-sweep output.
const (
	transcriptSkipNoSessionID     = "missing_session_id"
	transcriptSkipPolicyExclude   = "policy_excluded"
	transcriptSkipImportBudget    = "web_console_import_budget"
	webConsoleCodexHomeKind       = "web-console-codex-home"
	defaultWebConsoleImportBudget = 100
)

// recordSkip increments the total skip count and the per-reason breakdown.
func (s *TranscriptImportSummary) recordSkip(reason string) {
	s.Skipped++
	if s.SkipReasons == nil {
		s.SkipReasons = make(map[string]int)
	}
	s.SkipReasons[reason]++
}

// TranscriptImportScheduler keeps external harness evidence current. It uses
// ImportTranscript's source-session idempotency check, so repeated sweeps are
// safe and never create duplicate runs.
type TranscriptImportScheduler struct {
	orchestrator *Orchestrator
	interval     time.Duration
	// extraRoots are operator-configured directories swept in addition to the
	// built-in harness roots and the discovered web-console homes. Each entry is
	// an absolute path (a leading ~ is expanded against the user's home).
	extraRoots []string
	cancel     context.CancelFunc
	done       chan struct{}
	mu         sync.Mutex

	// sweeps counts completed background sweeps. A corpus can look stale for two
	// very different reasons — the loop never ran, or it ran and found nothing —
	// and this distinguishes them without inspecting the imported rows.
	sweeps atomic.Int64
}

// Sweeps reports how many background sweeps have completed since Start.
func (s *TranscriptImportScheduler) Sweeps() int64 {
	if s == nil {
		return 0
	}
	return s.sweeps.Load()
}

func NewTranscriptImportScheduler(orchestrator *Orchestrator, interval time.Duration, extraRoots ...string) *TranscriptImportScheduler {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	cleaned := make([]string, 0, len(extraRoots))
	for _, root := range extraRoots {
		if trimmed := strings.TrimSpace(root); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return &TranscriptImportScheduler{orchestrator: orchestrator, interval: interval, extraRoots: cleaned}
}

func (s *TranscriptImportScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil || s.orchestrator == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	// done is captured locally: Stop clears the field, and a worker that had not
	// yet reached its deferred close would then close a nil channel and panic
	// the process. The channel, not the field, is what this goroutine owns.
	done := make(chan struct{})
	s.cancel, s.done = cancel, done
	go func() {
		defer close(done)
		// Sweep once at startup. The ticker's first tick is a whole interval
		// away and every restart restarts that interval, so a deployment that
		// restarts more often than the interval would never import at all and
		// the corpus would silently stay stale.
		s.sweep(workerCtx)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.sweep(workerCtx)
			case <-workerCtx.Done():
				return
			}
		}
	}()
}

// sweep runs one import pass, reporting failure without ending the loop: a
// transient harness-root problem must not silently stop future sweeps.
func (s *TranscriptImportScheduler) sweep(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	if _, err := s.RunOnce(ctx); err != nil {
		obs.Component("transcript-importer").Warn("scheduled transcript import failed", obs.KeyError, err.Error())
	}
	s.sweeps.Add(1)
}

func (s *TranscriptImportScheduler) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// RunOnce scans only governed durable harness roots. It is exported so the
// lifecycle can be tested without waiting six hours and so operators can use
// the same idempotent operation when diagnosing a scheduler.
func (s *TranscriptImportScheduler) RunOnce(ctx context.Context) (TranscriptImportSummary, error) {
	var summary TranscriptImportSummary
	if s == nil || s.orchestrator == nil {
		return summary, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return summary, err
	}
	// Codex launched by web-console runs under a per-session CODEX_HOME, so its
	// rollouts land outside the default ~/.codex home the static roots cover.
	// Discovering those homes is what keeps web-console efforts (the majority of
	// current activity) from being silently absent from the evidence corpus.
	// Web-console homes are swept FIRST (newest-first) so recent, relevant effort
	// evidence is adopted before the sweep spends time on the large, stale
	// default-home history — otherwise a big ~/.codex backlog delays importing an
	// actively-running effort's transcript by the whole backlog's parse time.
	roots := s.discoverWebConsoleCodexRoots(home)
	roots = append(roots,
		transcriptRoot{path: filepath.Join(home, ".codex", "sessions"), harness: "codex", runner: domain.RunnerTypeCodex, kind: "codex-home"},
		transcriptRoot{path: filepath.Join(home, ".claude", "projects"), harness: "claude-code", runner: domain.RunnerTypeClaudeCode, kind: "claude-home"},
	)
	// Operator-configured extra roots are swept in addition to the built-ins so
	// externally-run / remote agents writing transcripts into a custom directory
	// are importable. The runner is left unset so ImportTranscript sniffs the
	// format per file, matching the manual-import path for ad-hoc locations. A
	// stable "configured" harness label gives each file a provenance key
	// (harness + session id) so a repeat sweep is idempotent rather than
	// re-importing the same transcript.
	//
	// A configured root that points at (or inside) a built-in / web-console root
	// must NOT be swept again under the "configured" harness: provenance keys on
	// (harness, sessionID), so the same transcript walked once as "codex" and
	// once as "configured" would import TWICE — the exact double-import W2's
	// idempotency guarantee forbids. Pointing a scan root at ~/.codex/sessions is
	// the most natural operator reaction to a missing chat, so this foot-gun is
	// deduped here. Built-ins keep their real harness label (existing codex/claude
	// provenance preserved); only the redundant configured entry is dropped.
	builtinRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		builtinRoots = append(builtinRoots, filepath.Clean(root.path))
	}
	for _, extra := range s.extraRoots {
		cleaned := filepath.Clean(expandHome(extra, home))
		if pathCoveredBy(cleaned, builtinRoots) {
			continue
		}
		roots = append(roots, transcriptRoot{path: cleaned, harness: "configured", runner: domain.RunnerType(""), kind: "configured"})
		// Fold this configured root into the covered set so a second configured
		// root nested under it (or equal to it) is also dropped.
		builtinRoots = append(builtinRoots, cleaned)
	}
	// Resolve every web-console session's managed-run status in one batched
	// lookup before the loop. Discovery can yield thousands of homes, so a
	// per-root query would hammer the store once per session.
	wcSessionIDs := make([]string, 0, len(roots))
	for _, root := range roots {
		if root.webConsoleSessionID != "" {
			wcSessionIDs = append(wcSessionIDs, root.webConsoleSessionID)
		}
	}
	managedWC := s.managedWebConsoleSessions(ctx, wcSessionIDs)
	importBudget := webConsoleImportBudget()
	wcImported := 0
	for _, root := range roots {
		rootSummary := TranscriptRootSummary{Path: root.path, Kind: root.kind, WebConsoleSessionID: root.webConsoleSessionID}
		// An isolated-home session that already has a live/managed run is
		// captured through the admission path; importing its transcript again
		// would duplicate that run. Only skip on a non-imported run, so fresh
		// rollout segments of an otherwise-unadmitted session still import.
		if root.webConsoleSessionID != "" && managedWC[root.webConsoleSessionID] {
			rootSummary.Present = true
			summary.Roots = append(summary.Roots, rootSummary)
			continue
		}
		// Bound the web-console backfill: once this sweep has imported its
		// budget of new web-console sessions, skip the rest (newest were visited
		// first) and let the next idempotent sweep drain more. Budget <=0 is
		// unbounded.
		if root.kind == webConsoleCodexHomeKind && importBudget > 0 && wcImported >= importBudget {
			rootSummary.Present = true
			rootSummary.Skipped++
			summary.recordSkip(transcriptSkipImportBudget)
			summary.Roots = append(summary.Roots, rootSummary)
			continue
		}
		// Report a configured/absent root explicitly: a root that cannot be
		// stat'd for any reason other than "does not exist" is surfaced as an
		// error rather than silently yielding nothing.
		if info, statErr := os.Stat(root.path); statErr != nil {
			if !os.IsNotExist(statErr) {
				rootSummary.Error = statErr.Error()
			}
			summary.Roots = append(summary.Roots, rootSummary)
			continue
		} else if !info.IsDir() {
			rootSummary.Present = true
			rootSummary.Error = "not a directory"
			summary.Roots = append(summary.Roots, rootSummary)
			continue
		}
		rootSummary.Present = true
		_ = filepath.WalkDir(root.path, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				return nil
			}
			summary.Scanned++
			rootSummary.Scanned++
			sessionID := transcriptSessionIDFromPath(path)
			if sessionID == "" {
				// Match the manual-import path (handlers/session_import.go),
				// which falls back to the filename stem when the header has no
				// session_id. Without this the sweep silently skipped sessions
				// the manual importer would have adopted.
				sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			}
			if sessionID == "" {
				summary.recordSkip(transcriptSkipNoSessionID)
				rootSummary.Skipped++
				return nil
			}
			// Resolve provenance before importing. ImportTranscript returns an
			// already-adopted run with no error, so counting "no error" as an
			// import reported the whole corpus as freshly imported on every
			// sweep. Checking first also skips reopening and reparsing a
			// transcript that is already in the corpus.
			if existing, lookupErr := s.orchestrator.runs.GetByImportProvenance(ctx, root.harness, sessionID); lookupErr == nil && existing != nil {
				summary.Existing++
				rootSummary.Existing++
				return nil
			}
			if _, importErr := s.orchestrator.ImportTranscript(ctx, ImportTranscriptRequest{Path: path, RunnerType: root.runner, SourceHarness: root.harness, SourceSessionID: sessionID, WebConsoleSessionID: root.webConsoleSessionID}); importErr != nil {
				if IsImportExcluded(importErr) {
					// A visibility policy kept this session out of the corpus. It is
					// an operator-configured exclusion, not a failure, and it is
					// counted so the exclusion is observable in the sweep summary.
					summary.recordSkip(transcriptSkipPolicyExclude)
					rootSummary.Skipped++
				} else if strings.Contains(importErr.Error(), "already") {
					summary.Existing++
					rootSummary.Existing++
				} else if existing, lookupErr := s.orchestrator.runs.GetByImportProvenance(ctx, root.harness, sessionID); lookupErr == nil && existing != nil {
					summary.Existing++
					rootSummary.Existing++
				} else {
					summary.Failures = append(summary.Failures, path+": "+importErr.Error())
				}
				return nil
			}
			summary.Imported++
			rootSummary.Imported++
			if root.kind == webConsoleCodexHomeKind {
				wcImported++
			}
			return nil
		})
		summary.Roots = append(summary.Roots, rootSummary)
	}
	return summary, nil
}

// expandHome replaces a leading ~ (alone or as ~/...) with the user's home dir.
// Other paths are returned unchanged so an absolute configured root is used
// verbatim.
func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// pathCoveredBy reports whether cleaned equals, or is nested under, any path in
// covered. Both cleaned and every covered entry are expected to already be
// filepath.Clean'd. Nesting uses a separator-anchored prefix so "/a/bc" is not
// treated as nested under "/a/b".
func pathCoveredBy(cleaned string, covered []string) bool {
	for _, c := range covered {
		if cleaned == c {
			return true
		}
		if strings.HasPrefix(cleaned, c+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// transcriptRoot is one discovered harness session root. webConsoleSessionID is
// set only for isolated per-session homes, where it attributes the rollout back
// to its launching web-console session and gates duplicate-run suppression.
type transcriptRoot struct {
	path                string
	harness             string
	runner              domain.RunnerType
	webConsoleSessionID string
	// kind is a stable label for the per-root sweep summary so an operator can
	// tell a built-in harness root from a discovered web-console home or a
	// configured extra root.
	kind string
}

// webConsoleCodexSessionsBase is the parent of the per-session isolated
// CODEX_HOME directories web-console creates. Each child is one session id whose
// own sessions/ subtree receives that session's codex rollouts.
func webConsoleCodexSessionsBase(home string) string {
	return filepath.Join(home, ".vrooli", "state", "vrooli", "web-console", "sessions", "codex")
}

// discoverWebConsoleCodexRoots lists each isolated codex session home and points
// at the sessions/ subtree codex writes rollouts into. A missing base simply
// yields no roots, so hosts without web-console activity are unaffected.
//
// The base can hold tens of thousands of empty scaffolding homes, so a session
// is only emitted as a root when its sessions/ subtree actually exists — this
// keeps the sweep from creating a root (and a dedup lookup) per empty directory.
// Roots are returned newest-first so a bounded sweep spends its import budget on
// actively-running efforts before draining old backlog.
func (s *TranscriptImportScheduler) discoverWebConsoleCodexRoots(home string) []transcriptRoot {
	base := webConsoleCodexSessionsBase(home)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	type candidate struct {
		root    transcriptRoot
		modTime time.Time
	}
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || !entry.IsDir() {
			continue
		}
		sessionID := entry.Name()
		sessionsDir := filepath.Join(base, sessionID, "sessions")
		info, statErr := os.Stat(sessionsDir)
		if statErr != nil || !info.IsDir() {
			continue
		}
		candidates = append(candidates, candidate{
			root: transcriptRoot{
				path:                sessionsDir,
				harness:             "codex",
				runner:              domain.RunnerTypeCodex,
				webConsoleSessionID: sessionID,
				kind:                webConsoleCodexHomeKind,
			},
			modTime: info.ModTime(),
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].modTime.After(candidates[j].modTime) })
	roots := make([]transcriptRoot, 0, len(candidates))
	for _, c := range candidates {
		roots = append(roots, c.root)
	}
	return roots
}

// managedWebConsoleSessions returns the set of web-console session ids that are
// captured by a LIVE admission-path run, which importing would duplicate. Only
// a non-imported run in an active state (pending/starting/running/parked) counts:
// a terminal attached run is a stub (web-console live-attach records at most a
// placeholder event, never the codex rollout), so it must NOT shadow the real
// transcript — otherwise a session whose only runs are failed/needs_review
// attach stubs can never be imported. A prior sweep's imported run never counts,
// so fresh rollout segments still import. Resolved in one batched lookup —
// discovery can yield thousands of homes and a per-session query would hammer
// the store. A lookup failure marks every candidate managed so the conservative
// outcome is a skipped import the next sweep retries, never a duplicate.
func (s *TranscriptImportScheduler) managedWebConsoleSessions(ctx context.Context, sessionIDs []string) map[string]bool {
	managed := make(map[string]bool, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return managed
	}
	runs, err := s.orchestrator.runs.ListByWebConsoleSessionIDs(ctx, sessionIDs)
	if err != nil {
		for _, id := range sessionIDs {
			managed[id] = true
		}
		return managed
	}
	for _, run := range runs {
		if run != nil && run.ExecutionMode != domain.ExecutionModeImported && isLiveManagedStatus(run.Status) && strings.TrimSpace(run.WebConsoleSessionID) != "" {
			managed[run.WebConsoleSessionID] = true
		}
	}
	return managed
}

// isLiveManagedStatus reports whether a run is still actively owned by the
// admission path. Terminal states are excluded so a dead attach stub does not
// block importing the session's real transcript.
func isLiveManagedStatus(status domain.RunStatus) bool {
	switch status {
	case domain.RunStatusPending, domain.RunStatusStarting, domain.RunStatusRunning, domain.RunStatusParked:
		return true
	default:
		return false
	}
}

// webConsoleImportBudget bounds how many NEW web-console transcripts one sweep
// imports. Discovery spans thousands of web-console homes (tens of GB); an
// unbounded first sweep would copy them all into the corpus at once. A bounded
// sweep drains newest-first across successive idempotent runs. Env override
// AGENT_MANAGER_SWEEP_MAX_WEBCONSOLE_IMPORTS; <=0 means unbounded.
func webConsoleImportBudget() int {
	if raw := strings.TrimSpace(os.Getenv("AGENT_MANAGER_SWEEP_MAX_WEBCONSOLE_IMPORTS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			return n
		}
	}
	return defaultWebConsoleImportBudget
}

func transcriptSessionIDFromPath(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	return transcriptSessionID(file)
}
