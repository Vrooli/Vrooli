// Package review is the CLI's server-owned findings and dispositions surface.
// Mirrors the API's Connect-RPC ReviewService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifestPrimitives.
package review

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "review"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ReviewService.ListFindings":      cliapp.ProtoList(h.findingsCall, h.findingsReport),
		"ReviewService.GetFinding":        cliapp.ProtoList(h.findingCall, h.findingReport),
		"ReviewService.RecordDisposition": cliapp.ProtoMutation(h.dispositionCall, h.dispositionReport),
		"ReviewService.AssignFinding":     cliapp.ProtoMutation(h.assignCall, h.assignReport),
		"ReviewService.GetReviewSummary":  cliapp.ProtoList(h.summaryCall, h.summaryReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("review: load from manifest: %w", err)
	}
	return group, nil
}
