package flow

import (
	"testing"

	"landing-page-business-suite-api/internal/commerce/fundinglot/flow/generated"
)

func TestFundingLotFormalReplay(t *testing.T) {
	generated.RunReplay(t, TransitionFundingLot)
}
