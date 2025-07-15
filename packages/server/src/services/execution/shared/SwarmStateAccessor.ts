/**
 * SwarmStateAccessor - Safe data access with security checks
 * 
 * This component provides controlled access to swarm state data with
 * built-in security validation and sensitive data protection.
 * 
 * Key Features:
 * - Works directly with TriggerContext for agent-friendly paths
 * - Builds TriggerContext from SwarmState for JEXL evaluation
 * - Pre/post action safety events for sensitive data
 * - Configurable access controls based on data sensitivity
 * - Batch access optimization for multiple data points
 * - Comprehensive audit logging for security compliance
 * 
 * Permission Hierarchy:
 * 1. SwarmPolicy (global swarm visibility: open/restricted/private)
 * 2. Agent presence in swarm execution context
 * 3. Agent ResourceSpec permissions (resource type and scope matching)
 * 4. Pre-computed permissions (if available)
 * 
 * Path Examples (agent-friendly format):
 * - "blackboard.key" - Access blackboard data
 * - "swarm.resources.remaining.credits" - Check remaining credits
 * - "swarm.state" - Get execution state
 * - "goal" - Access swarm goal
 * - "subtasks[0].status" - Access subtask status
 */

import type { BlackboardItem, BotParticipant, DataSensitivityConfig, ResourceSpec, SwarmState, TriggerContext } from "@vrooli/shared";
import { DataSensitivityType, EventTypes, SECONDS_1_MS, valueFromDot } from "@vrooli/shared";
import { CustomError } from "../../../events/error.js";
import { logger } from "../../../events/logger.js";
import { EventPublisher } from "../../events/publisher.js";
import type { ServiceEvent } from "../../events/types.js";

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
 * SwarmStateAccessor provides secure, audited access to swarm state data
 */
export class SwarmStateAccessor {
    /**
     * Build TriggerContext from SwarmState for agent JEXL evaluation
     * 
     * @param swarmState Current swarm state
     * @param event Triggering event (optional)
     * @param bot The bot needing context
     * @returns TriggerContext for JEXL evaluation
     */
    buildTriggerContext(
        swarmState: SwarmState,
        event?: ServiceEvent,
        bot?: BotParticipant,
    ): TriggerContext {
        // Calculate total allocated resources from allocation entries
        function calculateAllocated() {
            if (!swarmState.resources.allocated || swarmState.resources.allocated.length === 0) {
                // If no allocations, use consumed + remaining as a fallback
                return {
                    credits: swarmState.resources.consumed.credits + swarmState.resources.remaining.credits,
                    tokens: swarmState.resources.consumed.tokens + swarmState.resources.remaining.tokens,
                    time: swarmState.resources.consumed.time + swarmState.resources.remaining.time,
                };
            }

            // Sum up all allocations
            const totalAllocated = swarmState.resources.allocated.reduce(
                (totals, allocation) => {
                    // Credits: Parse BigInt string and convert to number
                    const maxCredits = Number(BigInt(allocation.limits.maxCredits));

                    // Time: Convert milliseconds to seconds for consistency
                    const maxTimeSeconds = Math.floor(allocation.limits.maxDurationMs / SECONDS_1_MS);

                    return {
                        credits: totals.credits + maxCredits,
                        tokens: totals.tokens, // Tokens not tracked in allocations yet
                        time: totals.time + maxTimeSeconds,
                    };
                },
                { credits: 0, tokens: 0, time: 0 },
            );

            // For tokens, use the remaining + consumed since it's not in allocations
            totalAllocated.tokens = swarmState.resources.consumed.tokens + swarmState.resources.remaining.tokens;

            return totalAllocated;
        }

        // Build resource consumption tracking
        const consumed = {
            credits: swarmState.resources.consumed.credits,
            tokens: swarmState.resources.consumed.tokens,
            time: swarmState.resources.consumed.time, // This is already in seconds from SwarmState
        };

        const remaining = {
            credits: swarmState.resources.remaining.credits,
            tokens: swarmState.resources.remaining.tokens,
            time: swarmState.resources.remaining.time, // This is already in seconds from SwarmState
        };

        // Build bot performance metrics from chat config stats
        const botStats = bot ? swarmState.chatConfig.stats?.botStats?.[bot.id] : undefined;
        const performance = {
            tasksCompleted: botStats?.tasksCompleted || 0,
            tasksFailed: 0, // TODO: Track failed tasks
            averageCompletionTime: botStats?.averageTaskDuration || 0,
            successRate: botStats?.successRate || 0,
            resourceEfficiency: 0, // TODO: Calculate efficiency metric
        };

        const allocated = calculateAllocated();

        const triggerContext: TriggerContext = {
            // Event information (if provided)
            event: event ? {
                type: event.type,
                data: event.data || {},
                timestamp: new Date(event.timestamp),
                metadata: event.metadata,
            } : {
                type: "none",
                data: {},
                timestamp: new Date(),
            },

            // Swarm state and resource information
            swarm: {
                state: swarmState.execution.status,
                resources: {
                    allocated,
                    consumed,
                    remaining,
                },
                agents: swarmState.execution.agents.length,
                id: swarmState.swarmId,
            },

            // Bot's own performance metrics
            bot: {
                id: bot?.id || "system",
                performance,
            },

            // Direct access to chat config data
            goal: swarmState.chatConfig.goal,
            subtasks: swarmState.chatConfig.subtasks?.map(task => ({
                id: task.id,
                description: task.description,
                status: task.status,
                assignee_bot_id: swarmState.chatConfig.subtaskLeaders?.[task.id],
                priority: task.priority,
            })),

            // Filter blackboard based on bot's resource permissions
            blackboard: this.filterBlackboardByPermissions(
                swarmState.chatConfig.blackboard,
                bot?.config?.agentSpec?.resources,
            ),

            // Filter records based on bot's permissions
            records: swarmState.chatConfig.records?.map(record => ({
                id: record.id,
                routine_name: record.routine_name,
                created_at: record.created_at,
                caller_bot_id: record.caller_bot_id,
            })),

            // Event subscription mappings (if available)
            eventSubscriptions: undefined, // TODO: Add event subscription tracking

            // Swarm statistics
            stats: swarmState.chatConfig.stats,
        };

        return triggerContext;
    }

