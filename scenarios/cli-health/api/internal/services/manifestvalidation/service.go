package manifestvalidation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// Service runs validation for one scenario at a time. All side effects flow
// through the injected seams; the service itself is pure orchestration.
type Service struct {
	manifests ManifestLoader
	schema    SchemaValidator
	protos    ProtoLoader
	logger    *log.Logger
}

// Deps holds the seams the service needs. Loaders can be nil to use the
// real implementations.
type Deps struct {
	Manifests ManifestLoader
	Schema    SchemaValidator
	Protos    ProtoLoader
	Logger    *log.Logger
}

// New returns a service bound to the given dependencies. Callers pass nil
// for any seam they want defaulted; tests pass stubs for all three.
func New(d Deps) *Service {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &Service{
		manifests: d.Manifests,
		schema:    d.Schema,
		protos:    d.Protos,
		logger:    d.Logger,
	}
}

// ValidateScenario produces a Report for the named scenario. Order of checks:
//  1. Load manifest (warning + early-return if missing).
//  2. Schema-validate raw bytes (collect findings; continue if non-empty).
//  3. Structurally parse via cliapp.ParseManifest (error + return if invalid).
//  4. Load proto descriptors via buf (error + return if buf fails).
//  5. Cross-check bindings, orphan methods, stale omissions, duplicates.
//
// The first three steps short-circuit when they fail because later steps
// rely on a well-formed manifest. Steps 4-5 always run together once we
// have a parsed manifest.
func (s *Service) ValidateScenario(ctx context.Context, scenario string) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Report{}, fmt.Errorf("scenario name is required")
	}

	raw, path, err := s.manifests.Load(ctx, scenario)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			f := []Finding{{
				Severity:   SeverityWarning,
				Code:       CodeManifestMissing,
				Location:   defaultManifestRel(scenario),
				Message:    "scenario has no cli/manifest.json; cli-health validation skipped",
				Suggestion: "generate a manifest via the scenario template or add one by hand",
			}}
			return finalize(scenario, f), nil
		}
		return Report{}, fmt.Errorf("load manifest for %q: %w", scenario, err)
	}

	var findings []Finding

	schemaFindings, schemaErr := s.schema.Validate(ctx, raw)
	if schemaErr != nil {
		findings = append(findings, Finding{
			Severity: SeverityError,
			Code:     CodeManifestSchemaError,
			Location: path,
			Message:  fmt.Sprintf("schema validator failed to run: %v", schemaErr),
		})
		return finalize(scenario, findings), nil
	}
	findings = append(findings, schemaFindings...)

	m, parseErr := cliapp.ParseManifest(raw)
	if parseErr != nil {
		findings = append(findings, Finding{
			Severity: SeverityError,
			Code:     CodeManifestParseError,
			Location: path,
			Message:  parseErr.Error(),
		})
		return finalize(scenario, findings), nil
	}

	surface, protoErr := s.protos.Load(ctx, scenario)
	if protoErr != nil {
		findings = append(findings, Finding{
			Severity:   SeverityError,
			Code:       CodeProtoBuildFailed,
			Location:   fmt.Sprintf("packages/proto/schemas/%s", scenario),
			Message:    protoErr.Error(),
			Suggestion: "run `buf build --path packages/proto/schemas/" + scenario + "` from packages/proto to reproduce",
		})
		return finalize(scenario, findings), nil
	}

	findings = append(findings, crossCheck(m, surface, path)...)

	return finalize(scenario, findings), nil
}

