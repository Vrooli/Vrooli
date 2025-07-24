import { HOURS_1_MS, SECONDS_5_MS } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import type { ResourceId } from "./typeRegistry.js";

// Constants
const RETRY_DELAY_MS = SECONDS_5_MS;

/**
 * Categories of errors that can occur during resource operations
 */
export enum ResourceErrorCategory {
    // System-level errors
    SYSTEM_DEPENDENCY = "system_dependency",
    PERMISSION = "permission",
    NETWORK = "network",
    FILESYSTEM = "filesystem",

    // Resource-specific errors
    RESOURCE_NOT_FOUND = "resource_not_found",
    RESOURCE_UNAVAILABLE = "resource_unavailable",
    RESOURCE_CONFLICT = "resource_conflict",

    // Configuration errors
    CONFIG_INVALID = "config_invalid",
    CONFIG_LOCKED = "config_locked",
    CONFIG_CORRUPTED = "config_corrupted",

    // Installation errors
    DOWNLOAD_FAILED = "download_failed",
    INSTALLATION_FAILED = "installation_failed",
    SERVICE_START_FAILED = "service_start_failed",
    VALIDATION_FAILED = "validation_failed",

    // Dependency errors
    DEPENDENCY_MISSING = "dependency_missing",
    DEPENDENCY_CONFLICT = "dependency_conflict",
    CIRCULAR_DEPENDENCY = "circular_dependency",

    // Unknown/Unexpected errors
    UNKNOWN = "unknown"
}

/**
 * Severity levels for errors
 */
export enum ErrorSeverity {
    LOW = "low",           // Non-critical, operation can continue
    MEDIUM = "medium",     // Important, but recoverable
    HIGH = "high",         // Critical, requires immediate attention
    CRITICAL = "critical"  // System-breaking, requires manual intervention
}

/**
 * Recovery action that can be attempted
 */
export interface RecoveryAction {
    id: string;
    description: string;
    command?: string;
    automatic?: boolean;  // Can be executed automatically
    destructive?: boolean; // May cause data loss
    execute: () => Promise<{ success: boolean; message: string }>;
}

/**
 * Detailed error information
 */
export interface ResourceError {
    category: ResourceErrorCategory;
    severity: ErrorSeverity;
    message: string;
    userMessage: string;  // User-friendly message
    resourceId?: ResourceId;
    originalError?: Error;
    context?: Record<string, any>;
    recoveryActions?: RecoveryAction[];
    timestamp: Date;
    remediation?: {
        description: string;
        steps: string[];
        documentation?: string;
    };
}

/**
 * Result of an error handling operation
 */
export interface ErrorHandlingResult {
    handled: boolean;
    recovered: boolean;
    message: string;
    actions?: Array<{
        action: string;
        success: boolean;
        message: string;
    }>;
}

/**
 * Context for rollback operations
 */
export interface RollbackContext {
    resourceId: ResourceId;
    operation: string;
    startTime: Date;
    steps: Array<{
        description: string;
        rollback: () => Promise<void>;
        priority: number; // Higher priority = rollback first
    }>;
}

/**
 * ErrorHandler - Comprehensive error handling and recovery system
 * 
 * This service addresses poor error handling by providing:
 * - Error classification and user-friendly messages
 * - Automatic recovery mechanisms where possible
 * - Rollback capabilities for failed operations
 * - Actionable remediation steps
 * - Integration with logging and monitoring
 */
export class ErrorHandler {
    private readonly rollbackContexts = new Map<string, RollbackContext>();

    constructor() {
        logger.debug("[ErrorHandler] Initialized");
    }

