package experimentarms

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"google.golang.org/protobuf/encoding/protojson"

	aisearch "github.com/vrooli/ai-go/search"
	ownerregistry "search-hub/internal/registry"
	"search-hub/internal/routing"
)

const providerFleetSourceSchema = `CREATE TABLE IF NOT EXISTS provider_description_fleet_snapshots (
  snapshot_id TEXT PRIMARY KEY,
  recipe_id TEXT NOT NULL,
  watermark TEXT NOT NULL,
  digest TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS provider_description_fleet_snapshot_profiles (
  snapshot_id TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  profile_json BLOB NOT NULL,
  PRIMARY KEY(snapshot_id, provider_id),
  FOREIGN KEY(snapshot_id) REFERENCES provider_description_fleet_snapshots(snapshot_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_provider_description_fleet_profiles_snapshot
  ON provider_description_fleet_snapshot_profiles(snapshot_id, provider_id);`

type providerFleetSourceDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// FleetDescriptionEncoder is a trusted runtime binding: the complete immutable
// recipe must describe the vectors it returns. An ordinary Embedder does not
// attest artifact, runtime build, tokenizer or precision and cannot implement
// this interface safely by copying those values from a request.
type FleetDescriptionEncoder interface {
	aisearch.FleetSemanticEmbedder
}

// ProviderDescriptionFleetSource adapts Search Hub's own authoritative
// provider registry into the same profile segments consumed by the router.
// Snapshots and change rows live in Search Hub's SQLite database; this source
// never reads Agent Manager or another scenario's vectors.
type ProviderDescriptionFleetSource struct {
	db      providerFleetSourceDB
	encoder FleetDescriptionEncoder
	recipe  aisearch.EmbeddingRecipe
	dim     int
}

var _ aisearch.FleetSource = (*ProviderDescriptionFleetSource)(nil)

func newProviderDescriptionFleetSource(db providerFleetSourceDB, encoder FleetDescriptionEncoder) (*ProviderDescriptionFleetSource, error) {
	if db == nil || encoder == nil {
		return nil, errors.New("provider-description fleet source requires its owner database and verified encoder")
	}
	recipe := encoder.Recipe()
	if err := recipe.Verify(); err != nil {
		return nil, fmt.Errorf("verified provider-description recipe: %w", err)
	}
	input, err := recipe.Input()
	if err != nil {
		return nil, err
	}
	return &ProviderDescriptionFleetSource{db: db, encoder: encoder, recipe: recipe, dim: input.Dimensions}, nil
}

// CreateFleetSnapshot atomically captures the current active provider profile
// set and the change-feed watermark. The generated ID is owner-controlled and
// remains available for generation replay until protected collection removes
// its final reference.
func (s *ProviderDescriptionFleetSource) CreateFleetSnapshot(ctx context.Context, recipeID string) (aisearch.FleetSnapshot, error) {
	snapshotID, err := randomSnapshotID()
	if err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	return s.createFleetSnapshot(ctx, recipeID, snapshotID)
}

// CreateFleetSnapshotForArm returns the same immutable snapshot for a retry of
// one arm/recipe pair. A retry can resume its generation cursor without
// silently pinning newer source state.
func (s *ProviderDescriptionFleetSource) CreateFleetSnapshotForArm(ctx context.Context, armID, recipeID string) (aisearch.FleetSnapshot, error) {
	armID = strings.TrimSpace(armID)
	if armID == "" {
		return aisearch.FleetSnapshot{}, errors.New("provider-description snapshot arm identity is required")
	}
	sum := sha256.Sum256([]byte(armID + "\x00" + recipeID))
	snapshotID := "provider-profile-snapshot-" + hex.EncodeToString(sum[:16])
	return s.createFleetSnapshot(ctx, recipeID, snapshotID)
}

