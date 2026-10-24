package composition

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"

	"music-tools/internal/capacity"
	core "music-tools/internal/composition"
	"music-tools/internal/jobs"
	musicstorage "music-tools/internal/storage"
	"music-tools/internal/styles"
)

func TestNewAdmissionsSnapshotProviderAndEntireStyle(t *testing.T) {
	s, db := testStateWithBrokerDB(t, nil)
	s.JobManager.Close()
	s.JobManager = jobs.New(db, jobs.Config{Runner: func(context.Context, jobs.Job, func(int, string)) (jobs.Result, error) { return jobs.Result{}, nil }})
	if e := s.JobManager.Start(t.Context()); e != nil {
		t.Fatal(e)
	}
	_, e := s.Styles.Create(styles.Style{ID: "song", Name: "Song", Caption: "original club song", Lyrics: "[Verse]\nA new song\n", Params: styles.Params{BPM: 150, Duration: 10}})
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ id, provider string }{{"launch-club", "stable-audio-3"}, {"launch-trap", "ace-step"}, {"song", "ace-step"}} {
		t.Run(tc.id, func(t *testing.T) {
			rec := httptest.NewRecorder()
			s.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"`+tc.id+`","takes":1,"seed":0}`)))
			if rec.Code != http.StatusAccepted {
				t.Fatalf("submit=%d %s", rec.Code, rec.Body.String())
			}
			var accepted struct {
				ID string `json:"job_id"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &accepted)
			job, e := s.JobManager.Wait(t.Context(), accepted.ID)
			if e != nil {
				t.Fatal(e)
			}
			var payload map[string]json.RawMessage
			_ = json.Unmarshal(job.Payload, &payload)
			var provider string
			_ = json.Unmarshal(payload["provider"], &provider)
			if provider != tc.provider || len(payload["resolved_style"]) == 0 {
				t.Fatalf("durable routing/input absent: %s", job.Payload)
			}
			var frozen styles.Compiled
			_ = json.Unmarshal(payload["resolved_style"], &frozen)
			wantVariant := "acestep-v15-turbo"
			if tc.provider == providerSA3 {
				wantVariant = "stable-audio-3-medium"
			}
			if frozen.Params.Variant != wantVariant {
				t.Fatalf("selected variant not frozen into admission snapshot: %+v", frozen)
			}
		})
	}
}

func TestSA3UnsupportedControlsRefusedBeforeSubmission(t *testing.T) {
	s, db := testStateWithBrokerDB(t, nil)
	s.JobManager.Close()
	s.JobManager = jobs.New(db, jobs.Config{})
	for _, extra := range []string{`,"shift":3`, `,"duration":121`} {
		rec := httptest.NewRecorder()
		s.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"launch-club","takes":1`+extra+`}`)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("unsupported SA3input accepted: %d %s", rec.Code, rec.Body.String())
		}
	}
	_, e := s.Styles.Create(styles.Style{ID: "unsupported", Name: "Unsupported", Caption: "club", Params: styles.Params{BPM: 150, Duration: 10}})
	if e != nil {
		t.Fatal(e)
	}
	rec := httptest.NewRecorder()
	s.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"unsupported","takes":1}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("SA3 structured BPM silently ignored: %d %s", rec.Code, rec.Body.String())
	}
}

// The fixture preserves the actual successful resource response's shape/model
// closure; test-specific request/seed/audio identities are rebuilt explicitly.
func sa3Server(t *testing.T, mutate func(map[string]any), secondFails bool) (*httptest.Server, *[]byte, *int) {
	t.Helper()
	fixture, e := os.ReadFile("testdata/sa3-native-shape.json")
	if e != nil {
		t.Fatal(e)
	}
	var last []byte
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"model_loaded":true,"model_id":"stabilityai/stable-audio-3-medium","model_revision":"d4d3579395a8ac0df4ab13b5b1d7138ce7290f81","device":"cuda","error":null}`)
			return
		}
		if r.Method == http.MethodPost {
			calls++
			if secondFails && calls == 2 {
				http.Error(w, "busy; no fallback", http.StatusServiceUnavailable)
				return
			}
			raw, _ := io.ReadAll(r.Body)
			var req sa3Request
			if e := json.Unmarshal(raw, &req); e != nil {
				t.Error(e)
			}
			frames := int(req.Duration * 44100)
			audio := sa3TestWAV(frames)
			var receipt map[string]any
			_ = json.Unmarshal(fixture, &receipt)
			receipt["request_sha256"] = fmt.Sprintf("%x", sha256.Sum256(raw))
			receipt["effective_request"] = req
			sample := receipt["sampling"].(map[string]any)
			sample["seed"] = req.Seed
			sample["steps"] = req.Steps
			sample["cfg_scale"] = req.CFG
			output := receipt["output"].(map[string]any)
			output["bytes"] = len(audio)
			output["frames"] = frames
			output["duration_seconds"] = req.Duration
			output["sha256"] = fmt.Sprintf("%x", sha256.Sum256(audio))
			if mutate != nil {
				mutate(receipt)
			}
			last, _ = json.Marshal(receipt)
			w.Header().Set("Content-Type", "application/json")
			w.Write(last)
			return
		}
		audio := sa3TestWAV(44100)
		w.Header().Set("Content-Type", "audio/wav")
		w.Write(audio)
	}))
	return server, &last, &calls
}

