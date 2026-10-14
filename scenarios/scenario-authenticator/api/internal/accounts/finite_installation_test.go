package accounts

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
)

func TestReadinessDisabledIssuerInstallationHasNoProviderOrPublicationEffects(t *testing.T) {
	var s *Service
	p, e := s.PrepareFiniteInstallation(context.Background(), effortauthority.Installation{}, FiniteIssuerSetup{})
	if e != nil {
		t.Fatal(e)
	}
	h := p.Handlers()
	if h.Broker != nil || h.Consent != nil {
		t.Fatal("disabled handlers published")
	}
	if _, e = s.PrepareFiniteInstallation(context.Background(), effortauthority.Installation{Enabled: true}, FiniteIssuerSetup{}); e == nil {
		t.Fatal("incomplete enabled startup")
	}
}
