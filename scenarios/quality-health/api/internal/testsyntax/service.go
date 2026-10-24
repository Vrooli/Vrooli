// Package testsyntax owns bounded, reusable native test-lint observations.
package testsyntax

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Input struct {
	File   string `json:"file"`
	Source string `json:"source"`
	Digest string `json:"sourceDigest"`
}
type Check struct {
	Rule   string `json:"rule"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}
type Diagnostic struct {
	RuleID        string `json:"ruleId"`
	CanonicalRule string `json:"canonicalRule"`
	MessageID     string `json:"messageId"`
	Message       string `json:"message"`
	Line          int    `json:"line"`
	Column        int    `json:"column"`
	EndLine       int    `json:"endLine"`
	EndColumn     int    `json:"endColumn"`
	Severity      int    `json:"severity"`
}
type Observation struct {
	ConfigDigest  string       `json:"configDigest"`
	Schema        string       `json:"schemaVersion"`
	File          string       `json:"file"`
	Digest        string       `json:"sourceDigest"`
	Profile       string       `json:"profile"`
	PluginVersion string       `json:"pluginVersion"`
	ESLintVersion string       `json:"eslintVersion"`
	ParserVersion string       `json:"parserVersion"`
	Status        string       `json:"status"`
	Reason        string       `json:"reason"`
	Checks        []Check      `json:"checks"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
	Limitations   []string     `json:"limitations"`
}
type Runner interface {
	Identity() string
	Run(context.Context, []Input) ([]Observation, error)
}
type flight struct {
	done chan struct{}
	rows []Observation
	err  error
}
type Service struct {
	Runner  Runner
	mu      sync.Mutex
	entries map[string]*flight
}

func (s *Service) Observe(ctx context.Context, root string, files []string) ([]Observation, error) {
	if s.Runner == nil {
		return nil, fmt.Errorf("native lint owner runner unavailable")
	}
	if !filepath.IsAbs(root) || len(files) == 0 || len(files) > 100 {
		return nil, fmt.Errorf("absolute root and 1..100 source files required")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	inputs := make([]Input, 0, len(files))
	seen := map[string]bool{}
	total := 0
	for _, file := range files {
		if !filepath.IsLocal(file) {
			return nil, fmt.Errorf("source must be root-relative: %s", file)
		}
		path, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, file))
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(resolvedRoot, path)
		if err != nil || !filepath.IsLocal(rel) {
			return nil, fmt.Errorf("source escapes root")
		}
		if seen[rel] {
			return nil, fmt.Errorf("duplicate source: %s", rel)
		}
		seen[rel] = true
		if !supportedFile(rel) {
			return nil, fmt.Errorf("unsupported test source: %s", rel)
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source must be a regular file: %s", rel)
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(f, 1024*1024+1))
		f.Close()
		if readErr != nil {
			return nil, readErr
		}
		total += len(data)
		if len(data) > 1024*1024 || total > 8*1024*1024 {
			return nil, fmt.Errorf("source batch exceeds limit")
		}
		digest := sha256.Sum256(data)
		inputs = append(inputs, Input{File: filepath.ToSlash(rel), Source: string(data), Digest: hex.EncodeToString(digest[:])})
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].File < inputs[j].File })
	encoded, _ := json.Marshal(struct {
		Root, Runner string
		Inputs       []Input
	}{resolvedRoot, s.Runner.Identity(), inputs})
	keyBytes := sha256.Sum256(encoded)
	key := hex.EncodeToString(keyBytes[:])
	s.mu.Lock()
	if old := s.entries[key]; old != nil {
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-old.done:
			return clone(old.rows), old.err
		}
	}
	if s.entries == nil {
		s.entries = map[string]*flight{}
	}
	// Bound retained entries without evicting live requests (which would permit
	// duplicate work). Backpressure is explicit when all slots are active.
	if len(s.entries) >= 32 {
		for k, v := range s.entries {
			select {
			case <-v.done:
				delete(s.entries, k)
			default:
			}
		}
		if len(s.entries) >= 32 {
			s.mu.Unlock()
			return nil, fmt.Errorf("native lint observation capacity reached")
		}
	}
	current := &flight{done: make(chan struct{})}
	s.entries[key] = current
	s.mu.Unlock()
	rows, err := s.Runner.Run(ctx, inputs)
	if err == nil {
		err = validate(inputs, rows)
	}
	s.mu.Lock()
	current.rows = clone(rows)
	current.err = err
	if err != nil {
		delete(s.entries, key)
	}
	close(current.done)
	s.mu.Unlock()
	return clone(rows), err
}
func supportedFile(file string) bool {
	base := filepath.Base(file)
	if !strings.Contains(base, ".test.") && !strings.Contains(base, ".spec.") {
		return false
	}
	switch filepath.Ext(base) {
	case ".ts", ".tsx", ".js", ".jsx", ".mts", ".mjs":
		return true
	}
	return false
}
func validate(inputs []Input, rows []Observation) error {
	if len(inputs) != len(rows) {
		return fmt.Errorf("native lint returned incomplete file inventory")
	}
	for i, row := range rows {
		if row.File != inputs[i].File || row.Digest != inputs[i].Digest || row.Schema != "vitest-lint/v1" {
			return fmt.Errorf("native lint identity mismatch")
		}
		if len(row.Diagnostics) > 1000 {
			return fmt.Errorf("native lint diagnostic limit exceeded")
		}
	}
	return nil
}
func clone(rows []Observation) []Observation {
	data, _ := json.Marshal(rows)
	var out []Observation
	_ = json.Unmarshal(data, &out)
	return out
}
