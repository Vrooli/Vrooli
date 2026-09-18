package feedback

import (
	"context"
	"fmt"
	"time"
)

type Feedback struct {
	WorkspaceID, Date, RecipeID, Portion string
	Minutes                              int
	RecordedAt                           time.Time
}
type Repository interface {
	Record(context.Context, Feedback) (Feedback, error)
	Get(context.Context, string, string) (Feedback, error)
	Undo(context.Context, string, string) error
}
type ErrNotFound struct{ WorkspaceID, Date string }

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("feedback for workspace %q date %q not found", e.WorkspaceID, e.Date)
}
