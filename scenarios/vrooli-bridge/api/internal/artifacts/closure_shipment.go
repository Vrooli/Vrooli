package artifacts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/coreset"
)

// ClosureFile is one owner-selected source inventory member. Its content is
// supplied only for staging and is never persisted in the Bridge receipt store.
type ClosureFile struct {
	Path   string
	SHA256 string
	Mode   uint32
	Size   int64
	Data   []byte
}

type ClosureShipmentRequest struct {
	IdempotencyKey         string
	TargetID               string
	SourceRef              string
	Destination            string
	Closure                coreset.DependencyClosure
	InventoryDigest        string
	PriorRevision          string
	NewRevision            string
	Files                  []ClosureFile
	SelectionReceiptRef    string
	SelectionDigest        string
	DeploymentID           string
	MachineID              string
	NodeID                 string
	EnrollmentGeneration   uint64
	DesiredRevision        uint64
	SelectionAuthorization string `json:"-"`
}

type ClosureShipmentState string

const (
	ShipmentPrepared   ClosureShipmentState = "prepared"
	ShipmentStaged     ClosureShipmentState = "staged"
	ShipmentVerified   ClosureShipmentState = "verified"
	ShipmentActivated  ClosureShipmentState = "activated"
	ShipmentRolledBack ClosureShipmentState = "rolled_back"
	ShipmentRejected   ClosureShipmentState = "rejected"
	ShipmentUncertain  ClosureShipmentState = "uncertain"
)

