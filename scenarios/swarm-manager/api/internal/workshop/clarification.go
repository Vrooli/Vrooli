// DOC: docs/internal/SEAMS.md#clarification-storage
//
// Clarification thread storage and impact XML parsing for workshop decision
// items. Threads are stored as JSON files alongside workshop rounds at:
//
//	{itemDir}/workshop/clarifications/round-{NNN}-item-{id}.json
package workshop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Clarification types
// ---------------------------------------------------------------------------

// ClarificationMessage is a single turn in a clarification conversation.
type ClarificationMessage struct {
	Role          string   `json:"role"`                     // "user" | "assistant"
	Content       string   `json:"content"`                  // message text (markdown)
	CreatedAt     string   `json:"created_at"`               // RFC3339
	AttachmentIDs []string `json:"attachment_ids,omitempty"` // uploaded via agent-manager
}

// ClarificationImpact is the structured impact assessment parsed from an
// assistant's clarification response.
type ClarificationImpact struct {
	Level           string `json:"level"`                      // "none" | "decision" | "round"
	Reasoning       string `json:"reasoning"`                  // why this level
	ContextNote     string `json:"context_note"`               // distilled learning for future rounds
	SuggestedUpdate string `json:"suggested_update,omitempty"` // rewritten decision if applicable
}

// ClarificationThread stores the full conversation for a single decision item.
type ClarificationThread struct {
	ID           string                 `json:"id"`
	RoundNumber  int                    `json:"round_number"`
	ItemID       string                 `json:"item_id"`
	RunID        string                 `json:"run_id"` // agent-manager run ID
	Messages     []ClarificationMessage `json:"messages"`
	LatestImpact *ClarificationImpact   `json:"latest_impact,omitempty"`
	Status       string                 `json:"status"`     // "active" | "resolved" | "dismissed"
	CreatedAt    string                 `json:"created_at"` // RFC3339
	UpdatedAt    string                 `json:"updated_at"` // RFC3339
}

// ---------------------------------------------------------------------------
// File I/O
// ---------------------------------------------------------------------------

const clarificationsDir = "clarifications"

// clarificationPath returns the file path for a clarification thread.
func clarificationPath(itemDir string, roundNum int, itemID string) string {
	return filepath.Join(itemDir, "workshop", clarificationsDir,
		fmt.Sprintf("round-%03d-item-%s.json", roundNum, itemID))
}

// LoadClarification reads a single clarification thread from disk.
func LoadClarification(itemDir string, roundNum int, itemID string) (*ClarificationThread, error) {
	data, err := os.ReadFile(clarificationPath(itemDir, roundNum, itemID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read clarification: %w", err)
	}
	var thread ClarificationThread
	if err := json.Unmarshal(data, &thread); err != nil {
		return nil, fmt.Errorf("parse clarification: %w", err)
	}
	return &thread, nil
}

// LoadClarificationByID searches for a thread matching the given ID.
func LoadClarificationByID(itemDir string, threadID string) (*ClarificationThread, error) {
	all, err := LoadAllClarifications(itemDir)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == threadID {
			return &all[i], nil
		}
	}
	return nil, nil
}

// SaveClarification writes a clarification thread to disk.
func SaveClarification(itemDir string, thread *ClarificationThread) error {
	dir := filepath.Join(itemDir, "workshop", clarificationsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create clarifications dir: %w", err)
	}
	data, err := json.MarshalIndent(thread, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal clarification: %w", err)
	}
	path := clarificationPath(itemDir, thread.RoundNumber, thread.ItemID)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write clarification: %w", err)
	}
	return nil
}

// DeleteClarification removes a clarification thread file.
func DeleteClarification(itemDir string, roundNum int, itemID string) error {
	path := clarificationPath(itemDir, roundNum, itemID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete clarification: %w", err)
	}
	return nil
}

// LoadAllClarifications reads all clarification threads for a backlog item.
func LoadAllClarifications(itemDir string) ([]ClarificationThread, error) {
	dir := filepath.Join(itemDir, "workshop", clarificationsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read clarifications dir: %w", err)
	}

	var threads []ClarificationThread
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var thread ClarificationThread
		if err := json.Unmarshal(data, &thread); err != nil {
			continue
		}
		threads = append(threads, thread)
	}

	sort.Slice(threads, func(i, j int) bool {
		if threads[i].RoundNumber != threads[j].RoundNumber {
			return threads[i].RoundNumber < threads[j].RoundNumber
		}
		return threads[i].ItemID < threads[j].ItemID
	})
	return threads, nil
}

