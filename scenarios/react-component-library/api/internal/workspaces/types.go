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
	"database/sql"
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

// ErrBindingConflict means another workspace already owns this experience page.
var ErrBindingConflict = errors.New("experience page is already bound to a workspace")

// ErrCaptureInProgress prevents a second BAS execution from racing the active
// token for the same state revision.
var ErrCaptureInProgress = errors.New("another capture attempt is already in progress for this workspace state")

// ErrInvalid reports an aggregate that cannot be persisted. Use errors.As with
// *InvalidError to obtain the path-specific issues.
var ErrInvalid = errors.New("invalid workspace")

// ErrCurated reports a write to a curated workspace. Curated workspaces are
// reviewed template snapshots; using one forks it, so the shelf cannot drift
// under everyone who already forked it.
var ErrCurated = errors.New("curated workspace is immutable; fork it to edit")

// ErrSourceBaselineAsset protects bytes still referenced by source drift evidence.
var ErrSourceBaselineAsset = errors.New("asset is referenced by a source baseline")

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
	AttachCaptureEvidence(ctx context.Context, workspaceID, stateID string, baseRevision int32, attemptID, requestToken, captureID, evidenceJSON, capturedAt string) (*workspacev1.Workspace, error)
	GetCaptureEvidence(ctx context.Context, workspaceID, stateID string) (CaptureEvidence, error)
	ListCaptureReceipts(ctx context.Context, workspaceID, stateID string) ([]CaptureReceipt, error)
	GetCaptureReceipt(ctx context.Context, workspaceID, stateID, receiptID string) (CaptureReceipt, error)
	CreateVisualComparison(ctx context.Context, workspaceID, stateID, baselineReceiptID, currentReceiptID, resultJSON, heatmapRef, createdAt string) (VisualComparison, error)
	ListVisualComparisons(ctx context.Context, workspaceID, stateID string) ([]VisualComparison, error)
	GetVisualComparison(ctx context.Context, workspaceID, stateID, comparisonID string) (VisualComparison, error)
	CreateSourceBaseline(ctx context.Context, workspaceID, stateID, assetID, sha256, sourceLocator, createdAt string) (SourceBaseline, error)
	GetCurrentSourceBaseline(ctx context.Context, workspaceID, stateID string) (SourceBaseline, error)
	ListSourceBaselines(ctx context.Context, workspaceID, stateID string) ([]SourceBaseline, error)
	GetAcceptedSourceBaseline(ctx context.Context, workspaceID, stateID string) (SourceBaseline, error)
	AcceptSourceBaseline(ctx context.Context, workspaceID, stateID, baselineID, acceptedBy, acceptedAt string) (SourceBaseline, error)
	CreateSourceVisualComparison(ctx context.Context, item SourceVisualComparison) (SourceVisualComparison, error)
	ListSourceVisualComparisons(ctx context.Context, workspaceID, stateID string) ([]SourceVisualComparison, error)
	GetSourceVisualComparison(ctx context.Context, workspaceID, stateID, comparisonID string) (SourceVisualComparison, error)
	CreateConversionReceipt(ctx context.Context, input ConversionReceiptInput) (ConversionReceipt, error)
	GetConversionReceipt(ctx context.Context, scenario, page string) (ConversionReceipt, error)
	CreateWorkspaceWithConversionReceipt(ctx context.Context, workspace *workspacev1.Workspace, input ConversionReceiptInput) (*workspacev1.Workspace, ConversionReceipt, error)
	BeginCaptureAttempt(ctx context.Context, workspaceID, stateID string, sourceRevision int32, attemptID, requestToken string) (CaptureAttempt, bool, error)
	GetActiveCaptureAttempt(ctx context.Context, workspaceID, stateID string, sourceRevision int32) (CaptureAttempt, error)
	FinishCaptureAttempt(ctx context.Context, workspaceID, stateID string, sourceRevision int32, attemptID, state, detail string) error
}

// ErrConversionReceiptConflict indicates a source page already has a durable
// receipt with different evidence. Receipts are immutable; callers must rerun
// conversion and explicitly reconcile the existing adoption.
var ErrConversionReceiptConflict = errors.New("conversion receipt conflicts with existing receipt")

type ConversionReceiptConflictError struct {
	Existing ConversionReceipt
}

func (e *ConversionReceiptConflictError) Error() string {
	return fmt.Sprintf("%s: %s/%s is already attached to workspace %s", ErrConversionReceiptConflict, e.Existing.Scenario, e.Existing.Page, e.Existing.WorkspaceID)
}
func (e *ConversionReceiptConflictError) Unwrap() error { return ErrConversionReceiptConflict }

