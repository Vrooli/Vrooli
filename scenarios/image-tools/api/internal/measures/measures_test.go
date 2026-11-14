package measures

import (
	"context"
	"strings"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
)

func newRecorder(t *testing.T) *Recorder {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return NewRecorder(d)
}

// TestRecordAndAggregate proves samples are recorded per op/model and that
// latency percentiles, throughput, and outcome counts are queryable.
func TestRecordAndAggregate(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)

	// 10 upscale runs with durations 100..1000ms, all succeeded.
	for i := 1; i <= 10; i++ {
		if err := r.Record(ctx, Sample{
			Operation:  "upscale",
			ModelID:    "real-esrgan",
			Tier:       "local-cpu",
			State:      "succeeded",
			DurationMS: int64(i * 100),
		}); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	// One failed resize.
	if err := r.Record(ctx, Sample{Operation: "resize", State: "failed", DurationMS: 5}); err != nil {
		t.Fatalf("record resize: %v", err)
	}

	stats, err := r.Stats(ctx, "")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(stats))
	}

	var up OpStat
	for _, s := range stats {
		if s.Operation == "upscale" {
			up = s
		}
	}
	if up.Count != 10 || up.Succeeded != 10 {
		t.Fatalf("upscale count/succeeded = %d/%d, want 10/10", up.Count, up.Succeeded)
	}
	// Nearest-rank p50 of 100..1000 (n=10) = rank 5 → 500; p95 = rank ceil(9.5)=10 → 1000.
	if up.LatencyP50MS != 500 {
		t.Fatalf("p50 = %d, want 500", up.LatencyP50MS)
	}
	if up.LatencyP95MS != 1000 {
		t.Fatalf("p95 = %d, want 1000", up.LatencyP95MS)
	}
}

// TestStatsFilterByOperation proves the operation filter narrows results.
func TestStatsFilterByOperation(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	_ = r.Record(ctx, Sample{Operation: "resize", State: "succeeded", DurationMS: 10})
	_ = r.Record(ctx, Sample{Operation: "crop", State: "succeeded", DurationMS: 20})

	stats, err := r.Stats(ctx, "crop")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 1 || stats[0].Operation != "crop" {
		t.Fatalf("expected only crop, got %+v", stats)
	}
}

func TestModelUsageSummaryUsesOnlySuccessfulDurableTraces(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	times := []string{"2026-11-10T10:00:00Z", "2026-11-10T11:00:00Z", "2026-11-10T12:00:00Z"}
	index := 0
	r.now = func() time.Time {
		value, _ := time.Parse(time.RFC3339, times[index])
		index++
		return value
	}
	traces := []JobTrace{
		{JobID: "sd1", Operation: "text_to_image", ModelID: "sd-1.5", Tier: "local-cpu", State: "succeeded"},
		{JobID: "sd2", Operation: "text_to_image", ModelID: "sd-1.5", Tier: "local-gpu", State: "failed"},
		{JobID: "sd3", Operation: "image_to_image", ModelID: "sd-1.5", Tier: "local-gpu", State: "succeeded"},
	}
	for _, trace := range traces {
		if err := r.RecordJobTrace(ctx, trace); err != nil {
			t.Fatalf("RecordJobTrace(%s): %v", trace.JobID, err)
		}
	}
	all, err := r.ModelUsageSummary(ctx, "")
	if err != nil {
		t.Fatalf("ModelUsageSummary(): %v", err)
	}
	if len(all) != 1 || all[0].ModelID != "sd-1.5" || all[0].SuccessfulJobs != 2 || all[0].LastOperation != "image_to_image" || all[0].LastTier != "local-gpu" || all[0].LastSuccessfulAt != times[2] {
		t.Fatalf("ModelUsageSummary() = %+v", all)
	}
	missing, err := r.ModelUsageSummary(ctx, "model-without-traces")
	if err != nil || len(missing) != 0 {
		t.Fatalf("missing-model usage = %+v, err=%v; want absent/unknown", missing, err)
	}
}

