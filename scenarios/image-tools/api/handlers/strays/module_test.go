package strays

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"image-tools/internal/modelenv"
	"image-tools/internal/modelstray"

	"github.com/gorilla/mux"
)

type fakeService struct {
	adopted, reclaimed []string
	fail               error
}

func (f *fakeService) Scan(context.Context) (modelstray.Report, error) {
	return modelstray.Report{Findings: []modelstray.Finding{{ID: "abc", Action: modelstray.ActionAdopt, Bytes: 7}}}, f.fail
}
func (f *fakeService) Adopt(_ context.Context, id string) (modelstray.Receipt, error) {
	f.adopted = append(f.adopted, id)
	return modelstray.Receipt{Action: "adopt", FindingID: id, Bytes: 7}, f.fail
}
func (f *fakeService) Reclaim(_ context.Context, id string) (modelstray.Receipt, error) {
	f.reclaimed = append(f.reclaimed, id)
	return modelstray.Receipt{Action: "reclaim", FindingID: id}, f.fail
}

func serve(svc Service, method, path, body string) *httptest.ResponseRecorder {
	r := mux.NewRouter()
	Module(svc, log.New(io.Discard, "", 0)).Mount(r)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestScanAdoptAndReclaimRoutes(t *testing.T) {
	svc := &fakeService{}
	rec := serve(svc, http.MethodGet, "/api/v1/models/strays", "")
	var report modelstray.Report
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &report) != nil || len(report.Findings) != 1 {
		t.Fatalf("scan: %d %s", rec.Code, rec.Body)
	}
	if rec = serve(svc, http.MethodPost, "/api/v1/models/strays/adopt", `{"id":"abc"}`); rec.Code != 200 || len(svc.adopted) != 1 {
		t.Fatalf("adopt: %d %s", rec.Code, rec.Body)
	}
	if rec = serve(svc, http.MethodPost, "/api/v1/models/strays/reclaim", `{"id":"abc"}`); rec.Code != 200 || len(svc.reclaimed) != 1 {
		t.Fatalf("reclaim: %d %s", rec.Code, rec.Body)
	}
}

func TestActionsRejectBadBodiesAndSurfaceRefusals(t *testing.T) {
	svc := &fakeService{}
	for _, body := range []string{``, `{}`, `{"id":" "}`, `{"id":"a","x":1}`, `{"id":"a"}{"id":"b"}`} {
		if rec := serve(svc, http.MethodPost, "/api/v1/models/strays/reclaim", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%q: %d", body, rec.Code)
		}
	}
	if len(svc.reclaimed) != 0 {
		t.Fatal("a malformed request must not reach the scanner")
	}
	svc.fail = errors.New("cannot be reclaimed (action report_only): another application owns it")
	rec := serve(svc, http.MethodPost, "/api/v1/models/strays/reclaim", `{"id":"abc"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "another application owns it") {
		t.Fatalf("a refusal must say why: %d %s", rec.Code, rec.Body)
	}
}

func TestContainmentRouteReportsPinsFloorAndReasons(t *testing.T) {
	t.Cleanup(modelenv.SwapRoot(t.TempDir()))
	rec := serve(&fakeService{}, http.MethodGet, "/api/v1/models/containment", "")
	var report ContainmentReport
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &report) != nil {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if report.Env.Pins["HF_HUB_CACHE"] == "" || len(report.Env.Inference) == 0 || report.Disk.FloorBytes == 0 || len(report.Disk.FloorReasons) == 0 {
		t.Fatalf("%+v", report)
	}
}
