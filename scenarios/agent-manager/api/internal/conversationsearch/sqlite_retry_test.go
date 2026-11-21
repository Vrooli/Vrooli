package conversationsearch

import (
	"context"
	"errors"
	"testing"
)

func TestRetryContentionRetriesOnlyTransientSQLiteErrors(t *testing.T) {
	attempts := 0
	err := retryContention(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("database is locked (5) (SQLITE_BUSY)")
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Fatalf("contention retry err=%v attempts=%d", err, attempts)
	}

	attempts = 0
	refused := errors.New("unsupported staged change")
	if err := retryContention(context.Background(), func() error { attempts++; return refused }); !errors.Is(err, refused) || attempts != 1 {
		t.Fatalf("non-contention error err=%v attempts=%d", err, attempts)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	attempts = 0
	if err := retryContention(cancelled, func() error { attempts++; return errors.New("SQLITE_BUSY") }); err == nil || attempts != 1 {
		t.Fatalf("cancelled retry err=%v attempts=%d", err, attempts)
	}
}

func TestRetryContentionBoundsTheAttempts(t *testing.T) {
	attempts := 0
	err := retryContention(context.Background(), func() error {
		attempts++
		return errors.New("SQLITE_BUSY")
	})
	if err == nil || attempts != 3 {
		t.Fatalf("bounded retry err=%v attempts=%d", err, attempts)
	}
}
