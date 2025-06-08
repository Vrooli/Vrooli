import { type Logger } from "winston";
import {
    type ResourceAllocation,
    type ResourceType,
    type ResourceReservation,
    type ResourceReturn,
    SwarmEventType as SwarmEventTypeEnum,
    generatePk,
} from "@vrooli/shared";

/**
 * Resource pool configuration
 */
export interface ResourcePool {
    type: ResourceType;
    total: number;
    available: number;
    reserved: number;
    unit: string;
}

/**
 * Resource request
 */
export interface ResourceRequest {
    swarmId: string;
    agentId?: string;
    type: ResourceType;
    amount: number;
    purpose: string;
    priority?: "low" | "medium" | "high";
    duration?: number; // milliseconds
}

/**
 * Resource status
 */
export interface ResourceStatus {
    pools: ResourcePool[];
    allocations: ResourceAllocation[];
    utilizationRate: number;
    projectedExhaustion?: Date;
}

/**
 * Allocation strategy
 */
export type AllocationStrategy = "fair" | "priority" | "performance" | "adaptive";

/**
 * Resource optimization suggestion
 */
export interface ResourceOptimization {
    type: "rebalance" | "release" | "increase" | "throttle";
    target: string; // swarmId or agentId
    resourceType: ResourceType;
    amount: number;
    rationale: string;
    expectedSavings: number;
}

/**
 * ResourceManager - Hierarchical resource allocation and optimization
 * 
 * This component manages resource allocation across the swarm using
 * economic models and optimization algorithms. It provides:
 * 
 * - Hierarchical budget allocation (swarm -> team -> agent)
 * - Multiple resource types (credits, time, memory, etc.)
 * - Dynamic rebalancing based on utilization
 * - Predictive exhaustion warnings
 * - Optimization recommendations
 * 
 * The ResourceManager ensures efficient resource utilization while
 * preventing exhaustion and maintaining fair allocation.
 */
export class ResourceManager {
    private readonly logger: Logger;
    private readonly resourcePools: Map<ResourceType, ResourcePool> = new Map();
    private readonly allocations: Map<string, ResourceAllocation[]> = new Map();
    private readonly allocationStrategy: AllocationStrategy = "adaptive";
    private readonly optimizationInterval: number = 60000; // 1 minute
    private optimizationTimer?: NodeJS.Timer;
    
    // Configuration constants
    private readonly DEFAULT_TIME_LIMIT_MS = 3600000; // 1 hour
    private readonly EFFICIENCY_THRESHOLD = 0.2;
    private readonly ALLOCATION_RATIO = 0.3;
    private readonly HIGH_UTILIZATION_THRESHOLD = 0.8;
    private readonly UTILIZATION_PENALTY = 0.7;

    constructor(logger: Logger) {
        this.logger = logger;
        this.initializeResourcePools();
        this.startOptimizationLoop();
    }

    /**
     * Initializes resource pools with default values
     */
    private initializeResourcePools(): void {
        const defaultPools: ResourcePool[] = [
            {
                type: "credits",
                total: 10000,
                available: 10000,
                reserved: 0,
                unit: "credits",
            },
            {
                type: "time",
                total: 3600000, // 1 hour in ms
                available: 3600000,
                reserved: 0,
                unit: "milliseconds",
            },
            {
                type: "memory",
                total: 1073741824, // 1GB in bytes
                available: 1073741824,
                reserved: 0,
                unit: "bytes",
            },
            {
                type: "tokens",
                total: 1000000,
                available: 1000000,
                reserved: 0,
                unit: "tokens",
            },
            {
                type: "api_calls",
                total: 10000,
                available: 10000,
                reserved: 0,
                unit: "calls",
            },
        ];

        for (const pool of defaultPools) {
            this.resourcePools.set(pool.type, pool);
        }

        this.logger.info("[ResourceManager] Initialized resource pools", {
            poolCount: this.resourcePools.size,
            totalCredits: defaultPools[0].total,
        });
    }

    /**
     * Allocates resources for a swarm or agent
     */
    async allocateResources(
        swarmId: string,
        purpose: string,
        amount: number,
        type: ResourceType = "credits",
        agentId?: string,
    ): Promise<ResourceAllocation> {
        const pool = this.resourcePools.get(type);
        if (!pool) {
            throw new Error(`Unknown resource type: ${type}`);
        }

        if (pool.available < amount) {
            throw new Error(
                `Insufficient ${type}: requested ${amount}, available ${pool.available}`,
            );
        }

        this.logger.info("[ResourceManager] Allocating resources", {
            swarmId,
            agentId,
            type,
            amount,
            purpose,
        });

        // Create allocation
        const allocation: ResourceAllocation = {
            id: generatePk(),
            swarmId,
            agentId: agentId || swarmId,
            resourceType: type,
            amount,
            purpose,
            expiresAt: new Date(Date.now() + 3600000), // 1 hour default
        };

        // Update pool
        pool.available -= amount;
        pool.reserved += amount;

        // Store allocation
        const key = agentId || swarmId;
        if (!this.allocations.has(key)) {
            this.allocations.set(key, []);
        }
        this.allocations.get(key)!.push(allocation);

        // Check for low resources
        if (pool.available < pool.total * 0.2) {
            this.logger.warn("[ResourceManager] Low resource warning", {
                type,
                available: pool.available,
                threshold: pool.total * 0.2,
            });
        }

        return allocation;
    }

