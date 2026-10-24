package journal

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingPersona = errors.New("persona_id is required")
	ErrMissingVerb    = errors.New("journal verb is required")
)

type Entry struct {
	ID               string
	PersonaID        string
	Actor            string
	Verb             string
	RunID            string
	AuthorisingHuman string
	At               time.Time
	Outcome          string
	Constraint       string
	Details          map[string]string
}

type Service interface {
	Append(context.Context, Entry) (Entry, error)
	List(context.Context, string, int) ([]Entry, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

var _ Service = (*service)(nil)

func (s *service) Append(ctx context.Context, entry Entry) (Entry, error) {
	if strings.TrimSpace(entry.PersonaID) == "" {
		return Entry{}, ErrMissingPersona
	}
	if strings.TrimSpace(entry.Verb) == "" {
		return Entry{}, ErrMissingVerb
	}
	return s.repo.Append(ctx, entry)
}

func (s *service) List(ctx context.Context, personaID string, limit int) ([]Entry, error) {
	if strings.TrimSpace(personaID) == "" {
		return nil, ErrMissingPersona
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	return s.repo.List(ctx, personaID, limit)
}

// Schema returns the append-only journal's SQL contribution.
func Schema() string { return schemaSQL }
