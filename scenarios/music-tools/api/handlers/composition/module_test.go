package composition

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"music-tools/internal/capacity"

	"github.com/vrooli/api-core/blobstore"
	musicstorage "music-tools/internal/storage"

	"github.com/gorilla/mux"
	testdb "github.com/vrooli/api-core/databasetest"
	compositionpb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition"
	stylespb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/styles"
	"google.golang.org/protobuf/encoding/protojson"
	core "music-tools/internal/composition"
	jobdomain "music-tools/internal/jobs"
	stylesdomain "music-tools/internal/styles"
)

func TestStyleCompileExposesLyricsOrInstrumentalDefault(t *testing.T) {
	for name, lyrics := range map[string]string{"empty legacy default": "", "song": "[Verse]\nOur original song\n"} {
		t.Run(name, func(t *testing.T) {
			state := &ModuleState{Styles: stylesdomain.NewStore()}
			body, _ := json.Marshal(map[string]any{"id": "song", "name": "Song", "caption": "energetic pop", "bpm": 150, "duration": 30, "lyrics": lyrics})
			rec := httptest.NewRecorder()
			state.createStyle(rec, httptest.NewRequest(http.MethodPost, "/api/v1/styles", strings.NewReader(string(body))))
			if rec.Code != http.StatusCreated {
				t.Fatalf("create=%d %s", rec.Code, rec.Body.String())
			}
			rec = httptest.NewRecorder()
			state.compileStyle(rec, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/styles/song/compile", nil), map[string]string{"id": "song"}))
			var compiled stylespb.CompiledStyle
			if err := protojson.Unmarshal(rec.Body.Bytes(), &compiled); err != nil || compiled.Style == nil {
				t.Fatalf("compile=%s err=%v", rec.Body.String(), err)
			}
			want := lyrics
			if want == "" {
				want = "[instrumental]"
			}
			if compiled.Style.Lyrics != want {
				t.Fatalf("lyrics=%q want=%q", compiled.Style.Lyrics, want)
			}
		})
	}
}

func TestComposeChoosesFreshBaseOnceAndPreservesExplicitSeedAndShift(t *testing.T) {
	state, d := testStateWithBrokerDB(t, nil)
	state.JobManager.Close()
	state.JobManager = jobdomain.New(d, jobdomain.Config{Runner: func(context.Context, jobdomain.Job, func(int, string)) (jobdomain.Result, error) {
		return jobdomain.Result{}, nil
	}})
	if err := state.JobManager.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	submit := func(extra string) map[string]json.RawMessage {
		t.Helper()
		rec := httptest.NewRecorder()
		state.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"launch-trap","takes":1,"duration":1`+extra+`}`)))
		if rec.Code != http.StatusAccepted {
			t.Fatalf("submit=%d %s", rec.Code, rec.Body.String())
		}
		var accepted struct {
			ID string `json:"job_id"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &accepted)
		job, err := state.JobManager.Wait(t.Context(), accepted.ID)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	first, second := submit(""), submit("")
	var one, two int64
	if len(first["seed"]) == 0 || json.Unmarshal(first["seed"], &one) != nil || len(second["seed"]) == 0 || json.Unmarshal(second["seed"], &two) != nil {
		t.Fatalf("omitted seed not snapshotted: %v %v", first, second)
	}
	if one == two {
		t.Fatal("independent admissions reused the same effective base")
	}
	for i := 0; i < 2; i++ {
		p := submit(`,"seed":0,"shift":3`)
		if string(p["seed"]) != "0" || string(p["shift"]) != "3" {
			t.Fatalf("explicit zero/shift lost: %v", p)
		}
	}
	if _, present := first["shift"]; present {
		t.Fatal("omitted shift replaced resource default")
	}
	for _, raw := range []string{"0", "-1", "6", "1e309"} {
		rec := httptest.NewRecorder()
		state.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"launch-trap","takes":1,"shift":`+raw+`}`)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("shift=%s status=%d", raw, rec.Code)
		}
	}
}

func TestSeedAndProfileReceiptsRequiredBeforeIngestion(t *testing.T) {
	for _, tc := range []struct {
		name, key, value string
		valid            bool
	}{
		{"valid", "", "", true},
		{"missing seed", "X-ACE-Step-Seed", "", false},
		{"wrong seed", "X-ACE-Step-Seed", "43", false},
		{"rounded seed", "X-ACE-Step-Seed", "42.0", false},
		{"overflow seed", "X-ACE-Step-Seed", "4294967296", false},
		{"missing planner", "X-ACE-Step-Planner", "", false},
		{"unknown planner", "X-ACE-Step-Planner", "unknown", false},
		{"wrong steps", "X-ACE-Step-Inference-Steps", "9", false},
		{"zero steps", "X-ACE-Step-Inference-Steps", "0", false},
		{"missing guidance", "X-ACE-Step-Guidance-Scale", "", false},
		{"nan guidance", "X-ACE-Step-Guidance-Scale", "NaN", false},
		{"invalid guidance", "X-ACE-Step-Guidance-Scale", "21", false},
		{"missing shift", "X-ACE-Step-Shift", "", false},
		{"infinite shift", "X-ACE-Step-Shift", "Inf", false},
		{"invalid shift", "X-ACE-Step-Shift", "0", false},
		{"orphan code hash", "X-ACE-Step-LM-Code-SHA256", strings.Repeat("a", 64), false},
		{"orphan code count", "X-ACE-Step-LM-Code-Count", "12", false},
		{"wrong variant", "X-ACE-Step-Model-ID", "ACE-Step/Ace-Step1.5:acestep-v15-sft", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := testState(t)
			resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeGenerationProfile(t, w, r)
				var payload map[string]any
				_ = json.NewDecoder(r.Body).Decode(&payload)
				if payload["expected_variant"] != "acestep-v15-turbo" {
					t.Errorf("style variant was not enforced: %v", payload["expected_variant"])
				}
				w.Header().Set("X-ACE-Step-Model-ID", "ACE-Step/Ace-Step1.5:acestep-v15-turbo")
				w.Header().Set("X-ACE-Step-License-Lane", "permissive")
				w.Header().Set("X-ACE-Step-Profile-Rung", "full")
				w.Header().Set("X-ACE-Step-Seed", "42")
				w.Header().Set("X-ACE-Step-Planner", "acestep-5Hz-lm-0.6B")
				w.Header().Set("X-ACE-Step-Inference-Steps", "8")
				w.Header().Set("X-ACE-Step-Guidance-Scale", "1")
				w.Header().Set("X-ACE-Step-Shift", "3")
				if tc.key != "" {
					w.Header().Set(tc.key, tc.value)
				}
				_, _ = w.Write(wavFixture())
			}))
			defer resource.Close()
			state.HTTP, state.ResourceURL = resource.Client(), resource.URL
			planned, _ := core.PlanBatch(core.BatchRequest{JobID: "receipt", StyleID: "launch-trap", Caption: "drums", Takes: 1, Duration: 1})
			planned[0].Provenance.Seed = 42
			compiled, _ := state.Styles.Compile("launch-trap")
			_, err := state.generateTake(t.Context(), &planned[0], composeRequest{Duration: 1}, compiled, "")
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
			if !tc.valid && len(state.Blobs.Entries()) != 0 {
				t.Fatal("unqualified receipt ingested owned output")
			}
		})
	}
}

func TestDurableLyricsSnapshotAndLegacyQueuedCompatibility(t *testing.T) {
	for _, snapshot := range []bool{true, false} {
		t.Run(fmt.Sprint(snapshot), func(t *testing.T) {
			const original = "[Verse]\nThe future is ours\nCafé sunrise\n"
			const changed = "[Chorus]\nA different song after recreation\n"
			state, d := testStateWithBrokerDB(t, nil)
			state.JobManager.Close()
			started, release := make(chan struct{}), make(chan struct{})
			state.JobManager = jobdomain.New(d, jobdomain.Config{Runner: func(ctx context.Context, j jobdomain.Job, emit func(int, string)) (jobdomain.Result, error) {
				close(started)
				select {
				case <-release:
				case <-ctx.Done():
					return jobdomain.Result{}, ctx.Err()
				}
				return state.runDurableJob(ctx, j, emit)
			}})
			if err := state.JobManager.Start(t.Context()); err != nil {
				t.Fatal(err)
			}
			create := func(text string) {
				t.Helper()
				body, _ := json.Marshal(map[string]any{"id": "song", "name": "Song", "caption": "energetic vocal pop", "bpm": 150, "duration": 1, "lyrics": text})
				if _, err := state.Styles.Import(body); err != nil {
					t.Fatal(err)
				}
			}
			create(original)
			wireLyrics := make(chan string, 2)
			resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeGenerationProfile(t, w, r)
				var payload struct {
					Lyrics string `json:"lyrics"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				wireLyrics <- payload.Lyrics
				w.Header().Set("Content-Type", "audio/wav")
				w.Header().Set("X-ACE-Step-Model-ID", "fake-model:acestep-v15-turbo")
				w.Header().Set("X-ACE-Step-License-Lane", "permissive")
				w.Header().Set("X-ACE-Step-Profile-Rung", "full")
				_, _ = w.Write(wavFixture())
			}))
			defer resource.Close()
			state.HTTP, state.ResourceURL = resource.Client(), resource.URL
			var id string
			if snapshot {
				response := httptest.NewRecorder()
				state.compose(response, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"song","takes":2,"duration":1}`)))
				if response.Code != http.StatusAccepted {
					t.Fatalf("submit=%d %s", response.Code, response.Body.String())
				}
				var accepted struct {
					JobID string `json:"job_id"`
				}
				_ = json.Unmarshal(response.Body.Bytes(), &accepted)
				id = accepted.JobID
			} else {
				created, err := state.JobManager.Submit(t.Context(), jobdomain.Spec{Operation: "compose", Lane: jobdomain.LaneGPU, Payload: []byte(`{"style_id":"song","takes":2,"duration":1}`)})
				if err != nil {
					t.Fatal(err)
				}
				id = created.ID
			}
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("queued runner did not start")
			}
			if err := state.Styles.Delete("song"); err != nil {
				t.Fatal(err)
			}
			create(changed)
			close(release)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			done, err := state.JobManager.Wait(ctx, id)
			if err != nil || done.State != jobdomain.StateSucceeded {
				t.Fatalf("job=%+v err=%v", done, err)
			}
			want := original
			source := "style-snapshot"
			if !snapshot {
				want = "[instrumental]"
				source = "legacy-instrumental-default"
			}
			for i := 0; i < 2; i++ {
				if got := <-wireLyrics; got != want {
					t.Fatalf("resource lyrics=%q want=%q", got, want)
				}
			}
			wantHash := fmt.Sprintf("%x", sha256.Sum256([]byte(want)))
			if done.Meta["lyrics_sent_sha256"] != wantHash {
				t.Fatalf("actual sent identity=%v want=%s", done.Meta, wantHash)
			}
			reloaded, err := jobdomain.New(d, jobdomain.Config{}).Get(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			wire := wireJob(reloaded)
			if wire.Meta["lyrics_sha256"] != wantHash || wire.Meta["lyrics_source"] != source {
				t.Fatalf("persisted request identity=%v", wire.Meta)
			}
			if _, exists := wire.Meta["lyrics_as_sent"]; exists {
				t.Fatal("default job metadata copied full lyrics")
			}
			for _, take := range state.Pool.ListFiltered("song", id) {
				if take.JobID != id {
					t.Fatal("take lost durable lyrics provenance link")
				}
			}
		})
	}
}

func TestJobLyricsMetadataIsCompactIntentAndDoesNotMutateOwner(t *testing.T) {
	const lyrics = "[Verse]\nOur original café song\n"
	for _, state := range []jobdomain.State{jobdomain.StateQueued, jobdomain.StateRunning, jobdomain.StateFailed, jobdomain.StateCanceled, jobdomain.StateSucceeded} {
		t.Run(string(state), func(t *testing.T) {
			payload, _ := json.Marshal(map[string]any{"style_id": "song", "takes": 1, "duration": 30, "resolved_lyrics": lyrics})
			owner := jobdomain.Job{Operation: "compose", State: state, Payload: payload, Meta: map[string]string{"capacity_admitted_rung": "full"}}
			wire := wireJob(owner)
			if wire.Meta["lyrics_intent"] != "lyrics-provided" || wire.Meta["lyrics_source"] != "style-snapshot" || wire.Meta["lyrics_sha256"] != lyricsHash(lyrics) || wire.Meta["lyrics_bytes"] != fmt.Sprint(len(lyrics)) {
				t.Fatalf("request metadata=%v", wire.Meta)
			}
			if _, exists := wire.Meta["lyrics_sent_sha256"]; exists {
				t.Fatal("state alone invented actual sent evidence")
			}
			for _, value := range wire.Meta {
				if value == lyrics {
					t.Fatal("default metadata leaked full lyric text")
				}
			}
			wire.Meta["capacity_admitted_rung"] = "changed by consumer"
			if len(owner.Meta) != 1 || owner.Meta["capacity_admitted_rung"] != "full" {
				t.Fatalf("wire mutated stored metadata: %v", owner.Meta)
			}
		})
	}
	for _, payload := range [][]byte{nil, []byte(`{`), []byte(`null`), []byte(`{}`)} {
		wire := wireJob(jobdomain.Job{Operation: "compose", Payload: payload})
		if _, exists := wire.Meta["lyrics_intent"]; exists {
			t.Fatalf("invalid payload invented intent: %s => %v", payload, wire.Meta)
		}
	}
}

func TestStyleReplenishmentRetainsVocalIntentAndSeparatePoolID(t *testing.T) {
	const lyrics = "[Chorus]\nWe keep our own rhythm\n"
	state := testState(t)
	style, _ := json.Marshal(map[string]any{"id": "vocal-song", "name": "Vocal Song", "caption": "energetic pop", "bpm": 150, "duration": 1, "lyrics": lyrics})
	if _, err := state.Styles.Import(style); err != nil {
		t.Fatal(err)
	}
	wireLyrics := make(chan string, 2)
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakeGenerationProfile(t, w, r)
		var payload struct {
			Lyrics string `json:"lyrics"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		wireLyrics <- payload.Lyrics
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("X-ACE-Step-Model-ID", "fake-model:acestep-v15-turbo")
		w.Header().Set("X-ACE-Step-License-Lane", "permissive")
		w.Header().Set("X-ACE-Step-Profile-Rung", "full")
		_, _ = w.Write(wavFixture())
	}))
	defer resource.Close()
	state.HTTP, state.ResourceURL = resource.Client(), resource.URL
	response := httptest.NewRecorder()
	state.configurePool(response, httptest.NewRequest(http.MethodPost, "/api/v1/pool/configure", strings.NewReader(`{"style_id":"vocal-song","target_depth":2,"replenish_below":1}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("configure=%d %s", response.Code, response.Body.String())
	}
	// Configuration sets policy; an empty draw is the existing replenishment trigger.
	_, _, _ = state.Pool.Draw("vocal-song", "test-consumer")
	id := state.Pool.LastReplenishJob("vocal-song")
	if id == "" {
		t.Fatal("replenishment was not queued")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done, err := state.JobManager.Wait(ctx, id)
	if err != nil || done.State != jobdomain.StateSucceeded {
		t.Fatalf("job=%+v err=%v", done, err)
	}
	for i := 0; i < 2; i++ {
		if got := <-wireLyrics; got != lyrics {
			t.Fatalf("replenish lyric=%q want=%q", got, lyrics)
		}
	}
	if wireJob(done).Meta["lyrics_source"] != "style-snapshot" {
		t.Fatal("replenishment lost its durable lyric snapshot")
	}
	if available, _, _, _ := state.Pool.Status("launch-trap"); available != 0 {
		t.Fatal("vocal takes entered a different instrumental style's pool")
	}
	if got := len(state.Pool.ListFiltered("vocal-song", id)); got != 2 {
		t.Fatalf("vocal inventory=%d", got)
	}
}

func TestStyleCreateCompileExportImportPreservesPortableParameters(t *testing.T) {
	for name, fields := range map[string]string{
		"canonical snake case": `"bpm":200,"keyscale":"F# minor","duration":48,"inference_steps":8,"guidance_scale":1.3,"variant":"acestep-v15-turbo"`,
		"canonical camel case": `"bpm":200,"keyscale":"F# minor","duration":48,"inferenceSteps":8,"guidanceScale":1.3,"variant":"acestep-v15-turbo"`,
		"legacy nested":        `"params":{"bpm":200,"key_scale":"F# minor","duration":48,"steps":8,"guidance":1.3,"variant":"acestep-v15-turbo"},"created_at":"2000-01-01T00:00:00Z","updated_at":"2000-01-01T00:00:00Z"`,
	} {
		t.Run(name, func(t *testing.T) {
			state := &ModuleState{Styles: stylesdomain.NewStore()}
			body := `{"id":"portable-create","name":"Portable create","caption":"driving drums","builtin":true,` + fields + `}`
			before := time.Now().UTC()
			created := httptest.NewRecorder()
			state.createStyle(created, httptest.NewRequest(http.MethodPost, "/api/v1/styles", strings.NewReader(body)))
			if created.Code != http.StatusCreated {
				t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
			}
			want := stylesdomain.Params{BPM: 200, KeyScale: "F# minor", Duration: 48, Steps: 8, Guidance: 1.3, Variant: "acestep-v15-turbo"}
			stored, ok := state.Styles.Get("portable-create")
			if !ok || stored.Params != want || stored.Builtin || stored.CreatedAt.Before(before) || stored.CreatedAt != stored.UpdatedAt {
				t.Fatalf("created style lost parameters/server identity: %+v", stored)
			}
			var wire stylespb.Style
			if err := protojson.Unmarshal(created.Body.Bytes(), &wire); err != nil || wire.Bpm != 200 || wire.Duration != 48 || wire.Builtin {
				t.Fatalf("create wire=%+v err=%v", &wire, err)
			}
			compile := func(owner *ModuleState) {
				t.Helper()
				rec := httptest.NewRecorder()
				request := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/styles/portable-create/compile", nil), map[string]string{"id": "portable-create"})
				owner.compileStyle(rec, request)
				var compiled stylespb.CompiledStyle
				if rec.Code != http.StatusOK || protojson.Unmarshal(rec.Body.Bytes(), &compiled) != nil || compiled.Style == nil {
					t.Fatalf("compile status=%d body=%s", rec.Code, rec.Body.String())
				}
				if compiled.CaptionAsSent != "driving drums" || compiled.Style.Bpm != 200 || compiled.Style.Keyscale != "F# minor" || compiled.Style.Duration != 48 || compiled.Style.InferenceSteps != 8 || compiled.Style.GuidanceScale != 1.3 || compiled.Style.Variant != "acestep-v15-turbo" {
					t.Fatalf("compiled parameters changed: %v", &compiled)
				}
			}
			compile(state)
			exported := httptest.NewRecorder()
			state.exportStyle(exported, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/styles/portable-create/export", nil), map[string]string{"id": "portable-create"}))
			if exported.Code != http.StatusOK {
				t.Fatalf("export status=%d", exported.Code)
			}
			other := &ModuleState{Styles: stylesdomain.NewStore()}
			imported := httptest.NewRecorder()
			other.importStyle(imported, httptest.NewRequest(http.MethodPost, "/api/v1/styles/import", strings.NewReader(exported.Body.String())))
			if imported.Code != http.StatusCreated {
				t.Fatalf("import status=%d body=%s", imported.Code, imported.Body.String())
			}
			compile(other)
		})
	}
}

func TestStyleCreateSeparatesDecodeErrorsFromDomainConflicts(t *testing.T) {
	for name, test := range map[string]struct {
		body   string
		status int
	}{
		"malformed":         {`{"id":`, http.StatusBadRequest},
		"unknown parameter": {`{"id":"bad","name":"Bad","caption":"drums","bpm":200,"duration":48,"inference_stepz":8}`, http.StatusBadRequest},
		"missing name":      {`{"id":"bad","caption":"drums","bpm":200,"duration":48}`, http.StatusConflict},
		"builtin collision": {`{"id":"launch-trap","name":"Bad","caption":"drums","bpm":200,"duration":48}`, http.StatusConflict},
	} {
		t.Run(name, func(t *testing.T) {
			state := &ModuleState{Styles: stylesdomain.NewStore()}
			response := httptest.NewRecorder()
			state.createStyle(response, httptest.NewRequest(http.MethodPost, "/api/v1/styles", strings.NewReader(test.body)))
			if response.Code != test.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.status, response.Body.String())
			}
			if _, ok := state.Styles.Get("bad"); ok {
				t.Fatal("rejected style persisted")
			}
		})
	}
}

func TestStyleIngressRejectsOversizedPrefixesWithoutPersisting(t *testing.T) {
	for _, operation := range []string{"create", "import"} {
		t.Run(operation, func(t *testing.T) {
			state := &ModuleState{Styles: stylesdomain.NewStore()}
			// The first MiB contains a valid document and whitespace. Truncating
			// there must not turn the complete oversized request into success.
			payload := `{"id":"prefix","name":"Prefix","caption":"drums","params":{"bpm":200,"duration":48}}` + strings.Repeat(" ", 1<<20) + `{}`
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/styles", strings.NewReader(payload))
			if operation == "create" {
				state.createStyle(response, request)
			} else {
				state.importStyle(response, request)
			}
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if _, ok := state.Styles.Get("prefix"); ok {
				t.Fatal("oversized prefix persisted")
			}
		})
	}
}

func TestNewStateHasDefaultStyleAndPool(t *testing.T) {
	state := testState(t)
	compiled, err := state.Styles.Compile("launch-trap")
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Caption == "" || state.Pool == nil {
		t.Fatal("state is incomplete")
	}
}

func TestDurableCompositionJobSurvivesSubmitReturn(t *testing.T) {
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakeGenerationProfile(t, w, r)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if _, exists := payload["admitted_rungs"]; exists {
			t.Error("nil-broker caller invented an admission restriction")
		}
		if _, exists := payload["expected_rung"]; exists {
			t.Error("nil-broker caller invented a strict selected rung")
		}
		w.Header().Set("X-ACE-Step-Profile-Rung", "offload-dit")
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("X-ACE-Step-Model-ID", "ACE-Step/Ace-Step1.5:acestep-v15-turbo")
		w.Header().Set("X-ACE-Step-License-Lane", "permissive")
		if w.Header().Get("X-ACE-Step-Profile-Rung") == "" {
			w.Header().Set("X-ACE-Step-Profile-Rung", "full")
		}
		_, _ = w.Write(wavFixture())
	}))
	defer resource.Close()

	state := testState(t)
	state.HTTP, state.ResourceURL = resource.Client(), resource.URL
	request := httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"launch-trap","takes":1,"duration":1,"seed":4}`))
	response := httptest.NewRecorder()
	state.compose(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d body=%s", response.Code, response.Body.String())
	}
	var accepted struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &accepted); err != nil || accepted.JobID == "" {
		t.Fatalf("submit response=%s err=%v", response.Body.String(), err)
	}
	done, err := state.JobManager.Wait(context.Background(), accepted.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != "succeeded" {
		t.Fatalf("durable job=%+v", done)
	}
	if done.ResultRef != "" {
		t.Fatalf("peer composition wrote a primary result_ref=%q", done.ResultRef)
	}
	if available, _, _, _ := state.Pool.Status("launch-trap"); available != 1 {
		t.Fatalf("available=%d", available)
	}
}

func TestDurableBatchRetainsCompletedTakesWhenResourceFails(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakeGenerationProfile(t, w, r)
		calls++
		if calls == 2 {
			http.Error(w, "synthetic resource failure", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("X-ACE-Step-Model-ID", "ACE-Step/Ace-Step1.5:acestep-v15-turbo")
		w.Header().Set("X-ACE-Step-License-Lane", "permissive")
		if w.Header().Get("X-ACE-Step-Profile-Rung") == "" {
			w.Header().Set("X-ACE-Step-Profile-Rung", "full")
		}
		_, _ = w.Write(wavFixture())
	}))
	defer server.Close()

	state := testState(t)
	state.HTTP = server.Client()
	state.ResourceURL = server.URL
	payload := []byte(`{"style_id":"launch-trap","takes":3,"duration":1,"seed":2}`)
	created, err := state.JobManager.Submit(context.Background(), jobdomain.Spec{Operation: "compose", Lane: jobdomain.LaneGPU, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	done, err := state.JobManager.Wait(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != jobdomain.StateFailed || done.Error == "" {
		t.Fatalf("got %+v", done)
	}

	if available, _, _, _ := state.Pool.Status("launch-trap"); available != 1 {
		t.Fatalf("pool available = %d, want one retained take", available)
	}
	var profiles map[string]generationProfile
	if err := json.Unmarshal([]byte(done.Meta["generation_profiles"]), &profiles); err != nil {
		t.Fatal(err)
	}
	p, ok := profiles[created.ID+"-take-01"]
	if len(profiles) != 1 || !ok || done.Meta["generation_profile_scope"] != "completed-takes" || done.Meta["seed_receipt"] != "verified-uint32" || done.Meta["lyrics_sent_sha256"] != lyricsHash("[instrumental]") || p.Planner == "" {
		t.Fatalf("partial actual facts lost: %+v", done)
	}
}

func TestReserveTakeUsesRequestedID(t *testing.T) {
	state := testState(t)
	takes, err := core.PlanBatch(core.BatchRequest{Duration: 45, JobID: "job", StyleID: "launch-trap", Caption: "instrumental trap", Takes: 2, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	state.Pool.Add(takes...)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/takes/"+takes[1].ID+"/reserve", strings.NewReader(`{"holder":"operator"}`))
	request = mux.SetURLVars(request, map[string]string{"id": takes[1].ID})
	response := httptest.NewRecorder()
	state.reserveTake(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("reserve status=%d body=%s", response.Code, response.Body.String())
	}
	reserved, ok := state.Pool.Get(takes[1].ID)
	if !ok || reserved.PoolState != "reserved" || reserved.ReservedBy != "operator" {
		t.Fatalf("requested take=%+v", reserved)
	}
	other, ok := state.Pool.Get(takes[0].ID)
	if !ok || other.PoolState != "available" {
		t.Fatalf("other take=%+v", other)
	}
}

func testState(t *testing.T) *ModuleState {
	return testStateWithBroker(t, nil)
}

func testStateWithBroker(t *testing.T, broker capacity.Broker) *ModuleState {
	state, _ := testStateWithBrokerDB(t, broker)
	return state
}

func testStateWithBrokerDB(t *testing.T, broker capacity.Broker) (*ModuleState, *sql.DB) {
	t.Helper()
	db := testdb.NewSQLite(t)
	for _, schema := range []string{jobdomain.Schema(), stylesdomain.Schema(), core.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	blobs := musicstorage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(filepath.Join(root, "blobs")), filepath.Join(root, "blobs"))
	state, err := NewStateWithCapacityAndStorage(db, broker, blobs)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(state.Close)
	return state, db
}

func TestInvalidCompositionRejectedBeforeJobAdmission(t *testing.T) {
	state := testState(t)
	for _, body := range []string{`{"style_id":"launch-trap","takes":101,"duration":45}`, `{"style_id":"launch-trap","takes":1,"duration":241}`} {
		response := httptest.NewRecorder()
		state.compose(response, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
	}
	listed, err := state.JobManager.List(context.Background(), 10)
	if err != nil || len(listed) != 0 {
		t.Fatalf("invalid jobs admitted %+v %v", listed, err)
	}
}

func TestGenerateRequiresActualIdentityAndWAVBeforePoolPublication(t *testing.T) {
	for _, tc := range []struct {
		name, model, body string
		ok                bool
	}{{"missing identity", "", "RIFF0000WAVE", false}, {"invalid WAV", "actual-sft:acestep-v15-sft", "not audio", false}, {"actual SFT", "actual-sft:acestep-v15-sft", "RIFF0000WAVE data", true}} {
		t.Run(tc.name, func(t *testing.T) {
			state := testState(t)
			resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeGenerationProfile(t, w, r)
				w.Header().Set("X-ACE-Step-Model-ID", tc.model)
				w.Header().Set("X-ACE-Step-License-Lane", "permissive")
				w.Header().Set("X-ACE-Step-Profile-Rung", "offload-dit")
				if tc.ok {
					w.Write(wavFixture())
				} else {
					w.Write([]byte(tc.body))
				}
			}))
			defer resource.Close()
			state.HTTP, state.ResourceURL = resource.Client(), resource.URL
			planned, err := core.PlanBatch(core.BatchRequest{JobID: "test", StyleID: "launch-trap", Caption: "authored", Takes: 1, Duration: 1})
			if err != nil {
				t.Fatal(err)
			}
			compiled, _ := state.Styles.Compile("launch-trap")
			compiled.Params.Variant = "acestep-v15-sft"
			_, err = state.generateTake(context.Background(), &planned[0], composeRequest{Duration: 1}, compiled, "")
			if (err == nil) != tc.ok {
				t.Fatalf("got %v", err)
			}
			if tc.ok && planned[0].Provenance.ModelID != "actual-sft:acestep-v15-sft" {
				t.Fatal("model identity fabricated")
			}
			if !tc.ok && len(state.Blobs.Entries()) != 0 {
				t.Fatal("invalid output published")
			}
		})
	}
}

func wavFixture() []byte {
	raw := make([]byte, 48)
	copy(raw, "RIFF")
	binary.LittleEndian.PutUint32(raw[4:], 40)
	copy(raw[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(raw[16:], 16)
	binary.LittleEndian.PutUint16(raw[20:], 1)
	binary.LittleEndian.PutUint16(raw[22:], 2)
	binary.LittleEndian.PutUint32(raw[24:], 48000)
	binary.LittleEndian.PutUint32(raw[28:], 192000)
	binary.LittleEndian.PutUint16(raw[32:], 4)
	binary.LittleEndian.PutUint16(raw[34:], 16)
	copy(raw[36:], "data")
	binary.LittleEndian.PutUint32(raw[40:], 4)
	return raw
}

type admissionBroker struct {
	releases atomic.Int32
	released chan struct{}
	rung     string
}

func (b *admissionBroker) Claim(context.Context, string, uint64, uint64) (capacity.Verdict, error) {
	rung := b.rung
	if rung == "" {
		rung = "offload-dit"
	}
	return capacity.Verdict{Granted: true, ClaimID: "lease", AppliedRung: rung}, nil
}
func (b *admissionBroker) Release(context.Context, string) error {
	b.releases.Add(1)
	if b.released != nil {
		b.released <- struct{}{}
	}
	return nil
}
func (b *admissionBroker) Heartbeat(context.Context, string) error { return nil }

func waitForAdmissionRelease(t *testing.T, broker *admissionBroker) {
	t.Helper()
	select {
	case <-broker.released:
	case <-time.After(time.Second):
		t.Fatal("terminal job did not release its capacity claim")
	}
	if got := broker.releases.Load(); got != 1 {
		t.Fatalf("claim release count=%d", got)
	}
}
func TestRestartRequiredDegradationRefusesBeforeGeneration(t *testing.T) {
	db := testdb.NewSQLite(t)
	for _, schema := range []string{jobdomain.Schema(), stylesdomain.Schema(), core.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	blobs := musicstorage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	broker := &admissionBroker{released: make(chan struct{}, 2)}
	state, err := NewStateWithCapacityAndStorage(db, broker, blobs)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	var calls atomic.Int32
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakeGenerationProfile(t, w, r)
		calls.Add(1)
		if r.URL.Path != "/v1/capacity/degrade" {
			t.Errorf("generation attempted after unapplied rung")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"applied_rung":"offload-dit","restart_required":true}`))
	}))
	defer resource.Close()
	state.ResourceURL, state.HTTP = resource.URL, resource.Client()
	j, err := state.JobManager.Submit(context.Background(), jobdomain.Spec{Operation: "compose", Lane: jobdomain.LaneGPU, Payload: []byte(`{"style_id":"launch-trap","takes":1,"duration":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	done, err := state.JobManager.Wait(context.Background(), j.ID)
	if err != nil || done.State != jobdomain.StateFailed || !strings.Contains(done.Error, "managed restart") {
		t.Fatalf("standing %+v %v", done, err)
	}
	waitForAdmissionRelease(t, broker)
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestDurableAdmissionCompatibilityKeepsActualOutputProvenance(t *testing.T) {
	for _, tc := range []struct {
		admitted, actual string
		allowed          []string
		success          bool
	}{
		{"full", "offload-dit", []string{"full", "offload-dit"}, true},
		{"full", "full", []string{"full", "offload-dit"}, true},
		{"offload-dit", "offload-dit", []string{"offload-dit"}, true},
		{"offload-dit", "full", []string{"offload-dit"}, false},
		{"unknown", "full", nil, false},
	} {
		t.Run(tc.admitted+"/"+tc.actual, func(t *testing.T) {
			broker := &admissionBroker{rung: tc.admitted, released: make(chan struct{}, 2)}
			state, db := testStateWithBrokerDB(t, broker)
			var generationCalls, degradeCalls atomic.Int32
			resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeGenerationProfile(t, w, r)
				if r.URL.Path == "/v1/capacity/degrade" {
					degradeCalls.Add(1)
					// The forged-full-output case passes acknowledgement first;
					// the consumer must still reject its incompatible WAV header.
					w.Write([]byte(`{"applied_rung":"offload-dit","restart_required":false}`))
					return
				}
				generationCalls.Add(1)
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				want, _ := json.Marshal(tc.allowed)
				actual, _ := json.Marshal(payload["admitted_rungs"])
				if string(actual) != string(want) || payload["expected_rung"] != nil {
					t.Errorf("admission payload=%v, want allowed %s without strict selection", payload, want)
				}
				w.Header().Set("X-ACE-Step-Profile-Rung", tc.actual)
				w.Header().Set("X-ACE-Step-Model-ID", "actual-model:acestep-v15-turbo")
				w.Header().Set("X-ACE-Step-License-Lane", "permissive")
				w.Write(wavFixture())
			}))
			defer resource.Close()
			state.ResourceURL, state.HTTP = resource.URL, resource.Client()
			job, err := state.JobManager.Submit(context.Background(), jobdomain.Spec{Operation: "compose", Lane: jobdomain.LaneGPU, Payload: []byte(`{"style_id":"launch-trap","takes":1,"duration":1}`)})
			if err != nil {
				t.Fatal(err)
			}
			done, err := state.JobManager.Wait(context.Background(), job.ID)
			if err != nil || (done.State == jobdomain.StateSucceeded) != tc.success {
				t.Fatalf("terminal=%+v err=%v", done, err)
			}
			waitForAdmissionRelease(t, broker)
			if tc.admitted == "unknown" {
				if generationCalls.Load() != 0 || degradeCalls.Load() != 0 {
					t.Fatal("unknown admission reached resource")
				}
			} else if generationCalls.Load() != 1 {
				t.Fatalf("generation calls=%d", generationCalls.Load())
			}
			available, _, _, _ := state.Pool.Status("launch-trap")
			if tc.success {
				if available != 1 || len(state.Blobs.Entries()) != 1 || done.Meta["applied_rung"] != tc.actual || done.Meta["capacity_admitted_rung"] != tc.admitted {
					t.Fatalf("lost actual/admitted distinction: available=%d blobs=%v job=%+v", available, state.Blobs.Entries(), done)
				}
				rows := state.Pool.List("launch-trap")
				if len(rows) != 1 || rows[0].Provenance.AppliedRung != tc.actual {
					t.Fatalf("take provenance=%+v", rows)
				}
				persistedPool, err := core.NewPoolWithStore(db)
				if err != nil {
					t.Fatal(err)
				}
				persistedTakes := persistedPool.ListFiltered("launch-trap", job.ID)
				if len(persistedTakes) != 1 || persistedTakes[0].Provenance.AppliedRung != tc.actual {
					t.Fatalf("persisted take provenance=%+v", persistedTakes)
				}
				observer := jobdomain.New(db, jobdomain.Config{})
				persistedJob, err := observer.Get(context.Background(), job.ID)
				if err != nil || persistedJob.Meta["applied_rung"] != tc.actual || persistedJob.Meta["capacity_admitted_rung"] != tc.admitted {
					t.Fatalf("persisted job provenance=%+v err=%v", persistedJob, err)
				}
			} else if available != 0 || len(state.Blobs.Entries()) != 0 || done.Meta["applied_rung"] != "" {
				t.Fatalf("failed output retained/fabricated: available=%d blobs=%v job=%+v", available, state.Blobs.Entries(), done)
			}
		})
	}
}

func TestDurableMixedOutputIdentityDoesNotLabelWholeBatchAsFirstTake(t *testing.T) {
	for _, tc := range []struct {
		name, secondModel, secondRung, summaryModel, summaryRung string
	}{
		{"rung changes", "model-a:acestep-v15-turbo", "full", "model-a:acestep-v15-turbo", ""},
		{"model changes", "model-b:acestep-v15-turbo", "offload-dit", "", "offload-dit"},
		{"both change", "model-b:acestep-v15-turbo", "full", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			broker := &admissionBroker{rung: "full", released: make(chan struct{}, 2)}
			state, db := testStateWithBrokerDB(t, broker)
			var calls atomic.Int32
			resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeGenerationProfile(t, w, r)
				model, rung := "model-a:acestep-v15-turbo", "offload-dit"
				if calls.Add(1) == 2 {
					model, rung = tc.secondModel, tc.secondRung
				}
				w.Header().Set("X-ACE-Step-Profile-Rung", rung)
				w.Header().Set("X-ACE-Step-Model-ID", model)
				w.Header().Set("X-ACE-Step-License-Lane", "permissive")
				w.Write(wavFixture())
			}))
			defer resource.Close()
			state.ResourceURL, state.HTTP = resource.URL, resource.Client()
			job, err := state.JobManager.Submit(context.Background(), jobdomain.Spec{Operation: "compose", Lane: jobdomain.LaneGPU, Payload: []byte(`{"style_id":"launch-trap","takes":2,"duration":1}`)})
			if err != nil {
				t.Fatal(err)
			}
			done, err := state.JobManager.Wait(context.Background(), job.ID)
			if err != nil || done.State != jobdomain.StateSucceeded || calls.Load() != 2 {
				t.Fatalf("terminal=%+v calls=%d err=%v", done, calls.Load(), err)
			}
			waitForAdmissionRelease(t, broker)
			if done.Meta["output_identity"] != "mixed" || done.Meta["model_id"] != tc.summaryModel || done.Meta["applied_rung"] != tc.summaryRung || done.Meta["capacity_admitted_rung"] != "full" {
				t.Fatalf("batch summary falsely labels output: %+v", done.Meta)
			}
			pool, err := core.NewPoolWithStore(db)
			if err != nil {
				t.Fatal(err)
			}
			takes := pool.ListFiltered("launch-trap", job.ID)
			if len(takes) != 2 || takes[0].Provenance.AppliedRung != "offload-dit" || takes[0].Provenance.ModelID != "model-a:acestep-v15-turbo" || takes[1].Provenance.AppliedRung != tc.secondRung || takes[1].Provenance.ModelID != tc.secondModel {
				t.Fatalf("actual persisted take identities=%+v", takes)
			}
		})
	}
}

// Fake the resource's actual wire receipts using exact JSON integers. Real
// validation remains exercised by the malformed-header table above.
func fakeGenerationProfile(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if r.URL.Path != "/v1/compose" {
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	var p map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		t.Error(err)
		return
	}
	seed, ok := p["seed"].(json.Number)
	if !ok {
		t.Error("fake received missing exact seed")
		return
	}
	w.Header().Set("X-ACE-Step-Seed", seed.String())
	w.Header().Set("X-ACE-Step-Planner", "acestep-5Hz-lm-0.6B")
	steps := 8
	if variant, _ := p["expected_variant"].(string); variant == "acestep-v15-sft" {
		steps = 50
	}
	if n, ok := p["inference_steps"].(json.Number); ok {
		steps, _ = strconv.Atoi(n.String())
	}
	w.Header().Set("X-ACE-Step-Inference-Steps", strconv.Itoa(steps))
	guidance := "1"
	if variant, _ := p["expected_variant"].(string); variant == "acestep-v15-sft" {
		guidance = "7"
		if n, ok := p["guidance_scale"].(json.Number); ok {
			guidance = n.String()
		}
	}
	w.Header().Set("X-ACE-Step-Guidance-Scale", guidance)
	shift := "3"
	if n, ok := p["shift"].(json.Number); ok {
		shift = n.String()
	}
	w.Header().Set("X-ACE-Step-Shift", shift)
}

func TestDurableStoredSeedAndOptionalProfileControlsReachResource(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprint(explicit), func(t *testing.T) {
			state := testState(t)
			// A legacy style with unset knobs reaches the resource defaults; an authored
			// nonzero guidance is not falsely compared with turbo's effective CFG1.
			styleJSON := `{"id":"defaults","name":"Defaults","caption":"rave drums","bpm":150,"duration":1,"variant":"acestep-v15-turbo"}`
			if _, err := state.Styles.Import([]byte(styleJSON)); err != nil {
				t.Fatal(err)
			}
			var seeds []string
			resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeGenerationProfile(t, w, r)
				var body map[string]json.RawMessage
				_ = json.NewDecoder(r.Body).Decode(&body)
				seeds = append(seeds, string(body["seed"]))
				if _, ok := body["inference_steps"]; ok {
					t.Error("unset inference steps sent instead of omitted")
				}
				if _, ok := body["guidance_scale"]; ok {
					t.Error("unset guidance sent instead of omitted")
				}
				if explicit {
					if string(body["shift"]) != "1" {
						t.Errorf("shift override missing: %s", body["shift"])
					}
				} else if _, ok := body["shift"]; ok {
					t.Error("nil shift replaced profile default")
				}
				w.Header().Set("X-ACE-Step-Model-ID", "ACE-Step/Ace-Step1.5:acestep-v15-turbo")
				w.Header().Set("X-ACE-Step-License-Lane", "permissive")
				w.Header().Set("X-ACE-Step-Profile-Rung", "full")
				w.Header().Set("X-ACE-Step-LM-Code-SHA256", strings.Repeat("a", 64))
				w.Header().Set("X-ACE-Step-LM-Code-Count", "2")
				_, _ = w.Write(wavFixture())
			}))
			defer resource.Close()
			state.ResourceURL, state.HTTP = resource.URL, resource.Client()
			raw := `{"style_id":"defaults","takes":1,"duration":1,"seed":424242,"resolved_lyrics":"[instrumental]"}`
			if explicit {
				raw = `{"style_id":"defaults","takes":1,"duration":1,"seed":424242,"shift":1,"resolved_lyrics":"[instrumental]"}`
			}
			// Re-decoding the same durable snapshot cannot re-sample a base seed.
			for _, id := range []string{"original", "recovered"} {
				result, err := state.runDurableJob(t.Context(), jobdomain.Job{ID: id, Payload: []byte(raw)}, func(int, string) {})
				if err != nil {
					t.Fatal(err)
				}
				var profiles map[string]generationProfile
				if err := json.Unmarshal([]byte(result.Meta["generation_profiles"]), &profiles); err != nil {
					t.Fatal(err)
				}
				p, ok := profiles[id+"-take-01"]
				if !ok || p.Planner != "acestep-5Hz-lm-0.6B" || p.InferenceSteps != 8 || p.GuidanceScale != 1 || p.LMCodeCount != 2 || p.LMCodeSHA256 != strings.Repeat("a", 64) || p.LyricsSHA256 != lyricsHash("[instrumental]") {
					t.Fatalf("lost actual profile: %+v", profiles)
				}
				wantShift := 3.0
				if explicit {
					wantShift = 1
				}
				if p.Shift != wantShift {
					t.Fatalf("actual shift=%v", p.Shift)
				}
			}
			if len(seeds) != 2 || seeds[0] != seeds[1] {
				t.Fatalf("durable base changed effective seed: %v", seeds)
			}
		})
	}
}

