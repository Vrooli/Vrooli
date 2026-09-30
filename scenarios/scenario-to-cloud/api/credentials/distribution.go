package credentials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// IngestSchemaVersion is the sealed-payload schema the target's
// `vrooli cloud-target credential ingest` verb accepts on standard input.
const IngestSchemaVersion = 1

// IngestEnvName is the environment variable a Bridge dispatch injects the
// value into for the ingest verb. The dispatched argv names the variable,
// never the value.
const IngestEnvName = "VROOLI_CREDENTIAL_INGEST_VALUE"

// Target is the deployment target a distribution addresses.
type Target struct {
	DeploymentID string
	Ref          identity.TargetRef
	Fence        uint64
}

// NodeLabel is the metadata-safe name of the target for receipts.
func (t Target) NodeLabel() string {
	if t.Ref.NodeID != "" {
		return "node:" + t.Ref.NodeID
	}
	if t.Ref.MachineID != "" {
		return "machine:" + t.Ref.MachineID
	}
	if t.Ref.Locator.Host != "" {
		return "host:" + t.Ref.Locator.Host
	}
	return "target:" + t.DeploymentID
}

// IngestPayload is the standard-input document the target ingests. It is the
// only place a value travels on the SSH transport, and it travels inside the
// SSH channel; on the bridge transport the value travels inside the Bridge
// sealed grant channel and the payload carries the grant reference instead.
type IngestPayload struct {
	SchemaVersion int    `json:"schema_version"`
	DeploymentID  string `json:"deployment_id"`
	BindingID     string `json:"binding_id"`
	LogicalID     string `json:"logical_id"`
	Field         string `json:"field"`
	Version       int64  `json:"version"`
	ContentRef    string `json:"content_ref"`
	Value         string `json:"value,omitempty"`
	GrantRef      string `json:"grant_ref,omitempty"`
}

// DeliverRequest distributes one version to one target.
type DeliverRequest struct {
	Target      Target
	Binding     domain.CredentialBinding
	Version     domain.CredentialVersion
	Value       string
	OperationID string
	Step        string
}

// RevokeRequest purges one version from one target.
type RevokeRequest struct {
	Target      Target
	Binding     domain.CredentialBinding
	Version     int64
	OperationID string
	Step        string
}

// AckRequest asks the target to prove one consumer can use one version.
type AckRequest struct {
	Target      Target
	Binding     domain.CredentialBinding
	Version     int64
	Consumer    string
	OperationID string
	Step        string
}

// ProbeResult is the metadata-only view of a binding on its target.
type ProbeResult struct {
	Configured    bool
	StoreUnlocked bool
	StoreState    string
}

