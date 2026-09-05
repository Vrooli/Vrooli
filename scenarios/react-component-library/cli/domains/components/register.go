// Package components is the CLI's component-registry surface. Mirrors
// the API's Connect-RPC ComponentsService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package components

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
)

const GroupName = "components"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ComponentsService.ListComponents":          cliapp.ProtoList(h.listCall, h.listReport),
		"ComponentsService.GetComponent":            cliapp.ProtoList(h.getCall, h.getReport),
		"ComponentsService.IngestComponent":         cliapp.ProtoMutation(h.ingestCall, h.ingestReport),
		"ComponentsService.BeginComponentVersion":   cliapp.ProtoMutation(h.versionBeginCall, h.versionBeginReport),
		"ComponentsService.PublishComponentVersion": cliapp.ProtoMutation(h.versionPublishCall, h.versionPublishReport),
		"ComponentsService.UpdateComponentContent":  cliapp.ProtoMutation(h.contentSetCall, h.contentSetReport),
		"ComponentTestsService.RunComponentTest":    cliapp.ProtoMutation(h.testCall, h.testReport),
		"ComponentTestsService.SweepComponentTests": cliapp.ProtoListOutcome(h.sweepCall, h.sweepReport, func(resp *componenttestsv1.SweepComponentTestsResponse) error {
			if resp.Blocked > 0 || !resp.Complete {
				return fmt.Errorf("component sweep is not complete: %d blocked result(s), %d error(s)", resp.Blocked, len(resp.Errors))
			}
			return nil
		}),
		// Local binding: the manifest declares this command with
		// binding.kind "local" and no handler name, so the loader keys it by
		// the command name. It must be registered here like any other — a
		// manifest command with no entry in this map is a startup panic, not
		// a missing subcommand.
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("components: load from manifest: %w", err)
	}
	return group, nil
}
