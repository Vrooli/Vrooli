package gates

import (
	"path/filepath"
	"testing"
)

func TestAuthoredQualityGatesReportUserFacingAndStylingViolations(t *testing.T) {
	root := t.TempDir()
	writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/catalog/assets/components/button.json"), `{"asset":{"id":"controls.button","kind":"component","name":"Button"}}`)
	writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/library/components/Button/component.json"), `{"catalogId":"controls.button","libraryId":"react-component-library:Button","latest":"1.0.0"}`)
	source := `import "theme.css";
export function Button(){ return <button aria-label="Save"><style></style>Save</button> }
/* @media (prefers-reduced-motion: reduce) { animation: none; } */`
	writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions/1.0.0/Button.tsx"), source)

	for _, test := range []struct {
		name string
		run  func(Scope) (Result, error)
	}{
		{name: "i18n", run: ValidateI18n},
		{name: "style ownership", run: ValidateStyleOwnership},
		{name: "style injection", run: ValidateStyleInjection},
		{name: "shared style ownership", run: ValidateSharedStyleOwnership},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.run(Scope{Root: root, Assets: []string{"controls.button"}})
			if err != nil {
				t.Fatal(err)
			}
			if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].AssetID != "controls.button" {
				t.Fatalf("result = %+v, want one actionable finding for the active button source", result)
			}
			if result.Findings[0].Remediation == "" || result.Findings[0].DocsRef == "" {
				t.Fatalf("finding lacks remediation evidence: %+v", result.Findings[0])
			}
		})
	}
}

func TestAuthoredQualityGatesAcceptCompliantImplementation(t *testing.T) {
	root := t.TempDir()
	writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/catalog/assets/components/button.json"), `{"asset":{"id":"controls.button","kind":"component","name":"Button"}}`)
	writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/library/components/Button/component.json"), `{"catalogId":"controls.button","libraryId":"react-component-library:Button","latest":"1.0.0"}`)
	writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions/1.0.0/Button.tsx"), `export function Button({label}: {label: string}) { return <button aria-label={label}>{label}</button> }`)

	for _, run := range []func(Scope) (Result, error){ValidateI18n, ValidateStyleOwnership, ValidateStyleInjection, ValidateSharedStyleOwnership} {
		result, err := run(Scope{Root: root, Assets: []string{"controls.button"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Findings) != 0 || len(result.RunnerError) != 0 {
			t.Fatalf("compliant source was rejected: %+v", result)
		}
	}
}
