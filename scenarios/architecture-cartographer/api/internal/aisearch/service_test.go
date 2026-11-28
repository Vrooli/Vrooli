package aisearch

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

type migrationSource struct {
	docs []pkg.SourceDoc
	err  error
}

func (s migrationSource) LoadAll(context.Context) ([]pkg.SourceDoc, error) { return s.docs, s.err }

type migrationEmbedder struct{ err error }

func (e migrationEmbedder) Embed(context.Context, string) ([]float64, error) {
	if e.err != nil {
		return nil, e.err
	}
	return []float64{1, 0}, nil
}
func (migrationEmbedder) Available(context.Context) bool { return true }

type migrationVectorStore struct {
	mu            sync.Mutex
	items         map[string]pkg.ScrollItem
	scrollErr     error
	entered       chan struct{}
	release       chan struct{}
	ensureEntered chan struct{}
	ensureRelease chan struct{}
	ensureErr     error
	ensureCalls   int
	ensureOnce    sync.Once
}

func newMigrationVectorStore() *migrationVectorStore {
	return &migrationVectorStore{items: make(map[string]pkg.ScrollItem)}
}

func (s *migrationVectorStore) EnsureCollection(ctx context.Context, _ pkg.CollectionSpec) error {
	s.mu.Lock()
	s.ensureCalls++
	entered, release, ensureErr := s.ensureEntered, s.ensureRelease, s.ensureErr
	s.mu.Unlock()
	if entered != nil {
		s.ensureOnce.Do(func() { close(entered) })
	}
	if release != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
		}
	}
	return ensureErr
}
func (s *migrationVectorStore) Upsert(_ context.Context, point pkg.Point) error {
	s.mu.Lock()
	s.items[point.ID] = pkg.ScrollItem{SourceID: point.ID}
	s.mu.Unlock()
	return nil
}
func (s *migrationVectorStore) SetPayload(context.Context, string, map[string]any) error { return nil }
func (s *migrationVectorStore) BatchDelete(_ context.Context, ids []string) error {
	s.mu.Lock()
	for _, id := range ids {
		delete(s.items, id)
	}
	s.mu.Unlock()
	return nil
}
func (*migrationVectorStore) Query(context.Context, pkg.HybridQuery) ([]pkg.SearchResult, error) {
	return nil, nil
}
func (s *migrationVectorStore) CountPoints(context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items), nil
}
func (s *migrationVectorStore) ScrollIDs(ctx context.Context) (map[string]pkg.ScrollItem, error) {
	if s.scrollErr != nil {
		return nil, s.scrollErr
	}
	if s.entered != nil {
		select {
		case <-s.entered:
		default:
			close(s.entered)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.release:
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make(map[string]pkg.ScrollItem, len(s.items))
	for id, item := range s.items {
		items[id] = item
	}
	return items, nil
}
func (*migrationVectorStore) Available(context.Context) bool { return true }

func (s *migrationVectorStore) collectionEnsures() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensureCalls
}

func newMigrationPackageService(source pkg.Source, store *migrationVectorStore, embedder ...migrationEmbedder) *pkg.Service {
	var selected migrationEmbedder
	if len(embedder) > 0 {
		selected = embedder[0]
	}
	reconciler := pkg.NewReconciler(selected, []pkg.SourceBinding{{
		Kind: "doc", Store: store, Source: source, Chunker: pkg.NewIdentityChunker(), IDPrefix: "migration-test:",
	}}, 1)
	return pkg.NewService(pkg.ServiceOptions{VectorStore: store, Reconciler: reconciler})
}

func testCartographerRecipe(t *testing.T, model string, dimensions int) (string, string) {
	t.Helper()
	recipe, err := pkg.NewEmbeddingRecipe(pkg.EmbeddingRecipeInput{
		SchemaVersion: "1", ModelID: model, ModelRevision: "revision-1", ArtifactSHA256: strings.Repeat("a", 64),
		TokenizerRevision: "tokenizer-1", Runtime: "test-runtime", RuntimeBuild: "build-1", Precision: "fp32",
		QueryInstruction: "none", DocumentInstruction: "none", Pooling: "mean", Normalization: "l2",
		Dimensions: dimensions, OutputDType: "float32", Truncation: "truncate-1", ChunkerVersion: "chunk-1", ComposerVersion: "cartographer-domain-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return recipe.ID(), string(recipe.Canonical())
}

func newMigrationService(shadowSvc *pkg.Service, shadowName string, jobID string) (*Service, *engine, *engine) {
	active := &engine{svc: pkg.NewService(pkg.ServiceOptions{VectorStore: newMigrationVectorStore()}), vectorStore: newMigrationVectorStore(), spec: pkg.CollectionSpec{Name: "live"}}
	shadow := &engine{svc: shadowSvc, vectorStore: newMigrationVectorStore(), spec: pkg.CollectionSpec{Name: shadowName}, shadowJobID: jobID}
	jobs := make(map[string]*pkg.Service)
	if jobID != "" {
		jobs[jobID] = shadowSvc
	}
	return &Service{eng: active, shadows: map[string]*engine{shadowName: shadow}, shadowJobs: jobs}, active, shadow
}

func awaitMigrationJob(t *testing.T, svc *pkg.Service, jobID, wantState string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := svc.ReindexStatus(jobID)
		if ok && job.State == wantState {
			return
		}
		if ok && (job.State == "succeeded" || job.State == "failed" || job.State == "cancelled") {
			t.Fatalf("reindex job ended %q, want %q", job.State, wantState)
		}
		runtime.Gosched()
	}
	t.Fatalf("reindex job %q did not reach %q", jobID, wantState)
}

func TestActivateShadowRequiresRecordedSuccessfulJobAndPreservesLiveEngine(t *testing.T) {
	t.Run("missing job", func(t *testing.T) {
		shadowSvc := pkg.NewService(pkg.ServiceOptions{VectorStore: newMigrationVectorStore()})
		service, active, _ := newMigrationService(shadowSvc, "candidate", "")
		if err := service.ActivateShadow(context.Background(), "candidate"); err == nil {
			t.Fatal("shadow without a recorded reindex job was activated")
		}
		if service.current() != active || service.retained != nil {
			t.Fatal("missing-job refusal changed the active engine or retained pointer")
		}
	})

	t.Run("running job", func(t *testing.T) {
		store := newMigrationVectorStore()
		store.entered, store.release = make(chan struct{}), make(chan struct{})
		shadowSvc := newMigrationPackageService(migrationSource{docs: []pkg.SourceDoc{{ID: "doc", Kind: "doc", Body: "candidate"}}}, store)
		job, err := shadowSvc.Reindex(context.Background(), "", false)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-store.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("shadow job did not enter its blocked reconciliation")
		}
		service, active, _ := newMigrationService(shadowSvc, "candidate", job.ID)
		if err := service.ActivateShadow(context.Background(), "candidate"); err == nil {
			t.Fatal("running shadow job was activated")
		}
		if service.current() != active || service.retained != nil {
			t.Fatal("running-job refusal changed the active engine or retained pointer")
		}
		close(store.release)
		awaitMigrationJob(t, shadowSvc, job.ID, "succeeded")
	})

	for _, tc := range []struct {
		name   string
		source pkg.Source
		store  *migrationVectorStore
	}{
		{name: "source read failure", source: migrationSource{err: errors.New("source unavailable")}, store: newMigrationVectorStore()},
		{name: "vector store read failure", source: migrationSource{docs: []pkg.SourceDoc{{ID: "doc", Kind: "doc", Body: "candidate"}}}, store: &migrationVectorStore{items: make(map[string]pkg.ScrollItem), scrollErr: errors.New("vector store unavailable")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shadowSvc := newMigrationPackageService(tc.source, tc.store)
			job, err := shadowSvc.Reindex(context.Background(), "", false)
			if err != nil {
				t.Fatal(err)
			}
			awaitMigrationJob(t, shadowSvc, job.ID, "failed")
			service, active, _ := newMigrationService(shadowSvc, "candidate", job.ID)
			if err := service.ActivateShadow(context.Background(), "candidate"); err == nil {
				t.Fatal("shadow with planning read failure was activated")
			}
			if service.current() != active || service.retained != nil {
				t.Fatal("planning read failure changed the active engine or retained pointer")
			}
		})
	}

	t.Run("succeeded job", func(t *testing.T) {
		shadowSvc := newMigrationPackageService(migrationSource{}, newMigrationVectorStore())
		job, err := shadowSvc.Reindex(context.Background(), "", false)
		if err != nil {
			t.Fatal(err)
		}
		awaitMigrationJob(t, shadowSvc, job.ID, "succeeded")
		service, active, shadow := newMigrationService(shadowSvc, "candidate", job.ID)
		if err := service.ActivateShadow(context.Background(), "candidate"); err != nil {
			t.Fatalf("succeeded shadow job refused: %v", err)
		}
		if service.current() != shadow || service.retained != active {
			t.Fatal("successful cutover did not atomically retain the prior active engine")
		}
		if service.lifecycleEpoch != 2 || shadow.lastActiveEpoch != 2 {
			t.Fatalf("cutover epoch=%d active epoch=%d, want 2", service.lifecycleEpoch, shadow.lastActiveEpoch)
		}
		if err := service.RollbackShadow(context.Background(), "live"); err != nil {
			t.Fatalf("rollback to retained engine: %v", err)
		}
		if service.current() != active || service.lifecycleEpoch != 3 || active.lastActiveEpoch != 3 {
			t.Fatalf("rollback did not advance pointer epoch: current=%p active=%p epoch=%d activation=%d", service.current(), active, service.lifecycleEpoch, active.lastActiveEpoch)
		}
	})
}

func TestShadowStageSerializesCollectionsAndRetriesIdempotently(t *testing.T) {
	store := newMigrationVectorStore()
	store.ensureEntered = make(chan struct{})
	store.ensureRelease = make(chan struct{})
	source := migrationSource{docs: []pkg.SourceDoc{{ID: "doc", Kind: "doc", Body: "candidate"}}}
	service := &Service{
		eng:        &engine{svc: pkg.NewService(pkg.ServiceOptions{VectorStore: newMigrationVectorStore()}), vectorStore: newMigrationVectorStore(), spec: pkg.CollectionSpec{Name: "live"}},
		shadowJobs: make(map[string]*pkg.Service), shadows: make(map[string]*engine), lifecycleEpoch: 1,
		buildShadow: func(policy pkg.EmbeddingPolicy, collection string) *engine {
			shadowService := newMigrationPackageService(source, store)
			return &engine{svc: shadowService, vectorStore: store, spec: pkg.CollectionSpec{
				Name: collection, Model: policy.Model, Role: policy.Role, DenseSize: policy.Dimensions, PolicySchemaVersion: policy.PolicySchemaVersion,
			}}
		},
	}
	recipeID, recipeCanonical := testCartographerRecipe(t, "model-v2", 768)
	request := ShadowRequest{Collection: "candidate-v2", Embedding: pkg.EmbeddingPolicy{Model: "model-v2", Role: "embedding.default", Dimensions: 768, PolicySchemaVersion: "v2"}, EmbeddingRecipeID: recipeID, EmbeddingRecipeCanonical: recipeCanonical}
	result := make(chan struct {
		job *pkg.ReindexJob
		err error
	}, 1)
	go func() {
		job, err := service.ReindexShadow(context.Background(), request)
		result <- struct {
			job *pkg.ReindexJob
			err error
		}{job: job, err: err}
	}()
	select {
	case <-store.ensureEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("shadow stage did not reach collection ensure")
	}
	if _, err := service.ReindexShadow(context.Background(), request); err == nil {
		t.Fatal("overlapping stage for one collection was accepted")
	}
	if got := store.collectionEnsures(); got != 1 {
		t.Fatalf("overlapping stage reached vector store %d times, want one", got)
	}
	close(store.ensureRelease)
	var staged struct {
		job *pkg.ReindexJob
		err error
	}
	select {
	case staged = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("shadow stage did not return after collection ensure")
	}
	if staged.err != nil || staged.job == nil {
		t.Fatalf("shadow stage job=%+v err=%v", staged.job, staged.err)
	}
	awaitMigrationJob(t, service.shadows[request.Collection].svc, staged.job.ID, "succeeded")
	retry, err := service.ReindexShadow(context.Background(), request)
	if err != nil || retry == nil || retry.ID != staged.job.ID {
		t.Fatalf("same shadow request did not return the recorded job: retry=%+v original=%+v err=%v", retry, staged.job, err)
	}
	if got := store.collectionEnsures(); got != 1 {
		t.Fatalf("idempotent retry re-ensured the shadow collection %d times", got)
	}
	conflict := request
	conflict.Embedding.Model = "different-model"
	if _, err := service.ReindexShadow(context.Background(), conflict); err == nil {
		t.Fatal("same collection accepted a conflicting recipe identity")
	}
	dryRun := request
	dryRun.Collection = "candidate-dry-run"
	dryRun.DryRun = true
	dryJob, err := service.ReindexShadow(context.Background(), dryRun)
	if err != nil || dryJob == nil {
		t.Fatalf("shadow dry-run job=%+v err=%v", dryJob, err)
	}
	if got := store.collectionEnsures(); got != 1 {
		t.Fatalf("dry-run created or ensured a physical collection; ensure calls=%d", got)
	}
	if err := service.ActivateShadow(context.Background(), dryRun.Collection); err == nil {
		t.Fatal("dry-run shadow was activated")
	}
	for _, entry := range service.GenerationCensus() {
		if entry.Collection == dryRun.Collection && (entry.LifecycleState != "shadow_dry_run" || entry.BuildJobID != dryJob.ID) {
			t.Fatalf("dry-run census lost its state or job identity: %+v", entry)
		}
	}
}

func TestShadowStageValidatesCanonicalRecipeBeforeStoreMutation(t *testing.T) {
	store := newMigrationVectorStore()
	builds := 0
	service := &Service{
		eng: &engine{svc: pkg.NewService(pkg.ServiceOptions{VectorStore: newMigrationVectorStore()}), vectorStore: newMigrationVectorStore(), spec: pkg.CollectionSpec{Name: "live"}},
		buildShadow: func(policy pkg.EmbeddingPolicy, collection string) *engine {
			builds++
			return &engine{svc: newMigrationPackageService(migrationSource{}, store), vectorStore: store, spec: pkg.CollectionSpec{
				Name: collection, DenseDistance: pkg.DefaultDenseDistance, Model: policy.Model, Role: policy.Role,
				DenseSize: policy.Dimensions, PolicySchemaVersion: policy.PolicySchemaVersion,
			}}
		},
		lifecycleEpoch: 1, shadows: make(map[string]*engine), shadowJobs: make(map[string]*pkg.Service),
	}
	recipeID, recipeCanonical := testCartographerRecipe(t, "model", 2)
	valid := ShadowRequest{Collection: "candidate", Embedding: pkg.EmbeddingPolicy{Model: "model", Role: "embedding.default", Dimensions: 2, PolicySchemaVersion: "v1"}, EmbeddingRecipeID: recipeID, EmbeddingRecipeCanonical: recipeCanonical}
	invalid := []ShadowRequest{
		{Collection: "candidate", Embedding: valid.Embedding},
		{Collection: "candidate", Embedding: valid.Embedding, EmbeddingRecipeID: recipeID, EmbeddingRecipeCanonical: `{ "schema_version":"1" }`},
		{Collection: "candidate", Embedding: valid.Embedding, EmbeddingRecipeID: recipeID, EmbeddingRecipeCanonical: recipeCanonical + " "},
		{Collection: "candidate", Embedding: valid.Embedding, EmbeddingRecipeID: "sha256:" + strings.Repeat("0", 64), EmbeddingRecipeCanonical: recipeCanonical},
	}
	for _, request := range invalid {
		if _, err := service.ReindexShadow(context.Background(), request); err == nil {
			t.Fatalf("invalid recipe request was accepted: %+v", request)
		}
	}
	if builds != 0 || store.collectionEnsures() != 0 || len(service.shadows) != 0 {
		t.Fatalf("invalid recipes reached the builder/store: builds=%d ensures=%d shadows=%d", builds, store.collectionEnsures(), len(service.shadows))
	}
	wrongModel := valid
	wrongModel.Embedding.Model = "other-model"
	if _, err := service.ReindexShadow(context.Background(), wrongModel); err == nil {
		t.Fatal("recipe whose model disagrees with the target was accepted")
	}
	wrongDimensions := valid
	wrongDimensions.Embedding.Dimensions = 3
	if _, err := service.ReindexShadow(context.Background(), wrongDimensions); err == nil {
		t.Fatal("recipe whose dimensions disagree with the target were accepted")
	}
	if builds != 0 || store.collectionEnsures() != 0 {
		t.Fatalf("mismatched recipes reached the builder/store: builds=%d ensures=%d", builds, store.collectionEnsures())
	}
}

func TestShadowLifecyclePersistsStageCutoverRestartAndRollback(t *testing.T) {
	path := t.TempDir() + "/embedding-lifecycle.json"
	source := migrationSource{docs: []pkg.SourceDoc{{ID: "doc", Kind: "doc", Body: "candidate"}}}
	makeService := func() *Service {
		store := newMigrationVectorStore()
		makeEngine := func(policy pkg.EmbeddingPolicy, collection string) *engine {
			spec := pkg.CollectionSpec{
				Name: collection, DenseDistance: pkg.DefaultDenseDistance, Model: policy.Model,
				Role: policy.Role, DenseSize: policy.Dimensions, PolicySchemaVersion: policy.PolicySchemaVersion,
			}
			service := newMigrationPackageService(source, store)
			return &engine{svc: service, vectorStore: store, spec: spec}
		}
		activePolicy := pkg.EmbeddingPolicy{Model: "live-model", Role: "embedding.default", Dimensions: 2, PolicySchemaVersion: "policy-v1"}
		active := makeEngine(activePolicy, "live")
		active.lastActiveEpoch = 1
		return &Service{
			eng: active, buildShadow: makeEngine, lifecyclePath: path,
			shadows: make(map[string]*engine), shadowJobs: make(map[string]*pkg.Service),
			lifecycleEpoch: 1,
		}
	}

	original := makeService()
	recipeID, recipeCanonical := testCartographerRecipe(t, "candidate-model", 2)
	request := ShadowRequest{
		Collection: "candidate", Scope: "", Embedding: pkg.EmbeddingPolicy{
			Model: "candidate-model", Role: "embedding.default", Dimensions: 2, PolicySchemaVersion: "policy-v2",
		},
		EmbeddingRecipeID: recipeID, EmbeddingRecipeCanonical: recipeCanonical,
	}
	job, err := original.ReindexShadow(context.Background(), request)
	if err != nil {
		t.Fatalf("stage candidate: %v", err)
	}
	awaitMigrationJob(t, original.shadows[request.Collection].svc, job.ID, "succeeded")
	if err := original.ActivateShadow(context.Background(), request.Collection); err != nil {
		t.Fatalf("cut over to candidate: %v", err)
	}

	restarted := makeService()
	if err := restarted.RestoreLifecycle(); err != nil {
		t.Fatalf("restore after process restart: %v", err)
	}
	if restarted.current().spec.Name != "candidate" || restarted.retained == nil || restarted.retained.spec.Name != "live" {
		t.Fatalf("restart did not recover active and retained generations: active=%+v retained=%+v", restarted.current().spec, restarted.retained)
	}
	entries := restarted.GenerationCensus()
	if len(entries) != 2 {
		t.Fatalf("restart census returned %d entries, want active + retained: %+v", len(entries), entries)
	}
	byCollection := make(map[string]GenerationCensusEntry, len(entries))
	for _, entry := range entries {
		byCollection[entry.Collection] = entry
		if entry.Durability != "durable_manifest" || entry.RestartRecovery != "recoverable" || entry.SentinelID == "" {
			t.Fatalf("recovered census must report durable pointer and expected sentinel identity: %+v", entry)
		}
	}
	if live := byCollection["live"]; live.RecipeIdentityState != "unknown" || live.RecipeID != "" {
		t.Fatalf("legacy live generation must remain unclaimed: %+v", live)
	}
	if candidate := byCollection["candidate"]; candidate.RecipeIdentityState != "verified_canonical" || candidate.RecipeID != recipeID {
		t.Fatalf("recovered candidate lost its verified recipe identity: %+v", candidate)
	}
	if err := restarted.RollbackShadow(context.Background(), "live"); err != nil {
		t.Fatalf("rollback after restart: %v", err)
	}
	if restarted.current().spec.Name != "live" || restarted.retained == nil || restarted.retained.spec.Name != "candidate" {
		t.Fatalf("rollback did not restore prior generation: active=%+v retained=%+v", restarted.current().spec, restarted.retained)
	}

	afterRollbackRestart := makeService()
	if err := afterRollbackRestart.RestoreLifecycle(); err != nil {
		t.Fatalf("restore after rollback restart: %v", err)
	}
	if afterRollbackRestart.current().spec.Name != "live" || afterRollbackRestart.retained == nil || afterRollbackRestart.retained.spec.Name != "candidate" {
		t.Fatalf("rollback pointer was not durable: active=%+v retained=%+v", afterRollbackRestart.current().spec, afterRollbackRestart.retained)
	}
}
