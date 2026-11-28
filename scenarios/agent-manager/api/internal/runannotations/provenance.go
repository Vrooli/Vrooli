package runannotations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"agent-manager/internal/domain"
)

// This file separates a person's words from text the system wrote in the
// operator's seat. Most runs in the corpus are opened by a workflow brief or an
// agent's delegation, so reading every operator-source message as the
// operator's mood would measure the briefs. The rule is pack data
// (runtime.provenance); this is the evaluator.

// Hash fingerprints the policy. A stored provenance row records the hash it was
// computed under, so editing the policy makes the old rows stale (counted apart
// and recomputed) instead of silently mixing two definitions of whose words a
// message was.
func (p Provenance) Hash() string {
	body, _ := json.Marshal(struct {
		Brief, Seed, Markers, Openings []string
		Lineage                        bool
	}{p.BriefWorkloads, p.SeedBriefWorkloads, p.AgentMarkers, p.DelegatingOpenings, p.DelegatedByLineage})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])[:16]
}

// provenanceSets is a Provenance resolved for lookup.
type provenanceSets struct {
	brief, seed map[string]bool
}

func (p Provenance) sets() provenanceSets {
	out := provenanceSets{brief: map[string]bool{}, seed: map[string]bool{}}
	for _, kind := range p.BriefWorkloads {
		out.brief[kind] = true
	}
	for _, kind := range p.SeedBriefWorkloads {
		out.seed[kind] = true
	}
	return out
}

// openedByAgent reports whether the run's opening message says an agent
// launched it.
func (p Provenance) openedByAgent(opening string) bool {
	for _, marker := range p.delegatingOpenings {
		if marker.MatchString(opening) {
			return true
		}
	}
	return false
}

// matchesAgentMarker reports whether text carries a marker only an agent or a
// supervisor writes.
func (p Provenance) matchesAgentMarker(text string) bool {
	for _, marker := range p.agentMarkers {
		if marker.MatchString(text) {
			return true
		}
	}
	return false
}

// Apply classifies a run's operator-source messages and fills the facts'
// human messages and brief count. A message is not a person's when:
//   - the run's workload kind is a brief workload;
//   - it is the run's seed in a seed-brief workload;
//   - another run spawned this one (an agent opened it, so an agent wrote every
//     operator-source message, whoever's token delivered it), unless the run is
//     an imported transcript, which is a person's own session whatever it was
//     later linked under;
//   - the run's opening message says an agent launched it, or
//   - its text carries an agent or supervisor marker.
//
// A message that records its sender (the verified caller who delivered it) is
// classified by it alone: person is a person, agent is not, whatever the run's
// lineage. Everything else the operator source reports is a person's text.
func (p Provenance) Apply(facts *RunFacts) {
	history := facts.OperatorHistory
	if len(history) == 0 {
		history = facts.OperatorMessages
	}
	sets := p.sets()
	kind := facts.Workload.Kind
	delegated := p.DelegatedByLineage && facts.Lineage.Spawned() || p.openedByAgent(facts.Lineage.Opening)
	human := make([]MessageFact, 0, len(history))
	humanIDs := make(map[string]bool, len(history))
	for index := range history {
		message := history[index]
		// A recorded sender is the authoritative answer: a person typing into a
		// run an agent spawned is a person, an agent steering a run is not.
		// Without one (imported text, harness text, rows from before senders were
		// recorded) the lineage and marker rules decide.
		var isBrief bool
		switch message.Sender {
		case domain.MessageSenderPerson:
			isBrief = false
		case domain.MessageSenderAgent:
			isBrief = true
		default:
			isBrief = sets.brief[kind] || (sets.seed[kind] && message.Seed) || delegated || p.matchesAgentMarker(message.Text)
		}
		if isBrief {
			continue
		}
		message.Human = true
		human = append(human, message)
		humanIDs[message.EventID] = true
	}
	facts.HumanMessages = human
	facts.BriefMessages = len(history) - len(human)
	// The digest tail shares the classification, so a renderer or a CEL guard
	// reading either list agrees on whose words they are.
	marked := make([]MessageFact, len(facts.OperatorMessages))
	for index, message := range facts.OperatorMessages {
		message.Human = humanIDs[message.EventID]
		marked[index] = message
	}
	facts.OperatorMessages = marked
}

