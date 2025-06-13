/**
 * Unified Resource Manager - Shared resource management across all tiers
 * 
 * This component provides core resource management functionality that can be
 * adapted for different tiers through configuration and adapters.
 * 
 * Complex optimization and prediction capabilities emerge from resource agents
 * analyzing the events emitted by this manager.
 */

import { logger } from "../../../../events/logger.js";
import { RedisEventBus } from "../events/eventBus.js";
import { EventPublisher } from "../../shared/EventPublisher.js";
import { ErrorHandler, ComponentErrorHandler } from "../../shared/ErrorHandler.js";
import { RateLimiter } from "./rateLimiter.js";
import { ResourcePool, ResourcePoolManager } from "./resourcePool.js";
import { UsageTracker } from "./usageTracker.js";

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
    private readonly eventBus: RedisEventBus;
    private readonly eventPublisher: EventPublisher;
    private readonly errorHandler: ComponentErrorHandler;
    private readonly config: ResourceManagerConfig;
    private readonly allocations = new Map<string, ResourceAllocation>();
    private readonly usage = new Map<string, ResourceAmount>();
    private cleanupTimer?: NodeJS.Timer;

    // Optional shared components (injected as needed)
    private rateLimiter?: RateLimiter;
    private poolManager?: ResourcePoolManager;
    private usageTracker?: UsageTracker;

    constructor(
        eventBus: RedisEventBus,
        config: ResourceManagerConfig,
        components?: {
            rateLimiter?: RateLimiter;
            poolManager?: ResourcePoolManager;
            usageTracker?: UsageTracker;
        },
    ) {
        this.eventBus = eventBus;
        this.config = config;
        this.eventPublisher = new EventPublisher(eventBus, logger, `ResourceManager-Tier${config.tier}`);
        this.errorHandler = new ErrorHandler(logger, this.eventPublisher).createComponentHandler(`ResourceManager-Tier${config.tier}`);
        this.rateLimiter = components?.rateLimiter;
        this.poolManager = components?.poolManager;
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

        // Emit allocation event
        await this.emitResourceEvent("RESOURCE_ALLOCATED", {
            allocation,
            tier: this.config.tier,
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

        // Track in usage tracker if available
        if (this.usageTracker) {
            await this.usageTracker.track(entityId, used);
        }

        // Emit usage event
        await this.emitResourceEvent("RESOURCE_USAGE", {
            entityId,
            used,
            total: updatedUsage,
            tier: this.config.tier,
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

        // Emit release event
        await this.emitResourceEvent("RESOURCE_RELEASED", {
            allocation,
            tier: this.config.tier,
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
                    return false;
                }
            }
            
            if (requested.tokens && allocated.tokens) {
                if ((usage.tokens || 0) + requested.tokens > allocated.tokens) {
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

    private async emitResourceEvent(type: string, data: any): Promise<void> {
        await this.eventPublisher.publish(`resource.${type.toLowerCase()}`, {
            tier: this.config.tier,
            ...data,
        });
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