func TestComposeGeneratedRequestPresenceAndStrictIngress(t *testing.T) {
	state, d := testStateWithBrokerDB(t, nil)
	state.JobManager.Close()
	state.JobManager = jobdomain.New(d, jobdomain.Config{Runner: func(context.Context, jobdomain.Job, func(int, string)) (jobdomain.Result, error) {
		return jobdomain.Result{}, nil
	}})
	if err := state.JobManager.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	seed := int64(0)
	shift := 1.0
	data, err := protojson.Marshal(&compositionpb.SubmitRequest{StyleId: "launch-trap", Takes: 1, Duration: 1, Seed: &seed, Shift: &shift})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	state.compose(rec, httptest.NewRequest("POST", "/api/v1/compose", bytes.NewReader(data)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("generated request refused: %d %s", rec.Code, rec.Body.String())
	}
	var accepted struct {
		ID string `json:"job_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &accepted)
	job, err := state.JobManager.Wait(t.Context(), accepted.ID)
	if err != nil {
		t.Fatal(err)
	}
	var payload composeJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Seed == nil || *payload.Seed != 0 || payload.Shift == nil || *payload.Shift != 1 {
		t.Fatalf("generated explicit zero lost: %+v", payload)
	}
	for _, bad := range []string{`{"style_id":"launch-trap","seed":0,"unexpected":1}`, `{"style_id":"launch-trap","lyrics":"override"}`, `{"style_id":"launch-trap","resolved_lyrics":"override"}`, `{"style_id":"launch-trap","shift":"NaN"}`, `{"style_id":"launch-trap","shift":3} {}`, `{"style_id":"launch-trap","seed":0,"seed":1}`, `null`} {
		rec := httptest.NewRecorder()
		state.compose(rec, httptest.NewRequest("POST", "/api/v1/compose", strings.NewReader(bad)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("bad request admitted: %s %d", bad, rec.Code)
		}
	}
	jobs, err := state.JobManager.List(t.Context(), 100)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("malformed controls queued jobs=%d %v", len(jobs), err)
	}
}

func TestCancelledBatchRetainsOnlyCompletedResourceProfile(t *testing.T) {
	state := testState(t)
	second := make(chan struct{})
	var calls atomic.Int32
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakeGenerationProfile(t, w, r)
		if calls.Add(1) == 2 {
			close(second)
			<-r.Context().Done()
			return
		}
		w.Header().Set("X-ACE-Step-Model-ID", "ACE-Step/Ace-Step1.5:acestep-v15-turbo")
		w.Header().Set("X-ACE-Step-Profile-Rung", "full")
		w.Header().Set("X-ACE-Step-License-Lane", "permissive")
		_, _ = w.Write(wavFixture())
	}))
	defer resource.Close()
	state.ResourceURL, state.HTTP = resource.URL, resource.Client()
	j, err := state.JobManager.Submit(t.Context(), jobdomain.Spec{Operation: "compose", Lane: jobdomain.LaneGPU, Payload: []byte(`{"style_id":"launch-trap","takes":3,"duration":1,"seed":0}`)})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-second:
	case <-time.After(time.Second * 3):
		t.Fatal("second request did not start")
	}
	if err := state.JobManager.Cancel(j.ID); err != nil {
		t.Fatal(err)
	}
	done, err := state.JobManager.Wait(t.Context(), j.ID)
	var profiles map[string]generationProfile
	decodeErr := json.Unmarshal([]byte(done.Meta["generation_profiles"]), &profiles)
	if err != nil || done.State != jobdomain.StateCanceled || done.ResultRef != "" || done.Error != "" || decodeErr != nil || len(profiles) != 1 || profiles[j.ID+"-take-01"].Planner == "" {
		t.Fatalf("cancel facts %+v err=%v decode=%v", done, err, decodeErr)
	}
	if len(state.Pool.ListFiltered("launch-trap", j.ID)) != 1 {
		t.Fatal("cancel lost completed take or retained unfinished output")
	}
}

func TestUnsafePlannedSeedRefusesBeforeResourceCall(t *testing.T) {
	state := testState(t)
	compiled, _ := state.Styles.Compile("launch-trap")
	var calls atomic.Int32
	resource := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer resource.Close()
	state.ResourceURL, state.HTTP = resource.URL, resource.Client()
	for _, seed := range []int64{-1, 4294967296} {
		take := core.Take{Provenance: core.Provenance{Seed: seed}}
		if _, err := state.generateTake(t.Context(), &take, composeRequest{Duration: 1}, compiled, ""); err == nil {
			t.Fatalf("unsafe seed accepted: %d", seed)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unsafe effective seed reached resource")
	}
}

func TestActualSFTGuidanceMustMatchExplicitIntentBeforeIngestion(t *testing.T) {
	for _, tc := range []struct {
		name, expectedVariant, actualVariant string
		authored, actual                     float64
		valid                                bool
	}{
		{"SFT mismatch", "acestep-v15-sft", "acestep-v15-sft", 7, 1, false},
		{"legacy actual SFT mismatch", "", "acestep-v15-sft", 7, 1, false},
		{"SFT exact", "acestep-v15-sft", "acestep-v15-sft", 7, 7, true},
		{"SFT unset default", "acestep-v15-sft", "acestep-v15-sft", 0, 7, true},
		{"turbo effective normalization", "acestep-v15-turbo", "acestep-v15-turbo", 7, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := testState(t)
			compiled, _ := state.Styles.Compile("launch-trap")
			compiled.Params.Variant = tc.expectedVariant
			compiled.Params.Guidance = tc.authored
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeGenerationProfile(t, w, r)
				var payload map[string]json.RawMessage
				_ = json.NewDecoder(r.Body).Decode(&payload)
				if tc.authored != 0 && string(payload["guidance_scale"]) != strconv.FormatFloat(tc.authored, 'g', -1, 64) {
					t.Errorf("authored guidance not sent: %v", payload)
				}
				w.Header().Set("X-ACE-Step-Guidance-Scale", strconv.FormatFloat(tc.actual, 'g', -1, 64))
				w.Header().Set("X-ACE-Step-Model-ID", "ACE-Step/Ace-Step1.5:"+tc.actualVariant)
				w.Header().Set("X-ACE-Step-License-Lane", "permissive")
				w.Header().Set("X-ACE-Step-Profile-Rung", "full")
				_, _ = w.Write(wavFixture())
			}))
			defer server.Close()
			state.ResourceURL, state.HTTP = server.URL, server.Client()
			planned, _ := core.PlanBatch(core.BatchRequest{JobID: "guidance", StyleID: "launch-trap", Caption: "drums", Takes: 1, Duration: 1})
			p, err := state.generateTake(t.Context(), &planned[0], composeRequest{Duration: 1}, compiled, "")
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v profile=%+v error=%v", tc.valid, p, err)
			}
			if tc.valid {
				if p.GuidanceScale != tc.actual || len(state.Blobs.Entries()) != 1 {
					t.Fatalf("actual guidance lost: %+v", p)
				}
			} else if len(state.Blobs.Entries()) != 0 || len(state.Pool.List("launch-trap")) != 0 {
				t.Fatal("mismatched SFT response published")
			}
		})
	}
}

func TestInvalidSamplerIntentCannotQueueOrReachResource(t *testing.T) {
	state := testState(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	state.ResourceURL, state.HTTP = server.URL, server.Client()
	for i, params := range []string{`"inference_steps":-1`, `"inference_steps":101`, `"guidance_scale":-1`, `"guidance_scale":21`, `"guidanceScale":"NaN"`, `"guidanceScale":"Infinity"`, `"guidanceScale":"-Infinity"`} {
		id := fmt.Sprintf("invalid-%d", i)
		input := `{"id":"` + id + `","name":"Draft","caption":"drums","bpm":150,"duration":1,` + params + `}`
		if _, err := state.Styles.Import([]byte(input)); err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		state.compose(rec, httptest.NewRequest("POST", "/api/v1/compose", strings.NewReader(`{"style_id":"`+id+`","takes":1}`)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("invalid sampler admitted: %s status=%d", params, rec.Code)
		}
	}
	jobs, err := state.JobManager.List(t.Context(), 100)
	if err != nil || len(jobs) != 0 || calls.Load() != 0 || len(state.Blobs.Entries()) != 0 {
		t.Fatalf("invalid input performed work jobs=%d calls=%d err=%v", len(jobs), calls.Load(), err)
	}
}

func TestResourcePayloadEncodingErrorDoesNotSendEmptyHTTPBody(t *testing.T) {
	state := testState(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	state.ResourceURL, state.HTTP = server.URL, server.Client()
	compiled, _ := state.Styles.Compile("launch-trap")
	compiled.Params.Guidance = math.NaN()
	take := core.Take{Provenance: core.Provenance{Seed: 42}}
	if _, err := state.generateTake(t.Context(), &take, composeRequest{Duration: 1}, compiled, ""); err == nil || !strings.Contains(err.Error(), "encode resource compose request") {
		t.Fatalf("encoding error not reported: %v", err)
	}
	if calls.Load() != 0 || len(state.Blobs.Entries()) != 0 {
		t.Fatal("unencodable payload reached resource or storage")
	}
}
