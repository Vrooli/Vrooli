// Package experimentarms owns durable identities and lifecycle state for
// embedding experiment arms. Physical index work is delegated to provider
// adapters and is deliberately not performed by this package.
package experimentarms

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
	corestorage "github.com/vrooli/api-core/storage"
)

var (
	ErrArmNotFound         = errors.New("embedding experiment arm not found")
	ErrImmutableConflict   = errors.New("embedding experiment arm identity conflict")
	ErrVersionConflict     = errors.New("embedding experiment arm version conflict")
	ErrIllegalTransition   = errors.New("illegal embedding experiment arm transition")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different transition")
)

// SQLExecutor is the narrow durable persistence boundary used by Registry.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// EvidenceRef points to immutable owner evidence. Digest is SHA-256 over the
// referenced artifact; this registry records references without copying or
// weakening source evidence.
type EvidenceRef struct {
	Kind   string `json:"kind"`
	URI    string `json:"uri"`
	SHA256 string `json:"sha256"`
}

// ArmCreate binds the recipe and source evidence to exact caller and build
// identities. Those identities are descriptive evidence and do not grant a
// physical provider operation.
type ArmCreate struct {
	ExperimentID string                   `json:"experiment_id"`
	ArmID        string                   `json:"arm_id"`
	OwnerID      string                   `json:"owner_id"`
	ProviderID   string                   `json:"provider_id"`
	CallerID     string                   `json:"caller_id"`
	BuildID      string                   `json:"build_id"`
	Recipe       aisearch.EmbeddingRecipe `json:"-"`
	Evidence     []EvidenceRef            `json:"evidence"`
}

// Arm is a versioned immutable record plus its CAS-controlled lifecycle state.
type Arm struct {
	ExperimentID    string          `json:"experiment_id"`
	ArmID           string          `json:"arm_id"`
	OwnerID         string          `json:"owner_id"`
	ProviderID      string          `json:"provider_id"`
	CallerID        string          `json:"caller_id"`
	BuildID         string          `json:"build_id"`
	RecipeID        string          `json:"recipe_id"`
	RecipeCanonical json.RawMessage `json:"recipe_canonical"`
	Evidence        []EvidenceRef   `json:"evidence"`
	EvidenceDigest  string          `json:"evidence_digest"`
	State           State           `json:"state"`
	Version         int64           `json:"version"`
}

// State names durable orchestration milestones. This package only records
// transitions; it does not start builds, catch-up, promotion, rollback or GC.
type State string

const (
	StateDiscovered         State = "discovered"
	StateCapabilityVerified State = "capability_verified"
	StatePlanned            State = "planned"
	StateAllocated          State = "allocated"
	StateSnapshotPinned     State = "snapshot_pinned"
	StateBackfilling        State = "backfilling"
	StateCatchingUp         State = "catching_up"
	StateValidating         State = "validating"
	StateReadyForEval       State = "ready_for_eval"
	StateEvaluating         State = "evaluating"
	StateQualified          State = "qualified"
	StatePromoting          State = "promoting"
	StateActive             State = "active"
	StateObserving          State = "observing"
	StateRetired            State = "retired"
	StateRollbackCatchup    State = "rollback_catchup"
	StateRollingBack        State = "rolling_back"
	StateRolledBack         State = "rolled_back"
	StateCancelling         State = "cancelling"
	StateCancelled          State = "cancelled"
	StateFailed             State = "failed"
	StateGCEligible         State = "gc_eligible"
	StateDeleted            State = "deleted"
)

// TransitionRequest is an optimistic CAS plus a stable idempotency key.
type TransitionRequest struct {
	ArmID           string
	ExpectedState   State
	ExpectedVersion int64
	To              State
	IdempotencyKey  string
	Receipt         EvidenceRef
}

// Registry persists experiment arms and transition receipts in SQLite.
type Registry struct {
	db            SQLExecutor
	fleetOwnerID  string
	fleetAlias    string
	fleetCatalog  *aisearch.SQLiteFleetGenerationCatalog
	fleetBoundary *aisearch.SQLiteFleetAliasMutationBoundary
}

func New(db SQLExecutor) (*Registry, error) {
	if db == nil {
		return nil, errors.New("experiment arm registry database is required")
	}
	return &Registry{db: db}, nil
}

