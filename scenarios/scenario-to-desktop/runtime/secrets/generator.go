// Package secrets provides secret management for the bundle runtime.
package secrets

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"scenario-to-desktop-runtime/manifest"
)

// DefaultCharset is the default character set for random secret generation.
const DefaultCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Generator handles automatic generation of secrets based on manifest templates.
type Generator struct {
	randReader RandReader
}

// RandReader abstracts random byte reading for testing.
type RandReader interface {
	Read(b []byte) (n int, err error)
}

// cryptoRandReader wraps crypto/rand for production use.
type cryptoRandReader struct{}

func (cryptoRandReader) Read(b []byte) (int, error) {
	return rand.Read(b)
}

// NewGenerator creates a new secret Generator with crypto/rand.
func NewGenerator() *Generator {
	return &Generator{randReader: cryptoRandReader{}}
}

// NewGeneratorWithRand creates a Generator with a custom random source (for testing).
func NewGeneratorWithRand(r RandReader) *Generator {
	return &Generator{randReader: r}
}

// GenerateSecrets generates values for per_install_generated secrets that don't exist.
// It returns a map of secret ID -> generated value for secrets that were generated.
func (g *Generator) GenerateSecrets(m *manifest.Manifest, existingSecrets map[string]string) (map[string]string, error) {
	generated := make(map[string]string)

	for _, sec := range m.Secrets {
		// Only generate for per_install_generated class
		if sec.Class != "per_install_generated" {
			continue
		}

		// Skip if secret already has a value
		if val, ok := existingSecrets[sec.ID]; ok && val != "" {
			continue
		}

		// Generate based on template
		value, err := g.generateFromTemplate(sec)
		if err != nil {
			return nil, fmt.Errorf("generate secret %s: %w", sec.ID, err)
		}

		generated[sec.ID] = value
	}

	return generated, nil
}

// generateFromTemplate creates a secret value based on the generator template.
func (g *Generator) generateFromTemplate(sec manifest.Secret) (string, error) {
	template := sec.Generator
	if template == nil {
		// Default: random 32-character alphanumeric
		return g.generateRandom(32, DefaultCharset)
	}

	genType, _ := template["type"].(string)
	switch genType {
	case "random", "":
		return g.generateRandomFromTemplate(template)
	default:
		return "", fmt.Errorf("unsupported generator type: %s", genType)
	}
}

// generateRandomFromTemplate generates a random string based on template config.
func (g *Generator) generateRandomFromTemplate(template map[string]any) (string, error) {
	// Extract length (default 32)
	length := 32
	if l, ok := template["length"].(float64); ok {
		length = int(l)
	} else if l, ok := template["length"].(int); ok {
		length = l
	}

	if length <= 0 {
		return "", fmt.Errorf("invalid length: %d", length)
	}

	// Extract charset (default alphanumeric)
	charset := DefaultCharset
	if c, ok := template["charset"].(string); ok && c != "" {
		switch c {
		case "alnum", "alphanumeric":
			charset = DefaultCharset
		case "alpha":
			charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		case "lower":
			charset = "abcdefghijklmnopqrstuvwxyz"
		case "upper":
			charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		case "digits", "numeric":
			charset = "0123456789"
		case "hex":
			charset = "0123456789abcdef"
		default:
			// Use the string directly as charset
			charset = c
		}
	}

	return g.generateRandom(length, charset)
}

// generateRandom creates a random string of specified length using the given charset.
func (g *Generator) generateRandom(length int, charset string) (string, error) {
	if len(charset) == 0 {
		return "", fmt.Errorf("empty charset")
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		idx, err := rand.Int(g.randReader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("random generation failed: %w", err)
		}
		result[i] = charset[idx.Int64()]
	}

	return string(result), nil
}
