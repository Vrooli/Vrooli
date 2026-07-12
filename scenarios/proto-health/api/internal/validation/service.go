package validation

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"proto-health/internal/protosurface"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

var versionDirRE = regexp.MustCompile(`^v[1-9][0-9]*$`)

type Service struct {
	loader                 SurfaceLoader
	genSyncChecker         GenSyncChecker
	codeFacts              CodeFactsClient
	repoRoot               string
	catalog                FindingCatalog
	fleetReachabilityIndex FleetReachabilityIndex
}

type Deps struct {
	Loader                 SurfaceLoader
	GenSyncChecker         GenSyncChecker
	CodeFacts              CodeFactsClient
	RepoRoot               string
	Catalog                FindingCatalog
	FleetReachabilityIndex FleetReachabilityIndex
}

func New(d Deps) *Service {
	return &Service{
		loader:                 d.Loader,
		genSyncChecker:         d.GenSyncChecker,
		codeFacts:              d.CodeFacts,
		repoRoot:               d.RepoRoot,
		catalog:                d.Catalog,
		fleetReachabilityIndex: d.FleetReachabilityIndex,
	}
}

func (s *Service) ValidateScenario(ctx context.Context, scenario string) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Report{}, fmt.Errorf("scenario name is required")
	}
	if s.loader == nil {
		return Report{}, fmt.Errorf("proto surface loader is not configured")
	}
	collector := metricsFrom(ctx)

	discover := collector.Stage("discover")
	surface, err := s.loader.LoadScenario(scenario)
	discover.End()
	if err != nil {
		return Report{}, err
	}
	if scenarioPath := ScenarioPathFrom(ctx); scenarioPath != "" {
		protosurface.ApplyTransportFactsAtScenarioPath(&surface, scenarioPath)
	}
	fleetIndex := s.fleetReachability()

	// analyze is the hot stage; its children separate CPU-bound static surface
	// checks from the I/O-bound code-facts RPCs so the consumer can drill down.
	analyze := collector.Stage("analyze")
	var findings []Finding
	static := analyze.Stage("static-checks")
	findings = append(findings, checkCycles(surface)...)
	findings = append(findings, s.checkGeneratedArtifacts(ctx, scenario)...)
	findings = append(findings, checkPackages(scenario, surface)...)
	findings = append(findings, checkVersions(surface)...)
	findings = append(findings, checkUnsupportedAnnotations(surface)...)
	findings = append(findings, checkConstraintProtovalidateCoverage(surface)...)
	findings = append(findings, s.checkTemplateSource(surface)...)
	findings = append(findings, checkCrossDomainImports(surface)...)
	findings = append(findings, checkImportClassification(surface)...)
	findings = append(findings, checkTransport(surface)...)
	findings = append(findings, checkRESTPayloadDeclarations(surface)...)
	findings = append(findings, checkStability(surface)...)
	findings = append(findings, checkSharedTypePlacement(surface)...)
	findings = append(findings, checkMissingHealth(surface)...)
	findings = append(findings, checkPossiblyUnused(surface, fleetIndex)...)
	findings = append(findings, s.checkDomainMismatch(surface)...)
	static.Gauge("findings", float64(len(findings)))
	static.End()
	codeFacts := analyze.Stage("code-facts")
	findings = append(findings, s.checkCodeFacts(ctx, scenario, surface)...)
	codeFacts.End()
	analyze.Gauge("proto_files", float64(len(surface.Files)))
	analyze.End()

	resolve := collector.Stage("resolve")
	findings, err = s.resolveSeverities(findings)
	if err != nil {
		resolve.End()
		return Report{}, err
	}
	sortFindings(findings)
	report := finalize(scenario, findings)
	resolve.End()
	return report, nil
}

func (s *Service) resolveSeverities(findings []Finding) ([]Finding, error) {
	out := make([]Finding, len(findings))
	for i, finding := range findings {
		severity, err := s.catalog.ResolveSeverity(finding.Code)
		if err != nil {
			return nil, err
		}
		finding.Severity = severity
		out[i] = finding
	}
	return out, nil
}

func (s *Service) checkCodeFacts(ctx context.Context, scenario string, surface protosurface.Surface) []Finding {
	if s.codeFacts == nil {
		return nil
	}
	findings := s.checkProtoAdoptionFacts(ctx, scenario)
	if len(surface.RESTExceptions) == 0 {
		return findings
	}
	endpointIDs := make([]string, 0, len(surface.RESTExceptions))
	for _, endpoint := range surface.RESTExceptions {
		endpointIDs = append(endpointIDs, endpoint.EndpointID)
	}
	sort.Strings(endpointIDs)
	findings = append(findings, s.checkEndpointProofFacts(ctx, scenario, endpointIDs)...)
	return findings
}

func (s *Service) checkProtoAdoptionFacts(ctx context.Context, scenario string) []Finding {
	report, err := s.codeFacts.CheckProtoAdoption(ctx, scenario)
	if err != nil {
		return []Finding{codeFactsUnavailableFinding("proto adoption", err)}
	}
	var findings []Finding
	for _, warning := range report.GetWarnings() {
		if isCodeFactsProofWarning(warning) {
			findings = append(findings, Finding{
				Code:       CodeProtoAdoptionUnsupported,
				Location:   "scenarios/" + scenario,
				Message:    "code-facts proto adoption analyzer warning: " + warning.GetMessage(),
				Suggestion: "start code-facts and its graph providers, or inspect the unsupported analyzer warning before treating adoption as proven",
			})
		}
	}
	for _, fact := range report.GetFacts() {
		if fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION {
			continue
		}
		status := factStatus(fact)
		switch status {
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING:
			findings = append(findings, Finding{
				Code:       CodeProtoAdoptionMissing,
				Location:   codeFactLocation(fact, "scenarios/"+scenario),
				Message:    fmt.Sprintf("surface %q has no code-facts evidence for generated proto adoption", fact.GetSubject()),
				Suggestion: "import and use generated proto clients/types on the surface, then rerun code-facts and proto-health",
			})
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED:
			findings = append(findings, Finding{
				Code:       CodeProtoAdoptionContradicted,
				Location:   codeFactLocation(fact, "scenarios/"+scenario),
				Message:    fmt.Sprintf("surface %q has contradictory code-facts proto adoption evidence", fact.GetSubject()),
				Suggestion: "align the surface with generated proto artifacts and remove hand-written contract drift",
			})
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
			findings = append(findings, Finding{
				Code:       CodeProtoAdoptionUnsupported,
				Location:   codeFactLocation(fact, "scenarios/"+scenario),
				Message:    fmt.Sprintf("surface %q proto adoption proof is %s", fact.GetSubject(), evidenceStatusLabel(status)),
				Suggestion: "treat the surface as unproven until code-facts can analyze it",
			})
		}
	}
	return findings
}

