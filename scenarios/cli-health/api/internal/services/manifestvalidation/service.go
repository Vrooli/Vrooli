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
	repocontract "github.com/vrooli/repo-contract-go"
)

// Service runs validation for one scenario at a time. All side effects flow
// through the injected seams; the service itself is pure orchestration.
type Service struct {
	manifests    ManifestLoader
	schema       SchemaValidator
	protos       ProtoLoader
	measures     MeasureSchemaReader
	runtimeProbe RuntimeProbe
	archEvidence ArchitectureEvidenceProvider
	logger       *log.Logger
}

// ArchitectureEvidenceProvider observes the cli-core primitive class each of a
// scenario's commands was actually built from, so cli-health can compare that
// unforgeable evidence against the manifest's declared architecture.primitive.
// It is the structural proof channel behind verified primitive maturity: a
// declaration cannot reach verified (L4) maturity on manifest text alone.
//
// seam: optional. A nil provider (the default during rollout) means no observed
// evidence is available, so every declared primitive classifies as
// not-yet-verified maturity debt (arch.primitive_unverified) rather than
// falsely-clean verified adoption. Evidence returns the observed evidence for a
// scenario; an error degrades to empty evidence (still valid, still not verified).
type ArchitectureEvidenceProvider interface {
	Evidence(ctx context.Context, scenario string) (ArchitectureEvidence, error)
}

// Deps holds the seams the service needs. Loaders can be nil to use the
// real implementations.
type Deps struct {
	Manifests ManifestLoader
	Schema    SchemaValidator
	Protos    ProtoLoader
	// Measures resolves proto param schemas for measure blocks. Optional: nil
	// disables measure validation (a no-op for manifests without measures).
	Measures MeasureSchemaReader
	// RuntimeProbe execs the scenario CLI to observe its runtime command surface.
	// Optional: nil disables runtime probing (the default static-only path). Even
	// when wired, the probe runs only when the caller requests execution
	// (include_execution) and the scenario declares a CLI surface.
	RuntimeProbe RuntimeProbe
	// ArchitectureEvidence observes the cli-core primitive each command was built
	// from so declared primitives can be verified (not just trusted). Optional:
	// nil leaves declared primitives at not-yet-verified maturity debt.
	ArchitectureEvidence ArchitectureEvidenceProvider
	Logger               *log.Logger
}

// New returns a service bound to the given dependencies. Callers pass nil
// for any seam they want defaulted; tests pass stubs for all three.
func New(d Deps) *Service {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &Service{
		manifests:    d.Manifests,
		schema:       d.Schema,
		protos:       d.Protos,
		measures:     d.Measures,
		runtimeProbe: d.RuntimeProbe,
		archEvidence: d.ArchitectureEvidence,
		logger:       d.Logger,
	}
}

// ValidateScenario produces a Report for the named scenario. Order of checks:
//  1. Load manifest. If missing: error when the scenario has its own proto
//     surface (a manifest is mandatory there), else skip-with-warning.
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

	collector := metricsFrom(ctx)

	load := collector.Stage("load-manifest")
	raw, path, err := s.manifests.Load(ctx, scenario)
	if err != nil {
		load.End()
		if errors.Is(err, os.ErrNotExist) {
			return s.missingManifest(ctx, scenario), nil
		}
		return Report{}, fmt.Errorf("load manifest for %q: %w", scenario, err)
	}
	load.End()

	var findings []Finding

	schema := collector.Stage("schema-validate")
	schemaFindings, schemaErr := s.schema.Validate(ctx, raw)
	if schemaErr != nil {
		schema.End()
		findings = append(findings, Finding{
			Severity: SeverityError,
			Code:     CodeManifestSchemaError,
			Location: path,
			Message:  fmt.Sprintf("schema validator failed to run: %v", schemaErr),
		})
		return finalize(scenario, findings), nil
	}
	findings = append(findings, schemaFindings...)
	schema.End()

	parse := collector.Stage("parse-manifest")
	m, parseErr := cliapp.ParseManifest(raw)
	if parseErr != nil {
		parse.End()
		// Surface architecture-metadata problems as arch.* codes so the
		// command_architecture capability is impacted precisely, not only the
		// generic manifest.parse_error. Both are legitimate: the manifest is
		// unusable (parse error) AND, when the cause is architecture, the metadata
		// is invalid (a command_architecture finding).
		findings = append(findings, architectureParseFindings(raw, path)...)
		findings = append(findings, Finding{
			Severity: SeverityError,
			Code:     CodeManifestParseError,
			Location: path,
			Message:  parseErr.Error(),
		})
		return finalize(scenario, findings), nil
	}
	parse.End()

	entrypoint := collector.Stage("entrypoint-structure")
	mainFindings := entrypointFindings(path)
	findings = append(findings, mainFindings...)
	entrypoint.Gauge("findings", float64(len(mainFindings)))
	entrypoint.End()

	loadProto := collector.Stage("load-proto")
	surface, protoErr := s.protos.Load(ctx, scenario)
	if protoErr != nil {
		loadProto.End()
		findings = append(findings, Finding{
			Severity:   SeverityError,
			Code:       CodeProtoBuildFailed,
			Location:   fmt.Sprintf("packages/proto/schemas/%s", scenario),
			Message:    protoErr.Error(),
			Suggestion: "run `buf build --path packages/proto/schemas/" + scenario + "` from packages/proto to reproduce",
		})
		return finalize(scenario, findings), nil
	}
	loadProto.End()

	cross := collector.Stage("cross-check")
	findings = append(findings, crossCheck(m, surface, path)...)
	findings = append(findings, s.measureCheck(raw, path)...)
	findings = append(findings, architectureStaticFindings(m, s.architectureEvidence(ctx, scenario), path)...)
	cross.Gauge("findings", float64(len(findings)))
	cross.End()

	// Runtime CLI probe — static-by-default. Runs only when the caller requested
	// execution (include_execution), a probe seam is wired, and the scenario
	// declares a CLI surface to exercise. This is the static→runtime extension of
	// cli-health: the probe execs the binary and reconciles its observed command
	// surface against the manifest. Degrades (warning, not error) when the binary
	// is simply absent in this run context.
	if s.runtimeProbe != nil && includeExecutionFrom(ctx) && hasCLISurface(m) {
		probe := collector.Stage("runtime-probe")
		obs, probeErr := s.runtimeProbe.Probe(ctx, scenario)
		if probeErr != nil {
			s.logger.Printf("validation: runtime probe for %q degraded: %v", scenario, probeErr)
			findings = append(findings, Finding{
				Severity:   SeverityWarning,
				Code:       CodeCLIBinaryUnrunnable,
				Location:   path,
				Message:    fmt.Sprintf("runtime CLI probe could not complete: %v", probeErr),
				Suggestion: "this is a probe-infrastructure degradation, not necessarily a scenario defect",
			})
		} else {
			findings = append(findings, runtimeFindings(obs, m, path)...)
			findings = append(findings, architectureRuntimeFindings(m, obs, path)...)
		}
		probe.Gauge("findings", float64(len(findings)))
		probe.End()
	}

	return finalize(scenario, findings), nil
}

