package measures

import (
	"context"
	"time"

	"audio-tools/internal/logx"
	measuretypes "audio-tools/internal/measures"
	"audio-tools/internal/store"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/duplex"
)

type BYOKCounter interface {
	CountCreatedBetween(context.Context, time.Time, time.Time) (int64, error)
}

type CorpusCounter interface {
	CountClipsCreatedInWindow(context.Context, time.Time, time.Time) (int64, error)
}

type ExperimentCounter = measuretypes.ExperimentCounter

type SpeakerCounter interface {
	CountCreatedBetween(context.Context, time.Time, time.Time) (int64, error)
}

type WakewordCounter interface {
	CountCreatedBetween(context.Context, time.Time, time.Time) (int64, error)
}

type DuplexAccountingCounter interface {
	CountCreatedBetween(context.Context, time.Time, time.Time) (int64, error)
}

type Logger interface {
	Printf(string, ...any)
}

var (
	_ BYOKCounter             = (*byokstore.Store)(nil)
	_ SpeakerCounter          = (*store.SpeakerStore)(nil)
	_ WakewordCounter         = (*store.WakeWordStore)(nil)
	_ DuplexAccountingCounter = (*duplex.SQLJournal)(nil)
	_ Logger                  = (logx.Logger)(nil)
)
