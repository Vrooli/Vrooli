package capability_test

import (
	"encoding/json"
	"testing"

	"image-tools/internal/capability"
)

// TestPublishedCatalogMirrorsManifests proves the cross-scenario contract is a
// projection and not a second declaration set: every manifest appears exactly
// once, and the fields a workflow builder type-checks on come through unchanged.
func TestPublishedCatalogMirrorsManifests(t *testing.T) {
	published := capability.Published("")
	manifests := capability.All()
	if len(published) != len(manifests) {
		t.Fatalf("published %d operations, want %d", len(published), len(manifests))
	}
	byID := make(map[string]capability.Manifest, len(manifests))
	for _, m := range manifests {
		byID[m.ID] = m
	}
	for _, op := range published {
		if op.GetScenario() != capability.ScenarioSlug {
			t.Errorf("%q scenario = %q, want %q", op.GetId(), op.GetScenario(), capability.ScenarioSlug)
		}
		m, ok := byID[op.GetId()]
		if !ok {
			t.Errorf("published operation %q has no manifest", op.GetId())
			continue
		}
		if op.GetSummary() != m.Summary || op.GetFamily() != string(m.Family) || op.GetMechanism() != string(m.Mechanism) {
			t.Errorf("%q projection drifted from its manifest: %+v", m.ID, op)
		}
		if op.GetConsentWeight() != string(m.ConsentWeight) {
			t.Errorf("%q consent weight = %q, want %q", m.ID, op.GetConsentWeight(), m.ConsentWeight)
		}
		if op.GetOutput().GetKind() != m.IO.Output.Kind {
			t.Errorf("%q output kind = %q, want %q", m.ID, op.GetOutput().GetKind(), m.IO.Output.Kind)
		}
		if len(op.GetInputs()) != len(m.IO.Inputs) {
			t.Errorf("%q has %d published inputs, want %d", m.ID, len(op.GetInputs()), len(m.IO.Inputs))
		}
		if len(op.GetBackends()) != len(m.Backends) {
			t.Errorf("%q has %d published backends, want %d", m.ID, len(op.GetBackends()), len(m.Backends))
		}
		if got := op.GetRequires().GetProcess(); got != m.Requires.Process {
			t.Errorf("%q process requirement = %t, want %t", m.ID, got, m.Requires.Process)
		}
		if string(m.ParamsSchema) != "" {
			var want, got any
			if err := json.Unmarshal(m.ParamsSchema, &want); err != nil {
				t.Fatalf("%q manifest schema is invalid JSON: %v", m.ID, err)
			}
			if err := json.Unmarshal([]byte(op.GetParamsSchemaJson()), &got); err != nil {
				t.Errorf("%q published schema is invalid JSON: %v", m.ID, err)
				continue
			}
			if len(op.GetParamsSchemaJson()) == 0 {
				t.Errorf("%q dropped its params schema", m.ID)
			}
		}
	}
}

// TestPublishedCatalogFamilyFilter checks the filter a builder uses to ask for
// one axis, and that every family is represented (a typo in a manifest family
// would otherwise hide an operation from every filtered query).
func TestPublishedCatalogFamilyFilter(t *testing.T) {
	seen := make(map[string]int)
	for _, op := range capability.Published("") {
		seen[op.GetFamily()]++
	}
	for _, family := range []string{"generate", "edit", "enhance", "restore", "extract", "compose"} {
		if seen[family] == 0 {
			t.Errorf("no published operation in family %q", family)
		}
		filtered := capability.Published(family)
		if len(filtered) != seen[family] {
			t.Errorf("family %q returned %d operations, want %d", family, len(filtered), seen[family])
		}
		for _, op := range filtered {
			if op.GetFamily() != family {
				t.Errorf("family filter %q returned %q", family, op.GetId())
			}
		}
	}
}
