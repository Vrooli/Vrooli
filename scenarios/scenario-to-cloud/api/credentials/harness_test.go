package credentials

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
)

// fakeTarget is an in-memory deployment target: it holds one value per
// binding, records every delivered version, and can be made unreachable.
type fakeTarget struct {
	mu          sync.Mutex
	values      map[string]string // binding id -> current value
	versions    map[string]int64
	revoked     map[string][]int64
	acks        map[string][]string // binding id -> consumers acked
	unreachable bool
	locked      bool
	revokedNode bool
	// unreachableConsumers makes acknowledge fail for the named consumers.
	unreachableConsumers map[string]bool
	// failDeliver injects a target-side verb failure.
	failDeliver bool
	// argv retains every argv the distributor sent; the canary scan reads it.
	argv [][]string
	// stdin retains every stdin payload; on a real transport this never lands
	// on disk, so it is recorded separately and NOT treated as a leak surface.
	stdin []string
}

func newFakeTarget() *fakeTarget {
	return &fakeTarget{values: map[string]string{}, versions: map[string]int64{}, revoked: map[string][]int64{}, acks: map[string][]string{}, unreachableConsumers: map[string]bool{}}
}

type fakeDistributor struct {
	target *fakeTarget
}

func (d *fakeDistributor) Transport() string { return identity.TransportSSH }

func (d *fakeDistributor) Probe(_ context.Context, target Target, binding domain.CredentialBinding) (ProbeResult, error) {
	t := d.target
	t.mu.Lock()
	defer t.mu.Unlock()
	t.argv = append(t.argv, []string{"vrooli", "credentials", "status", binding.Descriptor.LogicalID, binding.Descriptor.Field})
	if t.unreachable {
		return ProbeResult{}, &UnreachableError{Target: target.NodeLabel(), Cause: errors.New("dial tcp: i/o timeout")}
	}
	if t.locked {
		return ProbeResult{StoreState: "locked"}, StoreLocked(target.NodeLabel())
	}
	_, ok := t.values[binding.ID]
	return ProbeResult{Configured: ok, StoreUnlocked: true, StoreState: "available"}, nil
}

func (d *fakeDistributor) Deliver(_ context.Context, req DeliverRequest) (Receipt, error) {
	t := d.target
	t.mu.Lock()
	defer t.mu.Unlock()
	t.argv = append(t.argv, []string{"vrooli", "cloud-target", "credential", "ingest", "--binding", req.Binding.ID, "--version", fmt.Sprint(req.Version.Number), "--operation", req.OperationID, "--step", req.Step})
	t.stdin = append(t.stdin, req.Value)
	if t.unreachable {
		return Receipt{Transport: d.Transport()}, &UnreachableError{Target: req.Target.NodeLabel(), Cause: errors.New("dial tcp: i/o timeout")}
	}
	if t.locked {
		return Receipt{Transport: d.Transport()}, StoreLocked(req.Target.NodeLabel())
	}
	if t.revokedNode {
		return Receipt{Transport: d.Transport()}, forbiddenRevoked("grant-1", req.Binding.Descriptor)
	}
	if t.failDeliver {
		return Receipt{Transport: d.Transport()}, newError(CodeDistributionFailed, "the target verb failed")
	}
	t.values[req.Binding.ID] = req.Value
	t.versions[req.Binding.ID] = req.Version.Number
	return Receipt{Transport: d.Transport(), Ref: req.OperationID + "/" + req.Step, Details: map[string]any{"version": req.Version.Number}}, nil
}

