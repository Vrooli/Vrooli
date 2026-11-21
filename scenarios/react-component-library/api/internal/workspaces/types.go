// Package workspaces owns the durable composition aggregate: a workspace
// contains pages, a page contains device displays, and a display contains named
// states. It replaces the separate design and template stores described in
// docs/internal/WORKBENCH-PROGRAM.md §3.1–3.3.
//
// The generated proto messages are the domain model. This package adds only the
// durable rules the database must enforce: identity, stable enum spelling,
// normalized ordering, optimistic concurrency on the aggregate revision, and
// revision snapshots.
package workspaces

import (
	"context"
	"errors"
	"fmt"
	"strings"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// ErrNotFound reports a workspace id with no row.
var ErrNotFound = errors.New("workspace not found")

// ErrConflict reports a save whose base revision is stale. Use errors.As with
// *ConflictError to obtain the workspace that is currently stored.
var ErrConflict = errors.New("workspace revision conflict")

// ErrInvalid reports an aggregate that cannot be persisted. Use errors.As with
// *InvalidError to obtain the path-specific issues.
var ErrInvalid = errors.New("invalid workspace")

// ErrCurated reports a write to a curated workspace. Curated workspaces are
// reviewed template snapshots; using one forks it, so the shelf cannot drift
// under everyone who already forked it.
var ErrCurated = errors.New("curated workspace is immutable; fork it to edit")

// ConflictError carries the stored aggregate so a client can rebase against
// exactly the revision another writer won with. It unwraps to ErrConflict.
type ConflictError struct {
	Current *workspacev1.Workspace
}

func (e *ConflictError) Error() string {
	if e == nil || e.Current == nil {
		return ErrConflict.Error()
	}
	return fmt.Sprintf("%s: current revision is %d", ErrConflict.Error(), e.Current.GetRevision())
}

func (e *ConflictError) Unwrap() error { return ErrConflict }

// InvalidError carries every issue found in one validation pass so the client
// can mark the offending paths instead of guessing which field failed first.
type InvalidError struct {
	Issues []*workspacev1.ValidationIssue
}

func (e *InvalidError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ErrInvalid.Error()
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, issue.GetPath()+": "+issue.GetMessage())
	}
	return ErrInvalid.Error() + ": " + strings.Join(parts, "; ")
}

func (e *InvalidError) Unwrap() error { return ErrInvalid }

// Recorder is the activity seam. The workspaces package deliberately does not
// import the activity owner: main wires the concrete recorder, and a nil
// recorder means activity logging is not configured for this process.
//
// Record is called after the mutation commits. before is nil on create/fork,
// after is nil on delete, and both are protojson snapshots of the aggregate
// (UseProtoNames) so a reader never has to know this package's Go types.
type Recorder interface {
	Record(ctx context.Context, action, subjectID string, before, after []byte) error
}

// Repository is the storage seam. Implementations own identity, revisions,
// timestamps and nested-row ordering; callers never supply them. Validate is
// read-only and returns issues without writing, which is what the
// ValidateWorkspace RPC serves.
type Repository interface {
	List(ctx context.Context, provenance workspacev1.WorkspaceProvenance, query string, limit int32) ([]*workspacev1.WorkspaceSummary, error)
	Get(ctx context.Context, id string) (*workspacev1.Workspace, error)
	Create(ctx context.Context, req *workspacev1.CreateWorkspaceRequest) (*workspacev1.Workspace, error)
	Save(ctx context.Context, req *workspacev1.SaveWorkspaceRequest) (*workspacev1.Workspace, error)
	Delete(ctx context.Context, id string) error
	Fork(ctx context.Context, id, name string) (*workspacev1.Workspace, error)
	Revisions(ctx context.Context, id string, limit int32) ([]*workspacev1.WorkspaceRevision, error)
	Validate(ctx context.Context, workspace *workspacev1.Workspace) ([]*workspacev1.ValidationIssue, error)

	// Reference-image assets are workspace-scoped. They are not part of the
	// composition revision: uploading a reference never races an edit, and a
	// conflicted save must not roll an upload back.
	ListWorkspaceAssets(ctx context.Context, workspaceID string) ([]Asset, error)
	ListAllWorkspaceAssets(ctx context.Context) ([]Asset, error)
	GetWorkspaceAsset(ctx context.Context, workspaceID, assetID string) (Asset, []byte, error)
	CreateWorkspaceAsset(ctx context.Context, input AssetInput) (Asset, error)
	DeleteWorkspaceAsset(ctx context.Context, workspaceID, assetID string) error
}

