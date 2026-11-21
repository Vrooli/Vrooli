package preparedphoto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"device-studio/internal/library"
	"device-studio/internal/scenes"
	"github.com/gorilla/mux"
)

type sourceStub struct {
	id   string
	data []byte
}

func (s *sourceStub) PreparedPhotograph(_ context.Context, id string) ([]byte, error) {
	s.id = id
	return s.data, nil
}

type publisherStub struct{}

func (publisherStub) Publish(context.Context, library.PublishInput) (library.BundleRevision, error) {
	return library.BundleRevision{}, nil
}

type sceneStub struct{}

func (sceneStub) SaveCandidate(context.Context, scenes.SceneCandidate, uint32) (scenes.Revision, error) {
	return scenes.Revision{}, nil
}

func TestConvertFetchesByOwnerIDAndRejectsInvalidRetainedBundle(t *testing.T) {
	source := &sourceStub{data: []byte("not a zip")}
	router := mux.NewRouter()
	Module(source, publisherStub{}, sceneStub{}).Mount(router)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/photographic-scenes/from-preparation", strings.NewReader(`{"preparation_id":"retained-child"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusUnprocessableEntity || source.id != "retained-child" {
		t.Fatalf("status=%d source-id=%q body=%s", response.Code, source.id, response.Body.String())
	}
}

func TestConvertRejectsUnknownRequestFieldsBeforeOwnerRead(t *testing.T) {
	source := &sourceStub{data: []byte("not a zip")}
	router := mux.NewRouter()
	Module(source, publisherStub{}, sceneStub{}).Mount(router)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/photographic-scenes/from-preparation", strings.NewReader(`{"preparation_id":"child","bundle_path":"/tmp/fake.zip"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest || source.id != "" {
		t.Fatalf("status=%d source-id=%q", response.Code, source.id)
	}
}
