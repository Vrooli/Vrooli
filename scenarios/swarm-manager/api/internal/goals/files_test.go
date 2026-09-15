package goals

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestGoalFilesSupportEditingWhileProtectingCanonicalMetadata(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "delivery"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(svc.GoalDir("delivery"), "notes.md"), []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(svc)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/goals/delivery/files", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "notes.md") {
		t.Fatalf("list = %d: %s", list.Code, list.Body.String())
	}

	content := httptest.NewRecorder()
	router.ServeHTTP(content, httptest.NewRequest(http.MethodGet, "/api/v1/goals/delivery/files/notes.md", nil))
	if content.Code != http.StatusOK || content.Body.String() != "before" {
		t.Fatalf("content = %d: %q", content.Code, content.Body.String())
	}

	var uploadBody bytes.Buffer
	writer := multipart.NewWriter(&uploadBody)
	part, err := writer.CreateFormFile("file", "notes.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("after")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	upload := httptest.NewRequest(http.MethodPost, "/api/v1/goals/delivery/files", &uploadBody)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResponse := httptest.NewRecorder()
	router.ServeHTTP(uploadResponse, upload)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("upload = %d: %s", uploadResponse.Code, uploadResponse.Body.String())
	}

	rename := httptest.NewRequest(http.MethodPatch, "/api/v1/goals/delivery/files", strings.NewReader(`{"operation":"rename","sourcePath":"notes.md","destinationPath":"renamed.md"}`))
	rename.Header.Set("Content-Type", "application/json")
	renameResponse := httptest.NewRecorder()
	router.ServeHTTP(renameResponse, rename)
	if renameResponse.Code != http.StatusOK {
		t.Fatalf("rename = %d: %s", renameResponse.Code, renameResponse.Body.String())
	}
	if got, err := os.ReadFile(filepath.Join(svc.GoalDir("delivery"), "renamed.md")); err != nil || string(got) != "after" {
		t.Fatalf("renamed file = %q, err = %v", got, err)
	}

	protected := httptest.NewRequest(http.MethodPatch, "/api/v1/goals/delivery/files", strings.NewReader(`{"operation":"delete","sourcePath":"goal.json"}`))
	protected.Header.Set("Content-Type", "application/json")
	protectedResponse := httptest.NewRecorder()
	router.ServeHTTP(protectedResponse, protected)
	if protectedResponse.Code != http.StatusForbidden {
		t.Fatalf("protected metadata operation = %d: %s", protectedResponse.Code, protectedResponse.Body.String())
	}
}
