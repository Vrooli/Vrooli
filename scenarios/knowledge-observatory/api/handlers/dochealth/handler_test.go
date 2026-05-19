package dochealth

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"knowledge-observatory/internal/services/dochealth"

	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newServiceFixture(t *testing.T) (*dochealth.Service, string) {
	t.Helper()
	root := t.TempDir()
	scenario := filepath.Join(root, "demo")
	writeFile(t, filepath.Join(scenario, "README.md"), "# demo\n")
	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), `{"version":"1","docs":[]}`)
	svc, err := dochealth.NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, "demo"
}

func TestHandler_DocHealth_ReturnsResponse(t *testing.T) {
	svc, name := newServiceFixture(t)
	h := New(svc)

	resp, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: name,
	}))
	if err != nil {
		t.Fatalf("DocHealth: %v", err)
	}
	if resp == nil || resp.Msg == nil {
		t.Fatalf("nil response")
	}
	if resp.Msg.GetScenarioName() != name {
		t.Errorf("scenario_name = %q, want %q", resp.Msg.GetScenarioName(), name)
	}
	if resp.Msg.GetCounts() == nil {
		t.Errorf("counts is nil")
	}
	if resp.Msg.GetTimestamp() == "" {
		t.Errorf("timestamp empty")
	}
}

func TestHandler_DocHealth_InvalidName(t *testing.T) {
	svc, _ := newServiceFixture(t)
	h := New(svc)

	_, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: "../escape",
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestHandler_DocHealth_NotFound(t *testing.T) {
	svc, _ := newServiceFixture(t)
	h := New(svc)

	_, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: "no-such-scenario",
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestHandler_DocHealth_UnavailableWhenServiceNil(t *testing.T) {
	h := New(nil)
	_, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: "demo",
	}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("code = %v, want Unavailable", connect.CodeOf(err))
	}
}
