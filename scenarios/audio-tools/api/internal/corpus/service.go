package corpus

import (
	"context"
	"fmt"
	"time"

	"audio-tools/internal/blobbytes"
	"audio-tools/internal/clock"

	"github.com/google/uuid"
)

// Service is the corpus CRUD domain logic: it ties the metadata Repository
// to the audio BlobBytes store so a clip's bytes and row are written and
// deleted together. It is the seam the CorpusService Connect handler and
// the eval harness build on.
type Service struct {
	repo  Repository
	blobs blobbytes.Store
	clock clock.Clock
}

// NewService constructs a corpus Service.
func NewService(repo Repository, blobs blobbytes.Store, clk clock.Clock) *Service {
	return &Service{repo: repo, blobs: blobs, clock: clk}
}

// CreateClipInput is the create payload: the raw audio bytes plus the
// metadata. The service generates the id + blob key.
type CreateClipInput struct {
	Audio         []byte
	ReferenceText string
	Tags          []string
	DurationMs    int64
	SampleRateHz  int
	Format        string
	Source        Source
}

// CreateClip writes the audio to the blob store, then the metadata row. On
// a metadata failure it rolls the blob back so an orphan blob can't leak.
func (s *Service) CreateClip(ctx context.Context, in CreateClipInput) (Clip, error) {
	if len(in.Audio) == 0 {
		return Clip{}, fmt.Errorf("corpus: CreateClip requires audio bytes")
	}
	now := s.clock.Now().UTC()
	id := uuid.NewString()
	key := blobKey(id, in.Format, now)

	if err := s.blobs.Put(ctx, key, in.Audio, mimeForFormat(in.Format)); err != nil {
		return Clip{}, fmt.Errorf("corpus: store audio: %w", err)
	}
	clip := Clip{
		ID:            id,
		ReferenceText: in.ReferenceText,
		Tags:          in.Tags,
		DurationMs:    in.DurationMs,
		SampleRateHz:  in.SampleRateHz,
		Format:        in.Format,
		BlobKey:       key,
		Source:        in.Source.Normalize(),
		CreatedAt:     now,
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
