package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-desktop-api/build"
)

type validationTargetDiscoveryStub struct {
	targets []deliveryramp.Target
	err     error
}

func (s validationTargetDiscoveryStub) Discover(context.Context) ([]deliveryramp.Target, error) {
	return s.targets, s.err
}

func TestBuildValidationSelectionBindsEachArtifactToItsTarget(t *testing.T) {
	dir := t.TempDir()
	linuxPath := filepath.Join(dir, "demo.AppImage")
	macPath := filepath.Join(dir, "demo.dmg")
	if err := os.WriteFile(linuxPath, []byte("linux"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(macPath, []byte("mac"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := NewSmokeTestStage(WithSmokeTestService(&mockSmokeTestService{}), WithValidationTargetDiscovery(validationTargetDiscoveryStub{targets: []deliveryramp.Target{{ID: "mac-node", Label: "minimouse", OS: "darwin", Available: true}}}))
	input := &StageInput{Config: &PipelineConfig{ScenarioName: "demo"}, BuildResult: &build.Status{PlatformResults: map[string]*build.PlatformResult{
		"linux": {Platform: "linux", Status: BuildStatusReady, Artifact: linuxPath},
		"mac":   {Platform: "mac", Status: BuildStatusReady, Artifact: macPath},
	}}}
	selection, err := stage.buildValidationSelection(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(selection.Targets))
	}
	for _, target := range selection.Targets {
		id := target.Descriptor.GetTargetId()
		if selection.ArtifactPaths[id] == "" || selection.ArtifactDigests[id] == "" {
			t.Fatalf("target %q has no artifact binding: %#v", id, selection)
		}
	}
	if selection.Targets[0].Descriptor.GetTargetId() == selection.Targets[1].Descriptor.GetTargetId() {
		t.Fatal("platforms collapsed onto one target")
	}
}

func TestBuildValidationSelectionEmitsUnavailableRemoteTarget(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "demo.dmg")
	if err := os.WriteFile(artifact, []byte("mac"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := NewSmokeTestStage(WithSmokeTestService(&mockSmokeTestService{}), WithValidationTargetDiscovery(validationTargetDiscoveryStub{}))
	input := &StageInput{Config: &PipelineConfig{ScenarioName: "demo"}, BuildResult: &build.Status{PlatformResults: map[string]*build.PlatformResult{
		"mac": {Platform: "mac", Status: BuildStatusReady, Artifact: artifact},
	}}}
	selection, err := stage.buildValidationSelection(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Targets) != 1 || selection.Targets[0].Descriptor.GetAvailable() {
		t.Fatalf("selection = %#v, want unavailable mac target", selection)
	}
	if selection.Targets[0].Descriptor.GetReason() == "" {
		t.Fatal("unavailable target has no actionable reason")
	}
}

func TestBuildValidationSelectionNormalizesQualifiedMacArtifactPlatform(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "demo.zip")
	if err := os.WriteFile(artifact, []byte("mac"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := NewSmokeTestStage(WithSmokeTestService(&mockSmokeTestService{}), WithValidationTargetDiscovery(validationTargetDiscoveryStub{}))
	input := &StageInput{Config: &PipelineConfig{ScenarioName: "demo"}, BuildResult: &build.Status{PlatformResults: map[string]*build.PlatformResult{
		"macos-amd64": {Platform: "macos-amd64", Status: BuildStatusReady, Artifact: artifact},
	}}}
	selection, err := stage.buildValidationSelection(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Targets) != 1 {
		t.Fatalf("targets = %d, want one unavailable mac target", len(selection.Targets))
	}
	target := selection.Targets[0].Descriptor
	if target.GetAvailable() || target.GetTargetId() != "bridge-unavailable-macos-amd64" {
		t.Fatalf("qualified mac target = %#v, want unavailable bridge target", target)
	}
}

func TestBuildValidationSelectionDegradesWhenBridgeDiscoveryFails(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "demo.zip")
	if err := os.WriteFile(artifact, []byte("mac"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := NewSmokeTestStage(
		WithSmokeTestService(&mockSmokeTestService{}),
		WithValidationTargetDiscovery(validationTargetDiscoveryStub{err: errors.New("bridge unavailable")}),
	)
	input := &StageInput{Config: &PipelineConfig{ScenarioName: "demo"}, BuildResult: &build.Status{PlatformResults: map[string]*build.PlatformResult{
		"macos-amd64": {Platform: "macos-amd64", Status: BuildStatusReady, Artifact: artifact},
	}}}
	selection, err := stage.buildValidationSelection(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Targets) != 1 {
		t.Fatalf("targets = %d, want one unavailable mac target", len(selection.Targets))
	}
	target := selection.Targets[0].Descriptor
	if target.GetAvailable() || target.GetTargetId() != "bridge-unavailable-macos-amd64" {
		t.Fatalf("bridge discovery failure target = %#v, want unavailable bridge target", target)
	}
	if target.GetReason() == "" || !strings.Contains(target.GetReason(), "bridge unavailable") {
		t.Fatalf("bridge discovery failure reason = %q, want discovery error", target.GetReason())
	}
}

func TestBuildValidationSelectionCarriesUnavailableTargetAction(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "demo.zip")
	if err := os.WriteFile(artifact, []byte("mac"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := NewSmokeTestStage(
		WithSmokeTestService(&mockSmokeTestService{}),
		WithValidationTargetDiscovery(validationTargetDiscoveryStub{targets: []deliveryramp.Target{{
			ID: "mac-node", OS: "darwin", Available: false,
			Reason:            "bridge node is online but declares no supported capability",
			MissingCapability: "gui-session",
			NextAction:        "start or auto-login an interactive GUI session, then probe again",
		}}}),
	)
	input := &StageInput{Config: &PipelineConfig{ScenarioName: "demo"}, BuildResult: &build.Status{PlatformResults: map[string]*build.PlatformResult{
		"macos-amd64": {Platform: "macos-amd64", Status: BuildStatusReady, Artifact: artifact},
	}}}
	selection, err := stage.buildValidationSelection(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	reason := selection.Targets[0].Descriptor.GetReason()
	if !strings.Contains(reason, "gui-session") || !strings.Contains(reason, "start or auto-login") {
		t.Fatalf("unavailable target reason = %q, want missing capability and next action", reason)
	}
}
