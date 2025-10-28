package issues

import (
	"testing"
)

func TestValidStatuses(t *testing.T) {
	statuses := ValidStatuses()

	if len(statuses) == 0 {
		t.Error("ValidStatuses should return non-empty slice")
	}

	// Check for expected statuses
	expectedStatuses := []string{"open", "active", "completed", "failed", "archived"}
	for _, expected := range expectedStatuses {
		found := false
		for _, status := range statuses {
			if status == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected status %q not found in valid statuses", expected)
		}
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input     string
		want      string
		wantValid bool
	}{
		{"OPEN", "open", true},
		{"Open", "open", true},
		{"open", "open", true},
		{"ACTIVE", "active", true},
		{"completed", "completed", true},
		{"FAILED", "failed", true},
		{"invalid", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		got, valid := NormalizeStatus(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
		if valid != tt.wantValid {
			t.Errorf("NormalizeStatus(%q) valid = %v, want %v", tt.input, valid, tt.wantValid)
		}
	}
}

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"open", true},
		{"active", true},
		{"completed", true},
		{"OPEN", true},
		{"Active", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsValidStatus(tt.input)
		if got != tt.want {
			t.Errorf("IsValidStatus(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCloneStringSlice(t *testing.T) {
	original := []string{"a", "b", "c"}
	cloned := CloneStringSlice(original)

	if len(cloned) != len(original) {
		t.Errorf("cloned slice length = %d, want %d", len(cloned), len(original))
	}

	for i := range original {
		if cloned[i] != original[i] {
			t.Errorf("cloned[%d] = %q, want %q", i, cloned[i], original[i])
		}
	}

	// Modify cloned and ensure original is unchanged
	cloned[0] = "modified"
	if original[0] == "modified" {
		t.Error("modifying cloned slice affected original")
	}

	// Test nil slice
	nilClone := CloneStringSlice(nil)
	if nilClone != nil {
		t.Error("cloning nil slice should return nil")
	}
}

func TestNormalizeStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "remove empty strings",
			input: []string{"a", "", "b", "  ", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "trim whitespace",
			input: []string{"  a  ", "b", "  c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "all empty strings",
			input: []string{"", "  ", ""},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeStringSlice(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("length = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCloneStringMap(t *testing.T) {
	original := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	cloned := CloneStringMap(original)

	if len(cloned) != len(original) {
		t.Errorf("cloned map length = %d, want %d", len(cloned), len(original))
	}

	for k, v := range original {
		if cloned[k] != v {
			t.Errorf("cloned[%q] = %q, want %q", k, cloned[k], v)
		}
	}

	// Modify cloned and ensure original is unchanged
	cloned["key1"] = "modified"
	if original["key1"] == "modified" {
		t.Error("modifying cloned map affected original")
	}

	// Test nil map
	nilClone := CloneStringMap(nil)
	if nilClone != nil {
		t.Error("cloning nil map should return nil")
	}
}
