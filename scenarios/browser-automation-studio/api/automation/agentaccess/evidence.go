package agentaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/servicetoken"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

var ErrTrustedEvidenceUnavailable = errors.New("trusted browser evidence service identity is unavailable")

// EvidenceRecorder forwards bounded browser lifecycle/action metadata to the
// existing Agent Manager run transcript. It never receives page or input data.
type EvidenceRecorder interface {
	RecordBrowserSession(context.Context, string, sessionpermission.Backend, string, string) error
	RecordBrowserAction(context.Context, string, sessionpermission.Backend, string, string, string) error
}

// TrustedBrowserEvidenceReporter submits only a proof reference. The Agent
// Manager independently calls back into the broker for the closed summary.
type TrustedBrowserEvidenceReporter interface {
	RecordClosedBrowserSession(context.Context, string, string, string, string) (BrowserSettlementResult, error)
}

// BrowserSettlementResult is the bounded result returned by the authenticated
// Agent Manager proof-ingest callback. Receipt remains opaque JSON here so BAS
// relays LPBS's exact customer receipt without becoming a billing owner.
type BrowserSettlementResult struct {
	Status  string          `json:"status"`
	Receipt json.RawMessage `json:"receipt,omitempty"`
}

type BrowserEvidenceTokenSource interface {
	Token(context.Context) (string, error)
}

// AgentManagerEvidenceRecorder writes the minimal typed event contract owned
// by Agent Manager. Credentials are resolved from the authenticated request
// context and are never part of the JSON payload.
type AgentManagerEvidenceRecorder struct {
	RunContext    *AgentManagerRunContext
	ServiceTokens BrowserEvidenceTokenSource
}

// RecordClosedBrowserSession submits a proof reference to the BAS-only AM
// ingestion route. Session IDs and measurements are deliberately omitted; AM
// obtains them through its authenticated BAS proof-verifier callback.
func (r *AgentManagerEvidenceRecorder) RecordClosedBrowserSession(ctx context.Context, callerID, sessionID, grantReceipt, proofReference string) (BrowserSettlementResult, error) {
	if r == nil || r.RunContext == nil || r.ServiceTokens == nil {
		return BrowserSettlementResult{}, ErrTrustedEvidenceUnavailable
	}
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return BrowserSettlementResult{}, err
	}
	if credentials.grantReceipt != grantReceipt {
		return BrowserSettlementResult{}, errors.New("browser evidence grant does not match trusted request context")
	}
	runID, err := browserRunID(callerID)
	if err != nil {
		return BrowserSettlementResult{}, err
	}
	serviceToken, err := r.ServiceTokens.Token(ctx)
	if err != nil || strings.TrimSpace(serviceToken) == "" {
		if err == nil {
			err = errors.New("service identity returned an empty token")
		}
		if errors.Is(err, servicetoken.ErrNotIssued) || errors.Is(err, servicetoken.ErrNotConfigured) {
			return BrowserSettlementResult{}, fmt.Errorf("%w: %v", ErrTrustedEvidenceUnavailable, err)
		}
		return BrowserSettlementResult{}, fmt.Errorf("resolve BAS browser evidence service identity: %w", err)
	}
	endpoint, client, err := r.RunContext.endpoint("/api/v1/browser/bas-evidence/" + runID.String())
	if err != nil {
		return BrowserSettlementResult{}, err
	}
	body, err := json.Marshal(map[string]string{"proof_reference": proofReference})
	if err != nil {
		return BrowserSettlementResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return BrowserSettlementResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+serviceToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Identity-Token", credentials.identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", grantReceipt)
	resp, err := client.Do(req)
	if err != nil {
		return BrowserSettlementResult{}, fmt.Errorf("record BAS browser evidence with Agent Manager: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return BrowserSettlementResult{}, err
	}
	if resp.StatusCode != http.StatusAccepted {
		return BrowserSettlementResult{}, fmt.Errorf("Agent Manager BAS evidence returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var response struct {
		RunID      string `json:"run_id"`
		SessionID  string `json:"session_id"`
		Recorded   bool   `json:"recorded"`
		Idempotent bool   `json:"idempotent,omitempty"`
		ReportedBy string `json:"reported_by"`
		Settlement struct {
			Status  string          `json:"status"`
			Receipt json.RawMessage `json:"receipt"`
		} `json:"settlement"`
	}
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil || decoder.Decode(new(any)) != io.EOF || response.RunID != runID.String() || response.SessionID != sessionID || !response.Recorded || response.ReportedBy != "browser-automation-studio-broker" {
		return BrowserSettlementResult{}, errors.New("Agent Manager browser evidence response is invalid or not bound to the requested run")
	}
	result := BrowserSettlementResult{Status: response.Settlement.Status, Receipt: response.Settlement.Receipt}
	switch result.Status {
	case "settled":
		var receipt struct {
			Status        string `json:"status"`
			ReservationID string `json:"reservation_id"`
		}
		if len(result.Receipt) == 0 || json.Unmarshal(result.Receipt, &receipt) != nil || receipt.Status != "settled" || receipt.ReservationID == "" {
			return BrowserSettlementResult{}, errors.New("Agent Manager returned an invalid settled browser receipt")
		}
	case "pending", "not_applicable", "settlement_client_unavailable":
		if len(result.Receipt) != 0 {
			return BrowserSettlementResult{}, errors.New("Agent Manager returned a receipt for a non-settled browser session")
		}
	default:
		return BrowserSettlementResult{}, errors.New("Agent Manager browser evidence response has an unknown settlement status")
	}
	return result, nil
}

func (r *AgentManagerEvidenceRecorder) RecordBrowserSession(ctx context.Context, sessionID string, backend sessionpermission.Backend, profileID, state string) error {
	if state != "opened" && state != "closed" {
		return errors.New("browser session evidence state must be opened or closed")
	}
	mode, err := agentManagerMode(backend)
	if err != nil {
		return err
	}
	return r.record(ctx, map[string]any{"kind": "session", "session_id": sessionID, "mode": mode, "permission_profile": profileID, "state": state})
}

func (r *AgentManagerEvidenceRecorder) RecordBrowserAction(ctx context.Context, sessionID string, backend sessionpermission.Backend, profileID, action, outcome string) error {
	if !oneOf(action, "observe", "navigate", "click", "type", "scroll", "screenshot", "download", "upload", "attach", "close") {
		return fmt.Errorf("unsupported browser evidence action %q", action)
	}
	if !oneOf(outcome, "allowed", "denied", "completed", "failed") {
		return fmt.Errorf("unsupported browser evidence outcome %q", outcome)
	}
	mode, err := agentManagerMode(backend)
	if err != nil {
		return err
	}
	return r.record(ctx, map[string]any{"kind": "action", "session_id": sessionID, "mode": mode, "permission_profile": profileID, "action": action, "outcome": outcome})
}

func (r *AgentManagerEvidenceRecorder) record(ctx context.Context, payload map[string]any) error {
	if r == nil || r.RunContext == nil {
		return errors.New("Agent Manager browser evidence recorder unavailable")
	}
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return err
	}
	endpoint, client, err := r.RunContext.endpoint("/api/v1/browser/record-evidence")
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Identity-Token", credentials.identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", credentials.grantReceipt)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("record Agent Manager browser evidence: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Agent Manager browser evidence returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
