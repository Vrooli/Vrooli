package sketch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"react-component-library/internal/designcapture"
)

type screenshotDispatchStub struct{}

func (screenshotDispatchStub) Start(context.Context, designcapture.Operation) (string, error) {
	return "execution-1", nil
}

func (screenshotDispatchStub) ResolveScreenshot(context.Context, designcapture.Operation, designcapture.Artifact) (designcapture.Screenshot, error) {
	return designcapture.Screenshot{Reference: "capture-1"}, nil
}

func TestMountedReferenceAssetUploadAndServe(t *testing.T) {
	root := t.TempDir()
	module, ports := ModuleWithCaptureAndAcceptance(root, nil, nil, nil, nil, nil, nil, CaptureConfig{Dispatcher: screenshotDispatchStub{}}, nil)
	require.Equal(t, "sketch", module.Name)
	require.NotNil(t, ports.Evaluator)
	require.NotNil(t, ports.Screenshots)

	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("scenario", "demo"))
	require.NoError(t, writer.WriteField("page", "home"))
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="reference.png"`)
	partHeader.Set("Content-Type", "image/png")
	file, err := writer.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = io.WriteString(file, "reference-image-bytes")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/reference-assets", &body)
	require.NoError(t, err)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	var uploaded struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&uploaded))
	require.NotEmpty(t, uploaded.ID)
	require.Equal(t, "/api/v1/reference-assets/demo/home/"+uploaded.Name, uploaded.URL)

	served, err := server.Client().Get(server.URL + uploaded.URL)
	require.NoError(t, err)
	defer served.Body.Close()
	require.Equal(t, http.StatusOK, served.StatusCode)
	servedBytes, err := io.ReadAll(served.Body)
	require.NoError(t, err)
	require.Equal(t, "reference-image-bytes", string(servedBytes))
	require.FileExists(t, filepath.Join(root, "scenarios", "demo", "data", "experience", "reference-assets", "home", uploaded.Name))
}

func TestMountedReferenceAssetRejectsMissingUploadAndReturnsNotFound(t *testing.T) {
	module, _ := ModuleWithCaptureAndAcceptance(t.TempDir(), nil, nil, nil, nil, nil, nil, CaptureConfig{}, nil)
	router := mux.NewRouter()
	module.Mount(router)

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/reference-assets", bytes.NewBufferString("not multipart")))
	require.Equal(t, http.StatusBadRequest, missing.Code)

	missingAsset := httptest.NewRecorder()
	router.ServeHTTP(missingAsset, httptest.NewRequest(http.MethodGet, "/reference-assets/demo/home/missing.png", nil))
	require.Equal(t, http.StatusNotFound, missingAsset.Code)
}
