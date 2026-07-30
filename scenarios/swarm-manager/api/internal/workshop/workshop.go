// DOC: docs/concepts/ARCHITECTURE.md#workshop-refinement
// DOC: docs/internal/SEAMS.md#workshop-computation
//
// Package workshop provides types and computation for the universal workshop
// refinement system. It is a shared package imported by both backlog and
// execution to avoid import cycles.
package workshop

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"swarm-manager/internal/attempt"
	"swarm-manager/internal/attemptstore"
)

// ---------------------------------------------------------------------------
// Workshop types
// ---------------------------------------------------------------------------

// Round represents a single workshop round stored on disk.
type Round struct {
	RoundNum         int            `json:"round"`
	GeneratedAt      string         `json:"generated_at"`
	Mode             string         `json:"mode,omitempty"`
	PendingSynthesis bool           `json:"pending_synthesis,omitempty"`
	Readiness        map[string]int `json:"readiness"`
	Items            []Item         `json:"items"`
	PlanUpdates      string         `json:"plan_updates,omitempty"`
}

func (r Round) RoundNumber() int { return r.RoundNum }

// AsAttempt projects a legacy workshop round into the shared attempt shape.
// Workshop-specific question rendering remains local, but operators and
// cross-domain read models no longer need a second lifecycle vocabulary.
func (r Round) AsAttempt(subjectRef string) attempt.Attempt {
	readiness, _ := json.Marshal(r.Readiness)
	proposals := make([]attempt.Proposal, 0, len(r.Items))
	for _, item := range r.Items {
		payload, _ := json.Marshal(item)
		proposals = append(proposals, attempt.Proposal{ID: item.ID, Type: item.Type, Payload: string(payload)})
	}
	return attempt.Attempt{SubjectKind: "backlog-item", SubjectRef: subjectRef, TransitionKey: "plan.workshop", RoundNum: r.RoundNum, Status: "complete", GeneratedAt: r.GeneratedAt, Assessment: string(readiness), Proposals: proposals}
}

