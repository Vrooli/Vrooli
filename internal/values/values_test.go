package values

import (
	"reflect"
	"testing"
)

func TestFirstNonEmpty(t *testing.T) {
	if got := FirstNonEmpty(" ", "chosen", "later"); got != "chosen" {
		t.Fatalf("FirstNonEmpty = %q, want chosen", got)
	}
}

func TestUniqueStringsNormalizesAndSorts(t *testing.T) {
	got := UniqueStrings([]string{" b ", "", "a", "b", "a"})
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UniqueStrings = %#v, want %#v", got, want)
	}
}
