package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"context"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"testing"
)

func TestReadinessDisabledAMInstallationHasNoProviderOrPublicationEffects(t *testing.T) {
	p, e := PrepareFiniteInstallation(context.Background(), effortauthority.Installation{}, nil, isolation.Manifest{}, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	o := &Orchestrator{}
	if e = p.Apply(o); e != nil || o.effortAuthority != nil || o.finiteNativeFactory != nil || p.ProfileHandler() != nil {
		t.Fatal("disabled publication", e)
	}
	if _, e = PrepareFiniteInstallation(context.Background(), effortauthority.Installation{Enabled: true}, nil, isolation.Manifest{}, nil, nil); e == nil {
		t.Fatal("incomplete enabled startup")
	}
}

func TestReadinessMissingContinuingWitnessRefusesBeforeRowOrLedger(t *testing.T) {
	o, a, p, task, profile, key, _ := nativeEffortFixture(t)
	factory, e := runner.NewFiniteNativeFactory(fixtureNativeManifest(p, profile))
	if e != nil {
		t.Fatal(e)
	}
	if factory.Enabled() || o.InstallFiniteNativeIsolation(factory) == nil {
		t.Fatal("planning-only factory installed")
	}
	o.finiteNativeFactory = factory // deliberately offered unready concrete, no public setter
	req := effortRequest(t, o, p, task, profile, key, "missing-witness")
	before, e := a.Store.Get(context.Background(), p.ID)
	if e != nil {
		t.Fatal(e)
	}
	if r, e := o.CreateRun(context.Background(), req); e == nil || r != nil {
		t.Fatal("missing witness admitted")
	}
	after, e := a.Store.Get(context.Background(), p.ID)
	if e != nil || effortauthority.Digest(before) != effortauthority.Digest(after) {
		t.Fatal("missing witness changed ledger", e)
	}
	if r, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey); e != nil || r != nil {
		t.Fatal("missing witness created row", e)
	}
}
