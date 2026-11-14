package aisearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// FleetSemanticEmbedder carries the complete verified recipe and distinct
// query/document encoders used by one active generation.
type FleetSemanticEmbedder interface {
	Embedder
	Recipe() EmbeddingRecipe
	EmbedQuery(context.Context, string) ([]float64, error)
	EmbedDocument(context.Context, string) ([]float64, error)
}

type FleetSemanticOperation string

const (
	FleetSemanticQueryOperation  FleetSemanticOperation = "query"
	FleetSemanticUpsertOperation FleetSemanticOperation = "upsert"
	FleetSemanticDeleteOperation FleetSemanticOperation = "delete"
)

type FleetSemanticAttribution struct {
	CallerID              string `json:"caller_id"`
	RightsEvidenceURI     string `json:"rights_evidence_uri"`
	RightsEvidenceSHA256  string `json:"rights_evidence_sha256"`
	QualityEvidenceURI    string `json:"quality_evidence_uri"`
	QualityEvidenceSHA256 string `json:"quality_evidence_sha256"`
}

type FleetSemanticQueryRequest struct {
	OperationID string                   `json:"operation_id"`
	TenantID    string                   `json:"tenant_id"`
	ArmID       string                   `json:"arm_id"`
	RecipeID    string                   `json:"recipe_id"`
	Text        string                   `json:"text"`
	Limit       int                      `json:"limit"`
	Attribution FleetSemanticAttribution `json:"attribution"`
}

type FleetSemanticDocument struct {
	Text    string         `json:"text"`
	Payload map[string]any `json:"payload,omitempty"`
}

type FleetSemanticUpsertRequest struct {
	OperationID    string                   `json:"operation_id"`
	TenantID       string                   `json:"tenant_id"`
	ArmID          string                   `json:"arm_id"`
	RecipeID       string                   `json:"recipe_id"`
	SourceID       string                   `json:"source_id"`
	SourceSequence int64                    `json:"source_sequence"`
	Document       FleetSemanticDocument    `json:"document"`
	Attribution    FleetSemanticAttribution `json:"attribution"`
}

type FleetSemanticDeleteSource struct {
	SourceID       string `json:"source_id"`
	SourceSequence int64  `json:"source_sequence"`
}

type FleetSemanticDeleteRequest struct {
	OperationID string                      `json:"operation_id"`
	TenantID    string                      `json:"tenant_id"`
	ArmID       string                      `json:"arm_id"`
	RecipeID    string                      `json:"recipe_id"`
	Sources     []FleetSemanticDeleteSource `json:"sources"`
	Attribution FleetSemanticAttribution    `json:"attribution"`
}

type FleetSemanticOperationPlan struct {
	Operation         FleetSemanticOperation      `json:"operation"`
	OperationID       string                      `json:"operation_id"`
	OwnerID           string                      `json:"owner_id"`
	Alias             string                      `json:"alias"`
	Collection        string                      `json:"collection"`
	ArmID             string                      `json:"arm_id"`
	RecipeID          string                      `json:"recipe_id"`
	RuntimeRole       string                      `json:"runtime_role"`
	TenantID          string                      `json:"tenant_id"`
	GenerationVersion int64                       `json:"generation_version"`
	Dimensions        int                         `json:"dimensions"`
	SourceCount       int                         `json:"source_count"`
	PointCount        int                         `json:"point_count"`
	RequestBytes      int64                       `json:"request_bytes"`
	RequestSHA256     string                      `json:"request_sha256"`
	SourceSequence    int64                       `json:"source_sequence,omitempty"`
	SourceVersions    []FleetSemanticDeleteSource `json:"source_versions,omitempty"`
	Attribution       FleetSemanticAttribution    `json:"attribution"`
}

// FleetSemanticAdmissionRequest is assembled from the cataloged active
// generation and a trusted peak estimator. No target, role, generation,
// dimensions, attribution, or peak value is accepted from the caller.
type FleetSemanticAdmissionRequest struct {
	Plan      FleetSemanticOperationPlan `json:"plan"`
	PeakBytes int64                      `json:"peak_bytes"`
}

// FleetSemanticAdmissionReceipt must be a durable, short-lived owner receipt
// bound to the request exactly. The receipt is returned with each operation so
// the caller can preserve its rights and quality trail without unbounded local
// receipt-table growth.
type FleetSemanticAdmissionReceipt struct {
	Request    FleetSemanticAdmissionRequest `json:"request"`
	ReceiptID  string                        `json:"receipt_id"`
	ObservedAt time.Time                     `json:"observed_at"`
	ExpiresAt  time.Time                     `json:"expires_at"`
}

type FleetSemanticOperationState string

const (
	FleetSemanticOperationStarted   FleetSemanticOperationState = "started"
	FleetSemanticOperationCompleted FleetSemanticOperationState = "completed"
)

type FleetSemanticOperationRecord struct {
	OwnerID         string                        `json:"owner_id"`
	EndpointID      string                        `json:"endpoint_id"`
	Alias           string                        `json:"alias"`
	OperationID     string                        `json:"operation_id"`
	Operation       FleetSemanticOperation        `json:"operation"`
	ArmID           string                        `json:"arm_id"`
	Collection      string                        `json:"collection"`
	TenantID        string                        `json:"tenant_id"`
	RequestSHA256   string                        `json:"request_sha256"`
	ExpectedVersion int64                         `json:"expected_version"`
	State           FleetSemanticOperationState   `json:"state"`
	Admission       FleetSemanticAdmissionReceipt `json:"admission"`
	SourceVersions  []FleetSemanticSourceVersion  `json:"source_versions,omitempty"`
	ResultJSON      []byte                        `json:"result_json,omitempty"`
	UpdatedAt       time.Time                     `json:"updated_at"`
}

