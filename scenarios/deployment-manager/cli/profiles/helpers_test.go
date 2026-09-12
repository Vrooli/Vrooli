package profiles

import (
	"reflect"
	"testing"
)

func TestProfileHelpersAndDiff(t *testing.T) {
	p, err := decodeProfile([]byte(`{"id":"p","scenario":"demo","tiers":[2],"swaps":{"a":"b"}}`))
	if err != nil || len(p.Swaps) != 1 || p.Secrets == nil || p.Settings == nil {
		t.Fatalf("decode/normalize: %+v %v", p, err)
	}
	obj := map[string]interface{}{}
	if ensureNestedMap(obj, "nested") == nil || ensureNestedMap(map[string]interface{}{"nested": "bad"}, "nested") == nil {
		t.Fatal("nested map helper failed")
	}
	previous := Profile{Tiers: []int{1}, Swaps: Swaps{{From: "a", To: "b"}}, Secrets: map[string]interface{}{"x": 1}, Settings: map[string]interface{}{"mode": "a"}}
	current := Profile{Tiers: []int{2}, Swaps: Swaps{{From: "a", To: "c"}}, Secrets: map[string]interface{}{"x": 2}, Settings: map[string]interface{}{"mode": "b"}}
	diff := computeProfileDiff(previous, current)
	if len(diff) != 4 || !reflect.DeepEqual(extractUpdatableProfileFields(current)["tiers"], current.Tiers) {
		t.Fatalf("unexpected diff: %+v", diff)
	}
	if _, ok := findVersion([]Profile{{Version: 2}}, "2"); !ok || versionNumber(Profile{}) != 0 || intSliceToString([]int{1, 2}) != "1,2" {
		t.Fatal("version helpers failed")
	}
	if fallbackProfileValue("", "none") != "none" || firstNonEmpty("", "value") != "value" {
		t.Fatal("fallback helpers failed")
	}
}

func TestSwapsJSONAndMutation(t *testing.T) {
	var swaps Swaps
	if err := swaps.UnmarshalJSON([]byte(`{"postgres":"sqlite"}`)); err != nil || len(swaps) != 1 {
		t.Fatal(err)
	}
	swaps.set("postgres", "redis")
	swaps.set("api", "desktop")
	swaps.remove("postgres")
	if len(swaps) != 1 || swaps[0].To != "desktop" {
		t.Fatalf("unexpected swaps: %+v", swaps)
	}
	if err := swaps.UnmarshalJSON([]byte(`null`)); err != nil || swaps == nil {
		t.Fatal("null swaps should initialize")
	}
	if err := swaps.UnmarshalJSON([]byte(`"bad"`)); err == nil {
		t.Fatal("unsupported swaps format should fail")
	}
	if _, err := swaps.MarshalJSON(); err != nil {
		t.Fatal(err)
	}
}
