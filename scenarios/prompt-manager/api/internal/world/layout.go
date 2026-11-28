package world

import (
	"fmt"
	"path/filepath"
	"time"

	"prompt-manager/internal/store"
)

// Vec2 is a point on the slab (metres).
type Vec2 struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
}

// Override adjusts one generated place by id.
type Override struct {
	PlaceID  string   `json:"placeId"`
	Position *Vec2    `json:"position,omitempty"`
	Rotation *float64 `json:"rotation,omitempty"`
	Removed  bool     `json:"removed,omitempty"`
}

// Decor is an operator-placed prop outside the generated layout.
type Decor struct {
	ID       string  `json:"id"`
	PropID   string  `json:"propId"`
	Position Vec2    `json:"position"`
	Rotation float64 `json:"rotation"`
	Scale    float64 `json:"scale"`
}

// Layout is the per-scene operator edit set applied over the generated layout.
type Layout struct {
	Scene     string     `json:"scene"`
	Overrides []Override `json:"overrides"`
	Decor     []Decor    `json:"decor"`
	UpdatedAt string     `json:"updatedAt,omitempty"`
}

const (
	maxOverrides  = 2000
	maxDecor      = 2000
	minDecorScale = 0.1
	maxDecorScale = 10
)

// Validate checks ids and bounds; positions are not clamped here because the
// slab size is a UI-side layout output.
func (l Layout) Validate() error {
	if !validScenes[l.Scene] {
		return fmt.Errorf("scene must be park or office, got %q", l.Scene)
	}
	if len(l.Overrides) > maxOverrides {
		return fmt.Errorf("too many overrides: %d", len(l.Overrides))
	}
	if len(l.Decor) > maxDecor {
		return fmt.Errorf("too many decor entries: %d", len(l.Decor))
	}
	seen := map[string]bool{}
	for i, o := range l.Overrides {
		if o.PlaceID == "" {
			return fmt.Errorf("overrides[%d]: placeId is required", i)
		}
		if seen[o.PlaceID] {
			return fmt.Errorf("overrides[%d]: duplicate placeId %q", i, o.PlaceID)
		}
		seen[o.PlaceID] = true
	}
	ids := map[string]bool{}
	for i, d := range l.Decor {
		if d.ID == "" || d.PropID == "" {
			return fmt.Errorf("decor[%d]: id and propId are required", i)
		}
		if ids[d.ID] {
			return fmt.Errorf("decor[%d]: duplicate id %q", i, d.ID)
		}
		ids[d.ID] = true
		if d.Scale < minDecorScale || d.Scale > maxDecorScale {
			return fmt.Errorf("decor[%d]: scale must be between %v and %v", i, minDecorScale, maxDecorScale)
		}
	}
	return nil
}

func (s *Store) layoutPath(scene string) string {
	return filepath.Join(s.dir, "layout-"+scene+".json")
}

// LoadLayout returns the saved layout for a scene, or an empty one.
func (s *Store) LoadLayout(scene string) (Layout, error) {
	if !validScenes[scene] {
		return Layout{}, fmt.Errorf("scene must be park or office, got %q", scene)
	}
	path := s.layoutPath(scene)
	if !store.FileExists(path) {
		return Layout{Scene: scene, Overrides: []Override{}, Decor: []Decor{}}, nil
	}
	loaded, err := store.LoadJSON[Layout](path)
	if err != nil {
		return Layout{}, err
	}
	if loaded.Overrides == nil {
		loaded.Overrides = []Override{}
	}
	if loaded.Decor == nil {
		loaded.Decor = []Decor{}
	}
	if err := loaded.Validate(); err != nil {
		return Layout{}, fmt.Errorf("%s: %w", path, err)
	}
	return *loaded, nil
}

// SaveLayout validates and writes a scene layout, stamping updatedAt.
func (s *Store) SaveLayout(layout Layout) (Layout, error) {
	if layout.Overrides == nil {
		layout.Overrides = []Override{}
	}
	if layout.Decor == nil {
		layout.Decor = []Decor{}
	}
	if err := layout.Validate(); err != nil {
		return Layout{}, err
	}
	layout.UpdatedAt = s.now().Format(time.RFC3339)
	if err := store.SaveJSON(s.layoutPath(layout.Scene), &layout); err != nil {
		return Layout{}, err
	}
	return layout, nil
}
