package main

import (
	"strings"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/logx"
)

func planEnv(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func NewPlanStore(plansPath string) *commerce.PlanStore {
	return commerce.NewPlanStoreWithOptions(commerce.PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  planEnv(envx.Get("BUNDLE_KEY"), "business_suite"),
		DisplayEnv: planEnv(envx.Get("BUNDLE_ENVIRONMENT"), "production"),
		Log:        logx.Info,
	})
}

func NewPlanStoreWithOptions(opts commerce.PlanStoreOptions) *commerce.PlanStore {
	if opts.Log == nil {
		opts.Log = logx.Info
	}
	return commerce.NewPlanStoreWithOptions(opts)
}

func NewPlanService(_ any) *commerce.PlanService {
	bundle := planEnv(envx.Get("BUNDLE_KEY"), "business_suite")
	env := planEnv(envx.Get("BUNDLE_ENVIRONMENT"), "production")
	plansPath := commerce.ResolvePlansPath()
	store := commerce.NewPlanStoreWithOptions(commerce.PlanStoreOptions{PlansPath: plansPath, BundleKey: bundle, DisplayEnv: env, Log: logx.Info})
	if err := store.LoadAll(); err != nil {
		logx.Error("plan_store_load_failed", map[string]interface{}{"error": err.Error(), "path": plansPath})
	}
	return commerce.NewPlanServiceWithOptions(commerce.PlanServiceOptions{PlanStore: store, DefaultBundle: bundle, DisplayEnv: env})
}

func NewPlanServiceWithPlanStore(store *commerce.PlanStore) *commerce.PlanService {
	return NewPlanServiceWithOptions(commerce.PlanServiceOptions{PlanStore: store})
}

func NewPlanServiceWithOptions(opts commerce.PlanServiceOptions) *commerce.PlanService {
	if opts.Log == nil {
		opts.Log = logx.Info
	}
	return commerce.NewPlanServiceWithOptions(opts)
}
