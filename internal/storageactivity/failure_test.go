package storageactivity

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestFailureDetailPreservesOnlyProcessStageAndCause(t *testing.T) {
	err := &os.PathError{Op: "open", Path: "/proc/123/environ", Err: os.ErrPermission}
	if got := FailureDetail(err); got != "process 123 environ: permission denied" {
		t.Fatalf("detail=%q", got)
	}
	for _, err := range []error{nil, errors.New("secret environment contents"), &os.PathError{Op: "open", Path: "/home/owner/private", Err: os.ErrPermission}} {
		if got := FailureDetail(err); got != "process observation incomplete" {
			t.Fatalf("private detail escaped: %q", got)
		}
	}
	if got := FailureDetail(context.DeadlineExceeded); got != "process observation timed out" {
		t.Fatal(got)
	}
}
