package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const policyModelStateSchema = 1
const maxPolicyModelStateBytes = 1 << 20

type policyModelState struct {
	SchemaVersion int      `json:"schema_version"`
	Models        []string `json:"models"`
}

func readPolicyModelState(configPath string) ([]string, error) {
	path := configPath + ".ollama-policy-models.json"
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect OpenCode Ollama policy ownership state: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("OpenCode Ollama policy ownership state is not a regular file")
	}
	if info.Size() <= 0 || info.Size() > maxPolicyModelStateBytes {
		return nil, errors.New("OpenCode Ollama policy ownership state has an invalid size")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OpenCode Ollama policy ownership state: %w", err)
	}
	if len(data) == 0 || len(data) > maxPolicyModelStateBytes {
		return nil, errors.New("OpenCode Ollama policy ownership state has an invalid size")
	}
	var state policyModelState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode OpenCode Ollama policy ownership state: %w", err)
	}
	if state.SchemaVersion != policyModelStateSchema {
		return nil, fmt.Errorf("unsupported OpenCode Ollama policy ownership schema %d", state.SchemaVersion)
	}
	for _, model := range state.Models {
		if !validPolicyModelID(model) {
			return nil, fmt.Errorf("OpenCode Ollama policy ownership state contains invalid model identity %q", model)
		}
	}
	return mergeModelIDs(state.Models), nil
}

func writePolicyModelState(ctx context.Context, configPath string, models []string) error {
	path := configPath + ".ollama-policy-models.json"
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("OpenCode Ollama policy ownership state is not a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect OpenCode Ollama policy ownership state before write: %w", err)
	}
	models = mergeModelIDs(models)
	state := policyModelState{SchemaVersion: policyModelStateSchema, Models: models}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode OpenCode Ollama policy ownership state: %w", err)
	}
	if len(data) > maxPolicyModelStateBytes {
		return errors.New("OpenCode Ollama policy ownership state exceeds its size limit")
	}
	if existing, err := os.ReadFile(path); err == nil && string(existing) == string(data) {
		return nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read OpenCode Ollama policy ownership state before write: %w", err)
	}
	if err := WriteConfigAtomically(ctx, path, data); err != nil {
		return fmt.Errorf("write OpenCode Ollama policy ownership state: %w", err)
	}
	return nil
}

func mergeModelIDs(groups ...[]string) []string {
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, model := range group {
			model = strings.TrimSpace(model)
			if !validPolicyModelID(model) {
				continue
			}
			seen[model] = struct{}{}
		}
	}
	models := make([]string, 0, len(seen))
	for model := range seen {
		models = append(models, model)
	}
	sort.Strings(models)
	return models
}

func validPolicyModelID(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" || len(model) > 512 || strings.ContainsAny(model, "\x00\r\n") {
		return false
	}
	return true
}
