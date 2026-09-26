package authoring

import "context"

type Repository interface {
	SaveSession(ctx context.Context, session Session) error
	GetSession(ctx context.Context, id string) (Session, error)
	DeleteSession(ctx context.Context, id string) error
	SavePage(ctx context.Context, page PageDraft) error
	ListPages(ctx context.Context, sessionID string) ([]PageDraft, error)
}

type Session struct {
	ID         string
	Scenario   string
	TargetPath string
	Status     string
	CreatedAt  string
	UpdatedAt  string
}

type PageDraft struct {
	SessionID string
	PageID    string
	Path      string
	Title     string
	Status    string
	JSON      string
	UpdatedAt string
}
