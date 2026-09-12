// Package book owns custody context and the single-beneficiary boundary.
package book

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

var ErrInvalid = errors.New("invalid book")

type Book struct {
	ID                  string
	Name                string
	BeneficiaryIdentity string
	CreatedAt           time.Time
}

type CreateInput struct {
	ID                  string
	Name                string
	BeneficiaryIdentity string
}

type Service interface {
	Create(context.Context, CreateInput) (Book, error)
	Get(context.Context, string) (Book, error)
}

type service struct {
	repository Repository
	clock      schedule.Clock
}

func NewService(repository Repository, clock schedule.Clock) Service {
	return &service{repository: repository, clock: clock}
}

func (s *service) Create(ctx context.Context, in CreateInput) (Book, error) {
	if s.repository == nil || s.clock == nil {
		return Book{}, fmt.Errorf("%w: service dependencies are required", ErrInvalid)
	}
	in.ID = strings.TrimSpace(in.ID)
	in.Name = strings.TrimSpace(in.Name)
	in.BeneficiaryIdentity = strings.TrimSpace(in.BeneficiaryIdentity)
	if in.ID == "" {
		return Book{}, fmt.Errorf("%w: id is required", ErrInvalid)
	}
	if in.Name == "" {
		return Book{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if in.BeneficiaryIdentity == "" {
		return Book{}, fmt.Errorf("%w: beneficiary_identity is required", ErrInvalid)
	}
	return s.repository.Create(ctx, Book{
		ID:                  in.ID,
		Name:                in.Name,
		BeneficiaryIdentity: in.BeneficiaryIdentity,
		CreatedAt:           s.clock.Now().UTC(),
	})
}

func (s *service) Get(ctx context.Context, id string) (Book, error) {
	if s.repository == nil {
		return Book{}, fmt.Errorf("%w: repository is required", ErrInvalid)
	}
	return s.repository.Get(ctx, strings.TrimSpace(id))
}

var _ Service = (*service)(nil)
