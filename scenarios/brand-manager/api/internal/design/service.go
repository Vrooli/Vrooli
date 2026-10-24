package design

import (
	"context"
	"log"
	"strings"
)

// Service is the application-layer surface the design handlers depend on. It
// loads a brand and renders its canonical DESIGN.md document. The handler is
// intentionally thin around it: decode → call service → translate errors.
type Service interface {
	// GenerateDesignLanguage loads the brand and renders its DESIGN.md document.
	// Returns ErrInvalidDesign when brandID is blank, or ErrBrandNotFound when no
	// brand matches.
	GenerateDesignLanguage(ctx context.Context, brandID string) (Design, error)
}

type service struct {
	brands BrandStore
	logger *log.Logger
}

// NewService constructs the production Service. A nil logger defaults to
// log.Default().
func NewService(brands BrandStore, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	return &service{brands: brands, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) GenerateDesignLanguage(ctx context.Context, brandID string) (Design, error) {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return Design{}, ErrInvalidDesign{Field: "brand_id", Reason: "required"}
	}
	brand, err := s.brands.Get(ctx, brandID)
	if err != nil {
		return Design{}, err
	}
	return Design{
		BrandID:  brand.ID,
		Markdown: renderDesignLanguage(brand),
	}, nil
}
