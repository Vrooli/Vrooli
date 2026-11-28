package monetization

import (
	"fmt"
	"strings"
)

// CatalogSnapshot is the minimal LPBS catalog projection needed to reconcile
// a scenario declaration without coupling the provider to SQL drivers.
type CatalogSnapshot struct {
	Bundles map[string]BundleSnapshot
}

type BundleSnapshot struct {
	Apps  map[string]bool
	Tiers map[string]map[string]bool
}

// ReconcileDeclaration verifies bundle identity, app registration, and every
// declared limit against every tier in the trusted LPBS catalog projection.
func ReconcileDeclaration(decl declaration, snapshot CatalogSnapshot, location string) []string {
	bundle, ok := snapshot.Bundles[decl.BundleKey]
	if !ok {
		return []string{fmt.Sprintf("money.unknown_bundle_key: %s", decl.BundleKey)}
	}
	var findings []string
	if !bundle.Apps[decl.AppKey] {
		findings = append(findings, fmt.Sprintf("money.unregistered_app_key: %s", decl.AppKey))
	}
	for _, meter := range decl.Meters {
		if len(bundle.Tiers) == 0 {
			findings = append(findings, fmt.Sprintf("money.meter_missing_tier_limits: %s", meter.LimitKey))
			continue
		}
		for tier, limits := range bundle.Tiers {
			if !limits[meter.LimitKey] {
				findings = append(findings, fmt.Sprintf("money.meter_missing_tier_limits: %s/%s", tier, meter.LimitKey))
			}
		}
	}
	_ = location
	return findings
}

func reconciliationFindings(decl declaration, snapshot CatalogSnapshot, location string) []*assessmentFindingAlias {
	results := ReconcileDeclaration(decl, snapshot, location)
	findings := make([]*assessmentFindingAlias, 0, len(results))
	for _, result := range results {
		parts := strings.SplitN(result, ": ", 2)
		message := result
		if len(parts) == 2 {
			message = parts[1]
		}
		findings = append(findings, &assessmentFindingAlias{Code: parts[0], Message: message, Location: location})
	}
	return findings
}

// assessmentFindingAlias keeps reconciliation independent of generated proto
// details; scan converts it at the provider boundary.
type assessmentFindingAlias struct {
	Code, Message, Location string
}