func (s *Service) checkEndpointProofFacts(ctx context.Context, scenario string, endpointIDs []string) []Finding {
	report, err := s.codeFacts.CheckEndpointProof(ctx, scenario, endpointIDs)
	if err != nil {
		return []Finding{codeFactsUnavailableFinding("endpoint proof", err)}
	}
	var findings []Finding
	for _, warning := range report.GetWarnings() {
		if isCodeFactsProofWarning(warning) {
			findings = append(findings, Finding{
				Code:       CodeEndpointProofUnsupported,
				Location:   "scenarios/" + scenario + "/.vrooli/endpoints.json",
				Message:    "code-facts endpoint analyzer warning: " + warning.GetMessage(),
				Suggestion: "start code-facts and graph providers, or inspect unsupported analyzer output before treating REST exception implementation as proven",
			})
		}
	}
	for _, fact := range report.GetFacts() {
		if fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS {
			continue
		}
		status := factStatus(fact)
		switch status {
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING:
			findings = append(findings, Finding{
				Code:       CodeEndpointProofMissing,
				Location:   codeFactLocation(fact, "scenarios/"+scenario+"/.vrooli/endpoints.json"),
				Message:    fmt.Sprintf("REST exception endpoint %q has declarations but no code-facts implementation proof", fact.GetSubject()),
				Suggestion: "implement the declared proto payload with generated helpers/types, or change the declaration if the endpoint is not proto-backed",
			})
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED:
			findings = append(findings, Finding{
				Code:       CodeEndpointProofContradicted,
				Location:   codeFactLocation(fact, "scenarios/"+scenario+"/.vrooli/endpoints.json"),
				Message:    fmt.Sprintf("REST exception endpoint %q implementation contradicts its declared proto payload", fact.GetSubject()),
				Suggestion: "make the handler write the declared proto payload type or update the endpoint declaration",
			})
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
			findings = append(findings, Finding{
				Code:       CodeEndpointProofUnsupported,
				Location:   codeFactLocation(fact, "scenarios/"+scenario+"/.vrooli/endpoints.json"),
				Message:    fmt.Sprintf("REST exception endpoint %q implementation proof is %s", fact.GetSubject(), evidenceStatusLabel(status)),
				Suggestion: "treat the endpoint implementation as unproven until code-facts can analyze it",
			})
		}
	}
	return findings
}

func codeFactsUnavailableFinding(scope string, err error) Finding {
	return Finding{
		Code:       CodeCodeFactsUnavailable,
		Location:   "code-facts",
		Message:    fmt.Sprintf("code-facts %s evidence is unavailable: %v", scope, err),
		Suggestion: "start code-facts through the Vrooli lifecycle and rerun proto-health validation",
	}
}

func factStatus(fact *factsv1.GenericFact) factsv1.EvidenceStatus {
	if fact.GetFamily() == factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION || fact.GetFamily() == factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS {
		if len(fact.GetEvidence()) > 0 && fact.GetEvidence()[0].GetStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSPECIFIED {
			return fact.GetEvidence()[0].GetStatus()
		}
	}
	status := factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSPECIFIED
	for _, ev := range fact.GetEvidence() {
		switch ev.GetStatus() {
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED:
			return ev.GetStatus()
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING:
			status = ev.GetStatus()
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
			if status != factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING {
				status = ev.GetStatus()
			}
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED:
			if status == factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSPECIFIED || status == factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN {
				status = ev.GetStatus()
			}
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN:
			if status == factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSPECIFIED {
				status = ev.GetStatus()
			}
		}
	}
	return status
}

func isCodeFactsProofWarning(warning *factsv1.Warning) bool {
	switch warning.GetStatus() {
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED:
		return true
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
		code := warning.GetCode()
		return strings.HasSuffix(code, ".unavailable") ||
			strings.HasSuffix(code, ".empty_graph") ||
			strings.Contains(code, "_unsupported") ||
			strings.Contains(code, ".unsupported")
	default:
		return false
	}
}

func codeFactLocation(fact *factsv1.GenericFact, fallback string) string {
	for _, ev := range fact.GetEvidence() {
		if file := ev.GetRange().GetFile(); file != "" {
			return file
		}
	}
	if path := fact.GetAttributes()["path"]; path != "" {
		return path
	}
	return fallback
}

func evidenceStatusLabel(status factsv1.EvidenceStatus) string {
	switch status {
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED:
		return "unsupported"
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
		return "unknown"
	default:
		return status.String()
	}
}

func (s *Service) checkGeneratedArtifacts(ctx context.Context, scenario string) []Finding {
	if s.genSyncChecker == nil {
		return nil
	}
	status, err := s.genSyncChecker.CheckScenario(ctx, scenario)
	if err != nil {
		log.Printf("proto-health gen-sync scenario=%s outcome=error detail=%q", scenario, err.Error())
		return []Finding{{
			Code:       CodeGenOutOfSync,
			Location:   "packages/proto/gen",
			Message:    "generated proto artifact sync check failed: " + err.Error(),
			Suggestion: "run cd packages/proto && make generate, then rerun proto-health validation",
		}}
	}
	if status.Skipped {
		log.Printf("proto-health gen-sync scenario=%s outcome=skipped detail=%q", scenario, status.SkipMessage)
		return nil
	}
	if status.ManifestMissing {
		log.Printf("proto-health gen-sync scenario=%s outcome=manifest_missing detail=%q", scenario, status.Detail)
		location := filepath.ToSlash(filepath.Join("packages", "proto", "gen", "manifests", scenario+".lock.json"))
		return []Finding{{
			Code:       CodeGenManifestMissing,
			Location:   location,
			Message:    "generated proto manifest is missing",
			Suggestion: "run cd packages/proto && make generate and commit the generated manifest",
		}}
	}
	if status.InSync && !status.ToolchainDrift {
		log.Printf("proto-health gen-sync scenario=%s outcome=in_sync", scenario)
		return nil
	}
	var findings []Finding
	if len(status.Drift) > 0 {
		location := "packages/proto/gen"
		if len(status.Drift) > 0 {
			location = status.Drift[0]
		}
		message := "generated proto artifacts are out of sync with schemas"
		if status.Detail != "" {
			message += ": " + status.Detail
		}
		log.Printf("proto-health gen-sync scenario=%s outcome=drift drift=%q detail=%q", scenario, strings.Join(status.Drift, ","), status.Detail)
		findings = append(findings, Finding{
			Code:       CodeGenOutOfSync,
			Location:   location,
			Message:    message,
			Suggestion: "run cd packages/proto && make generate and commit the generated artifacts",
		})
	}
	if status.ToolchainDrift {
		log.Printf("proto-health gen-sync scenario=%s outcome=toolchain_drift detail=%q", scenario, status.Detail)
		findings = append(findings, Finding{
			Code:       CodeGenToolchainDrift,
			Location:   "packages/proto/gen/manifests",
			Message:    "proto codegen toolchain pins changed since the generation manifest was written",
			Suggestion: "run cd packages/proto && make generate and commit the refreshed manifests",
		})
	}
	return findings
}

