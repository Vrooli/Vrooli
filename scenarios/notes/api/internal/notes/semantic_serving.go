package notes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ActiveSemanticGenerationResolver returns the generation selected by the
// owner catalog. It must not infer a collection or recipe from environment
// defaults.
type ActiveSemanticGenerationResolver interface {
	ResolveActiveSemanticGeneration(context.Context) (FleetGenerationConfig, error)
}

type SemanticSearchHit struct {
	ID      string
	Score   float64
	Payload map[string]any
}

// SemanticAttribution is trusted, operation-level provenance. In particular,
// rights and quality evidence are required inputs; Notes does not invent
// them from a model name or from the fact that a caller is local.
type SemanticAttribution struct {
	CallerID              string
	RightsEvidenceURI     string
	RightsEvidenceSHA256  string
	QualityEvidenceURI    string
	QualityEvidenceSHA256 string
}

type SemanticAttributionResolver interface {
	ResolveSemanticAttribution(context.Context) (SemanticAttribution, error)
}

// SemanticTenantResolver binds a requested/source tenant to the caller's
// authenticated scope. UserID supplied in the HTTP body is never authority.
type SemanticTenantResolver interface {
	ResolveSemanticTenant(context.Context, string) (string, error)
}

type SemanticFleetQuery struct {
	OperationID string
	ArmID       string
	RecipeID    string
	TenantID    string
	Text        string
	Limit       int
	Attribution SemanticAttribution
}

type SemanticFleetDocument struct {
	Text    string
	Payload map[string]any
}

type SemanticFleetUpsert struct {
	OperationID    string
	ArmID          string
	RecipeID       string
	TenantID       string
	SourceID       string
	SourceSequence int64
	Document       SemanticFleetDocument
	Attribution    SemanticAttribution
}

type SemanticFleetDeleteSource struct {
	SourceID       string
	SourceSequence int64
}

type SemanticFleetDelete struct {
	OperationID string
	ArmID       string
	RecipeID    string
	TenantID    string
	Sources     []SemanticFleetDeleteSource
	Attribution SemanticAttribution
}

// SemanticFleetAdmissionReceipt is the Notes-owned projection of the durable
// shared receipt. It retains the exact operation, generation version, request
// digest, tenant, caller and evidence hashes needed to correlate retries and
// audits without retaining document contents a second time.
type SemanticFleetAdmissionReceipt struct {
	Operation         string
	OperationID       string
	ReceiptID         string
	OwnerID           string
	Alias             string
	Collection        string
	TenantID          string
	ArmID             string
	RecipeID          string
	RuntimeRole       string
	GenerationVersion int64
	Dimensions        int
	SourceCount       int
	PointCount        int
	RequestBytes      int64
	RequestSHA256     string
	SourceSequence    int64
	SourceVersions    []SemanticFleetDeleteSource
	PeakBytes         int64
	ObservedAt        time.Time
	ExpiresAt         time.Time
	Attribution       SemanticAttribution
}

// SemanticFleetGeneration retains the complete durable generation facts that
// the shared owner returns. Disk fields preserve their known/unknown state;
// Notes must never convert an absent observation to zero measured bytes.
type SemanticFleetGeneration struct {
	FleetGenerationConfig
	SnapshotID              string
	SnapshotWatermark       string
	CatchupWatermark        string
	IndexSpecDigest         string
	DataVersion             int64
	ValidationDataVersion   int64
	ValidationWatermark     string
	ValidationExpected      uint64
	ValidationActual        uint64
	ValidationValid         bool
	ValidationDigest        string
	CreatedAt               time.Time
	RetiredAt               time.Time
	RetentionExpiresAt      time.Time
	QdrantDiskBytes         int64
	QdrantDiskBytesKnown    bool
	DiskObservedAt          time.Time
	DiskReclaimedBytesKnown bool
	RollbackTo              string
	State                   string
	Cursor                  string
	Version                 int64
	Protected               bool
	UpdatedAt               time.Time
}

type SemanticFleetQueryResult struct {
	Generation SemanticFleetGeneration
	Admission  SemanticFleetAdmissionReceipt
	Results    []SemanticSearchHit
}

type SemanticFleetOperationResult struct {
	Generation SemanticFleetGeneration
	Admission  SemanticFleetAdmissionReceipt
}

