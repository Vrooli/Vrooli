package vector

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	aisearch "github.com/vrooli/ai-go/search"
)

var (
	ErrGenerationOperationConflict = errors.New("Source Ledger generation operation identity conflict")
	ErrGenerationCursorConflict    = errors.New("Source Ledger generation cursor changed concurrently")
	ErrGenerationNotReady          = errors.New("Source Ledger generation is not ready for this transition")
	ErrGenerationCatchUpRequired   = errors.New("Source Ledger generation requires catch-up before cutover")
)

const (
	vectorDatasetFacets          = "journal.facets"
	vectorDatasetSummaries       = "forest.summaries"
	vectorGenerationBatch        = 64
	vectorGenerationPeakFloor    = int64(16 << 20)
	vectorGenerationPayloadBound = int64(4 << 10)
	// catchUpPageLimit bounds one catch-up; admission reserves this many full
	// pages, not today's backlog, because changes keep arriving meanwhile.
	catchUpPageLimit = 128
	// catalogPeakBytes is the headroom of a catalog-only write.
	catalogPeakBytes = int64(64 << 10)
	// collectTarget is a collection's recorded generation: every eligible one.
	collectTarget = "all-eligible"
)

var generationDatasets = []string{vectorDatasetFacets, vectorDatasetSummaries}

const activeGenerationQuery = `SELECT g.generation_id,g.recipe_id,g.dimensions FROM vector_generation_active a JOIN vector_generations g ON g.generation_id=a.generation_id WHERE a.singleton=1 AND a.generation_id<>'' AND g.state='active'`

// GenerationRecipeProvider is the owner identity source. Stage resolves the
// current owner-approved recipe; resume, catch-up and rollback resolve the
// persisted recipe by ID. No API request can supply or construct a recipe.
type GenerationRecipeProvider interface {
	ResolveCurrentRecipe(context.Context) (aisearch.EmbeddingRecipe, error)
	ResolveRecipe(context.Context, string) (aisearch.EmbeddingRecipe, error)
}

// GenerationDocumentEmbedder must embed for the exact immutable recipe ID.
// Production implementations must route through the approved inference owner
// and return only after that owner has rechecked model rights/runtime identity.
type GenerationDocumentEmbedder interface {
	EmbedDocument(context.Context, string, string) ([]float64, error)
}

// GenerationDiskAdmission is the owner seam for a bounded disk job lease.
// Complete(false) must preserve the lease for explicit recovery when effects
// may remain; successful physical writes release only after Complete(true).
type GenerationDiskAdmission interface {
	Admit(context.Context, string, string, string, int64) (string, error)
	Complete(context.Context, string, string, bool) error
}

type GenerationPeakEstimator interface {
	Estimate(int, uint64) (int64, error)
}

type SQLiteGenerationOptions struct {
	Now            func() time.Time
	NewID          func() string
	RollbackWindow time.Duration
	BatchSize      int
}

type SQLiteGenerationBackend struct {
	db             *sql.DB
	recipes        GenerationRecipeProvider
	embedder       GenerationDocumentEmbedder
	disk           GenerationDiskAdmission
	peak           GenerationPeakEstimator
	now            func() time.Time
	newID          func() string
	rollbackWindow time.Duration
	batchSize      int
}

// SQLiteGenerationServing is the read-only serving view over the active
// generation. Callers read the pointer and its points in one SQLite read
// transaction, so they never combine vectors from two recipes. An empty active
// pointer means legacy serving is still in effect.
type SQLiteGenerationServing struct{ db generationReadDB }

type generationReadDB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type ActiveGenerationVectors struct {
	GenerationID string
	RecipeID     string
	Dimensions   int
	Active       bool
	Vectors      map[string][]float64
}

type sqlQueryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func NewSQLiteGenerationServing(db generationReadDB) (*SQLiteGenerationServing, error) {
	if db == nil {
		return nil, errors.New("Source Ledger generation serving requires its owner database")
	}
	return &SQLiteGenerationServing{db: db}, nil
}

// ReadActiveVectors returns vectors keyed by source row ID for one dataset.
// When a generation is active, missing points stay missing: serving never
// falls back to a legacy vector that may have a different recipe or dimension.
func (s *SQLiteGenerationServing) ReadActiveVectors(ctx context.Context, dataset string) (ActiveGenerationVectors, error) {
	if s == nil || s.db == nil {
		return ActiveGenerationVectors{}, errors.New("Source Ledger generation serving is unavailable")
	}
	if !slices.Contains(generationDatasets, dataset) {
		return ActiveGenerationVectors{}, fmt.Errorf("unsupported Source Ledger vector dataset %q", dataset)
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ActiveGenerationVectors{}, err
	}
	defer tx.Rollback()
	out, err := activeGeneration(ctx, tx)
	if err != nil || !out.Active {
		return out, err
	}
	out.Vectors = make(map[string][]float64)
	err = visitGenerationPoints(ctx, tx, out.GenerationID, dataset, func(sourceID string, blob []byte) error {
		vector, err := decodeGenerationVector(blob, out.Dimensions)
		if err != nil {
			return fmt.Errorf("read active Source Ledger generation vector %s/%s: %w", dataset, sourceID, err)
		}
		out.Vectors[sourceID] = vector
		return nil
	})
	if err != nil {
		return ActiveGenerationVectors{}, err
	}
	if err := tx.Commit(); err != nil {
		return ActiveGenerationVectors{}, err
	}
	return out, nil
}

func (s *SQLiteGenerationServing) ActiveRecipeID(ctx context.Context) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, errors.New("Source Ledger generation serving is unavailable")
	}
	active, err := activeGeneration(ctx, s.db)
	return active.RecipeID, active.Active, err
}

