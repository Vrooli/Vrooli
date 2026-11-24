package main

import (
	"strings"
	"testing"
)

func ac2SupplierFixturePolicy(providerID, status string, routes, agreements, models string) []byte {
	return []byte(`{"provider":{"id":"` + providerID + `","terms":{"plan_rules":[{"plan":"fixture","use_classes":{"own_account":"allowed","in_app":"unverified"},"conditions":["verify exact scope before use"],"clause":"fixture clause","reason":"fixture reason","source_anchors":[{"supplier_id":"` + providerID + `","claim_ref":"fixture","url":"https://example.invalid/terms","anchor":"fixture clause","document_date":"not stated on fixture page","observed_at":"2026-11-23","retrieval_status":"read_public_page","supports":"fixture source supports only the stated rule","does_not_establish":"account entitlement remains unknown"}]}],"evidence":{"url":"https://example.invalid/terms","document_date":"not stated on fixture page","read_at":"2026-11-23"},"review":{"status":"` + status + `"}},"hosted_routes":` + routes + `,"agreements":` + agreements + `},"models":` + models + `,"research":{"activation":"denied_until_review","model_reason":"Upstream rights and exact model scope remain unknown.","source_scope":"public official pages only","evidence":["https://example.invalid/terms"]}}`)
}

func TestAC2ProposalSupplierDeclarationsStayNonExecutable(t *testing.T) {
	base := supplierDeclaration{
		ID:       "fixture-router",
		Family:   "openai-compatible",
		Locality: "remote",
		Policy:   ac2SupplierFixturePolicy("fixture-router", "proposed", "[]", "[]", "{}"),
	}
	if err := validateSupplierDeclaration("fixture-router.json", base); err != nil {
		t.Fatalf("proposal-only declaration without endpoint rejected: %v", err)
	}
	endpointed := base
	endpointed.Endpoint = "https://api.example.invalid/v1"
	if err := validateSupplierDeclaration("fixture-router.json", endpointed); err != nil {
		t.Fatalf("proposal-only declaration with documented endpoint rejected: %v", err)
	}
	pending := base
	pending.Policy = ac2SupplierFixturePolicy("fixture-router", "pending_review", "[]", "[]", "{}")
	if err := validateSupplierDeclaration("fixture-router.json", pending); err != nil {
		t.Fatalf("pending-review declaration without endpoint rejected: %v", err)
	}

	whitespace := base
	whitespace.Endpoint = "   "
	if err := validateSupplierDeclaration("fixture-router.json", whitespace); err == nil {
		t.Fatal("whitespace-only supplier endpoint was accepted")
	}

	missingIdentity := base
	missingIdentity.Policy = ac2SupplierFixturePolicy("fixture-router", "proposed", "[]", "[]", `{"fixture-model":{"default_eligible":false,"provenance":{"catalog":{"source":"https://example.invalid/models; exact fixture-model entry","observed_at":"2026-11-23"}}}}`)
	if err := validateSupplierDeclaration("fixture-router.json", missingIdentity); err == nil {
		t.Fatal("supplier model without exact dated identity provenance was accepted")
	}

	approved := base
	approved.Policy = ac2SupplierFixturePolicy("fixture-router", "approved", "[]", "[]", "{}")
	if err := validateSupplierDeclaration("fixture-router.json", approved); err == nil {
		t.Fatal("endpoint-less approved supplier declaration was accepted")
	}

	routed := base
	routed.Policy = ac2SupplierFixturePolicy("fixture-router", "proposed", `[{"id":"route"}]`, "[]", "{}")
	if err := validateSupplierDeclaration("fixture-router.json", routed); err == nil {
		t.Fatal("endpoint-less supplier declaration with a hosted route was accepted")
	}

	for _, status := range []string{"proposed", "pending_review", "denied", "reopened", "expired", ""} {
		for _, artifact := range []string{`"hosted_routes":[{"id":"route"}],"agreements":[]`, `"hosted_routes":[],"agreements":[{"decision_ref":"fixture"}]`} {
			endpointed := base
			endpointed.Endpoint = "https://api.example.invalid/v1"
			if artifact == `"hosted_routes":[{"id":"route"}],"agreements":[]` {
				endpointed.Policy = ac2SupplierFixturePolicy("fixture-router", status, `[{"id":"route"}]`, "[]", "{}")
			} else {
				endpointed.Policy = ac2SupplierFixturePolicy("fixture-router", status, "[]", `[{"decision_ref":"fixture"}]`, "{}")
			}
			if err := validateSupplierDeclaration("fixture-router.json", endpointed); err == nil {
				t.Errorf("supplier with status %q accepted executable artifact %s", status, artifact)
			}
		}
	}

	mismatched := base
	mismatched.Policy = ac2SupplierFixturePolicy("other-router", "proposed", "[]", "[]", "{}")
	if err := validateSupplierDeclaration("fixture-router.json", mismatched); err == nil {
		t.Fatal("supplier declaration with mismatched policy identity was accepted")
	}
}

