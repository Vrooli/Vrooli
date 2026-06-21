package dependencyhealth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeSurfaceDiscoverer struct {
	inventory surfaceInventory
}

func (f fakeSurfaceDiscoverer) Discover(context.Context, string, string, string, bool) (surfaceInventory, error) {
	return f.inventory, nil
}

type fakeCommandRunner struct {
	calls []string
}

func (r *fakeCommandRunner) Run(_ context.Context, dir string, name string, args ...string) (string, error) {
	r.calls = append(r.calls, filepath.ToSlash(filepath.Join(append([]string{dir, name}, args...)...)))
	switch name {
	case "go":
		if len(args) >= 2 && args[0] == "mod" && args[1] == "tidy" {
			return "", nil
		}
		return "go version go1.25.0 linux/amd64", nil
	case "node":
		return "v24.0.0", nil
	case "python3":
		return "Python 3.12.0", nil
	default:
		return "", nil
	}
}

type staticCommandRunner struct {
	output string
	err    error
}

func (r staticCommandRunner) Run(context.Context, string, string, ...string) (string, error) {
	return r.output, r.err
}

type routedCommandRunner struct {
	outputs map[string]string
}

func (r routedCommandRunner) Run(_ context.Context, _ string, name string, args ...string) (string, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	if output, ok := r.outputs[key]; ok {
		return output, nil
	}
	return "", errors.New("unexpected command: " + key)
}

type fakeRuntimeStatusFetcher struct {
	resources     *cliv1.ResourceStatusesResponse
	resourceErr   error
	scenarios     map[string]*cliv1.ScenarioStatusSingle
	scenarioErrs  map[string]error
	scenarioCalls []string
}

func (f *fakeRuntimeStatusFetcher) ResourceStatuses(context.Context) (*cliv1.ResourceStatusesResponse, error) {
	if f.resourceErr != nil {
		return nil, f.resourceErr
	}
	return f.resources, nil
}

func (f *fakeRuntimeStatusFetcher) ScenarioStatus(_ context.Context, name string) (*cliv1.ScenarioStatusSingle, error) {
	f.scenarioCalls = append(f.scenarioCalls, name)
	if err := f.scenarioErrs[name]; err != nil {
		return nil, err
	}
	return f.scenarios[name], nil
}

func TestFinalizeCountsSeverityAndDegradedDependencies(t *testing.T) {
	resp := &healthv1.DependencyHealthResponse{
		Sections: []*healthv1.DependencyHealthSection{
			section("graph", "Dependency graph drift", "warn", "one warning"),
		},
		Findings: []*healthv1.DependencyHealthFinding{
			{Severity: "WARNING"},
			{Severity: "ERROR"},
			{Severity: "INFO"},
		},
		Surfaces: []*healthv1.DependencyHealthSurface{
			{Id: "api"},
		},
		DegradedDependencies: []*healthv1.DegradedDependency{
			{Id: "code-facts"},
		},
	}

	finalize(resp)

	if resp.GetPassed() {
		t.Fatalf("response with error finding and degraded dependency should not pass")
	}
	if got := resp.GetSummary().GetSections(); got != 1 {
		t.Fatalf("sections = %d, want 1", got)
	}
	if got := resp.GetSummary().GetSurfaces(); got != 1 {
		t.Fatalf("surfaces = %d, want 1", got)
	}
	if got := resp.GetSummary().GetFindings(); got != 3 {
		t.Fatalf("findings = %d, want 3", got)
	}
	if got := resp.GetSummary().GetErrors(); got != 1 {
		t.Fatalf("errors = %d, want 1", got)
	}
	if got := resp.GetSummary().GetWarnings(); got != 1 {
		t.Fatalf("warnings = %d, want 1", got)
	}
	if got := resp.GetSummary().GetInfos(); got != 1 {
		t.Fatalf("infos = %d, want 1", got)
	}
	if got := resp.GetSummary().GetDegradedDependencies(); got != 1 {
		t.Fatalf("degraded dependencies = %d, want 1", got)
	}
}

