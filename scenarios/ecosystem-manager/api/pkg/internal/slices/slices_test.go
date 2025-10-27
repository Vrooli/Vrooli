package slices

import (
	"reflect"
	"testing"
)

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]struct{}
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]struct{}{},
			expected: []string{},
		},
		{
			name: "single key",
			input: map[string]struct{}{
				"foo": {},
			},
			expected: []string{"foo"},
		},
		{
			name: "multiple keys sorted",
			input: map[string]struct{}{
				"zebra":  {},
				"alpha":  {},
				"middle": {},
			},
			expected: []string{"alpha", "middle", "zebra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortedKeys(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SortedKeys() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDedupeAndSort(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "single item",
			input:    []string{"foo"},
			expected: []string{"foo"},
		},
		{
			name:     "duplicates removed",
			input:    []string{"foo", "bar", "foo", "baz", "bar"},
			expected: []string{"bar", "baz", "foo"},
		},
		{
			name:     "empty strings removed",
			input:    []string{"foo", "", "bar", "", "baz"},
			expected: []string{"bar", "baz", "foo"},
		},
		{
			name:     "sorted output",
			input:    []string{"zebra", "alpha", "middle"},
			expected: []string{"alpha", "middle", "zebra"},
		},
		{
			name:     "all empty strings",
			input:    []string{"", "", ""},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DedupeAndSort(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DedupeAndSort() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "foo",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			item:     "foo",
			expected: false,
		},
		{
			name:     "item present",
			slice:    []string{"foo", "bar", "baz"},
			item:     "bar",
			expected: true,
		},
		{
			name:     "item not present",
			slice:    []string{"foo", "bar", "baz"},
			item:     "qux",
			expected: false,
		},
		{
			name:     "item at start",
			slice:    []string{"foo", "bar", "baz"},
			item:     "foo",
			expected: true,
		},
		{
			name:     "item at end",
			slice:    []string{"foo", "bar", "baz"},
			item:     "baz",
			expected: true,
		},
		{
			name:     "empty string search",
			slice:    []string{"foo", "", "bar"},
			item:     "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("Contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEqualStringSlices(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "one nil one empty",
			a:        nil,
			b:        []string{},
			expected: true, // Both have length 0, so they're equal
		},
		{
			name:     "identical slices",
			a:        []string{"foo", "bar", "baz"},
			b:        []string{"foo", "bar", "baz"},
			expected: true,
		},
		{
			name:     "different order",
			a:        []string{"foo", "bar", "baz"},
			b:        []string{"bar", "foo", "baz"},
			expected: false,
		},
		{
			name:     "different length",
			a:        []string{"foo", "bar"},
			b:        []string{"foo", "bar", "baz"},
			expected: false,
		},
		{
			name:     "different content",
			a:        []string{"foo", "bar", "baz"},
			b:        []string{"foo", "qux", "baz"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EqualStringSlices(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("EqualStringSlices() = %v, want %v", result, tt.expected)
			}
		})
	}
}