// activeGeneration reads the serving pointer; none means legacy serving.
func activeGeneration(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (ActiveGenerationVectors, error) {
	var out ActiveGenerationVectors
	err := q.QueryRowContext(ctx, activeGenerationQuery).Scan(&out.GenerationID, &out.RecipeID, &out.Dimensions)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ActiveGenerationVectors{}, nil
	case err != nil:
		return ActiveGenerationVectors{}, err
	}
	out.Active = true
	return out, nil
}

func visitGenerationPoints(ctx context.Context, q sqlQueryer, generationID, dataset string, visit func(sourceID string, blob []byte) error) error {
	rows, err := q.QueryContext(ctx, `SELECT source_id,vector_blob FROM vector_generation_points WHERE generation_id=? AND dataset_id=? ORDER BY source_id`, generationID, dataset)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var sourceID string
		var blob []byte
		if err := rows.Scan(&sourceID, &blob); err != nil {
			return err
		}
		if err := visit(sourceID, blob); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return rows.Close()
}

func decodeGenerationVector(blob []byte, dimensions int) ([]float64, error) {
	vector, err := Decode(blob)
	if err == nil {
		err = validateGenerationVector(vector, dimensions)
	}
	return vector, err
}

type SQLiteGeneration struct {
	ID                string
	RecipeID          string
	Dimensions        int
	State             string
	SnapshotID        string
	SnapshotWatermark int64
	CatchupWatermark  int64
	SnapshotCursor    int64
	PredecessorID     string
	RollbackExpires   time.Time
	Protected         bool
	Version           int64
}

type generationSource struct {
	Index    int64
	Dataset  string
	SourceID string
	Text     string
	Scope    string
	Present  bool
}

type generationChange struct {
	Sequence  int64
	Dataset   string
	SourceID  string
	Operation string
}

type generationEncodedPoint struct {
	source generationSource
	vector []float64
}

func NewSQLiteGenerationBackend(db *sql.DB, recipes GenerationRecipeProvider, embedder GenerationDocumentEmbedder, disk GenerationDiskAdmission, estimator GenerationPeakEstimator, options SQLiteGenerationOptions) (*SQLiteGenerationBackend, error) {
	if db == nil || recipes == nil || embedder == nil || disk == nil || estimator == nil {
		return nil, errors.New("Source Ledger SQLite generations require database, owner recipe provider, embedder, disk admission, and peak estimator")
	}
	options = options.withDefaults()
	return &SQLiteGenerationBackend{
		db: db, recipes: recipes, embedder: embedder, disk: disk, peak: estimator,
		now: options.Now, newID: options.NewID, rollbackWindow: options.RollbackWindow,
		batchSize: options.BatchSize,
	}, nil
}

func (o SQLiteGenerationOptions) withDefaults() SQLiteGenerationOptions {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.NewID == nil {
		o.NewID = func() string { return uuid.NewString() }
	}
	if o.RollbackWindow <= 0 {
		o.RollbackWindow = 24 * time.Hour
	}
	if o.BatchSize <= 0 || o.BatchSize > 1024 {
		o.BatchSize = vectorGenerationBatch
	}
	return o
}

// SQLiteGenerationPeakEstimate bounds the new vectors, SQLite index/WAL
// amplification, and the durable snapshot ID list. It uses a source-owned
// count and recipe dimensions, never caller-supplied bytes.
type SQLiteGenerationPeakEstimate struct{}

func (SQLiteGenerationPeakEstimate) Estimate(dimensions int, pointCount uint64) (int64, error) {
	if dimensions <= 0 || pointCount > math.MaxInt64 {
		return 0, errors.New("Source Ledger generation estimate requires positive dimensions and a bounded point count")
	}
	perPoint := int64(dimensions)*4 + vectorGenerationPayloadBound + 256
	if perPoint <= 0 || perPoint > math.MaxInt64/4 {
		return 0, errors.New("Source Ledger generation per-point estimate overflows")
	}
	perPoint *= 4
	count := int64(pointCount)
	if count > (math.MaxInt64-vectorGenerationPeakFloor)/perPoint {
		return 0, errors.New("Source Ledger generation peak estimate overflows")
	}
	return vectorGenerationPeakFloor + count*perPoint, nil
}

// operationIdentity: replaying an operation ID with any other field conflicts.
type operationIdentity struct {
	kind, generationID, recipeID, requestHash string
}

// generationLease is one admitted disk job. Outcomes use a detached context so
// a cancelled request still reports them.
type generationLease struct {
	ctx         context.Context
	disk        GenerationDiskAdmission
	operationID string
	receipt     string
	peakBytes   int64
	step        string
}

// fail keeps the lease: the operation may have left durable partial effects,
// so its headroom stays reserved until recovery completes the operation.
func (l generationLease) fail(err error) error {
	_ = l.disk.Complete(l.ctx, l.operationID, l.receipt, false)
	return err
}

// abandon releases the lease of an operation that wrote nothing.
func (l generationLease) abandon(err error) error {
	_ = l.disk.Complete(l.ctx, l.operationID, l.receipt, true)
	return err
}

func (l generationLease) release() error {
	if err := l.disk.Complete(l.ctx, l.operationID, l.receipt, true); err != nil {
		return fmt.Errorf("release successful Source Ledger %s admission: %w", l.step, err)
	}
	return nil
}

func (b *SQLiteGenerationBackend) admit(ctx context.Context, operationID, targetID, recipeID string, peakBytes int64, step string) (generationLease, error) {
	receipt, err := b.disk.Admit(ctx, operationID, targetID, recipeID, peakBytes)
	if err != nil {
		return generationLease{}, fmt.Errorf("admit Source Ledger generation %s: %w", step, err)
	}
	if strings.TrimSpace(receipt) == "" {
		return generationLease{}, fmt.Errorf("Source Ledger %s admission returned no durable receipt", step)
	}
	return generationLease{
		ctx: context.WithoutCancel(ctx), disk: b.disk, operationID: operationID,
		receipt: receipt, peakBytes: peakBytes, step: step,
	}, nil
}

