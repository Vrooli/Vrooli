package composition

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	"music-tools/internal/module"
	musicstorage "music-tools/internal/storage"
	"music-tools/internal/styles"

	"github.com/gorilla/mux"
)

type ModuleState struct {
	Styles      *styles.Store
	Pool        *core.Pool
	HTTP        *http.Client
	ResourceURL string
	AudioDir    string
	Blobs       *musicstorage.Store
	mu          sync.Mutex
	jobs        map[string]*job
	JobManager  *jobs.Manager
}

// NewStateWithJobs wires the composition domain to the durable server-owned
// job manager. Tests that exercise only the HTTP shape may continue using
// NewState, while production always supplies the database-backed manager.
func NewStateWithJobs(db jobs.SQLExecutor) (*ModuleState, error) {
	return NewStateWithCapacity(db, nil)
}

// NewStateWithCapacity is the production constructor. The broker is injected
// so unit tests can exercise durable jobs without claiming host VRAM.
func NewStateWithCapacity(db jobs.SQLExecutor, broker capacity.Broker) (*ModuleState, error) {
	return NewStateWithCapacityAndStorage(db, broker, nil)
}

func NewStateWithCapacityAndStorage(db jobs.SQLExecutor, broker capacity.Broker, blobs *musicstorage.Store) (*ModuleState, error) {
	state := NewState()
	state.Pool = core.NewPoolWithStore(db)
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
	manager = jobs.New(db, jobs.Config{Runner: state.runDurableJob, Admit: func(ctx context.Context, job jobs.Job) (func(), map[string]string, error) {
		release, meta, err := capacity.Admit(ctx, broker, job.ID, 7516192768, 6442450944)
		if err != nil || meta["capacity_degrade"] == "" {
			return release, meta, err
		}
		degradeURL := state.ResourceURL + "/v1/capacity/degrade?to=" + url.QueryEscape(meta["capacity_degrade"])
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, degradeURL, nil)
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
		payload, marshalErr := json.Marshal(composeRequest{StyleID: styleID, Takes: needed, Duration: 45, Seed: time.Now().UnixNano()})
		if marshalErr != nil {
			state.Pool.ReplenishmentFinished(styleID)
			return
		}
		if submitted, submitErr := manager.Submit(context.Background(), jobs.Spec{Operation: "compose-replenish", Lane: jobs.LaneGPU, Payload: payload, EstimatedSeconds: needed * 45}); submitErr != nil {
			state.Pool.ReplenishmentFinished(styleID)
		} else {
			state.Pool.SetLastReplenishJob(styleID, submitted.ID)
		}
	})
	return state, nil
}

func (s *ModuleState) Close() {
	if s.JobManager != nil {
		s.JobManager.Close()
	}
}

type job struct {
	ID    string        `json:"id"`
	State string        `json:"state"`
	Error string        `json:"error,omitempty"`
	Takes []core.Take   `json:"takes"`
	Done  chan struct{} `json:"-"`
}

func NewState() *ModuleState {
	dir := os.Getenv("MUSIC_TOOLS_AUDIO_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "music-tools-audio")
	}
	return &ModuleState{Styles: styles.NewStore(), Pool: core.NewPool(), HTTP: &http.Client{Timeout: 20 * time.Minute}, ResourceURL: getenv("ACE_STEP_URL", "http://127.0.0.1:8895"), AudioDir: dir, jobs: map[string]*job{}}
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
	var style styles.Style
	if err := readJSON(r, &style); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := s.Styles.Create(style)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeProtoJSON(w, http.StatusCreated, wireStyle(created))
}

func (s *ModuleState) importStyle(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
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
	return &stylespb.Style{Id: style.ID, Name: style.Name, Description: style.Description, Caption: style.Caption, Bpm: int32(style.Params.BPM), Keyscale: style.Params.KeyScale, Duration: int32(style.Params.Duration), InferenceSteps: int32(style.Params.Steps), GuidanceScale: style.Params.Guidance, Variant: style.Params.Variant, Builtin: style.Builtin}
}

func wireStyleFromCompiled(compiled styles.Compiled) *stylespb.Style {
	return &stylespb.Style{Id: compiled.StyleID, Caption: compiled.Caption, Bpm: int32(compiled.Params.BPM), Keyscale: compiled.Params.KeyScale, Duration: int32(compiled.Params.Duration), InferenceSteps: int32(compiled.Params.Steps), GuidanceScale: compiled.Params.Guidance, Variant: compiled.Params.Variant}
}

type composeRequest struct {
	StyleID  string `json:"style_id"`
	Takes    int    `json:"takes"`
	Duration int    `json:"duration"`
	Seed     int64  `json:"seed"`
}