func (s *ProviderDescriptionFleetSource) createFleetSnapshot(ctx context.Context, recipeID, snapshotID string) (aisearch.FleetSnapshot, error) {
	if err := s.checkRecipe(recipeID); err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return aisearch.FleetSnapshot{}, fmt.Errorf("begin provider-description snapshot: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO provider_description_fleet_snapshots(snapshot_id, recipe_id, watermark, digest) VALUES(?, ?, '0', '') ON CONFLICT(snapshot_id) DO NOTHING`, snapshotID, recipeID)
	if err != nil {
		return aisearch.FleetSnapshot{}, fmt.Errorf("reserve provider-description snapshot identity: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	if created == 0 {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			return aisearch.FleetSnapshot{}, err
		}
		existing, err := s.PinFleetSnapshot(ctx, snapshotID)
		if err != nil {
			return aisearch.FleetSnapshot{}, err
		}
		return existing, nil
	}
	watermark, err := currentProviderChangeWatermark(ctx, tx)
	if err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	descriptors, err := loadActiveProviderDescriptors(ctx, tx)
	if err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	profiles := routing.ProviderProfilesForFleet(descriptors)
	canonical := make([][]byte, 0, len(profiles))
	for _, profile := range profiles {
		raw, err := json.Marshal(profile)
		if err != nil {
			return aisearch.FleetSnapshot{}, fmt.Errorf("encode provider-description profile: %w", err)
		}
		canonical = append(canonical, raw)
		if _, err := tx.ExecContext(ctx, `INSERT INTO provider_description_fleet_snapshot_profiles(snapshot_id, provider_id, profile_json) VALUES(?, ?, ?)`, snapshotID, profile.ProviderID, raw); err != nil {
			return aisearch.FleetSnapshot{}, fmt.Errorf("persist provider-description snapshot profile %q: %w", profile.ProviderID, err)
		}
	}
	snapshotDigest, err := digestProviderProfiles(canonical)
	if err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE provider_description_fleet_snapshots SET watermark = ?, digest = ? WHERE snapshot_id = ?`, watermark, snapshotDigest, snapshotID); err != nil {
		return aisearch.FleetSnapshot{}, fmt.Errorf("finish provider-description snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return aisearch.FleetSnapshot{}, fmt.Errorf("commit provider-description snapshot: %w", err)
	}
	return aisearch.FleetSnapshot{ID: snapshotID, Watermark: watermark}, nil
}

func (s *ProviderDescriptionFleetSource) CurrentFleetPointCount(ctx context.Context) (uint64, error) {
	descriptors, err := loadActiveProviderDescriptors(ctx, s.db)
	if err != nil {
		return 0, err
	}
	profiles := routing.ProviderProfilesForFleet(descriptors)
	var count uint64
	for _, profile := range profiles {
		segments := len(routing.ProviderDescriptionSegments(profile))
		if math.MaxUint64-count < uint64(segments) {
			return 0, errors.New("provider-description point count overflows")
		}
		count += uint64(segments)
	}
	return count, nil
}

func (s *ProviderDescriptionFleetSource) PinFleetSnapshot(ctx context.Context, snapshotID string) (aisearch.FleetSnapshot, error) {
	var snapshot aisearch.FleetSnapshot
	var recipeID, digest string
	err := s.db.QueryRowContext(ctx, `SELECT snapshot_id, recipe_id, watermark, digest FROM provider_description_fleet_snapshots WHERE snapshot_id = ?`, strings.TrimSpace(snapshotID)).Scan(&snapshot.ID, &recipeID, &snapshot.Watermark, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return aisearch.FleetSnapshot{}, fmt.Errorf("provider-description snapshot %q is not present in the owner registry", snapshotID)
	}
	if err != nil {
		return aisearch.FleetSnapshot{}, fmt.Errorf("read provider-description snapshot: %w", err)
	}
	if err := s.checkRecipe(recipeID); err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	profiles, err := s.snapshotProfiles(ctx, snapshot.ID)
	if err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	canonical := make([][]byte, 0, len(profiles))
	for _, profile := range profiles {
		raw, err := json.Marshal(profile)
		if err != nil {
			return aisearch.FleetSnapshot{}, fmt.Errorf("encode pinned provider profile: %w", err)
		}
		canonical = append(canonical, raw)
	}
	verifiedDigest, err := digestProviderProfiles(canonical)
	if err != nil {
		return aisearch.FleetSnapshot{}, err
	}
	if digest == "" || digest != verifiedDigest {
		return aisearch.FleetSnapshot{}, errors.New("provider-description snapshot profile digest mismatch")
	}
	return snapshot, nil
}

func (s *ProviderDescriptionFleetSource) ReadFleetSnapshotPage(ctx context.Context, recipeID string, snapshot aisearch.FleetSnapshot, cursor string, limit int) ([]aisearch.FleetPoint, string, bool, error) {
	if err := s.checkRecipe(recipeID); err != nil {
		return nil, cursor, false, err
	}
	if limit < 1 || limit > 1024 {
		return nil, cursor, false, errors.New("provider-description snapshot page limit must be between 1 and 1024")
	}
	pinned, err := s.PinFleetSnapshot(ctx, snapshot.ID)
	if err != nil || pinned.Watermark != snapshot.Watermark {
		return nil, cursor, false, errors.New("provider-description snapshot identity or watermark changed")
	}
	offset, err := parseSourceOffset(cursor)
	if err != nil {
		return nil, cursor, false, err
	}
	profiles, err := s.snapshotProfiles(ctx, snapshot.ID)
	if err != nil {
		return nil, cursor, false, err
	}
	points := make([]aisearch.FleetPoint, 0, limit)
	var seen int
	for _, profile := range profiles {
		for index, segment := range routing.ProviderDescriptionSegments(profile) {
			if seen < offset {
				seen++
				continue
			}
			if len(points) == limit {
				next, _ := json.Marshal(offset + len(points))
				return points, string(next), false, nil
			}
			point, err := s.point(ctx, recipeID, profile, index, segment)
			if err != nil {
				return nil, cursor, false, err
			}
			points = append(points, point)
			seen++
		}
	}
	next := strconv.Itoa(offset + len(points))
	return points, next, true, nil
}

func (s *ProviderDescriptionFleetSource) ReadFleetChanges(ctx context.Context, recipeID, snapshotWatermark, after string, limit int) ([]aisearch.FleetChange, string, error) {
	if err := s.checkRecipe(recipeID); err != nil {
		return nil, after, err
	}
	if limit < 1 || limit > 1024 {
		return nil, after, errors.New("provider-description change page limit must be between 1 and 1024")
	}
	start, err := parseSequence(snapshotWatermark)
	if err != nil {
		return nil, after, err
	}
	last, err := parseSequence(after)
	if err != nil {
		return nil, after, err
	}
	if last < start {
		return nil, after, errors.New("provider-description cursor precedes its pinned snapshot")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT sequence, before_profile, after_profile FROM provider_description_changes WHERE sequence > ? ORDER BY sequence LIMIT ?`, last, limit)
	if err != nil {
		return nil, after, fmt.Errorf("read provider-description change feed: %w", err)
	}
	defer rows.Close()
	changes := make([]aisearch.FleetChange, 0)
	next := last
	for rows.Next() {
		var sequence int64
		var before, current sql.NullString
		if err := rows.Scan(&sequence, &before, &current); err != nil {
			return nil, after, fmt.Errorf("scan provider-description change: %w", err)
		}
		change := aisearch.FleetChange{Sequence: strconv.FormatInt(sequence, 10)}
		if before.Valid {
			profile, ok, err := profileFromStoredJSON(before.String)
			if err != nil {
				return nil, after, err
			}
			if ok {
				change.Deletes = profilePointIDs(recipeID, profile)
			}
		}
		if current.Valid {
			profile, ok, err := profileFromStoredJSON(current.String)
			if err != nil {
				return nil, after, err
			}
			if ok {
				segments := routing.ProviderDescriptionSegments(profile)
				for index, segment := range segments {
					point, err := s.point(ctx, recipeID, profile, index, segment)
					if err != nil {
						return nil, after, err
					}
					change.Upserts = append(change.Upserts, point)
				}
			}
		}
		changes = append(changes, change)
		if sequence <= next {
			return nil, after, errors.New("provider-description change feed sequence is not strictly increasing")
		}
		next = sequence
	}
	if err := rows.Err(); err != nil {
		return nil, after, fmt.Errorf("iterate provider-description change feed: %w", err)
	}
	return changes, strconv.FormatInt(next, 10), nil
}

