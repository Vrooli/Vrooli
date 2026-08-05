package crossosgate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"deployment-manager/shared"
)

// httpBridge speaks bridge's GateService over the Connect unary JSON protocol: a
// POST to {baseURL}/vrooli.vrooli_bridge.v1.gate.GateService/{Method} with the
// request message as a JSON body and the response message as a JSON body. No
// proto/Connect dependency — deployment-manager owns only this thin wire client.
type httpBridge struct {
	baseURL string
	token   string
	client  *http.Client
}

var _ Bridge = (*httpBridge)(nil)

// NewHTTPBridge constructs a Bridge against a live bridge control plane. token is
// the owner/service bearer credential bridge's owner-gated GateService requires
// (empty omits the Authorization header — useful in tests).
func NewHTTPBridge(baseURL, token string, client *http.Client) Bridge {
	if client == nil {
		client = &http.Client{Timeout: 0} // gate waits are long-lived; no client deadline
	}
	return &httpBridge{baseURL: strings.TrimRight(baseURL, "/"), token: token, client: client}
}

// gate proto wire shapes (snake_case JSON, matching the generated messages).
type runGateWire struct {
	Scenario       string   `json:"scenario"`
	TargetRevision string   `json:"target_revision"`
	TargetOses     []string `json:"target_oses"`
	TimeoutSeconds int64    `json:"timeout_seconds,omitempty"`
}

type osResultWire struct {
	OS          string `json:"os"`
	NodeID      string `json:"node_id"`
	RunID       string `json:"run_id"`
	Disposition string `json:"disposition"`
	ExitCode    int32  `json:"exit_code"`
	Detail      string `json:"detail"`
}

type runGateRespWire struct {
	GateID  string         `json:"gate_id"`
	Verdict string         `json:"verdict"`
	Results []osResultWire `json:"results"`
}

type waitGateReqWire struct {
	ID             string `json:"id"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type gateWire struct {
	ID      string `json:"id"`
	Verdict string `json:"verdict"`
}

type waitGateRespWire struct {
	Gate     gateWire       `json:"gate"`
	Results  []osResultWire `json:"results"`
	TimedOut bool           `json:"timed_out"`
}

func (h *httpBridge) RunGate(ctx context.Context, in Request) (string, []OSResult, error) {
	var resp runGateRespWire
	err := h.call(ctx, "RunGate", runGateWire{
		Scenario:       in.Scenario,
		TargetRevision: in.Revision,
		TargetOses:     in.TargetOSes,
		TimeoutSeconds: in.TimeoutSeconds,
	}, &resp)
	if err != nil {
		return "", nil, err
	}
	return resp.GateID, fromWire(resp.Results), nil
}

func (h *httpBridge) WaitGate(ctx context.Context, gateID string, timeoutSeconds int64) (string, bool, []OSResult, error) {
	var resp waitGateRespWire
	err := h.call(ctx, "WaitGate", waitGateReqWire{ID: gateID, TimeoutSeconds: timeoutSeconds}, &resp)
	if err != nil {
		return "", false, nil, err
	}
	return resp.Gate.Verdict, resp.TimedOut, fromWire(resp.Results), nil
}

// call performs one Connect unary JSON round-trip.
func (h *httpBridge) call(ctx context.Context, method string, reqMsg, respMsg any) error {
	body, err := json.Marshal(reqMsg)
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", method, err)
	}
	url := fmt.Sprintf("%s/vrooli.vrooli_bridge.v1.gate.GateService/%s", h.baseURL, method)
	validatedURL, err := shared.ValidateServiceURL(url)
	if err != nil {
		return fmt.Errorf("invalid bridge service URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, validatedURL, bytes.NewReader(body)) //nolint:gosec // bridge URL is validated immediately above
	if err != nil {
		return fmt.Errorf("build %s request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if h.token != "" {
		req.Header.Set("Authorization", "Bearer "+h.token)
	}

	resp, err := h.client.Do(req) //nolint:gosec // request target passed the bridge URL validation boundary
	if err != nil {
		return fmt.Errorf("call bridge GateService/%s: %w", method, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		// Connect errors are a JSON {"code","message"} body on a non-200 status.
		var connectErr struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &connectErr) == nil && connectErr.Message != "" {
			return fmt.Errorf("bridge GateService/%s failed (%s): %s", method, connectErr.Code, connectErr.Message)
		}
		return fmt.Errorf("bridge GateService/%s failed: HTTP %d: %s", method, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.Unmarshal(raw, respMsg); err != nil {
		return fmt.Errorf("decode %s response: %w", method, err)
	}
	return nil
}

func fromWire(in []osResultWire) []OSResult {
	out := make([]OSResult, 0, len(in))
	for _, r := range in {
		// osResultWire and OSResult share field names + types (differing only in
		// JSON tags), so a direct struct conversion is exact.
		out = append(out, OSResult(r))
	}
	return out
}

// NewHTTPHandler builds the HTTP handler for the cross-OS gate route. When
// baseURL is empty (bridge not configured) it returns a handler that responds
// 503, so the route is purely additive and inert until an operator wires the
// bridge endpoint via VROOLI_BRIDGE_URL.
func NewHTTPHandler(baseURL, token string) *Handler {
	if strings.TrimSpace(baseURL) == "" {
		return &Handler{}
	}
	return &Handler{gate: New(NewHTTPBridge(baseURL, token, &http.Client{Timeout: 65 * time.Minute}))}
}
