package stringutil

import (
	"reflect"
	"testing"
)

func TestSortedUnique(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"nil input", nil, nil},
		{"empty input", []string{}, nil},
		{"single element", []string{"a"}, []string{"a"}},
		{"already unique and sorted", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"duplicates removed", []string{"b", "a", "b", "c", "a"}, []string{"a", "b", "c"}},
		{"empty strings skipped", []string{"b", "", "a", ""}, []string{"a", "b"}},
		{"all empty strings", []string{"", "", ""}, nil},
		{"unsorted input sorted", []string{"z", "m", "a"}, []string{"a", "m", "z"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortedUnique(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortedUnique(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrderedUnique(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"nil input", nil, []string{}},
		{"empty input", []string{}, []string{}},
		{"preserves order", []string{"b", "a", "c"}, []string{"b", "a", "c"}},
		{"duplicates removed preserving first", []string{"b", "a", "b", "c", "a"}, []string{"b", "a", "c"}},
		{"trims whitespace", []string{" a ", "  b  "}, []string{"a", "b"}},
		{"whitespace duplicates", []string{"a", " a "}, []string{"a"}},
		{"empty and whitespace skipped", []string{"", "  ", "a", "", "b"}, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OrderedUnique(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrderedUnique(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		slice []string
		value string
		want  bool
	}{
		{"nil slice", nil, "a", false},
		{"empty slice", []string{}, "a", false},
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty value not found in non-empty", []string{"a", "b"}, "", false},
		{"empty value found", []string{"a", "", "b"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Contains(tt.slice, tt.value)
			if got != tt.want {
				t.Errorf("Contains(%v, %q) = %v, want %v", tt.slice, tt.value, got, tt.want)
			}
		})
	}
}
