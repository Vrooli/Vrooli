package hub

import (
	"bytes"
	"context"
	"log"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClose_StopsTheWorkerPromptlyAndIsIdempotent(t *testing.T) {
	db := OpenTestDB(t, "hub.db")
	t.Cleanup(func() { _ = db.Close() })
	service := New(db, nil, log.New(&bytes.Buffer{}, "", 0))

	closed := make(chan struct{})
	go func() {
		service.Close()
		service.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return: the worker never observed it")
	}
	select {
	case <-service.life.done:
	default:
		t.Fatal("the worker is still running after Close")
	}
}

func TestDispatchAfterCloseDoesNotLeakASender(t *testing.T) {
	db := OpenTestDB(t, "hub.db")
	t.Cleanup(func() { _ = db.Close() })
	service := New(db, nil, log.New(&bytes.Buffer{}, "", 0))
	service.Close()
	baseline := runtime.NumGoroutine()

	// With the worker gone the queue fills and every further dispatch parks a
	// sender goroutine; they must end with the service, not wait forever.
	for i := 0; i < cap(service.worker)+25; i++ {
		service.dispatch(context.Background(), "never-processed")
	}
	// Eventually runs its condition on goroutines of its own, hence the slack;
	// 25 parked senders would still be far above it.
	const slack = 3
	require.Eventually(t, func() bool { return runtime.NumGoroutine() <= baseline+slack }, 2*time.Second, 10*time.Millisecond,
		"%d goroutines, baseline %d: overflow senders are still blocked", runtime.NumGoroutine(), baseline)
}

// The maintenance pass runs on a timer with nobody to report to; a database
// closed underneath it must cost one logged step, never the process.
// lockedBuffer is a log sink the service's worker goroutine and the test can
// both touch; a bytes.Buffer is not safe for that.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestMaintain_AClosedDatabaseIsLoggedNotFatal(t *testing.T) {
	db := OpenTestDB(t, "hub.db")
	var logs lockedBuffer
	service := New(db, nil, log.New(&logs, "", 0))
	t.Cleanup(service.Close)
	require.NoError(t, db.Close())
	logs.Reset()

	require.NotPanics(t, func() { service.maintain(context.Background()) })
	require.Contains(t, logs.String(), "recovered from", "the failed step is reported")
}

func TestMaintain_IsSkippedOnceClosing(t *testing.T) {
	db := OpenTestDB(t, "hub.db")
	var logs lockedBuffer
	service := New(db, nil, log.New(&logs, "", 0))
	service.Close()
	require.NoError(t, db.Close())
	logs.Reset() // New logged migrations against the empty database

	service.maintain(context.Background())
	require.Empty(t, logs.String(), "a closed service touches nothing")
}
