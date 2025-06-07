/**
 * Unified Resource Manager - Central coordination for all resource allocation and tracking
 * 
 * This component provides:
 * - Hierarchical allocation (Swarm → Run → Step)
 * - Real-time usage tracking across all tiers
 * - Budget enforcement and alerts
 * - Resource pooling and sharing
 * - Usage reports and analytics
 */

import { type Logger } from "winston";
import {
    type ResourceAllocation as SharedResourceAllocation,
    type ResourceUsage as SharedResourceUsage,
    type ResourceType as SharedResourceType,
    type AllocationPriority,
    type AllocationResult,
    type AllocatedResource,
    type DeniedResource,
    type ResourceLimitConfig,
    type LimitScope,
    type ResourceLimit,
    type ResourceAccounting,
    type ResourceUsageSummary,
    type ResourceCostSummary,
    type EfficiencyMetrics,
    type OptimizationSuggestion,
    type ResourceConflict,
    type ConflictResolution,
    SwarmEventType,
    RunEventType,
    generatePk,
} from "@vrooli/shared";
import { getRedisConnection } from "../../../../../redisConn.js";
import type { Redis } from "ioredis";
import { getEventBus } from "../../cross-cutting/events/eventBus.js";
import { ResourceManager as Tier1ResourceManager } from "../../tier1/organization/resourceManager.js";
import { ResourceManager as Tier3ResourceManager } from "../../tier3/engine/resourceManager.js";

/**
 * Resource tracking entry
 */
interface ResourceTracking {
    id: string;
    scope: LimitScope;
    scopeId: string;
    resourceType: SharedResourceType;
    allocated: number;
    used: number;
    reserved: number;
    startTime: Date;
    lastUpdate: Date;
    metadata: Record<string, unknown>;
}

/**
 * Resource allocation context
 */
interface AllocationContext {
    userId: string;
    swarmId?: string;
    runId?: string;
    stepId?: string;
    priority: AllocationPriority;
    purpose: string;
    estimatedDuration?: number;
}

/**
 * Resource usage event
 */
interface ResourceUsageEvent {
    id: string;
    timestamp: Date;
    allocationId: string;
    resourceType: SharedResourceType;
    amount: number;
    operation: "allocate" | "use" | "release" | "expire";
    context: AllocationContext;
}

/**
 * Resource pool definition
 */
interface ResourcePoolDefinition {
    id: string;
    type: SharedResourceType;
    capacity: number;
    available: number;
    reserved: number;
    costPerUnit: number;
    replenishRate?: number; // Units per minute
    lastReplenish?: Date;
}

/**
 * Unified Resource Manager implementation
 */
export class UnifiedResourceManager {
    private readonly logger: Logger;
    private redis: Redis | null = null;
    private tier1Manager: Tier1ResourceManager;
    private tier3Manager: Tier3ResourceManager;
    
    // Resource pools by type
    private resourcePools = new Map<SharedResourceType, ResourcePoolDefinition>();
    
    // Active allocations
    private allocations = new Map<string, ResourceTracking>();
    
    // Resource limits by scope
    private resourceLimits = new Map<string, ResourceLimitConfig>();
    
    // Usage tracking
    private usageHistory: ResourceUsageEvent[] = [];
    
    // Configuration
    private readonly TRACKING_PREFIX = "resource:tracking:";
    private readonly LIMIT_PREFIX = "resource:limits:";
    private readonly USAGE_PREFIX = "resource:usage:";
    private readonly POOL_PREFIX = "resource:pools:";
    private readonly CLEANUP_INTERVAL = 60000; // 1 minute
    private readonly USAGE_HISTORY_SIZE = 10000;
    private readonly ALERT_THRESHOLD = 0.8; // 80% usage triggers alert
    
    // Timers
    private cleanupTimer?: NodeJS.Timer;
    private replenishTimer?: NodeJS.Timer;

    constructor(
        logger: Logger,
        tier1Manager: Tier1ResourceManager,
        tier3Manager: Tier3ResourceManager,
    ) {
        this.logger = logger;
        this.tier1Manager = tier1Manager;
        this.tier3Manager = tier3Manager;
        this.initializeResourcePools();
    }

