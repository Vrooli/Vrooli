package conflicts

import (
	"sort"
	"strings"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

const (
	SurfaceAny = "any"
	SurfaceAPI = "api"
	SurfaceCLI = "cli"
	SurfaceUI  = "ui"
)

// SurfaceProfile selects detectors for a surface/language pair. SurfaceAny and
// an empty Language act as wildcards. Profiles keep detector applicability near
// the registry instead of spreading surface checks across every detector.
type SurfaceProfile struct {
	Surface   string
	Language  graph.Language
	Detectors []string
}

// DefaultSurfaceProfiles is the Phase 5 production matrix. It intentionally
// keeps cycle, naming, and mislocated_file as the universal floor.
func DefaultSurfaceProfiles() []SurfaceProfile {
	return []SurfaceProfile{
		{Surface: SurfaceAny, Detectors: []string{"cycle", "glossary_drift", "mislocated_file", "naming"}},
		{Surface: SurfaceAPI, Language: graph.LanguageGo, Detectors: []string{
			"coupling_smell",
			"cross_scenario",
			"cycle",
			"convergence_drift",
			"domains_doc_parse_warning",
			"layering",
			"mislocated_file",
			"naming",
			"surface_coherence",
		}},
		{Surface: SurfaceCLI, Language: graph.LanguageGo, Detectors: []string{
			"cross_scenario",
			"cycle",
			"convergence_drift",
			"layering",
			"mislocated_file",
			"naming",
		}},
		{Surface: SurfaceUI, Language: graph.LanguageTypeScript, Detectors: []string{
			"cycle",
			"mislocated_file",
			"naming",
			"surface_coherence",
		}},
	}
}

func detectorsForProfiles(in DetectInput, profiles []SurfaceProfile) map[string]struct{} {
	active := activeSurfaceLanguages(in)
	out := map[string]struct{}{}
	for _, p := range profiles {
		if p.Surface == SurfaceAny {
			addDetectorNames(out, p.Detectors)
			continue
		}
		for _, sl := range active {
			if profileMatches(p, sl) {
				addDetectorNames(out, p.Detectors)
				break
			}
		}
	}
	return out
}

type surfaceLanguage struct {
	surface  string
	language graph.Language
}

func activeSurfaceLanguages(in DetectInput) []surfaceLanguage {
	seen := map[surfaceLanguage]struct{}{}
	for _, f := range in.Snapshot.Files {
		surface := surfaceForPath(f.Path)
		if surface == "" {
			continue
		}
		seen[surfaceLanguage{surface: surface, language: f.Language}] = struct{}{}
	}
	for _, p := range in.Snapshot.Packages {
		surface := surfaceForPath(p.RepoPath)
		if surface == "" {
			continue
		}
		seen[surfaceLanguage{surface: surface, language: p.Language}] = struct{}{}
	}
	for _, decl := range in.DomainMap.Declarations {
		surface := surfaceForSource(decl.Source)
		if surface == "" || len(decl.DomainNames) == 0 {
			continue
		}
		seen[surfaceLanguage{surface: surface, language: defaultLanguageForSurface(surface)}] = struct{}{}
	}
	for _, d := range in.DomainMap.Domains {
		for _, p := range d.Paths {
			surface := surfaceForPath(p)
			if surface == "" {
				continue
			}
			seen[surfaceLanguage{surface: surface, language: defaultLanguageForSurface(surface)}] = struct{}{}
		}
	}
	out := make([]surfaceLanguage, 0, len(seen))
	for sl := range seen {
		out = append(out, sl)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].surface != out[j].surface {
			return out[i].surface < out[j].surface
		}
		return out[i].language < out[j].language
	})
	return out
}

func profileMatches(p SurfaceProfile, sl surfaceLanguage) bool {
	if p.Surface != sl.surface {
		return false
	}
	return p.Language == "" || sl.language == "" || p.Language == sl.language
}

func addDetectorNames(out map[string]struct{}, names []string) {
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			out[n] = struct{}{}
		}
	}
}

func surfaceForSource(src domains.Source) string {
	switch src {
	case domains.SourceAPIFolders:
		return SurfaceAPI
	case domains.SourceCLIGroups:
		return SurfaceCLI
	case domains.SourceUIFeatures:
		return SurfaceUI
	default:
		return ""
	}
}

func surfaceForPath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	switch {
	case strings.HasPrefix(path, "api/"):
		return SurfaceAPI
	case strings.HasPrefix(path, "cli/"):
		return SurfaceCLI
	case strings.HasPrefix(path, "ui/"):
		return SurfaceUI
	default:
		return ""
	}
}

func defaultLanguageForSurface(surface string) graph.Language {
	switch surface {
	case SurfaceAPI, SurfaceCLI:
		return graph.LanguageGo
	case SurfaceUI:
		return graph.LanguageTypeScript
	default:
		return graph.LanguageUnspecified
	}
}
