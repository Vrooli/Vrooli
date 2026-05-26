// Package destinations is the domain-scoped home for backup destinations: kopia
// repositories identified by a unique name and backed by filesystem or S3.
//
// Layering mirrors the canonical Vrooli per-domain pattern:
//
//	Connect handler → Service (validates, decides, caps) → Repository (persists)
//	                      ↑                                      ↑
//	                      FakeService (handler tests)            FakeRepository (service tests)
//	                                                             real sqlite (repository tests)
//
// The proto wire types live one floor up (packages/proto/...) and never import
// this package; the handler is the only translation point (including proto
// BackendKind/CapPolicy/UsageState enum ↔ domain types).
package destinations

import (
	"fmt"
	"time"
)

// BackendKind classifies the kopia repository backend.
type BackendKind string

const (
	BackendFilesystem BackendKind = "filesystem"
	BackendS3         BackendKind = "s3"
)

// Valid reports whether k is a recognised, non-empty backend kind.
func (k BackendKind) Valid() bool {
	return k == BackendFilesystem || k == BackendS3
}

// CapPolicy is the over-cap behavior. Defaults to AlertBlock.
type CapPolicy string

const (
	// CapPolicyAlertBlock alerts and refuses new writes when usage would exceed
	// the cap. Never evicts. This is the default.
	CapPolicyAlertBlock CapPolicy = "alert_block"
	// CapPolicyAlertOnly alerts only; allows the write to proceed past the cap.
	CapPolicyAlertOnly CapPolicy = "alert_only"
)

// UsageState classifies current usage against the cap.
type UsageState string

const (
	// UsageStateWithin means usage is comfortably under the cap (< 90%).
	UsageStateWithin UsageState = "within"
	// UsageStateNear means usage is approaching the cap (>= 90%, < 100%).
	UsageStateNear UsageState = "near"
	// UsageStateOver means usage is at or over the cap (>= 100%).
	UsageStateOver UsageState = "over"
)

// Destination is the internal domain shape for a registered backup destination.
// Distinct from the proto wire type; handlers translate at the boundary so the
// domain layer never imports proto.
type Destination struct {
	ID                  string
	Name                string
	BackendKind         BackendKind
	Location            string
	CapBytes            int64
	CapPolicy           CapPolicy
	EncryptionAlgorithm string
	SecretRef           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// CreateInput is the explicit DTO Service.CreateDestination accepts. Distinct
// from Destination so callers cannot pass an ID or timestamp the service has no
// way to honour — those belong to the persistence layer.
type CreateInput struct {
	Name      string
	Backend   BackendKind
	Location  string
	CapBytes  int64
	CapPolicy CapPolicy
	SecretRef string
}

// UpdateInput carries the mutable fields Service.UpdateDestination accepts.
type UpdateInput struct {
	ID        string
	CapBytes  int64
	CapPolicy CapPolicy
}

// ErrDestinationNotFound is the typed sentinel returned by Repository.GetByID /
// GetByName when no row matches. Handlers translate it into a 404 /
// connect.CodeNotFound.
type ErrDestinationNotFound struct {
	ID   string
	Name string
}

func (e ErrDestinationNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("destination %q not found", e.ID)
	}
	return fmt.Sprintf("destination %q not found", e.Name)
}

// ErrInvalidDestination is the typed sentinel returned by Service validation.
// Handlers translate it into a 400 / connect.CodeInvalidArgument carrying
// "<field>: <reason>".
type ErrInvalidDestination struct {
	Field  string
	Reason string
}

func (e ErrInvalidDestination) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
