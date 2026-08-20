package deployments

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deployment-manager/bundles"
	"deployment-manager/profiles"
)

type orchestratorProfileRepo struct {
	profile *profiles.Profile
	err     error
	swaps   []profiles.Swap
}

type orchestratorApprovalsRepo struct {
	gate *ReleaseGateStatus
	err  error
}

func (f *orchestratorApprovalsRepo) Create(context.Context, *DeploymentApproval) error { return nil }
func (f *orchestratorApprovalsRepo) Get(context.Context, string) (*DeploymentApproval, error) {
	return nil, nil
}

func (f *orchestratorApprovalsRepo) ListByCommit(context.Context, string, string) ([]*DeploymentApproval, error) {
	return nil, nil
}

func (f *orchestratorApprovalsRepo) ListByProfile(context.Context, string, int) ([]*DeploymentApproval, error) {
	return nil, nil
}

func (f *orchestratorApprovalsRepo) UpdateDecision(context.Context, string, string, string, string) error {
	return nil
}

func (f *orchestratorApprovalsRepo) MarkStale(context.Context, string, string, string) error {
	return nil
}

func (f *orchestratorApprovalsRepo) GetRequiredPlatforms(context.Context, string) ([]string, error) {
	return nil, nil
}

func (f *orchestratorApprovalsRepo) SetRequiredPlatforms(context.Context, string, []string) error {
	return nil
}

func (f *orchestratorApprovalsRepo) CheckReleaseGate(context.Context, string, string) (*ReleaseGateStatus, error) {
	return f.gate, f.err
}

func (f *orchestratorApprovalsRepo) GetRequiredTargets(context.Context, string) ([]RequiredTarget, error) {
	return nil, nil
}

func (f *orchestratorApprovalsRepo) SetRequiredTargets(context.Context, string, []RequiredTarget) error {
	return nil
}

func (f *orchestratorProfileRepo) List(context.Context) ([]profiles.Profile, error) { return nil, nil }
func (f *orchestratorProfileRepo) Get(context.Context, string) (*profiles.Profile, error) {
	return f.profile, f.err
}

func (f *orchestratorProfileRepo) Create(context.Context, *profiles.Profile) (string, error) {
	return "", nil
}

func (f *orchestratorProfileRepo) Update(context.Context, string, map[string]interface{}) (*profiles.Profile, error) {
	return nil, nil
}
func (f *orchestratorProfileRepo) Delete(context.Context, string) (bool, error) { return false, nil }
func (f *orchestratorProfileRepo) GetVersions(context.Context, string) ([]profiles.Version, error) {
	return nil, nil
}

func (f *orchestratorProfileRepo) GetScenarioAndTier(context.Context, string) (string, int, error) {
	return "", 0, nil
}
func (f *orchestratorProfileRepo) AddSwap(context.Context, string, profiles.Swap) error { return nil }
func (f *orchestratorProfileRepo) GetSwaps(context.Context, string) ([]profiles.Swap, error) {
	return f.swaps, nil
}

func newCoreOrchestrator(repo profiles.Repository) *Orchestrator {
	return &Orchestrator{profileRepo: repo, vrooli: testrunRepoRoot(), log: func(string, map[string]interface{}) {}}
}

func testrunRepoRoot() string {
	root, _ := os.Getwd()
	return filepath.Clean(filepath.Join(root, "..", "..", ".."))
}