// NewForNamespace binds future fleet operations to the lifecycle-injected
// Search Hub namespace. The owner and collection are process identity; request
// payloads never provide either value. The registry deliberately does not
// construct a physical adapter until a source and exact-operation admission
// owner are available.
func NewForNamespace(db SQLExecutor, namespace corestorage.Namespace) (*Registry, error) {
	if db == nil || strings.TrimSpace(namespace.Root()) == "" {
		return nil, errors.New("embedding experiment registry requires a trusted storage namespace")
	}
	alias, err := namespace.Collection("embedding-fleet")
	if err != nil {
		return nil, fmt.Errorf("resolve Search Hub embedding fleet alias: %w", err)
	}
	catalogDB, ok := db.(aisearch.FleetCatalogDB)
	if !ok {
		return nil, errors.New("embedding experiment registry database does not expose the durable fleet catalog boundary")
	}
	fleetCatalog, err := aisearch.NewSQLiteFleetGenerationCatalog(catalogDB)
	if err != nil {
		return nil, err
	}
	fleetBoundary, err := aisearch.NewSQLiteFleetAliasMutationBoundary(db)
	if err != nil {
		return nil, err
	}
	return &Registry{db: db, fleetOwnerID: namespace.Root() + ".embedding-fleet", fleetAlias: alias, fleetCatalog: fleetCatalog, fleetBoundary: fleetBoundary}, nil
}

// NewProviderDescriptionFleetAdapter constructs the production Search Hub
// adapter from the namespace-bound registry, its own descriptor source, a
// complete runtime-verified encoder, and the resource owner's exact-operation
// admission gate. The caller's Source/OwnerID/Alias/Catalog/Boundary fields
// are ignored. Missing runtime or admission dependencies fail closed.
func (r *Registry) NewProviderDescriptionFleetAdapter(options aisearch.QdrantFleetOptions, encoder FleetDescriptionEncoder) (*aisearch.QdrantFleetAdapter, error) {
	if options.Admission == nil {
		return nil, aisearch.ErrFleetAdmissionRequired
	}
	source, err := r.NewProviderDescriptionFleetSource(encoder)
	if err != nil {
		return nil, err
	}
	options.Source = source
	return r.newQdrantFleetAdapter(options)
}

// NewProviderDescriptionFleetSemanticStore binds the steady-state semantic
// path to the same trusted Search Hub source, catalog, alias fence and
// lifecycle encoder as generation staging. It does not invent an admission
// issuer or peak estimator; missing semantic admission fails before adapter
// construction or any Qdrant request.
func (r *Registry) NewProviderDescriptionFleetSemanticStore(options aisearch.QdrantFleetOptions, encoder FleetDescriptionEncoder, semanticOptions aisearch.FleetSemanticOptions) (aisearch.FleetSemanticStore, error) {
	if semanticOptions.Admission == nil {
		return nil, aisearch.ErrFleetAdmissionRequired
	}
	if encoder == nil {
		return nil, errors.New("Search Hub semantic store requires its lifecycle-registered verified recipe encoder")
	}
	source, err := r.NewProviderDescriptionFleetSource(encoder)
	if err != nil {
		return nil, err
	}
	options.Source = source
	if r == nil || r.fleetOwnerID == "" || r.fleetAlias == "" || r.fleetCatalog == nil || r.fleetBoundary == nil {
		return nil, errors.New("Qdrant semantic adapter requires the trusted startup namespace")
	}
	options.OwnerID = r.fleetOwnerID
	options.Alias = r.fleetAlias
	options.Catalog = r.fleetCatalog
	options.AliasBoundary = r.fleetBoundary
	adapter, err := aisearch.NewQdrantFleetSemanticAdapter(options)
	if err != nil {
		return nil, err
	}
	semanticOptions.Embedder = encoder
	return adapter.NewSemanticStore(semanticOptions)
}

// newQdrantFleetAdapter is the lower-level injected seam used by adapter
// integration tests. Production construction must use
// NewProviderDescriptionFleetAdapter so the source is tied to this owner DB.
func (r *Registry) newQdrantFleetAdapter(options aisearch.QdrantFleetOptions) (*aisearch.QdrantFleetAdapter, error) {
	if r == nil || r.fleetOwnerID == "" || r.fleetAlias == "" || r.fleetCatalog == nil || r.fleetBoundary == nil {
		return nil, errors.New("Qdrant fleet adapter requires the trusted startup namespace")
	}
	options.OwnerID = r.fleetOwnerID
	options.Alias = r.fleetAlias
	options.Catalog = r.fleetCatalog
	options.AliasBoundary = r.fleetBoundary
	return aisearch.NewQdrantFleetAdapter(options)
}

