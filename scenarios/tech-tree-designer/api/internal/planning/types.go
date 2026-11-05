package planning

import (
	"fmt"
	"strings"
	"time"

	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning"
)

const (
	DefaultTargetStability = "experimental"
	TransportWorldPlanned  = "none"
)

type Scenario struct {
	Slug            string
	DisplayName     string
	Sector          string
	Tier            string
	TargetStability string
	Files           []ProtoFile
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ProtoFile struct {
	Path      string
	Text      string
	UpdatedAt time.Time
}

type CreateInput struct {
	Slug            string
	DisplayName     string
	Sector          string
	Tier            string
	TargetStability string
}

type ListFilter struct {
	Sector string
	Tier   string
}

type PutFileInput struct {
	Slug string
	Path string
	Text string
}

type MaterializeResult struct {
	Slug         string
	WrittenPaths []string
	Generated    bool
}

type PlanFinding struct {
	Severity   planningv1.PlanFindingSeverity
	Code       string
	Location   string
	Message    string
	Suggestion string
}

func NormalizeSlug(slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", ErrInvalidArgument{Field: "slug", Reason: "required"}
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return "", ErrInvalidArgument{Field: "slug", Reason: "use lowercase letters, numbers, and hyphens"}
	}
	return slug, nil
}

func NormalizeProtoPath(path string) (string, error) {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", ErrInvalidArgument{Field: "path", Reason: "required"}
	}
	if strings.Contains(path, "..") || strings.HasPrefix(path, ".") {
		return "", ErrInvalidArgument{Field: "path", Reason: "must stay inside the planned proto tree"}
	}
	if !strings.HasSuffix(path, ".proto") {
		return "", ErrInvalidArgument{Field: "path", Reason: "must end with .proto"}
	}
	return path, nil
}

func DefaultDisplayName(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type ErrInvalidArgument struct {
	Field  string
	Reason string
}

func (e ErrInvalidArgument) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

type ErrScenarioNotFound struct{ Slug string }

func (e ErrScenarioNotFound) Error() string {
	return fmt.Sprintf("planned scenario %q not found", e.Slug)
}

type ErrProtoFileNotFound struct {
	Slug string
	Path string
}

func (e ErrProtoFileNotFound) Error() string {
	return fmt.Sprintf("planned proto file %q for scenario %q not found", e.Path, e.Slug)
}
