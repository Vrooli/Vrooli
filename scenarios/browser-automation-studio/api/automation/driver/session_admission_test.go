package driver

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestSessionAdmissionWaitsForCapacity(t *testing.T) {
	attempts := 0
	err := waitForSessionAdmission(context.Background(), time.Second, time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			return &Error{Status: http.StatusTooManyRequests, Message: "Maximum concurrent sessions reached: 10"}
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Fatalf("capacity recovery: attempts=%d err=%v", attempts, err)
	}
}
func TestSessionAdmissionNeverRetriesAmbiguousCreation(t *testing.T) {
	for _, failure := range []error{errors.New("connection lost after dispatch"), &Error{Status: 500, Message: "Maximum concurrent sessions reached"}, &Error{Status: 429, Message: "other limit"}} {
		attempts := 0
		err := waitForSessionAdmission(context.Background(), time.Second, time.Millisecond, func() error { attempts++; return failure })
		if err != failure || attempts != 1 {
			t.Fatalf("redispatched ambiguous creation: attempts=%d err=%v", attempts, err)
		}
	}
}
func TestSessionAdmissionCancellationAndBudget(t *testing.T) {
	rejection := &Error{Status: 429, Message: "Maximum concurrent sessions reached: 10"}
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	err := waitForSessionAdmission(ctx, time.Second, time.Second, func() error { attempts++; cancel(); return rejection })
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("cancellation: attempts=%d err=%v", attempts, err)
	}
	attempts = 0
	err = waitForSessionAdmission(context.Background(), 0, time.Millisecond, func() error { attempts++; return rejection })
	if err != rejection || attempts != 1 {
		t.Fatalf("budget: attempts=%d err=%v", attempts, err)
	}
}
