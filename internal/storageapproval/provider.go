// Package storageapproval exposes storage-manager's host-local standing
// approvals through the generic setup capability contract.
package storageapproval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/operatorcapability"
)

const CapabilityID = "storage-manager-standing-approvals"

var providerIDs = []string{
	"docker-unused-images",
	"docker-unused-volumes",
	"journald",
	"log-volume-force-rotate",
}

type approval struct {
	HostID     string    `json:"host_id"`
	ApprovedAt time.Time `json:"approved_at"`
}

type API struct {
	baseURL func() string
	client  *http.Client
	now     func() time.Time
	hostID  func() string
}

func New() *API {
	return &API{
		baseURL: cliutil.DetectPortFromVrooli("storage-manager", "API_PORT"),
		client:  http.DefaultClient,
		now:     time.Now,
		hostID: func() string {
			host, _ := os.Hostname()
			return strings.TrimSpace(host)
		},
	}
}

func NewWithClient(baseURL func() string, client *http.Client, hostID string) *API {
	if client == nil {
		client = http.DefaultClient
	}
	return &API{baseURL: baseURL, client: client, now: time.Now, hostID: func() string { return hostID }}
}

func (a *API) Descriptor() operatorcapability.Descriptor {
	inputs := make([]operatorcapability.InputDescriptor, 0, len(providerIDs))
	for _, id := range providerIDs {
		inputs = append(inputs, operatorcapability.InputDescriptor{
			ID: id, Kind: operatorcapability.KindBoolean, Label: "Approve " + id + " under storage pressure",
			Description: "Allow storage-manager to invoke this provider on this host without a fresh approval token.",
			Default:     "false", Required: false, Validation: "true or false; approval is host-local and revocable",
		})
	}
	return operatorcapability.Descriptor{
		Version: operatorcapability.ContractVersion, ID: CapabilityID, Owner: "storage-manager",
		Scope: "host storage recovery", Purpose: "grant standing approval to named storage recovery providers",
		Sensitivity: operatorcapability.SensitivityOperator, Disposition: operatorcapability.DispositionConfigurable,
		Provenance:  operatorcapability.PermissionProvenance{Requester: "onboarding operator", Scope: "this host and the named storage providers", GrantSource: "reviewed capability apply with explicit confirmation", RevocationLimit: "storage-manager can revoke each provider approval; approval never grants an arbitrary path or command"},
		Lifecycle:   operatorcapability.Lifecycle{Preview: true, Apply: true, Verify: true, Revoke: true, Recovery: "start storage-manager and retry the owner approval operation"},
		Title:       "Storage recovery standing approvals",
		Description: "Choose which privileged storage recovery providers may run autonomously on this host.",
		Risk:        "Approved providers may delete regenerable system data under the recovery policy.",
		Inputs:      inputs,
		Policy:      operatorcapability.Policy{Idempotent: true, Retryable: true, RequiresConfirmation: true, Remediation: "Start storage-manager and retry setup; approvals can be revoked through storage-manager cleanup approvals."},
		Evidence: operatorcapability.EvidenceContract{
			Kinds:          []string{"storage-standing-approval"},
			Stages:         []operatorcapability.VerificationStage{operatorcapability.VerificationAuthorization},
			RequiredFields: []string{"artifact_identity", "target_id", "operation", "status", "coverage", "observed_at", "expires_at", "verified"},
			SecretFree:     true,
			Freshness:      "approval observation and target binding must remain current",
		},
		Remediation: "Review each provider carefully. Approvals are stored per host and may be revoked with storage-manager cleanup approvals revoke.",
	}
}

