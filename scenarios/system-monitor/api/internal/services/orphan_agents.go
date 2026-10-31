package services

// DOC: docs/concepts/ARCHITECTURE.md#alerting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Orphaned-agent alert boundaries.
const (
	OrphanedAgentAgeLimit    = time.Hour
	OrphanedAgentMemoryLimit = 2 * 1024 * 1024 * 1024
	// orphanInventoryRefresh bounds how often the control plane is asked;
	// the inventory read forks the vrooli CLI.
	orphanInventoryRefresh = 5 * time.Minute
	orphanInventoryTimeout = 15 * time.Second
	orphanUnsupportedText  = "control plane does not classify orphans yet"
)

// AgentOwner is the control plane's owner attribution for a session.
type AgentOwner struct {
	Kind     string `json:"kind"`
	Ref      string `json:"ref"`
	Alive    bool   `json:"alive"`
	Liveness string `json:"liveness"`
	Detail   string `json:"detail"`
	Source   string `json:"source"`
}

// AgentSessionRecord is one row of `vrooli agent list --json`. Memory, swap,
// and reapability come from the control plane's reap planner; when a field is
// absent or null it stays unknown, never zero.
type AgentSessionRecord struct {
	Session     string
	Scope       string
	State       string
	KernelState string
	Owner       *AgentOwner
	PID         int
	Age         time.Duration
	AgeKnown    bool
	MemoryBytes int64
	MemoryKnown bool
	SwapBytes   int64
	SwapKnown   bool
	// MemoryStatus/SwapStatus are the control plane's measurement states
	// (measured, partial, unsupported, unavailable). A partial value is a
	// lower bound; a null value is unknown.
	MemoryStatus  string
	SwapStatus    string
	Reapable      bool
	ReapableKnown bool
	ReapReason    string
}

// AgentInventory is the parsed `vrooli agent list --json` result.
type AgentInventory struct {
	Sessions []AgentSessionRecord
	// ClassifiesOrphans is true when rows carry the control plane's owner
	// attribution, which is what makes its "orphaned" state meaningful.
	ClassifiesOrphans bool
}

// AgentInventorySource reads the control plane's agent inventory.
type AgentInventorySource interface {
	ListAgentSessions(ctx context.Context) (AgentInventory, error)
}

