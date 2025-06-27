import { generatePK } from "@vrooli/shared";
import { type Logger } from "winston";
import { type EventBus } from "../events/eventBus.js";

/**
 * Resource pool configuration
 */
export interface ResourcePoolConfig {
    id: string;
    name: string;
    capacity: number;
    refillRate?: number; // per second
    refillAmount?: number;
    maxBurst?: number;
    metadata?: Record<string, unknown>;
}

/**
 * Pool allocation
 */
export interface PoolAllocation {
    id: string;
    poolId: string;
    consumerId: string;
    amount: number;
    timestamp: Date;
    expiresAt?: Date;
    metadata?: Record<string, unknown>;
}

/**
 * Pool state
 */
export interface PoolState {
    available: number;
    allocated: number;
    reserved: number;
    capacity: number;
    lastRefill: Date;
    allocations: Map<string, PoolAllocation>;
}

/**
 * Allocation result
 */
export interface PoolAllocationResult {
    success: boolean;
    allocationId?: string;
    amount?: number;
    reason?: string;
}

/**
 * Shared resource pool management
 * 
 * Manages finite resource pools with:
 * - Capacity management
 * - Fair allocation
 * - Automatic refilling
 * - Expiration handling
 * - Event notifications
 */
export class ResourcePool {
    private readonly logger: Logger;
    private readonly eventBus?: EventBus;
    private readonly config: ResourcePoolConfig;
    private state: PoolState;
    private refillTimer?: NodeJS.Timeout;

    constructor(
        config: ResourcePoolConfig,
        logger: Logger,
        eventBus?: EventBus,
    ) {
        this.config = config;
        this.logger = logger;
        this.eventBus = eventBus;

        this.state = {
            available: config.capacity,
            allocated: 0,
            reserved: 0,
            capacity: config.capacity,
            lastRefill: new Date(),
            allocations: new Map(),
        };

        // Start refill timer if configured
        if (config.refillRate && config.refillAmount) {
            this.startRefillTimer();
        }

        this.logger.debug("[ResourcePool] Initialized", {
            poolId: config.id,
            capacity: config.capacity,
            refillRate: config.refillRate,
        });
    }

