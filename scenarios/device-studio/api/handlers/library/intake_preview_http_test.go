package library

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "device-studio/internal/library"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func intakePreviewHTTPFixture() []byte {
	bin := make([]byte, 48)
	positions := [][3]float32{{-1, -1, 0}, {1, -1, 0}, {0, 1, 1}}
	offset := 0
	for _, p := range positions {
		for _, value := range p {
			binary.LittleEndian.PutUint32(bin[offset:offset+4], math.Float32bits(value))
			offset += 4
		}
	}
	copy(bin[offset:], []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255})
	doc := map[string]any{
		"asset": map[string]any{"version": "2.0"}, "nodes": []any{map[string]any{"mesh": 0}},
		"meshes":      []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]any{"POSITION": 0, "COLOR_0": 1}}}}},
		"buffers":     []any{map[string]any{"byteLength": len(bin)}},
		"bufferViews": []any{map[string]any{"buffer": 0, "byteLength": 36}, map[string]any{"buffer": 0, "byteOffset": 36, "byteLength": 12}},
		"accessors":   []any{map[string]any{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3"}, map[string]any{"bufferView": 1, "componentType": 5121, "normalized": true, "count": 3, "type": "VEC4"}},
	}
	jsonChunk, _ := json.Marshal(doc)
	for len(jsonChunk)%4 != 0 {
		jsonChunk = append(jsonChunk, ' ')
	}
	out := make([]byte, 12+8+len(jsonChunk)+8+len(bin))
	copy(out[:4], "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(jsonChunk)))
	binary.LittleEndian.PutUint32(out[16:20], 0x4e4f534a)
	copy(out[20:], jsonChunk)
	position := 20 + len(jsonChunk)
	binary.LittleEndian.PutUint32(out[position:position+4], uint32(len(bin)))
	binary.LittleEndian.PutUint32(out[position+4:position+8], 0x004e4942)
	copy(out[position+8:], bin)
	return out
}

func TestIntakePreviewHTTPIsReadOnlyAndBoundToExactRevision(t *testing.T) {
	service := handlerIntakeService(t)
	data := intakePreviewHTTPFixture()
	staged, err := service.BeginImport(context.Background(), domain.IntakeClaims{CandidateID: "preview-http", DisplayName: "Preview", SourceLabel: "local", LicenseClaim: "unverified", Attribution: "operator"}, "preview.glb", int64(len(data)), bytes.NewReader(data))
	require.NoError(t, err)
	candidate, err := staged.Commit(context.Background())
	require.NoError(t, err)
	router := mux.NewRouter()
	router.Handle("/api/v1/library/intake-candidates/{candidate_id}/revisions/{revision}/preview", newIntakePreviewHandler(service, nil, nil, nil)).Methods(http.MethodGet)
	path := "/api/v1/library/intake-candidates/preview-http/revisions/1/preview?revision_sha256=" + candidate.RevisionSHA256
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "image/png", response.Header().Get("Content-Type"))
	require.Equal(t, candidate.ID, response.Header().Get("X-Intake-Candidate-ID"))
	require.Equal(t, "1", response.Header().Get("X-Intake-Revision"))
	require.Equal(t, candidate.RevisionSHA256, response.Header().Get("X-Intake-Revision-SHA256"))
	require.Equal(t, candidate.OriginalSHA256, response.Header().Get("X-Intake-GLB-SHA256"))
	require.Equal(t, "true", response.Header().Get("X-Intake-Preview-Vertex-Colors"))
	require.Equal(t, "unreviewed", response.Header().Get("X-Intake-Preview-Review"))
	require.True(t, bytes.HasPrefix(response.Body.Bytes(), []byte("\x89PNG\r\n\x1a\n")))
	require.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))

	wrongDigest := httptest.NewRecorder()
	router.ServeHTTP(wrongDigest, httptest.NewRequest(http.MethodGet, "/api/v1/library/intake-candidates/preview-http/revisions/1/preview?revision_sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil))
	require.Equal(t, http.StatusBadRequest, wrongDigest.Code)
}
