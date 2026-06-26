// Package plans is the CLI's plans-domain command surface. It owns three
// manifest groups — `plans` (the structured-plan SSOT CRUD + render + graph +
// import/migrate), `phase` (first-class phase add/update) and `template`
// (per-surface plan templates) — all backed by the API's PlansService. The
// manifest (cli/manifest.json) carries the declarative command shape;
// handlers.go builds each typed request and renders the response.
package plans

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Manifest group names this package owns.
const (
	PlansGroup    = "plans"
	PhaseGroup    = "phase"
	TemplateGroup = "template"
)

// Register builds the plans + phase + template subcommand groups from the
// embedded manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	h := newHandlers(core)

	plansGroup, err := cliapp.LoadFromManifest(manifest, PlansGroup, map[string]func(cliapp.RunContext) error{
		"PlansService.ListPlans":        h.list,
		"PlansService.GetPlan":          h.get,
		"PlansService.CreatePlan":       h.create,
		"PlansService.UpdatePlan":       h.update,
		"PlansService.ArchivePlan":      h.archive,
		"PlansService.RenderMarkdown":   h.render,
		"PlansService.GetGraph":         h.graph,
		"PlansService.LinkSupersession": h.link,
		"PlansService.LinkDependency":   h.depend,
		"PlansService.ImportPlan":       h.importPlan,
		"PlansService.MigratePlan":      h.migrate,
	})
	if err != nil {
		return nil, fmt.Errorf("plans: load plans group: %w", err)
	}

	phaseGroup, err := cliapp.LoadFromManifest(manifest, PhaseGroup, map[string]func(cliapp.RunContext) error{
		"PlansService.AddPhase":    h.phaseAdd,
		"PlansService.UpdatePhase": h.phaseUpdate,
	})
	if err != nil {
		return nil, fmt.Errorf("plans: load phase group: %w", err)
	}

	templateGroup, err := cliapp.LoadFromManifest(manifest, TemplateGroup, map[string]func(cliapp.RunContext) error{
		"PlansService.ListTemplates":      h.templateList,
		"PlansService.CreateFromTemplate": h.templateNew,
	})
	if err != nil {
		return nil, fmt.Errorf("plans: load template group: %w", err)
	}

	return []cliapp.SubcommandGroup{plansGroup, phaseGroup, templateGroup}, nil
}
