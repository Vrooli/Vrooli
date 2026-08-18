package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	bindingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings/bindings_v1connect"
)

type suite struct {
	Name        string     `json:"name"`
	Floor       int        `json:"floor"`
	FloorReason string     `json:"floor_reason,omitempty"`
	Cases       []caseSpec `json:"cases"`
}

type caseSpec struct {
	ID        string `json:"id"`
	Intent    string `json:"intent"`
	BindingID string `json:"binding_id,omitempty"`
	Negative  bool   `json:"negative,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type caseResult struct {
	ID             string   `json:"id"`
	Intent         string   `json:"intent"`
	Expected       string   `json:"expected_binding_id,omitempty"`
	Selected       []string `json:"selected_binding_ids,omitempty"`
	Met            bool     `json:"met"`
	NullVerdict    bool     `json:"null_verdict"`
	WrongSelection bool     `json:"wrong_selection"`
	Reason         string   `json:"reason,omitempty"`
}

type report struct {
	Suite          string       `json:"suite"`
	Cases          int          `json:"cases"`
	Positive       int          `json:"positive"`
	Negative       int          `json:"negative"`
	Met            int          `json:"met"`
	Missed         int          `json:"missed"`
	WrongSelection int          `json:"wrong_selection"`
	NullVerdict    int          `json:"null_verdict"`
	Floor          int          `json:"floor"`
	FloorReason    string       `json:"floor_reason"`
	Results        []caseResult `json:"results"`
}

func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "discovery",
		Description: "Measure intent discovery against a versioned golden corpus.",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{{
			Name:        "eval",
			Description: "Run the discovery golden corpus and emit falsifiable counts.",
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "suite", Required: true, Description: "Path to the discovery suite JSON."}, {Name: "mode", Description: "Discovery mode: fast, judged, or deep.", Default: "judged"}}},
			RunCtx:      run,
		}},
	}
}

func run(ctx cliapp.RunContext) error {
	path, err := resolveSuitePath(ctx.Flag("suite"))
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read discovery suite %q: %w", path, err)
	}
	var input suite
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("decode discovery suite %q: %w", path, err)
	}
	if len(input.Cases) == 0 {
		return fmt.Errorf("discovery suite %q has no cases", path)
	}
	client, base := cliapp.NewConnectHTTPClient(ctx.Core())
	registry := bindingsconnect.NewBindingRegistryServiceClient(client, base)
	floorReason := input.FloorReason
	if floorReason == "" {
		floorReason = "provider-direct Search Hub floor is the pre-change reference tier; this runner records the federated baseline before retrieval changes."
	}
	out := report{Suite: input.Name, Cases: len(input.Cases), Floor: input.Floor, FloorReason: floorReason, Results: make([]caseResult, 0, len(input.Cases))}
	for _, item := range input.Cases {
		if item.Negative {
			out.Negative++
		} else {
			out.Positive++
		}
		response, callErr := registry.ResolveIntent(context.Background(), connect.NewRequest(&bindingsv1.ResolveIntentRequest{Intent: item.Intent, Limit: 50, Mode: ctx.Flag("mode")}))
		result := caseResult{ID: item.ID, Intent: item.Intent, Expected: item.BindingID}
		if callErr != nil {
			result.Reason = callErr.Error()
		} else {
			// The candidate list is diagnostic context, not a successful
			// discovery verdict. Score only the typed result so a retrieval
			// response cannot claim success merely because the expected binding
			// appeared somewhere in its alternatives.
			if typed := response.Msg.GetResult(); typed != nil && typed.GetBindingId() != "" {
				result.Selected = []string{typed.GetBindingId()}
				result.Reason = typed.GetReason()
			} else {
				result.Reason = response.Msg.GetReason()
			}
		}
		if item.Negative {
			result.NullVerdict = len(result.Selected) == 0
			result.Met = result.NullVerdict
			if result.NullVerdict {
				out.NullVerdict++
			} else {
				result.WrongSelection = true
				out.WrongSelection++
			}
		} else {
			result.Met = contains(result.Selected, item.BindingID)
			switch {
			case result.Met:
				out.Met++
			case len(result.Selected) == 0:
				out.Missed++
			default:
				out.WrongSelection++
				result.WrongSelection = true
			}
		}
		out.Results = append(out.Results, result)
	}
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), out)
	}
	if err := ctx.RenderOperational(cliapp.OperationalReport{Status: []string{fmt.Sprintf("Discovery eval: met=%d missed=%d wrong-selection=%d null-verdict=%d floor=%d.", out.Met, out.Missed, out.WrongSelection, out.NullVerdict, out.Floor)}, NextSteps: []string{reportJSON(out)}}); err != nil {
		return err
	}
	return nil
}

func reportJSON(value report) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func resolveSuitePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("--suite is required")
	}
	path := filepath.Clean(value)
	if !filepath.IsAbs(path) {
		path, _ = filepath.Abs(path)
	}
	return path, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
