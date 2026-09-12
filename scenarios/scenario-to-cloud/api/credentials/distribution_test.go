package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// scriptedRunner answers argv by prefix and retains every call.
type scriptedRunner struct {
	responses map[string]string
	exit      map[string]int
	transport error
	calls     [][]string
	stdin     []string
}

func (r *scriptedRunner) Run(_ context.Context, _ string, args []string, stdin io.Reader) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if stdin != nil {
		data, _ := io.ReadAll(stdin)
		r.stdin = append(r.stdin, string(data))
	}
	if r.transport != nil {
		return nil, r.transport
	}
	joined := strings.Join(args, " ")
	for prefix, out := range r.responses {
		if strings.HasPrefix(joined, prefix) {
			if code := r.exit[prefix]; code != 0 {
				return []byte(out), &RemoteExitError{ExitCode: code, Stdout: out, Stderr: "refused"}
			}
			return []byte(out), nil
		}
	}
	return []byte(`{}`), nil
}

func sshFixture(t *testing.T, runner *scriptedRunner) (*SSHDistributor, Target, domain.CredentialBinding) {
	t.Helper()
	client, err := credentialclient.NewClient(credentialclient.ClientOptions{RemoteTarget: "root@203.0.113.10", RemoteRunner: runner})
	if err != nil {
		t.Fatal(err)
	}
	target := Target{DeploymentID: "dep-1", Ref: identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10", User: "root"}}, Fence: 7}
	descriptor := domain.CredentialDescriptor{LogicalID: "fixture/store", Field: "password"}
	binding := domain.CredentialBinding{ID: BindingID("dep-1", descriptor), DeploymentID: "dep-1", Descriptor: descriptor, Class: domain.CredentialClassGeneratedDatabasePassword, ConsumerRefs: []string{"scenario:app"}}
	return NewSSHDistributor(runner, client, "root@203.0.113.10"), target, binding
}

// [REQ:STC-P0-032] SECRET-01/05: on the SSH transport the value travels only
// inside the ingest payload on standard input; argv carries metadata, the
// fence and the receipt identity, and the target receipt is parsed back.
func TestSSHDistributorSendsValueOnStdinOnly(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]string{
		"vrooli cloud-target credential ingest": `{"receipt":{"operation_id":"op-1","step":"materialize-password-v1","outcome":"succeeded","details":{"version":1}},"replayed":false}`,
	}}
	distributor, target, binding := sshFixture(t, runner)
	const canary = "canary-secret-9f3a1c7e"
	receipt, err := distributor.Deliver(context.Background(), DeliverRequest{Target: target, Binding: binding, Version: domain.CredentialVersion{Number: 1, ContentRef: "cref_x"}, Value: canary, OperationID: "op-1", Step: "materialize-password-v1"})
	if err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if receipt.Ref != "op-1/materialize-password-v1" || receipt.Transport != identity.TransportSSH {
		t.Fatalf("receipt = %+v", receipt)
	}
	argv := strings.Join(runner.calls[0], " ")
	if strings.Contains(argv, canary) {
		t.Fatalf("value in argv: %s", argv)
	}
	for _, want := range []string{"vrooli cloud-target credential ingest", "--deployment dep-1", "--binding " + binding.ID, "--version 1", "--operation op-1", "--step materialize-password-v1", "--fence 7", "--json"} {
		if !strings.Contains(argv, want) {
			t.Fatalf("argv %q lacks %q", argv, want)
		}
	}
	var payload IngestPayload
	if err := json.Unmarshal([]byte(runner.stdin[0]), &payload); err != nil {
		t.Fatalf("stdin payload: %v", err)
	}
	if payload.Value != canary || payload.SchemaVersion != IngestSchemaVersion || payload.ContentRef != "cref_x" || payload.LogicalID != "fixture/store" || payload.Field != "password" {
		t.Fatalf("payload = %+v", payload)
	}
}