func TestAC2ProposalModelDraftsRequireDisabledFlagAndDatedCatalogSource(t *testing.T) {
	base := supplierDeclaration{
		ID:       "fixture-router",
		Family:   "openai-compatible",
		Locality: "remote",
		Endpoint: "https://api.example.invalid/v1",
		Policy:   ac2SupplierFixturePolicy("fixture-router", "proposed", "[]", "[]", `{"fixture-model":{"default_eligible":false,"caveats":["upstream rights remain unknown"],"provenance":{"catalog":{"source":"https://example.invalid/models; exact fixture-model entry","observed_at":"2026-11-23"},"identity":{"source":"https://example.invalid/models; exact fixture-model identity","observed_at":"2026-11-23"},"terms":{"source":"https://example.invalid/terms; model-scope terms unresolved","observed_at":"2026-11-23","source_anchors":[{"supplier_id":"fixture-router","model_id":"fixture-model","url":"https://example.invalid/terms","anchor":"model-scope terms unresolved","document_date":"not stated on fixture page","observed_at":"2026-11-23","retrieval_status":"read_public_page","supports":"supplier source supports unresolved terms scope","does_not_establish":"model rights remain unknown"}]}}}}`),
	}
	if err := validateSupplierDeclaration("fixture-router.json", base); err != nil {
		t.Fatalf("sourced disabled proposal model rejected: %v", err)
	}

	cases := []struct {
		name string
		data string
	}{
		{name: "missing explicit disabled flag", data: `{"provenance":{"catalog":{"source":"https://example.invalid/models","observed_at":"2026-11-23"}}}`},
		{name: "enabled proposal model", data: `{"default_eligible":true,"provenance":{"catalog":{"source":"https://example.invalid/models","observed_at":"2026-11-23"}}}`},
		{name: "missing direct catalog source", data: `{"default_eligible":false,"provenance":{"catalog":{"source":"catalog snapshot","observed_at":"2026-11-23"}}}`},
		{name: "buried link is not direct source", data: `{"default_eligible":false,"provenance":{"catalog":{"source":"unverified page claimed at https://example.invalid/models","observed_at":"2026-11-23"}}}`},
		{name: "malformed HTTPS URL", data: `{"default_eligible":false,"provenance":{"catalog":{"source":"https://","observed_at":"2026-11-23"}}}`},
		{name: "missing catalog observation date", data: `{"default_eligible":false,"provenance":{"catalog":{"source":"https://example.invalid/models"}}}`},
		{name: "invalid catalog observation date", data: `{"default_eligible":false,"provenance":{"catalog":{"source":"https://example.invalid/models","observed_at":"today"}}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			declaration := base
			declaration.Policy = []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"proposed"}},"hosted_routes":[],"agreements":[]},"models":{"fixture-model":` + tc.data + `}}`)
			if err := validateSupplierDeclaration("fixture-router.json", declaration); err == nil {
				t.Fatal("invalid proposal model was accepted")
			}
		})
	}
	approved := base
	approved.Policy = []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"approved"}},"hosted_routes":[],"agreements":[]},"models":{"fixture-model":{"default_eligible":false}}}`)
	if err := validateSupplierDeclaration("fixture-router.json", approved); err == nil {
		t.Fatal("model without dated catalog provenance was accepted after approval")
	}
}

func TestAC2ModelIdentityProvenanceRequiresExactDatedHTTPSSource(t *testing.T) {
	valid := []byte(`{"models":{"fixture-model":{"provenance":{"identity":{"source":"https://example.invalid/models/fixture-model; exact fixture-model ID","observed_at":"2026-11-23"}}}}}`)
	if err := validateExactModelIdentityProvenance("fixture-model-policy.json", valid); err != nil {
		t.Fatalf("exact dated model identity rejected: %v", err)
	}

	if err := validateExactModelIdentityProvenance("fixture-model-policy.json", []byte(`{"models":{}}`)); err == nil {
		t.Fatal("empty model policy was accepted")
	}

	cases := []struct {
		name     string
		identity string
	}{
		{name: "missing identity", identity: `{}`},
		{name: "mismatched ID", identity: `{"source":"https://example.invalid/models/other; different model ID","observed_at":"2026-11-23"}`},
		{name: "missing date", identity: `{"source":"https://example.invalid/models/fixture-model; exact fixture-model ID"}`},
		{name: "invalid date", identity: `{"source":"https://example.invalid/models/fixture-model; exact fixture-model ID","observed_at":"today"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := []byte(`{"models":{"fixture-model":{"provenance":{"identity":` + tc.identity + `}}}}`)
			if err := validateExactModelIdentityProvenance("fixture-model-policy.json", policy); err == nil {
				t.Fatal("invalid exact-ID identity provenance was accepted")
			}
		})
	}
}