func TestBuildMaturityAssessmentIncludesLocalMaturity(t *testing.T) {
	spec := testMaturitySpec(t)
	resp := &healthv1.DependencyHealthResponse{
		Scenario: "demo",
		Findings: []*healthv1.DependencyHealthFinding{
			{
				Id:          "release-age.policy-ui-minimum-too-low",
				Severity:    "ERROR",
				Title:       "pnpm release-age minimum is too low",
				Description: "This pnpm workspace allows dependency versions newer than the Vrooli default cooldown.",
				Remediation: "Raise minimumReleaseAge to at least 10080 minutes.",
				FilePath:    "scenarios/demo/ui/pnpm-workspace.yaml",
				RuleId:      "dependency.release_age.minimum_value",
			},
		},
	}

	got, err := buildMaturityAssessment(resp, spec)
	if err != nil {
		t.Fatalf("buildMaturityAssessment() error = %v", err)
	}
	if got.GetProvider() != "scenario-dependency-analyzer" {
		t.Fatalf("provider = %q, want scenario-dependency-analyzer", got.GetProvider())
	}
	if got.GetPhase() != "dependencies" {
		t.Fatalf("phase = %q, want dependencies", got.GetPhase())
	}
	if got.GetLocal().GetCurrentLevel() == "" {
		t.Fatalf("local current level must be set: %+v", got.GetLocal())
	}
	if len(got.GetFindings()) != 1 || got.GetFindings()[0].GetMaturity() == nil {
		t.Fatalf("assessment findings missing maturity metadata: %+v", got.GetFindings())
	}
}

func TestBuildMaturityAssessmentRequiresSpec(t *testing.T) {
	_, err := buildMaturityAssessment(&healthv1.DependencyHealthResponse{Scenario: "demo"}, nil)
	if err == nil {
		t.Fatal("buildMaturityAssessment() unexpectedly accepted nil spec")
	}
}

func TestMaturitySpecCoversDependencyHealthRuleFamilies(t *testing.T) {
	spec := testMaturitySpec(t)
	for _, code := range []string{
		"dependency.surfaces.none",
		"dependency.runtime.resource_running",
		"dependency.runtime.scenario_healthy",
		"dependency.command.available",
		"dependency.go.tidy",
		"dependency.node.lockfile_present",
		"dependency.release_age.minimum_value",
		"dependency.governance.approved_dependency",
		"dependency.graph.undeclared",
	} {
		if _, ok := spec.Findings[code]; !ok {
			t.Fatalf("maturity spec missing emitted rule %q", code)
		}
	}
	if spec.Fallback.LocalLevelImpact == "" {
		t.Fatal("maturity spec fallback must cover newly introduced dependency-health rules explicitly")
	}
}

func TestStatusFromFindingsTreatsInfoOnlyAsPass(t *testing.T) {
	findings := []*healthv1.DependencyHealthFinding{
		{SourceDomain: "graph", Severity: "INFO"},
		{SourceDomain: "governance", Severity: "WARNING"},
	}

	if got := statusFromFindings(findings, "graph"); got != "pass" {
		t.Fatalf("graph status = %q, want pass for info-only findings", got)
	}
	if got := statusFromFindings(findings, "governance"); got != "warn" {
		t.Fatalf("governance status = %q, want warn", got)
	}
}