// FleetSemanticOperationLedger durably binds an owner operation ID to the
// exact request and stores its completed result for restart-safe replay.
// Started records remain until recovery proves the operation effect and
// records the catalog revision exactly once.
type FleetSemanticOperationLedger interface {
	LoadFleetSemanticOperation(context.Context, FleetSemanticOperationRecord) (FleetSemanticOperationRecord, bool, error)
	BeginFleetSemanticOperation(context.Context, FleetSemanticOperationRecord) (FleetSemanticOperationRecord, error)
	CompleteFleetSemanticOperation(context.Context, FleetSemanticOperationRecord, []byte) error
	CheckFleetSemanticSourceSequences(context.Context, []FleetSemanticSourceVersion) error
	ReserveFleetSemanticSourceSequences(context.Context, []FleetSemanticSourceVersion) error
	CompleteFleetSemanticSourceSequences(context.Context, FleetSemanticOperationRecord, []FleetSemanticSourceVersion) error
}

type FleetSemanticSourceVersion struct {
	OwnerID        string                      `json:"owner_id"`
	EndpointID     string                      `json:"endpoint_id"`
	Alias          string                      `json:"alias"`
	TenantID       string                      `json:"tenant_id"`
	SourceID       string                      `json:"source_id"`
	SourceSequence int64                       `json:"source_sequence"`
	OperationID    string                      `json:"operation_id"`
	RequestSHA256  string                      `json:"request_sha256"`
	State          FleetSemanticOperationState `json:"state,omitempty"`
}

type FleetSemanticPeakEstimator interface {
	EstimateFleetSemanticPeakBytes(context.Context, FleetSemanticOperationPlan) (int64, error)
}

type FleetSemanticAdmissionGate interface {
	// Implementations must durably bind (OwnerID, OperationID) to
	// RequestSHA256 and reject reuse with a different digest. Durable completed
	// result replay is owned by FleetSemanticOperationLedger.
	AdmitQdrantFleetSemanticOperation(context.Context, FleetSemanticAdmissionRequest) (FleetSemanticAdmissionReceipt, error)
}

type FleetSemanticQueryResult struct {
	Generation FleetGeneration               `json:"generation"`
	Admission  FleetSemanticAdmissionReceipt `json:"admission"`
	Results    []SearchResult                `json:"results"`
}

type FleetSemanticOperationResult struct {
	Generation FleetGeneration               `json:"generation"`
	Admission  FleetSemanticAdmissionReceipt `json:"admission"`
}

// FleetSemanticStore operates only on an exact cataloged active generation.
// The adapter identity, verified recipe encoder, runtime role, estimator and
// admission gate are injected at construction, not supplied per request.
// The durable operation ledger replays an exact completed request without
// another vector effect or catalog version increment. Source sequence state
// prevents older writes and tombstones from overwriting newer source events.
type FleetSemanticStore interface {
	QueryActive(context.Context, FleetSemanticQueryRequest) (FleetSemanticQueryResult, error)
	UpsertActive(context.Context, FleetSemanticUpsertRequest) (FleetSemanticOperationResult, error)
	DeleteActive(context.Context, FleetSemanticDeleteRequest) (FleetSemanticOperationResult, error)
}

type FleetSemanticBounds struct {
	MaxQueryBytes    int
	MaxQueryResults  int
	MaxUpsertPoints  int
	MaxSourceIDs     int
	MaxDocumentBytes int
	MaxPayloadBytes  int
	UpsertBatchSize  int
}

func (b FleetSemanticBounds) normalized() (FleetSemanticBounds, error) {
	defaults := FleetSemanticBounds{MaxQueryBytes: 16 << 10, MaxQueryResults: 100, MaxUpsertPoints: 128, MaxSourceIDs: 256, MaxDocumentBytes: 1 << 20, MaxPayloadBytes: 64 << 10, UpsertBatchSize: 64}
	if b == (FleetSemanticBounds{}) {
		return defaults, nil
	}
	if b.MaxQueryBytes <= 0 || b.MaxQueryBytes > defaults.MaxQueryBytes || b.MaxQueryResults <= 0 || b.MaxQueryResults > defaults.MaxQueryResults || b.MaxUpsertPoints <= 0 || b.MaxUpsertPoints > defaults.MaxUpsertPoints || b.MaxSourceIDs <= 0 || b.MaxSourceIDs > defaults.MaxSourceIDs || b.MaxDocumentBytes <= 0 || b.MaxDocumentBytes > defaults.MaxDocumentBytes || b.MaxPayloadBytes <= 0 || b.MaxPayloadBytes > defaults.MaxPayloadBytes || b.UpsertBatchSize <= 0 || b.UpsertBatchSize > defaults.UpsertBatchSize {
		return FleetSemanticBounds{}, errors.New("fleet semantic bounds must be positive and no larger than the governed defaults")
	}
	return b, nil
}

type FleetSemanticOptions struct {
	Embedder    FleetSemanticEmbedder
	RuntimeRole string
	Estimator   FleetSemanticPeakEstimator
	Admission   FleetSemanticAdmissionGate
	Bounds      FleetSemanticBounds
}

type QdrantFleetSemanticStore struct {
	adapter     *QdrantFleetAdapter
	ledger      FleetSemanticOperationLedger
	embedder    FleetSemanticEmbedder
	recipe      EmbeddingRecipe
	runtimeRole string
	estimator   FleetSemanticPeakEstimator
	admission   FleetSemanticAdmissionGate
	bounds      FleetSemanticBounds
}

var _ FleetSemanticStore = (*QdrantFleetSemanticStore)(nil)

func (s *QdrantFleetSemanticStore) FleetIdentity() (ownerID, alias string) {
	if s == nil || s.adapter == nil {
		return "", ""
	}
	return s.adapter.FleetIdentity()
}