// ClosureShipment is a durable intent/receipt. It binds the canonical BR1
// graph closure to its separate source-file inventory and target revision.
type ClosureShipment struct {
	ID                    string               `json:"id"`
	IdempotencyKey        string               `json:"idempotency_key"`
	TargetID              string               `json:"target_id"`
	SourceRef             string               `json:"source_ref"`
	Destination           string               `json:"destination"`
	ClosureDigest         string               `json:"closure_digest"`
	InventoryDigest       string               `json:"inventory_digest"`
	PriorRevision         string               `json:"prior_revision"`
	NewRevision           string               `json:"new_revision"`
	StageRef              string               `json:"stage_ref,omitempty"`
	State                 ClosureShipmentState `json:"state"`
	ActualInventoryDigest string               `json:"actual_inventory_digest,omitempty"`
	ActiveRevision        string               `json:"active_revision,omitempty"`
	ActualFiles           []StagedFile         `json:"actual_files,omitempty"`
	SelectionReceiptRef   string               `json:"selection_receipt_ref"`
	SelectionDigest       string               `json:"selection_digest"`
	DeploymentID          string               `json:"deployment_id"`
	MachineID             string               `json:"machine_id"`
	NodeID                string               `json:"node_id"`
	EnrollmentGeneration  uint64               `json:"enrollment_generation"`
	DesiredRevision       uint64               `json:"desired_revision"`
	Result                string               `json:"result,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

// ClosureNodeReceipt is the authenticated target response bound to one
// persisted shipment. The Connect handler supplies the already-authenticated
// node identity separately and calls RecordNodeReceipt only after it matches.
type ClosureNodeReceipt struct {
	ShipmentID, IdempotencyKey, TargetID, SourceRef, Destination          string
	ClosureDigest, ExpectedInventoryDigest, ActualInventoryDigest         string
	PriorRevision, NewRevision, ActiveRevision, StageRef                  string
	SelectionReceiptRef, SelectionDigest, DeploymentID, MachineID, NodeID string
	EnrollmentGeneration, DesiredRevision                                 uint64
	State                                                                 ClosureShipmentState
	ActualFiles                                                           []StagedFile
	Result, Error                                                         string
}

type StagedFile = coreset.FileInventoryEntry

// ClosureDestination owns node-side effects. Stage must not activate; Inspect
// reports the exact staged membership; Activate must be idempotent by stageRef
// and compare-and-swap priorRevision; Rollback restores that prior revision.
type ClosureDestination interface {
	CheckRevision(context.Context, ClosureShipment) error
	Stage(context.Context, ClosureShipment, []ClosureFile) (stageRef string, err error)
	Inspect(context.Context, string) ([]StagedFile, error)
	Activate(context.Context, ClosureShipment, string) error
	Rollback(context.Context, ClosureShipment, string) error
	Discard(context.Context, string) error
}

type ClosureShipmentRepository interface {
	CreateShipment(context.Context, ClosureShipment) (ClosureShipment, error)
	GetShipment(context.Context, string) (ClosureShipment, error)
	UpdateShipment(context.Context, ClosureShipment) error
}

// ClosureShipmentService is Bridge's owner-service boundary for selected
// closure shipment. Fault is a test seam invoked at durable stage boundaries.
type ClosureShipmentService struct {
	repo               ClosureShipmentRepository
	destination        ClosureDestination
	AuthorizeSelection func(context.Context, ClosureShipmentRequest) error
	TargetPolicy       func(context.Context, ClosureShipment) error
	clock              func() time.Time
	Fault              func(stage string) error
}

func NewClosureShipmentService(repo ClosureShipmentRepository, destination ClosureDestination, clock func() time.Time) *ClosureShipmentService {
	if clock == nil {
		clock = time.Now
	}
	return &ClosureShipmentService{repo: repo, destination: destination, clock: clock}
}

func (s *ClosureShipmentService) Ship(ctx context.Context, req ClosureShipmentRequest) (ClosureShipment, error) {
	if err := validateClosureShipmentBytes(req); err != nil {
		return ClosureShipment{}, err
	}
	shipment, err := s.Prepare(ctx, req)
	if err != nil {
		return shipment, err
	}
	if err = s.resume(ctx, &shipment, req.Files); err != nil {
		return shipment, err
	}
	return shipment, nil
}

// Prepare persists an owner-authorized shipment intent without target effects.
// The channel adapter uses it before pushing BEGIN/chunk/VERIFY/ACTIVATE
// frames; source bytes are passed to that adapter separately.
func (s *ClosureShipmentService) Prepare(ctx context.Context, req ClosureShipmentRequest) (ClosureShipment, error) {
	if err := validateClosureShipment(req); err != nil {
		return ClosureShipment{}, err
	}
	// Re-attach an existing durable operation before revalidating the selection
	// receipt. A response-loss retry must return the original operation even if
	// the one-shot selection authority has since advanced.
	prior, priorErr := s.repo.GetShipment(ctx, strings.TrimSpace(req.IdempotencyKey))
	if priorErr == nil {
		if !shipmentMatchesRequest(prior, req) {
			return prior, errors.New("idempotency key already belongs to a different closure shipment")
		}
		return prior, nil
	}
	var missing ErrDistributionNotFound
	if !errors.Is(priorErr, sql.ErrNoRows) && !errors.As(priorErr, &missing) {
		return ClosureShipment{}, priorErr
	}
	if s.AuthorizeSelection == nil {
		return ClosureShipment{}, errors.New("closure selection authority is not configured")
	}
	if err := s.AuthorizeSelection(ctx, req); err != nil {
		return ClosureShipment{}, err
	}
	now := s.clock().UTC()
	shipment := ClosureShipment{
		ID: uuid.NewString(), IdempotencyKey: strings.TrimSpace(req.IdempotencyKey), TargetID: strings.TrimSpace(req.TargetID), SourceRef: strings.TrimSpace(req.SourceRef),
		Destination: strings.TrimSpace(req.Destination), ClosureDigest: req.Closure.Digest,
		InventoryDigest: req.InventoryDigest, PriorRevision: strings.TrimSpace(req.PriorRevision),
		NewRevision: strings.TrimSpace(req.NewRevision), State: ShipmentPrepared, CreatedAt: now, UpdatedAt: now,
		SelectionReceiptRef: strings.TrimSpace(req.SelectionReceiptRef), SelectionDigest: strings.TrimSpace(req.SelectionDigest), DeploymentID: strings.TrimSpace(req.DeploymentID), MachineID: strings.TrimSpace(req.MachineID), NodeID: strings.TrimSpace(req.NodeID), EnrollmentGeneration: req.EnrollmentGeneration, DesiredRevision: req.DesiredRevision,
	}
	var err error
	shipment, err = s.repo.CreateShipment(ctx, shipment)
	if err != nil {
		return ClosureShipment{}, err
	}
	if !shipmentMatchesRequest(shipment, req) {
		return shipment, errors.New("idempotency key already belongs to a different closure shipment")
	}
	if s.TargetPolicy == nil {
		return shipment, errors.New("closure target policy is not configured")
	}
	if err := s.TargetPolicy(ctx, shipment); err != nil {
		shipment.State, shipment.Result = ShipmentRejected, err.Error()
		if saveErr := s.save(ctx, &shipment); saveErr != nil {
			return shipment, fmt.Errorf("target policy rejected closure shipment (%v); persist rejection: %w", err, saveErr)
		}
		return shipment, err
	}
	return shipment, nil
}

// Get returns the durable shipment receipt by id or idempotency key.
func (s *ClosureShipmentService) Get(ctx context.Context, id string) (ClosureShipment, error) {
	return s.repo.GetShipment(ctx, strings.TrimSpace(id))
}

// Resume recovers one durable intent after interruption. Prepared intents
// stage; staged intents re-verify membership and retry idempotent activation.
func (s *ClosureShipmentService) Resume(ctx context.Context, id string, files []ClosureFile) (ClosureShipment, error) {
	shipment, err := s.repo.GetShipment(ctx, strings.TrimSpace(id))
	if err != nil {
		return ClosureShipment{}, err
	}
	if err := validateFileManifest(files, shipment.InventoryDigest, true); err != nil {
		return shipment, err
	}
	if err := s.resume(ctx, &shipment, files); err != nil {
		return shipment, err
	}
	return shipment, nil
}

func (s *ClosureShipmentService) resume(ctx context.Context, shipment *ClosureShipment, files []ClosureFile) error {
	if shipment.State == ShipmentActivated {
		return nil
	}
	if shipment.State == ShipmentRolledBack || shipment.State == ShipmentRejected {
		return errors.New("closure shipment is terminal")
	}
	if s.TargetPolicy == nil {
		return errors.New("closure target policy is not configured")
	}
	if err := s.TargetPolicy(ctx, *shipment); err != nil {
		if shipment.StageRef != "" {
			_ = s.destination.Discard(ctx, shipment.StageRef)
		}
		shipment.State, shipment.Result = ShipmentRejected, err.Error()
		_ = s.save(ctx, shipment)
		return err
	}
	if shipment.State == ShipmentPrepared {
		// The revision CAS is a pre-effect fence. Once this durable intent has
		// staged or verified content, recovery must reattach to that same
		// operation; requiring the old revision again would reject a retry when
		// activation already succeeded but its receipt was lost. Destination
		// Stage/Activate remain responsible for idempotent replay and refusing a
		// conflicting/newer revision.
		if err := s.destination.CheckRevision(ctx, *shipment); err != nil {
			shipment.State, shipment.Result = ShipmentRejected, err.Error()
			_ = s.save(ctx, shipment)
			return err
		}
		ref, err := s.destination.Stage(ctx, *shipment, files)
		if err != nil {
			return err
		}
		if strings.TrimSpace(ref) == "" {
			return errors.New("closure destination returned an empty stage reference")
		}
		shipment.StageRef, shipment.State = ref, ShipmentStaged
		if err := s.save(ctx, shipment); err != nil {
			return err
		}
		if err := s.inject("staged"); err != nil {
			return err
		}
	}
	actual, err := s.destination.Inspect(ctx, shipment.StageRef)
	if err != nil {
		return err
	}
	if err := verifyStagedFiles(files, actual, shipment.InventoryDigest); err != nil {
		shipment.State, shipment.Result = ShipmentRejected, err.Error()
		_ = s.destination.Discard(ctx, shipment.StageRef)
		_ = s.save(ctx, shipment)
		return err
	}
	shipment.State, shipment.ActualInventoryDigest = ShipmentVerified, shipment.InventoryDigest
	shipment.ActualFiles = append([]StagedFile(nil), actual...)
	if err := s.save(ctx, shipment); err != nil {
		return err
	}
	if err := s.inject("verified"); err != nil {
		return err
	}
	if err := s.destination.Activate(ctx, *shipment, shipment.StageRef); err != nil {
		rbErr := s.destination.Rollback(ctx, *shipment, shipment.StageRef)
		if rbErr == nil {
			rbErr = s.destination.Discard(ctx, shipment.StageRef)
		}
		if rbErr == nil {
			shipment.State = ShipmentRolledBack
			shipment.Result = "activation failed; prior revision restored and staged files discarded"
		} else {
			shipment.Result = "activation failed; rollback uncertain: " + rbErr.Error()
		}
		_ = s.save(ctx, shipment)
		return fmt.Errorf("activate closure shipment: %w", err)
	}
	if err := s.inject("activated"); err != nil {
		return err
	}
	shipment.State, shipment.Result = ShipmentActivated, "closure digest verified and revision activated"
	shipment.ActualInventoryDigest, shipment.ActiveRevision = shipment.InventoryDigest, shipment.NewRevision
	return s.save(ctx, shipment)
}

// RecordNodeReceipt persists an authenticated target state while rejecting any
// response that changes the immutable shipment identity or verified inventory.
func (s *ClosureShipmentService) RecordNodeReceipt(ctx context.Context, r ClosureNodeReceipt) error {
	x, err := s.repo.GetShipment(ctx, r.ShipmentID)
	if err != nil {
		return err
	}
	if x.IdempotencyKey != r.IdempotencyKey || x.TargetID != r.TargetID || x.SourceRef != r.SourceRef || x.Destination != r.Destination || x.ClosureDigest != r.ClosureDigest || x.InventoryDigest != r.ExpectedInventoryDigest || x.PriorRevision != r.PriorRevision || x.NewRevision != r.NewRevision || x.SelectionReceiptRef != r.SelectionReceiptRef || x.SelectionDigest != r.SelectionDigest || x.DeploymentID != r.DeploymentID || x.MachineID != r.MachineID || x.NodeID != r.NodeID || x.EnrollmentGeneration != r.EnrollmentGeneration || x.DesiredRevision != r.DesiredRevision {
		return errors.New("closure shipment receipt binding mismatch")
	}
	if r.State == ShipmentVerified || r.State == ShipmentActivated {
		actualDigest, digestErr := StagedInventoryDigest(r.ActualFiles)
		if digestErr != nil || actualDigest != r.ActualInventoryDigest || r.ActualInventoryDigest != x.InventoryDigest {
			return errors.New("closure shipment actual inventory digest mismatch")
		}
	}
	if r.State == ShipmentActivated && r.ActiveRevision != x.NewRevision {
		return errors.New("activated closure receipt has the wrong active revision")
	}
	if r.State == ShipmentRolledBack && r.ActiveRevision != x.PriorRevision {
		return errors.New("rolled-back closure receipt does not show the prior revision")
	}
	if !validShipmentTransition(x.State, r.State) {
		return fmt.Errorf("invalid closure shipment transition %s -> %s", x.State, r.State)
	}
	if r.StageRef != "" {
		x.StageRef = r.StageRef
	}
	x.State = r.State
	x.Result = r.Result
	if r.Error != "" {
		x.Result = r.Error
	}
	x.ActualInventoryDigest = r.ActualInventoryDigest
	x.ActiveRevision = r.ActiveRevision
	x.ActualFiles = append([]StagedFile(nil), r.ActualFiles...)
	return s.save(ctx, &x)
}

func validShipmentTransition(from, to ClosureShipmentState) bool {
	if from == to {
		return true
	}
	switch from {
	case ShipmentPrepared:
		return to == ShipmentStaged || to == ShipmentVerified || to == ShipmentActivated || to == ShipmentRolledBack || to == ShipmentRejected || to == ShipmentUncertain
	case ShipmentStaged:
		return to == ShipmentVerified || to == ShipmentActivated || to == ShipmentRolledBack || to == ShipmentRejected || to == ShipmentUncertain
	case ShipmentVerified:
		return to == ShipmentActivated || to == ShipmentRolledBack || to == ShipmentRejected || to == ShipmentUncertain
	case ShipmentUncertain:
		return to == ShipmentStaged || to == ShipmentVerified || to == ShipmentActivated || to == ShipmentRolledBack || to == ShipmentRejected
	}
	return false
}

func (s *ClosureShipmentService) save(ctx context.Context, shipment *ClosureShipment) error {
	shipment.UpdatedAt = s.clock().UTC()
	return s.repo.UpdateShipment(ctx, *shipment)
}
func (s *ClosureShipmentService) inject(stage string) error {
	if s.Fault != nil {
		return s.Fault(stage)
	}
	return nil
}

func validateClosureShipment(req ClosureShipmentRequest) error {
	if strings.TrimSpace(req.IdempotencyKey) == "" || strings.TrimSpace(req.TargetID) == "" || strings.TrimSpace(req.SourceRef) == "" || strings.TrimSpace(req.Destination) == "" || strings.TrimSpace(req.NewRevision) == "" {
		return errors.New("idempotency key, target, source, destination and new revision are required")
	}
	if strings.TrimSpace(req.SelectionReceiptRef) == "" || strings.TrimSpace(req.SelectionDigest) == "" || strings.TrimSpace(req.DeploymentID) == "" || strings.TrimSpace(req.MachineID) == "" || strings.TrimSpace(req.NodeID) == "" || req.EnrollmentGeneration == 0 || req.DesiredRevision == 0 {
		return errors.New("immutable onboarding selection receipt and target revision fences are required")
	}
	if strings.TrimSpace(req.TargetID) != strings.TrimSpace(req.NodeID) {
		return errors.New("Bridge target ID must match the onboarding handoff node ID")
	}
	if strings.TrimSpace(req.NewRevision) != strconv.FormatUint(req.DesiredRevision, 10) {
		return errors.New("new closure revision must match the onboarding desired revision")
	}
	if err := coreset.VerifyDigest(req.Closure); err != nil {
		return err
	}
	return validateFileManifest(req.Files, req.InventoryDigest, false)
}

func shipmentMatchesRequest(x ClosureShipment, req ClosureShipmentRequest) bool {
	return x.IdempotencyKey == strings.TrimSpace(req.IdempotencyKey) &&
		x.TargetID == strings.TrimSpace(req.TargetID) &&
		x.SourceRef == strings.TrimSpace(req.SourceRef) &&
		x.Destination == strings.TrimSpace(req.Destination) &&
		x.ClosureDigest == req.Closure.Digest &&
		x.InventoryDigest == req.InventoryDigest &&
		x.PriorRevision == strings.TrimSpace(req.PriorRevision) &&
		x.NewRevision == strings.TrimSpace(req.NewRevision) &&
		x.SelectionReceiptRef == strings.TrimSpace(req.SelectionReceiptRef) &&
		x.SelectionDigest == strings.TrimSpace(req.SelectionDigest) &&
		x.DeploymentID == strings.TrimSpace(req.DeploymentID) &&
		x.MachineID == strings.TrimSpace(req.MachineID) &&
		x.NodeID == strings.TrimSpace(req.NodeID) &&
		x.EnrollmentGeneration == req.EnrollmentGeneration &&
		x.DesiredRevision == req.DesiredRevision
}

func validateClosureShipmentBytes(req ClosureShipmentRequest) error {
	if err := validateClosureShipment(req); err != nil {
		return err
	}
	return validateFileManifest(req.Files, req.InventoryDigest, true)
}

func validateFileManifest(files []ClosureFile, expected string, requireData bool) error {
	if len(files) == 0 {
		return errors.New("closure file inventory is empty")
	}
	seen := map[string]bool{}
	ordered := append([]ClosureFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, f := range ordered {
		p := strings.TrimSpace(f.Path)
		if err := coreset.ValidateInventoryPath(p); err != nil {
			return err
		}
		if seen[p] {
			return fmt.Errorf("duplicate closure path %q", p)
		}
		seen[p] = true
		size := f.Size
		if size == 0 {
			size = int64(len(f.Data))
		}
		if requireData || f.Data != nil {
			sum := sha256.Sum256(f.Data)
			if f.SHA256 != hex.EncodeToString(sum[:]) || int64(len(f.Data)) != size {
				return fmt.Errorf("closure file digest or size mismatch: %s", p)
			}
		}
	}
	actual, err := ClosureInventoryDigest(ordered)
	if err != nil {
		return err
	}
	if expected == "" || expected != actual {
		return errors.New("closure inventory digest mismatch")
	}
	return nil
}

// ClosureInventoryDigest binds exact path, bytes, mode and membership. It is
// intentionally separate from the BR1 graph closure digest.
func ClosureInventoryDigest(files []ClosureFile) (string, error) {
	ordered := append([]ClosureFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	canonical := make([]StagedFile, 0, len(ordered))
	for _, f := range ordered {
		size := f.Size
		if size == 0 {
			size = int64(len(f.Data))
		}
		if f.Data != nil || size == 0 {
			sum := sha256.Sum256(f.Data)
			if f.SHA256 != hex.EncodeToString(sum[:]) {
				return "", fmt.Errorf("closure file digest mismatch: %s", f.Path)
			}
		}
		canonical = append(canonical, StagedFile{Path: f.Path, SHA256: f.SHA256, Mode: normalizedMode(f.Mode), Size: size})
	}
	return StagedInventoryDigest(canonical)
}

func StagedInventoryDigest(files []StagedFile) (string, error) {
	return coreset.FileInventoryDigest(files)
}

func verifyStagedFiles(expected []ClosureFile, actual []StagedFile, inventoryDigest string) error {
	want := make([]StagedFile, 0, len(expected))
	for _, f := range expected {
		want = append(want, StagedFile{Path: f.Path, SHA256: f.SHA256, Mode: normalizedMode(f.Mode), Size: int64(len(f.Data))})
	}
	sort.Slice(want, func(i, j int) bool { return want[i].Path < want[j].Path })
	got := append([]StagedFile(nil), actual...)
	sort.Slice(got, func(i, j int) bool { return got[i].Path < got[j].Path })
	if len(want) != len(got) {
		return errors.New("staged closure membership differs from selected inventory")
	}
	for i := range want {
		if want[i] != got[i] {
			return fmt.Errorf("staged closure member mismatch at %q", want[i].Path)
		}
	}
	return nil
}

func normalizedMode(mode uint32) uint32 {
	return coreset.NormalizeFileMode(mode)
}
