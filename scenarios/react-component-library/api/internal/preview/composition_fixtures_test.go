package preview

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func fixtureBindings() map[string]any {
	return map[string]any{"$template": map[string]any{}, "$labels": map[string]any{"missing": "Missing fixture", "failed": "Region failed"}, "inspector": map[string]any{"onClick": map[string]any{"$handler": "inspect"}}}
}
func TestCompositionFixturesReuseDeterministicFamilies(t *testing.T) {
	_, input := compositionFixture(t)
	bindings := fixtureBindings()
	refs := []CompositionFixture{{Target: "inspector", Asset: "fixtures.user-directory", Version: "1.0.0", State: "typical", Field: "records", Prop: []string{"data", "users"}}}
	got, err := PrepareComposition(input, bindings, refs)
	if err != nil {
		t.Fatal(err)
	}
	again, err := PrepareComposition(input, bindings, refs)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("same fixture inputs changed")
	}
	if got.Fixtures[0].Clock != "2026-01-15T12:00:00Z" || len(got.Fixtures[0].Records) != 3 {
		t.Fatal("fixture family not used")
	}
	if _, exists := bindings["inspector"].(map[string]any)["data"]; exists {
		t.Fatal("fixture preparation mutated caller bindings")
	}
	refs[0].Version = "999.0.0"
	if _, err := PrepareComposition(input, bindings, refs); err == nil {
		t.Fatal("invented fixture version accepted")
	}
}
func TestMissingFixtureBecomesScopedGap(t *testing.T) {
	_, input := compositionFixture(t)
	bindings := fixtureBindings()
	delete(bindings, "inspector")
	got, err := PrepareComposition(input, bindings, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Gaps) != 1 || got.Gaps[0].Code != "fixture_missing" || got.Composition.Regions[0].Asset != nil || input.Regions[0].Asset == nil {
		t.Fatalf("fixture gap lost: %+v", got)
	}
}
func TestCompositionRejectsExecutableAndExternalFixtureBindings(t *testing.T) {
	_, input := compositionFixture(t)
	for _, props := range []map[string]any{{"onClick": "fetch('/production')"}, {"$node": "script"}, {"dangerouslySetInnerHTML": map[string]any{"__html": "<script/>"}}, {"href": "https://production.invalid"}, {"__proto__": map[string]any{}}, {"onClick": map[string]any{"$handler": "click", "extra": true}}} {
		bindings := fixtureBindings()
		bindings["inspector"] = props
		if _, err := PrepareComposition(input, bindings, nil); err == nil {
			t.Fatalf("unsafe fixture accepted: %+v", props)
		}
	}
	bindings := fixtureBindings()
	bindings["unrelated"] = map[string]any{}
	if _, err := PrepareComposition(input, bindings, nil); err == nil {
		t.Fatal("undeclared target accepted")
	}
}

func TestPrepareCompositionPreviewInteractions(t *testing.T) {
	_, c := compositionFixture(t)
	valid := `{"initial":"list","states":{"list":{"$template":{"mobilePane":"collection"}},"detail":{"$template":{"mobilePane":"inspector"}}},"actions":{"open":[{"argument":["id"],"equals":"one","state":"detail","result":true}],"back":[{"state":"list","result":false}]}}`
	var interaction any
	if err := json.Unmarshal([]byte(valid), &interaction); err != nil {
		t.Fatal(err)
	}
	bindings := fixtureBindings()
	bindings["$preview"] = interaction
	prepared, err := PrepareComposition(c, bindings, nil)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Bindings["$preview"] == nil || bindings["$preview"] == nil {
		t.Fatal("interaction was lost or caller mutated")
	}
	invalid := []string{
		strings.Replace(valid, `"result":true`, `"result":"fetch('/live')"`, 1),
		strings.Replace(valid, `"result":true`, `"result":{"accepted":true}`, 1),
		strings.Replace(valid, `"initial":"list"`, `"initial":"absent"`, 1),
		strings.Replace(valid, `"state":"detail"`, `"state":"absent"`, 1),
		strings.Replace(valid, `"argument":["id"]`, `"argument":["__proto__"]`, 1),
		strings.Replace(valid, `"mobilePane":"inspector"`, `"onClick":"fetch('/live')"`, 1),
		strings.Replace(valid, `"$template":{"mobilePane":"inspector"}`, `"undeclared":{"mobilePane":"inspector"}`, 1),
		strings.Replace(valid, `"initial":"list"`, `"initial":"list","code":"alert(1)"`, 1),
	}
	for _, raw := range invalid {
		if err := json.Unmarshal([]byte(raw), &interaction); err != nil {
			t.Fatal(err)
		}
		bindings["$preview"] = interaction
		if _, err := PrepareComposition(c, bindings, nil); err == nil {
			t.Fatalf("accepted invalid interaction: %s", raw)
		}
	}
}

