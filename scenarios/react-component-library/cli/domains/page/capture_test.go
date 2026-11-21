package page

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

// fakeBAS serves the ux-capture HTTP surface. Dark captures return the same
// pixels as light unless darkDiffers; blankPhone makes the phone view fail L0
// the way BAS reports a flat image.
func fakeBAS(t *testing.T, darkDiffers, blankPhone bool) *httptest.Server {
	t.Helper()
	bundles := map[string]map[string][]byte{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ux-capture/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/v1/ux-capture/bundles/"), "/artifacts/", 2)
			body, ok := bundles[parts[0]][parts[1]]
			if !ok {
				http.NotFound(w, r)
				return
			}
			switch filepath.Ext(parts[1]) {
			case ".png":
				w.Header().Set("Content-Type", "image/png")
			case ".html":
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			case ".json":
				w.Header().Set("Content-Type", "application/json")
			}
			_, _ = w.Write(body)
			return
		}
		var request map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		width, height, theme := int(request["width"].(float64)), int(request["height"].(float64)), request["theme"].(string)
		id := "00000000-0000-4000-8000-00000000000" + string(rune('0'+len(bundles)))
		pixels := []byte("png-" + strings.Repeat("x", width/100))
		if darkDiffers && theme == "dark" {
			pixels = append(pixels, 'd')
		}
		l0 := map[string]string{"status": "present"}
		artifacts := []map[string]any{}
		files := map[string][]byte{
			"page/frozen.html":     []byte(`<html><body><nav data-ux-node="n1">Home</nav></body></html>`),
			"outline/outline.json": []byte(`{"actionCount":1,"nodes":[{"id":"n2","kind":"action","tag":"a","name":"Home","actionType":"navigate","rect":{"x":0,"y":800,"width":48,"height":44}},{"id":"n1","kind":"landmark","tag":"nav","text":"Home","rect":{"x":0,"y":790,"width":390,"height":54}}]}`),
		}
		if blankPhone && width == 390 {
			l0 = map[string]string{"status": "absent", "reason": "Pixels were one flat colour"}
		} else {
			files["pixels/p.png"] = pixels
		}
		kinds := map[string]string{"page/frozen.html": "frozen-page", "outline/outline.json": "outline", "pixels/p.png": "pixels"}
		for path := range files {
			digest := sha256.Sum256(files[path])
			artifacts = append(artifacts, map[string]any{"kind": kinds[path], "path": path, "size_bytes": len(files[path]), "sha256": hex.EncodeToString(digest[:])})
		}
		bundles[id] = files
		w.WriteHeader(http.StatusCreated)
		scenario, _ := request["scenario"].(string)
		route, _ := request["route"].(string)
		if route == "" {
			route = "/"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"manifest": map[string]any{
			"schema_version": "ux-capture.v1", "id": id,
			"source":    map[string]any{"kind": "scenario", "scenario": scenario, "route": redactRouteQuery(route)},
			"session":   map[string]any{"id": "00000000-0000-4000-8000-000000000100"},
			"viewport":  map[string]any{"width": width, "height": height},
			"theme":     theme,
			"privacy":   map[string]any{"redaction_status": "complete", "external": false, "ephemeral": false},
			"artifacts": artifacts,
			"lenses":    map[string]any{"L0": l0, "L1": map[string]string{"status": "present"}, "L2": map[string]string{"status": "present"}, "L3": map[string]string{"status": "absent", "reason": "fixture"}, "L4": map[string]string{"status": "absent", "reason": "fixture"}, "L5": map[string]string{"status": "absent", "reason": "fixture"}, "L6": map[string]string{"status": "absent", "reason": "fixture"}},
		}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func runCapture(t *testing.T, server *httptest.Server, args ...string) (string, error) {
	t.Helper()
	previousPorts := scenarioPorts
	scenarioPorts = func(context.Context, string) (map[string]int, string, error) {
		return map[string]int{"UI_PORT": 1}, "2026-11-15T00:00:00Z", nil
	}
	t.Cleanup(func() { scenarioPorts = previousPorts })
	h := captureHandlers{bas: func(context.Context) (basClient, error) {
		return basClient{base: server.URL, http: server.Client()}, nil
	}}
	out := filepath.Join(t.TempDir(), "bundle")
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scenario"}, {Name: "route"}, {Name: "step", Repeated: true}, {Name: "wait-selector"}, {Name: "url"}, {Name: "file"}, {Name: "views"}, {Name: "themes"}, {Name: "out"}}}
	rc, err := cliapp.NewTestRunContextFromArgs(schema, append(args, "--out", out), nil, &bytes.Buffer{}, &bytes.Buffer{})
	require.NoError(t, err)
	return out, h.capture(rc)
}

func TestCaptureWritesOneBundleWithEveryViewAndAnHonestThemeReport(t *testing.T) {
	out, err := runCapture(t, fakeBAS(t, false, false), "--scenario", "image-tools")
	require.NoError(t, err)
	var set captureSet
	raw, err := os.ReadFile(filepath.Join(out, "capture.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &set))
	require.Equal(t, captureSetSchema, set.Schema)
	require.True(t, strings.HasPrefix(set.ID, "uxc-image-tools-"))
	require.Equal(t, "2026-11-15T00:00:00Z", set.Build["served_since"])
	require.Len(t, set.Views, 4)
	for _, name := range []string{"phone-light", "phone-dark", "desktop-light", "desktop-dark"} {
		require.FileExists(t, filepath.Join(out, "pixels", name+".png"))
		require.FileExists(t, filepath.Join(out, "page", name+".html"))
	}
	require.NotNil(t, set.Views[1].ThemeFollowed)
	require.False(t, *set.Views[1].ThemeFollowed, "identical dark pixels must be reported as an ignored theme")
	readme, _ := os.ReadFile(filepath.Join(out, "README.md"))
	require.Contains(t, string(readme), "ignores the system theme")
	require.Contains(t, string(readme), "cp page/phone-light.html proposal-light.html", "next steps name a page the bundle holds")
	require.Contains(t, string(readme), "page render proposal-light.html --views phone --themes light")
	outline, _ := os.ReadFile(filepath.Join(out, "outline.md"))
	require.Contains(t, string(outline), "`n1` landmark <nav>")
	require.Less(t, strings.Index(string(outline), "`n1`"), strings.Index(string(outline), "`n2`"), "outline reads top to bottom")
}

func TestCaptureFailsWhenAViewHasNoUsablePixels(t *testing.T) {
	out, err := runCapture(t, fakeBAS(t, true, true), "--scenario", "image-tools")
	require.Error(t, err)
	require.Contains(t, err.Error(), "phone-light: no usable pixels")
	readme, _ := os.ReadFile(filepath.Join(out, "README.md"))
	require.Contains(t, string(readme), "**Incomplete.**")
	require.NoFileExists(t, filepath.Join(out, "pixels", "phone-light.png"))
	require.FileExists(t, filepath.Join(out, "pixels", "desktop-dark.png"))
}

func TestParseViewsAcceptsPresetsAndSizes(t *testing.T) {
	views, err := parseViews("phone, 1440x900")
	require.NoError(t, err)
	require.Equal(t, []view{{Name: "phone", Width: 390, Height: 844}, {Name: "1440x900", Width: 1440, Height: 900}}, views)
	_, err = parseViews("watch")
	require.Error(t, err)
}

func TestCaptureManifestRejectsCrossScenarioIdentity(t *testing.T) {
	manifest := `{"manifest":{"schema_version":"ux-capture.v1","id":"00000000-0000-4000-8000-000000000001","source":{"kind":"scenario","scenario":"other-scenario","route":"/"},"session":{"id":"00000000-0000-4000-8000-000000000002"},"viewport":{"width":390,"height":844},"theme":"light","privacy":{"redaction_status":"complete","external":false,"ephemeral":false},"lenses":{"L0":{"status":"present"},"L1":{"status":"present"},"L2":{"status":"present"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"absent","reason":"fixture"},"L5":{"status":"absent","reason":"fixture"},"L6":{"status":"absent","reason":"fixture"}},"artifacts":[]}}`
	var response basResponse
	require.NoError(t, json.Unmarshal([]byte(manifest), &response))
	err := validateCaptureManifest(response, "scenario", map[string]any{"scenario": "image-tools", "route": "/", "width": 390, "height": 844, "theme": "light"})
	require.ErrorContains(t, err, "different scenario route")
}

func TestCaptureManifestAcceptsBASRedactedScenarioQuery(t *testing.T) {
	manifest := `{"manifest":{"schema_version":"ux-capture.v1","id":"00000000-0000-4000-8000-000000000001","source":{"kind":"scenario","scenario":"image-tools","route":"/harness.html?story=%5BREDACTED%5D&version=%5BREDACTED%5D"},"session":{"id":"00000000-0000-4000-8000-000000000002"},"viewport":{"width":390,"height":844},"theme":"light","privacy":{"redaction_status":"complete","external":false,"ephemeral":false},"lenses":{"L0":{"status":"present"},"L1":{"status":"present"},"L2":{"status":"present"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"absent","reason":"fixture"},"L5":{"status":"absent","reason":"fixture"},"L6":{"status":"absent","reason":"fixture"}},"artifacts":[]}}`
	var response basResponse
	require.NoError(t, json.Unmarshal([]byte(manifest), &response))
	err := validateCaptureManifest(response, "scenario", map[string]any{"scenario": "image-tools", "route": "/harness.html?story=button&version=1", "width": 390, "height": 844, "theme": "light"})
	require.NoError(t, err)
}

func TestArtifactFetchRejectsTamperedBytes(t *testing.T) {
	expected := []byte("declared artifact")
	digest := sha256.Sum256(expected)
	manifest := `{"manifest":{"schema_version":"ux-capture.v1","id":"00000000-0000-4000-8000-000000000001","artifacts":[{"kind":"pixels","path":"pixels/capture.png","size_bytes":` + strconv.Itoa(len(expected)) + `,"sha256":"` + hex.EncodeToString(digest[:]) + `"}]}}`
	var response basResponse
	require.NoError(t, json.Unmarshal([]byte(manifest), &response))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("tampered artifact"))
	}))
	defer server.Close()
	client := basClient{base: server.URL, http: server.Client()}
	_, err := client.artifact(context.Background(), response, "pixels")
	require.ErrorContains(t, err, "digest does not match")
}

func TestBASCaptureHonorsCallerCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	client := basClient{base: server.URL, http: server.Client()}
	done := make(chan error, 1)
	go func() {
		_, err := client.capture(ctx, "scenario", map[string]any{"scenario": "image-tools"})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("BAS capture request did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("BAS request ignored caller cancellation")
	}
	close(release)
}

func TestBASArtifactFetchHonorsCallerCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}))
	defer server.Close()
	var response basResponse
	require.NoError(t, json.Unmarshal([]byte(`{"manifest":{"id":"00000000-0000-4000-8000-000000000001","artifacts":[{"kind":"pixels","path":"pixels/capture.png","size_bytes":1,"sha256":"`+strings.Repeat("0", 64)+`"}]}}`), &response))
	client := basClient{base: server.URL, http: server.Client()}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := client.artifact(ctx, response, "pixels"); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("BAS artifact request did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("BAS artifact request ignored caller cancellation")
	}
	close(release)
}