func (a *QdrantFleetAdapter) NewSemanticStore(options FleetSemanticOptions) (*QdrantFleetSemanticStore, error) {
	if a == nil || a.catalog == nil || a.aliasBoundary == nil {
		return nil, ErrFleetAliasBoundaryRequired
	}
	if options.Embedder == nil || options.Estimator == nil || options.Admission == nil || strings.TrimSpace(options.RuntimeRole) == "" {
		return nil, errors.New("fleet semantic store requires a verified encoder, runtime role, trusted peak estimator and admission gate")
	}
	ledger, ok := a.catalog.(FleetSemanticOperationLedger)
	if !ok || ledger == nil {
		return nil, errors.New("fleet semantic store requires an owner-bound durable operation ledger")
	}
	recipe := options.Embedder.Recipe()
	if err := recipe.Verify(); err != nil {
		return nil, fmt.Errorf("verify fleet semantic encoder recipe: %w", err)
	}
	bounds, err := options.Bounds.normalized()
	if err != nil {
		return nil, err
	}
	return &QdrantFleetSemanticStore{adapter: a, ledger: ledger, embedder: options.Embedder, recipe: recipe, runtimeRole: strings.TrimSpace(options.RuntimeRole), estimator: options.Estimator, admission: options.Admission, bounds: bounds}, nil
}

func (s *QdrantFleetSemanticStore) QueryActive(ctx context.Context, request FleetSemanticQueryRequest) (FleetSemanticQueryResult, error) {
	if err := s.validateOperation(request.OperationID, request.TenantID, request.ArmID, request.RecipeID, request.Attribution); err != nil {
		return FleetSemanticQueryResult{}, err
	}
	if strings.TrimSpace(request.Text) == "" || len(request.Text) > s.bounds.MaxQueryBytes || request.Limit < 1 || request.Limit > s.bounds.MaxQueryResults {
		return FleetSemanticQueryResult{}, errors.New("fleet semantic query text or limit is outside its bounded policy")
	}
	requestSHA256, err := fleetSemanticRequestDigest(request)
	if err != nil {
		return FleetSemanticQueryResult{}, err
	}
	identity := s.operationIdentity(FleetSemanticQueryOperation, request.OperationID, request.ArmID, request.TenantID, requestSHA256)
	if completed, replay, err := s.replayCompletedOperation(ctx, identity); err != nil {
		return FleetSemanticQueryResult{}, err
	} else if replay {
		var result FleetSemanticQueryResult
		if err := json.Unmarshal(completed.ResultJSON, &result); err != nil {
			return FleetSemanticQueryResult{}, fmt.Errorf("decode completed fleet semantic query result: %w", err)
		}
		return result, nil
	}
	initial, err := s.resolveActive(ctx, request.ArmID, request.RecipeID)
	if err != nil {
		return FleetSemanticQueryResult{}, err
	}
	var result FleetSemanticQueryResult
	retainLock := false
	err = s.withActiveOperation(ctx, initial, FleetSemanticQueryOperation, request.OperationID, &retainLock, func(record FleetGeneration) error {
		completed, exists, err := s.ledger.LoadFleetSemanticOperation(ctx, identity)
		if err != nil {
			return err
		}
		if exists && completed.State == FleetSemanticOperationCompleted {
			return json.Unmarshal(completed.ResultJSON, &result)
		}
		plan := s.operationPlan(FleetSemanticQueryOperation, request.OperationID, request.TenantID, requestSHA256, record, 0, 0, int64(len(request.Text)), request.Attribution)
		receipt, err := s.admit(ctx, plan)
		if err != nil {
			return err
		}
		operationRecord, err := s.beginSemanticOperation(ctx, identity, record, receipt, nil)
		if err != nil {
			return err
		}
		if operationRecord.State == FleetSemanticOperationCompleted {
			return json.Unmarshal(operationRecord.ResultJSON, &result)
		}
		retainLock = true
		vector, err := s.embedder.EmbedQuery(ctx, request.Text)
		if err != nil {
			return fmt.Errorf("embed fleet semantic query: %w", err)
		}
		dense, err := checkedFleetVector(vector, record.Dimensions)
		if err != nil {
			return err
		}
		store := NewVectorStoreWithClient(s.adapter.baseURL, s.adapter.apiKey, record.Collection, s.adapter.client)
		filter := &QueryFilter{Must: []FieldMatch{{Key: "embedding_recipe_id", Value: record.RecipeID}, {Key: "fleet_owner_id", Value: record.OwnerID}, {Key: "fleet_alias", Value: record.Alias}, {Key: "fleet_arm_id", Value: record.ArmID}, {Key: "fleet_tenant_id", Value: request.TenantID}}}
		hits, err := store.Query(ctx, HybridQuery{Dense: dense, Filter: filter, Limit: request.Limit})
		if err != nil {
			return err
		}
		for index := range hits {
			if sourceID, ok := hits[index].Payload[sourceIDKey].(string); ok {
				hits[index].SourceID = sourceID
			}
		}
		result = FleetSemanticQueryResult{Generation: record, Admission: receipt, Results: hits}
		if err := s.completeSemanticOperation(ctx, operationRecord, result); err != nil {
			return err
		}
		retainLock = false
		return nil
	})
	if err != nil {
		return FleetSemanticQueryResult{}, err
	}
	return result, nil
}

