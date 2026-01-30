package skills

import (
	"testing"
)

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Variable
	}{
		{
			name:     "no variables",
			content:  "Hello world, this is plain text",
			expected: nil,
		},
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:    "single variable",
			content: "Hello {{NAME}}",
			expected: []Variable{
				{Name: "NAME", Placeholder: "{{NAME}}", Occurrences: 1},
			},
		},
		{
			name:    "multiple different variables",
			content: "Use {{TARGET}} in {{FOLDER}} with {{CONFIG}}",
			expected: []Variable{
				{Name: "CONFIG", Placeholder: "{{CONFIG}}", Occurrences: 1},
				{Name: "FOLDER", Placeholder: "{{FOLDER}}", Occurrences: 1},
				{Name: "TARGET", Placeholder: "{{TARGET}}", Occurrences: 1},
			},
		},
		{
			name:    "multiple occurrences of same variable",
			content: "{{TARGET}} and {{TARGET}} again, also {{TARGET}}",
			expected: []Variable{
				{Name: "TARGET", Placeholder: "{{TARGET}}", Occurrences: 3},
			},
		},
		{
			name:    "variables sorted alphabetically",
			content: "{{ZEBRA}} {{ALPHA}} {{BETA}}",
			expected: []Variable{
				{Name: "ALPHA", Placeholder: "{{ALPHA}}", Occurrences: 1},
				{Name: "BETA", Placeholder: "{{BETA}}", Occurrences: 1},
				{Name: "ZEBRA", Placeholder: "{{ZEBRA}}", Occurrences: 1},
			},
		},
		{
			name:     "lowercase patterns ignored",
			content:  "{{lowercase}} and {{mixedCase}} and {{VALID}}",
			expected: []Variable{
				{Name: "VALID", Placeholder: "{{VALID}}", Occurrences: 1},
			},
		},
		{
			name:     "invalid patterns ignored - numbers only",
			content:  "{{123}} and {{456}}",
			expected: nil,
		},
		{
			name:     "invalid patterns ignored - empty braces",
			content:  "{{}} and {{ }}",
			expected: nil,
		},
		{
			name:    "underscores and numbers allowed",
			content: "{{MY_VAR_1}} and {{ANOTHER_2}} and {{A1_B2_C3}}",
			expected: []Variable{
				{Name: "A1_B2_C3", Placeholder: "{{A1_B2_C3}}", Occurrences: 1},
				{Name: "ANOTHER_2", Placeholder: "{{ANOTHER_2}}", Occurrences: 1},
				{Name: "MY_VAR_1", Placeholder: "{{MY_VAR_1}}", Occurrences: 1},
			},
		},
		{
			name:    "variable at start and end",
			content: "{{START}}middle{{END}}",
			expected: []Variable{
				{Name: "END", Placeholder: "{{END}}", Occurrences: 1},
				{Name: "START", Placeholder: "{{START}}", Occurrences: 1},
			},
		},
		{
			name:    "multiline content",
			content: "Line 1: {{VAR1}}\nLine 2: {{VAR2}}\nLine 3: {{VAR1}}",
			expected: []Variable{
				{Name: "VAR1", Placeholder: "{{VAR1}}", Occurrences: 2},
				{Name: "VAR2", Placeholder: "{{VAR2}}", Occurrences: 1},
			},
		},
		{
			name:     "partial braces not matched",
			content:  "{TARGET} and {{TARGET and TARGET}}",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariables(tt.content)
			if len(got) != len(tt.expected) {
				t.Errorf("ExtractVariables() returned %d vars, want %d", len(got), len(tt.expected))
				t.Errorf("got: %+v", got)
				t.Errorf("want: %+v", tt.expected)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("Variable %d = %+v, want %+v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		values   map[string]string
		expected string
	}{
		{
			name:     "nil values map",
			content:  "Hello {{NAME}}",
			values:   nil,
			expected: "Hello {{NAME}}",
		},
		{
			name:     "empty values map",
			content:  "Hello {{NAME}}",
			values:   map[string]string{},
			expected: "Hello {{NAME}}",
		},
		{
			name:     "single substitution",
			content:  "Hello {{NAME}}",
			values:   map[string]string{"NAME": "World"},
			expected: "Hello World",
		},
		{
			name:     "multiple occurrences",
			content:  "{{TARGET}} and {{TARGET}} again",
			values:   map[string]string{"TARGET": "foo"},
			expected: "foo and foo again",
		},
		{
			name:     "multiple different variables",
			content:  "{{GREETING}} {{NAME}}, welcome to {{PLACE}}",
			values:   map[string]string{"GREETING": "Hello", "NAME": "User", "PLACE": "Earth"},
			expected: "Hello User, welcome to Earth",
		},
		{
			name:     "missing value left unchanged",
			content:  "{{KNOWN}} and {{UNKNOWN}}",
			values:   map[string]string{"KNOWN": "value"},
			expected: "value and {{UNKNOWN}}",
		},
		{
			name:     "empty string substitution",
			content:  "Hello {{NAME}} there",
			values:   map[string]string{"NAME": ""},
			expected: "Hello  there",
		},
		{
			name:     "value contains special characters",
			content:  "Path: {{PATH}}",
			values:   map[string]string{"PATH": "/home/user/my folder/file.txt"},
			expected: "Path: /home/user/my folder/file.txt",
		},
		{
			name:     "value contains braces",
			content:  "Config: {{CONFIG}}",
			values:   map[string]string{"CONFIG": "{{nested}}"},
			expected: "Config: {{nested}}",
		},
		{
			name:     "no variables in content",
			content:  "Plain text without variables",
			values:   map[string]string{"UNUSED": "value"},
			expected: "Plain text without variables",
		},
		{
			name:     "multiline substitution",
			content:  "Line 1: {{VAR}}\nLine 2: {{VAR}}",
			values:   map[string]string{"VAR": "replaced"},
			expected: "Line 1: replaced\nLine 2: replaced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteVariables(tt.content, tt.values)
			if got != tt.expected {
				t.Errorf("SubstituteVariables() = %q, want %q", got, tt.expected)
			}
		})
	}
}
