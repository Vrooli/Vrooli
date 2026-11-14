package composition

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-tools/internal/jobs"
	"music-tools/internal/models"
	"music-tools/internal/styles"
)

func songTestModel(t *testing.T, variant string) models.Model {
	t.Helper()
	model, err := declaredCompositionModel("songgen", variant)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func songTestServer(t *testing.T, model models.Model, mutate func(*songReceipt)) *httptest.Server {
	t.Helper()
	audio := sa3TestWAV(4410)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			json.NewEncoder(w).Encode(map[string]any{"protocol": "song-v1", "catalog_sha256": model.Runtime.CatalogSHA256, "artifact_sha256": model.Runtime.ArtifactSHA256,
				"profiles": map[string]any{model.Runtime.Profile: map[string]any{"available": true, "model_id": model.ID, "revision": model.Revision}}})
			return
		}
		if r.Method == http.MethodPost {
			raw, _ := io.ReadAll(r.Body)
			var request songRequest
			if err := json.Unmarshal(raw, &request); err != nil {
				t.Error(err)
			}
			if !strings.Contains(request.Lyrics, "authored words") {
				t.Error("authored lyrics did not reach native input")
			}
			receipt := songReceipt{Version: 1, Status: "completed", Operation: strings.Repeat("a", 32), RequestSHA: fmt.Sprintf("%x", sha256.Sum256(raw)), Effective: request, CatalogSHA: model.Runtime.CatalogSHA256, ArtifactSHA: model.Runtime.ArtifactSHA256}
			receipt.Model.ID, receipt.Model.Revision, receipt.Model.Source, receipt.Model.License = model.ID, model.Revision, model.Runtime.SourceRevision, model.License
			receipt.Worker.Seed, receipt.Worker.ModelID, receipt.Worker.Revision = &request.Seed, model.ID, model.Revision
			receipt.Worker.FilesVerified, receipt.Worker.Versions = model.Runtime.WeightFiles, model.Runtime.Versions
			receipt.Worker.Sampling.Guidance, receipt.Worker.Sampling.Steps, receipt.Worker.Sampling.DecoderSeed = &request.Guidance, &request.Steps, &request.Seed
			receipt.Output.SHA, receipt.Output.Bytes, receipt.Output.Frames = fmt.Sprintf("%x", sha256.Sum256(audio)), int64(len(audio)), 4410
			receipt.Output.Rate, receipt.Output.Channels, receipt.Output.Duration = 44100, 2, 0.1
			if mutate != nil {
				mutate(&receipt)
			}
			json.NewEncoder(w).Encode(receipt)
			return
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.Write(audio)
	}))
}