func (s *QdrantFleetSemanticStore) UpsertActive(ctx context.Context, request FleetSemanticUpsertRequest) (FleetSemanticOperationResult, error) {
	if err := s.validateOperation(request.OperationID, request.TenantID, request.ArmID, request.RecipeID, request.Attribution); err != nil {
		return FleetSemanticOperationResult{}, err
	}
	request.SourceID = strings.TrimSpace(request.SourceID)
	if request.SourceID == "" || len(request.SourceID) > 512 || request.SourceSequence <= 0 {
		return FleetSemanticOperationResult{}, errors.New("fleet semantic upsert requires a stable source ID and positive owner sequence")
	}
	if strings.TrimSpace(request.Document.Text) == "" || len(request.Document.Text) > s.bounds.MaxDocumentBytes {
		return FleetSemanticOperationResult{}, errors.New("fleet semantic document text is empty or exceeds its bound")
	}
	payloadJSON, err := json.Marshal(request.Document.Payload)
	if err != nil || len(payloadJSON) > s.bounds.MaxPayloadBytes {
		return FleetSemanticOperationResult{}, errors.New("fleet semantic document payload is invalid or exceeds its bound")
	}
	if hasReservedFleetPayloadKey(request.Document.Payload) {
		return FleetSemanticOperationResult{}, errors.New("fleet semantic document payload cannot override owner, source, recipe, arm or attribution fields")
	}
	if rawTenant, present := request.Document.Payload["user_id"]; present && rawTenant != nil {
		if payloadTenant, ok := rawTenant.(string); !ok || payloadTenant != request.TenantID {
			return FleetSemanticOperationResult{}, errors.New("fleet semantic document user_id must match its admitted tenant identity")
		}
	}
	requestBytes := int64(len(request.SourceID) + len(request.Document.Text) + len(payloadJSON))
	requestSHA256, err := fleetSemanticRequestDigest(request)
	if err != nil {
		return FleetSemanticOperationResult{}, err
	}
	identity := s.operationIdentity(FleetSemanticUpsertOperation, request.OperationID, request.ArmID, request.TenantID, requestSHA256)
	sourceVersions := []FleetSemanticSourceVersion{s.semanticSourceVersion(request.TenantID, request.SourceID, request.SourceSequence, request.OperationID, requestSHA256)}
	if completed, replay, err := s.replayCompletedOperation(ctx, identity); err != nil {
		return FleetSemanticOperationResult{}, err
	} else if replay {
		var result FleetSemanticOperationResult
		if err := json.Unmarshal(completed.ResultJSON, &result); err != nil {
			return FleetSemanticOperationResult{}, fmt.Errorf("decode completed fleet semantic upsert result: %w", err)
		}
		return result, nil
	}
	initial, err := s.resolveActive(ctx, request.ArmID, request.RecipeID)
	if err != nil {
		return FleetSemanticOperationResult{}, err
	}
	var result FleetSemanticOperationResult
	retainLock := false
	err = s.withActiveOperation(ctx, initial, FleetSemanticUpsertOperation, request.OperationID, &retainLock, func(record FleetGeneration) error {
		completed, exists, err := s.ledger.LoadFleetSemanticOperation(ctx, identity)
		if err != nil {
			return err
		}
		if exists && completed.State == FleetSemanticOperationCompleted {
			return json.Unmarshal(completed.ResultJSON, &result)
		}
		prior, found, err := s.ledger.LoadFleetSemanticOperation(ctx, identity)
		if err != nil {
			return err
		}
		if found && !sameSemanticRequestIdentity(prior, identity) {
			return ErrFleetSemanticOperationConflict
		}
		if found && prior.State == FleetSemanticOperationStarted {
			if record.LastSemanticOperationID == prior.OperationID && record.LastSemanticRequestSHA256 == prior.RequestSHA256 && record.Version == prior.ExpectedVersion+1 {
				result = FleetSemanticOperationResult{Generation: record, Admission: prior.Admission}
				retainLock = true
				if err := s.completeSemanticOperation(ctx, prior, result); err != nil {
					return err
				}
				retainLock = false
				return nil
			}
			if record.Version != prior.ExpectedVersion {
				return ErrFleetSemanticOperationUnresolved
			}
		}
		if err := s.ledger.CheckFleetSemanticSourceSequences(ctx, sourceVersions); err != nil {
			return err
		}
		plan := s.operationPlan(FleetSemanticUpsertOperation, request.OperationID, request.TenantID, requestSHA256, record, 1, 1, requestBytes, request.Attribution)
		plan.SourceSequence = request.SourceSequence
		receipt, err := s.admit(ctx, plan)
		if err != nil {
			return err
		}
		operationRecord, err := s.beginSemanticOperation(ctx, identity, record, receipt, sourceVersions)
		if err != nil {
			return err
		}
		if operationRecord.State == FleetSemanticOperationCompleted {
			return json.Unmarshal(operationRecord.ResultJSON, &result)
		}
		retainLock = true
		if err := s.ledger.ReserveFleetSemanticSourceSequences(ctx, sourceVersions); err != nil {
			return err
		}
		pointID := FleetSemanticPointID(record.OwnerID, record.Alias, record.ArmID, request.TenantID, request.RecipeID, request.SourceID)
		alreadyApplied, err := s.semanticPointHasOperation(ctx, record, pointID, request.TenantID, request.SourceID, request.OperationID, requestSHA256)
		if err != nil {
			return err
		}
		if !alreadyApplied {
			vector, err := s.embedder.EmbedDocument(ctx, request.Document.Text)
			if err != nil {
				return fmt.Errorf("embed fleet semantic document: %w", err)
			}
			dense, err := checkedFleetVector(vector, record.Dimensions)
			if err != nil {
				return err
			}
			payload := make(map[string]any, len(request.Document.Payload)+10)
			for key, value := range request.Document.Payload {
				payload[key] = value
			}
			payload["source_id"] = request.SourceID
			payload["embedding_recipe_id"] = request.RecipeID
			payload["fleet_owner_id"] = record.OwnerID
			payload["fleet_alias"] = record.Alias
			payload["fleet_arm_id"] = record.ArmID
			payload["fleet_tenant_id"] = request.TenantID
			payload["fleet_runtime_role"] = s.runtimeRole
			payload["fleet_caller_id"] = request.Attribution.CallerID
			payload["fleet_rights_evidence_sha256"] = request.Attribution.RightsEvidenceSHA256
			payload["fleet_quality_evidence_sha256"] = request.Attribution.QualityEvidenceSHA256
			payload["fleet_semantic_operation_id"] = request.OperationID
			payload["fleet_semantic_request_sha256"] = requestSHA256
			points := []Point{{ID: pointID, Dense: dense, Payload: payload}}
			store := NewVectorStoreWithClient(s.adapter.baseURL, s.adapter.apiKey, record.Collection, s.adapter.client)
			batch, ok := store.(BatchVectorStore)
			if !ok {
				return errors.New("fleet semantic Qdrant store does not support bounded batch upsert")
			}
			if err := batch.UpsertBatch(ctx, points, s.bounds.UpsertBatchSize); err != nil {
				return err
			}
		}
		record, err = s.bumpActiveGeneration(ctx, record, operationRecord)
		if err != nil {
			return fmt.Errorf("semantic upsert vector effect applied or was reconciled but generation CAS failed: %w", err)
		}
		result = FleetSemanticOperationResult{Generation: record, Admission: receipt}
		if err := s.completeSemanticOperation(ctx, operationRecord, result); err != nil {
			return err
		}
		retainLock = false
		return nil
	})
	if err != nil {
		return FleetSemanticOperationResult{}, err
	}
	return result, nil
}

