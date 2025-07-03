/**
 * SwarmStateAccessor - Safe data access with security checks
 * 
 * This component provides controlled access to swarm state data with
 * built-in security validation and sensitive data protection.
 * 
 * Key Features:
 * - Pre/post action safety events for sensitive data
 * - Configurable access controls based on data sensitivity
 * - Batch access optimization for multiple data points
 * - Comprehensive audit logging for security compliance
 */

import type { DataSensitivityConfig } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { valueFromDot } from "@vrooli/shared";

/**
 * Execution context for data access operations
 */
export interface ExecutionContext {
    /** ID of the agent requesting access */
    agentId: string;
    /** ID of the current swarm */
    swarmId: string;
    /** ID of the user (if applicable) */
    userId?: string;
    /** Swarm configuration with secrets and state */
    swarmConfig: {
        secrets?: Record<string, DataSensitivityConfig>;
        [key: string]: unknown;
    };
    /** Additional context data */
    [key: string]: unknown;
}

/**
 * Options for data access operations
 */
export interface AccessOptions {
    /** Transform the data before returning */
    transform?: (data: unknown) => unknown;
    /** Cache the result for subsequent access */
    cache?: boolean;
    /** Timeout for the access operation in milliseconds */
    timeout?: number;
}

/**
 * Request for batch data access
 */
export interface DataAccessRequest {
    /** Path to the data (dot notation) */
    path: string;
    /** Options for this specific request */
    options?: AccessOptions;
}

/**
 * Safety event for auditing and control
 */
interface SafetyEvent {
    type: "safety/pre_action" | "safety/post_action";
    timestamp: number;
    source: {
        agentId: string;
        component: string;
    };
    data: {
        action: string;
        path: string;
        sensitivity: string;
        result?: unknown;
    };
}

/**
 * Security error for blocked access
 */
export class SecurityError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "SecurityError";
    }
}

/**
 * SwarmStateAccessor provides secure, audited access to swarm state data
 */
export class SwarmStateAccessor {
    private readonly eventBus?: any; // EventBus interface would be defined elsewhere
    private readonly securityValidator?: any; // SecurityValidator interface would be defined elsewhere

    constructor(
        eventBus?: any,
        securityValidator?: any,
    ) {
        this.eventBus = eventBus;
        this.securityValidator = securityValidator;
    }

    /**
     * Safely access data from swarm state with security checks
     */
    async accessData<T>(
        path: string,
        context: ExecutionContext,
        options?: AccessOptions,
    ): Promise<T> {
        // 1. Check data sensitivity
        const sensitivity = this.checkSensitivity(path, context);
        
        // 2. Emit pre-action safety event if sensitive
        if (sensitivity) {
            await this.emitSafetyEvent("pre_action", path, context, sensitivity);
        }
        
        // 3. Validate access permissions
        await this.validateAccess(path, context);
        
        // 4. Retrieve data
        const data = this.retrieveData(path, context);
        
        // 5. Emit post-action safety event
        if (sensitivity) {
            await this.emitSafetyEvent("post_action", path, context, sensitivity, data);
        }
        
        // 6. Apply data transformations if needed
        return this.transformData(data, options);
    }

    /**
     * Batch access multiple data points with optimization
     */
    async accessMultiple(
        requests: DataAccessRequest[],
        context: ExecutionContext,
    ): Promise<Map<string, unknown>> {
        // Group by sensitivity level for efficient processing
        const grouped = this.groupBySensitivity(requests, context);
        
        // Process each group with appropriate safety measures
        const results = new Map<string, unknown>();
        
        for (const [sensitivity, paths] of grouped) {
            if (sensitivity) {
                // Batch safety check for sensitive data
                await this.batchSafetyCheck(paths, context, sensitivity);
            }
            
            // Retrieve all data in group
            for (const request of paths) {
                const data = await this.accessData(request.path, context, request.options);
                results.set(request.path, data);
            }
        }
        
        return results;
    }

    /**
     * Check if data path is sensitive
     */
    private checkSensitivity(
        path: string,
        context: ExecutionContext,
    ): DataSensitivityConfig | null {
        const secrets = context.swarmConfig?.secrets;
        if (!secrets) return null;

        // Check if path matches any secret pattern
        for (const [pattern, sensitivity] of Object.entries(secrets)) {
            if (this.pathMatchesPattern(path, pattern)) {
                return sensitivity;
            }
        }

        return null;
    }

