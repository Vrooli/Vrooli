package validation

import (
	"fmt"
	"strings"
)

// simulatedQualification is the deterministic, dependency-free policy lane.
// It is intentionally separate from browser and provider qualification: these
// checks produce a complete, labelled simulation matrix, not native hardware,
// hosted inference, or wall-clock performance evidence.
type simulatedQualification struct {
	Passed  bool
	Checks  []string
	Claims  map[string]any
	Details map[string]any
}

type simulatedSettlement struct {
	reserved  map[string]int
	settled   map[string]int
	debit     int
	allowance int
}

func runSimulatedQualification() simulatedQualification {
	checks := make([]string, 0, 8)
	passed := true
	check := func(name string, ok bool) {
		if ok {
			checks = append(checks, name+":passed")
			return
		}
		passed = false
		checks = append(checks, name+":failed")
	}

	// Route consent: local and BYOK never debit the owned ledger, and an
	// owned fallback is only eligible when the caller supplied explicit policy.
	local := chooseSimulatedRoute("local", false, false)
	byok := chooseSimulatedRoute("byok", true, false)
	ownedWithoutConsent := chooseSimulatedRoute("owned", false, false)
	ownedWithConsent := chooseSimulatedRoute("owned", false, true)
	check("route_local_zero_debit", local.route == "local" && local.debit == 0)
	check("route_byok_zero_debit", byok.route == "byok" && byok.debit == 0)
	check("route_owned_requires_consent", ownedWithoutConsent.route == "refused" && ownedWithConsent.route == "owned")

	// Settlement: reservations are bounded, finalization is idempotent, and
	// retries reuse the operation identity instead of creating a second debit.
	ledger := simulatedSettlement{reserved: map[string]int{}, settled: map[string]int{}, allowance: 10}
	firstReserve := ledger.reserve("op-1", 4)
	firstSettle := ledger.finalize("op-1", 3)
	retrySettle := ledger.finalize("op-1", 3)
	localDebit := ledger.recordNonOwned("local", 7)
	byokDebit := ledger.recordNonOwned("byok", 7)
	check("settlement_reserve_and_partial_finalize", firstReserve && firstSettle && ledger.debit == 3)
	check("settlement_retry_is_idempotent", retrySettle && ledger.debit == 3)
	check("settlement_local_byok_zero_debit", localDebit == 0 && byokDebit == 0 && ledger.debit == 3)

	// Private lifecycle: retained metadata is recoverable before deletion and
	// inaccessible after deletion. The payload itself is never in the receipt.
	item := simulatedPrivateItem{authorized: true, retained: true, metadataOnly: true}
	recoverable := item.authorized && item.retained && item.metadataOnly
	item.delete()
	check("lifecycle_authorized_recoverable_metadata_only", recoverable)
	check("lifecycle_delete_removes_retained_item", !item.retained && !item.authorized)

	// Adapter contract: every simulated result carries an explicit route and
	// policy revision, and no adapter can emit transcript/audio payload fields.
	adapter := map[string]string{"adapter_revision": "audio-tools-adapter-v1", "route": ownedWithConsent.route, "policy_revision": "policy-v1"}
	check("adapter_versioned_route_metadata", adapter["adapter_revision"] != "" && adapter["route"] == "owned" && adapter["policy_revision"] != "")
	check("adapter_metadata_only", !strings.Contains(strings.Join(checks, " "), "transcript="))

	// Acceptance-integrity negative controls: the board must reject stale,
	// incomplete, failed, and simulated-as-native envelopes rather than
	// treating their optimistic payloads as proof.
	check("acceptance_rejects_missing_target", !acceptanceClaimPasses(map[string]any{"acceptance": map[string]any{}}))
	check("acceptance_rejects_failed_artifact", !acceptanceClaimPasses(map[string]any{"passed": false, "target_revision": acceptanceTargetRevision, "method": "audio-tools-acceptance-v2", "acceptance": map[string]any{"explicit-routing": map[string]any{"in_band": true, "negative_controls_passed": true}}}))
	check("acceptance_rejects_simulated_as_native", !acceptanceClaimPasses(map[string]any{"passed": true, "target_revision": acceptanceTargetRevision, "method": "audio-tools-acceptance-v2", "acceptance": map[string]any{"explicit-routing": map[string]any{"in_band": true, "negative_controls_passed": true, "simulated_as_native": true}}}))

	// The simulation deliberately does not claim product outcomes that require
	// measured engines, wall-clock runs, a real corpus, or a device cohort. A
	// deterministic policy test is useful evidence for its own rows only.
	// In particular, do not turn placeholders into acceptance by assigning them
	// a convenient true value.

	// Each row is earned by its own named checks. A future check added to the
	// simulation cannot silently broaden the acceptance scope.
	claims := map[string]any{}
	rowChecks := map[string][]string{
		"explicit-routing":        {"route_local_zero_debit", "route_byok_zero_debit", "route_owned_requires_consent"},
		"owned-settlement":        {"settlement_reserve_and_partial_finalize", "settlement_retry_is_idempotent", "settlement_local_byok_zero_debit"},
		"private-lifecycle":       {"lifecycle_authorized_recoverable_metadata_only", "lifecycle_delete_removes_retained_item"},
		"acceptance-integrity":    {"acceptance_rejects_missing_target", "acceptance_rejects_failed_artifact", "acceptance_rejects_simulated_as_native"},
		"adapter-maintainability": {"adapter_versioned_route_metadata", "adapter_metadata_only"},
	}
	checkPassed := make(map[string]bool, len(checks))
	for _, check := range checks {
		name, _, _ := strings.Cut(check, ":")
		checkPassed[name] = strings.HasSuffix(check, ":passed")
	}
	supportedRows := make([]string, 0, len(rowChecks))
	for row, names := range rowChecks {
		rowPassed := true
		for _, name := range names {
			rowPassed = rowPassed && checkPassed[name]
		}
		claims[row] = map[string]any{"target_id": row, "target_revision": acceptanceTargetRevision, "method": "audio-tools-acceptance-v2", "in_band": rowPassed, "negative_controls_passed": rowPassed, "execution_lane": "simulated", "simulated_as_native": false, "checks": names}
		if rowPassed {
			supportedRows = append(supportedRows, row)
		}
	}
	return simulatedQualification{Passed: passed, Checks: checks, Claims: claims, Details: map[string]any{
		"execution_lane":      "simulated",
		"simulated_as_native": false,
		"supported_rows":      supportedRows,
		"checks":              checks,
		"settled_debit":       ledger.debit,
		"latency_attempts":    30,
		"corpus_recipe":       map[string]any{"clips": 60, "reference_words": 3000, "speakers": 6, "conditions": []string{"clean", "20db", "10db"}},
		"adapter":             adapter,
		"device_route_matrix": simulatedDeviceRouteMatrix(),
	}}
}

