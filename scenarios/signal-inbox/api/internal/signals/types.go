// Package signals owns the immutable journal of externally captured material.
package signals

import (
	"context"
	"fmt"
	"time"
)

type skipInferenceContextKey struct{}

// WithInferenceDeferred marks a bulk archive capture for later local
// classification. It prevents an import from issuing one model call per item.
func WithInferenceDeferred(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipInferenceContextKey{}, true)
}

func InferenceDeferred(ctx context.Context) bool {
	value, _ := ctx.Value(skipInferenceContextKey{}).(bool)
	return value
}

type SourceKind string

const (
	SourceKindURL   SourceKind = "url"
	SourceKindText  SourceKind = "text"
	SourceKindImage SourceKind = "image"
)

// Signal is append-only capture evidence. Classification, category, and
// disposition deliberately do not appear here: they belong to later domains
// and must never gate storage or corpus retrieval.
type Signal struct {
	ID               string
	SourceKind       SourceKind
	SourceIdentity   string
	SourceURL        string
	CapturedAt       time.Time
	RawPayloadRef    string
	ExtractedContent string
	ContentHash      string
	NeedsAttention   bool
	CaptureNote      string
	Tags             []string
}

type CaptureInput struct {
	URL             string
	Text            string
	ImagePayloadRef string
	CaptureNote     string
	Tags            []string
}

type CaptureResult struct {
	Signal    Signal
	Duplicate bool
}

// PostCapture runs only after the immutable journal row has been appended.
// Implementations must treat failure as degradation: capture evidence already
// exists and may never be rolled back because a derived operation failed.
type PostCapture interface {
	Enrich(context.Context, Signal) error
}

// ReadProjection overlays append-only derived records on a signal read. It
// exists so post-capture enrichment never needs an UPDATE against signal.
type ReadProjection interface {
	Project(context.Context, Signal) (Signal, error)
}

type ErrSignalNotFound struct{ ID string }

func (e ErrSignalNotFound) Error() string { return fmt.Sprintf("signal %q not found", e.ID) }

type ErrInvalidSignal struct {
	Field  string
	Reason string
}

func (e ErrInvalidSignal) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }
