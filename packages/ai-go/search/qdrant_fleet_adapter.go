package aisearch

// This file provides the owner-side migration boundary for a Qdrant-backed
// embedding fleet. It deliberately requires an admission owner and a durable
// catalog; constructing the adapter alone cannot create or mutate collections.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrFleetAdmissionRequired           = errors.New("qdrant fleet: fresh operation admission required")
	ErrFleetIdentityConflict            = errors.New("qdrant fleet: immutable generation identity conflict")
	ErrFleetStateConflict               = errors.New("qdrant fleet: generation state conflict")
	ErrFleetAliasBoundaryRequired       = errors.New("qdrant fleet: owner-wide alias mutation boundary required")
	ErrFleetSemanticOperationConflict   = errors.New("qdrant fleet: semantic operation ID conflicts with durable request identity")
	ErrFleetSemanticOperationUnresolved = errors.New("qdrant fleet: semantic operation effect is unresolved; refusing unsafe replay")
	ErrFleetSemanticSequenceStale       = errors.New("qdrant fleet: source sequence is stale or conflicts with a pending operation")
)

type FleetOperation string

const (
	FleetStage          FleetOperation = "stage"
	FleetCatchUp        FleetOperation = "catch_up"
	FleetPromote        FleetOperation = "promote"
	FleetRollback       FleetOperation = "rollback"
	FleetCollect        FleetOperation = "collect"
	FleetSemanticQuery  FleetOperation = "semantic_query"
	FleetSemanticUpsert FleetOperation = "semantic_upsert"
	FleetSemanticDelete FleetOperation = "semantic_delete"
)

// FleetAdmission is an exact-operation, short-lived capacity decision from
// the supported resource owner. Implementations must reject stale, under-sized
// or differently targeted approvals. This package never synthesizes it.
type FleetAdmission struct {
	OperationID string
	ReceiptID   string
	TargetID    string
	PeakBytes   int64
	ObservedAt  time.Time
	ExpiresAt   time.Time
}

type FleetAdmissionReceipt struct {
	OperationID string         `json:"operation_id"`
	ReceiptID   string         `json:"receipt_id"`
	Operation   FleetOperation `json:"operation"`
	ArmID       string         `json:"arm_id"`
	TargetID    string         `json:"target_id"`
	PeakBytes   int64          `json:"peak_bytes"`
	ObservedAt  time.Time      `json:"observed_at"`
	ExpiresAt   time.Time      `json:"expires_at"`
}

type FleetAdmissionGate interface {
	AdmitQdrantFleetOperation(context.Context, FleetOperation, string, string, int64) (FleetAdmission, error)
}

// FleetAliasMutationBoundary is owner supplied and must be shared by every
// in-process writer for the endpoint+alias. Qdrant applies alias action batches
// atomically but does not condition them on the prior target; arbitrary
// out-of-band writers therefore remain outside this cooperative fence.
type FleetAliasMutationBoundary interface {
	AcquireFleetAliasMutation(context.Context, FleetAliasOperation) (FleetAliasMutationFence, error)
}

type FleetAliasMutationFence interface {
	Release(context.Context) error
}

type fleetAliasBoundaryDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// SQLiteFleetAliasMutationBoundary is an owner-wide, cross-process alias fence.
// A same-operation retry is rejected while the previous holder process is
// alive. After a process restart, only the exact same durable operation may
// reclaim a fence left behind by a dead process; a different operation stays
// blocked for explicit owner recovery.
type SQLiteFleetAliasMutationBoundary struct {
	db          fleetAliasBoundaryDB
	holderPID   int
	holderStart string
	processLive func(int, string) bool
}

type sqliteFleetAliasMutationFence struct {
	boundary    *SQLiteFleetAliasMutationBoundary
	operation   FleetAliasOperation
	holderPID   int
	holderStart string
}

var _ FleetAliasMutationBoundary = (*SQLiteFleetAliasMutationBoundary)(nil)

func NewSQLiteFleetAliasMutationBoundary(db fleetAliasBoundaryDB) (*SQLiteFleetAliasMutationBoundary, error) {
	if db == nil {
		return nil, errors.New("Qdrant alias boundary database is required")
	}
	pid := os.Getpid()
	start, err := processStartIdentity(pid)
	if err != nil {
		return nil, fmt.Errorf("read current process identity for Qdrant alias boundary: %w", err)
	}
	return &SQLiteFleetAliasMutationBoundary{db: db, holderPID: pid, holderStart: start, processLive: processInstanceIsLive}, nil
}

func SQLiteFleetAliasMutationBoundarySchema() string {
	return `CREATE TABLE IF NOT EXISTS qdrant_fleet_alias_fences (
  endpoint_id TEXT NOT NULL,
  alias_name TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  holder_pid INTEGER NOT NULL,
  holder_start TEXT NOT NULL,
  record_json BLOB NOT NULL,
  PRIMARY KEY(endpoint_id, alias_name)
);`
}

func (b *SQLiteFleetAliasMutationBoundary) Initialize(ctx context.Context) error {
	_, err := b.db.ExecContext(ctx, SQLiteFleetAliasMutationBoundarySchema())
	return err
}

func (b *SQLiteFleetAliasMutationBoundary) AcquireFleetAliasMutation(ctx context.Context, operation FleetAliasOperation) (FleetAliasMutationFence, error) {
	if b == nil || b.db == nil || strings.TrimSpace(operation.EndpointID) == "" || strings.TrimSpace(operation.Alias) == "" || strings.TrimSpace(operation.OwnerID) == "" || strings.TrimSpace(operation.OperationID) == "" {
		return nil, errors.New("Qdrant alias operation requires endpoint, alias, owner and operation identity")
	}
	encoded, err := json.Marshal(operation)
	if err != nil {
		return nil, err
	}
	result, err := b.db.ExecContext(ctx, `INSERT INTO qdrant_fleet_alias_fences(endpoint_id, alias_name, owner_id, operation_id, holder_pid, holder_start, record_json)
VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(endpoint_id, alias_name) DO NOTHING`, operation.EndpointID, operation.Alias, operation.OwnerID, operation.OperationID, b.holderPID, b.holderStart, encoded)
	if err != nil {
		return nil, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed == 1 {
		return sqliteFleetAliasMutationFence{boundary: b, operation: operation, holderPID: b.holderPID, holderStart: b.holderStart}, nil
	}
	var existing struct {
		OwnerID     string `db:"owner_id"`
		OperationID string `db:"operation_id"`
		HolderPID   int    `db:"holder_pid"`
		HolderStart string `db:"holder_start"`
		RecordJSON  []byte `db:"record_json"`
	}
	if err := b.db.QueryRowContext(ctx, `SELECT owner_id, operation_id, holder_pid, holder_start, record_json FROM qdrant_fleet_alias_fences WHERE endpoint_id = ? AND alias_name = ?`, operation.EndpointID, operation.Alias).Scan(&existing.OwnerID, &existing.OperationID, &existing.HolderPID, &existing.HolderStart, &existing.RecordJSON); err != nil {
		return nil, err
	}
	var previous FleetAliasOperation
	if err := json.Unmarshal(existing.RecordJSON, &previous); err != nil {
		return nil, err
	}
	if !sameFleetAliasOperation(previous, operation) || b.processLive(existing.HolderPID, existing.HolderStart) {
		return nil, ErrFleetStateConflict
	}
	result, err = b.db.ExecContext(ctx, `UPDATE qdrant_fleet_alias_fences SET owner_id = ?, operation_id = ?, holder_pid = ?, holder_start = ?, record_json = ?
WHERE endpoint_id = ? AND alias_name = ? AND owner_id = ? AND operation_id = ? AND holder_pid = ? AND holder_start = ?`, operation.OwnerID, operation.OperationID, b.holderPID, b.holderStart, encoded, operation.EndpointID, operation.Alias, existing.OwnerID, existing.OperationID, existing.HolderPID, existing.HolderStart)
	if err != nil {
		return nil, err
	}
	changed, err = result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed != 1 {
		return nil, ErrFleetStateConflict
	}
	return sqliteFleetAliasMutationFence{boundary: b, operation: operation, holderPID: b.holderPID, holderStart: b.holderStart}, nil
}

func (f sqliteFleetAliasMutationFence) Release(ctx context.Context) error {
	result, err := f.boundary.db.ExecContext(ctx, `DELETE FROM qdrant_fleet_alias_fences WHERE endpoint_id = ? AND alias_name = ? AND owner_id = ? AND operation_id = ? AND holder_pid = ? AND holder_start = ?`, f.operation.EndpointID, f.operation.Alias, f.operation.OwnerID, f.operation.OperationID, f.holderPID, f.holderStart)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrFleetStateConflict
	}
	return nil
}

func sameFleetAliasOperation(left, right FleetAliasOperation) bool {
	return left.EndpointID == right.EndpointID && left.OwnerID == right.OwnerID && left.Alias == right.Alias && left.OperationID == right.OperationID && left.Kind == right.Kind && left.ArmID == right.ArmID && left.Target == right.Target
}

func processStartIdentity(pid int) (string, error) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", err
	}
	end := strings.LastIndex(string(raw), ") ")
	if end < 0 {
		return "", errors.New("unexpected process stat format")
	}
	fields := strings.Fields(string(raw)[end+2:])
	if len(fields) <= 19 {
		return "", errors.New("process stat has no start-time field")
	}
	return strings.TrimSpace(fields[19]), nil
}

func processInstanceIsLive(pid int, start string) bool {
	current, err := processStartIdentity(pid)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return current == start
}

