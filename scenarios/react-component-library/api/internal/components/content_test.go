package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func materializeFixture() ComponentVersion {
	body := "export const Button = () => null;\n"
	return ComponentVersion{
		LibraryID:  "react-component-library:Button",
		Version:    "1.0.0",
		SourcePath: "library/components/Button/versions/1.0.0/Button.tsx",
		Files:      []ComponentVersionFile{{Path: "Button.tsx", Content: body, ContentSHA256: digest([]byte(body)), IsEntry: true}},
	}
}

// TestFSContentStore_RoundTrip exercises the happy path: read a known
// file, write it back with a new body, read again to confirm the new
// digest sticks.
func TestFSContentStore_RoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Button.tsx"), []byte("export const Button = () => null;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFSContentStore(root)
	c := Component{SourcePath: "Button.tsx"}

	got, err := store.Read(context.Background(), c)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Body == "" || got.SHA256 == "" {
		t.Fatalf("Read returned empty body or sha")
	}

	newBody := "export const Button = () => <button>x</button>;\n"
	written, err := store.Write(context.Background(), c, WriteContentInput{Body: newBody, ExpectedSHA256: got.SHA256})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if written.Body != newBody {
		t.Fatalf("written body mismatch")
	}

	after, err := store.Read(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if after.Body != newBody {
		t.Fatalf("re-read body mismatch")
	}
	if after.SHA256 != written.SHA256 {
		t.Fatalf("sha mismatch after roundtrip")
	}
}

// TestFSContentStore_PathEscape — the guard must reject paths that
// resolve outside the configured root.
func TestFSContentStore_PathEscape(t *testing.T) {
	root := t.TempDir()
	store := NewFSContentStore(root)

	cases := []string{
		"../etc/passwd",
		"a/../../escape.tsx",
		"/etc/passwd",
		"",
		"../" + filepath.Base(root) + "_sibling/x.tsx",
	}
	for _, sp := range cases {
		t.Run(sp, func(t *testing.T) {
			_, err := store.Read(context.Background(), Component{SourcePath: sp})
			var esc ErrPathEscape
			if !errors.As(err, &esc) {
				t.Fatalf("expected ErrPathEscape for %q, got %v", sp, err)
			}
			_, err = store.Write(context.Background(), Component{SourcePath: sp}, WriteContentInput{Body: "x"})
			if !errors.As(err, &esc) {
				t.Fatalf("Write expected ErrPathEscape for %q, got %v", sp, err)
			}
		})
	}
}

