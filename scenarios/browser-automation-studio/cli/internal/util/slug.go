package util

import (
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(input string) string {
	slug := strings.ToLower(strings.TrimSpace(input))
	slug = slugPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "workflow"
	}
	return slug
}