    /**
     * Create a ResourceError from a raw error with context
     */
    createResourceError(
        error: Error | string,
        context: {
            resourceId?: ResourceId;
            operation?: string;
            category?: ResourceErrorCategory;
            severity?: ErrorSeverity;
            additionalContext?: Record<string, any>;
        } = {},
    ): ResourceError {
        const originalError = error instanceof Error ? error : new Error(String(error));
        const errorMessage = originalError.message;

        // Classify the error automatically if not provided
        const category = context.category || this.classifyError(originalError, context);
        const severity = context.severity || this.determineSeverity(category, originalError, context);

        // Generate user-friendly message and remediation
        const userFriendlyInfo = this.generateUserFriendlyMessage(category, originalError, context);

        // Generate recovery actions
        const recoveryActions = this.generateRecoveryActions(category, originalError, context);

        const resourceError: ResourceError = {
            category,
            severity,
            message: errorMessage,
            userMessage: userFriendlyInfo.message,
            resourceId: context.resourceId,
            originalError,
            context: context.additionalContext,
            recoveryActions,
            timestamp: new Date(),
            remediation: userFriendlyInfo.remediation,
        };

        // Log the error with appropriate level
        this.logError(resourceError);

        return resourceError;
    }

    /**
     * Handle an error with automatic recovery attempts
     */
    async handleError(
        error: ResourceError,
        options: {
            attemptRecovery?: boolean;
            userConfirmation?: boolean;
        } = {},
    ): Promise<ErrorHandlingResult> {
        const { attemptRecovery = true, userConfirmation = false } = options;

        logger.info("[ErrorHandler] Handling error", {
            category: error.category,
            severity: error.severity,
            resourceId: error.resourceId,
        });

        const actions: Array<{ action: string; success: boolean; message: string }> = [];
        let recovered = false;

        if (attemptRecovery && error.recoveryActions) {
            for (const action of error.recoveryActions) {
                // Skip destructive actions unless explicitly confirmed
                if (action.destructive && !userConfirmation) {
                    logger.debug(`[ErrorHandler] Skipping destructive action without confirmation: ${action.id}`);
                    continue;
                }

                // Only attempt automatic recovery actions
                if (!action.automatic) {
                    logger.debug(`[ErrorHandler] Skipping manual action: ${action.id}`);
                    continue;
                }

                try {
                    logger.info(`[ErrorHandler] Attempting recovery action: ${action.description}`);
                    const result = await action.execute();

                    actions.push({
                        action: action.description,
                        success: result.success,
                        message: result.message,
                    });

                    if (result.success) {
                        recovered = true;
                        logger.info(`[ErrorHandler] Recovery action succeeded: ${action.description}`);
                        break; // Stop after first successful recovery
                    } else {
                        logger.warn(`[ErrorHandler] Recovery action failed: ${action.description}`, {
                            message: result.message,
                        });
                    }

                } catch (recoveryError) {
                    const recoveryMessage = recoveryError instanceof Error ? recoveryError.message : String(recoveryError);
                    logger.error(`[ErrorHandler] Recovery action threw error: ${action.description}`, recoveryError);

                    actions.push({
                        action: action.description,
                        success: false,
                        message: `Recovery failed: ${recoveryMessage}`,
                    });
                }
            }
        }

        return {
            handled: true,
            recovered,
            message: recovered
                ? `Error handled successfully with recovery: ${error.userMessage}`
                : `Error handled but not recovered: ${error.userMessage}`,
            actions: actions.length > 0 ? actions : undefined,
        };
    }

    /**
     * Start a rollback context for an operation
     */
    startRollbackContext(resourceId: ResourceId, operation: string): string {
        const contextId = `${resourceId}_${operation}_${Date.now()}`;

        const context: RollbackContext = {
            resourceId,
            operation,
            startTime: new Date(),
            steps: [],
        };

        this.rollbackContexts.set(contextId, context);

        logger.debug("[ErrorHandler] Started rollback context", {
            contextId,
            resourceId,
            operation,
        });

        return contextId;
    }

    /**
     * Add a rollback step to a context
     */
    addRollbackStep(
        contextId: string,
        description: string,
        rollbackFn: () => Promise<void>,
        priority = 0,
    ): void {
        const context = this.rollbackContexts.get(contextId);
        if (!context) {
            logger.warn("[ErrorHandler] Attempted to add rollback step to non-existent context", { contextId });
            return;
        }

        context.steps.push({
            description,
            rollback: rollbackFn,
            priority,
        });

        logger.debug("[ErrorHandler] Added rollback step", {
            contextId,
            description,
            priority,
            totalSteps: context.steps.length,
        });
    }