// CanonicalQdrantEndpointID normalizes the HTTP base URL used to scope durable
// alias locks. It rejects embedded credentials and query tokens; callers pass
// API keys through the dedicated header option.
func CanonicalQdrantEndpointID(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Hostname() == "" || (strings.ToLower(parsed.Scheme) != "http" && strings.ToLower(parsed.Scheme) != "https") {
		return "", errors.New("Qdrant endpoint must be an absolute http(s) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("Qdrant endpoint identity cannot contain userinfo, query, or fragment")
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	return scheme + "://" + host + path, nil
}

// FleetPoint carries an already encoded source point. The source owner binds
// the encoder/model to the immutable recipe before returning points here.
type FleetPoint struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Sparse  *SparseVector  `json:"sparse,omitempty"`
	Payload map[string]any `json:"payload"`
}

type FleetChange struct {
	Sequence string
	Upserts  []FleetPoint
	Deletes  []string
}

type FleetSnapshot struct {
	ID        string
	Watermark string
}

// FleetSource must provide a stable snapshot and a gap-free ordered change
// stream. A change's sequence is its durable cursor; deletes are tombstones.
type FleetSource interface {
	PinFleetSnapshot(context.Context, string) (FleetSnapshot, error)
	ReadFleetSnapshotPage(context.Context, string, FleetSnapshot, string, int) ([]FleetPoint, string, bool, error)
	ReadFleetChanges(context.Context, string, string, string, int) ([]FleetChange, string, error)
	CompareFleetSequence(context.Context, string, string) (int, error)
	FleetWatermark(context.Context) (string, error)
	ExpectedFleetPointCount(context.Context, string, string, string) (uint64, error)
	AcquireFleetCutover(context.Context, string) (FleetCutoverFence, error)
}

// FleetCutoverFence is a source-owner write fence. While held, writers must
// buffer or block source changes so the indexed watermark stays exact through
// the Qdrant alias transaction.
type FleetCutoverFence interface {
	CurrentWatermark(context.Context) (string, error)
	Release(context.Context) error
}

type FleetGenerationState string

const (
	FleetBuilding   FleetGenerationState = "building"
	FleetCandidate  FleetGenerationState = "candidate"
	FleetActive     FleetGenerationState = "active"
	FleetRetired    FleetGenerationState = "retired"
	FleetPromoting  FleetGenerationState = "promoting"
	FleetCollecting FleetGenerationState = "collecting"
	FleetProtected  FleetGenerationState = "protected"
	FleetDeleted    FleetGenerationState = "deleted"
)

type FleetGeneration struct {
	OwnerID                   string               `json:"owner_id"`
	Alias                     string               `json:"alias"`
	ArmID                     string               `json:"arm_id"`
	RecipeID                  string               `json:"recipe_id"`
	SnapshotID                string               `json:"snapshot_id"`
	SnapshotWatermark         string               `json:"snapshot_watermark"`
	CatchupWatermark          string               `json:"catchup_watermark"`
	Dimensions                int                  `json:"dimensions"`
	IndexSpecDigest           string               `json:"index_spec_digest"`
	DataVersion               int64                `json:"data_version"`
	LastSemanticOperationID   string               `json:"last_semantic_operation_id,omitempty"`
	LastSemanticRequestSHA256 string               `json:"last_semantic_request_sha256,omitempty"`
	ValidationDataVersion     int64                `json:"validation_data_version,omitempty"`
	ValidationWatermark       string               `json:"validation_watermark,omitempty"`
	ValidationExpected        uint64               `json:"validation_expected_points,omitempty"`
	ValidationActual          uint64               `json:"validation_actual_points,omitempty"`
	ValidationValid           bool                 `json:"validation_valid,omitempty"`
	ValidationDigest          string               `json:"validation_digest,omitempty"`
	CreatedAt                 time.Time            `json:"created_at"`
	RetiredAt                 time.Time            `json:"retired_at,omitempty"`
	RetentionExpiresAt        time.Time            `json:"retention_expires_at,omitempty"`
	QdrantDiskBytes           int64                `json:"qdrant_disk_bytes,omitempty"`
	QdrantDiskBytesKnown      bool                 `json:"qdrant_disk_bytes_known"`
	DiskObservedAt            time.Time            `json:"disk_observed_at,omitempty"`
	DiskReclaimedBytesKnown   bool                 `json:"disk_reclaimed_bytes_known"`
	Collection                string               `json:"collection"`
	RollbackTo                string               `json:"rollback_to,omitempty"`
	State                     FleetGenerationState `json:"state"`
	Cursor                    string               `json:"cursor,omitempty"`
	Version                   int64                `json:"version"`
	Protected                 bool                 `json:"protected"`
	UpdatedAt                 time.Time            `json:"updated_at"`
}

type FleetValidation struct {
	ArmID              string `json:"arm_id"`
	RecipeID           string `json:"recipe_id"`
	Watermark          string `json:"watermark"`
	DataVersion        int64  `json:"data_version"`
	ReceiptID          string `json:"receipt_id"`
	ExpectedPointCount uint64 `json:"expected_point_count"`
	ActualPointCount   uint64 `json:"actual_point_count"`
	Valid              bool   `json:"valid"`
}

// FleetAliasOperation is a durable owner+alias operation identity. The active
// fence has a process holder; this operation record remains until the same
// idempotent effect is replayed and reconciled after an interruption.
type FleetAliasOperation struct {
	OwnerID     string         `json:"owner_id"`
	EndpointID  string         `json:"endpoint_id"`
	Alias       string         `json:"alias"`
	OperationID string         `json:"operation_id"`
	Kind        FleetOperation `json:"kind"`
	ArmID       string         `json:"arm_id"`
	Target      string         `json:"target"`
	StartedAt   time.Time      `json:"started_at"`
}

// FleetGenerationCatalog is implemented by the collection owner using its
// durable database. Save uses compare-and-swap on Version (new rows use 0).
type FleetGenerationCatalog interface {
	LoadFleetGeneration(context.Context, string) (FleetGeneration, bool, error)
	LoadFleetGenerationByCollection(context.Context, string) (FleetGeneration, bool, error)
	SaveFleetGenerationCAS(context.Context, FleetGeneration, int64) error
	ListFleetGenerations(context.Context, string, string) ([]FleetGeneration, error)
	RecordFleetAdmissionReceipt(context.Context, FleetAdmissionReceipt) error
	ListFleetAdmissionReceipts(context.Context, string) ([]FleetAdmissionReceipt, error)
	AcquireFleetAliasOperation(context.Context, FleetAliasOperation) error
	ReleaseFleetAliasOperation(context.Context, FleetAliasOperation) error
}

// FleetCatalogDB lets a scenario store the shared catalog in its own durable
// SQLite database. The adapter never opens a database or chooses a path itself.
type FleetCatalogDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteFleetGenerationCatalog struct{ db FleetCatalogDB }

func NewSQLiteFleetGenerationCatalog(db FleetCatalogDB) (*SQLiteFleetGenerationCatalog, error) {
	if db == nil {
		return nil, errors.New("Qdrant fleet catalog database is required")
	}
	return &SQLiteFleetGenerationCatalog{db: db}, nil
}

func (c *SQLiteFleetGenerationCatalog) Initialize(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx, SQLiteFleetGenerationCatalogSchema())
	return err
}

// SQLiteFleetGenerationCatalogSchema returns the additive catalog schema for
// scenarios that apply database changes through a central schema registry.
func SQLiteFleetGenerationCatalogSchema() string {
	return `CREATE TABLE IF NOT EXISTS qdrant_fleet_generations (
  arm_id TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL,
  alias_name TEXT NOT NULL,
  recipe_id TEXT NOT NULL,
  snapshot_id TEXT NOT NULL,
  dimensions INTEGER NOT NULL,
  index_spec_digest TEXT NOT NULL,
  collection_name TEXT NOT NULL UNIQUE,
  version INTEGER NOT NULL,
  record_json BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS qdrant_fleet_admission_receipts (
  operation_id TEXT PRIMARY KEY,
  arm_id TEXT NOT NULL,
  record_json BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS qdrant_fleet_alias_operations (
  endpoint_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  alias_name TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  record_json BLOB NOT NULL,
  PRIMARY KEY(endpoint_id, alias_name)
);
CREATE TABLE IF NOT EXISTS qdrant_fleet_semantic_operations (
  endpoint_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  alias_name TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  state TEXT NOT NULL,
  request_sha256 TEXT NOT NULL,
  record_json BLOB NOT NULL,
  PRIMARY KEY(endpoint_id, owner_id, alias_name, operation_id)
);
CREATE TABLE IF NOT EXISTS qdrant_fleet_semantic_source_versions (
  endpoint_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  alias_name TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  source_id TEXT NOT NULL,
  source_sequence INTEGER NOT NULL,
  operation_id TEXT NOT NULL,
  request_sha256 TEXT NOT NULL,
  state TEXT NOT NULL,
  record_json BLOB NOT NULL,
  PRIMARY KEY(endpoint_id, owner_id, alias_name, tenant_id, source_id)
);
CREATE INDEX IF NOT EXISTS qdrant_fleet_generations_owner_alias ON qdrant_fleet_generations(owner_id, alias_name);`
}

func (c *SQLiteFleetGenerationCatalog) LoadFleetSemanticOperation(ctx context.Context, identity FleetSemanticOperationRecord) (FleetSemanticOperationRecord, bool, error) {
	var raw []byte
	err := c.db.QueryRowContext(ctx, `SELECT record_json FROM qdrant_fleet_semantic_operations WHERE endpoint_id = ? AND owner_id = ? AND alias_name = ? AND operation_id = ?`, identity.EndpointID, identity.OwnerID, identity.Alias, identity.OperationID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return FleetSemanticOperationRecord{}, false, nil
	}
	if err != nil {
		return FleetSemanticOperationRecord{}, false, err
	}
	var record FleetSemanticOperationRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return FleetSemanticOperationRecord{}, false, err
	}
	if !sameFleetSemanticOperationIdentity(record, identity) || record.RequestSHA256 != identity.RequestSHA256 {
		return FleetSemanticOperationRecord{}, false, ErrFleetSemanticOperationConflict
	}
	return record, true, nil
}

func (c *SQLiteFleetGenerationCatalog) BeginFleetSemanticOperation(ctx context.Context, record FleetSemanticOperationRecord) (FleetSemanticOperationRecord, error) {
	if err := validateFleetSemanticOperationRecord(record); err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	record.State = FleetSemanticOperationStarted
	record.ResultJSON = nil
	encoded, err := json.Marshal(record)
	if err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	result, err := c.db.ExecContext(ctx, `INSERT INTO qdrant_fleet_semantic_operations(endpoint_id, owner_id, alias_name, operation_id, state, request_sha256, record_json) VALUES(?, ?, ?, ?, ?, ?, ?) ON CONFLICT(endpoint_id, owner_id, alias_name, operation_id) DO NOTHING`, record.EndpointID, record.OwnerID, record.Alias, record.OperationID, record.State, record.RequestSHA256, encoded)
	if err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	if changed == 1 {
		return record, nil
	}
	current, exists, err := c.LoadFleetSemanticOperation(ctx, record)
	if err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	if !exists {
		return FleetSemanticOperationRecord{}, ErrFleetSemanticOperationUnresolved
	}
	if current.Operation != record.Operation || current.ArmID != record.ArmID || current.Collection != record.Collection || current.TenantID != record.TenantID {
		return FleetSemanticOperationRecord{}, ErrFleetSemanticOperationConflict
	}
	if current.State == FleetSemanticOperationCompleted {
		return current, nil
	}
	if current.ExpectedVersion != record.ExpectedVersion {
		return FleetSemanticOperationRecord{}, ErrFleetSemanticOperationConflict
	}
	current.Admission = record.Admission
	current.UpdatedAt = record.UpdatedAt
	encoded, err = json.Marshal(current)
	if err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	result, err = c.db.ExecContext(ctx, `UPDATE qdrant_fleet_semantic_operations SET record_json = ? WHERE endpoint_id = ? AND owner_id = ? AND alias_name = ? AND operation_id = ? AND state = ? AND request_sha256 = ?`, encoded, current.EndpointID, current.OwnerID, current.Alias, current.OperationID, FleetSemanticOperationStarted, current.RequestSHA256)
	if err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	changed, err = result.RowsAffected()
	if err != nil {
		return FleetSemanticOperationRecord{}, err
	}
	if changed != 1 {
		return FleetSemanticOperationRecord{}, ErrFleetSemanticOperationUnresolved
	}
	return current, nil
}

