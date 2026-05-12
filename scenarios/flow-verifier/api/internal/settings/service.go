package settings

import "context"

// Service is the application-level seam over Repository. It owns the
// merge-and-validate step: a partial Patch from the handler is merged
// into the current row (or DefaultSettings if no row exists) and the
// result is validated before being persisted.
type Service struct {
	repo Repository
}

// NewService wires a Service over the provided repository.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Get returns the current settings for the local principal. Never
// returns ErrNotFound — Repository.Get falls back to DefaultSettings
// when the row is missing.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	return s.repo.Get(ctx, PrincipalLocal)
}

// Upsert merges a Patch into the stored row (or DefaultSettings if
// none) and persists the result. Returns ValidationError when the
// patch carries an unknown enum value; that error is the handler's
// signal to emit 400.
func (s *Service) Upsert(ctx context.Context, p Patch) (Settings, error) {
	current, err := s.repo.Get(ctx, PrincipalLocal)
	if err != nil {
		return Settings{}, err
	}
	merged := mergePatch(current, p)
	if err := validate(merged); err != nil {
		return Settings{}, err
	}
	return s.repo.Upsert(ctx, merged)
}

func mergePatch(base Settings, p Patch) Settings {
	if p.Theme != nil {
		base.Theme = *p.Theme
	}
	if p.FontScale != nil {
		base.FontScale = *p.FontScale
	}
	if p.ReducedMotion != nil {
		base.ReducedMotion = *p.ReducedMotion
	}
	if p.RTL != nil {
		base.RTL = *p.RTL
	}
	if p.DefaultRoot != nil {
		base.DefaultRoot = *p.DefaultRoot
	}
	if p.Density != nil {
		base.Density = *p.Density
	}
	if p.SidebarWidth != nil {
		base.SidebarWidth = *p.SidebarWidth
	}
	if p.InventoryFilters != nil {
		base.InventoryFilters = *p.InventoryFilters
	}
	return base
}

func validate(s Settings) error {
	switch s.Theme {
	case ThemeLight, ThemeDark, ThemeSystem:
	default:
		return ValidationError{Field: "theme", Value: string(s.Theme), Message: "must be one of light|dark|system"}
	}
	switch s.FontScale {
	case FontScaleSm, FontScaleMd, FontScaleLg:
	default:
		return ValidationError{Field: "fontScale", Value: string(s.FontScale), Message: "must be one of sm|md|lg"}
	}
	switch s.Density {
	case DensityComfortable, DensityCompact:
	default:
		return ValidationError{Field: "density", Value: string(s.Density), Message: "must be one of comfortable|compact"}
	}
	if s.SidebarWidth < 0 {
		return ValidationError{Field: "sidebarWidth", Value: itoa(s.SidebarWidth), Message: "must be non-negative"}
	}
	if s.InventoryFilters.Sort.Dir != "" && s.InventoryFilters.Sort.Dir != "asc" && s.InventoryFilters.Sort.Dir != "desc" {
		return ValidationError{Field: "inventoryFilters.sort.dir", Value: s.InventoryFilters.Sort.Dir, Message: "must be asc or desc"}
	}
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
