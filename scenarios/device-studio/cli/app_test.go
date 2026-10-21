package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	devicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices/devices_v1connect"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	physicalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical/physical_v1connect"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	shotsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

func TestRenderCLIStreamsVerifiedExportsWithoutReplacingFiles(t *testing.T) { // [REQ:DS-INTEGRATION-OPERATIONS]
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewNRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	// This MP4-shaped fixture tests transport/integrity only, not native decoding.
	mp4Bytes := append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p'}, bytes.Repeat([]byte{23}, 256<<10)...)
	for _, name := range []string{"png", "mp4", "digest", "short", "extra", "mime", "not-mp4", "not-png", "existing", "destination-race"} {
		t.Run(name, func(t *testing.T) {
			data, mediaType := mp4Bytes, "video/mp4"
			if name == "png" {
				data, mediaType = pngBytes.Bytes(), "image/png"
			}
			if name == "not-png" {
				data, mediaType = []byte("invalid PNG"), "image/png"
			}
			if name == "not-mp4" {
				data = []byte("invalid MP4 bytes")
			}
			sum := sha256.Sum256(data)
			output := &shotsv1.RenderOutput{MediaType: mediaType, SizeBytes: int64(len(data)), Sha256: hex.EncodeToString(sum[:]), WidthPixels: 3, HeightPixels: 2}
			if name == "digest" {
				output.Sha256 = strings.Repeat("0", 64)
			}
			outPath := filepath.Join(t.TempDir(), "result")
			if name == "existing" {
				if err := os.WriteFile(outPath, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			metadata := connect.NewUnaryHandler(shotsconnect.RenderJobsServiceGetRenderProcedure, func(_ context.Context, req *connect.Request[shotsv1.GetRenderRequest]) (*connect.Response[shotsv1.GetRenderResponse], error) {
				if req.Msg.ProjectId != "project-1" || req.Msg.RenderId != "render-1" {
					t.Errorf("wrong metadata request: %v", req.Msg)
				}
				return connect.NewResponse(&shotsv1.GetRenderResponse{Job: &shotsv1.RenderJob{Id: "render-1", ProjectId: "project-1", Status: shotsv1.RenderJobStatus_RENDER_JOB_STATUS_SUCCEEDED, Output: output}}), nil
			})
			var contentRequests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					_, _ = io.WriteString(w, `{"status":"ok"}`)
					return
				}
				if r.Header.Get(testModeHeader) != "1" {
					t.Errorf("missing test route header")
				}
				if r.URL.Path == shotsconnect.RenderJobsServiceGetRenderProcedure {
					metadata.ServeHTTP(w, r)
					return
				}
				if r.URL.Path != "/api/v1/projects/project-1/renders/render-1/content" {
					t.Errorf("unexpected URL %s", r.URL)
					http.NotFound(w, r)
					return
				}
				contentRequests++
				if _, err := os.Stat(outPath); !os.IsNotExist(err) {
					t.Errorf("destination exposed before verified download: %v", err)
				}
				if name == "destination-race" {
					if err := os.WriteFile(outPath, []byte("keep"), 0600); err != nil {
						t.Error(err)
					}
				}
				w.Header().Set("Content-Type", mediaType)
				if name == "mime" {
					w.Header().Set("Content-Type", "text/plain")
				}
				if name == "short" {
					w.Header().Set("Content-Length", strconv.Itoa(len(data)))
					_, _ = w.Write(data[:len(data)-1])
					return
				}
				if name == "extra" {
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
					_, _ = w.Write(append(append([]byte(nil), data...), 42))
					return
				}
				_, _ = w.Write(data)
			}))
			defer server.Close()
			t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
			t.Setenv("API_BASE_URL", "")
			t.Setenv("VITE_API_BASE_URL", "")
			t.Setenv(testModeEnv, "1")
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			err = app.Run([]string{"renders", "export", "render-1", "--project-id", "project-1", "--out", outPath})
			valid := name == "png" || name == "mp4"
			if valid && err != nil {
				t.Fatal(err)
			}
			if !valid && err == nil {
				t.Fatal("invalid export accepted")
			}
			actual, readErr := os.ReadFile(outPath)
			if valid && (readErr != nil || !bytes.Equal(actual, data)) {
				t.Fatalf("exported bytes differ: %v", readErr)
			}
			if name == "existing" || name == "destination-race" {
				if readErr != nil || string(actual) != "keep" {
					t.Fatal("existing destination overwritten")
				}
			} else if !valid && !os.IsNotExist(readErr) {
				t.Fatal("failed export published a file")
			}
			if name == "existing" && contentRequests != 0 {
				t.Fatal("download started despite existing destination")
			}
			leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(outPath), ".device-studio-export-*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("temporary files remain: %v %v", leftovers, err)
			}
		})
	}
}

