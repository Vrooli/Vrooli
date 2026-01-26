package explorer

// DOC: docs/reference/api-endpoints.md#scenario-documentation-tree

import "time"

// DocTreeNode represents a file or directory in the documentation tree.
type DocTreeNode struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	Type       string        `json:"type"`
	DocType    string        `json:"doc_type,omitempty"`
	Size       int64         `json:"size,omitempty"`
	ModifiedAt time.Time     `json:"modified_at,omitempty"`
	Children   []DocTreeNode `json:"children,omitempty"`
	Warning    *DocWarning   `json:"warning,omitempty"`
}

// DocWarning indicates a documentation health issue.
type DocWarning struct {
	Type         string `json:"type"`
	Message      string `json:"message"`
	ExpectedPath string `json:"expected_path,omitempty"`
	Severity     string `json:"severity"`
}
