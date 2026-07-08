// Package notes is the CLI's notes-domain command surface. Mirrors
// the API's Connect-RPC Notes service, plus the /api/v1/notes/{id}/attachments
// REST exception, and the UI's api/notes.ts client.
//
// New domain packages copy this shape: a Register(core, manifest) returning
// a cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifestPrimitives, plus one cli-core primitive per
// Connect-RPC subcommand (a `<name>Call` + `<name>Report` pair in handlers.go
// wired through cliapp.ProtoList / ProtoMutation / ProtoOperational). Building
// each handler with the matching primitive is what carries a command to
// VERIFIED L4 command-architecture maturity: the PrimitiveHandler proves the
// primitive class by construction, and LoadFromManifestPrimitives reconciles
// that observed evidence against the manifest's declared architecture.primitive
// (a mismatch fails fast here). The manifest carries the declarative surface
// (governance, flags, positionals, RPC bindings, architecture) and is the SINGLE
// source of truth for the command-line shape; do not hand-author SubcommandGroup
// literals for Connect-RPC commands. The `attach` subcommand is the documented
// REST multipart exception and is appended directly because the cli-manifest/v1
// schema only models connect-rpc bindings.
package notes

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns. Exported so
// the package's tests can call cliapp.RequireProtoServiceCoverage against
// the same manifest the runtime loads.
const GroupName = "notes"

// Register builds the notes subcommand group from the embedded manifest
// and wires each Connect-RPC command to a matching cli-core primitive, so the
// declared architecture.primitive is proven by construction. The `notes attach`
// REST exception is appended outside the manifest path because cli-manifest/v1
// only supports binding.kind=connect-rpc.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		// Each binding is built with the primitive its manifest command declares,
		// so its operation runs outside any output-format branch and the observed
		// primitive proves the declaration (LoadFromManifestPrimitives fails fast on
		// any mismatch).
		"NotesService.ListNotes":  cliapp.ProtoList(h.listCall, h.listReport),
		"NotesService.CreateNote": cliapp.ProtoMutation(h.createCall, h.createReport),
		"NotesService.GetNote":    cliapp.ProtoList(h.getCall, h.getReport),
		"NotesService.CountNotes": cliapp.ProtoList(h.countCall, h.countReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("notes: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "attach",
		Description: "Attach a file to a note",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "id", Required: true, Description: "Note id"},
			},
			Flags: []cliapp.Flag{
				{Name: "file", Required: true, Description: "File path to upload"},
			},
		},
		RunCtx: h.attach,
	})
	return group, nil
}