// ProvenanceRow is one run's stored provenance.
type ProvenanceRow struct {
	RunID            string
	WorkloadKind     string
	OperatorMessages int
	HumanMessages    int
	BriefMessages    int
	LastHumanAt      time.Time
	UpdatedAt        time.Time
	// PolicyHash is the Provenance.Hash the row was computed under.
	PolicyHash string
}

// SaveProvenance upserts one run's provenance. It is written as a by-product of
// annotating a run (the facts are already read), never by a scan.
func (s *Store) SaveProvenance(ctx context.Context, row ProvenanceRow) error {
	last := ""
	if !row.LastHumanAt.IsZero() {
		last = formatTime(row.LastHumanAt)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO run_annotation_provenance (run_id, workload_kind, operator_messages, human_messages, brief_messages, last_human_at, updated_at, policy_hash)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (run_id) DO UPDATE SET
  workload_kind = excluded.workload_kind, operator_messages = excluded.operator_messages,
  human_messages = excluded.human_messages, brief_messages = excluded.brief_messages,
  last_human_at = excluded.last_human_at, updated_at = excluded.updated_at, policy_hash = excluded.policy_hash`,
		row.RunID, row.WorkloadKind, row.OperatorMessages, row.HumanMessages, row.BriefMessages, last, formatTime(row.UpdatedAt), row.PolicyHash)
	if err != nil {
		return fmt.Errorf("save run provenance %s: %w", row.RunID, err)
	}
	return nil
}

// ProvenanceSummary counts the annotated runs in a window by whose text they
// carried, under the current policy. It is the denominator for an affect label:
// a rate of frustration among runs where nobody spoke is not a rate of anything.
// Rows computed under an older policy are counted in RunsStale and in no other
// bucket until they are recomputed.
type ProvenanceSummary struct {
	RunsAnnotated      int `json:"runs_annotated"`
	RunsHumanText      int `json:"runs_human_text"`
	RunsBriefOnly      int `json:"runs_brief_only"`
	RunsNoOperatorText int `json:"runs_no_operator_text"`
	HumanMessages      int `json:"human_messages"`
	BriefMessages      int `json:"brief_messages"`
	// RunsStale are annotated runs whose provenance was computed under another
	// policy; `annotations provenance` recomputes them.
	RunsStale int `json:"runs_stale,omitempty"`
}

// ProvenanceSummary aggregates run provenance rows last updated in the window.
func (s *Store) ProvenanceSummary(ctx context.Context, since, until time.Time, policyHash string) (ProvenanceSummary, error) {
	where := `1 = 1`
	var args []any
	if !since.IsZero() {
		where += ` AND updated_at >= ?`
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		where += ` AND updated_at <= ?`
		args = append(args, formatTime(until))
	}
	current := func(column string) string {
		return "CASE WHEN policy_hash = ? THEN " + column + " ELSE 0 END"
	}
	var summary ProvenanceSummary
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*),
  COALESCE(SUM(CASE WHEN policy_hash = ? AND human_messages > 0 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN policy_hash = ? AND human_messages = 0 AND brief_messages > 0 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN policy_hash = ? AND operator_messages = 0 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(`+current("human_messages")+`), 0), COALESCE(SUM(`+current("brief_messages")+`), 0),
  COALESCE(SUM(CASE WHEN policy_hash <> ? THEN 1 ELSE 0 END), 0)
FROM run_annotation_provenance WHERE `+where,
		append([]any{policyHash, policyHash, policyHash, policyHash, policyHash, policyHash}, args...)...).Scan(
		&summary.RunsAnnotated, &summary.RunsHumanText, &summary.RunsBriefOnly,
		&summary.RunsNoOperatorText, &summary.HumanMessages, &summary.BriefMessages, &summary.RunsStale)
	if err != nil {
		return ProvenanceSummary{}, fmt.Errorf("run provenance summary: %w", err)
	}
	return summary, nil
}

// StaleProvenanceRuns lists up to limit annotated runs whose provenance was
// computed under another policy, oldest first.
func (s *Store) StaleProvenanceRuns(ctx context.Context, policyHash string, limit int) ([]string, error) {
	var ids []string
	err := s.db.SelectContext(ctx, &ids, `
SELECT run_id FROM run_annotation_provenance WHERE policy_hash <> ? ORDER BY updated_at LIMIT ?`, policyHash, limit)
	if err != nil {
		return nil, fmt.Errorf("list stale provenance: %w", err)
	}
	return ids, nil
}