// Receipt is the distribution outcome. Ref points at the target-side receipt
// (operation/step on SSH; run id on bridge).
type Receipt struct {
	Transport   string         `json:"transport"`
	Ref         string         `json:"ref,omitempty"`
	Replayed    bool           `json:"replayed,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
	Limitations []string       `json:"limitations,omitempty"`
}

// Distributor is the transport seam: SSH runner today, Bridge grants and
// dispatch when the target binding says so.
type Distributor interface {
	Transport() string
	Probe(ctx context.Context, target Target, binding domain.CredentialBinding) (ProbeResult, error)
	Deliver(ctx context.Context, req DeliverRequest) (Receipt, error)
	// Acknowledge runs the target's `credential acknowledge` verb for one
	// consumer; an UnreachableError means the consumer's standing is unknown.
	Acknowledge(ctx context.Context, req AckRequest) (Receipt, error)
	Revoke(ctx context.Context, req RevokeRequest) (Receipt, error)
}

// UnreachableError means the target was not reached at all; no effect can
// be assumed either way, which is why revocation stays incomplete.
type UnreachableError struct {
	Target string
	Cause  error
}

func (e *UnreachableError) Error() string {
	return fmt.Sprintf("target %s unreachable: %v", e.Target, e.Cause)
}

func (e *UnreachableError) Unwrap() error { return e.Cause }

// IsUnreachable reports whether err is a transport-level failure.
func IsUnreachable(err error) bool {
	var unreachable *UnreachableError
	return errors.As(err, &unreachable)
}

// targetReceipt is the JSON the cloud-target verbs print.
type targetReceipt struct {
	Receipt struct {
		OperationID string         `json:"operation_id"`
		Step        string         `json:"step"`
		Outcome     string         `json:"outcome"`
		Details     map[string]any `json:"details"`
		Error       *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	} `json:"receipt"`
	Replayed bool `json:"replayed"`
	Error    *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// SSHDistributor reaches the target's credential authority over the SSH
// runner. Metadata reads go through credentialclient; ingestion and
// revocation go through the receipted cloud-target verbs.
type SSHDistributor struct {
	Runner ArgvRunner
	Client credentialclient.Client
	Label  string
}

// NewSSHDistributor wires the runner and metadata client for one target.
func NewSSHDistributor(runner ArgvRunner, client credentialclient.Client, label string) *SSHDistributor {
	return &SSHDistributor{Runner: runner, Client: client, Label: label}
}

func (d *SSHDistributor) Transport() string { return identity.TransportSSH }

func (d *SSHDistributor) Probe(ctx context.Context, target Target, binding domain.CredentialBinding) (ProbeResult, error) {
	store, err := d.Client.StoreStatus(ctx)
	if err != nil {
		return ProbeResult{}, d.classify(target, err)
	}
	if store.Initialized && !store.Unlocked {
		return ProbeResult{StoreUnlocked: false, StoreState: "locked"}, StoreLocked(target.NodeLabel())
	}
	status, err := d.Client.Status(ctx, binding.Descriptor.LogicalID, binding.Descriptor.Field)
	if err != nil {
		return ProbeResult{}, d.classify(target, err)
	}
	if status.ProviderState != "" && status.ProviderState != "available" {
		return ProbeResult{StoreState: status.ProviderState}, newError(CodeDistributionFailed, "the target credential store is "+status.ProviderState+"; refusing to decide whether the credential exists").WithDetail("descriptor", binding.Descriptor.Address())
	}
	return ProbeResult{Configured: status.Configured, StoreUnlocked: true, StoreState: "available"}, nil
}

func (d *SSHDistributor) Deliver(ctx context.Context, req DeliverRequest) (Receipt, error) {
	payload := IngestPayload{SchemaVersion: IngestSchemaVersion, DeploymentID: req.Target.DeploymentID, BindingID: req.Binding.ID, LogicalID: req.Binding.Descriptor.LogicalID, Field: req.Binding.Descriptor.Field, Version: req.Version.Number, ContentRef: req.Version.ContentRef, Value: req.Value}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Receipt{}, err
	}
	args := []string{
		"vrooli", "cloud-target", "credential", "ingest",
		"--deployment", req.Target.DeploymentID, "--binding", req.Binding.ID,
		"--version", strconv.FormatInt(req.Version.Number, 10),
		"--operation", req.OperationID, "--step", req.Step,
		"--fence", strconv.FormatUint(req.Target.Fence, 10), "--json",
	}
	return d.runReceipted(ctx, req.Target, args, bytes.NewReader(encoded))
}

func (d *SSHDistributor) Acknowledge(ctx context.Context, req AckRequest) (Receipt, error) {
	args := []string{
		"vrooli", "cloud-target", "credential", "acknowledge",
		"--deployment", req.Target.DeploymentID, "--binding", req.Binding.ID,
		"--version", strconv.FormatInt(req.Version, 10), "--consumer", req.Consumer,
		"--operation", req.OperationID, "--step", req.Step,
		"--fence", strconv.FormatUint(req.Target.Fence, 10), "--json",
	}
	return d.runReceipted(ctx, req.Target, args, nil)
}

func (d *SSHDistributor) Revoke(ctx context.Context, req RevokeRequest) (Receipt, error) {
	args := []string{
		"vrooli", "cloud-target", "credential", "revoke",
		"--deployment", req.Target.DeploymentID, "--binding", req.Binding.ID,
		"--version", strconv.FormatInt(req.Version, 10),
		"--operation", req.OperationID, "--step", req.Step,
		"--fence", strconv.FormatUint(req.Target.Fence, 10), "--json",
	}
	receipt, err := d.runReceipted(ctx, req.Target, args, nil)
	receipt.Limitations = append(receipt.Limitations, RevocationLimitation)
	return receipt, err
}

// RevocationLimitation is stated on every revocation receipt: control-plane
// revocation is not proof that a host which already revealed a value forgot it.
const RevocationLimitation = "revocation purges the target store and denies new distribution; it cannot prove that a previously revealed value was forgotten on a compromised host"

func (d *SSHDistributor) runReceipted(ctx context.Context, target Target, args []string, stdin *bytes.Reader) (Receipt, error) {
	var in io.Reader
	if stdin != nil {
		in = stdin
	}
	output, err := d.Runner.Run(ctx, d.Label, args, in)
	var exit *RemoteExitError
	if err != nil && !errors.As(err, &exit) {
		return Receipt{Transport: d.Transport()}, &UnreachableError{Target: target.NodeLabel(), Cause: err}
	}
	var parsed targetReceipt
	if len(bytes.TrimSpace(output)) > 0 {
		if decodeErr := json.Unmarshal(output, &parsed); decodeErr != nil {
			return Receipt{Transport: d.Transport()}, newError(CodeDistributionFailed, "the target verb printed no receipt").WithDetail("target", target.NodeLabel())
		}
	}
	receipt := Receipt{Transport: d.Transport(), Replayed: parsed.Replayed, Details: parsed.Receipt.Details}
	if parsed.Receipt.OperationID != "" {
		receipt.Ref = parsed.Receipt.OperationID + "/" + parsed.Receipt.Step
	}
	if parsed.Error != nil {
		if parsed.Error.Code == "credential_store_locked" {
			return receipt, StoreLocked(target.NodeLabel())
		}
		return receipt, newError(CodeDistributionFailed, parsed.Error.Message).WithDetail("target_code", parsed.Error.Code).WithDetail("target", target.NodeLabel())
	}
	if err != nil {
		return receipt, newError(CodeDistributionFailed, "the target verb failed").WithDetail("exit_code", exit.ExitCode).WithDetail("target", target.NodeLabel())
	}
	return receipt, nil
}

func (d *SSHDistributor) classify(target Target, err error) error {
	var exit *RemoteExitError
	if errors.As(err, &exit) {
		return newError(CodeDistributionFailed, "the target credential authority refused a metadata read").WithDetail("exit_code", exit.ExitCode).WithDetail("target", target.NodeLabel())
	}
	return &UnreachableError{Target: target.NodeLabel(), Cause: err}
}

// Grant is the metadata-only view of a Bridge credential grant.
type Grant struct {
	ID              string
	NodeID          string
	LogicalID       string
	Field           string
	Generation      int64
	AckedGeneration int64
	Revoked         bool
	ReceiptAccepted bool
	PurgeState      string
}

// GrantSpec creates or answers a grant.
type GrantSpec struct {
	NodeID    string
	LogicalID string
	Field     string
	Class     string
	Retention string
}

// GrantClient is the Bridge CredentialGrantService seam. Values pass only
// through AnswerSecret, whose transport is the Bridge sealed channel.
type GrantClient interface {
	ListGrants(ctx context.Context, nodeID string) ([]Grant, error)
	CreateGrant(ctx context.Context, spec GrantSpec) (Grant, error)
	AnswerSecret(ctx context.Context, spec GrantSpec, value string) (Grant, error)
	RevokeGrant(ctx context.Context, id string) (Grant, error)
}

// CredentialInjection is the metadata-only dispatch mapping.
type CredentialInjection struct {
	LogicalID string
	Field     string
	EnvName   string
}

// DispatchJob is a typed Bridge job: argv only, never a shell string.
type DispatchJob struct {
	NodeID     string
	Scenario   string
	Verb       string
	Args       []string
	Injections []CredentialInjection
}

// Dispatcher queues a job on a node through Bridge.
type Dispatcher interface {
	Dispatch(ctx context.Context, job DispatchJob) (runID string, err error)
}

// BridgeDistributor distributes through Bridge grants and dispatch. The value
// enters the Bridge sealed channel through AnswerSecret and is never placed in
// a dispatch argument; the dispatched ingest verb receipts the grant.
type BridgeDistributor struct {
	Grants   GrantClient
	Dispatch Dispatcher
	Probe_   func(ctx context.Context, target Target, binding domain.CredentialBinding) (ProbeResult, error)
}

func (d *BridgeDistributor) Transport() string { return identity.TransportBridge }

// CheckGrant is the grant recheck run before distribution. A revoked grant
// denies new distribution with forbidden_revoked; no fallback exists.
func CheckGrant(ctx context.Context, grants GrantClient, nodeID string, descriptor domain.CredentialDescriptor) (*Grant, error) {
	list, err := grants.ListGrants(ctx, nodeID)
	if err != nil {
		return nil, &UnreachableError{Target: "node:" + nodeID, Cause: err}
	}
	var found *Grant
	for i := range list {
		g := list[i]
		if g.LogicalID != descriptor.LogicalID || g.Field != descriptor.Field {
			continue
		}
		if g.Revoked {
			return &g, forbiddenRevoked(g.ID, descriptor)
		}
		if found == nil || g.Generation > found.Generation {
			found = &g
		}
	}
	return found, nil
}

// forbiddenRevoked is the typed refusal for distribution under a revoked
// grant (P13-A04). There is no fallback transport.
func forbiddenRevoked(grantID string, descriptor domain.CredentialDescriptor) *apierrors.Error {
	return apierrors.New(apierrors.CodeForbiddenRevoked, "the credential grant for this descriptor was revoked; new distribution is denied").
		WithDetail("grant_id", grantID).WithDetail("descriptor", descriptor.Address()).
		WithNextAction(apierrors.NextAction{Owner: "vrooli-bridge", Kind: "command", Reference: "vrooli-bridge credentials grant", Label: "Issue a new grant if access is still intended"})
}

func (d *BridgeDistributor) Probe(ctx context.Context, target Target, binding domain.CredentialBinding) (ProbeResult, error) {
	if d.Probe_ != nil {
		return d.Probe_(ctx, target, binding)
	}
	grant, err := CheckGrant(ctx, d.Grants, target.Ref.NodeID, binding.Descriptor)
	if err != nil {
		return ProbeResult{}, err
	}
	return ProbeResult{Configured: grant != nil && grant.ReceiptAccepted, StoreUnlocked: true, StoreState: "bridge"}, nil
}

func (d *BridgeDistributor) Deliver(ctx context.Context, req DeliverRequest) (Receipt, error) {
	nodeID := req.Target.Ref.NodeID
	if strings.TrimSpace(nodeID) == "" {
		return Receipt{}, newError(CodeDistributionFailed, "the bridge transport needs a node id on the target binding")
	}
	if _, err := CheckGrant(ctx, d.Grants, nodeID, req.Binding.Descriptor); err != nil {
		return Receipt{Transport: d.Transport()}, err
	}
	spec := GrantSpec{NodeID: nodeID, LogicalID: req.Binding.Descriptor.LogicalID, Field: req.Binding.Descriptor.Field, Class: string(req.Binding.Class), Retention: "durable"}
	grant, err := d.Grants.AnswerSecret(ctx, spec, req.Value)
	if err != nil {
		return Receipt{Transport: d.Transport()}, &UnreachableError{Target: "node:" + nodeID, Cause: err}
	}
	// The dispatched argv carries metadata only; the value reaches the verb
	// through the Bridge credential injection as IngestEnvName.
	args := []string{
		"credential", "ingest", "--deployment", req.Target.DeploymentID, "--binding", req.Binding.ID,
		"--logical-id", req.Binding.Descriptor.LogicalID, "--field", req.Binding.Descriptor.Field,
		"--version", strconv.FormatInt(req.Version.Number, 10), "--content-ref", req.Version.ContentRef,
		"--operation", req.OperationID, "--step", req.Step,
		"--fence", strconv.FormatUint(req.Target.Fence, 10), "--grant", grant.ID, "--from-env", IngestEnvName, "--json",
	}
	runID, err := d.Dispatch.Dispatch(ctx, DispatchJob{
		NodeID: nodeID, Scenario: "vrooli", Verb: "cloud-target", Args: args,
		Injections: []CredentialInjection{{LogicalID: req.Binding.Descriptor.LogicalID, Field: req.Binding.Descriptor.Field, EnvName: IngestEnvName}},
	})
	if err != nil {
		return Receipt{Transport: d.Transport(), Details: map[string]any{"grant_id": grant.ID}}, &UnreachableError{Target: "node:" + nodeID, Cause: err}
	}
	return Receipt{Transport: d.Transport(), Ref: runID, Details: map[string]any{"grant_id": grant.ID, "generation": grant.Generation}}, nil
}

func (d *BridgeDistributor) Acknowledge(ctx context.Context, req AckRequest) (Receipt, error) {
	nodeID := req.Target.Ref.NodeID
	args := []string{
		"credential", "acknowledge", "--deployment", req.Target.DeploymentID, "--binding", req.Binding.ID,
		"--version", strconv.FormatInt(req.Version, 10), "--consumer", req.Consumer,
		"--operation", req.OperationID, "--step", req.Step,
		"--fence", strconv.FormatUint(req.Target.Fence, 10), "--json",
	}
	runID, err := d.Dispatch.Dispatch(ctx, DispatchJob{NodeID: nodeID, Scenario: "vrooli", Verb: "cloud-target", Args: args})
	if err != nil {
		return Receipt{Transport: d.Transport()}, &UnreachableError{Target: "node:" + nodeID, Cause: err}
	}
	return Receipt{Transport: d.Transport(), Ref: runID, Details: map[string]any{"consumer": req.Consumer, "version": req.Version}}, nil
}

func (d *BridgeDistributor) Revoke(ctx context.Context, req RevokeRequest) (Receipt, error) {
	nodeID := req.Target.Ref.NodeID
	receipt := Receipt{Transport: d.Transport(), Limitations: []string{RevocationLimitation}}
	if req.Binding.GrantRef != "" {
		grant, err := d.Grants.RevokeGrant(ctx, req.Binding.GrantRef)
		if err != nil {
			return receipt, &UnreachableError{Target: "node:" + nodeID, Cause: err}
		}
		receipt.Details = map[string]any{"grant_id": grant.ID, "purge_state": grant.PurgeState}
	}
	args := []string{
		"credential", "revoke", "--deployment", req.Target.DeploymentID, "--binding", req.Binding.ID,
		"--version", strconv.FormatInt(req.Version, 10), "--operation", req.OperationID, "--step", req.Step,
		"--fence", strconv.FormatUint(req.Target.Fence, 10), "--json",
	}
	runID, err := d.Dispatch.Dispatch(ctx, DispatchJob{NodeID: nodeID, Scenario: "vrooli", Verb: "cloud-target", Args: args})
	if err != nil {
		return receipt, &UnreachableError{Target: "node:" + nodeID, Cause: err}
	}
	receipt.Ref = runID
	return receipt, nil
}
