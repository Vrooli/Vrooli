package transcriptindex

import (
	"context"
	"errors"
	"testing"
)

func TestRetryCommitRetriesContentionAndStopsOnOtherErrors(t *testing.T) {
	attempts := 0
	err := retryCommit(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New("database is locked (5) (SQLITE_BUSY)")
		}
		return nil
	})
	if err != nil || attempts != 2 {
		t.Fatalf("contention retry err=%v attempts=%d", err, attempts)
	}

	attempts = 0
	refused := errors.New("capture intent changed before commit")
	if err := retryCommit(context.Background(), func() error { attempts++; return refused }); !errors.Is(err, refused) || attempts != 1 {
		t.Fatalf("non-contention error err=%v attempts=%d", err, attempts)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	attempts = 0
	if err := retryCommit(cancelled, func() error { attempts++; return errors.New("SQLITE_BUSY") }); err == nil || attempts != 1 {
		t.Fatalf("cancelled retry err=%v attempts=%d", err, attempts)
	}
}
