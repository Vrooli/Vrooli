/**
 * Base Resource Manager
 * 
 * Common resource tracking and management functionality shared across all tiers.
 * Provides hierarchical resource allocation, tracking, and enforcement.
 * 
 * This consolidates the duplicate resource management logic while maintaining
 * tier-specific flexibility through delegation.
 */

import { type Logger } from "winston";
import { type ResourceUsage } from "@vrooli/shared";

/**
 * Resource limits configuration
 */
export interface ResourceLimits {
    maxCredits: number;
    maxTokens: number;
    maxTime: number; // in milliseconds
}

/**
 * Resource allocation record
 */
export interface ResourceAllocation {
    id: string;
    parentId?: string; // For hierarchical allocation
    limits: ResourceLimits;
    consumed: ResourceUsage;
    reserved: ResourceUsage; // Reserved for children
    createdAt: Date;
    updatedAt: Date;
    metadata?: Record<string, unknown>;
}

/**
 * Resource allocation request
 */
export interface ResourceRequest {
    credits: number;
    tokens: number;
    duration: number; // in milliseconds
}

/**
 * Resource allocation result
 */
export interface AllocationResult {
    success: boolean;
    allocation?: ResourceAllocation;
    reason?: string;
}

/**
 * Abstract base class for resource managers
 */
export abstract class BaseResourceManager {
    protected readonly allocations: Map<string, ResourceAllocation> = new Map();
    
    constructor(protected readonly logger: Logger) {}

    /**
     * Create a new resource allocation
     */
    async createAllocation(
        id: string,
        limits: ResourceLimits,
        parentId?: string,
    ): Promise<AllocationResult> {
        // Check if allocation already exists
        if (this.allocations.has(id)) {
            return {
                success: false,
                reason: `Allocation ${id} already exists`,
            };
        }

        // If parent specified, validate and reserve from parent
        if (parentId) {
            const parent = this.allocations.get(parentId);
            if (!parent) {
                return {
                    success: false,
                    reason: `Parent allocation ${parentId} not found`,
                };
            }

            const reserveResult = this.reserveFromParent(parent, limits);
            if (!reserveResult.success) {
                return reserveResult;
            }
        }

        // Create allocation
        const allocation: ResourceAllocation = {
            id,
            parentId,
            limits,
            consumed: {
                credits: 0,
                tokens: 0,
                time: 0,
            },
            reserved: {
                credits: 0,
                tokens: 0,
                time: 0,
            },
            createdAt: new Date(),
            updatedAt: new Date(),
        };

        this.allocations.set(id, allocation);
        
        this.logger.info(`[${this.constructor.name}] Created allocation`, {
            id,
            parentId,
            limits,
        });

        return {
            success: true,
            allocation,
        };
    }

    /**
     * Track resource consumption
     */
    async trackUsage(
        id: string,
        usage: ResourceUsage,
    ): Promise<boolean> {
        const allocation = this.allocations.get(id);
        if (!allocation) {
            this.logger.error(`[${this.constructor.name}] Allocation not found: ${id}`);
            return false;
        }

        // Update consumed resources
        allocation.consumed.credits += usage.credits;
        allocation.consumed.tokens += usage.tokens;
        allocation.consumed.time += usage.time;
        allocation.updatedAt = new Date();

        // Check if limits exceeded
        const exceeded = this.checkLimitsExceeded(allocation);
        if (exceeded.length > 0) {
            this.logger.warn(`[${this.constructor.name}] Resource limits exceeded`, {
                id,
                exceeded,
                consumed: allocation.consumed,
                limits: allocation.limits,
            });
            
            // Let subclass handle limit exceeded
            await this.onLimitsExceeded(id, allocation, exceeded);
        }

        // Update parent consumption if hierarchical
        if (allocation.parentId) {
            await this.trackUsage(allocation.parentId, usage);
        }

        return true;
    }

    /**
     * Check if allocation can satisfy a resource request
     */
    canAllocate(id: string, request: ResourceRequest): boolean {
        const allocation = this.allocations.get(id);
        if (!allocation) {
            return false;
        }

        const available = this.calculateAvailable(allocation);
        
        return (
            request.credits <= available.credits &&
            request.tokens <= available.tokens &&
            request.duration <= available.time
        );
    }

    /**
     * Get remaining resources for an allocation
     */
    getRemaining(id: string): ResourceUsage | null {
        const allocation = this.allocations.get(id);
        if (!allocation) {
            return null;
        }

        return this.calculateAvailable(allocation);
    }

    /**
     * Get usage statistics for an allocation
     */
    getUsageStats(id: string): {
        consumed: ResourceUsage;
        reserved: ResourceUsage;
        remaining: ResourceUsage;
        percentUsed: {
            credits: number;
            tokens: number;
            time: number;
        };
    } | null {
        const allocation = this.allocations.get(id);
        if (!allocation) {
            return null;
        }

        const remaining = this.calculateAvailable(allocation);
        
        return {
            consumed: { ...allocation.consumed },
            reserved: { ...allocation.reserved },
            remaining,
            percentUsed: {
                credits: this.calculatePercentage(allocation.consumed.credits, allocation.limits.maxCredits),
                tokens: this.calculatePercentage(allocation.consumed.tokens, allocation.limits.maxTokens),
                time: this.calculatePercentage(allocation.consumed.time, allocation.limits.maxTime),
            },
        };
    }