// ParseAgentInventory decodes `vrooli agent list --json`, binding exactly the
// control plane's field names: owner{kind,ref,alive,liveness,detail,source},
// state, kernel_state, age, pid, memory_bytes, swap_bytes (null when unknown),
// memory_status/swap_status, reapable, reap_reason. Rows are read, never the
// top-level summary, so the rule does not depend on the summary's shape.
func ParseAgentInventory(raw []byte) (AgentInventory, error) {
	var doc struct {
		Sessions []map[string]json.RawMessage `json:"sessions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return AgentInventory{}, fmt.Errorf("decode agent inventory: %w", err)
	}
	if doc.Sessions == nil {
		return AgentInventory{}, errors.New("agent inventory has no sessions array")
	}
	inventory := AgentInventory{Sessions: make([]AgentSessionRecord, 0, len(doc.Sessions))}
	for _, row := range doc.Sessions {
		var record AgentSessionRecord
		decodeString(row["session"], &record.Session)
		decodeString(row["scope"], &record.Scope)
		decodeString(row["state"], &record.State)
		decodeString(row["kernel_state"], &record.KernelState)
		if rawOwner, ok := row["owner"]; ok && !isJSONNull(rawOwner) {
			var owner AgentOwner
			if json.Unmarshal(rawOwner, &owner) == nil {
				record.Owner = &owner
				inventory.ClassifiesOrphans = true
			}
		}
		_ = json.Unmarshal(row["pid"], &record.PID)
		var age string
		if decodeString(row["age"], &age) {
			if parsed, err := time.ParseDuration(age); err == nil {
				record.Age, record.AgeKnown = parsed, true
			}
		}
		record.MemoryBytes, record.MemoryKnown = decodeOptionalInt(row["memory_bytes"])
		record.SwapBytes, record.SwapKnown = decodeOptionalInt(row["swap_bytes"])
		decodeString(row["memory_status"], &record.MemoryStatus)
		decodeString(row["swap_status"], &record.SwapStatus)
		if value, ok := row["reapable"]; ok && !isJSONNull(value) {
			record.ReapableKnown = json.Unmarshal(value, &record.Reapable) == nil
		}
		decodeString(row["reap_reason"], &record.ReapReason)
		inventory.Sessions = append(inventory.Sessions, record)
	}
	return inventory, nil
}

func isJSONNull(raw json.RawMessage) bool {
	return len(raw) == 0 || string(raw) == "null"
}

// decodeOptionalInt reads a byte count; absent or null is unknown.
func decodeOptionalInt(raw json.RawMessage) (int64, bool) {
	if isJSONNull(raw) {
		return 0, false
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func decodeString(raw json.RawMessage, dest *string) bool {
	if len(raw) == 0 {
		return false
	}
	return json.Unmarshal(raw, dest) == nil
}

// CLIAgentInventory reads the inventory by running the vrooli CLI. The
// control plane's agent listing lives in its CLI layer, so the exec is the
// stable read contract. Results are cached for orphanInventoryRefresh.
type CLIAgentInventory struct {
	runner CommandRunner
	binary string
	clock  Clock

	mu      sync.Mutex
	fetched time.Time
	cached  AgentInventory
	err     error
}

// NewCLIAgentInventory builds the exec adapter. A nil runner uses os/exec.
func NewCLIAgentInventory(runner CommandRunner, clock Clock) *CLIAgentInventory {
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	if clock == nil {
		clock = RealClock{}
	}
	return &CLIAgentInventory{runner: runner, binary: "vrooli", clock: clock}
}

// ListAgentSessions returns the cached inventory, refreshing it when stale.
func (c *CLIAgentInventory) ListAgentSessions(ctx context.Context) (AgentInventory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.clock.Now()
	if !c.fetched.IsZero() && now.Sub(c.fetched) < orphanInventoryRefresh {
		return c.cached, c.err
	}
	runCtx, cancel := context.WithTimeout(ctx, orphanInventoryTimeout)
	defer cancel()
	stdout, stderr, _, err := c.runner.Run(runCtx, c.binary, []string{"agent", "list", "--json"}, "")
	c.fetched = now
	if err != nil {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = err.Error()
		}
		c.cached, c.err = AgentInventory{}, fmt.Errorf("vrooli agent list: %s", detail)
		return c.cached, c.err
	}
	c.cached, c.err = ParseAgentInventory([]byte(stdout))
	return c.cached, c.err
}

// orphanedAgentsRule reads the control plane's orphan classification, owner,
// memory, and reap plan; it never classifies, reaps, or kills anything.
//
//   - warning: any state=="orphaned" row older than one hour.
//   - critical: orphaned rows hold >= 2 GiB of known memory+swap, or a
//     reapable==true row has stayed reapable for an hour (auto-reap is off or
//     failing).
//
// Unknown memory is reported as unknown and never counted as zero.
type orphanedAgentsRule struct {
	source AgentInventorySource
	clock  Clock

	mu            sync.Mutex
	reapableSince map[string]time.Time
}

// NewOrphanedAgentsRule builds the rule over an inventory source. The clock
// times how long a row has stayed reapable.
func NewOrphanedAgentsRule(source AgentInventorySource, clock Clock) AlertRule {
	if clock == nil {
		clock = RealClock{}
	}
	return &orphanedAgentsRule{source: source, clock: clock, reapableSince: map[string]time.Time{}}
}

func (*orphanedAgentsRule) ID() string { return RuleOrphanedAgents }

func sessionKey(s AgentSessionRecord) string {
	if s.Scope != "" {
		return s.Scope
	}
	return s.Session
}

func (r *orphanedAgentsRule) Evaluate(ctx context.Context, _ AlertSnapshot) RuleEvaluation {
	if r.source == nil {
		return unsupportedRule("no agent inventory source is configured")
	}
	inventory, err := r.source.ListAgentSessions(ctx)
	if err != nil {
		return failedRule(err.Error())
	}
	if !inventory.ClassifiesOrphans {
		return unsupportedRule(orphanUnsupportedText)
	}
	now := r.clock.Now()

	r.mu.Lock()
	seen := make(map[string]bool, len(inventory.Sessions))
	var stuckReapable []AgentSessionRecord
	for _, session := range inventory.Sessions {
		if !session.ReapableKnown || !session.Reapable {
			continue
		}
		key := sessionKey(session)
		seen[key] = true
		since, ok := r.reapableSince[key]
		if !ok {
			since = now
			r.reapableSince[key] = now
		}
		if now.Sub(since) >= OrphanedAgentAgeLimit {
			stuckReapable = append(stuckReapable, session)
		}
	}
	for key := range r.reapableSince {
		if !seen[key] {
			delete(r.reapableSince, key)
		}
	}
	r.mu.Unlock()

	var orphans, old []AgentSessionRecord
	var knownBytes int64
	unknownMemory := 0
	for _, session := range inventory.Sessions {
		if session.State != "orphaned" {
			continue
		}
		orphans = append(orphans, session)
		if session.AgeKnown && session.Age > OrphanedAgentAgeLimit {
			old = append(old, session)
		}
		if session.MemoryKnown {
			knownBytes += session.MemoryBytes
		} else {
			unknownMemory++
		}
		if session.SwapKnown {
			knownBytes += session.SwapBytes
		}
	}
	heavy := knownBytes >= OrphanedAgentMemoryLimit
	if len(old) == 0 && !heavy && len(stuckReapable) == 0 {
		return clearRule()
	}

	severity := SeverityWarning
	var reasons []string
	if len(old) > 0 {
		reasons = append(reasons, fmt.Sprintf("%d orphaned session(s) older than %s", len(old), OrphanedAgentAgeLimit))
	}
	if heavy {
		severity = SeverityCritical
		reasons = append(reasons, fmt.Sprintf("orphaned sessions hold %.1f GiB of memory+swap", float64(knownBytes)/(1<<30)))
	}
	if len(stuckReapable) > 0 {
		severity = SeverityCritical
		reasons = append(reasons, fmt.Sprintf("%d session(s) reapable for over %s (auto-reap is off or failing)", len(stuckReapable), OrphanedAgentAgeLimit))
	}
	if unknownMemory > 0 {
		reasons = append(reasons, fmt.Sprintf("memory unknown for %d orphaned session(s)", unknownMemory))
	}

	listed := make([]interface{}, 0, len(orphans)+len(stuckReapable))
	included := map[string]bool{}
	for _, group := range [][]AgentSessionRecord{orphans, stuckReapable} {
		for _, session := range group {
			key := sessionKey(session)
			if included[key] {
				continue
			}
			included[key] = true
			listed = append(listed, orphanDetail(session))
		}
	}
	message := strings.Join(reasons, "; ") + ". Reclaim them through the control plane; system-monitor does not terminate processes."
	memoryDetail := interface{}(knownBytes)
	if unknownMemory == len(orphans) && len(orphans) > 0 {
		memoryDetail = "unknown"
	}
	return activeRule(AlertCondition{
		Key:        hostKey,
		Severity:   severity,
		Message:    message,
		MetricName: "orphaned_agent_sessions",
		Value:      float64(len(orphans)),
		Threshold:  OrphanedAgentMemoryLimit,
		Details: map[string]interface{}{
			"sessions":                listed,
			"orphaned":                len(orphans),
			"older_than_limit":        len(old),
			"stuck_reapable":          len(stuckReapable),
			"known_memory_swap_bytes": memoryDetail,
			"memory_unknown_sessions": unknownMemory,
		},
	})
}

func orphanDetail(session AgentSessionRecord) map[string]interface{} {
	row := map[string]interface{}{
		"session":      session.Session,
		"scope":        session.Scope,
		"state":        session.State,
		"kernel_state": session.KernelState,
		"reap_reason":  session.ReapReason,
	}
	if session.Owner != nil {
		row["owner_kind"] = session.Owner.Kind
		row["owner_ref"] = session.Owner.Ref
		row["owner_liveness"] = session.Owner.Liveness
	}
	if session.AgeKnown {
		row["age_seconds"] = session.Age.Seconds()
	}
	row["memory_bytes"] = knownOrUnknown(session.MemoryBytes, session.MemoryKnown, session.MemoryStatus)
	row["swap_bytes"] = knownOrUnknown(session.SwapBytes, session.SwapKnown, session.SwapStatus)
	if session.MemoryStatus != "" {
		row["memory_status"] = session.MemoryStatus
	}
	if session.SwapStatus != "" {
		row["swap_status"] = session.SwapStatus
	}
	if session.ReapableKnown {
		row["reapable"] = session.Reapable
	} else {
		row["reapable"] = "unknown"
	}
	return row
}

func knownOrUnknown(value int64, known bool, status string) interface{} {
	if known {
		return value
	}
	if status != "" {
		return "unknown: " + status
	}
	return "unknown"
}