// TestObserveRecordsOpLevelFacts proves the job-driven path records latency +
// queue wait + outcome without a model id.
func TestObserveRecordsOpLevelFacts(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	if err := r.Observe(ctx, JobLike{Operation: "denoise", State: "succeeded", DurationMS: 250, QueueMS: 40}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	stats, err := r.Stats(ctx, "denoise")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 1 || stats[0].LatencyP50MS != 250 || stats[0].QueueWaitP50 != 40 {
		t.Fatalf("unexpected denoise stat: %+v", stats)
	}
}

// TestFallbackCounted proves fallback-tier usage is tallied.
func TestFallbackCounted(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	_ = r.Record(ctx, Sample{Operation: "text_to_image", State: "succeeded", DurationMS: 9000, Tier: "local-cpu", FallbackUsed: true})
	_ = r.Record(ctx, Sample{Operation: "text_to_image", State: "succeeded", DurationMS: 800, Tier: "local-gpu"})

	stats, err := r.Stats(ctx, "text_to_image")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats[0].FallbackCount != 1 {
		t.Fatalf("fallback count = %d, want 1", stats[0].FallbackCount)
	}
}

func TestRecordJobTracePersistsStructuredFacts(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	trace := JobTrace{
		JobID:       "job-1",
		Operation:   "text_to_image",
		ModelID:     "sd-1.5",
		Backend:     "stable-diffusion.cpp",
		Tier:        "local-gpu",
		Lane:        "gpu",
		State:       "succeeded",
		DurationMS:  812,
		QueueWaitMS: 24,
		ResultRef:   "out/1.png",
	}
	if err := r.RecordJobTrace(ctx, trace); err != nil {
		t.Fatalf("record trace: %v", err)
	}

	got, err := r.JobTraceByID(ctx, "job-1")
	if err != nil {
		t.Fatalf("get trace: %v", err)
	}
	if got != trace {
		t.Fatalf("trace = %+v, want %+v", got, trace)
	}
}

func TestModelProfilesAggregateOwnerMeasurementsAndRetainUnavailable(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	for _, p := range []ModelProfile{
		{Operation: "background_removal", ModelID: "isnet-general-use", Profile: "cold", Status: "measured", DurationMS: 1200, PeakMemoryB: 1024, Device: "cpu", Evidence: "run-1"},
		{Operation: "background_removal", ModelID: "isnet-general-use", Profile: "cold", Status: "measured", DurationMS: 800, PeakMemoryB: 2048, Device: "cpu", Evidence: "run-2"},
		{Operation: "text_to_image", ModelID: "sd-1.5", Profile: "warm", Status: "unavailable", Device: "cpu", Evidence: "weights-not-installed"},
	} {
		if err := r.RecordModelProfile(ctx, p); err != nil {
			t.Fatalf("record profile: %v", err)
		}
	}
	got, err := r.ModelProfiles(ctx, "", "")
	if err != nil {
		t.Fatalf("query profiles: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("profiles = %+v, want two profile groups", got)
	}
	if got[0].ModelID != "isnet-general-use" || got[0].Samples != 2 || got[0].LatencyP50MS != 800 || got[0].LatencyP95MS != 1200 || got[0].PeakMemoryB != 2048 {
		t.Fatalf("cold profile aggregation = %+v", got[0])
	}
	if got[1].ModelID != "sd-1.5" || got[1].Status != "unavailable" || got[1].Samples != 0 || got[1].LastEvidence != "weights-not-installed" {
		t.Fatalf("unavailable profile = %+v", got[1])
	}
}

func TestModelProfileRejectsUntruthfulOrMalformedEntries(t *testing.T) {
	r := newRecorder(t)
	for _, p := range []ModelProfile{
		{Operation: "op", ModelID: "m", Profile: "other", Status: "measured"},
		{Operation: "op", ModelID: "m", Profile: "cold", Status: "unknown"},
		{Operation: "op", ModelID: "m", Profile: "cold", Status: "unavailable", DurationMS: 20},
		{Operation: "op", ModelID: "m", Profile: "cold", Status: "measured"},
		{Operation: "op", ModelID: "m", Profile: "peak-memory", Status: "measured"},
	} {
		if err := r.RecordModelProfile(context.Background(), p); err == nil {
			t.Fatalf("accepted invalid profile %+v", p)
		}
	}
}

func TestRecordModelJobProfilesUsesExplicitResidencyAndMarksMemoryUnavailable(t *testing.T) {
	r := newRecorder(t)
	ctx := context.Background()
	sample := Sample{Operation: "caption", ModelID: "smolvlm-256m", Tier: "local-cpu", State: "succeeded", DurationMS: 125}
	if err := r.RecordModelJobProfiles(ctx, sample, "job-cold", map[string]string{"warm_reused": "false"}); err != nil {
		t.Fatal(err)
	}
	profiles, err := r.ModelProfiles(ctx, "smolvlm-256m", "caption")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Fatalf("profiles = %#v, want cold and peak-memory", profiles)
	}
	byName := map[string]ModelProfileStat{}
	for _, profile := range profiles {
		byName[profile.Profile] = profile
	}
	cold := byName["cold"]
	if cold.Status != "measured" || cold.Samples != 1 || cold.LatencyP50MS != 125 || cold.Device != "local-cpu" || !strings.Contains(cold.LastEvidence, "job-cold") {
		t.Fatalf("cold profile = %#v", cold)
	}
	peak := byName["peak-memory"]
	if peak.Status != "unavailable" || peak.PeakMemoryB != 0 || !strings.Contains(peak.LastEvidence, "no attributable child-process measurement") {
		t.Fatalf("peak-memory profile = %#v", peak)
	}

	if err := r.RecordModelJobProfiles(ctx, sample, "job-no-residency", nil); err != nil {
		t.Fatalf("missing metadata must be ignored instead of inferred: %v", err)
	}
	profiles, err = r.ModelProfiles(ctx, "smolvlm-256m", "caption")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Fatalf("missing metadata created inferred profiles: %#v", profiles)
	}
	for _, profile := range profiles {
		if profile.Profile == "warm" || (profile.Profile == "cold" && profile.Samples != 1) {
			t.Fatalf("missing residency metadata inferred cold/warm data: %#v", profile)
		}
	}
}

