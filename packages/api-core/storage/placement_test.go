package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePlacementPlatformAbsent(t *testing.T) {
	_, err := ResolvePlacement(PlacementRequest{Entry: "optional", Platform: PlatformWindows, Path: PortablePath{ByOS: map[Platform]*string{PlatformWindows: nil}}}, PlatformSeams{})
	var absent *NotApplicable
	if !errors.As(err, &absent) {
		t.Fatalf("error = %v, want typed NotApplicable", err)
	}
}

func TestMigrateRequiresApproval(t *testing.T) {
	if err := Migrate("", "/a", "/b", false); err == nil {
		t.Fatal("expected migration gate")
	}
}

func TestMigrateCopiesVerifiesAndRemovesOnlyAfterApproval(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "data.bin"), []byte("important"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := MigrateVerified("plan-1", source, destination, true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified || result.SourcePreserved || result.Bytes != int64(len("important")) {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source remains, err = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "nested", "data.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "important" {
		t.Fatalf("destination = %q", data)
	}
}
