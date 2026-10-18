package composition

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	compositionpb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition"
	jobspb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/jobs"
	stylespb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/styles"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"music-tools/internal/capacity"
	core "music-tools/internal/composition"
	"music-tools/internal/jobs"
	"music-tools/internal/measures"
	"music-tools/internal/models"
	"music-tools/internal/module"
	musicstorage "music-tools/internal/storage"
	"music-tools/internal/styles"
)

type ModuleState struct {
	Styles         *styles.Store
	Pool           *core.Pool
	HTTP           *http.Client
	ResourceURL    string
	SA3URL         string
	PermissiveOnly bool
	Blobs          *musicstorage.Store
	JobManager     *jobs.Manager
}

// NewStateWithCapacityAndStorage wires the only production batch engine.
// Tests inject an isolated owner-schema database, fake broker and BlobStore.
func NewStateWithCapacityAndStorage(db jobs.SQLExecutor, broker capacity.Broker, blobs *musicstorage.Store) (*ModuleState, error) {
	if blobs == nil {
		return nil, fmt.Errorf("managed audio storage is required")
	}
	state := newState()
	var err error
	state.Pool, err = core.NewPoolWithStore(db)
	if err != nil {
		return nil, err
	}
	state.Blobs = blobs
	if blobs != nil {
		state.Pool.SetBlobPolicy(blobs)
	}
	if persistentStyles, err := styles.NewStoreWithDB(db); err == nil {
		state.Styles = persistentStyles
	} else {
		return nil, fmt.Errorf("styles store initialization failed: %w", err)
	}
	recorder := measures.NewRecorder(db)
	var manager *jobs.Manager
	manager = jobs.New(db, jobs.Config{Runner: state.runDurableJob, Admit: func(ctx context.Context, cancel context.CancelCauseFunc, job jobs.Job) (func(), map[string]string, error) {
		var input composeJobPayload
		if err := json.Unmarshal(job.Payload, &input); err != nil {
			return nil, nil, err
		}
		provider, err := input.provider()
		if err != nil {
			return nil, nil, err
		}
		if provider == providerSA3 {
			return state.sa3Admission(ctx)
		}
		if input.ModelSnapshot != nil {
			return state.songAdmission(ctx, input)
		}
		release, meta, err := capacity.Admit(ctx, cancel, broker, job.ID, 7516192768, 6442450944)
		if err != nil || meta["capacity_degrade"] == "" {
			return release, meta, err
		}
		degradeURL := state.ResourceURL + "/v1/capacity/degrade?to=" + url.QueryEscape(meta["capacity_degrade"])
		applyCtx, applyCancel := context.WithTimeout(ctx, 10*time.Second)
		defer applyCancel()
		request, requestErr := http.NewRequestWithContext(applyCtx, http.MethodPost, degradeURL, nil)
		if requestErr != nil {
			if release != nil {
				release()
			}
			return nil, nil, requestErr
		}
		response, requestErr := state.HTTP.Do(request)
		if requestErr != nil {
			if release != nil {
				release()
			}
			return nil, nil, fmt.Errorf("capacity degrade: %w", requestErr)
		}
		defer response.Body.Close()
		if response.StatusCode >= http.StatusMultipleChoices {
			body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
			if release != nil {
				release()
			}
			return nil, nil, fmt.Errorf("capacity degrade returned %s: %s", response.Status, string(body))
		}
		var applied struct {
			RestartRequired bool   `json:"restart_required"`
			Rung            string `json:"applied_rung"`
		}
		if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&applied); err != nil || applied.RestartRequired || applied.Rung != meta["capacity_degrade"] {
			if release != nil {
				release()
			}
			return nil, nil, fmt.Errorf("resource cannot apply capacity rung without managed restart; no tenant was restarted")
		}
		return release, meta, nil
	}, OnComplete: func(job jobs.Job) {
		var replenishment composeRequest
		if job.Operation == "compose-replenish" && json.Unmarshal(job.Payload, &replenishment) == nil {
			state.Pool.ReplenishmentFinished(replenishment.StyleID)
		}
		var duration, queueWait int64
		if job.StartedAt != nil && job.FinishedAt != nil {
			duration = job.FinishedAt.Sub(*job.StartedAt).Milliseconds()
		}
		if job.StartedAt != nil {
			queueWait = job.StartedAt.Sub(job.CreatedAt).Milliseconds()
		}
		_ = recorder.Observe(context.Background(), measures.JobLike{Operation: job.Operation, State: string(job.State), DurationMS: duration, QueueMS: queueWait})
	}})
	if err := manager.Start(context.Background()); err != nil {
		return nil, err
	}
	state.JobManager = manager
	state.Pool.SetReplenishTrigger(func(styleID string, needed int) {
		compiled, compileErr := state.Styles.Compile(styleID)
		if compileErr != nil {
			state.Pool.ReplenishmentFinished(styleID)
			return
		}
		seed, seedErr := freshBaseSeed()
		if seedErr != nil {
			state.Pool.ReplenishmentFinished(styleID)
			return
		}
		request := composeRequest{StyleID: styleID, Takes: needed, Duration: compiled.Params.Duration, Seed: seed}
		selected, resolveErr := resolveComposition(compiled)
		provider := selected.Engine
		if resolveErr == nil {
			compiled.Params.Variant = selected.Variant
			resolveErr = validateProviderInput(provider, compiled, request, state.PermissiveOnly)
		}
		if resolveErr != nil {
			state.Pool.ReplenishmentFinished(styleID)
			return
		}
		payload, marshalErr := json.Marshal(composeJobPayload{composeRequest: request, ResolvedLyrics: &compiled.Lyrics, SeedSource: "fresh-at-admission", Provider: provider, ResolvedStyle: snapshotStyle(provider, compiled), ModelSnapshot: snapshotModel(selected)})
		if marshalErr != nil {
			state.Pool.ReplenishmentFinished(styleID)
			return
		}
		if submitted, submitErr := manager.Submit(context.Background(), jobs.Spec{Operation: "compose-replenish", Lane: jobs.LaneGPU, Payload: payload, EstimatedSeconds: needed * request.Duration}); submitErr != nil {
			state.Pool.ReplenishmentFinished(styleID)
		} else {
			if err := state.Pool.SetLastReplenishJob(styleID, submitted.ID); err != nil {
				_ = manager.Cancel(submitted.ID)
				state.Pool.ReplenishmentFinished(styleID)
			}
		}
	})
	return state, nil
}

