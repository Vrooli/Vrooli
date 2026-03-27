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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/jsonutil"
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

// Item is a single decision or info item within a workshop round.
type Item struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`               // "decision" | "info"
	Topic    string   `json:"topic,omitempty"`    // decision: what's being decided
	Text     string   `json:"text,omitempty"`     // info: the informational text
	Context  string   `json:"context,omitempty"`  // background/rationale
	Options  []Option `json:"options,omitempty"`  // decision: lettered choices
	Selected *string  `json:"selected,omitempty"` // decision: chosen option key (e.g. "A")
	Freeform *string  `json:"freeform,omitempty"` // decision: free-text if "Other" selected
	Notes    *string  `json:"notes,omitempty"`    // decision: optional additional context
}

// Option is a single lettered choice within a decision item.
type Option struct {
	Key         string `json:"key"`                   // e.g. "A", "B", "C"
	Label       string `json:"label"`                 // short description
	Rationale   string `json:"rationale"`             // agent's analysis of this option
	Recommended bool   `json:"recommended,omitempty"` // agent's pick
}

// ---------------------------------------------------------------------------
// Readiness dimensions and boost computation
// ---------------------------------------------------------------------------

// ReadinessDimensions are the 5 universal dimensions scored per round.
var ReadinessDimensions = []string{
	"problem_clarity",
	"scope_defined",
	"approach_solid",
	"testable",
	"risk_awareness",
}

// BoostN maps each backlog kind (as string) to its boost divisor N.
// The boost formula is: effective = raw >= 2 ? min(3, raw + floor(rounds/N)) : raw
var BoostN = map[string]int{
	"idea":     2,
	"research": 2,
	"fix":      1,
	"execute":  2,
	"chore":    1,
}

// ComputeEffectiveScores applies the round-based boost formula to raw scores.
func ComputeEffectiveScores(raw map[string]int, roundsCompleted int, kind string) map[string]int {
	n := BoostN[kind]
	if n <= 0 {
		n = 2 // safe default
	}
	boost := 0
	if roundsCompleted > 0 {
		boost = roundsCompleted / n
	}

	effective := make(map[string]int, len(ReadinessDimensions))
	for _, dim := range ReadinessDimensions {
		score := raw[dim]
		if score >= 2 {
			eff := score + boost
			if eff > 3 {
				eff = 3
			}
			effective[dim] = eff
		} else {
			effective[dim] = score
		}
	}
	return effective
}

// IsReady returns true when all effective dimension scores are >= 3.
func IsReady(effective map[string]int) bool {
	for _, dim := range ReadinessDimensions {
		if effective[dim] < 3 {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Workshop round I/O
// ---------------------------------------------------------------------------

// LoadRounds reads all workshop/round-*.json files from the item
// directory, sorted by round number ascending.
func LoadRounds(itemDir string) ([]Round, error) {
	workshopDir := filepath.Join(itemDir, "workshop")
	entries, err := os.ReadDir(workshopDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workshop dir: %w", err)
	}

	var rounds []Round
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "round-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(workshopDir, entry.Name()))
		if err != nil {
			continue
		}
		var round Round
		if err := json.Unmarshal(data, &round); err != nil {
			if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
				if json.Unmarshal(repaired, &round) == nil {
					rounds = append(rounds, round)
				}
			}
			continue
		}
		rounds = append(rounds, round)
	}

	sort.Slice(rounds, func(i, j int) bool {
		return rounds[i].RoundNum < rounds[j].RoundNum
	})
	return rounds, nil
}

// LoadLatestRound returns the most recent workshop round and total round count.
// Returns nil round and 0 count if no rounds exist.
func LoadLatestRound(itemDir string) (*Round, int, error) {
	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return nil, 0, err
	}
	if len(rounds) == 0 {
		return nil, 0, nil
	}
	return &rounds[len(rounds)-1], len(rounds), nil
}

// CountPendingDecisions counts decision items that have not been answered yet.
func CountPendingDecisions(round *Round) int {
	if round == nil {
		return 0
	}
	count := 0
	for _, item := range round.Items {
		if item.Type == "decision" && (item.Selected == nil || strings.TrimSpace(*item.Selected) == "") {
			count++
		}
	}
	return count
}

// CountDecisionItems counts all decision items in a round, answered or not.
func CountDecisionItems(round *Round) int {
	if round == nil {
		return 0
	}
	count := 0
	for _, item := range round.Items {
		if item.Type == "decision" {
			count++
		}
	}
	return count
}

// RoundMode returns the normalized round mode, defaulting to "workshop"
// for legacy rounds that predate explicit mode metadata.
func RoundMode(round *Round) string {
	if round == nil {
		return ""
	}
	mode := strings.ToLower(strings.TrimSpace(round.Mode))
	if mode == "" {
		return "workshop"
	}
	return mode
}

// IsFinalizeRound reports whether the round is an explicit finalize round.
func IsFinalizeRound(round *Round) bool {
	return RoundMode(round) == "finalize"
}

// NeedsSynthesis reports whether the latest round should be followed by a
// finalize pass. This supports both explicit new-format rounds and legacy
// answered rounds that predate the pending_synthesis marker.
func NeedsSynthesis(round *Round) bool {
	if round == nil {
		return false
	}
	if round.PendingSynthesis {
		return true
	}
	if IsFinalizeRound(round) {
		return false
	}
	return CountDecisionItems(round) > 0 && CountPendingDecisions(round) == 0
}

// HasPlan checks whether a plan.md file exists for the item.
func HasPlan(itemDir string) bool {
	return HasPlanByName(itemDir, "plan.md")
}

// HasPlanByName checks whether the named deliverable file exists for the item.
func HasPlanByName(itemDir, filename string) bool {
	_, err := os.Stat(filepath.Join(itemDir, filename))
	return err == nil
}

// LoadPlanContent reads plan.md and returns its content. Returns empty string
// if the file does not exist.
func LoadPlanContent(itemDir string) string {
	return LoadPlanContentByName(itemDir, "plan.md")
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

// RoundFilename returns the standard zero-padded filename for a round number.
func RoundFilename(n int) string {
	return fmt.Sprintf("round-%03d.json", n)
}

// DeleteRoundAndRenumber removes a workshop round file and renumbers all
// subsequent rounds so they remain contiguous. Returns the number of remaining
// rounds after deletion.
func DeleteRoundAndRenumber(itemDir string, roundNum int) (remaining int, err error) {
	workshopDir := filepath.Join(itemDir, "workshop")
	rounds, err := LoadRounds(itemDir)
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
	targetFile := filepath.Join(workshopDir, RoundFilename(roundNum))
	if err := os.Remove(targetFile); err != nil {
		return len(rounds), fmt.Errorf("delete round file: %w", err)
	}

	// Renumber subsequent rounds (ascending order — always writing to a
	// lower slot that is already free).
	for _, r := range rounds {
		if r.RoundNum <= roundNum {
			continue
		}
		oldFile := filepath.Join(workshopDir, RoundFilename(r.RoundNum))
		newNum := r.RoundNum - 1
		r.RoundNum = newNum
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return 0, fmt.Errorf("marshal round %d: %w", r.RoundNum, err)
		}
		newFile := filepath.Join(workshopDir, RoundFilename(newNum))
		if err := os.WriteFile(newFile, data, 0o644); err != nil {
			return 0, fmt.Errorf("write round %d: %w", newNum, err)
		}
		// Only remove old file if it differs from the new path (i.e., not the
		// round immediately after the deleted one when there's a gap).
		if oldFile != newFile {
			_ = os.Remove(oldFile)
		}
	}

	return len(rounds) - 1, nil
}
