package harness

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "harness"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	filteredManifest, err := manifestForRPCCommands(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("harness: filter local commands: %w", err)
	}
	g, err := cliapp.LoadFromManifestPrimitives(filteredManifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"HarnessService.RunImport":            cliapp.ProtoMutation(h.importCall, h.importReport),
		"HarnessService.GetImportStatus":      cliapp.ProtoList(h.statusCall, h.statusReport),
		"HarnessService.RefreshProjection":    cliapp.ProtoMutation(h.projectCall, h.projectReport),
		"HarnessService.CaptureWrite":         cliapp.ProtoMutation(h.captureCall, h.captureReport),
		"HarnessService.GetMaintenanceStatus": cliapp.ProtoList(h.maintenanceCall, h.maintenanceReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("harness: load manifest: %w", err)
	}
	g.Subcommands = append(g.Subcommands,
		cliapp.Command{Name: "hook", Description: "Capture a native memory write from a harness hook", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "runtime", Description: "source harness runtime"}}}, Run: h.hook},
		cliapp.Command{Name: "hooks", Description: "Install or remove native memory capture hooks", Run: h.hooks},
	)
	return g, nil
}

// manifestForRPCCommands leaves local hook commands in the public manifest
// for discovery, but excludes them from cli-core's RPC binding loader. The
// hooks are appended as native commands below because they intentionally do
// not have a Connect-RPC binding.
func manifestForRPCCommands(raw []byte) ([]byte, error) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	groups, ok := document["groups"].([]any)
	if !ok {
		return raw, nil
	}
	for _, rawGroup := range groups {
		group, ok := rawGroup.(map[string]any)
		if !ok || group["name"] != GroupName {
			continue
		}
		commands, ok := group["commands"].([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(commands))
		for _, rawCommand := range commands {
			command, ok := rawCommand.(map[string]any)
			if !ok {
				filtered = append(filtered, rawCommand)
				continue
			}
			binding, _ := command["binding"].(map[string]any)
			if binding["kind"] == "local" {
				continue
			}
			filtered = append(filtered, rawCommand)
		}
		group["commands"] = filtered
	}
	return json.Marshal(document)
}

// Commands exposes the durable import contract at the scenario CLI root so
// agents can use `vrooli-memory import` without depending on group topology.
func Commands(core *cliapp.ScenarioApp) []cliapp.Command {
	h := newHandlers(core)
	return []cliapp.Command{
		cliapp.Command{Name: "import", Description: "Import coding-agent memory", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "runtime", Description: "Harness runtime"}, {Name: "dry-run", Description: "Read and validate sources without writing entries", Bool: true}}}}.WithPrimitive(cliapp.ProtoMutation(h.importCall, h.importReport)),
		cliapp.Command{Name: "import-status", Description: "Show durable import progress", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "run-id", Description: "Specific import run ID"}, {Name: "runtime", Description: "Harness runtime"}}}}.WithPrimitive(cliapp.ProtoList(h.statusCall, h.statusReport)),
		cliapp.Command{Name: "project", Description: "Project unified memory into a harness file", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "harness", Description: "Harness runtime"}, {Name: "dry-run", Description: "Render without writing", Bool: true}}}}.WithPrimitive(cliapp.ProtoMutation(h.projectCall, h.projectReport)),
		cliapp.Command{Name: "capture", Description: "Capture a native harness memory write", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "runtime", Description: "Harness runtime"}, {Name: "source-path", Description: "Native memory source path"}, {Name: "content", Description: "Native memory content"}}}}.WithPrimitive(cliapp.ProtoMutation(h.captureCall, h.captureReport)),
		cliapp.Command{Name: "maintenance-status", Description: "Show the last automatic import and projection run"}.WithPrimitive(cliapp.ProtoList(h.maintenanceCall, h.maintenanceReport)),
		cliapp.Command{Name: "hook", Description: "Capture a native memory write from a harness hook", Run: h.hook},
		cliapp.Command{Name: "hooks", Description: "Install or remove native memory capture hooks", Run: h.hooks},
	}
}
