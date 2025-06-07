import { type Logger } from "winston";
import {
    type AvailableResources,
    type ExecutionConstraints,
    type ResourceUsage,
    nanoid,
} from "@vrooli/shared";

/**
 * Budget reservation for a step execution
 */
export interface BudgetReservation {
    approved: boolean;
    reservationId: string;
    allocation: AvailableResources;
    reason?: string;
}

/**
 * Resource allocation tracking
 */
interface ResourceAllocation {
    reservationId: string;
    stepId: string;
    allocated: AvailableResources;
    used: ResourceUsage;
    reserved: ResourceUsage;
    startTime: number;
    endTime?: number;
}

/**
 * ResourceManager - Manages computational resources for step execution
 * 
 * This component handles:
 * - Credit tracking and budget enforcement
 * - Time management and timeout prevention
 * - API rate limiting and quota management
 * - Resource allocation for child processes
 * - Cost optimization and tracking
 * 
 * Key features:
 * - Pre-execution budget reservation
 * - Real-time usage tracking
 * - Hierarchical limit enforcement
 * - Graceful degradation when limits approached
 */
export class ResourceManager {
    private readonly allocations: Map<string, ResourceAllocation>;
    private readonly logger: Logger;
    
    // Global resource pools (would be loaded from config/database in production)
    private globalCredits = 1000000; // Total credits available
    private globalRateLimits: Map<string, RateLimit> = new Map();

    constructor(logger: Logger) {
        this.logger = logger;
        this.allocations = new Map();
        this.initializeRateLimits();
    }

    /**
     * Reserves budget for step execution
     */
    async reserveBudget(
        available: AvailableResources,
        constraints: ExecutionConstraints,
    ): Promise<BudgetReservation> {
        const reservationId = nanoid();

        // Check credit limits
        if (constraints.maxCost && available.credits < constraints.maxCost) {
            return {
                approved: false,
                reservationId,
                allocation: available,
                reason: `Insufficient credits: ${available.credits} < ${constraints.maxCost}`,
            };
        }

        // Check time limits
        if (constraints.maxTime && available.timeLimit && available.timeLimit < constraints.maxTime) {
            return {
                approved: false,
                reservationId,
                allocation: available,
                reason: `Insufficient time: ${available.timeLimit}ms < ${constraints.maxTime}ms`,
            };
        }

        // Check model availability
        const requiredModels = this.extractRequiredModels(constraints);
        const availableModels = available.models.filter(m => m.available);
        const hasRequiredModels = requiredModels.every(req => 
            availableModels.some(avail => avail.model === req),
        );

        if (!hasRequiredModels) {
            return {
                approved: false,
                reservationId,
                allocation: available,
                reason: "Required models not available",
            };
        }

        // Reserve resources
        const allocation: ResourceAllocation = {
            reservationId,
            stepId: "", // Will be set later
            allocated: available,
            reserved: this.estimateResourceUsage(constraints),
            used: {},
            startTime: Date.now(),
        };

        this.allocations.set(reservationId, allocation);

        this.logger.debug("[ResourceManager] Budget reserved", {
            reservationId,
            credits: available.credits,
            timeLimit: available.timeLimit,
        });

        return {
            approved: true,
            reservationId,
            allocation: available,
        };
    }

    /**
     * Tracks actual resource usage during execution
     */
    async trackUsage(
        stepId: string,
        usage: ResourceUsage,
    ): Promise<void> {
        // Find allocation by stepId
        const allocation = Array.from(this.allocations.values())
            .find(a => a.stepId === stepId);

        if (!allocation) {
            this.logger.warn(`[ResourceManager] No allocation found for step ${stepId}`);
            return;
        }

        // Update usage
        allocation.used = this.mergeResourceUsage(allocation.used, usage);

        // Check if usage exceeds reservation
        if (this.isUsageExceeded(allocation)) {
            this.logger.warn("[ResourceManager] Usage exceeded reservation", {
                stepId,
                used: allocation.used,
                reserved: allocation.reserved,
            });
        }
    }

    /**
     * Finalizes resource usage and returns report
     */
    async finalizeUsage(
        reservationId: string,
        actualUsage: ResourceUsage,
    ): Promise<ResourceUsage> {
        const allocation = this.allocations.get(reservationId);
        
        if (!allocation) {
            this.logger.warn(`[ResourceManager] No allocation found for ${reservationId}`);
            return actualUsage;
        }

        // Update final usage
        allocation.used = actualUsage;
        allocation.endTime = Date.now();

        // Deduct from global credits
        if (actualUsage.cost) {
            this.globalCredits -= actualUsage.cost;
        }

        // Update rate limit counters
        if (actualUsage.apiCalls) {
            this.updateRateLimitCounters(actualUsage);
        }

        // Calculate final report
        const report: ResourceUsage = {
            ...actualUsage,
            computeTime: allocation.endTime - allocation.startTime,
        };

        this.logger.info("[ResourceManager] Finalized usage", {
            reservationId,
            duration: report.computeTime,
            cost: report.cost,
        });

        // Clean up allocation after a delay
        setTimeout(() => {
            this.allocations.delete(reservationId);
        }, 60000); // Keep for 1 minute for debugging

        return report;
    }