func TestSA3DurableBatchRetainsExactReceiptAndSurvivesStyleDeletion(t *testing.T) {
	s, db := testStateWithBrokerDB(t, nil)
	server, raw, calls := sa3Server(t, nil, false)
	defer server.Close()
	s.SA3URL = server.URL
	s.HTTP = server.Client()
	compiled, _ := s.Styles.Compile("launch-club")
	compiled.StyleID = "later-deleted"
	if _, e := s.Styles.Create(styles.Style{ID: compiled.StyleID, Name: "Soon deleted", Caption: compiled.Caption, Params: compiled.Params}); e != nil {
		t.Fatal(e)
	}
	if e := s.Styles.Delete(compiled.StyleID); e != nil {
		t.Fatal(e)
	}
	lyrics := compiled.Lyrics
	seed := int64(42)
	payload, _ := json.Marshal(composeJobPayload{composeRequest: composeRequest{StyleID: compiled.StyleID, Takes: 1, Duration: 1, Seed: &seed}, Provider: providerSA3, ResolvedStyle: &compiled, ResolvedLyrics: &lyrics})
	result, e := s.runDurableJob(t.Context(), jobs.Job{ID: "durable-sa3", Payload: payload}, func(int, string) {})
	if e != nil {
		t.Fatal(e)
	}
	if *calls != 1 || result.Meta["model_id"] != sa3Model || result.Meta["lyrics_sent"] != "false" {
		t.Fatalf("wrong provider/intent: %+v", result)
	}
	var profiles map[string]sa3Profile
	if e = json.Unmarshal([]byte(result.Meta["generation_profiles"]), &profiles); e != nil {
		t.Fatal(e)
	}
	profile := profiles["durable-sa3-take-01"]
	if profile.ReceiptSHA256 != fmt.Sprintf("%x", sha256.Sum256(*raw)) {
		t.Fatal("exact receipt hash lost")
	}
	// Reopen actual SQL owner and the same durable blob substrate; no live style lookup.
	restored, e := core.NewPoolWithStore(db)
	if e != nil {
		t.Fatal(e)
	}
	s.Pool = restored
	take, ok := restored.Get("durable-sa3-take-01")
	if !ok || take.Provenance.LicenseLane != "restricted" || take.Provenance.AppliedRung != "resource-resident-medium" {
		t.Fatalf("fake/lost ACE identity %+v", take)
	}
	rec := httptest.NewRecorder()
	s.receiptTake(rec, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"id": take.ID}))
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), *raw) || rec.Header().Get("X-Music-Receipt-SHA256") != profile.ReceiptSHA256 {
		t.Fatalf("retained receipt lost: %d %s", rec.Code, rec.Body.String())
	}
	body, _, e := s.Blobs.Get(t.Context(), take.BlobRef)
	if e != nil {
		t.Fatal(e)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if e = validateSA3WAV(data, 44100); e != nil {
		t.Fatal(e)
	}
}