func (d *fakeDistributor) Acknowledge(_ context.Context, req AckRequest) (Receipt, error) {
	t := d.target
	t.mu.Lock()
	defer t.mu.Unlock()
	t.argv = append(t.argv, []string{"vrooli", "cloud-target", "credential", "acknowledge", "--binding", req.Binding.ID, "--version", fmt.Sprint(req.Version), "--consumer", req.Consumer})
	if t.unreachable || t.unreachableConsumers[req.Consumer] {
		return Receipt{Transport: d.Transport()}, &UnreachableError{Target: req.Target.NodeLabel(), Cause: errors.New("consumer host unreachable")}
	}
	if t.versions[req.Binding.ID] != req.Version {
		return Receipt{Transport: d.Transport()}, newError(CodeVerificationFailed, "target holds a different version")
	}
	t.acks[req.Binding.ID] = append(t.acks[req.Binding.ID], req.Consumer)
	return Receipt{Transport: d.Transport(), Ref: req.OperationID + "/" + req.Step}, nil
}

func (d *fakeDistributor) Revoke(_ context.Context, req RevokeRequest) (Receipt, error) {
	t := d.target
	t.mu.Lock()
	defer t.mu.Unlock()
	t.argv = append(t.argv, []string{"vrooli", "cloud-target", "credential", "revoke", "--binding", req.Binding.ID, "--version", fmt.Sprint(req.Version), "--operation", req.OperationID, "--step", req.Step})
	receipt := Receipt{Transport: d.Transport(), Limitations: []string{RevocationLimitation}}
	if t.unreachable {
		return receipt, &UnreachableError{Target: req.Target.NodeLabel(), Cause: errors.New("dial tcp: i/o timeout")}
	}
	t.revoked[req.Binding.ID] = append(t.revoked[req.Binding.ID], req.Version)
	if t.versions[req.Binding.ID] == req.Version {
		delete(t.values, req.Binding.ID)
	}
	receipt.Ref = req.OperationID + "/" + req.Step
	return receipt, nil
}

// fakeRecovery is an in-memory recovery bundle.
type fakeRecovery struct {
	covers    []domain.CredentialDescriptor
	restoreTo *fakeTarget
	bindings  []domain.CredentialBinding
	locked    bool
	seen      []string
}

func (r *fakeRecovery) Verify(_ context.Context, bundleRef, passphrase string) ([]domain.CredentialDescriptor, error) {
	r.seen = append(r.seen, "verify:"+bundleRef)
	if r.locked {
		return nil, StoreLocked("replacement")
	}
	if passphrase == "" {
		return nil, errors.New("passphrase required")
	}
	return r.covers, nil
}

func (r *fakeRecovery) Restore(_ context.Context, bundleRef, passphrase string) error {
	r.seen = append(r.seen, "restore:"+bundleRef)
	if r.locked {
		return StoreLocked("replacement")
	}
	r.restoreTo.mu.Lock()
	defer r.restoreTo.mu.Unlock()
	for _, b := range r.bindings {
		r.restoreTo.values[b.ID] = "restored"
		r.restoreTo.versions[b.ID] = b.Version.Number
	}
	return nil
}

type fakeRestarter struct{ restarted [][]string }

func (r *fakeRestarter) Restart(_ context.Context, _ Target, consumers []string) error {
	r.restarted = append(r.restarted, append([]string(nil), consumers...))
	return nil
}

type harness struct {
	t         *testing.T
	store     *MemoryStore
	target    *fakeTarget
	svc       *Service
	logs      *bytes.Buffer
	restarter *fakeRestarter
	clock     time.Time
	ids       int
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, store: NewMemoryStore(), target: newFakeTarget(), logs: &bytes.Buffer{}, restarter: &fakeRestarter{}, clock: time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)}
	h.svc = &Service{
		Store:       h.store,
		Distributor: &fakeDistributor{target: h.target},
		Providers:   DefaultProviders(argvRecorder{target: h.target}, "fixture"),
		Restarter:   h.restarter,
		Logger:      log.New(h.logs, "", 0),
		Now:         func() time.Time { h.clock = h.clock.Add(time.Second); return h.clock },
		NewID:       func() string { h.ids++; return fmt.Sprintf("op-%d", h.ids) },
	}
	return h
}

