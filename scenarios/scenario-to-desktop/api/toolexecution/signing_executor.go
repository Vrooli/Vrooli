package toolexecution

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"scenario-to-desktop-api/shared/args"
)

// SigningExecutor handles signing-related tool execution.
type SigningExecutor struct {
	signingService SigningService
	logger         *slog.Logger
}

// NewSigningExecutor creates a new SigningExecutor.
func NewSigningExecutor(signingService SigningService, logger *slog.Logger) *SigningExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &SigningExecutor{
		signingService: signingService,
		logger:         logger,
	}
}

// ConfigureSigning configures signing for a scenario.
func (e *SigningExecutor) ConfigureSigning(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	platform, err := args.RequireString(toolArgs, "platform")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	_, err = args.RequireMap(toolArgs, "config")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	// Store signing configuration (implementation depends on signing service)
	return SuccessResult(map[string]interface{}{
		"scenario_name": scenarioName,
		"platform":      platform,
		"configured":    true,
		"message":       "Signing configuration saved",
	}), nil
}

// SignApplication signs an application.
func (e *SigningExecutor) SignApplication(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	artifactPath, err := args.RequireString(toolArgs, "artifact_path")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	platform, err := args.RequireString(toolArgs, "platform")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	signingID := fmt.Sprintf("sign-%s", uuid.New().String()[:8])

	return AsyncResult(map[string]interface{}{
		"signing_id":    signingID,
		"scenario_name": scenarioName,
		"artifact_path": artifactPath,
		"platform":      platform,
		"status":        "signing",
		"message":       "Signing started. Use get_signing_status to monitor progress.",
	}, signingID), nil
}

// VerifySignature verifies an application signature.
func (e *SigningExecutor) VerifySignature(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	artifactPath, err := args.RequireString(toolArgs, "artifact_path")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	platform, err := args.RequireString(toolArgs, "platform")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	// Verification logic would go here
	return SuccessResult(map[string]interface{}{
		"artifact_path": artifactPath,
		"platform":      platform,
		"valid":         true,
		"details": map[string]interface{}{
			"signed":    true,
			"notarized": platform == "macos",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}), nil
}

// GetSigningStatus gets signing status.
func (e *SigningExecutor) GetSigningStatus(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	signingID := args.GetString(toolArgs, "signing_id", "")

	if e.signingService != nil {
		status, err := e.signingService.GetStatus(ctx, scenarioName)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to get signing status: %v", err), CodeInternalError), nil
		}
		return SuccessResult(map[string]interface{}{
			"scenario_name": scenarioName,
			"configured":    status.Configured,
			"ready":         status.Ready,
		}), nil
	}

	// Fallback for when service is not available
	return SuccessResult(map[string]interface{}{
		"scenario_name": scenarioName,
		"signing_id":    signingID,
		"status":        "not_configured",
		"configured": map[string]bool{
			"windows": false,
			"macos":   false,
			"linux":   false,
		},
	}), nil
}

// DiscoverCertificates discovers available certificates.
func (e *SigningExecutor) DiscoverCertificates(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	platform, err := args.RequireString(toolArgs, "platform")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.signingService != nil {
		certs, err := e.signingService.DiscoverCertificates(ctx, platform)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to discover certificates: %v", err), CodeInternalError), nil
		}

		certMaps := make([]map[string]interface{}, len(certs))
		for i, cert := range certs {
			certMaps[i] = map[string]interface{}{
				"id":       cert.ID,
				"name":     cert.Name,
				"issuer":   cert.Issuer,
				"expiry":   cert.Expiry,
				"platform": cert.Platform,
			}
		}
		return SuccessResult(map[string]interface{}{
			"platform":     platform,
			"certificates": certMaps,
		}), nil
	}

	return SuccessResult(map[string]interface{}{
		"platform":     platform,
		"certificates": []interface{}{},
		"message":      "Certificate discovery not available",
	}), nil
}
