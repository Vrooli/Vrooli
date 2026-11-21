package branch

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseCreateFlags(t *testing.T) {
	got := parseCreateFlags([]string{
		"feature/test-seams",
		"--from=main",
		"--no-checkout",
		"--allow-dirty",
	})

	if got.name != "feature/test-seams" {
		t.Fatalf("name = %q", got.name)
	}
	if got.from != "main" {
		t.Fatalf("from = %q", got.from)
	}
	if got.checkout {
		t.Fatal("checkout should be false after --no-checkout")
	}
	if !got.allowDirty {
		t.Fatal("allowDirty should be true after --allow-dirty")
	}
}

func TestPrintCreateResultWarningSuggestsAllowDirty(t *testing.T) {
	output := captureStdout(t, func() {
		printCreateResult(&createResponse{
			Warning: &warning{
				Message:              "working tree has changes",
				RequiresConfirmation: true,
			},
		}, "feature/test-seams", false)
	})

	for _, want := range []string{
		"Warning: working tree has changes",
		"Retry with --allow-dirty to force checkout",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("create warning output missing %q:\n%s", want, output)
		}
	}
}

func TestPrintSwitchResultWarningSuggestsRequiredFlags(t *testing.T) {
	output := captureStdout(t, func() {
		printSwitchResult(&switchResponse{
			Warning: &warning{
				Message:              "remote branch needs tracking",
				RequiresTracking:     true,
				RequiresConfirmation: true,
			},
		}, switchFlags{name: "origin/feature/test-seams"})
	})

	for _, want := range []string{
		"Warning: remote branch needs tracking",
		"Retry with --track-remote to track and switch",
		"Retry with --allow-dirty to force switch",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("switch warning output missing %q:\n%s", want, output)
		}
	}
}

func TestPrintPublishResultWarningSuggestsFetch(t *testing.T) {
	output := captureStdout(t, func() {
		printPublishResult(&publishResponse{
			Warning: &warning{
				Message:       "remote state is stale",
				RequiresFetch: true,
			},
		}, false)
	})

	for _, want := range []string{
		"Warning: remote state is stale",
		"Retry with --fetch to refresh remote status",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("publish warning output missing %q:\n%s", want, output)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	t.Cleanup(func() {
		os.Stdout = original
	})
	os.Stdout = writer

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("closing stdout pipe writer failed: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("reading stdout pipe failed: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("closing stdout pipe reader failed: %v", err)
	}
	return buf.String()
}