func TestFSContentStore_ReadPath_AllowsExperienceContract(t *testing.T) {
	root := t.TempDir()
	versionDir := filepath.Join(root, "library", "primitives", "MorphingIcon", "versions", "2.0.7")
	if err := os.MkdirAll(versionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "MorphingIcon.tsx"), []byte("export const MorphingIcon = () => null;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := `{"contract":{"kind":"rcl-component-experience-contract"}}`
	if err := os.WriteFile(filepath.Join(versionDir, "experience-contract.json"), []byte(contract), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := NewFSContentStore(root).ReadPath(context.Background(), Component{
		SourcePath: "library/primitives/MorphingIcon/versions/2.0.7/MorphingIcon.tsx",
	}, "experience-contract.json")
	if err != nil {
		t.Fatalf("ReadPath experience contract: %v", err)
	}
	if got.Body != contract {
		t.Fatalf("experience contract body: got %q", got.Body)
	}
	if got.SourcePath != "library/primitives/MorphingIcon/versions/2.0.7/experience-contract.json" {
		t.Fatalf("experience contract path: got %q", got.SourcePath)
	}
}

// TestFSContentStore_OptimisticConcurrency — when the caller pins an
// expected sha that doesn't match disk, Write must reject without
// modifying the file.
func TestFSContentStore_OptimisticConcurrency(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Card.tsx")
	original := []byte("v1")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFSContentStore(root)
	c := Component{SourcePath: "Card.tsx"}

	_, err := store.Write(context.Background(), c, WriteContentInput{
		Body:           "v2",
		ExpectedSHA256: "deadbeef",
	})
	var conflict ErrContentConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ErrContentConflict, got %v", err)
	}
	on, _ := os.ReadFile(path)
	if string(on) != "v1" {
		t.Fatalf("file mutated despite conflict; got %q", on)
	}
}

// TestFSContentStore_ReadMissingFile — surfaces a wrapped os.* error
// (mapped to Internal at the transport edge); we just verify the error
// is non-nil and not a typed ErrPathEscape.
func TestFSContentStore_ReadMissingFile(t *testing.T) {
	store := NewFSContentStore(t.TempDir())
	_, err := store.Read(context.Background(), Component{SourcePath: "no-such.tsx"})
	if err == nil {
		t.Fatal("expected error")
	}
	var esc ErrPathEscape
	if errors.As(err, &esc) {
		t.Fatalf("missing file must not classify as ErrPathEscape")
	}
}

func TestFSContentStore_MaterializeRestoresByteIdenticalFiles(t *testing.T) {
	root := t.TempDir()
	version := materializeFixture()
	result, err := NewFSContentStore(root).Materialize(context.Background(), version, "")
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if result.FilesWritten != 1 || result.AlreadyPresent {
		t.Fatalf("unexpected materialization result: %+v", result)
	}
	got, err := os.ReadFile(filepath.Join(root, "library/components/Button/versions/1.0.0/Button.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != version.Files[0].Content || digest(got) != version.Files[0].ContentSHA256 {
		t.Fatalf("materialized bytes differ: %q", got)
	}
}

func TestFSContentStore_RestoreRetiredSourceUsesDurableArchive(t *testing.T) {
	root := t.TempDir()
	version := ComponentVersion{
		LibraryID:  "react-component-library:Icon",
		Version:    "1.1.3",
		SourcePath: "components/Icon/versions/1.1.3/Icon.tsx",
	}
	key := sha256.Sum256([]byte(version.LibraryID + "@" + version.Version))
	archive := filepath.Join(root, ".retired", hex.EncodeToString(key[:]))
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archive, "Icon.tsx"), []byte("export const Icon = () => null;\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	restored, err := NewFSContentStore(root).RestoreRetiredSource(version)
	if err != nil {
		t.Fatalf("RestoreRetiredSource: %v", err)
	}
	if !restored {
		t.Fatal("expected the durable retired archive to be restored")
	}
	if _, err := os.Stat(filepath.Join(root, version.SourcePath)); err != nil {
		t.Fatalf("restored source missing: %v", err)
	}
	if _, err := os.Stat(archive); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired archive should move atomically, stat error=%v", err)
	}
}

func TestFSContentStore_MaterializeIsIdempotentAndConcurrencySafe(t *testing.T) {
	root := t.TempDir()
	store := NewFSContentStore(root)
	version := materializeFixture()
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Materialize(context.Background(), version, "")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := store.Materialize(context.Background(), version, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent || result.FilesWritten != 0 {
		t.Fatalf("retry should report an existing complete directory: %+v", result)
	}
	entries, err := os.ReadDir(filepath.Dir(filepath.Join(root, version.SourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "Button.tsx" {
		t.Fatalf("unexpected version directory contents: %+v", entries)
	}
}

func TestFSContentStore_MaterializeRepairsDivergentWorkingTreeFromVerifiedMirror(t *testing.T) {
	root := t.TempDir()
	version := materializeFixture()
	destination := filepath.Join(root, "library/components/Button/versions/1.0.0")
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "Button.tsx"), []byte("mutated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "unexpected.ts"), []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := NewFSContentStore(root).Materialize(context.Background(), version, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent || result.FilesWritten != 1 {
		t.Fatalf("repair result = %+v", result)
	}
	got, err := os.ReadFile(filepath.Join(destination, "Button.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != version.Files[0].Content {
		t.Fatalf("repaired source = %q", got)
	}
	if _, err := os.Stat(filepath.Join(destination, "unexpected.ts")); !os.IsNotExist(err) {
		t.Fatalf("unexpected file survived exact repair: %v", err)
	}
}

func TestFSContentStore_MaterializeAbortsOnHashMismatch(t *testing.T) {
	root := t.TempDir()
	version := materializeFixture()
	version.Files[0].ContentSHA256 = "bad-hash"
	_, err := NewFSContentStore(root).Materialize(context.Background(), version, "")
	var mismatch ErrMaterializationHashMismatch
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected hash mismatch, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "library/components/Button/versions/1.0.0")); !os.IsNotExist(statErr) {
		t.Fatalf("hash failure created destination: %v", statErr)
	}
}

func TestFSContentStore_MaterializeIntoLeavesWorkingTreeUnchanged(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	version := materializeFixture()
	result, err := NewFSContentStore(root).Materialize(context.Background(), version, target)
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesWritten != 1 {
		t.Fatalf("expected one projected file: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "library/components/Button/versions/1.0.0")); !os.IsNotExist(err) {
		t.Fatalf("projection touched working tree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "library/components/Button/versions/1.0.0/Button.tsx")); err != nil {
		t.Fatalf("projection missing file: %v", err)
	}
}