    /**
     * Allocate resources from pool
     */
    async allocate(
        consumerId: string,
        amount: number,
        duration?: number, // milliseconds
        metadata?: Record<string, unknown>,
    ): Promise<PoolAllocationResult> {
        // Clean up expired allocations first
        this.cleanupExpiredAllocations();

        // Check availability
        if (amount > this.state.available) {
            const reason = `Insufficient resources: requested ${amount}, available ${this.state.available}`;

            if (this.eventBus) {
                await this.eventBus.emit({
                    type: "resource_pool.allocation_failed",
                    timestamp: new Date(),
                    data: {
                        poolId: this.config.id,
                        consumerId,
                        requested: amount,
                        available: this.state.available,
                    },
                });
            }

            return {
                success: false,
                reason,
            };
        }

        // Create allocation
        const allocationId = generatePK();
        const allocation: PoolAllocation = {
            id: allocationId,
            poolId: this.config.id,
            consumerId,
            amount,
            timestamp: new Date(),
            expiresAt: duration ? new Date(Date.now() + duration) : undefined,
            metadata,
        };

        // Update state
        this.state.allocations.set(allocationId, allocation);
        this.state.available -= amount;
        this.state.allocated += amount;

        // Emit allocation event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: "resource_pool.allocated",
                timestamp: new Date(),
                data: {
                    poolId: this.config.id,
                    allocationId,
                    consumerId,
                    amount,
                    available: this.state.available,
                },
            });
        }

        this.logger.debug("[ResourcePool] Allocated resources", {
            poolId: this.config.id,
            allocationId,
            consumerId,
            amount,
            available: this.state.available,
        });

        // Schedule automatic release if duration specified
        if (duration) {
            setTimeout(() => {
                this.release(allocationId).catch(err => {
                    this.logger.error("[ResourcePool] Auto-release failed", {
                        allocationId,
                        error: err,
                    });
                });
            }, duration);
        }

        return {
            success: true,
            allocationId,
            amount,
        };
    }

    /**
     * Release allocated resources
     */
    async release(allocationId: string): Promise<boolean> {
        const allocation = this.state.allocations.get(allocationId);
        if (!allocation) {
            return false;
        }

        // Update state
        this.state.allocations.delete(allocationId);
        this.state.available += allocation.amount;
        this.state.allocated -= allocation.amount;

        // Emit release event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: "resource_pool.released",
                timestamp: new Date(),
                data: {
                    poolId: this.config.id,
                    allocationId,
                    consumerId: allocation.consumerId,
                    amount: allocation.amount,
                    available: this.state.available,
                },
            });
        }

        this.logger.debug("[ResourcePool] Released resources", {
            poolId: this.config.id,
            allocationId,
            amount: allocation.amount,
            available: this.state.available,
        });

        return true;
    }

    /**
     * Reserve resources (not yet allocated)
     */
    async reserve(amount: number): Promise<boolean> {
        if (amount > this.state.available) {
            return false;
        }

        this.state.available -= amount;
        this.state.reserved += amount;

        return true;
    }

    /**
     * Confirm reservation (convert to allocation)
     */
    async confirmReservation(
        consumerId: string,
        amount: number,
        metadata?: Record<string, unknown>,
    ): Promise<PoolAllocationResult> {
        if (amount > this.state.reserved) {
            return {
                success: false,
                reason: "Invalid reservation amount",
            };
        }

        this.state.reserved -= amount;

        // Allocate without checking availability (already reserved)
        const allocationId = generatePK();
        const allocation: PoolAllocation = {
            id: allocationId,
            poolId: this.config.id,
            consumerId,
            amount,
            timestamp: new Date(),
            metadata,
        };

        this.state.allocations.set(allocationId, allocation);
        this.state.allocated += amount;

        return {
            success: true,
            allocationId,
            amount,
        };
    }

    /**
     * Cancel reservation
     */
    async cancelReservation(amount: number): Promise<void> {
        const actualAmount = Math.min(amount, this.state.reserved);
        this.state.reserved -= actualAmount;
        this.state.available += actualAmount;
    }

    /**
     * Get current pool state
     */
    getState(): PoolState {
        this.cleanupExpiredAllocations();
        return {
            ...this.state,
            allocations: new Map(this.state.allocations),
        };
    }

    /**
     * Get allocations for a consumer
     */
    getConsumerAllocations(consumerId: string): PoolAllocation[] {
        const allocations: PoolAllocation[] = [];
        for (const allocation of this.state.allocations.values()) {
            if (allocation.consumerId === consumerId) {
                allocations.push(allocation);
            }
        }
        return allocations;
    }

    /**
     * Clean up expired allocations
     */
    private cleanupExpiredAllocations(): void {
        const now = new Date();
        const expired: string[] = [];

        for (const [id, allocation] of this.state.allocations) {
            if (allocation.expiresAt && allocation.expiresAt <= now) {
                expired.push(id);
            }
        }

        for (const id of expired) {
            this.release(id).catch(err => {
                this.logger.error("[ResourcePool] Cleanup failed", {
                    allocationId: id,
                    error: err,
                });
            });
        }
    }

    /**
     * Start automatic refill timer
     */
    private startRefillTimer(): void {
        if (!this.config.refillRate || !this.config.refillAmount) {
            return;
        }

        const intervalMs = 1000 / this.config.refillRate;

        this.refillTimer = setInterval(() => {
            this.refill();
        }, intervalMs);
    }

    /**
     * Refill pool resources
     */
    private refill(): void {
        if (!this.config.refillAmount) {
            return;
        }

        const maxCapacity = this.config.maxBurst || this.config.capacity;
        const currentTotal = this.state.available + this.state.allocated + this.state.reserved;

        if (currentTotal >= maxCapacity) {
            return; // Already at max capacity
        }

        const refillAmount = Math.min(
            this.config.refillAmount,
            maxCapacity - currentTotal,
        );

        this.state.available += refillAmount;
        this.state.lastRefill = new Date();

        this.logger.debug("[ResourcePool] Refilled pool", {
            poolId: this.config.id,
            refillAmount,
            available: this.state.available,
        });
    }

    /**
     * Shutdown pool
     */
    shutdown(): void {
        if (this.refillTimer) {
            clearInterval(this.refillTimer);
            this.refillTimer = undefined;
        }

        this.logger.info("[ResourcePool] Shutdown", {
            poolId: this.config.id,
            allocated: this.state.allocated,
        });
    }
}

/**
 * Resource pool manager
 * 
 * Manages multiple resource pools
 */
export class ResourcePoolManager {
    private readonly logger: Logger;
    private readonly eventBus?: EventBus;
    private readonly pools: Map<string, ResourcePool> = new Map();

    constructor(logger: Logger, eventBus?: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
    }

    /**
     * Create a new pool
     */
    createPool(config: ResourcePoolConfig): ResourcePool {
        if (this.pools.has(config.id)) {
            throw new Error(`Pool ${config.id} already exists`);
        }

        const pool = new ResourcePool(config, this.logger, this.eventBus);
        this.pools.set(config.id, pool);

        return pool;
    }

    /**
     * Get a pool by ID
     */
    getPool(poolId: string): ResourcePool | undefined {
        return this.pools.get(poolId);
    }

    /**
     * Remove a pool
     */
    removePool(poolId: string): boolean {
        const pool = this.pools.get(poolId);
        if (pool) {
            pool.shutdown();
            this.pools.delete(poolId);
            return true;
        }
        return false;
    }

    /**
     * Get all pools
     */
    getAllPools(): Map<string, ResourcePool> {
        return new Map(this.pools);
    }

    /**
     * Shutdown all pools
     */
    shutdown(): void {
        for (const pool of this.pools.values()) {
            pool.shutdown();
        }
        this.pools.clear();
    }
}
