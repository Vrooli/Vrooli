package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderKitTemplateValidatesAndEscapesDataInItsSingleSlot(t *testing.T) {
	kitDir, err := notificationHubKitDir()
	require.NoError(t, err)
	sample, err := os.ReadFile(kitDir + "/samples/usage-mode.json")
	require.NoError(t, err)
	var data map[string]any
	require.NoError(t, json.Unmarshal(sample, &data))
	data["meta"].(map[string]any)["title"] = "</script><script>alert(1)</script>"

	page, err := RenderKitTemplate("usage-mode", data)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(page, `<script type="application/json" id="hub-data">`))
	require.Contains(t, page, `"title":"\u003c/script\u003e\u003cscript\u003ealert(1)\u003c/script\u003e"`)
	require.NotContains(t, page, `id="hub-data"></script><script>alert(1)`)
}

func TestRenderKitTemplateAcceptsAllPublishedSamples(t *testing.T) {
	kitDir, err := notificationHubKitDir()
	require.NoError(t, err)
	for _, name := range []string{"usage-mode", "daily-briefing", "opportunity-roadmap", "structural-change"} {
		t.Run(name, func(t *testing.T) {
			sample, err := os.ReadFile(filepath.Join(kitDir, "samples", name+".json"))
			require.NoError(t, err)
			var data map[string]any
			require.NoError(t, json.Unmarshal(sample, &data))
			page, err := RenderKitTemplate(name, data)
			require.NoError(t, err)
			require.Contains(t, page, `<script type="application/json" id="hub-data">`)
		})
	}
}

func TestRenderKitTemplateRejectsDataOutsideTheSchema(t *testing.T) {
	_, err := RenderKitTemplate("daily-briefing", map[string]any{"unexpected": true})
	require.ErrorContains(t, err, `does not match "daily-briefing"`)
}

func TestRenderKitTemplateRejectsUnknownTemplateNames(t *testing.T) {
	_, err := RenderKitTemplate("not-published", map[string]any{})
	require.ErrorContains(t, err, `kit template "not-published" is unavailable`)
}
