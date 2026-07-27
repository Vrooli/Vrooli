package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/shared/errors"
	"strings"
	"testing"
)

func TestBuildStageReturnsStructuredFailuresBeforeStartingExternalBuild(t *testing.T) {
	timeProvider := &mockTimeProvider{now: 100}
	missingService := NewBuildStage(WithBuildTimeProvider(timeProvider))
	result := missingService.Execute(context.Background(), &StageInput{Config: &Config{ScenarioName: "demo"}})
	if result.Status != StatusFailed || result.ErrorInfo == nil || result.ErrorInfo.Code != string(errors.CodeServiceStartError) {
		t.Fatalf("missing service result = %#v", result)
	}
	missingPath := NewBuildStage(WithBuildService(&buildServiceSpy{}), WithBuildTimeProvider(timeProvider))
	result = missingPath.Execute(context.Background(), &StageInput{Config: &Config{ScenarioName: "demo"}})
	if result.Status != StatusFailed || result.ErrorInfo == nil || result.ErrorInfo.Code != string(errors.CodeDependencyError) {
		t.Fatalf("missing desktop path result = %#v", result)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	result = missingService.Execute(cancelled, &StageInput{Config: &Config{ScenarioName: "demo"}})
	if result.Status != StatusCancelled {
		t.Fatalf("cancelled result = %#v", result)
	}
}

func TestBuildStageInitializesProvenanceAndMapsTerminalResults(t *testing.T) {
	store := build.NewStore()
	stage := NewBuildStage(WithBuildStore(store), WithBuildTimeProvider(&mockTimeProvider{now: 123}))
	stage.initBuildStore("build-1", "demo", []string{"linux", "win"}, &BuildProvenance{GitCommitHash: "abc", Version: "1.0"})
	status, ok := store.Get("build-1")
	if !ok || status.Status != "building" || status.Metadata["git_commit_hash"] != "abc" || status.PlatformResults["linux"].Status != "pending" {
		t.Fatalf("initialized status = %#v", status)
	}
	for _, wait := range []*WaitError{
		{Kind: WaitErrorStore}, {Kind: WaitErrorTimeout}, {Kind: WaitErrorCancelled}, {Kind: WaitErrorOther},
	} {
		if mapped := stage.buildWaitError(wait, "build-1", []string{"linux"}); mapped == nil || mapped.Code == "" {
			t.Fatalf("buildWaitError(%v) = %#v", wait.Kind, mapped)
		}
	}
	result := &StageResult{}
	if stage.handleBuildResult(result, &build.Status{Status: BuildStatusReady}) || len(result.Logs) == 0 {
		t.Fatalf("ready build result = %#v", result)
	}
	result = &StageResult{}
	if stage.handleBuildResult(result, &build.Status{Status: BuildStatusPartial}) || len(result.Logs) == 0 {
		t.Fatalf("partial build result = %#v", result)
	}
	result = &StageResult{}
	failed := &build.Status{Status: BuildStatusFailed, ErrorLog: []string{"boom"}, PlatformResults: map[string]*build.PlatformResult{"linux": {Status: BuildStatusFailed}}}
	if !stage.handleBuildResult(result, failed) || result.ErrorInfo == nil || result.Details != failed {
		t.Fatalf("failed build result = %#v", result)
	}
}

type buildServiceSpy struct{}

func (*buildServiceSpy) PerformDesktopBuild(string, *build.BuildRequest)                    {}
func (*buildServiceSpy) PerformScenarioDesktopBuild(string, string, string, []string, bool) {}
func (*buildServiceSpy) BuildPlatform(string, string, string, string, string)               {}

func writeBundleValidationFixture(t *testing.T, packageJSON string) string {
	t.Helper()
	desktop := t.TempDir()
	if err := os.Mkdir(filepath.Join(desktop, "bundle"), 0o755); err != nil {
		t.Fatal(err)
	}
	if packageJSON != "" {
		if err := os.WriteFile(filepath.Join(desktop, "package.json"), []byte(packageJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return desktop
}

func TestValidateBundleConfigurationRequiresBundledResources(t *testing.T) {
	withoutBundle := t.TempDir()
	if result := validateBundleConfiguration(withoutBundle); !result.Valid || result.BundleExists {
		t.Fatalf("external-server bundle validation = %#v", result)
	}
	for _, check := range []struct{ name, packageJSON, want string }{
		{"missing package", "", "Cannot read package.json"},
		{"invalid JSON", "{", "Invalid package.json format"},
		{"missing extra resource", `{"build":{"extraResources":["assets"]}}`, "not included"},
	} {
		t.Run(check.name, func(t *testing.T) {
			result := validateBundleConfiguration(writeBundleValidationFixture(t, check.packageJSON))
			if result.Valid || !result.BundleExists || result.Error == "" || result.Suggestion == "" {
				t.Fatalf("result = %#v", result)
			}
			if !strings.Contains(result.Error, check.want) {
				t.Fatalf("error = %q, want %q", result.Error, check.want)
			}
		})
	}
}

func TestValidateBundleConfigurationAcceptsStringAndObjectResources(t *testing.T) {
	for _, packageJSON := range []string{
		`{"build":{"extraResources":["bundle/**"]}}`,
		`{"build":{"extraResources":[{"from":"bundle","to":"bundle"}]}}`,
	} {
		result := validateBundleConfiguration(writeBundleValidationFixture(t, packageJSON))
		if !result.Valid || !result.BundleExists || !result.IncludedInBuild || result.BundlePath == "" {
			t.Fatalf("valid bundle result = %#v", result)
		}
	}
}
