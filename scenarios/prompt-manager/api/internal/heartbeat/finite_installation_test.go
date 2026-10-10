package heartbeat

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
)

func TestReadinessDisabledPMInstallationHasNoProviderOrPublicationEffects(t *testing.T) {
	p, e := PrepareFiniteInstallation(context.Background(), effortauthority.Installation{}, nil, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	r := &FiniteLeaderRuntime{}
	if e = p.Apply(r); e != nil || r.Efforts != nil {
		t.Fatal("disabled publication", e)
	}
	if _, e = PrepareFiniteInstallation(context.Background(), effortauthority.Installation{Enabled: true}, nil, nil, nil); e == nil {
		t.Fatal("incomplete enabled startup")
	}
}
