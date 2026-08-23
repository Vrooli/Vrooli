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
	"path/filepath"
	"regexp"
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
	ID          string
	Name        string
	BackendKind BackendKind
	// Location is the operator-facing destination root. For a filesystem backend
	// this is the self-describing bundle root that holds README.txt, RECOVERY.txt,
	// vrooli-backup-destination.json, and the repositories/ subfolder. For S3 it
	// is the bucket/prefix reference.
	Location string
	// RepositoryLocation is the concrete backend repository path handed to
	// resource-kopia. For a filesystem backend it nests inside the bundle root
	// (RepositoryPath below); for S3 it equals Location.
	RepositoryLocation  string
	CapBytes            int64
	CapPolicy           CapPolicy
	EncryptionAlgorithm string
	SecretRef           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// repoSubdir is the bundle subfolder that holds the vanilla kopia repository.
// The operator-facing bundle root holds README/RECOVERY/manifest; the opaque
// repository bytes live under repositories/<slug>.kopia so a file browser never
// mistakes the encrypted repository for the destination itself.
const repoSubdir = "repositories"

// slugPattern is the slug-safe repository-name contract enforced on Name. A
// destination name is also its kopia repository name, so it must be a stable
// slug — never an arbitrary label with spaces or slashes (see plan §8).
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidSlug reports whether name is a slug-safe kopia repository name.
func ValidSlug(name string) bool {
	return slugPattern.MatchString(name)
}

// RepositoryPathFor derives the concrete filesystem kopia repository path for a
// bundle root and slug-safe name: <bundleRoot>/repositories/<name>.kopia. It is
// pure (filepath.Join only) so callers and tests agree on the layout without
// touching the filesystem; the OS separator is applied for the running platform.
func RepositoryPathFor(bundleRoot, name string) string {
	return filepath.Join(filepath.Clean(bundleRoot), repoSubdir, name+".kopia")
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
	// AcknowledgeReadinessFailure lets an operator create a filesystem
	// destination whose readiness report fails, for the cases where the failing
	// condition is a judgement call rather than a defect. It deliberately does
	// not override a refusal raised against the mount driver: that failure
	// protects the host, not the backup, and has a cheap remedy.
	AcknowledgeReadinessFailure bool
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
