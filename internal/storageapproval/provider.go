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
	HostID string `json:"host_id"`
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
		Title:       "Storage recovery standing approvals",
		Description: "Choose which privileged storage recovery providers may run autonomously on this host.",
		Risk:        "Approved providers may delete regenerable system data under the recovery policy.",
		Inputs:      inputs,
		Policy:      operatorcapability.Policy{Idempotent: true, Retryable: true, RequiresConfirmation: true, Remediation: "Start storage-manager and retry setup; approvals can be revoked through storage-manager cleanup approvals."},
		Evidence:    operatorcapability.EvidenceContract{Kinds: []string{"storage-standing-approval"}, RequiredFields: []string{"provider_id", "host_id", "approved_at"}, SecretFree: true},
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
	return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateReady, Outcome: fmt.Sprintf("recorded %d standing approval(s) for host %s", approvedCount, hostID), Retryable: true, Evidence: []operatorcapability.EvidenceReference{{Kind: "storage-standing-approval", ArtifactIdentity: CapabilityID, ObservedAt: a.now().UTC(), Verified: true}}}, nil
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
		return fmt.Errorf("storage-manager approvals returned HTTP %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode storage-manager approvals: %w", err)
	}
	return nil
}
