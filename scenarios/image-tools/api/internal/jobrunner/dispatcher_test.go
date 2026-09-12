package jobrunner

import (
	"context"
	"errors"
	"testing"

	internaljobs "image-tools/internal/jobs"
)

func TestDispatcherUnregisteredOperationFailsCleanly(t *testing.T) {
	d := New()
	_, err := d.Run(context.Background(), internaljobs.Job{Operation: "upscale"}, func(int, string) {})
	if !errors.Is(err, ErrNoRunner) {
		t.Fatalf("expected ErrNoRunner, got %v", err)
	}
}

func TestDispatcherRoutesToRegisteredHandler(t *testing.T) {
	d := New()
	d.Register("resize", func(_ context.Context, job internaljobs.Job, emit func(int, string)) (internaljobs.Result, error) {
		emit(50, "halfway")
		return internaljobs.Result{Ref: "out/" + job.ID}, nil
	})

	var gotProgress int
	result, err := d.Run(context.Background(), internaljobs.Job{ID: "j1", Operation: "resize"}, func(p int, _ string) {
		gotProgress = p
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ref := result.Ref; ref != "out/j1" {
		t.Fatalf("ref = %q, want out/j1", ref)
	}
	if gotProgress != 50 {
		t.Fatalf("progress = %d, want 50", gotProgress)
	}
}

func TestDispatcherDuplicateRegistrationPanics(t *testing.T) {
	d := New()
	d.Register("resize", func(context.Context, internaljobs.Job, func(int, string)) (internaljobs.Result, error) {
		return internaljobs.Result{}, nil
	})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	d.Register("resize", func(context.Context, internaljobs.Job, func(int, string)) (internaljobs.Result, error) {
		return internaljobs.Result{}, nil
	})
}
