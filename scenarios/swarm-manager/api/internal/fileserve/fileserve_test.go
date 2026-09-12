package fileserve

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/fileops"

	"github.com/gorilla/mux"

	"google.golang.org/protobuf/encoding/protojson"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// --- FileNodesToProto / FileNodeToProto ---

func TestFileNodesToProto_NilAndEmpty(t *testing.T) {
	if got := FileNodesToProto(nil); got != nil {
		t.Fatalf("nil slice: want nil, got %v", got)
	}
	if got := FileNodesToProto([]fileops.FileNode{}); got != nil {
		t.Fatalf("empty slice: want nil, got %v", got)
	}
}

func TestFileNodeToProto_FileSetsSizeDirLeavesNil(t *testing.T) {
	fileNode := fileops.FileNode{Name: "a.txt", Path: "a.txt", Type: "file", Size: 42}
	got := FileNodeToProto(fileNode)
	if got.GetName() != "a.txt" || got.GetPath() != "a.txt" || got.GetType() != "file" {
		t.Fatalf("file node fields wrong: %+v", got)
	}
	if got.Size == nil {
		t.Fatalf("file node: Size should be non-nil")
	}
	if *got.Size != 42 {
		t.Fatalf("file node: Size = %d, want 42", *got.Size)
	}

	dirNode := fileops.FileNode{Name: "d", Path: "d", Type: "directory", Size: 99}
	gotDir := FileNodeToProto(dirNode)
	if gotDir.Size != nil {
		t.Fatalf("directory node: Size should be nil, got %d", *gotDir.Size)
	}
}

func TestFileNodesToProto_RecursesChildren(t *testing.T) {
	nodes := []fileops.FileNode{
		{
			Name: "dir", Path: "dir", Type: "directory",
			Children: []fileops.FileNode{
				{Name: "child.go", Path: "dir/child.go", Type: "file", Size: 7},
			},
		},
	}
	got := FileNodesToProto(nodes)
	if len(got) != 1 {
		t.Fatalf("want 1 top node, got %d", len(got))
	}
	children := got[0].GetChildren()
	if len(children) != 1 {
		t.Fatalf("want 1 child, got %d", len(children))
	}
	child := children[0]
	if child.GetName() != "child.go" || child.GetPath() != "dir/child.go" {
		t.Fatalf("child fields wrong: %+v", child)
	}
	if child.Size == nil || *child.Size != 7 {
		t.Fatalf("child size wrong: %v", child.Size)
	}
	// Parent directory must not carry a size.
	if got[0].Size != nil {
		t.Fatalf("parent dir size should be nil")
	}
}

// --- GetContent ---

func newGetContentRequest(filePath string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	return mux.SetURLVars(req, map[string]string{"filepath": filePath})
}

func TestGetContent_HappyPath(t *testing.T) {
	root := t.TempDir()
	want := []byte("{\"hello\":true}")
	if err := os.WriteFile(filepath.Join(root, "data.json"), want, 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	GetContent(rec, newGetContentRequest("data.json"), root, "[test]")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != string(want) {
		t.Fatalf("body = %q, want %q", rec.Body.String(), want)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
}

func TestGetContent_PathTraversal(t *testing.T) {
	root := t.TempDir()
	rec := httptest.NewRecorder()
	GetContent(rec, newGetContentRequest("../etc/passwd"), root, "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetContent_IsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	GetContent(rec, newGetContentRequest("sub"), root, "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "directory") {
		t.Fatalf("body should mention directory, got %q", rec.Body.String())
	}
}

func TestGetContent_NotFound(t *testing.T) {
	root := t.TempDir()
	rec := httptest.NewRecorder()
	GetContent(rec, newGetContentRequest("missing.txt"), root, "[test]")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

// --- Upload ---

// buildMultipart returns a body and content-type for a multipart upload with
// the given file field, filename, and optional extra form fields.
func buildMultipart(t *testing.T, fieldName, filename string, content []byte, extra map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	for k, v := range extra {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if fieldName != "" {
		part, err := mw.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return body, mw.FormDataContentType()
}

func TestUpload_HappyPath(t *testing.T) {
	root := t.TempDir()
	content := []byte("uploaded bytes")
	body, contentType := buildMultipart(t, "file", "note.txt", content, nil)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	Upload(rec, req, root, "spec.json", "[test]")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	onDisk, err := os.ReadFile(filepath.Join(root, "note.txt"))
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if !bytes.Equal(onDisk, content) {
		t.Fatalf("on-disk bytes = %q, want %q", onDisk, content)
	}

	var resp apipb.BacklogFileResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rec.Body.String())
	}
	if resp.GetFile().GetName() != "note.txt" || resp.GetFile().GetType() != "file" {
		t.Fatalf("response file node wrong: %+v", resp.GetFile())
	}
	if resp.GetFile().Size == nil || *resp.GetFile().Size != int64(len(content)) {
		t.Fatalf("response size wrong: %v", resp.GetFile().Size)
	}
}

func TestUpload_MissingFileField(t *testing.T) {
	root := t.TempDir()
	body, contentType := buildMultipart(t, "", "", nil, map[string]string{"path": "x"})
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	Upload(rec, req, root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpload_ProtectedFile(t *testing.T) {
	root := t.TempDir()
	body, contentType := buildMultipart(t, "file", "spec.json", []byte("nope"), nil)
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	Upload(rec, req, root, "spec.json", "[test]")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "spec.json")); !os.IsNotExist(err) {
		t.Fatalf("protected file should not have been written")
	}
}

func TestUpload_MalformedMultipart(t *testing.T) {
	root := t.TempDir()
	// No multipart content-type set → ParseMultipartForm fails.
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("not multipart"))
	rec := httptest.NewRecorder()

	Upload(rec, req, root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpload_NestedPath(t *testing.T) {
	root := t.TempDir()
	content := []byte("nested")
	body, contentType := buildMultipart(t, "file", "deep.txt", content, map[string]string{"path": "a/b"})
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	Upload(rec, req, root, "spec.json", "[test]")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	onDisk, err := os.ReadFile(filepath.Join(root, "a", "b", "deep.txt"))
	if err != nil {
		t.Fatalf("nested file not written: %v", err)
	}
	if !bytes.Equal(onDisk, content) {
		t.Fatalf("nested bytes = %q, want %q", onDisk, content)
	}

	var resp apipb.BacklogFileResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.GetFile().GetPath() != "a/b/deep.txt" {
		t.Fatalf("response path = %q, want a/b/deep.txt", resp.GetFile().GetPath())
	}
}

func TestUpload_TargetIsExistingDirectory(t *testing.T) {
	root := t.TempDir()
	// Pre-create a directory whose name collides with the upload filename.
	if err := os.Mkdir(filepath.Join(root, "collide"), 0o755); err != nil {
		t.Fatal(err)
	}
	body, contentType := buildMultipart(t, "file", "collide", []byte("data"), nil)
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	Upload(rec, req, root, "spec.json", "[test]")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpload_InvalidPath(t *testing.T) {
	root := t.TempDir()
	body, contentType := buildMultipart(t, "file", "f.txt", []byte("data"), map[string]string{"path": "../escape"})
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	Upload(rec, req, root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

// --- Operate ---

func newOperateRequest(t *testing.T, req *apipb.BacklogFileOperationRequest) *http.Request {
	t.Helper()
	payload, err := protojson.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/operate", bytes.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func strptr(s string) *string { return &s }

func TestOperate_Delete(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gone.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:  "delete",
		SourcePath: "gone.txt",
	}), root, "spec.json", "[test]")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "gone.txt")); !os.IsNotExist(err) {
		t.Fatalf("file should have been deleted")
	}
	var resp apipb.BacklogFileOperationResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.GetDeletedPath() != "gone.txt" {
		t.Fatalf("deleted_path = %q, want gone.txt", resp.GetDeletedPath())
	}
}

func TestOperate_RenameSameDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:       "rename",
		SourcePath:      "old.txt",
		DestinationPath: strptr("new.txt"),
	}), root, "spec.json", "[test]")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old.txt should be gone")
	}
	got, err := os.ReadFile(filepath.Join(root, "new.txt"))
	if err != nil || string(got) != "data" {
		t.Fatalf("new.txt wrong: err=%v content=%q", err, got)
	}
	var resp apipb.BacklogFileOperationResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.GetFile().GetName() != "new.txt" {
		t.Fatalf("response file name = %q, want new.txt", resp.GetFile().GetName())
	}
}

