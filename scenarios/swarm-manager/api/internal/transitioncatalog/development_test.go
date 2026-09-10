package transitioncatalog

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"swarm-manager/internal/development"
	"swarm-manager/internal/transitions"
)

func TestDevelopmentPreviewNeverDispatchesOrClaimsLaunchReadiness(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios/example/.vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios/example/.vrooli/service.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	s := NewService(transitions.Registry{}, runner)
	s.developmentReviewer = &development.Reviewer{RepoRoot: root}
	got, err := s.PreviewDevelopment(context.Background(), connect.NewRequest(&api.PreviewDevelopmentRequest{Scenario: "example"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.LaunchReady || got.Msg.ReviewComplete || len(got.Msg.Findings) == 0 || runner.startKey != "" || runner.applyKey != "" {
		t.Fatalf("preview crossed authority boundary: %#v %#v", got.Msg, runner)
	}
}

// This repository integration test exercises the real review packet without
// starting Swarm, invoking Agent Manager, or changing the proposed target.
// The ordinary fixture tests above remain independent of Audio Tools.
func TestAudioPilotReviewPacketRemainsUnapproved(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source location unavailable")
	}
	// Test Genie may compile with -trimpath, so use the package working
	// directory when runtime.Caller no longer identifies an absolute file.
	packageDir := filepath.Dir(source)
	if !filepath.IsAbs(source) {
		var err error
		packageDir, err = os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Clean(filepath.Join(packageDir, "..", "..", "..", "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "scenarios/audio-tools/docs/internal/local-dictation-proposal.json"))
	if err != nil {
		t.Fatal(err)
	}
	var proposal api.PreviewDevelopmentRequest
	if err := protojson.Unmarshal(data, &proposal); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	s := NewService(transitions.Registry{}, runner)
	s.developmentReviewer = &development.Reviewer{RepoRoot: root}
	result, err := s.PreviewDevelopment(context.Background(), connect.NewRequest(&proposal))
	if err != nil {
		t.Fatal(err)
	}
	if result.Msg.LaunchReady || result.Msg.ReviewComplete || len(result.Msg.Findings) < 2 {
		t.Fatalf("pilot must resolve its artifacts and retain the missing-budget finding: %s", result.Msg)
	}
	foundBudget, foundUntyped := false, false
	for _, finding := range result.Msg.Findings {
		foundBudget = foundBudget || finding.Code == "budget_required"
		foundUntyped = foundUntyped || finding.Code == "effects_untyped"
	}
	if !foundBudget || !foundUntyped {
		t.Fatalf("pilot findings=%v, want budget_required and effects_untyped", result.Msg.Findings)
	}
	if runner.startKey != "" || runner.applyKey != "" {
		t.Fatal("pilot preview dispatched work")
	}
	encoded, err := protojson.Marshal(result.Msg)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("preview-json: %s", encoded)
}

func TestDevelopmentPreviewRejectsMissingSourceAndInvalidRequest(t *testing.T) {
	s := NewService(transitions.Registry{}, &fakeRunner{})
	_, err := s.PreviewDevelopment(context.Background(), nil)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("nil request: %v", err)
	}
	_, err = s.PreviewDevelopment(context.Background(), connect.NewRequest(&api.PreviewDevelopmentRequest{Scenario: "example"}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("unconfigured reader: %v", err)
	}
}
