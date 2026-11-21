package main

import (
	"strconv"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/logx"
)

func newAccountService(db commerce.Store, planService *commerce.PlanService, limits ...commerce.LimitsServicer) *commerce.Service {
	var limitsSvc commerce.LimitsServicer
	if len(limits) > 0 {
		limitsSvc = limits[0]
	}
	return commerce.NewService(db, planService, commerce.Runtime{
		CacheTTL: accountCacheTTL(), LeaseTTL: entitlementLeaseTTL(), LimitsService: limitsSvc, NormalizeEmail: NormalizeEmail, Log: logx.Info,
	})
}

func entitlementLeaseTTL() time.Duration {
	const defaultTTL = 7 * 24 * time.Hour
	value := strings.TrimSpace(envx.Get("LPBS_ENTITLEMENT_LEASE_TTL_SECONDS"))
	if value == "" {
		return defaultTTL
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return defaultTTL
	}
	return time.Duration(seconds) * time.Second
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
