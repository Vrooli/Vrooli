package notes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FleetSnapshot and the related types are the Notes-owned source boundary for
// semantic-index migration. The shared ai-go contract can be adapted here once
// its module is available through the governed dependency path.
type FleetSnapshot struct {
	ID        string
	Watermark string
}

type FleetPoint struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type FleetChange struct {
	Sequence          string
	Upserts           []FleetPoint
	Deletes           []string
	DeleteUserID      string
	DeleteTenantKnown bool
	MoveGroup         string
}

type FleetDocument struct {
	NoteID     string  `json:"note_id"`
	UserID     string  `json:"user_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Summary    *string `json:"summary"`
	FolderID   *string `json:"folder_id"`
	IsPinned   bool    `json:"is_pinned"`
	IsArchived bool    `json:"is_archived"`
	IsFavorite bool    `json:"is_favorite"`
}

// RecipeEmbeddingResolver must resolve the recipe ID to its approved runtime
// role/policy before embedding. Returning the recipe identity alongside the
// vector prevents a role change from silently reusing a different recipe.
type RecipeEmbeddingResolver interface {
	EmbedDocument(context.Context, string, string) (vector []float32, dimensions int, err error)
}

type EmbeddingRolePolicy struct {
	RecipeID   string
	Role       string
	Dimensions int
}

type EmbeddingPolicyResolver interface {
	ResolveEmbeddingPolicy(context.Context, string) (EmbeddingRolePolicy, error)
}

// FleetGenerationConfig identifies the exact candidate generation and encoder
// a Notes source may serve. Collection and embedding settings must come from an
// owner; this type intentionally has no environment or default values.
type FleetGenerationConfig struct {
	GenerationID        string
	OwnerID             string
	Alias               string
	Collection          string
	RecipeID            string
	EmbeddingRole       string
	EmbeddingDimensions int
}

type FleetGenerationConfigResolver interface {
	ResolveFleetGenerationConfig(context.Context, string) (FleetGenerationConfig, error)
}

type CatalogFleetGenerationConfigResolver struct {
	Catalog  FleetCatalog
	Policies EmbeddingPolicyResolver
}

type ActiveFleetGenerationCatalog interface {
	LoadActive(context.Context, string, string) (CatalogIdentity, bool, error)
}

// CatalogActiveSemanticGenerationResolver resolves serving identity only from
// a protected active Notes generation and verifies its immutable recipe policy.
// Owner and alias are explicit inputs; neither collection nor embedding role
// can come from process environment defaults.
type CatalogActiveSemanticGenerationResolver struct {
	Catalog  ActiveFleetGenerationCatalog
	Policies EmbeddingPolicyResolver
	OwnerID  string
	Alias    string
}

func (r CatalogActiveSemanticGenerationResolver) ResolveActiveSemanticGeneration(ctx context.Context) (FleetGenerationConfig, error) {
	if r.Catalog == nil || r.Policies == nil || strings.TrimSpace(r.OwnerID) == "" || strings.TrimSpace(r.Alias) == "" {
		return FleetGenerationConfig{}, errors.New("active Notes fleet catalog, policy, owner and alias are required")
	}
	item, found, err := r.Catalog.LoadActive(ctx, r.OwnerID, r.Alias)
	if err != nil {
		return FleetGenerationConfig{}, err
	}
	if !found || item.State != "active" || !item.Protected {
		return FleetGenerationConfig{}, errors.New("Notes semantic serving has no protected active generation for the requested owner and alias")
	}
	policy, err := r.Policies.ResolveEmbeddingPolicy(ctx, item.RecipeID)
	if err != nil {
		return FleetGenerationConfig{}, err
	}
	if policy.RecipeID != item.RecipeID || policy.Dimensions != item.Dimensions || strings.TrimSpace(policy.Role) == "" {
		return FleetGenerationConfig{}, errors.New("active Notes generation differs from its immutable embedding policy")
	}
	return FleetGenerationConfig{GenerationID: item.ArmID, OwnerID: item.OwnerID, Alias: item.Alias, Collection: item.Collection, RecipeID: item.RecipeID, EmbeddingRole: policy.Role, EmbeddingDimensions: policy.Dimensions}, nil
}

func (r CatalogFleetGenerationConfigResolver) ResolveFleetGenerationConfig(ctx context.Context, armID string) (FleetGenerationConfig, error) {
	if r.Policies == nil {
		return FleetGenerationConfig{}, errors.New("Notes fleet embedding policy resolver is required")
	}
	item, found, err := r.Catalog.Load(ctx, armID)
	if err != nil {
		return FleetGenerationConfig{}, err
	}
	if !found || item.State == "deleted" || strings.TrimSpace(item.ArmID) == "" {
		return FleetGenerationConfig{}, errors.New("Notes fleet generation is unavailable")
	}
	policy, err := r.Policies.ResolveEmbeddingPolicy(ctx, item.RecipeID)
	if err != nil {
		return FleetGenerationConfig{}, err
	}
	if policy.RecipeID != item.RecipeID || policy.Dimensions != item.Dimensions || strings.TrimSpace(policy.Role) == "" {
		return FleetGenerationConfig{}, errors.New("Notes fleet generation differs from its immutable embedding policy")
	}
	return FleetGenerationConfig{
		GenerationID: item.ArmID, OwnerID: item.OwnerID, Alias: item.Alias, Collection: item.Collection,
		RecipeID: item.RecipeID, EmbeddingRole: policy.Role, EmbeddingDimensions: policy.Dimensions,
	}, nil
}

// ConfiguredFleetSource binds source pages and changes to one durable arm's
// collection and immutable embedding recipe. It does not perform Qdrant I/O;
// the shared fleet adapter remains the only candidate collection writer.
type ConfiguredFleetSource struct {
	Source FleetSource
	Config FleetGenerationConfig
}

func NewConfiguredFleetSource(ctx context.Context, db *sql.DB, armID string, configs FleetGenerationConfigResolver, policies EmbeddingPolicyResolver, gateway RoleGateway) (ConfiguredFleetSource, error) {
	if db == nil || strings.TrimSpace(armID) == "" || configs == nil || policies == nil || gateway == nil {
		return ConfiguredFleetSource{}, errors.New("Notes fleet database, arm, owner config, embedding policy and gateway are required")
	}
	config, err := configs.ResolveFleetGenerationConfig(ctx, armID)
	if err != nil {
		return ConfiguredFleetSource{}, err
	}
	if strings.TrimSpace(config.GenerationID) != strings.TrimSpace(armID) || strings.TrimSpace(config.OwnerID) == "" || strings.TrimSpace(config.Alias) == "" || strings.TrimSpace(config.Collection) == "" || strings.TrimSpace(config.RecipeID) == "" || strings.TrimSpace(config.EmbeddingRole) == "" || config.EmbeddingDimensions <= 0 {
		return ConfiguredFleetSource{}, errors.New("Notes fleet owner config must identify the requested generation, owner, alias, collection, recipe, embedding role and positive dimensions")
	}
	policy, err := policies.ResolveEmbeddingPolicy(ctx, config.RecipeID)
	if err != nil {
		return ConfiguredFleetSource{}, err
	}
	if policy.RecipeID != config.RecipeID || policy.Role != config.EmbeddingRole || policy.Dimensions != config.EmbeddingDimensions {
		return ConfiguredFleetSource{}, errors.New("Notes fleet embedding policy differs from the generation's immutable recipe identity")
	}
	boundPolicy := policy
	return ConfiguredFleetSource{
		Source: FleetSource{DB: db, Embedder: RoleBoundEmbeddingResolver{Policies: policies, Gateway: gateway, Expected: &boundPolicy}},
		Config: config,
	}, nil
}

func (s ConfiguredFleetSource) requireRecipe(recipeID string) error {
	if strings.TrimSpace(s.Config.RecipeID) == "" || recipeID != s.Config.RecipeID {
		return errors.New("Notes fleet request recipe does not match the configured generation")
	}
	return nil
}

func (s ConfiguredFleetSource) PinFleetSnapshot(ctx context.Context, id string) (FleetSnapshot, error) {
	return s.Source.PinFleetSnapshot(ctx, id)
}

func (s ConfiguredFleetSource) ReadFleetSnapshotPage(ctx context.Context, recipeID string, snapshot FleetSnapshot, after string, limit int) ([]FleetPoint, string, bool, error) {
	if err := s.requireRecipe(recipeID); err != nil {
		return nil, "", false, err
	}
	return s.Source.ReadFleetSnapshotPage(ctx, recipeID, snapshot, after, limit)
}

func (s ConfiguredFleetSource) ReadFleetChanges(ctx context.Context, recipeID, snapshotWatermark, after string, limit int) ([]FleetChange, string, error) {
	if err := s.requireRecipe(recipeID); err != nil {
		return nil, "", err
	}
	return s.Source.ReadFleetChanges(ctx, recipeID, snapshotWatermark, after, limit)
}

func (s ConfiguredFleetSource) CompareFleetSequence(ctx context.Context, before, after string) (int, error) {
	return s.Source.CompareFleetSequence(ctx, before, after)
}

func (s ConfiguredFleetSource) FleetWatermark(ctx context.Context) (string, error) {
	return s.Source.FleetWatermark(ctx)
}

func (s ConfiguredFleetSource) ExpectedFleetPointCount(ctx context.Context, recipeID, snapshotID, watermark string) (uint64, error) {
	if err := s.requireRecipe(recipeID); err != nil {
		return 0, err
	}
	return s.Source.ExpectedFleetPointCount(ctx, recipeID, snapshotID, watermark)
}

func (s ConfiguredFleetSource) AcquireFleetCutover(ctx context.Context, expected string) (FleetCutoverFence, error) {
	return s.Source.AcquireFleetCutover(ctx, expected)
}

type RoleGateway interface {
	Embed(context.Context, string, string) ([]float32, error)
}

// RoleBoundEmbeddingResolver resolves every immutable recipe through the
// configured policy before calling the role gateway. This keeps role selection
// out of environment defaults and rejects recipe or vector dimension drift.
type RoleBoundEmbeddingResolver struct {
	Policies EmbeddingPolicyResolver
	Gateway  RoleGateway
	Expected *EmbeddingRolePolicy
}

func (r RoleBoundEmbeddingResolver) EmbedDocument(ctx context.Context, recipeID, document string) ([]float32, int, error) {
	if r.Policies == nil || r.Gateway == nil || strings.TrimSpace(recipeID) == "" {
		return nil, 0, errors.New("embedding policy resolver, role gateway, and recipe identity are required")
	}
	policy, err := r.Policies.ResolveEmbeddingPolicy(ctx, recipeID)
	if err != nil {
		return nil, 0, err
	}
	if policy.RecipeID != recipeID || strings.TrimSpace(policy.Role) == "" || policy.Dimensions <= 0 {
		return nil, 0, errors.New("embedding policy does not match the immutable recipe identity")
	}
	if r.Expected != nil && policy != *r.Expected {
		return nil, 0, errors.New("embedding policy changed from the configured generation identity")
	}
	vector, err := r.Gateway.Embed(ctx, policy.Role, document)
	if err != nil {
		return nil, 0, err
	}
	if len(vector) != policy.Dimensions {
		return nil, 0, fmt.Errorf("embedding role %q returned %d values, recipe %q requires %d dimensions", policy.Role, len(vector), recipeID, policy.Dimensions)
	}
	return vector, policy.Dimensions, nil
}

type FleetSource struct {
	DB       *sql.DB
	Embedder RecipeEmbeddingResolver
}

// NotesFleetSourceContract mirrors the shared fleet adapter's source methods
// while keeping this scenario buildable until governed ai-go admission is
// available. The method order, recipe, watermark and page cursor boundaries are
// deliberately explicit; no Qdrant operation is exposed here.
type NotesFleetSourceContract interface {
	PinFleetSnapshot(context.Context, string) (FleetSnapshot, error)
	ReadFleetSnapshotPage(context.Context, string, FleetSnapshot, string, int) ([]FleetPoint, string, bool, error)
	ReadFleetChanges(context.Context, string, string, string, int) ([]FleetChange, string, error)
	CompareFleetSequence(context.Context, string, string) (int, error)
	FleetWatermark(context.Context) (string, error)
	ExpectedFleetPointCount(context.Context, string, string, string) (uint64, error)
	AcquireFleetCutover(context.Context, string) (FleetCutoverFence, error)
}

var (
	_ NotesFleetSourceContract = FleetSource{}
	_ NotesFleetSourceContract = ConfiguredFleetSource{}
)

const notesFleetBaselineVersion = 1

func (s FleetSource) embedDocument(ctx context.Context, recipeID, id string, document FleetDocument) (FleetPoint, error) {
	if strings.TrimSpace(document.NoteID) == "" || document.NoteID != id {
		return FleetPoint{}, fmt.Errorf("Notes fleet source document identity does not match note %q", id)
	}
	vector, dimensions, err := s.Embedder.EmbedDocument(ctx, recipeID, document.Title+"\n"+document.Content)
	if err != nil {
		return FleetPoint{}, fmt.Errorf("embed note %q for recipe %q: %w", id, recipeID, err)
	}
	if dimensions <= 0 || len(vector) != dimensions {
		return FleetPoint{}, fmt.Errorf("embedding for recipe %q returned %d values, expected %d dimensions", recipeID, len(vector), dimensions)
	}
	sourceID := fleetSourceID(document.UserID, document.NoteID)
	userID := fleetUserPayload(document.UserID)
	return FleetPoint{ID: sourceID, Vector: vector, Payload: map[string]any{
		"note_id": document.NoteID, "user_id": userID, "user_id_known": userID != nil, "title": document.Title,
		"content": document.Content, "summary": document.Summary, "folder_id": document.FolderID,
		"is_pinned": document.IsPinned, "is_archived": document.IsArchived, "is_favorite": document.IsFavorite,
		"source_id": sourceID, "embedding_recipe_id": recipeID,
	}}, nil
}

func (s FleetSource) PinFleetSnapshot(ctx context.Context, id string) (FleetSnapshot, error) {
	if s.DB == nil || strings.TrimSpace(id) == "" {
		return FleetSnapshot{}, errors.New("notes fleet database and snapshot ID are required")
	}
	id = strings.TrimSpace(id)
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO notes_fleet_snapshots(snapshot_id, watermark, baseline_version)
		SELECT $1, sequence.current_sequence, baseline.version
		FROM notes_fleet_sequence AS sequence
		CROSS JOIN notes_fleet_backfill AS baseline
		WHERE sequence.singleton = TRUE AND baseline.singleton = TRUE
		ON CONFLICT (snapshot_id) DO NOTHING`, id)
	if err != nil {
		return FleetSnapshot{}, err
	}
	// Loading and validating after insert makes retries resume the original
	// pinned boundary while rejecting snapshots made before a completed seed.
	var watermark int64
	if err := s.DB.QueryRowContext(ctx, `SELECT watermark FROM notes_fleet_snapshots WHERE snapshot_id = $1`, id).Scan(&watermark); err != nil {
		return FleetSnapshot{}, err
	}
	if err := s.validateFleetSnapshot(ctx, id, strconv.FormatInt(watermark, 10)); err != nil {
		return FleetSnapshot{}, err
	}
	return FleetSnapshot{ID: id, Watermark: strconv.FormatInt(watermark, 10)}, nil
}

