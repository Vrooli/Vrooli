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

func TestSetEnvDoesNotMutateInputOrItsSpareCapacity(t *testing.T) {
	input := make([]string, 1, 4)
	input[0] = "A=1"
	got := SetEnv(input, "B", "2")
	if got[0] != "A=1" || got[1] != "B=2" {
		t.Fatalf("SetEnv = %#v", got)
	}
	if len(input) != 1 || cap(input) != 4 {
		t.Fatalf("input shape changed: len=%d cap=%d", len(input), cap(input))
	}
	if len(input) == cap(input) {
		t.Fatal("test requires spare capacity")
	}
	input = append(input, "C=3")
	if reflect.DeepEqual(input, got) {
		t.Fatal("result unexpectedly aliases input backing array")
	}
}

func TestSetEnvDoesNotMutateExistingEntry(t *testing.T) {
	input := []string{"A=1", "B=2"}
	got := SetEnv(input, "B", "updated")
	if got[1] != "B=updated" {
		t.Fatalf("SetEnv = %#v", got)
	}
	if input[1] != "B=2" {
		t.Fatalf("input mutated: %#v", input)
	}
}

func TestEnvValue(t *testing.T) {
	if got := EnvValue([]string{"A=1", "B=two=parts"}, "B"); got != "two=parts" {
		t.Fatalf("EnvValue = %q", got)
	}
}
