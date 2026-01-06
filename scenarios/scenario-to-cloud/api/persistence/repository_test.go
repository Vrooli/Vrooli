package persistence

import "testing"

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		substr string
		expect bool
	}{
		{name: "empty substr", input: "abc", substr: "", expect: true},
		{name: "match", input: "hello world", substr: "world", expect: true},
		{name: "no match", input: "hello", substr: "WORLD", expect: false},
		{name: "exact", input: "abc", substr: "abc", expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.input, tt.substr); got != tt.expect {
				t.Fatalf("contains(%q, %q) = %v, want %v", tt.input, tt.substr, got, tt.expect)
			}
		})
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		substr string
		expect bool
	}{
		{name: "prefix", input: "abcdef", substr: "abc", expect: true},
		{name: "middle", input: "abcdef", substr: "cd", expect: true},
		{name: "suffix", input: "abcdef", substr: "ef", expect: true},
		{name: "absent", input: "abcdef", substr: "gh", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findSubstring(tt.input, tt.substr); got != tt.expect {
				t.Fatalf("findSubstring(%q, %q) = %v, want %v", tt.input, tt.substr, got, tt.expect)
			}
		})
	}
}