// Item is a single decision or info item within a workshop round.
type Item struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`                       // "decision" | "info"
	Topic           string   `json:"topic,omitempty"`            // decision: what's being decided
	Text            string   `json:"text,omitempty"`             // info: the informational text
	Context         string   `json:"context,omitempty"`          // background/rationale
	Options         []Option `json:"options,omitempty"`          // decision: lettered choices
	Selected        *string  `json:"selected,omitempty"`         // decision: chosen option key (e.g. "A")
	Freeform        *string  `json:"freeform,omitempty"`         // decision: free-text if "Other" selected
	Notes           *string  `json:"notes,omitempty"`            // decision: optional additional context
	ContextNote     *string  `json:"context_note,omitempty"`     // distilled from clarification
	ClarificationID *string  `json:"clarification_id,omitempty"` // thread ID for full conversation access
}

// Option is a single lettered choice within a decision item.
type Option struct {
	Key         string `json:"key"`                   // e.g. "A", "B", "C"
	Label       string `json:"label"`                 // short description
	Rationale   string `json:"rationale"`             // agent's analysis of this option
	Recommended bool   `json:"recommended,omitempty"` // agent's pick
}

// OtherKey is the sentinel value for the "Other" option in workshop decisions.
// When selected, the user must provide a freeform explanation.
const OtherKey = "__other__"

// ---------------------------------------------------------------------------
// Workshop round I/O
// ---------------------------------------------------------------------------

// LoadRounds reads historical legacy round files from an item directory,
// sorted by round number ascending. It is a read-only compatibility parser;
// current Plan Workshop state is stored separately.
func ReadRounds(itemDir string) ([]Round, error) {
	return attemptstore.LoadRounds(itemDir, "workshop", decodeRound)
}

// LoadLatestRound returns the most recent workshop round and total round count.
// Returns nil round and 0 count if no rounds exist.
func ReadLatestRound(itemDir string) (*Round, int, error) {
	rounds, err := ReadRounds(itemDir)
	if err != nil {
		return nil, 0, err
	}
	if len(rounds) == 0 {
		return nil, 0, nil
	}
	return &rounds[len(rounds)-1], len(rounds), nil
}

// HasPlanByName checks whether the named deliverable file exists for the item.
func HasPlanByName(itemDir, filename string) bool {
	_, err := os.Stat(filepath.Join(itemDir, filename))
	return err == nil
}

// LoadPlanContentByName reads the named deliverable file and returns its content.
// Returns empty string if the file does not exist.
func LoadPlanContentByName(itemDir, filename string) string {
	data, err := os.ReadFile(filepath.Join(itemDir, filename))
	if err != nil {
		return ""
	}
	return string(data)
}

// BuildHistory serializes all rounds as a JSON array string for passing
// to prompt-manager as a variable.
func BuildHistory(rounds []Round) string {
	if len(rounds) == 0 {
		return "[]"
	}
	data, err := json.Marshal(rounds)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func decodeRound(data []byte) (Round, error) {
	var round Round
	return round, json.Unmarshal(data, &round)
}

// DeleteRoundAndRenumber removes a workshop round file and renumbers all
// subsequent rounds so they remain contiguous. Returns the number of remaining
// rounds after deletion.
func DeleteRoundAndRenumber(itemDir string, roundNum int) (remaining int, err error) {
	workshopDir := filepath.Join(itemDir, "workshop")
	rounds, err := ReadRounds(itemDir)
	if err != nil {
		return 0, fmt.Errorf("load rounds: %w", err)
	}

	// Verify the target round exists.
	found := false
	for _, r := range rounds {
		if r.RoundNum == roundNum {
			found = true
			break
		}
	}
	if !found {
		return len(rounds), fmt.Errorf("round %d not found", roundNum)
	}

	// Delete the target round file.
	targetFile := filepath.Join(workshopDir, attemptstore.RoundFilename(roundNum))
	if err := os.Remove(targetFile); err != nil {
		return len(rounds), fmt.Errorf("delete round file: %w", err)
	}

	// Renumber subsequent rounds (ascending order — always writing to a
	// lower slot that is already free).
	for _, r := range rounds {
		if r.RoundNum <= roundNum {
			continue
		}
		oldFile := filepath.Join(workshopDir, attemptstore.RoundFilename(r.RoundNum))
		newNum := r.RoundNum - 1
		r.RoundNum = newNum
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return 0, fmt.Errorf("marshal round %d: %w", r.RoundNum, err)
		}
		newFile := filepath.Join(workshopDir, attemptstore.RoundFilename(newNum))
		if err := os.WriteFile(newFile, data, 0o600); err != nil {
			return 0, fmt.Errorf("write round %d: %w", newNum, err)
		}
		// Only remove old file if it differs from the new path (i.e., not the
		// round immediately after the deleted one when there's a gap).
		if oldFile != newFile {
			if rmErr := os.Remove(oldFile); rmErr != nil && !os.IsNotExist(rmErr) {
				slog.Debug("workshop: remove old round file failed", "err", rmErr, "path", oldFile)
			}
		}
	}

	return len(rounds) - 1, nil
}

// ResetWorkshop removes all workshop data from an item directory:
// the workshop/ directory (rounds, clarifications, attachments) and the named
// local deliverable file, when one still exists.
// Returns the number of rounds that existed before deletion.
func ResetWorkshop(itemDir string, deliverableFile string) (deletedRounds int, err error) {
	rounds, err := ReadRounds(itemDir)
	if err != nil {
		return 0, fmt.Errorf("load rounds: %w", err)
	}

	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.RemoveAll(workshopDir); err != nil {
		return 0, fmt.Errorf("remove workshop dir: %w", err)
	}

	// Remove deliverable file at item root (ignore if absent).
	if deliverableFile != "" {
		deliverablePath := filepath.Join(itemDir, deliverableFile)
		if rmErr := os.Remove(deliverablePath); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.Debug("workshop: remove deliverable failed", "err", rmErr, "path", deliverablePath)
		}
	}

	return len(rounds), nil
}

// RoundSummary captures the aggregate decision-item counts for a single
// workshop round. It feeds the recommendation-acceptance metric: callers
// emit it as part of decision.workshop_round_completed so the stats
// engine can compute global and per-kind acceptance rates without
// re-reading the round file.
type RoundSummary struct {
	ItemsTotal             int
	ItemsAnswered          int
	ItemsRecommendedChosen int
	ItemsFreeformChosen    int
}

// SummarizeRound walks the decision items in a workshop round and returns
// the counters used by recommendation-acceptance stats.
//
// Counting rules (per the recommendation-acceptance plan, §8 contract):
//   - Only Type == "decision" items count.
//   - An item with Selected == nil is unanswered: increments ItemsTotal only.
//   - An item with Selected == OtherKey counts toward ItemsAnswered and
//     ItemsFreeformChosen, never ItemsRecommendedChosen — picking "Other"
//     rejects the recommended option set.
//   - An item with Selected pointing at a non-Other option counts toward
//     ItemsAnswered, and toward ItemsRecommendedChosen iff the matching
//     option's Recommended flag is true.
func SummarizeRound(round *Round) RoundSummary {
	var s RoundSummary
	if round == nil {
		return s
	}
	for i := range round.Items {
		item := &round.Items[i]
		if item.Type != "decision" {
			continue
		}
		s.ItemsTotal++
		if item.Selected == nil {
			continue
		}
		s.ItemsAnswered++
		selected := *item.Selected
		if selected == OtherKey {
			s.ItemsFreeformChosen++
			continue
		}
		for j := range item.Options {
			if item.Options[j].Key == selected && item.Options[j].Recommended {
				s.ItemsRecommendedChosen++
				break
			}
		}
	}
	return s
}