func (c *SQLiteFleetGenerationCatalog) CompleteFleetSemanticOperation(ctx context.Context, identity FleetSemanticOperationRecord, resultJSON []byte) error {
	current, exists, err := c.LoadFleetSemanticOperation(ctx, identity)
	if err != nil {
		return err
	}
	if !exists || current.Operation != identity.Operation || current.ArmID != identity.ArmID || current.Collection != identity.Collection || current.TenantID != identity.TenantID {
		return ErrFleetSemanticOperationConflict
	}
	if current.State == FleetSemanticOperationCompleted {
		if !bytes.Equal(current.ResultJSON, resultJSON) {
			return ErrFleetSemanticOperationConflict
		}
		return nil
	}
	current.State = FleetSemanticOperationCompleted
	current.ResultJSON = append([]byte(nil), resultJSON...)
	current.UpdatedAt = identity.UpdatedAt
	encoded, err := json.Marshal(current)
	if err != nil {
		return err
	}
	result, err := c.db.ExecContext(ctx, `UPDATE qdrant_fleet_semantic_operations SET state = ?, record_json = ? WHERE endpoint_id = ? AND owner_id = ? AND alias_name = ? AND operation_id = ? AND state = ? AND request_sha256 = ?`, FleetSemanticOperationCompleted, encoded, current.EndpointID, current.OwnerID, current.Alias, current.OperationID, FleetSemanticOperationStarted, current.RequestSHA256)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrFleetSemanticOperationUnresolved
	}
	return nil
}

func (c *SQLiteFleetGenerationCatalog) CheckFleetSemanticSourceSequences(ctx context.Context, versions []FleetSemanticSourceVersion) error {
	for _, version := range versions {
		if err := validateFleetSemanticSourceVersion(version); err != nil {
			return err
		}
		current, exists, err := c.loadFleetSemanticSourceVersion(ctx, version)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if current.SourceSequence == version.SourceSequence && current.OperationID == version.OperationID && current.RequestSHA256 == version.RequestSHA256 {
			continue
		}
		if current.State == FleetSemanticOperationStarted || version.SourceSequence <= current.SourceSequence {
			return ErrFleetSemanticSequenceStale
		}
	}
	return nil
}

func (c *SQLiteFleetGenerationCatalog) ReserveFleetSemanticSourceSequences(ctx context.Context, versions []FleetSemanticSourceVersion) error {
	for _, version := range versions {
		if err := validateFleetSemanticSourceVersion(version); err != nil {
			return err
		}
		version.State = FleetSemanticOperationStarted
		encoded, err := json.Marshal(version)
		if err != nil {
			return err
		}
		result, err := c.db.ExecContext(ctx, `INSERT INTO qdrant_fleet_semantic_source_versions(endpoint_id, owner_id, alias_name, tenant_id, source_id, source_sequence, operation_id, request_sha256, state, record_json) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(endpoint_id, owner_id, alias_name, tenant_id, source_id) DO NOTHING`, version.EndpointID, version.OwnerID, version.Alias, version.TenantID, version.SourceID, version.SourceSequence, version.OperationID, version.RequestSHA256, version.State, encoded)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed == 1 {
			continue
		}
		current, exists, err := c.loadFleetSemanticSourceVersion(ctx, version)
		if err != nil {
			return err
		}
		if !exists {
			return ErrFleetSemanticOperationUnresolved
		}
		if current.SourceSequence == version.SourceSequence && current.OperationID == version.OperationID && current.RequestSHA256 == version.RequestSHA256 {
			continue
		}
		if current.State == FleetSemanticOperationStarted || version.SourceSequence <= current.SourceSequence {
			return ErrFleetSemanticSequenceStale
		}
		update, err := c.db.ExecContext(ctx, `UPDATE qdrant_fleet_semantic_source_versions SET source_sequence = ?, operation_id = ?, request_sha256 = ?, state = ?, record_json = ? WHERE endpoint_id = ? AND owner_id = ? AND alias_name = ? AND tenant_id = ? AND source_id = ? AND source_sequence = ? AND state = ?`, version.SourceSequence, version.OperationID, version.RequestSHA256, version.State, encoded, version.EndpointID, version.OwnerID, version.Alias, version.TenantID, version.SourceID, current.SourceSequence, current.State)
		if err != nil {
			return err
		}
		changed, err = update.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return ErrFleetSemanticOperationUnresolved
		}
	}
	return nil
}

func (c *SQLiteFleetGenerationCatalog) CompleteFleetSemanticSourceSequences(ctx context.Context, operation FleetSemanticOperationRecord, versions []FleetSemanticSourceVersion) error {
	for _, expected := range versions {
		if expected.OwnerID != operation.OwnerID || expected.EndpointID != operation.EndpointID || expected.Alias != operation.Alias || expected.OperationID != operation.OperationID || expected.RequestSHA256 != operation.RequestSHA256 {
			return ErrFleetSemanticOperationConflict
		}
		current, exists, err := c.loadFleetSemanticSourceVersion(ctx, expected)
		if err != nil {
			return err
		}
		if !exists || current.SourceSequence != expected.SourceSequence || current.OperationID != operation.OperationID || current.RequestSHA256 != operation.RequestSHA256 {
			return ErrFleetSemanticOperationUnresolved
		}
		if current.State == FleetSemanticOperationCompleted {
			continue
		}
		current.State = FleetSemanticOperationCompleted
		encoded, err := json.Marshal(current)
		if err != nil {
			return err
		}
		result, err := c.db.ExecContext(ctx, `UPDATE qdrant_fleet_semantic_source_versions SET state = ?, record_json = ? WHERE endpoint_id = ? AND owner_id = ? AND alias_name = ? AND tenant_id = ? AND source_id = ? AND source_sequence = ? AND operation_id = ? AND request_sha256 = ? AND state = ?`, FleetSemanticOperationCompleted, encoded, current.EndpointID, current.OwnerID, current.Alias, current.TenantID, current.SourceID, current.SourceSequence, operation.OperationID, operation.RequestSHA256, FleetSemanticOperationStarted)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return ErrFleetSemanticOperationUnresolved
		}
	}
	return nil
}

func (c *SQLiteFleetGenerationCatalog) loadFleetSemanticSourceVersion(ctx context.Context, identity FleetSemanticSourceVersion) (FleetSemanticSourceVersion, bool, error) {
	var raw []byte
	err := c.db.QueryRowContext(ctx, `SELECT record_json FROM qdrant_fleet_semantic_source_versions WHERE endpoint_id = ? AND owner_id = ? AND alias_name = ? AND tenant_id = ? AND source_id = ?`, identity.EndpointID, identity.OwnerID, identity.Alias, identity.TenantID, identity.SourceID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return FleetSemanticSourceVersion{}, false, nil
	}
	if err != nil {
		return FleetSemanticSourceVersion{}, false, err
	}
	var version FleetSemanticSourceVersion
	if err := json.Unmarshal(raw, &version); err != nil {
		return FleetSemanticSourceVersion{}, false, err
	}
	return version, true, nil
}

func validateFleetSemanticSourceVersion(version FleetSemanticSourceVersion) error {
	if strings.TrimSpace(version.OwnerID) == "" || strings.TrimSpace(version.EndpointID) == "" || strings.TrimSpace(version.Alias) == "" || strings.TrimSpace(version.TenantID) == "" || strings.TrimSpace(version.SourceID) == "" || version.SourceSequence <= 0 || strings.TrimSpace(version.OperationID) == "" || !validFleetEvidenceDigest(version.RequestSHA256) {
		return errors.New("fleet semantic source sequence identity is incomplete")
	}
	return nil
}

func validateFleetSemanticOperationRecord(record FleetSemanticOperationRecord) error {
	if strings.TrimSpace(record.OwnerID) == "" || strings.TrimSpace(record.EndpointID) == "" || strings.TrimSpace(record.Alias) == "" || strings.TrimSpace(record.OperationID) == "" || record.Operation == "" || strings.TrimSpace(record.ArmID) == "" || strings.TrimSpace(record.Collection) == "" || strings.TrimSpace(record.TenantID) == "" || !validFleetEvidenceDigest(record.RequestSHA256) || record.ExpectedVersion < 0 || record.Admission.ReceiptID == "" {
		return errors.New("durable semantic operation record is incomplete")
	}
	return nil
}

func sameFleetSemanticOperationIdentity(left, right FleetSemanticOperationRecord) bool {
	return left.OwnerID == right.OwnerID && left.EndpointID == right.EndpointID && left.Alias == right.Alias && left.OperationID == right.OperationID
}

func (c *SQLiteFleetGenerationCatalog) AcquireFleetAliasOperation(ctx context.Context, operation FleetAliasOperation) error {
	encoded, err := json.Marshal(operation)
	if err != nil {
		return err
	}
	result, err := c.db.ExecContext(ctx, `INSERT INTO qdrant_fleet_alias_operations(endpoint_id, owner_id, alias_name, operation_id, record_json) VALUES(?, ?, ?, ?, ?) ON CONFLICT(endpoint_id, alias_name) DO NOTHING`, operation.EndpointID, operation.OwnerID, operation.Alias, operation.OperationID, encoded)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 1 {
		return nil
	}
	var currentID string
	var currentRaw []byte
	if err := c.db.QueryRowContext(ctx, `SELECT operation_id, record_json FROM qdrant_fleet_alias_operations WHERE endpoint_id = ? AND alias_name = ?`, operation.EndpointID, operation.Alias).Scan(&currentID, &currentRaw); err != nil {
		return err
	}
	var current FleetAliasOperation
	if err := json.Unmarshal(currentRaw, &current); err != nil {
		return err
	}
	if currentID == operation.OperationID && current.OwnerID == operation.OwnerID && current.EndpointID == operation.EndpointID && current.Kind == operation.Kind && current.ArmID == operation.ArmID && current.Target == operation.Target {
		return nil
	}
	return ErrFleetStateConflict
}

func (c *SQLiteFleetGenerationCatalog) ReleaseFleetAliasOperation(ctx context.Context, operation FleetAliasOperation) error {
	result, err := c.db.ExecContext(ctx, `DELETE FROM qdrant_fleet_alias_operations WHERE endpoint_id = ? AND owner_id = ? AND alias_name = ? AND operation_id = ?`, operation.EndpointID, operation.OwnerID, operation.Alias, operation.OperationID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrFleetStateConflict
	}
	return nil
}

func (c *SQLiteFleetGenerationCatalog) LoadFleetGeneration(ctx context.Context, armID string) (FleetGeneration, bool, error) {
	return c.load(ctx, `SELECT record_json FROM qdrant_fleet_generations WHERE arm_id = ?`, strings.TrimSpace(armID))
}