func (s *ModuleState) compose(w http.ResponseWriter, r *http.Request) {
	var request composeRequest
	if err := readJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
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
	if s.JobManager != nil {
		payload, _ := json.Marshal(request)
		created, submitErr := s.JobManager.Submit(context.Background(), jobs.Spec{Operation: "compose", Lane: jobs.LaneGPU, Payload: payload, EstimatedSeconds: request.Duration * request.Takes})
		if submitErr != nil {
			http.Error(w, submitErr.Error(), http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"job_id": created.ID, "state": created.State, "takes": request.Takes})
		return
	}
	id := fmt.Sprintf("job-%d", time.Now().UnixNano())
	item := &job{ID: id, State: "queued", Done: make(chan struct{})}
	s.mu.Lock()
	s.jobs[id] = item
	s.mu.Unlock()
	go s.runBatch(item, request, compiled.Caption)
	writeJSON(w, http.StatusAccepted, map[string]any{"job_id": id, "state": item.State, "takes": request.Takes})
}

func (s *ModuleState) runDurableJob(ctx context.Context, queued jobs.Job, emit func(int, string)) (jobs.Result, error) {
	var request composeRequest
	if err := json.Unmarshal(queued.Payload, &request); err != nil {
		return jobs.Result{}, err
	}
	compiled, err := s.Styles.Compile(request.StyleID)
	if err != nil {
		return jobs.Result{}, err
	}
	planned, err := core.PlanBatch(core.BatchRequest{JobID: queued.ID, StyleID: request.StyleID, Caption: compiled.Caption, Takes: request.Takes, Duration: request.Duration, Seed: request.Seed})
	if err != nil {
		return jobs.Result{}, err
	}
	if err := os.MkdirAll(s.AudioDir, 0o755); err != nil {
		return jobs.Result{}, err
	}
	for i := range planned {
		select {
		case <-ctx.Done():
			return jobs.Result{}, ctx.Err()
		default:
		}
		if err := s.generateTake(ctx, &planned[i], request, compiled); err != nil {
			return jobs.Result{}, err
		}
		s.Pool.Add(planned[i])
		s.mu.Lock()
		if item := s.jobs[queued.ID]; item != nil {
			item.Takes = append(item.Takes, planned[i])
		}
		s.mu.Unlock()
		emit((i+1)*100/len(planned), fmt.Sprintf("take %d/%d", i+1, len(planned)))
	}
	return jobs.Result{Meta: map[string]string{"model_id": planned[0].Provenance.ModelID, "applied_rung": planned[0].Provenance.AppliedRung}}, nil
}

func (s *ModuleState) generateTake(ctx context.Context, take *core.Take, request composeRequest, compiled styles.Compiled) error {
	payload := map[string]any{"caption": take.Provenance.CaptionAsAuthored, "lyrics": "[instrumental]", "duration": request.Duration, "bpm": compiled.Params.BPM, "keyscale": compiled.Params.KeyScale, "inference_steps": compiled.Params.Steps, "guidance_scale": compiled.Params.Guidance, "seed": take.Provenance.Seed, "rewrite_caption": false}
	body, _ := json.Marshal(payload)
	resourceRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ResourceURL+"/v1/compose", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resourceRequest.Header.Set("Content-Type", "application/json")
	response, err := s.HTTP.Do(resourceRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("ace-step returned %s: %s", response.Status, string(data))
	}
	rung := response.Header.Get("X-Ace-Step-Profile-Rung")
	if rung == "" {
		rung = "full"
	}
	take.Provenance.AppliedRung = rung
	if s.Blobs != nil {
		data, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return readErr
		}
		ref := "out/" + take.ID + ".wav"
		if writeErr := s.Blobs.PutOwned(ctx, ref, take.ID, bytes.NewReader(data), "audio/wav", true); writeErr != nil {
			return writeErr
		}
		take.BlobRef = ref
		return nil
	}
	target := filepath.Join(s.AudioDir, take.ID+".wav")
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	take.BlobRef = target
	return nil
}

