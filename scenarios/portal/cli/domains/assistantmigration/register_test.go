package assistantmigration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

func TestMigrationCommandsExportAndReconcileWithoutChangingSource(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(source, "data", "tasks"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(source, "data", "contexts"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(source, "data/tasks", "issue-a.md"), []byte("# Issue: Legacy report\n\n**Scenario**: manual\n**URL**: cli\n**Captured**: 2025-10-03T06:27:33-04:00\n\n## Description\nA legacy report.\n\n## Context\n- Issue ID: issue-a\n- Status: captured\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(source, "data/contexts", "capture-a.txt"), []byte("context"), 0o600))

	group, err := Register(nil, readManifest(t))
	require.NoError(t, err)
	find := func(name string) cliapp.Command {
		for _, command := range group.Subcommands {
			if command.Name == name {
				return command
			}
		}
		t.Fatalf("missing command %q", name)
		return cliapp.Command{}
	}

	var inventory bytes.Buffer
	command := find("inventory")
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: command.Args, Flags: map[string]string{"source": source}, JSON: true, Stdout: &inventory})
	require.NoError(t, command.RunCtx(ctx))
	var manifest struct {
		SourceRoot string `json:"source_root"`
		Records    []struct {
			RelativePath string `json:"relative_path"`
		} `json:"records"`
	}
	require.NoError(t, json.Unmarshal(inventory.Bytes(), &manifest))
	require.Equal(t, source, manifest.SourceRoot)
	require.Len(t, manifest.Records, 2)

	destination := filepath.Join(t.TempDir(), "review")
	command = find("export")
	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: command.Args, Flags: map[string]string{"source": source, "destination": destination}, JSON: true, Stdout: &bytes.Buffer{}})
	require.NoError(t, command.RunCtx(ctx))
	manifestPath := filepath.Join(destination, "manifest.json")
	info, err := os.Stat(manifestPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	command = find("reconcile")
	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: command.Args, Flags: map[string]string{"source": source, "manifest": manifestPath}, JSON: true, Stdout: &bytes.Buffer{}})
	require.NoError(t, command.RunCtx(ctx))
	require.FileExists(t, filepath.Join(source, "data/tasks", "issue-a.md"))

	command = find("review")
	var review bytes.Buffer
	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: command.Args, Flags: map[string]string{"source": source, "manifest": manifestPath}, JSON: true, Stdout: &review})
	require.NoError(t, command.RunCtx(ctx))
	var projection struct {
		Requests []struct {
			LegacyID string `json:"legacy_id"`
			Owner    string `json:"owner"`
		} `json:"requests"`
	}
	require.NoError(t, json.Unmarshal(review.Bytes(), &projection))
	require.Len(t, projection.Requests, 1)
	require.Equal(t, "scenario-qa", projection.Requests[0].Owner)

	destination = filepath.Join(t.TempDir(), "owner-handoffs")
	command = find("capture")
	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: command.Args, Flags: map[string]string{"source": source, "manifest": manifestPath, "destination": destination}, JSON: true, Stdout: &bytes.Buffer{}})
	require.NoError(t, command.RunCtx(ctx))
	handoff := filepath.Join(destination, "scenario-qa", "tasks", "issue-a.json")
	require.FileExists(t, handoff)
	before, err := os.ReadFile(handoff)
	require.NoError(t, err)
	require.NoError(t, command.RunCtx(ctx))
	after, err := os.ReadFile(handoff)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestReconcileRejectsManifestForAnotherSource(t *testing.T) {
	group, err := Register(nil, readManifest(t))
	require.NoError(t, err)
	var command cliapp.Command
	for _, candidate := range group.Subcommands {
		if candidate.Name == "reconcile" {
			command = candidate
		}
	}
	source := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{"version":1,"source_root":"/different","records":[]}`), 0o600))
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: command.Args, Flags: map[string]string{"source": source, "manifest": manifestPath}, Stdout: &bytes.Buffer{}})
	require.ErrorContains(t, command.RunCtx(ctx), "different source root")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	return data
}