// [REQ:STC-P0-032] SECRET-07: a locked target store and an unreachable
// target are classified apart, before any value is sent.
func TestSSHDistributorClassifiesLockedAndUnreachable(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]string{
		"vrooli credentials store status": `{"initialized":true,"unlocked":false}`,
	}}
	distributor, target, binding := sshFixture(t, runner)
	if _, err := distributor.Probe(context.Background(), target, binding); !apierrors.Is(err, CodeStoreLocked) {
		t.Fatalf("locked probe = %v", err)
	}
	runner.responses["vrooli cloud-target credential ingest"] = `{"error":{"code":"credential_store_locked","message":"sealed"}}`
	runner.exit = map[string]int{"vrooli cloud-target credential ingest": 2}
	if _, err := distributor.Deliver(context.Background(), DeliverRequest{Target: target, Binding: binding, Version: domain.CredentialVersion{Number: 1}, Value: "v", OperationID: "op", Step: "s"}); !apierrors.Is(err, CodeStoreLocked) {
		t.Fatalf("locked ingest = %v", err)
	}
	runner.transport = errors.New("ssh: connect to host 203.0.113.10 port 22: connection timed out")
	_, err := distributor.Revoke(context.Background(), RevokeRequest{Target: target, Binding: binding, Version: 1, OperationID: "op", Step: "revoke"})
	if !IsUnreachable(err) {
		t.Fatalf("unreachable revoke = %v", err)
	}
	if _, err := distributor.Probe(context.Background(), target, binding); !IsUnreachable(err) {
		t.Fatalf("unreachable probe = %v", err)
	}
	if _, err := distributor.Acknowledge(context.Background(), AckRequest{Target: target, Binding: binding, Version: 1, Consumer: "scenario:app", OperationID: "op", Step: "ack"}); !IsUnreachable(err) {
		t.Fatalf("unreachable ack = %v", err)
	}
}

// [REQ:STC-P0-032] A target verb that ran and refused is a distribution
// failure carrying the target's code, not an unreachable target; a replayed
// receipt is reported as such.
func TestSSHDistributorSurfacesTargetRefusalsAndReplays(t *testing.T) {
	runner := &scriptedRunner{
		responses: map[string]string{"vrooli cloud-target credential revoke": `{"receipt":{"operation_id":"op","step":"revoke","outcome":"failed"},"error":{"code":"fence_stale","message":"fence 7 is below 9"}}`},
		exit:      map[string]int{"vrooli cloud-target credential revoke": 2},
	}
	distributor, target, binding := sshFixture(t, runner)
	_, err := distributor.Revoke(context.Background(), RevokeRequest{Target: target, Binding: binding, Version: 1, OperationID: "op", Step: "revoke"})
	if !apierrors.Is(err, CodeDistributionFailed) || IsUnreachable(err) {
		t.Fatalf("refused revoke = %v", err)
	}
	if apierrors.As(err).Details["target_code"] != "fence_stale" {
		t.Fatalf("target code not surfaced: %v", apierrors.As(err).Details)
	}
	runner.responses["vrooli cloud-target credential acknowledge"] = `{"receipt":{"operation_id":"op","step":"ack","outcome":"succeeded","details":{"consumer":"scenario:app"}},"replayed":true}`
	receipt, err := distributor.Acknowledge(context.Background(), AckRequest{Target: target, Binding: binding, Version: 1, Consumer: "scenario:app", OperationID: "op", Step: "ack"})
	if err != nil || !receipt.Replayed || receipt.Details["consumer"] != "scenario:app" {
		t.Fatalf("replayed ack = %+v, %v", receipt, err)
	}
}

type fakeGrants struct {
	grants   []Grant
	answered []GrantSpec
	revoked  []string
	err      error
}

func (g *fakeGrants) ListGrants(context.Context, string) ([]Grant, error) { return g.grants, g.err }
func (g *fakeGrants) CreateGrant(_ context.Context, spec GrantSpec) (Grant, error) {
	return Grant{ID: "grant-new", NodeID: spec.NodeID, LogicalID: spec.LogicalID, Field: spec.Field}, nil
}

func (g *fakeGrants) AnswerSecret(_ context.Context, spec GrantSpec, _ string) (Grant, error) {
	g.answered = append(g.answered, spec)
	return Grant{ID: "grant-1", NodeID: spec.NodeID, LogicalID: spec.LogicalID, Field: spec.Field, Generation: 2}, nil
}

