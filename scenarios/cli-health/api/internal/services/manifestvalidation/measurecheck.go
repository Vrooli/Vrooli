package manifestvalidation

import (
	"fmt"
	"sort"

	"github.com/vrooli/measures-go/manifestscan"

	measures "github.com/vrooli/measures-go"
)

// measureCheck validates the `measure` blocks declared on a manifest's commands.
// It is intentionally narrow — Phase 2 of the measures plan covers *static
// well-formedness* only:
//
//   - a manifest param `type` annotation that is not a known canonical
//     convention (time_window/enum) → ERROR (measure.unknown_param_type);
//   - a measure that fails assembly against its proto-derived param schema
//     (drift: a manifest param naming a field absent from the request message,
//     or a malformed result/effect) → ERROR (measure.invalid);
//   - a measure whose proto param schema cannot be resolved (descriptor image
//     unavailable) → WARNING (measure.schema_unread), degraded not fatal;
//   - a well-formed measure → INFO (measure.tier) recording its graded adoption
//     tier (full/partial/fallback) for downstream surfacing.
//
// Coverage grading (which stateful domains must be covered / are waived) is NOT
// here — that is measures-health's responsibility in a later phase. When the
// service has no MeasureSchemaReader seam (nil), measure validation is skipped
// entirely so manifests without measures and measure-agnostic unit tests are
// unaffected.
func (s *Service) measureCheck(raw []byte, manifestPath string) []Finding {
	if s.measures == nil {
		return nil
	}

	parsed, err := manifestscan.Parse(raw)
	if err != nil {
		// The schema validator already ran; a parse failure here means the
		// measure surface is malformed in a way the schema missed.
		return []Finding{{
			Severity: SeverityError,
			Code:     CodeMeasureInvalid,
			Location: manifestPath + "#/measures",
			Message:  fmt.Sprintf("could not parse measure blocks: %v", err),
		}}
	}
	if len(parsed.Commands) == 0 {
		return nil
	}

	cmds := append([]manifestscan.CommandMeasure(nil), parsed.Commands...)
	sort.Slice(cmds, func(i, j int) bool {
		if cmds[i].Group != cmds[j].Group {
			return cmds[i].Group < cmds[j].Group
		}
		return cmds[i].Command < cmds[j].Command
	})

	var findings []Finding
	for _, cm := range cmds {
		loc := fmt.Sprintf("%s#/groups/%s/commands/%s/measure", manifestPath, cm.Group, cm.Command)

		// Unknown manifest param-type annotation — caught independently of proto
		// resolution because assembly would otherwise pass an invented type
		// through unchecked. Sorted so findings are deterministic.
		types := cm.ManifestParamTypes()
		names := make([]string, 0, len(types))
		for name := range types {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !manifestscan.KnownManifestParamType(types[name]) {
				findings = append(findings, Finding{
					Severity:   SeverityError,
					Code:       CodeMeasureUnknownType,
					Location:   loc,
					Message:    fmt.Sprintf("measure %s: param %q declares unknown type %q (only the canonical conventions time_window/enum may be annotated; real kinds come from proto)", cm.MeasureName(), name, types[name]),
					Suggestion: "drop the type annotation (let the proto-derived kind stand) or use time_window/enum",
				})
			}
		}

		protoParams, perr := s.measures.RequestParams(cm.Binding.Service, cm.Binding.Method)
		if perr != nil {
			findings = append(findings, Finding{
				Severity:   SeverityWarning,
				Code:       CodeMeasureSchemaUnread,
				Location:   loc,
				Message:    fmt.Sprintf("measure %s: could not resolve proto param schema for %s.%s: %v", cm.MeasureName(), cm.Binding.Service, cm.Binding.Method, perr),
				Suggestion: "ensure packages/proto/gen/descriptor/image.binpb is built (make -C packages/proto generate)",
			})
			continue
		}

		decl, aerr := measures.Assemble(cm.MeasureName(), cm.Domain, cm.Binding, cm.Measure, cm.Governance, protoParams)
		if aerr != nil {
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeMeasureInvalid,
				Location:   loc,
				Message:    fmt.Sprintf("measure %s is not well-formed: %v", cm.MeasureName(), aerr),
				Suggestion: "align the measure block's params/result with the bound proto request message",
			})
			continue
		}

		tier := manifestscan.GradeTier(decl)
		findings = append(findings, Finding{
			Severity: SeverityInfo,
			Code:     CodeMeasureTier,
			Location: loc,
			Message:  fmt.Sprintf("measure %s graded tier=%s (%d params)", decl.Name, tier, len(decl.Params)),
		})
	}
	return findings
}
