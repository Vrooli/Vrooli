package toolprofiles

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestResolveBlenderWithoutDeclarationReportsMissingWithoutPATHLookup(t *testing.T) {
	message, err := Resolve(context.Background(), filepath.Join(t.TempDir(), "bundle"), BlenderToolID, nil)
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := message.(*cliv1.BlenderToolProfile)
	if !ok || profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING || profile.GetProfileId() != blenderrenderer.ProfileID {
		t.Fatalf("unexpected absent profile result: %#v", message)
	}
}

func TestResolveRejectsUnrecognizedDeclaredTreeBeforeInspectingPayload(t *testing.T) {
	declaration := manifest.ToolProfilePackage{ToolID: BlenderToolID, ProfileID: blenderrenderer.ProfileID, OS: "linux", Architecture: "amd64", TreeSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	_, err := Resolve(context.Background(), t.TempDir(), BlenderToolID, []manifest.ToolProfilePackage{declaration})
	if err == nil {
		t.Fatal("expected pinned-tree declaration mismatch")
	}
}

func TestResolveRejectsCallerSelectedProfileAndRelativeBundleRoot(t *testing.T) {
	if _, err := Resolve(context.Background(), t.TempDir(), "../blender", nil); err == nil {
		t.Fatal("unsupported selector accepted")
	}
	if _, err := Resolve(context.Background(), "relative", BlenderToolID, nil); err == nil {
		t.Fatal("relative bundle root accepted")
	}
}

func TestResolveRejectsBundleToolsSymlink(t *testing.T) {
	base := t.TempDir()
	bundle := filepath.Join(base, "bundle")
	external := filepath.Join(base, "outside")
	if err := os.MkdirAll(bundle, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(external, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(bundle, "tools")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Resolve(context.Background(), bundle, BlenderToolID, nil); err == nil {
		t.Fatal("bundle tools symlink was followed")
	}
}

func TestResolveRejectsSelectedPackageSymlink(t *testing.T) {
	base := t.TempDir()
	bundle := filepath.Join(base, "bundle")
	external := filepath.Join(base, "outside")
	if err := os.MkdirAll(filepath.Join(bundle, "tools"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(external, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(bundle, "tools", BlenderToolID)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Resolve(context.Background(), bundle, BlenderToolID, nil); err == nil {
		t.Fatal("selected renderer package symlink was followed")
	}
}
