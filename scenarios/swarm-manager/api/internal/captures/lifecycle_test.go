package captures

import (
	"errors"
	"testing"
)

func TestCaptureTransitionStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		from string
		to   CaptureStatus
		want string
	}{
		{"classification failure", string(CaptureStatusClassifying), CaptureStatusFailed, string(CaptureStatusFailed)},
		{"retry", string(CaptureStatusFailed), CaptureStatusClassifying, string(CaptureStatusClassifying)},
		{"active reentry", string(CaptureStatusClassifying), CaptureStatusClassifying, string(CaptureStatusClassifying)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cap := capture{Status: tc.from}
			if err := cap.TransitionStatus(tc.to); err != nil {
				t.Fatalf("TransitionStatus(%q) error = %v", tc.to, err)
			}
			if cap.Status != tc.want {
				t.Fatalf("status = %q, want %q", cap.Status, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name string
		from string
		to   CaptureStatus
	}{
		{"removed success state", "classified", CaptureStatusFailed},
		{"success cannot be persisted", string(CaptureStatusClassifying), CaptureStatus("classified")},
		{"unknown state", "mystery", CaptureStatusClassifying},
		{"failed self transition", string(CaptureStatusFailed), CaptureStatusFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cap := capture{Status: tc.from}
			if err := cap.TransitionStatus(tc.to); !errors.Is(err, ErrInvalidCaptureTransition) {
				t.Fatalf("TransitionStatus(%q) error = %v, want invalid transition", tc.to, err)
			}
			if cap.Status != tc.from {
				t.Fatalf("rejected transition mutated status to %q", cap.Status)
			}
		})
	}
}