func TestSA3MismatchedReceiptOrAudioNeverPublishesTake(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"request hash": func(j map[string]any) { j["request_sha256"] = strings.Repeat("0", 64) },
		"model":        func(j map[string]any) { j["model"].(map[string]any)["model_revision"] = "wrong" },
		"seed":         func(j map[string]any) { j["sampling"].(map[string]any)["seed"] = 999 },
		"steps":        func(j map[string]any) { j["sampling"].(map[string]any)["steps"] = 31 },
		"audio hash":   func(j map[string]any) { j["output"].(map[string]any)["sha256"] = strings.Repeat("0", 64) },
		"format":       func(j map[string]any) { j["output"].(map[string]any)["subtype"] = "PCM_16" },
		"source":       func(j map[string]any) { j["runtime"].(map[string]any)["source_files"] = map[string]string{} },
		"changed recipe": func(j map[string]any) {
			j["runtime"].(map[string]any)["declared_composed_sha256"] = strings.Repeat("1", 64)
		},
		"changed code": func(j map[string]any) {
			j["runtime"].(map[string]any)["source_files"].(map[string]any)["server.py"] = strings.Repeat("1", 64)
		},
		"compile":                 func(j map[string]any) { j["runtime"].(map[string]any)["torch_compile"] = true },
		"missing compile witness": func(j map[string]any) { delete(j["runtime"].(map[string]any), "torch_compile") },
		"dtype":                   func(j map[string]any) { j["runtime"].(map[string]any)["conditioner_parameter_dtype"] = "torch.float32" },
		"attention": func(j map[string]any) {
			j["runtime"].(map[string]any)["attention"].(map[string]any)["upstream_fa2_bindings_verified"] = false
		},
		"truncated prompt": func(j map[string]any) { j["sampling"].(map[string]any)["prompt_truncated"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			s, _ := testStateWithBrokerDB(t, nil)
			server, _, _ := sa3Server(t, mutate, false)
			defer server.Close()
			s.HTTP = server.Client()
			s.SA3URL = server.URL
			c, _ := s.Styles.Compile("launch-club")
			seed := int64(3)
			payload, _ := json.Marshal(composeJobPayload{composeRequest: composeRequest{StyleID: c.StyleID, Takes: 1, Duration: 1, Seed: &seed}, Provider: providerSA3, ResolvedStyle: &c})
			if _, e := s.runDurableJob(t.Context(), jobs.Job{ID: "bad", Payload: payload}, func(int, string) {}); e == nil {
				t.Fatal("mismatched output accepted")
			}
			if len(s.Pool.ListFiltered("", "")) != 0 || len(s.Blobs.Entries()) != 0 {
				t.Fatal("invalid receipt/audio reached storage or take pool")
			}
		})
	}
}

func TestSA3PartialFailureRetainsCompletedFactsWithoutACEClaim(t *testing.T) {
	s, _ := testStateWithBrokerDB(t, nil)
	server, _, _ := sa3Server(t, nil, true)
	defer server.Close()
	s.HTTP = server.Client()
	s.SA3URL = server.URL
	c, _ := s.Styles.Compile("launch-club")
	seed := int64(3)
	payload, _ := json.Marshal(composeJobPayload{composeRequest: composeRequest{StyleID: c.StyleID, Takes: 2, Duration: 1, Seed: &seed}, Provider: providerSA3, ResolvedStyle: &c})
	result, e := s.runDurableJob(t.Context(), jobs.Job{ID: "partial", Payload: payload}, func(int, string) {})
	if e == nil || !strings.Contains(e.Error(), "503") || result.Meta["composition_provider"] != providerSA3 || len(s.Pool.ListFiltered("", "")) != 1 {
		t.Fatalf("partial facts lost or fallback used: %+v %v", result, e)
	}
}

