package phaseadmission

import (
	"context"
	"testing"

	"test-genie/internal/orchestrator/phases"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

// outageRecoveryBroker models one unavailable owner while another phase can
// still acquire capacity. The owner ID is captured so recovery can prove it is
// retrying the same run/phase work identity.
type outageRecoveryBroker struct {
	deniedOwner string
	denied      bool
	owners      []string
}

func (b *outageRecoveryBroker) Acquire(_ context.Context, owner string, _, _ int64) (sharedcapacity.Lease, sharedcapacity.Verdict, error) {
	b.owners = append(b.owners, owner)
	if owner == b.deniedOwner && !b.denied {
		b.denied = true
		return nil, sharedcapacity.Verdict{Kind: "unknown", Reason: "capacity broker unavailable"}, nil
	}
	return stubLease{}, sharedcapacity.Verdict{Kind: "grant"}, nil
}

// A broker outage must leave the rejected phase unresolved, while independent
// work can still be admitted. Once capacity returns, retrying the same durable
// run/phase identity must obtain a grant rather than manufacture new work.
func TestCapacityOutageKeepsDeniedWorkUnknownAndRecoversSameIdentity(t *testing.T) {
	const runID = "maintenance-window-20261121"
	blocked := []phases.Definition{{Name: phases.Name("blocked"), Concurrency: phases.Concurrency{Mode: "exclusive"}}}
	independent := []phases.Definition{{Name: phases.Name("independent"), Concurrency: phases.Concurrency{Mode: "exclusive"}}}
	blockedOwner := sharedcapacity.OwnerIDFor("test-genie", runID, "blocked")
	broker := &outageRecoveryBroker{deniedOwner: blockedOwner}
	deps := Deps{Broker: broker, Estimator: schedulerCostStub{}}

	leases, reason, admitted, _ := Admit(context.Background(), deps, "demo", runID, blocked)
	if len(leases) != 0 || admitted != 1 || reason != "capacity broker unavailable" {
		t.Fatalf("outage admission = leases:%d admitted:%d reason:%q; want unresolved unknown capacity", len(leases), admitted, reason)
	}
	independentLeases, independentReason, independentAdmitted, _ := Admit(context.Background(), deps, "demo", runID, independent)
	if len(independentLeases) != 1 || independentReason != "" || independentAdmitted != 1 {
		t.Fatalf("independent admission = leases:%d admitted:%d reason:%q; want work to proceed", len(independentLeases), independentAdmitted, independentReason)
	}
	_ = independentLeases[0].Release(context.Background())

	recovered, recoveredReason, recoveredAdmitted, _ := Admit(context.Background(), deps, "demo", runID, blocked)
	if len(recovered) != 1 || recoveredReason != "" || recoveredAdmitted != 1 {
		t.Fatalf("recovery admission = leases:%d admitted:%d reason:%q; want same work to be granted", len(recovered), recoveredAdmitted, recoveredReason)
	}
	_ = recovered[0].Release(context.Background())
	if len(broker.owners) != 3 || broker.owners[0] != blockedOwner || broker.owners[1] == blockedOwner || broker.owners[2] != blockedOwner {
		t.Fatalf("broker owner sequence = %v; want blocked identity, independent identity, same blocked identity", broker.owners)
	}
}

type schedulerCostStub struct {
	// unsizable names phases the estimator has no reliable size for. Empty
	// means every phase is sizable.
	unsizable map[string]bool
}

func (s schedulerCostStub) PhaseCostEstimate(_ context.Context, _, phase string) (int64, int64, bool) {
	if s.unsizable[phase] {
		return 0, 0, false
	}
	return 64, 100, true
}

// schedulerBrokerStub grants the first grantCount acquisitions and then answers
// with kind/reason, so a case can describe a host that fills up mid-batch.
type schedulerBrokerStub struct {
	kind       string
	reason     string
	grantCount int
	granted    int
	acquires   int
	released   *int
}

func (s *schedulerBrokerStub) Acquire(context.Context, string, int64, int64) (sharedcapacity.Lease, sharedcapacity.Verdict, error) {
	s.acquires++
	if s.granted < s.grantCount {
		s.granted++
		if s.released != nil {
			return trackedLease{released: s.released}, sharedcapacity.Verdict{Kind: "grant"}, nil
		}
		return stubLease{}, sharedcapacity.Verdict{Kind: "grant"}, nil
	}
	return nil, sharedcapacity.Verdict{Kind: s.kind, Reason: s.reason}, nil
}

type stubLease struct{}

func (stubLease) Release(context.Context) error { return nil }

func TestAdmitPhaseBatchDenialFallsBackWithReason(t *testing.T) {
	deps := Deps{Broker: &schedulerBrokerStub{kind: "deny", reason: "insufficient ram"}, Estimator: schedulerCostStub{}}
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	leases, reason, admitted, _ := Admit(context.Background(), deps, "demo", "run-1", defs)
	if admitted != 1 {
		t.Fatalf("admitted = %d, want a serial fallback of 1", admitted)
	}
	if len(leases) != 0 {
		t.Fatalf("held %d leases after falling back to serial", len(leases))
	}
	if reason != "insufficient ram" {
		t.Fatalf("fallback reason = %q, want broker reason", reason)
	}
}

func TestAdmitPhaseBatchUsesConservativeReservationWhenEstimateMissing(t *testing.T) {
	deps := Deps{Broker: &schedulerBrokerStub{kind: "grant", grantCount: 2}, Estimator: schedulerCostStub{unsizable: map[string]bool{"unknown": true}}}
	defs := []phases.Definition{
		{Name: phases.Name("unknown"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("known"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	leases, reason, admitted, fallback := Admit(context.Background(), deps, "demo", "run-1", defs)
	if admitted != 2 || len(leases) != 2 || reason != "" {
		t.Fatalf("estimated admission = leases:%d reason:%q admitted:%d, want two granted leases without serial fallback", len(leases), reason, admitted)
	}
	if fallback != 1 {
		t.Fatalf("fallback admissions = %d, want one", fallback)
	}
}

func TestAdmitPhaseBatchReportsNoFallbackWhenEstimatesAreReliable(t *testing.T) {
	deps := Deps{Broker: &schedulerBrokerStub{grantCount: 2}, Estimator: schedulerCostStub{}}
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	_, _, admitted, fallback := Admit(context.Background(), deps, "demo", "run-1", defs)
	if admitted != 2 || fallback != 0 {
		t.Fatalf("admitted=%d fallback=%d, want two admitted and no fallback admissions", admitted, fallback)
	}
}

// A host that fills up part-way through a batch should run what fits. Releasing
// the grants already held and dropping to serial throws away admitted capacity
// and makes the run pay for the denial twice.
func TestAdmitPhaseBatchRunsTheGrantedPrefix(t *testing.T) {
	deps := Deps{Broker: &schedulerBrokerStub{kind: "queue", reason: "host full", grantCount: 3}, Estimator: schedulerCostStub{}}
	defs := make([]phases.Definition, 0, 5)
	for _, name := range []string{"one", "two", "three", "four", "five"} {
		defs = append(defs, phases.Definition{Name: phases.Name(name), Concurrency: phases.Concurrency{Mode: "parallel-safe"}})
	}

	leases, reason, admitted, _ := Admit(context.Background(), deps, "demo", "run-1", defs)
	if admitted != 3 {
		t.Fatalf("admitted = %d, want the three granted phases", admitted)
	}
	if len(leases) != 3 {
		t.Fatalf("held %d leases, want one per admitted phase", len(leases))
	}
	if reason != "host full" {
		t.Fatalf("reason = %q, want the broker's stopping reason recorded", reason)
	}
}

// The whole batch fitting is the ordinary case and must not be reported as a
// partial admission.
func TestAdmitPhaseBatchAdmitsFullBatch(t *testing.T) {
	deps := Deps{Broker: &schedulerBrokerStub{grantCount: 10}, Estimator: schedulerCostStub{}}
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	_, reason, admitted, _ := Admit(context.Background(), deps, "demo", "run-1", defs)
	if admitted != len(defs) || reason != "" {
		t.Fatalf("admitted = %d reason = %q, want the full batch with no reason", admitted, reason)
	}
}

func TestAdmitPhaseBatchClaimsSoloPhase(t *testing.T) {
	deps := Deps{Broker: &schedulerBrokerStub{grantCount: 1}, Estimator: schedulerCostStub{}}
	defs := []phases.Definition{{Name: phases.Name("expensive"), Concurrency: phases.Concurrency{Mode: "exclusive"}}}

	leases, reason, admitted, fallback := Admit(context.Background(), deps, "demo", "run-1", defs)
	if len(leases) != 1 || reason != "" || admitted != 1 || fallback != 0 {
		t.Fatalf("solo admission = leases:%d reason:%q admitted:%d fallback:%d, want one claim", len(leases), reason, admitted, fallback)
	}
}

func TestAdmitPhaseBatchDeniedSoloPhaseStillMakesProgress(t *testing.T) {
	broker := &schedulerBrokerStub{kind: "deny", reason: "host full"}
	defs := []phases.Definition{{Name: phases.Name("expensive"), Concurrency: phases.Concurrency{Mode: "exclusive"}}}

	leases, reason, admitted, fallback := Admit(context.Background(), Deps{Broker: broker, Estimator: schedulerCostStub{}}, "demo", "run-1", defs)
	if len(leases) != 0 || reason != "host full" || admitted != 1 || fallback != 0 {
		t.Fatalf("denied solo admission = leases:%d reason:%q admitted:%d fallback:%d, want progress prefix 1", len(leases), reason, admitted, fallback)
	}
	if broker.acquires != 1 {
		t.Fatalf("broker acquisitions = %d, want one claim attempt", broker.acquires)
	}
}

func TestAdmitPhaseBatchCallerReleasesSoloPhaseClaim(t *testing.T) {
	released := 0
	broker := &schedulerBrokerStub{grantCount: 1, released: &released}
	defs := []phases.Definition{{Name: phases.Name("expensive"), Concurrency: phases.Concurrency{Mode: "exclusive"}}}

	leases, _, _, _ := Admit(context.Background(), Deps{Broker: broker, Estimator: schedulerCostStub{}}, "demo", "run-1", defs)
	if len(leases) != 1 {
		t.Fatalf("solo leases = %d, want one", len(leases))
	}
	if err := leases[0].Release(context.Background()); err != nil {
		t.Fatalf("release solo claim: %v", err)
	}
	if released != 1 {
		t.Fatalf("released claims = %d, want one", released)
	}
}

type trackedLease struct {
	released *int
}

func (l trackedLease) Release(context.Context) error {
	*l.released++
	return nil
}
