// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package prompttrace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"swarm-manager/internal/storage"
	"time"
)

// Trace captures the prompt selected and rendered for a specific run.
type Trace struct {
	SkillID        string            `json:"skill_id"`
	Purpose        string            `json:"purpose"`
	Variables      map[string]string `json:"variables,omitempty"`
	Prompt         string            `json:"prompt"`
	PromptRevision string            `json:"prompt_revision,omitempty"`
	UsedFallback   bool              `json:"used_fallback"`
	CapturedAt     string            `json:"captured_at"`
	ExperimentID   string            `json:"experiment_id,omitempty"`
	VariantID      string            `json:"variant_id,omitempty"`
}

func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func Save(path string, trace Trace) error {
	if strings.TrimSpace(trace.CapturedAt) == "" {
		trace.CapturedAt = NowRFC3339()
	}
	if strings.TrimSpace(trace.PromptRevision) == "" {
		trace.PromptRevision = PromptRevision(trace.Prompt)
	}
	return storage.WriteJSONAtomic(path, trace)
}

func PromptRevision(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func Load(path string) (Trace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Trace{}, err
	}
	var trace Trace
	if err := json.Unmarshal(data, &trace); err != nil {
		return Trace{}, err
	}
	return trace, nil
}

func ResearchTracePath(itemDir string) string {
	return filepath.Join(itemDir, ".swarm", "last-research-prompt-trace.json")
}
