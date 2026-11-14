package models

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"image-tools/internal/fetch"
)

// TestLiveMoGeInstallerQualification is an explicit, opt-in acquisition check.
// It uses the same Installer and artifact validation as the product command,
// while allowing a qualification root outside the shared service's model tree.
func TestLiveMoGeInstallerQualification(t *testing.T) {
	root := os.Getenv("IMAGE_TOOLS_MOGE_QUALIFY_ROOT")
	if root == "" {
		t.Skip("set IMAGE_TOOLS_MOGE_QUALIFY_ROOT to install the pinned ~141 MB model into an isolated root")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 8 * time.Minute}
	installer := &Installer{
		Root: root, Reg: reg, Download: fetch.HTTPDownloader{Client: client},
		DiskAvail: fetch.DefaultDiskAvail,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	record, err := installer.Install(ctx, "moge-2-vits-normal", func(_ int, _ string) {})
	if err != nil {
		t.Fatal(err)
	}
	if record.Checksum == "" || record.SizeBytes < 140_000_000 {
		t.Fatalf("install record lacks verified artifact identity: %+v", record)
	}
	if _, err := os.Stat(record.Path + "/model.onnx"); err != nil {
		t.Fatalf("installed ONNX checkpoint is unavailable: %v", err)
	}
	t.Logf("qualified model=%s revision=e50ffda41565591092adea54c6ac83d6212e1e23 sha256=%s bytes=%d path=%s", record.ID, record.Checksum, record.SizeBytes, record.Path)
	fmt.Fprintf(os.Stdout, "MOGE_QUALIFICATION model=%s revision=e50ffda41565591092adea54c6ac83d6212e1e23 sha256=%s bytes=%d path=%s\n", record.ID, record.Checksum, record.SizeBytes, record.Path)
}