func (g *fakeGrants) RevokeGrant(_ context.Context, id string) (Grant, error) {
	g.revoked = append(g.revoked, id)
	return Grant{ID: id, Revoked: true, PurgeState: "requested"}, nil
}

type fakeDispatch struct {
	jobs []DispatchJob
	err  error
}

func (d *fakeDispatch) Dispatch(_ context.Context, job DispatchJob) (string, error) {
	d.jobs = append(d.jobs, job)
	return "run-1", d.err
}

// [REQ:STC-P0-032] P13-A04 / SECRET-01: on the bridge transport the grant is
// rechecked before distribution, a revoked grant is forbidden_revoked with
// no fallback, and the dispatched argv names the injection variable, never
// the value.
func TestBridgeDistributorRechecksGrantsAndInjectsThroughEnv(t *testing.T) {
	descriptor := domain.CredentialDescriptor{LogicalID: "fixture/store", Field: "password"}
	binding := domain.CredentialBinding{ID: BindingID("dep-1", descriptor), DeploymentID: "dep-1", Descriptor: descriptor, Class: domain.CredentialClassGeneratedDatabasePassword, Target: domain.BundleSecretTarget{Type: "env", Name: "STORE_PASSWORD"}}
	target := Target{DeploymentID: "dep-1", Ref: identity.TargetRef{Transport: identity.TransportBridge, NodeID: "node-1"}, Fence: 2}
	grants := &fakeGrants{grants: []Grant{{ID: "grant-0", NodeID: "node-1", LogicalID: "fixture/store", Field: "password", Revoked: true}}}
	dispatch := &fakeDispatch{}
	distributor := &BridgeDistributor{Grants: grants, Dispatch: dispatch}
	const canary = "canary-secret-bridge-42"
	_, err := distributor.Deliver(context.Background(), DeliverRequest{Target: target, Binding: binding, Version: domain.CredentialVersion{Number: 1, ContentRef: "cref_b"}, Value: canary, OperationID: "op", Step: "s"})
	if !apierrors.Is(err, apierrors.CodeForbiddenRevoked) {
		t.Fatalf("revoked grant = %v", err)
	}
	if len(grants.answered) != 0 || len(dispatch.jobs) != 0 {
		t.Fatal("a revoked grant still distributed")
	}
	grants.grants = nil
	receipt, err := distributor.Deliver(context.Background(), DeliverRequest{Target: target, Binding: binding, Version: domain.CredentialVersion{Number: 1, ContentRef: "cref_b"}, Value: canary, OperationID: "op", Step: "s"})
	if err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if receipt.Ref != "run-1" || receipt.Details["grant_id"] != "grant-1" {
		t.Fatalf("receipt = %+v", receipt)
	}
	job := dispatch.jobs[0]
	argv := strings.Join(job.Args, " ")
	if strings.Contains(argv, canary) {
		t.Fatalf("value in dispatched argv: %s", argv)
	}
	if !strings.Contains(argv, "--from-env "+IngestEnvName) || !strings.Contains(argv, "--grant grant-1") || !strings.Contains(argv, "--content-ref cref_b") {
		t.Fatalf("argv = %s", argv)
	}
	if len(job.Injections) != 1 || job.Injections[0].EnvName != IngestEnvName || job.Injections[0].LogicalID != "fixture/store" {
		t.Fatalf("injections = %+v", job.Injections)
	}
	if job.Scenario != "vrooli" || job.Verb != "cloud-target" {
		t.Fatalf("job = %+v", job)
	}
	binding.GrantRef = "grant-1"
	revoke, err := distributor.Revoke(context.Background(), RevokeRequest{Target: target, Binding: binding, Version: 1, OperationID: "op", Step: "revoke"})
	if err != nil || len(grants.revoked) != 1 || revoke.Details["purge_state"] != "requested" || len(revoke.Limitations) == 0 {
		t.Fatalf("bridge revoke = %+v, %v (revoked %v)", revoke, err, grants.revoked)
	}
	grants.err = errors.New("bridge unavailable")
	if _, err := distributor.Probe(context.Background(), target, binding); !IsUnreachable(err) {
		t.Fatalf("bridge outage = %v", err)
	}
}
