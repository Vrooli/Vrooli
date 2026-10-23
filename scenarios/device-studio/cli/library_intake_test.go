package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
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

func TestLibraryMaterialCLIUsesExactParentAndStableChildRESTContract(t *testing.T) {
	parentSHA := strings.Repeat("a", 64)
	var mux http.ServeMux
	mux.HandleFunc("/api/v1/library/intake-candidates/material-parent/revisions/1/materials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("revision_sha256") != parentSHA {
			t.Errorf("unexpected material inventory request: %s %s", r.Method, r.URL.String())
		}
		_, _ = io.WriteString(w, `{"candidate_id":"material-parent","revision":1,"revision_sha256":"`+parentSHA+`","glb_sha256":"`+strings.Repeat("b", 64)+`","materials":[{"index":0,"name":"shell","base_color_factor":[1,1,1,1],"metallic_factor":0,"roughness_factor":0.8,"emissive_factor":[0,0,0],"editable":true}]}`)
	})
	mux.HandleFunc("/api/v1/library/intake-candidates/material-parent/revisions/1/material-corrections", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode correction: %v", err)
		}
		if r.Method != http.MethodPost || body["parent_revision_sha256"] != parentSHA || body["child_candidate_id"] != "material-child" || body["material_name"] != "shell" || body["roughness_factor"] != 0.2 {
			t.Errorf("unexpected correction request: %+v", body)
		}
		_, _ = io.WriteString(w, `{"candidate":{"id":"material-child","revision":1,"revision_sha256":"`+strings.Repeat("c", 64)+`","original_sha256":"`+strings.Repeat("d", 64)+`","review_status":"unreviewed","render_status":"not_reviewed","parent_candidate_id":"material-parent","parent_revision":1,"parent_sha256":"`+parentSHA+`","import_method":"local_material_correction"}}`)
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_, _ = io.WriteString(w, `{"status":"ok"}`)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()
	t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
	t.Setenv("API_BASE_URL", "")
	t.Setenv("VITE_API_BASE_URL", "")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"library", "intake-materials", "material-parent", "--revision", "1", "--revision-sha256", parentSHA}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"library", "intake-material-correct", "material-parent", "--revision", "1", "--revision-sha256", parentSHA, "--child-candidate-id", "material-child", "--material-index", "0", "--material-name", "shell", "--roughness", "0.2"}); err != nil {
		t.Fatal(err)
	}
}

func TestLibraryIntakePreviewCLIVerifiesExactRevisionAndDoesNotOverwrite(t *testing.T) {
	revisionSHA := strings.Repeat("a", 64)
	glbSHA := strings.Repeat("b", 64)
	img := image.NewRGBA(image.Rect(0, 0, 768, 256))
	img.SetRGBA(12, 12, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	previewBytes := encoded.Bytes()
	previewHash := sha256.Sum256(previewBytes)
	previewSHA := hex.EncodeToString(previewHash[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_, _ = io.WriteString(w, `{"status":"ok"}`)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/library/intake-candidates/candidate-preview/revisions/1/preview" || r.URL.Query().Get("revision_sha256") != revisionSHA {
			t.Errorf("unexpected preview request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("X-Intake-Candidate-ID", "candidate-preview")
		w.Header().Set("X-Intake-Revision", "1")
		w.Header().Set("X-Intake-Revision-SHA256", revisionSHA)
		w.Header().Set("X-Intake-GLB-SHA256", glbSHA)
		w.Header().Set("X-Intake-Preview-SHA256", previewSHA)
		w.Header().Set("X-Intake-Preview-Vertex-Colors", "true")
		w.Header().Set("X-Intake-Preview-Scale", "not-estimated")
		w.Header().Set("X-Intake-Preview-Review", "unreviewed")
		_, _ = w.Write(previewBytes)
	}))
	defer server.Close()
	t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
	t.Setenv("API_BASE_URL", "")
	t.Setenv("VITE_API_BASE_URL", "")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "candidate.png")
	args := []string{"library", "intake-preview", "candidate-preview", "--revision", "1", "--revision-sha256", revisionSHA, "--out", out}
	if err := app.Run(args); err != nil {
		t.Fatalf("preview CLI failed: %v", err)
	}
	written, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(written, previewBytes) {
		t.Fatalf("preview output mismatch: %v", err)
	}
	if err := app.Run(args); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing preview path was overwritten or gave wrong error: %v", err)
	}
}
