package handlers

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	hexport "github.com/vrooli/browser-automation-studio/handlers/export"
	"github.com/vrooli/browser-automation-studio/internal/testutil/hubmocks"
	"github.com/vrooli/browser-automation-studio/internal/testutil/videofixture"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/export/source"
	ws "github.com/vrooli/browser-automation-studio/websocket"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
)

type exportFailureRepo struct {
	*MockRepository
	completeErr, errorErr error
}

func (r *exportFailureRepo) UpdateExportComplete(ctx context.Context, id uuid.UUID, path string, size int64, settings database.JSONMap) error {
	if r.completeErr != nil {
		return r.completeErr
	}
	return r.MockRepository.UpdateExportComplete(ctx, id, path, size, settings)
}
func (r *exportFailureRepo) UpdateExportStatus(ctx context.Context, id uuid.UUID, status, message string) error {
	if r.errorErr != nil {
		return r.errorErr
	}
	return r.MockRepository.UpdateExportStatus(ctx, id, status, message)
}

type exportEventHub struct {
	hubmocks.Hub
	events []*ws.ExportProgress
}

func (h *exportEventHub) BroadcastExportProgress(event *ws.ExportProgress) {
	copy := *event
	h.events = append(h.events, &copy)
}

func TestExportCompletionEventsRequirePersistedResult(t *testing.T) {
	for _, test := range []struct {
		name                  string
		completeErr, errorErr error
		terminal              string
		input                 []byte
	}{
		{name: "success", terminal: "completed"},
		{name: "corrupt raw media", input: []byte("not a video"), terminal: "failed"},
		{name: "completion commit fails", completeErr: errors.New("completion database unavailable"), terminal: "failed"},
		{name: "both commits fail", completeErr: errors.New("completion unavailable"), errorErr: errors.New("status unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			id := uuid.New()
			root := t.TempDir()
			input := filepath.Join(root, id.String(), "artifacts", "videos", "recording.mp4")
			if err := os.MkdirAll(filepath.Dir(input), 0700); err != nil {
				t.Fatal(err)
			}
			data := test.input
			if data == nil {
				data = videofixture.MP4()
			}
			if err := os.WriteFile(input, data, 0600); err != nil {
				t.Fatal(err)
			}
			repo := &exportFailureRepo{MockRepository: NewMockRepository(), completeErr: test.completeErr, errorErr: test.errorErr}
			repo.AddExecution(&database.ExecutionIndex{ID: id, ResultPath: "result.json"})
			row := &database.ExportIndex{ID: uuid.New(), ExecutionID: id, Name: "film", Format: "mp4", Status: "processing", Settings: database.JSONMap{}}
			if err := repo.CreateExport(t.Context(), row); err != nil {
				t.Fatal(err)
			}
			hub := &exportEventHub{}
			h := &Handler{repo: repo, wsHub: hub, recordingsRoot: root, log: logrus.New()}
			output := t.TempDir()
			h.renderExportInBackground(t.Context(), row, id, "mp4", hexport.Request{Stylization: "raw"}, output, source.RenderSourceRecordedVideo)
			var terminal string
			for _, event := range hub.events {
				if event.Status != "processing" {
					if terminal != "" {
						t.Fatal("multiple terminal events")
					}
					terminal = event.Status
				}
			}
			if terminal != test.terminal {
				t.Fatalf("terminal=%q want %q events=%+v", terminal, test.terminal, hub.events)
			}
			persisted, err := repo.GetExport(t.Context(), row.ID)
			if err != nil {
				t.Fatal(err)
			}
			if test.terminal != "" && persisted.Status != test.terminal {
				t.Fatalf("event disagrees with row: %+v", persisted)
			}
			if test.terminal == "completed" && len(persisted.Settings["output_sha256"].(string)) != 64 {
				t.Fatal("success did not persist exact output identity")
			}
			if test.input != nil {
				if _, err := os.Stat(filepath.Join(output, "film.mp4")); !os.IsNotExist(err) {
					t.Fatalf("invalid media was published: %v", err)
				}
			}
			if test.completeErr != nil {
				if _, err := os.Stat(filepath.Join(output, "film.mp4.recovery.json")); err != nil {
					t.Fatal("lost recovery receipt", err)
				}
			}
			if _, err := os.Stat(input); err != nil {
				t.Fatal("lost original source", err)
			}
		})
	}
}
func TestRecordingFallbackIgnoresNonVideoAndRefusesAmbiguity(t *testing.T) {
	id := uuid.New()
	root := t.TempDir()
	directory := filepath.Join(root, id.String(), "artifacts", "videos")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000-not-video.txt", "recording.webm"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("source"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewMockRepository()
	repo.AddExecution(&database.ExecutionIndex{ID: id, ResultPath: "result.json"})
	h := &Handler{repo: repo, recordingsRoot: root}
	video, err := h.loadRecordedVideo(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(video.Path) != "recording.webm" || len(video.SHA256) != 64 || video.SizeBytes == nil {
		t.Fatalf("untruthful source %+v", video)
	}
	if err = os.WriteFile(filepath.Join(directory, "second.webm"), []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = h.loadRecordedVideo(t.Context(), id); err == nil {
		t.Fatal("silently chose among multiple recordings")
	}
}

func TestExportPresentationPreservesExplicitStylesAndWatermarkPolicy(t *testing.T) {
	repo := NewMockRepository()
	if err := repo.SetSetting(t.Context(), replayConfigSettingsKey, `{"style":{"deviceFrameTheme":"slim","browserScale":0.8,"presentation":{"showDeviceFrame":true}}}`); err != nil {
		t.Fatal(err)
	}
	h := &Handler{repo: repo, entitlementService: &entitlement.Service{}}
	for _, required := range []bool{true, false} {
		features := []string{}
		if !required {
			features = []string{entitlement.FeatureWatermarkFree}
		}
		ctx := entitlement.WithEntitlement(t.Context(), &entitlement.Entitlement{Status: entitlement.StatusActive, Features: features})
		incoming := &exportsv1.ReplaySpec{Decor: &exportsv1.ReplayDecor{ShowDeviceFrame: proto.Bool(false), BrowserScale: proto.Float64(.9)}}
		spec := proto.Clone(incoming).(*exportsv1.ReplaySpec)
		if err := h.resolveExportPresentation(ctx, spec, hexport.Request{MovieSpec: incoming}); err != nil {
			t.Fatal(err)
		}
		if spec.GetDecor().GetShowDeviceFrame() || spec.GetDecor().GetBrowserScale() != .9 {
			t.Fatal("saved styles overrode explicit presentation")
		}
		if spec.GetWatermark().GetEnabled() != required {
			t.Fatalf("watermark policy required=%v spec=%+v", required, spec.GetWatermark())
		}
	}
}

func TestExportAdmissionRejectsUnsupportedAndIncompleteBinaryRequests(t *testing.T) {
	h := &Handler{log: logrus.New()}
	for _, body := range []string{`{"format":"avi"}`, `{"format":"mp4"}`, `{"format":"webm","stylization":"stylized","output_dir":"unused"}`} {
		router := chi.NewRouter()
		router.Post("/executions/{id}/export", h.PostExecutionExport)
		request := httptest.NewRequest(http.MethodPost, "/executions/"+uuid.NewString()+"/export", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("admitted invalid request %s: %d %s", body, response.Code, response.Body.String())
		}
	}
}
