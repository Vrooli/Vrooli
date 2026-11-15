package conversations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

const (
	// uploadTimeout bounds one Stat or Put request. It must cover a 64 MiB blob
	// over a slow link; the scenario's short default timeout would not.
	uploadTimeout = 5 * time.Minute
	// uploadConcurrency is how many blobs travel at once.
	uploadConcurrency = 4
	// uploadInFlightBytes caps the file bytes held in memory at once. A blob
	// travels as one message, so concurrency alone could hold 4 x 64 MiB.
	uploadInFlightBytes int64 = 128 << 20
	// statBatch is how many hashes one StatAttachmentBlobs call carries.
	statBatch = 500
)

// uploader sends attachment bytes to the Hub. Callers with inventory access
// first check which hashes are missing; other callers upload by hash directly.
// Put is idempotent either way.
type uploader struct {
	client   connectv1.ConversationsServiceClient
	progress io.Writer
	// retryDelay replaces the Hub's Retry-After pause between attempts when
	// set; tests use it to stay fast.
	retryDelay time.Duration
}

// upload makes sure the Hub holds every file. Identical content is sent once.
func (u uploader) upload(ctx context.Context, files []localFile) error {
	byHash := make(map[string]localFile, len(files))
	var hashes []string
	for _, file := range files {
		if _, seen := byHash[file.SHA256]; !seen {
			byHash[file.SHA256] = file
			hashes = append(hashes, file.SHA256)
		}
	}
	if len(hashes) == 0 {
		return nil
	}
	missing, err := u.stat(ctx, hashes)
	if err != nil {
		if !cannotInspectInventory(err) {
			return err
		}
		// A sender may write packet content without learning which other callers'
		// content hashes the Hub already holds.
		missing = hashes
	}
	var toSend []localFile
	var bytes int64
	for _, hash := range missing {
		if file, ok := byHash[hash]; ok {
			toSend = append(toSend, file)
			bytes += file.Size
			delete(byHash, hash) // a hash the Hub lists twice still uploads once
		}
	}
	if len(toSend) == 0 {
		return nil
	}
	if u.progress != nil {
		fmt.Fprintf(u.progress, "Uploading %d attachment file(s), %s...\n", len(toSend), formatBytes(bytes))
	}
	return u.put(ctx, toSend)
}

// stat returns the hashes the Hub does not hold, in request order.
func (u uploader) stat(ctx context.Context, hashes []string) ([]string, error) {
	var missing []string
	for start := 0; start < len(hashes); start += statBatch {
		end := min(start+statBatch, len(hashes))
		resp, err := withRetry(ctx, u.retryDelay, func() (*connect.Response[v1.StatAttachmentBlobsResponse], error) {
			return u.client.StatAttachmentBlobs(ctx, connect.NewRequest(&v1.StatAttachmentBlobsRequest{Sha256: hashes[start:end]}))
		})
		if err != nil {
			if cannotInspectInventory(err) {
				return nil, err
			}
			return nil, uploadError("check which attachments the hub already holds", err)
		}
		missing = append(missing, resp.Msg.GetMissing()...)
	}
	return missing, nil
}

func cannotInspectInventory(err error) bool {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return false
	}
	return connectErr.Code() == connect.CodeUnauthenticated || connectErr.Code() == connect.CodePermissionDenied
}

