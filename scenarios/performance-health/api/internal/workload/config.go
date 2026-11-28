// Package workload owns declared out-of-band latency workloads and their
// applicable evidence. It does not run browsers inside the performance gate.
package workload

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"performance-health/internal/scenarioroot"
)

var slug = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,79}$`)

// Definition is authored in performance.workloads, alongside existing budgets.
// Requests select a name; none of these fields can be overridden by an RPC.
type Definition struct {
	Command        []string          `json:"command"`
	Directory      string            `json:"directory"`
	Sources        []string          `json:"sources"`
	Contract       string            `json:"contract"`
	Endpoints      map[string]string `json:"endpoints"`
	Samples        int               `json:"samples"`
	Warmups        int               `json:"warmups"`
	P95MaxMS       float64           `json:"p95_max_ms"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	MaxAgeSeconds  int               `json:"max_age_seconds"`
	MinCPUCores    int32             `json:"min_cpu_cores"`
	MinMemoryBytes int64             `json:"min_memory_bytes"`
}

type declaration struct {
	Definition
	root, name, configDigest, producerDigest, contractDigest string
}

func declarations(repoRoot, scenario string) (map[string]declaration, error) {
	if !slug.MatchString(scenario) {
		return nil, errors.New("invalid scenario name")
	}
	root, err := scenarioroot.Resolve(repoRoot, scenario, "")
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(root, ".vrooli/testing.json"))
	if errors.Is(err, os.ErrNotExist) {
		return map[string]declaration{}, nil
	}
	if err != nil {
		return nil, err
	}
	var config struct {
		Performance struct {
			Workloads map[string]json.RawMessage `json:"workloads"`
		} `json:"performance"`
	}
	if err := json.Unmarshal(b, &config); err != nil {
		return nil, err
	}
	out := make(map[string]declaration, len(config.Performance.Workloads))
	if len(config.Performance.Workloads) > 16 {
		return nil, errors.New("at most 16 workloads may be declared")
	}
	for name, raw := range config.Performance.Workloads {
		var d Definition
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&d); err != nil {
			return nil, fmt.Errorf("workload %s: %w", name, err)
		}
		if err := validateDefinition(name, d); err != nil {
			return nil, err
		}
		producer, contract, err := sourceDigests(root, d)
		if err != nil {
			return nil, fmt.Errorf("workload %s: %w", name, err)
		}
		canonical, _ := json.Marshal(d)
		out[name] = declaration{d, root, name, digest(canonical), producer, contract}
	}
	return out, nil
}

func validateDefinition(name string, d Definition) error {
	if !slug.MatchString(name) || len(d.Command) == 0 || len(d.Command) > 32 || len(d.Sources) == 0 || len(d.Sources) > 100 || d.Directory == "" || d.Contract == "" {
		return fmt.Errorf("workload %s requires a name, command, directory, contract and producer sources", name)
	}
	if d.Samples < 1 || d.Samples > 10000 || d.Warmups < 0 || d.Warmups > 10 || !(d.P95MaxMS > 0) || d.P95MaxMS > 3600000 || d.TimeoutSeconds < 1 || d.TimeoutSeconds > 600 || d.MaxAgeSeconds < 1 || d.MaxAgeSeconds > 604800 {
		return fmt.Errorf("workload %s has an invalid sample denominator, latency budget or time bound", name)
	}
	if d.Endpoints["api_url"] != "API_PORT" {
		return errors.New("workload must declare api_url as API_PORT for candidate health")
	}
	if d.MinCPUCores < 0 || d.MinMemoryBytes < 0 {
		return errors.New("workload hardware minimums cannot be negative")
	}
	for _, arg := range d.Command {
		if strings.TrimSpace(arg) == "" {
			return errors.New("empty workload command argument")
		}
	}
	return nil
}

func sourceDigests(root string, d Definition) (string, string, error) {
	if _, err := confinedPath(root, d.Directory); err != nil {
		return "", "", err
	}
	paths := append([]string(nil), d.Sources...)
	sort.Strings(paths)
	var source bytes.Buffer
	for i, rel := range paths {
		if i > 0 && paths[i-1] == rel {
			return "", "", errors.New("duplicate producer source")
		}
		path, err := confinedPath(root, rel)
		if err != nil {
			return "", "", err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		fmt.Fprintf(&source, "%s\x00%s\x00", filepath.ToSlash(rel), b)
	}
	path, err := confinedPath(root, d.Contract)
	if err != nil {
		return "", "", err
	}
	b, err := os.ReadFile(path)
	return digest(source.Bytes()), digest(b), err
}

func confinedPath(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", errors.New("workload paths must be scenario-relative")
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, rel))
	if err != nil {
		return "", err
	}
	actualRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.Rel(actualRoot, path)
	if err != nil || resolved == ".." || strings.HasPrefix(resolved, ".."+string(filepath.Separator)) {
		return "", errors.New("workload path escapes its scenario")
	}
	return path, nil
}

func digest(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
