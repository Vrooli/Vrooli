package signals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"signal-inbox/internal/clock"
)

const defaultListLimit = 100

type Service interface {
	Capture(ctx context.Context, in CaptureInput) (CaptureResult, error)
	Get(ctx context.Context, id string) (Signal, error)
	List(ctx context.Context, limit int) ([]Signal, error)
}

type service struct {
	repo         Repository
	clock        clock.Clock
	postCaptures []PostCapture
	projection   ReadProjection
}

func NewService(repo Repository, clk clock.Clock, hooks ...PostCapture) Service {
	svc := &service{repo: repo, clock: clk}
	for _, hook := range hooks {
		svc.postCaptures = append(svc.postCaptures, hook)
		if projection, ok := hook.(ReadProjection); ok {
			svc.projection = projection
		}
	}
	return svc
}

func (s *service) Capture(ctx context.Context, in CaptureInput) (CaptureResult, error) {
	signal, err := newSignal(in, s.clock.Now())
	if err != nil {
		return CaptureResult{}, err
	}
	result, err := s.repo.Append(ctx, signal)
	if err != nil || result.Duplicate {
		return result, err
	}
	// The journal append is the durable operation. Derived enrichment is
	// deliberately best-effort so an unavailable extractor cannot lose a save.
	for _, hook := range s.postCaptures {
		_ = hook.Enrich(ctx, result.Signal)
		// A hook that also owns a read projection (enrichment today) may make
		// newly derived content available to later post-capture hooks without
		// mutating the journal row.
		if projection, ok := hook.(ReadProjection); ok {
			if projected, projectErr := projection.Project(ctx, result.Signal); projectErr == nil {
				result.Signal = projected
			}
		}
	}
	return s.project(ctx, result)
}

func (s *service) Get(ctx context.Context, id string) (Signal, error) {
	signal, err := s.repo.Get(ctx, id)
	if err != nil {
		return Signal{}, err
	}
	if s.projection == nil {
		return signal, nil
	}
	return s.projection.Project(ctx, signal)
}

func (s *service) List(ctx context.Context, limit int) ([]Signal, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	results, err := s.repo.List(ctx, limit)
	if err != nil || s.projection == nil {
		return results, err
	}
	for i := range results {
		results[i], err = s.projection.Project(ctx, results[i])
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (s *service) project(ctx context.Context, result CaptureResult) (CaptureResult, error) {
	if s.projection == nil {
		return result, nil
	}
	projected, err := s.projection.Project(ctx, result.Signal)
	if err != nil {
		return result, nil // projection loss must not turn a successful capture into failure
	}
	result.Signal = projected
	return result, nil
}

func newSignal(in CaptureInput, capturedAt time.Time) (Signal, error) {
	variants := 0
	if strings.TrimSpace(in.URL) != "" {
		variants++
	}
	if strings.TrimSpace(in.Text) != "" {
		variants++
	}
	if strings.TrimSpace(in.ImagePayloadRef) != "" {
		variants++
	}
	if variants != 1 {
		return Signal{}, ErrInvalidSignal{Field: "source", Reason: "supply exactly one of url, text, or image payload reference"}
	}

	s := Signal{CapturedAt: capturedAt.UTC(), CaptureNote: strings.TrimSpace(in.CaptureNote), Tags: normalizeTags(in.Tags)}
	switch {
	case strings.TrimSpace(in.URL) != "":
		identity, err := NormalizeSourceIdentity(in.URL)
		if err != nil {
			return Signal{}, ErrInvalidSignal{Field: "url", Reason: err.Error()}
		}
		s.SourceKind, s.SourceURL, s.SourceIdentity = SourceKindURL, strings.TrimSpace(in.URL), identity
		s.ContentHash = hash("url:\x00" + identity)
	case strings.TrimSpace(in.Text) != "":
		content := normalizeText(in.Text)
		s.SourceKind, s.SourceIdentity, s.ExtractedContent = SourceKindText, hash("text-identity:\x00"+content), content
		s.ContentHash = hash("text:\x00" + content)
	default:
		ref := strings.TrimSpace(in.ImagePayloadRef)
		s.SourceKind, s.SourceIdentity, s.RawPayloadRef, s.NeedsAttention = SourceKindImage, ref, ref, true
		s.ContentHash = hash("image:\x00" + ref)
	}
	return s, nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
