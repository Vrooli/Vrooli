package tts

import (
	"context"

	"audio-tools/internal/store"
)

// seam: TTSConfigRepository is the tts-admin config persistence seam.
type TTSConfigRepository interface {
	Get(ctx context.Context) (configJSON, summarizeJSON string, ok bool, err error)
	Set(ctx context.Context, configJSON, summarizeJSON string) error
}

// seam: PlaybackRepository is the tts-admin playback event persistence seam.
type PlaybackRepository interface {
	Insert(ctx context.Context, e store.PlaybackEvent) error
	List(ctx context.Context, limit int) ([]store.PlaybackEvent, error)
}

var (
	_ TTSConfigRepository = (*store.TTSConfigStore)(nil)
	_ PlaybackRepository  = (*store.PlaybackStore)(nil)
)
