package lifecycle

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
)

// deployReportScene is a repo whose alpha api builds from its own sources and
// a shared package through a local replace, with a build stamped by the
// lifecycle. The running process is reported as executing binPath.
func deployReportScene(t *testing.T) (r *Runner, item scenario.Scenario, binPath string) {
	t.Helper()
	repoRoot, appPath, binPath, srcPath := freshnessTestScene(t)
	apiDir := filepath.Dir(srcPath)
	writeSceneFile(t, filepath.Join(apiDir, "go.mod"), "module alpha\n\ngo 1.25.0\n\nrequire example.com/shared v0.0.0\n\nreplace example.com/shared => ../../../packages/shared\n")
	writeSceneFile(t, srcPath, "package main\n\nimport (\n\t\"os\"\n\n\t\"example.com/shared\"\n)\n\nfunc main() { _ = os.Getenv(\"VROOLI_EXISTING\"); shared.Run() }\n")
	writeSceneFile(t, filepath.Join(repoRoot, "packages", "shared", "go.mod"), "module example.com/shared\n\ngo 1.25.0\n")
	writeSceneFile(t, filepath.Join(repoRoot, "packages", "shared", "shared.go"), "package shared\n\nfunc Run() {}\n")

	r = &Runner{Root: repoRoot}
	item = freshnessTestItem(appPath)
	artifacts, err := componentFreshnessArtifacts(appPath, repoRoot, item.Manifest.Components["api"], defaultHostProbeDeps())
	if err != nil || len(artifacts) != 1 {
		t.Fatalf("freshness artifacts: %v (%d)", err, len(artifacts))
	}
	if err := r.stampArtifactFreshness(artifacts[0]); err != nil {
		t.Fatalf("stamp: %v", err)
	}
	setDeployRunningExecutables(t, []string{binPath}, true)
	return r, item, binPath
}

func writeSceneFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setDeployRunningExecutables(t *testing.T, executables []string, ok bool) {
	t.Helper()
	previous := deployRunningExecutables
	deployRunningExecutables = func() ([]string, bool) { return executables, ok }
	t.Cleanup(func() { deployRunningExecutables = previous })
}

func onlyArtifact(t *testing.T, report DeployReport) DeployReportArtifact {
	t.Helper()
	if len(report.Artifacts) != 1 {
		t.Fatalf("artifacts = %+v, want one", report.Artifacts)
	}
	return report.Artifacts[0]
}

type deployReportEngagementResolver struct {
	info             EngagementInfo
	engaged          bool
	location         ServingSourceLocation
	resolveErr       error
	lookupCalls      int
	locatorCalls     int
	materializeCalls int
}

func (r *deployReportEngagementResolver) Engagement(string) (EngagementInfo, bool, error) {
	r.lookupCalls++
	return r.info, r.engaged, r.resolveErr
}

func (r *deployReportEngagementResolver) ServingSourceLocation(string, EngagementInfo, string) (ServingSourceLocation, error) {
	r.locatorCalls++
	return r.location, r.resolveErr
}

func (r *deployReportEngagementResolver) MaterializeServingSource(string, EngagementInfo, string) (string, error) {
	r.materializeCalls++
	return "unexpected-materialization", nil
}

