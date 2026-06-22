package hosttool

import (
	"context"
	"errors"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type fakeInstaller struct {
	gotTool   string
	gotDryRun bool
	status    *cliv1.CliHostInstallStatus
	err       error
}

func (f *fakeInstaller) HostInstall(_ context.Context, tool string, dryRun bool) (*cliv1.CliHostInstallStatus, error) {
	f.gotTool = tool
	f.gotDryRun = dryRun
	return f.status, f.err
}

func TestEnsurerInspectIsDryRun(t *testing.T) {
	fake := &fakeInstaller{status: &cliv1.CliHostInstallStatus{ExecutionState: "would_install"}}
	e := NewEnsurerWithClient(fake)
	st, err := e.Inspect(context.Background(), "realesrgan-ncnn-vulkan")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !fake.gotDryRun {
		t.Fatalf("Inspect should call HostInstall with dryRun=true")
	}
	if st.GetExecutionState() != "would_install" {
		t.Fatalf("state = %q", st.GetExecutionState())
	}
}

func TestEnsurerEnsureIsNotDryRun(t *testing.T) {
	fake := &fakeInstaller{status: &cliv1.CliHostInstallStatus{ExecutionState: "installed", Ok: true}}
	e := NewEnsurerWithClient(fake)
	if _, err := e.Ensure(context.Background(), "realesrgan-ncnn-vulkan"); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if fake.gotDryRun {
		t.Fatalf("Ensure should call HostInstall with dryRun=false")
	}
}

func TestEnsurePropagatesError(t *testing.T) {
	fake := &fakeInstaller{err: errors.New("boom")}
	e := NewEnsurerWithClient(fake)
	if _, err := e.Ensure(context.Background(), "x"); err == nil {
		t.Fatalf("expected error to propagate")
	}
}

func TestToolForOperation(t *testing.T) {
	tool, ok := ToolForOperation("upscale")
	if !ok || tool != "realesrgan-ncnn-vulkan" {
		t.Fatalf("ToolForOperation(upscale) = %q,%v; want realesrgan-ncnn-vulkan,true", tool, ok)
	}
	if _, ok := ToolForOperation("not-an-op"); ok {
		t.Fatalf("unknown op should not resolve")
	}
}

func TestKnownTool(t *testing.T) {
	for _, tool := range []string{"sd", "realesrgan-ncnn-vulkan", "iopaint", "rembg", "llama-cpp", "python"} {
		if !KnownTool(tool) {
			t.Errorf("KnownTool(%q) = false, want true", tool)
		}
	}
	if KnownTool("totally-not-a-backend") {
		t.Fatalf("KnownTool should reject an unknown tool")
	}
}
