// Package backlog types defines the core domain types, constants, and
// conversion functions for backlog items. These are pure data definitions
// with no HTTP or filesystem dependencies.
package backlog

import (
	"fmt"
	"path/filepath"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// BacklogStatus represents the lifecycle state of a backlog item.
type BacklogStatus string

const (
	StatusBacklog     BacklogStatus = "backlog"
	StatusResearching BacklogStatus = "researching"
	StatusReady       BacklogStatus = "ready"
	StatusQueued      BacklogStatus = "queued"
	StatusInProgress  BacklogStatus = "in_progress"
	StatusCompleted   BacklogStatus = "completed"
	StatusFailed      BacklogStatus = "failed"
	StatusArchived    BacklogStatus = "archived"
)

// BacklogKind represents a category of backlog work.
type BacklogKind string

const (
	KindIdea     BacklogKind = "idea"
	KindResearch BacklogKind = "research"
	KindFix      BacklogKind = "fix"
	KindExecute  BacklogKind = "execute"
	KindChore    BacklogKind = "chore"
)

// backlogKindDirs maps each BacklogKind to its on-disk directory name.
var backlogKindDirs = map[BacklogKind]string{
	KindIdea:     "ideas",
	KindResearch: "research",
	KindFix:      "fix",
	KindExecute:  "execute",
	KindChore:    "chore",
}

// BacklogItem represents a unit of work stored on disk.
type BacklogItem struct {
	Name            string        `json:"name"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	Status          BacklogStatus `json:"status"`
	Priority        int           `json:"priority"`
	Tags            []string      `json:"tags"`
	Created         string        `json:"created"`
	Updated         string        `json:"updated"`
	Kind            BacklogKind   `json:"kind"`
	ResearchTarget  string        `json:"research_target,omitempty"`
	DependsOn       []string      `json:"depends_on,omitempty"`
	Initiative      string        `json:"initiative,omitempty"`
	Effort          string        `json:"effort,omitempty"`
	Scope           string        `json:"scope,omitempty"`
	AcceptanceAllow []string      `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string      `json:"acceptance_deny,omitempty"`
}

// BacklogFile represents a file or directory within a backlog item folder.
type BacklogFile struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"` // "file" or "directory"
	Size     int64         `json:"size,omitempty,string"`
	Children []BacklogFile `json:"children,omitempty"`
}

// protectedBacklogFileName is the spec file that cannot be renamed or deleted
// through the file operation API.
const protectedBacklogFileName = "spec.json"

// ResearchMode describes the intent for backlog agent work.
type ResearchMode string

const (
	ResearchModeWorkshop   ResearchMode = "workshop"
	ResearchModeResearch   ResearchMode = "research"
	ResearchModeInitialize ResearchMode = "initialize"
)

// promptSelection holds the resolved skill ID, variables, and rendered prompt
// text returned by prompt-manager.
type promptSelection struct {
	SkillID   string
	Variables map[string]string
	Prompt    string
}

// ParseBacklogKind validates and normalizes a raw kind string.
func ParseBacklogKind(raw string) (BacklogKind, error) {
	candidate := BacklogKind(strings.ToLower(strings.TrimSpace(raw)))
	if _, ok := backlogKindDirs[candidate]; ok {
		return candidate, nil
	}
	return "", fmt.Errorf("%w: %s", ErrInvalidKind, raw)
}

// validateBacklogStatus returns true if the given status string is a known
// backlog status value.
func validateBacklogStatus(status string) bool {
	switch status {
	case "backlog", "researching", "ready", "queued", "in_progress", "completed", "failed", "archived":
		return true
	default:
		return false
	}
}

// normalizeResearchTarget validates and normalizes a research_target value.
func normalizeResearchTarget(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}
	switch value {
	case "idea", "fix", "execute", "unspecified":
		return value, nil
	default:
		return "", fmt.Errorf("research_target must be idea, fix, execute, or unspecified")
	}
}

// validateEffort checks that an effort value is one of the valid t-shirt sizes.
// Returns the normalized (uppercased) value, or an error if invalid.
func validateEffort(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}
	switch value {
	case "XS", "S", "M", "L", "XL":
		return value, nil
	default:
		return "", fmt.Errorf("effort must be XS, S, M, L, or XL")
	}
}

// validateScope checks that a scope path is safe (relative, no "..").
func validateScope(scope string) error {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil
	}
	if filepath.IsAbs(scope) {
		return fmt.Errorf("scope must be a relative path, got absolute: %s", scope)
	}
	// Check for ".." components in both raw and cleaned forms.
	for _, part := range strings.Split(scope, "/") {
		if part == ".." {
			return fmt.Errorf("scope must not contain '..': %s", scope)
		}
	}
	cleaned := filepath.Clean(scope)
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("scope must not contain '..': %s", scope)
		}
	}
	return nil
}

// validateGlobs checks that each glob pattern is non-empty, relative, and
// syntactically valid.
func validateGlobs(globs []string) error {
	for i, g := range globs {
		if strings.TrimSpace(g) == "" {
			return fmt.Errorf("glob[%d]: empty string not allowed", i)
		}
		if filepath.IsAbs(g) {
			return fmt.Errorf("glob[%d]: absolute paths not allowed: %s", i, g)
		}
		if _, err := filepath.Match(g, ""); err != nil {
			return fmt.Errorf("glob[%d]: invalid pattern %q: %w", i, g, err)
		}
	}
	return nil
}

// sanitizeName converts a name to a folder-safe format.
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// backlogToProto converts a BacklogItem to its protobuf representation.
func backlogToProto(item BacklogItem) *domainpb.BacklogItem {
	result := &domainpb.BacklogItem{
		Name:        item.Name,
		Title:       item.Title,
		Description: item.Description,
		Status:      string(item.Status),
		Priority:    int32(item.Priority),
		Tags:        item.Tags,
		Created:     item.Created,
		Updated:     item.Updated,
		Kind:        string(item.Kind),
	}
	if strings.TrimSpace(item.ResearchTarget) != "" {
		result.ResearchTarget = &item.ResearchTarget
	}
	if len(item.DependsOn) > 0 {
		result.DependsOn = item.DependsOn
	}
	if strings.TrimSpace(item.Initiative) != "" {
		result.Initiative = &item.Initiative
	}
	if strings.TrimSpace(item.Effort) != "" {
		result.Effort = &item.Effort
	}
	if strings.TrimSpace(item.Scope) != "" {
		result.Scope = &item.Scope
	}
	if len(item.AcceptanceAllow) > 0 {
		result.AcceptanceAllow = item.AcceptanceAllow
	}
	if len(item.AcceptanceDeny) > 0 {
		result.AcceptanceDeny = item.AcceptanceDeny
	}
	return result
}

// backlogFilesToProto converts a slice of BacklogFile to protobuf.
func backlogFilesToProto(files []BacklogFile) []*domainpb.BacklogFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]*domainpb.BacklogFile, 0, len(files))
	for _, file := range files {
		result = append(result, backlogFileToProto(file))
	}
	return result
}

// backlogFileToProto converts a single BacklogFile to protobuf.
func backlogFileToProto(file BacklogFile) *domainpb.BacklogFile {
	children := backlogFilesToProto(file.Children)
	var size *int64
	if file.Type == "file" {
		size = &file.Size
	}
	return &domainpb.BacklogFile{
		Name:     file.Name,
		Path:     file.Path,
		Type:     file.Type,
		Size:     size,
		Children: children,
	}
}
