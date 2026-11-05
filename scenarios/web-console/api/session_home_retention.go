package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/storage"
	"web-console/internal/sessionstore"
)

// Session homes hold two kinds of content. The conversation record (rollout
// JSONL under sessions/, codex history.jsonl and its thread/state/goal/memory
// sqlite stores, grok updates.jsonl) is the only durable copy of a transcript:
// agent-manager evicts its imported corpus and relies on these files staying
// re-importable. Everything else is agent runtime state that the agent CLI
// re-creates on the next launch: per-home installs of the codex app-server
// daemon (~430 MB each), plugin and cache trees from homes provisioned before
// those directories were linked to the shared home, codex trace-log sqlite, and
// grok binary downloads. Retention removes only that runtime state, and only
// from sessions that are not open and have been idle past the threshold. It
// never deletes or rewrites a transcript.
var sessionHomeRegenerableDirs = map[string]map[string]bool{
	"codex": {
		"packages": true, ".tmp": true, "tmp": true, "log": true, "logs": true,
		"cache": true, "plugins": true, "shell_snapshots": true,
		"app-server-control": true, "app-server-daemon": true, "thread-writer-locks": true,
	},
	"grok": {"downloads": true},
}

var codexTraceLogSQLite = regexp.MustCompile(`^logs_\d+\.sqlite(-wal|-shm)?$`)

const (
	sessionHomeRetentionOwner      = "web-console"
	sessionHomeRetentionCycleFile  = "session-homes-cycle.json"
	defaultSessionHomeMinIdle      = 7 * 24 * time.Hour
	defaultSessionHomeMaxActions   = 25000
	defaultSessionHomeInterval     = time.Hour
	defaultSessionHomeInitialDelay = 5 * time.Minute
	defaultSessionHomeMaxBytes     = int64(60) << 30
)

// Protection reasons recorded in the cycle receipt.
const (
	homeProtectedLiveSession     = "live_session"
	homeProtectedOpenSessionRow  = "open_session_row"
	homeProtectedRecentActivity  = "recent_activity"
	homeProtectedScaffoldWithRow = "scaffold_with_session_row"
	homeProtectedTranscript      = "transcript_retained"
	homeProtectedActionBudget    = "cycle_action_budget"
)

type sessionHomeRetention struct {
	root       string
	minIdle    time.Duration
	maxActions int
	maxBytes   int64
	isLive     func(id string) bool
	rows       func(ctx context.Context) ([]sessionstore.Metadata, error)
	now        func() time.Time
}

type sessionHomeRetentionResult struct {
	StartedAt         time.Time      `json:"started_at"`
	FinishedAt        time.Time      `json:"finished_at"`
	MinIdleSeconds    int64          `json:"min_idle_seconds"`
	MaxBytes          int64          `json:"max_bytes"`
	ScannedHomes      int            `json:"scanned_homes"`
	RemovedScaffolds  int            `json:"removed_scaffolds"`
	StrippedHomes     int            `json:"stripped_homes"`
	RemovedEntries    int            `json:"removed_entries"`
	FreedBytes        int64          `json:"freed_bytes"`
	UsedBytesBefore   int64          `json:"used_bytes_before"`
	UsedBytesAfter    int64          `json:"used_bytes_after"`
	TranscriptBytes   int64          `json:"transcript_bytes"`
	Protected         map[string]int `json:"protected"`
	Failures          int            `json:"failures"`
	FirstFailure      string         `json:"first_failure,omitempty"`
	ActionBudgetSpent bool           `json:"action_budget_spent"`
}

func (r sessionHomeRetentionResult) protectedSummary() string {
	keys := make([]string, 0, len(r.Protected))
	for key := range r.Protected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strconv.Itoa(r.Protected[key]))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",")
}

// homeUsage is one lstat walk of a session home. Symlinks are counted but
// never followed, so the shared agent home is never measured or traversed.
type homeUsage struct {
	bytes  int64
	files  int
	newest time.Time
}

func measureSessionHome(path string) homeUsage {
	var usage homeUsage
	if info, err := os.Lstat(path); err == nil {
		usage.newest = info.ModTime()
	}
	_ = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		if info.ModTime().After(usage.newest) {
			usage.newest = info.ModTime()
		}
		if !entry.IsDir() {
			usage.files++
			usage.bytes += info.Size()
		}
		return nil
	})
	return usage
}

// regenerableEntries lists the top-level runtime entries of one home. A
// symlinked entry points at the shared agent home and is never a candidate.
func regenerableEntries(agent, home string) []string {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil
	}
	dirs := sessionHomeRegenerableDirs[agent]
	out := make([]string, 0)
	for _, entry := range entries {
		if entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		name := entry.Name()
		if (entry.IsDir() && dirs[name]) || (agent == "codex" && entry.Type().IsRegular() && codexTraceLogSQLite.MatchString(name)) {
			out = append(out, filepath.Join(home, name))
		}
	}
	return out
}