func (c *SQLiteFleetGenerationCatalog) LoadFleetGenerationByCollection(ctx context.Context, collection string) (FleetGeneration, bool, error) {
	return c.load(ctx, `SELECT record_json FROM qdrant_fleet_generations WHERE collection_name = ?`, strings.TrimSpace(collection))
}

func (c *SQLiteFleetGenerationCatalog) load(ctx context.Context, query string, key string) (FleetGeneration, bool, error) {
	var raw []byte
	err := c.db.QueryRowContext(ctx, query, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return FleetGeneration{}, false, nil
	}
	if err != nil {
		return FleetGeneration{}, false, err
	}
	var record FleetGeneration
	if err := json.Unmarshal(raw, &record); err != nil {
		return FleetGeneration{}, false, fmt.Errorf("decode persisted Qdrant fleet generation: %w", err)
	}
	return record, true, nil
}

func (c *SQLiteFleetGenerationCatalog) SaveFleetGenerationCAS(ctx context.Context, record FleetGeneration, expectedVersion int64) error {
	if record.Version != expectedVersion+1 || record.ArmID == "" || record.OwnerID == "" || record.Alias == "" || record.Collection == "" || record.RecipeID == "" || record.SnapshotID == "" || record.Dimensions <= 0 || record.IndexSpecDigest == "" {
		return ErrFleetStateConflict
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	result, err := c.db.ExecContext(ctx, `INSERT INTO qdrant_fleet_generations(arm_id, owner_id, alias_name, recipe_id, snapshot_id, dimensions, index_spec_digest, collection_name, version, record_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(arm_id) DO UPDATE SET version=excluded.version, record_json=excluded.record_json
WHERE qdrant_fleet_generations.version = ? AND qdrant_fleet_generations.owner_id = ? AND qdrant_fleet_generations.alias_name = ? AND qdrant_fleet_generations.recipe_id = ? AND qdrant_fleet_generations.snapshot_id = ? AND qdrant_fleet_generations.dimensions = ? AND qdrant_fleet_generations.index_spec_digest = ? AND qdrant_fleet_generations.collection_name = ?`, record.ArmID, record.OwnerID, record.Alias, record.RecipeID, record.SnapshotID, record.Dimensions, record.IndexSpecDigest, record.Collection, record.Version, raw, expectedVersion, record.OwnerID, record.Alias, record.RecipeID, record.SnapshotID, record.Dimensions, record.IndexSpecDigest, record.Collection)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrFleetStateConflict
	}
	return nil
}

