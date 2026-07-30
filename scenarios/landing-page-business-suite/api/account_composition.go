package main

import (
	"strconv"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"
)

func newAccountService(db commerce.Store, planService *commerce.PlanService) *commerce.Service {
	return commerce.NewService(db, planService, commerce.Runtime{
		CacheTTL: accountCacheTTL(), NormalizeEmail: NormalizeEmail, Log: logStructured,
	})
}

func accountCacheTTL() time.Duration {
	const defaultTTL = 60 * time.Second
	value := strings.TrimSpace(envx.Get("SUBSCRIPTION_CACHE_TTL_SECONDS"))
	if value == "" {
		return defaultTTL
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return defaultTTL
	}
	return time.Duration(seconds) * time.Second
}
