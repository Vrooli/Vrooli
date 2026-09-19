package types

import "testing"

func TestStringPtrTreatsEmptyStringAsUnset(t *testing.T) {
	if StringPtr("") != nil {
		t.Fatal("expected empty string to return nil")
	}
	ptr := StringPtr("value")
	if ptr == nil || *ptr != "value" {
		t.Fatalf("unexpected string pointer: %+v", ptr)
	}
}

func TestBoolPtrReturnsPointerToValue(t *testing.T) {
	ptr := BoolPtr(true)
	if ptr == nil || !*ptr {
		t.Fatalf("unexpected bool pointer: %+v", ptr)
	}
}