func (s *Service) DescribeScenarioProtos(_ context.Context, scenario string) (protosurface.Surface, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return protosurface.Surface{}, fmt.Errorf("scenario name is required")
	}
	if s.loader == nil {
		return protosurface.Surface{}, fmt.Errorf("proto surface loader is not configured")
	}
	return s.loader.LoadScenario(scenario)
}

func (s *Service) DescribeScenariosProtos(_ context.Context, scenarios []string, limit int32, stabilityFilter string) ([]SurfaceResult, error) {
	if s.loader == nil {
		return nil, fmt.Errorf("proto surface loader is not configured")
	}
	if limit < 0 || limit > 500 {
		return nil, fmt.Errorf("limit must be between 0 and 500")
	}
	scenarios = normalizeScenarios(scenarios)
	if len(scenarios) == 0 {
		listed, err := s.loader.ListScenarios()
		if err != nil {
			return nil, err
		}
		scenarios = listed
	}
	if limit > 0 && len(scenarios) > int(limit) {
		scenarios = scenarios[:limit]
	}
	results := make([]SurfaceResult, 0, len(scenarios))
	for _, scenario := range scenarios {
		surface, err := s.loader.LoadScenario(scenario)
		result := SurfaceResult{Scenario: scenario}
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Surface = filterSurfaceByStability(surface, strings.TrimSpace(stabilityFilter))
		}
		results = append(results, result)
	}
	return results, nil
}

func normalizeScenarios(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, scenario := range in {
		scenario = strings.TrimSpace(scenario)
		if scenario == "" || seen[scenario] {
			continue
		}
		seen[scenario] = true
		out = append(out, scenario)
	}
	return out
}

func filterSurfaceByStability(surface protosurface.Surface, stability string) protosurface.Surface {
	if stability == "" {
		return surface
	}
	keptFiles := map[string]bool{}
	filtered := surface
	filtered.Files = nil
	for _, file := range surface.Files {
		if file.Stability != stability {
			continue
		}
		filtered.Files = append(filtered.Files, file)
		keptFiles[file.Path] = true
	}
	filtered.Services = filterServicesByFiles(surface.Services, keptFiles)
	filtered.Messages = filterMessagesByFiles(surface.Messages, keptFiles)
	filtered.IntraScenarioImports = filterImportsBySourceFile(surface.IntraScenarioImports, keptFiles)
	filtered.CrossScenarioImports = filterImportsBySourceFile(surface.CrossScenarioImports, keptFiles)
	filtered.RESTExceptionRefs = filterRESTExceptionRefsByKeptDomains(surface, keptFiles)
	filtered.RESTExceptions = filterRESTExceptionsByKeptDomains(surface, keptFiles)
	filtered.RESTExceptionPayloads = filterRESTPayloadsByKeptDomains(surface, keptFiles)
	return filtered
}

func filterServicesByFiles(services []protosurface.Service, keptFiles map[string]bool) []protosurface.Service {
	var out []protosurface.Service
	for _, service := range services {
		if keptFiles[service.FilePath] {
			out = append(out, service)
		}
	}
	return out
}

func filterMessagesByFiles(messages []protosurface.Message, keptFiles map[string]bool) []protosurface.Message {
	var out []protosurface.Message
	for _, message := range messages {
		if keptFiles[message.FilePath] {
			out = append(out, message)
		}
	}
	return out
}

func filterImportsBySourceFile(imports []protosurface.Import, keptFiles map[string]bool) []protosurface.Import {
	var out []protosurface.Import
	for _, imp := range imports {
		if keptFiles[imp.FromFile] {
			out = append(out, imp)
		}
	}
	return out
}

func filterRESTExceptionRefsByKeptDomains(surface protosurface.Surface, keptFiles map[string]bool) []protosurface.RESTExceptionRef {
	var out []protosurface.RESTExceptionRef
	for _, ref := range surface.RESTExceptionRefs {
		if ref.Domain == "" || domainHasKeptFile(surface, keptFiles, ref.Domain) {
			out = append(out, ref)
		}
	}
	return out
}

func filterRESTExceptionsByKeptDomains(surface protosurface.Surface, keptFiles map[string]bool) []protosurface.RESTExceptionEndpoint {
	var out []protosurface.RESTExceptionEndpoint
	for _, endpoint := range surface.RESTExceptions {
		if endpoint.Domain == "" || domainHasKeptFile(surface, keptFiles, endpoint.Domain) {
			out = append(out, endpoint)
		}
	}
	return out
}

func filterRESTPayloadsByKeptDomains(surface protosurface.Surface, keptFiles map[string]bool) []protosurface.RESTExceptionPayloadRef {
	var out []protosurface.RESTExceptionPayloadRef
	for _, payload := range surface.RESTExceptionPayloads {
		if payload.Domain == "" || domainHasKeptFile(surface, keptFiles, payload.Domain) {
			out = append(out, payload)
		}
	}
	return out
}

func domainHasKeptFile(surface protosurface.Surface, keptFiles map[string]bool, domain string) bool {
	for _, file := range surface.Files {
		if file.Domain == domain && keptFiles[file.Path] {
			return true
		}
	}
	return false
}

