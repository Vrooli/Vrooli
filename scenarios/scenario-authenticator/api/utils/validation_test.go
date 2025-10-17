package utils

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	defaultReqs := DefaultPasswordRequirements()

	tests := []struct {
		name          string
		password      string
		requirements  PasswordRequirements
		expectValid   bool
		expectedError string
	}{
		{
			name:         "valid password with all requirements",
			password:     "SecureP@ss123",
			requirements: defaultReqs,
			expectValid:  true,
		},
		{
			name:          "too short",
			password:      "Short1",
			requirements:  defaultReqs,
			expectValid:   false,
			expectedError: "Password must be at least 8 characters long",
		},
		{
			name:          "missing uppercase",
			password:      "lowercase123",
			requirements:  defaultReqs,
			expectValid:   false,
			expectedError: "Password must contain at least one uppercase letter",
		},
		{
			name:          "missing lowercase",
			password:      "UPPERCASE123",
			requirements:  defaultReqs,
			expectValid:   false,
			expectedError: "Password must contain at least one lowercase letter",
		},
		{
			name:          "missing number",
			password:      "NoNumbersHere",
			requirements:  defaultReqs,
			expectValid:   false,
			expectedError: "Password must contain at least one number",
		},
		{
			name:         "exactly 8 characters valid",
			password:     "Valid123",
			requirements: defaultReqs,
			expectValid:  true,
		},
		{
			name:         "long password valid",
			password:     "ThisIsAVeryLongPasswordWithNumbers12345AndUppercase",
			requirements: defaultReqs,
			expectValid:  true,
		},
		{
			name:     "password with special chars (when required)",
			password: "Secure@Pass123!",
			requirements: PasswordRequirements{
				MinLength:      8,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  true,
				RequireSpecial: true,
			},
			expectValid: true,
		},
		{
			name:     "missing special chars (when required)",
			password: "SecurePass123",
			requirements: PasswordRequirements{
				MinLength:      8,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  true,
				RequireSpecial: true,
			},
			expectValid:   false,
			expectedError: "Password must contain at least one special character",
		},
		{
			name:     "relaxed requirements (no numbers)",
			password: "OnlyLetters",
			requirements: PasswordRequirements{
				MinLength:    8,
				RequireUpper: true,
				RequireLower: true,
			},
			expectValid: true,
		},
		{
			name:          "empty password",
			password:      "",
			requirements:  defaultReqs,
			expectValid:   false,
			expectedError: "Password must be at least 8 characters long",
		},
		{
			name:          "whitespace password",
			password:      "        ",
			requirements:  defaultReqs,
			expectValid:   false,
			expectedError: "Password must contain at least one uppercase letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errMsg := ValidatePassword(tt.password, tt.requirements)

			if valid != tt.expectValid {
				t.Errorf("ValidatePassword() valid = %v, want %v", valid, tt.expectValid)
			}

			if !tt.expectValid && errMsg != tt.expectedError {
				t.Errorf("ValidatePassword() error = %q, want %q", errMsg, tt.expectedError)
			}

			if tt.expectValid && errMsg != "" {
				t.Errorf("ValidatePassword() expected no error but got %q", errMsg)
			}
		})
	}
}

func TestDefaultPasswordRequirements(t *testing.T) {
	reqs := DefaultPasswordRequirements()

	if reqs.MinLength != 8 {
		t.Errorf("DefaultPasswordRequirements().MinLength = %d, want 8", reqs.MinLength)
	}

	if !reqs.RequireUpper {
		t.Error("DefaultPasswordRequirements().RequireUpper should be true")
	}

	if !reqs.RequireLower {
		t.Error("DefaultPasswordRequirements().RequireLower should be true")
	}

	if !reqs.RequireNumber {
		t.Error("DefaultPasswordRequirements().RequireNumber should be true")
	}

	if reqs.RequireSpecial {
		t.Error("DefaultPasswordRequirements().RequireSpecial should be false")
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "valid simple email",
			email: "user@example.com",
			want:  true,
		},
		{
			name:  "valid email with subdomain",
			email: "user@mail.example.com",
			want:  true,
		},
		{
			name:  "valid email with plus",
			email: "user+tag@example.com",
			want:  true,
		},
		{
			name:  "valid email with dots",
			email: "first.last@example.com",
			want:  true,
		},
		{
			name:  "valid email with numbers",
			email: "user123@example456.com",
			want:  true,
		},
		{
			name:  "valid email with hyphen",
			email: "user@ex-ample.com",
			want:  true,
		},
		{
			name:  "invalid - no @",
			email: "userexample.com",
			want:  false,
		},
		{
			name:  "invalid - no domain",
			email: "user@",
			want:  false,
		},
		{
			name:  "invalid - no local part",
			email: "@example.com",
			want:  false,
		},
		{
			name:  "invalid - no TLD",
			email: "user@example",
			want:  false,
		},
		{
			name:  "invalid - empty string",
			email: "",
			want:  false,
		},
		{
			name:  "invalid - whitespace only",
			email: "   ",
			want:  false,
		},
		{
			name:  "invalid - spaces in email",
			email: "user name@example.com",
			want:  false,
		},
		{
			name:  "valid email with leading/trailing spaces (trimmed)",
			email: "  user@example.com  ",
			want:  true,
		},
		{
			name:  "invalid - multiple @",
			email: "user@@example.com",
			want:  false,
		},
		{
			name:  "invalid - TLD too short",
			email: "user@example.c",
			want:  false,
		},
		{
			name:  "valid - long TLD",
			email: "user@example.museum",
			want:  true,
		},
		{
			name:  "invalid - special chars in local part",
			email: "user!#$%@example.com",
			want:  false,
		},
		{
			name:  "valid - underscore in local part",
			email: "user_name@example.com",
			want:  true,
		},
		{
			name:  "valid - percent in local part",
			email: "user%test@example.com",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			if got != tt.want {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}