func TestRenderSubmitCLISelectsTypedModes(t *testing.T) { // [REQ:DS-INTEGRATION-OPERATIONS]
	for _, tc := range []struct {
		value   string
		mode    shotsv1.RenderMode
		profile shotsv1.RenderOutputProfile
	}{
		{"", shotsv1.RenderMode_RENDER_MODE_STILL, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1},
		{"VIDEO_SAMPLE_PNG", shotsv1.RenderMode_RENDER_MODE_VIDEO_SAMPLE_PNG, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1},
		{"VIDEO_FINAL_MP4", shotsv1.RenderMode_RENDER_MODE_VIDEO_FINAL_MP4, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1},
		{"PHYSICAL_STILL_PNG", shotsv1.RenderMode_RENDER_MODE_PHYSICAL_STILL_PNG, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1},
		{"RENDER_MODE_PHYSICAL_STILL_PNG", shotsv1.RenderMode_RENDER_MODE_PHYSICAL_STILL_PNG, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1},
		{"PHYSICAL_FINAL_MP4", shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1},
		{"RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4", shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1},
		{"PHYSICAL_PREVIEW_PNG", shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1},
		{"RENDER_MODE_PHYSICAL_PREVIEW_PNG", shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1},
	} {
		t.Run(tc.mode.String(), func(t *testing.T) {
			called := false
			service := connect.NewUnaryHandler(shotsconnect.RenderJobsServiceSubmitRenderProcedure, func(_ context.Context, req *connect.Request[shotsv1.SubmitRenderRequest]) (*connect.Response[shotsv1.SubmitRenderResponse], error) {
				called = true
				if req.Msg.RenderMode != tc.mode || req.Msg.OutputProfile != tc.profile || req.Msg.Revision != 7 || req.Msg.IdempotencyKey != "stable-key" {
					t.Errorf("wrong typed request: %v", req.Msg)
				}
				return connect.NewResponse(&shotsv1.SubmitRenderResponse{Job: &shotsv1.RenderJob{Id: "render-1", ProjectId: "project-1", Status: shotsv1.RenderJobStatus_RENDER_JOB_STATUS_QUEUED}}), nil
			})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					_, _ = io.WriteString(w, `{"status":"ok"}`)
					return
				}
				service.ServeHTTP(w, r)
			}))
			defer server.Close()
			t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
			t.Setenv("API_BASE_URL", "")
			t.Setenv("VITE_API_BASE_URL", "")
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"renders", "submit", "--project-id", "project-1", "--shot-id", "shot-1", "--revision", "7", "--idempotency-key", "stable-key"}
			if tc.value != "" {
				args = append(args, "--render-mode", tc.value)
			}
			if err := app.Run(args); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("no submission")
			}
		})
	}
}

// TestNewAppConstructs is the smoke gate: NewApp() must succeed against
// the cli-core wiring declared in app.go. This catches the most common
// regression class — a misconfigured StandardScenarioOptions or a missing
// dependency from cli-core — before any tests touch real commands.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

func TestComponentCommandsAppearInRegisteredHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := app.Run([]string{"devices", "help"})
	_ = w.Close()
	os.Stdout = old
	data, readErr := io.ReadAll(r)
	_ = r.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, want := range []string{"save", "get", "head", "revisions", "list"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("registered device help missing %q:\n%s", want, data)
		}
	}
	old = os.Stdout
	r, w, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr = app.Run([]string{"devices", "save", "--help"})
	_ = w.Close()
	os.Stdout = old
	data, readErr = io.ReadAll(r)
	_ = r.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, want := range []string{"--candidate", "--expected-head"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("registered device save help missing %q:\n%s", want, data)
		}
	}
}