func checkCycles(surface protosurface.Surface) []Finding {
	graph := map[string][]string{}
	for _, f := range surface.Files {
		graph[f.Path] = nil
	}
	for _, imp := range surface.IntraScenarioImports {
		graph[imp.FromFile] = append(graph[imp.FromFile], imp.ToFile)
	}

	const (
		unseen = 0
		active = 1
		done   = 2
	)
	state := map[string]int{}
	var stack []string
	var findings []Finding
	var visit func(string)
	visit = func(path string) {
		state[path] = active
		stack = append(stack, path)
		for _, next := range graph[path] {
			if _, ok := graph[next]; !ok {
				continue
			}
			switch state[next] {
			case unseen:
				visit(next)
			case active:
				cycle := append([]string{}, stack...)
				for len(cycle) > 0 && cycle[0] != next {
					cycle = cycle[1:]
				}
				cycle = append(cycle, next)
				findings = append(findings, Finding{
					Code:       CodeCycle,
					Location:   path,
					Message:    "proto imports form a cycle: " + strings.Join(cycle, " -> "),
					Suggestion: "move shared types into the scenario's shared proto domain and break the recursive import",
				})
			}
		}
		stack = stack[:len(stack)-1]
		state[path] = done
	}

	paths := make([]string, 0, len(graph))
	for path := range graph {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if state[path] == unseen {
			visit(path)
		}
	}
	return findings
}

func checkPackages(scenario string, surface protosurface.Surface) []Finding {
	var findings []Finding
	wantScenario := strings.ReplaceAll(scenario, "-", "_")
	for _, f := range surface.Files {
		parts := strings.Split(f.Package, ".")
		if len(parts) < 4 || parts[0] != "vrooli" || parts[1] != wantScenario {
			findings = append(findings, Finding{
				Code:       CodePackageMismatch,
				Location:   f.Path,
				Message:    fmt.Sprintf("package %q does not match scenario %q", f.Package, scenario),
				Suggestion: "use package vrooli.<scenario_with_underscores>.<version>.<domain>",
			})
			continue
		}
		if parts[2] != f.Version || parts[3] != strings.ReplaceAll(f.Domain, "-", "_") {
			findings = append(findings, Finding{
				Code:       CodePackageMismatch,
				Location:   f.Path,
				Message:    fmt.Sprintf("package %q does not match path version/domain %s/%s", f.Package, f.Version, f.Domain),
				Suggestion: "align the proto package with its schemas/<scenario>/<version>/<domain>/ path",
			})
		}
	}
	return findings
}

func checkVersions(surface protosurface.Surface) []Finding {
	seen := map[int]bool{}
	raw := map[string]bool{}
	var findings []Finding
	for _, f := range surface.Files {
		raw[f.Version] = true
		if !versionDirRE.MatchString(f.Version) {
			findings = append(findings, Finding{
				Code:       CodeVersionNaming,
				Location:   f.Path,
				Message:    fmt.Sprintf("version directory %q is not a natural proto version", f.Version),
				Suggestion: "use contiguous version directories named v1, v2, ...",
			})
			continue
		}
		n, _ := strconv.Atoi(strings.TrimPrefix(f.Version, "v"))
		seen[n] = true
	}
	if len(raw) == 0 || len(seen) == 0 {
		return findings
	}
	max := 0
	for n := range seen {
		if n > max {
			max = n
		}
	}
	for n := 1; n <= max; n++ {
		if !seen[n] {
			findings = append(findings, Finding{
				Code:       CodeVersionNaming,
				Location:   "packages/proto/schemas/" + surface.Scenario,
				Message:    fmt.Sprintf("scenario proto versions are not contiguous; missing v%d", n),
				Suggestion: "start at v1 and avoid gaps between version directories",
			})
		}
	}
	return findings
}

func checkUnsupportedAnnotations(surface protosurface.Surface) []Finding {
	supported := map[string]bool{
		"stability":  true,
		"deprecated": true,
		"example":    true,
		"see":        true,
		"format":     true,
		"unit":       true,
		"default":    true,
		"template":   true,
		"constraint": true,
	}
	deprecated := map[string]bool{"layer": true, "domain": true, "imports": true}
	var findings []Finding
	for _, f := range surface.Files {
		for _, a := range f.Annotations {
			if supported[a.Name] {
				continue
			}
			msg := fmt.Sprintf("annotation @%s is not in the supported registry", a.Name)
			if deprecated[a.Name] {
				msg = fmt.Sprintf("annotation @%s is deprecated; the value is derivable from proto structure", a.Name)
			}
			findings = append(findings, Finding{
				Code:       CodeUnsupportedAnnotation,
				Location:   f.Path,
				Message:    msg,
				Suggestion: "keep only annotations listed in packages/proto/STYLE_GUIDE.md",
			})
		}
	}
	return findings
}

func checkConstraintProtovalidateCoverage(surface protosurface.Surface) []Finding {
	var findings []Finding
	for _, msg := range surface.Messages {
		if hasAnnotation(msg.Annotations, "constraint") && !msg.HasValidationRules {
			findings = append(findings, Finding{
				Code:       CodeConstraintMissingProtovalidate,
				Location:   msg.FilePath,
				Message:    fmt.Sprintf("message %s uses @constraint without a Protovalidate message rule", msg.FullName),
				Suggestion: "move enforceable constraints into (buf.validate.message); keep prose only for semantic guidance that cannot be machine-checked",
			})
		}
		for _, field := range msg.Fields {
			if hasAnnotation(field.Annotations, "constraint") && !field.HasValidationRules {
				findings = append(findings, Finding{
					Code:       CodeConstraintMissingProtovalidate,
					Location:   msg.FilePath,
					Message:    fmt.Sprintf("field %s.%s uses @constraint without a Protovalidate field rule", msg.FullName, field.Name),
					Suggestion: "move enforceable constraints into (buf.validate.field); keep prose only for semantic guidance that cannot be machine-checked",
				})
			}
		}
	}
	return findings
}

func hasAnnotation(annotations []protosurface.Annotation, name string) bool {
	for _, a := range annotations {
		if a.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) checkTemplateSource(surface protosurface.Surface) []Finding {
	var findings []Finding
	for _, f := range surface.Files {
		for _, a := range f.Annotations {
			if a.Name != "template" {
				continue
			}
			if s.templateSourceAdopted(surface.Scenario, f, a.Value) {
				continue
			}
			findings = append(findings, Finding{
				Code:       CodeTemplateSource,
				Location:   f.Path,
				Message:    fmt.Sprintf("proto file is marked as template-sourced (%s)", a.Value),
				Suggestion: "remove @template after the file diverges from its template scaffold and becomes scenario-owned surface",
			})
		}
	}
	return findings
}

func (s *Service) templateSourceAdopted(scenario string, f protosurface.File, templateValue string) bool {
	if s.repoRoot == "" {
		return false
	}
	templatePath, ok := templateSourcePath(s.repoRoot, scenario, f.Path, templateValue)
	if !ok {
		return false
	}
	scenarioPath := filepath.Join(s.repoRoot, "packages", "proto", "schemas", filepath.FromSlash(f.Path))
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return false
	}
	scenarioBytes, err := os.ReadFile(scenarioPath)
	if err != nil {
		return false
	}
	templateBytes = normalizeTemplateProtoBytes(templateBytes, scenario)
	scenarioBytes = normalizeTemplateProtoBytes(scenarioBytes, scenario)
	return !bytes.Equal(templateBytes, scenarioBytes)
}