func TestSongModelsUseDurablePipelineWithoutACEClaims(t *testing.T) {
	for _, variant := range []string{"minimax-music3-nf4", "heartmula-3b", "yue2-q8"} {
		t.Run(variant, func(t *testing.T) {
			model := songTestModel(t, variant)
			server := songTestServer(t, model, nil)
			defer server.Close()
			t.Setenv("SONGGEN_URL", server.URL)
			broker := &forbiddenACEBroker{}
			state, _ := testStateWithBrokerDB(t, broker)
			state.HTTP = server.Client()
			_, err := state.Styles.Create(styles.Style{ID: "song-native", Name: "Authored song", Caption: "bright syncopated dance pop", Lyrics: "[Chorus]\nauthored words", Params: styles.Params{Duration: 1, Variant: variant}})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			state.compose(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"song-native","takes":1,"seed":0}`)))
			if recorder.Code != http.StatusAccepted {
				t.Fatalf("admission %d: %s", recorder.Code, recorder.Body.String())
			}
			var accepted struct {
				ID string `json:"job_id"`
			}
			json.Unmarshal(recorder.Body.Bytes(), &accepted)
			job, err := state.JobManager.Wait(t.Context(), accepted.ID)
			if err != nil {
				t.Fatal(err)
			}
			if job.State != jobs.StateSucceeded || broker.calls.Load() != 0 || job.Meta["capacity_ownership"] != "resource:songgen" {
				t.Fatalf("wrong result/claim ownership: %+v claims=%d", job, broker.calls.Load())
			}
			var payload composeJobPayload
			json.Unmarshal(job.Payload, &payload)
			if payload.ModelSnapshot == nil || payload.ModelSnapshot.ID != model.ID || payload.ResolvedStyle.Params.Variant != variant {
				t.Fatalf("missing immutable model: %s", job.Payload)
			}
			if job.Meta["lyrics_sent_sha256"] == "" || !strings.Contains(job.Meta["generation_profiles"], "actual_duration_seconds") {
				t.Fatalf("missing actual song provenance: %+v", job.Meta)
			}
			takes := state.Pool.ListFiltered("", "")
			if len(takes) != 1 || takes[0].Provenance.ModelID != model.ID {
				t.Fatalf("wrong published model: %+v", takes)
			}
			body, mime, err := state.Blobs.Get(t.Context(), "generation-receipts/"+takes[0].ID+".json")
			if err != nil {
				t.Fatal(err)
			}
			body.Close()
			if mime != "application/json" {
				t.Fatal("native receipt was not retained")
			}
		})
	}
}

func TestSongReceiptMismatchNeverPublishesTake(t *testing.T) {
	for _, field := range []string{"seed", "model", "catalog", "artifact", "versions", "frames", "hash", "guidance", "steps", "decoder_seed"} {
		t.Run(field, func(t *testing.T) {
			model := songTestModel(t, "minimax-music3-nf4")
			server := songTestServer(t, model, func(receipt *songReceipt) {
				switch field {
				case "seed":
					wrong := uint32(99)
					receipt.Worker.Seed = &wrong
				case "model":
					receipt.Model.ID = "other"
				case "catalog":
					receipt.CatalogSHA = strings.Repeat("b", 64)
				case "artifact":
					receipt.ArtifactSHA = strings.Repeat("b", 64)
				case "versions":
					receipt.Worker.Versions = map[string]string{"torch": "other"}
				case "frames":
					receipt.Output.Frames++
				case "guidance":
					receipt.Worker.Sampling.Guidance = nil
				case "steps":
					receipt.Worker.Sampling.Steps = nil
				case "decoder_seed":
					wrong := uint32(19)
					receipt.Worker.Sampling.DecoderSeed = &wrong
				case "hash":
					receipt.Output.SHA = strings.Repeat("b", 64)
				}
			})
			defer server.Close()
			t.Setenv("SONGGEN_URL", server.URL)
			state, _ := testStateWithBrokerDB(t, nil)
			state.HTTP = server.Client()
			compiled := styles.Compiled{Caption: "authored pop", Lyrics: "[Chorus]\nauthored words", Params: styles.Params{Variant: model.Variant}}
			request := composeJobPayload{composeRequest: composeRequest{StyleID: "test", Takes: 1, Duration: 1}, Provider: model.Engine, ResolvedStyle: &compiled, ModelSnapshot: &model}
			raw, _ := json.Marshal(request)
			if _, err := state.runDurableJob(context.Background(), jobs.Job{ID: "song-test", Payload: raw}, func(int, string) {}); err == nil {
				t.Fatal("mismatched native receipt accepted")
			}
			if len(state.Pool.ListFiltered("", "")) != 0 || len(state.Blobs.Entries()) != 0 {
				t.Fatal("unverified take or receipt was published")
			}
		})
	}
}

func TestSongCapabilitiesAndSnapshotDoNotFallBack(t *testing.T) {
	model := songTestModel(t, "heartmula-3b")
	compiled := styles.Compiled{Caption: "authored pop", Lyrics: "authored words", Params: styles.Params{Variant: model.Variant}}
	if err := validateProviderInput(model.Engine, compiled, composeRequest{Duration: 1}, true); err != nil {
		t.Fatal(err)
	}
	compiled.Params.Steps = 30
	if err := validateProviderInput(model.Engine, compiled, composeRequest{Duration: 1}, false); err == nil {
		t.Fatal("unsupported denoising control silently ignored")
	}
	model.Revision = "other"
	if err := validSongSnapshot(model); err == nil {
		t.Fatal("stale model identity accepted")
	}
}

func TestSongMissingDataHasExactOwnerRecoveryCommand(t *testing.T) {
	model := songTestModel(t, "heartmula-3b")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"profiles": map[string]any{model.Runtime.Profile: map[string]any{"available": false}}})
	}))
	defer server.Close()
	t.Setenv("SONGGEN_URL", server.URL)
	state, _ := testStateWithBrokerDB(t, nil)
	state.HTTP = server.Client()
	err := state.songReady(t.Context(), model)
	if err == nil || !strings.Contains(err.Error(), "vrooli resource acquisition data songgen --group heartmula-3b") {
		t.Fatalf("missing recovery: %v", err)
	}
}
