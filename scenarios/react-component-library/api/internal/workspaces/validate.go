package workspaces

import (
	"context"
	"fmt"
	"strings"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// Validate reports the four authoring rules that make an aggregate persistable
// and addressable: a workspace name, globally unique non-empty ids, a concrete
// device on every display, and unique state names within a display. It never
// writes and never assigns ids, so a client can validate a draft before create.
//
// Sort order is not an input rule: Save and Create normalize it to 0..n-1 so a
// client that reorders a list does not have to renumber every row.
func (r *SQLiteRepository) Validate(_ context.Context, workspace *workspacev1.Workspace) ([]*workspacev1.ValidationIssue, error) {
	return validationIssues(workspace), nil
}

// validationIssues is the pure validator shared by the repository seam and
// Create/Save. Keeping it pure makes the write path and the ValidateWorkspace
// RPC answer with exactly the same list.
func validationIssues(workspace *workspacev1.Workspace) []*workspacev1.ValidationIssue {
	if workspace == nil {
		return []*workspacev1.ValidationIssue{{Path: "workspace", Message: "workspace is required"}}
	}

	issues := make([]*workspacev1.ValidationIssue, 0)
	if strings.TrimSpace(workspace.GetName()) == "" {
		issues = append(issues, &workspacev1.ValidationIssue{Path: "name", Message: "name is required"})
	}
	if _, ok := provenanceToSQL(workspace.GetProvenance()); !ok {
		issues = append(issues, &workspacev1.ValidationIssue{Path: "provenance", Message: "provenance is not a known value"})
	}

	// Ids are the stable address of every node (workspace/page/display/state),
	// so a duplicate anywhere in the aggregate is one collision, not four
	// independent errors. Empty ids are allowed here because Validate also
	// serves drafts; Save and Create assign ids before they persist.
	seen := make(map[string]string)
	recordID := func(id, path string) {
		if id == "" {
			return
		}
		if first, exists := seen[id]; exists {
			issues = append(issues, &workspacev1.ValidationIssue{
				Path:    path + ".id",
				Message: fmt.Sprintf("id %q is already used at %s", id, first),
			})
			return
		}
		seen[id] = path
	}
	recordID(workspace.GetId(), "workspace")

	for pageIndex, page := range workspace.GetPages() {
		pagePath := fmt.Sprintf("pages[%d]", pageIndex)
		if page == nil {
			issues = append(issues, &workspacev1.ValidationIssue{Path: pagePath, Message: "page is required"})
			continue
		}
		recordID(page.GetId(), pagePath)
		for displayIndex, display := range page.GetDisplays() {
			displayPath := fmt.Sprintf("%s.displays[%d]", pagePath, displayIndex)
			if display == nil {
				issues = append(issues, &workspacev1.ValidationIssue{Path: displayPath, Message: "display is required"})
				continue
			}
			recordID(display.GetId(), displayPath)
			if _, ok := deviceToSQL(display.GetDevice()); !ok {
				issues = append(issues, &workspacev1.ValidationIssue{
					Path:    displayPath + ".device",
					Message: "device must be desktop, tablet or phone",
				})
			}

			names := make(map[string]int, len(display.GetStates()))
			for stateIndex, state := range display.GetStates() {
				statePath := fmt.Sprintf("%s.states[%d]", displayPath, stateIndex)
				if state == nil {
					issues = append(issues, &workspacev1.ValidationIssue{Path: statePath, Message: "state is required"})
					continue
				}
				recordID(state.GetId(), statePath)
				if _, ok := statusToSQL(state.GetStatus()); !ok {
					issues = append(issues, &workspacev1.ValidationIssue{Path: statePath + ".status", Message: "status is not a known value"})
				}
				name := strings.TrimSpace(state.GetName())
				if name == "" {
					continue
				}
				if first, exists := names[name]; exists {
					issues = append(issues, &workspacev1.ValidationIssue{
						Path:    statePath + ".name",
						Message: fmt.Sprintf("state name %q is already used at states[%d]", name, first),
					})
					continue
				}
				names[name] = stateIndex
			}
		}
	}
	return issues
}

// normalizeAggregate assigns identity and ordering before persistence. It is
// shared by Create, Save and Fork so all three write the same invariants. The
// returned error is only an identity-source failure; authoring problems are
// reported by validationIssues.
//
// Missing ids are assigned here rather than in SQL because a save replaces the
// nested rows: a row without an id would either collide with a sibling or
// silently reuse the previous deployment's identity.
func normalizeAggregate(workspace *workspacev1.Workspace, assignMissingIDs bool) error {
	workspace.Name = strings.TrimSpace(workspace.Name)
	workspace.Summary = strings.TrimSpace(workspace.Summary)
	workspace.BindingScenario = strings.TrimSpace(workspace.BindingScenario)
	workspace.BindingPage = strings.TrimSpace(workspace.BindingPage)
	// UNSPECIFIED is the proto default, not a fifth provenance. Normalizing it
	// here makes the create response identical to a subsequent Get instead of
	// returning a value the database immediately serialises as authored.
	if workspace.Provenance == workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED {
		workspace.Provenance = workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED
	}
	if workspace.ThemeJson == "" {
		workspace.ThemeJson = "{}"
	}
	if workspace.OptionsJson == "" {
		workspace.OptionsJson = "{}"
	}

	for pageIndex, page := range workspace.Pages {
		if page == nil {
			continue
		}
		if assignMissingIDs && page.Id == "" {
			id, err := newID("pg_")
			if err != nil {
				return fmt.Errorf("assign page id: %w", err)
			}
			page.Id = id
		}
		if page.SpecJson == "" {
			page.SpecJson = "{}"
		}
		page.SortOrder = int32(pageIndex)
		for displayIndex, display := range page.Displays {
			if display == nil {
				continue
			}
			if assignMissingIDs && display.Id == "" {
				id, err := newID("dp_")
				if err != nil {
					return fmt.Errorf("assign display id: %w", err)
				}
				display.Id = id
			}
			display.SortOrder = int32(displayIndex)
			for stateIndex, state := range display.States {
				if state == nil {
					continue
				}
				if assignMissingIDs && state.Id == "" {
					id, err := newID("st_")
					if err != nil {
						return fmt.Errorf("assign state id: %w", err)
					}
					state.Id = id
				}
				state.SortOrder = int32(stateIndex)
				if state.Status == workspacev1.StateStatus_STATE_STATUS_UNSPECIFIED {
					state.Status = workspacev1.StateStatus_STATE_STATUS_DRAFT
				}
			}
		}
	}
	return nil
}

// seedDefaultComposition gives an empty create request the one page, desktop
// display and default state a studio needs before the user has authored
// anything. Only Create calls it; Save must never invent structure.
func seedDefaultComposition() (*workspacev1.WorkspacePage, error) {
	pageID, err := newID("pg_")
	if err != nil {
		return nil, fmt.Errorf("seed page id: %w", err)
	}
	displayID, err := newID("dp_")
	if err != nil {
		return nil, fmt.Errorf("seed display id: %w", err)
	}
	stateID, err := newID("st_")
	if err != nil {
		return nil, fmt.Errorf("seed state id: %w", err)
	}
	width, height := deviceMetrics(workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP)
	return &workspacev1.WorkspacePage{
		Id:        pageID,
		Name:      "Page 1",
		SpecJson:  "{}",
		SortOrder: 0,
		Displays: []*workspacev1.WorkspaceDisplay{{
			Id:        displayID,
			Device:    workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP,
			Width:     width,
			Height:    height,
			SortOrder: 0,
			States: []*workspacev1.WorkspaceState{{
				Id:        stateID,
				Name:      "Default",
				ArgsJson:  "{}",
				Status:    workspacev1.StateStatus_STATE_STATUS_DRAFT,
				SortOrder: 0,
			}},
		}},
	}, nil
}

func invalidFromIssues(issues []*workspacev1.ValidationIssue) error {
	if len(issues) == 0 {
		return nil
	}
	return &InvalidError{Issues: issues}
}