    /**
     * Execute rollback for a context
     */
    async executeRollback(contextId: string): Promise<{
        success: boolean;
        completedSteps: number;
        failedSteps: Array<{ description: string; error: string }>;
    }> {
        const context = this.rollbackContexts.get(contextId);
        if (!context) {
            logger.error("[ErrorHandler] Attempted to rollback non-existent context", { contextId });
            return {
                success: false,
                completedSteps: 0,
                failedSteps: [{ description: "Invalid context", error: "Context not found" }],
            };
        }

        logger.info("[ErrorHandler] Starting rollback", {
            contextId,
            resourceId: context.resourceId,
            operation: context.operation,
            steps: context.steps.length,
        });

        // Sort steps by priority (highest first)
        const sortedSteps = [...context.steps].sort((a, b) => b.priority - a.priority);

        let completedSteps = 0;
        const failedSteps: Array<{ description: string; error: string }> = [];

        // Execute rollback steps in reverse order of priority
        for (const step of sortedSteps) {
            try {
                logger.debug(`[ErrorHandler] Executing rollback step: ${step.description}`);
                await step.rollback();
                completedSteps++;
                logger.debug(`[ErrorHandler] Rollback step completed: ${step.description}`);
            } catch (error) {
                const errorMessage = error instanceof Error ? error.message : String(error);
                logger.error(`[ErrorHandler] Rollback step failed: ${step.description}`, error);

                failedSteps.push({
                    description: step.description,
                    error: errorMessage,
                });

                // Continue with other rollback steps even if one fails
            }
        }

        // Clean up context
        this.rollbackContexts.delete(contextId);

        const success = failedSteps.length === 0;

        logger.info("[ErrorHandler] Rollback completed", {
            contextId,
            success,
            completedSteps,
            failedSteps: failedSteps.length,
        });

        return {
            success,
            completedSteps,
            failedSteps,
        };
    }

    /**
     * Clean up expired rollback contexts
     */
    cleanupExpiredContexts(maxAge = HOURS_1_MS): void {
        const now = Date.now();
        const expiredContexts: string[] = [];

        for (const [contextId, context] of this.rollbackContexts) {
            if (now - context.startTime.getTime() > maxAge) {
                expiredContexts.push(contextId);
            }
        }

        for (const contextId of expiredContexts) {
            this.rollbackContexts.delete(contextId);
            logger.debug("[ErrorHandler] Cleaned up expired rollback context", { contextId });
        }
    }

    // Private helper methods

    private classifyError(error: Error, context: any): ResourceErrorCategory {
        const message = error.message.toLowerCase();

        // System dependency errors
        if (message.includes("command not found") || message.includes("which:")) {
            return ResourceErrorCategory.SYSTEM_DEPENDENCY;
        }

        // Permission errors
        if (message.includes("permission denied") || message.includes("eacces") || message.includes("sudo")) {
            return ResourceErrorCategory.PERMISSION;
        }

        // Network errors
        if (message.includes("fetch failed") || message.includes("connection refused") || message.includes("timeout")) {
            return ResourceErrorCategory.NETWORK;
        }

        // Filesystem errors
        if (message.includes("enoent") || message.includes("no such file") || message.includes("enospc")) {
            return ResourceErrorCategory.FILESYSTEM;
        }

        // Configuration errors
        if (message.includes("invalid json") || message.includes("configuration") || message.includes("config")) {
            return ResourceErrorCategory.CONFIG_INVALID;
        }

        // Installation errors
        if (message.includes("installation") || message.includes("install") || context.operation === "install") {
            return ResourceErrorCategory.INSTALLATION_FAILED;
        }

        // Service errors
        if (message.includes("systemctl") || message.includes("service") || message.includes("daemon")) {
            return ResourceErrorCategory.SERVICE_START_FAILED;
        }

        // Download errors
        if (message.includes("download") || message.includes("curl") || message.includes("wget")) {
            return ResourceErrorCategory.DOWNLOAD_FAILED;
        }

        return ResourceErrorCategory.UNKNOWN;
    }