func (b *SQLiteGenerationBackend) admitPoints(ctx context.Context, operationID string, want operationIdentity, dimensions int, pointCount uint64, step string) (generationLease, error) {
	peakBytes, err := b.peak.Estimate(dimensions, pointCount)
	if err != nil {
		return generationLease{}, err
	}
	return b.admit(ctx, operationID, want.generationID, want.recipeID, peakBytes, step)
}

// completeOperation releases the lease, then marks the operation complete.
func (b *SQLiteGenerationBackend) completeOperation(ctx context.Context, lease generationLease) error {
	if err := lease.release(); err != nil {
		return err
	}
	return b.finishOperation(ctx, lease.operationID, lease.receipt)
}

func (b *SQLiteGenerationBackend) loaded(ctx context.Context, generationID string, err error) (SQLiteGeneration, error) {
	if err != nil {
		return SQLiteGeneration{}, err
	}
	return b.Load(ctx, generationID)
}

func (b *SQLiteGenerationBackend) Stage(ctx context.Context, operationID string) (SQLiteGeneration, error) {
	if b == nil || b.db == nil || anyBlank(operationID) {
		return SQLiteGeneration{}, errors.New("Source Ledger stage requires its backend and stable operation ID")
	}
	recipe, err := b.recipes.ResolveCurrentRecipe(ctx)
	if err != nil {
		return SQLiteGeneration{}, fmt.Errorf("resolve owner-approved Source Ledger recipe: %w", err)
	}
	return b.stageWithRecipe(ctx, operationID, recipe)
}

func (b *SQLiteGenerationBackend) stageWithRecipe(ctx context.Context, operationID string, recipe aisearch.EmbeddingRecipe) (SQLiteGeneration, error) {
	if err := recipe.Verify(); err != nil {
		return SQLiteGeneration{}, fmt.Errorf("verify Source Ledger recipe: %w", err)
	}
	input, err := recipe.Input()
	if err != nil {
		return SQLiteGeneration{}, err
	}
	genID := generationIDForOperation(operationID)
	want := operationIdentity{"stage", genID, recipe.ID(), generationRequestDigest("stage", operationID, recipe.ID())}
	found, done, err := matchOperation(ctx, b.db, operationID, want)
	if err != nil || done {
		return b.loaded(ctx, genID, err)
	}
	lease, err := b.startStage(ctx, operationID, want, recipe, input.Dimensions, found)
	if err != nil {
		return SQLiteGeneration{}, err
	}
	if err := b.stageSnapshot(ctx, genID, recipe.ID(), input.Dimensions); err != nil {
		return SQLiteGeneration{}, lease.fail(err)
	}
	return b.loaded(ctx, genID, b.completeOperation(ctx, lease))
}

func generationIDForOperation(operationID string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("source-ledger-generation:"+operationID)).String()
}

// startStage, under the writer mutex, pins a new source snapshot or re-admits
// an interrupted build of the existing one.
func (b *SQLiteGenerationBackend) startStage(ctx context.Context, operationID string, want operationIdentity, recipe aisearch.EmbeddingRecipe, dimensions int, operationExists bool) (generationLease, error) {
	tx, err := b.lockedTx(ctx)
	if err != nil {
		return generationLease{}, err
	}
	defer tx.Rollback()
	var state, snapshotID string
	err = tx.QueryRowContext(ctx, `SELECT state,snapshot_id FROM vector_generations WHERE generation_id=?`, want.generationID).Scan(&state, &snapshotID)
	var lease generationLease
	switch {
	case errors.Is(err, sql.ErrNoRows):
		lease, err = b.createStage(ctx, tx, operationID, want, recipe, dimensions, operationExists)
	case err != nil:
		return generationLease{}, err
	case state == "building" || state == "candidate":
		lease, err = b.resumeStage(ctx, tx, operationID, want, dimensions, snapshotID)
	default:
		return generationLease{}, ErrGenerationNotReady
	}
	if err != nil {
		return generationLease{}, err
	}
	if err := tx.Commit(); err != nil {
		return generationLease{}, lease.fail(err)
	}
	return lease, nil
}

