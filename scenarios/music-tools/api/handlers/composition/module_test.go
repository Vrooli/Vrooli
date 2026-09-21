package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	testdb "github.com/vrooli/api-core/databasetest"
	core "music-tools/internal/composition"
	jobdomain "music-tools/internal/jobs"
	stylesdomain "music-tools/internal/styles"
)

func TestNewStateHasDefaultStyleAndPool(t *testing.T) {
	state := NewState()
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
		w.Header().Set("X-ACE-Step-Profile-Rung", "offload-dit")
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("RIFF durable"))
	}))
	defer resource.Close()

	db := testdb.NewSQLite(t)
	if _, err := db.Exec(jobdomain.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(stylesdomain.Schema()); err != nil {
		t.Fatal(err)
	}
	state, err := NewStateWithJobs(db)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
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

func TestRunBatchRetainsCompletedTakesWhenResourceFails(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 2 {
			http.Error(w, "synthetic resource failure", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("RIFF synthetic"))
	}))
	defer server.Close()

	state := NewState()
	state.HTTP = server.Client()
	state.ResourceURL = server.URL
	item := &job{ID: fmt.Sprintf("job-%d", time.Now().UnixNano()), Done: make(chan struct{})}
	state.runBatch(item, composeRequest{StyleID: "launch-trap", Takes: 3, Duration: 1}, "instrumental trap")

	if item.State != "failed" || item.Error == "" {
		t.Fatalf("job state = %q error=%q, want failed job with error", item.State, item.Error)
	}
	if len(item.Takes) != 1 {
		t.Fatalf("retained takes = %d, want one completed take", len(item.Takes))
	}
	if available, _, _, _ := state.Pool.Status("launch-trap"); available != 1 {
		t.Fatalf("pool available = %d, want one retained take", available)
	}
}

func TestReserveTakeUsesRequestedID(t *testing.T) {
	state := NewState()
	takes, err := core.PlanBatch(core.BatchRequest{JobID: "job", StyleID: "launch-trap", Caption: "instrumental trap", Takes: 2, Seed: 4})
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