    /**
     * Releases allocated resources
     */
    async releaseResources(
        allocationId: string,
    ): Promise<void> {
        let found = false;
        
        // Find and remove allocation
        for (const [key, allocations] of this.allocations) {
            const index = allocations.findIndex(a => a.id === allocationId);
            if (index !== -1) {
                const allocation = allocations[index];
                const pool = this.resourcePools.get(allocation.resourceType);
                
                if (pool) {
                    pool.available += allocation.amount;
                    pool.reserved -= allocation.amount;
                }
                
                allocations.splice(index, 1);
                found = true;
                
                this.logger.info("[ResourceManager] Released resources", {
                    allocationId,
                    type: allocation.resourceType,
                    amount: allocation.amount,
                });
                
                break;
            }
        }

        if (!found) {
            this.logger.warn("[ResourceManager] Allocation not found", { allocationId });
        }
    }

    /**
     * Gets resource status for a swarm
     */
    async getResourceStatus(swarmId: string): Promise<ResourceStatus> {
        const swarmAllocations: ResourceAllocation[] = [];
        
        // Collect allocations for this swarm
        for (const [key, allocations] of this.allocations) {
            if (key === swarmId || key.startsWith(`${swarmId}-`)) {
                swarmAllocations.push(...allocations.filter(a => a.swarmId === swarmId));
            }
        }

        // Calculate utilization
        let totalUsed = 0;
        let totalAvailable = 0;
        
        for (const pool of this.resourcePools.values()) {
            totalUsed += pool.reserved;
            totalAvailable += pool.total;
        }
        
        const utilizationRate = totalAvailable > 0 ? totalUsed / totalAvailable : 0;

        // Predict exhaustion
        const projectedExhaustion = this.predictExhaustion();

        return {
            pools: Array.from(this.resourcePools.values()),
            allocations: swarmAllocations,
            utilizationRate,
            projectedExhaustion,
        };
    }

    /**
     * Requests resources with priority handling
     */
    async requestResources(
        request: ResourceRequest,
    ): Promise<ResourceAllocation> {
        // Apply allocation strategy
        const adjustedAmount = await this.applyAllocationStrategy(request);
        
        // Allocate resources
        return this.allocateResources(
            request.swarmId,
            request.purpose,
            adjustedAmount,
            request.type,
            request.agentId,
        );
    }

    /**
     * Optimizes resource allocation
     */
    async optimizeAllocations(): Promise<ResourceOptimization[]> {
        const optimizations: ResourceOptimization[] = [];
        
        // Analyze current allocations
        const utilizationByEntity = new Map<string, Map<ResourceType, number>>();
        
        for (const [entity, allocations] of this.allocations) {
            const usage = new Map<ResourceType, number>();
            
            for (const allocation of allocations) {
                const current = usage.get(allocation.resourceType) || 0;
                usage.set(allocation.resourceType, current + allocation.amount);
            }
            
            utilizationByEntity.set(entity, usage);
        }

        // Find optimization opportunities
        for (const [entity, usage] of utilizationByEntity) {
            for (const [type, amount] of usage) {
                const pool = this.resourcePools.get(type);
                if (!pool) continue;
                
                // Check for over-allocation
                if (amount > pool.total * this.ALLOCATION_RATIO) { // Using more than 30% of total
                    optimizations.push({
                        type: "throttle",
                        target: entity,
                        resourceType: type,
                        amount: amount * 0.2, // Reduce by 20%
                        rationale: "High resource consumption detected",
                        expectedSavings: amount * 0.2,
                    });
                }
            }
        }

        // Check for idle allocations
        const now = Date.now();
        for (const [entity, allocations] of this.allocations) {
            for (const allocation of allocations) {
                if (allocation.expiresAt && allocation.expiresAt.getTime() < now) {
                    optimizations.push({
                        type: "release",
                        target: entity,
                        resourceType: allocation.resourceType,
                        amount: allocation.amount,
                        rationale: "Expired allocation",
                        expectedSavings: allocation.amount,
                    });
                }
            }
        }

        return optimizations;
    }

    /**
     * Updates resource pool
     */
    async updateResourcePool(
        type: ResourceType,
        updates: Partial<ResourcePool>,
    ): Promise<void> {
        const pool = this.resourcePools.get(type);
        if (!pool) {
            throw new Error(`Unknown resource type: ${type}`);
        }

        Object.assign(pool, updates);
        
        // Recalculate available if total changed
        if (updates.total !== undefined) {
            pool.available = pool.total - pool.reserved;
        }

        this.logger.info("[ResourceManager] Updated resource pool", {
            type,
            updates,
            newTotal: pool.total,
            newAvailable: pool.available,
        });
    }

