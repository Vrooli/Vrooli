package readiness

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestManifestReadinessCarriesBlockersToBothProcessModes(t *testing.T) {
	core := cliapptest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"READINESS_STATE_MISSING","target":"local","blockers":[{"kind":"credential","name":"calendar:jwt","reason":"secret is not configured","remediation":"Provide it in Credentials."}],"degraded":[]}`))
	}))
	handler := ManifestHandlers(core)["ReadinessService.GetReadiness"]
	modes := cliapptest.RunPrimitiveHandlerModes(t, handler, cliapp.ArgSchema{}, nil, core)
	for name, err := range map[string]error{"human": modes.HumanErr, "json": modes.JSONErr} {
		exit, ok := err.(*ExitError)
		if !ok || exit.Code != 2 || !strings.Contains(exit.Error(), "calendar:jwt") {
			t.Fatalf("%s mode error = %#v; want readiness exit 2 with blocker", name, err)
		}
	}
	if !strings.Contains(modes.Human, "Required blockers") || !strings.Contains(modes.Human, "Provide it in Credentials") {
		t.Fatalf("human readiness report omitted actionable blocker: %q", modes.Human)
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(modes.JSON), &document); err != nil {
		t.Fatalf("JSON output invalid: %q: %v", modes.JSON, err)
	}
	if strings.Contains(modes.JSON, "Required blockers") {
		t.Fatalf("human report leaked into JSON output: %q", modes.JSON)
	}
}

func TestManifestReadinessAcceptsReadyAndAcknowledgedDegradedStates(t *testing.T) {
	for name, body := range map[string]string{
		"ready":                 `{"status":"READINESS_STATE_READY","target":"local","blockers":[],"degraded":[]}`,
		"acknowledged degraded": `{"status":"READINESS_STATE_DEGRADED","target":"local","blockers":[],"degraded":[{"kind":"credential","name":"optional:key"}],"degradedDigest":"digest-a","degradedAcknowledged":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			core := cliapptest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			handler := ManifestHandlers(core)["ReadinessService.GetReadiness"]
			modes := cliapptest.RunPrimitiveHandlerModes(t, handler, cliapp.ArgSchema{}, nil, core)
			if modes.HumanErr != nil || modes.JSONErr != nil {
				t.Fatalf("readiness errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
			}
		})
	}
}

func TestManifestReadinessRequiresDegradedAcknowledgement(t *testing.T) {
	core := cliapptest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"READINESS_STATE_DEGRADED","target":"local","blockers":[],"degraded":[{"kind":"host","name":"optional-engine","reason":"not installed","remediation":"Install only if needed."}],"degradedDigest":"digest-b","degradedAcknowledged":false}`))
	}))
	modes := cliapptest.RunPrimitiveHandlerModes(t, ManifestHandlers(core)["ReadinessService.GetReadiness"], cliapp.ArgSchema{}, nil, core)
	for name, err := range map[string]error{"human": modes.HumanErr, "json": modes.JSONErr} {
		exit, ok := err.(*ExitError)
		if !ok || exit.Code != 2 || !strings.Contains(exit.Error(), "optional-engine") {
			t.Fatalf("%s mode error = %#v; want unacknowledged degradation exit 2", name, err)
		}
	}
}

func TestManifestAcknowledgeWithoutDegradedDigestRefusesBeforeMutation(t *testing.T) {
	var mutations int
	core := cliapptest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/GetReadiness") {
			_, _ = w.Write([]byte(`{"status":"READINESS_STATE_READY","target":"local","blockers":[],"degraded":[]}`))
			return
		}
		mutations++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	handler := ManifestHandlers(core)["ReadinessService.AcknowledgeDegradedReadiness"]
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "target"}, {Name: "digest"}}}
	modes := cliapptest.RunPrimitiveHandlerModes(t, handler, schema, nil, core)
	for name, err := range map[string]error{"human": modes.HumanErr, "json": modes.JSONErr} {
		exit, ok := err.(*ExitError)
		if !ok || exit.Code != 2 || exit.Error() != "there is no degraded set to acknowledge" {
			t.Fatalf("%s mode error = %#v; want no-degraded-set exit 2", name, err)
		}
	}
	if mutations != 0 {
		t.Fatalf("acknowledgement mutation calls = %d, want none", mutations)
	}
}