func (r *sessionHomeRetention) Run(ctx context.Context) (sessionHomeRetentionResult, error) {
	now := time.Now
	if r.now != nil {
		now = r.now
	}
	result := sessionHomeRetentionResult{
		StartedAt:      now().UTC(),
		MinIdleSeconds: int64(r.minIdle / time.Second),
		MaxBytes:       r.maxBytes,
		Protected:      map[string]int{},
	}
	rows, err := r.rows(ctx)
	if err != nil {
		return result, fmt.Errorf("list session rows: %w", err)
	}
	byID := make(map[string]sessionstore.Metadata, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	cutoff := now().Add(-r.minIdle)
	actions := 0
	fail := func(path string, err error) {
		result.Failures++
		if result.FirstFailure == "" {
			result.FirstFailure = fmt.Sprintf("%s: %v", path, err)
		}
	}
	for _, agent := range []string{"codex", "grok"} {
		agentRoot := filepath.Join(r.root, agent)
		entries, readErr := os.ReadDir(agentRoot)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			id := entry.Name()
			if !entry.IsDir() || filepath.Base(id) != id {
				continue
			}
			home := filepath.Join(agentRoot, id)
			usage := measureSessionHome(home)
			result.ScannedHomes++
			result.UsedBytesBefore += usage.bytes
			freed := r.enforceHome(ctx, agent, agentRoot, home, id, usage, byID, cutoff, &actions, &result, fail)
			result.FreedBytes += freed
			result.UsedBytesAfter += usage.bytes - freed
		}
	}
	result.FinishedAt = now().UTC()
	return result, nil
}

func (r *sessionHomeRetention) enforceHome(
	ctx context.Context,
	agent, agentRoot, home, id string,
	usage homeUsage,
	rows map[string]sessionstore.Metadata,
	cutoff time.Time,
	actions *int,
	result *sessionHomeRetentionResult,
	fail func(string, error),
) int64 {
	row, hasRow := rows[id]
	switch {
	case r.isLive != nil && r.isLive(id):
		result.Protected[homeProtectedLiveSession]++
		return 0
	case hasRow && row.ArchivedAt.IsZero() && row.Status == sessionstore.StatusLive:
		result.Protected[homeProtectedOpenSessionRow]++
		return 0
	case usage.newest.After(cutoff):
		result.Protected[homeProtectedRecentActivity]++
		return 0
	}
	if usage.files == 0 {
		// An empty scaffold holds no transcript. A row-backed scaffold still
		// signals "this session had an agent home" to the reopen path, so only
		// scaffolds no session row claims are removed.
		if hasRow {
			result.Protected[homeProtectedScaffoldWithRow]++
			return 0
		}
		if *actions >= r.maxActions {
			result.Protected[homeProtectedActionBudget]++
			result.ActionBudgetSpent = true
			return 0
		}
		*actions++
		if err := coreRetention.DeleteContained(ctx, agentRoot, home, nil); err != nil {
			fail(home, err)
			return 0
		}
		result.RemovedScaffolds++
		return usage.bytes
	}
	targets := regenerableEntries(agent, home)
	var freed int64
	removed := 0
	for _, target := range targets {
		if *actions >= r.maxActions {
			result.Protected[homeProtectedActionBudget]++
			result.ActionBudgetSpent = true
			break
		}
		*actions++
		bytes := measureSessionHome(target).bytes
		if info, err := os.Lstat(target); err == nil && info.Mode().IsRegular() {
			bytes = info.Size()
		}
		if err := coreRetention.DeleteContained(ctx, home, target, nil); err != nil {
			fail(target, err)
			continue
		}
		freed += bytes
		removed++
	}
	if removed > 0 {
		result.StrippedHomes++
		result.RemovedEntries += removed
	}
	if usage.bytes-freed > 0 {
		result.TranscriptBytes += usage.bytes - freed
		result.Protected[homeProtectedTranscript]++
	}
	return freed
}

