package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type requirementsSyncer interface {
	Sync(ctx context.Context, input requirementsSyncInput) error
}

type requirementsSyncInput struct {
	ScenarioName     string
	ScenarioDir      string
	PhaseDefinitions []phaseDefinition
	PhaseResults     []PhaseExecutionResult
	CommandHistory   []string
}

type nodeRequirementsSyncer struct {
	projectRoot string
	scriptPath  string
}

func newNodeRequirementsSyncer(projectRoot string) requirementsSyncer {
	scriptPath := filepath.Join(projectRoot, "scripts", "requirements", "report.js")
	if _, err := os.Stat(scriptPath); err != nil {
		return nil
	}
	return &nodeRequirementsSyncer{
		projectRoot: projectRoot,
		scriptPath:  scriptPath,
	}
}

func (s *nodeRequirementsSyncer) Sync(ctx context.Context, input requirementsSyncInput) error {
	if input.ScenarioName == "" {
		return nil
	}
	requirementsDir := filepath.Join(input.ScenarioDir, "requirements")
	if _, err := os.Stat(requirementsDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("requirements directory validation failed: %w", err)
	}

	if err := ensureCommandAvailable("node"); err != nil {
		return fmt.Errorf("node command not available: %w", err)
	}

	phasePayload := buildPhaseStatusPayload(input.PhaseDefinitions, input.PhaseResults)
	if len(phasePayload) == 0 {
		return fmt.Errorf("phase execution metadata missing")
	}
	phaseJSON, err := json.Marshal(phasePayload)
	if err != nil {
		return fmt.Errorf("failed to encode phase execution metadata: %w", err)
	}

	commandHistory := input.CommandHistory
	if len(commandHistory) == 0 {
		commandHistory = []string{fmt.Sprintf("suite %s", input.ScenarioName)}
	}
	commandJSON, err := json.Marshal(commandHistory)
	if err != nil {
		return fmt.Errorf("failed to encode command history: %w", err)
	}

	cmd := exec.CommandContext(ctx, "node", s.scriptPath, "--scenario", input.ScenarioName, "--mode", "sync")
	cmd.Dir = s.projectRoot
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("REQUIREMENTS_SYNC_PHASE_STATUS=%s", string(phaseJSON)),
		fmt.Sprintf("REQUIREMENTS_SYNC_TEST_COMMANDS=%s", string(commandJSON)),
	)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("requirements sync command failed: %w: %s", err, strings.TrimSpace(output.String()))
	}
	return nil
}

type phaseStatusEntry struct {
	Phase    string `json:"phase"`
	Status   string `json:"status"`
	Optional bool   `json:"optional"`
	Recorded bool   `json:"recorded"`
}

func buildPhaseStatusPayload(defs []phaseDefinition, results []PhaseExecutionResult) []phaseStatusEntry {
	if len(defs) == 0 {
		return nil
	}
	payload := make([]phaseStatusEntry, 0, len(defs))
	resultLookup := make(map[string]PhaseExecutionResult, len(results))
	for _, result := range results {
		key := strings.ToLower(strings.TrimSpace(result.Name))
		if key == "" {
			continue
		}
		resultLookup[key] = result
	}
	for _, def := range defs {
		key := strings.ToLower(strings.TrimSpace(def.Name))
		if key == "" {
			continue
		}
		entry := phaseStatusEntry{
			Phase:    def.Name,
			Optional: def.Optional,
		}
		if result, ok := resultLookup[key]; ok {
			status := strings.ToLower(strings.TrimSpace(result.Status))
			if status == "" {
				status = "unknown"
			}
			entry.Status = status
			entry.Recorded = true
		} else {
			entry.Status = "not_run"
			entry.Recorded = false
		}
		payload = append(payload, entry)
	}
	return payload
}
