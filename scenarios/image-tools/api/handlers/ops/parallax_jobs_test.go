package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"image-tools/internal/jobrunner"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"
)

type parallaxQueueFixture struct {
	fakeRecorder
	specs []internaljobs.Spec
	job   internaljobs.Job
}

func (q *parallaxQueueFixture) GetByClientOperation(_ context.Context, id string) (internaljobs.Job, error) {
	if q.job.ClientOperationID != id {
		return internaljobs.Job{}, internaljobs.ErrNotFound
	}
	return q.job, nil
}
func (q *parallaxQueueFixture) Submit(_ context.Context, s internaljobs.Spec) (internaljobs.Job, error) {
	q.specs = append(q.specs, s)
	q.job = internaljobs.Job{ID: "parallax-owned", Operation: s.Operation, Payload: s.Payload, ClientOperationID: s.ClientOperationID, RequestFingerprint: s.RequestFingerprint, State: internaljobs.StateQueued}
	return q.job, nil
}

func TestParallaxQueuesRetainedInputsAndReusesJobAfterRequestEnds(t *testing.T) {
	_, store, _ := newTestServer(t)
	q := &parallaxQueueFixture{}
	h := &Deps{Store: store, Jobs: q, Guard: storage.DefaultGuard()}
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/ops/{operation}", h.runHandler)
	source := pngBytes(t, 32, 24)
	params := `{"parallax":{"scene":{"width":32,"height":24,"background":"#071422","plates":[{"name":"plane","opacity":1,"distance":1}]},"startCamera":{"requireOpaque":true},"endCamera":{"x":0.01,"requireOpaque":true},"frames":3,"framesPerSecond":30}}`
	request := func(key string, pixels []byte) *http.Request {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		for _, name := range []string{"file", "plate0"} {
			part, err := mw.CreateFormFile(name, name+".png")
			if err != nil {
				t.Fatal(err)
			}
			part.Write(pixels)
		}
		mw.WriteField("params", params)
		mw.Close()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/ops/parallax", &b)
		r.Header.Set("Content-Type", mw.FormDataContentType())
		r.Header.Set("Idempotency-Key", key)
		return r
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := request("same-operation", source).WithContext(ctx)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	cancel()
	if w.Code != http.StatusAccepted || len(q.specs) != 1 {
		t.Fatalf("not queued: %d %s", w.Code, w.Body.String())
	}
	if q.specs[0].Lane != internaljobs.LaneCPU || q.specs[0].ExecutionTimeoutSeconds != 180 {
		t.Fatal("lost owner lane/deadline")
	}
	var saved parallaxPayload
	if err := json.Unmarshal(q.specs[0].Payload, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Params.Plates[0].Image) != 0 || len(saved.Plates) != 1 {
		t.Fatal("payload embeds pixels or loses source closure")
	}
	dispatcher := jobrunner.New()
	RegisterParallaxRunner(dispatcher, store)
	result, err := dispatcher.Run(context.Background(), q.job, func(int, string) {})
	if err != nil {
		t.Fatal(err)
	}
	reader, mime, err := store.Get(context.Background(), result.Ref)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil || mime != "application/zip" || len(data) == 0 || len(result.Meta["sha256"]) != 64 {
		t.Fatal("retained job did not publish a verified frame sequence")
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, request("same-operation", source))
	if w.Code != 202 || len(q.specs) != 1 {
		t.Fatal("retry created another job")
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, request("same-operation", pngBytes(t, 34, 24)))
	if w.Code != 409 || len(q.specs) != 1 {
		t.Fatal("changed inputs adopted old operation")
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, request("", source))
	if w.Code != 400 {
		t.Fatal(fmt.Sprintf("missing stable operation id accepted: %d", w.Code))
	}
}