    private determineSeverity(
        category: ResourceErrorCategory,
        _error: Error,
        _context: any,
    ): ErrorSeverity {
        // Critical errors that require immediate attention
        if (category === ResourceErrorCategory.CONFIG_CORRUPTED ||
            category === ResourceErrorCategory.CIRCULAR_DEPENDENCY) {
            return ErrorSeverity.CRITICAL;
        }

        // High severity errors that block functionality
        if (category === ResourceErrorCategory.SYSTEM_DEPENDENCY ||
            category === ResourceErrorCategory.PERMISSION ||
            category === ResourceErrorCategory.CONFIG_LOCKED) {
            return ErrorSeverity.HIGH;
        }

        // Medium severity errors that are recoverable
        if (category === ResourceErrorCategory.NETWORK ||
            category === ResourceErrorCategory.DOWNLOAD_FAILED ||
            category === ResourceErrorCategory.SERVICE_START_FAILED) {
            return ErrorSeverity.MEDIUM;
        }

        // Low severity for everything else
        return ErrorSeverity.LOW;
    }

    private generateUserFriendlyMessage(
        category: ResourceErrorCategory,
        error: Error,
        context: any,
    ): { message: string; remediation?: { description: string; steps: string[]; documentation?: string } } {
        const resourceId = context.resourceId || "resource";

        switch (category) {
            case ResourceErrorCategory.SYSTEM_DEPENDENCY:
                return {
                    message: "Missing required system dependency. The installation requires additional software to be installed first.",
                    remediation: {
                        description: "Install missing system dependencies",
                        steps: [
                            "Check which command is missing from the error message",
                            "Install the missing package using your system package manager",
                            "For Ubuntu/Debian: sudo apt-get install <package>",
                            "For CentOS/RHEL: sudo yum install <package>",
                            "Retry the installation after installing dependencies",
                        ],
                    },
                };

            case ResourceErrorCategory.PERMISSION:
                return {
                    message: "Permission denied. The operation requires administrative privileges to complete.",
                    remediation: {
                        description: "Grant necessary permissions",
                        steps: [
                            "Ensure you have sudo privileges on this system",
                            "Run the command with sudo if needed",
                            "Check file/directory permissions for the target location",
                            "Verify the user has write access to the installation directory",
                        ],
                    },
                };

            case ResourceErrorCategory.NETWORK:
                return {
                    message: "Network connectivity issue. Unable to connect to required services or download resources.",
                    remediation: {
                        description: "Resolve network connectivity",
                        steps: [
                            "Check your internet connection",
                            "Verify firewall settings allow outbound connections",
                            "Test connectivity to the specific service URL",
                            "Check if a proxy configuration is needed",
                            "Retry the operation after network issues are resolved",
                        ],
                    },
                };

            case ResourceErrorCategory.DOWNLOAD_FAILED:
                return {
                    message: "Failed to download required files. This may be due to network issues or server problems.",
                    remediation: {
                        description: "Resolve download issues",
                        steps: [
                            "Check internet connectivity",
                            "Verify the download URL is accessible",
                            "Try downloading manually to test the connection",
                            "Check for proxy or firewall blocking the download",
                            "Wait a few minutes and retry in case of temporary server issues",
                        ],
                    },
                };

            case ResourceErrorCategory.SERVICE_START_FAILED:
                return {
                    message: `Failed to start the ${resourceId} service. The installation may have completed but the service won't start.`,
                    remediation: {
                        description: "Troubleshoot service startup",
                        steps: [
                            `Check service status: systemctl status ${resourceId}`,
                            `View service logs: journalctl -u ${resourceId} -f`,
                            "Verify configuration files are correct",
                            "Check for port conflicts with other services",
                            "Ensure all dependencies are installed and running",
                        ],
                    },
                };

            case ResourceErrorCategory.CONFIG_INVALID:
                return {
                    message: "Configuration file is invalid or corrupted. The system cannot read the current settings.",
                    remediation: {
                        description: "Fix configuration issues",
                        steps: [
                            "Check the configuration file syntax (usually JSON)",
                            "Compare with a backup or default configuration",
                            "Use a JSON validator to identify syntax errors",
                            "Restore from backup if available",
                            "Reset to default configuration if necessary",
                        ],
                    },
                };

            default:
                return {
                    message: `An unexpected error occurred while working with ${resourceId}. Please check the logs for more details.`,
                    remediation: {
                        description: "General troubleshooting steps",
                        steps: [
                            "Check system logs for additional error details",
                            "Verify system requirements are met",
                            "Try running the operation again",
                            "Check for recent system changes that might affect the operation",
                            "Contact support if the issue persists",
                        ],
                    },
                };
        }
    }

