package main

// engine.go defines the resource-local seam around the ONNX token-classification
// model. The HTTP layer never learns whether the implementation is the real
// ONNX Runtime binding (engine_onnx.go, behind the `pii_onnx` + cgo-free
// dlopen build tag) or the fail-closed stub (engine_stub.go). This mirrors the
// sherpa-onnx resource: native inference is confined to one build-tagged file
// and the service remains usable (regex pass only) when the model is absent.

import (
	"context"
	"os"
	"strings"
)

// NERModel runs a token-classification PII model over text and returns the
// model-predicted spans. The implementation owns the tokenizer and the ONNX
// session; it exposes only plain span data so the HTTP layer stays buildable
// in the stub configuration.
type NERModel interface {
	// Detect returns zero or more spans with Source="model". Byte offsets must
	// index into the original UTF-8 text.
	Detect(ctx context.Context, text string) ([]Span, error)
	// ModelID is the active model identifier (e.g. the HF repo id) for /health
	// and /info surfacing.
	ModelID() string
	Close()
}

// UnavailableModel is the fail-closed NER implementation used when no model
// artifact is present or the build lacks the native tag. The service still
// serves the deterministic regex pass; model spans are simply empty and
// Detect reports the degradation to the caller via an error the handler
// swallows into a health flag.
type UnavailableModel struct {
	reason string
}

func (m UnavailableModel) Detect(context.Context, string) ([]Span, error) {
	return nil, errModelUnavailable{reason: m.reason}
}

func (m UnavailableModel) ModelID() string { return "" }

func (UnavailableModel) Close() {}

type errModelUnavailable struct{ reason string }

func (e errModelUnavailable) Error() string {
	if e.reason == "" {
		return "pii-detector NER model is unavailable in this build"
	}
	return "pii-detector NER model is unavailable: " + e.reason
}

// IsModelUnavailable reports whether err is the model-unavailable sentinel.
func IsModelUnavailable(err error) bool {
	_, ok := err.(errModelUnavailable)
	return ok
}

// modelLoadDisabled reports whether the capacity degrade ladder has pinned the
// server to the regex-only rung via PII_DETECTOR_LOAD_MODEL=off. This is the one
// place the ladder's env toggle is honored, so the "regex-only" lifecycle step
// (cli capacitySteps) is a real transition and not a cosmetic no-op: when it is
// set off, newNERModelFromEnv returns the fail-closed model regardless of build
// tag. The companion ladder restarts the server with this env, so a plain
// string compare here is the whole contract.
func modelLoadDisabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PII_DETECTOR_LOAD_MODEL")), "off")
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
