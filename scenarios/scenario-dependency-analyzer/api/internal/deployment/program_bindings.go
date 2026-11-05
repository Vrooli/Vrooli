package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// ProgramBindingSource supplies program-to-scenario edges independently of
// service manifests. source identifies the evidence authority in the DAG.
type ProgramBindingSource interface {
	ProgramBindingTargets(scenario string) (targets []types.ProgramBindingTarget, source string, err error)
}

// CompositeProgramBindingSource prefers the live program-runtime library and
// falls back to checked-in contracts when the runtime is unavailable.
type CompositeProgramBindingSource struct {
	BaseURL  string
	Client   *http.Client
	RepoRoot string
	// ScenariosDir is used by the analyzer service, whose workspace root is
	// already the scenarios directory rather than the repository root.
	ScenariosDir string
}

func (s CompositeProgramBindingSource) ProgramBindingTargets(scenario string) ([]types.ProgramBindingTarget, string, error) {
	if strings.TrimSpace(s.BaseURL) != "" {
		client := libraryconnect.NewLibraryServiceClient(s.httpClient(), strings.TrimRight(s.BaseURL, "/"))
		response, err := client.ListLibrary(context.Background(), connect.NewRequest(&libraryv1.ListLibraryRequest{Limit: 1000}))
		if err == nil {
			targets := libraryTargets(response.Msg.GetPrograms(), scenario)
			if len(targets) > 0 {
				return targets, "program-binding", nil
			}
		}
	}
	targets, err := fileProgramBindingTargets(s.RepoRoot, s.ScenariosDir, scenario)
	return targets, "program-binding-file", err
}

func (s CompositeProgramBindingSource) httpClient() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return http.DefaultClient
}

func libraryTargets(programs []*sharedv1.LibraryProgram, scenario string) []types.ProgramBindingTarget {
	targets := make([]types.ProgramBindingTarget, 0)
	for _, program := range programs {
		if program == nil || (program.GetScenario() != "" && program.GetScenario() != scenario && !strings.HasPrefix(program.GetSourceProgramId(), scenario+".")) {
			continue
		}
		for _, bindingID := range program.GetCalledBindingIds() {
			if target, ok := bindingScenario(bindingID); ok {
				targets = append(targets, types.ProgramBindingTarget{Scenario: target, BindingID: bindingID, Program: program.GetName()})
			}
		}
	}
	return uniqueProgramTargets(targets)
}

func fileProgramBindingTargets(repoRoot, scenariosDir, scenario string) ([]types.ProgramBindingTarget, error) {
	base := strings.TrimSpace(scenariosDir)
	if base == "" {
		if strings.TrimSpace(repoRoot) == "" {
			return nil, fmt.Errorf("repository root is required for program-binding fallback")
		}
		base = filepath.Join(repoRoot, "scenarios")
	}
	dir := filepath.Join(base, scenario, ".vrooli", "program-runtime")
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var targets []types.ProgramBindingTarget
	for _, path := range paths {
		if strings.HasSuffix(path, ".output.schema.json") {
			continue
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var contract struct {
			Name     string `json:"name"`
			Bindings []struct {
				ID       string `json:"id"`
				Effect   string `json:"effect"`
				Optional bool   `json:"optional"`
			} `json:"bindings"`
		}
		if err := json.Unmarshal(data, &contract); err != nil {
			continue
		}
		for _, binding := range contract.Bindings {
			if target, ok := bindingScenario(binding.ID); ok {
				targets = append(targets, types.ProgramBindingTarget{Scenario: target, BindingID: binding.ID, Effect: binding.Effect, Program: contract.Name, Optional: binding.Optional})
			}
		}
	}
	return uniqueProgramTargets(targets), nil
}

func bindingScenario(id string) (string, bool) {
	prefix, _, ok := strings.Cut(strings.TrimSpace(id), "/")
	return strings.ReplaceAll(prefix, "_", "-"), ok && prefix != ""
}

func uniqueProgramTargets(targets []types.ProgramBindingTarget) []types.ProgramBindingTarget {
	seen := map[string]struct{}{}
	out := make([]types.ProgramBindingTarget, 0, len(targets))
	for _, target := range targets {
		key := target.Program + "\x00" + target.BindingID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scenario == out[j].Scenario {
			return out[i].BindingID < out[j].BindingID
		}
		return out[i].Scenario < out[j].Scenario
	})
	return out
}