func TestRecordModelJobProfilesAcceptsOnlyExplicitChildRSS(t *testing.T) {
	r := newRecorder(t)
	ctx := context.Background()
	sample := Sample{Operation: "caption", ModelID: "smolvlm-256m", Tier: "local-cpu", State: "succeeded", DurationMS: 125}
	meta := map[string]string{
		"process_peak_rss_bytes":  "73400320",
		"process_peak_rss_source": "child_wait4_rusage_maxrss",
	}
	if err := r.RecordModelJobProfiles(ctx, sample, "job-rss", meta); err != nil {
		t.Fatal(err)
	}
	profiles, err := r.ModelProfiles(ctx, "smolvlm-256m", "caption")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].Profile != "peak-memory" {
		t.Fatalf("RSS metadata without residency must create only a peak profile: %#v", profiles)
	}
	for _, profile := range profiles {
		if profile.Profile == "peak-memory" {
			if profile.Status != "measured" || profile.PeakMemoryB != 73400320 || !strings.Contains(profile.LastEvidence, "child_wait4_rusage_maxrss") {
				t.Fatalf("peak-memory profile = %#v", profile)
			}
			return
		}
	}
	t.Fatalf("peak-memory profile missing: %#v", profiles)
}

func TestRecordModelJobProfilesStoresUnavailableRSSWithoutResidencyGuess(t *testing.T) {
	r := newRecorder(t)
	sample := Sample{Operation: "caption", ModelID: "smolvlm-256m", Tier: "local-cpu", State: "succeeded", DurationMS: 125}
	meta := map[string]string{
		"process_peak_rss_status": "unavailable",
		"process_peak_rss_reason": "persistent warm worker does not isolate a per-job peak RSS",
	}
	if err := r.RecordModelJobProfiles(context.Background(), sample, "job-warm-unavailable", meta); err != nil {
		t.Fatal(err)
	}
	profiles, err := r.ModelProfiles(context.Background(), "smolvlm-256m", "caption")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].Profile != "peak-memory" || profiles[0].Status != "unavailable" || profiles[0].PeakMemoryB != 0 || !strings.Contains(profiles[0].LastEvidence, "persistent warm worker") {
		t.Fatalf("unavailable RSS profile = %#v", profiles)
	}
}