func TestDeployDesktopRejectsMalformedAndIncompleteRequests(t *testing.T) {
	o := newCoreOrchestrator(nil)
	for name, body := range map[string]string{
		"invalid JSON":    "{",
		"missing profile": `{"git_commit_hash":"abc"}`,
		"missing commit":  `{"profile_id":"profile-demo"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			o.DeployDesktop(rec, httptest.NewRequest(http.MethodPost, "/api/v1/deploy-desktop", strings.NewReader(body)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want bad request; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestDeployLoadProfileReportsRepositoryAndGateOutcomes(t *testing.T) {
	profile := &profiles.Profile{ID: "p1", Scenario: "demo"}
	for name, repo := range map[string]*orchestratorProfileRepo{
		"repository error": {err: errors.New("database down")},
		"missing profile":  {},
		"success":          {profile: profile},
	} {
		t.Run(name, func(t *testing.T) {
			o := newCoreOrchestrator(repo)
			ds := newDeployState("p1", "", "", "", nil)
			status := o.deployLoadProfile(ds)
			if name == "repository error" && status != http.StatusBadGateway {
				t.Fatalf("status = %d", status)
			}
			if name == "missing profile" && status != http.StatusNotFound {
				t.Fatalf("status = %d", status)
			}
			if name == "success" && (status != 0 || ds.profile != profile || ds.response.Scenario != "demo") {
				t.Fatalf("success status=%d state=%+v", status, ds)
			}
		})
	}
}

func TestDeployLoadProfileReportsReleaseGateOutcomes(t *testing.T) {
	profile := &profiles.Profile{ID: "p1", Scenario: "demo"}
	for name, repo := range map[string]*orchestratorApprovalsRepo{
		"error":   {err: errors.New("gate unavailable")},
		"blocked": {gate: &ReleaseGateStatus{Ready: false, Platforms: []PlatformGateStatus{{Platform: "linux", Status: ApprovalStatusPending}}}},
		"ready":   {gate: &ReleaseGateStatus{Ready: true, Platforms: []PlatformGateStatus{{Platform: "linux", Status: ApprovalStatusApproved}}}},
	} {
		t.Run(name, func(t *testing.T) {
			o := newCoreOrchestrator(&orchestratorProfileRepo{profile: profile})
			o.approvalsRepo = repo
			ds := newDeployState("p1", "", "", "commit", nil)
			status := o.deployLoadProfile(ds)
			switch name {
			case "error":
				if status != http.StatusInternalServerError {
					t.Fatalf("status=%d", status)
				}
			case "blocked":
				if status != http.StatusPreconditionFailed || ds.response.Status != "blocked" {
					t.Fatalf("blocked status=%d response=%+v", status, ds.response)
				}
			case "ready":
				if status != 0 || ds.response.Status != "" {
					t.Fatalf("ready status=%d response=%+v", status, ds.response)
				}
			}
		})
	}
}

func TestDeployValidateAndBuildSkipPaths(t *testing.T) {
	o := newCoreOrchestrator(&orchestratorProfileRepo{profile: &profiles.Profile{ID: "p1", Scenario: "demo"}})
	ds := newDeployState("p1", "", "", "", nil)
	ds.profile = &profiles.Profile{ID: "p1", Scenario: "demo"}
	ds.req.SkipValidation = true
	ds.req.SigningConfig = map[string]interface{}{"platform": "linux"}
	ds.req.DryRun = true
	if status := o.deployValidateAndSign(ds); status != 0 || len(ds.response.Steps) != 3 || ds.response.Steps[0].Status != "skipped" || ds.response.Steps[1].Status == "failed" {
		t.Fatalf("validate skip state = %d %+v", status, ds.response.Steps)
	}
	ds.manifest = &bundles.Manifest{Services: []bundles.ServiceEntry{{ID: "api"}}}
	ds.req.SkipBuild = true
	if status := o.deployBuildBinaries(ds); status != 0 || ds.response.Steps[len(ds.response.Steps)-1].Status != "skipped" {
		t.Fatalf("build skip state = %d %+v", status, ds.response.Steps)
	}
	ds.req.SkipBuild = false
	ds.req.DryRun = true
	ds.manifest.Services = []bundles.ServiceEntry{{ID: "api", Build: &bundles.BuildConfig{Type: "go", SourceDir: "api"}}}
	if status := o.deployBuildBinaries(ds); status != 0 || ds.response.Steps[len(ds.response.Steps)-1].Status != "skipped" {
		t.Fatalf("build dry-run state = %d %+v", status, ds.response.Steps)
	}
}

func TestDeployPackageAndRuntimeDecisionPaths(t *testing.T) {
	o := newCoreOrchestrator(nil)
	ds := newDeployState("p1", "", "", "", nil)
	ds.req.SkipPackaging = true
	ds.req.SkipInstallers = true
	o.deployPackageAndInstall(ds)
	if len(ds.response.Steps) != 2 || ds.response.Steps[0].Status != "skipped" || ds.response.Steps[1].Status != "skipped" {
		t.Fatalf("skip package state = %+v", ds.response.Steps)
	}
	ds = newDeployState("p1", "", "", "", nil)
	ds.deploymentMode = "bundled"
	ds.response.DesktopPath = t.TempDir()
	o.deployValidateRuntime(ds)
	if len(ds.response.Steps) != 1 || ds.response.Steps[0].Status != "failed" {
		t.Fatalf("missing runtime state = %+v", ds.response.Steps)
	}
	runtime := filepath.Join(ds.response.DesktopPath, "bundle", "runtime")
	if err := os.MkdirAll(runtime, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtime, "supervisor"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	ds.response.Steps = nil
	o.deployValidateRuntime(ds)
	if ds.response.Steps[0].Status != "success" {
		t.Fatalf("runtime success state = %+v", ds.response.Steps)
	}
	ds.req.DryRun = true
	ds.req.VisualValidation = true
	o.deployVisualValidation(ds)
	if ds.response.Steps[len(ds.response.Steps)-1].Status != "skipped" {
		t.Fatalf("visual dry-run state = %+v", ds.response.Steps)
	}
	ds.req.DryRun = false
	ds.req.SkipPackaging = true
	o.deployBuildInstallers(ds)
	if ds.response.Steps[len(ds.response.Steps)-1].Status != "skipped" {
		t.Fatalf("installer skip state = %+v", ds.response.Steps)
	}
	ds = newDeployState("p1", "", "", "", nil)
	ds.profile = &profiles.Profile{Scenario: "demo"}
	ds.req.DryRun = true
	ds.response.DesktopPath = t.TempDir()
	ds.deploymentMode = "bundled"
	o.deployGenerateWrapper(ds)
	o.deployBuildInstallers(ds)
	ds.req.VisualValidation = true
	o.deployVisualValidation(ds)
	if len(ds.response.Steps) != 3 || ds.response.Steps[0].Status != "skipped" || ds.response.Steps[1].Status != "skipped" || ds.response.Steps[2].Status != "skipped" {
		t.Fatalf("dry-run package state = %+v", ds.response.Steps)
	}
}

func TestOrchestrationStepAndFinalizeHelpers(t *testing.T) {
	o := newCoreOrchestrator(nil)
	step := o.startStep("test")
	if step.Status != "running" {
		t.Fatal(step)
	}
	o.successStep(&step, "done")
	if step.Status != "success" || step.Message != "done" {
		t.Fatal(step)
	}
	o.failStep(&step, "bad")
	if step.Status != "failed" || step.Error != "bad" {
		t.Fatal(step)
	}
	ds := newDeployState("p1", "", "", "", nil)
	ds.profile = &profiles.Profile{Scenario: "demo"}
	o.deployFinalizeAndPublish(ds)
	if ds.response.Status != "success" || len(ds.response.NextSteps) == 0 {
		t.Fatalf("finalized response = %+v", ds.response)
	}
	ds.response.Steps = append(ds.response.Steps, OrchestrationStep{Status: "failed"})
	o.deployFinalizeAndPublish(ds)
	if ds.response.Status != "failed" {
		t.Fatalf("failed response status = %q", ds.response.Status)
	}
}