const (
	provenanceAuthored = "authored"
	provenanceCurated  = "curated"
	provenanceImported = "imported"
	provenanceForked   = "forked"

	deviceDesktop = "desktop"
	deviceTablet  = "tablet"
	devicePhone   = "phone"

	statusDraft    = "draft"
	statusReady    = "ready"
	statusCaptured = "captured"
)

// provenanceToSQL maps the proto enum to the stable lowercase spelling stored
// in SQL. UNSPECIFIED is the proto default and means authored; it is not a
// fifth provenance.
func provenanceToSQL(value workspacev1.WorkspaceProvenance) (string, bool) {
	switch value {
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED,
		workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED:
		return provenanceAuthored, true
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED:
		return provenanceCurated, true
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED:
		return provenanceImported, true
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED:
		return provenanceForked, true
	default:
		return "", false
	}
}

func provenanceFromSQL(value string) (workspacev1.WorkspaceProvenance, error) {
	switch value {
	case provenanceAuthored:
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED, nil
	case provenanceCurated:
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, nil
	case provenanceImported:
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, nil
	case provenanceForked:
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, nil
	default:
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, fmt.Errorf("unknown workspace provenance %q", value)
	}
}

// deviceToSQL returns ok=false for UNSPECIFIED and unknown values. Validation
// rejects both; a stored row must always name a real device.
func deviceToSQL(value workspacev1.DisplayDevice) (string, bool) {
	switch value {
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP:
		return deviceDesktop, true
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET:
		return deviceTablet, true
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE:
		return devicePhone, true
	default:
		return "", false
	}
}

func deviceFromSQL(value string) (workspacev1.DisplayDevice, error) {
	switch value {
	case deviceDesktop:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP, nil
	case deviceTablet:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET, nil
	case devicePhone:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE, nil
	default:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_UNSPECIFIED, fmt.Errorf("unknown display device %q", value)
	}
}

// statusToSQL maps UNSPECIFIED to draft: a new state is a draft until an
// author or a capture marks it otherwise.
func statusToSQL(value workspacev1.StateStatus) (string, bool) {
	switch value {
	case workspacev1.StateStatus_STATE_STATUS_UNSPECIFIED,
		workspacev1.StateStatus_STATE_STATUS_DRAFT:
		return statusDraft, true
	case workspacev1.StateStatus_STATE_STATUS_READY:
		return statusReady, true
	case workspacev1.StateStatus_STATE_STATUS_CAPTURED:
		return statusCaptured, true
	default:
		return "", false
	}
}

func statusFromSQL(value string) (workspacev1.StateStatus, error) {
	switch value {
	case statusDraft:
		return workspacev1.StateStatus_STATE_STATUS_DRAFT, nil
	case statusReady:
		return workspacev1.StateStatus_STATE_STATUS_READY, nil
	case statusCaptured:
		return workspacev1.StateStatus_STATE_STATUS_CAPTURED, nil
	default:
		return workspacev1.StateStatus_STATE_STATUS_UNSPECIFIED, fmt.Errorf("unknown state status %q", value)
	}
}

// deviceMetrics are the one fallback used by the legacy import when an old
// frame carries a device but no explicit dimensions. They match the studio's
// FRAME_SIZE table so an imported workspace opens at the same size it had.
func deviceMetrics(device workspacev1.DisplayDevice) (width, height int32) {
	switch device {
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE:
		return 390, 844
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET:
		return 834, 900
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP:
		return 1180, 780
	default:
		return 0, 0
	}
}
