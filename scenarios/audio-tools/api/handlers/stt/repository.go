package stt

import (
	"context"

	"audio-tools/internal/store"
)

// seam: STTStreamConfigRepository / WakewordRepository / SpeakerRepository
// are the consumer-side persistence seams for the STT admin handlers.
// Production wires the concrete sqlite store types; tests wire
// handlers/stt/mocks fakes.
type STTStreamConfigRepository interface {
	Get(ctx context.Context) (string, bool, error)
	Set(ctx context.Context, configJSON string) error
}

// SpeakerConfigRepository persists the speaker-verification config doc
// (mode/threshold/profile bindings) so it survives a restart. The enrolled
// profiles themselves persist via SpeakerRepository.
type SpeakerConfigRepository interface {
	Get(ctx context.Context) (string, bool, error)
	Set(ctx context.Context, configJSON string) error
}

type WakewordRepository interface {
	Upsert(ctx context.Context, t store.WakeWordTemplate) error
	Delete(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (store.WakeWordTemplate, bool, error)
	List(ctx context.Context) ([]store.WakeWordTemplate, error)
}

type SpeakerRepository interface {
	Upsert(ctx context.Context, p store.SpeakerProfile) error
	Delete(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (store.SpeakerProfile, bool, error)
	List(ctx context.Context) ([]store.SpeakerProfile, error)
}

var (
	_ STTStreamConfigRepository = (*store.STTStreamConfigStore)(nil)
	_ SpeakerConfigRepository   = (*store.STTSpeakerConfigStore)(nil)
	_ WakewordRepository        = (*store.WakeWordStore)(nil)
	_ SpeakerRepository         = (*store.SpeakerStore)(nil)
)