func (s *ModuleState) Close() {
	if s.JobManager != nil {
		s.JobManager.Close()
	}
}

func newState() *ModuleState {
	return &ModuleState{Styles: styles.NewStore(), Pool: core.NewPool(), HTTP: &http.Client{Timeout: 20 * time.Minute}, ResourceURL: getenv("ACE_STEP_URL", "http://127.0.0.1:8895"), SA3URL: getenv("STABLE_AUDIO_3_URL", "http://127.0.0.1:8896"), PermissiveOnly: os.Getenv("MUSIC_LICENSE_LANE") == "permissive"}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Module(state *ModuleState) module.Module {
	return module.Module{Name: "composition", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/styles", state.listStyles).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/styles", state.createStyle).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/styles/import", state.importStyle).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/styles/{id}", state.deleteStyle).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/styles/{id}/export", state.exportStyle).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/styles/{id}/compile", state.compileStyle).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/compose", state.compose).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/jobs/{id}", state.getJob).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/jobs", state.listJobs).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/jobs/{id}/wait", state.waitJob).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/jobs/{id}/cancel", state.cancelJob).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/takes", state.listTakes).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/takes/{id}", state.getTake).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/takes/{id}/audio", state.audioTake).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/takes/{id}/receipt", state.receiptTake).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/takes/{id}/reserve", state.reserveTake).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/takes/{id}/consume", state.consumeTake).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/takes/{id}/release", state.releaseTake).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/takes/{id}/discard", state.discardTake).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/pool/configure", state.configurePool).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/pool/draw", state.draw).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/pool/status", state.poolStatus).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeProtoJSON(w http.ResponseWriter, status int, value proto.Message) {
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(value)
	if err != nil {
		http.Error(w, "response encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func wireTake(take core.Take) *compositionpb.Take {
	result := &compositionpb.Take{
		Id: take.ID, JobId: take.JobID, StyleId: take.StyleID, PoolState: take.PoolState,
		BlobRef: take.BlobRef, ReservedBy: take.ReservedBy, TimesOffered: int32(take.TimesOffered),
		Provenance: &compositionpb.Provenance{
			ModelId: take.Provenance.ModelID, LicenseLane: take.Provenance.LicenseLane,
			AppliedRung: take.Provenance.AppliedRung, Seed: take.Provenance.Seed,
			CaptionAsAuthored: take.Provenance.CaptionAsAuthored,
			CaptionAsSent:     take.Provenance.CaptionAsSent,
		},
	}
	if !take.CreatedAt.IsZero() {
		result.CreatedAt = timestamppb.New(take.CreatedAt)
	}
	if take.ReservedAt != nil {
		result.ReservedAt = timestamppb.New(*take.ReservedAt)
	}
	return result
}
func readJSON(r *http.Request, value any) error { return json.NewDecoder(r.Body).Decode(value) }
func (s *ModuleState) listStyles(w http.ResponseWriter, _ *http.Request) {
	response := &stylespb.ListStylesResponse{}
	for _, style := range s.Styles.List() {
		response.Styles = append(response.Styles, wireStyle(style))
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (s *ModuleState) createStyle(w http.ResponseWriter, r *http.Request) {
	data, err := readPortableJSON(r, "style")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := s.Styles.Import(data)
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, styles.ErrDecode) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeProtoJSON(w, http.StatusCreated, wireStyle(created))
}

func (s *ModuleState) importStyle(w http.ResponseWriter, r *http.Request) {
	data, err := readPortableJSON(r, "style")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := s.Styles.Import(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeProtoJSON(w, http.StatusCreated, wireStyle(created))
}

// Portable ingress consumes one complete bounded document. Read one
// extra byte so a valid prefix at the existing import limit cannot hide a tail.
func readPortableJSON(r *http.Request, kind string) ([]byte, error) {
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%s JSON exceeds 1 MiB", kind)
	}
	return data, nil
}

func (s *ModuleState) exportStyle(w http.ResponseWriter, r *http.Request) {
	data, err := s.Styles.Export(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *ModuleState) deleteStyle(w http.ResponseWriter, r *http.Request) {
	if err := s.Styles.Delete(mux.Vars(r)["id"]); err != nil {
		status := http.StatusNotFound
		if err == styles.ErrBuiltinImmutable {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *ModuleState) compileStyle(w http.ResponseWriter, r *http.Request) {
	compiled, err := s.Styles.Compile(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeProtoJSON(w, http.StatusOK, &stylespb.CompiledStyle{Style: wireStyleFromCompiled(compiled), CaptionAsSent: compiled.Caption})
}

func wireStyle(style styles.Style) *stylespb.Style {
	return &stylespb.Style{Id: style.ID, Name: style.Name, Description: style.Description, Caption: style.Caption, Lyrics: style.Lyrics, Bpm: int32(style.Params.BPM), Keyscale: style.Params.KeyScale, Duration: int32(style.Params.Duration), InferenceSteps: int32(style.Params.Steps), GuidanceScale: style.Params.Guidance, Variant: style.Params.Variant, Builtin: style.Builtin}
}

func wireStyleFromCompiled(compiled styles.Compiled) *stylespb.Style {
	return &stylespb.Style{Id: compiled.StyleID, Caption: compiled.Caption, Lyrics: compiled.Lyrics, Bpm: int32(compiled.Params.BPM), Keyscale: compiled.Params.KeyScale, Duration: int32(compiled.Params.Duration), InferenceSteps: int32(compiled.Params.Steps), GuidanceScale: compiled.Params.Guidance, Variant: compiled.Params.Variant}
}

type composeRequest struct {
	StyleID  string   `json:"style_id"`
	Takes    int      `json:"takes"`
	Duration int      `json:"duration"`
	Seed     *int64   `json:"seed,omitempty"`
	Shift    *float64 `json:"shift,omitempty"`
}

func (r composeRequest) baseSeed() int64 {
	if r.Seed == nil {
		return 0
	}
	return *r.Seed
}

func freshBaseSeed() (*int64, error) {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return nil, err
	}
	seed := int64(binary.LittleEndian.Uint64(data[:]))
	return &seed, nil
}

// ResolvedLyrics is server-owned durable input, never a per-compose override.
// A missing snapshot belongs to the old always-instrumental request contract.
type composeJobPayload struct {
	ModelSnapshot *models.Model `json:"model_snapshot,omitempty"`
	composeRequest
	ResolvedLyrics *string          `json:"resolved_lyrics,omitempty"`
	SeedSource     string           `json:"seed_source,omitempty"`
	Provider       string           `json:"provider,omitempty"`
	ResolvedStyle  *styles.Compiled `json:"resolved_style,omitempty"`
}

func (request composeJobPayload) lyrics() string {
	if request.ResolvedLyrics == nil {
		return styles.InstrumentalLyrics
	}
	return styles.ResolveLyrics(*request.ResolvedLyrics)
}

func lyricsHash(text string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(text))) }

func (request composeJobPayload) lyricsMetadata() map[string]string {
	text := request.lyrics()
	intent, source := "lyrics-provided", "style-snapshot"
	if text == styles.InstrumentalLyrics {
		intent = "instrumental"
	}
	if request.ResolvedLyrics == nil {
		source = "legacy-instrumental-default"
	}
	return map[string]string{"lyrics_intent": intent, "lyrics_source": source, "lyrics_sha256": lyricsHash(text), "lyrics_bytes": strconv.Itoa(len(text))}
}

func (s *ModuleState) compose(w http.ResponseWriter, r *http.Request) {
	data, err := readPortableJSON(r, "compose")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var wire compositionpb.SubmitRequest
	if err := protojson.Unmarshal(data, &wire); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	request := composeRequest{StyleID: wire.StyleId, Takes: int(wire.Takes), Duration: int(wire.Duration), Seed: wire.Seed, Shift: wire.Shift}
	compiled, err := s.Styles.Compile(request.StyleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if request.Takes == 0 {
		request.Takes = 10
	}
	if request.Duration == 0 {
		request.Duration = compiled.Params.Duration
	}
	if _, err = core.PlanBatch(core.BatchRequest{JobID: "validate", StyleID: request.StyleID, Caption: compiled.Caption, Takes: request.Takes, Duration: request.Duration}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if request.Shift != nil && (math.IsNaN(*request.Shift) || math.IsInf(*request.Shift, 0) || *request.Shift < 1 || *request.Shift > 5) {
		http.Error(w, "shift must be finite and between 1 and 5", http.StatusBadRequest)
		return
	}
	selected, err := resolveComposition(compiled)
	provider := selected.Engine
	if err == nil {
		compiled.Params.Variant = selected.Variant
		err = validateProviderInput(provider, compiled, request, s.PermissiveOnly)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	seedSource := "explicit"
	if request.Seed == nil {
		seedSource = "fresh-at-admission"
		request.Seed, err = freshBaseSeed()
		if err != nil {
			http.Error(w, "choose base seed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	payload, _ := json.Marshal(composeJobPayload{composeRequest: request, ResolvedLyrics: &compiled.Lyrics, SeedSource: seedSource, Provider: provider, ResolvedStyle: snapshotStyle(provider, compiled), ModelSnapshot: snapshotModel(selected)})
	created, err := s.JobManager.Submit(r.Context(), jobs.Spec{Operation: "compose", Lane: jobs.LaneGPU, Payload: payload, EstimatedSeconds: request.Duration * request.Takes})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job_id": created.ID, "state": created.State, "takes": request.Takes})
}

func (s *ModuleState) runDurableJob(ctx context.Context, queued jobs.Job, emit func(int, string)) (jobs.Result, error) {
	var request composeJobPayload
	if err := json.Unmarshal(queued.Payload, &request); err != nil {
		return jobs.Result{}, err
	}
	compiled, err := s.resolvedInput(request)
	if err != nil {
		return jobs.Result{}, err
	}
	provider, err := request.provider()
	if err != nil {
		return jobs.Result{}, err
	}
	if provider != providerACE {
		if err := validateProviderInput(provider, compiled, request.composeRequest, s.PermissiveOnly); err != nil {
			return jobs.Result{}, err
		}
	}
	planned, err := core.PlanBatch(core.BatchRequest{JobID: queued.ID, StyleID: request.StyleID, Caption: compiled.Caption, Takes: request.Takes, Duration: request.Duration, Seed: request.baseSeed()})
	if err != nil {
		return jobs.Result{}, err
	}
	profiles := map[string]any{}
	completed := []core.Take{}
	result := func() jobs.Result {
		if len(completed) == 0 {
			return jobs.Result{}
		}
		first := completed[0].Provenance
		uniformModel, uniformRung := true, true
		ids := make([]string, 0, len(completed))
		for _, take := range completed {
			ids = append(ids, take.ID)
			uniformModel = uniformModel && take.Provenance.ModelID == first.ModelID
			uniformRung = uniformRung && take.Provenance.AppliedRung == first.AppliedRung
		}
		encoded, _ := json.Marshal(profiles)
		encodedIDs, _ := json.Marshal(ids)
		meta := map[string]string{"composition_provider": provider, "generation_profiles": string(encoded), "generation_profile_scope": "completed-takes", "completed_take_ids": string(encodedIDs), "seed_receipt": "verified-uint32"}
		if provider == providerACE || request.ModelSnapshot != nil {
			meta["lyrics_sent_sha256"] = lyricsHash(compiled.Lyrics)
		} else {
			meta["lyrics_sent"] = "false"
		}
		if uniformModel {
			meta["model_id"] = first.ModelID
		}
		if uniformRung {
			meta["applied_rung"] = first.AppliedRung
		}
		if !uniformModel || !uniformRung {
			meta["output_identity"] = "mixed"
		}
		return jobs.Result{Meta: meta}
	}
	for i := range planned {
		if err := ctx.Err(); err != nil {
			return result(), err
		}
		var profile any
		if provider == providerSA3 {
			profile, err = s.generateSA3Take(ctx, &planned[i], request.composeRequest, compiled)
		} else if request.ModelSnapshot != nil {
			profile, err = s.generateSongTake(ctx, &planned[i], request.composeRequest, compiled, *request.ModelSnapshot)
		} else {
			profile, err = s.generateTake(ctx, &planned[i], request.composeRequest, compiled, queued.Meta["capacity_admitted_rung"])
		}
		if err != nil {
			return result(), err
		}
		if err := s.Pool.Add(planned[i]); err != nil {
			return result(), fmt.Errorf("retain completed take %s: %w", planned[i].ID, err)
		}
		completed = append(completed, planned[i])
		profiles[planned[i].ID] = profile
		emit((i+1)*100/len(planned), fmt.Sprintf("take %d/%d", i+1, len(planned)))
	}
	return result(), nil
}

func (s *ModuleState) generateTake(ctx context.Context, take *core.Take, request composeRequest, compiled styles.Compiled, admittedRung string) (generationProfile, error) {
	if take.Provenance.Seed < 0 || take.Provenance.Seed > math.MaxUint32 {
		return generationProfile{}, fmt.Errorf("planned effective seed must be uint32")
	}
	var allowed []string
	switch admittedRung {
	case "full":
		allowed = []string{"full", "offload-dit"}
	case "offload-dit":
		allowed = []string{"offload-dit"}
	case "": // Explicit nil-broker development/test posture.
	default:
		return generationProfile{}, fmt.Errorf("unsupported capacity admission rung %q", admittedRung)
	}
	payload := map[string]any{"caption": take.Provenance.CaptionAsAuthored, "lyrics": styles.ResolveLyrics(compiled.Lyrics), "duration": request.Duration, "bpm": compiled.Params.BPM, "keyscale": compiled.Params.KeyScale, "seed": take.Provenance.Seed, "rewrite_caption": false}
	if compiled.Params.Steps > 0 {
		payload["inference_steps"] = compiled.Params.Steps
	}
	if compiled.Params.Guidance != 0 {
		payload["guidance_scale"] = compiled.Params.Guidance
	}
	if compiled.Params.Variant != "" {
		payload["expected_variant"] = compiled.Params.Variant
	}
	if request.Shift != nil {
		payload["shift"] = *request.Shift
	}
	if len(allowed) > 0 {
		payload["admitted_rungs"] = allowed
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return generationProfile{}, fmt.Errorf("encode resource compose request: %w", err)
	}
	resourceRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ResourceURL+"/v1/compose", bytes.NewReader(body))
	if err != nil {
		return generationProfile{}, err
	}
	resourceRequest.Header.Set("Content-Type", "application/json")
	response, err := s.HTTP.Do(resourceRequest)
	if err != nil {
		return generationProfile{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return generationProfile{}, fmt.Errorf("ace-step returned %s: %s", response.Status, string(data))
	}
	rung := response.Header.Get("X-Ace-Step-Profile-Rung")
	model := response.Header.Get("X-Ace-Step-Model-ID")
	lane := response.Header.Get("X-Ace-Step-License-Lane")
	if (rung != "full" && rung != "offload-dit") || model == "" || lane != "permissive" || (len(allowed) > 0 && !slices.Contains(allowed, rung)) {
		return generationProfile{}, fmt.Errorf("resource output identity missing or unsupported")
	}
	if compiled.Params.Variant != "" && !strings.HasSuffix(model, ":"+compiled.Params.Variant) {
		return generationProfile{}, fmt.Errorf("resource model does not match expected variant %q", compiled.Params.Variant)
	}
	profile, err := readGenerationProfile(response.Header, take.Provenance.Seed, request, compiled)
	if err != nil {
		return generationProfile{}, err
	}
	take.Provenance.AppliedRung = rung
	take.Provenance.ModelID = model
	take.Provenance.LicenseLane = lane
	if s.Blobs == nil {
		return generationProfile{}, fmt.Errorf("managed audio storage is required")
	}
	ref := "out/" + take.ID + ".wav"
	media, err := core.ValidatedWAV(response.Body)
	if err != nil {
		return generationProfile{}, err
	}
	if err := s.Blobs.PutOwned(ctx, ref, take.ID, media, "audio/wav", true); err != nil {
		return generationProfile{}, err
	}
	take.BlobRef = ref
	return profile, nil
}

func (s *ModuleState) getJob(w http.ResponseWriter, r *http.Request) {
	item, err := s.JobManager.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		jobError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, wireJob(item))
}

func (s *ModuleState) listJobs(w http.ResponseWriter, r *http.Request) {
	items, err := s.JobManager.List(r.Context(), 100)
	if err != nil {
		jobError(w, err)
		return
	}
	wire := &jobspb.ListJobsResponse{}
	for _, item := range items {
		wire.Jobs = append(wire.Jobs, wireJob(item))
	}
	writeProtoJSON(w, http.StatusOK, wire)
}

func (s *ModuleState) waitJob(w http.ResponseWriter, r *http.Request) {
	item, err := s.JobManager.Wait(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		if r.Context().Err() == nil {
			jobError(w, err)
		}
		return
	}
	writeProtoJSON(w, http.StatusOK, wireJob(item))
}

func (s *ModuleState) cancelJob(w http.ResponseWriter, r *http.Request) {
	if err := s.JobManager.Cancel(mux.Vars(r)["id"]); err != nil {
		jobError(w, err)
		return
	}
	s.getJob(w, r)
}

func jobError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	if errors.Is(err, jobs.ErrNotFound) {
		status = http.StatusNotFound
	}
	http.Error(w, err.Error(), status)
}

func wireJob(item jobs.Job) *jobspb.Job {
	meta := maps.Clone(item.Meta)
	if item.Operation == "compose" || item.Operation == "compose-replenish" {
		var request composeJobPayload
		if json.Unmarshal(item.Payload, &request) == nil && request.StyleID != "" {
			if meta == nil {
				meta = map[string]string{}
			}
			maps.Copy(meta, request.lyricsMetadata())
			if provider, err := request.provider(); err == nil {
				meta["composition_provider"] = provider
			}
			meta["seed_base"] = strconv.FormatInt(request.baseSeed(), 10)
			source := request.SeedSource
			if source == "" {
				source = "legacy-recorded"
				if request.Seed == nil {
					source = "legacy-zero-default"
				}
			}
			meta["seed_base_source"] = source
		}
	}
	return &jobspb.Job{
		Id: item.ID, Kind: item.Operation, State: string(item.State), Lane: string(item.Lane), Error: item.Error,
		Total: 100, Completed: int32(item.Progress), ResultRef: item.ResultRef, Meta: meta,
	}
}

func (s *ModuleState) listTakes(w http.ResponseWriter, r *http.Request) {
	wire := &compositionpb.ListTakesResponse{}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			http.Error(w, "limit must be 1–100", http.StatusBadRequest)
			return
		}
		limit = n
	}
	for _, take := range s.Pool.ListFiltered(r.URL.Query().Get("style_id"), r.URL.Query().Get("job_id")) {
		if len(wire.Takes) >= limit {
			w.Header().Set("X-Music-Takes-Truncated", "true")
			break
		}
		wire.Takes = append(wire.Takes, wireTake(take))
	}
	writeProtoJSON(w, http.StatusOK, wire)
}

func (s *ModuleState) reserveTake(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Holder string `json:"holder"`
	}
	_ = readJSON(r, &body)
	// The caller has already selected an exact take id. Use the pool's
	// id-specific atomic transition rather than style-level Draw, which can
	// legitimately select a different peer when several share the style.
	drawn, _, err := s.Pool.Reserve(mux.Vars(r)["id"], body.Holder)
	if err != nil {
		http.Error(w, "take is not available", http.StatusConflict)
		return
	}
	writeProtoJSON(w, http.StatusOK, wireTake(drawn))
}

func (s *ModuleState) consumeTake(w http.ResponseWriter, r *http.Request) {
	take, err := s.Pool.MarkConsumed(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeProtoJSON(w, http.StatusOK, wireTake(take))
}

func (s *ModuleState) releaseTake(w http.ResponseWriter, r *http.Request) {
	if err := s.Pool.Release(mux.Vars(r)["id"]); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "available"})
}

func (s *ModuleState) discardTake(w http.ResponseWriter, r *http.Request) {
	if err := s.Pool.Discard(mux.Vars(r)["id"]); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "discarded"})
}

func (s *ModuleState) getTake(w http.ResponseWriter, r *http.Request) {
	take, ok := s.Pool.Get(mux.Vars(r)["id"])
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeProtoJSON(w, http.StatusOK, wireTake(take))
}

func (s *ModuleState) audioTake(w http.ResponseWriter, r *http.Request) {
	take, ok := s.Pool.Get(mux.Vars(r)["id"])
	if !ok || take.BlobRef == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	if s.Blobs != nil {
		body, mime, err := s.Blobs.Get(r.Context(), take.BlobRef)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer body.Close()
		if mime != "" {
			w.Header().Set("Content-Type", mime)
		}
		_, _ = io.Copy(w, body)
		return
	}
	http.ServeFile(w, r, take.BlobRef)
}

func (s *ModuleState) configurePool(w http.ResponseWriter, r *http.Request) {
	var p struct {
		StyleID   string `json:"style_id"`
		Target    int    `json:"target_depth"`
		Threshold int    `json:"replenish_below"`
	}
	if err := readJSON(r, &p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Pool.Configure(p.StyleID, p.Target, p.Threshold); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *ModuleState) draw(w http.ResponseWriter, r *http.Request) {
	var p struct {
		StyleID string `json:"style_id"`
		Holder  string `json:"holder"`
	}
	if err := readJSON(r, &p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	take, replenish, err := s.Pool.Draw(p.StyleID, p.Holder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"take": take, "replenishment_needed": replenish})
}

func (s *ModuleState) poolStatus(w http.ResponseWriter, r *http.Request) {
	a, res, target, threshold := s.Pool.Status(r.URL.Query().Get("style_id"))
	writeJSON(w, http.StatusOK, map[string]int{"available": a, "reserved": res, "target_depth": target, "replenish_below": threshold})
}