func (s *ProviderDescriptionFleetSource) CompareFleetSequence(_ context.Context, before, after string) (int, error) {
	a, err := parseSequence(before)
	if err != nil {
		return 0, err
	}
	b, err := parseSequence(after)
	if err != nil {
		return 0, err
	}
	if a < b {
		return -1, nil
	}
	if a > b {
		return 1, nil
	}
	return 0, nil
}

func (s *ProviderDescriptionFleetSource) FleetWatermark(ctx context.Context) (string, error) {
	return currentProviderChangeWatermark(ctx, s.db)
}

func (s *ProviderDescriptionFleetSource) ExpectedFleetPointCount(ctx context.Context, recipeID, snapshotID, watermark string) (uint64, error) {
	if err := s.checkRecipe(recipeID); err != nil {
		return 0, err
	}
	pinned, err := s.PinFleetSnapshot(ctx, snapshotID)
	if err != nil {
		return 0, err
	}
	target, err := parseSequence(watermark)
	if err != nil {
		return 0, err
	}
	base, err := parseSequence(pinned.Watermark)
	if err != nil || target < base {
		return 0, errors.New("expected point count watermark precedes the source snapshot")
	}
	current, err := s.FleetWatermark(ctx)
	if err != nil {
		return 0, err
	}
	ceiling, _ := parseSequence(current)
	if target > ceiling {
		return 0, errors.New("expected point count watermark is beyond the current owner feed")
	}
	profiles, err := s.snapshotProfiles(ctx, snapshotID)
	if err != nil {
		return 0, err
	}
	state := make(map[string]routing.ProviderProfile, len(profiles))
	for _, profile := range profiles {
		state[profile.ProviderID] = profile
	}
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id, after_profile FROM provider_description_changes WHERE sequence > ? AND sequence <= ? ORDER BY sequence`, base, target)
	if err != nil {
		return 0, fmt.Errorf("read owner changes for point-count validation: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var providerID string
		var after sql.NullString
		if err := rows.Scan(&providerID, &after); err != nil {
			return 0, err
		}
		if !after.Valid {
			delete(state, providerID)
			continue
		}
		profile, ok, err := profileFromStoredJSON(after.String)
		if err != nil {
			return 0, err
		}
		if ok {
			state[profile.ProviderID] = profile
		} else {
			delete(state, providerID)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var count uint64
	for _, profile := range state {
		segments := len(routing.ProviderDescriptionSegments(profile))
		if math.MaxUint64-count < uint64(segments) {
			return 0, errors.New("provider-description point count overflows")
		}
		count += uint64(segments)
	}
	return count, nil
}

func (s *ProviderDescriptionFleetSource) AcquireFleetCutover(ctx context.Context, expected string) (aisearch.FleetCutoverFence, error) {
	release, err := ownerregistry.AcquireProviderDescriptionWrite(ctx)
	if err != nil {
		return nil, err
	}
	watermark, err := s.FleetWatermark(ctx)
	if err != nil {
		release()
		return nil, err
	}
	if watermark != expected {
		release()
		return nil, errors.New("provider registry changed before the Search Hub fleet cutover fence was acquired")
	}
	return &providerFleetCutoverFence{source: s, release: release}, nil
}

type providerFleetCutoverFence struct {
	source  *ProviderDescriptionFleetSource
	release func()
	once    sync.Once
}

func (f *providerFleetCutoverFence) CurrentWatermark(ctx context.Context) (string, error) {
	return f.source.FleetWatermark(ctx)
}

func (f *providerFleetCutoverFence) Release(context.Context) error {
	f.once.Do(f.release)
	return nil
}

func (s *ProviderDescriptionFleetSource) point(ctx context.Context, recipeID string, profile routing.ProviderProfile, segmentIndex int, segment string) (aisearch.FleetPoint, error) {
	vector, err := routing.EmbedProviderDescription(ctx, s.encoder, segment)
	if err != nil {
		return aisearch.FleetPoint{}, fmt.Errorf("embed provider-description segment %q: %w", profile.ProviderID, err)
	}
	if len(vector) != s.dim {
		return aisearch.FleetPoint{}, fmt.Errorf("provider-description encoder returned %d dimensions; recipe requires %d", len(vector), s.dim)
	}
	dense := make([]float32, len(vector))
	for index, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > math.MaxFloat32 {
			return aisearch.FleetPoint{}, fmt.Errorf("provider-description encoder returned a value outside finite float32 range for %q", profile.ProviderID)
		}
		dense[index] = float32(value)
	}
	return aisearch.FleetPoint{
		ID: providerFleetPointID(recipeID, profile.ProviderID, segmentIndex), Vector: dense,
		Payload: map[string]any{"source_id": profile.ProviderID, "provider_id": profile.ProviderID, "embedding_recipe_id": recipeID, "segment_index": segmentIndex},
	}, nil
}

func (s *ProviderDescriptionFleetSource) snapshotProfiles(ctx context.Context, snapshotID string) ([]routing.ProviderProfile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT profile_json FROM provider_description_fleet_snapshot_profiles WHERE snapshot_id = ? ORDER BY provider_id`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("read provider-description snapshot profiles: %w", err)
	}
	defer rows.Close()
	profiles := make([]routing.ProviderProfile, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var profile routing.ProviderProfile
		if err := json.Unmarshal(raw, &profile); err != nil {
			return nil, fmt.Errorf("decode provider-description snapshot profile: %w", err)
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *ProviderDescriptionFleetSource) checkRecipe(recipeID string) error {
	if recipeID != s.recipe.ID() {
		return errors.New("provider-description fleet recipe does not match the verified owner encoder")
	}
	return nil
}

func loadActiveProviderDescriptors(ctx context.Context, query interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]*registryv1.ProviderDescriptor, error) {
	rows, err := query.QueryContext(ctx, `SELECT descriptor FROM providers WHERE state = ? ORDER BY provider_id`, int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE))
	if err != nil {
		return nil, fmt.Errorf("read active Search Hub provider descriptors: %w", err)
	}
	defer rows.Close()
	descriptors := make([]*registryv1.ProviderDescriptor, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		descriptor := &registryv1.ProviderDescriptor{}
		if err := protojson.Unmarshal([]byte(raw), descriptor); err != nil {
			return nil, fmt.Errorf("decode Search Hub provider descriptor: %w", err)
		}
		descriptors = append(descriptors, descriptor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return descriptors, nil
}

func profileFromStoredJSON(raw string) (routing.ProviderProfile, bool, error) {
	var profile routing.ProviderProfile
	if err := json.Unmarshal([]byte(raw), &profile); err != nil {
		return routing.ProviderProfile{}, false, fmt.Errorf("decode provider-description change profile: %w", err)
	}
	if strings.TrimSpace(profile.ProviderID) == "" || strings.TrimSpace(profile.Type) == "" {
		return routing.ProviderProfile{}, false, errors.New("provider-description change profile lacks a stable provider identity")
	}
	return profile, true, nil
}

func profilePointIDs(recipeID string, profile routing.ProviderProfile) []string {
	segments := routing.ProviderDescriptionSegments(profile)
	ids := make([]string, len(segments))
	for index := range segments {
		ids[index] = providerFleetPointID(recipeID, profile.ProviderID, index)
	}
	return ids
}

func providerFleetPointID(recipeID, providerID string, index int) string {
	return aisearch.FleetPointID(recipeID, providerID, index)
}

func randomSnapshotID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate provider-description snapshot ID: %w", err)
	}
	return "provider-profile-snapshot-" + hex.EncodeToString(value[:]), nil
}

func digestProviderProfiles(canonical [][]byte) (string, error) {
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode provider-description snapshot digest input: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func parseSourceOffset(cursor string) (int, error) {
	if strings.TrimSpace(cursor) == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(cursor)
	if err != nil || value < 0 {
		return 0, errors.New("provider-description snapshot cursor is invalid")
	}
	return value, nil
}

func parseSequence(value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed < 0 {
		return 0, errors.New("provider-description change sequence is invalid")
	}
	return parsed, nil
}

func currentProviderChangeWatermark(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (string, error) {
	var sequence int64
	if err := query.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) FROM provider_description_changes`).Scan(&sequence); err != nil {
		return "", fmt.Errorf("read Search Hub provider change watermark: %w", err)
	}
	return strconv.FormatInt(sequence, 10), nil
}

var _ = protojson.MarshalOptions{}