func sa3TestWAV(frames int) []byte {
	audio := make([]byte, 44+frames*6)
	copy(audio, "RIFF")
	binary.LittleEndian.PutUint32(audio[4:], uint32(len(audio)-8))
	copy(audio[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(audio[16:], 16)
	binary.LittleEndian.PutUint16(audio[20:], 1)
	binary.LittleEndian.PutUint16(audio[22:], 2)
	binary.LittleEndian.PutUint32(audio[24:], 44100)
	binary.LittleEndian.PutUint32(audio[28:], 264600)
	binary.LittleEndian.PutUint16(audio[32:], 6)
	binary.LittleEndian.PutUint16(audio[34:], 24)
	copy(audio[36:], "data")
	binary.LittleEndian.PutUint32(audio[40:], uint32(frames*6))
	return audio
}

type forbiddenACEBroker struct{ calls atomic.Int32 }

func (b *forbiddenACEBroker) Claim(context.Context, string, uint64, uint64) (capacity.Verdict, error) {
	b.calls.Add(1)
	return capacity.Verdict{}, errors.New("ACE admission must not run for SA3")
}
func (b *forbiddenACEBroker) Release(context.Context, capacity.ClaimRef) error {
	b.calls.Add(1)
	return nil
}
func (b *forbiddenACEBroker) Heartbeat(context.Context, capacity.ClaimRef) error {
	b.calls.Add(1)
	return nil
}

func TestSA3RealJobAdmissionBypassesACEClaimsAndRetainsDurableProvenance(t *testing.T) {
	broker := &forbiddenACEBroker{}
	s, _ := testStateWithBrokerDB(t, broker)
	server, _, _ := sa3Server(t, nil, false)
	defer server.Close()
	s.SA3URL = server.URL
	s.HTTP = server.Client()
	rec := httptest.NewRecorder()
	s.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"launch-club","takes":1,"duration":1,"seed":0}`)))
	var accepted struct {
		ID string `json:"job_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &accepted)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("submit %d %s", rec.Code, rec.Body.String())
	}
	job, e := s.JobManager.Wait(t.Context(), accepted.ID)
	if e != nil {
		t.Fatal(e)
	}
	if job.State != jobs.StateSucceeded || broker.calls.Load() != 0 || job.Meta["capacity_ownership"] != "resource:stable-audio-3" || job.Meta["capacity_admitted_rung"] != "" {
		t.Fatalf("wrong provider/capacity: %+v calls%d", job, broker.calls.Load())
	}
	if !strings.Contains(job.Meta["generation_profiles"], "receipt_sha256") {
		t.Fatal("durable receipt reference missing")
	}
}

func TestSA3ReadinessAndPermissiveRefusalsDoNotFallBack(t *testing.T) {
	for _, permissive := range []bool{false, true} {
		t.Run(fmt.Sprint(permissive), func(t *testing.T) {
			broker := &forbiddenACEBroker{}
			s, _ := testStateWithBrokerDB(t, broker)
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "model not loaded", http.StatusServiceUnavailable)
			}))
			defer server.Close()
			s.HTTP = server.Client()
			s.SA3URL = server.URL
			s.PermissiveOnly = permissive
			rec := httptest.NewRecorder()
			s.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"launch-club","takes":1,"duration":1}`)))
			if permissive {
				if rec.Code != http.StatusBadRequest || calls.Load() != 0 {
					t.Fatalf("restricted job was admitted: %d", rec.Code)
				}
				return
			}
			var accepted struct {
				ID string `json:"job_id"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &accepted)
			job, e := s.JobManager.Wait(t.Context(), accepted.ID)
			if e != nil || job.State != jobs.StateFailed || !strings.Contains(job.Error, "vrooli resource start stable-audio-3") || broker.calls.Load() != 0 || calls.Load() != 1 {
				t.Fatalf("readiness/fallback mismatch: %+v %v calls%d", job, e, calls.Load())
			}
		})
	}
}