func (s FleetSource) ReadFleetSnapshotPage(ctx context.Context, recipeID string, snapshot FleetSnapshot, after string, limit int) ([]FleetPoint, string, bool, error) {
	if s.DB == nil || s.Embedder == nil || strings.TrimSpace(recipeID) == "" || limit <= 0 {
		return nil, "", false, errors.New("notes fleet source, recipe, embedder, and positive page size are required")
	}
	watermark, err := parseFleetCursor(snapshot.Watermark)
	if err != nil {
		return nil, "", false, err
	}
	if err := s.validateFleetSnapshot(ctx, snapshot.ID, snapshot.Watermark); err != nil {
		return nil, "", false, err
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT note_id::text, payload
		FROM (
		  SELECT DISTINCT ON (note_id) note_id, event_type, payload
		  FROM notes_fleet_changes WHERE sequence <= $1
		  ORDER BY note_id, sequence DESC
		) AS latest
		WHERE event_type = 'upsert' AND note_id::text > $2
		ORDER BY note_id LIMIT $3`, watermark, after, limit+1)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()
	type item struct {
		id      string
		payload []byte
	}
	items := make([]item, 0, limit+1)
	for rows.Next() {
		var value item
		if err := rows.Scan(&value.id, &value.payload); err != nil {
			return nil, "", false, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}
	done := len(items) <= limit
	if !done {
		items = items[:limit]
	}
	points := make([]FleetPoint, 0, len(items))
	for _, value := range items {
		var document FleetDocument
		if err := json.Unmarshal(value.payload, &document); err != nil {
			return nil, "", false, fmt.Errorf("decode note snapshot %q: %w", value.id, err)
		}
		point, err := s.embedDocument(ctx, recipeID, value.id, document)
		if err != nil {
			return nil, "", false, err
		}
		points = append(points, point)
	}
	next := after
	if len(items) > 0 {
		next = items[len(items)-1].id
	}
	return points, next, done, nil
}

func (s FleetSource) ReadFleetChanges(ctx context.Context, recipeID, snapshotWatermark, after string, limit int) ([]FleetChange, string, error) {
	if s.DB == nil || s.Embedder == nil || strings.TrimSpace(recipeID) == "" || limit <= 0 {
		return nil, "", errors.New("notes fleet source, recipe, embedder, and positive page size are required")
	}
	baseline, err := parseFleetCursor(snapshotWatermark)
	if err != nil {
		return nil, "", err
	}
	cursor, err := parseFleetCursor(after)
	if err != nil {
		return nil, "", err
	}
	if cursor < baseline {
		return nil, "", errors.New("Notes fleet change cursor precedes its pinned snapshot")
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT sequence, note_id::text, user_id::text, event_type, payload, move_group::text
		FROM notes_fleet_changes WHERE sequence > GREATEST($1::BIGINT, $2::BIGINT)
		ORDER BY sequence LIMIT $3`, baseline, cursor, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	type sourceChange struct {
		sequence  int64
		noteID    string
		userID    sql.NullString
		eventType string
		payload   []byte
		moveGroup sql.NullString
	}
	page := make([]sourceChange, 0, limit+1)
	for rows.Next() {
		var item sourceChange
		if err := rows.Scan(&item.sequence, &item.noteID, &item.userID, &item.eventType, &item.payload, &item.moveGroup); err != nil {
			return nil, "", err
		}
		page = append(page, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	changes := make([]FleetChange, 0, limit)
	consumed := 0
	for i := 0; i < len(page) && consumed < limit; {
		item := page[i]
		if item.sequence != cursor+1 {
			return nil, "", fmt.Errorf("Notes fleet change stream gap: expected sequence %d, got %d", cursor+1, item.sequence)
		}
		change := FleetChange{}
		if item.moveGroup.Valid {
			if item.eventType != "delete" {
				return nil, "", errors.New("Notes fleet cursor starts inside a tenant-move event group")
			}
			if i+1 >= len(page) {
				return nil, "", errors.New("Notes fleet tenant-move tombstone is missing its paired upsert")
			}
			next := page[i+1]
			if next.sequence != item.sequence+1 || !next.moveGroup.Valid || next.moveGroup.String != item.moveGroup.String || next.eventType != "upsert" || next.noteID != item.noteID || next.userID.String == item.userID.String && next.userID.Valid == item.userID.Valid {
				return nil, "", errors.New("Notes fleet tenant-move pair is incomplete or out of order")
			}
			var document FleetDocument
			if err := json.Unmarshal(next.payload, &document); err != nil {
				return nil, "", fmt.Errorf("decode note tenant-move upsert %d: %w", next.sequence, err)
			}
			if document.UserID != next.userID.String {
				return nil, "", errors.New("Notes fleet tenant-move payload and event ownership differ")
			}
			point, err := s.embedDocument(ctx, recipeID, next.noteID, document)
			if err != nil {
				return nil, "", err
			}
			change = FleetChange{
				Sequence:     strconv.FormatInt(next.sequence, 10),
				Upserts:      []FleetPoint{point},
				Deletes:      []string{fleetSourceID(item.userID.String, item.noteID)},
				DeleteUserID: item.userID.String, DeleteTenantKnown: item.userID.Valid,
				MoveGroup: item.moveGroup.String,
			}
			cursor = next.sequence
			consumed += 2
			i += 2
			changes = append(changes, change)
			continue
		}
		change.Sequence = strconv.FormatInt(item.sequence, 10)
		switch item.eventType {
		case "delete":
			change.Deletes = []string{fleetSourceID(item.userID.String, item.noteID)}
			change.DeleteUserID = item.userID.String
			change.DeleteTenantKnown = item.userID.Valid
		case "upsert":
			var document FleetDocument
			if err := json.Unmarshal(item.payload, &document); err != nil {
				return nil, "", fmt.Errorf("decode note change %d: %w", item.sequence, err)
			}
			point, err := s.embedDocument(ctx, recipeID, item.noteID, document)
			if err != nil {
				return nil, "", err
			}
			change.Upserts = []FleetPoint{point}
		default:
			return nil, "", fmt.Errorf("Notes fleet change %d has unknown event type %q", item.sequence, item.eventType)
		}
		changes = append(changes, change)
		cursor = item.sequence
		consumed++
		i++
	}
	next := after
	if len(changes) > 0 {
		next = changes[len(changes)-1].Sequence
	}
	return changes, next, nil
}

func (s FleetSource) FleetWatermark(ctx context.Context) (string, error) {
	if s.DB == nil {
		return "", errors.New("notes fleet database is required")
	}
	var watermark int64
	if err := s.DB.QueryRowContext(ctx, `SELECT current_sequence FROM notes_fleet_sequence WHERE singleton = TRUE`).Scan(&watermark); err != nil {
		return "", err
	}
	return strconv.FormatInt(watermark, 10), nil
}

// ExpectedFleetPointCount returns the number of live, unchunked Notes points
// through a source watermark. This remains one point per note; if Notes adopts
// chunk expansion, both this count and the point IDs must change together.
func (s FleetSource) ExpectedFleetPointCount(ctx context.Context, recipeID, snapshotID, watermark string) (uint64, error) {
	if s.DB == nil || strings.TrimSpace(recipeID) == "" || strings.TrimSpace(snapshotID) == "" {
		return 0, errors.New("Notes fleet database, recipe, and snapshot identity are required")
	}
	through, err := parseFleetCursor(watermark)
	if err != nil {
		return 0, err
	}
	pinned, err := s.loadFleetSnapshotWatermark(ctx, snapshotID)
	if err != nil {
		return 0, err
	}
	if through < pinned {
		return 0, errors.New("Notes fleet point-count watermark precedes its pinned snapshot")
	}
	current, err := s.FleetWatermark(ctx)
	if err != nil {
		return 0, err
	}
	currentSequence, err := parseFleetCursor(current)
	if err != nil {
		return 0, err
	}
	if through > currentSequence {
		return 0, errors.New("Notes fleet point-count watermark is ahead of the source")
	}
	var count int64
	if err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
		  SELECT DISTINCT ON (note_id) event_type
		  FROM notes_fleet_changes WHERE sequence <= $1
		  ORDER BY note_id, sequence DESC
		) AS latest WHERE event_type = 'upsert'`, through).Scan(&count); err != nil {
		return 0, err
	}
	if count < 0 {
		return 0, errors.New("Notes fleet point count cannot be negative")
	}
	return uint64(count), nil
}

func (s FleetSource) validateFleetSnapshot(ctx context.Context, snapshotID, expectedWatermark string) error {
	expected, err := parseFleetCursor(expectedWatermark)
	if err != nil {
		return err
	}
	watermark, err := s.loadFleetSnapshotWatermark(ctx, snapshotID)
	if err != nil {
		return err
	}
	if watermark != expected {
		return errors.New("Notes fleet snapshot watermark does not match its pinned identity")
	}
	return nil
}

func (s FleetSource) loadFleetSnapshotWatermark(ctx context.Context, snapshotID string) (int64, error) {
	if s.DB == nil || strings.TrimSpace(snapshotID) == "" {
		return 0, errors.New("Notes fleet database and snapshot identity are required")
	}
	var watermark, snapshotVersion, baselineVersion, baselineWatermark int64
	if err := s.DB.QueryRowContext(ctx, `
		SELECT snapshot.watermark, snapshot.baseline_version, baseline.version, baseline.baseline_watermark
		FROM notes_fleet_snapshots AS snapshot
		CROSS JOIN notes_fleet_backfill AS baseline
		WHERE snapshot.snapshot_id = $1 AND baseline.singleton = TRUE`, snapshotID).Scan(&watermark, &snapshotVersion, &baselineVersion, &baselineWatermark); err != nil {
		return 0, err
	}
	if baselineVersion != notesFleetBaselineVersion || snapshotVersion != baselineVersion || watermark < baselineWatermark {
		return 0, errors.New("Notes fleet snapshot predates the completed source backfill")
	}
	return watermark, nil
}

// AcquireFleetCutover serializes the alias switch with every indexed note
// mutation. The database trigger takes the same singleton row lock before it
// commits an indexed change, so holding this transaction freezes the source
// watermark until Release commits the fence.
func (s FleetSource) AcquireFleetCutover(ctx context.Context, expected string) (FleetCutoverFence, error) {
	if s.DB == nil {
		return nil, errors.New("Notes fleet database is required")
	}
	if _, err := parseFleetCursor(expected); err != nil {
		return nil, err
	}
	expectedSequence, _ := parseFleetCursor(expected)
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT current_sequence FROM notes_fleet_sequence WHERE singleton = TRUE FOR UPDATE`).Scan(&current); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if current != expectedSequence {
		_ = tx.Rollback()
		return nil, errors.New("Notes fleet cutover watermark is stale; catch up before fencing")
	}
	return &notesFleetCutoverFence{tx: tx, watermark: strconv.FormatInt(current, 10)}, nil
}

type FleetCutoverFence interface {
	CurrentWatermark(context.Context) (string, error)
	Release(context.Context) error
}

type notesFleetCutoverFence struct {
	mu        sync.Mutex
	tx        *sql.Tx
	watermark string
	released  bool
}

func (f *notesFleetCutoverFence) CurrentWatermark(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.released || f.tx == nil {
		return "", errors.New("Notes fleet cutover fence has been released")
	}
	var current int64
	if err := f.tx.QueryRowContext(ctx, `SELECT current_sequence FROM notes_fleet_sequence WHERE singleton = TRUE`).Scan(&current); err != nil {
		return "", err
	}
	value := strconv.FormatInt(current, 10)
	if value != f.watermark {
		return "", errors.New("Notes fleet source changed while its cutover fence was held")
	}
	return value, nil
}

func (f *notesFleetCutoverFence) Release(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.released || f.tx == nil {
		return errors.New("Notes fleet cutover fence has already been released")
	}
	f.released = true
	return f.tx.Commit()
}

var notesFleetSourceNamespace = uuid.MustParse("2cc1e394-7d4a-5a3e-9f8d-e6e4c9b7c430")

// fleetSourceID is a stable Qdrant-compatible UUID derived from both tenant
// and note identity. The payload retains user_id and note_id for auditability.
func fleetSourceID(userID, noteID string) string {
	userID = strings.TrimSpace(userID)
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return ""
	}
	scope := "unowned"
	if userID != "" {
		scope = "tenant:" + userID
	}
	return uuid.NewSHA1(notesFleetSourceNamespace, []byte(scope+"\x00"+noteID)).String()
}

