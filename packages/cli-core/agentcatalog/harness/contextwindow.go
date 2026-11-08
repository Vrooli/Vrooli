package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Where a harness reports its model context windows.
const (
	WindowSourceClaudeStatusline = "claude-statusline"
	WindowSourceCodexModelsCache = "codex-models-cache"
)

const (
	// ModelWindowsEnv overrides the Claude status line's window record path.
	ModelWindowsEnv = "VROOLI_CLAUDE_MODEL_WINDOWS"

	maxWindowFileBytes = 16 << 20
	maxRecordedModels  = 128
	maxModelIDLength   = 128
)

// ModelWindowRecordSchemaVersion is the stored context-window record format.
const ModelWindowRecordSchemaVersion = 1

// ModelWindowRecord is the durable model-to-window record the Claude status
// line feed writes. Claude reports the window only for the running session's
// model, so every model seen so far stays here.
type ModelWindowRecord struct {
	SchemaVersion int                    `json:"schema_version"`
	Models        map[string]ModelWindow `json:"models"`
}

// ModelWindow is one model's last reported context window.
type ModelWindow struct {
	ContextWindowTokens int64     `json:"context_window_tokens"`
	ObservedAt          time.Time `json:"observed_at"`
}

// DefaultModelWindowsPath is beside Claude's quota samples.
func DefaultModelWindowsPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(ModelWindowsEnv)); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".vrooli", "state", "vrooli", "claude-code", "model-context-windows.json"), nil
}

// RecordModelWindow stores a reported window. The status line runs on every
// render, so an unchanged value leaves the file alone. A concurrent writer can
// lose an update, which the next render of that model restores.
func RecordModelWindow(path, model string, tokens int64, now time.Time) error {
	model = strings.TrimSpace(model)
	if path == "" || model == "" || len(model) > maxModelIDLength || tokens <= 0 {
		return nil
	}
	record := readModelWindowRecord(path)
	if record.Models[model].ContextWindowTokens == tokens {
		return nil
	}
	record.SchemaVersion = ModelWindowRecordSchemaVersion
	if record.Models == nil {
		record.Models = map[string]ModelWindow{}
	}
	record.Models[model] = ModelWindow{ContextWindowTokens: tokens, ObservedAt: now.UTC()}
	if len(record.Models) > maxRecordedModels {
		ids := make([]string, 0, len(record.Models))
		for id := range record.Models {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return record.Models[ids[i]].ObservedAt.After(record.Models[ids[j]].ObservedAt) })
		for _, id := range ids[maxRecordedModels:] {
			delete(record.Models, id)
		}
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func readModelWindowRecord(path string) ModelWindowRecord {
	var record ModelWindowRecord
	if data, err := readBounded(path); err == nil {
		_ = json.Unmarshal(data, &record)
	}
	return record
}

func readBounded(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxWindowFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxWindowFileBytes {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, maxWindowFileBytes)
	}
	return data, nil
}

func codexModelWindows(path string) map[string]int64 {
	data, err := readBounded(path)
	if err != nil {
		return nil
	}
	var cache struct {
		Models []struct {
			Slug          string `json:"slug"`
			ContextWindow int64  `json:"context_window"`
		} `json:"models"`
	}
	if json.Unmarshal(data, &cache) != nil {
		return nil
	}
	windows := make(map[string]int64, len(cache.Models))
	for _, model := range cache.Models {
		if model.Slug != "" && model.ContextWindow > 0 {
			windows[model.Slug] = model.ContextWindow
		}
	}
	return windows
}

func codexModelsCachePath() (string, error) {
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "models_cache.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "models_cache.json"), nil
}

func (d Descriptor) modelWindows() map[string]int64 {
	switch d.Features.ContextWindowSource {
	case WindowSourceClaudeStatusline:
		path, err := DefaultModelWindowsPath()
		if err != nil {
			return nil
		}
		record := readModelWindowRecord(path)
		windows := make(map[string]int64, len(record.Models))
		for model, window := range record.Models {
			if window.ContextWindowTokens > 0 {
				windows[model] = window.ContextWindowTokens
			}
		}
		return windows
	case WindowSourceCodexModelsCache:
		path, err := codexModelsCachePath()
		if err != nil {
			return nil
		}
		return codexModelWindows(path)
	default:
		return nil
	}
}

// ContextWindowTokens returns the first reported window among modelIDs. Zero
// means unknown: nothing here estimates a window from a model's name, and an
// ID must match exactly, because claude-opus-5-5[1m] and claude-opus-5-5 can
// differ.
func (d Descriptor) ContextWindowTokens(modelIDs ...string) int64 {
	windows := d.modelWindows()
	for _, id := range modelIDs {
		if tokens := windows[id]; tokens > 0 {
			return tokens
		}
	}
	return 0
}

// ContextLimitOptions filters candidates to the limits this harness accepts for
// a window. An unknown window (zero) applies only the harness's own minimum. A
// harness with no context limit gets none.
func (d Descriptor) ContextLimitOptions(windowTokens int64, candidates []int64) []int64 {
	limit := d.Features.ContextLimit
	if limit == nil {
		return nil
	}
	var options []int64
	for _, candidate := range candidates {
		if candidate < limit.Minimum || (windowTokens > 0 && candidate > windowTokens) {
			continue
		}
		options = append(options, candidate)
	}
	return options
}
