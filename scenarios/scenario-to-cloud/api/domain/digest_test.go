package domain

import (
	"strings"
	"testing"
)

func TestIsSHA256Hex(t *testing.T) {
	valid := strings.Repeat("a", 64)
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "lowercase", value: valid, want: true},
		{name: "uppercase", value: strings.Repeat("A", 64), want: true},
		{name: "short", value: strings.Repeat("a", 62)},
		{name: "too long", value: strings.Repeat("a", 66)},
		{name: "non-hex", value: strings.Repeat("g", 64)},
		{name: "prefix is not bare hex", value: "sha256:" + valid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSHA256Hex(test.value); got != test.want {
				t.Fatalf("IsSHA256Hex(%q) = %t, want %t", test.value, got, test.want)
			}
		})
	}
}
