// Package toolregistry provides tool definitions for scenario-to-desktop.
//
// This file defines code signing tools for desktop applications.
// These tools configure and execute code signing for Windows, macOS, and Linux.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// SigningToolProvider provides code signing tools.
type SigningToolProvider struct{}

// NewSigningToolProvider creates a new SigningToolProvider.
func NewSigningToolProvider() *SigningToolProvider {
	return &SigningToolProvider{}
}

// Name returns the provider identifier.
func (p *SigningToolProvider) Name() string {
	return "scenario-to-desktop-signing"
}

// Categories returns the tool categories for signing tools.
func (p *SigningToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "code_signing",
			Name:         "Code Signing",
			Description:  "Tools for signing and notarizing desktop applications",
			Icon:         "shield-check",
			DisplayOrder: 2,
		},
	}
}

// Tools returns the signing tool definitions.
func (p *SigningToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.configureSigningTool(),
		p.signApplicationTool(),
		p.verifySignatureTool(),
		p.getSigningStatusTool(),
		p.discoverCertificatesTool(),
	}
}

// configureSigningTool returns the signing configuration tool.
func (p *SigningToolProvider) configureSigningTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "configure_signing",
		Description: "Configure code signing for a scenario. This sets up the signing certificates and credentials for Windows Authenticode, macOS notarization, or Linux GPG signing. Requires approval as it involves credential configuration.",
		Category:    "code_signing",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario to configure signing for",
				},
				"platform": {
					Type:        "string",
					Description: "Target platform: windows, macos, or linux",
				},
				"config": {
					Type:        "object",
					Description: "Platform-specific signing configuration (certificate paths, Apple ID credentials, GPG key ID, etc.)",
				},
			},
			Required: []string{"scenario_name", "platform", "config"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // Involves credentials
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"signing", "config", "credentials"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Configure macOS notarization",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"platform":      "macos",
						"config": map[string]interface{}{
							"apple_id":       "developer@example.com",
							"team_id":        "ABC123XYZ",
							"identity":       "Developer ID Application: Example Inc",
							"keychain_path":  "/path/to/keychain",
							"keychain_name":  "build.keychain",
							"notarize":       true,
							"staple":         true,
						},
					},
				),
				NewToolExample(
					"Configure Windows Authenticode",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"platform":      "windows",
						"config": map[string]interface{}{
							"certificate_path":     "/path/to/cert.pfx",
							"timestamp_server":     "http://timestamp.digicert.com",
							"sign_algorithm":       "sha256",
						},
					},
				),
			},
		},
	}
}

// signApplicationTool returns the application signing tool (async for notarization).
func (p *SigningToolProvider) signApplicationTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "sign_application",
		Description: "Sign a built desktop application with configured certificates. For macOS, this includes notarization which is a long-running async operation (10-30 minutes). Requires approval as it modifies artifacts.",
		Category:    "code_signing",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario whose built application to sign",
				},
				"artifact_path": {
					Type:        "string",
					Description: "Path to the built artifact (e.g., .exe, .app, .dmg, .AppImage)",
				},
				"platform": {
					Type:        "string",
					Description: "Target platform: windows, macos, or linux",
				},
			},
			Required: []string{"scenario_name", "artifact_path", "platform"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // Modifies artifacts
			TimeoutSeconds:     60,
			RateLimitPerMinute: 5,
			CostEstimate:       "medium",
			LongRunning:        true, // Notarization is slow
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"signing", "notarization", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Sign macOS DMG",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"artifact_path": "/path/to/AgentInbox-1.0.0.dmg",
						"platform":      "macos",
					},
				),
				NewToolExample(
					"Sign Windows installer",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"artifact_path": "/path/to/AgentInbox-Setup-1.0.0.exe",
						"platform":      "windows",
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "get_signing_status",
					OperationIdField:       "signing_id",
					StatusToolIdParam:      "signing_id",
					PollIntervalSeconds:    30, // Notarization is slow
					MaxPollDurationSeconds: 3600,
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"signed", "notarized"},
					FailureValues: []string{"failed", "rejected"},
					PendingValues: []string{"signing", "notarizing", "pending"},
				},
			},
		},
	}
}

// verifySignatureTool returns the signature verification tool.
func (p *SigningToolProvider) verifySignatureTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "verify_signature",
		Description: "Verify the code signature of a built desktop application. Returns detailed information about the signature including certificate chain, timestamp, and validity.",
		Category:    "code_signing",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"artifact_path": {
					Type:        "string",
					Description: "Path to the signed artifact to verify",
				},
				"platform": {
					Type:        "string",
					Description: "Platform of the artifact: windows, macos, or linux",
				},
			},
			Required: []string{"artifact_path", "platform"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"signing", "verify", "validation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Verify macOS app signature",
					map[string]interface{}{
						"artifact_path": "/path/to/AgentInbox.app",
						"platform":      "macos",
					},
				),
			},
		},
	}
}

// getSigningStatusTool returns the signing status check tool.
func (p *SigningToolProvider) getSigningStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_signing_status",
		Description: "Get the signing configuration status and readiness for a scenario. Also used to poll async signing/notarization operations.",
		Category:    "code_signing",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario to check signing status for",
				},
				"signing_id": {
					Type:        "string",
					Description: "Optional signing operation ID to check status of a specific operation",
				},
			},
			Required: []string{"scenario_name"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"signing", "status", "polling"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check signing readiness",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
					},
				),
				NewToolExample(
					"Check notarization progress",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"signing_id":    "sign-abc123",
					},
				),
			},
		},
	}
}

// discoverCertificatesTool returns the certificate discovery tool.
func (p *SigningToolProvider) discoverCertificatesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "discover_certificates",
		Description: "Discover available signing certificates on the system. For macOS, lists certificates in keychains. For Windows, lists certificates in the certificate store. For Linux, lists GPG keys.",
		Category:    "code_signing",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"platform": {
					Type:        "string",
					Description: "Platform to discover certificates for: windows, macos, or linux",
				},
			},
			Required: []string{"platform"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 20,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"signing", "certificates", "discovery"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Discover macOS certificates",
					map[string]interface{}{
						"platform": "macos",
					},
				),
			},
		},
	}
}
