package capture

import capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"

// captureTypeMeta is the single source of truth for everything a
// CaptureType "is" on the handler side: its human/file label, on-disk
// extension, and whether the executor's folder export can produce it
// today. The five label/ext/availability switches that used to be
// scattered across service.go collapse into this one table.
//
// availableReason is set only when available is false; it is surfaced
// verbatim in the unavailable artifact's metadata so callers see the
// gap explicitly rather than a silent omission.
type captureTypeMeta struct {
	shortName       string
	ext             string
	available       bool
	availableReason string
}

const unavailableExportReason = "executor folder export does not produce this artifact type yet"

// captureTypeMetadata maps every CaptureType to its metadata. PERFORMANCE
// is available: the driver's PerformanceTracer streams a CDP trace +
// web-vitals into the execution artifact root, ExportToFolder copies them
// into outDir/performance/, and performanceProducer surfaces them.
var captureTypeMetadata = map[capturev1.CaptureType]captureTypeMeta{
	capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT: {
		shortName: "screenshot",
		ext:       ".png",
		available: true,
	},
	capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS: {
		shortName: "console-logs",
		ext:       ".md",
		available: true,
	},
	capturev1.CaptureType_CAPTURE_TYPE_NETWORK: {
		shortName: "network",
		ext:       ".md",
		available: true,
	},
	capturev1.CaptureType_CAPTURE_TYPE_VIDEO: {
		shortName:       "video",
		ext:             ".webm",
		available:       false,
		availableReason: unavailableExportReason,
	},
	capturev1.CaptureType_CAPTURE_TYPE_DOM: {
		shortName:       "dom",
		ext:             ".html",
		available:       false,
		availableReason: unavailableExportReason,
	},
	capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE: {
		shortName: "performance",
		ext:       ".json",
		available: true,
	},
}

// metaFor returns the metadata for a CaptureType, falling back to a
// deterministic "unknown" entry so callers never have to nil-check.
func metaFor(c capturev1.CaptureType) captureTypeMeta {
	if m, ok := captureTypeMetadata[c]; ok {
		return m
	}
	return captureTypeMeta{
		shortName:       "unknown",
		ext:             "",
		available:       false,
		availableReason: "unknown capture type",
	}
}

// canonicalFileName returns the canonical per-type artifact filename
// (shortName+ext) used for synthesized and unavailable artifacts.
func canonicalFileName(c capturev1.CaptureType) string {
	m := metaFor(c)
	return m.shortName + m.ext
}
