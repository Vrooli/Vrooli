package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// maxSkillReadEntries bounds the read log size regardless of age. Reads are
	// far more frequent than discover calls, so this bound is higher.
	maxSkillReadEntries = 20000
	// skillReadRetention matches discoveryCallRetention deliberately: the
	// returned-vs-read ratio joins the two logs, and a shorter window on either
	// side would silently bias the ratio toward whichever log kept more history.
	skillReadRetention = 30 * 24 * time.Hour
)

// SkillRead is one record of a skill being served through POST /skills/read.
//
// Why the server records this rather than the CLI: `skill read` resolves many
// identifiers in one call, and non-CLI callers (UI, MCP, other scenarios) reach
// the same endpoint. Recording at the handler counts every consumer exactly
// once per resolved skill.
type SkillRead struct {
	ID      string `json:"id"`
	At      string `json:"at"` // RFC3339
	SkillID string `json:"skillId"`
	// Caller is the same best-effort attribution string discovery records, so
	// the two logs join on equal terms.
	Caller string `json:"caller,omitempty"`
	// CallerKind is the attribution kind alone (agent-member, operator-direct,
	// writer-skill, investigation) without the member or skill suffix. It is
	// what separates demand from audit traffic: an optimizer reading a skill to
	// audit it must not inflate that skill's demand signal.
	CallerKind string `json:"callerKind,omitempty"`
	// AgentRunID is the agent-manager run this read belongs to, when the caller
	// presented an identity token. Empty for operator sessions.
	AgentRunID string `json:"agentRunId,omitempty"`
	// ViaDiscovery is true when a discover call by the same caller returned this
	// skill shortly before the read. It converts "discovery offered it" into
	// "discovery offered it and the agent took it".
	ViaDiscovery bool `json:"viaDiscovery,omitempty"`
	// DiscoveryCallID names the call that surfaced it, when ViaDiscovery.
	DiscoveryCallID string `json:"discoveryCallId,omitempty"`
}

// SkillReadStore persists skill reads to a bounded, time-windowed JSONL file,
// mirroring DiscoveryCallStore. It lives under the runtime-data root, never the
// git-tracked store tree.
type SkillReadStore struct {
	path       string
	now        func() time.Time
	maxEntries int
	retention  time.Duration
}

// NewSkillReadStore builds a store rooted at runtimeDataDir, resolved by the
// caller through the api-core/storage path layer (no hard-coded ~/.vrooli).
func NewSkillReadStore(runtimeDataDir string) *SkillReadStore {
	return &SkillReadStore{
		path:       filepath.Join(runtimeDataDir, "skill-reads.jsonl"),
		now:        time.Now,
		maxEntries: maxSkillReadEntries,
		retention:  skillReadRetention,
	}
}

// Append records one read, stamping ID/At when absent, then prunes entries
// older than the retention window and trims the file to the size bound.
func (s *SkillReadStore) Append(read SkillRead) error {
	now := s.now().UTC()
	if strings.TrimSpace(read.At) == "" {
		read.At = now.Format(time.RFC3339)
	}
	if strings.TrimSpace(read.ID) == "" {
		read.ID = newCallID(now)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if err := AppendJSONL(s.path, read); err != nil {
		return err
	}
	if err := s.prune(now); err != nil {
		return err
	}
	return trimJSONLLines(s.path, s.maxEntries)
}

// ReadSince returns reads with an At timestamp within the given window
// (relative to now), newest entries last (file order).
func (s *SkillReadStore) ReadSince(window time.Duration) ([]SkillRead, error) {
	all, err := s.readAll()
	if err != nil {
		return nil, err
	}
	if window <= 0 {
		return all, nil
	}
	cutoff := s.now().UTC().Add(-window)
	out := make([]SkillRead, 0, len(all))
	for _, read := range all {
		if at, err := time.Parse(time.RFC3339, read.At); err == nil && at.Before(cutoff) {
			continue
		}
		out = append(out, read)
	}
	return out, nil
}

func (s *SkillReadStore) prune(now time.Time) error {
	all, err := s.readAll()
	if err != nil {
		return err
	}
	cutoff := now.Add(-s.retention)
	kept := make([]SkillRead, 0, len(all))
	dropped := false
	for _, read := range all {
		if at, err := time.Parse(time.RFC3339, read.At); err == nil && at.Before(cutoff) {
			dropped = true
			continue
		}
		kept = append(kept, read)
	}
	if !dropped {
		return nil
	}
	return s.rewrite(kept)
}

func (s *SkillReadStore) readAll() ([]SkillRead, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []SkillRead
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var read SkillRead
		if err := json.Unmarshal([]byte(line), &read); err != nil {
			continue // skip malformed lines rather than failing the read
		}
		out = append(out, read)
	}
	return out, nil
}

func (s *SkillReadStore) rewrite(entries []SkillRead) error {
	var b strings.Builder
	for _, entry := range entries {
		raw, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	return os.WriteFile(s.path, []byte(b.String()), 0o644)
}
