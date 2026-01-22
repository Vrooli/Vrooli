// Package prompts provides the core domain types and operations for prompt management.
package prompts

import "strings"

// FilterOptions defines criteria for filtering prompts.
type FilterOptions struct {
	Tag    string   // Filter by tag (empty = no filter)
	Folder string   // Filter by folder (empty = no filter)
	Modes  []string // Filter by modes (empty = no filter)
}

// Filter applies all filter criteria to a list of prompts.
// Domain logic: how prompts are filtered based on various criteria.
func Filter(prompts []Metadata, opts FilterOptions) []Metadata {
	result := prompts

	if opts.Tag != "" {
		result = filterByTag(result, opts.Tag)
	}

	if opts.Folder != "" {
		result = filterByFolder(result, opts.Folder)
	}

	if len(opts.Modes) > 0 {
		result = filterByModes(result, opts.Modes)
	}

	return result
}

// filterByTag filters prompts to only those containing the specified tag.
func filterByTag(prompts []Metadata, tag string) []Metadata {
	var filtered []Metadata
	for _, p := range prompts {
		for _, t := range p.Tags {
			if t == tag {
				filtered = append(filtered, p)
				break
			}
		}
	}
	return filtered
}

// filterByFolder filters prompts to only those in the specified folder.
func filterByFolder(prompts []Metadata, folder string) []Metadata {
	var filtered []Metadata
	for _, p := range prompts {
		if strings.HasPrefix(p.File, folder+"/") {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// filterByModes filters prompts to only those matching any of the specified modes.
func filterByModes(prompts []Metadata, modes []string) []Metadata {
	var filtered []Metadata
	for _, p := range prompts {
		for _, mode := range modes {
			for _, pm := range p.Modes {
				if pm == mode {
					filtered = append(filtered, p)
					goto nextPrompt
				}
			}
		}
	nextPrompt:
	}
	return filtered
}

// Slugify converts a string to a URL-safe slug.
// Used for generating prompt IDs from names.
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