func TestRuntimeDependencyHealthReportsRequiredResourceAndScenarioFailures(t *testing.T) {
	// [REQ:SDA-P0-001] [REQ:SDA-P0-014]
	tmp := t.TempDir()
	scenarioDir := filepath.Join(tmp, "demo")
	writeFile(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), `{
		"dependencies": {
			"resources": {
				"postgres": {"enabled": true, "required": true},
				"redis": {"enabled": false, "required": true}
			},
			"scenarios": {
				"code-facts": {"enabled": true, "required": true},
				"proto-health": {"enabled": true, "startup_policy": "must_start"},
				"security-health": {"enabled": false, "required": true}
			}
		}
	}`)
	fetcher := &fakeRuntimeStatusFetcher{
		resources: &cliv1.ResourceStatusesResponse{Resources: []*cliv1.ResourceStatus{
			{
				Resource: &cliv1.Resource{Name: "postgres"},
				Running:  false,
				Healthy:  structpb.NewBoolValue(false),
				Message:  "stopped",
			},
		}},
		scenarios: map[string]*cliv1.ScenarioStatusSingle{
			"code-facts": {
				Scenario: &cliv1.ScenarioStatusItem{
					Name:         "code-facts",
					Status:       "running",
					HealthStatus: structpb.NewBoolValue(true),
				},
			},
			"proto-health": {
				Scenario: &cliv1.ScenarioStatusItem{
					Name:   "proto-health",
					Status: "stopped",
				},
			},
		},
	}
	handler := &connectHandler{
		scenariosDir:  func() string { return tmp },
		statusFetcher: fetcher,
	}

	section, findings, degraded := handler.evaluateRuntime(context.Background(), "demo")

	if len(degraded) != 0 {
		t.Fatalf("degraded = %d, want 0", len(degraded))
	}
	if section.GetStatus() != "fail" {
		t.Fatalf("runtime status = %q, want fail", section.GetStatus())
	}
	if !containsFinding(findings, "runtime.resource-postgres-stopped") {
		t.Fatalf("missing stopped resource finding: %v", findingIDs(findings, ""))
	}
	if !containsFinding(findings, "runtime.scenario-proto-health-not-running") {
		t.Fatalf("missing stopped scenario finding: %v", findingIDs(findings, ""))
	}
	if containsFinding(findings, "runtime.resource-redis-stopped") {
		t.Fatalf("disabled resource should not be checked: %v", findingIDs(findings, ""))
	}
	if strings.Join(fetcher.scenarioCalls, ",") != "code-facts,proto-health" {
		t.Fatalf("scenario calls = %v, want code-facts/proto-health only", fetcher.scenarioCalls)
	}
}

func TestRuntimeDependencyHealthAllowsUnknownHealthWhenRunning(t *testing.T) {
	tmp := t.TempDir()
	scenarioDir := filepath.Join(tmp, "demo")
	writeFile(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), `{
		"dependencies": {
			"resources": {"qdrant": {"required": true}},
			"scenarios": {"code-facts": {"required": true}}
		}
	}`)
	handler := &connectHandler{
		scenariosDir: func() string { return tmp },
		statusFetcher: &fakeRuntimeStatusFetcher{
			resources: &cliv1.ResourceStatusesResponse{Resources: []*cliv1.ResourceStatus{
				{Resource: &cliv1.Resource{Name: "qdrant"}, Running: true},
			}},
			scenarios: map[string]*cliv1.ScenarioStatusSingle{
				"code-facts": {Scenario: &cliv1.ScenarioStatusItem{Name: "code-facts", Status: "running"}},
			},
		},
	}

	section, findings, degraded := handler.evaluateRuntime(context.Background(), "demo")

	if len(degraded) != 0 {
		t.Fatalf("degraded = %d, want 0", len(degraded))
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findingIDs(findings, ""))
	}
	if section.GetStatus() != "pass" {
		t.Fatalf("runtime status = %q, want pass", section.GetStatus())
	}
}

func TestApprovedDependencyGuidanceIsNotAllowlist(t *testing.T) {
	for _, phrase := range []string{
		"not an exhaustive allowlist",
		"suggest it with purpose",
		"security/license notes",
	} {
		if !strings.Contains(approvedDependencyGuidance, phrase) {
			t.Fatalf("approved dependency guidance missing %q: %s", phrase, approvedDependencyGuidance)
		}
	}
}

