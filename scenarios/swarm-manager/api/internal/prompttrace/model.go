package prompttrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/storage"
)

// Trace captures the prompt selected and rendered for a specific run.
type Trace struct {
	SkillID      string            `json:"skill_id"`
	Purpose      string            `json:"purpose"`
	Variables    map[string]string `json:"variables,omitempty"`
	Prompt       string            `json:"prompt"`
	UsedFallback bool              `json:"used_fallback"`
	CapturedAt   string            `json:"captured_at"`
}

func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func Save(path string, trace Trace) error {
	if strings.TrimSpace(trace.CapturedAt) == "" {
		trace.CapturedAt = NowRFC3339()
	}
	return storage.WriteJSONAtomic(path, trace)
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
