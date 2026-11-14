package scan

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestSelectiveApplyNeverFallsBackToLegacyRoute(t *testing.T) {
	for _, supported := range []bool{false, true} {
		t.Run(map[bool]string{false: "old-server", true: "supported"}[supported], func(t *testing.T) {
			selectedPosts, legacyPosts := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/scan-selected") {
					selectedPosts++
					if !supported {
						http.NotFound(w, r)
						return
					}
					var body struct {
						OnlyScenarios []string `json:"only_scenarios"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.OnlyScenarios) != 1 || body.OnlyScenarios[0] != "web-console" {
						t.Errorf("selection lost: %+v err=%v", body, err)
					}
					_, _ = w.Write([]byte(`{"applied":true,"apply_summary":{"scenarios_added":["web-console"]}}`))
				} else if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/scan") {
					legacyPosts++ // An old server would apply broadly here.
				}
			}))
			defer server.Close()
			t.Setenv("SDA_SCAN_TEST_CONFIG_DIR", t.TempDir())
			app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "sda-scan-fixture", DefaultAPIBase: server.URL, ConfigDirEnvVars: []string{"SDA_SCAN_TEST_CONFIG_DIR"}, AllowAnonymous: true})
			if err != nil {
				t.Fatal(err)
			}
			err = run(app, []string{"program-runtime", "--apply-scenarios", "--only-scenarios", "web-console"})
			if supported && (err != nil || selectedPosts != 1 || legacyPosts != 0) {
				t.Fatalf("supported server err=%v selected=%d legacy=%d", err, selectedPosts, legacyPosts)
			}
			if !supported && (err == nil || selectedPosts != 1 || legacyPosts != 0) {
				t.Fatalf("old server err=%v selected=%d legacy=%d", err, selectedPosts, legacyPosts)
			}
		})
	}
}
