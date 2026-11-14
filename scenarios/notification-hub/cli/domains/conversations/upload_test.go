package conversations

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func filesOf(t *testing.T, dir string, contents ...string) []localFile {
	t.Helper()
	var files []localFile
	for i, content := range contents {
		file, err := describeFile(writeFile(t, dir, "f"+strconv.Itoa(i), content), maxAttachmentTotalBytes)
		require.NoError(t, err)
		files = append(files, file)
	}
	return files
}

func TestUploadStatsThenPutsOnlyMissingBlobs(t *testing.T) {
	files := filesOf(t, t.TempDir(), "one", "two", "three")
	stub := &hubStub{held: map[string]bool{files[1].SHA256: true}}
	svc := newTestService(t, stub)

	require.NoError(t, uploader{client: svc.client}.upload(context.Background(), files))
	require.Len(t, stub.statCalls, 1)
	require.Equal(t, []string{files[0].SHA256, files[1].SHA256, files[2].SHA256}, stub.statCalls[0])
	var put []string
	for _, p := range stub.puts {
		put = append(put, string(p.GetContent()))
		require.NotEmpty(t, p.GetMediaType())
	}
	require.ElementsMatch(t, []string{"one", "three"}, put)
}

func TestUploadSendsIdenticalContentOnce(t *testing.T) {
	files := filesOf(t, t.TempDir(), "same", "same", "same")
	stub := &hubStub{}
	svc := newTestService(t, stub)
	require.NoError(t, uploader{client: svc.client}.upload(context.Background(), files))
	require.Len(t, stub.statCalls[0], 1)
	require.Len(t, stub.puts, 1)
}

func TestUploadMakesNoPutWhenTheHubHoldsEverything(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a", "b")
	stub := &hubStub{held: map[string]bool{files[0].SHA256: true, files[1].SHA256: true}}
	svc := newTestService(t, stub)
	var progress bytes.Buffer
	require.NoError(t, uploader{client: svc.client, progress: &progress}.upload(context.Background(), files))
	require.Empty(t, stub.puts)
	require.Empty(t, progress.String(), "nothing to upload, nothing to say")
}

func TestUploadBatchesLargeStatRequests(t *testing.T) {
	dir := t.TempDir()
	var files []localFile
	for i := 0; i < statBatch+5; i++ {
		files = append(files, localFile{Path: dir, SHA256: "h" + strconv.Itoa(i), Size: 1})
	}
	stub := &hubStub{held: map[string]bool{}}
	for _, f := range files {
		stub.held[f.SHA256] = true
	}
	svc := newTestService(t, stub)
	require.NoError(t, uploader{client: svc.client}.upload(context.Background(), files))
	require.Len(t, stub.statCalls, 2)
	require.Len(t, stub.statCalls[0], statBatch)
	require.Len(t, stub.statCalls[1], 5)
}

func TestUploadBoundsConcurrency(t *testing.T) {
	contents := make([]string, 16)
	for i := range contents {
		contents[i] = "blob-" + strconv.Itoa(i)
	}
	files := filesOf(t, t.TempDir(), contents...)
	stub := &hubStub{}
	svc := newTestService(t, stub)
	require.NoError(t, uploader{client: svc.client}.upload(context.Background(), files))
	require.Len(t, stub.puts, 16)
	require.LessOrEqual(t, stub.maxInFlight, uploadConcurrency)
}

func TestUploadStopsAtTheFirstFailureAndNamesTheFile(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a", "b", "c", "d", "e", "f", "g", "h")
	stub := &hubStub{putErr: connectError("disk full")}
	svc := newTestService(t, stub)
	err := uploader{client: svc.client}.upload(context.Background(), files)
	require.ErrorContains(t, err, "upload attachment")
	require.ErrorContains(t, err, "disk full")
}

func TestUploadRefusesAFileThatChangedAfterItWasHashed(t *testing.T) {
	dir := t.TempDir()
	files := filesOf(t, dir, "before")
	require.NoError(t, os.WriteFile(files[0].Path, []byte("after!"), 0o644))
	stub := &hubStub{}
	svc := newTestService(t, stub)
	err := uploader{client: svc.client}.upload(context.Background(), files)
	require.ErrorContains(t, err, "changed while the ask was being prepared")
	require.Empty(t, stub.puts, "bytes that do not match their hash must never be sent")
}

