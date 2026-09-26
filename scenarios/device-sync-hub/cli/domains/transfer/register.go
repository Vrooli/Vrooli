// Package transfer is the CLI's transfer-domain command surface and the
// programmatic compound-value seam: the verbs another scenario or agent calls
// to "deliver this file/text to one of the owner's devices". It mirrors the
// TransferService Connect-RPC service (send-text | list | get | delete) plus the
// two documented REST byte edges (upload a file, download an item's bytes),
// which are appended outside the manifest because cli-manifest/v1 models only
// connect-rpc bindings.
//
// Auth here is by DEVICE TOKEN, not the owner JWT: every transfer call presents
// the opaque hub token the device received at pairing in the `X-Device-Token`
// header. The token is resolved from the `--device-token` flag, falling back to
// $DEVICE_SYNC_HUB_DEVICE_TOKEN (see deviceToken in handlers.go).
package transfer

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns. Exported so the
// package's tests can call cliapp.RequireProtoServiceCoverage against the same
// manifest the runtime loads.
const GroupName = "transfer"

// Register builds the transfer subcommand group from the embedded manifest,
// wires Connect-RPC bindings to handlers, and appends the `upload` and
// `download` REST exceptions (multipart request / binary response) that
// cli-manifest/v1 cannot express.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"TransferService.CreateTextItem": h.sendText,
		"TransferService.ListItems":      h.list,
		"TransferService.GetItem":        h.get,
		"TransferService.DeleteItem":     h.delete,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("transfer: load from manifest: %w", err)
	}

	deviceTokenFlag := cliapp.Flag{Name: "device-token", Description: "Hub device token (defaults to $DEVICE_SYNC_HUB_DEVICE_TOKEN)"}

	group.Subcommands = append(group.Subcommands,
		cliapp.Command{
			Name:        "upload",
			Description: "Upload a file to the trust group (or directed to one device)",
			Args: cliapp.ArgSchema{
				Flags: []cliapp.Flag{
					{Name: "file", Required: true, Description: "Path of the file to upload"},
					deviceTokenFlag,
					{Name: "name", Description: "Override the item name (defaults to the filename)"},
					{Name: "retention", Description: "Lifetime policy (live|held|pinned)"},
					{Name: "target", Description: "Direct the file to a single device id (empty = broadcast)"},
				},
			},
			RunCtx: h.upload,
		},
		cliapp.Command{
			Name:        "download",
			Description: "Download an item's bytes to a local path, preserving the original filename",
			Args: cliapp.ArgSchema{
				Positionals: []cliapp.Positional{
					{Name: "id", Required: true, Description: "Item id"},
				},
				Flags: []cliapp.Flag{
					{Name: "out", Description: "Output file path or directory (defaults to the current directory)"},
					deviceTokenFlag,
					{Name: "thumb", Description: "Download the image thumbnail instead of the original bytes", Bool: true},
				},
			},
			RunCtx: h.download,
		},
	)
	return group, nil
}
