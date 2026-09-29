package structuredresult

import (
	"encoding/json"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

func TestReviewCandidateVerdictContract(t *testing.T) {
	original := &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{"type":"object","properties":{"accepted":{"type":"boolean"},"candidateSha256":{"type":"string"},"reason":{"type":"string","minLength":1}},"required":["accepted","candidateSha256","reason"],"additionalProperties":false}`)}
	digest := strings.Repeat("a", 64)
	bound, err := BindReviewCandidate(original, digest)
	if err != nil {
		t.Fatal(err)
	}
	for _, accepted := range []bool{false, true} {
		value, _ := json.Marshal(map[string]any{"accepted": accepted, "candidateSha256": digest, "reason": "reviewed"})
		if err := ValidateValue(bound.Schema, value); err != nil {
			t.Fatalf("valid verdict: %v", err)
		}
		resolved := (&Resolver{}).Resolve(t.Context(), bound, selected(string(value)))
		if resolved.Status != domain.StructuredResultSuccess || resolved.SchemaDigest != bound.SchemaDigest {
			t.Fatalf("valid verdict lost bound receipt: %+v", resolved)
		}
	}
	for _, value := range []string{
		`{"accepted":true,"candidateSha256":"other","reason":"reviewed"}`,
		`{"accepted":true,"reason":"reviewed"}`,
		`{"accepted":"true","candidateSha256":"` + digest + `","reason":"reviewed"}`,
		`{"accepted":true,"candidateSha256":"` + digest + `","reason":""}`,
		`{"accepted":true,"candidateSha256":"` + digest + `","reason":"reviewed","unexpected":1}`,
	} {
		if ValidateValue(bound.Schema, json.RawMessage(value)) == nil {
			t.Fatalf("invalid verdict accepted: %s", value)
		}
		if resolved := (&Resolver{}).Resolve(t.Context(), bound, selected(value)); resolved.Status == domain.StructuredResultSuccess {
			t.Fatalf("resolver published invalid verdict: %+v", resolved)
		}
	}
	again, err := BindReviewCandidate(bound, digest)
	if err != nil || bound.SchemaDigest != again.SchemaDigest {
		t.Fatalf("binding not idempotent: %+v %v", again, err)
	}
	if _, err := BindReviewCandidate(bound, strings.Repeat("b", 64)); err == nil {
		t.Fatal("bound candidate was replaced")
	}
	if strings.Contains(string(original.Schema), `"const"`) || original.SchemaDigest != "" {
		t.Fatal("authored schema mutated")
	}
	for _, digest := range []string{"", "a", strings.Repeat("A", 64), strings.Repeat("g", 64)} {
		if _, err := BindReviewCandidate(original, digest); err == nil {
			t.Fatal("invalid owner digest accepted", digest)
		}
	}
}

func TestReviewVerdictRequiresTypedDeterministicIdentity(t *testing.T) {
	for _, schema := range []string{
		`{"type":"object"}`,
		`{"type":"string"}`,
		`{"type":"object","properties":{"accepted":{"type":"boolean"},"candidateSha256":{"type":"string"}},"required":["accepted"]}`,
		`{"type":"object","properties":{"accepted":{"type":"string"},"candidateSha256":{"type":"string"}},"required":["accepted","candidateSha256"]}`,
	} {
		if _, err := NormalizeReviewSpec(&domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(schema)}); err == nil {
			t.Fatal("unbound verdict contract accepted", schema)
		}
	}
	if _, err := NormalizeReviewSpec(nil); err == nil {
		t.Fatal("missing verdict contract accepted")
	}
	if _, err := NormalizeReviewSpec(&domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, ExtractionMode: domain.StructuredExtractionConstrained, Schema: json.RawMessage(`{"type":"object"}`)}); err == nil {
		t.Fatal("a second model may infer the independent verdict")
	}
}
