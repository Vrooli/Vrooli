package collectors

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiskAttributionStopsOnCanceledContext(t *testing.T) {
	if collectorOS != "linux" {
		t.Skip("disk attribution uses Linux filesystem behavior")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "large.bin"), make([]byte, 1024), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := GetLargestDirectories(ctx, root, 2, 8, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("directory scan error = %v, want context canceled", err)
	}
	if _, err := GetLargestFiles(ctx, root, 8, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("file scan error = %v, want context canceled", err)
	}
}

func TestDiskAttributionSkipsNestedMountRoots(t *testing.T) {
	if collectorOS != "linux" {
		t.Skip("disk attribution uses Linux filesystem behavior")
	}
	root := t.TempDir()
	nested := filepath.Join(root, "external")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local.bin"), make([]byte, 16), 0o600); err != nil {
		t.Fatal(err)
	}
	externalFile := filepath.Join(nested, "external.bin")
	if err := os.WriteFile(externalFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(externalFile, 51*1024*1024); err != nil {
		t.Fatal(err)
	}

	got, err := GetLargestFiles(context.Background(), root, 8, []string{nested})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range got {
		if entry.Path == filepath.Join(nested, "external.bin") {
			t.Fatalf("nested mount contents leaked into parent scan: %+v", got)
		}
	}
	directories, err := GetLargestDirectories(context.Background(), root, 3, 8, []string{nested})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range directories {
		if entry.Path == nested {
			t.Fatalf("nested mount directory leaked into parent attribution: %+v", directories)
		}
	}
}
