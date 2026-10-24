//go:build integration

package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Exercises the actual isolated UI build and continuous source through the native
// driver, never a protected product workspace or a fake bas:render listener.
func TestPlaywrightCaptureIntegration(t *testing.T) {
	dist := os.Getenv("BAS_REPLAY_TEST_DIST")
	if dist == "" || os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
		t.Skip("requires isolated BAS UI build and an admitted driver")
	}
	dir := t.TempDir()
	video := filepath.Join(dir, "source.mp4")
	// Solid red then blue makes first/end correctness inspectable in actual pixels.
	command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=red:s=320x240:d=4.2:r=25", "-f", "lavfi", "-i", "color=blue:s=320x240:d=4.2:r=25", "-f", "lavfi", "-i", "sine=frequency=440:duration=8.4", "-filter_complex", "[0:v][1:v]concat=n=2:v=1:a=0[out]", "-map", "[out]", "-map", "2:a", "-c:a", "aac", "-c:v", "libx264", "-threads", "2", "-pix_fmt", "yuv420p", video)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v: %s", err, output)
	}
	compatible, cleanup, err := BrowserCompatibleRecording(context.Background(), video, filepath.Join(dir, "derived"))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(dist)))
	mux.HandleFunc("/source.webm", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, compatible) })
	server := httptest.NewServer(mux)
	defer server.Close()
	original := &ReplayMovieSpec{Decor: &exportsv1.ReplayDecor{ChromeTheme: "chromium", BackgroundTheme: "charcoal", ShowDesktop: proto.Bool(true), ShowBrowserFrame: proto.Bool(true), BrowserScale: proto.Float64(.85)}, Presentation: &exportsv1.ReplayPresentation{Canvas: &exportsv1.ReplayDimensions{Width: 640, Height: 480}}, Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 40}}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	spec, err := PrepareRecordedVideoSpec(ctx, original, compatible, server.URL+"/source.webm")
	if err != nil {
		t.Fatal(err)
	}
	client := newPlaywrightCaptureClient(server.URL + "/src/export/composer.html")
	result, err := client.Capture(ctx, spec, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Frames) != 211 || result.Width != 640 || result.Height != 480 {
		t.Fatalf("wrong coverage/raster: %+v", result)
	}
	for _, point := range []struct {
		index int
		red   bool
	}{{0, true}, {len(result.Frames) - 1, false}} {
		raw, err := base64.StdEncoding.DecodeString(result.Frames[point.index].Data)
		if err != nil {
			t.Fatal(err)
		}
		frame, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		r, _, b, _ := frame.At(320, 260).RGBA()
		if point.red && r <= b || !point.red && b <= r {
			t.Fatalf("source frame %d did not reach expected color: red=%d blue=%d", point.index, r, b)
		}
	}
	if out := os.Getenv("BAS_REPLAY_TEST_EVIDENCE"); out != "" {
		if err := os.Mkdir(out, 0o755); err != nil {
			t.Fatal(err)
		}
		r := newTestRenderer(t)
		r.captureClient = &stubCaptureClient{response: result}
		media, err := r.Render(ctx, spec, RenderFormatMP4, "native.mp4")
		if err != nil {
			t.Fatal(err)
		}
		defer media.Cleanup()
		if err := FinishRecordedVideo(ctx, media, video, spec.Summary.TotalDurationMs, "mp4"); err != nil {
			t.Fatal(err)
		}
		encoded, err := os.ReadFile(media.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "native.mp4"), encoded, 0o644); err != nil {
			t.Fatal(err)
		}
		for i, frame := range result.Frames {
			raw, _ := base64.StdEncoding.DecodeString(frame.Data)
			if err := os.WriteFile(filepath.Join(out, fmt.Sprintf("native-%02d.png", i)), raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// One normal executor session/page first recovers from a published parse error,
// then adopts two real generated same-ID requests with visibly different styles.
// No API deployment, shared database, driver restart or video encoder is needed.
func TestPlaywrightSamePageReplayAdoptionIntegration(t *testing.T) {
	dist := os.Getenv("BAS_REPLAY_TEST_DIST")
	if dist == "" || os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
		t.Skip("requires isolated current BAS UI build and an admitted driver")
	}
	server := httptest.NewServer(http.FileServer(http.Dir(dist)))
	defer server.Close()
	var imageBytes bytes.Buffer
	asset := image.NewRGBA(image.Rect(0, 0, 64, 48))
	draw.Draw(asset, asset.Bounds(), &image.Uniform{C: color.RGBA{R: 128, G: 128, B: 128, A: 255}}, image.Point{}, draw.Src)
	if err := png.Encode(&imageBytes, asset); err != nil {
		t.Fatal(err)
	}
	makeSpec := func(backgroundColor string) *ReplayMovieSpec {
		background, err := structpb.NewStruct(map[string]any{"type": "gradient", "value": map[string]any{
			"type": "linear", "angle": 90, "stops": []any{map[string]any{"color": backgroundColor, "position": 0}, map[string]any{"color": backgroundColor, "position": 100}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		return &ReplayMovieSpec{
			Execution:    &exportsv1.ReplayExecutionMetadata{ExecutionId: "same-page-style-qualification"},
			Decor:        &exportsv1.ReplayDecor{Background: background, ShowDesktop: proto.Bool(true), ShowBrowserFrame: proto.Bool(false), BrowserScale: proto.Float64(.5)},
			Presentation: &exportsv1.ReplayPresentation{Canvas: &exportsv1.ReplayDimensions{Width: 320, Height: 240}},
			Summary:      &exportsv1.ReplaySummary{TotalDurationMs: 100},
			Playback:     &exportsv1.ReplayPlayback{FrameIntervalMs: 100},
			Assets:       []*exportsv1.ReplayAsset{{Id: "screen", Type: "image/png", Source: "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes.Bytes()), Width: 64, Height: 48}},
			Frames:       []*exportsv1.ReplayFrame{{DurationMs: 100, ScreenshotAssetId: "screen", Viewport: &exportsv1.ReplayDimensions{Width: 64, Height: 48}}},
		}
	}
	first, err := buildPlaywrightCaptureInstructions(server.URL+"/src/export/composer.html", makeSpec("#ff0000"), 100)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildPlaywrightCaptureInstructions(server.URL+"/src/export/composer.html", makeSpec("#0000ff"), 100)
	if err != nil {
		t.Fatal(err)
	}
	seedError, err := autocompiler.BuildActionDefinition("evaluate", map[string]any{"expression": `(async()=>{
      const deadline=Date.now()+5000;
      while(!window.basExport || window.basExport.error==='initializing'){if(Date.now()>deadline)throw Error('Composer unavailable before error fixture');await new Promise(r=>setTimeout(r,25));}
      window.postMessage({type:'bas:spec:set',requestToken:'prior-invalid-attempt',spec:{invalidProtoField:true}},window.location.origin);
      while(window.basExport.getMetadata().requestError?.requestToken!=='prior-invalid-attempt'){if(Date.now()>deadline)throw Error('Prior parse error was not published');await new Promise(r=>setTimeout(r,25));}
      if(window.basExport.ready)throw Error('Invalid spec became ready');return {priorError:true};})()`})
	if err != nil {
		t.Fatal(err)
	}
	instructions := append([]autocontracts.CompiledInstruction{}, first[:2]...)
	instructions = append(instructions, autocontracts.CompiledInstruction{NodeID: "seed-prior-error", Action: seedError})
	instructions = append(instructions, first[2:]...)
	// Keep the existing page/session: only inject, seek and capture the successor.
	instructions = append(instructions, second[2:]...)
	for index := range instructions {
		instructions[index].Index = index
		instructions[index].NodeID = fmt.Sprintf("adoption-%02d-%s", index, instructions[index].NodeID)
	}
	factory, err := autoengine.DefaultFactory(nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := &inMemoryCaptureRecorder{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	artifacts := config.ArtifactCollectionSettings{ScreenshotPolicy: basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_NEVER}
	err = autoexecutor.NewSimpleExecutor(nil).Execute(ctx, autoexecutor.Request{
		Plan:       autocontracts.ExecutionPlan{SchemaVersion: autocontracts.ExecutionPlanSchemaVersion, PayloadVersion: autocontracts.PayloadVersion, ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: instructions, CreatedAt: time.Now().UTC(), Metadata: map[string]any{"executionViewport": map[string]any{"width": 320, "height": 240}}},
		EngineName: "playwright", EngineFactory: factory, Recorder: recorder,
		EventSink: autoevents.NewMemorySink(autocontracts.DefaultEventBufferLimits), ArtifactConfig: &artifacts,
		BrowserProfile: &sessionprofile.BrowserProfile{Fingerprint: &sessionprofile.FingerprintSettings{DeviceScaleFactor: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var screenshots [][]byte
	for _, outcome := range recorder.outcomes {
		if strings.Contains(outcome.NodeID, "screenshot-") && outcome.Screenshot != nil {
			screenshots = append(screenshots, outcome.Screenshot.Data)
		}
	}
	if len(screenshots) != 4 {
		t.Fatalf("expected 2 screenshots per same-page style, got %d", len(screenshots))
	}
	evidenceDir := ""
	if out := os.Getenv("BAS_REPLAY_TEST_EVIDENCE"); out != "" {
		evidenceDir = out + "-same-page"
		if err := os.Mkdir(evidenceDir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for index, raw := range screenshots {
		frame, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		if frame.Bounds().Dx() != 320 || frame.Bounds().Dy() != 240 {
			t.Fatalf("unexpected native raster: %v", frame.Bounds())
		}
		r, g, b, _ := frame.At(8, 8).RGBA()
		if index < 2 && (r < 50000 || g > 10000 || b > 10000) || index >= 2 && (b < 50000 || r > 10000 || g > 10000) {
			t.Fatalf("same-ID successor style was stale at screenshot %d: rgb=%d,%d,%d", index, r, g, b)
		}
		if evidenceDir != "" {
			if err := os.WriteFile(filepath.Join(evidenceDir, fmt.Sprintf("adoption-%02d.png", index)), raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestPlaywrightPhoneFrameIntegration(t *testing.T) {
	dist := os.Getenv("BAS_REPLAY_TEST_DIST")
	if dist == "" || os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
		t.Skip("requires isolated BAS UI build and admitted driver")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "mobile.webm")
	if raw, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=green:s=160x320:d=1:r=5", "-c:v", "libvpx-vp9", "-threads", "2", path).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v: %s", err, raw)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(dist)))
	mux.HandleFunc("/mobile.webm", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, path) })
	server := httptest.NewServer(mux)
	defer server.Close()
	style := &ReplayMovieSpec{Decor: &exportsv1.ReplayDecor{ChromeTheme: "minimal", DeviceFrameTheme: "slim", ShowDesktop: proto.Bool(true), ShowBrowserFrame: proto.Bool(false), ShowDeviceFrame: proto.Bool(true), BrowserScale: proto.Float64(.8)}, Presentation: &exportsv1.ReplayPresentation{Canvas: &exportsv1.ReplayDimensions{Width: 400, Height: 700}}, Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 200}}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	spec, err := PrepareRecordedVideoSpec(ctx, style, path, server.URL+"/mobile.webm")
	if err != nil {
		t.Fatal(err)
	}
	result, err := newPlaywrightCaptureClient(server.URL+"/src/export/composer.html").Capture(ctx, spec, 200)
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 400 || result.Height != 700 {
		t.Fatal("phone canvas mismatch")
	}
	if out := os.Getenv("BAS_REPLAY_TEST_EVIDENCE"); out != "" {
		out += "-phone"
		if err := os.Mkdir(out, 0o755); err != nil {
			t.Fatal(err)
		}
		raw, _ := base64.StdEncoding.DecodeString(result.Frames[0].Data)
		if err := os.WriteFile(filepath.Join(out, "phone.png"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