// NewProviderDescriptionFleetSource binds the Search Hub source to the same
// database and namespace identity captured when this registry was created.
// Request values cannot select another scenario's source, owner, or alias.
func (r *Registry) NewProviderDescriptionFleetSource(encoder FleetDescriptionEncoder) (*ProviderDescriptionFleetSource, error) {
	if r == nil || r.fleetOwnerID == "" || r.fleetAlias == "" {
		return nil, errors.New("provider-description fleet source requires a trusted Search Hub namespace")
	}
	db, ok := r.db.(providerFleetSourceDB)
	if !ok {
		return nil, errors.New("provider-description fleet source requires the owner registry query boundary")
	}
	return newProviderDescriptionFleetSource(db, encoder)
}

// Initialize creates the registry's additive schema. Calling it repeatedly is
// safe, so startup can recover after a process restart.
func (r *Registry) Initialize(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, schemaSQL+"\n"+providerFleetSourceSchema+"\n"+qdrantFleetDiskOperationSchema)
	if err != nil {
		return fmt.Errorf("initialize embedding arm registry: %w", err)
	}
	if r.fleetCatalog != nil {
		if err := r.fleetCatalog.Initialize(ctx); err != nil {
			return fmt.Errorf("initialize embedding fleet catalog: %w", err)
		}
	}
	if r.fleetBoundary != nil {
		if err := r.fleetBoundary.Initialize(ctx); err != nil {
			return fmt.Errorf("initialize embedding fleet alias boundary: %w", err)
		}
	}
	return nil
}

func Schema() string {
	return schemaSQL + "\n" + aisearch.SQLiteFleetGenerationCatalogSchema() + "\n" + aisearch.SQLiteFleetAliasMutationBoundarySchema() + "\n" + providerFleetSourceSchema + "\n" + qdrantFleetDiskOperationSchema
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS embedding_experiment_arms (
  arm_id TEXT PRIMARY KEY,
  experiment_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  caller_id TEXT NOT NULL,
  build_id TEXT NOT NULL,
  recipe_id TEXT NOT NULL,
  recipe_canonical BLOB NOT NULL,
  evidence_json BLOB NOT NULL,
  evidence_digest TEXT NOT NULL,
  state TEXT NOT NULL,
  version INTEGER NOT NULL,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS embedding_arm_transitions (
  arm_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  request_json BLOB NOT NULL,
  receipt_json BLOB NOT NULL,
  result_json BLOB NOT NULL,
  PRIMARY KEY (arm_id, idempotency_key),
  FOREIGN KEY (arm_id) REFERENCES embedding_experiment_arms(arm_id)
);`

// Create inserts an arm, or returns the existing exact arm when the same
// immutable identity is retried. Reusing an arm ID with changed data conflicts.
func (r *Registry) Create(ctx context.Context, create ArmCreate) (Arm, error) {
	if r.fleetOwnerID != "" {
		create.OwnerID = r.fleetOwnerID
	}
	arm, err := makeArm(create)
	if err != nil {
		return Arm{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO embedding_experiment_arms
(arm_id, experiment_id, owner_id, provider_id, caller_id, build_id, recipe_id, recipe_canonical, evidence_json, evidence_digest, state, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
ON CONFLICT(arm_id) DO NOTHING`, arm.ArmID, arm.ExperimentID, arm.OwnerID, arm.ProviderID, arm.CallerID, arm.BuildID, arm.RecipeID, []byte(arm.RecipeCanonical), mustJSON(arm.Evidence), arm.EvidenceDigest, arm.State)
	if err != nil {
		return Arm{}, fmt.Errorf("insert embedding arm: %w", err)
	}
	stored, err := r.Get(ctx, arm.ArmID)
	if err != nil {
		return Arm{}, err
	}
	if !sameImmutableArm(stored, arm) {
		return Arm{}, ErrImmutableConflict
	}
	return stored, nil
}