func fleetUserPayload(userID string) any {
	if strings.TrimSpace(userID) == "" {
		return nil
	}
	return userID
}

func parseFleetCursor(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	cursor, err := strconv.ParseInt(value, 10, 64)
	if err != nil || cursor < 0 {
		return 0, fmt.Errorf("invalid Notes fleet cursor %q", value)
	}
	return cursor, nil
}

// CompareFleetSequence orders the decimal source sequence used by change-feed
// generations. Snapshot page cursors are note IDs and remain lexically ordered
// by the snapshot query; they must never be compared with this numeric cursor.
func (FleetSource) CompareFleetSequence(_ context.Context, before, after string) (int, error) {
	left, err := parseFleetCursor(before)
	if err != nil {
		return 0, err
	}
	right, err := parseFleetCursor(after)
	if err != nil {
		return 0, err
	}
	switch {
	case left < right:
		return -1, nil
	case left > right:
		return 1, nil
	default:
		return 0, nil
	}
}

// CatalogIdentity and CatalogCAS are intentionally independent of the shared
// adapter package to keep this scenario module compilable without an unmanaged
// cross-module dependency.
type CatalogIdentity struct {
	ArmID, OwnerID, Alias, RecipeID, SnapshotID, SnapshotWatermark, CatchupWatermark string
	Dimensions                                                                       int
	Collection, RollbackTo, State, Cursor                                            string
	Version                                                                          int64
	Protected                                                                        bool
	UpdatedAt                                                                        time.Time
}

