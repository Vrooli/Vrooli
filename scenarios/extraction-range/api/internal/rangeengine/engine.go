// Package rangeengine loads the owned practice range's data-defined challenges
// and deterministically scores extraction submissions against their answer key.
package rangeengine

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

//go:embed levels/*.json
var levelFiles embed.FS

type Record map[string]any

type PageSpec struct {
	Title           string         `json:"title"`
	Records         []Record       `json:"records"`
	PageSize        int            `json:"page_size"`
	Variant         int            `json:"variant"`
	SearchFields    []string       `json:"search_fields"`
	RequiresSession bool           `json:"requires_session"`
	SessionCode     string         `json:"session_code"`
	DownloadFormat  string         `json:"download_format"`
	LayoutVariant   string         `json:"layout_variant"`
	Behavior        map[string]any `json:"behavior"`
	BotGate         string         `json:"bot_gate"`
}

type Level struct {
	ID         string   `json:"id"`
	Family     string   `json:"family"`
	FamilyName string   `json:"family_name"`
	Difficulty int      `json:"difficulty"`
	Title      string   `json:"title"`
	Task       string   `json:"task"`
	Mechanism  string   `json:"mechanism"`
	Tags       []string `json:"tags"`
	Page       PageSpec `json:"page"`
	Answer     []Record `json:"answer"`
}

type Result struct {
	Tool      string  `json:"tool"`
	LevelID   string  `json:"level_id"`
	Family    string  `json:"family"`
	Correct   int     `json:"correct"`
	Submitted int     `json:"submitted"`
	Expected  int     `json:"expected"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
	Passed    bool    `json:"passed"`
}

type Engine struct {
	levels []Level
	byID   map[string]Level
	mu     sync.RWMutex
	best   map[string]Result
}

func New() (*Engine, error) {
	entries, err := fs.ReadDir(levelFiles, "levels")
	if err != nil {
		return nil, fmt.Errorf("read embedded levels: %w", err)
	}
	e := &Engine{byID: map[string]Level{}, best: map[string]Result{}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		b, err := levelFiles.ReadFile("levels/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read level %s: %w", entry.Name(), err)
		}
		var level Level
		if err := json.Unmarshal(b, &level); err != nil {
			return nil, fmt.Errorf("decode level %s: %w", entry.Name(), err)
		}
		if level.ID == "" || level.Family == "" || level.Title == "" || len(level.Answer) == 0 || len(level.Page.Records) == 0 {
			return nil, fmt.Errorf("level %s is missing required content", entry.Name())
		}
		if _, exists := e.byID[level.ID]; exists {
			return nil, fmt.Errorf("duplicate level id %q", level.ID)
		}
		e.byID[level.ID] = level
		e.levels = append(e.levels, level)
	}
	sort.Slice(e.levels, func(i, j int) bool { return e.levels[i].ID < e.levels[j].ID })
	return e, nil
}

func (e *Engine) List() []Level { return append([]Level(nil), e.levels...) }

func (e *Engine) Get(id string) (Level, bool) { level, ok := e.byID[id]; return level, ok }

// Score compares complete records and scores precision/recall as an F1 value.
// Correct records are matched as a multiset, so duplicate submissions cannot
// inflate a score and source ordering does not affect correctness.
func (e *Engine) Score(tool, id string, submitted []Record) (Result, error) {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return Result{}, fmt.Errorf("tool is required")
	}
	level, ok := e.byID[id]
	if !ok {
		return Result{}, fmt.Errorf("unknown level %q", id)
	}
	matched := make([]bool, len(level.Answer))
	correct := 0
	for _, got := range submitted {
		for i, want := range level.Answer {
			if !matched[i] && equalRecord(got, want) {
				matched[i] = true
				correct++
				break
			}
		}
	}
	precision, recall := 0.0, 0.0
	if len(submitted) > 0 {
		precision = float64(correct) / float64(len(submitted))
	}
	if len(level.Answer) > 0 {
		recall = float64(correct) / float64(len(level.Answer))
	}
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	result := Result{Tool: tool, LevelID: id, Family: level.Family, Correct: correct, Submitted: len(submitted), Expected: len(level.Answer), Precision: precision, Recall: recall, F1: f1, Passed: correct == len(level.Answer) && len(submitted) == len(level.Answer)}
	key := tool + "\x00" + id
	e.mu.Lock()
	if current, exists := e.best[key]; !exists || result.F1 > current.F1 || (result.F1 == current.F1 && result.Precision > current.Precision) {
		e.best[key] = result
	}
	e.mu.Unlock()
	return result, nil
}

func (e *Engine) Scoreboard() []Result {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Result, 0, len(e.best))
	for _, result := range e.best {
		out = append(out, result)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tool != out[j].Tool {
			return out[i].Tool < out[j].Tool
		}
		return out[i].LevelID < out[j].LevelID
	})
	return out
}

func equalRecord(a, b Record) bool {
	// JSON round-tripping canonicalizes numeric values from embedded data and
	// client-decoded payloads while preserving field and value equality.
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	var lm, rm map[string]any
	if json.Unmarshal(left, &lm) != nil || json.Unmarshal(right, &rm) != nil {
		return false
	}
	return mapsEqual(lm, rm)
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		aa, _ := json.Marshal(av)
		bb, _ := json.Marshal(bv)
		if string(aa) != string(bb) {
			return false
		}
	}
	return true
}