func writeDeployReportManifest(t *testing.T, item scenario.Scenario) {
	t.Helper()
	data, err := json.Marshal(item.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeSceneFile(t, filepath.Join(item.Path, ".vrooli", "service.json"), string(data))
}

func TestDeployReportByNameUsesServingBaselineAndCandidateInputsWithoutMaterializing(t *testing.T) {
	r, item, candidateBin := deployReportScene(t)
	writeDeployReportManifest(t, item)

	servingRepo := t.TempDir()
	servingApp := filepath.Join(servingRepo, "scenarios", item.Slug)
	for _, name := range []string{"api/main.go", "api/go.mod"} {
		data, err := os.ReadFile(filepath.Join(item.Path, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		writeSceneFile(t, filepath.Join(servingApp, filepath.FromSlash(name)), string(data))
	}
	for _, name := range []string{"go.mod", "shared.go"} {
		data, err := os.ReadFile(filepath.Join(r.Root, "packages", "shared", name))
		if err != nil {
			t.Fatal(err)
		}
		writeSceneFile(t, filepath.Join(servingRepo, "packages", "shared", name), string(data))
	}
	servingBin := filepath.Join(servingApp, "api", "alpha-api")
	writeSceneFile(t, servingBin, "serving executable")
	if err := os.Chmod(servingBin, 0o755); err != nil {
		t.Fatal(err)
	}
	component := item.Manifest.Components["api"]
	servingArtifacts, err := componentFreshnessArtifactsContextWithName(context.Background(), servingApp, r.Root, item.Slug, "api", component, defaultHostProbeDeps())
	if err != nil || len(servingArtifacts) != 1 {
		t.Fatalf("serving freshness artifacts: %v (%d)", err, len(servingArtifacts))
	}
	if err := r.stampArtifactFreshness(servingArtifacts[0]); err != nil {
		t.Fatalf("stamp serving baseline: %v", err)
	}
	servingManifest, ok, err := cliutil.ReadFreshnessManifest(servingArtifacts[0].ManifestPath)
	if err != nil || !ok {
		t.Fatalf("read serving freshness manifest: ok=%v err=%v", ok, err)
	}

	if err := os.WriteFile(candidateBin, []byte("candidate executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSceneFile(t, filepath.Join(item.Path, "api", "main.go"), "package main\n\nconst gate = \"VROOLI_NEW_GATE_ENFORCE\"\n\nfunc main(){}\n")
	resolver := &deployReportEngagementResolver{
		engaged: true,
		info: EngagementInfo{
			Slug: "goal", CaptureID: "capture-17", Generation: "capture-17", ProtoArtifactID: "proto-4",
		},
		location: ServingSourceLocation{RepoRoot: servingRepo, ScenarioDir: servingApp},
	}
	r.Engagements = resolver
	setDeployRunningExecutables(t, []string{servingBin}, true)

	report, err := r.DeployReportByName(context.Background(), item.Slug, "")
	if err != nil {
		t.Fatal(err)
	}
	api := onlyArtifact(t, report)
	if report.SourceRole != "frozen_serving_tree" || report.CandidateSource != "scenarios/alpha" || report.ServingSource != servingApp || report.Engagement != "goal" || report.CaptureID != "capture-17" || report.Generation != "capture-17" || report.ProtoArtifactID != "proto-4" {
		t.Fatalf("source provenance = %+v", report)
	}
	if api.BuiltAt == nil || api.BuiltAt.UnixNano() != servingManifest.WrittenAtNS {
		t.Fatalf("built_at = %v, want serving manifest timestamp %d", api.BuiltAt, servingManifest.WrittenAtNS)
	}
	if api.Running != DeployRunningCurrent {
		t.Fatalf("running = %q, want current serving artifact", api.Running)
	}
	if !report.Changed || !reflect.DeepEqual(api.Files, []DeployReportFile{{Path: "scenarios/alpha/api/main.go", Change: "modified"}}) {
		t.Fatalf("candidate delta = %+v, changed=%v", api.Files, report.Changed)
	}
	if !api.SwitchesKnown || !reflect.DeepEqual(api.NewSwitches, []DeployReportSwitch{{Name: "VROOLI_NEW_GATE_ENFORCE", Files: []string{"scenarios/alpha/api/main.go"}}}) {
		t.Fatalf("candidate switch changes = known:%v switches:%+v", api.SwitchesKnown, api.NewSwitches)
	}
	if resolver.lookupCalls != 1 || resolver.materializeCalls != 0 {
		t.Fatalf("engagement reads=%d materializations=%d, want one read and no materialization", resolver.lookupCalls, resolver.materializeCalls)
	}
}

func TestDeployReportByNameFailsClosedWhenEngagementCannotBeResolved(t *testing.T) {
	r, item, _ := deployReportScene(t)
	writeDeployReportManifest(t, item)
	resolver := &deployReportEngagementResolver{resolveErr: os.ErrPermission}
	r.Engagements = resolver
	_, err := r.DeployReportByName(context.Background(), item.Slug, "")
	if err == nil || !strings.Contains(err.Error(), "resolve serving source") {
		t.Fatalf("deploy report error = %v, want explicit engagement resolution failure", err)
	}
	if resolver.materializeCalls != 0 {
		t.Fatalf("read-only report materialized serving tree %d times", resolver.materializeCalls)
	}
}

func TestDeployReportByNameResolvesExplicitShadowVariant(t *testing.T) {
	r, item, _ := deployReportScene(t)
	writeDeployReportManifest(t, item)
	resolver := &deployReportEngagementResolver{engaged: true, info: EngagementInfo{Slug: "goal"}}
	r.Engagements = resolver

	report, err := r.DeployReportByName(context.Background(), item.Slug+"@shadow", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Scenario != item.Slug+"@shadow" || report.SourceRole != "working_tree" || report.ServingSource != item.Path {
		t.Fatalf("shadow report identity/source = %+v", report)
	}
	if resolver.lookupCalls != 1 || resolver.locatorCalls != 0 || resolver.materializeCalls != 0 {
		t.Fatalf("shadow resolver calls: engagement=%d locator=%d materialize=%d", resolver.lookupCalls, resolver.locatorCalls, resolver.materializeCalls)
	}
}

func TestDeployReportFreshTreeReportsNoChanges(t *testing.T) {
	r, item, _ := deployReportScene(t)
	report, err := r.DeployReport(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	art := onlyArtifact(t, report)
	if report.Changed || len(art.Files) != 0 || len(art.NewSwitches) != 0 {
		t.Fatalf("unchanged tree reported changes: %+v", report)
	}
	if art.Baseline != "recorded" || art.Running != DeployRunningCurrent || !art.SwitchesKnown {
		t.Fatalf("baseline/running/switches = %s/%s/%v", art.Baseline, art.Running, art.SwitchesKnown)
	}
}

// The O28/O29 shape: another session leaves a rollout-held gate in a shared
// package and in the scenario. The report names the files, marks the shared
// directory, and names the new switch, before any restart.
func TestDeployReportListsChangedFilesAndNewSwitches(t *testing.T) {
	r, item, _ := deployReportScene(t)
	writeSceneFile(t, filepath.Join(r.Root, "packages", "shared", "shared.go"), "package shared\n\nconst gateEnv = \"VROOLI_SHARED_GATE_ENFORCE\"\n\nfunc Run() {}\n")
	writeSceneFile(t, filepath.Join(r.Root, "scenarios", "alpha", "api", "gate.go"), "package main\n\nconst effortGate = \"VROOLI_EFFORT_ENFORCE\"\n")

	report, err := r.DeployReport(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	art := onlyArtifact(t, report)
	if !report.Changed {
		t.Fatal("report.Changed = false for an edited tree")
	}
	wantFiles := []DeployReportFile{
		{Path: "packages/shared/shared.go", Change: "modified"},
		{Path: "scenarios/alpha/api/gate.go", Change: "added"},
	}
	if !reflect.DeepEqual(art.Files, wantFiles) {
		t.Fatalf("files = %+v, want %+v", art.Files, wantFiles)
	}
	wantPackages := []DeployReportPackage{
		{Dir: "scenarios/alpha/api", Added: 1},
		{Dir: "packages/shared", Shared: true, Modified: 1},
	}
	if !reflect.DeepEqual(art.Packages, wantPackages) {
		t.Fatalf("packages = %+v, want %+v", art.Packages, wantPackages)
	}
	wantSwitches := []DeployReportSwitch{
		{Name: "VROOLI_EFFORT_ENFORCE", Files: []string{"scenarios/alpha/api/gate.go"}},
		{Name: "VROOLI_SHARED_GATE_ENFORCE", Files: []string{"packages/shared/shared.go"}},
	}
	if !art.SwitchesKnown || !reflect.DeepEqual(art.NewSwitches, wantSwitches) {
		t.Fatalf("switches known=%v new=%+v, want %+v", art.SwitchesKnown, art.NewSwitches, wantSwitches)
	}
}

func TestDeployReportBeforeSwitchRecordingNamesCandidatesInChangedFiles(t *testing.T) {
	r, item, binPath := deployReportScene(t)
	manifestPath := cliutil.FreshnessManifestPath(binPath)
	manifest, ok, err := cliutil.ReadFreshnessManifest(manifestPath)
	if err != nil || !ok {
		t.Fatalf("read manifest: %v %v", ok, err)
	}
	manifest.EnvSwitches, manifest.EnvSwitchesRecorded = nil, false
	if err := cliutil.WriteFreshnessManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	writeSceneFile(t, filepath.Join(r.Root, "packages", "shared", "shared.go"), "package shared\n\nconst gateEnv = \"VROOLI_SHARED_GATE_ENFORCE\"\n\nfunc Run() {}\n")

	art := onlyArtifact(t, mustDeployReport(t, r, item))
	want := []DeployReportSwitch{{Name: "VROOLI_SHARED_GATE_ENFORCE", Files: []string{"packages/shared/shared.go"}}}
	if art.SwitchesKnown || len(art.NewSwitches) != 0 || !reflect.DeepEqual(art.SwitchesInChangedFiles, want) {
		t.Fatalf("known=%v new=%+v candidates=%+v, want candidates %+v (VROOLI_EXISTING is in an unchanged file)", art.SwitchesKnown, art.NewSwitches, art.SwitchesInChangedFiles, want)
	}
}

func TestDeployReportFlagsAProcessRunningAReplacedBinary(t *testing.T) {
	r, item, binPath := deployReportScene(t)
	setDeployRunningExecutables(t, []string{"/usr/bin/other", binPath + " (deleted)"}, true)
	report := mustDeployReport(t, r, item)
	if art := onlyArtifact(t, report); art.Running != DeployRunningReplaced || !report.Changed {
		t.Fatalf("running=%s changed=%v, want replaced and changed", art.Running, report.Changed)
	}
}

func TestDeployReportWithoutManifestIsUnknownNotClean(t *testing.T) {
	r, item, binPath := deployReportScene(t)
	if err := os.Remove(cliutil.FreshnessManifestPath(binPath)); err != nil {
		t.Fatal(err)
	}
	report := mustDeployReport(t, r, item)
	if art := onlyArtifact(t, report); art.Baseline != "missing" || !report.Changed {
		t.Fatalf("baseline=%s changed=%v, want missing and changed", art.Baseline, report.Changed)
	}
}

func mustDeployReport(t *testing.T, r *Runner, item scenario.Scenario) DeployReport {
	t.Helper()
	report, err := r.DeployReport(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	return report
}
