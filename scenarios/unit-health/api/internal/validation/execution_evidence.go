package validation

import (
	"fmt"
	"path/filepath"
	"strings"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

type executionEvidenceCollection struct {
	index                  int
	workspace, root, runID string
	adapter                adapters.ExecutionEvidenceAdapter
}

func prepareExecutionEvidence(plan *ExecutionPlan, workspaces []Workspace, runID string) []executionEvidenceCollection {
	var collections []executionEvidenceCollection
	registry := adapterregistry.Default()
	for i := range plan.Commands {
		command := &plan.Commands[i]
		if command.TestKind != "unit" && command.TestKind != "local-integration" {
			continue
		}
		for _, ws := range workspaces {
			if ws.ID != command.WorkspaceID || filepath.Clean(ws.RootPath) != filepath.Clean(command.WorkingDirectory) {
				continue
			}
			owner, ok := registry.ResolveSource(adapters.Identity{ID: ws.AdapterID, Version: ws.AdapterVersion}, adapters.Match{Language: ws.Language, Framework: ws.Framework})
			if !ok {
				continue
			}
			adapter, ok := owner.(adapters.ExecutionEvidenceAdapter)
			if !ok {
				continue
			}
			args, ok := adapter.PrepareExecutionEvidence(command.Executable, command.Args)
			if !ok {
				continue
			}
			command.Args, command.CaptureStdout = args, true
			command.Command = command.Executable + " " + strings.Join(args, " ")
			collections = append(collections, executionEvidenceCollection{index: i, workspace: ws.ID, root: ws.RootPath, runID: fmt.Sprintf("%s:command:%d", runID, i), adapter: adapter})
			break
		}
	}
	return collections
}

func collectExecutionEvidence(collections []executionEvidenceCollection, results []CommandResult, source []testquality.TestLinks) ([]testquality.TestLinks, testquality.Reason) {
	out := append([]testquality.TestLinks(nil), source...)
	unavailable := testquality.ReasonNone
	seen := map[string]bool{}
	for _, collection := range collections {
		// More than one command for a workspace needs an explicit evidence merge
		// policy. Do not let the last command silently replace earlier results.
		if seen[collection.workspace] {
			unavailable = testquality.MissingInput
			continue
		}
		seen[collection.workspace] = true
		var declarations []testquality.TestLinks
		var positions []int
		for i, declaration := range out {
			if declaration.Target.Workspace == collection.workspace {
				declarations = append(declarations, declaration)
				positions = append(positions, i)
			}
		}
		var observed []testquality.TestLinks
		reason := testquality.MissingInput
		if collection.index < len(results) {
			result := results[collection.index]
			if result.Status == "passed" || result.Status == "failed" {
				observed, reason = collection.adapter.ObserveExecutionEvidence(adapters.ExecutionEvidenceInput{Root: collection.root, RunID: collection.runID, Stdout: result.StdoutEvidence, Complete: result.StdoutEvidenceComplete, Declarations: declarations})
			}
		}
		if reason != testquality.ReasonNone || len(observed) != len(positions) {
			if reason == testquality.ReasonNone {
				reason = testquality.MissingInput
			}
			unavailable = reason
			for _, pos := range positions {
				out[pos].EvidenceKind = testquality.Runtime
				out[pos].Execution = testquality.ExecutionUnknown
				out[pos].RunID = collection.runID
				out[pos].ExpectedRunID = collection.runID
			}
			continue
		}
		for i, pos := range positions {
			out[pos] = observed[i]
		}
	}
	return out, unavailable
}
