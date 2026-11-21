package styles

import (
	"context"
	"errors"
	"strings"
)

// ConstellationMidnight is the shared container style of the star-named line,
// seeded from the approved Aquila tile.
const ConstellationMidnight = "constellation-midnight"

// StarLine is the first product line.
const StarLine = "Star line"

// Service is the application-layer surface for container styles and lines.
type Service struct {
	store Store
}

// NewService constructs the styles service.
func NewService(store Store) *Service { return &Service{store: store} }

func (s *Service) ListStyles(ctx context.Context) ([]ContainerStyle, error) {
	return s.store.ListStyles(ctx)
}

func (s *Service) GetStyle(ctx context.Context, id string) (ContainerStyle, error) {
	return s.store.GetStyle(ctx, id)
}

func (s *Service) GetStyleByName(ctx context.Context, name string) (ContainerStyle, error) {
	return s.store.GetStyleByName(ctx, name)
}

func (s *Service) CreateStyle(ctx context.Context, st ContainerStyle) (ContainerStyle, error) {
	if strings.TrimSpace(st.Name) == "" {
		return ContainerStyle{}, errors.New("styles: name is required")
	}
	return s.store.CreateStyle(ctx, st)
}

func (s *Service) UpdateStyle(ctx context.Context, st ContainerStyle) (ContainerStyle, error) {
	return s.store.UpdateStyle(ctx, st)
}

func (s *Service) ListProductLines(ctx context.Context) ([]ProductLine, error) {
	return s.store.ListProductLines(ctx)
}

func (s *Service) CreateProductLine(ctx context.Context, l ProductLine) (ProductLine, error) {
	if strings.TrimSpace(l.Name) == "" {
		return ProductLine{}, errors.New("styles: name is required")
	}
	return s.store.CreateProductLine(ctx, l)
}

// Seed inserts constellation-midnight and the Star line once each. It is
// idempotent: an existing row with the same name is left untouched, so a later
// operator edit is never overwritten.
func (s *Service) Seed(ctx context.Context) error {
	style, err := s.store.GetStyleByName(ctx, ConstellationMidnight)
	if isNotFound(err) {
		style, err = s.store.CreateStyle(ctx, ContainerStyle{
			Name:                 ConstellationMidnight,
			Shape:                "rounded_square",
			CornerRatio:          0.21875,
			BackgroundKind:       "gradient",
			BackgroundTop:        "#15243c",
			BackgroundBottom:     "#0b1728",
			MarkScale:            0.86,
			MaskableScale:        0.40,
			AccentColor:          "#22d3ee",
			Glow:                 []GlowLayer{{Width: 2, Opacity: 0.45}, {Width: 5, Opacity: 0.22}, {Width: 9, Opacity: 0.10}},
			SmallMarkThresholdPx: 32,
		})
	}
	if err != nil {
		return err
	}

	if _, err := s.store.GetProductLineByName(ctx, StarLine); isNotFound(err) {
		if _, cerr := s.store.CreateProductLine(ctx, ProductLine{
			Name:             StarLine,
			ContainerStyleID: style.ID,
			Products:         []string{"Aquila", "Vega", "Rigel", "Mizar", "Pan", "Messier"},
		}); cerr != nil {
			return cerr
		}
	} else if err != nil {
		return err
	}
	return nil
}

func isNotFound(err error) bool {
	var nf ErrNotFound
	return errors.As(err, &nf)
}
