package report

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	intcorpus "audio-tools/internal/corpus"
	localdb "audio-tools/internal/database"
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/experiment/evaldeps"
	exprecipe "audio-tools/internal/experiment/recipe"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/testutil/db"
	"audio-tools/internal/testutil/mocks"

	apidb "github.com/vrooli/api-core/database"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

type memoryBlobs struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (b *memoryBlobs) Put(_ context.Context, key string, data []byte, _ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.m[key] = append([]byte(nil), data...)
	return nil
}

func (b *memoryBlobs) Get(_ context.Context, key string) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	data, ok := b.m[key]
	if !ok {
		return nil, errors.New("blob missing")
	}
	return append([]byte(nil), data...), nil
}

func (b *memoryBlobs) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.m, key)
	return nil
}

func experimentCorpus(t *testing.T) *intcorpus.Service {
	t.Helper()
	database := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(intcorpus.Schema)); err != nil {
		t.Fatalf("apply corpus schema: %v", err)
	}
	clk := mocks.NewFakeClock(time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC))
	return intcorpus.NewService(intcorpus.NewSQLiteRepository(database, clk), &memoryBlobs{m: map[string][]byte{}}, clk)
}

func TestRunExperimentReport_DefaultRecipeExecutesCorpusThroughProvider(t *testing.T) {
	ctx := context.Background()
	corpus := experimentCorpus(t)
	clip, err := corpus.CreateClip(ctx, intcorpus.CreateClipInput{
		Audio: []byte{0, 0, 0, 0}, ReferenceText: "expected", DurationMs: 1,
		SampleRateHz: 16_000, Format: "pcm_s16le", Source: intcorpus.SourceScripted,
	})
	if err != nil {
		t.Fatalf("create clip: %v", err)
	}
	provider := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	provider.Result = &sttchain.Result{Text: "expected", Tier: sttchain.TierLocal}
	report, realized, err := RunExperimentReport(ctx, evaldeps.New(nil, corpus, func(string) sttchain.Provider { return provider }, sttpkg.Defaults(), nil), corpus, nil, nil, &experimentv1.ExperimentRecipe{
		ClipIds: []string{clip.ID}, ChunkMs: 100,
		Strategies: []*evalv1.EvalStrategy{{Kind: "batch"}},
	}, nil)
	if err != nil {
		t.Fatalf("run experiment report: %v", err)
	}
	if realized["phase"] != "default" || len(report.PerStrategy) != 1 || report.PerStrategy[0].WER != 0 {
		t.Fatalf("unexpected report=%#v realized=%#v", report, realized)
	}
}

func TestRunExperimentReport_LongFormMaterializesAndEvaluates(t *testing.T) {
	ctx := context.Background()
	corpus := experimentCorpus(t)
	clip, err := corpus.CreateClip(ctx, intcorpus.CreateClipInput{
		Audio: make([]byte, exprecipe.CanonicalSampleRate*2), ReferenceText: "expected", DurationMs: 1_000,
		SampleRateHz: exprecipe.CanonicalSampleRate, Format: "pcm_s16le", Source: intcorpus.SourceScripted,
	})
	if err != nil {
		t.Fatalf("create clip: %v", err)
	}
	provider := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	provider.Result = &sttchain.Result{Text: "expected", Tier: sttchain.TierLocal}
	recipe := &experimentv1.ExperimentRecipe{
		ClipIds: []string{clip.ID}, ChunkMs: 100, Seed: 7,
		Strategies: []*evalv1.EvalStrategy{{Kind: "batch"}},
		LongForm:   &experimentv1.LongFormRecipe{Enabled: true, TargetDurationSeconds: 1},
	}
	report, realized, err := RunExperimentReport(ctx, evaldeps.New(nil, corpus, func(string) sttchain.Provider { return provider }, sttpkg.Defaults(), nil), corpus, nil, nil, recipe, nil)
	if err != nil {
		t.Fatalf("run long-form report: %v", err)
	}
	if realized["phase"] != "long_form" || !realized["long_form"].(bool) || recipe.RealizedDurationMs == 0 || len(recipe.RealizedClipIds) == 0 {
		t.Fatalf("long-form realization = %#v recipe=%#v", realized, recipe)
	}
	if len(report.PerStrategy) != 1 || report.PerStrategy[0].WER != 0 {
		t.Fatalf("long-form report = %#v", report)
	}
}