    /**
     * Emit safety event for auditing and control
     */
    private async emitSafetyEvent(
        type: "pre_action" | "post_action",
        path: string,
        context: ExecutionContext,
        sensitivity: DataSensitivityConfig,
        data?: unknown,
    ): Promise<void> {
        const event: SafetyEvent = {
            type: `safety/${type}`,
            timestamp: Date.now(),
            source: {
                agentId: context.agentId,
                component: "SwarmStateAccessor",
            },
            data: {
                action: "data_access",
                path,
                sensitivity: sensitivity.type,
                ...(type === "post_action" && { 
                    result: this.sanitizeForLogging(data, sensitivity), 
                }),
            },
        };
        
        if (this.eventBus) {
            const response = await this.eventBus.emit(event);
            
            if (response?.blocked) {
                throw new SecurityError(
                    `Access blocked by safety system: ${response.reason}`,
                );
            }
        } else {
            // Fallback logging when no event bus is available
            logger.info("[SwarmStateAccessor] Safety event", event);
        }
    }

    /**
     * Validate access permissions for the given path
     */
    private async validateAccess(
        path: string,
        context: ExecutionContext,
    ): Promise<void> {
        // Basic validation - in full implementation this would use
        // the securityValidator to check permissions
        if (!context.agentId) {
            throw new SecurityError("No agent ID provided for data access");
        }

        // Additional validation logic would go here
        logger.debug("[SwarmStateAccessor] Access validated", {
            path,
            agentId: context.agentId,
        });
    }

    /**
     * Retrieve data from the context using dot notation
     */
    private retrieveData(
        path: string,
        context: ExecutionContext,
    ): unknown {
        try {
            return valueFromDot(context, path);
        } catch (error) {
            logger.error("[SwarmStateAccessor] Failed to retrieve data", {
                path,
                error,
            });
            return undefined;
        }
    }

    /**
     * Transform data based on access options
     */
    private transformData<T>(
        data: unknown,
        options?: AccessOptions,
    ): T {
        if (options?.transform) {
            return options.transform(data) as T;
        }
        return data as T;
    }

    /**
     * Group requests by sensitivity level for batch processing
     */
    private groupBySensitivity(
        requests: DataAccessRequest[],
        context: ExecutionContext,
    ): Map<DataSensitivityConfig | null, DataAccessRequest[]> {
        const grouped = new Map<DataSensitivityConfig | null, DataAccessRequest[]>();

        for (const request of requests) {
            const sensitivity = this.checkSensitivity(request.path, context);
            
            if (!grouped.has(sensitivity)) {
                grouped.set(sensitivity, []);
            }
            grouped.get(sensitivity)!.push(request);
        }

        return grouped;
    }

    /**
     * Perform batch safety check for multiple paths
     */
    private async batchSafetyCheck(
        requests: DataAccessRequest[],
        context: ExecutionContext,
        sensitivity: DataSensitivityConfig,
    ): Promise<void> {
        // Emit batch pre-action safety event
        const paths = requests.map(req => req.path);
        
        logger.info("[SwarmStateAccessor] Batch safety check", {
            paths,
            sensitivity: sensitivity.type,
            agentId: context.agentId,
        });

        // In full implementation, this would emit proper safety events
        // for the entire batch and handle batch approval/rejection
    }

    /**
     * Check if path matches pattern (simple glob-like matching)
     */
    private pathMatchesPattern(path: string, pattern: string): boolean {
        // Convert glob pattern to regex
        const regexPattern = pattern
            .replace(/\*/g, ".*")
            .replace(/\?/g, ".")
            .replace(/\./g, "\\.");
        
        const regex = new RegExp(`^${regexPattern}$`);
        return regex.test(path);
    }

    /**
     * Sanitize data for logging based on sensitivity
     */
    private sanitizeForLogging(
        data: unknown,
        sensitivity: DataSensitivityConfig,
    ): unknown {
        if (!data) return data;

        // High sensitivity data should be completely redacted
        if (["PII", "PHI", "FINANCIAL", "CREDENTIAL"].includes(sensitivity.type)) {
            return "[REDACTED]";
        }

        // Proprietary data might be partially visible
        if (sensitivity.type === "PROPRIETARY") {
            if (typeof data === "string") {
                return data.length > 20 ? `${data.substring(0, 20)}...` : data;
            }
            if (typeof data === "object") {
                return "[OBJECT]";
            }
        }

        return data;
    }
}
