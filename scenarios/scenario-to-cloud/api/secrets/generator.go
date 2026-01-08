package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"scenario-to-cloud/domain"
)

// GeneratedSecret holds a generated secret with metadata.
type GeneratedSecret struct {
	ID    string `json:"id"`
	Key   string `json:"key"`   // env var name (e.g., POSTGRES_PASSWORD)
	Value string `json:"value"` // generated value
}

// GeneratorFunc defines the interface for generating secrets.
// This seam enables testing with deterministic values instead of crypto/rand.
type GeneratorFunc interface {
	GenerateSecrets(plans []domain.BundleSecretPlan) ([]GeneratedSecret, error)
}

// Generator generates secrets for per_install_generated class.
// Generator implements GeneratorFunc.
type Generator struct{}

// Ensure Generator implements GeneratorFunc at compile time.
var _ GeneratorFunc = (*Generator)(nil)

// NewGenerator creates a new Generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateSecrets processes bundle secret plans and generates per_install_generated secrets.
// Only secrets with class "per_install_generated" are processed.
func (g *Generator) GenerateSecrets(plans []domain.BundleSecretPlan) ([]GeneratedSecret, error) {
	var generated []GeneratedSecret

	for _, plan := range plans {
		if plan.Class != "per_install_generated" {
			continue
		}

		value, err := g.generateValue(plan)
		if err != nil {
			return nil, fmt.Errorf("generate %s: %w", plan.ID, err)
		}

		generated = append(generated, GeneratedSecret{
			ID:    plan.ID,
			Key:   plan.Target.Name,
			Value: value,
		})
	}

	return generated, nil
}

// generateValue generates a secret value based on the generator spec.
func (g *Generator) generateValue(plan domain.BundleSecretPlan) (string, error) {
	gen := plan.Generator
	if gen == nil {
		// Default: 25-char alphanumeric (matches postgres::common::generate_password)
		return g.generateRandom(25, "alnum")
	}

	genType, _ := gen["type"].(string)
	length := 25 // Default to match postgres::common::generate_password
	if l, ok := gen["length"].(float64); ok {
		length = int(l)
	}
	charset := "alnum"
	if c, ok := gen["charset"].(string); ok {
		charset = c
	}

	switch genType {
	case "random", "password", "":
		return g.generateRandom(length, charset)
	case "uuid":
		return g.generateUUID()
	default:
		return g.generateRandom(length, charset)
	}
}

// generateRandom generates a random string matching postgres::common::generate_password() format.
// Uses crypto/rand for cryptographic randomness, similar to openssl rand -base64.
func (g *Generator) generateRandom(length int, charset string) (string, error) {
	if length <= 0 {
		length = 25
	}

	// Generate more bytes than needed, then filter
	// We need extra bytes because base64 encoding and filtering reduces usable chars
	rawLen := length * 2
	raw := make([]byte, rawLen)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}

	// Base64 encode (similar to openssl rand -base64)
	encoded := base64.StdEncoding.EncodeToString(raw)

	// Remove =, +, / to match postgres::common::generate_password behavior
	// (tr -d "=+/" in the shell script)
	cleaned := strings.Map(func(r rune) rune {
		if r == '=' || r == '+' || r == '/' {
			return -1 // remove
		}
		return r
	}, encoded)

	// Apply charset filter if specified
	switch charset {
	case "alnum", "alphanumeric":
		// Already mostly alphanumeric from base64 minus special chars
		re := regexp.MustCompile(`[^A-Za-z0-9]`)
		cleaned = re.ReplaceAllString(cleaned, "")
	case "alpha", "alphabetic":
		re := regexp.MustCompile(`[^A-Za-z]`)
		cleaned = re.ReplaceAllString(cleaned, "")
	case "numeric", "digits":
		re := regexp.MustCompile(`[^0-9]`)
		cleaned = re.ReplaceAllString(cleaned, "")
	case "hex":
		re := regexp.MustCompile(`[^A-Fa-f0-9]`)
		cleaned = re.ReplaceAllString(cleaned, "")
	case "base64":
		// Keep as-is after standard base64 encoding
		cleaned = encoded
	}

	// Ensure we have enough characters
	if len(cleaned) < length {
		// Recursive call if we don't have enough chars
		more, err := g.generateRandom(length-len(cleaned), charset)
		if err != nil {
			return "", err
		}
		cleaned = cleaned + more
	}

	return cleaned[:length], nil
}

// generateUUID generates a UUID v4 string.
func (g *Generator) generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed for UUID: %w", err)
	}
	// UUID v4 format: set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
