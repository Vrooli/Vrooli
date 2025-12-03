// Package main provides bundle secret plan derivation for deployment manifests.
//
// The BundlePlanBuilder converts deployment secret entries into bundle-ready
// secret plans that can be consumed by desktop bundle runtimes. It determines:
//   - Secret classification (user_prompt, per_install_generated, remote_fetch, infrastructure)
//   - Target injection method (env var, file)
//   - Prompt metadata for user-provided secrets
//   - Generator templates for auto-generated secrets
package main

import (
	"fmt"
	"strings"
)

// BundlePlanBuilder derives bundle secret plans from deployment entries.
type BundlePlanBuilder struct{}

// NewBundlePlanBuilder creates a new BundlePlanBuilder instance.
func NewBundlePlanBuilder() *BundlePlanBuilder {
	return &BundlePlanBuilder{}
}

// DeriveBundlePlans creates BundleSecretPlan entries from deployment secrets.
// Infrastructure secrets are excluded from bundle plans as they are handled
// separately by the deployment infrastructure.
func (b *BundlePlanBuilder) DeriveBundlePlans(entries []DeploymentSecretEntry) []BundleSecretPlan {
	plans := make([]BundleSecretPlan, 0, len(entries))
	for _, entry := range entries {
		if plan := b.bundleSecretFromEntry(entry); plan != nil {
			plans = append(plans, *plan)
		}
	}
	return plans
}

// bundleSecretFromEntry converts a single deployment entry to a bundle plan.
// Returns nil for infrastructure secrets, which are not included in bundles.
func (b *BundlePlanBuilder) bundleSecretFromEntry(entry DeploymentSecretEntry) *BundleSecretPlan {
	class := b.deriveSecretClass(entry)

	// Infrastructure secrets are not emitted into bundle plans
	if class == "infrastructure" {
		return nil
	}

	plan := &BundleSecretPlan{
		ID:          b.deriveSecretID(entry),
		Class:       class,
		Required:    entry.Required,
		Description: entry.Description,
		Format:      entry.ValidationPattern,
		Target:      b.deriveBundleTarget(entry),
	}

	// Add prompt metadata for user-provided secrets
	if class == "user_prompt" {
		plan.Prompt = b.derivePrompt(entry)
	}

	// Add generator template for auto-generated secrets
	if class == "per_install_generated" {
		plan.Generator = b.deriveGenerator(entry)
	}

	return plan
}

// deriveSecretClass determines the bundle secret classification.
//
// Classification rules:
//   - "infrastructure" classification maps to infrastructure class
//   - "prompt" handling strategy maps to user_prompt class
//   - "generate" handling strategy maps to per_install_generated class
//   - "delegate" handling strategy maps to remote_fetch class
//   - "strip" handling strategy maps to infrastructure class
//   - RequiresUserInput=true defaults to user_prompt class
//   - Otherwise defaults to per_install_generated class
func (b *BundlePlanBuilder) deriveSecretClass(entry DeploymentSecretEntry) string {
	// Check classification first
	if strings.EqualFold(entry.Classification, "infrastructure") {
		return "infrastructure"
	}

	// Check handling strategy
	switch strings.ToLower(entry.HandlingStrategy) {
	case "prompt":
		return "user_prompt"
	case "generate":
		return "per_install_generated"
	case "delegate":
		return "remote_fetch"
	case "strip":
		return "infrastructure"
	}

	// Fall back to user input flag
	if entry.RequiresUserInput {
		return "user_prompt"
	}

	// Default to generated
	return "per_install_generated"
}

// deriveSecretID generates a stable identifier for the bundle secret.
// Uses the entry ID if available, otherwise constructs from resource and key.
func (b *BundlePlanBuilder) deriveSecretID(entry DeploymentSecretEntry) string {
	if id := strings.TrimSpace(entry.ID); id != "" {
		return id
	}

	key := strings.TrimSpace(entry.SecretKey)
	resource := strings.TrimSpace(entry.ResourceName)

	if key == "" && resource == "" {
		return "secret"
	}
	if resource == "" {
		return strings.ToLower(key)
	}
	if key == "" {
		return strings.ToLower(resource)
	}
	return strings.ToLower(fmt.Sprintf("%s_%s", resource, key))
}

// deriveBundleTarget determines how the secret should be injected at runtime.
// Defaults to environment variable injection, but can be overridden via bundle hints.
func (b *BundlePlanBuilder) deriveBundleTarget(entry DeploymentSecretEntry) BundleSecretTarget {
	targetType := "env"
	targetName := strings.ToUpper(entry.SecretKey)

	if entry.BundleHints != nil {
		// Check for explicit target type
		if v, ok := entry.BundleHints["target_type"].(string); ok && strings.TrimSpace(v) != "" {
			targetType = strings.TrimSpace(v)
		}

		// Check for explicit target name
		if v, ok := entry.BundleHints["target_name"].(string); ok && strings.TrimSpace(v) != "" {
			targetName = strings.TrimSpace(v)
		} else if v, ok := entry.BundleHints["file_path"].(string); ok && strings.TrimSpace(v) != "" {
			// File path implies file target type
			targetType = "file"
			targetName = strings.TrimSpace(v)
		}
	}

	return BundleSecretTarget{
		Type: targetType,
		Name: targetName,
	}
}

// derivePrompt builds prompt metadata for user_prompt secrets.
// Uses existing prompt metadata if available, otherwise generates defaults.
func (b *BundlePlanBuilder) derivePrompt(entry DeploymentSecretEntry) *PromptMetadata {
	// Use existing prompt if it has meaningful content
	if entry.Prompt != nil {
		if strings.TrimSpace(entry.Prompt.Label) != "" || strings.TrimSpace(entry.Prompt.Description) != "" {
			return entry.Prompt
		}
	}

	// Generate default prompt
	label := strings.TrimSpace(entry.SecretKey)
	if label == "" {
		label = "Provide secret"
	}

	description := entry.Description
	if strings.TrimSpace(description) == "" {
		description = "Enter the value for this bundled secret."
	}

	return &PromptMetadata{
		Label:       label,
		Description: description,
	}
}

// deriveGenerator creates a generator template for per_install_generated secrets.
// Uses the entry's generator template if available, otherwise provides defaults.
func (b *BundlePlanBuilder) deriveGenerator(entry DeploymentSecretEntry) map[string]interface{} {
	if entry.GeneratorTemplate != nil && len(entry.GeneratorTemplate) > 0 {
		return entry.GeneratorTemplate
	}

	// Default generator for random tokens
	return map[string]interface{}{
		"type":    "random",
		"length":  32,
		"charset": "alnum",
	}
}