func (c *SQLiteFleetGenerationCatalog) ListFleetGenerations(ctx context.Context, ownerID, alias string) ([]FleetGeneration, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT record_json FROM qdrant_fleet_generations WHERE owner_id = ? AND alias_name = ? ORDER BY arm_id`, strings.TrimSpace(ownerID), strings.TrimSpace(alias))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []FleetGeneration
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var record FleetGeneration
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, fmt.Errorf("decode persisted Qdrant fleet generation: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (c *SQLiteFleetGenerationCatalog) RecordFleetAdmissionReceipt(ctx context.Context, receipt FleetAdmissionReceipt) error {
	if receipt.OperationID == "" || receipt.ReceiptID == "" || receipt.ArmID == "" || receipt.TargetID == "" || receipt.PeakBytes <= 0 {
		return ErrFleetAdmissionRequired
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	result, err := c.db.ExecContext(ctx, `INSERT INTO qdrant_fleet_admission_receipts(operation_id, arm_id, record_json) VALUES (?, ?, ?) ON CONFLICT(operation_id) DO NOTHING`, receipt.OperationID, receipt.ArmID, raw)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 1 {
		return nil
	}
	var previous []byte
	if err := c.db.QueryRowContext(ctx, `SELECT record_json FROM qdrant_fleet_admission_receipts WHERE operation_id = ?`, receipt.OperationID).Scan(&previous); err != nil {
		return err
	}
	if !bytes.Equal(previous, raw) {
		return ErrFleetIdentityConflict
	}
	return nil
}

func (c *SQLiteFleetGenerationCatalog) ListFleetAdmissionReceipts(ctx context.Context, armID string) ([]FleetAdmissionReceipt, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT record_json FROM qdrant_fleet_admission_receipts WHERE arm_id = ? ORDER BY operation_id`, strings.TrimSpace(armID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var receipts []FleetAdmissionReceipt
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var receipt FleetAdmissionReceipt
		if err := json.Unmarshal(raw, &receipt); err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	return receipts, rows.Err()
}

type QdrantFleetOptions struct {
	BaseURL       string
	APIKey        string
	OwnerID       string
	Alias         string
	IndexSpec     CollectionSpec
	RetiredMaxAge time.Duration
	PageSize      int
	Client        interface {
		Do(*http.Request) (*http.Response, error)
	}
	Admission     FleetAdmissionGate
	AliasBoundary FleetAliasMutationBoundary
	Source        FleetSource
	Catalog       FleetGenerationCatalog
	Now           func() time.Time
}

type QdrantFleetAdapter struct {
	baseURL, apiKey, ownerID, alias string
	indexSpec                       CollectionSpec
	pageSize                        int
	retiredMaxAge                   time.Duration
	client                          interface {
		Do(*http.Request) (*http.Response, error)
	}
	gate          FleetAdmissionGate
	aliasBoundary FleetAliasMutationBoundary
	source        FleetSource
	catalog       FleetGenerationCatalog
	now           func() time.Time
}

// FleetIdentity returns the immutable owner and stable alias this adapter was
// constructed to manage. It is safe for diagnostics and identity assertions;
// it does not grant access to an operation or change either value.
func (a *QdrantFleetAdapter) FleetIdentity() (ownerID, alias string) {
	if a == nil {
		return "", ""
	}
	return a.ownerID, a.alias
}

type unavailableFleetAdmission struct{}

func (unavailableFleetAdmission) AdmitQdrantFleetOperation(context.Context, FleetOperation, string, string, int64) (FleetAdmission, error) {
	return FleetAdmission{}, ErrFleetAdmissionRequired
}

// NewQdrantFleetSemanticAdapter constructs an adapter that is intentionally
// unavailable for generation lifecycle operations unless their separate
// resource gate is injected. Steady-state semantic methods use their own
// richer owner admission gate; omitting the migration gate never authorizes
// Stage/CatchUp/Promote/Rollback/Collect.
func NewQdrantFleetSemanticAdapter(options QdrantFleetOptions) (*QdrantFleetAdapter, error) {
	if options.Admission == nil {
		options.Admission = unavailableFleetAdmission{}
	}
	return NewQdrantFleetAdapter(options)
}

func NewQdrantFleetAdapter(options QdrantFleetOptions) (*QdrantFleetAdapter, error) {
	if strings.TrimSpace(options.BaseURL) == "" {
		options.BaseURL = DefaultQdrantURL
	}
	endpointID, err := CanonicalQdrantEndpointID(options.BaseURL)
	if err != nil {
		return nil, err
	}
	options.BaseURL = endpointID
	if strings.TrimSpace(options.OwnerID) == "" || !validMigrationCollection(strings.TrimSpace(options.Alias)) {
		return nil, errors.New("qdrant fleet owner and alias are required")
	}
	ownerNamespace := strings.SplitN(strings.TrimSpace(options.OwnerID), ".", 2)[0] + "_"
	if !strings.HasPrefix(strings.TrimSpace(options.Alias), ownerNamespace) {
		return nil, fmt.Errorf("Qdrant fleet alias %q must be namespaced beneath owner %q", options.Alias, ownerNamespace)
	}
	if options.PageSize <= 0 {
		options.PageSize = 128
	}
	if options.PageSize > 1024 {
		return nil, errors.New("qdrant fleet page size exceeds 1024")
	}
	if options.RetiredMaxAge < 0 {
		return nil, errors.New("Qdrant retired generation retention age cannot be negative")
	}
	if options.RetiredMaxAge == 0 {
		options.RetiredMaxAge = 72 * time.Hour
	}
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 30 * time.Second}
	}
	if options.Admission == nil {
		return nil, ErrFleetAdmissionRequired
	}
	if options.Source == nil || options.Catalog == nil {
		return nil, errors.New("qdrant fleet source and durable catalog are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &QdrantFleetAdapter{baseURL: strings.TrimRight(options.BaseURL, "/"), apiKey: options.APIKey, ownerID: strings.TrimSpace(options.OwnerID), alias: strings.TrimSpace(options.Alias), indexSpec: options.IndexSpec, pageSize: options.PageSize, retiredMaxAge: options.RetiredMaxAge, client: options.Client, gate: options.Admission, aliasBoundary: options.AliasBoundary, source: options.Source, catalog: options.Catalog, now: options.Now}, nil
}

// Stage creates an isolated immutable physical collection and fills it from a
// pinned source snapshot. Repeating the same arm/recipe/snapshot resumes from
// the catalog cursor; a changed identity fails closed.
func (a *QdrantFleetAdapter) Stage(ctx context.Context, armID string, recipe EmbeddingRecipe, snapshotID string, peakBytes int64) (FleetGeneration, error) {
	if err := recipe.Verify(); err != nil {
		return FleetGeneration{}, err
	}
	recipeInput, err := recipe.Input()
	if err != nil {
		return FleetGeneration{}, err
	}
	armID = strings.TrimSpace(armID)
	if !validMigrationCollection(armID) || strings.TrimSpace(snapshotID) == "" {
		return FleetGeneration{}, errors.New("qdrant fleet arm and snapshot identities are required")
	}
	collection := fleetCollectionName(a.alias, a.ownerID, armID, recipe.Digest())
	if !validMigrationCollection(collection) {
		return FleetGeneration{}, errors.New("qdrant fleet collection identity exceeds supported name bounds")
	}
	if _, err := a.admit(ctx, FleetStage, armID, collection, peakBytes); err != nil {
		return FleetGeneration{}, err
	}
	indexSpec, indexSpecDigest, err := a.resolvedIndexSpec(recipeInput.Dimensions, collection, recipeInput.ModelID)
	if err != nil {
		return FleetGeneration{}, err
	}
	record, exists, err := a.catalog.LoadFleetGeneration(ctx, armID)
	if err != nil {
		return FleetGeneration{}, err
	}
	if exists {
		if record.OwnerID != a.ownerID || record.Alias != a.alias || record.RecipeID != recipe.ID() || record.SnapshotID != snapshotID || record.Collection != collection || record.Dimensions != recipeInput.Dimensions || record.IndexSpecDigest != indexSpecDigest {
			return FleetGeneration{}, ErrFleetIdentityConflict
		}
		if record.State == FleetCandidate || record.State == FleetActive || record.State == FleetRetired {
			found, err := a.verifyExistingCollection(ctx, record.Collection, indexSpec)
			if err != nil {
				return FleetGeneration{}, err
			}
			if !found {
				return FleetGeneration{}, errors.New("catalogued Qdrant generation is missing")
			}
			return record, nil
		}
		if record.State != FleetBuilding {
			return FleetGeneration{}, ErrFleetStateConflict
		}
	} else {
		snapshot, err := a.source.PinFleetSnapshot(ctx, snapshotID)
		if err != nil {
			return FleetGeneration{}, err
		}
		if snapshot.ID != snapshotID || strings.TrimSpace(snapshot.Watermark) == "" {
			return FleetGeneration{}, errors.New("source returned an invalid pinned snapshot")
		}
		physical, err := a.physicalCollectionExists(ctx, collection)
		if err != nil {
			return FleetGeneration{}, err
		}
		if physical {
			return FleetGeneration{}, fmt.Errorf("untracked physical Qdrant collection %q is quarantined; owner must reconcile its identity before reuse", collection)
		}
		record = FleetGeneration{OwnerID: a.ownerID, Alias: a.alias, ArmID: armID, RecipeID: recipe.ID(), SnapshotID: snapshot.ID, SnapshotWatermark: snapshot.Watermark, Dimensions: recipeInput.Dimensions, IndexSpecDigest: indexSpecDigest, Collection: collection, State: FleetBuilding, Version: 1, CreatedAt: a.now().UTC(), UpdatedAt: a.now().UTC()}
		if err := a.catalog.SaveFleetGenerationCAS(ctx, record, 0); err != nil {
			return FleetGeneration{}, err
		}
	}
	if _, err := a.admitAndRecord(ctx, FleetStage, armID, collection, peakBytes); err != nil {
		return record, err
	}
	if err := a.createCollection(ctx, collection, indexSpec); err != nil {
		return record, err
	}
	snapshot := FleetSnapshot{ID: record.SnapshotID, Watermark: record.SnapshotWatermark}
	for {
		points, next, done, readErr := a.source.ReadFleetSnapshotPage(ctx, record.RecipeID, snapshot, record.Cursor, a.pageSize)
		if readErr != nil {
			return record, readErr
		}
		if err := validateFleetPoints(points, record.Dimensions, indexSpec.Sparse); err != nil {
			return record, err
		}
		if len(points) > 0 {
			if _, err := a.admitAndRecord(ctx, FleetStage, armID, collection, peakBytes); err != nil {
				return record, err
			}
			if err := a.upsertPoints(ctx, record.ArmID, collection, points, record.RecipeID, indexSpec); err != nil {
				return record, err
			}
			record.DataVersion++
		}
		oldVersion := record.Version
		record.Cursor = next
		record.Version++
		record.UpdatedAt = a.now().UTC()
		if done {
			record.State = FleetCandidate
			record.CatchupWatermark = record.SnapshotWatermark
		}
		if err := a.catalog.SaveFleetGenerationCAS(ctx, record, oldVersion); err != nil {
			return record, err
		}
		if done {
			return record, nil
		}
		if next == "" || next == snapshot.Watermark {
			return record, errors.New("source snapshot returned a non-advancing page cursor")
		}
	}
}

// CatchUp applies owner sequence changes, including tombstones, until the
// candidate reaches a fresh source watermark. A missing stream page is an
// error and leaves the active alias untouched.
func (a *QdrantFleetAdapter) CatchUp(ctx context.Context, armID string, peakBytes int64) (FleetGeneration, error) {
	record, ok, err := a.catalog.LoadFleetGeneration(ctx, strings.TrimSpace(armID))
	if err != nil {
		return FleetGeneration{}, err
	}
	if !ok {
		return FleetGeneration{}, ErrFleetStateConflict
	}
	if record.State != FleetCandidate && record.State != FleetRetired && record.State != FleetActive {
		return FleetGeneration{}, ErrFleetStateConflict
	}
	// Invalidate prior structural proof before the first possible point write.
	// If the process dies mid-page, replayed mutations cannot leave a stale
	// validation receipt eligible for promotion.
	if err := a.invalidateGenerationValidation(ctx, &record); err != nil {
		return record, err
	}
	for {
		barrier, err := a.source.FleetWatermark(ctx)
		if err != nil {
			return record, err
		}
		changes, next, err := a.source.ReadFleetChanges(ctx, record.RecipeID, record.SnapshotWatermark, record.CatchupWatermark, a.pageSize)
		if err != nil {
			return record, err
		}
		if len(changes) == 0 {
			if record.CatchupWatermark != barrier {
				return record, errors.New("source change stream has a gap before its current watermark")
			}
			return record, nil
		}
		if _, err := a.validateFleetChangePage(ctx, record.CatchupWatermark, changes, next); err != nil {
			return record, err
		}
		for _, change := range changes {
			if strings.TrimSpace(change.Sequence) == "" {
				return record, errors.New("source change lacks sequence identity")
			}
			spec, _, err := a.resolvedIndexSpec(record.Dimensions, record.Collection, "")
			if err != nil {
				return record, err
			}
			if err := validateFleetPoints(change.Upserts, record.Dimensions, spec.Sparse); err != nil {
				return record, err
			}
			if len(change.Deletes) > 0 {
				if _, err := a.admitAndRecord(ctx, FleetCatchUp, record.ArmID, record.Collection, peakBytes); err != nil {
					return record, err
				}
				if err := a.deleteSources(ctx, record.Collection, change.Deletes); err != nil {
					return record, err
				}
			}
			if len(change.Upserts) > 0 {
				if _, err := a.admitAndRecord(ctx, FleetCatchUp, record.ArmID, record.Collection, peakBytes); err != nil {
					return record, err
				}
				if err := a.upsertPoints(ctx, record.ArmID, record.Collection, change.Upserts, record.RecipeID, spec); err != nil {
					return record, err
				}
			}
			record.CatchupWatermark = change.Sequence
		}
		if next != "" && next != record.CatchupWatermark {
			return record, errors.New("source change cursor does not match applied sequence")
		}
		oldVersion := record.Version
		record.Version++
		record.UpdatedAt = a.now().UTC()
		if err := a.catalog.SaveFleetGenerationCAS(ctx, record, oldVersion); err != nil {
			return record, err
		}
		if record.CatchupWatermark == barrier {
			return record, nil
		}
	}
}

// Validate compares an exact Qdrant count for the recipe-tagged candidate
// against the owner source count at the same pinned watermark. It is structural
// migration proof; semantic quality evaluation remains with the evaluator.
func (a *QdrantFleetAdapter) Validate(ctx context.Context, armID string) (FleetValidation, error) {
	record, ok, err := a.catalog.LoadFleetGeneration(ctx, strings.TrimSpace(armID))
	if err != nil {
		return FleetValidation{}, err
	}
	if !ok || (record.State != FleetCandidate && record.State != FleetActive && record.State != FleetRetired) || record.CatchupWatermark == "" {
		return FleetValidation{}, ErrFleetStateConflict
	}
	return a.validateAndPersist(ctx, record)
}

// ObserveStorage records Qdrant's per-collection disk_data_size when exposed.
// It is a Qdrant-reported collection metric, not a host-filesystem reclaim
// receipt; unsupported or absent values are persisted as explicitly unknown.
func (a *QdrantFleetAdapter) ObserveStorage(ctx context.Context, armID string) (FleetGeneration, error) {
	record, ok, err := a.catalog.LoadFleetGeneration(ctx, strings.TrimSpace(armID))
	if err != nil {
		return FleetGeneration{}, err
	}
	if !ok || record.State == FleetDeleted {
		return FleetGeneration{}, ErrFleetStateConflict
	}
	var response struct {
		Result struct {
			DiskDataSize *int64 `json:"disk_data_size"`
		} `json:"result"`
	}
	if err := a.read(ctx, "/collections/"+url.PathEscape(record.Collection), &response); err != nil {
		return record, err
	}
	oldVersion := record.Version
	record.QdrantDiskBytesKnown = response.Result.DiskDataSize != nil && *response.Result.DiskDataSize >= 0
	record.QdrantDiskBytes = 0
	if record.QdrantDiskBytesKnown {
		record.QdrantDiskBytes = *response.Result.DiskDataSize
	}
	record.DiskObservedAt = a.now().UTC()
	record.Version++
	record.UpdatedAt = record.DiskObservedAt
	if err := a.catalog.SaveFleetGenerationCAS(ctx, record, oldVersion); err != nil {
		return FleetGeneration{}, err
	}
	return record, nil
}

func (a *QdrantFleetAdapter) acquireAliasOperation(ctx context.Context, kind FleetOperation, armID, target string) (FleetAliasOperation, FleetAliasMutationFence, error) {
	return a.acquireAliasOperationWithID(ctx, kind, armID, target, string(kind)+":"+armID+":"+target)
}

func (a *QdrantFleetAdapter) acquireAliasOperationWithID(ctx context.Context, kind FleetOperation, armID, target, operationID string) (FleetAliasOperation, FleetAliasMutationFence, error) {
	if a.aliasBoundary == nil {
		return FleetAliasOperation{}, nil, ErrFleetAliasBoundaryRequired
	}
	op := FleetAliasOperation{
		OwnerID: a.ownerID, EndpointID: a.baseURL, Alias: a.alias,
		OperationID: strings.TrimSpace(operationID),
		Kind:        kind, ArmID: armID, Target: target, StartedAt: a.now().UTC(),
	}
	if op.OperationID == "" {
		return FleetAliasOperation{}, nil, errors.New("Qdrant fleet alias operation ID is required")
	}
	fence, err := a.aliasBoundary.AcquireFleetAliasMutation(ctx, op)
	if err != nil {
		return FleetAliasOperation{}, nil, fmt.Errorf("acquire owner-wide Qdrant alias mutation boundary: %w", err)
	}
	if fence == nil {
		return FleetAliasOperation{}, nil, ErrFleetAliasBoundaryRequired
	}
	if err := a.catalog.AcquireFleetAliasOperation(ctx, op); err != nil {
		_ = fence.Release(context.WithoutCancel(ctx))
		return FleetAliasOperation{}, nil, fmt.Errorf("Qdrant alias %q has another unfinished operation: %w", a.alias, err)
	}
	return op, fence, nil
}

func (a *QdrantFleetAdapter) releaseAliasOperation(ctx context.Context, op FleetAliasOperation) error {
	if err := a.catalog.ReleaseFleetAliasOperation(ctx, op); err != nil {
		return fmt.Errorf("release durable Qdrant alias operation: %w", err)
	}
	return nil
}

func (a *QdrantFleetAdapter) validateAndPersist(ctx context.Context, record FleetGeneration) (FleetValidation, error) {
	expected, err := a.source.ExpectedFleetPointCount(ctx, record.RecipeID, record.SnapshotID, record.CatchupWatermark)
	if err != nil {
		return FleetValidation{}, err
	}
	actual, err := a.countRecipePoints(ctx, record.Collection, record.RecipeID)
	if err != nil {
		return FleetValidation{}, err
	}
	validation := FleetValidation{ArmID: record.ArmID, RecipeID: record.RecipeID, Watermark: record.CatchupWatermark, DataVersion: record.DataVersion, ExpectedPointCount: expected, ActualPointCount: actual, Valid: expected == actual}
	validation.ReceiptID = fleetValidationDigest(validation)
	oldVersion := record.Version
	record.ValidationDataVersion = record.DataVersion
	record.ValidationWatermark = record.CatchupWatermark
	record.ValidationExpected = expected
	record.ValidationActual = actual
	record.ValidationValid = validation.Valid
	record.ValidationDigest = validation.ReceiptID
	record.Version++
	record.UpdatedAt = a.now().UTC()
	if err := a.catalog.SaveFleetGenerationCAS(ctx, record, oldVersion); err != nil {
		return FleetValidation{}, err
	}
	return validation, nil
}

func (a *QdrantFleetAdapter) invalidateGenerationValidation(ctx context.Context, record *FleetGeneration) error {
	oldVersion := record.Version
	record.Version++
	record.DataVersion++
	record.ValidationValid = false
	record.ValidationDigest = ""
	record.ValidationWatermark = ""
	record.UpdatedAt = a.now().UTC()
	return a.catalog.SaveFleetGenerationCAS(ctx, *record, oldVersion)
}

// Promote atomically swaps the stable Qdrant alias only after source catch-up
// reaches the current watermark. The former active generation remains retained.
func (a *QdrantFleetAdapter) Promote(ctx context.Context, armID string, peakBytes int64) (result FleetGeneration, retErr error) {
	record, ok, err := a.catalog.LoadFleetGeneration(ctx, strings.TrimSpace(armID))
	if err != nil {
		return FleetGeneration{}, err
	}
	if !ok {
		return FleetGeneration{}, ErrFleetStateConflict
	}
	if record.OwnerID != a.ownerID || record.Alias != a.alias {
		return record, errors.New("generation owner/alias identity does not match adapter configuration")
	}
	operation, aliasFence, err := a.acquireAliasOperation(ctx, FleetPromote, record.ArmID, record.Collection)
	if err != nil {
		return record, err
	}
	defer func() {
		if err := aliasFence.Release(context.WithoutCancel(ctx)); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release owner-wide Qdrant alias boundary: %w", err))
		}
	}()
	completed := false
	defer func() {
		if completed {
			if err := a.releaseAliasOperation(context.WithoutCancel(ctx), operation); err != nil {
				retErr = errors.Join(retErr, err)
			}
		}
	}()
	if record.State != FleetCandidate && record.State != FleetRetired && record.State != FleetActive && record.State != FleetPromoting {
		return record, ErrFleetStateConflict
	}
	if !fleetGenerationValidationCurrent(record) {
		return record, errors.New("candidate lacks a current, valid recipe-bound structural validation receipt")
	}
	watermark, err := a.source.FleetWatermark(ctx)
	if err != nil {
		return record, err
	}
	if watermark != record.CatchupWatermark {
		return record, errors.New("candidate must catch up to a fresh source watermark before promotion")
	}
	fence, err := a.source.AcquireFleetCutover(ctx, record.CatchupWatermark)
	if err != nil {
		return record, err
	}
	defer func() {
		if err := fence.Release(context.WithoutCancel(ctx)); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release source cutover fence: %w", err))
		}
	}()
	fencedWatermark, err := fence.CurrentWatermark(ctx)
	if err != nil {
		return record, err
	}
	if fencedWatermark != record.CatchupWatermark {
		return record, errors.New("source changed while acquiring the cutover fence; catch up again")
	}
	previous, hasPrevious, err := a.aliasTarget(ctx)
	if err != nil {
		return record, err
	}
	if !hasPrevious {
		physical, err := a.physicalCollectionExists(ctx, a.alias)
		if err != nil {
			return record, err
		}
		if physical {
			return record, fmt.Errorf("stable Qdrant name %q is an existing physical collection; perform an owner-reviewed alias bootstrap before migration", a.alias)
		}
	} else if previous != record.Collection {
		previousRecord, found, err := a.catalog.LoadFleetGenerationByCollection(ctx, previous)
		if err != nil {
			return record, err
		}
		if !found || previousRecord.OwnerID != a.ownerID || previousRecord.Alias != a.alias || previousRecord.State != FleetActive || previousRecord.Protected {
			return record, fmt.Errorf("active Qdrant collection %q lacks a mutable owner catalog record for safe rollback", previous)
		}
	}
	if previous == record.Collection {
		if record.State != FleetActive {
			oldVersion := record.Version
			a.markGenerationActive(&record)
			record.Version++
			if err := a.catalog.SaveFleetGenerationCAS(ctx, record, oldVersion); err != nil {
				return record, err
			}
		}
		if record.RollbackTo != "" {
			if old, found, err := a.catalog.LoadFleetGenerationByCollection(ctx, record.RollbackTo); err != nil {
				return record, fmt.Errorf("active alias reconciled but previous generation lookup failed: %w", err)
			} else if found && old.State == FleetActive {
				oldVersion := old.Version
				a.markGenerationRetired(&old)
				old.Version++
				if err := a.catalog.SaveFleetGenerationCAS(ctx, old, oldVersion); err != nil {
					return record, fmt.Errorf("active alias reconciled but previous generation state is stale: %w", err)
				}
			}
		}
		completed = true
		return record, nil
	}
	if record.State != FleetPromoting || record.RollbackTo != previous {
		oldVersion := record.Version
		record.RollbackTo = previous
		record.State = FleetPromoting
		record.Version++
		record.UpdatedAt = a.now().UTC()
		if err := a.catalog.SaveFleetGenerationCAS(ctx, record, oldVersion); err != nil {
			return record, err
		}
	}
	if _, err := a.admitAndRecord(ctx, FleetPromote, record.ArmID, record.Collection, peakBytes); err != nil {
		return record, err
	}
	if err := a.setAlias(ctx, record.Collection, previous, hasPrevious); err != nil {
		return record, err
	}
	oldVersion := record.Version
	record.RollbackTo = previous
	a.markGenerationActive(&record)
	record.Version++
	record.UpdatedAt = a.now().UTC()
	if err := a.catalog.SaveFleetGenerationCAS(ctx, record, oldVersion); err != nil {
		return record, fmt.Errorf("alias switched but generation catalog write failed; reconcile from Qdrant alias: %w", err)
	}
	if hasPrevious {
		if old, found, e := a.catalog.LoadFleetGenerationByCollection(ctx, previous); e != nil {
			return record, fmt.Errorf("alias switched but previous generation lookup failed: %w", e)
		} else if found {
			a.markGenerationRetired(&old)
			old.Version++
			if e := a.catalog.SaveFleetGenerationCAS(ctx, old, old.Version-1); e != nil {
				return record, fmt.Errorf("alias switched but previous generation catalog update failed: %w", e)
			}
		}
	}
	completed = true
	return record, nil
}

