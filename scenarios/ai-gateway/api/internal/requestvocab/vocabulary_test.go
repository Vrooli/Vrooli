package requestvocab

import (
	"testing"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestRequestKindCatalogCoversProtoAndLookup(t *testing.T) {
	if err := ValidateRequestKinds(); err != nil {
		t.Fatal(err)
	}
	for value, name := range sharedv1.RequestKind_name {
		kind := sharedv1.RequestKind(value)
		definition, ok := LookupRequestKind(kind)
		if kind == sharedv1.RequestKind_REQUEST_KIND_UNSPECIFIED {
			if ok {
				t.Fatalf("unspecified kind %s was accepted", name)
			}
			continue
		}
		if !ok || definition.Kind != kind || len(definition.RequiredInputs) == 0 || definition.OutputKind == "" || len(definition.RequiredCapabilities) == 0 {
			t.Errorf("proto kind %s has no executable catalog definition: %+v", name, definition)
		}
	}
	if _, ok := LookupRequestKind(sharedv1.RequestKind(999)); ok {
		t.Fatal("unknown request kind was accepted")
	}
}

func TestValidateRequestKindDefinitionBoundaries(t *testing.T) {
	valid := RequestKindDefinition{
		Kind:                 sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		RequiredInputs:       []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT},
		OutputKind:           "text",
		RequiredCapabilities: []string{"generate"},
	}
	tests := []struct {
		name string
		edit func(*RequestKindDefinition)
	}{
		{name: "unspecified kind", edit: func(d *RequestKindDefinition) { d.Kind = sharedv1.RequestKind_REQUEST_KIND_UNSPECIFIED }},
		{name: "missing inputs", edit: func(d *RequestKindDefinition) { d.RequiredInputs = nil }},
		{name: "unspecified input", edit: func(d *RequestKindDefinition) {
			d.RequiredInputs = []sharedv1.Modality{sharedv1.Modality_MODALITY_UNSPECIFIED}
		}},
		{name: "missing output", edit: func(d *RequestKindDefinition) { d.OutputKind = "" }},
		{name: "missing capabilities", edit: func(d *RequestKindDefinition) { d.RequiredCapabilities = nil }},
		{name: "blank capability", edit: func(d *RequestKindDefinition) { d.RequiredCapabilities = []string{" \t"} }},
		{name: "normalized duplicate capabilities", edit: func(d *RequestKindDefinition) { d.RequiredCapabilities = []string{"Generate", " generate "} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition := valid
			tt.edit(&definition)
			if err := validateRequestKindDefinition(definition); err == nil {
				t.Fatal("invalid request kind definition was accepted")
			}
		})
	}
	if err := validateRequestKindDefinition(valid); err != nil {
		t.Fatalf("valid request kind definition rejected: %v", err)
	}
}

func TestValidateRequestKindCoverageRequiresEveryProtoKind(t *testing.T) {
	if err := validateRequestKindCoverage(nil); err == nil {
		t.Fatal("empty catalog coverage was accepted")
	}
	seen := make(map[sharedv1.RequestKind]struct{})
	for _, definition := range RequestKindCatalog() {
		seen[definition.Kind] = struct{}{}
	}
	if err := validateRequestKindCoverage(seen); err != nil {
		t.Fatalf("complete catalog coverage rejected: %v", err)
	}
}

func TestRoleKindVocabulary(t *testing.T) {
	for input, want := range map[string]string{"": RoleKindTypedValue, RoleKindTypedValue: RoleKindTypedValue, RoleKindDecision: RoleKindDecision} {
		got, err := NormalizeRoleKind(input)
		if err != nil || got != want {
			t.Errorf("NormalizeRoleKind(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if _, err := NormalizeRoleKind("unknown"); err == nil {
		t.Fatal("unknown role kind was accepted")
	}
}

func TestKnownResourceCapabilityUsesRequestKindVocabulary(t *testing.T) {
	for _, capability := range []string{"generate", "embedding", "extract", "detect_pii", "decide", "image_output", "video_output"} {
		if !KnownResourceCapability(capability) {
			t.Errorf("known request capability %q was not recognized", capability)
		}
	}
	if KnownResourceCapability("resource_specific_custom_capability") {
		t.Fatal("resource-specific extension was incorrectly made a gateway capability")
	}
}
