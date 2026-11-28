//go:build audio_lab

package corpus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"audio-tools/internal/blobbytes"

	"github.com/vrooli/api-core/schedule"
)

// Service is the corpus CRUD domain logic: it ties the metadata Repository
// to the audio BlobBytes store so a clip's bytes and row are written and
// deleted together. It is the seam the CorpusService Connect handler and
// the eval harness build on.
type Service struct {
	repo  Repository
	blobs blobbytes.Store
	clock schedule.Clock
}

// NewService constructs a corpus Service.
func NewService(repo Repository, blobs blobbytes.Store, clk schedule.Clock) *Service {
	return &Service{repo: repo, blobs: blobs, clock: clk}
}

// CreateClipInput is the create payload: the raw audio bytes plus the
// metadata. The service generates the id + blob key.
type CreateClipInput struct {
	Audio            []byte
	ReferenceText    string
	Tags             []string
	DurationMs       int64
	SampleRateHz     int
	Format           string
	Source           Source
	CorpusRevision   string
	LicenseReference string
	SpeakerPseudonym string
	Split            Split
	NoiseCondition   NoiseCondition
	SNRDB            float64
	TimedReference   *TimedReference
	ReferenceReview  *ReferenceReview
}

// CreateClip writes the audio to the blob store, then the metadata row. On
// a metadata failure it rolls the blob back so an orphan blob can't leak.
func (s *Service) CreateClip(ctx context.Context, in CreateClipInput) (Clip, error) {
	if len(in.Audio) == 0 {
		return Clip{}, fmt.Errorf("corpus: CreateClip requires audio bytes")
	}
	if err := in.TimedReference.Validate(in.DurationMs); err != nil {
		return Clip{}, fmt.Errorf("corpus: invalid timed reference: %w", err)
	}
	if err := in.TimedReference.ValidateTranscript(in.ReferenceText); err != nil {
		return Clip{}, fmt.Errorf("corpus: timed reference transcript mismatch: %w", err)
	}
	if in.TimedReference != nil {
		encoded, err := json.Marshal(in.TimedReference)
		if err != nil {
			return Clip{}, fmt.Errorf("corpus: encode timed reference: %w", err)
		}
		if len(encoded) > MaxTimedReferenceBytes {
			return Clip{}, fmt.Errorf("corpus: timed reference exceeds %d bytes", MaxTimedReferenceBytes)
		}
	}
	if in.ReferenceReview != nil {
		in.ReferenceReview.ReferenceSHA256 = ReferenceDigest(in.ReferenceText)
		if err := in.ReferenceReview.Validate(in.ReferenceText); err != nil {
			return Clip{}, fmt.Errorf("corpus: invalid reference review: %w", err)
		}
	}
	now := s.clock.Now().UTC()
	id := uuid.NewString()
	key := blobKey(id, in.Format, now)

	if err := s.blobs.Put(ctx, key, in.Audio, mimeForFormat(in.Format)); err != nil {
		return Clip{}, fmt.Errorf("corpus: store audio: %w", err)
	}
	digest := sha256.Sum256(in.Audio)
	clip := Clip{
		ID:               id,
		ReferenceText:    in.ReferenceText,
		Tags:             in.Tags,
		DurationMs:       in.DurationMs,
		SampleRateHz:     in.SampleRateHz,
		Format:           in.Format,
		BlobKey:          key,
		Source:           in.Source.Normalize(),
		CorpusRevision:   in.CorpusRevision,
		LicenseReference: in.LicenseReference,
		SpeakerPseudonym: in.SpeakerPseudonym,
		Split:            in.Split,
		NoiseCondition:   in.NoiseCondition,
		SNRDB:            in.SNRDB,
		TimedReference:   in.TimedReference,
		ReferenceReview:  in.ReferenceReview,
		ContentSHA256:    hex.EncodeToString(digest[:]),
		CreatedAt:        now,
	}
	saved, err := s.repo.Create(ctx, clip)
	if err != nil {
		// Roll the blob back so a failed metadata write doesn't leak bytes.
		_ = s.blobs.Delete(ctx, key)
		return Clip{}, err
	}
	return saved, nil
}

