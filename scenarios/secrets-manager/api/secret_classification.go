// Package main provides secret classification domain rules.
//
// This file consolidates the business logic for determining secret types,
// required status, and keyword-based classification. Previously this logic
// was duplicated across scanner.go and vault_fallback.go.
package main

import (
	"strings"
)

// secretKeywords identifies variable names that likely contain secrets.
var secretKeywords = []string{
	"PASSWORD", "PASS", "PWD", "SECRET", "KEY", "TOKEN", "AUTH", "API",
	"CREDENTIAL", "CERT", "TLS", "SSL", "PRIVATE",
}

// configKeywords identifies configuration variables that may be sensitive.
var configKeywords = []string{
	"HOST", "PORT", "URL", "ADDR", "DATABASE", "DB", "USER", "NAMESPACE",
}

// requiredKeywords identifies secrets that are typically required.
var requiredKeywords = []string{
	"PASSWORD", "SECRET", "TOKEN", "KEY", "DATABASE", "DB", "HOST",
}

// ClassifySecretType determines the secret type based on the variable name.
// Returns one of: password, token, api_key, credential, certificate, env_var.
func ClassifySecretType(varName string) string {
	upperVar := strings.ToUpper(varName)

	switch {
	case strings.Contains(upperVar, "PASSWORD") ||
		strings.Contains(upperVar, "PASS") ||
		strings.Contains(upperVar, "PWD"):
		return "password"
	case strings.Contains(upperVar, "TOKEN"):
		return "token"
	case strings.Contains(upperVar, "KEY") &&
		(strings.Contains(upperVar, "API") || strings.Contains(upperVar, "ACCESS")):
		return "api_key"
	case strings.Contains(upperVar, "SECRET") ||
		strings.Contains(upperVar, "CREDENTIAL"):
		return "credential"
	case strings.Contains(upperVar, "CERT") ||
		strings.Contains(upperVar, "TLS") ||
		strings.Contains(upperVar, "SSL"):
		return "certificate"
	default:
		return "env_var"
	}
}

// IsLikelySecret determines if a variable name represents a secret or sensitive config.
func IsLikelySecret(varName string) bool {
	upperVar := strings.ToUpper(varName)

	for _, keyword := range secretKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}
	for _, keyword := range configKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}
	return false
}

// IsLikelyRequired determines if a secret is typically required based on its name.
func IsLikelyRequired(varName string) bool {
	upperVar := strings.ToUpper(varName)

	for _, keyword := range requiredKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}
	return false
}

// GetSecretKeywords returns the combined list of keywords used to identify secrets.
// This provides a single source of truth for keyword-based secret detection.
func GetSecretKeywords() []string {
	result := make([]string, 0, len(secretKeywords)+len(configKeywords))
	result = append(result, secretKeywords...)
	result = append(result, configKeywords...)
	return result
}