func (r *Registry) Get(ctx context.Context, armID string) (Arm, error) {
	var arm Arm
	var recipe, evidence []byte
	err := r.db.QueryRowContext(ctx, `SELECT experiment_id, arm_id, owner_id, provider_id, caller_id, build_id, recipe_id, recipe_canonical, evidence_json, evidence_digest, state, version
FROM embedding_experiment_arms WHERE arm_id = ?`, strings.TrimSpace(armID)).Scan(&arm.ExperimentID, &arm.ArmID, &arm.OwnerID, &arm.ProviderID, &arm.CallerID, &arm.BuildID, &arm.RecipeID, &recipe, &evidence, &arm.EvidenceDigest, &arm.State, &arm.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return Arm{}, ErrArmNotFound
	}
	if err != nil {
		return Arm{}, fmt.Errorf("read embedding arm: %w", err)
	}
	arm.RecipeCanonical = append(json.RawMessage(nil), recipe...)
	if _, err := aisearch.EmbeddingRecipeFromCanonical(recipe, strings.TrimPrefix(arm.RecipeID, "sha256:")); err != nil {
		return Arm{}, fmt.Errorf("stored embedding recipe is corrupt: %w", err)
	}
	if err := json.Unmarshal(evidence, &arm.Evidence); err != nil {
		return Arm{}, fmt.Errorf("decode stored evidence references: %w", err)
	}
	if digestEvidence(arm.Evidence) != arm.EvidenceDigest {
		return Arm{}, errors.New("stored evidence references are corrupt: digest mismatch")
	}
	return arm, nil
}