// ListClips returns clip metadata (no audio).
func (s *Service) ListClips(ctx context.Context, filter ListFilter) ([]Clip, error) {
	return s.repo.List(ctx, filter)
}

// CountClipsCreatedInWindow counts clips created in the UTC interval [from, to).
func (s *Service) CountClipsCreatedInWindow(ctx context.Context, from, to time.Time) (int64, error) {
	return s.repo.CountCreatedInWindow(ctx, from, to)
}

// GetClip returns one clip's metadata.
func (s *Service) GetClip(ctx context.Context, id string) (Clip, error) {
	return s.repo.Get(ctx, id)
}

// GetClipAudio returns a clip's audio bytes alongside its metadata.
func (s *Service) GetClipAudio(ctx context.Context, id string) ([]byte, Clip, error) {
	clip, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, Clip{}, err
	}
	audio, err := s.blobs.Get(ctx, clip.BlobKey)
	if err != nil {
		return nil, Clip{}, fmt.Errorf("corpus: load audio for %q: %w", id, err)
	}
	return audio, clip, nil
}

// ReviewReference saves a corrected transcript and review provenance only
// when the submitted digest still identifies the currently stored text.
func (s *Service) ReviewReference(ctx context.Context, id, expectedDigest, referenceText, reviewer, protocol string) (Clip, error) {
	if strings.TrimSpace(reviewer) == "" || strings.TrimSpace(protocol) == "" {
		return Clip{}, fmt.Errorf("corpus: reference review requires reviewer and protocol")
	}
	if expectedDigest == "" {
		return Clip{}, fmt.Errorf("corpus: expected reference digest is required")
	}
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Clip{}, err
	}
	if ReferenceDigest(current.ReferenceText) != expectedDigest {
		return Clip{}, ErrStaleReference{ID: id}
	}
	review := ReferenceReview{Reviewer: strings.TrimSpace(reviewer), Protocol: strings.TrimSpace(protocol), ReviewedAt: s.clock.Now().UTC().Format(time.RFC3339Nano), ReferenceSHA256: ReferenceDigest(referenceText)}
	return s.repo.ReviewReference(ctx, id, current.ReferenceText, referenceText, review)
}