func TestSA3CancellationAbortsTransportAndNeverPublishesTake(t *testing.T) {
	s, _ := testStateWithBrokerDB(t, nil)
	entered := make(chan struct{})
	drained := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"model_loaded":true,"model_id":"stabilityai/stable-audio-3-medium","model_revision":"d4d3579395a8ac0df4ab13b5b1d7138ce7290f81","device":"cuda"}`)
			return
		}
		io.Copy(io.Discard, r.Body)
		close(entered)
		<-r.Context().Done()
		close(drained)
	}))
	defer server.Close()
	s.SA3URL = server.URL
	s.HTTP = server.Client()
	rec := httptest.NewRecorder()
	s.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"launch-club","takes":1,"duration":1}`)))
	var accepted struct {
		ID string `json:"job_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &accepted)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("resource POST not entered")
	}
	if e := s.JobManager.Cancel(accepted.ID); e != nil {
		t.Fatal(e)
	}
	job, e := s.JobManager.Wait(t.Context(), accepted.ID)
	if e != nil || job.State != jobs.StateCanceled {
		t.Fatalf("cancel outcome %+v %v", job, e)
	}
	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("resource transport did not see cancellation")
	}
	if len(s.Blobs.Entries()) != 0 || len(s.Pool.ListFiltered("", "")) != 0 {
		t.Fatal("canceled output advertised/stored")
	}
}

type publicationFailureStore struct {
	blobstore.BlobStore
	prefix string
}

func (b publicationFailureStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if strings.HasPrefix(key, b.prefix) {
		return errors.New("injected publication failure")
	}
	return b.BlobStore.Put(ctx, key, r, mime)
}

func TestSA3PublicationFailuresPreserveSourceAndNeverAdvertiseReadyTake(t *testing.T) {
	for _, prefix := range []string{"generation-receipts/", "out/"} {
		t.Run(prefix, func(t *testing.T) {
			s, _ := testStateWithBrokerDB(t, nil)
			root := t.TempDir()
			s.Blobs = musicstorage.NewWithBlobStore(publicationFailureStore{blobstore.NewFilesystemBlobStore(root), prefix}, root)
			server, raw, _ := sa3Server(t, nil, false)
			defer server.Close()
			s.HTTP = server.Client()
			s.SA3URL = server.URL
			c, _ := s.Styles.Compile("launch-club")
			seed := int64(3)
			payload, _ := json.Marshal(composeJobPayload{composeRequest: composeRequest{StyleID: c.StyleID, Takes: 1, Duration: 1, Seed: &seed}, Provider: providerSA3, ResolvedStyle: &c})
			result, e := s.runDurableJob(t.Context(), jobs.Job{ID: "publish", Payload: payload}, func(int, string) {})
			if e == nil || !strings.Contains(e.Error(), "retained") || result.Meta["completed_take_ids"] != "" || len(s.Pool.ListFiltered("", "")) != 0 {
				t.Fatalf("publication failure faked ready: %+v %v", result, e)
			}
			if prefix == "generation-receipts/" {
				if len(s.Blobs.Entries()) != 0 {
					t.Fatal("audio stored before receipt failure")
				}
			} else {
				data, e := os.ReadFile(filepath.Join(root, "generation-receipts/publish-take-01.json"))
				if e != nil || !bytes.Equal(data, *raw) {
					t.Fatalf("receipt for original completed resource output lost %v", e)
				}
				entries := s.Blobs.Entries()
				if len(entries) != 1 || !entries[0].Protected || entries[0].Regenerable {
					t.Fatalf("retained receipt is reclaimable: %+v", entries)
				}
			}
		})
	}
}
