/**
 * Unified Resource Manager - Core Resource Infrastructure
 * 
 * This component serves as the foundational layer for all resource management across
 * Vrooli's three-tier execution architecture. It provides deterministic resource
 * allocation, tracking, and lifecycle management while remaining agnostic to
 * optimization strategies that emerge from AI agents.
 * 
 * ## Architecture Role
 * 
 * The Unified Resource Manager implements the **"Reserve → Execute → Return"** pattern:
 * 1. **Reserve**: Allocate resources upfront with clear limits
 * 2. **Execute**: Track actual consumption during operations  
 * 3. **Return**: Clean up and return unused resources to parent tier
 * 
 * This pattern ensures:
 * - Predictable resource consumption across all operations
 * - Simplified conflict resolution and resource contention handling
 * - Clear audit trail for resource usage and cost tracking
 * - Automatic cleanup of expired or abandoned allocations
 * 
 * ## Event-Driven Intelligence Architecture
 * 
 * The manager emits comprehensive resource events that AI agents analyze to provide
 * emergent optimization capabilities:
 * 
 * ### Core Resource Events:
 * - `RESOURCE_ALLOCATED`: New allocation with entity context and limits
 * - `RESOURCE_USAGE`: Real-time usage tracking with cumulative totals
 * - `RESOURCE_RELEASED`: Allocation cleanup and resource return events
 * - `RESOURCE_EXHAUSTED`: Resource limit violations and constraint events
 * 
 * ### Emergent Intelligence from Events:
 * AI agents analyze these events to provide:
 * - **Pattern Recognition**: Usage patterns across entities and time
 * - **Predictive Allocation**: Pre-allocate based on learned patterns
 * - **Cost Optimization**: Identify over-allocation and waste
 * - **Performance Analysis**: Detect bottlenecks and scaling needs
 * - **Anomaly Detection**: Unusual resource usage requiring attention
 * 
 * ## Resource Types and Semantics
 * 
 * ### Computational Resources:
 * - **Credits**: Primary cost tracking (monetary value in cents/credits)
 * - **Time**: Execution duration limits (milliseconds)
 * - **Memory**: Peak memory usage tracking (bytes)  
 * - **Tokens**: LLM token consumption (input/output tokens)
 * - **API Calls**: External service rate limiting (calls/period)
 * 
 * ### Resource Allocation Semantics:
 * - **Credits**: Cumulative consumption (depleted over time)
 * - **Time**: Maximum execution duration (hard timeout limit)
 * - **Memory**: Peak usage tracking (highest point reached)
 * - **Tokens**: Cumulative consumption (input + output tokens)
 * - **API Calls**: Rate-limited consumption (sliding window)
 * 
 * ## Tier Integration Strategy
 * 
 * The unified manager adapts to different tiers through configuration:
 * 
 * ```
 * Tier 1 (Swarm): 100k credits, 1 hour, 1GB memory - Strategic coordination
 * Tier 2 (Run):    10k credits, 5 min, 512MB       - Routine execution  
 * Tier 3 (Step):   1k credits,  1 min, 256MB       - Individual steps
 * ```
 * 
 * This hierarchical approach ensures:
 * - Clear resource boundaries between tiers
 * - Automatic resource distribution and inheritance
 * - Simplified debugging and performance analysis
 * - Consistent resource accounting across all operations
 * 
 * ## Philosophy: Infrastructure without Intelligence
 * 
 * This manager intentionally avoids complex optimization logic, instead providing:
 * - **Reliable Infrastructure**: Deterministic allocation and tracking
 * - **Rich Event Streams**: Comprehensive data for AI agent analysis
 * - **Extensible Design**: Easy integration of new resource types
 * - **Performance Focus**: Fast allocation/deallocation operations
 * 
 * Complex optimization strategies emerge from AI agents that analyze the event
 * streams and propose improvements through routine execution, ensuring the system
 * can evolve and improve without requiring changes to core infrastructure.
 */

import { logger } from "../../../../events/logger.js";
import {
    EventTypes,
    EventUtils,
    type IEventBus,
} from "../../../events/index.js";
import { ErrorHandler, type ComponentErrorHandler } from "../../shared/ErrorHandler.js";
import { type UsageTracker } from "./usageTracker.js";
import { type RateLimiter } from "./rateLimiter.js";

/**
 * Resource allocation
 */
export interface ResourceAllocation {
    id: string;
    entityId: string; // swarmId, runId, or userId depending on tier
    entityType: "swarm" | "team" | "run" | "step" | "user";
    resources: ResourceAmount;
    allocated: Date;
    expires?: Date;
    parentAllocationId?: string;
}

/**
 * Resource amounts
 */