func templateSourcePath(repoRoot, scenario, filePath, templateValue string) (string, bool) {
	if strings.TrimSpace(templateValue) != "react-vite/example" {
		return "", false
	}
	rel, ok := strings.CutPrefix(filepath.ToSlash(filePath), strings.TrimSpace(scenario)+"/")
	if !ok || rel == "" {
		return "", false
	}
	return filepath.Join(repoRoot, "templates", "scenarios", "react-vite", "proto", filepath.FromSlash(rel)), true
}

func normalizeTemplateProtoBytes(in []byte, scenario string) []byte {
	out := bytes.ReplaceAll(in, []byte("{{SCENARIO_ID}}"), []byte(scenario))
	out = bytes.ReplaceAll(out, []byte("{{SCENARIO_ID_SNAKE}}"), []byte(strings.ReplaceAll(scenario, "-", "_")))
	return bytes.TrimSpace(out)
}

func checkCrossDomainImports(surface protosurface.Surface) []Finding {
	suppressed := crossDomainImportsCoveredBySharedTypeErrors(surface)
	var findings []Finding
	for _, imp := range surface.IntraScenarioImports {
		if imp.FromDomain == "" || imp.ToDomain == "" || imp.FromDomain == imp.ToDomain || imp.ToDomain == "shared" {
			continue
		}
		if suppressed[crossDomainImportKey(imp.FromFile, imp.ToFile)] {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodeCrossDomainImport,
			Location:   imp.FromFile,
			Message:    fmt.Sprintf("domain %q imports scenario domain %q via %s", imp.FromDomain, imp.ToDomain, imp.ToFile),
			Suggestion: "move cross-domain shared types into v1/shared/ and import that shared proto instead",
		})
	}
	return findings
}

func crossDomainImportsCoveredBySharedTypeErrors(surface protosurface.Surface) map[string]bool {
	referenceDomains := sharedTypeReferenceDomains(surface)
	suppressed := map[string]bool{}
	for _, msg := range surface.Messages {
		domains := referenceDomains[msg.FullName]
		if msg.Domain == "shared" || len(domains) < 2 {
			continue
		}
		for _, imp := range surface.IntraScenarioImports {
			if imp.ToFile != msg.FilePath || imp.FromDomain == "" || imp.FromDomain == imp.ToDomain || imp.ToDomain == "shared" {
				continue
			}
			if domains[imp.FromDomain] && domains[msg.Domain] {
				suppressed[crossDomainImportKey(imp.FromFile, imp.ToFile)] = true
			}
		}
	}
	return suppressed
}

func crossDomainImportKey(fromFile, toFile string) string {
	return fromFile + "\x00" + toFile
}

func checkImportClassification(surface protosurface.Surface) []Finding {
	var findings []Finding
	for _, imp := range append(append([]protosurface.Import{}, surface.IntraScenarioImports...), surface.CrossScenarioImports...) {
		if imp.Kind != protosurface.ImportKindUnspecified && imp.Kind != "" {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodeImportKindUnknown,
			Location:   imp.FromFile,
			Message:    fmt.Sprintf("proto import %s -> %s has no classified import kind", imp.FromFile, imp.ToFile),
			Suggestion: "ensure proto-health can classify imports as scenario-local, cross-scenario, or external from descriptor metadata",
		})
	}
	return findings
}

func checkTransport(surface protosurface.Surface) []Finding {
	if surface.TransportWorld != protosurface.TransportWorldHandRolled && surface.TransportWorld != protosurface.TransportWorldMixed {
		return nil
	}
	return []Finding{{
		Code:       CodeHandRolledTransport,
		Location:   "scenarios/" + surface.Scenario + "/api",
		Message:    fmt.Sprintf("scenario proto transport world is %s", surface.TransportWorld),
		Suggestion: "prefer generated Connect handlers for proto-owned RPCs; keep REST only for documented transport exceptions",
	}}
}

func checkRESTPayloadDeclarations(surface protosurface.Surface) []Finding {
	messageByName := map[string]protosurface.Message{}
	for _, m := range surface.Messages {
		messageByName[m.FullName] = m
	}
	payloadsByEndpoint := map[string][]protosurface.RESTExceptionPayloadRef{}
	for _, ref := range surface.RESTExceptionPayloads {
		payloadsByEndpoint[ref.EndpointID] = append(payloadsByEndpoint[ref.EndpointID], ref)
	}

	var findings []Finding
	for _, endpoint := range surface.RESTExceptions {
		payloads := payloadsByEndpoint[endpoint.EndpointID]
		if !endpoint.HasPayloadDeclarations || len(payloads) == 0 {
			if isConventionalInfraEndpoint(endpoint) {
				continue
			}
			findings = append(findings, Finding{
				Code:       CodeRESTPayloadMissingDeclaration,
				Location:   endpoint.Path,
				Message:    fmt.Sprintf("REST exception endpoint %q does not declare request, response, and error proto payload intent", endpoint.EndpointID),
				Suggestion: "add rest_exception.proto_payloads with explicit request, response, and error declarations",
			})
			continue
		}
		seenRoles := map[protosurface.RESTPayloadRole]bool{}
		for _, ref := range payloads {
			seenRoles[ref.Role] = true
			if !validRESTPayloadConformance(ref.Conformance) {
				findings = append(findings, Finding{
					Code:       CodeRESTPayloadInvalidConformance,
					Location:   ref.Path,
					Message:    fmt.Sprintf("REST exception endpoint %q %s conformance %q is unsupported", ref.EndpointID, ref.Role, ref.Conformance),
					Suggestion: "use one of none, transport_only, external_shape, or protojson",
				})
			}
			if ref.Conformance == "protojson" && ref.ProtoFullName == "" {
				findings = append(findings, Finding{
					Code:       CodeRESTPayloadMissingDeclaration,
					Location:   ref.Path,
					Message:    fmt.Sprintf("REST exception endpoint %q %s declares protojson without proto_full_name", ref.EndpointID, ref.Role),
					Suggestion: "set proto_full_name to the fully qualified proto message name, or use a non-proto conformance mode",
				})
				continue
			}
			if ref.ProtoFullName == "" {
				continue
			}
			if _, ok := messageByName[ref.ProtoFullName]; !ok {
				findings = append(findings, Finding{
					Code:       CodeRESTPayloadUnknownMessage,
					Location:   ref.Path,
					Message:    fmt.Sprintf("REST exception endpoint %q %s declares unknown proto message %q", ref.EndpointID, ref.Role, ref.ProtoFullName),
					Suggestion: "use a full proto message name present in the descriptor image",
				})
			}
		}
		for _, role := range []protosurface.RESTPayloadRole{protosurface.RESTPayloadRoleRequest, protosurface.RESTPayloadRoleResponse, protosurface.RESTPayloadRoleError} {
			if seenRoles[role] {
				continue
			}
			findings = append(findings, Finding{
				Code:       CodeRESTPayloadMissingDeclaration,
				Location:   endpoint.Path,
				Message:    fmt.Sprintf("REST exception endpoint %q is missing %s payload intent", endpoint.EndpointID, role),
				Suggestion: "declare every REST payload role explicitly, even when the role has no proto payload",
			})
		}
	}
	return findings
}

