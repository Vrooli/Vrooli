package codecs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

// codexRolloutMaxLineBytes bounds one rollout line read during stop recovery.
// Compacted-history lines reach a few megabytes; a longer line aborts recovery
// rather than guessing.
const codexRolloutMaxLineBytes = 32 << 20

// codexRolloutUsageLine is the subset of a rollout record that usage recovery
// reads. token_usage_record (newer Codex builds) carries the per-turn
// cumulative usage of completed model responses; token_count carries the
// thread-cumulative total and is the fallback for older rollouts.
type codexRolloutUsageLine struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type           string               `json:"type"`
		Role           string               `json:"role"`
		TurnID         string               `json:"turn_id"`
		Info           *codexTokenCountInfo `json:"info"`
		TurnTokenUsage *CodexUsage          `json:"turn_token_usage"`
	} `json:"payload"`
}

// codexInterruptedTurn accumulates what the rollout says about the invocation's
// own turn: the first task_started at or after the invocation start.
type codexInterruptedTurn struct {
	found    bool
	turnID   string
	record   *CodexUsage // latest token_usage_record turn_token_usage
	baseline *CodexUsage // thread total before the turn (token_count fallback)
	latest   *CodexUsage // latest thread total inside the turn
	closed   bool        // task_complete / turn_completed / turn_aborted seen
	pending  bool        // a model request was outstanding at the end of the log
}

// RecoverInterruptedUsage satisfies [InterruptedUsageRecoverer]. It reads the
// run-scoped rollout ($CODEX_HOME/sessions/**/rollout-*<session>.jsonl) for the
// turn this invocation started, after `codex exec` stopped before printing
// turn.completed (for example when a park ends the process mid-turn).
//
// The receipt is authoritative when the rollout closed the turn, or when its
// last completed response is newer than any tool output or prompt (no model
// request was outstanding). Otherwise the completed responses' usage is
// persisted as an ordinary usage event, a lower bound for the turn.
func (c *Codex) RecoverInterruptedUsage(req InterruptedUsageRequest) ([]*domain.RunEvent, string) {
	home := launchEnvValue(req.Env, "CODEX_HOME")
	sessionID := strings.TrimSpace(req.SessionID)
	if home == "" || sessionID == "" || req.StartedAt.IsZero() {
		return nil, ""
	}
	path, err := codexRolloutPath(home, sessionID)
	if err != nil {
		return nil, ""
	}
	turn, err := scanCodexInterruptedTurn(path, req.StartedAt)
	if err != nil {
		return nil, fmt.Sprintf("codex rollout usage recovery skipped: %v", err)
	}
	if !turn.found {
		return nil, ""
	}

	var tokens usageTokens
	switch {
	case turn.record != nil:
		tokens = codexUsageDelta(turn.record.InputTokens, turn.record.OutputTokens, turn.record.CachedInputTokens, true)
	case turn.latest != nil:
		tokens = codexThreadTotalDelta(turn.baseline, turn.latest)
	default:
		return nil, ""
	}
	if tokens.InputTokens <= 0 && tokens.OutputTokens <= 0 && tokens.CacheReadTokens <= 0 {
		return nil, ""
	}

	model := strings.TrimSpace(req.Model)
	events := buildCostEvents(req.RunID, domain.RunnerTypeCodex, c.pricingService, model, tokens, req.Billing)
	usage := events[0].Data.(*domain.UsageEventData)
	usage.TurnIndex = 1
	for _, event := range events {
		event.Timestamp = req.EndedAt
	}
	authoritative := turn.closed || (turn.record != nil && !turn.pending)
	if !authoritative {
		return events, fmt.Sprintf("recovered %d tokens of completed model calls from the codex rollout after the process stopped mid-turn; a request was still in flight, so the turn's usage is a lower bound", tokens.InputTokens+tokens.OutputTokens+tokens.CacheReadTokens)
	}
	// Close the turn exactly as turn.completed would: one authoritative receipt
	// carrying its charge, with the charge also kept as its own event.
	usage.ReconciliationAuthority, usage.Turns = true, 1
	usage.ReconciliationSource = domain.ReconciliationSourceInterruptedTurn
	if turn.closed {
		usage.ReconciliationSource = domain.ReconciliationSourceTranscriptRecovery
	}
	for _, event := range events[1:] {
		if charge, ok := event.Data.(*domain.ChargeEventData); ok {
			usage.Charge = charge
		}
	}
	return events, fmt.Sprintf("recovered the interrupted turn's usage receipt (%s) from the codex rollout after the process stopped mid-turn", usage.ReconciliationSource)
}