func (a *API) Discover(ctx context.Context) (operatorcapability.Status, error) {
	if a == nil || a.baseURL == nil {
		return operatorcapability.Status{}, fmt.Errorf("storage approval API is not configured")
	}
	base := strings.TrimRight(strings.TrimSpace(a.baseURL()), "/")
	if base == "" {
		return operatorcapability.Status{Descriptor: a.Descriptor(), State: operatorcapability.StateNeedsInput, MissingInputs: append([]string(nil), providerIDs...), Remediation: "storage-manager is not running; start it before applying these standing approvals.", UpdatedAt: a.now().UTC()}, nil
	}
	hostID := strings.TrimSpace(a.hostID())
	var approvals map[string]approval
	if err := a.get(ctx, base+"/api/v1/cleanup/approvals", &approvals); err != nil {
		return operatorcapability.Status{Descriptor: a.Descriptor(), State: operatorcapability.StateNeedsInput, MissingInputs: append([]string(nil), providerIDs...), Remediation: err.Error(), UpdatedAt: a.now().UTC()}, nil
	}
	missing := make([]string, 0, len(providerIDs))
	for _, id := range providerIDs {
		if strings.TrimSpace(approvals[id].HostID) == "" || strings.TrimSpace(approvals[id].HostID) != hostID {
			missing = append(missing, id)
		}
	}
	state := operatorcapability.StateReady
	if len(missing) > 0 {
		state = operatorcapability.StateNeedsInput
	}
	return operatorcapability.Status{Descriptor: a.Descriptor(), State: state, MissingInputs: missing, UpdatedAt: a.now().UTC()}, nil
}

func (a *API) Preview(_ context.Context, inputs operatorcapability.InputSet) (operatorcapability.Preview, error) {
	mutations := make([]operatorcapability.Mutation, 0, len(providerIDs))
	for _, id := range providerIDs {
		approved, ok := inputs.Boolean(id)
		if ok && approved {
			mutations = append(mutations, operatorcapability.Mutation{ID: id, Summary: "record host-local standing approval", Reversible: true})
		}
	}
	return operatorcapability.Preview{CapabilityID: CapabilityID, State: operatorcapability.StateReadyToPreview, Mutations: mutations, Remediation: "Only selected providers will receive standing approval."}, nil
}

func (a *API) Apply(ctx context.Context, inputs operatorcapability.InputSet) (result operatorcapability.Result, err error) {
	if a == nil || a.baseURL == nil {
		return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateRetryableFailure, Retryable: true, ErrorCode: "api_unconfigured"}, fmt.Errorf("storage approval API is not configured")
	}
	base := strings.TrimRight(strings.TrimSpace(a.baseURL()), "/")
	hostID := strings.TrimSpace(a.hostID())
	if base == "" || hostID == "" {
		return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateRetryableFailure, Retryable: true, ErrorCode: "host_or_api_unavailable", Remediation: "Start storage-manager and ensure the host identity is available."}, nil
	}
	approvedCount := 0
	for _, id := range providerIDs {
		approved, ok := inputs.Boolean(id)
		if !ok || !approved {
			continue
		}
		body, _ := json.Marshal(map[string]any{"approved_at": a.now().UTC(), "approved_by": "vrooli setup", "host_id": hostID, "subject_constraints": map[string]string{}})
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/cleanup/approvals/"+id, bytes.NewReader(body))
		if reqErr != nil {
			return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateRetryableFailure, Retryable: true, ErrorCode: "request_failed", Remediation: reqErr.Error()}, nil
		}
		req.Header.Set("Content-Type", "application/json")
		resp, doErr := a.client.Do(req)
		if doErr != nil {
			return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateRetryableFailure, Retryable: true, ErrorCode: "storage_manager_unavailable", Remediation: doErr.Error()}, nil
		}
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateRetryableFailure, Retryable: true, ErrorCode: "approval_rejected", Remediation: fmt.Sprintf("storage-manager returned HTTP %s for %s", resp.Status, id)}, nil
		}
		approvedCount++
	}
	return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateReady, Outcome: fmt.Sprintf("recorded %d standing approval(s) for host %s", approvedCount, hostID), Retryable: true, Evidence: []operatorcapability.EvidenceReference{{Stage: operatorcapability.VerificationAuthorization, Kind: "storage-standing-approval", ArtifactIdentity: CapabilityID, ObservedAt: a.now().UTC(), Verified: true}}}, nil
}

