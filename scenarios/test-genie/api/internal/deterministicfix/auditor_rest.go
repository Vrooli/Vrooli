package deterministicfix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// StatusSkipped marks a reachable provider whose deterministic fix was not
// attempted (e.g. scenario-auditor, which requires explicit rule ids).
const StatusSkipped = "skipped"

type auditorFixRequest struct {
	ScenarioNames []string `json:"scenario_names"`
	RuleIDs       []string `json:"rule_ids"`
	DryRun        bool     `json:"dry_run"`
}

type auditorFixChange struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

type auditorFixResult struct {
	ScenarioName string             `json:"scenario_name"`
	RuleID       string             `json:"rule_id"`
	Fixed        bool               `json:"fixed"`
	FilePath     string             `json:"file_path"`
	Changes      []auditorFixChange `json:"changes"`
	Error        string             `json:"error,omitempty"`
}

type auditorFixResponse struct {
	Results        []auditorFixResult `json:"results"`
	Count          int                `json:"count"`
	UnfixableRules []string           `json:"unfixable_rules,omitempty"`
	Errors         []string           `json:"errors,omitempty"`
}

// auditorRESTFix reaches scenario-auditor's REST deterministic-fix endpoint. The
// endpoint requires explicit rule ids, so when none are supplied the provider is
// reported as skipped (noted debt: scenario-auditor has no Connect Fix RPC).
func auditorRESTFix(ctx context.Context, baseURL, scenario string, ruleIDs []string, apply bool) (ProviderReport, error) {
	if len(ruleIDs) == 0 {
		return ProviderReport{
			Status:   StatusSkipped,
			Messages: []string{"scenario-auditor deterministic fix requires explicit --rule ids; skipped"},
		}, nil
	}
	body, err := json.Marshal(auditorFixRequest{ScenarioNames: []string{scenario}, RuleIDs: ruleIDs, DryRun: !apply})
	if err != nil {
		return ProviderReport{}, err
	}
	url := strings.TrimRight(baseURL, "/") + "/api/v1/standards/fix"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ProviderReport{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return ProviderReport{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ProviderReport{}, fmt.Errorf("scenario-auditor fix returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed auditorFixResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ProviderReport{}, fmt.Errorf("parse scenario-auditor fix response: %w", err)
	}
	pr := ProviderReport{}
	for _, r := range parsed.Results {
		desc := r.RuleID
		if len(r.Changes) > 0 {
			details := make([]string, 0, len(r.Changes))
			for _, c := range r.Changes {
				details = append(details, c.Detail)
			}
			desc = strings.Join(details, "; ")
		}
		if r.Error != "" {
			pr.Messages = append(pr.Messages, fmt.Sprintf("%s: %s", r.RuleID, r.Error))
			continue
		}
		pr.Candidates = append(pr.Candidates, Candidate{
			RuleID:      r.RuleID,
			FilePath:    r.FilePath,
			Description: desc,
			Applied:     apply && r.Fixed,
		})
	}
	for _, ruleID := range parsed.UnfixableRules {
		pr.Messages = append(pr.Messages, "no fixer for rule "+ruleID)
	}
	pr.Messages = append(pr.Messages, parsed.Errors...)
	if len(pr.Candidates) == 0 {
		pr.Status = StatusClean
	} else {
		pr.Status = StatusFixed
	}
	return pr, nil
}
