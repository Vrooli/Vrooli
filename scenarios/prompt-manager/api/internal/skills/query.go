// Package skills provides the core domain types and operations for skill management.
package skills

import (
	"fmt"
	"strings"
)

// FilterOptions defines criteria for filtering skills.
type FilterOptions struct {
	Tag    string   // Filter by tag (empty = no filter)
	Folder string   // Filter by folder (empty = no filter)
	Modes  []string // Filter by modes (empty = no filter)
	// WithoutProgrammaticHome, when true, keeps only skills whose
	// ProgrammaticHome is nil/empty — i.e. detection has NOT yet graduated to a
	// programmatic engine. This is the quality-auditor's "frontier" query:
	// combined with Modes:["steer"] it yields the live steer-skill rotation.
	WithoutProgrammaticHome bool
}

// Filter applies all filter criteria to a list of skills.
// Domain logic: how skills are filtered based on various criteria.
func Filter(skills []Metadata, opts FilterOptions) []Metadata {
	result := skills

	if opts.Tag != "" {
		result = filterByTag(result, opts.Tag)
	}

	if opts.Folder != "" {
		result = filterByFolder(result, opts.Folder)
	}

	if len(opts.Modes) > 0 {
		result = filterByModes(result, opts.Modes)
	}

	if opts.WithoutProgrammaticHome {
		result = filterWithoutProgrammaticHome(result)
	}

	return result
}

// filterWithoutProgrammaticHome keeps only skills whose ProgrammaticHome is
// unset (nil or empty string) — the not-yet-graduated frontier set.
func filterWithoutProgrammaticHome(skills []Metadata) []Metadata {
	var filtered []Metadata
	for _, p := range skills {
		if p.ProgrammaticHome == nil || strings.TrimSpace(*p.ProgrammaticHome) == "" {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// filterByTag filters skills to only those containing the specified tag.
func filterByTag(skills []Metadata, tag string) []Metadata {
	var filtered []Metadata
	for _, p := range skills {
		for _, t := range p.Tags {
			if t == tag {
				filtered = append(filtered, p)
				break
			}
		}
	}
	return filtered
}

// filterByFolder filters skills to only those in the specified folder.
func filterByFolder(skills []Metadata, folder string) []Metadata {
	var filtered []Metadata
	for _, p := range skills {
		if strings.HasPrefix(p.File, folder+"/") {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// filterByModes filters skills to only those matching any of the specified modes.
func filterByModes(skills []Metadata, modes []string) []Metadata {
	var filtered []Metadata
	for _, p := range skills {
		for _, mode := range modes {
			for _, pm := range p.Modes {
				if pm == mode {
					filtered = append(filtered, p)
					goto nextSkill
				}
			}
		}
	nextSkill:
	}
	return filtered
}

// Slugify converts a string to a URL-safe slug.
// Used for generating skill IDs from names.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Remove consecutive hyphens
	slug := result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")

	return slug
}

const (
	MaxIDSuffixAttempts   = 100
	DefaultFallbackPrefix = "skill"
)

// GenerateUniqueID creates a unique ID by appending numeric suffixes if needed.
func GenerateUniqueID(name string, idExists func(id string) bool) (string, error) {
	baseID := Slugify(name)

	// Handle empty slug (e.g., name was "!!!")
	if baseID == "" {
		baseID = DefaultFallbackPrefix
	}

	// Try base ID first
	if !idExists(baseID) {
		return baseID, nil
	}

	// Try suffixed versions
	for i := 1; i <= MaxIDSuffixAttempts; i++ {
		candidateID := fmt.Sprintf("%s-%d", baseID, i)
		if !idExists(candidateID) {
			return candidateID, nil
		}
	}

	return "", fmt.Errorf("unable to generate unique ID after %d attempts", MaxIDSuffixAttempts)
}
