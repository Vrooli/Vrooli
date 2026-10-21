package creation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"device-studio/internal/jobs"
	"device-studio/internal/renderjobs"
	"github.com/gorilla/mux"
)

type fakeCoordinator struct {
	request       renderjobs.GeometryCreationRequest
	calls         int
	cancelVersion int64
	submitErr     error
	cancelErr     error
}

func (f *fakeCoordinator) SubmitGeometryCreation(_ context.Context, r renderjobs.GeometryCreationRequest) (jobs.Job, bool, error) {
	f.calls++
	f.request = r
	if f.submitErr != nil {
		return jobs.Job{}, false, f.submitErr
	}
	return sampleJob(), true, nil
}
func (f *fakeCoordinator) FindGeometryCreationByKey(_ context.Context, projectID, key string) (jobs.Job, error) {
	if projectID != "project-1" || key != "request-1" {
		return jobs.Job{}, jobs.ErrNotFound
	}
	return sampleJob(), nil
}
func (f *fakeCoordinator) Cancel(_ context.Context, id string, v int64) (jobs.Job, error) {
	f.cancelVersion = v
	if f.cancelErr != nil {
		return jobs.Job{}, f.cancelErr
	}
	return sampleJob(), nil
}

type routeActivatorStub struct {
	calls int
	err   error
	ctx   context.Context
}

func (s *routeActivatorStub) ActivateTestRoute(ctx context.Context) error {
	s.calls++
	s.ctx = ctx
	return s.err
}

type fakeJobReader struct{ job jobs.Job }

func (f fakeJobReader) Get(context.Context, string) (jobs.Job, error) { return f.job, nil }
func sampleJob() jobs.Job {
	return jobs.Job{ID: "job-1", Input: jobs.Input{ProjectID: "project-1", SubjectID: "candidate-1", SubjectKind: jobs.SubjectGeometryCreate}, State: jobs.Queued, Version: 3, CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(1, 0).UTC()}
}

func TestCreateRequiresExplicitLocalAIDisclosureAndPinsProjectRoute(t *testing.T) {
	c := &fakeCoordinator{}
	h := createHandler(c, nil)
	request := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/geometry-creations", strings.NewReader(body))
		r = mux.SetURLVars(r, map[string]string{"project_id": "project-1"})
		w := httptest.NewRecorder()
		h(w, r)
		return w
	}
	body := `{"candidate_id":"candidate-1","idempotency_key":"request-1","brief":"field device","envelope_mm":[80,150,12],"reference_media_ids":["media-1"]}`
	if w := request(body); w.Code != http.StatusUnprocessableEntity || c.calls != 0 {
		t.Fatalf("undisclosed AI request was dispatched: status=%d calls=%d", w.Code, c.calls)
	}
	body = strings.TrimSuffix(body, "}") + `,"confirm_local_ai":true}`
	if w := request(body); w.Code != http.StatusAccepted || c.calls != 1 {
		t.Fatalf("confirmed local request was not accepted: status=%d calls=%d", w.Code, c.calls)
	}
	if c.request.ProjectID != "project-1" || c.request.CandidateID != "candidate-1" || len(c.request.ReferenceMediaIDs) != 1 {
		t.Fatalf("project-scoped request changed: %+v", c.request)
	}
}

func TestAcceptedCreateActivatesExactRequestContextAndRejectedCreateDoesNot(t *testing.T) {
	type routeKey struct{}
	c := &fakeCoordinator{}
	activator := &routeActivatorStub{}
	h := createHandler(c, activator)
	body := `{"candidate_id":"candidate-1","idempotency_key":"request-1","brief":"field device","envelope_mm":[80,150,12],"reference_media_ids":["media-1"],"confirm_local_ai":true}`
	request := func(ctx context.Context) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/geometry-creations", strings.NewReader(body)).WithContext(ctx)
		r = mux.SetURLVars(r, map[string]string{"project_id": "project-1"})
		w := httptest.NewRecorder()
		h(w, r)
		return w
	}
	ctx := context.WithValue(context.Background(), routeKey{}, "pinned-lease-17")
	if w := request(ctx); w.Code != http.StatusAccepted || activator.calls != 1 {
		t.Fatalf("accepted create status=%d activation calls=%d body=%s", w.Code, activator.calls, w.Body.String())
	}
	if activator.ctx.Value(routeKey{}) != "pinned-lease-17" {
		t.Fatal("creation handler did not pass its exact routed request context to the pump")
	}
	c.submitErr = errors.New("invalid request")
	if w := request(ctx); w.Code != http.StatusUnprocessableEntity || activator.calls != 1 {
		t.Fatalf("refused create status=%d must not activate pump; calls=%d", w.Code, activator.calls)
	}
}

