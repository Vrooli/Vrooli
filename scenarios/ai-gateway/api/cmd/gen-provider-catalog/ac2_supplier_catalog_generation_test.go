package main

import "testing"

func TestAC2ProposalSupplierDeclarationsStayNonExecutable(t *testing.T) {
	base := supplierDeclaration{
		ID:       "fixture-router",
		Family:   "openai-compatible",
		Locality: "remote",
		Policy:   []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"proposed"}},"hosted_routes":[],"agreements":[]}}`),
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
	pending.Policy = []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"pending_review"}},"hosted_routes":[],"agreements":[]}}`)
	if err := validateSupplierDeclaration("fixture-router.json", pending); err != nil {
		t.Fatalf("pending-review declaration without endpoint rejected: %v", err)
	}

	whitespace := base
	whitespace.Endpoint = "   "
	if err := validateSupplierDeclaration("fixture-router.json", whitespace); err == nil {
		t.Fatal("whitespace-only supplier endpoint was accepted")
	}

	approved := base
	approved.Policy = []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"approved"}},"hosted_routes":[],"agreements":[]}}`)
	if err := validateSupplierDeclaration("fixture-router.json", approved); err == nil {
		t.Fatal("endpoint-less approved supplier declaration was accepted")
	}

	routed := base
	routed.Policy = []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"proposed"}},"hosted_routes":[{"id":"route"}],"agreements":[]}}`)
	if err := validateSupplierDeclaration("fixture-router.json", routed); err == nil {
		t.Fatal("endpoint-less supplier declaration with a hosted route was accepted")
	}

	for _, status := range []string{"proposed", "pending_review", "denied", "reopened", "expired", ""} {
		for _, artifact := range []string{`"hosted_routes":[{"id":"route"}],"agreements":[]`, `"hosted_routes":[],"agreements":[{"decision_ref":"fixture"}]`} {
			endpointed := base
			endpointed.Endpoint = "https://api.example.invalid/v1"
			endpointed.Policy = []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"` + status + `"}},` + artifact + `}}`)
			if err := validateSupplierDeclaration("fixture-router.json", endpointed); err == nil {
				t.Errorf("supplier with status %q accepted executable artifact %s", status, artifact)
			}
		}
	}

	mismatched := base
	mismatched.Policy = []byte(`{"provider":{"id":"other-router","terms":{"review":{"status":"proposed"}},"hosted_routes":[],"agreements":[]}}`)
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
		Policy:   []byte(`{"provider":{"id":"fixture-router","terms":{"review":{"status":"proposed"}},"hosted_routes":[],"agreements":[]},"models":{"fixture-model":{"default_eligible":false,"provenance":{"catalog":{"source":"https://example.invalid/models; exact fixture-model entry","observed_at":"2026-11-23"}}}}}`),
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
