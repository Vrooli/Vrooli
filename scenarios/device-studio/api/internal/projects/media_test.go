package projects

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"reflect"
	"strings"
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

type commitThenFailMediaRepository struct{ SourceMediaRepository }

func (r commitThenFailMediaRepository) CreateSourceMedia(ctx context.Context, media SourceMedia) (SourceMedia, bool, error) {
	created, inserted, err := r.SourceMediaRepository.CreateSourceMedia(ctx, media)
	if err != nil {
		return created, inserted, err
	}
	return created, inserted, errors.New("simulated lost commit response")
}

func TestSourceMediaUncertainCommitRetainsReferencedObject(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	svc, repo, projectID := newMediaFixture(t, store, func() string { return "uncertain-commit" })
	svc.repo = commitThenFailMediaRepository{SourceMediaRepository: repo}
	source := grayPNG(t)
	_, err := svc.Import(context.Background(), projectID, "gray.png", int64(len(source)), bytes.NewReader(source))
	if err == nil || !strings.Contains(err.Error(), "uncertain") {
		t.Fatalf("commit response loss error = %v, want uncertainty", err)
	}
	row, err := repo.GetSourceMediaByDigest(context.Background(), projectID, digestBytes(source))
	if err != nil {
		t.Fatalf("committed row missing: %v", err)
	}
	reader, _, err := store.Get(context.Background(), row.storageKey)
	if err != nil {
		t.Fatalf("referenced object was deleted: %v", err)
	}
	got, err := io.ReadAll(reader)
	if closeErr := reader.Close(); err == nil {
		err = closeErr
	}
	if err != nil || !bytes.Equal(got, source) {
		t.Fatalf("retained object bytes = %d, err = %v", len(got), err)
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
	_, reader, err = svc.Open(context.Background(), projectID, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	served, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr == nil && closeErr == nil {
		t.Fatal("tampered bytes were served as verified")
	}
	if len(served) >= len(source) {
		t.Fatalf("tampered response exposed final byte before verification: served %d of %d", len(served), len(source))
	}
}

type countedReadCloser struct {
	reads  int
	closed bool
	data   *bytes.Reader
}

type transientEOFReadCloser struct {
	data      *bytes.Reader
	zeroAtEOF bool
	closed    bool
}

func (r *transientEOFReadCloser) Read(p []byte) (int, error) {
	if r.data.Len() == 0 && !r.zeroAtEOF {
		r.zeroAtEOF = true
		return 0, nil
	}
	return r.data.Read(p)
}

func (r *transientEOFReadCloser) Close() error { r.closed = true; return nil }

func TestVerifiedMediaReaderRetriesFinalEOFProbeAndSupportsSmallBuffers(t *testing.T) {
	for _, source := range [][]byte{[]byte("exact-sized-buffer"), []byte("x")} {
		t.Run(string(source), func(t *testing.T) {
			underlying := &transientEOFReadCloser{data: bytes.NewReader(source)}
			reader := &verifiedMediaReader{reader: underlying, expectedSize: int64(len(source)), expectedSHA: digestBytes(source), maxBytes: int64(len(source))}
			var got bytes.Buffer
			buf := make([]byte, len(source))
			for attempts := 0; attempts < len(source)+4; attempts++ {
				n, err := reader.Read(buf)
				if n > 0 {
					got.Write(buf[:n])
				}
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Fatalf("Read() error: %v", err)
				}
			}
			if !bytes.Equal(got.Bytes(), source) {
				t.Fatalf("Read() = %q, want %q", got.Bytes(), source)
			}
			if !underlying.zeroAtEOF {
				t.Fatal("reader did not exercise the transient (0, nil) final probe")
			}
			if err := reader.Close(); err != nil {
				t.Fatalf("Close() error after verification: %v", err)
			}
		})
	}
}

func (r *countedReadCloser) Read(p []byte) (int, error) {
	r.reads++
	return r.data.Read(p)
}

func (r *countedReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestVerifiedMediaReaderCloseDoesNotDrainUnconsumedBytes(t *testing.T) {
	source := bytes.Repeat([]byte("x"), 1<<20)
	underlying := &countedReadCloser{data: bytes.NewReader(source)}
	reader := &verifiedMediaReader{reader: underlying, expectedSize: int64(len(source)), expectedSHA: digestBytes(source), maxBytes: int64(len(source)), hash: sha256.New()}
	buf := make([]byte, 128)
	if _, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	}
	readsBeforeClose := underlying.reads
	if err := reader.Close(); !errors.Is(err, ErrSourceMediaUnverified) {
		t.Fatalf("partial close error = %v, want incomplete verification", err)
	}
	if underlying.reads != readsBeforeClose || !underlying.closed {
		t.Fatalf("Close drained input or failed to close: reads %d -> %d, closed=%v", readsBeforeClose, underlying.reads, underlying.closed)
	}
}

func mp4Header() []byte {
	return []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0}
}

