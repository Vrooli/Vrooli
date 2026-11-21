package rehearsals

import (
	"reflect"
	"testing"
)

func TestBoundedStaticSetJSONAcceptsNestedArrayAndObjectLiterals(t *testing.T) {
	array, err := decodeBoundedSetJSON(`=[{"owner":"synthetic-owner","name":"synthetic-repo"}]`, true)
	if err != nil {
		t.Fatalf("decode bounded static array: %v", err)
	}
	wantArray := []any{map[string]any{"owner": "synthetic-owner", "name": "synthetic-repo"}}
	if !reflect.DeepEqual(array, wantArray) {
		t.Fatalf("decoded array = %#v, want %#v", array, wantArray)
	}
	object, err := decodeBoundedSetJSON(`={"jira-user-example":"synthetic-user","github-login-example":"synthetic-login"}`, false)
	if err != nil {
		t.Fatalf("decode bounded static object: %v", err)
	}
	wantObject := map[string]any{"jira-user-example": "synthetic-user", "github-login-example": "synthetic-login"}
	if !reflect.DeepEqual(object, wantObject) {
		t.Fatalf("decoded object = %#v, want %#v", object, wantObject)
	}
}

func TestBoundedStaticSetJSONRejectsExpressionsAndOversizedValues(t *testing.T) {
	for _, value := range []any{
		`={{ $json.items }}`,
		`=[{"unsafe key":"value"}]`,
		`=[[[[[[[[[[[[[[[[]]]]]]]]]]]]]]]]`,
		"=[\"" + string(make([]byte, 4100)) + "\"]",
	} {
		if _, err := decodeBoundedSetJSON(value, true); err == nil {
			t.Errorf("accepted unsafe static Set JSON %#v", value)
		}
	}
}
