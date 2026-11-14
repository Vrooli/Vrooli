package capture

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// writeAccessibilitySnapshot lands an accessibility.json in the export folder,
// mirroring what ExportToFolder copies out of the driver's artifact root.
func writeAccessibilitySnapshot(content string) func(f *fakeExecutor, outputDir string) error {
	return func(_ *fakeExecutor, outputDir string) error {
		return os.WriteFile(filepath.Join(outputDir, "accessibility.json"), []byte(content), 0o644)
	}
}

func TestCapture_AccessibilityCaptureType_SetsOpts(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:      "https://example.com",
		Captures: []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY},
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)
	require.NotNil(t, exec.LastOpts)
	require.True(t, exec.LastOpts.RequiresAccessibility, "ACCESSIBILITY capture must set RequiresAccessibility opt")
	// No inline requested → accessibility_json empty even if a file exists.
	require.Empty(t, resp.Msg.AccessibilityJson)
}

func TestCapture_InlineAccessibility_ReturnsSnapshotJSON(t *testing.T) {
	const snapshot = `{"contract":"bas-accessibility-snapshot/v1","node_count":2,"root":{"role":"WebArea"}}`
	exec := &fakeExecutor{ExportFunc: writeAccessibilitySnapshot(snapshot)}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com",
		Captures:            []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY},
		InlineAccessibility: true,
	}))
	require.NoError(t, err)
	require.Equal(t, snapshot, resp.Msg.AccessibilityJson)
}

func TestCapture_InlineAccessibility_IndependentlyDrivesCapture(t *testing.T) {
	// inline_accessibility alone (no ACCESSIBILITY in captures) must still set
	// the RequiresAccessibility opt, mirroring how inline_dom drives its own read.
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	_, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com",
		InlineAccessibility: true,
	}))
	require.NoError(t, err)
	require.NotNil(t, exec.LastOpts)
	require.True(t, exec.LastOpts.RequiresAccessibility)
}

func TestCapture_InlineAccessibility_MissingFileDegradesToEmpty(t *testing.T) {
	// No accessibility.json written: capture still succeeds, accessibility_json
	// is empty (the documented degraded contract).
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com",
		InlineAccessibility: true,
	}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.AccessibilityJson)
}

func TestCapture_InlineAccessibility_TruncatesOversizedPayload(t *testing.T) {
	oversized := `{"contract":"bas-accessibility-snapshot/v1","pad":"` +
		strings.Repeat("x", defaultInlineAccessibilityMaxBytes) + `"}`
	exec := &fakeExecutor{ExportFunc: writeAccessibilitySnapshot(oversized)}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com",
		InlineAccessibility: true,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.AccessibilityJson, defaultInlineAccessibilityMaxBytes)
}

func TestCapture_InlineAccessibility_DryRun_StaysEmpty(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	req := connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com",
		InlineAccessibility: true,
	})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := client.Capture(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.DryRun)
	require.Empty(t, resp.Msg.AccessibilityJson)
	require.Zero(t, exec.Calls)
}