// SemanticFleetStore is the Notes composition boundary for the shared
// owner-bound fleet semantic API. Its adapter must validate the cataloged
// active generation, embed with that generation's complete recipe, bind a
// fresh per-operation admission receipt, and constrain queries by tenant.
// The eventual ai-go adapter maps these Requests directly to
// aisearch.FleetSemanticStore; keeping the dependency out of this module until
// Scenario Dependency Analyzer admits it prevents a local replace bypass.
type SemanticFleetStore interface {
	QueryActive(context.Context, SemanticFleetQuery) (SemanticFleetQueryResult, error)
}

// OrderedSemanticFleetMutationStore is intentionally separate from serving
// queries. Implementations must enforce per-source monotonic SourceSequence
// fences before vector I/O; a feed sequence carried only as metadata is not
// sufficient to prevent stale upserts from resurrecting deleted Notes.
type OrderedSemanticFleetMutationStore interface {
	UpsertActive(context.Context, SemanticFleetUpsert) (SemanticFleetOperationResult, error)
	DeleteActive(context.Context, SemanticFleetDelete) (SemanticFleetOperationResult, error)
}

// SemanticServing connects Notes production routes to the protected active
// generation and the owner-bound semantic store. There is deliberately no
// direct Qdrant client, encoder, or separately invented write admission path.
type SemanticServing struct {
	Generations ActiveSemanticGenerationResolver
	Tenants     SemanticTenantResolver
	Attribution SemanticAttributionResolver
	Store       SemanticFleetStore
	Mutations   OrderedSemanticFleetMutationStore
}

func (s *SemanticServing) ValidateActiveGeneration(ctx context.Context) (FleetGenerationConfig, error) {
	if s == nil || s.Generations == nil || s.Tenants == nil || s.Attribution == nil || s.Store == nil {
		return FleetGenerationConfig{}, errors.New("Notes semantic serving requires an active generation, authenticated tenant resolver, trusted rights/quality attribution, and admitted owner store")
	}
	config, err := s.Generations.ResolveActiveSemanticGeneration(ctx)
	if err != nil {
		return FleetGenerationConfig{}, err
	}
	if strings.TrimSpace(config.GenerationID) == "" || strings.TrimSpace(config.OwnerID) == "" || strings.TrimSpace(config.Alias) == "" || strings.TrimSpace(config.Collection) == "" || strings.TrimSpace(config.RecipeID) == "" || strings.TrimSpace(config.EmbeddingRole) == "" || config.EmbeddingDimensions <= 0 {
		return FleetGenerationConfig{}, errors.New("active Notes semantic generation has incomplete generation, owner, collection, recipe, role or dimension identity")
	}
	return config, nil
}

func (s *SemanticServing) operation(ctx context.Context) (FleetGenerationConfig, SemanticAttribution, error) {
	config, err := s.ValidateActiveGeneration(ctx)
	if err != nil {
		return FleetGenerationConfig{}, SemanticAttribution{}, err
	}
	attribution, err := s.Attribution.ResolveSemanticAttribution(ctx)
	if err != nil {
		return FleetGenerationConfig{}, SemanticAttribution{}, fmt.Errorf("resolve Notes semantic rights and quality attribution: %w", err)
	}
	if err := attribution.validate(); err != nil {
		return FleetGenerationConfig{}, SemanticAttribution{}, err
	}
	return config, attribution, nil
}

func (s *SemanticServing) authorizeTenant(ctx context.Context, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" || s == nil || s.Tenants == nil {
		return "", errors.New("Notes semantic operation requires an authenticated tenant scope")
	}
	resolved, err := s.Tenants.ResolveSemanticTenant(ctx, requested)
	if err != nil {
		return "", fmt.Errorf("authorize Notes semantic tenant: %w", err)
	}
	resolved = strings.TrimSpace(resolved)
	if resolved == "" || resolved != requested {
		return "", errors.New("requested Notes tenant does not match authenticated tenant scope")
	}
	return resolved, nil
}

