import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { getExecutionConfig, generateConfigurationErrorMessage } from "../tasks/sandbox/executionConfig.js";

/**
 * Determines the current execution mode based on configuration and environment
 */
export type ExecutionMode = "local" | "web";

/**
 * Check if local execution mode is enabled and safe
 * This is the single source of truth for whether dangerous operations are allowed
 */
export function isLocalExecutionEnabled(): boolean {
    try {
        console.log("=== isLocalExecutionEnabled: Starting check ===");
        console.log("=== NODE_ENV:", process.env.NODE_ENV);
        console.log("=== PROJECT_DIR:", process.env.PROJECT_DIR);
        
        const config = getExecutionConfig();
        console.log("=== Config loaded:", { enabled: config.enabled });
        
        const isDevOrTest = process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";
        console.log("=== isDevOrTest:", isDevOrTest);
        
        const result = config.enabled && isDevOrTest;
        console.log("=== isLocalExecutionEnabled result:", result);
        
        return result;
    } catch (error) {
        // If we can't load config, default to disabled for safety
        console.log("=== isLocalExecutionEnabled error:", error);
        logger.warn("Failed to load execution config, defaulting to disabled", {
            error: error instanceof Error ? error.message : String(error),
            trace: "isLocalExecutionEnabled",
        });
        return false;
    }
}

/**
 * Get the current execution mode
 */
export function getExecutionMode(): ExecutionMode {
    return isLocalExecutionEnabled() ? "local" : "web";
}

/**
 * Validates that local execution is safe and allowed
 * Throws an error if local execution is not permitted
 */
export function validateLocalExecutionSafety(): void {
    const config = getExecutionConfig();
    const isDevOrTest = process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";

    if (!isDevOrTest) {
        throw new CustomError("0630", "Unauthorized", { 
            reason: "Local execution requires development or test environment", 
        });
    }

    if (!config.enabled) {
        const errorMessage = generateConfigurationErrorMessage();
        throw new CustomError("0630", "Unauthorized", { reason: errorMessage });
    }

    logger.info("Local execution safety check passed", {
        environment: process.env.NODE_ENV,
        configEnabled: config.enabled,
        mode: getExecutionMode(),
        trace: "validateLocalExecutionSafety",
    });
}

/**
 * Check if a specific dangerous operation should be allowed
 * This can be extended in the future for more granular control
 */
export function isDangerousOperationAllowed(operation: "bash" | "file-write" | "claude-code"): boolean {
    // For now, all dangerous operations follow the same rule
    // In the future, we might want different rules for different operations
    return isLocalExecutionEnabled();
}

/**
 * Get a human-readable description of why local execution is disabled
 */
export function getLocalExecutionDisabledReason(): string {
    const config = getExecutionConfig();
    const isDevOrTest = process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";
    
    if (!isDevOrTest) {
        return "Local execution is disabled in production environments for security";
    }
    
    if (!config.enabled) {
        return "Local execution is disabled in configuration. " + generateConfigurationErrorMessage();
    }
    
    return "Local execution is enabled";
}