export interface ResourceAmount {
    credits?: number;
    time?: number; // milliseconds
    memory?: number; // bytes
    tokens?: number;
    apiCalls?: number;
}

/**
 * Resource manager configuration
 */
export interface ResourceManagerConfig {
    tier: 1 | 2 | 3;
    defaultLimits: ResourceAmount;
    cleanupInterval?: number; // milliseconds
    eventTopic?: string;
}

/**
 * Unified Resource Manager
 * 
 * Provides basic resource allocation, tracking, and cleanup.
 * Tier-specific behavior is added through adapters.
 */
export class ResourceManager {
    private readonly eventBus: IEventBus;
    private readonly config: ResourceManagerConfig;
    private readonly allocations = new Map<string, ResourceAllocation>();
    private readonly usage = new Map<string, ResourceAmount>();
    private cleanupTimer?: NodeJS.Timer;
    private errorHandler: ComponentErrorHandler;

    // Optional shared components (injected as needed)
    private rateLimiter?: RateLimiter;
    private usageTracker?: UsageTracker;

    constructor(
        eventBus: IEventBus,
        config: ResourceManagerConfig,
        components?: {
            rateLimiter?: RateLimiter;
            usageTracker?: UsageTracker;
        },
    ) {
        this.eventBus = eventBus;
        this.config = config;
        this.errorHandler = new ErrorHandler(logger).createComponentHandler(`ResourceManager-Tier${config.tier}`);
        this.rateLimiter = components?.rateLimiter;
        this.usageTracker = components?.usageTracker;

        // Start cleanup timer
        if (config.cleanupInterval) {
            this.startCleanupTimer();
        }
    }

    /**
     * Allocates resources to an entity
     */
    async allocate(
        entityId: string,
        entityType: ResourceAllocation["entityType"],
        requested: ResourceAmount,
        parentAllocationId?: string,
    ): Promise<ResourceAllocation> {
        logger.debug("[ResourceManager] Allocating resources", {
            tier: this.config.tier,
            entityId,
            entityType,
            requested,
        });

        // Create allocation
        const allocation: ResourceAllocation = {
            id: `alloc-${Date.now()}-${Math.random()}`,
            entityId,
            entityType,
            resources: this.applyLimits(requested),
            allocated: new Date(),
            parentAllocationId,
        };

        // Store allocation
        this.allocations.set(allocation.id, allocation);

        // Initialize usage tracking
        if (!this.usage.has(entityId)) {
            this.usage.set(entityId, {
                credits: 0,
                time: 0,
                memory: 0,
                tokens: 0,
                apiCalls: 0,
            });
        }

        // Emit allocation event using unified event system
        await this.eventBus.publish({
            ...EventUtils.createBaseEvent(
                EventTypes.RESOURCE_ALLOCATED,
                {
                    allocation,
                    tier: this.config.tier,
                },
                EventUtils.createEventSource("cross-cutting", "resource-manager"),
                EventUtils.createEventMetadata("reliable", "medium", {
                    tags: ["resource", "allocation"],
                }),
            ),
        });

        return allocation;
    }

    /**
     * Tracks resource usage
     */
    async trackUsage(
        entityId: string,
        used: ResourceAmount,
    ): Promise<void> {
        const currentUsage = this.usage.get(entityId) || {};

        // Update usage
        const updatedUsage: ResourceAmount = {
            credits: (currentUsage.credits || 0) + (used.credits || 0),
            time: (currentUsage.time || 0) + (used.time || 0),
            memory: Math.max(currentUsage.memory || 0, used.memory || 0), // Memory is peak, not cumulative
            tokens: (currentUsage.tokens || 0) + (used.tokens || 0),
            apiCalls: (currentUsage.apiCalls || 0) + (used.apiCalls || 0),
        };

        this.usage.set(entityId, updatedUsage);

        // Emit usage event using unified event system with fire-and-forget for performance
        await this.eventBus.publish({
            ...EventUtils.createBaseEvent(
                EventTypes.RESOURCE_USAGE,
                {
                    entityId,
                    used,
                    total: updatedUsage,
                    tier: this.config.tier,
                },
                EventUtils.createEventSource("cross-cutting", "resource-manager"),
                EventUtils.createEventMetadata("fire-and-forget", "low", {
                    tags: ["resource", "usage", "metrics"],
                }),
            ),
        });
    }

