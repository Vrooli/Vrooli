package profile

import (
	"context"
	"fmt"
)

type (
	Profile struct {
		WorkspaceID                                        string
		Revision, PresetVersion                            int64
		Preset                                             string
		ActiveRules, ExcludedGroups, Allergies, Appliances []string
		CostWeight, EffortWeight, VarietyWeight            float64
		DraftJSON                                          string
	}
	ApplyInput struct {
		WorkspaceID, Preset                     string
		ExcludedGroups, Allergies, Appliances   []string
		CostWeight, EffortWeight, VarietyWeight float64
	}
	Repository interface {
		Get(context.Context, string) (Profile, error)
		SaveDraft(context.Context, string, string) (Profile, error)
		Apply(context.Context, ApplyInput) (Profile, error)
	}
	ErrNotFound struct{ WorkspaceID string }
)

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("profile for workspace %q not found", e.WorkspaceID)
}