// sessionHomeReceiptError makes the shared enforcement receipt honest: deletion
// failures, an exhausted per-cycle action budget, or a tree still over its byte
// budget mean the declared budget was not enforced this cycle. The message
// names what protected the remaining bytes.
func sessionHomeReceiptError(result sessionHomeRetentionResult, cycleErr error) error {
	if cycleErr != nil {
		return cycleErr
	}
	var problems []string
	if result.Failures > 0 {
		problems = append(problems, fmt.Sprintf("%d removals failed (first: %s)", result.Failures, result.FirstFailure))
	}
	if result.ActionBudgetSpent {
		problems = append(problems, "per-cycle action budget spent; backlog continues next cycle")
	}
	if result.MaxBytes > 0 && result.UsedBytesAfter > result.MaxBytes {
		problems = append(problems, fmt.Sprintf("session homes over budget after cycle: used %d of %d bytes", result.UsedBytesAfter, result.MaxBytes))
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s; freed %d bytes, protected (%s)", strings.Join(problems, "; "), result.FreedBytes, result.protectedSummary())
}

func writeSessionHomeCycleReceipt(result sessionHomeRetentionResult) error {
	path, err := coreRetention.EnforcementReceiptPath(sessionHomeRetentionOwner)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(filepath.Join(filepath.Dir(path), sessionHomeRetentionCycleFile), append(data, '\n'), 0o600)
}

func sessionHomeRetentionPolicy() (enabled bool, minIdle, interval time.Duration, maxActions int, maxBytes int64) {
	enabled, minIdle, interval = true, defaultSessionHomeMinIdle, defaultSessionHomeInterval
	maxActions, maxBytes = defaultSessionHomeMaxActions, defaultSessionHomeMaxBytes
	if raw := strings.TrimSpace(os.Getenv("WC_SESSION_HOME_RETENTION_ENABLED")); raw != "" {
		enabled = raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
	}
	if parsed, err := time.ParseDuration(strings.TrimSpace(os.Getenv("WC_SESSION_HOME_RETENTION_MIN_IDLE"))); err == nil && parsed >= time.Hour {
		minIdle = parsed
	}
	if parsed, err := time.ParseDuration(strings.TrimSpace(os.Getenv("WC_SESSION_HOME_RETENTION_INTERVAL"))); err == nil && parsed > 0 {
		interval = parsed
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WC_SESSION_HOME_RETENTION_MAX_ACTIONS"))); err == nil && parsed > 0 {
		maxActions = parsed
	}
	if parsed, err := coreRetention.ParseBytes(strings.TrimSpace(os.Getenv("WC_SESSION_HOME_RETENTION_MAX_BYTES"))); err == nil && parsed > 0 {
		maxBytes = parsed
	}
	return enabled, minIdle, interval, maxActions, maxBytes
}

// startSessionHomeRetention runs the session-home owner policy. Only the live
// instance enforces: a variant derives a private session root and must not
// spend its cycles or write the live owner's receipt.
func (h *webConsoleCleanup) startSessionHomeRetention(ctx context.Context) {
	enabled, minIdle, interval, maxActions, maxBytes := sessionHomeRetentionPolicy()
	if variant := strings.TrimSpace(os.Getenv(storage.EnvVariant)); !enabled || (variant != "" && variant != "live") {
		return
	}
	// Without session rows an unclaimed scaffold cannot be told apart from a
	// row-backed one, so the owner does not run at all.
	if h.server == nil || h.server.sessionStore == nil {
		return
	}
	owner := &sessionHomeRetention{
		root:       resolveSessionStateRoot(),
		minIdle:    minIdle,
		maxActions: maxActions,
		maxBytes:   maxBytes,
		isLive:     h.isProtected,
		rows:       h.server.sessionStore.List,
	}
	go func() {
		timer := time.NewTimer(defaultSessionHomeInitialDelay)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				h.sessionHomeCycle(ctx, owner)
				timer.Reset(interval)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (h *webConsoleCleanup) sessionHomeCycle(ctx context.Context, owner *sessionHomeRetention) {
	release, err := h.acquireRecoveryLock()
	if err != nil {
		log.Printf("web-console session-home retention deferred: %v", err)
		return
	}
	defer release()
	workCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	result, cycleErr := owner.Run(workCtx)
	log.Printf("web-console session-home retention cycle: scanned=%d removed_scaffolds=%d stripped_homes=%d freed_bytes=%d used_bytes=%d max_bytes=%d transcript_bytes=%d protected=%s failures=%d err=%v",
		result.ScannedHomes, result.RemovedScaffolds, result.StrippedHomes, result.FreedBytes, result.UsedBytesAfter,
		result.MaxBytes, result.TranscriptBytes, result.protectedSummary(), result.Failures, cycleErr)
	if err := writeSessionHomeCycleReceipt(result); err != nil {
		log.Printf("web-console session-home retention: write cycle receipt: %v", err)
	}
	if err := coreRetention.RecordEnforcementReceipt(sessionHomeRetentionOwner, time.Now().UTC(), sessionHomeReceiptError(result, cycleErr)); err != nil {
		log.Printf("web-console session-home retention: record receipt: %v", err)
	}
}
