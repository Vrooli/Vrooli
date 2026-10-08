// Package controls contains pure precondition contracts for human repository
// actions. It never invokes Git and cannot authorize an agent.
package controls

import "fmt"

type MergePreview struct {
	RepositoryID  string   `json:"repositoryId"`
	SourceRef     string   `json:"sourceRef"`
	Destination   string   `json:"destination"`
	ExpectedBase  string   `json:"expectedBase"`
	SourceHead    string   `json:"sourceHead"`
	Conflicts     []string `json:"conflicts,omitempty"`
	SubjectDigest string   `json:"subjectDigest"`
}

func (m MergePreview) Validate() error {
	if m.RepositoryID == "" || m.SourceRef == "" || m.Destination == "" || m.ExpectedBase == "" || m.SourceHead == "" || m.SubjectDigest == "" {
		return fmt.Errorf("merge preview requires exact repository, refs, revisions, and subject")
	}
	return nil
}

type UndoFile struct {
	Path             string `json:"path"`
	CurrentDigest    string `json:"currentDigest"`
	PreimageDigest   string `json:"preimageDigest"`
	AttributionState string `json:"attributionState"`
}

type UndoPreparation struct {
	RepositoryID  string     `json:"repositoryId"`
	SubjectDigest string     `json:"subjectDigest"`
	Files         []UndoFile `json:"files"`
	HumanOnly     bool       `json:"humanOnly"`
}

func (u UndoPreparation) Validate() error {
	if u.RepositoryID == "" || u.SubjectDigest == "" || len(u.Files) == 0 || !u.HumanOnly {
		return fmt.Errorf("undo requires human-only exact subject and files")
	}
	for _, file := range u.Files {
		if file.Path == "" || file.CurrentDigest == "" || file.PreimageDigest == "" || file.AttributionState != "verified" {
			return fmt.Errorf("undo preimage or attribution is not safe for %q", file.Path)
		}
	}
	return nil
}
