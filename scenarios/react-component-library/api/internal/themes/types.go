// Package themes owns the preview iframe's theme registry. The host UI
// picks either a built-in theme (seeded once at boot into
// `builtin_themes`) or a theme derived on demand from a target
// scenario's DESIGN.md YAML front-matter. Tokens are a flat map of
// CSS custom-property → value; the harness applies them as `:root`
// variables before mounting the previewed component.
//
// Layering:
//
//	HTTP → handler → Service → Repository (sqlite, builtins only)
//	                       ↑
//	                       DesignMDReader (target scenarios' DESIGN.md)
package themes

import "fmt"

// Theme is the internal domain shape returned by every service verb.
type Theme struct {
	ID     string
	Name   string
	Tokens map[string]string // CSS custom-property → value (keys carry the leading "--")
	Source string            // "builtin" or "scenario:<id>"
}

// ErrThemeNotFound is the typed sentinel handlers translate to a 404.
type ErrThemeNotFound struct {
	ID string
}

func (e ErrThemeNotFound) Error() string {
	return fmt.Sprintf("theme %q not found", e.ID)
}

// ErrInvalidDesignMD is the typed sentinel ResolveFromScenario returns
// when the target scenario's DESIGN.md cannot be parsed into a
// recognisable theme shape. Handlers translate to 422 (Unprocessable).
type ErrInvalidDesignMD struct {
	Scenario string
	Reason   string
}

func (e ErrInvalidDesignMD) Error() string {
	return fmt.Sprintf("scenario %q DESIGN.md is not a valid theme: %s", e.Scenario, e.Reason)
}

// ErrScenarioDesignMDMissing is returned when the target scenario does
// not have a DESIGN.md to resolve.
type ErrScenarioDesignMDMissing struct {
	Scenario string
	Cause    error
}

func (e ErrScenarioDesignMDMissing) Error() string {
	return fmt.Sprintf("scenario %q DESIGN.md missing: %v", e.Scenario, e.Cause)
}

func (e ErrScenarioDesignMDMissing) Unwrap() error { return e.Cause }
