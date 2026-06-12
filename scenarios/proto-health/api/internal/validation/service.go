package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"proto-health/internal/protosurface"
)

var versionDirRE = regexp.MustCompile(`^v[1-9][0-9]*$`)

type Service struct {
	loader         SurfaceLoader
	genSyncChecker GenSyncChecker
	repoRoot       string
}

type Deps struct {
	Loader         SurfaceLoader
	GenSyncChecker GenSyncChecker
	RepoRoot       string
}

func New(d Deps) *Service {
	return &Service{loader: d.Loader, genSyncChecker: d.GenSyncChecker, repoRoot: d.RepoRoot}
}

func (s *Service) ValidateScenario(ctx context.Context, scenario string) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Report{}, fmt.Errorf("scenario name is required")
	}
	if s.loader == nil {
		return Report{}, fmt.Errorf("proto surface loader is not configured")
	}
	surface, err := s.loader.LoadScenario(scenario)
	if err != nil {
		return Report{}, err
	}

	var findings []Finding
	findings = append(findings, checkCycles(surface)...)
	findings = append(findings, s.checkGeneratedArtifacts(ctx, scenario)...)
	findings = append(findings, checkPackages(scenario, surface)...)
	findings = append(findings, checkVersions(surface)...)
	findings = append(findings, checkUnsupportedAnnotations(surface)...)
	findings = append(findings, checkTemplateSource(surface)...)
	findings = append(findings, checkCrossDomainImports(surface)...)
	findings = append(findings, checkAdoption(surface)...)
	findings = append(findings, checkTransport(surface)...)
	findings = append(findings, checkStability(surface)...)
	findings = append(findings, checkMissingHealth(surface)...)
	findings = append(findings, checkPossiblyUnused(surface)...)
	findings = append(findings, s.checkDomainMismatch(surface)...)
	sortFindings(findings)
	return finalize(scenario, findings), nil
}

func (s *Service) checkGeneratedArtifacts(ctx context.Context, scenario string) []Finding {
	if s.genSyncChecker == nil {
		return nil
	}
	status, err := s.genSyncChecker.CheckScenario(ctx, scenario)
	if err != nil {
		return []Finding{{
			Severity:   SeverityError,
			Code:       CodeGenOutOfSync,
			Location:   "packages/proto/gen",
			Message:    "generated proto artifact sync check failed: " + err.Error(),
			Suggestion: "run cd packages/proto && make generate, then rerun proto-health validation",
		}}
	}
	if status.InSync {
		return nil
	}
	location := "packages/proto/gen"
	if len(status.Drift) > 0 {
		location = status.Drift[0]
	}
	message := "generated proto artifacts are out of sync with schemas"
	if status.Detail != "" {
		message += ": " + status.Detail
	}
	return []Finding{{
		Severity:   SeverityError,
		Code:       CodeGenOutOfSync,
		Location:   location,
		Message:    message,
		Suggestion: "run cd packages/proto && make generate and commit the generated artifacts",
	}}
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
					Severity:   SeverityError,
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
				Severity:   SeverityError,
				Code:       CodePackageMismatch,
				Location:   f.Path,
				Message:    fmt.Sprintf("package %q does not match scenario %q", f.Package, scenario),
				Suggestion: "use package vrooli.<scenario_with_underscores>.<version>.<domain>",
			})
			continue
		}
		if parts[2] != f.Version || parts[3] != strings.ReplaceAll(f.Domain, "-", "_") {
			findings = append(findings, Finding{
				Severity:   SeverityError,
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
				Severity:   SeverityWarning,
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
				Severity:   SeverityWarning,
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
				Severity:   SeverityWarning,
				Code:       CodeUnsupportedAnnotation,
				Location:   f.Path,
				Message:    msg,
				Suggestion: "keep only annotations listed in packages/proto/STYLE_GUIDE.md",
			})
		}
	}
	return findings
}

func checkTemplateSource(surface protosurface.Surface) []Finding {
	var findings []Finding
	for _, f := range surface.Files {
		for _, a := range f.Annotations {
			if a.Name != "template" {
				continue
			}
			findings = append(findings, Finding{
				Severity:   SeverityWarning,
				Code:       CodeTemplateSource,
				Location:   f.Path,
				Message:    fmt.Sprintf("proto file is marked as template-sourced (%s)", a.Value),
				Suggestion: "keep @template while this is scaffold reference code; remove the annotation only when this contract has been replaced or intentionally adopted as scenario-owned surface",
			})
		}
	}
	return findings
}

