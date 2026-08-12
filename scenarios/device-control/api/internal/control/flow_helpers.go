package control

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"device-control/internal/evidence"
)

func sensitiveRegions(step Step) []evidence.Region {
	raw, ok := step.Arguments["sensitive_regions"].([]any)
	if !ok {
		return nil
	}
	regions := make([]evidence.Region, 0, len(raw))
	for _, item := range raw {
		values, ok := item.([]any)
		if !ok || len(values) != 4 {
			continue
		}
		coordinates := make([]int, 4)
		valid := true
		for i, value := range values {
			number, ok := value.(float64)
			if !ok {
				valid = false
				break
			}
			coordinates[i] = int(number)
		}
		if valid {
			regions = append(regions, evidence.Region{X0: coordinates[0], Y0: coordinates[1], X1: coordinates[2], Y1: coordinates[3]})
		}
	}
	return regions
}

func coordinates(step Step) (float64, float64, error) {
	if raw, ok := step.Arguments["x"]; ok {
		x, xok := number(raw)
		y, yok := number(step.Arguments["y"])
		if xok && yok {
			return x, y, nil
		}
	}
	parts := strings.Split(step.Target, ",")
	if len(parts) == 2 {
		x, xerr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		y, yerr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if xerr == nil && yerr == nil {
			return x, y, nil
		}
	}
	return 0, 0, fmt.Errorf("tap step %q requires target coordinates x,y", step.ID)
}

func number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *Service) persistArtifact(id string, data []byte, kind string) error {
	if err := os.MkdirAll(s.evidenceDir, 0o700); err != nil {
		return fmt.Errorf("create evidence store: %w", err)
	}
	path := filepath.Join(s.evidenceDir, id+".bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("retain evidence: %w", err)
	}
	s.mu.Lock()
	s.artifacts[id] = path
	s.artifactKinds[id] = kind
	s.mu.Unlock()
	return nil
}

// Artifact returns a retained evidence payload without exposing its filesystem
// path. The id is deliberately constrained to a single generated filename so
// the HTTP read surface cannot turn into a path traversal primitive.
func (s *Service) Artifact(id string) ([]byte, string, error) {
	if id == "" || filepath.Base(id) != id || strings.ContainsAny(id, `/\\`) {
		return nil, "", fmt.Errorf("invalid artifact id %q", id)
	}
	s.mu.Lock()
	path := s.artifacts[id]
	kind := s.artifactKinds[id]
	s.mu.Unlock()
	if path == "" {
		path = filepath.Join(s.evidenceDir, id+".bin")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read artifact %q: %w", id, err)
	}
	return data, kind, nil
}
