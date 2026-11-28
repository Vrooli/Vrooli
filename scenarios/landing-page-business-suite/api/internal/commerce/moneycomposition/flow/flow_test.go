package flow

import (
	"testing"

	"landing-page-business-suite-api/internal/commerce/moneycomposition/flow/generated"
)

func TestMoneyCompositionFormalReplay(t *testing.T) {
	generated.RunReplay(t, TransitionMoneyComposition)
}