func TestReleaseAgeReportsMissingAndLowPNPMPolicy(t *testing.T) {
	// [REQ:SDA-P0-016]
	repoRoot := t.TempDir()
	scenariosDir := filepath.Join(repoRoot, "scenarios")
	scenarioDir := filepath.Join(scenariosDir, "demo")
	missingRoot := filepath.Join(scenarioDir, "ui")
	lowRoot := filepath.Join(scenarioDir, "worker")
	for _, dir := range []string{missingRoot, lowRoot} {
		if err := mkdirAll(dir); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies":{"react":"^19.0.0"}}`)
		writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")
	}
	writeFile(t, filepath.Join(lowRoot, "pnpm-workspace.yaml"), "packages:\n  - .\nminimumReleaseAge: 60\n")

	handler := &connectHandler{scenariosDir: func() string { return scenariosDir }}
	section, findings, summary := handler.evaluateReleaseAge(context.Background(), "demo", []*healthv1.DependencyHealthSurface{
		{Id: "ui", Language: "typescript", RootPath: missingRoot, PackageManager: "pnpm"},
		{Id: "worker", Language: "typescript", RootPath: lowRoot, PackageManager: "pnpm"},
	})

	if section.GetStatus() != "fail" {
		t.Fatalf("release-age status = %q, want fail", section.GetStatus())
	}
	if summary.GetReleaseAgeMinimumMinutes() != releaseAgeMinimumMinutes {
		t.Fatalf("minimum = %d, want %d", summary.GetReleaseAgeMinimumMinutes(), releaseAgeMinimumMinutes)
	}
	if !containsFinding(findings, "release-age.policy-ui-missing") {
		t.Fatalf("missing policy finding absent: %v", findingIDs(findings, ""))
	}
	if !containsFinding(findings, "release-age.policy-worker-minimum-too-low") {
		t.Fatalf("low policy finding absent: %v", findingIDs(findings, ""))
	}
}

