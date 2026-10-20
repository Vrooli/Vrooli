package shots

import (
	"fmt"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type ErrInvalidShot struct {
	Field  string
	Reason string
}

func (e ErrInvalidShot) Error() string { return fmt.Sprintf("invalid shot %s: %s", e.Field, e.Reason) }

type ErrShotNotFound struct{ ID string }

func (e ErrShotNotFound) Error() string { return fmt.Sprintf("shot %q not found", e.ID) }

type ErrRevisionNotFound struct {
	ShotID   string
	Revision int64
}

func (e ErrRevisionNotFound) Error() string {
	return fmt.Sprintf("shot %q revision %d not found", e.ShotID, e.Revision)
}

type ErrRevisionConflict struct {
	ShotID   string
	Expected int64
	Current  int64
}

func (e ErrRevisionConflict) Error() string {
	return fmt.Sprintf("shot %q revision conflict: expected %d, current %d", e.ShotID, e.Expected, e.Current)
}

type ErrRenderBlocked struct {
	ShotID   string
	Revision int64
	Issues   []*shotsv1.ShotIssue
}

type ErrSourceMediaUnavailable struct{}

func (ErrSourceMediaUnavailable) Error() string { return "shot source media is unavailable" }

func (e ErrRenderBlocked) Error() string {
	return fmt.Sprintf("shot %q revision %d is not eligible for rendering", e.ShotID, e.Revision)
}
