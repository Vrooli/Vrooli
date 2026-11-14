//go:build linux

package resources

import (
	"os"
	"testing"
)

func TestOpenRGBNativeLinuxPrerequisiteReadiness(t *testing.T) {
	bundle := os.Getenv("VROOLI_OPENRGB_NATIVE_PREREQUISITE_BUNDLE")
	if bundle == "" {
		t.Skip("explicit native OS-library metadata qualification only")
	}
	if err := verifyOpenRGBReleaseRuntimePrerequisites(t.Context(), bundle); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRGBSystemLibraryResolverRejectsCallerPaths(t *testing.T) {
	for _, name := range []string{"../libc.so.6", "/tmp/libc.so.6", "C:\\libc.so.6", "libc.so.6\x00"} {
		if _, err := openRGBSystemLibrary("amd64", name); err == nil {
			t.Fatalf("accepted unbounded loader input %q", name)
		}
	}
}