func TestReleaseAgeAcceptsGovernedExcludes(t *testing.T) {
	// [REQ:SDA-P0-016]
	repoRoot := t.TempDir()
	scenariosDir := filepath.Join(repoRoot, "scenarios")
	scenarioDir := filepath.Join(scenariosDir, "demo")
	uiRoot := filepath.Join(scenarioDir, "ui")
	if err := mkdirAll(uiRoot); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(uiRoot, "package.json"), `{"dependencies":{"react":"^19.0.0"}}`)
	writeFile(t, filepath.Join(uiRoot, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")
	writeFile(t, filepath.Join(uiRoot, "pnpm-workspace.yaml"), "packages:\n  - .\nminimumReleaseAge: 10080\nminimumReleaseAgeExclude:\n  - react\n  - vite@7.0.0\n")
	writeFile(t, filepath.Join(repoRoot, ".vrooli", "dependencies", "approved-dependencies.json"), `{
		"release_age_exceptions": [
			{
				"package_name": "react",
				"state": "approved",
				"rationale": "Emergency compatibility fix.",
				"review_expires": "2026-07-01"
			}
		]
	}`)

	handler := &connectHandler{scenariosDir: func() string { return scenariosDir }}
	section, findings, summary := handler.evaluateReleaseAge(context.Background(), "demo", []*healthv1.DependencyHealthSurface{
		{Id: "ui", Language: "typescript", RootPath: uiRoot, PackageManager: "pnpm"},
	})

	if section.GetStatus() != "fail" {
		t.Fatalf("release-age status = %q, want fail for ungoverned vite exclude", section.GetStatus())
	}
	if summary.GetReleaseAgeExceptions() != 2 {
		t.Fatalf("exceptions = %d, want 2", summary.GetReleaseAgeExceptions())
	}
	if containsFinding(findings, "release-age.policy-ui-exclude-react") {
		t.Fatalf("governed react exclude should not produce finding: %v", findingIDs(findings, ""))
	}
	if !containsFinding(findings, "release-age.policy-ui-exclude-vite-7-0-0") {
		t.Fatalf("ungoverned vite exclude finding absent: %v", findingIDs(findings, ""))
	}
}

func TestSecurityHealthStatusReportsIndexReadiness(t *testing.T) {
	handler := &connectHandler{
		scenariosDir: func() string { return filepath.Join(t.TempDir(), "scenarios") },
		commandLookup: func(name string) (string, error) {
			if name == "security-health" {
				return "/usr/bin/security-health", nil
			}
			return "", errors.New("unexpected command")
		},
		commandRunner: staticCommandRunner{output: `{"available":true,"indexed_count":42,"vulnerable_count":3,"index_ready":true,"last_reconcile_at":"2026-06-16T00:00:00Z"}`},
	}

	section, findings, degraded := handler.evaluateSecurityHealth(context.Background(), "demo")

	if len(degraded) != 0 {
		t.Fatalf("degraded = %d, want 0", len(degraded))
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %d, want 0", len(findings))
	}
	if section.GetStatus() != "pass" {
		t.Fatalf("security status = %q, want pass", section.GetStatus())
	}
	if !strings.Contains(section.GetSummary(), "indexed=42") {
		t.Fatalf("summary missing indexed count: %s", section.GetSummary())
	}
}

func TestSecurityHealthStatusDegradesWhenUnavailable(t *testing.T) {
	handler := &connectHandler{
		commandLookup: func(string) (string, error) {
			return "", errors.New("not found")
		},
	}

	section, _, degraded := handler.evaluateSecurityHealth(context.Background(), "demo")

	if section.GetStatus() != "degraded" {
		t.Fatalf("security status = %q, want degraded", section.GetStatus())
	}
	if len(degraded) != 1 || degraded[0].GetDomain() != "security-index" {
		t.Fatalf("degraded = %#v, want one security-index degraded dependency", degraded)
	}
}

func TestSecurityHealthVulnerabilityEvidenceStaysOutOfHealthFindings(t *testing.T) {
	// [REQ:SDA-P0-017]
	handler := &connectHandler{
		scenariosDir: func() string { return filepath.Join(t.TempDir(), "scenarios") },
		commandLookup: func(name string) (string, error) {
			if name == "security-health" {
				return "/usr/bin/security-health", nil
			}
			return "", errors.New("unexpected command")
		},
		commandRunner: routedCommandRunner{outputs: map[string]string{
			"security-health deps status --json": `{"available":true,"indexed_count":42,"vulnerable_count":1,"index_ready":true}`,
		}},
	}

	section, findings, degraded := handler.evaluateSecurityHealth(context.Background(), "demo")

	if len(degraded) != 0 {
		t.Fatalf("degraded = %d, want 0", len(degraded))
	}
	if section.GetId() != "security-index" {
		t.Fatalf("security section id = %q, want security-index", section.GetId())
	}
	if section.GetStatus() != "pass" {
		t.Fatalf("security status = %q, want pass", section.GetStatus())
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %d, want 0", len(findings))
	}
	if !strings.Contains(section.GetSummary(), "vulnerable=1") {
		t.Fatalf("summary should keep index count context: %s", section.GetSummary())
	}
}

func TestReadinessRoutesByDiscoveredSurfaceLanguage(t *testing.T) {
	// [REQ:SDA-P0-013]
	tmp := t.TempDir()
	apiRoot := filepath.Join(tmp, "api")
	cliRoot := filepath.Join(tmp, "custom-cli")
	workerRoot := filepath.Join(tmp, "worker")
	for _, dir := range []string{apiRoot, cliRoot, workerRoot, filepath.Join(workerRoot, "node_modules")} {
		if err := mkdirAll(dir); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(apiRoot, "go.mod"), "module example.com/api\n")
	writeFile(t, filepath.Join(cliRoot, "go.mod"), "module example.com/cli\n")
	writeFile(t, filepath.Join(workerRoot, "package.json"), `{"dependencies":{"vite":"^7.0.0"}}`)
	writeFile(t, filepath.Join(workerRoot, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")

	runner := &fakeCommandRunner{}
	handler := &connectHandler{
		scenariosDir: func() string { return filepath.Dir(tmp) },
		surfaceDiscoverer: fakeSurfaceDiscoverer{inventory: surfaceInventory{Surfaces: []*healthv1.DependencyHealthSurface{
			{Id: "api", Language: "go", RootPath: apiRoot},
			{Id: "cli-tool", Language: "go", RootPath: cliRoot},
			{Id: "worker-events", Language: "typescript", RootPath: workerRoot, PackageManager: "pnpm"},
		}}},
		commandLookup: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		commandRunner: runner,
	}

	surfaces, _, readinessSection, findings, commands, degraded := handler.evaluateReadiness(context.Background(), "fixture", true)

	if len(degraded) != 0 {
		t.Fatalf("degraded = %d, want 0", len(degraded))
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findingIDs(findings, ""))
	}
	if readinessSection.GetStatus() != "pass" {
		t.Fatalf("readiness status = %q, want pass", readinessSection.GetStatus())
	}
	if len(surfaces) != 3 {
		t.Fatalf("surfaces = %d, want 3", len(surfaces))
	}
	gotCommands := make([]string, 0, len(commands))
	for _, command := range commands {
		gotCommands = append(gotCommands, command.GetCommand())
	}
	sort.Strings(gotCommands)
	wantCommands := []string{"bash", "curl", "go", "jq", "node", "pnpm"}
	if strings.Join(gotCommands, ",") != strings.Join(wantCommands, ",") {
		t.Fatalf("commands = %v, want %v", gotCommands, wantCommands)
	}
	var tidyCalls int
	var workerTouched bool
	for _, call := range runner.calls {
		if strings.Contains(call, "/go/mod/tidy/-diff") {
			tidyCalls++
		}
		if strings.Contains(call, "/worker/") {
			workerTouched = true
		}
	}
	if tidyCalls != 2 {
		t.Fatalf("go tidy calls = %d, want 2; calls=%v", tidyCalls, runner.calls)
	}
	if workerTouched {
		t.Fatalf("worker surface should use package-file checks, not command runner calls: %v", runner.calls)
	}
}

func TestReadinessReportsUnsupportedSurfaceAndMissingCommand(t *testing.T) {
	handler := &connectHandler{
		commandLookup: func(name string) (string, error) {
			if name == "jq" {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + name, nil
		},
		commandRunner: &fakeCommandRunner{},
	}
	findings, commands := handler.checkReadiness(context.Background(), "fixture", []*healthv1.DependencyHealthSurface{
		{Id: "data", Language: "ruby", RootPath: filepath.Join(t.TempDir(), "data")},
	})
	if statusFromFindings(findings, "readiness") != "fail" {
		t.Fatalf("status = %s, want fail", statusFromFindings(findings, "readiness"))
	}
	if !containsFinding(findings, "readiness.unsupported-surface-data") {
		t.Fatalf("unsupported language finding missing: %v", findingIDs(findings, ""))
	}
	if !containsFinding(findings, "readiness.command.jq.missing") {
		t.Fatalf("missing jq finding missing: %v", findingIDs(findings, ""))
	}
	if len(commands) != 3 {
		t.Fatalf("commands = %d, want baseline 3", len(commands))
	}
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	// Construct a minimal handler: stub surface discoverer + command seams so
	// ValidateDependencyHealth completes without calling real external binaries;
	// real maturity spec so buildMaturityAssessment does not error.
	tmp := t.TempDir()
	scenarioDir := filepath.Join(tmp, "test-scenario")
	writeFile(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), `{"dependencies":{}}`)

	h := &connectHandler{
		scenariosDir:      func() string { return tmp },
		spec:              testMaturitySpec(t),
		surfaceDiscoverer: fakeSurfaceDiscoverer{inventory: surfaceInventory{}},
		commandLookup: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		commandRunner: &fakeCommandRunner{},
		statusFetcher: &fakeRuntimeStatusFetcher{
			resources: nil,
		},
	}

	nativeReq := connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "test-scenario"})
	resp, err := h.ValidateScenario(context.Background(), nativeReq)
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	m := resp.Msg.GetMetrics()
	if m == nil {
		t.Fatal("metrics must be attached to the response")
	}
	if m.GetWallClockMs() < 0 {
		t.Fatalf("wall clock must be non-negative, got %d", m.GetWallClockMs())
	}
	env := m.GetEnvironment()
	if env == nil {
		t.Fatal("metrics environment must be populated with the stdlib baseline")
	}
	if env.GetOs() != runtime.GOOS {
		t.Fatalf("env os = %q, want %q", env.GetOs(), runtime.GOOS)
	}
	if env.GetArch() != runtime.GOARCH {
		t.Fatalf("env arch = %q, want %q", env.GetArch(), runtime.GOARCH)
	}
	if env.GetNumCpu() != int32(runtime.NumCPU()) {
		t.Fatalf("env num_cpu = %d, want %d", env.GetNumCpu(), runtime.NumCPU())
	}
}

func containsFinding(findings []*healthv1.DependencyHealthFinding, id string) bool {
	for _, finding := range findings {
		if finding.GetId() == id {
			return true
		}
	}
	return false
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testMaturitySpec(t *testing.T) *assessment.Spec {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatalf("read maturity spec: %v", err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatalf("parse maturity spec: %v", err)
	}
	return spec
}