func TestRunExperimentReport_AugmentationEvaluatesEveryCondition(t *testing.T) {
	ctx := context.Background()
	corpus := experimentCorpus(t)
	clip, err := corpus.CreateClip(ctx, intcorpus.CreateClipInput{
		Audio: make([]byte, exprecipe.CanonicalSampleRate*2), ReferenceText: "expected", DurationMs: 1_000,
		SampleRateHz: exprecipe.CanonicalSampleRate, Format: "pcm_s16le", Source: intcorpus.SourceScripted,
	})
	if err != nil {
		t.Fatalf("create clip: %v", err)
	}
	provider := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	provider.Result = &sttchain.Result{Text: "expected", Tier: sttchain.TierLocal}
	recipe := &experimentv1.ExperimentRecipe{
		ClipIds: []string{clip.ID}, ChunkMs: 100, Seed: 7,
		Strategies:   []*evalv1.EvalStrategy{{Kind: "batch"}},
		Augmentation: &experimentv1.AugmentationRecipe{NoiseTypes: []string{"white"}, SnrDb: []float64{12}},
	}
	report, realized, err := RunExperimentReport(ctx, evaldeps.New(nil, corpus, func(string) sttchain.Provider { return provider }, sttpkg.Defaults(), nil), corpus, nil, nil, recipe, nil)
	if err != nil {
		t.Fatalf("run augmented report: %v", err)
	}
	augmentation, ok := realized["augmentation"].(map[string]any)
	if !ok || augmentation["condition_count"] != 2 || len(recipe.RealizedAugmentationConditions) != 2 {
		t.Fatalf("augmentation realization = %#v recipe=%#v", realized, recipe.RealizedAugmentationConditions)
	}
	if len(report.PerStrategy) != 2 {
		t.Fatalf("augmented report rows = %#v", report.PerStrategy)
	}
}

func TestRunExperimentReport_SpeakerResourceAbsenceIsAnExplicitSkippedReport(t *testing.T) {
	ctx := context.Background()
	corpus := experimentCorpus(t)
	clip, err := corpus.CreateClip(ctx, intcorpus.CreateClipInput{
		Audio: make([]byte, exprecipe.CanonicalSampleRate*2), ReferenceText: "expected", DurationMs: 1_000,
		SampleRateHz: exprecipe.CanonicalSampleRate, Format: "pcm_s16le", Source: intcorpus.SourceScripted,
	})
	if err != nil {
		t.Fatalf("create clip: %v", err)
	}
	provider := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	provider.Result = &sttchain.Result{Text: "expected", Tier: sttchain.TierLocal}
	recipe := &experimentv1.ExperimentRecipe{
		ClipIds: []string{clip.ID}, ChunkMs: 100,
		Strategies: []*evalv1.EvalStrategy{{Kind: "batch"}},
		Speaker:    &experimentv1.SpeakerExperimentRecipe{TargetProfileId: "profile", ExtractionEnabled: true},
	}
	report, realized, err := RunExperimentReport(ctx, evaldeps.New(nil, corpus, func(string) sttchain.Provider { return provider }, sttpkg.Defaults(), nil), corpus, nil, nil, recipe, nil)
	if err != nil {
		t.Fatalf("run speaker degradation report: %v", err)
	}
	if len(report.PerStrategy) != 0 || len(report.Warnings) != 1 || report.Warnings[0].Code != "speaker_condition_skipped" {
		t.Fatalf("speaker degradation report = %#v", report)
	}
	if realized["speaker"].(map[string]any)["enabled"] != true || !recipe.RealizedSpeakerConditions[0].Skipped {
		t.Fatalf("speaker realization = %#v recipe=%#v", realized, recipe.RealizedSpeakerConditions)
	}
}

