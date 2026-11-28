package enrichment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	enrichmentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/enrichment"
	enrichmentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/enrichment/enrichment_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

func TestRegisterExposesManifestEnrichmentCommands(t *testing.T) {
	group, err := Register(&cliapp.ScenarioApp{}, readManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	if group.Name != "enrichment" || len(group.Subcommands) != 2 {
		t.Fatalf("registered group = %q with %d commands, want enrichment with enrich and embed", group.Name, len(group.Subcommands))
	}

	want := map[string]map[string]bool{
		"enrich": {"document-hash": true, "text": true, "privacy": false},
		"embed":  {"document-hash": true, "unit-id": true, "text": true, "privacy": false, "role": false},
	}
	for _, command := range group.Subcommands {
		flags, ok := want[command.Name]
		if !ok {
			t.Errorf("unexpected command %q", command.Name)
			continue
		}
		delete(want, command.Name)
		if command.RunCtx == nil || !command.NeedsAPI {
			t.Errorf("%s has no handler or is not marked API-backed", command.Name)
		}
		for flag, required := range flags {
			found := false
			for _, declared := range command.Args.Flags {
				if declared.Name == flag {
					found = true
					if declared.Required != required {
						t.Errorf("%s --%s required=%v, want %v", command.Name, flag, declared.Required, required)
					}
				}
			}
			if !found {
				t.Errorf("%s does not declare --%s", command.Name, flag)
			}
		}
	}
	for missing := range want {
		t.Errorf("manifest command %q was not registered", missing)
	}
}

func TestEnrichCommandSendsInputsAndReportsResult(t *testing.T) {
	var received *enrichmentv1.EnrichRequest
	server := enrichmentServer(t, serviceFuncs{
		enrich: func(_ context.Context, req *connect.Request[enrichmentv1.EnrichRequest]) (*connect.Response[enrichmentv1.EnrichResponse], error) {
			received = req.Msg
			return connect.NewResponse(&enrichmentv1.EnrichResponse{Enriched: true, Summary: "three useful facts", SuggestedPrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, Status: "complete"}), nil
		},
	})
	defer server.Close()
	core := scenarioApp(t, server.URL)

	command := registeredCommand(t, core, "enrich")
	var output strings.Builder
	ctx, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{"--document-hash", "sha256:abc", "--text", "local source text", "--privacy", "  SECRET  "}, core, &output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := command.RunCtx(ctx); err != nil {
		t.Fatalf("enrich command failed: %v", err)
	}
	if received == nil || received.DocumentHash != "sha256:abc" || received.Text != "local source text" || received.PrivacyClass != sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET {
		t.Fatalf("Enrich request = %+v, want supplied content and normalized secret privacy", received)
	}
	if got := strings.TrimSpace(output.String()); got != "Result:\n  Enrichment status=complete\nWhat Changed:\n  (none)\nNext Command:\n  (none)" {
		t.Fatalf("command output = %q, want successful enrichment status", got)
	}
}

func TestEmbedCommandUsesManifestDefaultsAndReportsResult(t *testing.T) {
	var received *enrichmentv1.EmbedRequest
	server := enrichmentServer(t, serviceFuncs{
		embed: func(_ context.Context, req *connect.Request[enrichmentv1.EmbedRequest]) (*connect.Response[enrichmentv1.EmbedResponse], error) {
			received = req.Msg
			return connect.NewResponse(&enrichmentv1.EmbedResponse{EmbeddingId: "embedding-42", Enriched: true, Dimension: 768, Status: "stored"}), nil
		},
	})
	defer server.Close()
	core := scenarioApp(t, server.URL)
	command := registeredCommand(t, core, "embed")
	var output strings.Builder
	ctx, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{"--document-hash", "sha256:def", "--unit-id", "unit-9", "--text", "search phrase"}, core, &output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := command.RunCtx(ctx); err != nil {
		t.Fatalf("embed command failed: %v", err)
	}
	if received == nil || received.DocumentHash != "sha256:def" || received.UnitId != "unit-9" || received.Text != "search phrase" || received.PrivacyClass != sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL || received.Role != "semantic-query" {
		t.Fatalf("Embed request = %+v, want supplied values plus manifest defaults", received)
	}
	if got := strings.TrimSpace(output.String()); got != "Result:\n  Embedding status=stored dimension=768\nWhat Changed:\n  (none)\nNext Command:\n  (none)" {
		t.Fatalf("command output = %q, want stored status and dimension", got)
	}
}

func TestCommandsPropagateGatewayErrors(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		service serviceFuncs
	}{
		{
			name: "enrich",
			args: []string{"--document-hash", "sha256:abc", "--text", "source"},
			service: serviceFuncs{
				enrich: func(context.Context, *connect.Request[enrichmentv1.EnrichRequest]) (*connect.Response[enrichmentv1.EnrichResponse], error) {
					return nil, connect.NewError(connect.CodeResourceExhausted, context.DeadlineExceeded)
				},
			},
		},
		{
			name: "embed",
			args: []string{"--document-hash", "sha256:abc", "--unit-id", "unit-1", "--text", "source"},
			service: serviceFuncs{
				embed: func(context.Context, *connect.Request[enrichmentv1.EmbedRequest]) (*connect.Response[enrichmentv1.EmbedResponse], error) {
					return nil, connect.NewError(connect.CodeResourceExhausted, context.DeadlineExceeded)
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := enrichmentServer(t, tc.service)
			defer server.Close()
			core := scenarioApp(t, server.URL)
			command := registeredCommand(t, core, tc.name)
			var output strings.Builder
			ctx, err := cliapp.NewTestRunContextFromArgs(command.Args, tc.args, core, &output, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := command.RunCtx(ctx); err == nil || !strings.Contains(err.Error(), "resource_exhausted") {
				t.Fatalf("%s error = %v, want propagated resource-exhausted Connect error", tc.name, err)
			}
			if output.Len() != 0 {
				t.Fatalf("failed %s wrote output %q", tc.name, output.String())
			}
		})
	}
}

type serviceFuncs struct {
	enrich func(context.Context, *connect.Request[enrichmentv1.EnrichRequest]) (*connect.Response[enrichmentv1.EnrichResponse], error)
	embed  func(context.Context, *connect.Request[enrichmentv1.EmbedRequest]) (*connect.Response[enrichmentv1.EmbedResponse], error)
}

func (s serviceFuncs) Enrich(ctx context.Context, req *connect.Request[enrichmentv1.EnrichRequest]) (*connect.Response[enrichmentv1.EnrichResponse], error) {
	return s.enrich(ctx, req)
}

func (s serviceFuncs) Embed(ctx context.Context, req *connect.Request[enrichmentv1.EmbedRequest]) (*connect.Response[enrichmentv1.EmbedResponse], error) {
	return s.embed(ctx, req)
}

func enrichmentServer(t *testing.T, service serviceFuncs) *httptest.Server {
	t.Helper()
	path, handler := enrichmentconnect.NewEnrichmentServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return httptest.NewServer(mux)
}

func scenarioApp(t *testing.T, apiBase string) *cliapp.ScenarioApp {
	t.Helper()
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:             "document-manager-enrichment-test",
		DefaultAPIBase:   apiBase,
		APIPrefix:        "/",
		APIEnvVars:       []string{"DOCUMENT_MANAGER_ENRICHMENT_TEST_API_BASE"},
		ConfigDirEnvVars: []string{"DOCUMENT_MANAGER_ENRICHMENT_TEST_CONFIG_DIR"},
	})
	if err != nil {
		t.Fatal(err)
	}
	core.APIOverride = apiBase
	return core
}

func registeredCommand(t *testing.T, core *cliapp.ScenarioApp, name string) cliapp.Command {
	t.Helper()
	group, err := Register(core, readManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range group.Subcommands {
		if command.Name == name {
			return command
		}
	}
	t.Fatalf("command %q is not registered", name)
	return cliapp.Command{}
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}
