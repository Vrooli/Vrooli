package capture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func TestDefaultProducerRegistry_LookupCoversEveryType(t *testing.T) {
	r := DefaultProducerRegistry()
	for ct := range captureTypeMetadata {
		_, ok := r.producers[ct]
		require.Truef(t, ok, "no producer registered for %v", ct)
	}
}

func TestProducerRegistry_ProduceAll_UnknownTypeErrors(t *testing.T) {
	r := NewProducerRegistry() // empty registry
	_, err := r.ProduceAll([]capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT}, t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported capture type")
}

func TestProducerRegistry_ProduceAll_PreservesRequestOrder(t *testing.T) {
	r := DefaultProducerRegistry()
	order := []capturev1.CaptureType{
		capturev1.CaptureType_CAPTURE_TYPE_NETWORK,
		capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
		capturev1.CaptureType_CAPTURE_TYPE_DOM,
	}
	arts, err := r.ProduceAll(order, t.TempDir())
	require.NoError(t, err)
	require.Len(t, arts, 3)
	require.Equal(t, order[0], arts[0].Type)
	require.Equal(t, order[1], arts[1].Type)
	require.Equal(t, order[2], arts[2].Type)
}

func TestScreenshotProducer_ExposesEachFile(t *testing.T) {
	dir := t.TempDir()
	shots := filepath.Join(dir, "screenshots")
	require.NoError(t, os.MkdirAll(shots, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(shots, "step-01-nav.png"), []byte("png-a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(shots, "step-02-wait.png"), []byte("png-bb"), 0o644))

	arts, err := screenshotProducer{}.Produce(dir)
	require.NoError(t, err)
	require.Len(t, arts, 2)
	for _, a := range arts {
		require.Equal(t, capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, a.Type)
		require.NotEqual(t, "true", a.Metadata["unavailable"])
		require.Positive(t, a.SizeBytes)
	}
}

func TestScreenshotProducer_MissingDirMarksUnavailable(t *testing.T) {
	arts, err := screenshotProducer{}.Produce(t.TempDir())
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "true", arts[0].Metadata["unavailable"])
}

func TestFileProducer_PresentAndMissing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "console-logs.md"), []byte("# console\n"), 0o644))

	p := fileProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS, file: "console-logs.md"}
	arts, err := p.Produce(dir)
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.EqualValues(t, len("# console\n"), arts[0].SizeBytes)
	require.NotEqual(t, "true", arts[0].Metadata["unavailable"])

	missing := fileProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_NETWORK, file: "network-activity.md"}
	arts, err = missing.Produce(dir)
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "true", arts[0].Metadata["unavailable"])
}

func TestUnavailableProducer_AlwaysUnavailable(t *testing.T) {
	p := unavailableProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_VIDEO}
	arts, err := p.Produce(t.TempDir())
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "true", arts[0].Metadata["unavailable"])
	require.Contains(t, arts[0].Path, canonicalFileName(capturev1.CaptureType_CAPTURE_TYPE_VIDEO))
}

func TestAccessibilityProducer_PresentAndMissing(t *testing.T) {
	dir := t.TempDir()
	const snapshot = `{"contract":"bas-accessibility-snapshot/v1","node_count":3}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "accessibility.json"), []byte(snapshot), 0o644))

	p := fileProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY, file: "accessibility.json"}
	arts, err := p.Produce(dir)
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY, arts[0].Type)
	require.EqualValues(t, len(snapshot), arts[0].SizeBytes)
	require.NotEqual(t, "true", arts[0].Metadata["unavailable"])

	// Absent file degrades to a single unavailable artifact, never an error.
	arts, err = p.Produce(t.TempDir())
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "true", arts[0].Metadata["unavailable"])
	require.Contains(t, arts[0].Path, "accessibility.json")
}

func TestDefaultProducerRegistry_AccessibilityIsRealProducer(t *testing.T) {
	r := DefaultProducerRegistry()
	p, ok := r.producers[capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY]
	require.True(t, ok)
	_, isUnavailable := p.(unavailableProducer)
	require.False(t, isUnavailable, "accessibility must be a real fileProducer, not the unavailable placeholder")
}

func TestMetadata_PerformanceAvailable(t *testing.T) {
	// P2 contract: PERFORMANCE is now produced by the driver tracer.
	require.True(t, metaFor(capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE).available)
	require.True(t, metaFor(capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT).available)
	require.Equal(t, "performance.json", canonicalFileName(capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE))

	// Accessibility is a driver-produced, available artifact.
	require.True(t, metaFor(capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY).available)
	require.Equal(t, "accessibility.json", canonicalFileName(capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY))
}

func TestPerformanceProducer_TraceAndVitals(t *testing.T) {
	dir := t.TempDir()
	perfDir := filepath.Join(dir, "performance")
	require.NoError(t, os.MkdirAll(perfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(perfDir, "performance.json"), []byte(`{"traceEvents":[]}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(perfDir, "performance.web-vitals.json"), []byte(`{"lcp":1}`), 0o644))

	arts, err := performanceProducer{}.Produce(dir)
	require.NoError(t, err)
	require.Len(t, arts, 2)
	require.Equal(t, capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE, arts[0].Type)
	require.Equal(t, "cdp-trace", arts[0].Metadata["artifact"])
	require.Positive(t, arts[0].SizeBytes)
	require.Equal(t, "web-vitals", arts[1].Metadata["artifact"])
	require.NotEqual(t, "true", arts[0].Metadata["unavailable"])
}

func TestPerformanceProducer_TraceOnlyNoVitals(t *testing.T) {
	dir := t.TempDir()
	perfDir := filepath.Join(dir, "performance")
	require.NoError(t, os.MkdirAll(perfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(perfDir, "performance.json"), []byte(`{"traceEvents":[]}`), 0o644))

	arts, err := performanceProducer{}.Produce(dir)
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "cdp-trace", arts[0].Metadata["artifact"])
}

func TestPerformanceProducer_MissingTraceMarksUnavailable(t *testing.T) {
	arts, err := performanceProducer{}.Produce(t.TempDir())
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "true", arts[0].Metadata["unavailable"])
	require.Contains(t, arts[0].Path, "performance.json")
}

func TestDefaultProducerRegistry_PerformanceIsRealProducer(t *testing.T) {
	r := DefaultProducerRegistry()
	p, ok := r.producers[capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE]
	require.True(t, ok)
	_, isUnavailable := p.(unavailableProducer)
	require.False(t, isUnavailable, "performance must be a real producer, not the unavailable placeholder")
}