func (a SemanticAttribution) validate() error {
	if strings.TrimSpace(a.CallerID) == "" || strings.TrimSpace(a.RightsEvidenceURI) == "" || !validSHA256(a.RightsEvidenceSHA256) || strings.TrimSpace(a.QualityEvidenceURI) == "" || !validSHA256(a.QualityEvidenceSHA256) {
		return errors.New("Notes semantic operations require caller identity and exact rights and quality evidence references")
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func (s *SemanticServing) Search(ctx context.Context, query, tenantID string, limit int) (SemanticFleetQueryResult, error) {
	query = strings.TrimSpace(query)
	tenantID = strings.TrimSpace(tenantID)
	if query == "" || tenantID == "" || limit <= 0 {
		return SemanticFleetQueryResult{}, errors.New("semantic query, tenant identity and positive limit are required")
	}
	authorizedTenant, err := s.authorizeTenant(ctx, tenantID)
	if err != nil {
		return SemanticFleetQueryResult{}, err
	}
	config, attribution, err := s.operation(ctx)
	if err != nil {
		return SemanticFleetQueryResult{}, err
	}
	return s.Store.QueryActive(ctx, SemanticFleetQuery{
		OperationID: uuid.NewString(), ArmID: config.GenerationID, RecipeID: config.RecipeID,
		TenantID: authorizedTenant, Text: query, Limit: limit, Attribution: attribution,
	})
}

func (s *SemanticServing) UpsertNote(ctx context.Context, document FleetDocument, operationID string, sourceSequence int64) (SemanticFleetOperationResult, error) {
	document.NoteID = strings.TrimSpace(document.NoteID)
	document.UserID = strings.TrimSpace(document.UserID)
	operationID = strings.TrimSpace(operationID)
	if document.NoteID == "" || document.UserID == "" || operationID == "" || sourceSequence == 0 {
		return SemanticFleetOperationResult{}, errors.New("semantic note upsert requires note, tenant, durable operation identity and positive source sequence")
	}
	if s == nil || s.Mutations == nil {
		return SemanticFleetOperationResult{}, errors.New("Notes semantic mutation route remains disabled until an ordered source-sequence owner is installed")
	}
	authorizedTenant, err := s.authorizeTenant(ctx, document.UserID)
	if err != nil {
		return SemanticFleetOperationResult{}, err
	}
	document.UserID = authorizedTenant
	config, attribution, err := s.operation(ctx)
	if err != nil {
		return SemanticFleetOperationResult{}, err
	}
	text := semanticDocumentText(document)
	if text == "" {
		return SemanticFleetOperationResult{}, errors.New("semantic note upsert requires non-empty searchable content")
	}
	return s.Mutations.UpsertActive(ctx, SemanticFleetUpsert{
		OperationID: operationID, ArmID: config.GenerationID, RecipeID: config.RecipeID,
		TenantID: document.UserID, SourceID: fleetSourceID(document.UserID, document.NoteID), SourceSequence: sourceSequence,
		Document:    SemanticFleetDocument{Text: text, Payload: semanticDocumentPayload(document)},
		Attribution: attribution,
	})
}

func (s *SemanticServing) DeleteNote(ctx context.Context, tenantID, noteID, operationID string, sourceSequence int64) (SemanticFleetOperationResult, error) {
	tenantID = strings.TrimSpace(tenantID)
	noteID = strings.TrimSpace(noteID)
	operationID = strings.TrimSpace(operationID)
	if tenantID == "" || noteID == "" || operationID == "" || sourceSequence == 0 {
		return SemanticFleetOperationResult{}, errors.New("semantic note delete requires note, tenant, durable operation identity and positive source sequence")
	}
	if s == nil || s.Mutations == nil {
		return SemanticFleetOperationResult{}, errors.New("Notes semantic mutation route remains disabled until an ordered source-sequence owner is installed")
	}
	authorizedTenant, err := s.authorizeTenant(ctx, tenantID)
	if err != nil {
		return SemanticFleetOperationResult{}, err
	}
	config, attribution, err := s.operation(ctx)
	if err != nil {
		return SemanticFleetOperationResult{}, err
	}
	return s.Mutations.DeleteActive(ctx, SemanticFleetDelete{
		OperationID: operationID, ArmID: config.GenerationID, RecipeID: config.RecipeID,
		TenantID: authorizedTenant, Sources: []SemanticFleetDeleteSource{{SourceID: fleetSourceID(authorizedTenant, noteID), SourceSequence: sourceSequence}}, Attribution: attribution,
	})
}

func semanticDocumentText(document FleetDocument) string {
	parts := []string{strings.TrimSpace(document.Title), strings.TrimSpace(document.Content)}
	if document.Summary != nil {
		parts = append(parts, strings.TrimSpace(*document.Summary))
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func semanticDocumentPayload(document FleetDocument) map[string]any {
	payload := map[string]any{
		"note_id": document.NoteID, "user_id": document.UserID, "title": document.Title,
		"content": document.Content, "is_pinned": document.IsPinned,
		"is_archived": document.IsArchived, "is_favorite": document.IsFavorite,
	}
	if document.Summary != nil {
		payload["summary"] = *document.Summary
	}
	if document.FolderID != nil {
		payload["folder_id"] = *document.FolderID
	}
	return payload
}
