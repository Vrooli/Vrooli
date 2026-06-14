package roadmap

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	roadmapv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type Sector struct {
	Slug        string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Milestone struct {
	ID                string
	Name              string
	Description       string
	RequiredScenarios []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ProgressFilter struct {
	Sector string
	Tier   string
}

type ErrInvalidArgument struct {
	Field  string
	Reason string
}

func (e ErrInvalidArgument) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}

func NormalizeID(field, value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", ErrInvalidArgument{Field: field, Reason: "is required"}
	}
	if !idPattern.MatchString(value) {
		return "", ErrInvalidArgument{Field: field, Reason: "must use lowercase letters, numbers, and hyphens"}
	}
	return value, nil
}

func NormalizeTier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	switch value {
	case "foundation", "operational", "analytics", "integration", "digital_twin":
		return value, nil
	default:
		return "", ErrInvalidArgument{Field: "tier", Reason: "must be one of foundation, operational, analytics, integration, digital_twin"}
	}
}

func ProgressBucketKey(sector, tier string) string {
	return sector + "\x00" + tier
}

func DefaultSectorName(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func normalizeStability(stabilities []string) string {
	for _, stability := range stabilities {
		switch strings.TrimSpace(stability) {
		case "stable":
			return "stable"
		case "beta":
			return "beta"
		case "experimental":
			return "experimental"
		}
	}
	return ""
}

func ProgressBucketProto(sector, tier string) *roadmapv1.ProgressBucket {
	return &roadmapv1.ProgressBucket{Sector: sector, Tier: tier}
}