func (b *SQLiteGenerationBackend) resumeStage(ctx context.Context, tx *sql.Tx, operationID string, want operationIdentity, dimensions int, snapshotID string) (generationLease, error) {
	var count uint64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_snapshot_items WHERE snapshot_id=?`, snapshotID).Scan(&count); err != nil {
		return generationLease{}, err
	}
	lease, err := b.admitPoints(ctx, operationID, want, dimensions, count, "stage resume")
	if err != nil {
		return generationLease{}, err
	}
	if err := markRunning(ctx, tx, lease); err != nil {
		return generationLease{}, lease.fail(err)
	}
	return lease, nil
}

func (b *SQLiteGenerationBackend) createStage(ctx context.Context, tx *sql.Tx, operationID string, want operationIdentity, recipe aisearch.EmbeddingRecipe, dimensions int, operationExists bool) (generationLease, error) {
	snapshotID := b.newID()
	watermark, err := currentChangeWatermark(ctx, tx)
	if err != nil {
		return generationLease{}, err
	}
	var sourceCount uint64
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM facet_texts)+(SELECT COUNT(*) FROM summaries)`).Scan(&sourceCount); err != nil {
		return generationLease{}, err
	}
	lease, err := b.admitPoints(ctx, operationID, want, dimensions, sourceCount, "stage")
	if err != nil {
		return generationLease{}, err
	}
	if operationExists {
		err = markRunning(ctx, tx, lease)
	} else if _, err = tx.ExecContext(ctx, `INSERT INTO vector_generation_operations(operation_id,operation,generation_id,recipe_id,request_sha256,state,receipt_id,peak_bytes,started_at)
VALUES(?,'stage',?,?,?,'running',?,?,?)`, operationID, want.generationID, want.recipeID, want.requestHash, lease.receipt, lease.peakBytes, b.timestamp()); err != nil {
		err = errors.Join(ErrGenerationOperationConflict, err)
	}
	if err != nil {
		return generationLease{}, lease.abandon(err)
	}
	now := b.timestamp()
	err = execAll(ctx, tx,
		sqlStatement{`INSERT INTO vector_generation_snapshots(snapshot_id,watermark,created_at) VALUES(?,?,?)`, []any{snapshotID, watermark, now}},
		sqlStatement{`INSERT INTO vector_generation_snapshot_items(snapshot_id,dataset_id,source_id)
SELECT ?,?,id FROM facet_texts ORDER BY id`, []any{snapshotID, vectorDatasetFacets}},
		sqlStatement{`INSERT INTO vector_generation_snapshot_items(snapshot_id,dataset_id,source_id)
SELECT ?,?,id FROM summaries ORDER BY id`, []any{snapshotID, vectorDatasetSummaries}},
		sqlStatement{`INSERT INTO vector_generations(generation_id,recipe_id,recipe_canonical,dimensions,state,snapshot_id,snapshot_watermark,catchup_watermark,created_at,updated_at)
VALUES(?,?,?,?,'building',?,?,?, ?,?)`, []any{want.generationID, want.recipeID, recipe.Canonical(), dimensions, snapshotID, watermark, watermark, now, now}},
	)
	if err != nil {
		return generationLease{}, lease.fail(err)
	}
	return lease, nil
}

func (b *SQLiteGenerationBackend) stageSnapshot(ctx context.Context, generationID, recipeID string, dimensions int) error {
	for {
		generation, err := b.Load(ctx, generationID)
		if err != nil {
			return err
		}
		if generation.State == "candidate" {
			return nil
		}
		page, err := b.snapshotPage(ctx, generation, b.batchSize)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			_, err := b.db.ExecContext(ctx, `UPDATE vector_generations SET state='candidate',version=version+1,updated_at=? WHERE generation_id=? AND state='building'`, b.timestamp(), generationID)
			return err
		}
		points, err := b.embedPage(ctx, recipeID, dimensions, page)
		if err != nil {
			return err
		}
		if err := b.saveSnapshotPage(ctx, generation, page, points); err != nil && !errors.Is(err, ErrGenerationCursorConflict) {
			return err
		}
	}
}

// embedPage re-resolves the exact recipe before each embedding, so a rights or
// runtime change stops the page instead of mixing identities.
func (b *SQLiteGenerationBackend) embedPage(ctx context.Context, recipeID string, dimensions int, sources []generationSource) ([]generationEncodedPoint, error) {
	points := make([]generationEncodedPoint, 0, len(sources))
	for _, source := range sources {
		if !source.Present || strings.TrimSpace(source.Text) == "" {
			continue
		}
		if _, err := b.exactRecipe(ctx, recipeID, "Source Ledger recipe identity changed while embedding a generation"); err != nil {
			return nil, err
		}
		vector, err := b.embedder.EmbedDocument(ctx, recipeID, source.Text)
		if err != nil {
			return nil, fmt.Errorf("embed Source Ledger %s source %s: %w", source.Dataset, source.SourceID, err)
		}
		if err := validateGenerationVector(vector, dimensions); err != nil {
			return nil, fmt.Errorf("validate Source Ledger %s source %s vector: %w", source.Dataset, source.SourceID, err)
		}
		points = append(points, generationEncodedPoint{source: source, vector: vector})
	}
	return points, nil
}