    /**
     * Checks quota for a specific resource operation
     */
    async checkQuota(
        resourceType: string,
        estimatedCost: number,
    ): Promise<{ allowed: boolean; reason?: string }> {
        // Check global credits
        if (this.globalCredits < estimatedCost) {
            return {
                allowed: false,
                reason: "Insufficient global credits",
            };
        }

        // Check rate limits
        const rateLimit = this.globalRateLimits.get(resourceType);
        if (rateLimit && !this.isWithinRateLimit(rateLimit)) {
            return {
                allowed: false,
                reason: `Rate limit exceeded for ${resourceType}`,
            };
        }

        return { allowed: true };
    }

    /**
     * Configures resource allocation for a child process
     */
    configureChildAllocation(
        parentAllocation: AvailableResources,
        allocationRatio = 0.3,
    ): AvailableResources {
        return {
            models: parentAllocation.models,
            tools: parentAllocation.tools,
            apis: parentAllocation.apis,
            credits: Math.floor(parentAllocation.credits * allocationRatio),
            timeLimit: parentAllocation.timeLimit 
                ? Math.floor(parentAllocation.timeLimit * allocationRatio)
                : undefined,
        };
    }

    /**
     * Private helper methods
     */
    private initializeRateLimits(): void {
        // Initialize common rate limits
        this.globalRateLimits.set("openai", {
            resource: "openai",
            limit: 3000,
            window: 60000, // 1 minute
            current: 0,
            resetTime: Date.now() + 60000,
        });

        this.globalRateLimits.set("anthropic", {
            resource: "anthropic",
            limit: 1000,
            window: 60000,
            current: 0,
            resetTime: Date.now() + 60000,
        });
    }

    private extractRequiredModels(constraints: ExecutionConstraints): string[] {
        // Extract from allowed strategies or other constraints
        const models: string[] = [];
        
        if (constraints.allowedStrategies?.includes("DETERMINISTIC" as any)) {
            models.push("gpt-3.5-turbo"); // Efficient model for deterministic
        }
        
        if (constraints.allowedStrategies?.includes("REASONING" as any)) {
            models.push("gpt-4"); // More capable model for reasoning
        }

        return models;
    }

    private estimateResourceUsage(constraints: ExecutionConstraints): ResourceUsage {
        return {
            tokens: constraints.maxTokens || 1000,
            apiCalls: 5, // Estimated
            computeTime: constraints.maxTime || 30000,
            cost: constraints.maxCost || 0.1,
        };
    }

    private mergeResourceUsage(
        current: ResourceUsage,
        additional: ResourceUsage,
    ): ResourceUsage {
        return {
            tokens: (current.tokens || 0) + (additional.tokens || 0),
            apiCalls: (current.apiCalls || 0) + (additional.apiCalls || 0),
            computeTime: Math.max(current.computeTime || 0, additional.computeTime || 0),
            memory: Math.max(current.memory || 0, additional.memory || 0),
            cost: (current.cost || 0) + (additional.cost || 0),
        };
    }

    private isUsageExceeded(allocation: ResourceAllocation): boolean {
        const used = allocation.used;
        const reserved = allocation.reserved;

        if (used.cost && reserved.cost && used.cost > reserved.cost) {
            return true;
        }

        if (used.tokens && reserved.tokens && used.tokens > reserved.tokens) {
            return true;
        }

        return false;
    }

    private updateRateLimitCounters(usage: ResourceUsage): void {
        // Update rate limit counters based on usage
        if (usage.apiCalls) {
            for (const [resource, limit] of this.globalRateLimits) {
                // Simple increment - in production, would track specific API calls
                limit.current += usage.apiCalls;
                
                // Reset if window passed
                if (Date.now() > limit.resetTime) {
                    limit.current = usage.apiCalls;
                    limit.resetTime = Date.now() + limit.window;
                }
            }
        }
    }

    private isWithinRateLimit(limit: RateLimit): boolean {
        // Reset if window passed
        if (Date.now() > limit.resetTime) {
            limit.current = 0;
            limit.resetTime = Date.now() + limit.window;
        }

        return limit.current < limit.limit;
    }
}

/**
 * Rate limit tracking
 */
interface RateLimit {
    resource: string;
    limit: number;
    window: number; // milliseconds
    current: number;
    resetTime: number;
}
