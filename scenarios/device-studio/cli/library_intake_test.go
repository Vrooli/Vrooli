package main

import (
	"context"
	"encoding/binary"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func cliIntakeGLB() []byte {
	chunk := []byte(`{"asset":{"version":"2.0"}}`)
	for len(chunk)%4 != 0 {
		chunk = append(chunk, ' ')
	}
	out := make([]byte, 20+len(chunk))
	copy(out, "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(chunk)))
	binary.LittleEndian.PutUint32(out[16:20], 0x4e4f534a)
	copy(out[20:], chunk)
	return out
}

func TestLibraryIntakeCLIImportsAndReadsTypedCandidateMetadata(t *testing.T) {
	var mux http.ServeMux
	var candidate *libraryv1.IntakeCandidateRevision
	mux.Handle(libraryconnect.LibraryCatalogServiceListIntakeCandidatesProcedure, connect.NewUnaryHandler(libraryconnect.LibraryCatalogServiceListIntakeCandidatesProcedure, func(_ context.Context, req *connect.Request[libraryv1.ListIntakeCandidatesRequest]) (*connect.Response[libraryv1.ListIntakeCandidatesResponse], error) {
		if req.Msg.PageSize != 25 || req.Msg.AfterCandidateId != "intake-next" {
			t.Errorf("unexpected intake list request: %v", req.Msg)
		}
		return connect.NewResponse(&libraryv1.ListIntakeCandidatesResponse{Candidates: []*libraryv1.IntakeCandidateRevision{candidate}}), nil
	}))
	mux.Handle(libraryconnect.LibraryCatalogServiceGetIntakeCandidateProcedure, connect.NewUnaryHandler(libraryconnect.LibraryCatalogServiceGetIntakeCandidateProcedure, func(_ context.Context, req *connect.Request[libraryv1.GetIntakeCandidateRequest]) (*connect.Response[libraryv1.GetIntakeCandidateResponse], error) {
		if req.Msg.CandidateId != "candidate-cli" || req.Msg.Revision != 1 {
			t.Errorf("unexpected intake get request: %v", req.Msg)
		}
		return connect.NewResponse(&libraryv1.GetIntakeCandidateResponse{Candidate: candidate}), nil
	}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_, _ = io.WriteString(w, `{"status":"ok"}`)
			return
		}
		if r.URL.Path == "/api/v1/library/intake-candidates" && r.Method == http.MethodPost {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "multipart/form-data" {
				t.Errorf("import did not use multipart/form-data: %q %v", mediaType, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			meta, err := reader.NextPart()
			if err != nil || meta.FormName() != "metadata" {
				t.Errorf("missing metadata part: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var claims libraryv1.IntakeCandidateClaims
			metadata, _ := io.ReadAll(meta)
			if err := protojson.Unmarshal(metadata, &claims); err != nil || claims.CandidateId != "candidate-cli" || claims.DisplayName != "CLI geometry" || claims.SourceLabel != "operator source" {
				t.Errorf("wrong intake claims: %+v %v", claims, err)
			}
			file, err := reader.NextPart()
			if err != nil || file.FormName() != "file" || file.FileName() != "model.glb" {
				t.Errorf("missing GLB part: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			contents, _ := io.ReadAll(file)
			if string(contents) != string(cliIntakeGLB()) {
				t.Errorf("uploaded GLB changed in transit")
			}
			candidate = &libraryv1.IntakeCandidateRevision{Id: claims.CandidateId, Revision: 1, RevisionSha256: strings.Repeat("a", 64), DisplayName: claims.DisplayName, OriginalFilename: "model.glb", OriginalSha256: strings.Repeat("b", 64), SizeBytes: int64(len(contents)), Mime: "model/gltf-binary", ReviewStatus: "unreviewed", RenderStatus: "not_reviewed"}
			response, _ := protojson.Marshal(&libraryv1.GetIntakeCandidateResponse{Candidate: candidate})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(response)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()
	t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
	t.Setenv("API_BASE_URL", "")
	t.Setenv("VITE_API_BASE_URL", "")
	filePath := filepath.Join(t.TempDir(), "model.glb")
	if err := os.WriteFile(filePath, cliIntakeGLB(), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	commands := [][]string{
		{"library", "intake-import", "--file", filePath, "--candidate-id", "candidate-cli", "--display-name", "CLI geometry", "--source-label", "operator source", "--license-claim", "CC0-1.0"},
		{"library", "intake-list", "--after-candidate-id", "intake-next", "--page-size", "25"},
		{"library", "intake-get", "candidate-cli", "--revision", "1"},
	}
	for _, args := range commands {
		if err := app.Run(args); err != nil {
			t.Fatalf("app.Run(%v): %v", args, err)
		}
	}
	if candidate == nil || candidate.ReviewStatus != "unreviewed" || candidate.RenderStatus != "not_reviewed" {
		t.Fatalf("test endpoint fabricated a promoted candidate: %+v", candidate)
	}
}