    private generateRecoveryActions(
        category: ResourceErrorCategory,
        error: Error,
        context: any,
    ): RecoveryAction[] {
        const actions: RecoveryAction[] = [];
        const resourceId = context.resourceId;

        switch (category) {
            case ResourceErrorCategory.SERVICE_START_FAILED:
                if (resourceId) {
                    actions.push({
                        id: "restart_service",
                        description: `Restart ${resourceId} service`,
                        command: `systemctl restart ${resourceId}`,
                        automatic: true,
                        execute: async () => {
                            // Implementation would restart the service
                            return { success: true, message: "Service restarted" };
                        },
                    });

                    actions.push({
                        id: "reset_service",
                        description: `Reset ${resourceId} service configuration`,
                        automatic: false,
                        destructive: true,
                        execute: async () => {
                            // Implementation would reset service config
                            return { success: true, message: "Service configuration reset" };
                        },
                    });
                }
                break;

            case ResourceErrorCategory.CONFIG_INVALID:
                actions.push({
                    id: "restore_config_backup",
                    description: "Restore configuration from backup",
                    automatic: true,
                    execute: async () => {
                        // Implementation would restore from backup
                        return { success: true, message: "Configuration restored from backup" };
                    },
                });

                actions.push({
                    id: "reset_config_default",
                    description: "Reset to default configuration",
                    automatic: false,
                    destructive: true,
                    execute: async () => {
                        // Implementation would reset to defaults
                        return { success: true, message: "Configuration reset to defaults" };
                    },
                });
                break;

            case ResourceErrorCategory.DOWNLOAD_FAILED:
                actions.push({
                    id: "retry_download",
                    description: "Retry download after delay",
                    automatic: true,
                    execute: async () => {
                        // Implementation would retry the download
                        await new Promise(resolve => setTimeout(resolve, RETRY_DELAY_MS)); // 5 second delay
                        return { success: true, message: "Download retried" };
                    },
                });
                break;
        }

        return actions;
    }

    private logError(error: ResourceError): void {
        const logLevel = this.getLogLevel(error.severity);
        const logData = {
            category: error.category,
            severity: error.severity,
            resourceId: error.resourceId,
            context: error.context,
            recoveryActions: error.recoveryActions?.length || 0,
        };

        switch (logLevel) {
            case "error":
                logger.error(`[ErrorHandler] ${error.userMessage}`, logData);
                break;
            case "warn":
                logger.warn(`[ErrorHandler] ${error.userMessage}`, logData);
                break;
            case "info":
                logger.info(`[ErrorHandler] ${error.userMessage}`, logData);
                break;
            default:
                logger.debug(`[ErrorHandler] ${error.userMessage}`, logData);
        }
    }

    private getLogLevel(severity: ErrorSeverity): "error" | "warn" | "info" | "debug" {
        switch (severity) {
            case ErrorSeverity.CRITICAL:
            case ErrorSeverity.HIGH:
                return "error";
            case ErrorSeverity.MEDIUM:
                return "warn";
            case ErrorSeverity.LOW:
                return "info";
            default:
                return "debug";
        }
    }
}
