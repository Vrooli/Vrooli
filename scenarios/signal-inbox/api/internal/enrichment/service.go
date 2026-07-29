package enrichment

import (
	"context"
	"strings"

	"signal-inbox/internal/clock"
	"signal-inbox/internal/signals"
)

type Service struct {
	repo       Repository
	clock      clock.Clock
	extractors []Extractor
}

func NewService(repo Repository, clk clock.Clock, extractors ...Extractor) *Service {
	return &Service{repo: repo, clock: clk, extractors: extractors}
}

// Enrich records one immutable result for each post-capture attempt. It never
// returns a failure to capture: callers retain the error only for diagnostics.
func (s *Service) Enrich(ctx context.Context, signal signals.Signal) error {
	// A trusted tier-zero archive can carry the original text. Persist it as
	// enrichment evidence without re-fetching the platform URL; archive import
	// must not turn into automated platform traffic.
	if content := normalizeContent(signal.ExtractedContent); content != "" {
		return s.repo.Append(ctx, Record{SignalID: signal.ID, ExtractedContent: content, ContentUnits: 1, AttemptedAt: s.clock.Now().UTC()})
	}
	if hasTag(signal.Tags, "x") {
		return s.appendAttention(ctx, signal.ID, "X archive record has no text; platform fetch is disabled for archive import")
	}
	extractor := s.extractorFor(signal.SourceKind)
	if extractor == nil {
		return s.appendAttention(ctx, signal.ID, "no extractor supports "+string(signal.SourceKind))
	}
	result, err := extractor.Extract(ctx, signal)
	if err != nil {
		if appendErr := s.appendAttention(ctx, signal.ID, "extraction unavailable: "+err.Error()); appendErr != nil {
			return appendErr
		}
		return err
	}
	content := normalizeContent(result.Content)
	units := result.ContentUnits
	if units <= 0 || content == "" {
		return s.appendAttention(ctx, signal.ID, "extraction returned zero content units")
	}
	return s.repo.Append(ctx, Record{SignalID: signal.ID, ExtractedContent: content, ContentUnits: units, AttemptedAt: s.clock.Now().UTC()})
}

func (s *Service) Project(ctx context.Context, signal signals.Signal) (signals.Signal, error) {
	record, found, err := s.repo.Latest(ctx, signal.ID)
	if err != nil || !found {
		return signal, err
	}
	if record.ContentUnits == 0 || record.NeedsAttention {
		signal.ExtractedContent = ""
		signal.NeedsAttention = true
		return signal, nil
	}
	signal.ExtractedContent = record.ExtractedContent
	signal.NeedsAttention = false
	return signal, nil
}

func (s *Service) extractorFor(kind signals.SourceKind) Extractor {
	for _, extractor := range s.extractors {
		if extractor.Supports(kind) {
			return extractor
		}
	}
	return nil
}

func (s *Service) appendAttention(ctx context.Context, signalID, reason string) error {
	return s.repo.Append(ctx, Record{SignalID: signalID, NeedsAttention: true, AttentionReason: reason, AttemptedAt: s.clock.Now().UTC()})
}

func normalizeContent(value string) string { return strings.Join(strings.Fields(value), " ") }

func hasTag(tags []string, wanted string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), wanted) {
			return true
		}
	}
	return false
}

var (
	_ signals.PostCapture    = (*Service)(nil)
	_ signals.ReadProjection = (*Service)(nil)
)