func isConventionalInfraEndpoint(endpoint protosurface.RESTExceptionEndpoint) bool {
	// Liveness and IdP discovery endpoints are infrastructure probes, not
	// business contracts. Keep this allowlist narrow so REST exceptions cannot
	// bypass payload declarations for product APIs.
	id := strings.ToLower(strings.TrimSpace(endpoint.EndpointID))
	path := strings.ToLower(strings.TrimSpace(endpoint.Path))
	switch id {
	case "health", "jwks":
		return true
	}
	return path == "/health" || path == "/jwks" || strings.HasPrefix(path, "/.well-known/")
}

func validRESTPayloadConformance(v string) bool {
	switch v {
	case "none", "transport_only", "external_shape", "protojson":
		return true
	default:
		return false
	}
}

func checkStability(surface protosurface.Surface) []Finding {
	servedFiles := map[string]bool{}
	for _, svc := range surface.Services {
		for _, rpc := range svc.RPCs {
			if rpc.Transport == protosurface.TransportKindConnect || rpc.Transport == protosurface.TransportKindREST || rpc.Transport == protosurface.TransportKindHandRolled {
				servedFiles[svc.FilePath] = true
			}
		}
	}

	var findings []Finding
	for _, f := range surface.Files {
		if servedFiles[f.Path] && f.Stability == "experimental" {
			findings = append(findings, Finding{
				Code:       CodeStabilityDishonest,
				Location:   f.Path,
				Message:    "served proto service is still marked @stability experimental",
				Suggestion: "mark implemented public contracts stable, or stop serving the RPC until the contract is ready",
			})
		}
		if f.Stability == "stable" && fileHasService(surface, f.Path) && !servedFiles[f.Path] {
			findings = append(findings, Finding{
				Code:       CodeStabilityDishonest,
				Location:   f.Path,
				Message:    "stable proto service has no discovered implementation",
				Suggestion: "mount the generated Connect handler or lower the stability annotation until implemented",
			})
		}
	}
	findings = append(findings, checkStabilityDependencies(surface)...)
	return findings
}

func checkStabilityDependencies(surface protosurface.Surface) []Finding {
	fileStability := map[string]string{}
	for _, f := range surface.Files {
		fileStability[f.Path] = f.Stability
	}
	messageByName := map[string]protosurface.Message{}
	for _, m := range surface.Messages {
		messageByName[m.FullName] = m
	}

	reported := map[string]bool{}
	var findings []Finding
	for _, svc := range surface.Services {
		if stabilityRank(fileStability[svc.FilePath]) < stabilityRank("stable") {
			continue
		}
		for _, rpc := range svc.RPCs {
			if rpc.Transport != protosurface.TransportKindConnect && rpc.Transport != protosurface.TransportKindREST && rpc.Transport != protosurface.TransportKindHandRolled {
				continue
			}
			for _, root := range []string{rpc.Input, rpc.Output} {
				for _, dep := range transitiveMessages(root, messageByName) {
					depStability := fileStability[dep.FilePath]
					if stabilityRank(depStability) >= stabilityRank("stable") {
						continue
					}
					key := svc.FullName + "|" + rpc.Name + "|" + dep.FullName
					if reported[key] {
						continue
					}
					reported[key] = true
					findings = append(findings, Finding{
						Code:       CodeStabilityDependencyMismatch,
						Location:   dep.FilePath + "#" + dep.Name,
						Message:    fmt.Sprintf("stable RPC %s/%s depends on less-stable message %s (%s)", svc.FullName, rpc.Name, dep.FullName, stabilityLabel(depStability)),
						Suggestion: "mark the transitive payload stable before serving it from a stable contract, or lower the serving contract stability",
					})
				}
			}
		}
	}
	return findings
}

func transitiveMessages(root string, messages map[string]protosurface.Message) []protosurface.Message {
	seen := map[string]bool{}
	var out []protosurface.Message
	var walk func(string)
	walk = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		msg, ok := messages[name]
		if !ok {
			return
		}
		out = append(out, msg)
		for _, field := range msg.Fields {
			if field.MessageType != "" {
				walk(field.MessageType)
			}
		}
	}
	walk(root)
	return out
}

func stabilityRank(stability string) int {
	switch stability {
	case "stable":
		return 3
	case "beta":
		return 2
	case "experimental":
		return 1
	default:
		return 0
	}
}

func stabilityLabel(stability string) string {
	if stability == "" {
		return "unspecified"
	}
	return stability
}

func fileHasService(surface protosurface.Surface, path string) bool {
	for _, svc := range surface.Services {
		if svc.FilePath == path {
			return true
		}
	}
	return false
}

func checkMissingHealth(surface protosurface.Surface) []Finding {
	for _, f := range surface.Files {
		if f.Path == surface.Scenario+"/v1/shared/health.proto" || f.Path == surface.Scenario+"/v1/health/health.proto" {
			return nil
		}
	}
	return []Finding{{
		Code:       CodeMissingHealthProto,
		Location:   "packages/proto/schemas/" + surface.Scenario,
		Message:    "scenario has no health proto",
		Suggestion: "add v1/shared/health.proto for health payloads, or document why the scenario has no proto-owned health surface",
	}}
}

