package integrations

import (
	"fmt"
	"time"
)

type Connection struct {
	ID, Provider, DisplayName, SourceKind, Status, HealthMessage string
	ReadOnly                                                     bool
	CalendarCount, ImportedEventCount, BusyMinutes               int
	Revision                                                     int64
	LastSyncAt                                                   time.Time
}

type SyncInput struct {
	ID               string
	ExpectedRevision int64
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrConflict struct{ ID string }

func (e ErrConflict) Error() string {
	return fmt.Sprintf("integration connection %q changed; reload before retrying", e.ID)
}

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("integration connection %q not found", e.ID) }
