package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeoutPattern = regexp.MustCompile(`^([0-9]+)([smh]?)$`)

type testingConfig struct {
	Phases  map[string]testingPhaseSettings
	Presets map[string][]string
}

type testingPhaseSettings struct {
	Enabled *bool
	Timeout time.Duration
}

type rawTestingConfig struct {
	Phases  map[string]rawPhaseSettings `json:"phases"`
	Presets map[string][]string         `json:"presets"`
}

type rawPhaseSettings struct {
	Enabled *bool  `json:"enabled"`
	Timeout string `json:"timeout"`
}

func loadTestingConfig(scenarioDir string) (*testingConfig, error) {
	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	var raw rawTestingConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	cfg := &testingConfig{
		Phases:  map[string]testingPhaseSettings{},
		Presets: map[string][]string{},
	}

	for name, phase := range raw.Phases {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		timeout, err := parsePhaseTimeout(phase.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout for phase '%s': %w", name, err)
		}
		cfg.Phases[normalized] = testingPhaseSettings{
			Enabled: phase.Enabled,
			Timeout: timeout,
		}
	}

	for preset, phases := range raw.Presets {
		normalized := strings.ToLower(strings.TrimSpace(preset))
		if normalized == "" {
			continue
		}
		filtered := normalizePhaseList(phases)
		if len(filtered) == 0 {
			continue
		}
		cfg.Presets[normalized] = filtered
	}

	if len(cfg.Phases) == 0 && len(cfg.Presets) == 0 {
		return nil, nil
	}
	return cfg, nil
}

func normalizePhaseList(phases []string) []string {
	var filtered []string
	for _, phase := range phases {
		normalized := strings.ToLower(strings.TrimSpace(phase))
		if normalized != "" {
			filtered = append(filtered, normalized)
		}
	}
	return filtered
}

func parsePhaseTimeout(raw string) (time.Duration, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	matches := timeoutPattern.FindStringSubmatch(value)
	if len(matches) != 3 {
		return 0, fmt.Errorf("unsupported timeout value '%s'", raw)
	}
	number, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid timeout number '%s': %w", matches[1], err)
	}
	unit := matches[2]
	switch unit {
	case "", "s":
		return time.Duration(number) * time.Second, nil
	case "m":
		return time.Duration(number) * time.Minute, nil
	case "h":
		return time.Duration(number) * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported timeout unit '%s'", unit)
	}
}