type FleetCatalog struct{ DB *sql.DB }

func (c FleetCatalog) Load(ctx context.Context, armID string) (CatalogIdentity, bool, error) {
	return c.load(ctx, `WHERE arm_id = $1`, armID)
}

func (c FleetCatalog) LoadByCollection(ctx context.Context, collection string) (CatalogIdentity, bool, error) {
	return c.load(ctx, `WHERE collection = $1`, collection)
}

func (c FleetCatalog) LoadActive(ctx context.Context, ownerID, alias string) (CatalogIdentity, bool, error) {
	if c.DB == nil || strings.TrimSpace(ownerID) == "" || strings.TrimSpace(alias) == "" {
		return CatalogIdentity{}, false, errors.New("Notes fleet database, owner and alias are required")
	}
	rows, err := c.DB.QueryContext(ctx, `SELECT arm_id, owner_id, alias, recipe_id, snapshot_id, snapshot_watermark, catchup_watermark, dimensions, collection, COALESCE(rollback_to,''), state, cursor, version, protected, updated_at FROM notes_fleet_generations WHERE owner_id=$1 AND alias=$2 AND state='active' ORDER BY updated_at DESC LIMIT 2`, ownerID, alias)
	if err != nil {
		return CatalogIdentity{}, false, err
	}
	defer rows.Close()
	var item CatalogIdentity
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CatalogIdentity{}, false, err
		}
		return CatalogIdentity{}, false, nil
	}
	if err := rows.Scan(&item.ArmID, &item.OwnerID, &item.Alias, &item.RecipeID, &item.SnapshotID, &item.SnapshotWatermark, &item.CatchupWatermark, &item.Dimensions, &item.Collection, &item.RollbackTo, &item.State, &item.Cursor, &item.Version, &item.Protected, &item.UpdatedAt); err != nil {
		return CatalogIdentity{}, false, err
	}
	if rows.Next() {
		return CatalogIdentity{}, false, errors.New("Notes fleet catalog has multiple active generations for the owner alias")
	}
	if err := rows.Err(); err != nil {
		return CatalogIdentity{}, false, err
	}
	return item, true, nil
}

