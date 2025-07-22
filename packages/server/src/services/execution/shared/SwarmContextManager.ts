/**
 * Swarm Context Manager - Central State Authority for Emergent AI Swarms
 * 
 * This is the foundational component of the new three-tier execution architecture that
 * enables true emergent capabilities by ensuring all swarm behavior is data-driven.
 * 
 * ## Core Mission: Enable Emergent Intelligence
 * 
 * Traditional AI systems hard-code behavior in software. This manager enables emergent
 * capabilities by ensuring that:
 * 
 * 1. **All swarm behavior comes from configuration data** - No hardcoded agent logic
 * 2. **Agents can modify swarm behavior through events** - Dynamic reconfiguration
 * 3. **Historical data drives continuous improvement** - Self-optimizing systems
 * 4. **Real-time updates propagate instantly** - Live swarm adaptation
 * 
 * ## Architecture Integration:
 * 
 * ```
 * SwarmContextManager (This Class - Central Authority)
 *         ↓ provides unified state to
 * Tier 1: SwarmStateMachine  
 * Tier 2: RunStateMachine + RoutineOrchestrator
 * Tier 3: StepExecutor (simplified implementation)
 *         ↓ all receive updates via
 * EventBus (unified event publishing)
 * ```
 * 
 * ## Emergent Capabilities Enabled:
 * 
 * - **Resource Optimization Agents**: Monitor allocation efficiency and adjust strategies
 * - **Performance Learning Agents**: Analyze execution patterns and optimize configurations
 * - **Security Adaptation Agents**: Detect threats and dynamically update security policies
 * - **Organizational Evolution Agents**: Optimize team structures based on success patterns
 * - **Goal Generation Agents**: Create new objectives based on discovered opportunities
 * 
 * ## Data-Driven Design:
 * 
 * Every aspect of swarm behavior is controlled by the SwarmState:
 * - Resource allocation strategies → `context.policy.resource.allocation`
 * - Agent permissions → `context.policy.security.permissions`
 * - Team structures → `context.policy.organizational.structure`
 * - Tool approval rules → `context.policy.security.toolApproval`
 * - Performance thresholds → `context.policy.resource.thresholds`
 * 
 * This enables agents to modify swarm behavior by updating configuration data,
 * not by changing code.
 * 
 * @see SwarmState - The unified context type system
 * @see EventBus - Unified event publishing system
 * @see SwarmContextManager - Centralized resource allocation and state management
 * @see /docs/architecture/execution/swarm-state-management-redesign.md - Complete architecture
 */

import { EventTypes, generatePK, RunState, SECONDS_1_MS, SECONDS_30_MS, type BaseTierExecutionRequest, type ResourceAllocation, type StateMachineState, type SwarmId, type SwarmState } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { CacheService } from "../../../redisConn.js";
import { EventPublisher } from "../../events/publisher.js";
import { SwarmStateAccessor } from "./SwarmStateAccessor.js";

const BASE64_CHECKSUM_LENGTH = 16;

/**
 * Context update event for live propagation
 */
interface ContextUpdateEvent {
    /** Swarm being updated */
    chatId: SwarmId;

    /** Previous version number */
    previousVersion: number;

    /** New version number */
    newVersion: number;

    /** What changed (for efficient updates) */
    changes: {
        path: string; // JSONPath to changed field
        oldValue: any;
        newValue: any;
        changeType: "created" | "updated" | "deleted";
        operation: "create" | "update" | "delete";
    }[];

    /** Who made the change */
    updatedBy: string;

    /** When the change occurred */
    timestamp: Date;

    /** Change reason (for audit trail) */
    reason?: string;

    /** Whether this was an emergent change (made by an agent) */
    emergent: boolean;
}

/**
 * Simplified resource summary for status reporting
 */
interface ResourceSummary {
    credits?: number;
    tokens?: number;
    time?: number;
    [key: string]: number | undefined;
}

/**
 * Context query for retrieving specific context data
 */
interface ContextQuery {
    /** Swarm to query */
    chatId: SwarmId;

    /** JSONPath expressions for data to retrieve */
    select?: string[];

    /** Filters to apply */
    where?: {
        path: string;
        operator: "equals" | "contains" | "greaterThan" | "lessThan" | "exists";
        value: any;
    }[];

    /** Version constraints */
    version?: {
        exact?: number;
        minimum?: number;
        maximum?: number;
    };

    /** Include historical data */
    includeHistory?: boolean;
}

/**
 * Context validation result
 */
interface ContextValidationResult {
    /** Whether the context is valid */
    valid: boolean;