// ReviewReferenceGroup binds one correction to every persisted audio variant
// of the same manifest source. The caller must submit every exact member and
// expected digest; repository CAS makes the transcript/review writes atomic.
func (s *Service) ReviewReferenceGroup(ctx context.Context, members []ReviewMember, referenceText, reviewer, protocol string) ([]Clip, error) {
	if len(members) < 2 {
		return nil, fmt.Errorf("corpus: grouped review requires at least two variants")
	}
	if strings.TrimSpace(reviewer) == "" || strings.TrimSpace(protocol) == "" {
		return nil, fmt.Errorf("corpus: reference review requires reviewer and protocol")
	}
	byID := make(map[string]ReviewMember, len(members))
	for _, member := range members {
		if strings.TrimSpace(member.ID) == "" || member.ExpectedReferenceSHA256 == "" {
			return nil, fmt.Errorf("corpus: group members require id and expected transcript digest")
		}
		if _, exists := byID[member.ID]; exists {
			return nil, fmt.Errorf("corpus: duplicate group member %q", member.ID)
		}
		byID[member.ID] = member
	}
	seed, err := s.repo.Get(ctx, members[0].ID)
	if err != nil {
		return nil, err
	}
	if seed.SourceID == "" {
		return nil, fmt.Errorf("corpus: grouped review requires manifest source id, revision, speaker, and accent metadata")
	}
	all, err := s.repo.ListBySourceID(ctx, seed.SourceID)
	if err != nil {
		return nil, err
	}
	var selected []Clip
	for _, clip := range all {
		if _, ok := byID[clip.ID]; ok {
			selected = append(selected, clip)
		}
	}
	if len(selected) != len(members) {
		return nil, fmt.Errorf("corpus: grouped review contains a missing clip")
	}
	base := seed
	if len(selected) > 0 {
		base = selected[0]
	}
	if base.SourceID == "" || base.CorpusRevision == "" || base.SpeakerPseudonym == "" || base.AccentGroup == "" {
		return nil, fmt.Errorf("corpus: grouped review requires manifest source id, revision, speaker, and accent metadata")
	}
	if len(all) != len(members) {
		return nil, fmt.Errorf("corpus: grouped review must include every source variant (expected %d, received %d)", len(all), len(members))
	}
	transcriptDigest := ReferenceDigest(base.ReferenceText)
	if base.ManifestTranscriptSHA256 == "" || transcriptDigest != base.ManifestTranscriptSHA256 {
		return nil, fmt.Errorf("corpus: source transcript differs from its manifest digest; grouped review is unavailable")
	}
	for _, clip := range selected {
		if clip.CorpusRevision != base.CorpusRevision || clip.SpeakerPseudonym != base.SpeakerPseudonym || clip.AccentGroup != base.AccentGroup {
			return nil, fmt.Errorf("corpus: source variants have mismatched revision, speaker, or accent metadata")
		}
		if ReferenceDigest(clip.ReferenceText) != transcriptDigest {
			return nil, fmt.Errorf("corpus: source variants have mismatched transcripts")
		}
		if clip.ManifestTranscriptSHA256 != base.ManifestTranscriptSHA256 {
			return nil, fmt.Errorf("corpus: source variants have mismatched manifest transcript provenance")
		}
		member := byID[clip.ID]
		if member.ExpectedReferenceSHA256 != transcriptDigest {
			return nil, ErrStaleReference{ID: clip.ID}
		}
	}
	review := ReferenceReview{Reviewer: strings.TrimSpace(reviewer), Protocol: strings.TrimSpace(protocol), ReviewedAt: s.clock.Now().UTC().Format(time.RFC3339Nano), ReferenceSHA256: ReferenceDigest(referenceText), ReviewGroupID: uuid.NewString()}
	if err := review.Validate(referenceText); err != nil {
		return nil, err
	}
	if base.LineageManifestSHA256 == "" {
		return nil, fmt.Errorf("corpus: source lineage has no verified manifest evidence")
	}
	return s.repo.ReviewReferenceGroup(ctx, members, base.SourceID, base.CorpusRevision, base.SpeakerPseudonym, base.AccentGroup, base.ManifestTranscriptSHA256, base.LineageManifestSHA256, referenceText, review)
}

func (s *Service) BackfillSourceLineage(ctx context.Context, entries []LineageEntry, dryRun bool) (int, error) {
	return s.repo.BackfillSourceLineage(ctx, entries, dryRun)
}

// DeleteClip removes the metadata row and its blob. The row is deleted
// first (the source of truth); a best-effort blob delete follows so a
// missing blob never blocks removing the clip from the corpus.
func (s *Service) DeleteClip(ctx context.Context, id string) error {
	clip, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.blobs.Delete(ctx, clip.BlobKey)
	return nil
}

// blobKey builds the opaque hierarchical key clips/<yyyy-mm>/<uuid>.<ext>.
func blobKey(id, format string, now time.Time) string {
	return fmt.Sprintf("clips/%s/%s.%s", now.Format("2006-01"), id, extForFormat(format))
}

// extForFormat maps an audio format hint to a file extension for the key.
func extForFormat(format string) string {
	switch format {
	case "pcm_s16le", "pcm":
		return "pcm"
	case "wav":
		return "wav"
	case "webm":
		return "webm"
	case "opus":
		return "opus"
	case "":
		return "bin"
	default:
		return format
	}
}

// mimeForFormat maps an audio format hint to a MIME type for the blob meta.
func mimeForFormat(format string) string {
	switch format {
	case "pcm_s16le", "pcm":
		return "audio/L16"
	case "wav":
		return "audio/wav"
	case "webm":
		return "audio/webm"
	case "opus":
		return "audio/opus"
	default:
		return "application/octet-stream"
	}
}