func (c FleetCatalog) load(ctx context.Context, predicate, value string) (CatalogIdentity, bool, error) {
	if c.DB == nil {
		return CatalogIdentity{}, false, errors.New("notes fleet database is required")
	}
	query := `SELECT arm_id, owner_id, alias, recipe_id, snapshot_id, snapshot_watermark, catchup_watermark, dimensions, collection, COALESCE(rollback_to,''), state, cursor, version, protected, updated_at FROM notes_fleet_generations ` + predicate
	var item CatalogIdentity
	err := c.DB.QueryRowContext(ctx, query, value).Scan(&item.ArmID, &item.OwnerID, &item.Alias, &item.RecipeID, &item.SnapshotID, &item.SnapshotWatermark, &item.CatchupWatermark, &item.Dimensions, &item.Collection, &item.RollbackTo, &item.State, &item.Cursor, &item.Version, &item.Protected, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return CatalogIdentity{}, false, nil
	}
	if err != nil {
		return CatalogIdentity{}, false, err
	}
	return item, true, nil
}

func (c FleetCatalog) SaveCAS(ctx context.Context, item CatalogIdentity, expectedVersion int64) error {
	if c.DB == nil {
		return errors.New("notes fleet database is required")
	}
	if item.Version != expectedVersion+1 {
		return errors.New("Notes fleet catalog version must advance exactly once")
	}
	if expectedVersion == 0 {
		if item.Version != 1 || item.State != "building" || item.Protected || strings.TrimSpace(item.OwnerID) == "" || strings.TrimSpace(item.Alias) == "" || strings.TrimSpace(item.RecipeID) == "" || strings.TrimSpace(item.Collection) == "" || item.Dimensions <= 0 {
			return errors.New("Notes fleet generation must start as an unprotected, identified building arm")
		}
		watermark, err := parseFleetCursor(item.SnapshotWatermark)
		if err != nil {
			return err
		}
		catchup, err := parseFleetCursor(item.CatchupWatermark)
		if err != nil || catchup < watermark || strings.TrimSpace(item.Cursor) != "" {
			return errors.New("Notes fleet generation must start at its pinned watermark with an empty snapshot cursor")
		}
		if err := (FleetSource{DB: c.DB}).validateFleetSnapshot(ctx, item.SnapshotID, item.SnapshotWatermark); err != nil {
			return fmt.Errorf("Notes fleet generation snapshot is invalid: %w", err)
		}
		_, err = c.DB.ExecContext(ctx, `INSERT INTO notes_fleet_generations(arm_id, owner_id, alias, recipe_id, snapshot_id, snapshot_watermark, catchup_watermark, dimensions, collection, rollback_to, state, cursor, version, protected, updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),$11,$12,$13,$14,$15)`, item.ArmID, item.OwnerID, item.Alias, item.RecipeID, item.SnapshotID, item.SnapshotWatermark, item.CatchupWatermark, item.Dimensions, item.Collection, item.RollbackTo, item.State, item.Cursor, item.Version, item.Protected, item.UpdatedAt)
		return err
	}
	previous, found, err := c.Load(ctx, item.ArmID)
	if err != nil {
		return err
	}
	if !found || !validFleetCatalogAdvance(previous, item) {
		return errors.New("Notes fleet catalog state, cursor, protection, or identity did not advance validly")
	}
	catchup, err := parseFleetCursor(item.CatchupWatermark)
	if err != nil {
		return err
	}
	var sourceWatermark int64
	if err := c.DB.QueryRowContext(ctx, `SELECT current_sequence FROM notes_fleet_sequence WHERE singleton = TRUE`).Scan(&sourceWatermark); err != nil {
		return err
	}
	if catchup > sourceWatermark {
		return errors.New("Notes fleet generation catch-up watermark is ahead of the source")
	}
	result, err := c.DB.ExecContext(ctx, `UPDATE notes_fleet_generations SET owner_id=$2, alias=$3, recipe_id=$4, snapshot_id=$5, snapshot_watermark=$6, catchup_watermark=$7, dimensions=$8, collection=$9, rollback_to=NULLIF($10,''), state=$11, cursor=$12, version=$13, protected=$14, updated_at=$15 WHERE arm_id=$1 AND version=$16 AND owner_id=$2 AND alias=$3 AND recipe_id=$4 AND snapshot_id=$5 AND snapshot_watermark=$6 AND collection=$9 AND dimensions=$8`, item.ArmID, item.OwnerID, item.Alias, item.RecipeID, item.SnapshotID, item.SnapshotWatermark, item.CatchupWatermark, item.Dimensions, item.Collection, item.RollbackTo, item.State, item.Cursor, item.Version, item.Protected, item.UpdatedAt, expectedVersion)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("Notes fleet catalog compare-and-swap conflict or immutable identity mismatch")
	}
	return nil
}

