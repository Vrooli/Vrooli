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

type WakewordRepository interface {
	Upsert(ctx context.Context, t store.WakeWordTemplate) error
	Delete(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (store.WakeWordTemplate, bool, error)
	List(ctx context.Context) ([]store.WakeWordTemplate, error)
}

type SpeakerRepository interface {
	Upsert(ctx context.Context, p store.SpeakerProfile) error
	Delete(ctx context.Context, id string) (bool, error)
	List(ctx context.Context) ([]store.SpeakerProfile, error)
}

var (
	_ STTStreamConfigRepository = (*store.STTStreamConfigStore)(nil)
	_ WakewordRepository        = (*store.WakeWordStore)(nil)
	_ SpeakerRepository         = (*store.SpeakerStore)(nil)
)