// Rollback catches up the previous generation through the current source
// watermark before switching back, preserving visibility of new writes.
func (a *QdrantFleetAdapter) Rollback(ctx context.Context, armID string, peakBytes int64) (result FleetGeneration, retErr error) {
	record, ok, err := a.catalog.LoadFleetGeneration(ctx, strings.TrimSpace(armID))
	if err != nil {
		return FleetGeneration{}, err
	}
	if !ok {
		return record, ErrFleetStateConflict
	}
	if record.OwnerID != a.ownerID || record.Alias != a.alias {
		return record, errors.New("generation owner/alias identity does not match rollback adapter configuration")
	}
	if record.State == FleetRetired && record.RollbackTo != "" {
		// Recovery for a process that completed the Qdrant alias and catalog
		// writes but stopped before releasing its durable operation mutex.
		active, exists, err := a.aliasTarget(ctx)
		if err != nil {
			return record, err
		}
		if !exists || active != record.RollbackTo {
			return record, ErrFleetStateConflict
		}
		rollback, found, err := a.catalog.LoadFleetGenerationByCollection(ctx, record.RollbackTo)
		if err != nil {
			return record, err
		}
		if !found || rollback.OwnerID != a.ownerID || rollback.Alias != a.alias || rollback.Protected || rollback.State != FleetActive || !fleetGenerationValidationCurrent(rollback) {
			return record, errors.New("refusing rollback replay: target alias/catalog/validation state is not an unambiguous completed cutover")
		}
		operation, aliasFence, err := a.acquireAliasOperation(ctx, FleetRollback, record.ArmID, record.RollbackTo)
		if err != nil {
			return record, err
		}
		defer func() {
			if err := aliasFence.Release(context.WithoutCancel(ctx)); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("release owner-wide Qdrant alias boundary: %w", err))
			}
		}()
		active, exists, err = a.aliasTarget(ctx)
		if err != nil {
			return record, err
		}
		if !exists || active != record.RollbackTo {
			return record, ErrFleetStateConflict
		}
		if err := a.releaseAliasOperation(context.WithoutCancel(ctx), operation); err != nil {
			return record, err
		}
		return record, nil
	}
	if record.State != FleetActive || record.RollbackTo == "" {
		return record, ErrFleetStateConflict
	}
	operation, aliasFence, err := a.acquireAliasOperation(ctx, FleetRollback, record.ArmID, record.RollbackTo)
	if err != nil {
		return record, err
	}
	defer func() {
		if err := aliasFence.Release(context.WithoutCancel(ctx)); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release owner-wide Qdrant alias boundary: %w", err))
		}
	}()
	completed := false
	defer func() {
		if completed {
			if err := a.releaseAliasOperation(context.WithoutCancel(ctx), operation); err != nil {
				retErr = errors.Join(retErr, err)
			}
		}
	}()
	currentAlias, hasAlias, err := a.aliasTarget(ctx)
	if err != nil {
		return record, err
	}
	if !hasAlias || (currentAlias != record.Collection && currentAlias != record.RollbackTo) {
		return record, fmt.Errorf("refusing rollback: alias %q points to unexpected collection %q (expected current %q or rollback target %q)", a.alias, currentAlias, record.Collection, record.RollbackTo)
	}
	rollback, found, err := a.catalog.LoadFleetGenerationByCollection(ctx, record.RollbackTo)
	if err != nil {
		return record, err
	}
	if !found || rollback.Protected || rollback.State == FleetDeleted {
		return record, ErrFleetStateConflict
	}
	if rollback.State != FleetRetired && rollback.State != FleetActive {
		return record, ErrFleetStateConflict
	}
	if err := a.invalidateGenerationValidation(ctx, &rollback); err != nil {
		return record, err
	}
	rollbackWatermark, err := a.catchUpCollection(ctx, record.RollbackTo, rollback.SnapshotWatermark, rollback.CatchupWatermark, rollback.RecipeID, rollback.Dimensions, peakBytes, record.ArmID)
	if err != nil {
		return record, err
	}
	rollback.CatchupWatermark = rollbackWatermark
	oldRollbackVersion := rollback.Version
	rollback.Version++
	rollback.UpdatedAt = a.now().UTC()
	if err := a.catalog.SaveFleetGenerationCAS(ctx, rollback, oldRollbackVersion); err != nil {
		return record, err
	}
	validation, err := a.validateAndPersist(ctx, rollback)
	if err != nil {
		return record, err
	}
	if !validation.Valid {
		return record, errors.New("rollback target failed structural point-count validation")
	}
	rollback, found, err = a.catalog.LoadFleetGenerationByCollection(ctx, record.RollbackTo)
	if err != nil {
		return record, err
	}
	if !found {
		return record, ErrFleetStateConflict
	}
	fence, err := a.source.AcquireFleetCutover(ctx, rollback.CatchupWatermark)
	if err != nil {
		return record, err
	}
	defer func() {
		if err := fence.Release(context.WithoutCancel(ctx)); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release source cutover fence: %w", err))
		}
	}()
	fencedWatermark, err := fence.CurrentWatermark(ctx)
	if err != nil {
		return record, err
	}
	if fencedWatermark != rollback.CatchupWatermark {
		return record, errors.New("source changed while acquiring the rollback fence; catch up again")
	}
	if _, err := a.admitAndRecord(ctx, FleetRollback, record.ArmID, record.RollbackTo, peakBytes); err != nil {
		return record, err
	}
	currentAlias, hasAlias, err = a.aliasTarget(ctx)
	if err != nil {
		return record, err
	}
	if !hasAlias {
		return record, errors.New("refusing rollback: expected stable alias disappeared before cutover")
	}
	if currentAlias == record.Collection {
		if err := a.setAlias(ctx, record.RollbackTo, record.Collection, true); err != nil {
			return record, err
		}
	} else if currentAlias != record.RollbackTo {
		return record, fmt.Errorf("refusing rollback: alias %q changed to unexpected collection %q before cutover", a.alias, currentAlias)
	}
	oldRollbackVersion = rollback.Version
	a.markGenerationActive(&rollback)
	rollback.RollbackTo = record.Collection
	rollback.Version++
	rollback.UpdatedAt = a.now().UTC()
	if err := a.catalog.SaveFleetGenerationCAS(ctx, rollback, oldRollbackVersion); err != nil {
		return record, fmt.Errorf("alias rolled back but previous generation catalog update failed: %w", err)
	}
	old := record.Version
	a.markGenerationRetired(&record)
	record.Version++
	record.UpdatedAt = a.now().UTC()
	if err := a.catalog.SaveFleetGenerationCAS(ctx, record, old); err != nil {
		return record, err
	}
	completed = true
	return record, nil
}

