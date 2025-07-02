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
 * Tier 2: RunStateMachine + RunOrchestrator
 * Tier 3: TierThreeExecutor + ValidationEngine
 *         ↓ all subscribe to live updates via
 * Redis Pub/Sub + ContextSubscriptionManager
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
 * Every aspect of swarm behavior is controlled by the UnifiedSwarmContext:
 * - Resource allocation strategies → `context.policy.resource.allocation`
 * - Agent permissions → `context.policy.security.permissions`
 * - Team structures → `context.policy.organizational.structure`
 * - Tool approval rules → `context.policy.security.toolApproval`
 * - Performance thresholds → `context.policy.resource.thresholds`
 * 
 * This enables agents to modify swarm behavior by updating configuration data,
 * not by changing code.
 * 
 * @see UnifiedSwarmContext - The unified context type system
 * @see ContextSubscriptionManager - Live update propagation system
 * @see SwarmContextManager - Centralized resource allocation and state management
 * @see /docs/architecture/execution/swarm-state-management-redesign.md - Complete architecture
 */

import {
    generatePK,
    type SwarmId,
} from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { getEventBus } from "../../events/eventBus.js";
import { EventTypes, EventUtils } from "../../events/index.js";
import {
    type ContextQuery,
    type ContextSubscription,
    type ContextUpdateEvent,
    type ContextValidationResult,
    type ResourceAllocation,
    type ResourcePool,
    type UnifiedSwarmContext,
    UnifiedSwarmContextGuards,
} from "./UnifiedSwarmContext.js";

// SwarmContextManager constants
const THIRTY_SECONDS_MS = 30_000;
const SEVEN_DAYS = 7;
const BASE64_CHECKSUM_LENGTH = 16;
const SECONDS_PER_DAY = 86400;

/**
 * Context management operations interface
 */
export interface ISwarmContextManager {
    // Context lifecycle
    createContext(swarmId: SwarmId, initialConfig: Partial<UnifiedSwarmContext>): Promise<UnifiedSwarmContext>;
    getContext(swarmId: SwarmId): Promise<UnifiedSwarmContext | null>;
    updateContext(swarmId: SwarmId, updates: Partial<UnifiedSwarmContext>, reason?: string): Promise<UnifiedSwarmContext>;
    deleteContext(swarmId: SwarmId): Promise<void>;

    // Live updates and subscriptions
    subscribe(subscription: Omit<ContextSubscription, "id" | "metadata">): Promise<string>;
    unsubscribe(subscriptionId: string): Promise<void>;

    // Resource management
    allocateResources(swarmId: SwarmId, request: Omit<ResourceAllocation, "id" | "allocatedAt">): Promise<ResourceAllocation>;
    releaseResources(swarmId: SwarmId, allocationId: string): Promise<void>;
    getResourceStatus(swarmId: SwarmId): Promise<{ total: ResourcePool; allocated: ResourceAllocation[]; available: ResourcePool }>;

    // Context querying and validation
    query(query: ContextQuery): Promise<Partial<UnifiedSwarmContext>>;
    validate(context: UnifiedSwarmContext): Promise<ContextValidationResult>;

    // System management
    healthCheck(): Promise<{ healthy: boolean; details: Record<string, any> }>;
    getMetrics(): Promise<Record<string, number>>;
}

/**
 * Configuration for SwarmContextManager
 */
export interface SwarmContextManagerConfig {
    /** Redis TTL for context data (seconds) */
    contextTTL: number;

    /** Redis TTL for subscription data (seconds) */
    subscriptionTTL: number;

    /** Maximum number of versions to keep per context */
    maxVersionHistory: number;

    /** Batch size for bulk operations */
    batchSize: number;

    /** Enable/disable context validation */
    enableValidation: boolean;

    /** Enable/disable context compression */
    enableCompression: boolean;

    /** Pub/sub channel prefix */
    pubsubChannelPrefix: string;
}

/**
 * Internal context storage format with versioning and metadata
 */
interface ContextStorageRecord {
    context: UnifiedSwarmContext;
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
    private static readonly DEFAULT_CACHE_TTL_MS = THIRTY_SECONDS_MS;
    private static readonly DEFAULT_CONTEXT_TTL_DAYS = SEVEN_DAYS;
    private static readonly CHECKSUM_LENGTH = BASE64_CHECKSUM_LENGTH;

    private readonly config: SwarmContextManagerConfig;

    // Subscription management
    private readonly subscriptions = new Map<string, ContextSubscription>();
    private subscriptionHandler?: (channel: string, message: string) => void;

    // Caching and performance
    private readonly contextCache = new Map<SwarmId, ContextStorageRecord>();
    private readonly cacheTTL = SwarmContextManager.DEFAULT_CACHE_TTL_MS;

    // Metrics tracking
    private readonly metrics = {
        contextsCreated: 0,
        contextsUpdated: 0,
        contextsDeleted: 0,
        subscriptionsCreated: 0,
        subscriptionsNotified: 0,
        resourceAllocations: 0,
        cacheHits: 0,
        cacheMisses: 0,
        validationErrors: 0,
    };