type ConversionReceiptInput struct {
	Scenario    string
	Page        string
	WorkspaceID string
	SourceHash  string
	TargetHash  string
	ReceiptJSON string
	CreatedAt   string
}

type ConversionReceipt struct {
	ID          string `json:"id"`
	Scenario    string `json:"scenario"`
	Page        string `json:"page"`
	WorkspaceID string `json:"workspaceId"`
	SourceHash  string `json:"sourceHash"`
	TargetHash  string `json:"targetHash"`
	ReceiptJSON string `json:"receiptJson"`
	CreatedAt   string `json:"createdAt"`
}

// Database is the narrow SQL surface the workspace repository needs. Both
// *sql.DB and api-core's RoutedDB implement it, while tests can continue to
// inject an ordinary SQLite pool. Transactions remain bound to the pool
// selected from the request context by RoutedDB.BeginTx.
type Database interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type CaptureEvidence struct {
	WorkspaceID    string `db:"workspace_id" json:"workspaceId"`
	StateID        string `db:"state_id" json:"stateId"`
	RequestToken   string `db:"request_token" json:"requestToken"`
	SourceRevision int32  `db:"source_revision" json:"sourceRevision"`
	CaptureID      string `db:"capture_id" json:"captureId"`
	EvidenceJSON   string `db:"evidence_json" json:"-"`
	CapturedAt     string `db:"captured_at" json:"capturedAt"`
}

// CaptureReceipt is an immutable record of one successful capture operation.
// It remains addressable after a later capture replaces the current projection.
type CaptureReceipt struct {
	ID             string `db:"id" json:"id"`
	WorkspaceID    string `db:"workspace_id" json:"workspaceId"`
	StateID        string `db:"state_id" json:"stateId"`
	SourceRevision int32  `db:"source_revision" json:"sourceRevision"`
	CaptureID      string `db:"capture_id" json:"captureId"`
	AttemptID      string `db:"attempt_id" json:"attemptId"`
	EvidenceJSON   string `db:"evidence_json" json:"-"`
	CapturedAt     string `db:"captured_at" json:"capturedAt"`
}

type VisualComparison struct {
	ID                string `db:"id" json:"id"`
	WorkspaceID       string `db:"workspace_id" json:"workspaceId"`
	StateID           string `db:"state_id" json:"stateId"`
	BaselineReceiptID string `db:"baseline_receipt_id" json:"baselineReceiptId"`
	CurrentReceiptID  string `db:"current_receipt_id" json:"currentReceiptId"`
	ResultJSON        string `db:"result_json" json:"-"`
	HeatmapRef        string `db:"heatmap_ref" json:"heatmapRef"`
	CreatedAt         string `db:"created_at" json:"createdAt"`
}

// SourceBaseline is an immutable, per-state image reference. Acceptance is a
// separate immutable event so uploading an image can never imply approval.
type SourceBaseline struct {
	ID            string `json:"id"`
	WorkspaceID   string `json:"workspaceId"`
	StateID       string `json:"stateId"`
	AssetID       string `json:"assetId"`
	SHA256        string `json:"sha256"`
	SourceLocator string `json:"sourceLocator"`
	CreatedAt     string `json:"createdAt"`
	AcceptedAt    string `json:"acceptedAt,omitempty"`
	AcceptedBy    string `json:"acceptedBy,omitempty"`
}

type SourceVisualComparison struct {
	ID               string `json:"comparisonId"`
	WorkspaceID      string `json:"workspaceId"`
	StateID          string `json:"stateId"`
	SourceBaselineID string `json:"sourceBaselineId"`
	SourceAssetID    string `json:"sourceAssetId"`
	CurrentReceiptID string `json:"currentReceiptId"`
	ResultJSON       string `json:"resultJson"`
	HeatmapRef       string `json:"heatmapRef"`
	CreatedAt        string `json:"createdAt"`
}

type CaptureAttempt struct {
	WorkspaceID    string `db:"workspace_id" json:"workspaceId"`
	StateID        string `db:"state_id" json:"stateId"`
	SourceRevision int32  `db:"source_revision" json:"sourceRevision"`
	AttemptID      string `db:"attempt_id" json:"attemptId"`
	RequestToken   string `db:"request_token" json:"requestToken"`
	State          string `db:"state" json:"state"`
	Error          string `db:"error" json:"error,omitempty"`
	CreatedAt      string `db:"created_at" json:"createdAt"`
	UpdatedAt      string `db:"updated_at" json:"updatedAt"`
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
