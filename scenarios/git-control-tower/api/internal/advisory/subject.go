// Package advisory contains immutable, host-neutral contracts shared by
// summaries, drafts, reviews, and provenance investigations.
package advisory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type SubjectKind string

const (
	SubjectCurrent     SubjectKind = "current_changes"
	SubjectStaged      SubjectKind = "staged_changes"
	SubjectCommit      SubjectKind = "commit"
	SubjectRange       SubjectKind = "range"
	SubjectPullRequest SubjectKind = "pull_request"
)

type HostSubject struct {
	Provider     string `json:"provider"`
	InstanceID   string `json:"instance_id"`
	RepositoryID string `json:"repository_id"`
	ChangeNumber string `json:"change_number"`
}

type Scope struct {
	Paths           []string `json:"paths"`
	SelectionDigest string   `json:"selection_digest"`
}

type ChangeSubject struct {
	RepositoryID   string       `json:"repository_id"`
	Kind           SubjectKind  `json:"kind"`
	Host           *HostSubject `json:"host,omitempty"`
	BaseRevision   string       `json:"base_revision,omitempty"`
	HeadRevision   string       `json:"head_revision,omitempty"`
	ParentRevision string       `json:"parent_revision,omitempty"`
	Scope          Scope        `json:"scope"`
	SnapshotDigest string       `json:"snapshot_digest"`
}

func (s ChangeSubject) Validate() error {
	if strings.TrimSpace(s.RepositoryID) == "" || strings.TrimSpace(s.SnapshotDigest) == "" {
		return fmt.Errorf("repository_id and snapshot_digest are required")
	}
	if s.Kind != SubjectCurrent && s.Kind != SubjectStaged && s.Kind != SubjectCommit && s.Kind != SubjectRange && s.Kind != SubjectPullRequest {
		return fmt.Errorf("unsupported subject kind %q", s.Kind)
	}
	if len(s.Scope.Paths) == 0 || strings.TrimSpace(s.Scope.SelectionDigest) == "" {
		return fmt.Errorf("subject scope requires paths and selection_digest")
	}
	for _, path := range s.Scope.Paths {
		if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
			return fmt.Errorf("unsafe subject path %q", path)
		}
	}
	if s.Kind == SubjectPullRequest {
		if s.Host == nil || s.Host.Provider == "" || s.Host.InstanceID == "" || s.Host.RepositoryID == "" || s.Host.ChangeNumber == "" {
			return fmt.Errorf("pull request subject requires complete host identity")
		}
	}
	if s.Kind == SubjectCommit && s.HeadRevision == "" {
		return fmt.Errorf("commit subject requires head_revision")
	}
	if s.Kind == SubjectRange && (strings.TrimSpace(s.BaseRevision) == "" || strings.TrimSpace(s.HeadRevision) == "") {
		return fmt.Errorf("range subject requires base_revision and head_revision")
	}
	if s.Kind == SubjectPullRequest && (strings.TrimSpace(s.BaseRevision) == "" || strings.TrimSpace(s.HeadRevision) == "") {
		return fmt.Errorf("pull request subject requires base_revision and head_revision")
	}
	return nil
}

func (s ChangeSubject) Digest() (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	clone := s
	clone.Scope.Paths = append([]string(nil), s.Scope.Paths...)
	sort.Strings(clone.Scope.Paths)
	data, err := json.Marshal(clone)
	if err != nil {
		return "", fmt.Errorf("marshal subject: %w", err)
	}
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}

type Coverage struct {
	IncludedFiles int        `json:"included_files"`
	OmittedFiles  int        `json:"omitted_files"`
	Omissions     []Omission `json:"omissions,omitempty"`
}
type Omission struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}
type ValidationRef struct {
	ExecutionID  string `json:"execution_id"`
	Availability string `json:"availability"`
	Verdict      string `json:"verdict"`
}
type Claim struct {
	Text         string   `json:"text"`
	EvidenceRefs []string `json:"evidence_refs"`
}
type EvidenceBundle struct {
	OperationID   string            `json:"operation_id"`
	SubjectDigest string            `json:"subject_digest"`
	Status        string            `json:"status"`
	Claims        []Claim           `json:"claims,omitempty"`
	Coverage      Coverage          `json:"coverage"`
	Validation    []ValidationRef   `json:"validation,omitempty"`
	Unknowns      []string          `json:"unknowns,omitempty"`
	Versions      map[string]string `json:"versions"`
}

func (b EvidenceBundle) Validate() error {
	if strings.TrimSpace(b.OperationID) == "" || !strings.HasPrefix(b.SubjectDigest, "sha256:") {
		return fmt.Errorf("operation_id and subject_digest are required")
	}
	switch b.Status {
	case "refused", "failed", "unavailable", "partial", "complete":
	default:
		return fmt.Errorf("invalid evidence status %q", b.Status)
	}
	if b.Coverage.IncludedFiles < 0 || b.Coverage.OmittedFiles < 0 {
		return fmt.Errorf("coverage counts cannot be negative")
	}
	if b.Coverage.OmittedFiles != len(b.Coverage.Omissions) {
		return fmt.Errorf("omission count does not match coverage")
	}
	if b.Status == "complete" && b.Coverage.OmittedFiles != 0 {
		return fmt.Errorf("complete evidence cannot omit files")
	}
	for _, omission := range b.Coverage.Omissions {
		if strings.TrimSpace(omission.Path) == "" || strings.TrimSpace(omission.Reason) == "" {
			return fmt.Errorf("every omission requires a path and reason")
		}
		if strings.HasPrefix(omission.Path, "/") || strings.Contains(omission.Path, "..") {
			return fmt.Errorf("unsafe omitted path %q", omission.Path)
		}
	}
	for _, claim := range b.Claims {
		if strings.TrimSpace(claim.Text) == "" {
			return fmt.Errorf("claims require text")
		}
		if len(claim.EvidenceRefs) == 0 {
			return fmt.Errorf("claim %q has no evidence", claim.Text)
		}
	}
	for _, validation := range b.Validation {
		if strings.TrimSpace(validation.ExecutionID) == "" || strings.TrimSpace(validation.Availability) == "" || strings.TrimSpace(validation.Verdict) == "" {
			return fmt.Errorf("validation references require execution, availability, and verdict")
		}
	}
	return nil
}