// Transition performs a durable legal-state CAS. Repeating the same
// idempotency key and request returns the original transition receipt even if
// the arm has since advanced further. A key cannot be reused for another
// transition.
func (r *Registry) Transition(ctx context.Context, request TransitionRequest) (Arm, error) {
	request.ArmID = strings.TrimSpace(request.ArmID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.Receipt.Kind = strings.TrimSpace(request.Receipt.Kind)
	request.Receipt.URI = strings.TrimSpace(request.Receipt.URI)
	request.Receipt.SHA256 = strings.ToLower(strings.TrimSpace(request.Receipt.SHA256))
	if err := validateTransitionRequest(request); err != nil {
		return Arm{}, err
	}
	requestHash, err := transitionHash(request)
	if err != nil {
		return Arm{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Arm{}, fmt.Errorf("begin arm transition: %w", err)
	}
	defer tx.Rollback()
	var priorHash string
	var priorJSON []byte
	err = tx.QueryRowContext(ctx, `SELECT request_hash, result_json FROM embedding_arm_transitions WHERE arm_id = ? AND idempotency_key = ?`, request.ArmID, request.IdempotencyKey).Scan(&priorHash, &priorJSON)
	if err == nil {
		if priorHash != requestHash {
			return Arm{}, ErrIdempotencyConflict
		}
		var arm Arm
		if err := json.Unmarshal(priorJSON, &arm); err != nil {
			return Arm{}, fmt.Errorf("decode prior arm transition receipt: %w", err)
		}
		return arm, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Arm{}, fmt.Errorf("read prior arm transition: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE embedding_experiment_arms SET state = ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
WHERE arm_id = ? AND state = ? AND version = ?`, request.To, request.ArmID, request.ExpectedState, request.ExpectedVersion)
	if err != nil {
		if isSQLiteLocked(err) {
			_ = tx.Rollback()
			return Arm{}, r.versionConflict(ctx, request.ArmID)
		}
		return Arm{}, fmt.Errorf("compare-and-swap arm state: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Arm{}, fmt.Errorf("read arm CAS result: %w", err)
	}
	if changed != 1 {
		return Arm{}, r.versionConflict(ctx, request.ArmID)
	}
	arm, err := scanArm(tx.QueryRowContext(ctx, `SELECT experiment_id, arm_id, owner_id, provider_id, caller_id, build_id, recipe_id, recipe_canonical, evidence_json, evidence_digest, state, version FROM embedding_experiment_arms WHERE arm_id = ?`, request.ArmID))
	if err != nil {
		return Arm{}, err
	}
	resultJSON, err := json.Marshal(arm)
	if err != nil {
		return Arm{}, fmt.Errorf("encode arm transition receipt: %w", err)
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return Arm{}, fmt.Errorf("encode transition request: %w", err)
	}
	receiptJSON, err := json.Marshal(request.Receipt)
	if err != nil {
		return Arm{}, fmt.Errorf("encode transition evidence reference: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO embedding_arm_transitions(arm_id, idempotency_key, request_hash, request_json, receipt_json, result_json) VALUES (?, ?, ?, ?, ?, ?)`, request.ArmID, request.IdempotencyKey, requestHash, requestJSON, receiptJSON, resultJSON); err != nil {
		return Arm{}, fmt.Errorf("persist arm transition receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		if isSQLiteLocked(err) {
			return Arm{}, r.versionConflict(ctx, request.ArmID)
		}
		return Arm{}, fmt.Errorf("commit arm transition: %w", err)
	}
	return arm, nil
}

func (r *Registry) versionConflict(ctx context.Context, armID string) error {
	arm, err := r.Get(ctx, armID)
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: actual state=%s version=%d", ErrVersionConflict, arm.State, arm.Version)
}

func makeArm(create ArmCreate) (Arm, error) {
	create.ExperimentID = strings.TrimSpace(create.ExperimentID)
	create.ArmID = strings.TrimSpace(create.ArmID)
	create.OwnerID = strings.TrimSpace(create.OwnerID)
	create.ProviderID = strings.TrimSpace(create.ProviderID)
	create.CallerID = strings.TrimSpace(create.CallerID)
	create.BuildID = strings.TrimSpace(create.BuildID)
	for field, value := range map[string]string{"experiment_id": create.ExperimentID, "arm_id": create.ArmID, "owner_id": create.OwnerID, "provider_id": create.ProviderID, "caller_id": create.CallerID, "build_id": create.BuildID} {
		if value == "" {
			return Arm{}, fmt.Errorf("embedding arm %s is required", field)
		}
	}
	if err := create.Recipe.Verify(); err != nil {
		return Arm{}, fmt.Errorf("embedding arm recipe: %w", err)
	}
	if len(create.Evidence) == 0 {
		return Arm{}, errors.New("embedding arm source evidence references are required")
	}
	evidence := append([]EvidenceRef(nil), create.Evidence...)
	hasE5Reference := false
	hasW10Reference := false
	for i := range evidence {
		evidence[i].Kind = strings.TrimSpace(evidence[i].Kind)
		evidence[i].URI = strings.TrimSpace(evidence[i].URI)
		evidence[i].SHA256 = strings.ToLower(strings.TrimSpace(evidence[i].SHA256))
		if evidence[i].Kind == "" || evidence[i].URI == "" || !validDigest(evidence[i].SHA256) {
			return Arm{}, fmt.Errorf("embedding arm evidence reference %d is incomplete or has an invalid SHA-256 digest", i)
		}
		if strings.HasPrefix(evidence[i].Kind, "e5-") {
			hasE5Reference = true
		}
		if strings.HasPrefix(evidence[i].Kind, "w10-") {
			hasW10Reference = true
		}
	}
	if !hasE5Reference {
		return Arm{}, errors.New("embedding arm requires at least one E5 source-evidence reference")
	}
	if !hasW10Reference {
		return Arm{}, errors.New("embedding arm requires at least one W10 caller/build evidence reference")
	}
	sort.Slice(evidence, func(i, j int) bool {
		if evidence[i].Kind != evidence[j].Kind {
			return evidence[i].Kind < evidence[j].Kind
		}
		if evidence[i].URI != evidence[j].URI {
			return evidence[i].URI < evidence[j].URI
		}
		return evidence[i].SHA256 < evidence[j].SHA256
	})
	canonical, err := json.Marshal(evidence)
	if err != nil {
		return Arm{}, fmt.Errorf("encode evidence references: %w", err)
	}
	if err := json.Unmarshal(canonical, &evidence); err != nil {
		return Arm{}, fmt.Errorf("normalize evidence references: %w", err)
	}
	return Arm{ExperimentID: create.ExperimentID, ArmID: create.ArmID, OwnerID: create.OwnerID, ProviderID: create.ProviderID, CallerID: create.CallerID, BuildID: create.BuildID, RecipeID: create.Recipe.ID(), RecipeCanonical: create.Recipe.Canonical(), Evidence: evidence, EvidenceDigest: digestEvidence(evidence), State: StateDiscovered, Version: 1}, nil
}

func sameImmutableArm(a, b Arm) bool {
	return a.ExperimentID == b.ExperimentID && a.ArmID == b.ArmID && a.OwnerID == b.OwnerID && a.ProviderID == b.ProviderID && a.CallerID == b.CallerID && a.BuildID == b.BuildID && a.RecipeID == b.RecipeID && string(a.RecipeCanonical) == string(b.RecipeCanonical) && a.EvidenceDigest == b.EvidenceDigest && string(mustJSON(a.Evidence)) == string(mustJSON(b.Evidence))
}

func validateTransitionRequest(request TransitionRequest) error {
	if strings.TrimSpace(request.ArmID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return errors.New("arm_id and idempotency_key are required")
	}
	if request.ExpectedVersion < 1 {
		return errors.New("expected_version must be positive")
	}
	if !legalTransition(request.ExpectedState, request.To) {
		return fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, request.ExpectedState, request.To)
	}
	hasReceiptField := request.Receipt.Kind != "" || request.Receipt.URI != "" || request.Receipt.SHA256 != ""
	if !hasReceiptField {
		return errors.New("transition evidence receipt is required")
	}
	if request.Receipt.Kind == "" || request.Receipt.URI == "" || !validDigest(request.Receipt.SHA256) {
		return errors.New("transition receipt requires kind, URI and valid SHA-256")
	}
	return nil
}

func legalTransition(from, to State) bool {
	allowed := map[State][]State{
		StateDiscovered:         {StateCapabilityVerified, StateFailed, StateCancelling},
		StateCapabilityVerified: {StatePlanned, StateFailed, StateCancelling},
		StatePlanned:            {StateAllocated, StateFailed, StateCancelling},
		StateAllocated:          {StateSnapshotPinned, StateFailed, StateCancelling},
		StateSnapshotPinned:     {StateBackfilling, StateValidating, StateFailed, StateCancelling},
		StateBackfilling:        {StateCatchingUp, StateValidating, StateFailed, StateCancelling},
		StateCatchingUp:         {StateValidating, StateFailed, StateCancelling},
		StateValidating:         {StateReadyForEval, StateBackfilling, StateCatchingUp, StateFailed, StateCancelling},
		StateReadyForEval:       {StateEvaluating, StateFailed, StateCancelling},
		StateEvaluating:         {StateQualified, StateReadyForEval, StateFailed, StateCancelling},
		StateQualified:          {StatePromoting, StateFailed, StateCancelling},
		StatePromoting:          {StateActive, StateQualified, StateFailed},
		StateActive:             {StateObserving, StateRollbackCatchup},
		StateObserving:          {StateRetired, StateRollbackCatchup},
		StateRetired:            {StateGCEligible},
		StateRollbackCatchup:    {StateRollingBack, StateActive},
		StateRollingBack:        {StateRolledBack, StateRollbackCatchup},
		StateCancelling:         {StateCancelled, StateFailed},
		StateCancelled:          {StateGCEligible},
		StateFailed:             {StateCancelling, StateGCEligible},
		StateRolledBack:         {StateGCEligible},
		StateGCEligible:         {StateDeleted},
	}
	for _, candidate := range allowed[from] {
		if to == candidate {
			return true
		}
	}
	return false
}

func transitionHash(request TransitionRequest) (string, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode arm transition request: %w", err)
	}
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func scanArm(row interface{ Scan(...any) error }) (Arm, error) {
	var arm Arm
	var recipe, evidence []byte
	err := row.Scan(&arm.ExperimentID, &arm.ArmID, &arm.OwnerID, &arm.ProviderID, &arm.CallerID, &arm.BuildID, &arm.RecipeID, &recipe, &evidence, &arm.EvidenceDigest, &arm.State, &arm.Version)
	if err != nil {
		return Arm{}, err
	}
	arm.RecipeCanonical = append(json.RawMessage(nil), recipe...)
	if _, err := aisearch.EmbeddingRecipeFromCanonical(recipe, strings.TrimPrefix(arm.RecipeID, "sha256:")); err != nil {
		return Arm{}, fmt.Errorf("stored embedding recipe is corrupt: %w", err)
	}
	if err := json.Unmarshal(evidence, &arm.Evidence); err != nil {
		return Arm{}, fmt.Errorf("decode stored evidence references: %w", err)
	}
	if digestEvidence(arm.Evidence) != arm.EvidenceDigest {
		return Arm{}, errors.New("stored evidence references are corrupt: digest mismatch")
	}
	return arm, nil
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digestEvidence(evidence []EvidenceRef) string {
	hash := sha256.Sum256(mustJSON(evidence))
	return hex.EncodeToString(hash[:])
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func isSQLiteLocked(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "database table is locked")
}
