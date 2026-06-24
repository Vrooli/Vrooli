package providers

import (
	"testing"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestValidateAttestation(t *testing.T) {
	cite := []*routingv1.Citation{{Locator: "api/foo.go:10", Kind: "code"}}
	cases := []struct {
		name string
		att  *routingv1.AttestedAnswer
		ok   bool
	}{
		{"nil", nil, false},
		{"empty claim", &routingv1.AttestedAnswer{Basis: routingv1.Basis_BASIS_ABSENT}, false},
		{"derived no citation", &routingv1.AttestedAnswer{Claim: "x", Basis: routingv1.Basis_BASIS_DERIVED}, false},
		{"derived with citation", &routingv1.AttestedAnswer{Claim: "x", Basis: routingv1.Basis_BASIS_DERIVED, Citations: cite}, true},
		{"validated no citation", &routingv1.AttestedAnswer{Claim: "x", Basis: routingv1.Basis_BASIS_VALIDATED}, false},
		{"contradicted no citation", &routingv1.AttestedAnswer{Claim: "x", Basis: routingv1.Basis_BASIS_CONTRADICTED}, false},
		{"declared_unverified uncited ok", &routingv1.AttestedAnswer{Claim: "x", Basis: routingv1.Basis_BASIS_DECLARED_UNVERIFIED}, true},
		{"absent uncited ok", &routingv1.AttestedAnswer{Claim: "x", Basis: routingv1.Basis_BASIS_ABSENT}, true},
	}
	for _, c := range cases {
		err := ValidateAttestation(c.att)
		if (err == nil) != c.ok {
			t.Errorf("%s: ok=%v err=%v", c.name, c.ok, err)
		}
	}
}

func TestDecodeAttestationViaMapResults(t *testing.T) {
	desc := &registryv1.ProviderDescriptor{
		ProviderId: "meta-optimization-manager",
		Type:       "readiness",
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath:      "results",
			IdField:          "id",
			TitleField:       "title",
			AttestationField: "attestation",
		},
	}
	body := []byte(`{"results":[
		{"id":"1","title":"readiness","attestation":{
			"claim":"Answer projection is 8% NOW",
			"basis":"derived","sufficiency":"partial",
			"citations":[{"locator":"search-hub providers list","kind":"runtime"}],
			"gaps":["G5 ecosystem sparse"],
			"suggested_follow_ups":["meta-optimization-manager focus"]}},
		{"id":"2","title":"bad","attestation":{
			"claim":"uncited derived","basis":"derived"}}
	]}`)

	hits, err := MapResults(desc, body)
	if err != nil {
		t.Fatalf("MapResults: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits = %d", len(hits))
	}
	a := hits[0].GetAttestation()
	if a == nil {
		t.Fatal("hit0 attestation nil")
	}
	if a.GetBasis() != routingv1.Basis_BASIS_DERIVED || a.GetSufficiency() != routingv1.Sufficiency_SUFFICIENCY_PARTIAL {
		t.Errorf("hit0 basis/suff = %v/%v", a.GetBasis(), a.GetSufficiency())
	}
	if len(a.GetCitations()) != 1 || a.GetCitations()[0].GetLocator() != "search-hub providers list" {
		t.Errorf("hit0 citations = %+v", a.GetCitations())
	}
	if len(a.GetGaps()) != 1 || len(a.GetSuggestedFollowUps()) != 1 {
		t.Errorf("hit0 gaps/follow-ups = %+v / %+v", a.GetGaps(), a.GetSuggestedFollowUps())
	}
	// hit1 is a non-conformant DERIVED-without-citations attestation: dropped.
	if hits[1].GetAttestation() != nil {
		t.Errorf("hit1 attestation should be dropped, got %+v", hits[1].GetAttestation())
	}
}

func TestMeasureProvidersUnaffected(t *testing.T) {
	// A provider with no attestation_field never gets an attestation.
	desc := &registryv1.ProviderDescriptor{
		ProviderId:    "code-reference",
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title"},
	}
	hits, err := MapResults(desc, []byte(`{"results":[{"id":"1","title":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if hits[0].GetAttestation() != nil {
		t.Errorf("unexpected attestation: %+v", hits[0].GetAttestation())
	}
}
