package projects

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/blobstore"
	_ "modernc.org/sqlite"
)

func newMediaFixture(t *testing.T, store blobstore.BlobStore, id func() string) (*sourceMediaService, *sqliteRepository, string) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, nil).(*sqliteRepository)
	projectService := NewService(repo, func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) })
	project, err := projectService.Create(context.Background(), CreateInput{Name: "Fixture"})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewSourceMediaService(SourceMediaDeps{
		Projects: projectService, Repository: repo, Store: store,
		Now:   func() time.Time { return time.Date(2026, 10, 20, 12, 1, 0, 123, time.UTC) },
		NewID: id,
	}).(*sourceMediaService)
	return svc, repo, project.ID
}

func grayPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 5, 3))
	img.SetGray(2, 1, color.Gray{Y: 128})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 7, 4))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 82}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestSourceMediaImportPreservesGrayPNGAndIgnoresClientMetadata(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	svc, _, projectID := newMediaFixture(t, store, func() string { return "media-" + string(rune('0'+next.Add(1))) })
	source := grayPNG(t)
	got, err := svc.Import(context.Background(), projectID, "mask.png", int64(len(source)), bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if got.MediaType != "image/png" || got.Width != 5 || got.Height != 3 || got.ColorModel != "grayscale" {
		t.Fatalf("metadata = %+v", got)
	}
	if got.Orientation != "normal" || got.ColorProfileStatus != "unspecified" || got.ImportMethod != "local_upload" {
		t.Fatalf("provenance = %+v", got)
	}
	if got.SHA256 == "" || got.SizeBytes != int64(len(source)) || got.OriginalFilename != "mask.png" {
		t.Fatalf("incomplete source metadata = %+v", got)
	}
	r, mime, err := store.Get(context.Background(), got.storageKey)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	stored, err := io.ReadAll(r)
	if err != nil || !bytes.Equal(stored, source) || mime != "image/png" {
		t.Fatalf("stored source differs from original: mime=%q err=%v", mime, err)
	}
}

func TestSourceMediaImportDetectsJPEGAndIdempotentlyDeduplicates(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	svc, repo, projectID := newMediaFixture(t, store, func() string { return "media-" + string(rune('0'+next.Add(1))) })
	source := jpegBytes(t)
	first, err := svc.Import(context.Background(), projectID, "photo.jpeg", int64(len(source)), bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Import(context.Background(), projectID, "spoofed.png", int64(len(source)), bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.SHA256 != second.SHA256 || first.MediaType != "image/jpeg" || first.Width != 7 || first.Height != 4 {
		t.Fatalf("duplicate/import metadata differ: first=%+v second=%+v", first, second)
	}
	if _, _, err := store.Get(context.Background(), "projects/"+projectID+"/media/media-4"); err == nil {
		t.Fatal("losing duplicate staging object was not removed")
	}
	if _, err := repo.GetSourceMedia(context.Background(), projectID, first.ID); err != nil {
		t.Fatalf("winner metadata missing: %v", err)
	}
}

func TestSourceMediaConcurrentIdenticalImportsConverge(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	svc, _, projectID := newMediaFixture(t, store, func() string { return "concurrent-" + string(rune('a'+next.Add(1))) })
	source := grayPNG(t)
	const workers = 8
	ids := make(chan string, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			media, err := svc.Import(context.Background(), projectID, "same.png", int64(len(source)), bytes.NewReader(source))
			if err != nil {
				errs <- err
				return
			}
			ids <- media.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	winner := ""
	for id := range ids {
		if winner == "" {
			winner = id
		}
		if id != winner {
			t.Fatalf("concurrent imports returned multiple IDs %q and %q", winner, id)
		}
	}
	if winner == "" {
		t.Fatal("no source media returned")
	}
}

func TestSourceMediaRejectsInvalidAndOversizedInputsBeforeStorage(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	svc, _, projectID := newMediaFixture(t, store, func() string { return "media-id" })
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "unsupported", data: []byte("GIF89a")},
		{name: "corrupt", data: []byte("not an image")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Import(context.Background(), projectID, "file", int64(len(tc.data)), bytes.NewReader(tc.data))
			if err == nil {
				t.Fatal("invalid input unexpectedly imported")
			}
			var invalid ErrInvalidSourceMedia
			if !errors.As(err, &invalid) {
				t.Fatalf("error = %v, want invalid source media", err)
			}
		})
	}
	_, err := svc.Import(context.Background(), projectID, "large.png", MaxSourceMediaBytes+1, bytes.NewReader(nil))
	var tooLarge ErrSourceMediaTooLarge
	if !errors.As(err, &tooLarge) {
		t.Fatalf("oversize error = %v", err)
	}
	if _, _, err := store.Get(context.Background(), "projects/"+projectID+"/media/media-id"); err == nil {
		t.Fatal("rejected inputs wrote a blob")
	}
}

type countingReader struct {
	reads atomic.Int32
	data  *bytes.Reader
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads.Add(1)
	return r.data.Read(p)
}

func TestSourceMediaChecksProjectBeforeReadingBody(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	svc, _, _ := newMediaFixture(t, store, func() string { return "media-id" })
	data := grayPNG(t)
	reader := &countingReader{data: bytes.NewReader(data)}
	_, err := svc.Import(context.Background(), "missing-project", "gray.png", int64(len(data)), reader)
	var missing ErrProjectNotFound
	if !errors.As(err, &missing) {
		t.Fatalf("error = %v, want missing project", err)
	}
	if reader.reads.Load() != 0 {
		t.Fatalf("file read %d times before project validation", reader.reads.Load())
	}
}

type failingMediaRepository struct{ SourceMediaRepository }

func (f failingMediaRepository) CreateSourceMedia(context.Context, SourceMedia) (SourceMedia, bool, error) {
	return SourceMedia{}, false, errors.New("database unavailable")
}

func TestSourceMediaRepositoryFailureRemovesOnlyOwnedObject(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	svc, repo, projectID := newMediaFixture(t, store, func() string { return "media-own" })
	svc.repo = failingMediaRepository{SourceMediaRepository: repo}
	source := grayPNG(t)
	_, err := svc.Import(context.Background(), projectID, "gray.png", int64(len(source)), bytes.NewReader(source))
	if err == nil {
		t.Fatal("repository failure was hidden")
	}
	if _, _, err := store.Get(context.Background(), "projects/"+projectID+"/media/media-own"); err == nil {
		t.Fatal("unreferenced object remains after metadata failure")
	}
}

type writeThenFailBlobStore struct{ base blobstore.BlobStore }

func (s writeThenFailBlobStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := s.base.Put(ctx, key, r, mime); err != nil {
		return err
	}
	return errors.New("simulated metadata sidecar failure")
}

type cancelAfterPutBlobStore struct {
	base   blobstore.BlobStore
	cancel context.CancelFunc
}

func (s cancelAfterPutBlobStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := s.base.Put(ctx, key, r, mime); err != nil {
		return err
	}
	s.cancel()
	return nil
}
func (s cancelAfterPutBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	return s.base.Get(ctx, key)
}
func (s cancelAfterPutBlobStore) Delete(ctx context.Context, key string) error {
	return s.base.Delete(ctx, key)
}
func (s writeThenFailBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	return s.base.Get(ctx, key)
}
func (s writeThenFailBlobStore) Delete(ctx context.Context, key string) error {
	return s.base.Delete(ctx, key)
}

func TestSourceMediaPutFailureRemovesPartiallyCommittedUniqueObject(t *testing.T) {
	base := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	svc, _, projectID := newMediaFixture(t, writeThenFailBlobStore{base: base}, func() string { return "put-fail-" + string(rune('a'+next.Add(1))) })
	source := grayPNG(t)
	_, err := svc.Import(context.Background(), projectID, "mask.png", int64(len(source)), bytes.NewReader(source))
	if err == nil {
		t.Fatal("Put failure was hidden")
	}
	if _, _, err := base.Get(context.Background(), "projects/"+projectID+"/media/put-fail-b"); err == nil {
		t.Fatal("partially committed unique object remains after Put failure")
	}
}

func TestSourceMediaCleanupSurvivesRequestCancellationAndKeepsRoutingValues(t *testing.T) {
	base := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc, repo, projectID := newMediaFixture(t, cancelAfterPutBlobStore{base: base, cancel: cancel}, func() string { return "cancel-" + string(rune('a'+next.Add(1))) })
	svc.repo = failingMediaRepository{SourceMediaRepository: repo}
	source := grayPNG(t)
	_, err := svc.Import(ctx, projectID, "gray.png", int64(len(source)), bytes.NewReader(source))
	if err == nil {
		t.Fatal("metadata failure was hidden")
	}
	if _, _, err := base.Get(context.Background(), "projects/"+projectID+"/media/cancel-b"); err == nil {
		t.Fatal("request cancellation left the staged object behind")
	}
}

func TestSourceMediaOpenVerifiesStoredBytesBeforeServing(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	svc, _, projectID := newMediaFixture(t, store, func() string { return "open-" + string(rune('a'+next.Add(1))) })
	source := grayPNG(t)
	media, err := svc.Import(context.Background(), projectID, "gray.png", int64(len(source)), bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	_, reader, err := svc.Open(context.Background(), projectID, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source) {
		t.Fatal("open did not preserve original bytes")
	}
	if err := store.Put(context.Background(), media.storageKey, bytes.NewReader([]byte("tampered")), "image/png"); err != nil {
		t.Fatal(err)
	}
	if _, reader, err := svc.Open(context.Background(), projectID, media.ID); err == nil {
		_ = reader.Close()
		t.Fatal("tampered bytes were served")
	}
}