    /** Validation errors */
    errors: {
        path: string;
        message: string;
        severity: "error" | "warning" | "info";
    }[];

    /** Validation warnings */
    warnings: {
        path: string;
        message: string;
        suggestion?: string;
    }[];

    /** Validation performance metrics */
    metrics: {
        validationTimeMs: number;
        rulesChecked: number;
        constraintsValidated: number;
    };
}


/**
 * Context management operations interface
 */
export interface ISwarmContextManager {
    // Context lifecycle
    createContext(chatId: SwarmId, initialConfig: Partial<SwarmState>): Promise<SwarmState>;
    getContext(chatId: SwarmId): Promise<SwarmState | null>;
    updateContext(chatId: SwarmId, updates: Partial<SwarmState>, reason?: string): Promise<SwarmState>;
    deleteContext(chatId: SwarmId): Promise<void>;

    // Resource management
    allocateResources(chatId: SwarmId, request: Omit<ResourceAllocation, "id" | "allocatedAt">): Promise<ResourceAllocation>;
    releaseResources(chatId: SwarmId, allocationId: string): Promise<void>;
    getResourceStatus(chatId: SwarmId): Promise<{ total: ResourceSummary; allocated: ResourceAllocation[]; available: ResourceSummary }>;

    // Context querying and validation
    query(query: ContextQuery): Promise<Partial<SwarmState>>;
    validate(context: SwarmState): Promise<ContextValidationResult>;

    // System management
    healthCheck(): Promise<{ healthy: boolean; details: Record<string, any> }>;
    getMetrics(): Promise<Record<string, number>>;
}

/**
 * Internal context storage format with versioning and metadata
 */
interface ContextStorageRecord {
    __version: "1";
    context: SwarmState;
    metadata: {
        storageVersion: number;
        compressed: boolean;
        checksum: string;
        accessCount: number;
        lastAccessed: Date;
    };
}

/**
 * Swarm Context Manager Implementation
 * 
 * Provides unified state management for emergent AI swarms with:
 * - Atomic context updates with versioning
 * - Live update propagation via Redis pub/sub
 * - Hierarchical resource allocation
 * - Context validation and integrity checking
 * - Performance optimization and caching
 */
export class SwarmContextManager implements ISwarmContextManager {
    private static readonly DEFAULT_CACHE_TTL_MS = SECONDS_30_MS;
    private static readonly CHECKSUM_LENGTH = BASE64_CHECKSUM_LENGTH;
    private static readonly MAX_VERSION_HISTORY = 10;

    private readonly config: BaseTierExecutionRequest;
    private readonly stateAccessor: SwarmStateAccessor;

    // Caching and performance
    private readonly contextCache = new Map<SwarmId, ContextStorageRecord>();
    private readonly cacheTTL = SwarmContextManager.DEFAULT_CACHE_TTL_MS;

    // Metrics tracking
    private readonly metrics = {
        contextsCreated: 0,
        contextsUpdated: 0,
        contextsDeleted: 0,
        eventBusPublications: 0,
        resourceAllocations: 0,
        cacheHits: 0,
        cacheMisses: 0,
        validationErrors: 0,
    };

    constructor(config: BaseTierExecutionRequest) {
        this.config = config;
        this.stateAccessor = new SwarmStateAccessor();
        logger.info("[SwarmContextManager] Initialized", config);
    }

    /**
     * Start the context manager and setup pub/sub handlers
     */
    async start(): Promise<void> {
        // Add any initialization logic here
    }