func (s *QdrantFleetSemanticStore) DeleteActive(ctx context.Context, request FleetSemanticDeleteRequest) (FleetSemanticOperationResult, error) {
	if err := s.validateOperation(request.OperationID, request.TenantID, request.ArmID, request.RecipeID, request.Attribution); err != nil {
		return FleetSemanticOperationResult{}, err
	}
	if len(request.Sources) == 0 || len(request.Sources) > s.bounds.MaxSourceIDs {
		return FleetSemanticOperationResult{}, errors.New("fleet semantic delete requires a bounded nonempty source-ID list")
	}
	sources := make([]FleetSemanticDeleteSource, len(request.Sources))
	sourceIDs := make([]string, len(request.Sources))
	requestBytes := int64(0)
	seen := make(map[string]struct{}, len(request.Sources))
	for i, value := range request.Sources {
		value.SourceID = strings.TrimSpace(value.SourceID)
		if value.SourceID == "" || len(value.SourceID) > 512 || value.SourceSequence <= 0 {
			return FleetSemanticOperationResult{}, errors.New("fleet semantic delete source requires a bounded ID and positive owner sequence")
		}
		if _, ok := seen[value.SourceID]; ok {
			return FleetSemanticOperationResult{}, errors.New("fleet semantic delete source IDs must be unique")
		}
		seen[value.SourceID] = struct{}{}
		sources[i] = value
		requestBytes += int64(len(value.SourceID))
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].SourceID < sources[j].SourceID })
	for i, source := range sources {
		sourceIDs[i] = source.SourceID
	}
	request.Sources = sources
	requestSHA256, err := fleetSemanticRequestDigest(request)
	if err != nil {
		return FleetSemanticOperationResult{}, err
	}
	identity := s.operationIdentity(FleetSemanticDeleteOperation, request.OperationID, request.ArmID, request.TenantID, requestSHA256)
	sourceVersions := make([]FleetSemanticSourceVersion, len(sources))
	for i, source := range sources {
		sourceVersions[i] = s.semanticSourceVersion(request.TenantID, source.SourceID, source.SourceSequence, request.OperationID, requestSHA256)
	}
	if completed, replay, err := s.replayCompletedOperation(ctx, identity); err != nil {
		return FleetSemanticOperationResult{}, err
	} else if replay {
		var result FleetSemanticOperationResult
		if err := json.Unmarshal(completed.ResultJSON, &result); err != nil {
			return FleetSemanticOperationResult{}, fmt.Errorf("decode completed fleet semantic delete result: %w", err)
		}
		return result, nil
	}
	initial, err := s.resolveActive(ctx, request.ArmID, request.RecipeID)
	if err != nil {
		return FleetSemanticOperationResult{}, err
	}
	var result FleetSemanticOperationResult
	retainLock := false
	err = s.withActiveOperation(ctx, initial, FleetSemanticDeleteOperation, request.OperationID, &retainLock, func(record FleetGeneration) error {
		completed, exists, err := s.ledger.LoadFleetSemanticOperation(ctx, identity)
		if err != nil {
			return err
		}
		if exists && completed.State == FleetSemanticOperationCompleted {
			return json.Unmarshal(completed.ResultJSON, &result)
		}
		prior, found, err := s.ledger.LoadFleetSemanticOperation(ctx, identity)
		if err != nil {
			return err
		}
		if found && !sameSemanticRequestIdentity(prior, identity) {
			return ErrFleetSemanticOperationConflict
		}
		if found && prior.State == FleetSemanticOperationStarted {
			if record.LastSemanticOperationID == prior.OperationID && record.LastSemanticRequestSHA256 == prior.RequestSHA256 && record.Version == prior.ExpectedVersion+1 {
				result = FleetSemanticOperationResult{Generation: record, Admission: prior.Admission}
				retainLock = true
				if err := s.completeSemanticOperation(ctx, prior, result); err != nil {
					return err
				}
				retainLock = false
				return nil
			}
			if record.Version != prior.ExpectedVersion {
				return ErrFleetSemanticOperationUnresolved
			}
		}
		if err := s.ledger.CheckFleetSemanticSourceSequences(ctx, sourceVersions); err != nil {
			return err
		}
		plan := s.operationPlan(FleetSemanticDeleteOperation, request.OperationID, request.TenantID, requestSHA256, record, len(sourceIDs), 0, requestBytes, request.Attribution)
		plan.SourceVersions = append([]FleetSemanticDeleteSource(nil), sources...)
		receipt, err := s.admit(ctx, plan)
		if err != nil {
			return err
		}
		operationRecord, err := s.beginSemanticOperation(ctx, identity, record, receipt, sourceVersions)
		if err != nil {
			return err
		}
		if operationRecord.State == FleetSemanticOperationCompleted {
			return json.Unmarshal(operationRecord.ResultJSON, &result)
		}
		retainLock = true
		if err := s.ledger.ReserveFleetSemanticSourceSequences(ctx, sourceVersions); err != nil {
			return err
		}
		count, err := s.adapter.countSemanticSources(ctx, record, request.TenantID, sourceIDs)
		if err != nil {
			return err
		}
		if count > 0 {
			if err := s.adapter.deleteSemanticSources(ctx, record, request.TenantID, sourceIDs); err != nil {
				return err
			}
		}
		record, err = s.bumpActiveGeneration(ctx, record, operationRecord)
		if err != nil {
			return fmt.Errorf("semantic delete effect applied or was reconciled but generation CAS failed: %w", err)
		}
		result = FleetSemanticOperationResult{Generation: record, Admission: receipt}
		if err := s.completeSemanticOperation(ctx, operationRecord, result); err != nil {
			return err
		}
		retainLock = false
		return nil
	})
	if err != nil {
		return FleetSemanticOperationResult{}, err
	}
	return result, nil
}

