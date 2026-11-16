// Package analysis exposes durable MIR operations on the shared Music Tools
// jobs manager. It deliberately does not own another manager or GPU worker.
package analysis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"music-tools/internal/capacity"
	"music-tools/internal/jobs"
	"music-tools/internal/librarysync"
	"music-tools/internal/module"
	musicstorage "music-tools/internal/storage"
)

const (
	operationAnalyze  = "analyze"
	operationSeparate = "separate"
	maxResultBytes    = 64 << 20
	maxStemBytes      = musicstorage.MaxOwnedBytes
)

var requiredStems = []string{"vocals", "drums", "bass", "other"}

type sourceRequest struct {
	SourceBlobKey string                     `json:"source_blob_key"`
	Options       map[string]json.RawMessage `json:"options,omitempty"`
}

type queuedRequest struct {
	SourceBlobKey string                     `json:"source_blob_key"`
	SourceSHA256  string                     `json:"source_sha256"`
	Options       map[string]json.RawMessage `json:"options,omitempty"`
}

type moduleState struct {
	jobs   *jobs.Manager
	blobs  *musicstorage.Store
	client *http.Client
	url    string
	syncer *librarysync.Worker
}

// Module registers operations with the already-running manager. The request
// processor owns these durable jobs and GPU capacity claims; the resource only
// performs one synchronous inference per submitted job.
func Module(manager *jobs.Manager, blobs *musicstorage.Store, broker capacity.Broker, syncer *librarysync.Worker) (module.Module, error) {
	if manager == nil || blobs == nil || syncer == nil {
		return module.Module{}, errors.New("analysis requires the shared jobs manager and managed blob store")
	}
	state := &moduleState{jobs: manager, blobs: blobs, syncer: syncer, client: &http.Client{Timeout: 0}, url: strings.TrimRight(env("MUSIC_MIR_URL", "http://127.0.0.1:8898"), "/")}
	for _, op := range []string{operationAnalyze, operationSeparate} {
		operation := op
		if err := manager.RegisterOperation(operation, jobs.Operation{
			Runner: func(ctx context.Context, job jobs.Job, emit func(int, string)) (jobs.Result, error) {
				return state.run(ctx, job, emit, operation)
			},
			Admit: func(ctx context.Context, cancel context.CancelCauseFunc, job jobs.Job) (func(), map[string]string, error) {
				return capacity.Admit(ctx, cancel, broker, job.ID, 7516192768, 6442450944)
			},
		}); err != nil {
			return module.Module{}, err
		}
	}
	return module.Module{Name: "analysis", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/analyze", state.submit(operationAnalyze)).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/separate", state.submit(operationSeparate)).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/analysis/{id}/result", state.result).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/analysis/{id}/stems/{name}", state.stem).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/analysis/library-sync", state.syncStatus).Methods(http.MethodGet)
	}, Endpoints: Endpoints}, nil
}

func (s *moduleState) syncStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.syncer.Status(r.Context())
	if err != nil {
		http.Error(w, "library sync status unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *moduleState) submit(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input sourceRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&input); err != nil || strings.TrimSpace(input.SourceBlobKey) == "" {
			http.Error(w, "source_blob_key is required", http.StatusBadRequest)
			return
		}
		digest, err := s.digest(r.Context(), input.SourceBlobKey)
		if err != nil {
			http.Error(w, "source blob unavailable: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
		payload, err := json.Marshal(queuedRequest{SourceBlobKey: input.SourceBlobKey, SourceSHA256: digest, Options: input.Options})
		if err != nil {
			http.Error(w, "invalid analysis options", http.StatusBadRequest)
			return
		}
		job, err := s.jobs.Submit(r.Context(), jobs.Spec{Operation: operation, Lane: jobs.LaneGPU, Payload: payload})
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"job_id": job.ID, "state": job.State, "operation": operation, "source_sha256": digest})
	}
}

