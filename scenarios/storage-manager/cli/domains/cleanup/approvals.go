package cleanup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// standingApproval mirrors the API's host-local approval record. A standing
// approval is the operator's durable consent for one conditional-tier provider
// (for example go-build-cache) to run during autonomous pressure recovery on
// this host only; the API refuses a record whose host_id is not its own.
type standingApproval struct {
	Provider   string    `json:"provider,omitempty"`
	ApprovedAt time.Time `json:"approved_at"`
	ApprovedBy string    `json:"approved_by"`
	HostID     string    `json:"host_id"`
}

const standingApprovalsPath = "/api/v1/cleanup/approvals"

func (h *handlers) approvalsRequest(method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, h.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("standing approvals: %s", strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (h *handlers) approvalsListCall(cliapp.OperationContext) ([]standingApproval, error) {
	data, err := h.approvalsRequest(http.MethodGet, standingApprovalsPath, nil)
	if err != nil {
		return nil, err
	}
	var byProvider map[string]standingApproval
	if err := json.Unmarshal(data, &byProvider); err != nil {
		return nil, fmt.Errorf("decode standing approvals: %w", err)
	}
	out := make([]standingApproval, 0, len(byProvider))
	for provider, approval := range byProvider {
		approval.Provider = provider
		out = append(out, approval)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
	return out, nil
}

func approvalsListReport(_ cliapp.OperationContext, approvals []standingApproval) cliapp.MutationReport {
	rows := make([]string, 0, len(approvals))
	for _, a := range approvals {
		rows = append(rows, fmt.Sprintf("%s approved_by=%s host=%s at=%s", a.Provider, a.ApprovedBy, a.HostID, a.ApprovedAt.Format(time.RFC3339)))
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Found %d standing approval(s).", len(rows))}, Changes: rows}
}

func (h *handlers) approvalsGrantCall(ctx cliapp.OperationContext) ([]standingApproval, error) {
	provider := strings.TrimSpace(ctx.Flag("provider"))
	if provider == "" {
		return nil, fmt.Errorf("--provider is required")
	}
	approvedBy := strings.TrimSpace(ctx.Flag("approved-by"))
	if approvedBy == "" {
		approvedBy = strings.TrimSpace(os.Getenv("USER"))
	}
	if approvedBy == "" {
		return nil, fmt.Errorf("--approved-by is required when USER is unset")
	}
	// The API binds the approval to its own host identity; sending ours lets it
	// refuse a grant aimed at a different machine instead of silently accepting.
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("resolve host identity: %v", err)
	}
	approval := standingApproval{ApprovedAt: time.Now().UTC(), ApprovedBy: approvedBy, HostID: strings.TrimSpace(host)}
	data, err := h.approvalsRequest(http.MethodPost, standingApprovalsPath+"/"+provider, approval)
	if err != nil {
		return nil, err
	}
	var saved standingApproval
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("decode standing approval: %w", err)
	}
	saved.Provider = provider
	return []standingApproval{saved}, nil
}

func (h *handlers) approvalsRevokeCall(ctx cliapp.OperationContext) ([]standingApproval, error) {
	provider := strings.TrimSpace(ctx.Flag("provider"))
	if provider == "" {
		return nil, fmt.Errorf("--provider is required")
	}
	if _, err := h.approvalsRequest(http.MethodDelete, standingApprovalsPath+"/"+provider, nil); err != nil {
		return nil, err
	}
	return []standingApproval{{Provider: provider}}, nil
}

func approvalsChangeReport(verb string) func(cliapp.OperationContext, []standingApproval) cliapp.MutationReport {
	return func(_ cliapp.OperationContext, approvals []standingApproval) cliapp.MutationReport {
		rows := make([]string, 0, len(approvals))
		for _, a := range approvals {
			rows = append(rows, fmt.Sprintf("%s %s", verb, a.Provider))
		}
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Standing approval %s.", verb)}, Changes: rows}
	}
}