// Collect deletes only explicitly listed, unprotected, non-active generations.
// It rechecks aliases and catalog state immediately before each deletion.
func (a *QdrantFleetAdapter) Collect(ctx context.Context, armIDs []string, peakBytes int64) error {
	for _, id := range armIDs {
		record, ok, err := a.catalog.LoadFleetGeneration(ctx, strings.TrimSpace(id))
		if err != nil {
			return err
		}
		if !ok || record.Protected || (record.State != FleetRetired && record.State != FleetCollecting && record.State != FleetDeleted) {
			return ErrFleetStateConflict
		}
		if record.OwnerID != a.ownerID || record.Alias != a.alias {
			return errors.New("generation owner/alias identity does not match collection adapter configuration")
		}
		operation, aliasFence, err := a.acquireAliasOperation(ctx, FleetCollect, record.ArmID, record.Collection)
		if err != nil {
			return err
		}
		operationErr := func() (operationErr error) {
			defer func() {
				if err := aliasFence.Release(context.WithoutCancel(ctx)); err != nil {
					operationErr = errors.Join(operationErr, fmt.Errorf("release owner-wide Qdrant alias boundary: %w", err))
				}
			}()
			if record.State == FleetRetired && (record.RetentionExpiresAt.IsZero() || a.now().Before(record.RetentionExpiresAt)) {
				return fmt.Errorf("Qdrant generation %q is still within its rollback retention window", record.Collection)
			}
			active, exists, err := a.aliasTarget(ctx)
			if err != nil {
				return err
			}
			if exists && active == record.Collection {
				return errors.New("refusing to collect active Qdrant generation")
			}
			aliased, err := a.collectionReferencedByAnyAlias(ctx, record.Collection)
			if err != nil {
				return err
			}
			if aliased {
				return fmt.Errorf("refusing to collect Qdrant generation %q while any alias references it", record.Collection)
			}
			latest, ok, err := a.catalog.LoadFleetGeneration(ctx, id)
			if err != nil {
				return err
			}
			if !ok || latest.Version != record.Version || latest.Protected || (latest.State != FleetRetired && latest.State != FleetCollecting && latest.State != FleetDeleted) {
				return ErrFleetStateConflict
			}
			all, err := a.catalog.ListFleetGenerations(ctx, a.ownerID, a.alias)
			if err != nil {
				return err
			}
			for _, other := range all {
				if other.ArmID != record.ArmID && (other.RollbackTo == record.Collection || other.State == FleetActive && other.Collection == record.Collection) {
					return fmt.Errorf("refusing to collect generation %q while another record protects it", record.Collection)
				}
			}
			if record.State == FleetDeleted {
				exists, err := a.physicalCollectionExists(ctx, record.Collection)
				if err != nil {
					return err
				}
				if exists {
					return fmt.Errorf("catalog marks Qdrant generation %q deleted but the physical collection exists; refusing a new delete", record.Collection)
				}
				// The physical delete and catalog CAS already completed. Replaying
				// this exact operation only drains its durable operation/fence rows.
				return a.releaseAliasOperation(context.WithoutCancel(ctx), operation)
			}
			if record.State == FleetCollecting {
				exists, err := a.physicalCollectionExists(ctx, record.Collection)
				if err != nil {
					return err
				}
				if !exists {
					old := record.Version
					record.State = FleetDeleted
					record.DiskReclaimedBytesKnown = false
					record.Version++
					record.UpdatedAt = a.now().UTC()
					if err := a.catalog.SaveFleetGenerationCAS(ctx, record, old); err != nil {
						return err
					}
					return a.releaseAliasOperation(context.WithoutCancel(ctx), operation)
				}
			}
			if record.State == FleetRetired {
				old := record.Version
				record.State = FleetCollecting
				record.Version++
				record.UpdatedAt = a.now().UTC()
				if err := a.catalog.SaveFleetGenerationCAS(ctx, record, old); err != nil {
					return err
				}
			}
			record, err = a.ObserveStorage(ctx, id)
			if err != nil {
				return err
			}
			if _, err := a.admitAndRecord(ctx, FleetCollect, record.ArmID, record.Collection, peakBytes); err != nil {
				return err
			}
			if err := a.deleteCollection(ctx, record.Collection); err != nil {
				return err
			}
			old := record.Version
			record.State = FleetDeleted
			record.Version++
			record.UpdatedAt = a.now().UTC()
			if err := a.catalog.SaveFleetGenerationCAS(ctx, record, old); err != nil {
				return err
			}
			return a.releaseAliasOperation(context.WithoutCancel(ctx), operation)
		}()
		if operationErr != nil {
			return operationErr
		}
	}
	return nil
}

func (a *QdrantFleetAdapter) admit(ctx context.Context, operation FleetOperation, arm, target string, peak int64) (FleetAdmission, error) {
	if peak <= 0 {
		return FleetAdmission{}, errors.New("qdrant fleet operation peak-byte estimate must be positive")
	}
	decision, err := a.gate.AdmitQdrantFleetOperation(ctx, operation, arm, target, peak)
	if err != nil {
		return FleetAdmission{}, err
	}
	now := a.now()
	if decision.OperationID == "" || decision.ReceiptID == "" || decision.TargetID != target || decision.PeakBytes < peak || decision.ObservedAt.After(now) || now.Sub(decision.ObservedAt) > time.Minute || !decision.ExpiresAt.After(now) {
		return FleetAdmission{}, ErrFleetAdmissionRequired
	}
	return decision, nil
}

func (a *QdrantFleetAdapter) admitAndRecord(ctx context.Context, operation FleetOperation, arm, target string, peak int64) (FleetAdmission, error) {
	decision, err := a.admit(ctx, operation, arm, target, peak)
	if err != nil {
		return FleetAdmission{}, err
	}
	receipt := FleetAdmissionReceipt{OperationID: decision.OperationID, ReceiptID: decision.ReceiptID, Operation: operation, ArmID: arm, TargetID: target, PeakBytes: decision.PeakBytes, ObservedAt: decision.ObservedAt, ExpiresAt: decision.ExpiresAt}
	if err := a.catalog.RecordFleetAdmissionReceipt(ctx, receipt); err != nil {
		return FleetAdmission{}, err
	}
	return decision, nil
}

func (a *QdrantFleetAdapter) markGenerationRetired(record *FleetGeneration) {
	record.State = FleetRetired
	if record.RetiredAt.IsZero() {
		record.RetiredAt = a.now().UTC()
		record.RetentionExpiresAt = record.RetiredAt.Add(a.retiredMaxAge)
	}
}

func (a *QdrantFleetAdapter) markGenerationActive(record *FleetGeneration) {
	record.State = FleetActive
	record.RetiredAt = time.Time{}
	record.RetentionExpiresAt = time.Time{}
}

func (a *QdrantFleetAdapter) catchUpCollection(ctx context.Context, collection, snapshot, after, recipeID string, dimensions int, peakBytes int64, armID string) (string, error) {
	for {
		barrier, err := a.source.FleetWatermark(ctx)
		if err != nil {
			return "", err
		}
		changes, next, err := a.source.ReadFleetChanges(ctx, recipeID, snapshot, after, a.pageSize)
		if err != nil {
			return "", err
		}
		if len(changes) == 0 {
			if after != barrier {
				return "", errors.New("source change stream has a gap before rollback watermark")
			}
			return after, nil
		}
		if _, err := a.validateFleetChangePage(ctx, after, changes, next); err != nil {
			return "", err
		}
		for _, c := range changes {
			if len(c.Deletes) > 0 {
				if _, err := a.admitAndRecord(ctx, FleetRollback, armID, collection, peakBytes); err != nil {
					return "", err
				}
				if err := a.deleteSources(ctx, collection, c.Deletes); err != nil {
					return "", err
				}
			}
			if len(c.Upserts) > 0 {
				spec, _, err := a.resolvedIndexSpec(dimensions, collection, "")
				if err != nil {
					return "", err
				}
				if err := validateFleetPoints(c.Upserts, dimensions, spec.Sparse); err != nil {
					return "", err
				}
				if _, err := a.admitAndRecord(ctx, FleetRollback, armID, collection, peakBytes); err != nil {
					return "", err
				}
				if err := a.upsertPoints(ctx, armID, collection, c.Upserts, recipeID, spec); err != nil {
					return "", err
				}
			}
			after = c.Sequence
		}
		if next != "" && next != after {
			return "", errors.New("source change cursor does not match applied sequence")
		}
	}
}