func (s *moduleState) result(w http.ResponseWriter, r *http.Request) {
	job, err := s.analysisJob(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if job.State != jobs.StateSucceeded || job.ResultRef == "" {
		http.Error(w, "analysis result is not ready", http.StatusConflict)
		return
	}
	blob, mediaType, err := s.blobs.Get(r.Context(), job.ResultRef)
	if err != nil {
		http.Error(w, "analysis result unavailable: "+err.Error(), http.StatusGone)
		return
	}
	defer blob.Close()
	w.Header().Set("Content-Type", mediaType)
	_, _ = io.Copy(w, blob)
}

func (s *moduleState) stem(w http.ResponseWriter, r *http.Request) {
	job, err := s.analysisJob(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if job.Operation != operationSeparate || job.State != jobs.StateSucceeded || job.ResultRef == "" {
		http.Error(w, "separated stems are not ready", http.StatusConflict)
		return
	}
	name := mux.Vars(r)["name"]
	if !containsStem(name) {
		http.Error(w, "unknown stem", http.StatusNotFound)
		return
	}
	blob, _, err := s.blobs.Get(r.Context(), job.ResultRef)
	if err != nil {
		http.Error(w, "analysis result unavailable", http.StatusGone)
		return
	}
	defer blob.Close()
	var result analysisResult
	if err := json.NewDecoder(io.LimitReader(blob, maxResultBytes)).Decode(&result); err != nil {
		http.Error(w, "analysis result is invalid", http.StatusInternalServerError)
		return
	}
	key := result.StemBlobKeys[name]
	if key == "" {
		http.Error(w, "stem unavailable", http.StatusGone)
		return
	}
	stem, mediaType, err := s.blobs.Get(r.Context(), key)
	if err != nil {
		http.Error(w, "stem unavailable: "+err.Error(), http.StatusGone)
		return
	}
	defer stem.Close()
	w.Header().Set("Content-Type", mediaType)
	_, _ = io.Copy(w, stem)
}

func (s *moduleState) analysisJob(r *http.Request) (jobs.Job, error) {
	job, err := s.jobs.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		return jobs.Job{}, err
	}
	if job.Operation != operationAnalyze && job.Operation != operationSeparate {
		return jobs.Job{}, jobs.ErrNotFound
	}
	return job, nil
}

func containsStem(name string) bool {
	for _, allowed := range requiredStems {
		if name == allowed {
			return true
		}
	}
	return false
}

func (s *moduleState) run(ctx context.Context, job jobs.Job, emit func(int, string), operation string) (jobs.Result, error) {
	var input queuedRequest
	if err := json.Unmarshal(job.Payload, &input); err != nil || input.SourceBlobKey == "" || len(input.SourceSHA256) != 64 {
		return jobs.Result{}, errors.New("invalid durable analysis request")
	}
	if err := ctx.Err(); err != nil {
		return jobs.Result{}, err
	}
	currentDigest, err := s.digest(ctx, input.SourceBlobKey)
	if err != nil || currentDigest != input.SourceSHA256 {
		return jobs.Result{}, errors.New("source blob changed after job submission")
	}
	sourcePath, err := s.blobs.PathForKey(input.SourceBlobKey)
	if err != nil {
		return jobs.Result{}, err
	}
	storageRoot, err := filepath.EvalSymlinks(s.blobs.Root())
	if err != nil {
		return jobs.Result{}, fmt.Errorf("resolve managed storage root: %w", err)
	}
	sourcePath, err = filepath.EvalSymlinks(sourcePath)
	if err != nil || !withinRoot(storageRoot, sourcePath) {
		return jobs.Result{}, errors.New("source blob path is outside managed storage")
	}
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil || !sourceInfo.Mode().IsRegular() {
		return jobs.Result{}, errors.New("source blob path must resolve to a regular managed file")
	}
	outputDir := filepath.Join(storageRoot, "analysis", ".staging", job.ID)
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return jobs.Result{}, fmt.Errorf("create analysis staging directory: %w", err)
	}
	resolvedOutputDir, err := filepath.EvalSymlinks(outputDir)
	if err != nil || resolvedOutputDir != outputDir {
		return jobs.Result{}, errors.New("analysis staging directory must not contain symlinks")
	}
	defer os.RemoveAll(outputDir)
	var committedKeys []string
	keepArtifacts := false
	defer func() {
		if !keepArtifacts {
			for _, key := range committedKeys {
				_ = s.blobs.Delete(context.Background(), key)
			}
		}
	}()
	options := input.Options
	if options == nil {
		options = map[string]json.RawMessage{}
	}
	if operation == operationSeparate {
		stems, _ := json.Marshal(requiredStems)
		options["stems"] = stems
	}
	requestBody, err := json.Marshal(map[string]any{
		"request_id":   job.ID,
		"input":        map[string]string{"path": sourcePath, "sha256": input.SourceSHA256},
		"allowed_root": storageRoot,
		"output_dir":   outputDir,
		"options":      options,
	})
	if err != nil {
		return jobs.Result{}, err
	}
	endpoint := s.url + "/v1/" + operation
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return jobs.Result{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	emit(3, "sending local source to music-mir")
	response, err := s.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			emit(5, "waiting for music-mir cancellation drain; retaining GPU admission")
			s.cancelAndWaitForDrain(job.ID, emit)
		}
		return jobs.Result{}, fmt.Errorf("music-mir %s request: %w", operation, err)
	}
	defer response.Body.Close()
	if ctx.Err() != nil {
		emit(5, "waiting for music-mir cancellation drain; retaining GPU admission")
		s.cancelAndWaitForDrain(job.ID, emit)
		return jobs.Result{}, ctx.Err()
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return jobs.Result{}, fmt.Errorf("music-mir %s returned %s: %s", operation, response.Status, strings.TrimSpace(string(message)))
	}
	responseBytes, err := io.ReadAll(io.LimitReader(response.Body, maxResultBytes+1))
	if err != nil {
		return jobs.Result{}, fmt.Errorf("read music-mir result: %w", err)
	}
	if len(responseBytes) > maxResultBytes {
		return jobs.Result{}, errors.New("music-mir response exceeds the configured size limit")
	}
	var result mirResponse
	if err := json.Unmarshal(responseBytes, &result); err != nil {
		return jobs.Result{}, fmt.Errorf("decode music-mir result: %w", err)
	}
	if len(result.Analysis) > maxResultBytes || (operation == operationAnalyze && len(result.Analysis) == 0) {
		return jobs.Result{}, errors.New("music-mir analyze result must include a bounded analysis object")
	}
	if operation == operationAnalyze {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(result.Analysis, &object); err != nil || object == nil {
			return jobs.Result{}, errors.New("music-mir analysis field must be a JSON object")
		}
	}
	if strings.TrimSpace(result.ResourceVersion) == "" || strings.TrimSpace(result.Model) == "" || strings.TrimSpace(result.BackendRevision) == "" {
		return jobs.Result{}, errors.New("music-mir result must identify the resource version, model and backend revision")
	}
	emit(85, "persisting analysis and provenance")
	stemRefs := map[string]string{}
	if operation == operationSeparate {
		if len(result.Stems) != len(requiredStems) {
			return jobs.Result{}, fmt.Errorf("music-mir separation must return exactly %v", requiredStems)
		}
		for _, name := range requiredStems {
			stem, ok := result.Stems[name]
			if !ok {
				return jobs.Result{}, fmt.Errorf("music-mir separation omitted %q stem", name)
			}
			if err := ctx.Err(); err != nil {
				return jobs.Result{}, err
			}
			if !safeFilename(stem.File) || len(stem.SHA256) != 64 {
				return jobs.Result{}, fmt.Errorf("music-mir returned invalid %q stem file metadata", name)
			}
			filePath := filepath.Join(outputDir, stem.File)
			info, err := os.Lstat(filePath)
			if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxStemBytes {
				return jobs.Result{}, fmt.Errorf("music-mir returned missing, non-regular or oversized %q stem", name)
			}
			file, err := os.Open(filePath)
			if err != nil {
				return jobs.Result{}, err
			}
			key := fmt.Sprintf("analysis/%s/stems/%s", job.ID, name)
			hasher := sha256.New()
			writeErr := s.blobs.PutOwned(ctx, key, "music-tools-analysis:"+job.ID, io.TeeReader(file, hasher), safeMediaType(stem.MediaType), false)
			closeErr := file.Close()
			actualDigest := hex.EncodeToString(hasher.Sum(nil))
			if writeErr == nil {
				committedKeys = append(committedKeys, key)
			}
			if writeErr != nil || closeErr != nil || actualDigest != stem.SHA256 {
				_ = s.blobs.Delete(context.Background(), key)
				return jobs.Result{}, fmt.Errorf("persist %s stem with verified SHA-256: %w", name, errors.Join(writeErr, closeErr, digestMismatch(actualDigest, stem.SHA256)))
			}
			if err := ctx.Err(); err != nil {
				_ = s.blobs.Delete(context.Background(), key)
				return jobs.Result{}, err
			}
			stemRefs[name] = key
		}
	} else if len(result.Stems) != 0 {
		return jobs.Result{}, errors.New("analyze response unexpectedly included stems")
	}
	provenance := analysisResult{
		JobID: job.ID, Operation: operation, SourceBlobKey: input.SourceBlobKey,
		SourceSHA256: input.SourceSHA256, Resource: "music-mir", ResourceVersion: result.ResourceVersion,
		Model: result.Model, BackendRevision: result.BackendRevision, CompletedAt: time.Now().UTC(),
		Analysis: result.Analysis, StemBlobKeys: stemRefs, Timing: result.Timing,
		Confidence: result.Confidence, Alternatives: result.Alternatives,
	}
	encoded, err := json.Marshal(provenance)
	if err != nil {
		return jobs.Result{}, err
	}
	resultKey := "analysis/" + job.ID + "/result.json"
	if err := s.blobs.PutOwned(ctx, resultKey, "music-tools-analysis:"+job.ID, bytes.NewReader(encoded), "application/json", false); err != nil {
		return jobs.Result{}, fmt.Errorf("persist analysis result: %w", err)
	}
	committedKeys = append(committedKeys, resultKey)
	if err := ctx.Err(); err != nil {
		return jobs.Result{}, err
	}
	keepArtifacts = true
	emit(100, "analysis stored")
	meta, _ := json.Marshal(map[string]any{"resource": "music-mir", "resource_version": result.ResourceVersion, "model": result.Model, "backend_revision": result.BackendRevision, "source_sha256": input.SourceSHA256, "stem_blob_keys": stemRefs})
	return jobs.Result{Ref: resultKey, Meta: map[string]string{"analysis_provenance": string(meta), "source_sha256": input.SourceSHA256}}, nil
}

