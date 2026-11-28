package main

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	"github.com/vrooli/cli-core/cliutil"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"swarm-manager/cli/domains"
	"swarm-manager/cli/internal/support"
	"testing"

	"connectrpc.com/connect"
)

type developmentFixtureTransport func(*http.Request) (*http.Response, error)

func (f developmentFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const developmentReferenceJSON = `{"effortId":"effort-1","revision":"3","authorityDigest":"authority","contractDigest":"contract","commissionSubjectDigest":"subject"}`

func developmentFixture(t *testing.T, handler http.Handler) *cliapp.ScenarioApp {
	t.Helper()
	t.Setenv("DEVELOPMENT_TEST_CONFIG_DIR", t.TempDir())
	t.Setenv("DEVELOPMENT_TEST_TOKEN", "")
	t.Setenv(cliutil.EnvIdentityToken, "")
	old := http.DefaultTransport
	http.DefaultTransport = developmentFixtureTransport(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Result(), nil
	})
	t.Cleanup(func() { http.DefaultTransport = old })
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "development-fixture", DefaultAPIBase: "http://fixture.invalid", ConfigDirEnvVars: []string{"DEVELOPMENT_TEST_CONFIG_DIR"}, TokenEnvVars: []string{"DEVELOPMENT_TEST_TOKEN"}, AllowAnonymous: true})
	if err != nil {
		t.Fatal(err)
	}
	return core
}

