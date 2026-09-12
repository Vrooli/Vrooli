// Package provenance owns uncertainty-preserving attribution facts for agent
// changes. A path overlap is never sufficient to claim committed authorship.
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type State string

// Standing is deliberately ordered by evidentiary strength. The joiner must
// never promote path overlap or an asserted trailer to exact authorship.
type Standing string

const (
	ExactContent    Standing = "exact_content"
	CommitFile      Standing = "commit_file"
	RunFile         Standing = "run_file"
	WorkReference   Standing = "work_reference"
	Asserted        Standing = "asserted"
	Stale           Standing = "stale"
	Private         Standing = "private"
	Unavailable     Standing = "unavailable"
	UnknownStanding Standing = "unknown"
)

const (
	Pending   State = "pending"
	Applied   State = "applied"
	Reviewed  State = "reviewed"
	Committed State = "committed"
	Rejected  State = "rejected"
	Unknown   State = "unknown"
)

type Evidence struct {
	RunID              string    `json:"runId,omitempty"`
	SandboxID          string    `json:"sandboxId,omitempty"`
	ApplicationReceipt string    `json:"applicationReceipt,omitempty"`
	ContentDigest      string    `json:"contentDigest,omitempty"`
	CommitID           string    `json:"commitId,omitempty"`
	WorkReference      string    `json:"workReference,omitempty"`
	Visibility         string    `json:"visibility"`
	CommitState        string    `json:"commitState,omitempty"`
	RunOutcome         string    `json:"runOutcome,omitempty"`
	ConversationID     string    `json:"conversationId,omitempty"`
	CostUSD            float64   `json:"costUsd,omitempty"`
	CommittedAt        string    `json:"committedAt,omitempty"`
	Unavailable        []string  `json:"unavailable,omitempty"`
	WorkReferences     []WorkRef `json:"workReferences,omitempty"`
}

// WorkRef is a producer-neutral reference copied from an authorized run
// report. GCT treats it as context, never as authorship or mutation authority.
type WorkRef struct {
	Kind              string `json:"kind"`
	ID                string `json:"id"`
	Revision          string `json:"revision,omitempty"`
	Relationship      string `json:"relationship,omitempty"`
	Verified          bool   `json:"verified,omitempty"`
	Visibility        string `json:"visibility,omitempty"`
	State             string `json:"state,omitempty"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
}

type JoinInput struct {
	NativeCommit string
	NativeDigest string
	Complete     bool
	Evidence     []Evidence
}

type JoinResult struct {
	Standing Standing   `json:"standing"`
	Reasons  []string   `json:"reasons,omitempty"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// JoinEvidence applies the conservative standing rules shared by provenance
// consumers. Every downgrade remains machine-readable for review and audit.
func JoinEvidence(input JoinInput) JoinResult {
	result := JoinResult{Standing: UnknownStanding}
	for _, evidence := range input.Evidence {
		if evidence.Visibility == "private" {
			result.Standing = Private
			result.Reasons = append(result.Reasons, "private evidence withheld")
			continue
		}
		if evidence.ContentDigest != "" && input.Complete && evidence.ContentDigest == input.NativeDigest && evidence.CommitID == input.NativeCommit {
			result.Standing = ExactContent
			result.Evidence = append(result.Evidence, evidence)
			continue
		}
		if evidence.CommitID != "" && evidence.CommitID == input.NativeCommit {
			if result.Standing != ExactContent {
				result.Standing = CommitFile
			}
			result.Reasons = append(result.Reasons, "commit matched but exact content was not verified")
			result.Evidence = append(result.Evidence, evidence)
			continue
		}
		if evidence.RunID != "" {
			if result.Standing == UnknownStanding {
				result.Standing = RunFile
			}
			result.Reasons = append(result.Reasons, "run evidence overlaps the path without exact content proof")
			result.Evidence = append(result.Evidence, evidence)
			continue
		}
		if evidence.WorkReference != "" {
			if result.Standing == UnknownStanding {
				result.Standing = WorkReference
			}
			result.Reasons = append(result.Reasons, "work reference is asserted but not content proof")
			result.Evidence = append(result.Evidence, evidence)
		}
	}
	if result.Standing == UnknownStanding && len(input.Evidence) == 0 {
		result.Reasons = []string{"no linked evidence available"}
	}
	return result
}

type File struct {
	Path          string     `json:"path"`
	ContentDigest string     `json:"contentDigest,omitempty"`
	State         State      `json:"state"`
	Contributors  []string   `json:"contributors,omitempty"`
	Evidence      []Evidence `json:"evidence"`
}

type Record struct {
	RepositoryID  string   `json:"repositoryId"`
	SubjectDigest string   `json:"subjectDigest"`
	Files         []File   `json:"files"`
	Unknowns      []string `json:"unknowns,omitempty"`
}

func (r Record) Validate() error {
	if strings.TrimSpace(r.RepositoryID) == "" || !strings.HasPrefix(r.SubjectDigest, "sha256:") {
		return fmt.Errorf("repository and subject digest are required")
	}
	seen := map[string]struct{}{}
	for _, file := range r.Files {
		if file.Path == "" || strings.HasPrefix(file.Path, "/") || strings.Contains(file.Path, "../") {
			return fmt.Errorf("unsafe provenance path %q", file.Path)
		}
		if _, ok := seen[file.Path]; ok {
			return fmt.Errorf("duplicate provenance path %q", file.Path)
		}
		seen[file.Path] = struct{}{}
		switch file.State {
		case Pending, Applied, Reviewed, Committed, Rejected, Unknown:
		default:
			return fmt.Errorf("unsupported provenance state %q", file.State)
		}
		if file.State == Committed {
			if file.ContentDigest == "" {
				return fmt.Errorf("committed file %q requires content evidence", file.Path)
			}
			if !hasCommitEvidence(file.Evidence) {
				return fmt.Errorf("committed file %q requires commit evidence", file.Path)
			}
		}
		for _, evidence := range file.Evidence {
			if evidence.Visibility != "" && evidence.Visibility != "public" && evidence.Visibility != "private" {
				return fmt.Errorf("unsupported provenance visibility %q", evidence.Visibility)
			}
		}
	}
	return nil
}

func hasCommitEvidence(evidence []Evidence) bool {
	for _, item := range evidence {
		if item.CommitID != "" && item.ContentDigest != "" && item.Visibility != "private" {
			return true
		}
	}
	return false
}

func (r Record) Digest() (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	copyRecord := r
	copyRecord.Files = append([]File(nil), r.Files...)
	copyRecord.Unknowns = append([]string(nil), r.Unknowns...)
	sort.Strings(copyRecord.Unknowns)
	for i := range copyRecord.Files {
		copyRecord.Files[i].Contributors = append([]string(nil), copyRecord.Files[i].Contributors...)
		sort.Strings(copyRecord.Files[i].Contributors)
		copyRecord.Files[i].Evidence = append([]Evidence(nil), copyRecord.Files[i].Evidence...)
		sort.SliceStable(copyRecord.Files[i].Evidence, func(a, b int) bool {
			left, _ := json.Marshal(copyRecord.Files[i].Evidence[a])
			right, _ := json.Marshal(copyRecord.Files[i].Evidence[b])
			return string(left) < string(right)
		})
	}
	sort.Slice(copyRecord.Files, func(i, j int) bool { return copyRecord.Files[i].Path < copyRecord.Files[j].Path })
	data, err := json.Marshal(copyRecord)
	if err != nil {
		return "", fmt.Errorf("marshal provenance record: %w", err)
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}