    /**
     * Release resources (for cleanup)
     */
    async releaseAllocation(id: string): Promise<boolean> {
        const allocation = this.allocations.get(id);
        if (!allocation) {
            return false;
        }

        // If has parent, release reserved resources back to parent
        if (allocation.parentId) {
            const parent = this.allocations.get(allocation.parentId);
            if (parent) {
                // Calculate how much was reserved for this child
                const reserved = {
                    credits: allocation.limits.maxCredits,
                    tokens: allocation.limits.maxTokens,
                    time: allocation.limits.maxTime,
                };

                // Release back to parent
                parent.reserved.credits = Math.max(0, parent.reserved.credits - reserved.credits);
                parent.reserved.tokens = Math.max(0, parent.reserved.tokens - reserved.tokens);
                parent.reserved.time = Math.max(0, parent.reserved.time - reserved.time);
                parent.updatedAt = new Date();
            }
        }

        // Remove allocation
        this.allocations.delete(id);
        
        this.logger.info(`[${this.constructor.name}] Released allocation: ${id}`);
        
        return true;
    }

    /**
     * Get all child allocations
     */
    getChildren(parentId: string): ResourceAllocation[] {
        const children: ResourceAllocation[] = [];
        
        for (const allocation of this.allocations.values()) {
            if (allocation.parentId === parentId) {
                children.push(allocation);
            }
        }
        
        return children;
    }

    /**
     * Calculate available resources (limits - consumed - reserved)
     */
    protected calculateAvailable(allocation: ResourceAllocation): ResourceUsage {
        return {
            credits: Math.max(0, allocation.limits.maxCredits - allocation.consumed.credits - allocation.reserved.credits),
            tokens: Math.max(0, allocation.limits.maxTokens - allocation.consumed.tokens - allocation.reserved.tokens),
            time: Math.max(0, allocation.limits.maxTime - allocation.consumed.time - allocation.reserved.time),
        };
    }

    /**
     * Reserve resources from parent allocation
     */
    protected reserveFromParent(
        parent: ResourceAllocation,
        requested: ResourceLimits,
    ): AllocationResult {
        const available = this.calculateAvailable(parent);
        
        // Check if parent has enough available resources
        if (
            requested.maxCredits > available.credits ||
            requested.maxTokens > available.tokens ||
            requested.maxTime > available.time
        ) {
            return {
                success: false,
                reason: `Insufficient resources in parent allocation. Available: ${JSON.stringify(available)}, Requested: ${JSON.stringify(requested)}`,
            };
        }

        // Reserve from parent
        parent.reserved.credits += requested.maxCredits;
        parent.reserved.tokens += requested.maxTokens;
        parent.reserved.time += requested.maxTime;
        parent.updatedAt = new Date();

        return { success: true };
    }

    /**
     * Check which limits have been exceeded
     */
    protected checkLimitsExceeded(allocation: ResourceAllocation): string[] {
        const exceeded: string[] = [];
        
        if (allocation.consumed.credits > allocation.limits.maxCredits) {
            exceeded.push("credits");
        }
        
        if (allocation.consumed.tokens > allocation.limits.maxTokens) {
            exceeded.push("tokens");
        }
        
        if (allocation.consumed.time > allocation.limits.maxTime) {
            exceeded.push("time");
        }
        
        return exceeded;
    }

    /**
     * Calculate percentage (0-100)
     */
    protected calculatePercentage(used: number, limit: number): number {
        if (limit === 0) {
            return 0;
        }
        return Math.min(100, Math.round((used / limit) * 100));
    }

    /**
     * Clear all allocations
     */
    clear(): void {
        const count = this.allocations.size;
        this.allocations.clear();
        this.logger.info(`[${this.constructor.name}] Cleared ${count} allocations`);
    }

    // Abstract methods for tier-specific behavior

    /**
     * Called when resource limits are exceeded
     */
    protected abstract onLimitsExceeded(
        id: string,
        allocation: ResourceAllocation,
        exceededResources: string[],
    ): Promise<void>;
}

/**
 * Simple resource manager implementation
 */
export class SimpleResourceManager extends BaseResourceManager {
    protected async onLimitsExceeded(
        id: string,
        allocation: ResourceAllocation,
        exceededResources: string[],
    ): Promise<void> {
        // Simple implementation just logs - subclasses can override for specific behavior
        this.logger.warn(`[SimpleResourceManager] Limits exceeded for ${id}`, {
            exceededResources,
            consumed: allocation.consumed,
            limits: allocation.limits,
        });
    }
}
