package flows

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Anchor is a durable-safe visual checkpoint. Bounds are normalized to the
// capture dimensions, so a saved anchor can be replayed across device scales.
type Anchor struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Target     string    `json:"target"`
	Bounds     []float64 `json:"bounds"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

// AnchorStore is the resolver's small persistence seam. The control module can
// replace it with a database-backed implementation later without changing the
// flow or UI contract.
type AnchorStore struct {
	mu      sync.RWMutex
	anchors map[string]Anchor
}

func NewAnchorStore() *AnchorStore { return &AnchorStore{anchors: map[string]Anchor{}} }

func (s *AnchorStore) Create(name, target string, bounds []float64, confidence float64) (Anchor, error) {
	if s == nil || strings.TrimSpace(name) == "" || strings.TrimSpace(target) == "" || !validBounds(bounds) {
		return Anchor{}, fmt.Errorf("name, target, and normalized bounds are required")
	}
	if !finite(confidence) || confidence < 0 || confidence > 1 {
		return Anchor{}, fmt.Errorf("confidence must be between 0 and 1")
	}
	a := Anchor{ID: uuid.NewString(), Name: strings.TrimSpace(name), Target: strings.TrimSpace(target), Bounds: append([]float64(nil), bounds...), Confidence: confidence, CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.anchors[a.ID] = a
	s.mu.Unlock()
	return a, nil
}

func (s *AnchorStore) List() []Anchor {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Anchor, 0, len(s.anchors))
	for _, a := range s.anchors {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *AnchorStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.anchors[id]; !ok {
		return fmt.Errorf("anchor %q not found", id)
	}
	delete(s.anchors, id)
	return nil
}

func (s *AnchorStore) Resolve(target string) (Anchor, bool) {
	for _, a := range s.List() {
		if a.Name == target || a.Target == target {
			return a, true
		}
	}
	return Anchor{}, false
}