// Fixture tokens below model transport policy only; no cryptographic identity
// verification or live installed reader/owner qualification is claimed.
func TestDevelopmentPrimitivesExactPayloadAndHeldRefusal(t *testing.T) {
	ref := developmentReferenceJSON
	tests := []struct {
		group, name, path, data string
		argv                    []string
		response                proto.Message
		kind                    cliapp.PrimitiveClass
	}{
		{"development", "get", "/vrooli.swarm_manager.v1.api.DevelopmentService/GetDevelopment", "", []string{"--effort-id", "effort-1"}, &api.GetDevelopmentResponse{Development: &api.DevelopmentView{LaunchBlockers: []string{"commission held"}}}, cliapp.PrimitiveProtoList},
		{"development", "artifact", "/vrooli.swarm_manager.v1.api.DevelopmentService/GetDevelopmentArtifact", `{"reference":` + ref + `,"artifactId":"goal-message"}`, nil, &api.GetDevelopmentArtifactResponse{Content: []byte("exact\nbytes\x00")}, cliapp.PrimitiveProtoList},
		{"development", "approve", "/vrooli.swarm_manager.v1.api.DevelopmentService/ApproveDevelopment", `{"reference":` + ref + `,"expectedGeneration":"0","requestId":"request-approve"}`, nil, &api.DevelopmentDecisionResponse{Action: "approve", ReceiptDigest: "receipt"}, cliapp.PrimitiveProtoMutation},
		{"development", "revoke", "/vrooli.swarm_manager.v1.api.DevelopmentService/RevokeDevelopment", `{"reference":` + ref + `,"expectedGeneration":"9","requestId":"request-revoke"}`, nil, &api.DevelopmentDecisionResponse{Action: "revoke"}, cliapp.PrimitiveProtoMutation},
		{"development", "accept", "/vrooli.swarm_manager.v1.api.DevelopmentService/AcceptDevelopment", `{"reference":` + ref + `,"expectedGeneration":"9","requestId":"request-accept","testedProductDigest":"product","expectedDispositionVersion":"0"}`, nil, &api.DevelopmentDecisionResponse{Action: "accept"}, cliapp.PrimitiveProtoMutation},
		{"transitions", "preview-development", "/vrooli.swarm_manager.v1.api.TransitionService/PreviewDevelopment", `{"reference":` + ref + `,"proposal":{"workShape":"bounded_task","title":"Review only","scopeAllow":["source/**"],"sourceRelativePaths":["docs/contract.md"]}}`, nil, &api.PreviewDevelopmentResponse{LaunchBlockers: []string{"not approved"}}, cliapp.PrimitiveProtoList},
	}
	for _, tc := range tests {
		t.Run(tc.group+"/"+tc.name, func(t *testing.T) {
			var body []byte
			requests := 0
			accepted := 0
			offeredRun := ""
			offeredBearer := ""
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				offeredRun = r.Header.Get(cliutil.HeaderAgentIdentityToken)
				offeredBearer = r.Header.Get("Authorization")
				if r.URL.Path != tc.path {
					t.Errorf("path=%s", r.URL.Path)
				}
				read := tc.name == "get" || tc.name == "artifact" || tc.name == "preview-development"
				humanOnly := offeredBearer == "Bearer disposable-owner" && offeredRun == ""
				installedRunReaderOnly := read && offeredRun == "disposable-parent" && offeredBearer == ""
				if !humanOnly && !installedRunReaderOnly {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, `{"code":"unauthenticated","message":"held_acceptance: verified human caller required"}`)
					return
				}
				accepted++
				var err error
				body, err = ioReadBounded(r)
				if err != nil {
					t.Error(err)
				}
				// Validate exact binary bytes from the generated Connect client.
				var msg proto.Message
				switch tc.name {
				case "get":
					msg = &api.GetDevelopmentRequest{}
				case "artifact":
					msg = &api.GetDevelopmentArtifactRequest{}
				case "approve":
					msg = &api.ApproveDevelopmentRequest{}
				case "revoke":
					msg = &api.RevokeDevelopmentRequest{}
				case "accept":
					msg = &api.AcceptDevelopmentRequest{}
				default:
					msg = &api.PreviewDevelopmentRequest{}
				}
				if err := proto.Unmarshal(body, msg); err != nil {
					t.Fatal(err)
				}
				expected := proto.Clone(msg)
				proto.Reset(expected)
				payload := tc.data
				if tc.name == "get" {
					payload = `{"effortId":"effort-1"}`
				}
				if err := protojson.Unmarshal([]byte(payload), expected); err != nil {
					t.Fatal(err)
				}
				if !proto.Equal(msg, expected) {
					t.Errorf("request not exact: %s", msg)
				}
				raw, err := proto.Marshal(tc.response)
				if err != nil {
					t.Fatal(err)
				}
				w.Header().Set("Content-Type", "application/proto")
				w.WriteHeader(http.StatusOK)
				w.Write(raw)
			})
			core := developmentFixture(t, handler)
			cmd := cliapptest.FindCommand(t, domains.SubcommandGroups(support.Dependencies{}), tc.group, tc.name)
			cliapptest.AssertPrimitiveEvidence(t, cmd, tc.kind)
			argv := append([]string{}, tc.argv...)
			if tc.data != "" {
				argv = append(argv, "--data", tc.data)
			}
			argv = append(argv, "--json")
			out, err := cliapptest.RunCommand(t, cmd, core, argv...)
			if err == nil || !strings.Contains(err.Error(), "held_acceptance") {
				t.Fatalf("absent proof err=%v output=%s", err, out)
			}
			if accepted != 0 || requests != 1 {
				t.Fatalf("anonymous effects accepted=%d requests=%d", accepted, requests)
			}
			t.Setenv(cliutil.EnvIdentityToken, "disposable-parent")
			out, err = cliapptest.RunCommand(t, cmd, core, argv...)
			read := tc.name == "get" || tc.name == "artifact" || tc.name == "preview-development"
			runAccepted := 0
			if read {
				runAccepted = 1
				if err != nil || out == "" {
					t.Fatalf("installed run-reader transport refused: %v", err)
				}
				decoded := proto.Clone(tc.response)
				proto.Reset(decoded)
				if err := protojson.Unmarshal([]byte(out), decoded); err != nil || !proto.Equal(decoded, tc.response) {
					t.Fatalf("run-reader typed output not exact: %v", err)
				}
			} else if connect.CodeOf(err) != connect.CodeUnauthenticated || out != "" {
				t.Fatalf("run-only write refusal: %v", err)
			}
			if accepted != runAccepted || requests != 2 || offeredRun != "disposable-parent" || offeredBearer != "" {
				t.Fatal("run-only transport channel/count mismatch")
			}
			t.Setenv("DEVELOPMENT_TEST_TOKEN", "invalid-owner")
			out, err = cliapptest.RunCommand(t, cmd, core, argv...)
			if connect.CodeOf(err) != connect.CodeUnauthenticated || accepted != runAccepted || requests != 3 || out != "" {
				t.Fatalf("invalid offered human proof changed effects: %v", err)
			}
			t.Setenv("DEVELOPMENT_TEST_TOKEN", "disposable-owner")
			out, err = cliapptest.RunCommand(t, cmd, core, argv...)
			if connect.CodeOf(err) != connect.CodeUnauthenticated || out != "" || accepted != runAccepted || requests != 4 || offeredBearer != "Bearer disposable-owner" || offeredRun != "disposable-parent" {
				t.Fatalf("mixed proof did not preserve refusal/no alternate path: error=%v accepted=%d requests=%d output=%q", err, accepted, requests, out)
			}
			// Select exactly one offered channel; the CLI must not silently strip proof.
			t.Setenv(cliutil.EnvIdentityToken, "")
			out, err = cliapptest.RunCommand(t, cmd, core, argv...)
			if err != nil {
				t.Fatal(err)
			}
			if accepted != runAccepted+1 || requests != 5 || offeredBearer != "Bearer disposable-owner" || offeredRun != "" {
				t.Fatal("human-only JSON channel/count mismatch")
			}
			decoded := proto.Clone(tc.response)
			proto.Reset(decoded)
			if err := protojson.Unmarshal([]byte(out), decoded); err != nil {
				t.Fatalf("output decode: %v %s", err, out)
			}
			if !proto.Equal(decoded, tc.response) {
				t.Fatal("typed output not exact")
			}
			// Human-only plaintext uses the same typed operation, no alternate route.
			t.Setenv(cliutil.EnvIdentityToken, "")
			plainArgs := argv[:len(argv)-1]
			out, err = cliapptest.RunCommand(t, cmd, core, plainArgs...)
			if err != nil || accepted != runAccepted+2 || requests != 6 || offeredRun != "" || offeredBearer != "Bearer disposable-owner" || out == "" {
				t.Fatalf("human-only plaintext compatibility: %v", err)
			}

		})
	}
}

