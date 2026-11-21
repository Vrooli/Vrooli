//go:build linux

package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

type openRGBPrepareFixture struct {
	envelope ManagedServiceArtifact
	final    string
	tree     string
	payload  string
	owner    openRGBStagePreparation
	extracts atomic.Int32
}

func newOpenRGBPrepareFixture(t *testing.T) *openRGBPrepareFixture {
	t.Helper()
	f := &openRGBPrepareFixture{final: filepath.Join(t.TempDir(), "published")}
	archive := filepath.Join(t.TempDir(), "acquired.AppImage")
	if err := os.WriteFile(archive, []byte("approved archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.envelope = ManagedServiceArtifact{Resource: openRGBResourceName, Path: archive, Version: "fixture", SHA256: updateTestDigest("approved archive")}
	_, f.tree, f.payload = testOpenRGBAppDir(t)
	f.owner, _ = openRGBStagePreparationOwner()
	f.owner.verifySeal = func(ctx context.Context, root string) error {
		return verifyOpenRGBAppDirEntries(ctx, root, uint32(os.Geteuid()))
	}
	f.owner.extract = func(ctx context.Context, archive *os.File, destination string) error {
		f.extracts.Add(1)
		data, err := os.ReadFile(archive.Name())
		if err != nil || string(data) != "approved archive" {
			return errors.New("extractor did not receive the approved snapshot")
		}
		for _, dir := range []string{"usr/bin", "usr/lib"} {
			if err := os.MkdirAll(filepath.Join(destination, dir), 0o755); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join(destination, openRGBAppDirPayload), []byte("synthetic ELF"), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(destination, "usr/lib/library.so"), []byte("synthetic library"), 0o644)
	}
	t.Cleanup(func() {
		// Fixture publication is read-only; release its temporary directories.
		_ = filepath.WalkDir(filepath.Dir(f.final), func(path string, entry os.DirEntry, err error) error {
			if err == nil && entry.IsDir() {
				return os.Chmod(path, 0o700)
			}
			return err
		})
	})
	return f
}

func (f *openRGBPrepareFixture) prepare(ctx context.Context) (string, error) {
	return prepareOpenRGBAppImageLocked(ctx, f.envelope, f.final, f.tree, f.payload, f.owner)
}

func TestOpenRGBStagePreparationPublishesOnceAndPreservesTheEnvelope(t *testing.T) {
	f := newOpenRGBPrepareFixture(t)
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			path, err := f.prepare(context.Background())
			if err != nil || path != filepath.Join(f.final, openRGBSealedAppDir) {
				t.Errorf("stage publication failed: %s, %v", path, err)
			}
		}()
	}
	wg.Wait()
	if f.extracts.Load() != 1 {
		t.Fatalf("concurrent setup extracted %d times", f.extracts.Load())
	}
	for _, path := range []string{f.envelope.Path, filepath.Join(f.final, openRGBSealedEnvelope)} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "approved archive" {
			t.Fatalf("governed or retained rollback envelope was lost: %v", err)
		}
	}
}

func TestOpenRGBStagePreparationRecoversLostPublicationAcknowledgement(t *testing.T) {
	f := newOpenRGBPrepareFixture(t)
	syncDir := f.owner.syncDir
	f.owner.syncDir = func(string) error { return errors.New("lost fsync acknowledgement") }
	if _, err := f.prepare(context.Background()); !errors.Is(err, ErrOpenRGBStageCommitUnknown) {
		t.Fatalf("visible uncertain publication was not retained: %v", err)
	}
	f.owner.syncDir = syncDir
	if _, err := f.prepare(context.Background()); err != nil || f.extracts.Load() != 1 {
		t.Fatalf("retry did not reconcile the visible qualified stage: %v", err)
	}
}

func TestOpenRGBStagePreparationCancellationAndFailedExtractionLeaveNoPublication(t *testing.T) {
	for _, name := range []string{"cancel", "partial extraction", "tampered closure"} {
		t.Run(name, func(t *testing.T) {
			f := newOpenRGBPrepareFixture(t)
			extract := f.owner.extract
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.owner.extract = func(ctx context.Context, archive *os.File, destination string) error {
				if err := extract(ctx, archive, destination); err != nil {
					return err
				}
				switch name {
				case "cancel":
					cancel()
					return ctx.Err()
				case "partial extraction":
					return errors.New("interrupted extractor")
				default:
					return os.WriteFile(filepath.Join(destination, "usr/lib/library.so"), []byte("wrong bytes"), 0o644)
				}
			}
			if _, err := f.prepare(ctx); err == nil {
				t.Fatal("failed setup published a stage")
			}
			for _, path := range []string{f.final, f.final + ".pending"} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("failed setup retained a partial stage: %s, %v", path, err)
				}
			}
			f.owner.extract = extract
			if _, err := f.prepare(context.Background()); err != nil {
				t.Fatalf("normal setup retry could not recover: %v", err)
			}
		})
	}
}

func TestOpenRGBStagePreparationRejectsTamperedPublishedBytesAndSnapshot(t *testing.T) {
	f := newOpenRGBPrepareFixture(t)
	if _, err := f.prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(f.final, openRGBSealedEnvelope)
	if err := os.Chmod(archive, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("changed archive"), 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := f.prepare(context.Background()); err == nil || f.extracts.Load() != 1 {
		t.Fatal("setup replaced a tampered published stage")
	}
	f = newOpenRGBPrepareFixture(t)
	if err := os.WriteFile(f.envelope.Path, []byte("different acquired bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.prepare(context.Background()); err == nil || f.extracts.Load() != 0 {
		t.Fatal("changed acquired bytes reached the extractor")
	}
}

func TestOpenRGBStagePreparationRecoversPrivateScratchAndRefusesAliasedScratch(t *testing.T) {
	f := newOpenRGBPrepareFixture(t)
	pending := f.final + ".pending"
	if err := os.Mkdir(pending, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pending, "partial"), []byte("interrupted snapshot"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.prepare(context.Background()); err != nil {
		t.Fatalf("private crash scratch could not be recovered: %v", err)
	}
	f = newOpenRGBPrepareFixture(t)
	outside := t.TempDir()
	keep := filepath.Join(outside, "keep")
	if err := os.WriteFile(keep, []byte("outside the stage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, f.final+".pending"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.prepare(context.Background()); err == nil || f.extracts.Load() != 0 {
		t.Fatal("aliased scratch was reused or removed")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("aliased scratch cleanup affected an unrelated directory")
	}
}

func TestOpenRGBPinnedAppImageStageFixture(t *testing.T) {
	archive := os.Getenv("VROOLI_OPENRGB_STAGE_FIXTURE")
	if archive == "" {
		t.Skip("set VROOLI_OPENRGB_STAGE_FIXTURE to the approved pinned archive for offline extraction proof")
	}
	f := newOpenRGBPrepareFixture(t)
	f.envelope = ManagedServiceArtifact{Resource: openRGBResourceName, Path: archive, Version: openRGBAppImageVersion, SHA256: openRGBAppImageSHA256}
	f.tree, f.payload = openRGBAppDirSHA256, openRGBPayloadSHA256
	owner, err := openRGBStagePreparationOwner()
	if err != nil {
		t.Fatal(err)
	}
	f.owner.extract = owner.extract
	if _, err := f.prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := verifyOpenRGBStageEnvelope(context.Background(), ManagedServiceArtifact{Path: filepath.Join(f.final, openRGBSealedEnvelope), SHA256: openRGBAppImageSHA256}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.prepare(context.Background()); err != nil {
		t.Fatalf("exact staged closure did not survive retry: %v", err)
	}
}
