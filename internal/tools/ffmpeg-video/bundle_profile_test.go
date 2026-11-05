package ffmpegvideo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestVerifyTreeUsesSharedWholeTreeDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tool"), []byte("pinned"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := binaryfetch.TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTree(context.Background(), root, digest); err != nil {
		t.Fatalf("matching tree rejected: %v", err)
	}
	if err := VerifyTree(context.Background(), root, TreeSHA256); err == nil {
		t.Fatal("mismatched whole-tree digest accepted")
	}
}

func TestResolveBundleProfileAbsentAndTamperedStates(t *testing.T) {
	missing, err := ResolveBundleProfile(context.Background(), filepath.Join(t.TempDir(), "absent"), false)
	if err != nil {
		t.Fatal(err)
	}
	if missing.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING {
		t.Fatalf("absent optional profile state = %s", missing.GetState())
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "unexpected"), []byte("not the pinned archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	tampered, err := ResolveBundleProfile(context.Background(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	if tampered.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED {
		t.Fatalf("declared wrong tree state = %s", tampered.GetState())
	}
}
