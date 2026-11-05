package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Service interface {
	Create(context.Context, CreateInput) (Workspace, error)
	List(context.Context, string) ([]Workspace, error)
	Get(context.Context, string, string) (Workspace, error)
}
type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo} }
func (s *service) Create(ctx context.Context, in CreateInput) (Workspace, error) {
	if err := ValidateName(in.Name); err != nil {
		return Workspace{}, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if strings.TrimSpace(in.OwnerSubject) == "" {
		return Workspace{}, fmt.Errorf("owner subject is required")
	}
	return s.repo.Create(ctx, Workspace{ID: uuid.NewString(), Name: in.Name, OwnerSubject: in.OwnerSubject, IdempotencyKey: strings.TrimSpace(in.IdempotencyKey), RequestHash: RequestHash(in.Name)})
}

func (s *service) List(ctx context.Context, owner string) ([]Workspace, error) {
	if owner == "" {
		return nil, fmt.Errorf("owner subject is required")
	}
	return s.repo.List(ctx, owner)
}

func (s *service) Get(ctx context.Context, id, owner string) (Workspace, error) {
	if id == "" {
		return Workspace{}, ErrNotFound{id}
	}
	return s.repo.Get(ctx, id, owner)
}

func RequestHash(name string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(name)))
	return hex.EncodeToString(h[:])
}