func fleetCollectionName(alias, owner, arm, digest string) string {
	identity := sha256.Sum256([]byte(alias + "\x00" + owner + "\x00" + arm + "\x00" + digest))
	return "vf-" + hex.EncodeToString(identity[:20])
}

func (a *QdrantFleetAdapter) validateFleetChangePage(ctx context.Context, after string, changes []FleetChange, next string) (string, error) {
	previous := after
	for _, change := range changes {
		if strings.TrimSpace(change.Sequence) == "" {
			return "", errors.New("source change lacks sequence identity")
		}
		order, err := a.source.CompareFleetSequence(ctx, previous, change.Sequence)
		if err != nil {
			return "", fmt.Errorf("compare source change sequence: %w", err)
		}
		if order <= 0 {
			return "", fmt.Errorf("source change sequence %q is duplicate or not after %q", change.Sequence, previous)
		}
		upserted := map[string]struct{}{}
		for _, point := range change.Upserts {
			sourceID, _ := point.Payload[sourceIDKey].(string)
			upserted[sourceID] = struct{}{}
		}
		for _, sourceID := range change.Deletes {
			if _, exists := upserted[sourceID]; exists {
				return "", fmt.Errorf("source change %q both upserts and tombstones source %q", change.Sequence, sourceID)
			}
		}
		previous = change.Sequence
	}
	if len(changes) == 0 {
		return after, nil
	}
	if next != previous {
		return "", errors.New("source change cursor does not match the final ordered sequence")
	}
	return previous, nil
}

func fleetValidationDigest(validation FleetValidation) string {
	validation.ReceiptID = ""
	canonical, _ := json.Marshal(validation)
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func fleetGenerationValidationCurrent(record FleetGeneration) bool {
	validation := FleetValidation{ArmID: record.ArmID, RecipeID: record.RecipeID, Watermark: record.CatchupWatermark, DataVersion: record.DataVersion, ExpectedPointCount: record.ValidationExpected, ActualPointCount: record.ValidationActual, Valid: record.ValidationValid}
	return record.ValidationValid && record.ValidationDataVersion == record.DataVersion && record.ValidationWatermark == record.CatchupWatermark && record.ValidationExpected == record.ValidationActual && record.ValidationDigest != "" && record.ValidationDigest == fleetValidationDigest(validation)
}

func validateFleetPoints(points []FleetPoint, dimensions int, sparseExpected bool) error {
	for _, p := range points {
		sourceID, _ := p.Payload[sourceIDKey].(string)
		if strings.TrimSpace(p.ID) == "" || len(p.Vector) != dimensions || strings.TrimSpace(sourceID) == "" || (p.Sparse != nil) != sparseExpected {
			return fmt.Errorf("invalid Qdrant fleet point %q: expected nonempty ID, source_id, %d-dimensional dense vector, and sparse=%t", p.ID, dimensions, sparseExpected)
		}
		if p.Sparse != nil && (len(p.Sparse.Indices) == 0 || len(p.Sparse.Indices) != len(p.Sparse.Values)) {
			return fmt.Errorf("invalid Qdrant fleet point %q: sparse indices and values must be nonempty and aligned", p.ID)
		}
	}
	return nil
}

func (a *QdrantFleetAdapter) resolvedIndexSpec(dimensions int, collection, modelID string) (CollectionSpec, string, error) {
	spec := a.indexSpec
	if spec.DenseSize > 0 && spec.DenseSize != dimensions {
		return CollectionSpec{}, "", errors.New("Qdrant index dense size does not match embedding recipe")
	}
	spec.DenseSize = dimensions
	if spec.DenseDistance == "" {
		spec.DenseDistance = DefaultDenseDistance
	}
	if spec.Model == "" {
		spec.Model = modelID
	}
	spec.Name = ""
	canonical, err := json.Marshal(spec)
	if err != nil {
		return CollectionSpec{}, "", err
	}
	digest := sha256.Sum256(canonical)
	spec.Name = collection
	return spec, hex.EncodeToString(digest[:]), nil
}

func (a *QdrantFleetAdapter) verifyExistingCollection(ctx context.Context, name string, spec CollectionSpec) (bool, error) {
	store := NewVectorStoreWithClient(a.baseURL, a.apiKey, name, a.client)
	qdrant := store.(*qdrantVectorStore)
	layout, found, err := qdrant.inspect(ctx)
	if err != nil || !found {
		return found, err
	}
	if err := qdrant.checkLayout(ctx, spec, spec.DenseSize, spec.DenseDistance, layout); err != nil {
		return false, err
	}
	return true, nil
}

func (a *QdrantFleetAdapter) createCollection(ctx context.Context, name string, spec CollectionSpec) error {
	store := NewVectorStoreWithClient(a.baseURL, a.apiKey, name, a.client)
	spec.Name = name
	return store.EnsureCollection(ctx, spec)
}

func (a *QdrantFleetAdapter) upsertPoints(ctx context.Context, armID, name string, points []FleetPoint, recipeID string, spec CollectionSpec) error {
	prepared := make([]Point, len(points))
	for i, p := range points {
		dense := make([]float64, len(p.Vector))
		for j, value := range p.Vector {
			dense[j] = float64(value)
		}
		prepared[i] = Point{ID: p.ID, Dense: dense, Sparse: p.Sparse, Payload: make(map[string]any, len(p.Payload)+1)}
		for k, v := range p.Payload {
			prepared[i].Payload[k] = v
		}
		prepared[i].Payload["embedding_recipe_id"] = recipeID
		prepared[i].Payload["fleet_owner_id"] = a.ownerID
		prepared[i].Payload["fleet_alias"] = a.alias
		prepared[i].Payload["fleet_arm_id"] = armID
	}
	store := NewVectorStoreWithClient(a.baseURL, a.apiKey, name, a.client)
	batch, ok := store.(BatchVectorStore)
	if !ok {
		return errors.New("Qdrant vector store does not support bounded batch writes")
	}
	batchSize := spec.Storage.UpsertBatchSize
	if batchSize <= 0 {
		batchSize = DefaultSourcePageSize
	}
	return batch.UpsertBatch(ctx, prepared, batchSize)
}
func (a *QdrantFleetAdapter) deleteSources(ctx context.Context, name string, sourceIDs []string) error {
	return a.request(ctx, http.MethodPost, "/collections/"+url.PathEscape(name)+"/points/delete?wait=true", map[string]any{"filter": map[string]any{"must": []any{map[string]any{"key": sourceIDKey, "match": map[string]any{"any": sourceIDs}}}}}, http.StatusOK)
}

func (a *QdrantFleetAdapter) countRecipePoints(ctx context.Context, name, recipeID string) (uint64, error) {
	var response struct {
		Result struct {
			Count uint64 `json:"count"`
		} `json:"result"`
	}
	body := map[string]any{"exact": true, "filter": map[string]any{"must": []any{map[string]any{"key": "embedding_recipe_id", "match": map[string]any{"value": recipeID}}}}}
	if err := a.requestReadJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(name)+"/points/count", body, &response); err != nil {
		return 0, err
	}
	return response.Result.Count, nil
}

func (a *QdrantFleetAdapter) requestReadJSON(ctx context.Context, method, path string, body, target any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("api-key", a.apiKey)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Qdrant %s %s returned HTTP %d", method, path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
func (a *QdrantFleetAdapter) deleteCollection(ctx context.Context, name string) error {
	return a.request(ctx, http.MethodDelete, "/collections/"+url.PathEscape(name), nil, http.StatusOK, http.StatusNotFound)
}
func (a *QdrantFleetAdapter) aliasTarget(ctx context.Context) (string, bool, error) {
	var response struct {
		Result struct {
			Aliases []struct {
				AliasName      string `json:"alias_name"`
				CollectionName string `json:"collection_name"`
			} `json:"aliases"`
		} `json:"result"`
	}
	if err := a.read(ctx, "/collections/aliases", &response); err != nil {
		return "", false, err
	}
	for _, entry := range response.Result.Aliases {
		if entry.AliasName == a.alias {
			return entry.CollectionName, true, nil
		}
	}
	return "", false, nil
}

func (a *QdrantFleetAdapter) collectionReferencedByAnyAlias(ctx context.Context, collection string) (bool, error) {
	var response struct {
		Result struct {
			Aliases []struct {
				CollectionName string `json:"collection_name"`
			} `json:"aliases"`
		} `json:"result"`
	}
	if err := a.read(ctx, "/collections/aliases", &response); err != nil {
		return false, err
	}
	for _, alias := range response.Result.Aliases {
		if alias.CollectionName == collection {
			return true, nil
		}
	}
	return false, nil
}

func (a *QdrantFleetAdapter) physicalCollectionExists(ctx context.Context, name string) (bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/collections/"+url.PathEscape(name), nil)
	if err != nil {
		return false, err
	}
	if a.apiKey != "" {
		request.Header.Set("api-key", a.apiKey)
	}
	response, err := a.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("inspect Qdrant collection %q returned HTTP %d", name, response.StatusCode)
	}
	return true, nil
}
func (a *QdrantFleetAdapter) setAlias(ctx context.Context, target, previous string, hasPrevious bool) error {
	current, exists, err := a.aliasTarget(ctx)
	if err != nil {
		return err
	}
	if hasPrevious {
		if !exists || current != previous {
			return fmt.Errorf("refusing Qdrant alias switch: alias %q currently maps to %q, expected %q", a.alias, current, previous)
		}
	} else if exists {
		return fmt.Errorf("refusing Qdrant alias creation: alias %q already maps to %q", a.alias, current)
	}
	actions := []any{}
	if hasPrevious {
		actions = append(actions, map[string]any{"delete_alias": map[string]string{"alias_name": a.alias}})
	}
	actions = append(actions, map[string]any{"create_alias": map[string]string{"collection_name": target, "alias_name": a.alias}})
	if err := a.request(ctx, http.MethodPost, "/collections/aliases", map[string]any{"actions": actions}, http.StatusOK); err != nil {
		return err
	}
	current, exists, err = a.aliasTarget(ctx)
	if err != nil {
		return err
	}
	if !exists || current != target {
		return fmt.Errorf("Qdrant alias switch outcome is ambiguous: alias %q maps to %q, expected %q", a.alias, current, target)
	}
	return nil
}
func (a *QdrantFleetAdapter) request(ctx context.Context, method, path string, body any, statuses ...int) error {
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("api-key", a.apiKey)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	for _, want := range statuses {
		if resp.StatusCode == want {
			return nil
		}
	}
	return fmt.Errorf("Qdrant %s %s returned HTTP %d", method, path, resp.StatusCode)
}
func (a *QdrantFleetAdapter) read(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+path, nil)
	if err != nil {
		return err
	}
	if a.apiKey != "" {
		req.Header.Set("api-key", a.apiKey)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Qdrant GET %s returned HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
