package main

import "landing-page-business-suite-api/internal/commerce"

func NewAccountService(db commerce.Store, planService *commerce.PlanService) *commerce.Service {
	return newAccountService(db, planService)
}