    /**
     * Sets allocation strategy
     */
    setAllocationStrategy(strategy: AllocationStrategy): void {
        this.logger.info("[ResourceManager] Changed allocation strategy", {
            from: this.allocationStrategy,
            to: strategy,
        });
        (this as any).allocationStrategy = strategy;
    }

    /**
     * Private helper methods
     */
    private async applyAllocationStrategy(
        request: ResourceRequest,
    ): Promise<number> {
        switch (this.allocationStrategy) {
            case "fair":
                // Equal allocation
                return Math.min(request.amount, this.getFairShare(request.type));
                
            case "priority":
                // Adjust based on priority
                const multiplier = {
                    high: 1.0,
                    medium: 0.8,
                    low: 0.6,
                }[request.priority || "medium"];
                return Math.min(request.amount * multiplier, request.amount);
                
            case "performance":
                // Adjust based on past performance
                // For now, just return requested amount
                return request.amount;
                
            case "adaptive":
                // Dynamic adjustment based on current state
                const pool = this.resourcePools.get(request.type);
                if (!pool) return request.amount;
                
                const utilizationRate = pool.reserved / pool.total;
                if (utilizationRate > this.HIGH_UTILIZATION_THRESHOLD) {
                    // High utilization - reduce allocation
                    return request.amount * this.UTILIZATION_PENALTY;
                } else if (utilizationRate < this.ALLOCATION_RATIO) {
                    // Low utilization - allow full allocation
                    return request.amount;
                } else {
                    // Medium utilization - slight reduction
                    return request.amount * 0.9;
                }
                
            default:
                return request.amount;
        }
    }

    private getFairShare(type: ResourceType): number {
        const pool = this.resourcePools.get(type);
        if (!pool) return 0;
        
        // Estimate number of active entities
        const activeEntities = this.allocations.size || 1;
        
        // Fair share is available resources divided by active entities
        return Math.floor(pool.available / activeEntities);
    }

    private predictExhaustion(): Date | undefined {
        // Simple linear prediction based on current consumption rate
        // In production, this would use more sophisticated forecasting
        
        let earliestExhaustion: Date | undefined;
        
        for (const [type, pool] of this.resourcePools) {
            if (pool.available === 0) {
                return new Date(); // Already exhausted
            }
            
            // Calculate burn rate (simplified)
            const burnRate = pool.reserved / 3600000; // per millisecond
            if (burnRate > 0) {
                const timeToExhaustion = pool.available / burnRate;
                const exhaustionDate = new Date(Date.now() + timeToExhaustion);
                
                if (!earliestExhaustion || exhaustionDate < earliestExhaustion) {
                    earliestExhaustion = exhaustionDate;
                }
            }
        }
        
        return earliestExhaustion;
    }

    private startOptimizationLoop(): void {
        this.optimizationTimer = setInterval(async () => {
            try {
                await this.performOptimization();
            } catch (error) {
                this.logger.error("[ResourceManager] Optimization error", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }, this.optimizationInterval);
    }

    private async performOptimization(): Promise<void> {
        // Clean up expired allocations
        const now = Date.now();
        let cleanedCount = 0;
        
        for (const [entity, allocations] of this.allocations) {
            const activeAllocations = allocations.filter(a => {
                if (a.expiresAt && a.expiresAt.getTime() < now) {
                    // Release expired allocation
                    const pool = this.resourcePools.get(a.resourceType);
                    if (pool) {
                        pool.available += a.amount;
                        pool.reserved -= a.amount;
                    }
                    cleanedCount++;
                    return false;
                }
                return true;
            });
            
            if (activeAllocations.length === 0) {
                this.allocations.delete(entity);
            } else {
                this.allocations.set(entity, activeAllocations);
            }
        }
        
        if (cleanedCount > 0) {
            this.logger.info("[ResourceManager] Cleaned expired allocations", {
                count: cleanedCount,
            });
        }

        // Generate optimization suggestions
        const optimizations = await this.optimizeAllocations();
        if (optimizations.length > 0) {
            this.logger.info("[ResourceManager] Generated optimizations", {
                count: optimizations.length,
                types: optimizations.map(o => o.type),
            });
        }
    }

    /**
     * Stops the resource manager
     */
    async stop(): Promise<void> {
        if (this.optimizationTimer) {
            clearInterval(this.optimizationTimer);
            this.optimizationTimer = undefined;
        }
        
        this.logger.info("[ResourceManager] Stopped");
    }

    /**
     * Gets allocation summary
     */
    async getAllocationSummary(): Promise<{
        totalAllocations: number;
        byType: Record<ResourceType, number>;
        byEntity: Record<string, number>;
    }> {
        const byType: Record<string, number> = {};
        const byEntity: Record<string, number> = {};
        let totalAllocations = 0;
        
        for (const [entity, allocations] of this.allocations) {
            byEntity[entity] = allocations.length;
            totalAllocations += allocations.length;
            
            for (const allocation of allocations) {
                byType[allocation.resourceType] = (byType[allocation.resourceType] || 0) + 1;
            }
        }
        
        return {
            totalAllocations,
            byType: byType as Record<ResourceType, number>,
            byEntity,
        };
    }
}