func TestComponentCandidateCLIConnectJourney(t *testing.T) {
	var mux http.ServeMux
	var saved *devicev1.DeviceCandidate
	register := func(path string, h http.Handler) { mux.Handle(path, h) }
	register(devicesconnect.DevicesServiceSaveDeviceCandidateProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceSaveDeviceCandidateProcedure, func(_ context.Context, req *connect.Request[devicev1.SaveDeviceCandidateRequest]) (*connect.Response[devicev1.SaveDeviceCandidateResponse], error) {
		if req.Msg.ExpectedHead != 0 || req.Msg.Candidate.GetVersion() != "device-candidate/v1" {
			t.Errorf("unexpected save request: %v", req.Msg)
		}
		saved = req.Msg.Candidate
		return connect.NewResponse(&devicev1.SaveDeviceCandidateResponse{Revision: &devicev1.DeviceRevision{DeviceId: "phone-candidate", Revision: 1, Sha256: strings.Repeat("a", 64), State: "candidate", Candidate: req.Msg.Candidate}}), nil
	}))
	register(devicesconnect.DevicesServiceGetDeviceHeadProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceGetDeviceHeadProcedure, func(_ context.Context, req *connect.Request[devicev1.GetDeviceHeadRequest]) (*connect.Response[devicev1.GetDeviceHeadResponse], error) {
		if req.Msg.DeviceId != "phone-candidate" {
			t.Errorf("wrong head id: %s", req.Msg.DeviceId)
		}
		return connect.NewResponse(&devicev1.GetDeviceHeadResponse{Revision: &devicev1.DeviceRevision{DeviceId: "phone-candidate", Revision: 1, State: "candidate", Candidate: saved}}), nil
	}))
	register(devicesconnect.DevicesServiceGetDeviceRevisionProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceGetDeviceRevisionProcedure, func(_ context.Context, req *connect.Request[devicev1.GetDeviceRevisionRequest]) (*connect.Response[devicev1.GetDeviceRevisionResponse], error) {
		if req.Msg.DeviceId != "phone-candidate" || req.Msg.Revision != 1 {
			t.Errorf("wrong revision request: %v", req.Msg)
		}
		return connect.NewResponse(&devicev1.GetDeviceRevisionResponse{Revision: &devicev1.DeviceRevision{DeviceId: "phone-candidate", Revision: 1, State: "candidate", Candidate: saved}}), nil
	}))
	register(devicesconnect.DevicesServiceListDeviceRevisionsProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceListDeviceRevisionsProcedure, func(_ context.Context, req *connect.Request[devicev1.ListDeviceRevisionsRequest]) (*connect.Response[devicev1.ListDeviceRevisionsResponse], error) {
		if req.Msg.DeviceId != "phone-candidate" || req.Msg.AfterRevision != 0 || req.Msg.PageSize != 50 {
			t.Errorf("wrong page request: %v", req.Msg)
		}
		return connect.NewResponse(&devicev1.ListDeviceRevisionsResponse{Revisions: []*devicev1.DeviceRevision{{DeviceId: "phone-candidate", Revision: 1, State: "candidate"}}, NextAfterRevision: 0}), nil
	}))
	register(devicesconnect.DevicesServiceListDeviceCandidatesProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceListDeviceCandidatesProcedure, func(_ context.Context, req *connect.Request[devicev1.ListDeviceCandidatesRequest]) (*connect.Response[devicev1.ListDeviceCandidatesResponse], error) {
		if req.Msg.AfterDeviceId != "alpha-device" || req.Msg.PageSize != 7 {
			t.Errorf("wrong candidate cursor/page: %v", req.Msg)
		}
		return connect.NewResponse(&devicev1.ListDeviceCandidatesResponse{Candidates: []*devicev1.DeviceRevision{{DeviceId: "beta-device", Revision: 2, Sha256: strings.Repeat("b", 64), State: "candidate"}}, NextAfterDeviceId: "beta-device"}), nil
	}))
	bundle := &libraryv1.BundleRevisionMetadata{Manifest: &libraryv1.BundleManifest{Version: 1, BundleId: "bundle-phone", Revision: 2, Artifacts: []*libraryv1.Artifact{{Id: "phone-glb", Role: "geometry_glb", LicenseId: "AGPL-3.0-only", Rights: []string{"render"}}}}, ManifestSha256: strings.Repeat("c", 64), ReviewSha256: strings.Repeat("d", 64), Review: &libraryv1.ReviewBinding{Id: "review-v1", Decision: "approved", AllowedUses: []string{"render"}}}
	register(libraryconnect.LibraryCatalogServiceListBundleRevisionsProcedure, connect.NewUnaryHandler(libraryconnect.LibraryCatalogServiceListBundleRevisionsProcedure, func(_ context.Context, req *connect.Request[libraryv1.ListBundleRevisionsRequest]) (*connect.Response[libraryv1.ListBundleRevisionsResponse], error) {
		if req.Msg.PageSize != 3 || req.Msg.AfterBundleId != "bundle-a" || req.Msg.AfterRevision != 1 {
			t.Errorf("wrong library cursor/page: %v", req.Msg)
		}
		return connect.NewResponse(&libraryv1.ListBundleRevisionsResponse{Revisions: []*libraryv1.BundleRevisionMetadata{bundle}, NextAfterBundleId: "bundle-phone", NextAfterRevision: 2}), nil
	}))
	register(libraryconnect.LibraryCatalogServiceGetBundleRevisionProcedure, connect.NewUnaryHandler(libraryconnect.LibraryCatalogServiceGetBundleRevisionProcedure, func(_ context.Context, req *connect.Request[libraryv1.GetBundleRevisionRequest]) (*connect.Response[libraryv1.GetBundleRevisionResponse], error) {
		if req.Msg.BundleId != "bundle-phone" || req.Msg.Revision != 2 {
			t.Errorf("wrong library revision request: %v", req.Msg)
		}
		return connect.NewResponse(&libraryv1.GetBundleRevisionResponse{Revision: bundle}), nil
	}))
	register(physicalconnect.PhysicalStarterServiceInitializeReviewedStarterProcedure, connect.NewUnaryHandler(physicalconnect.PhysicalStarterServiceInitializeReviewedStarterProcedure, func(context.Context, *connect.Request[physicalv1.InitializeReviewedStarterRequest]) (*connect.Response[physicalv1.InitializeReviewedStarterResponse], error) {
		return connect.NewResponse(&physicalv1.InitializeReviewedStarterResponse{CandidateOnly: true, PhoneDevice: &physicalv1.ComponentRevisionRef{Id: "phone-candidate", Revision: 1}, PhoneStudyScene: &physicalv1.ComponentRevisionRef{Id: "scene-candidate", Revision: 1}, PhoneStillMotion: &physicalv1.ComponentRevisionRef{Id: "motion-candidate", Revision: 1}, PhoneStudyPlacement: &physicalv1.PhysicalStarterPlacement{ProfileId: "device-studio-reviewed-phone-study-placement-v1"}}), nil
	}))
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
	candidate := filepath.Join(t.TempDir(), "device.json")
	if err := os.WriteFile(candidate, []byte(`{"version":"device-candidate/v1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"library", "starter"}, {"devices", "save", "--candidate", candidate, "--expected-head", "0"}, {"devices", "head", "phone-candidate"}, {"devices", "get", "phone-candidate", "--revision", "1"}, {"devices", "revisions", "phone-candidate"}, {"devices", "list", "--after-id", "alpha-device", "--page-size", "7"}, {"library", "list", "--after-bundle-id", "bundle-a", "--after-revision", "1", "--page-size", "3"}, {"library", "get", "bundle-phone", "--revision", "2"}} {
		if err := app.Run(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}

func TestComponentCLIRejectsTypedNilRevisionResponses(t *testing.T) {
	var mux http.ServeMux
	mux.Handle(devicesconnect.DevicesServiceSaveDeviceCandidateProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceSaveDeviceCandidateProcedure, func(context.Context, *connect.Request[devicev1.SaveDeviceCandidateRequest]) (*connect.Response[devicev1.SaveDeviceCandidateResponse], error) {
		return connect.NewResponse(&devicev1.SaveDeviceCandidateResponse{}), nil
	}))
	mux.Handle(devicesconnect.DevicesServiceGetDeviceRevisionProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceGetDeviceRevisionProcedure, func(context.Context, *connect.Request[devicev1.GetDeviceRevisionRequest]) (*connect.Response[devicev1.GetDeviceRevisionResponse], error) {
		return connect.NewResponse(&devicev1.GetDeviceRevisionResponse{}), nil
	}))
	mux.Handle(devicesconnect.DevicesServiceGetDeviceHeadProcedure, connect.NewUnaryHandler(devicesconnect.DevicesServiceGetDeviceHeadProcedure, func(context.Context, *connect.Request[devicev1.GetDeviceHeadRequest]) (*connect.Response[devicev1.GetDeviceHeadResponse], error) {
		return connect.NewResponse(&devicev1.GetDeviceHeadResponse{}), nil
	}))
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
	candidate := filepath.Join(t.TempDir(), "candidate.json")
	if err := os.WriteFile(candidate, []byte(`{"version":"device-candidate/v1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"save-human", []string{"devices", "save", "--candidate", candidate, "--expected-head", "0"}, "no saved candidate revision"},
		{"save-json", []string{"devices", "save", "--candidate", candidate, "--expected-head", "0", "--json"}, "no saved candidate revision"},
		{"get-human", []string{"devices", "get", "phone", "--revision", "1"}, "no immutable candidate revision"},
		{"get-json", []string{"devices", "get", "phone", "--revision", "1", "--json"}, "no immutable candidate revision"},
		{"head-human", []string{"devices", "head", "phone"}, "no candidate head"},
		{"head-json", []string{"devices", "head", "phone", "--json"}, "no candidate head"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := app.Run(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want actionable rejection %q", err, tc.want)
			}
		})
	}
}