// DeleteClarificationsForRound removes all clarification threads for a given
// round number.
func DeleteClarificationsForRound(itemDir string, roundNum int) error {
	dir := filepath.Join(itemDir, "workshop", clarificationsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read clarifications dir: %w", err)
	}

	prefix := fmt.Sprintf("round-%03d-", roundNum)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".json") {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}

// RenumberClarifications renames clarification files when rounds are
// renumbered after a deletion. oldNum is the original round number and
// newNum is the target round number.
func RenumberClarifications(itemDir string, oldNum, newNum int) error {
	dir := filepath.Join(itemDir, "workshop", clarificationsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read clarifications dir: %w", err)
	}

	oldPrefix := fmt.Sprintf("round-%03d-", oldNum)
	newPrefix := fmt.Sprintf("round-%03d-", newNum)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), oldPrefix) && strings.HasSuffix(entry.Name(), ".json") {
			oldPath := filepath.Join(dir, entry.Name())
			newName := newPrefix + strings.TrimPrefix(entry.Name(), oldPrefix)
			newPath := filepath.Join(dir, newName)

			// Also update the round_number inside the JSON.
			data, err := os.ReadFile(oldPath)
			if err != nil {
				continue
			}
			var thread ClarificationThread
			if err := json.Unmarshal(data, &thread); err == nil {
				thread.RoundNumber = newNum
				if updated, err := json.MarshalIndent(thread, "", "  "); err == nil {
					data = updated
				}
			}
			if err := os.WriteFile(newPath, data, 0o644); err != nil {
				continue
			}
			_ = os.Remove(oldPath)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Impact XML parsing
// ---------------------------------------------------------------------------

var (
	impactBlockRe  = regexp.MustCompile(`(?s)<impact\s+level="(none|decision|round)">(.*?)</impact>`)
	reasoningRe    = regexp.MustCompile(`(?s)<reasoning>(.*?)</reasoning>`)
	contextNoteRe  = regexp.MustCompile(`(?s)<context_note>(.*?)</context_note>`)
	suggestedUpdRe = regexp.MustCompile(`(?s)<suggested_update>(.*?)</suggested_update>`)
)

// ParseImpactXML extracts the structured impact assessment from an assistant
// response. Returns nil if no valid impact block is found — this is expected
// and not an error (graceful degradation).
func ParseImpactXML(content string) *ClarificationImpact {
	match := impactBlockRe.FindStringSubmatch(content)
	if match == nil {
		return nil
	}

	impact := &ClarificationImpact{
		Level: match[1],
	}

	block := match[2]
	if m := reasoningRe.FindStringSubmatch(block); m != nil {
		impact.Reasoning = strings.TrimSpace(m[1])
	}
	if m := contextNoteRe.FindStringSubmatch(block); m != nil {
		impact.ContextNote = strings.TrimSpace(m[1])
	}
	if m := suggestedUpdRe.FindStringSubmatch(block); m != nil {
		impact.SuggestedUpdate = strings.TrimSpace(m[1])
	}

	return impact
}

// FormatOptionsForPrompt formats decision options as a readable string for
// injection into a clarification prompt.
func FormatOptionsForPrompt(options []Option) string {
	if len(options) == 0 {
		return "(no options defined)"
	}
	var sb strings.Builder
	for _, opt := range options {
		fmt.Fprintf(&sb, "- **%s**: %s", opt.Key, opt.Label)
		if opt.Recommended {
			sb.WriteString(" *(recommended)*")
		}
		if opt.Rationale != "" {
			fmt.Fprintf(&sb, "\n  Rationale: %s", opt.Rationale)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatClarificationHistory formats prior messages in a thread for injection
// into a continuation prompt.
func FormatClarificationHistory(messages []ClarificationMessage) string {
	if len(messages) == 0 {
		return "(no prior messages)"
	}
	var sb strings.Builder
	for _, msg := range messages {
		fmt.Fprintf(&sb, "**%s** (%s):\n%s\n\n", msg.Role, msg.CreatedAt, msg.Content)
	}
	return sb.String()
}
