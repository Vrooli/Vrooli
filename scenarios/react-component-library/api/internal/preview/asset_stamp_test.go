package preview

import "testing"

func TestStampPreviewSourceMarksOwnedRootAndRemovesAuthoredMarker(t *testing.T) {
	source := `export function Card() { return (<div data-rcl-asset="forged"><span data-rcl-asset="child" /></div>); }`
	got := stampPreviewSource(source, "library/components/Card/versions/1.1.0/Card.tsx", "primitives.card", "1.1.0")
	if want := `data-rcl-asset="primitives.card"`; !containsText(got, want) {
		t.Fatalf("stamped preview source missing %q: %s", want, got)
	}
	if containsText(got, "forged") || containsText(got, `data-rcl-asset="child"`) {
		t.Fatalf("preview source retained authored marker: %s", got)
	}
}

func TestStampPreviewSourceSkipsProviderAndStyleToFindOwnedRoot(t *testing.T) {
	source := `export function Live() { return <Context.Provider value={null}><style>{""}</style><div /></Context.Provider>; }`
	got := stampPreviewSource(source, "library/services/Live/versions/1.0.0/Live.tsx", "services.live", "1.0.0")
	if !containsText(got, `data-rcl-asset="services.live"`) {
		t.Fatalf("owned provider child was not stamped: %s", got)
	}
}

func TestStampPreviewSourceSupportsDynamicRoot(t *testing.T) {
	source := `import { createElement } from "react"; export function Presence({ Component }) { return createElement(Component, { className: "x" }); }`
	got := stampPreviewSource(source, "library/primitives/Presence/versions/1.0.0/Presence.tsx", "motion.presence", "1.0.0")
	if want := `"data-rcl-asset": "motion.presence"`; !containsText(got, want) {
		t.Fatalf("dynamic root was not stamped: %s", got)
	}
}

func containsText(value, want string) bool {
	for index := 0; index+len(want) <= len(value); index++ {
		if value[index:index+len(want)] == want {
			return true
		}
	}
	return false
}
