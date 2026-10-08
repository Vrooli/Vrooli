package doctemplates

import (
	"testing"

	"knowledge-observatory/internal/doccontract"
)

func TestInheritTemplateTableContracts(t *testing.T) {
	manifest := testManifestWithDomainsDoc()
	template := testManifestWithDomainsDoc()
	template.Sections[0].Documents[0].Validation.TableContracts = []doccontract.TableContract{{
		AnchorHeading: "Domain Inventory",
		Columns: []doccontract.TableColumnContract{{
			Name:     "Domain",
			Required: true,
			Type:     "text",
		}},
	}}

	inheritTemplateTableContracts(&manifest, &template)

	got := manifest.Sections[0].Documents[0].Validation.TableContracts
	if len(got) != 1 {
		t.Fatalf("table contracts = %d, want 1", len(got))
	}
	if got[0].AnchorHeading != "Domain Inventory" {
		t.Fatalf("anchor heading = %q", got[0].AnchorHeading)
	}
}

func TestInheritTemplateTableContracts_DoesNotOverrideScenarioContract(t *testing.T) {
	manifest := testManifestWithDomainsDoc()
	manifest.Sections[0].Documents[0].Validation.TableContracts = []doccontract.TableContract{{
		AnchorHeading: "Scenario Contract",
	}}
	template := testManifestWithDomainsDoc()
	template.Sections[0].Documents[0].Validation.TableContracts = []doccontract.TableContract{{
		AnchorHeading: "Template Contract",
	}}

	inheritTemplateTableContracts(&manifest, &template)

	got := manifest.Sections[0].Documents[0].Validation.TableContracts
	if len(got) != 1 || got[0].AnchorHeading != "Scenario Contract" {
		t.Fatalf("scenario contract overwritten: %#v", got)
	}
}

func testManifestWithDomainsDoc() doccontract.Manifest {
	return doccontract.Manifest{
		Sections: []doccontract.Section{{
			Documents: []doccontract.Document{{
				Path:    "concepts/DOMAINS.md",
				DocType: "domains",
			}},
		}},
	}
}
