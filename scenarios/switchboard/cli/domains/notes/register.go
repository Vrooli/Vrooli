// Package notes is the CLI's notes-domain command surface. Connect-RPC commands
// come from cli/manifest.json and are wired to matching cli-core primitives;
// attach is the documented REST multipart exception.
package notes

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns. Exported so
// the package's tests can call cliapp.RequireProtoServiceCoverage against
// the same manifest the runtime loads.
const GroupName = "notes"

// Register builds the notes subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
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
		Architecture: cliapp.CommandArchitecture{
			Exception:       cliapp.ExceptionUpload,
			ExceptionReason: "REST multipart file upload",
		},
	}.WithPrimitive(cliapp.Upload(h.attachCall, h.attachReport)))
	return group, nil
}