func checkSharedTypePlacement(surface protosurface.Surface) []Finding {
	referenceDomains := sharedTypeReferenceDomains(surface)
	var findings []Finding
	for _, msg := range surface.Messages {
		domains := sortedDomains(referenceDomains[msg.FullName])
		if len(domains) < 2 || msg.Domain == "shared" {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodeSharedTypeMisplaced,
			Location:   msg.FilePath + "#" + msg.Name,
			Message:    fmt.Sprintf("message %s is reused across domains (%s) but lives in %q", msg.FullName, strings.Join(domains, ", "), msg.Domain),
			Suggestion: "move reusable scenario-local support messages into v1/shared/ and update imports/declarations",
		})
	}
	return findings
}

func sharedTypeReferenceDomains(surface protosurface.Surface) map[string]map[string]bool {
	referenceDomains := map[string]map[string]bool{}
	addRef := func(fullName, domain string) {
		if fullName == "" || domain == "" {
			return
		}
		if referenceDomains[fullName] == nil {
			referenceDomains[fullName] = map[string]bool{}
		}
		referenceDomains[fullName][domain] = true
	}
	for _, svc := range surface.Services {
		for _, rpc := range svc.RPCs {
			addRef(rpc.Input, svc.Domain)
			addRef(rpc.Output, svc.Domain)
		}
	}
	for _, msg := range surface.Messages {
		for _, field := range msg.Fields {
			addRef(field.MessageType, msg.Domain)
		}
	}
	for _, ref := range surface.RESTExceptionPayloads {
		addRef(ref.ProtoFullName, ref.Domain)
	}
	return referenceDomains
}

func sortedDomains(domains map[string]bool) []string {
	out := make([]string, 0, len(domains))
	for domain := range domains {
		out = append(out, domain)
	}
	sort.Strings(out)
	return out
}

func (s *Service) fleetReachability() FleetReachabilityIndex {
	if s.fleetReachabilityIndex != nil {
		return s.fleetReachabilityIndex
	}
	if s.loader == nil {
		return nil
	}
	scenarios, err := s.loader.ListScenarios()
	if err != nil {
		log.Printf("proto-health: build fleet reachability index: list scenarios: %v", err)
		return nil
	}
	surfaces := make([]protosurface.Surface, 0, len(scenarios))
	for _, scenario := range normalizeScenarios(scenarios) {
		surface, err := s.loader.LoadScenario(scenario)
		if err != nil {
			log.Printf("proto-health: build fleet reachability index: load %s: %v", scenario, err)
			continue
		}
		surfaces = append(surfaces, surface)
	}
	return newFleetReachabilityIndex(surfaces)
}

type computedFleetReachabilityIndex struct {
	consumers map[string]map[string]bool
}

func newFleetReachabilityIndex(surfaces []protosurface.Surface) FleetReachabilityIndex {
	index := &computedFleetReachabilityIndex{consumers: map[string]map[string]bool{}}
	for _, surface := range surfaces {
		recordFleetReferences(index.consumers, surface)
	}
	return index
}

func recordFleetReferences(consumers map[string]map[string]bool, surface protosurface.Surface) {
	localMessages := messagesByFullName(surface.Messages)
	reachable := reachableMessages(surface)
	for name := range reachable {
		if _, local := localMessages[name]; !local {
			recordFleetReference(consumers, name, surface.Scenario)
		}
	}
	for name := range reachable {
		message, local := localMessages[name]
		if !local {
			continue
		}
		for _, field := range message.Fields {
			if field.MessageType == "" {
				continue
			}
			if _, local := localMessages[field.MessageType]; !local {
				recordFleetReference(consumers, field.MessageType, surface.Scenario)
			}
		}
	}
	for _, payload := range surface.RESTExceptionPayloads {
		if payload.ProtoFullName == "" {
			continue
		}
		if _, local := localMessages[payload.ProtoFullName]; !local {
			recordFleetReference(consumers, payload.ProtoFullName, surface.Scenario)
		}
	}
}

func recordFleetReference(consumers map[string]map[string]bool, messageFullName, consumerScenario string) {
	messageFullName = strings.TrimSpace(messageFullName)
	consumerScenario = strings.TrimSpace(consumerScenario)
	if messageFullName == "" || consumerScenario == "" {
		return
	}
	if consumers[messageFullName] == nil {
		consumers[messageFullName] = map[string]bool{}
	}
	consumers[messageFullName][consumerScenario] = true
}

func (i *computedFleetReachabilityIndex) Consumers(messageFullName string) []string {
	if i == nil {
		return nil
	}
	seen := i.consumers[strings.TrimSpace(messageFullName)]
	out := make([]string, 0, len(seen))
	for scenario := range seen {
		out = append(out, scenario)
	}
	sort.Strings(out)
	return out
}

func checkPossiblyUnused(surface protosurface.Surface, fleetIndex FleetReachabilityIndex) []Finding {
	reachable := reachableMessages(surface)
	unused := unusedMessages(surface, reachable, fleetIndex)
	if len(unused) == 0 {
		return nil
	}
	return []Finding{{
		Code:       CodePossiblyUnused,
		Location:   "packages/proto/schemas/" + surface.Scenario,
		Message:    fmt.Sprintf("%d message(s) are not reachable from this scenario's served RPCs: %s", len(unused), strings.Join(formatUnusedMessageNames(unused), ", ")),
		Suggestion: "remove dead messages if they are scenario-local, or add a typed retention annotation for a stable external consumer that proto-health cannot observe",
	}}
}

func reachableMessages(surface protosurface.Surface) map[string]bool {
	reachable := reachableRoots(surface)
	byFullName := messagesByFullName(surface.Messages)
	for name := range reachable {
		walkReachableMessage(name, byFullName, reachable)
	}
	return reachable
}

func reachableRoots(surface protosurface.Surface) map[string]bool {
	reachable := map[string]bool{}
	for _, svc := range surface.Services {
		for _, rpc := range svc.RPCs {
			markReachableRPC(reachable, rpc)
		}
	}
	for _, ref := range surface.RESTExceptionRefs {
		if ref.FullName != "" {
			reachable[ref.FullName] = true
		}
	}
	return reachable
}

func markReachableRPC(reachable map[string]bool, rpc protosurface.RPC) {
	if rpc.Transport != protosurface.TransportKindConnect &&
		rpc.Transport != protosurface.TransportKindREST &&
		rpc.Transport != protosurface.TransportKindHandRolled {
		return
	}
	reachable[rpc.Input] = true
	reachable[rpc.Output] = true
}