    /**
     * Safely access data from swarm state with security checks
     * 
     * @param path Dot-notation path to access (e.g., "blackboard.learning.patterns")
     * @param triggerContext The agent's current trigger context
     * @param securityContext Security information for permission validation
     * @param options Additional access options
     */
    async accessData<T>(
        path: string,
        triggerContext: TriggerContext,
        swarmState: SwarmState,
        options?: AccessOptions,
    ): Promise<T> {
        const startTime = Date.now();
        let success = false;
        let data: unknown;

        try {
            // 1. Check data sensitivity
            const sensitivity = this.checkSensitivity(path, swarmState);

            // 2. Emit data access requested event if sensitive
            if (sensitivity) {
                const allowed = await this.emitDataAccessEvent("requested", path, triggerContext, swarmState, sensitivity);
                if (!allowed) {
                    throw new CustomError("SWAC", "Unauthorized");
                }
            }

            // 3. Validate access permissions
            await this.validateAccess(path, swarmState, triggerContext);

            // 4. Retrieve data
            data = this.retrieveData(path, triggerContext);
            success = true;

            // 5. Emit data access completed event
            if (sensitivity) {
                const duration = Date.now() - startTime;
                await this.emitDataAccessEvent("completed", path, triggerContext, swarmState, sensitivity, data, success, duration);
            }

            // 6. Apply data transformations if needed
            return this.transformData(data, options);
        } catch (error) {
            // Emit completion event with failure
            const sensitivity = this.checkSensitivity(path, swarmState);
            if (sensitivity) {
                const duration = Date.now() - startTime;
                await this.emitDataAccessEvent("completed", path, triggerContext, swarmState, sensitivity, undefined, false, duration);
            }
            throw error;
        }
    }

