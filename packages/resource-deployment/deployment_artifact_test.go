package resourcedeployment

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestServiceArtifactVerifyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture-service")
	body := []byte("fixture service")
	if err := os.WriteFile(path, body, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	artifact := ServiceArtifact{Path: "bin/fixture-service", Version: "1.0.0", SHA256: fmt.Sprintf("%x", sum)}
	if err := artifact.VerifyFile(path); err != nil {
		t.Fatalf("VerifyFile() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := artifact.VerifyFile(path); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("VerifyFile() error = %v, want checksum mismatch", err)
	}
}

func TestServiceArtifactRejectsEscapingOrUnpinnedPath(t *testing.T) {
	t.Parallel()
	for _, artifact := range []ServiceArtifact{
		{Path: "../service", Version: "1.0.0", SHA256: strings.Repeat("a", 64)},
		{Path: "bin/service", Version: "", SHA256: strings.Repeat("a", 64)},
		{Path: "bin/service", Version: "1.0.0", SHA256: "not-a-checksum"},
	} {
		if err := artifact.Validate(); err == nil {
			t.Fatalf("Validate() succeeded for %#v", artifact)
		}
	}
}

func TestServiceArtifactResolvesTargetSpecificChecksum(t *testing.T) {
	t.Parallel()
	artifact := ServiceArtifact{Path: "bin/service", Version: "1.0.0", SHA256ByPlatform: map[string]string{
		"linux-amd64": strings.Repeat("a", 64),
	}}
	resolved, err := artifact.ForPlatform("linux", "amd64")
	if err != nil || resolved.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("ForPlatform() = %#v, %v", resolved, err)
	}
	if _, err := artifact.ForPlatform("windows", "amd64"); err == nil || !strings.Contains(err.Error(), "no checksum") {
		t.Fatalf("ForPlatform() error = %v, want target denial", err)
	}
}

func TestServiceArtifactResolvesBundledServerName(t *testing.T) {
	t.Parallel()
	artifact := ServiceArtifact{
		Path:           "bin/vault",
		Version:        "1.17.6",
		SHA256:         strings.Repeat("a", 64),
		BundleArtifact: "vault_${os}_${arch}",
	}
	name, err := artifact.BundleArtifactForPlatform("macos", "arm64")
	if err != nil || name != "vault_darwin_arm64" {
		t.Fatalf("BundleArtifactForPlatform() = %q, %v", name, err)
	}
	if _, err := (ServiceArtifact{Path: "bin/vault", Version: "1", SHA256: strings.Repeat("a", 64)}).BundleArtifactForPlatform("linux", "amd64"); err == nil {
		t.Fatal("BundleArtifactForPlatform() accepted undeclared bundle artifact")
	}
}

func TestServiceArtifactVerifiesDirectoryTreeAndResolvesEntry(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "service"), []byte("service"), 0o700); err != nil {
		t.Fatal(err)
	}
	digest, err := binaryfetch.TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	artifact := ServiceArtifact{Path: "server/service", Version: "1.0.0", Layout: "dir", EntryPath: "bin/service", SHA256: digest}
	if err := artifact.VerifyFile(root); err != nil {
		t.Fatalf("VerifyFile() error = %v", err)
	}
	launch, err := artifact.LaunchPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "bin", "service"); launch != want {
		t.Fatalf("LaunchPath() = %q, want %q", launch, want)
	}
}

func TestServiceArtifactContextRetainsTypedMismatchAndCancellation(t *testing.T) {
	for _, layout := range []string{"file", "dir"} {
		t.Run(layout, func(t *testing.T) {
			root := t.TempDir()
			file := filepath.Join(root, "service")
			body := []byte("original")
			if err := os.WriteFile(file, body, 0o700); err != nil {
				t.Fatal(err)
			}
			path := file
			digest := fmt.Sprintf("%x", sha256.Sum256(body))
			artifact := ServiceArtifact{Path: "service", Version: "1.0.0", Layout: layout, SHA256: digest}
			if layout == "dir" {
				path = root
				var err error
				digest, err = binaryfetch.TreeDigest(root)
				if err != nil {
					t.Fatal(err)
				}
				artifact.EntryPath = "service"
				artifact.SHA256 = digest
			}
			if err := artifact.VerifyFileContext(context.Background(), path); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := artifact.VerifyFileContext(ctx, path); !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
			if err := os.WriteFile(file, []byte("corrupt"), 0o700); err != nil {
				t.Fatal(err)
			}
			var mismatch *ArtifactChecksumMismatchError
			if err := artifact.VerifyFileContext(context.Background(), path); !errors.As(err, &mismatch) || mismatch.Want != digest || mismatch.Got == digest {
				t.Fatalf("lost typed checksum witness: %+v %v", mismatch, err)
			}
		})
	}
}
