package evals

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"

	"github.com/vrooli/cli-core/cliapp"
)

// TestEvalsManifestCoversEvalService asserts every RPC on EvalService has a
// bound command in cli/manifest.json — the CLI-side parity guard mirroring the
// API's TestProtoConnectParity. Adding an RPC to eval.proto without a CLI
// command (or vice versa) fails here.
func TestEvalsManifestCoversEvalService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, evalv1.File_search_hub_v1_eval_eval_proto, "EvalService")
}

func TestRunSuiteManifestDocumentsRouteSpecificTierDefault(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Flags       []struct {
					Name        string `json:"name"`
					Default     string `json:"default"`
					Description string `json:"description"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse cli/manifest.json: %v", err)
	}
	for _, group := range manifest.Groups {
		if group.Name != GroupName {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != "run" {
				continue
			}
			if !strings.Contains(command.Description, "router.routing through federated Query by default") {
				t.Fatalf("run description hides the synthetic suite's federated default: %q", command.Description)
			}
			for _, flag := range command.Flags {
				if flag.Name != "tier" {
					continue
				}
				if flag.Default != "" {
					t.Fatalf("tier must stay unset so the API can apply suite-specific defaults, got %q", flag.Default)
				}
				if !strings.Contains(flag.Description, "router.routing defaults to federated when omitted") {
					t.Fatalf("tier help omits the synthetic suite's default: %q", flag.Description)
				}
				return
			}
			t.Fatal("evals run tier flag was not declared")
		}
	}
	t.Fatal("evals run command was not registered")
}

func TestPublicBenchImportUsesLocalInputAndEvalServiceClient(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name     string `json:"name"`
				NeedsAPI bool   `json:"needs_api"`
				Binding  struct {
					Kind string `json:"kind"`
				} `json:"binding"`
				Architecture struct {
					Primitive string `json:"primitive"`
				} `json:"architecture"`
				Flags []struct {
					Name      string `json:"name"`
					Required  bool   `json:"required"`
					Default   string `json:"default"`
					LocalOnly bool   `json:"local_only"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse cli/manifest.json: %v", err)
	}
	for _, group := range manifest.Groups {
		if group.Name != GroupName {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != "import-publicbench" {
				continue
			}
			if command.Binding.Kind != "local" || !command.NeedsAPI || command.Architecture.Primitive != "proto_mutation" {
				t.Fatalf("publicbench import must use local input and register via the EvalService mutation path: %+v", command)
			}
			flags := map[string]struct {
				required bool
				defaultV string
				local    bool
			}{}
			for _, flag := range command.Flags {
				flags[flag.Name] = struct {
					required bool
					defaultV string
					local    bool
				}{flag.Required, flag.Default, flag.LocalOnly}
			}
			for _, name := range []string{"dataset", "data-dir", "provider-id"} {
				if !flags[name].required {
					t.Fatalf("--%s must be required: %+v", name, command.Flags)
				}
			}
			if flags["data-dir"].local != true || flags["max-cases"].defaultV != "20" || !flags["max-cases"].local {
				t.Fatalf("publicbench file/case bounds are not explicit: %+v", command.Flags)
			}
			return
		}
	}
	t.Fatal("evals import-publicbench command was not registered in the manifest")
}