// cancelAndWaitForDrain keeps the runner alive until music-mir confirms it has
// no active inference for this request. Each HTTP attempt has a hard timeout;
// retries are idempotent by request/job ID. There is intentionally no overall
// deadline: returning without an acknowledgement would release the shared
// manager's GPU lane and capacity claim while inference might still be active.
func (s *moduleState) cancelAndWaitForDrain(jobID string, emit func(int, string)) {
	payload, _ := json.Marshal(map[string]string{"job_id": jobID, "request_id": jobID})
	for {
		attemptCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		request, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, s.url+"/v1/cancel", bytes.NewReader(payload))
		if err == nil {
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json")
			response, requestErr := s.client.Do(request)
			if response != nil {
				if requestErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
					var ack struct {
						Drained bool `json:"drained"`
					}
					decodeErr := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&ack)
					_ = response.Body.Close()
					cancel()
					if decodeErr == nil && ack.Drained {
						return
					}
				} else {
					_ = response.Body.Close()
				}
			}
		}
		cancel()
		emit(5, "music-mir has not confirmed drain; retrying while GPU admission remains held")
		time.Sleep(time.Second)
	}
}

func (s *moduleState) digest(ctx context.Context, key string) (string, error) {
	reader, _, err := s.blobs.Get(ctx, key)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

type mirResponse struct {
	ResourceVersion string             `json:"resource_version"`
	Model           string             `json:"model"`
	BackendRevision string             `json:"backend_revision"`
	Analysis        json.RawMessage    `json:"analysis,omitempty"`
	Stems           map[string]mirStem `json:"stems,omitempty"`
	Timing          json.RawMessage    `json:"timing,omitempty"`
	Confidence      json.RawMessage    `json:"confidence,omitempty"`
	Alternatives    json.RawMessage    `json:"alternatives,omitempty"`
}

type mirStem struct {
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	MediaType string `json:"media_type"`
}

type analysisResult struct {
	JobID           string            `json:"job_id"`
	Operation       string            `json:"operation"`
	SourceBlobKey   string            `json:"source_blob_key"`
	SourceSHA256    string            `json:"source_sha256"`
	Resource        string            `json:"resource"`
	ResourceVersion string            `json:"resource_version,omitempty"`
	Model           string            `json:"model,omitempty"`
	BackendRevision string            `json:"backend_revision,omitempty"`
	CompletedAt     time.Time         `json:"completed_at"`
	Analysis        json.RawMessage   `json:"analysis,omitempty"`
	StemBlobKeys    map[string]string `json:"stem_blob_keys,omitempty"`
	Timing          json.RawMessage   `json:"timing,omitempty"`
	Confidence      json.RawMessage   `json:"confidence,omitempty"`
	Alternatives    json.RawMessage   `json:"alternatives,omitempty"`
}

func safeFilename(value string) bool {
	return value != "" && filepath.Base(value) == value && value != "." && value != ".." && !strings.ContainsAny(value, `/\\`) && !strings.HasPrefix(value, ".")
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func digestMismatch(actual, expected string) error {
	if actual == expected {
		return nil
	}
	return fmt.Errorf("digest mismatch: got %s, expected %s", actual, expected)
}

func safeMediaType(value string) string {
	if value == "audio/wav" || value == "audio/flac" || value == "audio/mpeg" || value == "audio/ogg" {
		return value
	}
	return "application/octet-stream"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