func TestRecordModelJobProfilesStoresWarmWorkerLifetimeRSSSeparately(t *testing.T) {
	r := newRecorder(t)
	sample := Sample{Operation: "background_removal", ModelID: "u2net", Tier: "local-cpu", State: "succeeded", DurationMS: 125}
	meta := map[string]string{
		"warm_reused":            "true",
		workerResidencyIDKey:     "image-tools-worker-residency:first-job",
		workerResidencyModelKey:  "u2net",
		workerResidencyPIDKey:    "4217",
		workerResidencyBytesKey:  "73400320",
		workerResidencySourceKey: workerResidencyLinuxSource,
		workerResidencyStatusKey: "measured",
	}
	if err := r.RecordModelJobProfiles(context.Background(), sample, "job-warm", meta); err != nil {
		t.Fatal(err)
	}
	profiles, err := r.ModelProfiles(context.Background(), "u2net", "background_removal")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]ModelProfileStat{}
	for _, profile := range profiles {
		byName[profile.Profile] = profile
	}
	perJob := byName["peak-memory"]
	if perJob.Status != "unavailable" || perJob.PeakMemoryB != 0 || !strings.Contains(perJob.LastEvidence, "per-job") {
		t.Fatalf("per-job peak profile = %#v", perJob)
	}
	residency := byName["worker-residency-peak-memory"]
	if residency.Status != "measured" || residency.Samples != 1 || residency.PeakMemoryB != 73400320 ||
		!strings.Contains(residency.LastEvidence, workerResidencyLinuxSource) || !strings.Contains(residency.LastEvidence, "residency image-tools-worker-residency:first-job") ||
		!strings.Contains(residency.LastEvidence, "not isolated per-job RSS") {
		t.Fatalf("worker residency peak profile = %#v", residency)
	}
}

func TestWorkerResidencyPeakRSSMetadataFailsClosed(t *testing.T) {
	r := newRecorder(t)
	sample := Sample{Operation: "background_removal", ModelID: "u2net", State: "succeeded", DurationMS: 10}
	base := map[string]string{
		"warm_reused":            "true",
		workerResidencyIDKey:     "image-tools-worker-residency:first-job",
		workerResidencyModelKey:  "u2net",
		workerResidencyPIDKey:    "4217",
		workerResidencyBytesKey:  "73400320",
		workerResidencySourceKey: workerResidencyLinuxSource,
		workerResidencyStatusKey: "measured",
	}
	for _, mutate := range []func(map[string]string){
		func(meta map[string]string) { meta[workerResidencyModelKey] = "other-model" },
		func(meta map[string]string) { meta[workerResidencyPIDKey] = "0" },
		func(meta map[string]string) { meta[workerResidencySourceKey] = "child_wait4_rusage_maxrss" },
		func(meta map[string]string) { delete(meta, workerResidencyIDKey) },
		func(meta map[string]string) { meta[workerResidencyStatusKey] = "unavailable" },
	} {
		meta := map[string]string{}
		for key, value := range base {
			meta[key] = value
		}
		mutate(meta)
		if err := r.RecordModelJobProfiles(context.Background(), sample, "job-warm", meta); err == nil {
			t.Fatalf("malformed worker residency RSS accepted: %#v", meta)
		}
	}

	cold := map[string]string{
		"warm_reused":            "false",
		workerResidencyIDKey:     "image-tools-worker-residency:someone-else",
		workerResidencyModelKey:  "u2net",
		workerResidencyPIDKey:    "4217",
		workerResidencyBytesKey:  "73400320",
		workerResidencySourceKey: workerResidencyLinuxSource,
		workerResidencyStatusKey: "measured",
	}
	if err := r.RecordModelJobProfiles(context.Background(), sample, "job-warm", cold); err == nil {
		t.Fatalf("cold residency was accepted without matching its first durable job: %#v", cold)
	}
}

func TestRecordModelJobProfilesRejectsMalformedRSSMetadata(t *testing.T) {
	r := newRecorder(t)
	sample := Sample{Operation: "caption", ModelID: "smolvlm-256m", State: "succeeded", DurationMS: 10}
	for _, meta := range []map[string]string{
		{"warm_reused": "false", "process_peak_rss_bytes": "100"},
		{"warm_reused": "false", "process_peak_rss_bytes": "0", "process_peak_rss_source": "child_wait4_rusage_maxrss"},
		{"warm_reused": "false", "process_peak_rss_bytes": "100", "process_peak_rss_source": "host_capacity_reservation"},
		{"warm_reused": "false", "process_peak_rss_status": "unavailable"},
		{"warm_reused": "false", "process_peak_rss_status": "unavailable", "process_peak_rss_reason": "warm process", "process_peak_rss_bytes": "100"},
	} {
		if err := r.RecordModelJobProfiles(context.Background(), sample, "job-malformed", meta); err == nil {
			t.Fatalf("malformed RSS metadata accepted: %#v", meta)
		}
	}
}