func TestAcceptedCreateActivationFailurePreservesSameKeyRecovery(t *testing.T) {
	c := &fakeCoordinator{}
	activator := &routeActivatorStub{err: errors.New("test route missing")}
	body := `{"candidate_id":"candidate-1","idempotency_key":"request-1","brief":"field device","envelope_mm":[80,150,12],"reference_media_ids":["media-1"],"confirm_local_ai":true}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/geometry-creations", strings.NewReader(body))
	r = mux.SetURLVars(r, map[string]string{"project_id": "project-1"})
	w := httptest.NewRecorder()
	createHandler(c, activator)(w, r)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "same request identity") || c.calls != 1 || activator.calls != 1 {
		t.Fatalf("activation failure lost accepted operation identity: status=%d submit=%d activation=%d body=%s", w.Code, c.calls, activator.calls, w.Body.String())
	}
}

func TestCreationStatusAndCancelExposeOnlyBoundedProjection(t *testing.T) {
	job := sampleJob()
	reader := fakeJobReader{job: job}
	cancelOwner := &fakeCoordinator{}
	status := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/geometry-creations/job-1", nil)
	req = mux.SetURLVars(req, map[string]string{"job_id": "job-1"})
	statusHandler(reader)(status, req)
	if status.Code != http.StatusOK {
		t.Fatalf("status code %d", status.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(status.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["review_status"] != "unreviewed" || got["candidate_id"] != "candidate-1" || got["bindings"] != nil {
		t.Fatalf("unsafe or inaccurate status projection: %s", status.Body.String())
	}
	body := httptest.NewRecorder()
	cancel := httptest.NewRequest(http.MethodPost, "/api/v1/geometry-creations/job-1/cancel", strings.NewReader(`{"version":3}`))
	cancel = mux.SetURLVars(cancel, map[string]string{"job_id": "job-1"})
	cancelHandler(cancelOwner, reader, nil)(body, cancel)
	if body.Code != http.StatusAccepted || cancelOwner.cancelVersion != 3 {
		t.Fatalf("cancel status=%d version=%d body=%s", body.Code, cancelOwner.cancelVersion, body.Body.String())
	}
}

func TestCancellationActivatesPumpOnlyAfterOwnerAcceptsIt(t *testing.T) {
	reader := fakeJobReader{job: sampleJob()}
	c := &fakeCoordinator{}
	activator := &routeActivatorStub{}
	call := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/geometry-creations/job-1/cancel", strings.NewReader(`{"version":3}`))
		r = mux.SetURLVars(r, map[string]string{"job_id": "job-1"})
		cancelHandler(c, reader, activator)(w, r)
		return w
	}
	if w := call(); w.Code != http.StatusAccepted || activator.calls != 1 {
		t.Fatalf("accepted cancel status=%d activation calls=%d body=%s", w.Code, activator.calls, w.Body.String())
	}
	c.cancelErr = errors.New("stale version")
	if w := call(); w.Code != http.StatusConflict || activator.calls != 1 {
		t.Fatalf("refused cancel status=%d must not activate pump; calls=%d", w.Code, activator.calls)
	}
}

func TestOperationReadbackIsDurableOwnerBackedAndProjectScoped(t *testing.T) {
	c := &fakeCoordinator{}
	h := operationHandler(c)
	request := func(projectID, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/geometry-creations/operations/"+key, nil)
		req = mux.SetURLVars(req, map[string]string{"project_id": projectID, "operation_key": key})
		w := httptest.NewRecorder()
		h(w, req)
		return w
	}
	if got := request("project-1", "request-1"); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"job_id":"job-1"`) {
		t.Fatalf("owner readback status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request("other-project", "request-1"); got.Code != http.StatusNotFound {
		t.Fatalf("cross-project operation lookup status=%d body=%s", got.Code, got.Body.String())
	}
}

func TestCreationBodyDecoderRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"candidate_id":"x","unexpected":true}`))
	w := httptest.NewRecorder()
	var body createBody
	if err := decodeBody(w, req, &body, 100); err == nil {
		t.Fatal("unknown creation fields must be rejected")
	}
}
