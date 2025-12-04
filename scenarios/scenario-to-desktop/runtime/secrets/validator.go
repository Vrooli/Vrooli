package secrets

import (
	"fmt"
	"regexp"
	"strings"

	"scenario-to-desktop-runtime/manifest"
)

// ValidationError represents a secret validation failure.
type ValidationError struct {
	SecretID string
	Message  string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("secret %s: %s", e.SecretID, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

// Validator validates secrets against manifest definitions.
type Validator struct {
	manifest *manifest.Manifest
	patterns map[string]*regexp.Regexp // Cached compiled patterns
}

// NewValidator creates a new Validator for the given manifest.
func NewValidator(m *manifest.Manifest) *Validator {
	return &Validator{
		manifest: m,
		patterns: make(map[string]*regexp.Regexp),
	}
}

// Validate checks secrets against manifest requirements (required, format).
// Returns ValidationErrors if any secrets fail validation.
func (v *Validator) Validate(secrets map[string]string) ValidationErrors {
	var errs ValidationErrors

	for _, sec := range v.manifest.Secrets {
		value := strings.TrimSpace(secrets[sec.ID])

		// Check required
		required := true
		if sec.Required != nil {
			required = *sec.Required
		}

		if value == "" {
			if required {
				errs = append(errs, ValidationError{
					SecretID: sec.ID,
					Message:  "required secret is missing",
				})
			}
			continue
		}

		// Check format pattern if specified
		if sec.Format != "" {
			if err := v.validateFormat(sec.ID, value, sec.Format); err != nil {
				errs = append(errs, *err)
			}
		}
	}

	return errs
}

// validateFormat checks if a value matches the secret's format pattern.
func (v *Validator) validateFormat(id, value, pattern string) *ValidationError {
	re, ok := v.patterns[id]
	if !ok {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return &ValidationError{
				SecretID: id,
				Message:  fmt.Sprintf("invalid format pattern %q: %v", pattern, err),
			}
		}
		v.patterns[id] = re
	}

	if !re.MatchString(value) {
		return &ValidationError{
			SecretID: id,
			Message:  fmt.Sprintf("value does not match required format %q", pattern),
		}
	}

	return nil
}

// ValidateSecret validates a single secret value.
func (v *Validator) ValidateSecret(id, value string) *ValidationError {
	var sec *manifest.Secret
	for i := range v.manifest.Secrets {
		if v.manifest.Secrets[i].ID == id {
			sec = &v.manifest.Secrets[i]
			break
		}
	}

	if sec == nil {
		return &ValidationError{
			SecretID: id,
			Message:  "unknown secret",
		}
	}

	value = strings.TrimSpace(value)
	required := true
	if sec.Required != nil {
		required = *sec.Required
	}

	if value == "" && required {
		return &ValidationError{
			SecretID: id,
			Message:  "required secret is missing",
		}
	}

	if value != "" && sec.Format != "" {
		return v.validateFormat(id, value, sec.Format)
	}

	return nil
}

// MissingRequired returns IDs of required secrets that are missing or empty.
func (v *Validator) MissingRequired(secrets map[string]string) []string {
	var missing []string
	for _, sec := range v.manifest.Secrets {
		required := true
		if sec.Required != nil {
			required = *sec.Required
		}
		if !required {
			continue
		}
		val := strings.TrimSpace(secrets[sec.ID])
		if val == "" {
			missing = append(missing, sec.ID)
		}
	}
	return missing
}

// InvalidFormats returns secrets that have values but fail format validation.
func (v *Validator) InvalidFormats(secrets map[string]string) ValidationErrors {
	var errs ValidationErrors
	for _, sec := range v.manifest.Secrets {
		if sec.Format == "" {
			continue
		}
		value := strings.TrimSpace(secrets[sec.ID])
		if value == "" {
			continue // Skip empty values - MissingRequired handles those
		}
		if err := v.validateFormat(sec.ID, value, sec.Format); err != nil {
			errs = append(errs, *err)
		}
	}
	return errs
}