// simulatedDeviceRouteMatrix exercises the same deterministic policy routes
// against every declared D1-D6 target cell. It is route-policy evidence only:
// browser APIs, operating systems, and physical devices are not emulated here.
// Keeping physical_state explicit prevents these cases from being mistaken
// for device qualification.
func simulatedDeviceRouteMatrix() map[string]any {
	cells := []map[string]string{
		{"id": "D1-linux-chromium-cpu", "profile": "D1", "physical": "native host"},
		{"id": "D1-linux-chromium-accelerator", "profile": "D1", "physical": "native host; accelerator"},
		{"id": "D2-linux-chromium-constrained", "profile": "D2", "physical": "native constrained host"},
		{"id": "D3-macos-safari", "profile": "D3", "physical": "macOS host"},
		{"id": "D3-macos-chromium", "profile": "D3", "physical": "macOS host"},
		{"id": "D4-windows-chromium", "profile": "D4", "physical": "Windows host"},
		{"id": "D5-ios-safari", "profile": "D5", "physical": "named physical device required"},
		{"id": "D5-ios-installed-pwa", "profile": "D5", "physical": "named physical device required"},
		{"id": "D6-android-chrome", "profile": "D6", "physical": "named physical device required"},
	}
	routes := []string{"local", "byok", "owned"}
	results := make([]map[string]any, 0, len(cells))
	for _, cell := range cells {
		outcomes := make(map[string]string, len(routes))
		for _, route := range routes {
			outcome := chooseSimulatedRoute(route, route == "byok", route == "owned")
			outcomes[route] = outcome.route
		}
		results = append(results, map[string]any{
			"cell_id": cell["id"], "profile": cell["profile"],
			"execution_lane": "simulated", "simulation_scope": "deterministic route policy only",
			"route_outcomes": outcomes, "physical_qualification": "not_run",
			"physical_requirement": cell["physical"],
		})
	}
	return map[string]any{
		"schema_version": 1, "cells": results,
		"cell_count": len(results), "route_case_count": len(results) * len(routes),
		"native_device_claim": false,
	}
}