func TestEstimateClipSecondsRoundsAndSelectsRequestedClips(t *testing.T) {
	ctx := context.Background()
	corpus := experimentCorpus(t)
	first, err := corpus.CreateClip(ctx, intcorpus.CreateClipInput{Audio: []byte{1}, ReferenceText: "one", DurationMs: 1_100, SampleRateHz: 16_000, Format: "pcm_s16le", Source: intcorpus.SourceScripted})
	if err != nil {
		t.Fatalf("create first clip: %v", err)
	}
	if _, err := corpus.CreateClip(ctx, intcorpus.CreateClipInput{Audio: []byte{2}, ReferenceText: "two", DurationMs: 1_100, SampleRateHz: 16_000, Format: "pcm_s16le", Source: intcorpus.SourceScripted}); err != nil {
		t.Fatalf("create second clip: %v", err)
	}
	estimate := estimateClipSeconds(corpus)
	if got, err := estimate(ctx, nil); err != nil || got != 3 {
		t.Fatalf("all-clip estimate = %d, %v; want 3, nil", got, err)
	}
	if got, err := estimate(ctx, []string{first.ID}); err != nil || got != 2 {
		t.Fatalf("selected estimate = %d, %v; want 2, nil", got, err)
	}
	if _, err := estimate(ctx, []string{"missing"}); err == nil {
		t.Fatal("missing requested clip must return its repository error")
	}
	if estimateClipSeconds(nil) != nil {
		t.Fatal("nil corpus must disable estimation")
	}
}

func TestExperimentProgressAndWorkUnitsCoverDefaults(t *testing.T) {
	if newExperimentProgress(0, func(int, string) {}) != nil {
		t.Fatal("zero-work progress must be disabled")
	}
	var updates []string
	progress := newExperimentProgress(1, func(percent int, message string) {
		updates = append(updates, message)
		if percent != 90 {
			t.Fatalf("progress = %d, want 90", percent)
		}
	})
	opts := progress.options("", inteval.EvalOptions{})
	opts.Progress(inteval.EvalProgress{})
	if len(updates) != 1 || !strings.Contains(updates[0], "condition default eval: strategy strategy 0/0, clip clip 0/0") {
		t.Fatalf("progress message = %v", updates)
	}

	clips := []inteval.Clip{{}, {}}
	conditions := []speakerEvalCondition{{ID: "run"}, {ID: "skip", Skipped: true}}
	if got := experimentWorkUnits(clips, nil, nil, -1, conditions, true); got != 6 {
		t.Fatalf("default work units = %d, want 6", got)
	}
	if got := experimentWorkUnits(clips, []exprecipe.ConditionGroup{{Clips: clips}}, []*evalv1.EvalStrategy{{}, {}}, 1, conditions, true); got != 8 {
		t.Fatalf("augmented work units = %d, want 8", got)
	}
	if got := cellExperimentWorkUnits(clips, []*experimentv1.EvaluationCell{nil, {RepeatCount: 0}, {RepeatCount: 3}}); got != 6 {
		t.Fatalf("cell work units = %d, want 6", got)
	}
}

func TestExperimentHelperEdgesAreExplicit(t *testing.T) {
	if got := safetyThreshold(&experimentv1.ExperimentRecipe{}); got != int32(inteval.DefaultDroppedSpanThresholdWords) {
		t.Fatalf("default threshold = %d", got)
	}
	if got := safetyThreshold(&experimentv1.ExperimentRecipe{DroppedSpanThresholdWords: 9}); got != 9 {
		t.Fatalf("configured threshold = %d", got)
	}
	if got := conditionScope("prefix", ""); got != "prefix" {
		t.Fatalf("condition scope = %q", got)
	}
	if got := conditionScope("", "scope"); got != "scope" {
		t.Fatalf("condition scope = %q", got)
	}
	if got := conditionScope("prefix", "scope"); got != "prefix / scope" {
		t.Fatalf("condition scope = %q", got)
	}
	if warning, ok := longFormSourceDurationWarning(nil, nil); ok || warning.Code != "" {
		t.Fatalf("nil long form = %#v, %t", warning, ok)
	}
	if got := sourceDurationMs([]exprecipe.Clip{{PCM: make([]byte, exprecipe.CanonicalSampleRate*2)}}); got != 1000 {
		t.Fatalf("default-rate source duration = %d, want 1000", got)
	}
	if _, err := synthesizeCanonicalVoice(nil, nil)(context.Background(), "voice", "text"); err == nil {
		t.Fatal("unconfigured synthesis must fail")
	}
}