    /**
     * Check if data path is sensitive
     */
    private checkSensitivity(
        path: string,
        swarmState: SwarmState,
    ): DataSensitivityConfig | null {
        const secrets = swarmState?.chatConfig?.secrets;

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
     * Emit data access event for auditing and control
     * @returns true if access should continue, false if blocked
     */
    private async emitDataAccessEvent(
        type: "requested" | "completed",
        path: string,
        triggerContext: TriggerContext,
        swarmState: SwarmState,
        sensitivity: DataSensitivityConfig,
        data?: unknown,
        success?: boolean,
        duration?: number,
    ): Promise<boolean> {
        const eventType = type === "requested"
            ? EventTypes.DATA.ACCESS_REQUESTED
            : EventTypes.DATA.ACCESS_COMPLETED;

        if (type === "requested") {
            // For requested events, check progression to see if access is allowed
            const { proceed: allowed, reason } = await EventPublisher.emit(eventType, {
                chatId: swarmState.swarmId,
                path,
                operation: "read" as const,
                sensitivity: this.mapSensitivityType(sensitivity.type),
                requesterId: triggerContext.bot.id,
                requesterType: "bot" as const,
                context: {
                    component: "SwarmStateAccessor",
                    swarmId: swarmState.swarmId,
                },
            });

            if (!allowed) {
                logger.debug(`[SwarmStateAccessor] Data access blocked: ${reason}`, {
                    path,
                    swarmId: swarmState.swarmId,
                    reason,
                });
            }

            return allowed;
        } else {
            // For completed events, check if they're allowed to be reported
            const { proceed, reason } = await EventPublisher.emit(eventType, {
                chatId: swarmState.swarmId,
                path,
                operation: "read" as const,
                success: success || false,
                sanitized: this.shouldSanitize(sensitivity),
                requesterId: triggerContext.bot.id,
                duration: duration || 0,
            });

            if (!proceed) {
                logger.warn("[SwarmStateAccessor] Data access completion event blocked", {
                    path,
                    swarmId: swarmState.swarmId,
                    reason,
                });
            }

            return true; // Completed events always return true for backward compatibility
        }
    }

    /**
     * Validate access permissions for the given path
     */
    private async validateAccess(
        path: string,
        swarmState: SwarmState,
        triggerContext: TriggerContext,
    ): Promise<void> {
        // 1. Basic validation
        const agentId = triggerContext.bot.id;
        if (!agentId) {
            throw new CustomError("SWAC", "Unauthorized", { code: "NoAgentId" });
        }

        // 2. Check SwarmPolicy visibility (if swarmState is available)
        if (swarmState) {
            const policy = swarmState.chatConfig.policy;
            if (policy) {
                if (policy.visibility === "private") {
                    // Only ACL members can access
                    if (!policy.acl?.includes(agentId)) {
                        throw new CustomError("SWAC", "Unauthorized", {
                            code: "PrivateSwarm",
                            details: "Access denied: Private swarm",
                        });
                    }
                } else if (policy.visibility === "restricted") {
                    // Read-only for non-ACL members (write operations would be checked here)
                    const isAclMember = policy.acl?.includes(agentId);
                    if (!isAclMember && this.isWriteOperation(path)) {
                        throw new CustomError("SWAC", "Unauthorized", {
                            code: "RestrictedSwarm",
                            details: "Access denied: Restricted swarm allows read-only access",
                        });
                    }
                }
            }

            // 3. Find the agent in swarm state
            const agent = swarmState.execution.agents.find(
                a => a.id === agentId,
            );

            if (!agent) {
                throw new CustomError("SWAC", "Unauthorized", {
                    code: "AgentNotInSwarm",
                    details: "Agent not found in swarm execution context",
                });
            }

            // 4. Check agent's resource permissions
            const resources = agent.config?.agentSpec?.resources || [];

            // Determine resource type from path
            const resourceType = this.getResourceTypeFromPath(path);

            // Find matching resource spec
            const hasAccess = resources.some(resource => {
                // Check resource type match
                if (resource.type !== resourceType) {
                    return false;
                }

                // Check scope pattern match if specified
                if (resource.scope && !this.pathMatchesPattern(path, resource.scope)) {
                    return false;
                }

                // Check permissions (default to read-only if not specified)
                const requiredPermission = "read"; // Currently only read is implemented
                return resource.permissions?.includes(requiredPermission) ?? true;
            });

            if (!hasAccess) {
                throw new CustomError("SWAC", "Unauthorized", {
                    code: "NoResourcePermission",
                    details: `Access denied: No permission for path '${path}' with resource type '${resourceType}'`,
                });
            }
        }

        logger.debug("[SwarmStateAccessor] Access validated", {
            path,
            agentId,
            resourceType: this.getResourceTypeFromPath(path),
            policyVisibility: swarmState?.chatConfig?.policy?.visibility || "open",
        });
    }

    /**
     * Retrieve data from the context using dot notation
     * 
     * Since we now use TriggerContext directly, paths are already in the
     * agent-friendly format (e.g., "blackboard.key", "swarm.state", etc.)
     */
    private retrieveData(
        path: string,
        triggerContext: TriggerContext,
    ): unknown {
        try {
            // Direct access to TriggerContext properties using valueFromDot
            return valueFromDot(triggerContext as unknown as Record<string, unknown>, path);
        } catch (error) {
            logger.error("[SwarmStateAccessor] Failed to retrieve data", {
                path,
                error,
            });
            return undefined;
        }
    }

    /**
     * Filter blackboard items based on bot's resource permissions
     * 
     * @param blackboard All blackboard items (using canonical BlackboardItem structure)
     * @param resources Bot's resource specifications
     * @returns Filtered blackboard as key-value pairs
     */
    private filterBlackboardByPermissions(
        blackboard?: Array<{ id: string; value: any; created_at: string }>,
        resources?: Array<{ type: string; scope?: string; permissions?: string[] }>,
    ): Record<string, any> | undefined {
        if (!blackboard || blackboard.length === 0) {
            return undefined;
        }

        // If no resources specified, bot has no blackboard access
        if (!resources || resources.length === 0) {
            return {};
        }

        // Find blackboard-specific resource specifications
        const blackboardSpecs = resources.filter(r => 
            r.type === "blackboard" || 
            r.type === "all", // "all" type grants access to everything
        );

        // If no blackboard specs found, check for legacy scope-based access
        if (blackboardSpecs.length === 0) {
            // Legacy check: scope "read" or "write" at resource level (deprecated pattern)
            const hasLegacyAccess = resources.some(r =>
                r.scope === "read" || r.scope === "write",
            );
            
            if (!hasLegacyAccess) {
                return {};
            }
            
            // For legacy access, grant full blackboard access
            blackboardSpecs.push({ type: "blackboard", permissions: ["read"] });
        }

        // Apply scope-based filtering for each blackboard item
        const filtered: Record<string, any> = {};
        
        for (const item of blackboard) {
            // Check if any blackboard spec grants access to this item
            const hasAccess = blackboardSpecs.some(spec => {
                // If no scope specified, grant access to all blackboard items
                if (!spec.scope) {
                    return true;
                }
                
                // Check if the item ID matches the scope pattern
                // Scope can be:
                // - Exact match: "config" matches blackboard.config
                // - Wildcard: "metrics.*" matches blackboard.metrics.cpu, blackboard.metrics.memory
                // - Prefix: "temp_" matches any item starting with temp_
                return this.pathMatchesPattern(`blackboard.${item.id}`, spec.scope);
            });
            
            if (hasAccess) {
                filtered[item.id] = item.value;
            }
        }

        return Object.keys(filtered).length > 0 ? filtered : undefined;
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
     * Map legacy sensitivity type to new enum
     */
    private mapSensitivityType(type: string): DataSensitivityType {
        switch (type.toUpperCase()) {
            case "PII":
                return DataSensitivityType.PII;
            case "PHI":
                return DataSensitivityType.PHI;
            case "FINANCIAL":
                return DataSensitivityType.FINANCIAL;
            case "CREDENTIAL":
                return DataSensitivityType.CREDENTIAL;
            case "PROPRIETARY":
                return DataSensitivityType.PROPRIETARY;
            default:
                return DataSensitivityType.PUBLIC;
        }
    }

    /**
     * Determine if data should be sanitized for logging
     */
    private shouldSanitize(sensitivity: DataSensitivityConfig): boolean {
        return ["PII", "PHI", "FINANCIAL", "CREDENTIAL"].includes(sensitivity.type);
    }

    /**
     * Get resource type from path for permission checking
     * Maps agent-facing data paths from TriggerContext to resource types defined in ResourceSpec
     */
    private getResourceTypeFromPath(path: string): string {
        // Blackboard access (shared memory)
        if (path === "blackboard" || path.startsWith("blackboard.")) {
            return "blackboard";
        }

        // Execution-related data maps to "routine" type
        if (path === "subtasks" || path.startsWith("subtasks")) {
            return "routine"; // Subtasks are execution/coordination related
        }
        if (path === "records" || path.startsWith("records")) {
            return "routine"; // Execution history
        }
        if (path.startsWith("swarm.state") || path.startsWith("swarm.resources")) {
            return "routine"; // Swarm execution state and resource tracking
        }

        // Tool-specific paths
        if (path.includes("tool")) {
            return "tool";
        }

        // Document/informational data
        if (path === "goal" || path === "stats" || path.startsWith("stats.")) {
            return "document"; // Read-only informational data
        }
        if (path.startsWith("swarm.id") || path.startsWith("swarm.agents")) {
            return "document"; // Basic swarm information
        }

        // Link-specific paths
        if (path.includes("link") || path.includes("url")) {
            return "link";
        }

        // Bot and event context would map to routine (execution context)
        if (path.startsWith("bot.") || path.startsWith("event.")) {
            return "routine";
        }

        // Default to document type for general/unknown data access
        return "document";
    }

    /**
     * Check if operation is a write operation (for future extension)
     */
    private isWriteOperation(path: string): boolean {
        // Currently we only support read operations
        // This is a placeholder for future write operation support
        return false;
    }
}