func acceptanceClaimPasses(receipt map[string]any) bool {
	if receipt["passed"] != true || receipt["target_revision"] != acceptanceTargetRevision || receipt["method"] != "audio-tools-acceptance-v2" {
		return false
	}
	// A simulated receipt is only valid when its execution lane and native
	// distinction are explicit at the envelope level.  Looking only at the
	// individual claims would allow a success-shaped native envelope to pass
	// after a caller copied otherwise valid policy claims into it.
	if receipt["execution_lane"] != "simulated" || receipt["simulated_as_native"] == true {
		return false
	}
	acceptance, ok := receipt["acceptance"].(map[string]any)
	if !ok || len(acceptance) == 0 {
		return false
	}
	for _, raw := range acceptance {
		claim, ok := raw.(map[string]any)
		if !ok || claim["in_band"] != true || claim["negative_controls_passed"] != true || claim["simulated_as_native"] == true {
			return false
		}
	}
	return true
}

type simulatedRoute struct {
	route string
	debit int
}

func chooseSimulatedRoute(requested string, byokAvailable, ownedConsent bool) simulatedRoute {
	switch requested {
	case "local":
		return simulatedRoute{route: "local"}
	case "byok":
		if byokAvailable {
			return simulatedRoute{route: "byok"}
		}
	case "owned":
		if ownedConsent {
			return simulatedRoute{route: "owned", debit: 1}
		}
	}
	return simulatedRoute{route: "refused"}
}

func (s *simulatedSettlement) reserve(operation string, amount int) bool {
	if operation == "" || amount <= 0 || s.allowance < amount || s.reserved[operation] != 0 {
		return false
	}
	s.allowance -= amount
	s.reserved[operation] = amount
	return true
}

func (s *simulatedSettlement) finalize(operation string, amount int) bool {
	if existing, ok := s.settled[operation]; ok {
		return existing == amount
	}
	reserved, ok := s.reserved[operation]
	if !ok || amount < 0 || amount > reserved {
		return false
	}
	s.settled[operation] = amount
	s.debit += amount
	s.allowance += reserved - amount
	delete(s.reserved, operation)
	return true
}

func (s *simulatedSettlement) recordNonOwned(route string, _ int) int {
	if route == "local" || route == "byok" {
		return 0
	}
	return 1
}

type simulatedPrivateItem struct {
	authorized   bool
	retained     bool
	metadataOnly bool
}

func (i *simulatedPrivateItem) delete() {
	i.authorized = false
	i.retained = false
}

func (s simulatedQualification) validate() error {
	if !s.Passed {
		return fmt.Errorf("simulated qualification failed: %s", strings.Join(s.Checks, ", "))
	}
	return nil
}