func (s *ModuleState) runBatch(item *job, request composeRequest, caption string) {
	s.mu.Lock()
	item.State = "running"
	s.mu.Unlock()
	planned, err := core.PlanBatch(core.BatchRequest{JobID: item.ID, StyleID: request.StyleID, Caption: caption, Takes: request.Takes, Duration: request.Duration, Seed: request.Seed})
	if err != nil {
		s.finish(item, err)
		return
	}
	_ = os.MkdirAll(s.AudioDir, 0o755)
	compiled, _ := s.Styles.Compile(request.StyleID)
	for i := range planned {
		payload := map[string]any{"caption": planned[i].Provenance.CaptionAsAuthored, "lyrics": "[instrumental]", "duration": request.Duration, "bpm": compiled.Params.BPM, "keyscale": compiled.Params.KeyScale, "inference_steps": compiled.Params.Steps, "guidance_scale": compiled.Params.Guidance, "seed": planned[i].Provenance.Seed, "rewrite_caption": false}
		body, _ := json.Marshal(payload)
		response, callErr := s.HTTP.Post(s.ResourceURL+"/v1/compose", "application/json", bytes.NewReader(body))
		if callErr != nil {
			s.finish(item, callErr)
			return
		}
		if response.StatusCode >= 300 {
			data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
			_ = response.Body.Close()
			s.finish(item, fmt.Errorf("ace-step returned %s: %s", response.Status, string(data)))
			return
		}
		rung := response.Header.Get("X-ACE-Step-Profile-Rung")
		if rung == "" {
			rung = "full"
		}
		planned[i].Provenance.AppliedRung = rung
		target := filepath.Join(s.AudioDir, planned[i].ID+".wav")
		file, fileErr := os.Create(target)
		if fileErr == nil {
			_, fileErr = io.Copy(file, response.Body)
			_ = file.Close()
		}
		_ = response.Body.Close()
		if fileErr != nil {
			s.finish(item, fileErr)
			return
		}
		planned[i].BlobRef = target
		// Retain each completed take immediately. A later resource failure must
		// not erase usable work from an otherwise partial batch.
		s.Pool.Add(planned[i])
		s.mu.Lock()
		item.Takes = append(item.Takes, planned[i])
		s.mu.Unlock()
	}
	s.mu.Lock()
	item.State = "succeeded"
	s.mu.Unlock()
	close(item.Done)
}
func (s *ModuleState) finish(item *job, err error) {
	s.mu.Lock()
	item.State, item.Error = "failed", err.Error()
	s.mu.Unlock()
	close(item.Done)
}
func (s *ModuleState) getJob(w http.ResponseWriter, r *http.Request) {
	if s.JobManager != nil {
		item, err := s.JobManager.Get(r.Context(), mux.Vars(r)["id"])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeProtoJSON(w, http.StatusOK, wireJob(item))
		return
	}
	s.mu.Lock()
	item := s.jobs[mux.Vars(r)["id"]]
	s.mu.Unlock()
	if item == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *ModuleState) listJobs(w http.ResponseWriter, r *http.Request) {
	if s.JobManager == nil {
		writeProtoJSON(w, http.StatusOK, &jobspb.ListJobsResponse{})
		return
	}
	items, err := s.JobManager.List(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := &jobspb.ListJobsResponse{}
	for _, item := range items {
		response.Jobs = append(response.Jobs, wireJob(item))
	}
	writeProtoJSON(w, http.StatusOK, response)
}
func (s *ModuleState) waitJob(w http.ResponseWriter, r *http.Request) {
	if s.JobManager != nil {
		item, err := s.JobManager.Wait(r.Context(), mux.Vars(r)["id"])
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			http.NotFound(w, r)
			return
		}
		writeProtoJSON(w, http.StatusOK, wireJob(item))
		return
	}
	s.mu.Lock()
	item := s.jobs[mux.Vars(r)["id"]]
	s.mu.Unlock()
	if item == nil {
		http.NotFound(w, r)
		return
	}
	select {
	case <-item.Done:
		writeJSON(w, http.StatusOK, item)
	case <-r.Context().Done():
		return
	}
}

func (s *ModuleState) cancelJob(w http.ResponseWriter, r *http.Request) {
	if s.JobManager == nil {
		http.Error(w, "durable jobs are not enabled", http.StatusNotImplemented)
		return
	}
	if err := s.JobManager.Cancel(mux.Vars(r)["id"]); err != nil {
		http.NotFound(w, r)
		return
	}
	item, err := s.JobManager.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeProtoJSON(w, http.StatusOK, wireJob(item))
}

func wireJob(item jobs.Job) *jobspb.Job {
	return &jobspb.Job{
		Id: item.ID, Kind: item.Operation, State: string(item.State), Lane: string(item.Lane), Error: item.Error,
		Total: 100, Completed: int32(item.Progress), ResultRef: item.ResultRef, Meta: item.Meta,
	}
}
func (s *ModuleState) listTakes(w http.ResponseWriter, r *http.Request) {
	wire := &compositionpb.ListTakesResponse{}
	for _, take := range s.Pool.ListFiltered(r.URL.Query().Get("style_id"), r.URL.Query().Get("job_id")) {
		wire.Takes = append(wire.Takes, wireTake(take))
	}
	writeProtoJSON(w, http.StatusOK, wire)
}

func (s *ModuleState) reserveTake(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Holder string `json:"holder"`
	}
	_ = readJSON(r, &body)
	take, ok := s.Pool.Get(mux.Vars(r)["id"])
	if !ok || take.PoolState != "available" {
		http.Error(w, "take is not available", http.StatusConflict)
		return
	}
	// Reservation is exposed as a transition endpoint while the pool's draw
	// operation remains the atomic style-level consumer path.
	drawn, _, err := s.Pool.Draw(take.StyleID, body.Holder)
	if err != nil || drawn.ID != take.ID {
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
