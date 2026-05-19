// Package notes is the CLI's notes-domain command surface. Mirrors
// the API's Connect-RPC Notes service, plus the /api/v1/notes/{id}/attachments
// REST exception, and the UI's api/notes.ts client.
//
// New domain packages copy this shape: a Register(core, manifest) returning
// a cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The manifest carries the declarative surface (governance,
// flags, positionals, RPC bindings) and is the SINGLE source of truth for
// the command-line shape; do not hand-author SubcommandGroup literals for
// Connect-RPC commands. The `attach` subcommand is the documented REST
// multipart exception and is appended directly because the cli-manifest/v1
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
// and wires Connect-RPC bindings to handlers in handlers.go. The
// `notes attach` REST exception is appended outside the manifest path
// because cli-manifest/v1 only supports binding.kind=connect-rpc.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"NotesService.ListNotes":  h.list,
		"NotesService.CreateNote": h.create,
		"NotesService.GetNote":    h.get,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
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
