// Package authoring is the CLI's authoring-domain command surface. It owns the
// `author` manifest group — the guided composer wizard: start a session, walk and
// submit sections, validate structure, autofill the mechanical sections, and
// finalize into a structured plan — backed by the API's AuthoringService.
// handlers.go builds each typed request and renders the response.
package authoring

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "author"

// Register builds the author subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"AuthoringService.StartSession":              h.start,
		"AuthoringService.GetSession":                h.getSession,
		"AuthoringService.GetSection":                h.sectionGet,
		"AuthoringService.SubmitSection":             h.sectionSubmit,
		"AuthoringService.SubmitFields":              h.submit,
		"AuthoringService.Next":                      h.next,
		"AuthoringService.ContinueAuthoring":         h.continueAuthoring,
		"AuthoringService.ValidateStructure":         h.validate,
		"AuthoringService.Autofill":                  h.autofill,
		"AuthoringService.SubmitRelevantContextItem": h.contextSubmit,
		"AuthoringService.ListRelevantContext":       h.contextList,
		"AuthoringService.UpdateRelevantContextItem": h.contextUpdate,
		"AuthoringService.RemoveRelevantContextItem": h.contextRemove,
		"AuthoringService.DiscoverSkillPack":         h.skillPack,
		"AuthoringService.AddPhase":                  h.phaseAdd,
		"AuthoringService.MovePhase":                 h.phaseMove,
		"AuthoringService.GetPhase":                  h.phaseGet,
		"AuthoringService.SubmitPhaseField":          h.phaseSubmit,
		"AuthoringService.NextPhase":                 h.phaseNext,
		"AuthoringService.PreviewPlan":               h.preview,
		"AuthoringService.Finalize":                  h.finalize,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("authoring: load author group: %w", err)
	}
	// `author status` is the alias agents guess for "show me the session" —
	// wire it explicitly to preview (the manifest schema binds each RPC once,
	// so the alias lives here rather than as a second manifest command).
	for i := range group.Subcommands {
		if group.Subcommands[i].Name == "preview" {
			group.Subcommands[i].Aliases = append(group.Subcommands[i].Aliases, "status")
		}
	}
	return group, nil
}
