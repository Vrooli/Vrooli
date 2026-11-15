package page

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			_, _ = w.Write(body)
			return
		}
		var request map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		width, theme := int(request["width"].(float64)), request["theme"].(string)
		id := "00000000-0000-4000-8000-00000000000" + string(rune('0'+len(bundles)))
		pixels := []byte("png-" + strings.Repeat("x", width/100))
		if darkDiffers && theme == "dark" {
			pixels = append(pixels, 'd')
		}
		l0 := map[string]string{"status": "present"}
		artifacts := []map[string]string{}
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
			artifacts = append(artifacts, map[string]string{"kind": kinds[path], "path": path})
		}
		bundles[id] = files
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"bundle_dir": "/x", "manifest": map[string]any{
			"schema_version": "ux-capture.v1", "id": id, "artifacts": artifacts,
			"lenses": map[string]any{"L0": l0, "L1": map[string]string{"status": "present"}},
		}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func runCapture(t *testing.T, server *httptest.Server, args ...string) (string, error) {
	t.Helper()
	scenarioPorts = func(string) (map[string]int, string, error) {
		return map[string]int{"UI_PORT": 1}, "2026-11-15T00:00:00Z", nil
	}
	h := captureHandlers{bas: func() (basClient, error) { return basClient{base: server.URL, http: server.Client()}, nil }}
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