func TestAC2SupplierTermsAndModelScopeRequireDatedSourcesAndExplicitUnknowns(t *testing.T) {
	valid := ac2SupplierFixturePolicy("fixture-router", "proposed", "[]", "[]", `{"fixture-model":{"caveats":["upstream provider and rights remain unknown"],"provenance":{"terms":{"source":"https://example.invalid/terms; model scope unresolved","observed_at":"2026-11-23","source_anchors":[{"supplier_id":"fixture-router","model_id":"fixture-model","url":"https://example.invalid/terms","anchor":"model scope unresolved","document_date":"not stated on fixture page","observed_at":"2026-11-23","retrieval_status":"read_public_page","supports":"fixture source supports the unresolved scope statement","does_not_establish":"model rights remain unknown"}]}}}}`)
	if err := validateSupplierSourceAndModelScope("fixture-router.json", valid); err != nil {
		t.Fatalf("complete sourced terms/scope fixture rejected: %v", err)
	}

	cases := []struct{ name, before, after string }{
		{"missing terms URL", `"url":"https://example.invalid/terms"`, `"url":""`},
		{"missing document-date statement", `"document_date":"not stated on fixture page"`, `"document_date":""`},
		{"ambiguous document-date statement", `"document_date":"not stated on fixture page"`, `"document_date":"recently revised"`},
		{"undated observation", `"read_at":"2026-11-23"`, `"read_at":"yesterday"`},
		{"missing rule clause", `"clause":"fixture clause"`, `"clause":""`},
		{"missing explicit uncertainty", `"model_reason":"Upstream rights and exact model scope remain unknown."`, `"model_reason":""`},
		{"asserted model rights without evidence", `"model_reason":"Upstream rights and exact model scope remain unknown."`, `"model_reason":"All rights are confirmed."`},
		{"invalid use-class decision", `"in_app":"unverified"`, `"in_app":"maybe"`},
		{"missing model terms source", `"source":"https://example.invalid/terms; model scope unresolved"`, `"source":"terms not checked"`},
		{"missing model terms date", `"observed_at":"2026-11-23"`, `"observed_at":"unknown"`},
		{"model caveat asserts resolved rights", `"upstream provider and rights remain unknown"`, `"upstream provider and rights confirmed"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := []byte(strings.Replace(string(valid), tc.before, tc.after, 1))
			if string(mutated) == string(valid) {
				t.Fatalf("mutation anchor not found: %s", tc.before)
			}
			if err := validateSupplierSourceAndModelScope("fixture-router.json", mutated); err == nil {
				t.Fatal("invalid source/scope record was accepted")
			}
		})
	}
	anchorMutations := []struct{ name, before, after string }{
		{"supplier mismatch", `"supplier_id":"fixture-router"`, `"supplier_id":"other-router"`},
		{"model mismatch", `"model_id":"fixture-model"`, `"model_id":"other-model"`},
		{"model source URL mismatch", `"url":"https://example.invalid/terms"`, `"url":"https://example.invalid/other"`},
		{"plan claim mismatch", `"claim_ref":"fixture"`, `"claim_ref":"other-plan"`},
		{"source no longer establishes claimed scope", `"supports":"fixture source supports the unresolved scope statement"`, `"supports":""`},
		{"overstated rights", `"does_not_establish":"model rights remain unknown"`, `"does_not_establish":"all model rights verified"`},
		{"ambiguous model source date", `"document_date":"not stated on fixture page"`, `"document_date":"recently reviewed"`},
	}
	for _, tc := range anchorMutations {
		t.Run(tc.name, func(t *testing.T) {
			mutated := []byte(strings.Replace(string(valid), tc.before, tc.after, 1))
			if string(mutated) == string(valid) {
				t.Fatalf("mutation anchor not found: %s", tc.before)
			}
			if err := validateSupplierSourceAndModelScope("fixture-router.json", mutated); err == nil {
				t.Fatal("mismatched or overconfident anchor was accepted")
			}
		})
	}
}

func TestAC2OpenRouterTermsAndModelScopeFailClosed(t *testing.T) {
	valid := []byte(`{"provider":{"terms":{"plan_rules":[{"clause":"Terms §7","reason":"Standard resale is prohibited.","use_classes":{"in_app":"denied"},"conditions":["separate agreement required"],"plan":"openrouter_standard","source_anchors":[{"supplier_id":"openrouter","claim_ref":"openrouter_standard","url":"https://openrouter.ai/terms","anchor":"Terms §7","document_date":"2026-08-31","observed_at":"2026-11-23","retrieval_status":"read_public_page","supports":"standard resale is prohibited","does_not_establish":"enterprise grant remains unknown"}]}],"evidence":{"url":"https://openrouter.ai/terms","document_date":"2026-08-31","read_at":"2026-11-23"}},"hosted_routes":[]},"models":{"fixture/model":{"provenance":{"terms":{"source":"https://openrouter.ai/terms; model-specific upstream terms remain unreviewed","observed_at":"2026-11-23","source_anchors":[{"supplier_id":"openrouter","model_id":"fixture/model","url":"https://openrouter.ai/terms","anchor":"Terms §§5.1–5.2","document_date":"2026-08-31","observed_at":"2026-11-23","retrieval_status":"read_public_page","supports":"separate model terms apply","does_not_establish":"exact model rights remain unknown"}]}},"caveats":["This model's upstream terms have not been evaluated."]}}}`)
	if err := validateOpenRouterModelTermsScope("model-policy.json", valid); err != nil {
		t.Fatalf("complete OpenRouter terms/model scope rejected: %v", err)
	}
	cases := []struct{ name, before, after string }{
		{"missing source date", `"document_date":"2026-08-31"`, `"document_date":""`},
		{"ambiguous source date", `"document_date":"2026-08-31"`, `"document_date":"recently revised"`},
		{"invalid read date", `"read_at":"2026-11-23"`, `"read_at":"today"`},
		{"missing model terms source", `"source":"https://openrouter.ai/terms; model-specific upstream terms remain unreviewed"`, `"source":"terms unchecked"`},
		{"missing model terms observation", `"observed_at":"2026-11-23"`, `"observed_at":"today"`},
		{"missing explicit model unknown", `"This model's upstream terms have not been evaluated."`, `"Model scope reviewed."`},
		{"unexpected hosted route", `"hosted_routes":[]`, `"hosted_routes":[{"id":"fixture"}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := []byte(strings.Replace(string(valid), tc.before, tc.after, 1))
			if string(mutated) == string(valid) {
				t.Fatalf("mutation anchor not found: %s", tc.before)
			}
			if err := validateOpenRouterModelTermsScope("model-policy.json", mutated); err == nil {
				t.Fatal("incomplete OpenRouter terms/model scope was accepted")
			}
		})
	}
}
