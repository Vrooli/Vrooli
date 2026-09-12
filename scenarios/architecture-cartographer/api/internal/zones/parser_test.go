package zones

import "testing"

const zoneMapDoc = `# Architecture

## Zone Map

| Zone | Declared Layer | Path Convention |
|---|---|---|
| transport | API transport edge | ` + "`api/handlers/<domain>/`" + ` |
| domain | Scenario core | ` + "`api/internal/<domain>/`" + ` |
| cli | Operator wrapper | ` + "`cli/domains/<domain>/`" + ` |

## Next Section

text
`

func TestParseDeclaredZoneMap(t *testing.T) {
	m := ParseDeclaredZoneMap(zoneMapDoc)
	if !m.Present || len(m.Zones) != 3 {
		t.Fatalf("expected 3 declared zones, got present=%v n=%d", m.Present, len(m.Zones))
	}
	zone, layer, ok := m.ZoneFor("api/handlers/orders")
	if !ok || zone != Transport || layer != "API transport edge" {
		t.Fatalf("ZoneFor transport = %q/%q ok=%v", zone, layer, ok)
	}
	if zone, _, ok := m.ZoneFor("cli/domains/orders/cmd.go"); !ok || zone != CLI {
		t.Fatalf("ZoneFor cli = %q ok=%v", zone, ok)
	}
	if _, _, ok := m.ZoneFor("vendor/x"); ok {
		t.Fatal("ZoneFor should not match unrelated path")
	}
}

func TestParseDeclaredZoneMap_Absent(t *testing.T) {
	if m := ParseDeclaredZoneMap("# Doc\n\n## Other\n\ntext\n"); m.Present {
		t.Fatal("absent Zone Map should not be Present")
	}
}

func TestConverge_AgreementValidates(t *testing.T) {
	m := ParseDeclaredZoneMap(zoneMapDoc)
	yes := true
	conv := Converge("api/handlers/orders", Info{Zone: Transport, Domain: "orders"}, m, &yes)
	if conv.Drift {
		t.Fatalf("declared==derived should not drift: %+v", conv)
	}
	if conv.DeclaredZone != Transport || conv.DeclaredLayer == "" {
		t.Fatalf("declared signal not captured: %+v", conv)
	}
	if conv.Confidence <= 0.6 {
		t.Fatalf("agreement + import-consistent should raise confidence, got %v", conv.Confidence)
	}
}

func TestConverge_CrossFamilyDrift(t *testing.T) {
	m := ParseDeclaredZoneMap(zoneMapDoc)
	// Code says this path is a domain, doc declares it transport — real drift.
	conv := Converge("api/handlers/orders", Info{Zone: Domain, Domain: "orders"}, m, nil)
	if !conv.Drift {
		t.Fatalf("cross-family declared/derived mismatch should drift: %+v", conv)
	}
}

func TestConverge_CoreFamilyNoDrift(t *testing.T) {
	doc := "# A\n\n## Zone Map\n\n| Zone | Declared Layer | Path Convention |\n|---|---|---|\n| domain | Scenario core | `api/internal/<domain>/` |\n"
	m := ParseDeclaredZoneMap(doc)
	// Declared domain, derived substrate — both core family, not drift.
	conv := Converge("api/internal/server", Info{Zone: Substrate}, m, nil)
	if conv.Drift {
		t.Fatalf("same-family (core) should not drift: %+v", conv)
	}
}

func TestConverge_DerivedOnly(t *testing.T) {
	conv := Converge("api/internal/graph", Info{Zone: Domain, Domain: "graph"}, DeclaredZoneMap{}, nil)
	if conv.Drift || conv.DeclaredZone != "" {
		t.Fatalf("no declared map should yield derived-only: %+v", conv)
	}
	if conv.Zone != Domain {
		t.Fatalf("resolved zone should be derived: %+v", conv)
	}
}