// argvRecorder is the ArgvRunner the database provider uses; it retains argv
// and stdin the same way the target does.
type argvRecorder struct{ target *fakeTarget }

func (r argvRecorder) Run(_ context.Context, _ string, args []string, stdin io.Reader) ([]byte, error) {
	r.target.mu.Lock()
	defer r.target.mu.Unlock()
	r.target.argv = append(r.target.argv, append([]string(nil), args...))
	if stdin != nil {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(stdin)
		r.target.stdin = append(r.target.stdin, buf.String())
	}
	return []byte("ALTER ROLE\n"), nil
}

func (h *harness) targetRef() Target {
	return Target{DeploymentID: "dep-1", Ref: identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10", User: "root"}}, Fence: 3}
}

func fixturePlans() []domain.BundleSecretPlan {
	return []domain.BundleSecretPlan{
		{ID: "store-password", Class: domain.SecretClassPerInstallGenerated, Required: true, Target: domain.BundleSecretTarget{Type: "env", Name: "STORE_PASSWORD"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/store", Field: "password"}},
		{ID: "mailer-token", Class: domain.SecretClassUserPrompt, Required: true, Target: domain.BundleSecretTarget{Type: "env", Name: "MAILER_TOKEN"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/mailer", Field: "api-token"}},
		{ID: "session-signing", Class: domain.SecretClassPerInstallGenerated, Required: true, Target: domain.BundleSecretTarget{Type: "env", Name: "SESSION_SIGNING_KEY"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/app", Field: "signing-key"}},
	}
}

func fixtureConsumers() map[string][]string {
	return map[string][]string{
		"fixture/store:password":   {"resource:store", "scenario:app"},
		"fixture/mailer:api-token": {"scenario:app"},
		"fixture/app:signing-key":  {"scenario:app"},
	}
}

func (h *harness) plan() []domain.CredentialBinding {
	h.t.Helper()
	bindings, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: fixturePlans(), Consumers: fixtureConsumers(), DatabaseResources: map[string]bool{"resource:store": true}, Now: h.clock})
	if err != nil {
		h.t.Fatalf("plan bindings: %v", err)
	}
	return bindings
}

func (h *harness) bindingFor(field string) domain.CredentialBinding {
	h.t.Helper()
	for _, b := range h.plan() {
		if b.Descriptor.Field == field {
			stored, err := h.store.GetBinding(context.Background(), "dep-1", b.ID)
			if err != nil {
				h.t.Fatal(err)
			}
			if stored != nil {
				return *stored
			}
			return b
		}
	}
	h.t.Fatalf("no binding for field %s", field)
	return domain.CredentialBinding{}
}

func (h *harness) materialize(operator map[string]string) *MaterializeResult {
	h.t.Helper()
	result, err := h.svc.Materialize(context.Background(), MaterializeRequest{Target: h.targetRef(), Bindings: h.plan(), OperatorValues: operator})
	if err != nil {
		h.t.Fatalf("materialize: %v", err)
	}
	return result
}

func (h *harness) mustState(r *domain.CredentialRotation, want domain.CredentialRotationState) {
	h.t.Helper()
	if r == nil {
		h.t.Fatalf("rotation is nil, want state %s", want)
	}
	if r.State != want {
		h.t.Fatalf("rotation %s state = %s, want %s (receipts: %s)", r.ID, r.State, want, receiptSummary(r))
	}
}

func receiptSummary(r *domain.CredentialRotation) string {
	parts := make([]string, 0, len(r.Receipts))
	for _, receipt := range r.Receipts {
		parts = append(parts, receipt.Step+"="+receipt.Outcome)
	}
	return strings.Join(parts, ",")
}

func (h *harness) argvJoined() string {
	h.target.mu.Lock()
	defer h.target.mu.Unlock()
	lines := make([]string, 0, len(h.target.argv))
	for _, argv := range h.target.argv {
		lines = append(lines, strings.Join(argv, " "))
	}
	return strings.Join(lines, "\n")
}
