package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestAggregatesExposeAllSurfaceGroups(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	flat, nested, err := LoadManifestGroups(&cliapp.ScenarioApp{}, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(flat) < 2 {
		t.Fatal("flat command groups are incomplete")
	}
	if len(nested) < 5 {
		t.Fatal("subcommand groups are incomplete")
	}
}

func TestPermissionDecisionListAndReadUseSharedRegistryPrimitives(t *testing.T) {
	data, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name    string `json:"name"`
				Binding struct {
					Kind    string `json:"kind"`
					Service string `json:"service"`
					Method  string `json:"method"`
				} `json:"binding"`
				Governance struct {
					Effect string `json:"effect"`
				} `json:"governance"`
				Architecture struct {
					Primitive string `json:"primitive"`
				} `json:"architecture"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	var permissions *struct {
		Name     string `json:"name"`
		Commands []struct {
			Name    string `json:"name"`
			Binding struct {
				Kind    string `json:"kind"`
				Service string `json:"service"`
				Method  string `json:"method"`
			} `json:"binding"`
			Governance struct {
				Effect string `json:"effect"`
			} `json:"governance"`
			Architecture struct {
				Primitive string `json:"primitive"`
			} `json:"architecture"`
		} `json:"commands"`
	}
	for i := range manifest.Groups {
		if manifest.Groups[i].Name == "permissions" {
			permissions = &manifest.Groups[i]
			break
		}
	}
	if permissions == nil {
		t.Fatal("permissions command group missing")
	}
	for _, want := range []struct{ name, method string }{{"list", "ListPermissions"}, {"read", "ReadPermission"}} {
		found := false
		for _, command := range permissions.Commands {
			if command.Name == want.name {
				found = true
				if command.Binding.Kind != "connect-rpc" || command.Binding.Service != "PermissionRegistryService" || command.Binding.Method != want.method || command.Governance.Effect != "read" || command.Architecture.Primitive != "proto_list" {
					t.Fatalf("permissions %s route diverged from shared read registry: %+v", want.name, command)
				}
			}
		}
		if !found {
			t.Fatalf("permissions %s command missing", want.name)
		}
	}
	_, nested, err := LoadManifestGroups(&cliapp.ScenarioApp{}, data)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range nested {
		if group.Name == "permissions" {
			return
		}
	}
	t.Fatal("permissions commands did not load into the shared CLI surface")
}

func TestManifestReadinessAcknowledgeFetchesCurrentDigestWhenOmitted(t *testing.T) {
	data, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	var posted string
	core := cliapptest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/GetReadiness") {
			calls = append(calls, "readiness")
			_, _ = w.Write([]byte(`{"status":"READINESS_STATE_DEGRADED","target":"local","blockers":[],"degraded":[{"kind":"credential","name":"api-key"}],"degradedDigest":"current-digest","degradedAcknowledged":false}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/AcknowledgeDegradedReadiness") {
			t.Errorf("unexpected API path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls = append(calls, "acknowledge")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read acknowledgement body: %v", err)
		}
		posted = string(body)
		if !strings.Contains(posted, `"readinessDigest":"current-digest"`) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"message":"readiness_digest required"}`)
			return
		}
		_, _ = w.Write([]byte(`{"status":"acknowledged","readinessDigest":"current-digest"}`))
	}))
	flat, _, err := LoadManifestGroups(core, data)
	if err != nil {
		t.Fatal(err)
	}
	var command *cliapp.Command
	for groupIndex := range flat {
		for commandIndex := range flat[groupIndex].Commands {
			candidate := &flat[groupIndex].Commands[commandIndex]
			if flat[groupIndex].Title == "readiness" && candidate.Name == "acknowledge-degraded" {
				command = candidate
			}
		}
	}
	if command == nil || command.RunCtx == nil {
		t.Fatal("manifest readiness acknowledgement command is not executable")
	}
	cliapptest.AssertPrimitiveEvidence(t, *command, cliapp.PrimitiveProtoMutation)
	var stdout, stderr bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(command.Args, nil, core, &stdout, &stderr)
	if err != nil {
		t.Fatalf("build manifest command context: %v", err)
	}
	if err := command.RunCtx(ctx); err != nil {
		t.Fatalf("manifest acknowledgement: %v", err)
	}
	if len(calls) != 2 || calls[0] != "readiness" || calls[1] != "acknowledge" {
		t.Fatalf("API call sequence = %v, want readiness then acknowledgement", calls)
	}
	if !strings.Contains(posted, `"target":"local"`) || !strings.Contains(posted, `"readinessDigest":"current-digest"`) {
		t.Fatalf("acknowledgement payload = %s", posted)
	}
}
