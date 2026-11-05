package gates

import (
	"fmt"
)

func ValidateFixtures(scope Scope) (Result, error) {
	root := scope.Root
	assets, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Fixture == nil {
			continue
		}
		result.Inspected++
		if len(asset.Fixture.DataShapes) == 0 {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.fixture_adversarial", AssetID: asset.Asset.ID,
				Message:     "fixture declares no data shapes at all",
				Remediation: "Add a dataShapes array to this fixture's catalog entry naming the shapes it supplies (at minimum one of \"failure\" or \"overflow\"). A fixture that only supplies happy-path data lets every component consuming it pass its gates without ever rendering an error or a string long enough to wrap, which is exactly where layout defects hide.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
		if !contains(asset.Fixture.DataShapes, "failure") && !contains(asset.Fixture.DataShapes, "overflow") {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.fixture_adversarial", AssetID: asset.Asset.ID,
				Message:     fmt.Sprintf("fixture declares %v but neither \"failure\" nor \"overflow\"", asset.Fixture.DataShapes),
				Remediation: "Add a \"failure\" shape (what this data looks like when the source errors) or an \"overflow\" shape (values long or numerous enough to exceed their container). These are the two shapes that reveal layout and error-state defects; without one of them the fixture cannot drive a component into the states its experience contract claims to handle.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
		if asset.Fixture.Satisfies != nil && asset.Fixture.Satisfies.Capability == "data-source" && len(asset.Fixture.Satisfies.TypeArguments) == 0 {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.fixture_data_source", AssetID: asset.Asset.ID,
				Message:     "fixture satisfies the data-source capability without declaring a type argument",
				Remediation: "Add typeArguments to this fixture's satisfies block naming the row type it produces. The data-source capability is generic; without the type argument a consuming asset cannot tell whether this fixture supplies the shape it needs, so the port match succeeds structurally and then fails at render.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
		if consumers, err := fixtureConsumers(root, asset.Asset.ID); err != nil {
			return Result{}, err
		} else if consumers > 0 {
			assertions, err := fixtureFailureAssertions(root, asset.Asset.ID)
			if err != nil {
				return Result{}, err
			}
			if assertions == 0 {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.fixture_adversarial_render", AssetID: asset.Asset.ID,
					Message:     "fixture failure shape has consumers but no rendered error-state assertion",
					Remediation: "Add a failure-shaped consumer story and assert role=alert or data-fixture-state=failure. The preview harness accepts fixtureShape=failure and renders the fixture's failure record, so this check exercises the consumer rather than trusting fixture metadata.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
		}
	}
	return nonEmpty(result, "fixture_adversarial"), nil
}

type fixtureStoryContract struct {
	Composition struct {
		Fixture struct {
			Asset string `json:"asset"`
		} `json:"fixture"`
	} `json:"composition"`
	Stories []struct {
		Expect []struct {
			Role      string `json:"role"`
			Selector  string `json:"selector"`
			Attribute string `json:"attribute"`
			Value     string `json:"value"`
		} `json:"expect"`
	} `json:"stories"`
}
