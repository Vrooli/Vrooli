package main

import "testing"

func TestValidateStringLengthUsesCompleteNumericLimits(t *testing.T) {
	t.Parallel()

	if err := ValidateStringLength("short", "description", 12, 0); err == nil || err.Error() != "description must be at least 12 characters" {
		t.Fatalf("minimum error = %v", err)
	}
	if err := ValidateStringLength("this value is too long", "description", 0, 12); err == nil || err.Error() != "description must be at most 12 characters" {
		t.Fatalf("maximum error = %v", err)
	}
}
