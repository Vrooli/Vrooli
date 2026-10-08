// finite_publication_test.go checks read publication cannot create acceptance authority from incomplete setup.
package backlog

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
)

func TestFiniteAcceptancePublicationIncompleteRefusesBeforeListener(t *testing.T) {
	if plan, e := PrepareFiniteReadPublication(effortauthority.Installation{}, nil, nil, nil); e == nil || plan != nil {
		t.Fatal("incomplete read publication accepted")
	}
	var plan *FiniteReadPublication
	if publication, e := plan.Publish(context.Background(), nil); e == nil || publication != nil {
		t.Fatal("nil canonical owner published")
	}
}
