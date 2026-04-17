package manifest

import (
	"scenario-to-cloud/domain"
	"testing"
)

func TestHasBlockingIssues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		issues []domain.ValidationIssue
		want   bool
	}{
		{
			name:   "empty issues returns false",
			issues: nil,
			want:   false,
		},
		{
			name: "only warnings returns false",
			issues: []domain.ValidationIssue{
				{Severity: domain.SeverityWarn, Message: "warning 1"},
				{Severity: domain.SeverityWarn, Message: "warning 2"},
			},
			want: false,
		},
		{
			name: "single error returns true",
			issues: []domain.ValidationIssue{
				{Severity: domain.SeverityError, Message: "error 1"},
			},
			want: true,
		},
		{
			name: "mixed warnings and errors returns true",
			issues: []domain.ValidationIssue{
				{Severity: domain.SeverityWarn, Message: "warning 1"},
				{Severity: domain.SeverityError, Message: "error 1"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasBlockingIssues(tt.issues)
			if got != tt.want {
				t.Errorf("HasBlockingIssues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLooksLikeDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		// Valid domains
		{"example.com", true},
		{"app.example.com", true},
		{"sub.app.example.com", true},
		{"test-site.io", true},
		{"my-app.example.org", true},

		// Invalid: empty or whitespace
		{"", false},
		{"  ", false},

		// Invalid: contains scheme
		{"https://example.com", false},
		{"http://example.com", false},

		// Invalid: contains path
		{"example.com/path", false},

		// Invalid: leading/trailing dots
		{".example.com", false},
		{"example.com.", false},

		// Invalid: no TLD (single label)
		{"localhost", false},

		// Invalid: empty labels
		{"example..com", false},

		// Invalid: label starts/ends with hyphen
		{"-example.com", false},
		{"example-.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeDomain(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeDomain(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStableUniqueStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "empty input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "already unique and sorted",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "removes duplicates",
			input: []string{"a", "b", "a", "c", "b"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "sorts output",
			input: []string{"c", "a", "b"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "trims whitespace",
			input: []string{"  a  ", "b", "  c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "ignores empty strings",
			input: []string{"a", "", "b", "  ", "c"},
			want:  []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StableUniqueStrings(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("len(StableUniqueStrings) = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("StableUniqueStrings[%d] = %q, want %q", i, got[i], tt.want[i])
				}
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
		{
			name:  "found in slice",
			slice: []string{"a", "b", "c"},
			value: "b",
			want:  true,
		},
		{
			name:  "not found in slice",
			slice: []string{"a", "b", "c"},
			value: "d",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: nil,
			value: "a",
			want:  false,
		},
		{
			name:  "exact match required",
			slice: []string{"abc", "def"},
			value: "ab",
			want:  false,
		},
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

func TestFindDuplicatePorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ports domain.ManifestPorts
		want  int // number of duplicates
	}{
		{
			name:  "no duplicates",
			ports: domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
			want:  0,
		},
		{
			name:  "has duplicates",
			ports: domain.ManifestPorts{"ui": 3000, "api": 3000, "ws": 3002},
			want:  1,
		},
		{
			name:  "invalid ports ignored",
			ports: domain.ManifestPorts{"ui": 0, "api": 0, "ws": 3002},
			want:  0,
		},
		{
			name:  "empty map",
			ports: domain.ManifestPorts{},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findDuplicatePorts(tt.ports)
			if len(got) != tt.want {
				t.Errorf("findDuplicatePorts() returned %d duplicates, want %d", len(got), tt.want)
			}
		})
	}
}

func TestFindInvalidPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ports domain.ManifestPorts
		want  int // number of invalid ports
	}{
		{
			name:  "all valid",
			ports: domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
			want:  0,
		},
		{
			name:  "zero is invalid",
			ports: domain.ManifestPorts{"ui": 0, "api": 3001},
			want:  1,
		},
		{
			name:  "negative is invalid",
			ports: domain.ManifestPorts{"ui": -1, "api": 3001},
			want:  1,
		},
		{
			name:  "above 65535 is invalid",
			ports: domain.ManifestPorts{"ui": 65536, "api": 3001},
			want:  1,
		},
		{
			name:  "boundary value 65535 is valid",
			ports: domain.ManifestPorts{"ui": 65535, "api": 1},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findInvalidPorts(tt.ports)
			if len(got) != tt.want {
				t.Errorf("findInvalidPorts() returned %d invalid, want %d", len(got), tt.want)
			}
		})
	}
}

func TestValidatePreservePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid scenario path", input: "scenarios/demo/data", wantErr: false},
		{name: "valid nested path", input: "scenarios/landing-page-business-suite/api/uploads", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "absolute", input: "/scenarios/demo/data", wantErr: true},
		{name: "parent traversal", input: "../etc/passwd", wantErr: true},
		{name: "too short", input: "scenarios/demo", wantErr: true},
		{name: "outside scenarios", input: ".vrooli/cloud/data", wantErr: true},
		{name: "whitespace", input: "scenarios/demo/my data", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePreservePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validatePreservePath(%q) error=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}