// Verify is the owner boundary for standing-approval evidence. The adapter
// re-reads storage-manager's approvals instead of treating a successful POST
// or a configured value as proof. Each receipt retains the approval's own
// observation time and is bound to the exact onboarding verification context.
func (a *API) Verify(ctx context.Context, request operatorcapability.VerificationRequest) ([]operatorcapability.EvidenceReference, error) {
	if a == nil || a.baseURL == nil {
		return nil, &operatorcapability.VerificationError{Code: "storage_approval_unconfigured", Retryable: false, NextAction: "configure-storage-manager-verification", Cause: fmt.Errorf("storage approval API is not configured")}
	}
	hostID := strings.TrimSpace(a.hostID())
	targetID := strings.TrimSpace(request.TargetID)
	if hostID == "" || targetID == "" {
		return nil, &operatorcapability.VerificationError{Code: "storage_approval_context_missing", Retryable: false, NextAction: "select-storage-manager-target", Cause: fmt.Errorf("storage approval verification requires a host and target")}
	}
	if targetID != "local" && targetID != hostID {
		return nil, &operatorcapability.VerificationError{Code: "storage_approval_target_mismatch", Retryable: false, NextAction: "select-local-storage-target", Cause: fmt.Errorf("storage-manager approvals are local to host %q, not target %q", hostID, targetID)}
	}
	base := strings.TrimRight(strings.TrimSpace(a.baseURL()), "/")
	if base == "" {
		return nil, &operatorcapability.VerificationError{Code: "storage_manager_unavailable", Retryable: true, NextAction: "retry-storage-manager-verification", Cause: fmt.Errorf("storage-manager is not running")}
	}
	var approvals map[string]approval
	if err := a.get(ctx, base+"/api/v1/cleanup/approvals", &approvals); err != nil {
		return nil, err
	}
	now := a.now().UTC()
	receipts := make([]operatorcapability.EvidenceReference, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		current, ok := approvals[providerID]
		if !ok || strings.TrimSpace(current.HostID) != hostID {
			return nil, &operatorcapability.VerificationError{Code: "storage_approval_missing", Retryable: false, NextAction: "apply-storage-manager-approval", Cause: fmt.Errorf("storage-manager approval for %q is missing or bound to another host", providerID)}
		}
		if current.ApprovedAt.IsZero() || current.ApprovedAt.After(now) {
			return nil, &operatorcapability.VerificationError{Code: "storage_approval_invalid", Retryable: false, NextAction: "repair-storage-manager-approval", Cause: fmt.Errorf("storage-manager approval for %q has no valid approval timestamp", providerID)}
		}
		receipts = append(receipts, operatorcapability.EvidenceReference{
			SchemaVersion:    operatorcapability.EvidenceSchemaVersion,
			CapabilityID:     request.CapabilityID,
			Kind:             "storage-standing-approval",
			Stage:            operatorcapability.VerificationAuthorization,
			ArtifactIdentity: "storage-manager/approval/" + providerID,
			TargetID:         targetID,
			Environment:      request.Environment,
			AccountIdentity:  request.AccountIdentity,
			Operation:        request.Operation,
			Status:           "approved",
			Coverage: []string{
				"provider_id:" + providerID,
				"host_id:" + hostID,
				"approved_at:" + current.ApprovedAt.UTC().Format(time.RFC3339Nano),
			},
			ObservedAt:  current.ApprovedAt.UTC(),
			ExpiresAt:   now.Add(24 * time.Hour),
			EffectClass: string(operatorcapability.EffectReadOnly),
			Verified:    true,
		})
	}
	return receipts, nil
}

func (a *API) get(ctx context.Context, url string, output any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("storage-manager approvals unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := "provider_rejected"
		retryable := false
		nextAction := "review-storage-manager-approval"
		retryAfter := time.Duration(0)
		if resp.StatusCode == http.StatusTooManyRequests {
			code = "provider_rate_limited"
			retryable = true
			nextAction = "retry-after-provider-backoff"
			if seconds, parseErr := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After"))); parseErr == nil && seconds > 0 {
				retryAfter = time.Duration(seconds) * time.Second
			}
		} else if resp.StatusCode >= 500 {
			code = "provider_unavailable"
			retryable = true
			nextAction = "retry-storage-manager-verification"
		}
		return &operatorcapability.VerificationError{Code: code, Retryable: retryable, RetryAfter: retryAfter, NextAction: nextAction, Cause: fmt.Errorf("storage-manager approvals returned HTTP %s", resp.Status)}
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return &operatorcapability.VerificationError{Code: "provider_malformed_response", Retryable: false, NextAction: "diagnose-storage-manager-provider", Cause: fmt.Errorf("decode storage-manager approvals: %w", err)}
	}
	return nil
}
