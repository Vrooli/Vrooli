package build

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type connectBuildService struct {
	desktop  chan *BuildRequest
	scenario chan scenarioCall
}

type scenarioCall struct {
	id, scenario, path string
	platforms          []string
	clean              bool
}

func (s *connectBuildService) PerformDesktopBuild(_ string, request *BuildRequest) {
	s.desktop <- request
}

func (s *connectBuildService) PerformScenarioDesktopBuild(id, scenario, path string, platforms []string, clean bool) {
	s.scenario <- scenarioCall{id: id, scenario: scenario, path: path, platforms: platforms, clean: clean}
}
func (*connectBuildService) BuildPlatform(string, string, string, string, string) {}

func newConnectBuildService() (*ConnectService, *connectBuildService, *InMemoryStore) {
	service := &connectBuildService{desktop: make(chan *BuildRequest, 1), scenario: make(chan scenarioCall, 1)}
	store := NewStore()
	return NewConnectService(NewHandler(service, store)), service, store
}

func TestStartBuildUsesTypedPlatformsAndDefaultsToLinux(t *testing.T) {
	handler, service, store := newConnectBuildService()
	response, err := handler.StartBuild(context.Background(), connect.NewRequest(&domainv1.BuildRequest{DesktopPath: "/tmp/desktop", Platforms: []sharedv1.Platform{sharedv1.Platform_PLATFORM_WIN, sharedv1.Platform_PLATFORM_LINUX}, Sign: boolPtr(true), Publish: boolPtr(true)}))
	if err != nil || response.Msg.GetBuildId() == "" || response.Msg.GetStatus() != "building" {
		t.Fatalf("StartBuild() = %#v, %v", response, err)
	}
	select {
	case request := <-service.desktop:
		if len(request.Platforms) != 2 || request.Platforms[0] != "win" || !request.Sign || !request.Publish {
			t.Fatalf("build request = %#v", request)
		}
	case <-time.After(time.Second):
		t.Fatal("desktop build was not dispatched")
	}
	status, ok := store.Get(response.Msg.GetBuildId())
	if !ok || status.PlatformResults["win"] == nil || status.PlatformResults["linux"] == nil {
		t.Fatalf("stored build = %#v, %v", status, ok)
	}

	response, err = handler.StartBuild(context.Background(), connect.NewRequest(&domainv1.BuildRequest{DesktopPath: "/tmp/default"}))
	if err != nil {
		t.Fatalf("default StartBuild() error = %v", err)
	}
	<-service.desktop
	if status, ok := store.Get(response.Msg.GetBuildId()); !ok || len(status.RequestedPlatforms) != 1 || status.RequestedPlatforms[0] != "linux" {
		t.Fatalf("default platforms = %#v, %v", status, ok)
	}
}

func TestBuildConnectValidationAndStatusContract(t *testing.T) {
	handler, service, store := newConnectBuildService()
	if _, err := handler.StartBuild(context.Background(), connect.NewRequest(&domainv1.BuildRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty desktop path code = %v", connect.CodeOf(err))
	}
	if _, err := handler.StartScenarioBuild(context.Background(), connect.NewRequest(&domainv1.ScenarioBuildRequest{DesktopPath: "/tmp/desktop"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty scenario name code = %v", connect.CodeOf(err))
	}
	response, err := handler.StartScenarioBuild(context.Background(), connect.NewRequest(&domainv1.ScenarioBuildRequest{ScenarioName: "calculator", DesktopPath: "/tmp/desktop", Platforms: []sharedv1.Platform{sharedv1.Platform_PLATFORM_MAC}, Clean: boolPtr(true)}))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case call := <-service.scenario:
		if call.id != response.Msg.GetBuildId() || call.scenario != "calculator" || call.platforms[0] != "mac" || !call.clean {
			t.Fatalf("scenario build call = %#v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("scenario build was not dispatched")
	}
	if _, err := handler.GetBuild(context.Background(), connect.NewRequest(&domainv1.BuildStatusRequest{BuildId: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing build code = %v", connect.CodeOf(err))
	}
	got, err := handler.GetBuild(context.Background(), connect.NewRequest(&domainv1.BuildStatusRequest{BuildId: response.Msg.GetBuildId()}))
	if err != nil || got.Msg.GetScenarioName() != "calculator" || got.Msg.GetRequestedPlatforms()[0] != sharedv1.Platform_PLATFORM_MAC {
		t.Fatalf("GetBuild() = %#v, %v", got, err)
	}
	_ = store
}

func TestStatusToProtoPreservesTerminalPlatformEvidence(t *testing.T) {
	started := time.Now().Add(-time.Minute).UTC()
	completed := time.Now().UTC()
	proto := StatusToProto(&Status{BuildID: "build-1", ScenarioName: "calculator", Status: "partial", RequestedPlatforms: []string{"windows", "unknown"}, PlatformResults: map[string]*PlatformResult{"win": {Platform: "win", Status: "failed", StartedAt: &started, CompletedAt: &completed, ErrorLog: []string{"signing failed"}, Artifact: "installer.exe", FileSize: 42}}, OutputPath: "/tmp/out", CreatedAt: started, CompletedAt: &completed, ErrorLog: []string{"partial"}, BuildLog: []string{"started"}, Artifacts: map[string]string{"win": "installer.exe"}})
	if proto.GetStatus() != sharedv1.BuildStatus_BUILD_STATUS_PARTIAL || proto.GetPlatformResults()["win"].GetStatus() != sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_FAILED || proto.GetPlatformResults()["win"].GetFileSize() != 42 || proto.GetRequestedPlatforms()[1] != sharedv1.Platform_PLATFORM_UNSPECIFIED {
		t.Fatalf("proto status = %#v", proto)
	}
}

func TestBuildProtoEnumMappingsCoverSupportedAndUnknownValues(t *testing.T) {
	for input, want := range map[string]sharedv1.BuildStatus{
		"building": sharedv1.BuildStatus_BUILD_STATUS_BUILDING,
		"ready":    sharedv1.BuildStatus_BUILD_STATUS_READY,
		"failed":   sharedv1.BuildStatus_BUILD_STATUS_FAILED,
		"other":    sharedv1.BuildStatus_BUILD_STATUS_UNSPECIFIED,
	} {
		if got := buildStatusProto(input); got != want {
			t.Fatalf("buildStatusProto(%q) = %v, want %v", input, got, want)
		}
	}
	for input, want := range map[string]sharedv1.PlatformBuildStatus{
		"pending": sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_BUILDING,
		"ready":   sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_READY,
		"skipped": sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_SKIPPED,
		"other":   sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_UNSPECIFIED,
	} {
		if got := platformStatusProto(input); got != want {
			t.Fatalf("platformStatusProto(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestPlatformProtoNormalizesCanonicalArchitectureValues(t *testing.T) {
	tests := map[string]sharedv1.Platform{
		"linux-amd64": sharedv1.Platform_PLATFORM_LINUX,
		"macos-arm64": sharedv1.Platform_PLATFORM_MAC,
		"windows-x64": sharedv1.Platform_PLATFORM_WIN,
	}
	for input, want := range tests {
		if got := platformProto(input); got != want {
			t.Fatalf("platformProto(%q) = %v, want %v", input, got, want)
		}
	}
}

func boolPtr(value bool) *bool { return &value }
