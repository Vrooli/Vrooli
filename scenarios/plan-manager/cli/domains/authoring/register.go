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
		"AuthoringService.GetSection":                h.sectionGet,
		"AuthoringService.SubmitSection":             h.sectionSubmit,
		"AuthoringService.Next":                      h.next,
		"AuthoringService.ValidateStructure":         h.validate,
		"AuthoringService.Autofill":                  h.autofill,
		"AuthoringService.SubmitRelevantContextItem": h.contextSubmit,
		"AuthoringService.ListRelevantContext":       h.contextList,
		"AuthoringService.DiscoverContextCandidates": h.contextDiscover,
		"AuthoringService.AcceptContextCandidate":    h.contextAccept,
		"AuthoringService.RejectContextCandidate":    h.contextReject,
		"AuthoringService.AddPhase":                  h.phaseAdd,
		"AuthoringService.GetPhase":                  h.phaseGet,
		"AuthoringService.SubmitPhaseField":          h.phaseSubmit,
		"AuthoringService.NextPhase":                 h.phaseNext,
		"AuthoringService.Finalize":                  h.finalize,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("authoring: load author group: %w", err)
	}
	return group, nil
}