func TestSelectCompositionPreviewStatePreservesCandidate(t *testing.T) {
	p := PreparedComposition{Bindings: map[string]any{"$preview": map[string]any{
		"initial": "list", "states": map[string]any{"list": map[string]any{}, "detail": map[string]any{}},
	}}}
	before, _ := json.Marshal(p)
	selected, err := SelectCompositionPreviewState(p, "detail")
	if err != nil || selected.Bindings["$preview"].(map[string]any)["initial"] != "detail" {
		t.Fatalf("declared state was not selected: %v", err)
	}
	after, _ := json.Marshal(p)
	if string(before) != string(after) {
		t.Fatal("state selection mutated saved input")
	}
	for _, state := range []string{"missing", "__proto__"} {
		if _, err := SelectCompositionPreviewState(p, state); err == nil {
			t.Fatalf("accepted undeclared state %q", state)
		}
	}
	if _, err := SelectCompositionPreviewState(PreparedComposition{}, "detail"); err == nil {
		t.Fatal("accepted selection without state graph")
	}
	if _, err := SelectCompositionPreviewState(PreparedComposition{}, ""); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewAssignmentsRequireBoundedSafeScalarSources(t *testing.T) {
	base := `{"initial":"list","states":{"list":{}},"actions":{"apply":[{"state":"list","set":[ASSIGNMENT]}]}}`
	for _, row := range []struct {
		assignment string
		valid      bool
	}{
		{`{"target":"$template","prop":"query","argument":["query"]}`, true},
		{`{"target":"$template","prop":"query","value":""}`, true},
		{`{"target":"$template","prop":"query","argument":[],"scope":"state"}`, true},
		{`{"target":"$template","prop":"query","argument":[],"scope":"external"}`, false},
		{`{"target":"missing","prop":"query","argument":[]}`, false},
		{`{"target":"$template","prop":"onClick","value":"code"}`, false},
		{`{"target":"$template","prop":"href","argument":[]}`, false},
		{`{"target":"$template","prop":"srcDoc","argument":[]}`, false},
		{`{"target":"$template","prop":"query","argument":["__proto__"]}`, false},
		{`{"target":"$template","prop":"query","argument":["a","b","c","d","e"]}`, false},
		{`{"target":"$template","prop":"query","value":{}}`, false},
		{`{"target":"$template","prop":"query","argument":[],"value":"ambiguous"}`, false},
	} {
		var spec any
		if err := json.Unmarshal([]byte(strings.Replace(base, "ASSIGNMENT", row.assignment, 1)), &spec); err != nil {
			t.Fatal(err)
		}
		err := validatePreviewInteraction(spec, map[string]bool{"$template": true})
		if (err == nil) != row.valid {
			t.Fatalf("assignment %s: %v", row.assignment, err)
		}
	}
}

func TestCompositionAcceptsDescriptiveActionProps(t *testing.T) {
	_, input := compositionFixture(t)
	bindings := fixtureBindings()
	bindings["inspector"] = map[string]any{"action": "inspect the attachment"}
	if _, err := PrepareComposition(input, bindings, nil); err != nil {
		t.Fatalf("plain-language decision action rejected: %v", err)
	}
	bindings["inspector"] = map[string]any{"formAction": "https://production.invalid"}
	if _, err := PrepareComposition(input, bindings, nil); err == nil {
		t.Fatal("explicit external form destination accepted")
	}
}
