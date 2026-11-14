package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internalhosttool "image-tools/internal/hosttool"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// fakeEnsurer returns a canned inspect status for EnsureBackend tests.
type fakeEnsurer struct {
	status *internalhosttool.Status
	err    error
	tool   string
}

func (f *fakeEnsurer) Inspect(_ context.Context, tool string) (*internalhosttool.Status, error) {
	f.tool = tool
	if f.err != nil {
		return nil, f.err
	}
	return f.status, nil
}

func ensureHandler(t *testing.T, ensurer BackendEnsurer, sub JobSubmitter) *connectHandler {
	t.Helper()
	return NewConnectHandler(Deps{Ensurer: ensurer, Jobs: sub})
}

func TestEnsureBackend_AlreadyPresent(t *testing.T) {
	h := ensureHandler(t, &fakeEnsurer{status: &internalhosttool.Status{ExecutionState: "already_present"}}, nil)
	resp, err := h.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{Tool: "realesrgan-ncnn-vulkan"}))
	if err != nil {
		t.Fatalf("EnsureBackend: %v", err)
	}
	if !resp.Msg.GetAlreadyInstalled() {
		t.Fatalf("expected already_installed, got %+v", resp.Msg)
	}
	if resp.Msg.GetJobId() != "" {
		t.Fatalf("no job should be submitted for an already-present tool")
	}
}

func TestEnsureBackend_ManualIsReportedNoJob(t *testing.T) {
	sub := &fakeSubmitter{}
	h := ensureHandler(t, &fakeEnsurer{status: &internalhosttool.Status{ExecutionState: "manual_action_required", Notes: []string{"pipx install iopaint"}}}, sub)
	resp, err := h.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{Tool: "iopaint"}))
	if err != nil {
		t.Fatalf("EnsureBackend: %v", err)
	}
	if !resp.Msg.GetManual() {
		t.Fatalf("expected manual=true, got %+v", resp.Msg)
	}
	if sub.submitted != 0 {
		t.Fatalf("manual tool must not submit a job")
	}
	if resp.Msg.GetDetail() == "" {
		t.Fatalf("manual response should carry guidance detail")
	}
}

func TestEnsureBackend_FetchableSubmitsJob(t *testing.T) {
	sub := &fakeSubmitter{}
	h := ensureHandler(t, &fakeEnsurer{status: &internalhosttool.Status{ExecutionState: "would_install"}}, sub)
	resp, err := h.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{Tool: "realesrgan-ncnn-vulkan"}))
	if err != nil {
		t.Fatalf("EnsureBackend: %v", err)
	}
	if resp.Msg.GetJobId() == "" {
		t.Fatalf("expected a job id for a fetchable tool")
	}
	if sub.submitted != 1 {
		t.Fatalf("expected one job submission, got %d", sub.submitted)
	}
	if sub.last.Operation != internalhosttool.EnsureJobOperation {
		t.Fatalf("job operation = %q, want %q", sub.last.Operation, internalhosttool.EnsureJobOperation)
	}
	if resp.Msg.GetEtaSeconds() <= 0 {
		t.Fatalf("expected a positive ETA")
	}
}

func TestEnsureBackend_DryRunNoJob(t *testing.T) {
	sub := &fakeSubmitter{}
	h := ensureHandler(t, &fakeEnsurer{status: &internalhosttool.Status{ExecutionState: "would_install"}}, sub)
	resp, err := h.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{Tool: "realesrgan-ncnn-vulkan", DryRun: true}))
	if err != nil {
		t.Fatalf("EnsureBackend: %v", err)
	}
	if resp.Msg.GetJobId() != "" || sub.submitted != 0 {
		t.Fatalf("dry-run must not submit a job")
	}
	if resp.Msg.GetState() != "would_install" {
		t.Fatalf("state = %q, want would_install", resp.Msg.GetState())
	}
}

func TestEnsureBackend_ResolvesOperation(t *testing.T) {
	f := &fakeEnsurer{status: &internalhosttool.Status{ExecutionState: "already_present"}}
	h := ensureHandler(t, f, nil)
	// "upscale" is served by realesrgan-ncnn-vulkan per provider bindings.
	_, err := h.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{Operation: "upscale"}))
	if err != nil {
		t.Fatalf("EnsureBackend: %v", err)
	}
	if f.tool != "realesrgan-ncnn-vulkan" {
		t.Fatalf("operation upscale resolved to %q, want realesrgan-ncnn-vulkan", f.tool)
	}
}

func TestEnsureBackend_UnknownToolRejected(t *testing.T) {
	h := ensureHandler(t, &fakeEnsurer{status: &internalhosttool.Status{}}, nil)
	_, err := h.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{Tool: "totally-not-a-backend"}))
	if err == nil {
		t.Fatalf("expected invalid-argument for unknown tool")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestEnsureBackend_Unimplemented(t *testing.T) {
	h := NewConnectHandler(Deps{}) // no Ensurer
	_, err := h.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{Tool: "realesrgan-ncnn-vulkan"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("code = %v, want unimplemented", connect.CodeOf(err))
	}
}