func checkCrossDomainImports(surface protosurface.Surface) []Finding {
	var findings []Finding
	for _, imp := range surface.IntraScenarioImports {
		if imp.FromDomain == "" || imp.ToDomain == "" || imp.FromDomain == imp.ToDomain || imp.ToDomain == "shared" {
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityWarning,
			Code:       CodeCrossDomainImport,
			Location:   imp.FromFile,
			Message:    fmt.Sprintf("domain %q imports scenario domain %q via %s", imp.FromDomain, imp.ToDomain, imp.ToFile),
			Suggestion: "move cross-domain shared types into v1/shared/ and import that shared proto instead",
		})
	}
	return findings
}

func checkAdoption(surface protosurface.Surface) []Finding {
	var findings []Finding
	for _, sig := range surface.AdoptionSignals {
		if sig.Present {
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityWarning,
			Code:       CodeNotAdopted,
			Location:   "scenarios/" + surface.Scenario,
			Message:    sig.Detail + " is missing",
			Suggestion: "consume the scenario's generated proto artifacts instead of parallel hand-written types",
		})
	}
	return findings
}

func checkTransport(surface protosurface.Surface) []Finding {
	if surface.TransportWorld != protosurface.TransportWorldHandRolled && surface.TransportWorld != protosurface.TransportWorldMixed {
		return nil
	}
	return []Finding{{
		Severity:   SeverityWarning,
		Code:       CodeHandRolledTransport,
		Location:   "scenarios/" + surface.Scenario + "/api",
		Message:    fmt.Sprintf("scenario proto transport world is %s", surface.TransportWorld),
		Suggestion: "prefer generated Connect handlers for proto-owned RPCs; keep REST only for documented transport exceptions",
	}}
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
				Severity:   SeverityError,
				Code:       CodeStabilityDishonest,
				Location:   f.Path,
				Message:    "served proto service is still marked @stability experimental",
				Suggestion: "mark implemented public contracts stable, or stop serving the RPC until the contract is ready",
			})
		}
		if f.Stability == "stable" && fileHasService(surface, f.Path) && !servedFiles[f.Path] {
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeStabilityDishonest,
				Location:   f.Path,
				Message:    "stable proto service has no discovered implementation",
				Suggestion: "mount the generated Connect handler or lower the stability annotation until implemented",
			})
		}
	}
	return findings
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
		Severity:   SeverityWarning,
		Code:       CodeMissingHealthProto,
		Location:   "packages/proto/schemas/" + surface.Scenario,
		Message:    "scenario has no health proto",
		Suggestion: "add v1/shared/health.proto for health payloads, or document why the scenario has no proto-owned health surface",
	}}
}

func checkPossiblyUnused(surface protosurface.Surface) []Finding {
	reachable := map[string]bool{}
	for _, svc := range surface.Services {
		for _, rpc := range svc.RPCs {
			if rpc.Transport != protosurface.TransportKindConnect && rpc.Transport != protosurface.TransportKindREST && rpc.Transport != protosurface.TransportKindHandRolled {
				continue
			}
			reachable[rpc.Input] = true
			reachable[rpc.Output] = true
		}
	}
	for _, ref := range surface.RESTExceptionRefs {
		if ref.FullName != "" {
			reachable[ref.FullName] = true
		}
	}
	byFullName := map[string]protosurface.Message{}
	for _, m := range surface.Messages {
		byFullName[m.FullName] = m
	}
	var walk func(string)
	walk = func(name string) {
		m, ok := byFullName[name]
		if !ok {
			return
		}
		for _, f := range m.Fields {
			if f.MessageType != "" && !reachable[f.MessageType] {
				reachable[f.MessageType] = true
				walk(f.MessageType)
			}
		}
	}
	for name := range reachable {
		walk(name)
	}

	var findings []Finding
	for _, m := range surface.Messages {
		if reachable[m.FullName] {
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityInfo,
			Code:       CodePossiblyUnused,
			Location:   m.FilePath + "#" + m.Name,
			Message:    fmt.Sprintf("message %s is not reachable from this scenario's served RPCs", m.FullName),
			Suggestion: "remove the message if it is dead, or keep it if a downstream scenario consumes it; fleet-aware usage belongs to scenario-dependency-analyzer",
		})
	}
	return findings
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
		if f.Domain != "shared" && f.Domain != "errors" {
			protoDomains[f.Domain] = true
		}
	}
	var findings []Finding
	for domain := range protoDomains {
		if handlerDomains[domain] {
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityWarning,
			Code:       CodeDomainMismatch,
			Location:   "packages/proto/schemas/" + surface.Scenario + "/v1/" + domain,
			Message:    fmt.Sprintf("proto domain %q has no matching api/handlers/%s directory", domain, domain),
			Suggestion: "add the handler domain when the proto is implemented, or move shared-only types into v1/shared/",
		})
	}
	for domain := range handlerDomains {
		if protoDomains[domain] || domain == "health" {
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityWarning,
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
