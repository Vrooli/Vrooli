package programs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

type DiscoveryEvalDeps struct {
	SuitePath string
	Resolve   func(context.Context, string, int32, string) (*bindingsv1.ResolveIntentResponse, error)
}

type discoverySuite struct {
	Name        string              `json:"name"`
	Floor       int                 `json:"floor"`
	FloorReason string              `json:"floor_reason"`
	Cases       []discoveryCaseSpec `json:"cases"`
}

type discoveryCaseSpec struct {
	ID        string `json:"id"`
	Intent    string `json:"intent"`
	BindingID string `json:"binding_id"`
	Negative  bool   `json:"negative"`
}

type DiscoveryEvalResult struct {
	Suite       string
	Status      string
	Reason      string
	Floor       int
	FloorReason string
	Cases       []DiscoveryCaseResult
	Met         int
	Missed      int
	Wrong       int
	Null        int
	FloorMet    bool
}

type DiscoveryCaseResult struct {
	ID             string
	Intent         string
	Expected       string
	Selected       string
	Met            bool
	NullVerdict    bool
	WrongSelection bool
	Reason         string
}

func RunDiscoveryEval(ctx context.Context, deps DiscoveryEvalDeps, mode string, maxCases int32) DiscoveryEvalResult {
	result := DiscoveryEvalResult{Status: "unavailable", Reason: "discovery evaluation is unavailable"}
	if deps.Resolve == nil {
		return result
	}
	data, err := os.ReadFile(deps.SuitePath)
	if err != nil {
		result.Reason = fmt.Sprintf("read discovery suite: %v", err)
		return result
	}
	var suite discoverySuite
	if err := json.Unmarshal(data, &suite); err != nil {
		result.Reason = fmt.Sprintf("decode discovery suite: %v", err)
		return result
	}
	if len(suite.Cases) == 0 {
		result.Reason = "discovery suite has no cases"
		return result
	}
	if maxCases > 0 && int(maxCases) < len(suite.Cases) {
		suite.Cases = suite.Cases[:maxCases]
	}
	result.Suite, result.Floor, result.FloorReason = suite.Name, suite.Floor, suite.FloorReason
	result.Cases = make([]DiscoveryCaseResult, 0, len(suite.Cases))
	for _, item := range suite.Cases {
		caseResult := DiscoveryCaseResult{ID: item.ID, Intent: item.Intent, Expected: item.BindingID}
		response, callErr := deps.Resolve(ctx, item.Intent, 50, mode)
		if callErr != nil {
			caseResult.Reason = callErr.Error()
		} else if response != nil {
			if selected := response.GetResult().GetBindingId(); selected != "" {
				caseResult.Selected = selected
			}
			caseResult.Reason = response.GetReason()
		}
		if item.Negative {
			caseResult.NullVerdict = caseResult.Selected == ""
			caseResult.Met = caseResult.NullVerdict
			if caseResult.NullVerdict {
				result.Null++
			} else {
				result.Wrong++
				caseResult.WrongSelection = true
			}
		} else {
			caseResult.Met = caseResult.Selected == item.BindingID
			switch {
			case caseResult.Met:
				result.Met++
			case caseResult.Selected == "":
				result.Missed++
			default:
				result.Wrong++
				caseResult.WrongSelection = true
			}
		}
		result.Cases = append(result.Cases, caseResult)
	}
	result.Status = "below_floor"
	result.FloorMet = result.Met >= result.Floor
	if result.FloorMet {
		result.Status = "met"
	}
	return result
}
