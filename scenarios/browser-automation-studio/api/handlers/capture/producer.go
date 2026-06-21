package capture

import (
	"fmt"
	"os"
	"path/filepath"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// ArtifactProducer turns one execution's on-disk export into the
// CaptureArtifact(s) for a single CaptureType. The executor's folder
// export owns the write layout (`screenshots/step-NN-*.png`,
// `console-logs.md`, `network-activity.md`, …); a producer is the
// read-side counterpart that walks outDir and assembles the response.
//
// This is the seam P2 plugs into: the performance producer (driver
// tracing → perf artifact) registers here without touching the handler's
// orchestration.
type ArtifactProducer interface {
	// Type is the CaptureType this producer is responsible for.
	Type() capturev1.CaptureType

	// Produce reads outDir and returns every artifact for this type. A
	// type whose files are absent returns a single artifact flagged
	// unavailable (via unavailableArtifact) rather than an error, so a
	// missing optional artifact never fails the whole capture.
	Produce(outDir string) ([]*capturev1.CaptureArtifact, error)
}

// ProducerRegistry resolves CaptureTypes to their ArtifactProducer and
// fans a requested set out to all of them.
type ProducerRegistry struct {
	producers map[capturev1.CaptureType]ArtifactProducer
}

// NewProducerRegistry builds a registry from the given producers. A later
// registration for the same type wins, letting P2 swap the placeholder
// performance producer for a real one.
func NewProducerRegistry(producers ...ArtifactProducer) *ProducerRegistry {
	r := &ProducerRegistry{producers: make(map[capturev1.CaptureType]ArtifactProducer, len(producers))}
	for _, p := range producers {
		r.producers[p.Type()] = p
	}
	return r
}

// DefaultProducerRegistry returns the registry wired with the producers
// BAS ships today: one per implemented capture type plus placeholder
// "unavailable" producers for the types the executor cannot export yet.
func DefaultProducerRegistry() *ProducerRegistry {
	return NewProducerRegistry(
		screenshotProducer{},
		fileProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS, file: "console-logs.md"},
		fileProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_NETWORK, file: "network-activity.md"},
		unavailableProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_VIDEO},
		unavailableProducer{captureType: capturev1.CaptureType_CAPTURE_TYPE_DOM},
		performanceProducer{},
	)
}

// ProduceAll assembles, in request order, every artifact for the given
// capture types from outDir. An unknown type (no registered producer) is
// a programmer error and returns an error.
func (r *ProducerRegistry) ProduceAll(types []capturev1.CaptureType, outDir string) ([]*capturev1.CaptureArtifact, error) {
	out := make([]*capturev1.CaptureArtifact, 0, len(types))
	for _, c := range types {
		p, ok := r.producers[c]
		if !ok {
			return nil, fmt.Errorf("unsupported capture type: %v", c)
		}
		arts, err := p.Produce(outDir)
		if err != nil {
			return nil, err
		}
		out = append(out, arts...)
	}
	return out, nil
}

// screenshotProducer exposes every file under screenshots/ as its own
// artifact — single-location captures usually produce one PNG, but the
// contract permits multi-step output (e.g. a future wait-for step).
type screenshotProducer struct{}

func (screenshotProducer) Type() capturev1.CaptureType {
	return capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT
}

func (p screenshotProducer) Produce(outDir string) ([]*capturev1.CaptureArtifact, error) {
	c := p.Type()
	shotsDir := filepath.Join(outDir, "screenshots")
	fallback := filepath.Join(shotsDir, "screenshot.png")
	entries, err := os.ReadDir(shotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*capturev1.CaptureArtifact{unavailableArtifact(c, fallback)}, nil
		}
		return nil, fmt.Errorf("read screenshots dir: %w", err)
	}
	out := make([]*capturev1.CaptureArtifact, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		full := filepath.Join(shotsDir, e.Name())
		info, err := e.Info()
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", full, err)
		}
		out = append(out, &capturev1.CaptureArtifact{
			Type:      c,
			Path:      full,
			SizeBytes: info.Size(),
			Metadata:  map[string]string{"filename": e.Name()},
		})
	}
	if len(out) == 0 {
		return []*capturev1.CaptureArtifact{unavailableArtifact(c, fallback)}, nil
	}
	return out, nil
}

// fileProducer exposes a single named file under outDir as the artifact
// for its capture type (console-logs.md, network-activity.md, …).
type fileProducer struct {
	captureType capturev1.CaptureType
	file        string
}

func (p fileProducer) Type() capturev1.CaptureType { return p.captureType }

func (p fileProducer) Produce(outDir string) ([]*capturev1.CaptureArtifact, error) {
	return []*capturev1.CaptureArtifact{artifactFromFile(p.captureType, filepath.Join(outDir, p.file))}, nil
}

// performanceProducer exposes the CDP performance trace and web-vitals
// files the driver streamed into the execution's artifact root and that
// ExportToFolder copied into outDir/performance/. The trace
// (performance.json) is the primary artifact; the web-vitals summary
// (performance.web-vitals.json) is emitted as a second artifact when the
// page produced any observable metrics. Absent files degrade to a single
// unavailable artifact (e.g. headless-without-browser), never an error.
type performanceProducer struct{}

func (performanceProducer) Type() capturev1.CaptureType {
	return capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE
}

func (p performanceProducer) Produce(outDir string) ([]*capturev1.CaptureArtifact, error) {
	c := p.Type()
	perfDir := filepath.Join(outDir, "performance")

	tracePath := filepath.Join(perfDir, "performance.json")
	traceInfo, err := os.Stat(tracePath)
	if err != nil {
		// No trace on disk: the run did not produce one (no browser / not
		// requested). Surface a single unavailable artifact at the canonical
		// path rather than failing the whole capture.
		return []*capturev1.CaptureArtifact{
			unavailableArtifact(c, filepath.Join(outDir, canonicalFileName(c))),
		}, nil
	}

	out := []*capturev1.CaptureArtifact{
		{
			Type:      c,
			Path:      tracePath,
			SizeBytes: traceInfo.Size(),
			Metadata: map[string]string{
				"filename": "performance.json",
				"artifact": "cdp-trace",
			},
		},
	}

	vitalsPath := filepath.Join(perfDir, "performance.web-vitals.json")
	if vitalsInfo, err := os.Stat(vitalsPath); err == nil {
		out = append(out, &capturev1.CaptureArtifact{
			Type:      c,
			Path:      vitalsPath,
			SizeBytes: vitalsInfo.Size(),
			Metadata: map[string]string{
				"filename": "performance.web-vitals.json",
				"artifact": "web-vitals",
			},
		})
	}

	return out, nil
}

// unavailableProducer is the placeholder for capture types the executor's
// folder export cannot produce yet (video, dom-file). It
// always returns a single unavailable artifact with the canonical path.
// P2 replaces the performance entry with a real producer.
type unavailableProducer struct {
	captureType capturev1.CaptureType
}

func (p unavailableProducer) Type() capturev1.CaptureType { return p.captureType }

func (p unavailableProducer) Produce(outDir string) ([]*capturev1.CaptureArtifact, error) {
	return []*capturev1.CaptureArtifact{
		unavailableArtifact(p.captureType, filepath.Join(outDir, canonicalFileName(p.captureType))),
	}, nil
}
