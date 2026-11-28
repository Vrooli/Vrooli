package profile

import (
	"fmt"
	"strings"
)

// Profile is the language/framework shape structure-health detected. Conformance
// rule packs key off ID; an unrecognized profile downgrades profile-conformance
// rules to advisory.
type Profile struct {
	ID              string
	BackendLanguage string
	UIFramework     string
	Recognized      bool
	Evidence        []string
	Surfaces        []Surface
	DegradedReason  string
}

// DefaultProfileID is the react-vite/Go shape every generated scenario uses; it
// reproduces today's structure rules verbatim.
const DefaultProfileID = "react-vite-go"

// Derive computes the profile from code facts: the backend language is the API
// (or worker/cli) surface's language, the UI framework is the UI surface's
// framework, and the profile id is their combination.
func Derive(f Facts) Profile {
	p := Profile{
		Surfaces:       f.Surfaces,
		DegradedReason: f.DegradedReason,
	}
	for _, s := range f.Surfaces {
		switch strings.ToLower(s.Kind) {
		case "api", "worker", "job":
			if p.BackendLanguage == "" && s.Language != "" && s.Language != "unknown" {
				p.BackendLanguage = s.Language
				p.Evidence = append(p.Evidence, fmt.Sprintf("backend %s from %s surface", s.Language, s.ID))
			}
		case "cli":
			if p.BackendLanguage == "" && s.Language != "" && s.Language != "unknown" {
				p.BackendLanguage = s.Language
				p.Evidence = append(p.Evidence, fmt.Sprintf("backend %s from cli surface %s", s.Language, s.ID))
			}
		case "ui":
			if p.UIFramework == "" && s.Framework != "" {
				p.UIFramework = s.Framework
				p.Evidence = append(p.Evidence, fmt.Sprintf("ui %s from %s surface", s.Framework, s.ID))
			}
		}
	}
	p.ID = profileID(p.BackendLanguage, p.UIFramework)
	p.Recognized = p.ID == DefaultProfileID
	if !p.Recognized {
		p.Evidence = append(p.Evidence, "profile not in the recognized set; profile-conformance rules are advisory")
	}
	return p
}

// HasUI reports whether the profile includes a UI surface.
func (p Profile) HasUI() bool {
	for _, s := range p.Surfaces {
		if strings.EqualFold(s.Kind, "ui") {
			return true
		}
	}
	return false
}

// SurfaceByKind returns the first detected surface of the given kind.
func (p Profile) SurfaceByKind(kind string) (Surface, bool) {
	for _, s := range p.Surfaces {
		if strings.EqualFold(s.Kind, kind) {
			return s, true
		}
	}
	return Surface{}, false
}

func profileID(backend, ui string) string {
	backend = strings.ToLower(strings.TrimSpace(backend))
	ui = strings.ToLower(strings.TrimSpace(ui))
	if backend == "go" && ui == "react-vite" {
		return DefaultProfileID
	}
	if backend == "" && ui == "" {
		return "unknown"
	}
	parts := make([]string, 0, 2)
	if ui != "" {
		parts = append(parts, ui)
	}
	if backend != "" {
		parts = append(parts, backend)
	}
	return strings.Join(parts, "-")
}