    /**
     * Starts the unified resource manager
     */
    async start(): Promise<void> {
        try {
            // Get Redis connection
            this.redis = await getRedisConnection();
            
            // Load resource configurations
            await this.loadResourceConfigurations();
            
            // Start background tasks
            this.startCleanupTask();
            this.startReplenishmentTask();
            
            // Subscribe to resource events
            await this.subscribeToResourceEvents();
            
            this.logger.info("[UnifiedResourceManager] Started successfully");
        } catch (error) {
            this.logger.error("[UnifiedResourceManager] Failed to start", error);
            throw error;
        }
    }

    /**
     * Stops the resource manager
     */
    async stop(): Promise<void> {
        // Stop timers
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
        }
        if (this.replenishTimer) {
            clearInterval(this.replenishTimer);
        }
        
        // Save current state
        await this.saveResourceState();
        
        this.logger.info("[UnifiedResourceManager] Stopped");
    }

    /**
     * Allocates resources with hierarchical enforcement
     */
    async allocateResources(
        request: SharedResourceAllocation,
        context: AllocationContext,
    ): Promise<AllocationResult> {
        const allocationId = generatePk();
        
        try {
            // Check hierarchical limits
            const limitChecks = await this.checkHierarchicalLimits(request, context);
            if (!limitChecks.allowed) {
                return {
                    allocationId,
                    status: "denied",
                    allocatedResources: [],
                    deniedResources: limitChecks.deniedResources,
                };
            }
            
            // Check resource availability
            const availabilityChecks = await this.checkResourceAvailability(request);
            if (!availabilityChecks.allAvailable) {
                return {
                    allocationId,
                    status: availabilityChecks.someAvailable ? "partial" : "denied",
                    allocatedResources: availabilityChecks.allocatedResources,
                    deniedResources: availabilityChecks.deniedResources,
                };
            }
            
            // Reserve resources
            const allocated = await this.reserveResources(request, context, allocationId);
            
            // Track allocation
            await this.trackAllocation(allocationId, allocated, context);
            
            // Emit allocation event
            await this.emitResourceEvent({
                id: generatePk(),
                timestamp: new Date(),
                allocationId,
                resourceType: request.resources[0]?.resourceId as SharedResourceType,
                amount: allocated.reduce((sum, r) => sum + r.amount, 0),
                operation: "allocate",
                context,
            });
            
            this.logger.info("[UnifiedResourceManager] Resources allocated", {
                allocationId,
                context,
                allocated: allocated.length,
            });
            
            return {
                allocationId,
                status: "allocated",
                allocatedResources: allocated,
                expiresAt: new Date(Date.now() + (request.duration || 3600) * 1000),
                token: this.generateAllocationToken(allocationId),
            };
            
        } catch (error) {
            this.logger.error("[UnifiedResourceManager] Allocation failed", {
                allocationId,
                error,
            });
            
            return {
                allocationId,
                status: "denied",
                allocatedResources: [],
                deniedResources: [{
                    resourceId: request.resources[0]?.resourceId || "unknown",
                    requestedAmount: request.resources[0]?.amount || 0,
                    availableAmount: 0,
                    reason: error instanceof Error ? error.message : "Unknown error",
                }],
            };
        }
    }

    /**
     * Tracks actual resource usage
     */
    async trackUsage(
        allocationId: string,
        usage: SharedResourceUsage,
    ): Promise<void> {
        const tracking = this.allocations.get(allocationId);
        if (!tracking) {
            this.logger.warn("[UnifiedResourceManager] Unknown allocation", { allocationId });
            return;
        }
        
        // Update usage
        tracking.used = usage.consumed;
        tracking.lastUpdate = new Date();
        
        // Check for overage
        if (usage.consumed > tracking.allocated) {
            await this.handleOverage(allocationId, tracking, usage);
        }
        
        // Update tier-specific managers
        if (tracking.metadata.tier === 1) {
            // Update Tier 1 manager
            await this.tier1Manager.getResourceStatus(tracking.scopeId);
        } else if (tracking.metadata.tier === 3) {
            // Update Tier 3 manager
            await this.tier3Manager.trackUsage(tracking.scopeId, {
                cost: usage.cost,
                tokens: usage.consumed,
            });
        }
        
        // Emit usage event
        await this.emitResourceEvent({
            id: generatePk(),
            timestamp: new Date(),
            allocationId,
            resourceType: tracking.resourceType,
            amount: usage.consumed,
            operation: "use",
            context: tracking.metadata.context as AllocationContext,
        });
        
        // Store in Redis for persistence
        await this.persistTracking(tracking);
    }

    /**
     * Releases allocated resources
     */
    async releaseResources(
        allocationId: string,
        token?: string,
    ): Promise<void> {
        const tracking = this.allocations.get(allocationId);
        if (!tracking) {
            return;
        }
        
        // Verify token if provided
        if (token && !this.verifyAllocationToken(allocationId, token)) {
            throw new Error("Invalid allocation token");
        }
        
        // Return unused resources to pool
        const pool = this.resourcePools.get(tracking.resourceType);
        if (pool) {
            const unused = tracking.allocated - tracking.used;
            pool.available += unused;
            pool.reserved -= tracking.reserved;
        }
        
        // Update tier managers
        if (tracking.metadata.tier === 1) {
            await this.tier1Manager.releaseResources(allocationId);
        }
        
        // Emit release event
        await this.emitResourceEvent({
            id: generatePk(),
            timestamp: new Date(),
            allocationId,
            resourceType: tracking.resourceType,
            amount: tracking.allocated,
            operation: "release",
            context: tracking.metadata.context as AllocationContext,
        });
        
        // Remove from active allocations
        this.allocations.delete(allocationId);
        
        // Remove from Redis
        if (this.redis) {
            await this.redis.del(`${this.TRACKING_PREFIX}${allocationId}`);
        }
        
        this.logger.info("[UnifiedResourceManager] Resources released", {
            allocationId,
            released: tracking.allocated - tracking.used,
        });
    }

    /**
     * Gets resource usage report
     */
    async getUsageReport(
        scope: LimitScope,
        scopeId: string,
        period?: { start: Date; end: Date },
    ): Promise<ResourceAccounting> {
        const now = new Date();
        const periodStart = period?.start || new Date(now.getTime() - 24 * 60 * 60 * 1000);
        const periodEnd = period?.end || now;
        
        // Filter allocations by scope
        const scopeAllocations = Array.from(this.allocations.values())
            .filter(a => this.matchesScope(a, scope, scopeId));
        
        // Calculate usage summaries
        const usageSummaries = await this.calculateUsageSummaries(
            scopeAllocations,
            periodStart,
            periodEnd,
        );
        
        // Calculate cost summaries
        const costSummaries = await this.calculateCostSummaries(
            scopeAllocations,
            periodStart,
            periodEnd,
        );
        
        // Calculate efficiency metrics
        const efficiency = await this.calculateEfficiencyMetrics(
            scopeAllocations,
            usageSummaries,
        );
        
        return {
            period: {
                start: periodStart,
                end: periodEnd,
            },
            usage: usageSummaries,
            costs: costSummaries,
            efficiency,
        };
    }

    /**
     * Sets resource limits for a scope
     */
    async setResourceLimits(
        config: ResourceLimitConfig,
    ): Promise<void> {
        const key = `${config.scope}:${config.scopeId}`;
        this.resourceLimits.set(key, config);
        
        // Persist to Redis
        if (this.redis) {
            await this.redis.set(
                `${this.LIMIT_PREFIX}${key}`,
                JSON.stringify(config),
            );
        }
        
        this.logger.info("[UnifiedResourceManager] Resource limits set", {
            scope: config.scope,
            scopeId: config.scopeId,
            limits: config.limits.length,
        });
    }

    /**
     * Gets optimization suggestions
     */
    async getOptimizationSuggestions(
        scope: LimitScope,
        scopeId: string,
    ): Promise<OptimizationSuggestion[]> {
        const suggestions: OptimizationSuggestion[] = [];
        
        // Get current usage
        const report = await this.getUsageReport(scope, scopeId);
        
        // Analyze patterns and generate suggestions
        for (const usage of report.usage) {
            // High utilization - suggest scaling
            if (usage.utilizationRate > 0.9) {
                suggestions.push({
                    id: generatePk(),
                    type: "batch",
                    targetResource: usage.resourceType.toString(),
                    currentUsage: usage.totalConsumed,
                    projectedSavings: usage.totalConsumed * 0.2,
                    implementation: "Batch similar operations to reduce overhead",
                    risk: "low",
                });
            }
            
            // Low utilization - suggest reduction
            if (usage.utilizationRate < 0.3) {
                suggestions.push({
                    id: generatePk(),
                    type: "reduce",
                    targetResource: usage.resourceType.toString(),
                    currentUsage: usage.totalConsumed,
                    projectedSavings: usage.totalConsumed * 0.5,
                    implementation: "Reduce allocation to match actual usage",
                    risk: "low",
                });
            }
        }
        
        // Get tier-specific optimizations
        const tier1Optimizations = await this.tier1Manager.optimizeAllocations();
        for (const opt of tier1Optimizations) {
            suggestions.push({
                id: generatePk(),
                type: opt.type as any,
                targetResource: opt.resourceType,
                currentUsage: opt.amount,
                projectedSavings: opt.expectedSavings,
                implementation: opt.rationale,
                risk: "medium",
            });
        }
        
        return suggestions;
    }

    /**
     * Handles resource conflicts
     */
    async resolveConflict(
        conflict: ResourceConflict,
    ): Promise<ConflictResolution> {
        // Simple priority-based resolution
        const allocations = conflict.requesters
            .map(id => this.allocations.get(id))
            .filter(Boolean) as ResourceTracking[];
        
        if (allocations.length === 0) {
            return {
                type: "queuing",
                reason: "No active allocations found",
            };
        }
        
        // Sort by priority and timestamp
        allocations.sort((a, b) => {
            const aPriority = (a.metadata.priority as AllocationPriority) || "NORMAL";
            const bPriority = (b.metadata.priority as AllocationPriority) || "NORMAL";
            
            if (aPriority !== bPriority) {
                const priorityOrder = ["CRITICAL", "HIGH", "NORMAL", "LOW"];
                return priorityOrder.indexOf(aPriority) - priorityOrder.indexOf(bPriority);
            }
            
            return a.startTime.getTime() - b.startTime.getTime();
        });
        
        const winner = allocations[0];
        
        return {
            type: "priority",
            winner: winner.id,
            reason: `Allocation ${winner.id} has highest priority`,
        };
    }

    /**
     * Private helper methods
     */
    private initializeResourcePools(): void {
        // Initialize default resource pools
        const defaultPools: Array<[SharedResourceType, ResourcePoolDefinition]> = [
            ["CREDITS", {
                id: generatePk(),
                type: "CREDITS",
                capacity: 1000000,
                available: 1000000,
                reserved: 0,
                costPerUnit: 0.0001,
                replenishRate: 1000, // 1000 credits per minute
            }],
            ["TOKENS", {
                id: generatePk(),
                type: "TOKENS",
                capacity: 10000000,
                available: 10000000,
                reserved: 0,
                costPerUnit: 0.00002,
                replenishRate: 10000, // 10k tokens per minute
            }],
            ["API_CALLS", {
                id: generatePk(),
                type: "API_CALLS",
                capacity: 100000,
                available: 100000,
                reserved: 0,
                costPerUnit: 0.001,
                replenishRate: 100, // 100 calls per minute
            }],
            ["TIME", {
                id: generatePk(),
                type: "TIME",
                capacity: 3600000, // 1 hour in ms
                available: 3600000,
                reserved: 0,
                costPerUnit: 0.0001,
            }],
            ["MEMORY", {
                id: generatePk(),
                type: "MEMORY",
                capacity: 8589934592, // 8GB in bytes
                available: 8589934592,
                reserved: 0,
                costPerUnit: 0.00000001,
            }],
        ];
        
        for (const [type, pool] of defaultPools) {
            this.resourcePools.set(type, pool);
        }
    }

    private async loadResourceConfigurations(): Promise<void> {
        if (!this.redis) return;
        
        try {
            // Load resource pools
            const poolKeys = await this.redis.keys(`${this.POOL_PREFIX}*`);
            for (const key of poolKeys) {
                const data = await this.redis.get(key);
                if (data) {
                    const pool = JSON.parse(data) as ResourcePoolDefinition;
                    this.resourcePools.set(pool.type, pool);
                }
            }
            
            // Load resource limits
            const limitKeys = await this.redis.keys(`${this.LIMIT_PREFIX}*`);
            for (const key of limitKeys) {
                const data = await this.redis.get(key);
                if (data) {
                    const config = JSON.parse(data) as ResourceLimitConfig;
                    this.resourceLimits.set(`${config.scope}:${config.scopeId}`, config);
                }
            }
            
            // Load active allocations
            const trackingKeys = await this.redis.keys(`${this.TRACKING_PREFIX}*`);
            for (const key of trackingKeys) {
                const data = await this.redis.get(key);
                if (data) {
                    const tracking = JSON.parse(data) as ResourceTracking;
                    this.allocations.set(tracking.id, tracking);
                }
            }
            
            this.logger.info("[UnifiedResourceManager] Loaded configurations", {
                pools: this.resourcePools.size,
                limits: this.resourceLimits.size,
                allocations: this.allocations.size,
            });
        } catch (error) {
            this.logger.error("[UnifiedResourceManager] Failed to load configurations", error);
        }
    }

    private async checkHierarchicalLimits(
        request: SharedResourceAllocation,
        context: AllocationContext,
    ): Promise<{ allowed: boolean; deniedResources?: DeniedResource[] }> {
        const deniedResources: DeniedResource[] = [];
        
        // Check limits at each level
        const scopes: Array<[LimitScope, string]> = [
            ["USER", context.userId],
        ];
        
        if (context.swarmId) {
            scopes.push(["SWARM", context.swarmId]);
        }
        if (context.runId) {
            scopes.push(["RUN", context.runId]);
        }
        if (context.stepId) {
            scopes.push(["STEP", context.stepId]);
        }
        
        for (const [scope, scopeId] of scopes) {
            const limitConfig = this.resourceLimits.get(`${scope}:${scopeId}`);
            if (!limitConfig) continue;
            
            for (const resourceRequest of request.resources) {
                const limit = limitConfig.limits.find(l => 
                    l.resourceType === resourceRequest.resourceId as any,
                );
                
                if (limit) {
                    // Check current usage against limit
                    const currentUsage = await this.getCurrentUsage(
                        scope,
                        scopeId,
                        limit.resourceType,
                    );
                    
                    if (currentUsage + resourceRequest.amount > limit.limit) {
                        deniedResources.push({
                            resourceId: resourceRequest.resourceId,
                            requestedAmount: resourceRequest.amount,
                            availableAmount: Math.max(0, limit.limit - currentUsage),
                            reason: `Exceeds ${scope} limit`,
                        });
                    }
                }
            }
        }
        
        return {
            allowed: deniedResources.length === 0,
            deniedResources: deniedResources.length > 0 ? deniedResources : undefined,
        };
    }

    private async checkResourceAvailability(
        request: SharedResourceAllocation,
    ): Promise<{
        allAvailable: boolean;
        someAvailable: boolean;
        allocatedResources: AllocatedResource[];
        deniedResources: DeniedResource[];
    }> {
        const allocatedResources: AllocatedResource[] = [];
        const deniedResources: DeniedResource[] = [];
        
        for (const resourceRequest of request.resources) {
            const pool = this.resourcePools.get(resourceRequest.resourceId as SharedResourceType);
            
            if (!pool) {
                deniedResources.push({
                    resourceId: resourceRequest.resourceId,
                    requestedAmount: resourceRequest.amount,
                    availableAmount: 0,
                    reason: "Resource type not found",
                });
                continue;
            }
            
            if (pool.available >= resourceRequest.amount) {
                allocatedResources.push({
                    resourceId: resourceRequest.resourceId,
                    amount: resourceRequest.amount,
                    actualCost: resourceRequest.amount * pool.costPerUnit,
                });
            } else {
                deniedResources.push({
                    resourceId: resourceRequest.resourceId,
                    requestedAmount: resourceRequest.amount,
                    availableAmount: pool.available,
                    reason: "Insufficient resources",
                });
            }
        }
        
        return {
            allAvailable: deniedResources.length === 0,
            someAvailable: allocatedResources.length > 0,
            allocatedResources,
            deniedResources,
        };
    }

    private async reserveResources(
        request: SharedResourceAllocation,
        context: AllocationContext,
        allocationId: string,
    ): Promise<AllocatedResource[]> {
        const allocated: AllocatedResource[] = [];
        
        for (const resourceRequest of request.resources) {
            const pool = this.resourcePools.get(resourceRequest.resourceId as SharedResourceType);
            if (!pool) continue;
            
            // Reserve from pool
            pool.available -= resourceRequest.amount;
            pool.reserved += resourceRequest.amount;
            
            allocated.push({
                resourceId: resourceRequest.resourceId,
                amount: resourceRequest.amount,
                actualCost: resourceRequest.amount * pool.costPerUnit,
                metadata: {
                    allocationId,
                    context,
                },
            });
        }
        
        // Persist pool state
        await this.persistResourcePools();
        
        return allocated;
    }

    private async trackAllocation(
        allocationId: string,
        allocated: AllocatedResource[],
        context: AllocationContext,
    ): Promise<void> {
        for (const resource of allocated) {
            const tracking: ResourceTracking = {
                id: allocationId,
                scope: context.stepId ? "STEP" : context.runId ? "RUN" : context.swarmId ? "SWARM" : "USER",
                scopeId: context.stepId || context.runId || context.swarmId || context.userId,
                resourceType: resource.resourceId as SharedResourceType,
                allocated: resource.amount,
                used: 0,
                reserved: resource.amount,
                startTime: new Date(),
                lastUpdate: new Date(),
                metadata: {
                    context,
                    cost: resource.actualCost,
                    tier: context.stepId ? 3 : context.runId ? 2 : 1,
                },
            };
            
            this.allocations.set(allocationId, tracking);
            await this.persistTracking(tracking);
        }
    }

    private async getCurrentUsage(
        scope: LimitScope,
        scopeId: string,
        resourceType: SharedResourceType,
    ): Promise<number> {
        let totalUsage = 0;
        
        for (const tracking of this.allocations.values()) {
            if (
                this.matchesScope(tracking, scope, scopeId) &&
                tracking.resourceType === resourceType
            ) {
                totalUsage += tracking.used;
            }
        }
        
        return totalUsage;
    }

    private matchesScope(
        tracking: ResourceTracking,
        scope: LimitScope,
        scopeId: string,
    ): boolean {
        const context = tracking.metadata.context as AllocationContext;
        
        switch (scope) {
            case "USER":
                return context.userId === scopeId;
            case "SWARM":
                return context.swarmId === scopeId;
            case "RUN":
                return context.runId === scopeId;
            case "STEP":
                return context.stepId === scopeId;
            case "GLOBAL":
                return true;
            default:
                return false;
        }
    }

    private async handleOverage(
        allocationId: string,
        tracking: ResourceTracking,
        usage: SharedResourceUsage,
    ): Promise<void> {
        const overage = usage.consumed - tracking.allocated;
        
        this.logger.warn("[UnifiedResourceManager] Resource overage detected", {
            allocationId,
            resourceType: tracking.resourceType,
            allocated: tracking.allocated,
            used: usage.consumed,
            overage,
        });
        
        // Check if overage is allowed
        const limitConfig = this.resourceLimits.get(`${tracking.scope}:${tracking.scopeId}`);
        if (limitConfig?.enforcement.overageAllowed) {
            // Apply overage multiplier to cost
            const pool = this.resourcePools.get(tracking.resourceType);
            if (pool) {
                const overageCost = overage * pool.costPerUnit * (limitConfig.enforcement.overageMultiplier || 2);
                tracking.metadata.overageCost = overageCost;
            }
        } else {
            // Emit overage alert
            await this.emitResourceAlert(tracking, "overage", overage);
        }
    }

    private async emitResourceEvent(event: ResourceUsageEvent): Promise<void> {
        this.usageHistory.push(event);
        
        // Trim history if too large
        if (this.usageHistory.length > this.USAGE_HISTORY_SIZE) {
            this.usageHistory = this.usageHistory.slice(-this.USAGE_HISTORY_SIZE);
        }
        
        // Emit to event bus
        const eventBus = getEventBus();
        await eventBus.publish({
            id: event.id,
            type: "RESOURCE_USAGE",
            timestamp: event.timestamp,
            data: event,
            correlationId: event.allocationId,
            metadata: {
                resourceType: event.resourceType,
                operation: event.operation,
            },
        });
    }

    private async emitResourceAlert(
        tracking: ResourceTracking,
        alertType: string,
        value: number,
    ): Promise<void> {
        const eventBus = getEventBus();
        await eventBus.publish({
            id: generatePk(),
            type: "RESOURCE_ALERT",
            timestamp: new Date(),
            data: {
                alertType,
                allocationId: tracking.id,
                resourceType: tracking.resourceType,
                scope: tracking.scope,
                scopeId: tracking.scopeId,
                value,
                threshold: tracking.allocated * this.ALERT_THRESHOLD,
            },
            correlationId: tracking.id,
            metadata: {
                severity: alertType === "overage" ? "high" : "medium",
            },
        });
    }

    private generateAllocationToken(allocationId: string): string {
        // Simple token generation - in production, use proper JWT or similar
        return Buffer.from(`${allocationId}:${Date.now()}`).toString("base64");
    }

    private verifyAllocationToken(allocationId: string, token: string): boolean {
        try {
            const decoded = Buffer.from(token, "base64").toString();
            return decoded.startsWith(`${allocationId}:`);
        } catch {
            return false;
        }
    }

    private async persistTracking(tracking: ResourceTracking): Promise<void> {
        if (!this.redis) return;
        
        await this.redis.set(
            `${this.TRACKING_PREFIX}${tracking.id}`,
            JSON.stringify(tracking),
            "EX",
            86400, // 24 hour TTL
        );
    }

    private async persistResourcePools(): Promise<void> {
        if (!this.redis) return;
        
        const pipeline = this.redis.pipeline();
        
        for (const [type, pool] of this.resourcePools) {
            pipeline.set(
                `${this.POOL_PREFIX}${type}`,
                JSON.stringify(pool),
            );
        }
        
        await pipeline.exec();
    }

    private async saveResourceState(): Promise<void> {
        await this.persistResourcePools();
        
        // Save all active allocations
        for (const tracking of this.allocations.values()) {
            await this.persistTracking(tracking);
        }
    }

    private startCleanupTask(): void {
        this.cleanupTimer = setInterval(async () => {
            try {
                await this.cleanupExpiredAllocations();
            } catch (error) {
                this.logger.error("[UnifiedResourceManager] Cleanup error", error);
            }
        }, this.CLEANUP_INTERVAL);
    }

    private startReplenishmentTask(): void {
        this.replenishTimer = setInterval(async () => {
            try {
                await this.replenishResources();
            } catch (error) {
                this.logger.error("[UnifiedResourceManager] Replenishment error", error);
            }
        }, 60000); // Every minute
    }

    private async cleanupExpiredAllocations(): Promise<void> {
        const now = Date.now();
        const expired: string[] = [];
        
        for (const [id, tracking] of this.allocations) {
            const duration = now - tracking.startTime.getTime();
            const maxDuration = (tracking.metadata.estimatedDuration as number) || 3600000;
            
            if (duration > maxDuration * 1.5) {
                expired.push(id);
            }
        }
        
        for (const id of expired) {
            await this.releaseResources(id);
        }
        
        if (expired.length > 0) {
            this.logger.info("[UnifiedResourceManager] Cleaned up expired allocations", {
                count: expired.length,
            });
        }
    }

    private async replenishResources(): Promise<void> {
        const now = new Date();
        
        for (const pool of this.resourcePools.values()) {
            if (!pool.replenishRate) continue;
            
            const lastReplenish = pool.lastReplenish || now;
            const timeSinceReplenish = now.getTime() - lastReplenish.getTime();
            const minutesSinceReplenish = timeSinceReplenish / 60000;
            
            if (minutesSinceReplenish >= 1) {
                const replenishAmount = Math.floor(pool.replenishRate * minutesSinceReplenish);
                const newAvailable = Math.min(pool.capacity, pool.available + replenishAmount);
                
                if (newAvailable > pool.available) {
                    pool.available = newAvailable;
                    pool.lastReplenish = now;
                    
                    this.logger.debug("[UnifiedResourceManager] Replenished resources", {
                        type: pool.type,
                        amount: newAvailable - pool.available,
                        newAvailable,
                    });
                }
            }
        }
        
        await this.persistResourcePools();
    }

    private async subscribeToResourceEvents(): Promise<void> {
        const eventBus = getEventBus();
        
        // Subscribe to swarm resource events
        await eventBus.subscribe({
            id: generatePk(),
            eventType: SwarmEventType.SWARM_RESOURCES_ALLOCATED,
            handler: "UnifiedResourceManager.handleSwarmResourceEvent",
            filters: [],
            config: {
                maxRetries: 3,
                retryDelay: 1000,
            },
        });
        
        // Subscribe to run resource events
        await eventBus.subscribe({
            id: generatePk(),
            eventType: RunEventType.RUN_STARTED,
            handler: "UnifiedResourceManager.handleRunResourceEvent",
            filters: [],
            config: {
                maxRetries: 3,
                retryDelay: 1000,
            },
        });
    }

    private async calculateUsageSummaries(
        allocations: ResourceTracking[],
        start: Date,
        end: Date,
    ): Promise<ResourceUsageSummary[]> {
        const summaries = new Map<SharedResourceType, ResourceUsageSummary>();
        
        for (const allocation of allocations) {
            if (allocation.startTime < end && allocation.lastUpdate > start) {
                const existing = summaries.get(allocation.resourceType) || {
                    resourceType: allocation.resourceType,
                    totalConsumed: 0,
                    peakUsage: 0,
                    averageUsage: 0,
                    utilizationRate: 0,
                };
                
                existing.totalConsumed += allocation.used;
                existing.peakUsage = Math.max(existing.peakUsage, allocation.used);
                
                summaries.set(allocation.resourceType, existing);
            }
        }
        
        // Calculate averages and utilization
        for (const summary of summaries.values()) {
            const pool = this.resourcePools.get(summary.resourceType);
            if (pool) {
                summary.utilizationRate = summary.totalConsumed / pool.capacity;
            }
        }
        
        return Array.from(summaries.values());
    }

    private async calculateCostSummaries(
        allocations: ResourceTracking[],
        start: Date,
        end: Date,
    ): Promise<ResourceCostSummary[]> {
        const summaries = new Map<SharedResourceType, ResourceCostSummary>();
        
        for (const allocation of allocations) {
            if (allocation.startTime < end && allocation.lastUpdate > start) {
                const pool = this.resourcePools.get(allocation.resourceType);
                if (!pool) continue;
                
                const cost = allocation.used * pool.costPerUnit;
                const overageCost = (allocation.metadata.overageCost as number) || 0;
                
                const existing = summaries.get(allocation.resourceType) || {
                    resourceType: allocation.resourceType,
                    totalCost: 0,
                    breakdown: [],
                };
                
                existing.totalCost += cost + overageCost;
                
                summaries.set(allocation.resourceType, existing);
            }
        }
        
        return Array.from(summaries.values());
    }

    private async calculateEfficiencyMetrics(
        allocations: ResourceTracking[],
        usageSummaries: ResourceUsageSummary[],
    ): Promise<EfficiencyMetrics> {
        let totalAllocated = 0;
        let totalUsed = 0;
        
        for (const allocation of allocations) {
            totalAllocated += allocation.allocated;
            totalUsed += allocation.used;
        }
        
        const overallEfficiency = totalAllocated > 0 ? totalUsed / totalAllocated : 0;
        const wastedResources = totalAllocated > 0 ? (totalAllocated - totalUsed) / totalAllocated : 0;
        
        // Get optimization suggestions
        const suggestions = await this.getOptimizationSuggestions("GLOBAL", "system");
        
        return {
            overallEfficiency,
            wastedResources: wastedResources * 100,
            optimizationPotential: suggestions.reduce((sum, s) => sum + s.projectedSavings, 0),
            recommendations: suggestions.slice(0, 5), // Top 5 suggestions
        };
    }
}