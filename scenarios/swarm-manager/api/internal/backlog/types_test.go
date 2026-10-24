package backlog

import "testing"

func TestValidateGlobs(t *testing.T) {
	tests := []struct {
		name    string
		globs   []string
		wantErr bool
	}{
		{name: "nil is ok", globs: nil, wantErr: false},
		{name: "empty slice ok", globs: []string{}, wantErr: false},
		{name: "valid glob ok", globs: []string{"api/**"}, wantErr: false},
		{name: "multiple valid ok", globs: []string{"api/**", "*.go"}, wantErr: false},
		{name: "empty string error", globs: []string{""}, wantErr: true},
		{name: "absolute path error", globs: []string{"/etc/*"}, wantErr: true},
		{name: "bad syntax error", globs: []string{"[invalid"}, wantErr: true},
		{name: "parent traversal error", globs: []string{"../secret"}, wantErr: true},
		{name: "dot prefix ok", globs: []string{"./api/**"}, wantErr: false},
		{name: "doublestar ok", globs: []string{"docs/**/*.md"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGlobs(tt.globs)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGlobs(%v) error = %v, wantErr %v", tt.globs, err, tt.wantErr)
			}
		})
	}
}
