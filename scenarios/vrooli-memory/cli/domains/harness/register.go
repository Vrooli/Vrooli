package harness

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "harness"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"HarnessService.RunImport":            cliapp.ProtoMutation(h.importCall, h.importReport),
		"HarnessService.GetImportStatus":      cliapp.ProtoList(h.statusCall, h.statusReport),
		"HarnessService.RefreshProjection":    cliapp.ProtoMutation(h.projectCall, h.projectReport),
		"HarnessService.CaptureWrite":         cliapp.ProtoMutation(h.captureCall, h.captureReport),
		"HarnessService.InstallPromptBlock":   cliapp.ProtoMutation(h.promptCall, h.promptReport),
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

// Commands exposes the durable import contract at the scenario CLI root so
// agents can use `vrooli-memory import` without depending on group topology.
func Commands(core *cliapp.ScenarioApp) []cliapp.Command {
	h := newHandlers(core)
	return []cliapp.Command{
		cliapp.Command{Name: "import", Description: "Import coding-agent memory", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "runtime", Description: "Harness runtime"}, {Name: "dry-run", Description: "Read and validate sources without writing entries", Bool: true}}}}.WithPrimitive(cliapp.ProtoMutation(h.importCall, h.importReport)),
		cliapp.Command{Name: "import-status", Description: "Show durable import progress", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "run-id", Description: "Specific import run ID"}, {Name: "runtime", Description: "Harness runtime"}}}}.WithPrimitive(cliapp.ProtoList(h.statusCall, h.statusReport)),
		cliapp.Command{Name: "project", Description: "Project unified memory into a harness file", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "harness", Description: "Harness runtime"}, {Name: "dry-run", Description: "Render without writing", Bool: true}}}}.WithPrimitive(cliapp.ProtoMutation(h.projectCall, h.projectReport)),
		cliapp.Command{Name: "capture", Description: "Capture a native harness memory write", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "runtime", Description: "Harness runtime"}, {Name: "source-path", Description: "Native memory source path"}, {Name: "content", Description: "Native memory content"}}}}.WithPrimitive(cliapp.ProtoMutation(h.captureCall, h.captureReport)),
		cliapp.Command{Name: "install-prompt", Description: "Install native-memory guidance in a harness convention file", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "runtime", Description: "Harness runtime"}}}}.WithPrimitive(cliapp.ProtoMutation(h.promptCall, h.promptReport)),
		cliapp.Command{Name: "maintenance-status", Description: "Show the last automatic import and projection run"}.WithPrimitive(cliapp.ProtoList(h.maintenanceCall, h.maintenanceReport)),
		cliapp.Command{Name: "hook", Description: "Capture a native memory write from a harness hook", Run: h.hook},
		cliapp.Command{Name: "hooks", Description: "Install or remove native memory capture hooks", Run: h.hooks},
	}
}
