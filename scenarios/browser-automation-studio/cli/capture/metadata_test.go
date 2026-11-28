package capture

import (
	"testing"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func TestParseCaptureType_LabelsAndAliases(t *testing.T) {
	cases := map[string]capturev1.CaptureType{
		"screenshot":   capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
		"console-logs": capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
		"console":      capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
		"logs":         capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
		"console_logs": capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS, // underscore→hyphen
		"NETWORK":      capturev1.CaptureType_CAPTURE_TYPE_NETWORK,      // case-insensitive
		"video":        capturev1.CaptureType_CAPTURE_TYPE_VIDEO,
		"performance":  capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE,
		"perf":         capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE,
	}
	for tok, want := range cases {
		got, err := parseCaptureType(tok)
		if err != nil {
			t.Fatalf("parseCaptureType(%q): %v", tok, err)
		}
		if got != want {
			t.Fatalf("parseCaptureType(%q) = %v, want %v", tok, got, want)
		}
	}
}

func TestParseCaptureType_Unknown(t *testing.T) {
	for _, value := range []string{"nope", "dom", "dom-tree", "computed-dom"} {
		if _, err := parseCaptureType(value); err == nil {
			t.Errorf("expected error for retired capture type %q", value)
		}
	}
}

func TestCaptureTypeLabel_RoundTrip(t *testing.T) {
	for ct, meta := range captureTypeMetadata {
		if got := captureTypeLabel(ct); got != meta.label {
			t.Fatalf("captureTypeLabel(%v) = %q, want %q", ct, got, meta.label)
		}
		// The canonical label must parse back to the same type.
		parsed, err := parseCaptureType(meta.label)
		if err != nil || parsed != ct {
			t.Fatalf("label %q failed round trip: parsed=%v err=%v", meta.label, parsed, err)
		}
	}
	if got := captureTypeLabel(capturev1.CaptureType_CAPTURE_TYPE_UNSPECIFIED); got != "unspecified" {
		t.Fatalf("unspecified label = %q", got)
	}
}