// put stores each file with bounded concurrency and memory. The first failure
// stops the rest.
func (u uploader) put(ctx context.Context, files []localFile) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	budget := newByteBudget(uploadInFlightBytes)
	go func() {
		<-ctx.Done()
		budget.wake()
	}()

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		slots    = make(chan struct{}, uploadConcurrency)
	)
	fail := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
			cancel()
		}
		mu.Unlock()
	}
	for _, file := range files {
		if ctx.Err() != nil {
			break
		}
		slots <- struct{}{}
		if err := budget.acquire(ctx, file.Size); err != nil {
			<-slots
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			defer budget.release(file.Size)
			if err := u.putOne(ctx, file); err != nil {
				fail(err)
			}
		}()
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

// putOne re-reads the file, confirms it is still the file that was hashed, and
// stores it.
func (u uploader) putOne(ctx context.Context, file localFile) error {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return fmt.Errorf("read %s for upload: %w", file.Path, unwrapPathError(err))
	}
	if int64(len(content)) != file.Size || sha256Hex(content) != file.SHA256 {
		return fmt.Errorf("%s changed while the ask was being prepared; run the command again", file.Path)
	}
	_, err = withRetry(ctx, u.retryDelay, func() (*connect.Response[v1.PutAttachmentBlobResponse], error) {
		return u.client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{
			Sha256:    file.SHA256,
			MediaType: file.MediaType,
			Content:   content,
		}))
	})
	if err != nil {
		return uploadError("upload attachment "+file.Path, err)
	}
	return nil
}

const (
	// uploadAttempts is how many times a busy Hub (Unavailable) is tried.
	uploadAttempts = 3
	// defaultRetryPause is the wait when the Hub names none; maxRetryPause caps
	// what it may ask for.
	defaultRetryPause = 5 * time.Second
	maxRetryPause     = 30 * time.Second
)

// withRetry repeats a call while the Hub answers Unavailable, which it does
// with a Retry-After header when its upload slots stay busy. Any other result
// is final. pause overrides the Retry-After wait when positive.
func withRetry[T any](ctx context.Context, pause time.Duration, call func() (*connect.Response[T], error)) (*connect.Response[T], error) {
	for attempt := 1; ; attempt++ {
		resp, err := call()
		var connectErr *connect.Error
		if err == nil || !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeUnavailable || attempt >= uploadAttempts {
			return resp, err
		}
		wait := pause
		if wait <= 0 {
			wait = retryAfter(connectErr)
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// retryAfter reads the Hub's Retry-After seconds, within sane bounds.
func retryAfter(err *connect.Error) time.Duration {
	seconds, convErr := strconv.Atoi(strings.TrimSpace(err.Meta().Get("Retry-After")))
	if convErr != nil || seconds <= 0 {
		return defaultRetryPause
	}
	return min(time.Duration(seconds)*time.Second, maxRetryPause)
}

// uploadError says in plain words why the Hub refused an attachment transfer.
func uploadError(action string, err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnauthenticated:
			return fmt.Errorf("%s: the Hub refused this attachment operation: %w", action, err)
		case connect.CodeResourceExhausted:
			return fmt.Errorf("%s: the Hub already holds its limit of uploads that no ask references yet (they expire after 24 hours); send fewer or smaller files, or try again later: %w", action, err)
		case connect.CodeUnavailable:
			return fmt.Errorf("%s: the Hub's upload slots stayed busy after %d tries; try again shortly: %w", action, uploadAttempts, err)
		}
	}
	return cliapp.WrapAPIError(action, err, nil)
}

// byteBudget limits the bytes in flight. One request larger than the limit is
// still admitted when nothing else is in flight, so it can never starve.
type byteBudget struct {
	mu    sync.Mutex
	cond  *sync.Cond
	inUse int64
	limit int64
}

func newByteBudget(limit int64) *byteBudget {
	b := &byteBudget{limit: limit}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *byteBudget) acquire(ctx context.Context, n int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for b.inUse > 0 && b.inUse+n > b.limit {
		if err := ctx.Err(); err != nil {
			return err
		}
		b.cond.Wait()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b.inUse += n
	return nil
}

func (b *byteBudget) release(n int64) {
	b.mu.Lock()
	b.inUse -= n
	b.mu.Unlock()
	b.cond.Broadcast()
}

// wake lets blocked acquirers re-check their context. Taking the lock first
// closes the window between a waiter's check and its Wait.
func (b *byteBudget) wake() {
	b.mu.Lock()
	b.cond.Broadcast()
	b.mu.Unlock()
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
