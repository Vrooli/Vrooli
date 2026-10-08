package preferences

import (
	"context"
	"time"
)

// Preference is one user's saved dashboard configuration. The three JSON
// columns hold opaque documents supplied by the UI.
type Preference struct {
	ID                string
	UserID            string
	DefaultCollection string
	SavedQueries      string
	DashboardLayout   string
	AlertPreferences  string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Repository is the preferences domain's storage surface.
//
// No handler calls it yet — the table predates any feature that writes it (see
// docs/internal/STORAGE_AUDIT.md §2, "Dead tables"). It exists so the domain is
// complete and its DDL is exercised; deleting the feature means deleting this
// folder.
type Repository interface {
	Upsert(ctx context.Context, p Preference) error
	Get(ctx context.Context, userID string) (Preference, bool, error)
}
