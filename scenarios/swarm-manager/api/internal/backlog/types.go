// Package backlog types defines the core domain types, constants, and
// conversion functions for backlog items. These are pure data definitions
// with no HTTP or filesystem dependencies.
package backlog

import (
	"encoding/json"
	"fmt"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/identity"
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
// Derived from KindConfig to maintain a single source of truth.
var backlogKindDirs = func() map[BacklogKind]string {
	m := make(map[BacklogKind]string, len(KindConfig))
	for k, meta := range KindConfig {
		m[k] = meta.Dir
	}
	return m
}()

// BacklogItem represents a unit of work stored on disk.
type BacklogItem struct {
	Name            string               `json:"name"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	Status          BacklogStatus        `json:"status"`
	Priority        int                  `json:"priority"`
	Tags            []string             `json:"tags"`
	Created         string               `json:"created"`
	Updated         string               `json:"updated"`
	Kind            BacklogKind          `json:"kind"`
	DependsOn       []string             `json:"depends_on,omitempty"`
	Initiative      string               `json:"initiative,omitempty"`
	Effort          string               `json:"effort,omitempty"`
	AcceptanceAllow []string             `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string             `json:"acceptance_deny,omitempty"`
	SpawnedFrom     string               `json:"spawned_from,omitempty"`
	Note            string               `json:"note,omitempty"`
	SuggestedSkills []string             `json:"suggested_skills,omitempty"`
	CreatedBy       *identity.Provenance `json:"created_by,omitempty"`
	ArchivedAt      *string              `json:"archived_at,omitempty"`
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
	ResearchModeFinalize   ResearchMode = "finalize"
	ResearchModeInitialize ResearchMode = "initialize"
)

// promptSelection holds the resolved skill ID, variables, and rendered prompt
// text returned by prompt-manager.
type promptSelection struct {
	SkillID      string
	Variables    map[string]string
	Prompt       string
	ExperimentID string
	VariantID    string
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
	case "backlog", "researching", "ready", "queued", "in_progress", "completed", "failed":
		return true
	default:
		return false
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

// validateGlobs is legacy compatibility logic until swarm-manager migrates to
// the shared repo contract. Keep behavior changes here aligned with
// docs/repo-contract.md and prefer contract-backed semantics in later phases.
//
// validateGlobs checks that each glob pattern is non-empty, relative, and
// syntactically valid.
func validateGlobs(globs []string) error {
	for i, g := range globs {
		if err := repocontract.ValidateRepoGlob(g); err != nil {
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

// backlogToProtoWithValidation converts a BacklogItem to its protobuf representation,
// including the plan validation report loaded from the item directory.
func backlogToProtoWithValidation(item BacklogItem, itemDir string) *domainpb.BacklogItem {
	result := backlogToProto(item)
	if item.Kind != KindResearch {
		report, err := LoadOrRefreshValidationReport(itemDir, item.Kind)
		if err == nil && report != nil {
			data, _ := json.Marshal(report)
			jsonStr := string(data)
			result.PlanValidationJson = &jsonStr
		}
	}
	return result
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
	if len(item.DependsOn) > 0 {
		result.DependsOn = item.DependsOn
	}
	if strings.TrimSpace(item.Initiative) != "" {
		result.Initiative = &item.Initiative
	}
	if strings.TrimSpace(item.Effort) != "" {
		result.Effort = &item.Effort
	}
	if len(item.AcceptanceAllow) > 0 {
		result.AcceptanceAllow = item.AcceptanceAllow
	}
	if len(item.AcceptanceDeny) > 0 {
		result.AcceptanceDeny = item.AcceptanceDeny
	}
	if strings.TrimSpace(item.SpawnedFrom) != "" {
		result.SpawnedFrom = &item.SpawnedFrom
	}
	if strings.TrimSpace(item.Note) != "" {
		result.Note = &item.Note
	}
	if len(item.SuggestedSkills) > 0 {
		result.SuggestedSkills = item.SuggestedSkills
	}
	if item.ArchivedAt != nil {
		result.ArchivedAt = item.ArchivedAt
	}
	return result
}