    constructor(
        config: Partial<SwarmContextManagerConfig> = {},
    ) {
        this.config = {
            contextTTL: SECONDS_PER_DAY * SwarmContextManager.DEFAULT_CONTEXT_TTL_DAYS,
            subscriptionTTL: SECONDS_PER_DAY, // 1 day
            maxVersionHistory: 10,
            batchSize: 50,
            enableValidation: true,
            enableCompression: false, // Disabled initially for debugging
            pubsubChannelPrefix: "swarm_context:",
            ...config,
        };

        // Redis connection provided via constructor

        logger.info("[SwarmContextManager] Initialized with emergent capabilities", {
            config: this.config,
        });
    }

    /**
     * Initialize the context manager (alias for start() for backward compatibility)
     */
    async initialize(): Promise<void> {
        return this.start();
    }

    /**
     * Start the context manager and setup pub/sub handlers
     */
    async start(): Promise<void> {
        try {
            // Setup pub/sub subscription handler for live updates
            await this.setupPubSubHandler();

            logger.info("[SwarmContextManager] Started successfully");
        } catch (error) {
            logger.error("[SwarmContextManager] Failed to start", {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Stop the context manager and cleanup resources
     */
    async stop(): Promise<void> {
        try {
            // Cleanup subscriptions
            this.subscriptions.clear();

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
        swarmId: SwarmId,
        initialConfig: Partial<UnifiedSwarmContext> = {},
    ): Promise<UnifiedSwarmContext> {
        logger.debug("[SwarmContextManager] Creating new swarm context", { swarmId });

        try {
            // Check if context already exists
            const existing = await this.getContext(swarmId);
            if (existing) {
                throw new Error(`Swarm context already exists: ${swarmId}`);
            }

            // Create default context that enables emergent capabilities
            const defaultContext = this.createDefaultContext(swarmId);

            // Merge with provided configuration
            const context: UnifiedSwarmContext = {
                ...defaultContext,
                ...initialConfig,
                // Ensure core fields cannot be overridden
                swarmId,
                version: 1,
                createdAt: new Date(),
                lastUpdated: new Date(),
                updatedBy: initialConfig.updatedBy || "system",
            };

            // Validate context
            if (this.config.enableValidation) {
                const validation = await this.validate(context);
                if (!validation.valid) {
                    throw new Error(`Context validation failed: ${validation.errors.map(e => e.message).join(", ")}`);
                }
            }

            // Store context
            await this.storeContext(context);

            // Update metrics
            this.metrics.contextsCreated++;

            logger.info("[SwarmContextManager] Created swarm context with emergent capabilities", {
                swarmId,
                version: context.version,
                emergentFeaturesEnabled: Object.values(context.configuration.features).filter(Boolean).length,
            });

            return context;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to create context", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get current swarm context with caching optimization
     */
    async getContext(swarmId: SwarmId): Promise<UnifiedSwarmContext | null> {
        try {
            // Check in-memory cache first
            const cached = this.contextCache.get(swarmId);
            if (cached && this.isCacheValid(cached)) {
                this.metrics.cacheHits++;
                cached.metadata.accessCount++;
                cached.metadata.lastAccessed = new Date();
                return cached.context;
            }

            this.metrics.cacheMisses++;

            // Load from Redis
            const redis = this.ensureRedisConnected();
            const key = this.getContextKey(swarmId);
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
                        swarmId,
                        recordKeys: Object.keys(parsed),
                    });
                    return null;
                }
                record = parsed;
            } catch (parseError) {
                logger.error("[SwarmContextManager] Failed to parse context record", {
                    swarmId,
                    error: parseError instanceof Error ? parseError.message : String(parseError),
                    dataLength: data.length,
                });
                return null;
            }

            // Update access metadata
            record.metadata.accessCount++;
            record.metadata.lastAccessed = new Date();

            // Store in cache
            this.contextCache.set(swarmId, record);

            // Update access count in Redis (fire and forget)
            redis.set(key, JSON.stringify(record), "EX", this.config.contextTTL).catch(error => {
                logger.warn("[SwarmContextManager] Failed to update access metadata", {
                    swarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            });

            return record.context;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to get context", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    /**
     * Update swarm context with atomic versioning and live propagation
     */
    async updateContext(
        swarmId: SwarmId,
        updates: Partial<UnifiedSwarmContext>,
        reason = "Context update",
    ): Promise<UnifiedSwarmContext> {
        logger.debug("[SwarmContextManager] Updating swarm context", {
            swarmId,
            reason,
            updateKeys: Object.keys(updates),
        });

        try {
            // Get current context
            const currentContext = await this.getContext(swarmId);
            if (!currentContext) {
                throw new Error(`Swarm context not found: ${swarmId}`);
            }

            // Create updated context with version increment
            const updatedContext: UnifiedSwarmContext = {
                ...currentContext,
                ...updates,
                // Ensure core fields are properly managed
                swarmId, // Cannot be changed
                version: currentContext.version + 1,
                lastUpdated: new Date(),
                updatedBy: updates.updatedBy || "system",
            };

            // Validate updated context
            if (this.config.enableValidation) {
                const validation = await this.validate(updatedContext);
                if (!validation.valid) {
                    throw new Error(`Context validation failed: ${validation.errors.map(e => e.message).join(", ")}`);
                }
            }

            // Store updated context atomically
            await this.storeContext(updatedContext);

            // Clear cache for this swarm
            this.contextCache.delete(swarmId);

            // Create and publish update event for live propagation
            const updateEvent: ContextUpdateEvent = {
                swarmId,
                previousVersion: currentContext.version,
                newVersion: updatedContext.version,
                changes: this.computeChanges(currentContext, updatedContext),
                updatedBy: updatedContext.updatedBy,
                timestamp: new Date(),
                reason,
                emergent: this.isEmergentUpdate(updates),
            };

            await this.publishUpdateEvent(updateEvent);

            // Update metrics
            this.metrics.contextsUpdated++;

            logger.info("[SwarmContextManager] Updated swarm context with live propagation", {
                swarmId,
                previousVersion: currentContext.version,
                newVersion: updatedContext.version,
                changesCount: updateEvent.changes.length,
                emergent: updateEvent.emergent,
            });

            return updatedContext;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to update context", {
                swarmId,
                reason,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Delete swarm context and cleanup all related data
     */
    async deleteContext(swarmId: SwarmId): Promise<void> {
        logger.debug("[SwarmContextManager] Deleting swarm context", { swarmId });

        try {
            // Remove from cache
            this.contextCache.delete(swarmId);

            // Remove from Redis
            const redis = this.ensureRedisConnected();
            const key = this.getContextKey(swarmId);
            await redis.del(key);

            // Cleanup version history
            const historyPattern = `${key}:version:*`;
            const historyKeys = await redis.keys(historyPattern);
            if (historyKeys.length > 0) {
                await redis.del(...historyKeys);
            }

            // Cleanup subscriptions for this swarm
            const swarmSubscriptions = Array.from(this.subscriptions.values())
                .filter(sub => sub.swarmId === swarmId);

            for (const subscription of swarmSubscriptions) {
                await this.unsubscribe(subscription.id);
            }

            // Update metrics
            this.metrics.contextsDeleted++;

            logger.info("[SwarmContextManager] Deleted swarm context and cleanup completed", {
                swarmId,
                cleanupSubscriptions: swarmSubscriptions.length,
                cleanupVersions: historyKeys.length,
            });

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to delete context", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Subscribe to live context updates for emergent coordination
     */
    async subscribe(
        subscriptionRequest: Omit<ContextSubscription, "id" | "metadata">,
    ): Promise<string> {
        const subscriptionId = generatePK();

        const subscription: ContextSubscription = {
            ...subscriptionRequest,
            id: subscriptionId,
            metadata: {
                createdAt: new Date(),
                lastNotified: new Date(),
                totalNotifications: 0,
                subscriptionType: "state_machine", // Default type
            },
        };

        this.subscriptions.set(subscriptionId, subscription);
        this.metrics.subscriptionsCreated++;

        logger.debug("[SwarmContextManager] Created context subscription", {
            subscriptionId,
            swarmId: subscription.swarmId,
            subscriberId: subscription.subscriberId,
            watchPaths: subscription.watchPaths,
        });

        return subscriptionId;
    }

    /**
     * Unsubscribe from context updates
     */
    async unsubscribe(subscriptionId: string): Promise<void> {
        const subscription = this.subscriptions.get(subscriptionId);
        if (subscription) {
            this.subscriptions.delete(subscriptionId);

            logger.debug("[SwarmContextManager] Removed context subscription", {
                subscriptionId,
                swarmId: subscription.swarmId,
                subscriberId: subscription.subscriberId,
                totalNotifications: subscription.metadata.totalNotifications,
            });
        }
    }

    /**
     * Allocate resources with hierarchical tracking for emergent optimization
     */
    async allocateResources(
        swarmId: SwarmId,
        request: Omit<ResourceAllocation, "id" | "allocatedAt">,
    ): Promise<ResourceAllocation> {
        logger.debug("[SwarmContextManager] Allocating resources", {
            swarmId,
            consumerId: request.consumerId,
            consumerType: request.consumerType,
        });

        try {
            const context = await this.getContext(swarmId);
            if (!context) {
                throw new Error(`Swarm context not found: ${swarmId}`);
            }

            // Create resource allocation
            const allocation: ResourceAllocation = {
                ...request,
                id: generatePK(),
                allocatedAt: new Date(),
            };

            // Validate allocation against available resources
            const validationResult = this.validateResourceAllocation(context, allocation);
            if (!validationResult.valid) {
                throw new Error(`Resource allocation validation failed: ${validationResult.reason}`);
            }

            // Update context with new allocation
            const updatedAllocations = [...context.resources.allocated, allocation];
            const updatedAvailable = this.calculateAvailableResources(
                context.resources.total,
                updatedAllocations,
            );

            await this.updateContext(swarmId, {
                resources: {
                    ...context.resources,
                    allocated: updatedAllocations,
                    available: updatedAvailable,
                },
            }, `Resource allocation for ${allocation.consumerType} ${allocation.consumerId}`);

            this.metrics.resourceAllocations++;

            logger.info("[SwarmContextManager] Allocated resources with emergent tracking", {
                swarmId,
                allocationId: allocation.id,
                consumerId: allocation.consumerId,
                consumerType: allocation.consumerType,
                creditsAllocated: allocation.allocation.maxCredits,
            });

            return allocation;

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to allocate resources", {
                swarmId,
                consumerId: request.consumerId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Release resources and update availability
     */
    async releaseResources(swarmId: SwarmId, allocationId: string): Promise<void> {
        logger.debug("[SwarmContextManager] Releasing resources", { swarmId, allocationId });

        try {
            const context = await this.getContext(swarmId);
            if (!context) {
                throw new Error(`Swarm context not found: ${swarmId}`);
            }

            // Find and remove allocation
            const allocationIndex = context.resources.allocated.findIndex(a => a.id === allocationId);
            if (allocationIndex === -1) {
                throw new Error(`Resource allocation not found: ${allocationId}`);
            }

            const allocation = context.resources.allocated[allocationIndex];
            const updatedAllocations = context.resources.allocated.filter(a => a.id !== allocationId);
            const updatedAvailable = this.calculateAvailableResources(
                context.resources.total,
                updatedAllocations,
            );

            await this.updateContext(swarmId, {
                resources: {
                    ...context.resources,
                    allocated: updatedAllocations,
                    available: updatedAvailable,
                },
            }, `Resource release for ${allocation.consumerType} ${allocation.consumerId}`);

            logger.info("[SwarmContextManager] Released resources", {
                swarmId,
                allocationId,
                consumerId: allocation.consumerId,
                consumerType: allocation.consumerType,
                creditsReleased: allocation.allocation.maxCredits,
            });

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to release resources", {
                swarmId,
                allocationId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get current resource status for monitoring and optimization
     */
    async getResourceStatus(swarmId: SwarmId): Promise<{
        total: ResourcePool;
        allocated: ResourceAllocation[];
        available: ResourcePool;
    }> {
        const context = await this.getContext(swarmId);
        if (!context) {
            throw new Error(`Swarm context not found: ${swarmId}`);
        }

        return {
            total: context.resources.total,
            allocated: context.resources.allocated,
            available: context.resources.available,
        };
    }

    /**
     * Query context data with JSONPath support
     */
    async query(query: ContextQuery): Promise<Partial<UnifiedSwarmContext>> {
        // Implementation would include JSONPath querying logic
        // For now, return basic implementation
        const context = await this.getContext(query.swarmId);
        if (!context) {
            return {};
        }

        // TODO: Implement full JSONPath querying
        return context;
    }

    /**
     * Validate context integrity and constraints
     */
    async validate(context: UnifiedSwarmContext): Promise<ContextValidationResult> {
        const startTime = Date.now();
        const errors: ContextValidationResult["errors"] = [];
        const warnings: ContextValidationResult["warnings"] = [];

        // Basic type validation
        if (!UnifiedSwarmContextGuards.isUnifiedSwarmContext(context)) {
            errors.push({
                path: "root",
                message: "Context does not match UnifiedSwarmContext schema",
                severity: "error",
            });
        }

        // Version validation
        if (context.version < 1) {
            errors.push({
                path: "version",
                message: "Context version must be >= 1",
                severity: "error",
            });
        }

        // Resource validation
        if (!UnifiedSwarmContextGuards.isResourcePool(context.resources.total)) {
            errors.push({
                path: "resources.total",
                message: "Invalid resource pool format",
                severity: "error",
            });
        }

        // TODO: Add more comprehensive validation rules

        const validationTimeMs = Date.now() - startTime;

        return {
            valid: errors.length === 0,
            errors,
            warnings,
            metrics: {
                validationTimeMs,
                rulesChecked: 3, // TODO: Count actual rules
                constraintsValidated: 1,
            },
        };
    }

    /**
     * Health check for monitoring
     */
    async healthCheck(): Promise<{ healthy: boolean; details: Record<string, any> }> {
        try {
            // Test Redis connection
            const redis = this.ensureRedisConnected();
            await redis.ping();

            return {
                healthy: true,
                details: {
                    redis: "connected",
                    cacheSize: this.contextCache.size,
                    subscriptions: this.subscriptions.size,
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

    private isValidContextStorageRecord(obj: any): obj is ContextStorageRecord {
        return obj &&
            typeof obj === "object" &&
            typeof obj.context === "object" &&
            typeof obj.metadata === "object" &&
            typeof obj.metadata.storageVersion === "number" &&
            typeof obj.metadata.compressed === "boolean" &&
            typeof obj.metadata.checksum === "string" &&
            typeof obj.metadata.accessCount === "number" &&
            obj.metadata.lastAccessed &&
            // Validate the context using existing guard
            UnifiedSwarmContextGuards.isUnifiedSwarmContext(obj.context);
    }

    private async storeContext(context: UnifiedSwarmContext): Promise<void> {
        const record: ContextStorageRecord = {
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
        const data = JSON.stringify(record);

        await redis.set(key, data, "EX", this.config.contextTTL);

        // Store version history
        const versionKey = `${key}:version:${context.version}`;
        await redis.set(versionKey, data, "EX", this.config.contextTTL);

        // Cleanup old versions
        await this.cleanupOldVersions(context.swarmId);
    }

    private async cleanupOldVersions(swarmId: SwarmId): Promise<void> {
        const redis = this.ensureRedisConnected();
        const baseKey = this.getContextKey(swarmId);
        const pattern = `${baseKey}:version:*`;
        const versionKeys = await redis.keys(pattern);

        if (versionKeys.length > this.config.maxVersionHistory) {
            // Sort by version number and remove oldest
            const sortedKeys = versionKeys.sort((a, b) => {
                const versionA = parseInt(a.split(":version:")[1]);
                const versionB = parseInt(b.split(":version:")[1]);
                return versionA - versionB;
            });

            const keysToDelete = sortedKeys.slice(0, sortedKeys.length - this.config.maxVersionHistory);
            if (keysToDelete.length > 0) {
                await redis.del(...keysToDelete);
            }
        }
    }

    private createDefaultContext(swarmId: SwarmId): UnifiedSwarmContext {
        // Create a default context that enables emergent capabilities
        const now = new Date();

        return {
            swarmId,
            version: 1,
            createdAt: now,
            lastUpdated: now,
            updatedBy: "system",

            resources: {
                total: {
                    credits: "10000", // Default credit allocation
                    durationMs: 3600000, // 1 hour
                    memoryMB: 2048, // 2GB
                    concurrentExecutions: 10,
                    models: {},
                    tools: {},
                },
                allocated: [],
                available: {
                    credits: "10000",
                    durationMs: 3600000,
                    memoryMB: 2048,
                    concurrentExecutions: 10,
                    models: {},
                    tools: {},
                },
                usageHistory: [],
            },

            policy: {
                security: {
                    permissions: {},
                    scanning: {
                        enabledScanners: ["xss", "pii"],
                        blockOnViolation: true,
                        alertingThresholds: {},
                    },
                    toolApproval: {
                        requireApprovalForTools: [],
                        autoApproveForRoles: ["admin"],
                        approvalTimeoutMs: 300000, // 5 minutes
                    },
                },
                resource: {
                    allocation: {
                        strategy: "elastic",
                        tierAllocationRatios: {
                            tier1ToTier2: 0.8,
                            tier2ToTier3: 0.6,
                        },
                        bufferPercentages: {
                            emergency: 10,
                            optimization: 5,
                            parallel: 15,
                        },
                        contention: {
                            strategy: "priority_based",
                            preemptionEnabled: false,
                            priorityWeights: {
                                low: 1,
                                medium: 2,
                                high: 4,
                                critical: 8,
                            },
                        },
                    },
                    thresholds: {
                        resourceUtilization: {
                            warning: 70,
                            critical: 90,
                            optimization: 60,
                        },
                        latency: {
                            targetMs: 5000,
                            warningMs: 15000,
                            criticalMs: 30000,
                        },
                        failureRate: {
                            warningPercent: 5,
                            criticalPercent: 15,
                        },
                    },
                    history: {
                        recentAllocations: [],
                        performanceMetrics: {
                            avgUtilization: 0,
                            peakUtilization: 0,
                            bottleneckFrequency: {},
                        },
                        optimizationHistory: [],
                    },
                },
                organizational: {
                    structure: {
                        hierarchy: [],
                        groups: [],
                        dependencies: [],
                    },
                    functional: {
                        missions: [],
                        goals: [],
                    },
                    normative: {
                        norms: [],
                        sanctions: [],
                    },
                },
            },

            configuration: {
                timeouts: {
                    routineExecutionMs: 300000, // 5 minutes
                    stepExecutionMs: 30000,     // 30 seconds
                    approvalTimeoutMs: 300000,  // 5 minutes
                    idleTimeoutMs: 600000,      // 10 minutes
                },
                retries: {
                    maxRetries: 3,
                    backoffStrategy: "exponential",
                    baseDelayMs: 1000,
                    maxDelayMs: 30000,
                },
                features: {
                    emergentGoalGeneration: true,    // Enable emergent capabilities
                    adaptiveResourceAllocation: true,
                    crossSwarmCommunication: false,  // Disabled by default for security
                    autonomousToolApproval: false,   // Disabled by default for security
                    contextualLearning: true,
                },
                coordination: {
                    maxParallelAgents: 5,
                    communicationProtocol: "event_driven",
                    consensusThreshold: 0.6,
                    leadershipElection: "automatic",
                },
            },

            blackboard: {
                items: {},
                subscriptions: [],
            },

            execution: {
                status: "initializing",
                teams: [],
                agents: [],
                activeRuns: [],
            },

            metadata: {
                createdBy: "system",
                subscribers: [],
                emergencyContacts: [],
                retentionPolicy: {
                    keepHistoryDays: 30,
                    archiveAfterDays: 90,
                    deleteAfterDays: 365,
                },
                diagnostics: {
                    contextSize: 0,
                    updateFrequency: 0,
                    subscriptionCount: 0,
                    lastOptimization: now,
                },
            },
        };
    }

    private getContextKey(swarmId: SwarmId): string {
        return `swarm_context:${swarmId}`;
    }

    private isCacheValid(record: ContextStorageRecord): boolean {
        const age = Date.now() - record.metadata.lastAccessed.getTime();
        return age < this.cacheTTL;
    }

    private calculateChecksum(context: UnifiedSwarmContext): string {
        // Simple checksum implementation - in production, use proper hashing
        return Buffer.from(JSON.stringify(context)).toString("base64").substring(0, SwarmContextManager.CHECKSUM_LENGTH);
    }

    private computeChanges(
        oldContext: UnifiedSwarmContext,
        newContext: UnifiedSwarmContext,
    ): ContextUpdateEvent["changes"] {
        // Simplified change detection - in production, use proper diff algorithm
        const changes: ContextUpdateEvent["changes"] = [];

        if (oldContext.resources !== newContext.resources) {
            changes.push({
                path: "resources",
                oldValue: oldContext.resources,
                newValue: newContext.resources,
                changeType: "updated",
            });
        }

        if (oldContext.policy !== newContext.policy) {
            changes.push({
                path: "policy",
                oldValue: oldContext.policy,
                newValue: newContext.policy,
                changeType: "updated",
            });
        }

        if (oldContext.configuration !== newContext.configuration) {
            changes.push({
                path: "configuration",
                oldValue: oldContext.configuration,
                newValue: newContext.configuration,
                changeType: "updated",
            });
        }

        return changes;
    }

    private isEmergentUpdate(updates: Partial<UnifiedSwarmContext>): boolean {
        // Check if update was made by an agent or through automated optimization
        return updates.updatedBy?.startsWith("agent_") ||
            updates.updatedBy?.startsWith("optimizer_") ||
            updates.updatedBy?.includes("emergent");
    }

    private async publishUpdateEvent(event: ContextUpdateEvent): Promise<void> {
        try {
            await this.emitUnifiedSwarmStateEvent(event);
            await this.emitDirectSocketEvents(event);

            // Notify local subscriptions (maintained for local state management)
            await this.notifySubscriptions(event);

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to publish update event", {
                swarmId: event.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private async notifySubscriptions(event: ContextUpdateEvent): Promise<void> {
        const swarmSubscriptions = Array.from(this.subscriptions.values())
            .filter(sub => sub.swarmId === event.swarmId);

        for (const subscription of swarmSubscriptions) {
            try {
                // Check if subscription is interested in these changes
                const isInterestedInChanges = event.changes.some(change =>
                    subscription.watchPaths.some(pattern =>
                        this.matchesPath(change.path, pattern),
                    ),
                );

                if (isInterestedInChanges) {
                    await subscription.handler(event);
                    subscription.metadata.lastNotified = new Date();
                    subscription.metadata.totalNotifications++;
                    this.metrics.subscriptionsNotified++;
                }

            } catch (error) {
                logger.error("[SwarmContextManager] Subscription handler failed", {
                    subscriptionId: subscription.id,
                    swarmId: event.swarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
    }

    /**
     * Emit unified events directly (Phase 2: replaces ExecutionEventBusAdapter)
     * 
     * This method implements the direct event emission pattern from the migration plan,
     * enabling emergent capabilities without complex adapter layers.
     */
    private async emitUnifiedSwarmStateEvent(contextEvent: ContextUpdateEvent): Promise<void> {
        try {
            // Emit swarm state update event
            const swarmStateEvent = EventUtils.createBaseEvent(
                EventTypes.STATE_SWARM_UPDATED,
                {
                    entityType: "swarm",
                    entityId: contextEvent.swarmId,
                    previousVersion: contextEvent.previousVersion,
                    newVersion: contextEvent.newVersion,
                    changes: contextEvent.changes.map(change => ({
                        path: change.path,
                        oldValue: change.oldValue,
                        newValue: change.newValue,
                        changeType: change.type,
                    })),
                    emergent: contextEvent.emergent,
                    reason: contextEvent.reason,
                    updatedBy: contextEvent.updatedBy,
                    timestamp: contextEvent.timestamp.toISOString(),
                },
                EventUtils.createEventSource(1, "SwarmContextManager"),
                EventUtils.createEventMetadata(
                    "fire-and-forget",
                    contextEvent.emergent ? "high" : "medium",
                    {
                        tags: ["swarm", "state", "context", contextEvent.emergent ? "emergent" : "standard"],
                        conversationId: contextEvent.swarmId, // Use swarmId as conversationId for routing
                    },
                ),
            );

            await getEventBus().publish(swarmStateEvent);

            // Also emit specific change events for fine-grained subscriptions
            for (const change of contextEvent.changes) {
                await this.emitChangeSpecificEvent(contextEvent, change);
            }

            logger.debug("[SwarmContextManager] Emitted unified swarm state events", {
                swarmId: contextEvent.swarmId,
                eventType: EventTypes.STATE_SWARM_UPDATED,
                changesCount: contextEvent.changes.length,
                emergent: contextEvent.emergent,
            });

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to emit unified events", {
                swarmId: contextEvent.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't re-throw - this shouldn't break the core context update
        }
    }

    /**
     * Emit change-specific events for fine-grained agent subscriptions
     */
    private async emitChangeSpecificEvent(contextEvent: ContextUpdateEvent, change: any): Promise<void> {
        // Determine event type based on change path
        let eventType = EventTypes.STATE_SWARM_UPDATED;
        if (change.path.startsWith("resources.")) {
            eventType = EventTypes.RESOURCE_SWARM_UPDATED;
        } else if (change.path.startsWith("policy.")) {
            eventType = EventTypes.CONFIG_SWARM_UPDATED;
        } else if (change.path.startsWith("executionState.")) {
            eventType = EventTypes.STATE_RUN_UPDATED;
        }

        const changeEvent = EventUtils.createBaseEvent(
            eventType,
            {
                entityType: "swarm",
                entityId: contextEvent.swarmId,
                changePath: change.path,
                oldValue: change.oldValue,
                newValue: change.newValue,
                changeType: change.type,
                emergent: contextEvent.emergent,
                reason: contextEvent.reason,
                updatedBy: contextEvent.updatedBy,
            },
            EventUtils.createEventSource(1, "SwarmContextManager"),
            EventUtils.createEventMetadata(
                "fire-and-forget",
                "low",
                {
                    tags: ["swarm", "change", change.path.split(".")[0]],
                    conversationId: contextEvent.swarmId,
                },
            ),
        );

        await getEventBus().publish(changeEvent);
    }

    /**
     * Emit socket events directly (Phase 3: replaces SocketEventAdapter)
     * 
     * This method publishes context updates to the unified event bus,
     * which can be consumed by a socket adapter for real-time updates.
     */
    private async emitDirectSocketEvents(contextEvent: ContextUpdateEvent): Promise<void> {
        try {
            // Get the conversation ID for socket room targeting
            const conversationId = await this.getConversationIdFromContext(contextEvent.swarmId);
            if (!conversationId) {
                logger.debug("[SwarmContextManager] No conversation ID found for socket emission", {
                    swarmId: contextEvent.swarmId,
                });
                return;
            }

            // Emit socket events based on change types
            for (const change of contextEvent.changes) {
                await this.emitSocketEventForChange(conversationId, contextEvent, change);
            }

            logger.debug("[SwarmContextManager] Emitted direct socket events", {
                swarmId: contextEvent.swarmId,
                conversationId,
                changesCount: contextEvent.changes.length,
            });

        } catch (error) {
            logger.error("[SwarmContextManager] Failed to emit direct socket events", {
                swarmId: contextEvent.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't re-throw - socket emission failures shouldn't break core functionality
        }
    }

    /**
     * Emit specific socket event based on change type
     */
    private async emitSocketEventForChange(
        conversationId: string,
        contextEvent: ContextUpdateEvent,
        change: any,
    ): Promise<void> {
        const { swarmId } = contextEvent;
        const eventSource = EventUtils.createEventSource("cross-cutting", "SwarmContextManager");
        const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
            conversationId,
            swarmId,
        });

        // Handle different types of changes with unified event emission
        if (change.path.startsWith("executionState.")) {
            // State changes
            await this.emitSwarmStateEvent(conversationId, swarmId, change, eventSource, metadata);
        } else if (change.path.startsWith("resources.")) {
            // Resource changes
            await this.emitSwarmResourceEvent(conversationId, swarmId, change, contextEvent, eventSource, metadata);
        } else if (change.path.startsWith("policy.organizational.")) {
            // Team/organizational changes
            await this.emitSwarmTeamEvent(conversationId, swarmId, change, eventSource, metadata);
        } else if (change.path.startsWith("policy.") || change.path.startsWith("config.")) {
            // Configuration changes
            await this.emitSwarmConfigEvent(conversationId, swarmId, change, eventSource, metadata);
        }
    }

    /**
     * Emit swarm state events
     */
    private async emitSwarmStateEvent(
        conversationId: string,
        swarmId: string,
        change: any,
        eventSource: any,
        metadata: any,
    ): Promise<void> {
        // Map internal state to socket-compatible state
        const socketState = this.mapToSocketState(change.newValue);
        const message = change.path.includes("statusMessage") ? change.newValue : undefined;

        await getEventBus().publish(
            EventUtils.createBaseEvent(
                EventTypes.STATE_SWARM_UPDATED,
                {
                    conversationId,
                    swarmId,
                    state: socketState,
                    message,
                },
                eventSource,
                metadata,
            ),
        );
    }

    /**
     * Emit swarm resource events
     */
    private async emitSwarmResourceEvent(
        conversationId: string,
        swarmId: string,
        _change: any,
        _contextEvent: ContextUpdateEvent,
        eventSource: any,
        metadata: any,
    ): Promise<void> {
        // Get current context to build resource payload
        const context = await this.getContext(swarmId);
        if (!context) return;

        // Build swarm-compatible resource object for event emission
        const swarmResourceData = {
            id: swarmId,
            state: context.executionState.currentStatus,
            resources: {
                allocated: { amount: context.resources.total.credits },
                consumed: { amount: context.resources.total.credits - context.resources.available.credits },
                remaining: { amount: context.resources.available.credits },
            },
        } as any;

        await getEventBus().publish(
            EventUtils.createBaseEvent(
                EventTypes.RESOURCE_SWARM_UPDATED,
                {
                    conversationId,
                    swarmId,
                    resourceData: swarmResourceData,
                },
                eventSource,
                metadata,
            ),
        );
    }

    /**
     * Emit swarm team events
     */
    private async emitSwarmTeamEvent(
        conversationId: string,
        swarmId: string,
        change: any,
        eventSource: any,
        metadata: any,
    ): Promise<void> {
        // Extract team information from the change
        const teamId = change.path.includes("teamId") ? change.newValue : undefined;
        const swarmLeader = change.path.includes("leader") ? change.newValue : undefined;
        const subtaskLeaders = change.path.includes("subtaskLeaders") ? change.newValue : undefined;

        await getEventBus().publish(
            EventUtils.createBaseEvent(
                EventTypes.TEAM_SWARM_UPDATED,
                {
                    conversationId,
                    swarmId,
                    teamId,
                    swarmLeader,
                    subtaskLeaders,
                },
                eventSource,
                metadata,
            ),
        );
    }

    /**
     * Emit swarm config events
     */
    private async emitSwarmConfigEvent(
        conversationId: string,
        swarmId: string,
        change: any,
        eventSource: any,
        metadata: any,
    ): Promise<void> {
        // Build config update from the change
        const configUpdate = {
            [change.path]: change.newValue,
        } as any;

        await getEventBus().publish(
            EventUtils.createBaseEvent(
                EventTypes.CONFIG_SWARM_UPDATED,
                {
                    conversationId,
                    swarmId,
                    configUpdate,
                },
                eventSource,
                metadata,
            ),
        );
    }

    /**
     * Map internal execution state to client-compatible state
     */
    private mapToSocketState(internalState: string): any {
        // Direct mapping for client compatibility
        const stateMap: Record<string, string> = {
            "UNINITIALIZED": "Initializing",
            "STARTING": "Initializing",
            "RUNNING": "Running",
            "IDLE": "PAUSED", // Map IDLE to PAUSED for client compatibility
            "PAUSED": "Paused",
            "STOPPED": "TERMINATED", // Map STOPPED to TERMINATED for client compatibility
            "FAILED": "Failed",
            "TERMINATED": "Cancelled",
        };

        return stateMap[internalState] || "Running";
    }

    /**
     * Get conversation ID from swarm context for socket room targeting
     */
    private async getConversationIdFromContext(swarmId: string): Promise<string | null> {
        try {
            const context = await this.getContext(swarmId);
            if (!context) return null;

            // Get conversation ID from blackboard or use swarmId as fallback
            const conversationIdItem = context.blackboard.items["conversationId"];
            const conversationId = conversationIdItem?.value as string;
            return conversationId || swarmId;

        } catch (error) {
            logger.warn("[SwarmContextManager] Failed to get conversation ID for socket emission", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return swarmId; // Fallback to swarmId
        }
    }

    private matchesPath(path: string, pattern: string): boolean {
        // Simple pattern matching - in production, use proper JSONPath matching
        return path.startsWith(pattern.replace("*", ""));
    }

    private async setupPubSubHandler(): Promise<void> {
        // Redis pub/sub setup would go here
        // For now, just log that it's ready
        logger.debug("[SwarmContextManager] Pub/sub handler ready for live updates");
    }

    private validateResourceAllocation(
        context: UnifiedSwarmContext,
        allocation: ResourceAllocation,
    ): { valid: boolean; reason?: string } {
        // Check if allocation exceeds available resources
        const available = context.resources.available;
        const requested = allocation.allocation;

        // Check credits
        if (requested.maxCredits !== "unlimited") {
            const availableCredits = BigInt(available.credits);
            const requestedCredits = BigInt(requested.maxCredits);
            if (requestedCredits > availableCredits) {
                return {
                    valid: false,
                    reason: `Insufficient credits: requested ${requested.maxCredits}, available ${available.credits}`,
                };
            }
        }

        // Check memory
        if (requested.maxMemoryMB > available.memoryMB) {
            return {
                valid: false,
                reason: `Insufficient memory: requested ${requested.maxMemoryMB}MB, available ${available.memoryMB}MB`,
            };
        }

        // Check duration
        if (requested.maxDurationMs > available.durationMs) {
            return {
                valid: false,
                reason: `Insufficient duration: requested ${requested.maxDurationMs}ms, available ${available.durationMs}ms`,
            };
        }

        return { valid: true };
    }

    private calculateAvailableResources(
        total: ResourcePool,
        allocations: ResourceAllocation[],
    ): ResourcePool {
        // Calculate remaining resources after allocations
        let totalCreditsAllocated = BigInt(0);
        let totalMemoryAllocated = 0;
        let totalDurationAllocated = 0;
        let totalConcurrentAllocated = 0;

        for (const allocation of allocations) {
            if (allocation.allocation.maxCredits !== "unlimited") {
                totalCreditsAllocated += BigInt(allocation.allocation.maxCredits);
            }
            totalMemoryAllocated += allocation.allocation.maxMemoryMB;
            totalDurationAllocated += allocation.allocation.maxDurationMs;
            totalConcurrentAllocated += allocation.allocation.maxConcurrentSteps;
        }

        const totalCredits = total.credits === "unlimited" ?
            BigInt(Number.MAX_SAFE_INTEGER) :
            BigInt(total.credits);

        return {
            credits: total.credits === "unlimited" ?
                "unlimited" :
                (totalCredits - totalCreditsAllocated).toString(),
            durationMs: Math.max(0, total.durationMs - totalDurationAllocated),
            memoryMB: Math.max(0, total.memoryMB - totalMemoryAllocated),
            concurrentExecutions: Math.max(0, total.concurrentExecutions - totalConcurrentAllocated),
            models: { ...total.models }, // TODO: Calculate model-specific availability
            tools: { ...total.tools },   // TODO: Calculate tool-specific availability
        };
    }
}