func (s *QdrantFleetSemanticStore) resolveActive(ctx context.Context, armID, recipeID string) (FleetGeneration, error) {
	armID = strings.TrimSpace(armID)
	if armID == "" || recipeID != s.recipe.ID() {
		return FleetGeneration{}, errors.New("fleet semantic arm or immutable recipe identity mismatch")
	}
	record, exists, err := s.adapter.catalog.LoadFleetGeneration(ctx, armID)
	if err != nil {
		return FleetGeneration{}, err
	}
	if !exists || record.State != FleetActive || record.OwnerID != s.adapter.ownerID || record.Alias != s.adapter.alias || record.RecipeID != recipeID || record.Dimensions != mustRecipeDimensions(s.recipe) {
		return FleetGeneration{}, ErrFleetStateConflict
	}
	active, exists, err := s.adapter.aliasTarget(ctx)
	if err != nil {
		return FleetGeneration{}, err
	}
	if !exists || active != record.Collection {
		return FleetGeneration{}, errors.New("Qdrant alias does not point to the cataloged active recipe generation")
	}
	return record, nil
}

func (s *QdrantFleetSemanticStore) withActiveOperation(ctx context.Context, initial FleetGeneration, operation FleetSemanticOperation, operationID string, retainCatalogOperation *bool, run func(FleetGeneration) error) (retErr error) {
	kind := map[FleetSemanticOperation]FleetOperation{FleetSemanticQueryOperation: FleetSemanticQuery, FleetSemanticUpsertOperation: FleetSemanticUpsert, FleetSemanticDeleteOperation: FleetSemanticDelete}[operation]
	if kind == "" {
		return errors.New("unknown fleet semantic operation")
	}
	op, fence, err := s.adapter.acquireAliasOperationWithID(ctx, kind, initial.ArmID, initial.Collection, operationID)
	if err != nil {
		return err
	}
	catalogHeld := false
	defer func() {
		if catalogHeld && (retainCatalogOperation == nil || !*retainCatalogOperation) {
			if err := s.adapter.releaseAliasOperation(context.WithoutCancel(ctx), op); err != nil && retErr == nil {
				retErr = err
			}
		}
		if err := fence.Release(context.WithoutCancel(ctx)); err != nil && retErr == nil {
			retErr = err
		}
	}()
	if err := s.adapter.catalog.AcquireFleetAliasOperation(ctx, op); err != nil {
		return fmt.Errorf("Qdrant fleet has an unfinished owner+alias operation: %w", err)
	}
	catalogHeld = true
	current, exists, err := s.adapter.catalog.LoadFleetGeneration(ctx, initial.ArmID)
	if err != nil {
		return err
	}
	if !exists || current.Version != initial.Version || current.State != FleetActive || current.Collection != initial.Collection || current.RecipeID != s.recipe.ID() {
		return ErrFleetStateConflict
	}
	active, exists, err := s.adapter.aliasTarget(ctx)
	if err != nil {
		return err
	}
	if !exists || active != current.Collection {
		return errors.New("Qdrant alias changed before admitted semantic operation")
	}
	return run(current)
}

func (s *QdrantFleetSemanticStore) operationPlan(operation FleetSemanticOperation, operationID, tenantID, requestSHA256 string, record FleetGeneration, sourceCount, pointCount int, requestBytes int64, attribution FleetSemanticAttribution) FleetSemanticOperationPlan {
	return FleetSemanticOperationPlan{Operation: operation, OperationID: operationID, OwnerID: record.OwnerID, Alias: record.Alias, Collection: record.Collection, ArmID: record.ArmID, RecipeID: record.RecipeID, RuntimeRole: s.runtimeRole, TenantID: tenantID, GenerationVersion: record.Version, Dimensions: record.Dimensions, SourceCount: sourceCount, PointCount: pointCount, RequestBytes: requestBytes, RequestSHA256: requestSHA256, Attribution: attribution}
}

func (s *QdrantFleetSemanticStore) operationIdentity(operation FleetSemanticOperation, operationID, armID, tenantID, requestSHA256 string) FleetSemanticOperationRecord {
	return FleetSemanticOperationRecord{OwnerID: s.adapter.ownerID, EndpointID: s.adapter.baseURL, Alias: s.adapter.alias, OperationID: operationID, Operation: operation, ArmID: armID, TenantID: tenantID, RequestSHA256: requestSHA256}
}

func (s *QdrantFleetSemanticStore) semanticSourceVersion(tenantID, sourceID string, sequence int64, operationID, requestSHA256 string) FleetSemanticSourceVersion {
	return FleetSemanticSourceVersion{OwnerID: s.adapter.ownerID, EndpointID: s.adapter.baseURL, Alias: s.adapter.alias, TenantID: tenantID, SourceID: sourceID, SourceSequence: sequence, OperationID: operationID, RequestSHA256: requestSHA256}
}

func sameSemanticRequestIdentity(left, right FleetSemanticOperationRecord) bool {
	return sameFleetSemanticOperationIdentity(left, right) && left.Operation == right.Operation && left.ArmID == right.ArmID && left.TenantID == right.TenantID && left.RequestSHA256 == right.RequestSHA256
}

func (s *QdrantFleetSemanticStore) beginSemanticOperation(ctx context.Context, identity FleetSemanticOperationRecord, generation FleetGeneration, admission FleetSemanticAdmissionReceipt, sourceVersions []FleetSemanticSourceVersion) (FleetSemanticOperationRecord, error) {
	identity.Collection = generation.Collection
	identity.ExpectedVersion = generation.Version
	identity.Admission = admission
	identity.SourceVersions = append([]FleetSemanticSourceVersion(nil), sourceVersions...)
	identity.State = FleetSemanticOperationStarted
	identity.UpdatedAt = s.adapter.now().UTC()
	return s.ledger.BeginFleetSemanticOperation(ctx, identity)
}

