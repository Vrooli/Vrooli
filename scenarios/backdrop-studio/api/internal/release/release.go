package release

import (
	"context"
	"fmt"
	"sync"

	"backdrop-studio/internal/catalog"
)

type Request struct {
	CandidateID, StyleID, Strategy, SurfaceID, Placement, AltText string
	Width, Height, ExpectedWidth, ExpectedHeight                  int
	Decorative, AIGeneratedSet, AIGenerated, LegibilityPasses     bool
	ContrastRatio, ContrastThreshold                              float64
	Regions                                                       []catalog.Region
	ImagePNG                                                      []byte
}
type Backdrop struct {
	ID, CandidateID, StyleID, SurfaceID, Placement, AltText string
	Width, Height                                           int
	Decorative, AIGenerated                                 bool
	ContrastRatio, ContrastThreshold                        float64
	Regions                                                 []catalog.Region
	ImagePNG                                                []byte
	AssetStudioRef                                          string
}
type AssetPublisher interface {
	Publish(context.Context, Request, bool) (string, error)
}
type Store struct {
	mu        sync.RWMutex
	items     map[string]Backdrop
	publisher AssetPublisher
}

func NewStore() *Store { return &Store{items: map[string]Backdrop{}} }
func NewStoreWithPublisher(publisher AssetPublisher) *Store {
	return &Store{items: map[string]Backdrop{}, publisher: publisher}
}

func (s *Store) Release(r Request) (Backdrop, error) {
	if r.CandidateID == "" || r.StyleID == "" {
		return Backdrop{}, fmt.Errorf("release: candidate_id and style_id are required")
	}
	if r.AIGeneratedSet {
		return Backdrop{}, fmt.Errorf("release: ai_generated is derived and cannot be set directly")
	}
	if !r.Decorative && r.AltText == "" {
		return Backdrop{}, fmt.Errorf("release: alt_text is required, or set decorative=true")
	}
	if r.ExpectedWidth > 0 && (r.Width != r.ExpectedWidth || r.Height != r.ExpectedHeight) {
		return Backdrop{}, fmt.Errorf("release: dimensions mismatch: expected %dx%d, got %dx%d", r.ExpectedWidth, r.ExpectedHeight, r.Width, r.Height)
	}
	if !r.LegibilityPasses {
		return Backdrop{}, fmt.Errorf("release: legibility verdict is absent or failing (ratio %.3f, threshold %.3f)", r.ContrastRatio, r.ContrastThreshold)
	}
	aig := r.Strategy == "guided" || r.Strategy == "synthesized"
	assetRef := ""
	if aig {
		if s.publisher == nil {
			return Backdrop{}, fmt.Errorf("release: model-backed candidate requires asset-studio publisher capability")
		}
		var err error
		assetRef, err = s.publisher.Publish(context.Background(), r, true)
		if err != nil {
			return Backdrop{}, fmt.Errorf("release: asset-studio handoff: %w", err)
		}
	}
	id := fmt.Sprintf("backdrop-%s", r.CandidateID)
	b := Backdrop{ID: id, CandidateID: r.CandidateID, StyleID: r.StyleID, SurfaceID: r.SurfaceID, Placement: r.Placement, AltText: r.AltText, Width: r.Width, Height: r.Height, Decorative: r.Decorative, AIGenerated: aig, ContrastRatio: r.ContrastRatio, ContrastThreshold: r.ContrastThreshold, Regions: append([]catalog.Region(nil), r.Regions...), ImagePNG: append([]byte(nil), r.ImagePNG...), AssetStudioRef: assetRef}
	s.mu.Lock()
	s.items[id] = b
	s.mu.Unlock()
	return b, nil
}

func (s *Store) Get(id string) (Backdrop, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.items[id]
	if !ok {
		return Backdrop{}, fmt.Errorf("release: backdrop %q not found", id)
	}
	return b, nil
}
