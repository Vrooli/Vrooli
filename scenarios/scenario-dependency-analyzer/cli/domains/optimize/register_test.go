package optimize

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestSelectedOptimizationNeverFallsBackToBroadMutation(t *testing.T) {
	for _, supported := range []bool{false, true} {
		t.Run(map[bool]string{false: "old-server", true: "supported"}[supported], func(t *testing.T) {
			selected, broad := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/optimize-scenarios") {
					selected++
					if !supported {
						http.NotFound(w, r)
						return
					}
					var body struct {
						Scenario      string   `json:"scenario"`
						OnlyScenarios []string `json:"only_scenarios"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Scenario != "architecture-cartographer" || len(body.OnlyScenarios) != 1 || body.OnlyScenarios[0] != "go-code-graph" {
						t.Errorf("selection lost: %+v %v", body, err)
					}
					_, _ = w.Write([]byte(`{"results":{}}`))
				} else if strings.HasSuffix(r.URL.Path, "/optimize") {
					broad++
				}
			}))
			defer server.Close()
			t.Setenv("SDA_OPTIMIZE_TEST_CONFIG", t.TempDir())
			app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "sda-optimize-fixture", DefaultAPIBase: server.URL, ConfigDirEnvVars: []string{"SDA_OPTIMIZE_TEST_CONFIG"}, AllowAnonymous: true})
			if err != nil {
				t.Fatal(err)
			}
			err = run(app, []string{"architecture-cartographer", "--apply", "--only-scenarios", "go-code-graph"})
			if (err == nil) != supported || selected != 1 || broad != 0 {
				t.Fatalf("err=%v selected=%d broad=%d", err, selected, broad)
			}
			err = run(app, []string{"architecture-cartographer", "--apply", "--only-scenarios", ""})
			if err == nil || broad != 0 {
				t.Fatal("empty selector admitted broad mutation")
			}
		})
	}
}