func (s *QdrantFleetSemanticStore) completeSemanticOperation(ctx context.Context, operation FleetSemanticOperationRecord, result any) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	operation.State = FleetSemanticOperationCompleted
	operation.ResultJSON = encoded
	operation.UpdatedAt = s.adapter.now().UTC()
	if len(operation.SourceVersions) > 0 {
		if err := s.ledger.CompleteFleetSemanticSourceSequences(ctx, operation, operation.SourceVersions); err != nil {
			return err
		}
	}
	return s.ledger.CompleteFleetSemanticOperation(ctx, operation, encoded)
}

func (s *QdrantFleetSemanticStore) replayCompletedOperation(ctx context.Context, operation FleetSemanticOperationRecord) (FleetSemanticOperationRecord, bool, error) {
	stored, found, err := s.ledger.LoadFleetSemanticOperation(ctx, operation)
	if err != nil || !found {
		return stored, false, err
	}
	if !sameSemanticRequestIdentity(stored, operation) {
		return FleetSemanticOperationRecord{}, false, ErrFleetSemanticOperationConflict
	}
	if stored.State != FleetSemanticOperationCompleted {
		return stored, false, nil
	}
	kind, err := semanticFleetOperation(stored.Operation)
	if err != nil {
		return FleetSemanticOperationRecord{}, false, err
	}
	op, fence, err := s.adapter.acquireAliasOperationWithID(ctx, kind, stored.ArmID, stored.Collection, stored.OperationID)
	if err != nil {
		return FleetSemanticOperationRecord{}, false, err
	}
	defer func() {
		_ = fence.Release(context.WithoutCancel(ctx))
	}()
	if err := s.adapter.releaseAliasOperation(ctx, op); err != nil {
		return FleetSemanticOperationRecord{}, false, err
	}
	return stored, true, nil
}

func semanticFleetOperation(operation FleetSemanticOperation) (FleetOperation, error) {
	switch operation {
	case FleetSemanticQueryOperation:
		return FleetSemanticQuery, nil
	case FleetSemanticUpsertOperation:
		return FleetSemanticUpsert, nil
	case FleetSemanticDeleteOperation:
		return FleetSemanticDelete, nil
	default:
		return "", errors.New("unknown fleet semantic operation")
	}
}