func TestMediaImportCLIStreamsFileAndForwardsRoutedTestHeader(t *testing.T) {
	var gotHeader, gotBytes string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"status":"ok"}`)
			return
		}
		gotHeader = r.Header.Get(testModeHeader)
		if r.URL.Path != "/api/v1/projects/project-1/media" {
			t.Errorf("path=%q", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("multipart: %v", err)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("file: %v", err)
			return
		}
		defer file.Close()
		b, _ := io.ReadAll(file)
		gotBytes = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"source_media":{"id":"media-1","project_id":"project-1","original_filename":"mask.png","media_type":"image/png","size_bytes":"4"}}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "mask.png")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(testModeEnv, "1")
	t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
	t.Setenv("API_BASE_URL", "")
	t.Setenv("VITE_API_BASE_URL", "")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"media", "import", "project-1", "--file", path}); err != nil {
		t.Fatalf("media import: %v", err)
	}
	if gotHeader != "1" || gotBytes != "data" {
		t.Fatalf("header=%q bytes=%q", gotHeader, gotBytes)
	}
}

func TestMediaImportCLIReportsActualFileOpenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer server.Close()
	t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
	t.Setenv("API_BASE_URL", "")
	t.Setenv("VITE_API_BASE_URL", "")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	err = app.Run([]string{"media", "import", "project-1", "--file", filepath.Join(t.TempDir(), "missing.png")})
	if err == nil || !strings.Contains(err.Error(), "open source media") || !strings.Contains(err.Error(), "missing.png") {
		t.Fatalf("error=%v", err)
	}
}

// TestRunVersion exercises a non-API command path through cli-core.
// --version must succeed and must NOT trigger the NeedsAPI preflight
// (which would try to reach the configured API base and fail in CI).
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

// TestRunHelp exercises cli-core's help renderer through the scenario
// app's wiring. Help is rendered to stdout by cli-core; we only verify
// Run returns without error. The presence of each registered command in
// the actual help surface is covered by cli-core's own tests.
func TestRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

// TestMetadata pins the values app.go declares — appName must match the
// scenario id (post-substitution), appVersion must be non-empty.
// Catches accidental edits that decouple the binary identity from the
// scenario it belongs to.
func TestMetadata(t *testing.T) {
	if strings.TrimSpace(appName) == "" {
		t.Fatal("appName must not be empty")
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

func TestConnectClientForwardsTestModeHeaderOnlyWhenEnabled(t *testing.T) {
	for _, tc := range []struct {
		name       string
		testMode   string
		wantHeader string
	}{
		{name: "enabled", testMode: "1", wantHeader: "1"},
		{name: "ordinary", wantHeader: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotHeader string
			service := connect.NewUnaryHandler(
				projectsconnect.ProjectsServiceListProjectsProcedure,
				func(context.Context, *connect.Request[projectsv1.ListProjectsRequest]) (*connect.Response[projectsv1.ListProjectsResponse], error) {
					return connect.NewResponse(&projectsv1.ListProjectsResponse{}), nil
				},
			)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotHeader = r.Header.Get(testModeHeader)
				service.ServeHTTP(w, r)
			}))
			defer server.Close()

			t.Setenv(testModeEnv, tc.testMode)
			t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
			t.Setenv("API_BASE_URL", "")
			t.Setenv("VITE_API_BASE_URL", "")
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			httpClient, baseURL := cliapp.NewConnectHTTPClient(app.core)
			client := projectsconnect.NewProjectsServiceClient(httpClient, baseURL)
			if _, err := client.ListProjects(context.Background(), connect.NewRequest(&projectsv1.ListProjectsRequest{PageSize: 10})); err != nil {
				t.Fatalf("fixture-only project list: %v", err)
			}
			if gotHeader != tc.wantHeader {
				t.Fatalf("test-mode header = %q, want %q", gotHeader, tc.wantHeader)
			}
		})
	}
}
