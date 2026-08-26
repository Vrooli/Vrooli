package components

import (
	"fmt"
	"strings"
)

// StoryNotFoundError identifies a requested story and the valid ids declared
// by the selected contract.
type StoryNotFoundError struct {
	ComponentID string
	StoryID     string
	DeclaredIDs []string
}

func (e StoryNotFoundError) Error() string {
	declared := strings.Join(e.DeclaredIDs, ", ")
	if declared == "" {
		declared = "none"
	}
	return fmt.Sprintf("preview: story %q not found for component %q (declared ids: %s)", e.StoryID, e.ComponentID, declared)
}

type StoryContractNotFoundError struct {
	ComponentID string
	Version     string
}

func (e StoryContractNotFoundError) Error() string {
	return fmt.Sprintf("preview: story contract not found for component %q@%s", e.ComponentID, e.Version)
}

type StoryContractParseError struct {
	ComponentID string
	Version     string
	Detail      string
}

func (e StoryContractParseError) Error() string {
	return fmt.Sprintf("preview: parse story contract for %q@%s: %s", e.ComponentID, e.Version, e.Detail)
}

type StoryEncodeError struct {
	ComponentID string
	StoryID     string
	Field       string
	Cause       error
}

func (e StoryEncodeError) Error() string {
	return fmt.Sprintf("preview: encode story %s for %q (%s): %v", e.Field, e.ComponentID, e.StoryID, e.Cause)
}

func (e StoryEncodeError) Unwrap() error { return e.Cause }