func messagesByFullName(messages []protosurface.Message) map[string]protosurface.Message {
	byFullName := map[string]protosurface.Message{}
	for _, m := range messages {
		byFullName[m.FullName] = m
	}
	return byFullName
}

func walkReachableMessage(name string, byFullName map[string]protosurface.Message, reachable map[string]bool) {
	m, ok := byFullName[name]
	if !ok {
		return
	}
	for _, f := range m.Fields {
		if f.MessageType == "" || reachable[f.MessageType] {
			continue
		}
		reachable[f.MessageType] = true
		walkReachableMessage(f.MessageType, byFullName, reachable)
	}
}

func unusedMessages(surface protosurface.Surface, reachable map[string]bool, fleetIndex FleetReachabilityIndex) []protosurface.Message {
	var unused []protosurface.Message
	fileByPath := filesByPath(surface.Files)
	for _, m := range surface.Messages {
		if reachable[m.FullName] || m.IsMapEntry || isConventionalReachabilityRoot(m) {
			continue
		}
		if isStagedExperimentalMessage(surface, m, fileByPath[m.FilePath]) {
			continue
		}
		if hasFleetConsumer(surface.Scenario, m.FullName, fleetIndex) {
			continue
		}
		if hasValidRetentionAnnotation(surface, m, fileByPath[m.FilePath], fleetIndex) {
			continue
		}
		unused = append(unused, m)
	}
	sort.Slice(unused, func(i, j int) bool {
		return unused[i].FullName < unused[j].FullName
	})
	return unused
}

func filesByPath(files []protosurface.File) map[string]protosurface.File {
	byPath := map[string]protosurface.File{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	return byPath
}

func isStagedExperimentalMessage(surface protosurface.Surface, m protosurface.Message, file protosurface.File) bool {
	if fileHasService(surface, m.FilePath) {
		return false
	}
	return annotationValueLocal(m.Annotations, "stability") == "experimental" || file.Stability == "experimental"
}

func hasValidRetentionAnnotation(surface protosurface.Surface, m protosurface.Message, file protosurface.File, fleetIndex FleetReachabilityIndex) bool {
	for _, a := range m.Annotations {
		if a.Name != "see" {
			continue
		}
		kind, target, ok := parseRetentionAnnotation(a.Value)
		if !ok {
			continue
		}
		switch kind {
		case "external":
			return file.Stability == "stable" || annotationValueLocal(m.Annotations, "stability") == "stable"
		case "consumer":
			if file.Stability != "stable" && annotationValueLocal(m.Annotations, "stability") != "stable" {
				continue
			}
			for _, consumer := range consumersForMessage(m.FullName, fleetIndex) {
				if consumer == target && consumer != surface.Scenario {
					return true
				}
			}
		}
	}
	return false
}

func annotationValueLocal(annotations []protosurface.Annotation, name string) string {
	for _, a := range annotations {
		if a.Name == name {
			return a.Value
		}
	}
	return ""
}

func parseRetentionAnnotation(value string) (kind, target string, ok bool) {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "consumer:"):
		kind = "consumer"
		target = strings.TrimSpace(strings.TrimPrefix(value, "consumer:"))
	case strings.HasPrefix(value, "external:"):
		kind = "external"
		target = strings.TrimSpace(strings.TrimPrefix(value, "external:"))
	default:
		return "", "", false
	}
	if target == "" {
		return "", "", false
	}
	return kind, target, true
}

func hasFleetConsumer(producerScenario, messageFullName string, fleetIndex FleetReachabilityIndex) bool {
	if fleetIndex == nil {
		return false
	}
	for _, consumer := range consumersForMessage(messageFullName, fleetIndex) {
		if consumer != "" && consumer != producerScenario {
			return true
		}
	}
	return false
}

func consumersForMessage(messageFullName string, fleetIndex FleetReachabilityIndex) []string {
	if fleetIndex == nil {
		return nil
	}
	return fleetIndex.Consumers(messageFullName)
}

func formatUnusedMessageNames(unused []protosurface.Message) []string {
	const maxListed = 12
	names := make([]string, 0, minInt(len(unused), maxListed))
	for i := 0; i < len(unused) && i < maxListed; i++ {
		names = append(names, unused[i].FullName)
	}
	if len(unused) > maxListed {
		names = append(names, fmt.Sprintf("+%d more", len(unused)-maxListed))
	}
	return names
}

func isConventionalReachabilityRoot(m protosurface.Message) bool {
	if !isConventionalHandlerlessDomain(m.Domain) && m.Domain != "shared" {
		return false
	}
	switch m.Name {
	case "DependencyStatus", "ErrorEnvelope", "Response":
		return true
	default:
		return strings.HasPrefix(m.Name, "Health") || strings.HasSuffix(m.Name, "HealthStatus")
	}
}

func isConventionalHandlerlessDomain(domain string) bool {
	switch domain {
	case "errors", "health":
		return true
	default:
		return false
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Service) checkDomainMismatch(surface protosurface.Surface) []Finding {
	if s.repoRoot == "" {
		return nil
	}
	handlerRoot := filepath.Join(s.repoRoot, "scenarios", surface.Scenario, "api", "handlers")
	entries, err := os.ReadDir(handlerRoot)
	if err != nil {
		return nil
	}
	handlerDomains := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			handlerDomains[entry.Name()] = true
		}
	}
	protoDomains := map[string]bool{}
	for _, f := range surface.Files {
		if f.Domain != "shared" && !isConventionalHandlerlessDomain(f.Domain) {
			protoDomains[f.Domain] = true
		}
	}
	var findings []Finding
	for domain := range protoDomains {
		if handlerDomains[domain] {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodeDomainMismatch,
			Location:   "packages/proto/schemas/" + surface.Scenario + "/v1/" + domain,
			Message:    fmt.Sprintf("proto domain %q has no matching api/handlers/%s directory", domain, domain),
			Suggestion: "add the handler domain when the proto is implemented, or move shared-only types into v1/shared/",
		})
	}
	for domain := range handlerDomains {
		if protoDomains[domain] || isConventionalHandlerlessDomain(domain) {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodeDomainMismatch,
			Location:   filepath.Join("scenarios", surface.Scenario, "api", "handlers", domain),
			Message:    fmt.Sprintf("handler domain %q has no matching proto domain", domain),
			Suggestion: "add a proto domain for this handler or remove the unused scaffold handler",
		})
	}
	return findings
}

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		if findings[i].Location != findings[j].Location {
			return findings[i].Location < findings[j].Location
		}
		return findings[i].Message < findings[j].Message
	})
}

func severityRank(s Severity) int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}