func validFleetCatalogAdvance(old, next CatalogIdentity) bool {
	if old.ArmID != next.ArmID || old.OwnerID != next.OwnerID || old.Alias != next.Alias || old.RecipeID != next.RecipeID || old.SnapshotID != next.SnapshotID || old.SnapshotWatermark != next.SnapshotWatermark || old.Dimensions != next.Dimensions || old.Collection != next.Collection {
		return false
	}
	if strings.TrimSpace(next.Cursor) < strings.TrimSpace(old.Cursor) {
		return false
	}
	if old.State != "building" && next.Cursor != old.Cursor {
		return false
	}
	oldCatchup, err := parseFleetCursor(old.CatchupWatermark)
	if err != nil {
		return false
	}
	nextCatchup, err := parseFleetCursor(next.CatchupWatermark)
	if err != nil || nextCatchup < oldCatchup || nextCatchup < mustFleetCursor(next.SnapshotWatermark) {
		return false
	}
	if old.Protected && !next.Protected || next.State == "protected" && !next.Protected || next.State == "deleted" && (next.Protected || old.Protected) || old.State == "deleted" {
		return false
	}
	if next.State != old.State {
		allowed := map[string]map[string]bool{
			"building":  {"candidate": true, "protected": true},
			"candidate": {"active": true, "retired": true, "protected": true},
			"active":    {"retired": true, "protected": true},
			"retired":   {"active": true, "protected": true, "deleted": true},
		}
		if !allowed[old.State][next.State] {
			return false
		}
	}
	return true
}

func mustFleetCursor(value string) int64 {
	cursor, _ := parseFleetCursor(value)
	return cursor
}
