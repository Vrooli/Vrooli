package book

import (
	"context"
	"errors"
)

var (
	ErrNotFound            = errors.New("book not found")
	ErrBeneficiaryConflict = errors.New("book beneficiary conflicts with the operator beneficiary")
)

// Repository is the persistence seam for custody books.
type Repository interface {
	Create(context.Context, Book) (Book, error)
	Get(context.Context, string) (Book, error)
}