func fleetSemanticRequestDigest(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("canonicalize fleet semantic request for idempotency: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func (s *QdrantFleetSemanticStore) admit(ctx context.Context, plan FleetSemanticOperationPlan) (FleetSemanticAdmissionReceipt, error) {
	peak, err := s.estimator.EstimateFleetSemanticPeakBytes(ctx, plan)
	if err != nil {
		return FleetSemanticAdmissionReceipt{}, fmt.Errorf("estimate bounded semantic operation peak: %w", err)
	}
	if peak <= 0 {
		return FleetSemanticAdmissionReceipt{}, errors.New("trusted semantic peak estimator must return a positive byte estimate")
	}
	request := FleetSemanticAdmissionRequest{Plan: plan, PeakBytes: peak}
	receipt, err := s.admission.AdmitQdrantFleetSemanticOperation(ctx, request)
	if err != nil {
		return FleetSemanticAdmissionReceipt{}, err
	}
	now := s.adapter.now()
	receiptRequest, marshalErr := json.Marshal(receipt.Request)
	expectedRequest, expectedErr := json.Marshal(request)
	if marshalErr != nil || expectedErr != nil || !bytes.Equal(receiptRequest, expectedRequest) || strings.TrimSpace(receipt.ReceiptID) == "" || receipt.ObservedAt.After(now) || now.Sub(receipt.ObservedAt) > time.Minute || !receipt.ExpiresAt.After(now) || receipt.ExpiresAt.Sub(receipt.ObservedAt) > time.Minute {
		return FleetSemanticAdmissionReceipt{}, ErrFleetAdmissionRequired
	}
	return receipt, nil
}

func (s *QdrantFleetSemanticStore) bumpActiveGeneration(ctx context.Context, record FleetGeneration, operation FleetSemanticOperationRecord) (FleetGeneration, error) {
	if record.LastSemanticOperationID == operation.OperationID && record.LastSemanticRequestSHA256 == operation.RequestSHA256 && record.Version == operation.ExpectedVersion+1 {
		return record, nil
	}
	if record.Version != operation.ExpectedVersion {
		return FleetGeneration{}, ErrFleetSemanticOperationUnresolved
	}
	record.DataVersion++
	record.Version++
	record.LastSemanticOperationID = operation.OperationID
	record.LastSemanticRequestSHA256 = operation.RequestSHA256
	record.UpdatedAt = s.adapter.now().UTC()
	if err := s.adapter.catalog.SaveFleetGenerationCAS(ctx, record, record.Version-1); err != nil {
		return FleetGeneration{}, err
	}
	return record, nil
}

func (a *QdrantFleetAdapter) deleteSemanticSources(ctx context.Context, record FleetGeneration, tenantID string, sourceIDs []string) error {
	return a.request(ctx, http.MethodPost, "/collections/"+url.PathEscape(record.Collection)+"/points/delete?wait=true", map[string]any{"filter": map[string]any{"must": []any{
		map[string]any{"key": sourceIDKey, "match": map[string]any{"any": sourceIDs}},
		map[string]any{"key": "embedding_recipe_id", "match": map[string]any{"value": record.RecipeID}},
		map[string]any{"key": "fleet_owner_id", "match": map[string]any{"value": record.OwnerID}},
		map[string]any{"key": "fleet_alias", "match": map[string]any{"value": record.Alias}},
		map[string]any{"key": "fleet_arm_id", "match": map[string]any{"value": record.ArmID}},
		map[string]any{"key": "fleet_tenant_id", "match": map[string]any{"value": tenantID}},
	}}}, http.StatusOK)
}

func (s *QdrantFleetSemanticStore) semanticPointHasOperation(ctx context.Context, record FleetGeneration, pointID, tenantID, sourceID, operationID, requestSHA256 string) (bool, error) {
	var response struct {
		Result []struct {
			ID      string         `json:"id"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	path := "/collections/" + url.PathEscape(record.Collection) + "/points"
	if err := s.adapter.requestReadJSON(ctx, http.MethodPost, path, map[string]any{"ids": []string{pointID}, "with_payload": true, "with_vector": false}, &response); err != nil {
		return false, err
	}
	if len(response.Result) == 0 {
		return false, nil
	}
	if len(response.Result) != 1 || response.Result[0].ID != pointID {
		return false, ErrFleetSemanticOperationUnresolved
	}
	payload := response.Result[0].Payload
	for key, expected := range map[string]string{"source_id": sourceID, "embedding_recipe_id": record.RecipeID, "fleet_owner_id": record.OwnerID, "fleet_alias": record.Alias, "fleet_arm_id": record.ArmID, "fleet_tenant_id": tenantID} {
		if payload[key] != expected {
			return false, ErrFleetSemanticOperationUnresolved
		}
	}
	return payload["fleet_semantic_operation_id"] == operationID && payload["fleet_semantic_request_sha256"] == requestSHA256, nil
}

func (a *QdrantFleetAdapter) countSemanticSources(ctx context.Context, record FleetGeneration, tenantID string, sourceIDs []string) (uint64, error) {
	var response struct {
		Result struct {
			Count uint64 `json:"count"`
		} `json:"result"`
	}
	if err := a.requestReadJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(record.Collection)+"/points/count", map[string]any{"exact": true, "filter": semanticSourceFilter(record, tenantID, sourceIDs)}, &response); err != nil {
		return 0, err
	}
	return response.Result.Count, nil
}

func semanticSourceFilter(record FleetGeneration, tenantID string, sourceIDs []string) map[string]any {
	return map[string]any{"must": []any{
		map[string]any{"key": sourceIDKey, "match": map[string]any{"any": sourceIDs}},
		map[string]any{"key": "embedding_recipe_id", "match": map[string]any{"value": record.RecipeID}},
		map[string]any{"key": "fleet_owner_id", "match": map[string]any{"value": record.OwnerID}},
		map[string]any{"key": "fleet_alias", "match": map[string]any{"value": record.Alias}},
		map[string]any{"key": "fleet_arm_id", "match": map[string]any{"value": record.ArmID}},
		map[string]any{"key": "fleet_tenant_id", "match": map[string]any{"value": tenantID}},
	}}
}

func (s *QdrantFleetSemanticStore) validateOperation(operationID, tenantID, armID, recipeID string, attribution FleetSemanticAttribution) error {
	if s == nil || s.adapter == nil || s.embedder == nil || s.admission == nil || s.estimator == nil {
		return ErrFleetAdmissionRequired
	}
	if strings.TrimSpace(operationID) == "" || len(operationID) > 256 || strings.TrimSpace(tenantID) == "" || len(tenantID) > 512 || strings.TrimSpace(armID) == "" || recipeID != s.recipe.ID() {
		return errors.New("fleet semantic operation requires idempotency, tenant, arm and exact immutable recipe identities")
	}
	return validateFleetSemanticAttribution(attribution)
}

func validateFleetSemanticAttribution(attribution FleetSemanticAttribution) error {
	if strings.TrimSpace(attribution.CallerID) == "" || strings.TrimSpace(attribution.RightsEvidenceURI) == "" || !validFleetEvidenceDigest(attribution.RightsEvidenceSHA256) || strings.TrimSpace(attribution.QualityEvidenceURI) == "" || !validFleetEvidenceDigest(attribution.QualityEvidenceSHA256) {
		return errors.New("fleet semantic operation requires caller, rights evidence and quality evidence attribution")
	}
	return nil
}

func validFleetEvidenceDigest(value string) bool {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	return err == nil && len(decoded) == sha256.Size
}

func checkedFleetVector(vector []float64, dimensions int) ([]float64, error) {
	if len(vector) != dimensions {
		return nil, fmt.Errorf("fleet semantic encoder returned %d dimensions; recipe requires %d", len(vector), dimensions)
	}
	dense := make([]float64, len(vector))
	for index, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > math.MaxFloat32 {
			return nil, errors.New("fleet semantic encoder returned a value outside finite float32 range")
		}
		dense[index] = float64(float32(value))
	}
	return dense, nil
}

func mustRecipeDimensions(recipe EmbeddingRecipe) int {
	input, _ := recipe.Input()
	return input.Dimensions
}

func hasReservedFleetPayloadKey(payload map[string]any) bool {
	for _, key := range []string{"source_id", "embedding_recipe_id", "fleet_owner_id", "fleet_alias", "fleet_arm_id", "fleet_tenant_id", "fleet_runtime_role", "fleet_caller_id", "fleet_rights_evidence_sha256", "fleet_quality_evidence_sha256", "fleet_semantic_operation_id", "fleet_semantic_request_sha256"} {
		if _, exists := payload[key]; exists {
			return true
		}
	}
	return false
}

// FleetPointID is the migration path's stable recipe/source/chunk identity.
// The tenant-safe semantic API uses FleetSemanticPointID below.
func FleetPointID(recipeID, sourceID string, index int) string {
	sum := sha256.Sum256([]byte(recipeID + "\x00" + sourceID + "\x00" + fmt.Sprint(index)))
	return fleetPointUUID(sum)
}

// FleetSemanticPointID isolates identical source identifiers across tenants,
// arms and aliases. The semantic API uses this identity instead of the legacy
// migration point identity.
func FleetSemanticPointID(ownerID, alias, armID, tenantID, recipeID, sourceID string) string {
	sum := sha256.Sum256([]byte(ownerID + "\x00" + alias + "\x00" + armID + "\x00" + tenantID + "\x00" + recipeID + "\x00" + sourceID))
	return fleetPointUUID(sum)
}

func fleetPointUUID(sum [sha256.Size]byte) string {
	bytes := sum[:16]
	bytes[6] = (bytes[6] & 0x0f) | 0x50
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
