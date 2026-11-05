package photographic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"device-studio/internal/photographic"
	"device-studio/internal/projects"
	"github.com/gorilla/mux"
)

type projectFake struct{}

func (projectFake) Create(context.Context, projects.CreateInput) (projects.Project, error) {
	return projects.Project{}, nil
}
func (projectFake) Get(_ context.Context, id string) (projects.Project, error) {
	if id != "p1" {
		return projects.Project{}, projects.ErrProjectNotFound{ID: id}
	}
	return projects.Project{ID: id}, nil
}
func (projectFake) List(context.Context, int32, string) ([]projects.Project, string, error) {
	return nil, "", nil
}
func (projectFake) Rename(context.Context, projects.RenameInput) (projects.Project, error) {
	return projects.Project{}, nil
}

type ownerFake struct {
	called bool
	got    photographic.Operation
}

func (o *ownerFake) Submit(_ context.Context, id, prompt string, w, h int) (photographic.Operation, int, error) {
	o.called = true
	o.got = photographic.Operation{JobID: "job-1", State: "queued"}
	return o.got, 202, nil
}
func (*ownerFake) Lookup(context.Context, string) (photographic.Operation, error) {
	return photographic.Operation{}, nil
}
func (*ownerFake) Wait(context.Context, string) (photographic.Operation, error) {
	return photographic.Operation{}, nil
}
func (*ownerFake) Cancel(context.Context, string) (photographic.Operation, error) {
	return photographic.Operation{}, nil
}
func (*ownerFake) PNG(context.Context, string, int, int) ([]byte, error) { return nil, nil }

func TestSubmitValidatesProjectAndBoundsBeforeOwnerCall(t *testing.T) {
	o := &ownerFake{}
	m := Module(projectFake{}, o)
	r := mux.NewRouter()
	m.Mount(r)
	good := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/photographic-scenes", strings.NewReader(`{"operation_id":"f7f4f09f-a93a-4f34-93ef-4fa814891f5b","prompt":"a quiet room","width":768,"height":512}`))
	good.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, good)
	if w.Code != 202 || !o.called {
		t.Fatalf("valid submit status=%d called=%v body=%s", w.Code, o.called, w.Body.String())
	}
	var got photographic.Operation
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.JobID != "job-1" {
		t.Fatalf("response=%+v err=%v", got, err)
	}
	o.called = false
	bad := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/photographic-scenes", strings.NewReader(`{"operation_id":"f7f4f09f-a93a-4f34-93ef-4fa814891f5b","prompt":"x","width":512,"height":512}`))
	bad.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, bad)
	if w.Code != 400 || o.called {
		t.Fatalf("invalid output reached owner: status=%d called=%v", w.Code, o.called)
	}
	missing := httptest.NewRequest(http.MethodPost, "/api/v1/projects/missing/photographic-scenes", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, missing)
	if w.Code != 404 || o.called {
		t.Fatalf("missing project reached owner: status=%d called=%v", w.Code, o.called)
	}
}

func TestOwnerLookupFailureIsNotReportedAsMissingOperation(t *testing.T) {
	if got := ownerReadStatus(photographic.OwnerHTTPError{Status: http.StatusInternalServerError}); got != http.StatusBadGateway {
		t.Fatalf("owner outage mapped to %d", got)
	}
	if got := ownerReadStatus(photographic.OwnerHTTPError{Status: http.StatusNotFound}); got != http.StatusNotFound {
		t.Fatalf("missing operation mapped to %d", got)
	}
}