    /**
     * Stop the context manager and cleanup resources
     */
    async stop(): Promise<void> {
        try {
            // Clear cache
            this.contextCache.clear();

            logger.info("[SwarmContextManager] Stopped successfully");
        } catch (error) {
            logger.error("[SwarmContextManager] Error during stop", {
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Create a new swarm context with default configuration that enables emergent capabilities
     */
    async createContext(
        chatId: SwarmId,
        initialConfig: Partial<SwarmState> = {},
    ): Promise<SwarmState> {
        logger.debug("[SwarmContextManager] Creating new swarm context", { chatId });

        try {
            // Check if context already exists
            const existing = await this.getContext(chatId);
            if (existing) {
                throw new Error(`Swarm context already exists: ${chatId}`);
            }

            // Create default context that enables emergent capabilities
            const defaultContext = this.createDefaultContext(chatId);

            // Merge with provided configuration
            const context: SwarmState = {
                ...defaultContext,
                ...initialConfig,
                // Ensure core fields cannot be overridden
                swarmId: chatId,
                version: 1,
                // Merge chatConfig properly
                chatConfig: {
                    ...defaultContext.chatConfig,
                    ...(initialConfig.chatConfig || {}),
                },
                // Update metadata
                metadata: {
                    ...defaultContext.metadata,
                    ...(initialConfig.metadata || {}),
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: initialConfig.metadata?.updatedBy || "system",
                },
            };

            // Validate context
            const validation = await this.validate(context);
            if (!validation.valid) {
                throw new Error(`Context validation failed: ${validation.errors.map(e => e.message).join(", ")}`);
            }

            // Store context
            await this.storeContext(context);

            // Update metrics
            this.metrics.contextsCreated++;

            logger.info("[SwarmContextManager] Created swarm context with emergent capabilities", {
                chatId,
                version: context.version,
                chatConfigVersion: context.chatConfig.__version,
                resourcesInitialized: context.resources.remaining.credits > 0,
            });

            return context;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to create context", {
                chatId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get current swarm context with caching optimization
     */
    async getContext(chatId: SwarmId): Promise<SwarmState | null> {
        try {
            // Check in-memory cache first
            const cached = this.contextCache.get(chatId);
            if (cached && this.isCacheValid(cached)) {
                this.metrics.cacheHits++;
                cached.metadata.accessCount++;
                cached.metadata.lastAccessed = new Date();
                return cached.context;
            }

            this.metrics.cacheMisses++;

            // Load from Redis
            const redis = await CacheService.get().raw();
            const key = this.getContextKey(chatId);
            const data = await redis.get(key);

            if (!data) {
                return null;
            }

            // Parse and validate stored context
            let record: ContextStorageRecord;
            try {
                const parsed = JSON.parse(data);
                if (!this.isValidContextStorageRecord(parsed)) {
                    logger.error("[SwarmContextManager] Invalid context storage record format", {
                        chatId,
                        recordKeys: Object.keys(parsed),
                    });
                    return null;
                }
                record = parsed;
            } catch (parseError) {
                logger.error("[SwarmContextManager] Failed to parse context record", {
                    chatId,
                    error: parseError instanceof Error ? parseError.message : String(parseError),
                    dataLength: data.length,
                });
                return null;
            }

            // Update access metadata
            record.metadata.accessCount++;
            record.metadata.lastAccessed = new Date();

            // Store in cache
            this.contextCache.set(chatId, record);

            // Update access count in Redis (fire and forget)
            const updatedData = JSON.stringify(record, (key, value) => {
                if (typeof value === "bigint") {
                    return value.toString();
                }
                return value;
            });
            redis.set(key, updatedData, "EX", Math.floor(this.cacheTTL / SECONDS_1_MS)).catch(error => {
                logger.warn("[SwarmContextManager] Failed to update access metadata", {
                    chatId,
                    error: error instanceof Error ? error.message : String(error),
                });
            });

            return record.context;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to get context", {
                chatId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    /**
     * Update swarm context with atomic versioning and live propagation
     */
    async updateContext(
        chatId: SwarmId,
        updates: Partial<SwarmState>,
        reason = "Context update",
    ): Promise<SwarmState> {
        logger.debug("[SwarmContextManager] Updating swarm context", {
            chatId,
            reason,
            updateKeys: Object.keys(updates),
        });

        try {
            // Get current context
            const currentContext = await this.getContext(chatId);
            if (!currentContext) {
                throw new Error(`Swarm context not found: ${chatId}`);
            }

            // Create updated context with version increment
            const updatedContext: SwarmState = {
                ...currentContext,
                ...updates,
                // Ensure core fields are properly managed
                swarmId: chatId, // Cannot be changed
                version: currentContext.version + 1,
                metadata: {
                    ...currentContext.metadata,
                    lastUpdated: new Date(),
                    updatedBy: updates.metadata?.updatedBy || "system",
                },
            };

            // Validate updated context
            const validation = await this.validate(updatedContext);
            if (!validation.valid) {
                throw new Error(`Context validation failed: ${validation.errors.map(e => e.message).join(", ")}`);
            }

            // Store updated context atomically
            await this.storeContext(updatedContext);

            // Clear cache for this swarm
            this.contextCache.delete(chatId);

            // Create and publish update event for live propagation
            const updateEvent: ContextUpdateEvent = {
                chatId,
                previousVersion: currentContext.version,
                newVersion: updatedContext.version,
                changes: this.computeChanges(currentContext, updatedContext),
                updatedBy: updatedContext.metadata.updatedBy,
                timestamp: new Date(),
                reason,
                emergent: false, // Set to true when change is made by an agent
            };

            await this.publishUpdateEvent(updateEvent);

            // Update metrics
            this.metrics.contextsUpdated++;

            logger.info("[SwarmContextManager] Updated swarm context with live propagation", {
                chatId,
                previousVersion: currentContext.version,
                newVersion: updatedContext.version,
                changesCount: updateEvent.changes.length,
                emergent: updateEvent.emergent,
            });

            return updatedContext;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to update context", {
                chatId,
                reason,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Delete swarm context and cleanup all related data
     */
    async deleteContext(chatId: SwarmId): Promise<void> {
        logger.debug("[SwarmContextManager] Deleting swarm context", { chatId });

        try {
            // Remove from cache
            this.contextCache.delete(chatId);

            // Remove from Redis
            const redis = await CacheService.get().raw();
            const key = this.getContextKey(chatId);
            await redis.del(key);

            // Cleanup version history
            const historyPattern = `${key}:version:*`;
            const historyKeys = await redis.keys(historyPattern);
            if (historyKeys.length > 0) {
                await redis.del(...historyKeys);
            }

            // Update metrics
            this.metrics.contextsDeleted++;

            logger.info("[SwarmContextManager] Deleted swarm context and cleanup completed", {
                chatId,
                cleanupVersions: historyKeys.length,
            });

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to delete context", {
                chatId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    // Note: Live updates are handled through EventBus publishing.
    // Components that need context updates should subscribe to EventBus directly
    // using patterns like EventTypes.SWARM.STATE_CHANGED, etc.

    /**
     * Allocate resources with hierarchical tracking for emergent optimization
     */
    async allocateResources(
        chatId: SwarmId,
        request: Omit<ResourceAllocation, "id" | "allocated">,
    ): Promise<ResourceAllocation> {
        logger.debug("[SwarmContextManager] Allocating resources", {
            chatId,
            consumerId: request.consumerId,
            consumerType: request.consumerType,
            maxCredits: request.limits.maxCredits,
        });

        try {
            const context = await this.getContext(chatId);
            if (!context) {
                throw new Error(`Swarm context not found: ${chatId}`);
            }

            // Create resource allocation
            const creditsToAllocate = parseInt(request.limits.maxCredits);
            const allocation: ResourceAllocation = {
                ...request,
                id: generatePK().toString(),
                allocated: {
                    credits: creditsToAllocate,
                    timestamp: new Date(),
                },
            };

            // Validate allocation against available resources
            const validationResult = this.validateResourceAllocation(context, allocation);
            if (!validationResult.valid) {
                throw new Error(`Resource allocation validation failed: ${validationResult.reason}`);
            }

            // Update context with new allocation
            const updatedAllocations = [...context.resources.allocated, allocation];

            // Update remaining resources based on allocated credits
            const creditsAllocated = allocation.allocated.credits;

            await this.updateContext(chatId, {
                resources: {
                    ...context.resources,
                    allocated: updatedAllocations,
                    remaining: {
                        ...context.resources.remaining,
                        credits: Math.max(0, context.resources.remaining.credits - creditsAllocated),
                    },
                },
            }, `Resource allocation for ${allocation.consumerType} ${allocation.consumerId}`);

            this.metrics.resourceAllocations++;

            logger.info("[SwarmContextManager] Allocated resources with emergent tracking", {
                chatId,
                allocationId: allocation.id,
                consumerId: allocation.consumerId,
                consumerType: allocation.consumerType,
                creditsAllocated: allocation.allocated.credits,
            });

            return allocation;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to allocate resources", {
                chatId,
                consumerId: request.consumerId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Release resources and update availability
     */
    async releaseResources(chatId: SwarmId, allocationId: string): Promise<void> {
        logger.debug("[SwarmContextManager] Releasing resources", { chatId, allocationId });

        try {
            const context = await this.getContext(chatId);
            if (!context) {
                throw new Error(`Swarm context not found: ${chatId}`);
            }

            // Find and remove allocation
            const allocationIndex = context.resources.allocated.findIndex(a => a.id === allocationId);
            if (allocationIndex === -1) {
                throw new Error(`Resource allocation not found: ${allocationId}`);
            }

            const allocation = context.resources.allocated[allocationIndex];
            const updatedAllocations = context.resources.allocated.filter(a => a.id !== allocationId);

            // Return resources to remaining pool
            const creditsReleased = allocation.allocated.credits;

            await this.updateContext(chatId, {
                resources: {
                    ...context.resources,
                    allocated: updatedAllocations,
                    remaining: {
                        ...context.resources.remaining,
                        credits: context.resources.remaining.credits + creditsReleased,
                    },
                },
            }, `Resource release for ${allocation.consumerType} ${allocation.consumerId}`);

            logger.info("[SwarmContextManager] Released resources", {
                chatId,
                allocationId,
                consumerId: allocation.consumerId,
                consumerType: allocation.consumerType,
                creditsReleased: allocation.allocated.credits,
            });

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to release resources", {
                chatId,
                allocationId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get current resource status for monitoring and optimization
     */
    async getResourceStatus(chatId: SwarmId): Promise<{
        total: ResourceSummary;
        allocated: ResourceAllocation[];
        available: ResourceSummary;
    }> {
        const context = await this.getContext(chatId);
        if (!context) {
            throw new Error(`Swarm context not found: ${chatId}`);
        }

        // Calculate total from limits and current state
        const totalFromLimits = {
            credits: parseInt(context.chatConfig.limits?.maxCredits || "10000"),
        };

        return {
            total: totalFromLimits,
            allocated: context.resources.allocated,
            available: context.resources.remaining,
        };
    }

    /**
     * Query context data with JSONPath support
     */
    async query(query: ContextQuery): Promise<Partial<SwarmState>> {
        const context = await this.getContext(query.chatId);
        if (!context) {
            return {};
        }

        // Build TriggerContext for data access
        const triggerContext = this.stateAccessor.buildTriggerContext(context);

        // Handle select paths
        if (query.select && query.select.length > 0) {
            const result: Record<string, any> = {};

            for (const path of query.select) {
                try {
                    const value = await this.stateAccessor.accessData(
                        path,
                        triggerContext,
                        context,
                    );
                    result[path] = value;
                } catch (error) {
                    logger.warn("[SwarmContextManager] Failed to query path", {
                        path,
                        error: error instanceof Error ? error.message : String(error),
                    });
                    result[path] = undefined;
                }
            }

            return result;
        }

        // Handle where filters
        if (query.where && query.where.length > 0) {
            // Filter context based on where conditions
            let matches = true;

            for (const filter of query.where) {
                try {
                    const value = await this.stateAccessor.accessData(
                        filter.path,
                        triggerContext,
                        context,
                    );

                    switch (filter.operator) {
                        case "equals":
                            matches = matches && value === filter.value;
                            break;
                        case "contains":
                            matches = matches && String(value).includes(String(filter.value));
                            break;
                        case "greaterThan":
                            matches = matches && Number(value) > Number(filter.value);
                            break;
                        case "lessThan":
                            matches = matches && Number(value) < Number(filter.value);
                            break;
                        case "exists":
                            matches = matches && value !== undefined && value !== null;
                            break;
                    }
                } catch (error) {
                    matches = false;
                    break;
                }
            }

            return matches ? context : {};
        }

        // Return full context if no specific query
        return context;
    }

    /**
     * Validate context integrity and constraints
     */
    async validate(context: SwarmState): Promise<ContextValidationResult> {
        const startTime = Date.now();
        const errors: ContextValidationResult["errors"] = [];
        const warnings: ContextValidationResult["warnings"] = [];
        let rulesChecked = 0;
        let constraintsValidated = 0;

        // Basic type validation
        if (!context || typeof context !== "object") {
            errors.push({
                path: "root",
                message: "Context must be a valid object",
                severity: "error",
            });
        } else if (!("chatConfig" in context) || !("resources" in context)) {
            errors.push({
                path: "root",
                message: "Context missing required fields (chatConfig, resources)",
                severity: "error",
            });
        }
        rulesChecked++;

        // Version validation
        if (context.version < 1) {
            errors.push({
                path: "version",
                message: "Context version must be >= 1",
                severity: "error",
            });
        }
        rulesChecked++;

        // Resource validation
        if (!context.resources || !context.resources.remaining) {
            errors.push({
                path: "resources.remaining",
                message: "Invalid resource tracking format",
                severity: "error",
            });
        } else {
            // Validate resource consumption doesn't exceed limits
            const maxCredits = parseInt(context.chatConfig?.limits?.maxCredits || "10000");

            if (context.resources.consumed.credits > maxCredits) {
                errors.push({
                    path: "resources.consumed.credits",
                    message: `Credits consumed (${context.resources.consumed.credits}) exceeds limit (${maxCredits})`,
                    severity: "error",
                });
            }
            constraintsValidated++;
        }
        rulesChecked++;

        // Chat config validation
        if (!context.chatConfig || !context.chatConfig.__version) {
            errors.push({
                path: "chatConfig",
                message: "Invalid chat configuration format",
                severity: "error",
            });
        }
        rulesChecked++;

        // Use SwarmStateAccessor to validate data access patterns
        try {
            const triggerContext = this.stateAccessor.buildTriggerContext(context);

            // Validate critical paths are accessible
            const criticalPaths = ["goal", "swarm.state", "swarm.resources"];
            for (const path of criticalPaths) {
                try {
                    await this.stateAccessor.accessData(
                        path,
                        triggerContext,
                        context,
                    );
                } catch (error) {
                    warnings.push({
                        path,
                        message: `Critical path not accessible: ${error instanceof Error ? error.message : String(error)}`,
                        suggestion: "Ensure all required data structures are properly initialized",
                    });
                }
                rulesChecked++;
            }

            // Validate agent configurations
            for (const agent of context.execution.agents) {
                if (!agent.config?.agentSpec) {
                    warnings.push({
                        path: `execution.agents[${agent.id}]`,
                        message: "Agent missing agentSpec configuration",
                        suggestion: "Add agentSpec to enable proper resource access control",
                    });
                }
                rulesChecked++;
            }

            // Validate blackboard structure
            if (context.chatConfig.blackboard) {
                for (let i = 0; i < context.chatConfig.blackboard.length; i++) {
                    const item = context.chatConfig.blackboard[i];
                    if (!item.id || item.value === undefined) {
                        errors.push({
                            path: `chatConfig.blackboard[${i}]`,
                            message: "Blackboard item missing required fields (id, value)",
                            severity: "error",
                        });
                    }
                }
                constraintsValidated++;
            }

        } catch (error) {
            errors.push({
                path: "stateAccessor",
                message: `Failed to validate with SwarmStateAccessor: ${error instanceof Error ? error.message : String(error)}`,
                severity: "error",
            });
        }

        const validationTimeMs = Date.now() - startTime;

        return {
            valid: errors.length === 0,
            errors,
            warnings,
            metrics: {
                validationTimeMs,
                rulesChecked,
                constraintsValidated,
            },
        };
    }

    /**
     * Health check for monitoring
     */
    async healthCheck(): Promise<{ healthy: boolean; details: Record<string, any> }> {
        try {
            // Test Redis connection
            const redis = await CacheService.get().raw();
            await redis.ping();

            return {
                healthy: true,
                details: {
                    redis: "connected",
                    cacheSize: this.contextCache.size,
                    eventBusIntegration: "active",
                    metrics: this.metrics,
                },
            };
        } catch (error) {
            return {
                healthy: false,
                details: {
                    error: error instanceof Error ? error.message : String(error),
                    redis: "disconnected",
                },
            };
        }
    }

    /**
     * Get performance metrics
     */
    async getMetrics(): Promise<Record<string, number>> {
        return { ...this.metrics };
    }

    // Private helper methods

    private isValidContextStorageRecord(obj: unknown): obj is ContextStorageRecord {
        if (!obj || typeof obj !== "object" || obj === null) {
            return false;
        }
        return (obj as ContextStorageRecord).__version === "1";
    }

    private async storeContext(context: SwarmState): Promise<void> {
        const record: ContextStorageRecord = {
            __version: "1",
            context,
            metadata: {
                storageVersion: 1,
                compressed: false,
                checksum: this.calculateChecksum(context),
                accessCount: 0,
                lastAccessed: new Date(),
            },
        };

        const key = this.getContextKey(context.swarmId);
        const data = JSON.stringify(record, (key, value) => {
            if (typeof value === "bigint") {
                return value.toString();
            }
            return value;
        });

        const redis = await CacheService.get().raw();
        await redis.set(key, data, "EX", Math.floor(this.cacheTTL / SECONDS_1_MS));

        // Store version history
        const versionKey = `${key}:version:${context.version}`;
        await redis.set(versionKey, data, "EX", Math.floor(this.cacheTTL / SECONDS_1_MS));

        // Cleanup old versions
        await this.cleanupOldVersions(context.swarmId);
    }

    private async cleanupOldVersions(swarmId: SwarmId): Promise<void> {
        const redis = await CacheService.get().raw();
        const baseKey = this.getContextKey(swarmId);
        const pattern = `${baseKey}:version:*`;
        const versionKeys = await redis.keys(pattern);

        if (versionKeys.length > SwarmContextManager.MAX_VERSION_HISTORY) {
            // Sort by version number and remove oldest
            const sortedKeys = versionKeys.sort((a, b) => {
                const versionA = parseInt(a.split(":version:")[1]);
                const versionB = parseInt(b.split(":version:")[1]);
                return versionA - versionB;
            });

            const keysToDelete = sortedKeys.slice(0, sortedKeys.length - SwarmContextManager.MAX_VERSION_HISTORY);
            if (keysToDelete.length > 0) {
                await redis.del(...keysToDelete);
            }
        }
    }

    private createDefaultContext(swarmId: SwarmId): SwarmState {
        // Create a default context that uses the new simplified structure
        const now = new Date();

        return {
            swarmId,
            version: 1,

            // Persisted configuration (untransformed)
            chatConfig: {
                __version: "1.0.0",
                goal: "Follow the user's instructions.",
                subtasks: [],
                blackboard: [],
                resources: [],
                records: [],
                stats: {
                    totalToolCalls: 0,
                    totalCredits: "0",
                    startedAt: Date.now(),
                    lastProcessingCycleEndedAt: null,
                },
                limits: {
                    maxCredits: "10000",
                    maxDurationMs: 3600000, // 1 hour
                },
                scheduling: {
                    defaultDelayMs: 0,
                    requiresApprovalTools: "none",
                    approvalTimeoutMs: 300000, // 5 minutes
                    autoRejectOnTimeout: true,
                },
                pendingToolCalls: [],
            },

            // Runtime-only execution state
            execution: {
                status: RunState.UNINITIALIZED,
                agents: [],
                activeRuns: [],
                startedAt: now,
                lastActivityAt: now,
            },

            // Runtime resource tracking
            resources: {
                allocated: [],
                consumed: {
                    credits: 0,
                    tokens: 0,
                    time: 0,
                },
                remaining: {
                    credits: 10000,
                    tokens: 10000,
                    time: 3600,
                },
            },

            // System metadata
            metadata: {
                createdAt: now,
                lastUpdated: now,
                updatedBy: "system",
                subscribers: new Set<string>(),
            },
        };
    }

    private getContextKey(chatId: SwarmId): string {
        return `swarm_context:${chatId}`;
    }

    private isCacheValid(record: ContextStorageRecord): boolean {
        const age = Date.now() - record.metadata.lastAccessed.getTime();
        return age < this.cacheTTL;
    }

    private calculateChecksum(context: SwarmState): string {
        // Simple checksum implementation - in production, use proper hashing
        // Handle BigInt serialization by converting to string
        const serialized = JSON.stringify(context, (key, value) => {
            if (typeof value === "bigint") {
                return value.toString();
            }
            return value;
        });
        return Buffer.from(serialized).toString("base64").substring(0, SwarmContextManager.CHECKSUM_LENGTH);
    }

    private computeChanges(
        oldContext: SwarmState,
        newContext: SwarmState,
    ): ContextUpdateEvent["changes"] {
        // Simplified change detection - in production, use proper diff algorithm
        const changes: ContextUpdateEvent["changes"] = [];

        if (oldContext.resources !== newContext.resources) {
            changes.push({
                path: "resources",
                oldValue: oldContext.resources,
                newValue: newContext.resources,
                changeType: "updated",
                operation: "update",
            });
        }

        // Check chatConfig.policy if it exists
        const oldPolicy = oldContext.chatConfig?.policy;
        const newPolicy = newContext.chatConfig?.policy;
        if (oldPolicy !== newPolicy) {
            changes.push({
                path: "chatConfig.policy",
                oldValue: oldPolicy,
                newValue: newPolicy,
                changeType: "updated",
                operation: "update",
            });
        }

        return changes;
    }

    /**
     * Publish context update events to the event bus
     * 
     * This method analyzes the changes and emits appropriate events based on what was modified.
     * The EventBus automatically handles propagation to subscribers and socket clients.
     */
    private async publishUpdateEvent(event: ContextUpdateEvent): Promise<void> {
        try {
            // Get the current context for complete event data
            const context = await this.getContext(event.chatId);
            if (!context) {
                logger.warn("[SwarmContextManager] Cannot publish update event - context not found", {
                    chatId: event.chatId,
                });
                return;
            }

            // Group changes by category to emit appropriate events
            const stateChanges = event.changes.filter(c => c.path.startsWith("execution."));
            const resourceChanges = event.changes.filter(c => c.path.startsWith("resources."));
            const configChanges = event.changes.filter(c => c.path.startsWith("chatConfig."));
            const teamChanges = event.changes.filter(c =>
                c.path.includes("teamId") ||
                c.path.includes("swarmLeader") ||
                c.path.includes("subtaskLeaders"),
            );

            // Emit state change event if execution state changed
            if (stateChanges.length > 0) {
                const statusChange = stateChanges.find(c => c.path === "execution.status");
                if (statusChange) {
                    const oldState = statusChange.oldValue;
                    const newState = statusChange.newValue;

                    const { proceed, reason } = await EventPublisher.emit(EventTypes.SWARM.STATE_CHANGED, {
                        chatId: event.chatId,
                        oldState: oldState as StateMachineState,
                        newState: newState as StateMachineState,
                        message: event.reason || "State updated",
                    });

                    if (!proceed) {
                        logger.warn("[SwarmContextManager] State change event blocked", {
                            chatId: event.chatId,
                            oldState,
                            newState,
                            reason,
                        });
                    }
                }
            }

            // Emit resource update event if resources changed
            if (resourceChanges.length > 0) {
                const { proceed, reason } = await EventPublisher.emit(EventTypes.SWARM.RESOURCE_UPDATED, {
                    chatId: event.chatId,
                    allocated: context.resources.allocated.length,
                    consumed: context.resources.consumed.credits,
                    remaining: context.resources.remaining.credits,
                });

                if (!proceed) {
                    logger.warn("[SwarmContextManager] Resource update event blocked", {
                        chatId: event.chatId,
                        reason,
                    });
                }
            }

            // Emit config update event if configuration changed (excluding policy)
            const nonPolicyConfigChanges = configChanges.filter(c => !c.path.startsWith("chatConfig.policy."));
            if (nonPolicyConfigChanges.length > 0) {
                // Build partial config update object
                const configUpdate: Record<string, unknown> = {};
                for (const change of nonPolicyConfigChanges) {
                    // Extract the path after "chatConfig."
                    const configPath = change.path.replace("chatConfig.", "");
                    configUpdate[configPath] = change.newValue;
                }

                const { proceed, reason } = await EventPublisher.emit(EventTypes.SWARM.CONFIG_UPDATED, {
                    chatId: event.chatId,
                    config: {
                        __typename: "ChatConfigObject" as const,
                        ...configUpdate,
                    },
                });

                if (!proceed) {
                    logger.warn("[SwarmContextManager] Config update event blocked", {
                        chatId: event.chatId,
                        reason,
                    });
                }
            }

            // Emit team update event if team-related data changed
            if (teamChanges.length > 0) {
                // Extract team-related data from the context
                const teamId = context.chatConfig.teamId;
                const swarmLeader = context.chatConfig.swarmLeader;
                const subtaskLeaders = context.chatConfig.subtaskLeaders;

                const { proceed, reason } = await EventPublisher.emit(EventTypes.SWARM.TEAM_UPDATED, {
                    chatId: event.chatId,
                    teamId,
                    swarmLeader,
                    subtaskLeaders,
                });

                if (!proceed) {
                    logger.warn("[SwarmContextManager] Team update event blocked", {
                        chatId: event.chatId,
                        reason,
                    });
                }
            }

            this.metrics.eventBusPublications++;

            logger.debug("[SwarmContextManager] Published context update events", {
                chatId: event.chatId,
                changesCount: event.changes.length,
                stateChanges: stateChanges.length,
                resourceChanges: resourceChanges.length,
                configChanges: nonPolicyConfigChanges.length,
                teamChanges: teamChanges.length,
                emergent: event.emergent,
            });

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to publish update events", {
                chatId: event.chatId,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't re-throw - event emission failures shouldn't break core context updates
        }
    }

    private validateResourceAllocation(
        context: SwarmState,
        allocation: ResourceAllocation,
    ): { valid: boolean; reason?: string } {
        // Check if allocation exceeds available resources
        const remaining = context.resources.remaining;
        const requested = allocation.limits;

        // Check credits against remaining and chat config limits
        if (requested.maxCredits && requested.maxCredits !== "unlimited") {
            const requestedCredits = parseInt(requested.maxCredits);
            if (requestedCredits > remaining.credits) {
                return {
                    valid: false,
                    reason: `Insufficient credits: requested ${requested.maxCredits}, available ${remaining.credits}`,
                };
            }
        }

        const MAX_CONCURRENT_RUNS = 10;

        // Check concurrent runs
        if (context.resources.allocated.length >= MAX_CONCURRENT_RUNS) {
            return {
                valid: false,
                reason: `Max concurrent runs exceeded: ${MAX_CONCURRENT_RUNS}`,
            };
        }

        return { valid: true };
    }
}