// codexRolloutPath finds the session's rollout under a run-scoped CODEX_HOME.
func codexRolloutPath(home, sessionID string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(home, "sessions", "*", "*", "*", "rollout-*"+sessionID+".jsonl"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no rollout for session %s", sessionID)
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

// scanCodexInterruptedTurn streams the rollout and summarizes the first turn
// that started at or after startedAt. Earlier turns only seed the token_count
// baseline. A trailing partial line (the writer was killed) is ignored.
func scanCodexInterruptedTurn(path string, startedAt time.Time) (codexInterruptedTurn, error) {
	var turn codexInterruptedTurn
	f, err := os.Open(path)
	if err != nil {
		return turn, err
	}
	defer f.Close()
	// Rollout timestamps carry millisecond precision.
	startedAt = startedAt.Truncate(time.Millisecond)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64<<10), codexRolloutMaxLineBytes)
	for scanner.Scan() {
		var line codexRolloutUsageLine
		if json.Unmarshal(scanner.Bytes(), &line) != nil {
			continue
		}
		pl := line.Payload
		if !turn.found {
			switch {
			case line.Type == "event_msg" && pl.Type == "token_count" && pl.Info != nil && pl.Info.TotalTokenUsage != nil:
				turn.baseline = pl.Info.TotalTokenUsage
			case line.Type == "event_msg" && pl.Type == "task_started":
				at, perr := time.Parse(time.RFC3339Nano, line.Timestamp)
				if perr == nil && !at.Before(startedAt) {
					turn.found, turn.turnID, turn.pending = true, pl.TurnID, true
				}
			}
			continue
		}
		if turn.closed {
			// Anything after the turn closed belongs to a later turn.
			break
		}
		switch line.Type {
		case "token_usage_record":
			if pl.TurnTokenUsage != nil && (turn.turnID == "" || pl.TurnID == "" || pl.TurnID == turn.turnID) {
				turn.record, turn.pending = pl.TurnTokenUsage, false
			}
		case "response_item":
			role := strings.ToLower(strings.TrimSpace(pl.Role))
			if strings.HasSuffix(pl.Type, "_output") || (pl.Type == "message" && (role == "user" || role == "operator")) {
				// A tool result or prompt is sent back to the model at once.
				turn.pending = true
			}
		case "event_msg":
			switch pl.Type {
			case "token_count":
				if pl.Info != nil && pl.Info.TotalTokenUsage != nil {
					turn.latest = pl.Info.TotalTokenUsage
				}
			case "user_message":
				turn.pending = true
			case "task_complete", "turn_completed", "turn_aborted":
				turn.closed = true
			case "task_started":
				// One invocation runs one turn; a later turn is not this one's.
				return turn, nil
			}
		}
	}
	return turn, scanner.Err()
}

// codexThreadTotalDelta is the thread-cumulative usage added since baseline,
// with cached input split out. A reset (any component shrinking) starts a new
// series, so the latest total is the delta.
func codexThreadTotalDelta(baseline, latest *CodexUsage) usageTokens {
	if baseline == nil || latest.InputTokens < baseline.InputTokens || latest.CachedInputTokens < baseline.CachedInputTokens || latest.OutputTokens < baseline.OutputTokens {
		return codexUsageDelta(latest.InputTokens, latest.OutputTokens, latest.CachedInputTokens, true)
	}
	return codexUsageDelta(latest.InputTokens-baseline.InputTokens, latest.OutputTokens-baseline.OutputTokens, latest.CachedInputTokens-baseline.CachedInputTokens, true)
}
