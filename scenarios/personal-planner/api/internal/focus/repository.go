package focus

import "context"

type Repository interface {
	Create(context.Context, Session) (Session, error)
	Current(context.Context) (Session, bool, error)
	Get(context.Context, string) (Session, error)
	Transition(context.Context, Session, int64) (Session, error)
	SaveNote(context.Context, SessionNote) (SessionNote, error)
	ListNotes(context.Context, string) ([]SessionNote, error)
	RecordPauseEvent(context.Context, PauseEvent) (PauseEvent, error)
	ListPauseEvents(context.Context, string) ([]PauseEvent, error)
	CreateActual(context.Context, Actual) (Actual, error)
	ListActuals(context.Context, string) ([]Actual, error)
	ListCorrections(context.Context, string, string, int) ([]Correction, error)
	CorrectActual(context.Context, Actual, int64) (Actual, error)
}
