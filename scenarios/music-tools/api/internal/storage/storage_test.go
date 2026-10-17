package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/blobstore"
)

func TestOwnedBudgetEvictsOldestEligibleAndProtectsReserved(t *testing.T) {
	root := t.TempDir()
	store := NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	store.SetBudgetBytes(3)
	if err := store.PutOwned(context.Background(), "out/discarded.wav", "take-1", strings.NewReader("aa"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	if err := store.PutOwned(context.Background(), "out/reserved.wav", "take-2", strings.NewReader("bb"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected("out/discarded.wav", false)
	store.SetProtected("out/reserved.wav", true)
	removed, err := store.Evict(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != "out/discarded.wav" {
		t.Fatalf("removed=%v", removed)
	}
}

func TestEvictionPriorityPrecedesAge(t *testing.T) {
	store := newTestStore(t)
	store.SetBudgetBytes(2)
	if err := store.PutOwned(context.Background(), "out/available.wav", "take-a", strings.NewReader("aa"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	if err := store.PutOwned(context.Background(), "out/discarded.wav", "take-d", strings.NewReader("bb"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected("out/available.wav", false)
	store.SetProtected("out/discarded.wav", false)
	store.SetEvictionPriority("out/available.wav", 1)
	store.SetEvictionPriority("out/discarded.wav", 0)
	removed, err := store.Evict(context.Background())
	if err != nil || len(removed) != 1 || removed[0] != "out/discarded.wav" {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	return NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
}

func TestPutGetDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	want := []byte("PNGDATA")
	if err := s.Put(ctx, "inputs/a.png", bytes.NewReader(want), "audio/png"); err != nil {
		t.Fatal(err)
	}
	rc, mime, err := s.Get(ctx, "inputs/a.png")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !bytes.Equal(got, want) || mime != "audio/png" {
		t.Fatalf("got %q/%q", got, mime)
	}
	if err := s.Delete(ctx, "inputs/a.png"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Get(ctx, "inputs/a.png"); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestInvalidKeys(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	for _, k := range []string{"", "  ", "/abs/path", "../escape", "a/../../b"} {
		if err := s.Put(ctx, k, bytes.NewReader([]byte("x")), "audio/png"); err == nil {
			t.Errorf("key %q should be rejected", k)
		}
	}
}

func TestWriteToBlobKey(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	ref, err := s.Write(ctx, OutputTarget{BlobKey: "outputs/x.png"}, bytes.NewReader([]byte("data")), "audio/png")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "outputs/x.png" {
		t.Fatalf("ref=%q", ref)
	}
	if _, _, err := s.Get(ctx, "outputs/x.png"); err != nil {
		t.Fatalf("blob not stored: %v", err)
	}
}

func TestWriteToLocalPath(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	dest := filepath.Join(t.TempDir(), "nested", "result.png")
	ref, err := s.Write(ctx, OutputTarget{LocalPath: dest}, bytes.NewReader([]byte("local-owned")), "audio/png")
	if err != nil {
		t.Fatal(err)
	}
	if ref != dest {
		t.Fatalf("ref=%q want %q", ref, dest)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "local-owned" {
		t.Fatalf("local file wrong: %q err=%v", got, err)
	}
}

func TestOutputTargetValidate(t *testing.T) {
	cases := []struct {
		name   string
		target OutputTarget
		ok     bool
	}{
		{"blob", OutputTarget{BlobKey: "a/b.png"}, true},
		{"local-abs", OutputTarget{LocalPath: "/tmp/x.png"}, true},
		{"both", OutputTarget{BlobKey: "a", LocalPath: "/b"}, false},
		{"empty", OutputTarget{}, false},
		{"local-rel", OutputTarget{LocalPath: "rel/x.png"}, false},
		{"bad-blob", OutputTarget{BlobKey: "../escape"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.target.Validate(); (err == nil) != tc.ok {
				t.Fatalf("Validate ok=%v err=%v", tc.ok, err)
			}
		})
	}
}

func TestOwnedInventoryRestoresProtectedAndRetainsExactSize(t *testing.T) {
	root := t.TempDir()
	store := NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := store.PutOwned(context.Background(), "out/take.wav", "take", strings.NewReader("RIFF0000WAVE"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected("out/take.wav", false)
	restored := NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := restored.restoreEntries(); err != nil {
		t.Fatal(err)
	}
	entries := restored.Entries()
	if len(entries) != 1 || entries[0].Size != 12 || entries[0].Owner != "take" || !entries[0].Protected {
		t.Fatalf("inventory %+v", entries)
	}
	if err := restored.Delete(context.Background(), "out/take.wav"); err != nil {
		t.Fatal(err)
	}
	again := NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := again.restoreEntries(); err != nil || len(again.Entries()) != 0 {
		t.Fatalf("deleted inventory returned: %v", err)
	}
}

func TestOwnedReaderRejectsBeforeCommitAndObservesCancellation(t *testing.T) {
	r := &ownedReader{ctx: context.Background(), r: strings.NewReader("abc"), size: MaxOwnedBytes - 2}
	if _, err := r.Read(make([]byte, 8)); err == nil {
		t.Fatal("oversize accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r = &ownedReader{ctx: ctx, r: strings.NewReader("audio")}
	if _, err := r.Read(make([]byte, 8)); err != context.Canceled {
		t.Fatalf("cancel ignored: %v", err)
	}
}

func TestEligibleDeletionSerializesAgainstReservationCommit(t *testing.T) {
	store := newTestStore(t)
	if err := store.PutOwned(t.Context(), "out/take.wav", "take", strings.NewReader("audio"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected("out/take.wav", false)
	snapshot := store.Entries()[0]
	entered, proceed := make(chan struct{}), make(chan struct{})
	reserved := make(chan error, 1)
	go func() {
		_, err := store.ProtectAndCommit(snapshot.Key, func() error { close(entered); <-proceed; return nil })
		reserved <- err
	}()
	<-entered
	deletionStarted := make(chan struct{})
	deleted := make(chan bool, 1)
	deleteErrors := make(chan error, 1)
	go func() {
		close(deletionStarted)
		ok, err := store.DeleteEligible(t.Context(), snapshot)
		deleted <- ok
		deleteErrors <- err
	}()
	<-deletionStarted
	close(proceed)
	if err := <-reserved; err != nil {
		t.Fatal(err)
	}
	if <-deleted {
		t.Fatal("deleted reservation committed behind stale snapshot")
	}
	if err := <-deleteErrors; err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Get(t.Context(), snapshot.Key); err != nil {
		t.Fatal("reserved source lost", err)
	}
	// If deletion won first, a later reservation must fail without invoking commit.
	store.SetProtected(snapshot.Key, false)
	fresh := store.Entries()[0]
	if ok, err := store.DeleteEligible(t.Context(), fresh); err != nil || !ok {
		t.Fatalf("eligible delete %v %v", ok, err)
	}
	called := false
	if available, err := store.ProtectAndCommit(snapshot.Key, func() error { called = true; return nil }); err != nil || available || called {
		t.Fatal("reservation granted deleted source")
	}
}

func TestEligibleDeletionRejectsChangedSameSizeBytes(t *testing.T) {
	store := newTestStore(t)
	store.PutOwned(t.Context(), "out/take.wav", "take", strings.NewReader("first"), "audio/wav", true)
	store.SetProtected("out/take.wav", false)
	snapshot := store.Entries()[0]
	store.PutOwned(t.Context(), "out/take.wav", "take", strings.NewReader("other"), "audio/wav", true)
	store.SetProtected("out/take.wav", false)
	if removed, err := store.DeleteEligible(t.Context(), snapshot); err != nil || removed {
		t.Fatalf("stale identity deleted: %v %v", removed, err)
	}
}

// failingReservationStore injects substrate errors without host permission mutation.
type failingReservationStore struct {
	blobstore.BlobStore
	getErr       error
	receiptErr   error
	postAudioErr error
}

func (s *failingReservationStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if s.getErr != nil {
		return nil, "", s.getErr
	}
	return s.BlobStore.Get(ctx, key)
}
func (s *failingReservationStore) Delete(ctx context.Context, key string) error {
	if strings.HasPrefix(key, "ownership/") && s.receiptErr != nil {
		return s.receiptErr
	}
	err := s.BlobStore.Delete(ctx, key)
	if err == nil && !strings.HasPrefix(key, "ownership/") && s.postAudioErr != nil {
		return s.postAudioErr
	}
	return err
}

func TestReservationRefusesPhysicalLossAndNonregularAudio(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			store := newTestStore(t)
			key := "out/take.wav"
			if err := store.PutOwned(t.Context(), key, "take", strings.NewReader("audio"), "audio/wav", true); err != nil {
				t.Fatal(err)
			}
			path, _ := store.PathForKey(key)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink("missing.wav", path); err != nil {
					t.Fatal(err)
				}
			}
			called := false
			available, err := store.ProtectAndCommit(key, func() error { called = true; return nil })
			if available || called {
				t.Fatal("granted absent/nonregular audio")
			}
			if (err == nil) != (kind == "missing") {
				t.Fatalf("wrong absence classification: %v", err)
			}
		})
	}
}

func TestPartialOwnedDeletionRetiresAvailabilityAndPreservesReceiptError(t *testing.T) {
	root := t.TempDir()
	injected := errors.New("receipt cleanup unavailable")
	substrate := &failingReservationStore{BlobStore: blobstore.NewFilesystemBlobStore(root), receiptErr: injected}
	store := NewWithBlobStore(substrate, root)
	key := "out/take.wav"
	if err := store.PutOwned(t.Context(), key, "take", strings.NewReader("audio"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected(key, false)
	if removed, err := store.DeleteEligible(t.Context(), store.Entries()[0]); removed || !errors.Is(err, injected) {
		t.Fatalf("cleanup lost error: %v %v", removed, err)
	}
	if len(store.Entries()) != 0 {
		t.Fatal("removed audio remained available in catalog")
	}
	if _, err := os.Stat(filepath.Join(root, ownershipKey(key))); err != nil {
		t.Fatal("diagnostic receipt lost", err)
	}
	called := false
	if available, err := store.ProtectAndCommit(key, func() error { called = true; return nil }); available || called || err != nil {
		t.Fatalf("partial delete granted reservation: %v %v", available, err)
	}
}

func TestReservationReadFailuresPreserveCauseAndCatalog(t *testing.T) {
	for _, cause := range []error{os.ErrPermission, errors.New("transient read I/O"), context.DeadlineExceeded} {
		t.Run(cause.Error(), func(t *testing.T) {
			root := t.TempDir()
			substrate := &failingReservationStore{BlobStore: blobstore.NewFilesystemBlobStore(root), getErr: cause}
			store := NewWithBlobStore(substrate, root)
			key := "out/take.wav"
			if err := store.PutOwned(t.Context(), key, "take", strings.NewReader("audio"), "audio/wav", true); err != nil {
				t.Fatal(err)
			}
			store.SetProtected(key, false)
			before := store.Entries()[0]
			called := false
			available, err := store.ProtectAndCommit(key, func() error { called = true; return nil })
			if available || called || !errors.Is(err, cause) {
				t.Fatalf("read failure hidden: %v %v", available, err)
			}
			after := store.Entries()
			if len(after) != 1 || after[0] != before {
				t.Fatal("transient failure changed catalog")
			}
		})
	}
}

func TestAudioSidecarDeletionFailureAlsoRetiresAbsentAudio(t *testing.T) {
	root := t.TempDir()
	cause := errors.New("audio sidecar cleanup failed")
	substrate := &failingReservationStore{BlobStore: blobstore.NewFilesystemBlobStore(root), postAudioErr: cause}
	store := NewWithBlobStore(substrate, root)
	key := "out/take.wav"
	if err := store.PutOwned(t.Context(), key, "take", strings.NewReader("audio"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected(key, false)
	if removed, err := store.DeleteEligible(t.Context(), store.Entries()[0]); removed || !errors.Is(err, cause) {
		t.Fatalf("lost sidecar error: %v %v", removed, err)
	}
	if len(store.Entries()) != 0 {
		t.Fatal("audio removed but catalog remained available")
	}
	if _, err := os.Stat(filepath.Join(root, ownershipKey(key))); err != nil {
		t.Fatal("diagnostic receipt lost", err)
	}
}
