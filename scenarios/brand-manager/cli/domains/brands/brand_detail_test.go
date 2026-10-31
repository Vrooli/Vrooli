package brands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
)

// facetFlagNames are the typography and voice flags on brands create/update.
var facetFlagNames = []string{
	"heading-font", "body-font", "mono-font", "base-font-size",
	"tone", "voice-style", "voice-keywords",
}

// stubFlags builds a RunContext carrying only the named flag values. The
// context refuses a flag that is not declared in its schema, so the schema is
// built from the same list the manifest test checks.
func stubFlags(flags map[string]string) cliapp.RunContext {
	schema := cliapp.ArgSchema{}
	for _, name := range facetFlagNames {
		schema.Flags = append(schema.Flags, cliapp.Flag{Name: name})
	}
	return cliapptest.NewTestRunContext(cliapptest.TestRunContextOptions{Schema: schema, Flags: flags})
}

// `brands get` must show where a brand's identity actually lives. The one-line
// list summary carries only id, name and version, so a brand whose palette is
// inherited from its container style read as an empty record — which is how a
// fully configured product brand came to look "not yet defined".
func TestFormatBrandDetailShowsInheritedIdentityReferences(t *testing.T) {
	brand := &brandsv1.Brand{
		Id:               "2e797e31",
		Name:             "Aquila",
		Description:      "Web console product (star line)",
		Version:          5,
		Slug:             "aquila",
		MarkAssetId:      "4754a41d",
		ContainerStyleId: "631e6f93",
		ProductLineId:    "bba0b3ce",
		Identity:         &brandsv1.Identity{Tagline: "Browser-based terminal"},
		Colors:           &brandsv1.Colors{},
	}

	out := strings.Join(formatBrandDetail(brand), "\n")

	for _, want := range []string{"aquila", "631e6f93", "bba0b3ce", "4754a41d", "Browser-based terminal"} {
		if !strings.Contains(out, want) {
			t.Errorf("brand detail is missing %q:\n%s", want, out)
		}
	}
	// An unset palette must be reported as inherited, not silently omitted.
	if !strings.Contains(out, "inherited from the container style") {
		t.Errorf("unset colors should say where the palette comes from:\n%s", out)
	}
}

// A brand that declares its own colors should not claim to inherit them.
func TestFormatBrandDetailOmitsInheritanceHintWhenColorsSet(t *testing.T) {
	brand := &brandsv1.Brand{
		Id:               "abc",
		Name:             "Solo",
		ContainerStyleId: "style-1",
		Colors:           &brandsv1.Colors{Primary: "#15243c", Accent: "#22d3ee"},
	}
	out := strings.Join(formatBrandDetail(brand), "\n")
	if strings.Contains(out, "inherited from the container style") {
		t.Errorf("brand declares its own colors; it must not claim inheritance:\n%s", out)
	}
	// A declared palette must be shown, not merely acknowledged.
	for _, want := range []string{"#15243c", "#22d3ee"} {
		if !strings.Contains(out, want) {
			t.Errorf("declared color %s is missing from the detail view:\n%s", want, out)
		}
	}
}

// Typography and voice had no CLI surface at all: the proto carried both, but
// the only way to set them was `generation elements`, which produced fonts that
// contradicted the scenario's own design tokens and could not be corrected.
func TestTypographyAndVoiceFlagsBuildFacets(t *testing.T) {
	t.Run("typography", func(t *testing.T) {
		got := typographyFromFlags(stubFlags(map[string]string{
			"heading-font": "system-ui", "body-font": "system-ui",
			"mono-font": "ui-monospace", "base-font-size": "14px",
		}))
		if got == nil {
			t.Fatal("typography facet was not built from flags")
		}
		if got.MonoFont != "ui-monospace" || got.BaseFontSize != "14px" {
			t.Errorf("typography = %+v", got)
		}
	})

	t.Run("voice keywords split", func(t *testing.T) {
		got := voiceFromFlags(stubFlags(map[string]string{
			"tone": "Direct", "voice-keywords": "Precision, Reliability ,Insight",
		}))
		if got == nil {
			t.Fatal("voice facet was not built from flags")
		}
		if len(got.Keywords) != 3 || got.Keywords[1] != "Reliability" {
			t.Errorf("keywords = %#v", got.Keywords)
		}
	})

	t.Run("absent flags leave facets nil", func(t *testing.T) {
		if typographyFromFlags(stubFlags(nil)) != nil {
			t.Error("typography must stay nil so an update does not clear stored fonts")
		}
		if voiceFromFlags(stubFlags(nil)) != nil {
			t.Error("voice must stay nil so an update does not clear stored voice")
		}
	})
}

// The handler reads these flags, but the parser only accepts what the scenario
// manifest declares. A flag present in one and absent from the other is
// rejected as an unknown option even though the code supports it.
func TestFacetFlagsAreDeclaredInTheManifest(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var doc struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name  string `json:"name"`
				Flags []struct {
					Name string `json:"name"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	for _, group := range doc.Groups {
		if group.Name != "brands" {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != "update" && command.Name != "create" {
				continue
			}
			declared := map[string]bool{}
			for _, f := range command.Flags {
				declared[f.Name] = true
			}
			for _, name := range facetFlagNames {
				if !declared[name] {
					t.Errorf("brands %s: --%s is read by the handler but not declared in cli/manifest.json, so the parser rejects it", command.Name, name)
				}
			}
		}
	}
}