func (b *SQLiteGenerationBackend) snapshotPage(ctx context.Context, generation SQLiteGeneration, limit int) ([]generationSource, error) {
	rows, err := b.db.QueryContext(ctx, `SELECT i.item_index,i.dataset_id,i.source_id,
CASE WHEN i.dataset_id=? THEN ft.id IS NOT NULL ELSE s.id IS NOT NULL END AS present,
CASE WHEN i.dataset_id=? THEN COALESCE(ft.text,'') ELSE COALESCE(s.body,'') END AS source_text,
CASE WHEN i.dataset_id=? THEN COALESCE(e.scope,'') ELSE COALESCE(s.scope,'') END AS source_scope
FROM vector_generation_snapshot_items i
LEFT JOIN facet_texts ft ON i.dataset_id=? AND ft.id=i.source_id
LEFT JOIN entries e ON e.id=ft.entry_id
LEFT JOIN summaries s ON i.dataset_id=? AND s.id=i.source_id
WHERE i.snapshot_id=? AND i.item_index>? ORDER BY i.item_index LIMIT ?`,
		vectorDatasetFacets, vectorDatasetFacets, vectorDatasetFacets, vectorDatasetFacets, vectorDatasetSummaries, generation.SnapshotID, generation.SnapshotCursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []generationSource
	for rows.Next() {
		var item generationSource
		if err := rows.Scan(&item.Index, &item.Dataset, &item.SourceID, &item.Present, &item.Text, &item.Scope); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (b *SQLiteGenerationBackend) saveSnapshotPage(ctx context.Context, generation SQLiteGeneration, page []generationSource, points []generationEncodedPoint) error {
	tx, err := b.fencedTx(ctx, `SELECT snapshot_cursor FROM vector_generations WHERE generation_id=? AND state='building'`, generation.ID, generation.SnapshotCursor)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, point := range points {
		if _, err := tx.ExecContext(ctx, `INSERT INTO vector_generation_points(generation_id,dataset_id,source_id,scope,vector_blob,source_sequence)
VALUES(?,?,?,?,?,0) ON CONFLICT(generation_id,dataset_id,source_id) DO UPDATE SET scope=excluded.scope,vector_blob=excluded.vector_blob`, generation.ID, point.source.Dataset, point.source.SourceID, point.source.Scope, Encode(point.vector)); err != nil {
			return err
		}
	}
	last := page[len(page)-1].Index
	if _, err := tx.ExecContext(ctx, `UPDATE vector_generations SET snapshot_cursor=?,version=version+1,updated_at=? WHERE generation_id=? AND snapshot_cursor=?`, last, b.timestamp(), generation.ID, generation.SnapshotCursor); err != nil {
		return err
	}
	return tx.Commit()
}

// CatchUp applies the owner's ordered change feed through a bounded series of
// idempotent SQLite transactions. Repeating the same operation ID returns its
// durable completed result; each persisted sequence is applied at most once.
func (b *SQLiteGenerationBackend) CatchUp(ctx context.Context, operationID, generationID string) (SQLiteGeneration, error) {
	if anyBlank(operationID, generationID) {
		return SQLiteGeneration{}, errors.New("Source Ledger catch-up requires operation and generation identity")
	}
	generation, err := b.Load(ctx, generationID)
	if err != nil {
		return SQLiteGeneration{}, err
	}
	if _, err := b.exactRecipe(ctx, generation.RecipeID, "Source Ledger catch-up cannot resolve the generation's exact recipe"); err != nil {
		return SQLiteGeneration{}, err
	}
	peakBytes, err := b.peak.Estimate(generation.Dimensions, uint64(b.batchSize)*catchUpPageLimit)
	if err != nil {
		return SQLiteGeneration{}, err
	}
	want := operationIdentity{"catch_up", generationID, generation.RecipeID, generationRequestDigest("catch_up", operationID, generationID)}
	lease, done, err := b.beginOperation(ctx, operationID, want, peakBytes, "catch-up")
	if err != nil || done {
		return b.loaded(ctx, generationID, err)
	}
	generation, remaining, err := b.applyPendingChanges(ctx, generationID)
	if err != nil {
		return SQLiteGeneration{}, lease.fail(err)
	}
	if remaining != 0 {
		if err := lease.release(); err != nil {
			return SQLiteGeneration{}, err
		}
		return generation, ErrGenerationCatchUpRequired
	}
	return b.loaded(ctx, generationID, b.completeOperation(ctx, lease))
}

func (b *SQLiteGenerationBackend) applyPendingChanges(ctx context.Context, generationID string) (SQLiteGeneration, uint64, error) {
	for range catchUpPageLimit {
		generation, err := b.Load(ctx, generationID)
		if err != nil {
			return SQLiteGeneration{}, 0, err
		}
		changes, sources, err := b.changePage(ctx, generation.CatchupWatermark)
		if err != nil {
			return SQLiteGeneration{}, 0, err
		}
		if len(changes) == 0 {
			break
		}
		points, err := b.embedPage(ctx, generation.RecipeID, generation.Dimensions, sources)
		if err != nil {
			return SQLiteGeneration{}, 0, err
		}
		if err := b.applyChangePage(ctx, generation, changes, points); err != nil && !errors.Is(err, ErrGenerationCursorConflict) {
			return SQLiteGeneration{}, 0, err
		}
	}
	generation, err := b.Load(ctx, generationID)
	if err != nil {
		return SQLiteGeneration{}, 0, err
	}
	remaining, err := b.pendingChangeCount(ctx, generation.CatchupWatermark)
	return generation, remaining, err
}

// Promote atomically switches the one active SQLite generation after taking
// the same mutex row used by source change triggers. That lock makes the final
// feed watermark check and active-pointer update one ordered cutover.
func (b *SQLiteGenerationBackend) Promote(ctx context.Context, operationID, generationID string) (SQLiteGeneration, error) {
	return b.switchActive(ctx, operationID, generationID, false)
}

// Rollback resolves the predecessor from the active catalog, catches it up
// using its stored recipe, then atomically restores it while retaining the
// current generation for the new rollback window.
func (b *SQLiteGenerationBackend) Rollback(ctx context.Context, operationID string) (SQLiteGeneration, error) {
	if anyBlank(operationID) {
		return SQLiteGeneration{}, errors.New("Source Ledger rollback requires a stable operation ID")
	}
	var predecessorID string
	operation, err := b.Operation(ctx, operationID)
	switch {
	case err == nil:
		if operation.Operation != "rollback" || anyBlank(operation.GenerationID) {
			return SQLiteGeneration{}, ErrGenerationOperationConflict
		}
		if operation.State == "complete" {
			return b.loaded(ctx, operation.GenerationID, nil)
		}
		// The active pointer can commit before capacity lease completion. On a
		// retry, use the durable target instead of deriving its inverse from the
		// newly active generation's predecessor.
		predecessorID = operation.GenerationID
	case errors.Is(err, sql.ErrNoRows):
		var activeID string
		if err := b.db.QueryRowContext(ctx, `SELECT generation_id FROM vector_generation_active WHERE singleton=1`).Scan(&activeID); err != nil {
			return SQLiteGeneration{}, err
		}
		if activeID == "" {
			return SQLiteGeneration{}, ErrGenerationNotReady
		}
		if err := b.db.QueryRowContext(ctx, `SELECT predecessor_id FROM vector_generations WHERE generation_id=? AND state='active'`, activeID).Scan(&predecessorID); err != nil {
			return SQLiteGeneration{}, err
		}
		if predecessorID == "" {
			return SQLiteGeneration{}, errors.New("Source Ledger active generation has no retained rollback predecessor")
		}
	default:
		return SQLiteGeneration{}, err
	}
	predecessor, err := b.Load(ctx, predecessorID)
	if err != nil {
		return SQLiteGeneration{}, err
	}
	if !predecessor.RollbackExpires.IsZero() && b.now().UTC().After(predecessor.RollbackExpires) {
		return SQLiteGeneration{}, errors.New("Source Ledger predecessor rollback window expired")
	}
	catchupID := operationID + ":rollback-catch-up"
	if _, err := b.CatchUp(ctx, catchupID, predecessorID); err != nil {
		return SQLiteGeneration{}, fmt.Errorf("catch up rollback predecessor: %w", err)
	}
	return b.switchActive(ctx, operationID, predecessorID, true)
}

// switchSource is the state promotion (false) or rollback (true) activates.
var switchSource = map[bool]string{false: "candidate", true: "retained"}

func (b *SQLiteGenerationBackend) switchActive(ctx context.Context, operationID, generationID string, rollback bool) (SQLiteGeneration, error) {
	if anyBlank(operationID, generationID) {
		return SQLiteGeneration{}, errors.New("Source Ledger generation switch requires operation and generation identity")
	}
	// Validate also re-resolves and verifies the exact owner recipe.
	validation, err := b.Validate(ctx, generationID)
	if err != nil {
		return SQLiteGeneration{}, fmt.Errorf("validate Source Ledger generation before switch: %w", err)
	}
	if !validation.ReadyForPromotion {
		return SQLiteGeneration{}, fmt.Errorf("%w: generation %s is not ready (missing=%d orphaned=%d invalid=%d pending_changes=%d watermark=%d/%d state=%s)", ErrGenerationCatchUpRequired, generationID, validation.MissingPoints, validation.OrphanedPoints, validation.InvalidPoints, validation.PendingChanges, validation.CatchupWatermark, validation.CurrentWatermark, validation.State)
	}
	kind := map[bool]string{true: "rollback", false: "promote"}[rollback]
	want := operationIdentity{kind, generationID, validation.RecipeID, generationRequestDigest(kind, operationID, generationID)}
	lease, done, err := b.beginOperation(ctx, operationID, want, catalogPeakBytes, "pointer switch")
	if err != nil || done {
		return b.loaded(ctx, generationID, err)
	}
	if err := b.switchPointer(ctx, generationID, rollback); err != nil {
		return SQLiteGeneration{}, lease.fail(err)
	}
	return b.loaded(ctx, generationID, b.completeOperation(ctx, lease))
}

// switchPointer fences the feed watermark and moves the active pointer under
// the writer mutex, retaining the displaced generation for rollback.
func (b *SQLiteGenerationBackend) switchPointer(ctx context.Context, generationID string, rollback bool) error {
	tx, err := b.lockedTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	watermark, err := currentChangeWatermark(ctx, tx)
	if err != nil {
		return err
	}
	var state, previousID string
	var caughtUp int64
	if err := tx.QueryRowContext(ctx, `SELECT state,catchup_watermark FROM vector_generations WHERE generation_id=?`, generationID).Scan(&state, &caughtUp); err != nil {
		return err
	}
	if caughtUp != watermark {
		return ErrGenerationCatchUpRequired
	}
	if state != switchSource[rollback] && state != "active" {
		return ErrGenerationNotReady
	}
	if err := tx.QueryRowContext(ctx, `SELECT generation_id FROM vector_generation_active WHERE singleton=1`).Scan(&previousID); err != nil {
		return err
	}
	if previousID == generationID {
		return tx.Commit()
	}
	now := b.now().UTC()
	stamp := now.Format(time.RFC3339Nano)
	// Promotion retains only an active predecessor; rollback retains it always.
	err = execAll(ctx, tx,
		sqlStatement{`UPDATE vector_generations SET state='retained',rollback_expires_at=?,version=version+1,updated_at=? WHERE generation_id=? AND (state='active' OR ?)`, []any{now.Add(b.rollbackWindow).Format(time.RFC3339Nano), stamp, previousID, rollback}},
		sqlStatement{`UPDATE vector_generations SET state='active',predecessor_id=?,rollback_expires_at='',version=version+1,updated_at=? WHERE generation_id=?`, []any{previousID, stamp, generationID}},
		sqlStatement{`UPDATE vector_generation_active SET generation_id=?,version=version+1 WHERE singleton=1`, []any{generationID}},
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

const collectableGenerations = `SELECT generation_id FROM vector_generations WHERE state IN ('retained','rolled_back') AND protected=0 AND rollback_expires_at<>'' AND rollback_expires_at<=?
AND generation_id<>(SELECT generation_id FROM vector_generation_active WHERE singleton=1)`

// Collect removes only expired, unprotected, non-active vector data. Catalog
// records and source evidence remain as audit history. One cutoff bounds the
// estimate, deletion and catalog update alike.
func (b *SQLiteGenerationBackend) Collect(ctx context.Context, operationID string) (int64, error) {
	if anyBlank(operationID) {
		return 0, errors.New("Source Ledger collection requires a stable operation ID")
	}
	want := operationIdentity{"collect", collectTarget, "cleanup", generationRequestDigest("collect", operationID)}
	done, err := b.ensureOperation(ctx, operationID, want)
	if err != nil || done {
		return 0, err
	}
	cutoff := b.timestamp()
	peakBytes, err := b.collectionPeak(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	lease, err := b.admit(ctx, operationID, want.generationID, want.recipeID, peakBytes, "cleanup")
	if err != nil {
		return 0, err
	}
	removed, err := b.collectExpired(ctx, cutoff)
	if err != nil {
		return 0, lease.fail(err)
	}
	return removed, b.completeOperation(ctx, lease)
}

func (b *SQLiteGenerationBackend) collectionPeak(ctx context.Context, cutoff string) (int64, error) {
	var pointCount, vectorBytes int64
	if err := b.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(length(vector_blob)),0) FROM vector_generation_points WHERE generation_id IN (`+collectableGenerations+`)`, cutoff).Scan(&pointCount, &vectorBytes); err != nil {
		return 0, err
	}
	if pointCount < 0 || vectorBytes < 0 || pointCount > (math.MaxInt64-catalogPeakBytes)/2048 || vectorBytes > (math.MaxInt64-catalogPeakBytes-pointCount*2048)/2 {
		return 0, errors.New("Source Ledger cleanup peak estimate overflows")
	}
	return catalogPeakBytes + pointCount*2048 + vectorBytes*2, nil
}

func (b *SQLiteGenerationBackend) collectExpired(ctx context.Context, cutoff string) (int64, error) {
	tx, err := b.lockedTx(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM vector_generation_points WHERE generation_id IN (`+collectableGenerations+`)`, cutoff)
	if err != nil {
		return 0, err
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vector_generations SET state='deleted',version=version+1,updated_at=? WHERE generation_id IN (`+collectableGenerations+`)`, cutoff, cutoff); err != nil {
		return 0, err
	}
	return removed, tx.Commit()
}

func (b *SQLiteGenerationBackend) Load(ctx context.Context, generationID string) (SQLiteGeneration, error) {
	var out SQLiteGeneration
	var expires string
	var protected int
	err := b.db.QueryRowContext(ctx, `SELECT generation_id,recipe_id,dimensions,state,snapshot_id,snapshot_watermark,catchup_watermark,snapshot_cursor,predecessor_id,rollback_expires_at,protected,version FROM vector_generations WHERE generation_id=?`, generationID).Scan(&out.ID, &out.RecipeID, &out.Dimensions, &out.State, &out.SnapshotID, &out.SnapshotWatermark, &out.CatchupWatermark, &out.SnapshotCursor, &out.PredecessorID, &expires, &protected, &out.Version)
	if err != nil {
		return SQLiteGeneration{}, err
	}
	out.Protected = protected != 0
	if expires != "" {
		out.RollbackExpires, err = time.Parse(time.RFC3339Nano, expires)
		if err != nil {
			return SQLiteGeneration{}, err
		}
	}
	return out, nil
}

func (b *SQLiteGenerationBackend) Active(ctx context.Context) (SQLiteGeneration, bool, error) {
	var id string
	if err := b.db.QueryRowContext(ctx, `SELECT generation_id FROM vector_generation_active WHERE singleton=1`).Scan(&id); err != nil {
		return SQLiteGeneration{}, false, err
	}
	if id == "" {
		return SQLiteGeneration{}, false, nil
	}
	generation, err := b.Load(ctx, id)
	return generation, err == nil, err
}

// matchOperation refuses a recorded operation whose identity differs from want.
func matchOperation(ctx context.Context, q sqlQueryer, operationID string, want operationIdentity) (found, done bool, err error) {
	var stored operationIdentity
	var state string
	err = q.QueryRowContext(ctx, `SELECT operation,generation_id,recipe_id,request_sha256,state FROM vector_generation_operations WHERE operation_id=?`, operationID).Scan(&stored.kind, &stored.generationID, &stored.recipeID, &stored.requestHash, &state)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, false, nil
	case err != nil:
		return false, false, err
	case stored != want:
		return true, false, ErrGenerationOperationConflict
	}
	return true, state == "complete", nil
}

// ensureOperation records or matches operationID under the writer mutex.
func (b *SQLiteGenerationBackend) ensureOperation(ctx context.Context, operationID string, want operationIdentity) (bool, error) {
	tx, err := b.lockedTx(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	found, done, err := matchOperation(ctx, tx, operationID, want)
	if err != nil {
		return false, err
	}
	if !found {
		if _, err := tx.ExecContext(ctx, `INSERT INTO vector_generation_operations(operation_id,operation,generation_id,recipe_id,request_sha256,state,started_at)
VALUES(?,?,?,?,?,'planned',?)`, operationID, want.kind, want.generationID, want.recipeID, want.requestHash, b.timestamp()); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return done, nil
}

// beginOperation admits a catch-up or switch once; a completed replay reports
// done before admission. Otherwise the caller owns the running lease.
func (b *SQLiteGenerationBackend) beginOperation(ctx context.Context, operationID string, want operationIdentity, peakBytes int64, step string) (generationLease, bool, error) {
	if _, done, err := matchOperation(ctx, b.db, operationID, want); err != nil || done {
		return generationLease{}, done, err
	}
	lease, err := b.admit(ctx, operationID, want.generationID, want.recipeID, peakBytes, step)
	if err != nil {
		return generationLease{}, false, err
	}
	done, err := b.ensureOperation(ctx, operationID, want)
	if err != nil || done {
		return generationLease{}, done, lease.abandon(err)
	}
	if err := markRunning(ctx, b.db, lease); err != nil {
		return generationLease{}, false, lease.abandon(err)
	}
	return lease, false, nil
}

func markRunning(ctx context.Context, e sqlQueryer, lease generationLease) error {
	_, err := e.ExecContext(ctx, `UPDATE vector_generation_operations SET state='running',receipt_id=?,peak_bytes=?,error='' WHERE operation_id=?`, lease.receipt, lease.peakBytes, lease.operationID)
	return err
}

func (b *SQLiteGenerationBackend) finishOperation(ctx context.Context, operationID, receipt string) error {
	_, err := b.db.ExecContext(ctx, `UPDATE vector_generation_operations SET state='complete',receipt_id=?,completed_at=? WHERE operation_id=?`, receipt, b.timestamp(), operationID)
	return err
}

// lockedTx takes the mutex row the source-change triggers also take.
func (b *SQLiteGenerationBackend) lockedTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vector_generation_mutex SET version=version+1 WHERE singleton=1`); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

// fencedTx returns a locked transaction only while the cursor equals want.
func (b *SQLiteGenerationBackend) fencedTx(ctx context.Context, query, generationID string, want int64) (*sql.Tx, error) {
	tx, err := b.lockedTx(ctx)
	if err != nil {
		return nil, err
	}
	var cursor int64
	err = tx.QueryRowContext(ctx, query, generationID).Scan(&cursor)
	if err == nil && cursor != want {
		err = ErrGenerationCursorConflict
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

type sqlStatement struct {
	query string
	args  []any
}

func execAll(ctx context.Context, e sqlQueryer, statements ...sqlStatement) error {
	for _, statement := range statements {
		if _, err := e.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func currentChangeWatermark(ctx context.Context, q sqlQueryer) (int64, error) {
	var watermark int64
	err := q.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0) FROM vector_generation_changes`).Scan(&watermark)
	return watermark, err
}

func (b *SQLiteGenerationBackend) pendingChangeCount(ctx context.Context, after int64) (uint64, error) {
	var count uint64
	err := b.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_changes WHERE sequence>?`, after).Scan(&count)
	return count, err
}

// changePage reads one feed page and the current row behind each upsert.
func (b *SQLiteGenerationBackend) changePage(ctx context.Context, after int64) ([]generationChange, []generationSource, error) {
	rows, err := b.db.QueryContext(ctx, `SELECT sequence,dataset_id,source_id,operation FROM vector_generation_changes WHERE sequence>? ORDER BY sequence LIMIT ?`, after, b.batchSize)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var changes []generationChange
	for rows.Next() {
		var change generationChange
		if err := rows.Scan(&change.Sequence, &change.Dataset, &change.SourceID, &change.Operation); err != nil {
			return nil, nil, err
		}
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	sources := make([]generationSource, 0, len(changes))
	for _, change := range changes {
		if change.Operation != "upsert" {
			continue
		}
		source, err := b.loadSource(ctx, change)
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, source)
	}
	return changes, sources, nil
}

var sourceRowQueries = map[string]string{
	vectorDatasetFacets:    `SELECT ft.text,e.scope FROM facet_texts ft JOIN entries e ON e.id=ft.entry_id WHERE ft.id=?`,
	vectorDatasetSummaries: `SELECT body,scope FROM summaries WHERE id=?`,
}

func (b *SQLiteGenerationBackend) loadSource(ctx context.Context, change generationChange) (generationSource, error) {
	source := generationSource{Index: change.Sequence, Dataset: change.Dataset, SourceID: change.SourceID}
	query, ok := sourceRowQueries[change.Dataset]
	if !ok {
		return source, errors.New("Source Ledger vector change has unknown dataset")
	}
	err := b.db.QueryRowContext(ctx, query, change.SourceID).Scan(&source.Text, &source.Scope)
	if errors.Is(err, sql.ErrNoRows) {
		return source, nil
	}
	source.Present = err == nil
	return source, err
}

func (b *SQLiteGenerationBackend) applyChangePage(ctx context.Context, generation SQLiteGeneration, changes []generationChange, points []generationEncodedPoint) error {
	tx, err := b.fencedTx(ctx, `SELECT catchup_watermark FROM vector_generations WHERE generation_id=? AND state IN ('building','candidate','active','retained')`, generation.ID, generation.CatchupWatermark)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	pointBySource := make(map[string]generationEncodedPoint, len(points))
	for _, point := range points {
		pointBySource[point.source.Dataset+"\x00"+point.source.SourceID] = point
	}
	for _, change := range changes {
		point, ok := pointBySource[change.Dataset+"\x00"+change.SourceID]
		if change.Operation == "delete" || !ok {
			if _, err := tx.ExecContext(ctx, `DELETE FROM vector_generation_points WHERE generation_id=? AND dataset_id=? AND source_id=?`, generation.ID, change.Dataset, change.SourceID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO vector_generation_points(generation_id,dataset_id,source_id,scope,vector_blob,source_sequence)
VALUES(?,?,?,?,?,?) ON CONFLICT(generation_id,dataset_id,source_id) DO UPDATE SET scope=excluded.scope,vector_blob=excluded.vector_blob,source_sequence=excluded.source_sequence`, generation.ID, change.Dataset, change.SourceID, point.source.Scope, Encode(point.vector), change.Sequence); err != nil {
			return err
		}
	}
	last := changes[len(changes)-1].Sequence
	if _, err := tx.ExecContext(ctx, `UPDATE vector_generations SET catchup_watermark=?,version=version+1,updated_at=? WHERE generation_id=? AND catchup_watermark=?`, last, b.timestamp(), generation.ID, generation.CatchupWatermark); err != nil {
		return err
	}
	return tx.Commit()
}

func (b *SQLiteGenerationBackend) timestamp() string {
	return b.now().UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
}

func (b *SQLiteGenerationBackend) exactRecipe(ctx context.Context, recipeID, refusal string) (aisearch.EmbeddingRecipe, error) {
	recipe, err := b.recipes.ResolveRecipe(ctx, recipeID)
	return requireRecipe(recipe, err, recipeID, refusal)
}

// requireRecipe accepts only an error-free, intact resolution of recipeID.
func requireRecipe(recipe aisearch.EmbeddingRecipe, err error, recipeID, refusal string) (aisearch.EmbeddingRecipe, error) {
	if err != nil || recipe.ID() != recipeID || recipe.Verify() != nil {
		return aisearch.EmbeddingRecipe{}, errors.Join(errors.New(refusal), err)
	}
	return recipe, nil
}

func anyBlank(values ...string) bool {
	return slices.ContainsFunc(values, func(value string) bool { return strings.TrimSpace(value) == "" })
}

func generationRequestDigest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte{0})
		hash.Write([]byte(part))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validateGenerationVector(vector []float64, dimensions int) error {
	if dimensions <= 0 || len(vector) != dimensions {
		return fmt.Errorf("embedding returned %d dimensions; generation recipe requires %d", len(vector), dimensions)
	}
	for _, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("embedding contains a non-finite value")
		}
	}
	return nil
}
