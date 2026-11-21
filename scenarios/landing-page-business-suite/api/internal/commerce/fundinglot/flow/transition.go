package flow

import "landing-page-business-suite-api/internal/commerce/fundinglot/flow/generated"

// TransitionFundingLot delegates lifecycle topology to the generated contract.
func TransitionFundingLot(status generated.FundingLotStatus, event generated.FundingLotEvent) (generated.FundingLotStatus, error) {
	return generated.TransitionFundingLotStatus(status, event)
}