func TestOperate_RenameCrossDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:       "rename",
		SourcePath:      "old.txt",
		DestinationPath: strptr("sub/new.txt"),
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_Move(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "src.txt"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:       "move",
		SourcePath:      "src.txt",
		DestinationPath: strptr("dir/dst.txt"),
	}), root, "spec.json", "[test]")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "src.txt")); !os.IsNotExist(err) {
		t.Fatalf("source should be gone after move")
	}
	got, err := os.ReadFile(filepath.Join(root, "dir", "dst.txt"))
	if err != nil || string(got) != "payload" {
		t.Fatalf("moved file wrong: err=%v content=%q", err, got)
	}
}

func TestOperate_Copy(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "orig.txt"), []byte("copyme"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:       "copy",
		SourcePath:      "orig.txt",
		DestinationPath: strptr("copy.txt"),
	}), root, "spec.json", "[test]")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	src, err := os.ReadFile(filepath.Join(root, "orig.txt"))
	if err != nil || string(src) != "copyme" {
		t.Fatalf("source should still exist: err=%v content=%q", err, src)
	}
	dst, err := os.ReadFile(filepath.Join(root, "copy.txt"))
	if err != nil || string(dst) != "copyme" {
		t.Fatalf("copy dest wrong: err=%v content=%q", err, dst)
	}
}

func TestOperate_MissingBody(t *testing.T) {
	root := t.TempDir()
	r := httptest.NewRequest(http.MethodPost, "/operate", nil)
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	Operate(rec, r, root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_InvalidBody(t *testing.T) {
	root := t.TempDir()
	r := httptest.NewRequest(http.MethodPost, "/operate", strings.NewReader("{not valid json"))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	Operate(rec, r, root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_ProtectedSource(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "spec.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:  "delete",
		SourcePath: "spec.json",
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "spec.json")); err != nil {
		t.Fatalf("protected source must not be deleted: %v", err)
	}
}

func TestOperate_NonexistentSource(t *testing.T) {
	root := t.TempDir()
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:  "delete",
		SourcePath: "nope.txt",
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_DestinationExists(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:       "rename",
		SourcePath:      "a.txt",
		DestinationPath: strptr("b.txt"),
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_ProtectedDestination(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "src.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:       "rename",
		SourcePath:      "src.txt",
		DestinationPath: strptr("spec.json"),
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_MissingDestination(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "src.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	// rename with no destination_path → BadRequest "destination_path is required".
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:  "rename",
		SourcePath: "src.txt",
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_UnsupportedOperation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	// "frobnicate" is not in the proto enum, so validation rejects it as 400.
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:  "frobnicate",
		SourcePath: "x.txt",
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperate_DirectoryIntoItself(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "parent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "parent", "f.txt"), []byte("f"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Operate(rec, newOperateRequest(t, &apipb.BacklogFileOperationRequest{
		Operation:       "copy",
		SourcePath:      "parent",
		DestinationPath: strptr("parent/child"),
	}), root, "spec.json", "[test]")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}