// architectureEvidence resolves the cli-core primitive evidence for a scenario
// via the optional provider seam. A nil provider or a provider error degrades to
// empty evidence: declared primitives then classify as not-yet-verified debt
// (never verified), which is the honest rollout state when no evidence channel
// is wired for the target scenario.
func (s *Service) architectureEvidence(ctx context.Context, scenario string) ArchitectureEvidence {
	if s.archEvidence == nil {
		return ArchitectureEvidence{}
	}
	ev, err := s.archEvidence.Evidence(ctx, scenario)
	if err != nil {
		s.logger.Printf("validation: architecture evidence for %q degraded: %v", scenario, err)
		return ArchitectureEvidence{}
	}
	return ev
}

// missingManifest decides the verdict when a scenario has no cli/manifest.json.
// A scenario that exposes its own proto RPC surface MUST ship a manifest — it is
// the single source of truth for that scenario's CLI, and "no manifest" is the
// exact loophole that let proto-first scenarios skip the API↔CLI contract. So
// for a scenario with an own proto surface this is a hard error; only a
// genuinely proto-less scenario (or one whose proto surface cannot even be
// built) keeps the soft skip-with-warning.
func (s *Service) missingManifest(ctx context.Context, scenario string) Report {
	if surface, protoErr := s.protos.Load(ctx, scenario); protoErr == nil && surface.HasAnyMethod() {
		return finalize(scenario, []Finding{
			{
				Severity:   SeverityError,
				Code:       CodeManifestRequired,
				Location:   defaultManifestRel(scenario),
				Message:    "scenario exposes proto RPC services but has no cli/manifest.json — the single source of truth for its CLI surface",
				Suggestion: "add cli/manifest.json binding every proto method to a command (or listing it in omitted[] with a reason)",
			},
			{
				// Without a manifest there is nothing to classify command
				// architecture from, so the capability sits at L0 instead of
				// falsely reporting top maturity by absence of findings. WARNING +
				// required marks honest debt without failing the phase beyond the
				// manifest.required error above.
				Severity:   SeverityWarning,
				Code:       CodeArchUnclassifiable,
				Location:   defaultManifestRel(scenario),
				Message:    "scenario exposes a CLI/proto surface but has no manifest, so command architecture cannot be classified",
				Suggestion: "add cli/manifest.json and declare each command's architecture.primitive",
			},
		})
	}
	return finalize(scenario, []Finding{{
		Severity:   SeverityWarning,
		Code:       CodeManifestMissing,
		Location:   defaultManifestRel(scenario),
		Message:    "scenario has no cli/manifest.json and no own proto surface; cli-health validation skipped",
		Suggestion: "generate a manifest via the scenario template or add one by hand",
	}})
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
			// Local commands are catalogued CLI surfaces, not RPC bindings.
			// They have no proto service/method to cross-check and must not
			// participate in duplicate or proto-coverage validation.
			if b.Kind != "connect-rpc" {
				continue
			}
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
	rel, err := repocontract.ScenarioCLIManifestRepoRel("", scenario)
	if err != nil {
		return scenario
	}
	return rel
}
