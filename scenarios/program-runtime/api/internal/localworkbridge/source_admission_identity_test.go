package localworkbridge

import (
	"context"
	"testing"
)

func TestSourceAdmissionIdentityComesFromPermit(t *testing.T) {
	g, _, _ := sourceFixture(t, SourceCollector)
	p, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", sourceRunBody()))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := p.AdmissionIdentity("prog_123")
	if err != nil {
		t.Fatal(err)
	}
	if owner.ID != "prog_123" || owner.Subject != g.policy.Subject || owner.Owner != g.policy.Owner || owner.BindingID != g.policy.BindingID || owner.BindingVersion != g.policy.BindingVersion || owner.ScopeSHA256 != g.digest || owner.RequestKey != p.call.RequestKey || owner.Digest != g.policy.CollectorDigest {
		t.Fatal("ownership was not derived from verified permit")
	}
	if err = p.CheckProgram(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	p.call.Operation = "local-work.collector.observe"
	if _, err = p.AdmissionIdentity("prog_123"); err == nil {
		t.Fatal("observe must not create ownership")
	}
	if _, err = (*SourcePermit)(nil).AdmissionIdentity("prog_123"); err == nil {
		t.Fatal("nil permit")
	}
}