func ioReadBounded(r *http.Request) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, 256*1024))
}

func TestDevelopmentPrimitivesInvalidInputHasNoTransportEffects(t *testing.T) {
	tests := []struct {
		group, name string
		args        []string
	}{
		{"development", "get", nil}, {"development", "get", []string{"--effort-id", "  "}},
		{"development", "artifact", []string{"--data", `{"reference":{"effortId":"effort-1"}}`}},
		{"development", "approve", []string{"--data", `{"reference":{"effortId":"effort-1"},"requestId":"r"}`}},
		{"development", "approve", []string{"--data", `{"reference":{"effortId":"effort-1"},"expectedGeneration":"0"}`}},
		{"development", "revoke", []string{"--data", `{"reference":{"effortId":"effort-1"},"expectedGeneration":null,"requestId":"r"}`}},
		{"development", "accept", []string{"--data", `{"reference":{"effortId":"effort-1"},"expectedGeneration":"0","requestId":"r","testedProductDigest":"product"}`}},
		{"development", "accept", []string{"--data", `{"reference":{"effortId":"effort-1"},"expectedGeneration":"0","requestId":"r","expectedDispositionVersion":"0"}`}},
		{"transitions", "preview-development", []string{"--data", `{"reference":{"effortId":"effort-1"}}`}},
	}
	for _, bad := range []string{`{`, `{"unknownActor":"human"}`, `{"reference":{"effortId":"effort-1"},"expectedGeneration":"0","requestId":"r","actor":"operator"}`, strings.Repeat(" ", 128*1024+1)} {
		tests = append(tests, struct {
			group, name string
			args        []string
		}{"development", "approve", []string{"--data", bad}})
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("%s-%s-%d", tc.group, tc.name, i), func(t *testing.T) {
			calls := 0
			core := developmentFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) }))
			cmd := cliapptest.FindCommand(t, domains.SubcommandGroups(support.Dependencies{}), tc.group, tc.name)
			out, err := cliapptest.RunCommand(t, cmd, core, tc.args...)
			if err == nil {
				t.Fatal("invalid input accepted")
			}
			if calls != 0 || out != "" {
				t.Fatalf("effects calls=%d output=%q", calls, out)
			}
		})
	}
}

// This is refusal propagation at a synthetic transport boundary, not owner
// identity/fence qualification or evidence that actual acceptance is available.
func TestDevelopmentAcceptOfferedHumanStillPreservesOwnerHeldRefusal(t *testing.T) {
	requests := 0
	core := developmentFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/vrooli.swarm_manager.v1.api.DevelopmentService/AcceptDevelopment" || r.Header.Get("Authorization") != "Bearer disposable-owner" {
			t.Error("request left expected offered caller/path")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":"failed_precondition","message":"held_acceptance: owner disposition fence unavailable"}`)
	}))
	t.Setenv("DEVELOPMENT_TEST_TOKEN", "disposable-owner")
	cmd := cliapptest.FindCommand(t, domains.SubcommandGroups(support.Dependencies{}), "development", "accept")
	data := `{"reference":` + developmentReferenceJSON + `,"expectedGeneration":"9","requestId":"r","testedProductDigest":"product","expectedDispositionVersion":"0"}`
	out, err := cliapptest.RunCommand(t, cmd, core, "--data", data, "--json")
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || !strings.Contains(err.Error(), "held_acceptance: owner disposition fence unavailable") || out != "" || requests != 1 {
		t.Fatalf("held owner refusal was changed or retried: error=%v requests=%d output=%q", err, requests, out)
	}
}