    /**
     * Releases allocated resources
     */
    async release(allocationId: string): Promise<void> {
        const allocation = this.allocations.get(allocationId);
        if (!allocation) {
            logger.warn("[ResourceManager] Attempted to release unknown allocation", {
                allocationId,
            });
            return;
        }

        // Remove allocation
        this.allocations.delete(allocationId);

        // Emit release event using unified event system
        await this.eventBus.publish({
            ...EventUtils.createBaseEvent(
                EventTypes.RESOURCE_RELEASED,
                {
                    allocation,
                    tier: this.config.tier,
                },
                EventUtils.createEventSource("cross-cutting", "resource-manager"),
                EventUtils.createEventMetadata("reliable", "medium", {
                    tags: ["resource", "release"],
                }),
            ),
        });
    }

    /**
     * Gets current usage for an entity
     */
    getUsage(entityId: string): ResourceAmount {
        return this.usage.get(entityId) || {
            credits: 0,
            time: 0,
            memory: 0,
            tokens: 0,
            apiCalls: 0,
        };
    }

    /**
     * Gets all allocations for an entity
     */
    getAllocations(entityId: string): ResourceAllocation[] {
        return Array.from(this.allocations.values())
            .filter(alloc => alloc.entityId === entityId);
    }

    /**
     * Checks if entity can use requested resources
     */
    async canUse(entityId: string, requested: ResourceAmount): Promise<boolean> {
        const allocations = this.getAllocations(entityId);
        const usage = this.getUsage(entityId);

        // Check each resource type
        for (const allocation of allocations) {
            const allocated = allocation.resources;

            if (requested.credits && allocated.credits) {
                if ((usage.credits || 0) + requested.credits > allocated.credits) {
                    // Emit resource exhausted event
                    await this.eventBus.publish({
                        ...EventUtils.createBaseEvent(
                            EventTypes.RESOURCE_EXHAUSTED,
                            {
                                entityId,
                                resourceType: "credits",
                                requested: requested.credits,
                                used: usage.credits || 0,
                                allocated: allocated.credits,
                                tier: this.config.tier,
                            },
                            EventUtils.createEventSource("cross-cutting", "resource-manager"),
                            EventUtils.createEventMetadata("reliable", "high", {
                                tags: ["resource", "exhausted", "credits"],
                            }),
                        ),
                    });
                    return false;
                }
            }

            if (requested.tokens && allocated.tokens) {
                if ((usage.tokens || 0) + requested.tokens > allocated.tokens) {
                    // Emit resource exhausted event
                    await this.eventBus.publish({
                        ...EventUtils.createBaseEvent(
                            EventTypes.RESOURCE_EXHAUSTED,
                            {
                                entityId,
                                resourceType: "tokens",
                                requested: requested.tokens,
                                used: usage.tokens || 0,
                                allocated: allocated.tokens,
                                tier: this.config.tier,
                            },
                            EventUtils.createEventSource("cross-cutting", "resource-manager"),
                            EventUtils.createEventMetadata("reliable", "high", {
                                tags: ["resource", "exhausted", "tokens"],
                            }),
                        ),
                    });
                    return false;
                }
            }

            // Add other resource checks as needed
        }

        // Check rate limits if available
        if (this.rateLimiter && requested.apiCalls) {
            const allowed = await this.rateLimiter.checkLimit(
                entityId,
                "api_calls",
                requested.apiCalls,
            );
            if (!allowed) {
                return false;
            }
        }

        return true;
    }

    /**
     * Private helper methods
     */
    private applyLimits(requested: ResourceAmount): ResourceAmount {
        const limits = this.config.defaultLimits;

        return {
            credits: Math.min(requested.credits || 0, limits.credits || Infinity),
            time: Math.min(requested.time || 0, limits.time || Infinity),
            memory: Math.min(requested.memory || 0, limits.memory || Infinity),
            tokens: Math.min(requested.tokens || 0, limits.tokens || Infinity),
            apiCalls: Math.min(requested.apiCalls || 0, limits.apiCalls || Infinity),
        };
    }


    private startCleanupTimer(): void {
        this.cleanupTimer = setInterval(async () => {
            await this.cleanupExpiredAllocations();
        }, this.config.cleanupInterval);
    }

    private async cleanupExpiredAllocations(): Promise<void> {
        const now = new Date();
        const expired: string[] = [];

        for (const [id, allocation] of this.allocations) {
            if (allocation.expires && allocation.expires < now) {
                expired.push(id);
            }
        }

        for (const id of expired) {
            await this.release(id);
        }

        if (expired.length > 0) {
            logger.info("[ResourceManager] Cleaned up expired allocations", {
                count: expired.length,
                tier: this.config.tier,
            });
        }
    }

    /**
     * Stops the resource manager
     */
    async stop(): Promise<void> {
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
            this.cleanupTimer = undefined;
        }

        logger.info("[ResourceManager] Stopped", {
            tier: this.config.tier,
        });
    }
}