func TestUploadHonorsACanceledContext(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a", "b")
	svc := newTestService(t, &hubStub{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, uploader{client: svc.client}.upload(ctx, files))
}

func TestByteBudgetAdmitsOversizeRequestsAloneAndReleasesWaiters(t *testing.T) {
	budget := newByteBudget(10)
	ctx := context.Background()
	require.NoError(t, budget.acquire(ctx, 100), "a request over the limit runs when nothing else does")

	admitted := make(chan struct{})
	go func() {
		require.NoError(t, budget.acquire(ctx, 5))
		close(admitted)
	}()
	select {
	case <-admitted:
		t.Fatal("admitted while the budget was exhausted")
	case <-time.After(50 * time.Millisecond):
	}
	budget.release(100)
	select {
	case <-admitted:
	case <-time.After(2 * time.Second):
		t.Fatal("waiter was not released")
	}
}

func TestByteBudgetWaiterWakesOnCancel(t *testing.T) {
	budget := newByteBudget(10)
	require.NoError(t, budget.acquire(context.Background(), 10))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	var got error
	go func() {
		defer wg.Done()
		got = budget.acquire(ctx, 5)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	budget.wake()
	wg.Wait()
	require.ErrorIs(t, got, context.Canceled)
}

func busyError(retryAfter string) *connect.Error {
	err := connect.NewError(connect.CodeUnavailable, fmt.Errorf("upload slots busy"))
	if retryAfter != "" {
		err.Meta().Set("Retry-After", retryAfter)
	}
	return err
}

func TestUploadExplainsThatAttachmentsNeedAVerifiedSender(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a")
	stub := &hubStub{statErr: connect.NewError(connect.CodeUnauthenticated, nil)}
	svc := newTestService(t, stub)
	err := uploader{client: svc.client}.upload(context.Background(), files)
	require.ErrorContains(t, err, "attachments need a verified sender")
	require.ErrorContains(t, err, "agent run")

	stub = &hubStub{putErr: connect.NewError(connect.CodeUnauthenticated, nil)}
	svc = newTestService(t, stub)
	err = uploader{client: svc.client}.upload(context.Background(), files)
	require.ErrorContains(t, err, "attachments need a verified sender")
}

func TestUploadExplainsAFullUploadQuota(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a")
	svc := newTestService(t, &hubStub{putErr: connect.NewError(connect.CodeResourceExhausted, nil)})
	err := uploader{client: svc.client}.upload(context.Background(), files)
	require.ErrorContains(t, err, "no ask references yet")
	require.ErrorContains(t, err, "24 hours")
}

func TestUploadRetriesAHubWhoseUploadSlotsAreBusy(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a")
	stub := &hubStub{putSeq: []error{busyError("1"), busyError("1"), nil}}
	svc := newTestService(t, stub)
	require.NoError(t, uploader{client: svc.client, retryDelay: time.Millisecond}.upload(context.Background(), files))
	require.Equal(t, 3, stub.putCalls, "two busy answers, then the upload lands")
	require.Len(t, stub.puts, 1)
}

func TestUploadGivesUpAfterThreeBusyAnswers(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a")
	stub := &hubStub{putSeq: []error{busyError(""), busyError(""), busyError(""), busyError("")}}
	svc := newTestService(t, stub)
	err := uploader{client: svc.client, retryDelay: time.Millisecond}.upload(context.Background(), files)
	require.ErrorContains(t, err, "upload slots stayed busy after 3 tries")
	require.Equal(t, uploadAttempts, stub.putCalls)
}

func TestUploadDoesNotRetryOtherFailures(t *testing.T) {
	files := filesOf(t, t.TempDir(), "a")
	stub := &hubStub{putSeq: []error{connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("hash mismatch"))}}
	svc := newTestService(t, stub)
	err := uploader{client: svc.client, retryDelay: time.Millisecond}.upload(context.Background(), files)
	require.ErrorContains(t, err, "hash mismatch")
	require.Equal(t, 1, stub.putCalls)
}

func TestRetryAfterReadsTheHubsHeaderWithinBounds(t *testing.T) {
	require.Equal(t, 7*time.Second, retryAfter(busyError("7")))
	require.Equal(t, defaultRetryPause, retryAfter(busyError("")))
	require.Equal(t, defaultRetryPause, retryAfter(busyError("soon")))
	require.Equal(t, defaultRetryPause, retryAfter(busyError("-3")))
	require.Equal(t, maxRetryPause, retryAfter(busyError("3600")))
}

func TestRetryStopsWhenTheContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	_, err := withRetry(ctx, time.Hour, func() (*connect.Response[v1.PutAttachmentBlobResponse], error) {
		calls++
		return nil, busyError("")
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
}