// crossCheck implements the binding/coverage rules:
//   - every binding's service+method must exist in the proto surface;
//   - no two commands may bind the same (service, method);
//   - every proto method must be bound or listed in omitted[];
//   - every omitted[] entry must reference a real (service, method).
//
// Returns findings in a deterministic order (sorted by service, method,
// then group/command) so test assertions are stable.
func crossCheck(m *cliapp.Manifest, surface ProtoSurface, manifestPath string) []Finding {
	var findings []Finding

	type bindingSite struct{ group, cmd string }
	bound := map[string][]bindingSite{} // "Service.Method" -> sites

	for _, g := range m.Groups {
		for _, c := range g.Commands {
			b := c.Binding
			key := b.Service + "." + b.Method
			bound[key] = append(bound[key], bindingSite{group: g.Name, cmd: c.Name})

			loc := fmt.Sprintf("%s#/groups/%s/commands/%s/binding", manifestPath, g.Name, c.Name)
			if !surface.HasService(b.Service) {
				findings = append(findings, Finding{
					Severity:   SeverityError,
					Code:       CodeBindingUnknownSvc,
					Location:   loc,
					Message:    fmt.Sprintf("binding references service %q which is not declared in any of this scenario's proto files", b.Service),
					Suggestion: "fix the service name or add the service to packages/proto/schemas/<scenario>/v1/",
				})
				continue
			}
			if !surface.HasMethod(b.Service, b.Method) {
				findings = append(findings, Finding{
					Severity:   SeverityError,
					Code:       CodeBindingUnknownMethod,
					Location:   loc,
					Message:    fmt.Sprintf("binding references method %s.%s which is not declared on service %s", b.Service, b.Method, b.Service),
					Suggestion: "fix the method name or add the rpc to the proto file",
				})
			}
		}
	}

	// Duplicate-binding check (sorted for determinism).
	dupKeys := make([]string, 0, len(bound))
	for k, sites := range bound {
		if len(sites) > 1 {
			dupKeys = append(dupKeys, k)
		}
	}
	sort.Strings(dupKeys)
	for _, k := range dupKeys {
		sites := bound[k]
		sort.Slice(sites, func(i, j int) bool {
			if sites[i].group != sites[j].group {
				return sites[i].group < sites[j].group
			}
			return sites[i].cmd < sites[j].cmd
		})
		labels := make([]string, 0, len(sites))
		for _, s := range sites {
			labels = append(labels, s.group+"/"+s.cmd)
		}
		findings = append(findings, Finding{
			Severity:   SeverityError,
			Code:       CodeBindingDuplicate,
			Location:   manifestPath,
			Message:    fmt.Sprintf("%s is bound by multiple commands: %s", k, strings.Join(labels, ", ")),
			Suggestion: "keep one binding; remove or rename the others",
		})
	}

	// Omitted[] entries that point at non-existent (service, method).
	omitted := map[string]string{} // "Service.Method" -> reason
	for _, o := range m.Omitted {
		key := o.Service + "." + o.Method
		omitted[key] = o.Reason
		if !surface.HasMethod(o.Service, o.Method) {
			findings = append(findings, Finding{
				Severity:   SeverityWarning,
				Code:       CodeOmissionOrphan,
				Location:   manifestPath + "#/omitted",
				Message:    fmt.Sprintf("omitted entry %s.%s does not exist in this scenario's proto surface (stale entry?)", o.Service, o.Method),
				Suggestion: "remove the stale entry or fix the service/method name",
			})
		}
	}

	// Orphan-method check: every proto method must be bound or omitted.
	var orphans []string
	for _, svc := range surface.Services {
		for _, method := range svc.Methods {
			key := svc.Name + "." + method
			if _, isBound := bound[key]; isBound {
				continue
			}
			if _, isOmitted := omitted[key]; isOmitted {
				continue
			}
			orphans = append(orphans, key)
		}
	}
	sort.Strings(orphans)
	for _, key := range orphans {
		findings = append(findings, Finding{
			Severity:   SeverityError,
			Code:       CodeProtoOrphanMethod,
			Location:   manifestPath,
			Message:    fmt.Sprintf("proto method %s is neither bound to a command nor listed in omitted[]", key),
			Suggestion: "add a binding under groups[].commands[].binding, or add an entry to omitted[] with a reason",
		})
	}

	return findings
}

func finalize(scenario string, findings []Finding) Report {
	summary := summarize(findings)
	return Report{
		Scenario: scenario,
		Passed:   summary.Errors == 0,
		Findings: findings,
		Summary:  summary,
	}
}

func defaultManifestRel(scenario string) string {
	return fmt.Sprintf("scenarios/%s/cli/manifest.json", scenario)
}
