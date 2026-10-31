package projects

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type Service interface {
	Create(context.Context, CreateInput) (Project, error)
	Get(context.Context, string) (Project, error)
	List(context.Context, int32, string) ([]Project, string, error)
	Rename(context.Context, RenameInput) (Project, error)
}

type service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) Service {
	if now == nil {
		now = time.Now
	}
	return &service{repo: repo, now: now}
}

func (s *service) Create(ctx context.Context, in CreateInput) (Project, error) {
	name := strings.TrimSpace(in.Name)
	if err := ValidateName(name); err != nil {
		return Project{}, err
	}
	now := s.now().UTC()
	return s.repo.Create(ctx, Project{Name: name, CreatedAt: now, UpdatedAt: now, Revision: 1})
}

func (s *service) Get(ctx context.Context, id string) (Project, error) {
	if strings.TrimSpace(id) == "" {
		return Project{}, ErrInvalidProject{Field: "id", Reason: "required"}
	}
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, pageSize int32, token string) ([]Project, string, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var after *Project
	if token != "" {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			return nil, "", ErrInvalidProject{Field: "page_token", Reason: "invalid"}
		}
		var cursor Project
		if err = json.Unmarshal(raw, &cursor); err != nil || cursor.ID == "" || cursor.CreatedAt.IsZero() {
			return nil, "", ErrInvalidProject{Field: "page_token", Reason: "invalid"}
		}
		after = &cursor
	}
	rows, err := s.repo.List(ctx, int(pageSize)+1, after)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > int(pageSize) {
		rows = rows[:pageSize]
		raw, _ := json.Marshal(rows[len(rows)-1])
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return rows, next, nil
}

func (s *service) Rename(ctx context.Context, in RenameInput) (Project, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.Name = strings.TrimSpace(in.Name)
	if in.ID == "" {
		return Project{}, ErrInvalidProject{Field: "id", Reason: "required"}
	}
	if err := ValidateName(in.Name); err != nil {
		return Project{}, err
	}
	if in.ExpectedRevision <= 0 {
		return Project{}, ErrInvalidProject{Field: "expected_revision", Reason: "must be positive"}
	}
	return s.repo.Rename(ctx, in, in.Name, in.ExpectedRevision)
}