func TestSourceMediaVideoImportIsPendingAndInspectionProjectionIsBoundAndIdempotent(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	svc, repo, projectID := newMediaFixture(t, store, func() string { return "video-" + string(rune('a'+next.Add(1))) })
	source := mp4Header()
	row, err := svc.Import(context.Background(), projectID, "meeting.mp4", -1, bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if row.Kind != MediaKindVideo || row.Readiness != ReadinessPending || row.RasterKnown || row.Width != 0 || row.Height != 0 || row.MediaType != "video/mp4" {
		t.Fatalf("video admission fabricated facts: %+v", row)
	}
	if _, err := svc.BindInspectionAttempt(context.Background(), projectID, row.ID, row.SHA256, "inspect-1", "", "attempt-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BindInspectionAttempt(context.Background(), projectID, row.ID, row.SHA256, "inspect-1", "", "attempt-1"); err != nil {
		t.Fatalf("same inspection attempt binding should be idempotent: %v", err)
	}
	if _, err := svc.BindInspectionAttempt(context.Background(), projectID, row.ID, row.SHA256, "inspect-1", "attempt-1", "attempt-2"); !errors.Is(err, ErrInspectionConflict) {
		t.Fatalf("attempt advanced without a projection state transition: %v", err)
	}
	receipt := &InspectionReceipt{Version: 1, ID: "receipt-1", SHA256: digestBytes([]byte("receipt")), SourceSHA256: row.SHA256, SourceSizeBytes: row.SizeBytes, JobID: "inspect-1", AttemptID: "attempt-1", ToolProfileID: "ffmpeg-profile", InspectionProtocolVersion: 1, Classification: "accepted", VideoStreamIndex: new(int64), VideoWidth: 1920, VideoHeight: 1080, RotationDegrees: 0, DurationNumerator: 60, DurationDenominator: 1, FrameCount: 60, StreamCount: 1, ClockSHA256: digestBytes([]byte("clock")), ClockArtifactRef: "clock-artifact-1"}
	projection := InspectionProjection{ProjectID: projectID, MediaID: row.ID, SourceSHA256: row.SHA256, JobID: "inspect-1", AttemptID: "attempt-1", Readiness: ReadinessReady, Receipt: receipt, ReceiptStorageKey: InspectionReceiptObjectKey(projectID, row.ID, "inspect-1", receipt.ID)}
	if err := svc.ProjectInspection(context.Background(), projection); err != nil {
		t.Fatal(err)
	}
	if err := svc.ProjectInspection(context.Background(), projection); err != nil {
		t.Fatalf("same projection should be idempotent: %v", err)
	}
	got, err := repo.GetSourceMedia(context.Background(), projectID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Readiness != ReadinessReady || !got.RasterKnown || got.Width != receipt.VideoWidth || got.Height != receipt.VideoHeight || got.InspectionReceipt == nil || !reflect.DeepEqual(got.InspectionReceipt, receipt) || got.inspectionReceiptKey != projection.ReceiptStorageKey {
		t.Fatalf("receipt projection did not roundtrip: %+v", got)
	}
	bad := projection
	bad.JobID = "different-job"
	if err := svc.ProjectInspection(context.Background(), bad); !errors.Is(err, ErrInspectionInvalid) {
		t.Fatalf("wrong job receipt was not rejected: %v", err)
	}
	bad = projection
	bad.AttemptID = "old-attempt"
	if err := svc.ProjectInspection(context.Background(), bad); !errors.Is(err, ErrInspectionConflict) {
		t.Fatalf("stale attempt projection was not fenced: %v", err)
	}
	bad = projection
	bad.ReceiptStorageKey = "projects/other/media/receipt.json"
	if err := svc.ProjectInspection(context.Background(), bad); !errors.Is(err, ErrInspectionInvalid) {
		t.Fatalf("noncanonical private receipt key was not rejected: %v", err)
	}
	bad = projection
	bad.Receipt = new(InspectionReceipt)
	*bad.Receipt = *projection.Receipt
	bad.Receipt.ID = ".."
	bad.ReceiptStorageKey = InspectionReceiptObjectKey(projectID, row.ID, "inspect-1", bad.Receipt.ID)
	if err := svc.ProjectInspection(context.Background(), bad); !errors.Is(err, ErrInspectionInvalid) {
		t.Fatalf("traversal receipt identity was not rejected: %v", err)
	}
}

func TestRejectedInspectionReceiptFinalizesWithoutAcceptedOnlyFacts(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	svc, repo, projectID := newMediaFixture(t, store, func() string { return "rejected-video" })
	row, err := svc.Import(context.Background(), projectID, "unsupported.mp4", -1, bytes.NewReader(mp4Header()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BindInspectionAttempt(context.Background(), projectID, row.ID, row.SHA256, "inspect-rejected", "", "attempt-rejected"); err != nil {
		t.Fatal(err)
	}
	receipt := &InspectionReceipt{
		Version: 1, ID: "rejected-receipt", SHA256: digestBytes([]byte("rejected receipt")),
		SourceSHA256: row.SHA256, SourceSizeBytes: row.SizeBytes, JobID: "inspect-rejected", AttemptID: "attempt-rejected",
		ToolProfileID: "ffmpeg-profile", InspectionProtocolVersion: 1, Classification: "rejected",
		IssueCode: "video_raster_unsupported", IssueMessage: "The source raster is outside the supported profile.",
	}
	projection := InspectionProjection{
		ProjectID: projectID, MediaID: row.ID, SourceSHA256: row.SHA256, JobID: receipt.JobID, AttemptID: receipt.AttemptID,
		Readiness: ReadinessRejected, IssueCode: receipt.IssueCode, IssueMessage: receipt.IssueMessage,
		Receipt: receipt, ReceiptStorageKey: InspectionReceiptObjectKey(projectID, row.ID, receipt.JobID, receipt.ID),
	}
	if err := svc.ProjectInspection(context.Background(), projection); err != nil {
		t.Fatalf("valid worker rejection did not finalize: %v", err)
	}
	if err := svc.ProjectInspection(context.Background(), projection); err != nil {
		t.Fatalf("rejected finalization retry was not idempotent: %v", err)
	}
	got, err := repo.GetSourceMedia(context.Background(), projectID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Readiness != ReadinessRejected || got.InspectionReceipt == nil || got.InspectionReceipt.Classification != "rejected" || got.InspectionIssueCode != projection.IssueCode {
		t.Fatalf("rejected receipt projection was not retained: %+v", got)
	}

	wrongReady := projection
	wrongReady.Readiness = ReadinessReady
	wrongReady.IssueCode, wrongReady.IssueMessage = "", ""
	if err := svc.ProjectInspection(context.Background(), wrongReady); !errors.Is(err, ErrInspectionInvalid) {
		t.Fatalf("rejected classification was accepted as ready: %v", err)
	}
	fabricated := projection
	copyReceipt := *receipt
	copyReceipt.FrameCount = 1
	fabricated.Receipt = &copyReceipt
	if err := svc.ProjectInspection(context.Background(), fabricated); !errors.Is(err, ErrInspectionInvalid) {
		t.Fatalf("rejected receipt with fabricated accepted facts was accepted: %v", err)
	}
}

func TestKnownInspectionExecutionTerminalIsRetryableWithoutFabricatedReceipt(t *testing.T) {
	for _, state := range []string{"failed", "cancelled"} {
		t.Run(state, func(t *testing.T) {
			store := blobstore.NewMemoryBlobStore()
			svc, repo, projectID := newMediaFixture(t, store, func() string { return "terminal-" + state })
			row, err := svc.Import(context.Background(), projectID, "clip.mp4", -1, bytes.NewReader(mp4Header()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.BindInspectionAttempt(context.Background(), projectID, row.ID, row.SHA256, "inspect-job", "", "attempt-1"); err != nil {
				t.Fatal(err)
			}
			issueCode := "inspection_execution_" + state
			projection := InspectionProjection{ProjectID: projectID, MediaID: row.ID, SourceSHA256: row.SHA256, JobID: "inspect-job", AttemptID: "attempt-1", Readiness: ReadinessRejected, IssueCode: issueCode, IssueMessage: "The inspection operation reached a known terminal state; retry is available."}
			if err := svc.ProjectInspection(context.Background(), projection); err != nil {
				t.Fatalf("known terminal outcome did not project as retryable rejection: %v", err)
			}
			got, err := repo.GetSourceMedia(context.Background(), projectID, row.ID)
			if err != nil || got.Readiness != ReadinessRejected || got.InspectionReceipt != nil || got.InspectionIssueCode != issueCode || got.inspectionReceiptKey != "" {
				t.Fatalf("known terminal projection fabricated/omitted facts: media=%+v err=%v", got, err)
			}
			if err := svc.ProjectInspection(context.Background(), InspectionProjection{ProjectID: projectID, MediaID: row.ID, SourceSHA256: row.SHA256, JobID: "inspect-job", AttemptID: "attempt-1", Readiness: ReadinessRejected, IssueCode: "worker_rejected", IssueMessage: "not a known execution issue"}); !errors.Is(err, ErrInspectionInvalid) {
				t.Fatalf("receipt-free worker rejection was accepted: %v", err)
			}
			if _, err := svc.BindInspectionAttempt(context.Background(), projectID, row.ID, row.SHA256, "inspect-job", "attempt-1", "attempt-2"); err != nil {
				t.Fatalf("known failed/cancelled inspection could not be retried: %v", err)
			}
			got, err = repo.GetSourceMedia(context.Background(), projectID, row.ID)
			if err != nil || got.Readiness != ReadinessPending || got.InspectionAttemptID != "attempt-2" || got.InspectionIssueCode != "" {
				t.Fatalf("retry did not clear the terminal rejection: media=%+v err=%v", got, err)
			}
		})
	}
}

func TestSourceMediaSchemaRejectsNullOrMixedRasterDimensions(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	var next atomic.Int32
	svc, repo, projectID := newMediaFixture(t, store, func() string { return "schema-" + string(rune('a'+next.Add(1))) })
	imageRow, err := svc.Import(context.Background(), projectID, "mask.png", -1, bytes.NewReader(grayPNG(t)))
	if err != nil {
		t.Fatal(err)
	}
	videoRow, err := svc.Import(context.Background(), projectID, "clip.mp4", -1, bytes.NewReader(mp4Header()))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		id   string
		stmt string
	}{
		{name: "ready image null pair", id: imageRow.ID, stmt: `UPDATE project_source_media SET width=NULL,height=NULL WHERE id=?`},
		{name: "ready image mixed pair", id: imageRow.ID, stmt: `UPDATE project_source_media SET width=5,height=NULL WHERE id=?`},
		{name: "ready video null width", id: videoRow.ID, stmt: `UPDATE project_source_media SET readiness='ready',width=NULL,height=2,inspection_receipt_json='{}' WHERE id=?`},
		{name: "ready video mixed pair", id: videoRow.ID, stmt: `UPDATE project_source_media SET readiness='ready',width=5,height=NULL,inspection_receipt_json='{}' WHERE id=?`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := repo.db.ExecContext(context.Background(), tc.stmt, tc.id); err == nil {
				t.Fatal("schema accepted a ready media row with missing or mixed raster dimensions")
			}
		})
	}
}

func TestSourceMediaVideoProjectionCannotClaimWrongSourceSize(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	svc, _, projectID := newMediaFixture(t, store, func() string { return "video-x" })
	row, err := svc.Import(context.Background(), projectID, "clip.mp4", -1, bytes.NewReader(mp4Header()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.BindInspectionJob(context.Background(), projectID, row.ID, row.SHA256, "job")
	if err != nil {
		t.Fatal(err)
	}
	receipt := &InspectionReceipt{Version: 1, ID: "r", SHA256: digestBytes([]byte("r")), SourceSHA256: row.SHA256, SourceSizeBytes: row.SizeBytes + 1, JobID: "job", AttemptID: "a", ToolProfileID: "profile", InspectionProtocolVersion: 1, Classification: "accepted", VideoStreamIndex: new(int64), VideoWidth: 1, VideoHeight: 1, DurationNumerator: 1, DurationDenominator: 1, FrameCount: 1, StreamCount: 1, ClockSHA256: digestBytes([]byte("c")), ClockArtifactRef: "clock"}
	err = svc.ProjectInspection(context.Background(), InspectionProjection{ProjectID: projectID, MediaID: row.ID, SourceSHA256: row.SHA256, JobID: "job", Readiness: ReadinessReady, Receipt: receipt, ReceiptStorageKey: InspectionReceiptObjectKey(projectID, row.ID, "job", receipt.ID)})
	if !errors.Is(err, ErrInspectionInvalid) {
		t.Fatalf("wrong source size accepted: %v", err)
	}
}

func TestVideoStreamEnforcesObservedAndDeclaredSizesAndCleansStaging(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	var ids atomic.Int32
	svc, _, projectID := newMediaFixture(t, store, func() string { return fmt.Sprintf("bounded-%d", ids.Add(1)) })
	svc.limits.MaxVideoBytes = 32
	source := mp4Header()
	first, err := svc.Import(context.Background(), projectID, "clip.mp4", -1, bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := svc.Import(context.Background(), projectID, "again.mp4", int64(len(source)), bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != first.ID {
		t.Fatal("video duplicate did not converge")
	}
	tooLarge := append(append([]byte(nil), source...), make([]byte, 17)...)
	_, err = svc.Import(context.Background(), projectID, "large.mp4", -1, bytes.NewReader(tooLarge))
	var limit ErrSourceMediaTooLarge
	if !errors.As(err, &limit) {
		t.Fatalf("stream limit error=%v", err)
	}
	_, err = svc.Import(context.Background(), projectID, "wrong-length.mp4", int64(len(source)-1), bytes.NewReader(source))
	var invalid ErrInvalidSourceMedia
	if !errors.As(err, &invalid) {
		t.Fatalf("declared-size mismatch error=%v", err)
	}
	if _, _, err := store.Get(context.Background(), "projects/"+projectID+"/media/bounded-6"); err == nil {
		t.Fatal("rejected stream left its operation-owned staging object")
	}
}

func TestVideoDeclaredOverMaximumFailsBeforeReading(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	svc, _, projectID := newMediaFixture(t, store, func() string { return "large-declared" })
	reader := &countingReader{data: bytes.NewReader(mp4Header())}
	_, err := svc.Import(context.Background(), projectID, "clip.mp4", MaxSourceVideoBytes+1, reader)
	var limit ErrSourceMediaTooLarge
	if !errors.As(err, &limit) {
		t.Fatalf("size error=%v", err)
	}
	if reader.reads.Load() != 0 {
		t.Fatalf("oversized declaration read source %d times", reader.reads.Load())
	}
}

func TestAudioInspectionProjectionDistinguishesLegacyZeroAndDetected(t *testing.T) { // [REQ:DS-SCREEN-AUDIO]
	for _, tc := range []struct {
		name    string
		version uint32
		audio   *AudioInspectionFacts
	}{
		{"legacy", 1, nil},
		{"zero", 2, &AudioInspectionFacts{Version: 1, Streams: []AudioStreamFact{}}},
		{"detected", 2, &AudioInspectionFacts{Version: 1, Streams: []AudioStreamFact{{OriginalStreamIndex: 1, Codec: "pcm_s16le"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc, repo, projectID := newMediaFixture(t, blobstore.NewMemoryBlobStore(), func() string { return "audio-source" })
			source := mp4Header()
			row, err := svc.Import(ctx, projectID, "original.mp4", -1, bytes.NewReader(source))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = svc.BindInspectionAttempt(ctx, projectID, row.ID, row.SHA256, "job", "", "attempt"); err != nil {
				t.Fatal(err)
			}
			videoIndex := int64(0)
			receipt := &InspectionReceipt{Version: tc.version, Audio: tc.audio, ID: "receipt", SHA256: digestBytes([]byte("receipt")), SourceSHA256: row.SHA256, SourceSizeBytes: row.SizeBytes, JobID: "job", AttemptID: "attempt", ToolProfileID: "profile", InspectionProtocolVersion: 1, Classification: "accepted", VideoStreamIndex: &videoIndex, VideoWidth: 430, VideoHeight: 880, DurationNumerator: 2, DurationDenominator: 1, FrameCount: 60, StreamCount: 2, ClockSHA256: digestBytes([]byte("clock")), ClockArtifactRef: "clock"}
			projection := InspectionProjection{ProjectID: projectID, MediaID: row.ID, SourceSHA256: row.SHA256, JobID: "job", AttemptID: "attempt", Readiness: ReadinessReady, Receipt: receipt, ReceiptStorageKey: InspectionReceiptObjectKey(projectID, row.ID, "job", "receipt")}
			if err = svc.ProjectInspection(ctx, projection); err != nil {
				t.Fatal(err)
			}
			if err = svc.ProjectInspection(ctx, projection); err != nil {
				t.Fatalf("replaying identical audio facts must be idempotent: %v", err)
			}
			got, err := repo.GetSourceMedia(ctx, projectID, row.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.InspectionReceipt.Audio, tc.audio) {
				t.Fatalf("audio facts lost through JSON storage: %+v", got.InspectionReceipt.Audio)
			}
			if got.SHA256 != row.SHA256 || got.SizeBytes != row.SizeBytes {
				t.Fatal("inspection changed original source identity")
			}
			for name, audio := range map[string]*AudioInspectionFacts{"missing": nil, "null-list": {Version: 1}, "version": {Version: 2, Streams: []AudioStreamFact{}}, "duplicate": {Version: 1, Streams: []AudioStreamFact{{1, "aac"}, {1, "aac"}}}, "video-index": {Version: 1, Streams: []AudioStreamFact{{0, "aac"}}}, "outside": {Version: 1, Streams: []AudioStreamFact{{2, "aac"}}}, "empty-codec": {Version: 1, Streams: []AudioStreamFact{{1, ""}}}, "invalid-codec": {Version: 1, Streams: []AudioStreamFact{{1, "AAC/evil"}}}, "long-codec": {Version: 1, Streams: []AudioStreamFact{{1, strings.Repeat("a", 65)}}}} {
				bad := *receipt
				bad.Version = 2
				bad.Audio = audio
				if err := validateInspectionReceipt(bad, projection, ReadinessReady); err == nil {
					t.Errorf("accepted malformed %s", name)
				}
			}
			for _, index := range []*int64{nil, func() *int64 { v := int64(2); return &v }()} {
				bad := *receipt
				bad.VideoStreamIndex = index
				if err := validateInspectionReceipt(bad, projection, ReadinessReady); err == nil {
					t.Fatal("accepted missing or out-of-range selected video index")
				}
			}
			legacyWithFacts := *receipt
			legacyWithFacts.Version = 1
			legacyWithFacts.Audio = &AudioInspectionFacts{Version: 1, Streams: []AudioStreamFact{}}
			if err := validateInspectionReceipt(legacyWithFacts, projection, ReadinessReady); err == nil {
				t.Fatal("legacy receipt accepted audio facts")
			}
		})
	}
}
