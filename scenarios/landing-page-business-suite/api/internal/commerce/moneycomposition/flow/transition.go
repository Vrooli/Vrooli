package flow

import "landing-page-business-suite-api/internal/commerce/moneycomposition/flow/generated"

func TransitionMoneyComposition(status generated.MoneyStatus, event generated.MoneyEvent) (generated.MoneyStatus, error) {
	return generated.TransitionMoneyStatus(status, event)
}
