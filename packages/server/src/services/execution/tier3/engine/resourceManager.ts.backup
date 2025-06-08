import { type Logger } from "winston";
import {
    type AvailableResources,
    type ExecutionConstraints,
    type ResourceUsage,
    nanoid,
} from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/events/eventBus.js";

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
    userId?: string;
    swarmId?: string;
    allocated: AvailableResources;
    used: ResourceUsage;
    reserved: ResourceUsage;
    startTime: number;
    endTime?: number;
    status: 'active' | 'completed' | 'exceeded';
    warnings: string[];
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
 * - User-specific resource tracking
 * - Event-driven monitoring
 */
export class ResourceManager {
    private readonly allocations: Map<string, ResourceAllocation>;
    private readonly userCredits: Map<string, number>;
    private readonly logger: Logger;
    private readonly eventBus?: EventBus;
    
    // Global resource pools (would be loaded from config/database in production)
    private globalCredits = 1000000; // Total credits available
    private globalRateLimits: Map<string, RateLimit> = new Map();
    
    // Configuration
    private readonly config = {
        defaultUserCredits: 10000,
        warningThreshold: 0.8,
        criticalThreshold: 0.95,
    };
    
    // Metrics
    private metrics = {
        totalReservations: 0,
        approvedReservations: 0,
        rejectedReservations: 0,
        totalCreditsUsed: 0,
    };

    constructor(logger: Logger, eventBus?: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.allocations = new Map();
        this.userCredits = new Map();
        this.initializeRateLimits();
    }

    /**
     * Reserves budget for step execution
     * Supports both legacy (2 params) and enhanced (4 params) usage
     */
    async reserveBudget(
        available: AvailableResources,
        constraints: ExecutionConstraints,
        userId?: string,
        swarmId?: string,
    ): Promise<BudgetReservation> {
        const reservationId = nanoid();
        this.metrics.totalReservations++;

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

        // Check user credits if userId provided
        if (userId) {
            const userCreditsCheck = this.checkUserCredits(userId, constraints);
            if (!userCreditsCheck.allowed) {
                this.metrics.rejectedReservations++;
                return {
                    approved: false,
                    reservationId,
                    allocation: available,
                    reason: userCreditsCheck.reason,
                };
            }
        }

        // Reserve resources
        const allocation: ResourceAllocation = {
            reservationId,
            stepId: "", // Will be set later
            userId,
            swarmId,
            allocated: available,
            reserved: this.estimateResourceUsage(constraints),
            used: {},
            startTime: Date.now(),
            status: 'active',
            warnings: [],
        };

        this.allocations.set(reservationId, allocation);
        this.metrics.approvedReservations++;

        // Emit reservation event if event bus available
        if (this.eventBus) {
            await this.eventBus.emit({
                type: 'resource.reserved',
                timestamp: new Date(),
                data: {
                    reservationId,
                    userId,
                    swarmId,
                    credits: available.credits,
                    timeLimit: available.timeLimit,
                },
            });
        }

        this.logger.debug("[ResourceManager] Budget reserved", {
            reservationId,
            userId,
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
     * Sets the stepId for a reservation (called after reservation is approved)
     */
    setStepId(reservationId: string, stepId: string): void {
        const allocation = this.allocations.get(reservationId);
        if (allocation) {
            allocation.stepId = stepId;
        }
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

        // Check thresholds
        const usageRatio = this.calculateUsageRatio(allocation);
        
        if (usageRatio >= this.config.criticalThreshold) {
            allocation.status = 'exceeded';
            allocation.warnings.push(`Critical usage: ${Math.round(usageRatio * 100)}%`);
            
            // Emit critical event if event bus available
            if (this.eventBus) {
                await this.eventBus.emit({
                    type: 'resource.critical',
                    timestamp: new Date(),
                    data: {
                        stepId,
                        reservationId: allocation.reservationId,
                        usageRatio,
                        used: allocation.used,
                        reserved: allocation.reserved,
                    },
                });
            }
        } else if (usageRatio >= this.config.warningThreshold) {
            allocation.warnings.push(`High usage: ${Math.round(usageRatio * 100)}%`);
            
            // Emit warning event if event bus available
            if (this.eventBus) {
                await this.eventBus.emit({
                    type: 'resource.warning',
                    timestamp: new Date(),
                    data: {
                        stepId,
                        reservationId: allocation.reservationId,
                        usageRatio,
                    },
                });
            }
        }

        // Update user credits if userId available
        if (allocation.userId && usage.cost) {
            this.updateUserCredits(allocation.userId, usage.cost);
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
        allocation.status = 'completed';

        // Deduct from global credits
        if (actualUsage.cost) {
            this.globalCredits -= actualUsage.cost;
            this.metrics.totalCreditsUsed += actualUsage.cost;
        }

        // Update rate limit counters
        if (actualUsage.apiCalls) {
            this.updateRateLimitCounters(actualUsage);
        }

        // Calculate final report with efficiency metric
        const executionTime = allocation.endTime - allocation.startTime;
        const efficiency = this.calculateEfficiency(allocation);
        
        const report: ResourceUsage = {
            ...actualUsage,
            computeTime: executionTime,
            efficiency,
        };

        // Emit completion event if event bus available
        if (this.eventBus) {
            await this.eventBus.emit({
                type: 'resource.completed',
                timestamp: new Date(),
                data: {
                    reservationId,
                    userId: allocation.userId,
                    duration: executionTime,
                    cost: report.cost,
                    efficiency,
                    warnings: allocation.warnings,
                },
            });
        }

        this.logger.info("[ResourceManager] Finalized usage", {
            reservationId,
            duration: executionTime,
            cost: report.cost,
            efficiency,
            warnings: allocation.warnings,
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

    /**
     * Releases a reservation without finalizing (for cancellations)
     */
    async releaseReservation(reservationId: string): Promise<void> {
        const allocation = this.allocations.get(reservationId);
        if (!allocation) {
            return;
        }

        allocation.status = 'exceeded';
        allocation.endTime = Date.now();
        
        // Return credits if not used
        if (allocation.userId && allocation.reserved.cost) {
            const creditsToReturn = allocation.reserved.cost - (allocation.used.cost || 0);
            if (creditsToReturn > 0) {
                this.returnUserCredits(allocation.userId, creditsToReturn);
            }
        }
        
        // Clean up immediately
        this.allocations.delete(reservationId);
        
        this.logger.debug("[ResourceManager] Reservation released", {
            reservationId,
        });
    }

    /**
     * Gets current metrics for monitoring
     */
    getMetrics(): {
        activeAllocations: number;
        totalReservations: number;
        approvedReservations: number;
        rejectedReservations: number;
        totalCreditsUsed: number;
        globalCreditsRemaining: number;
    } {
        const activeAllocations = Array.from(this.allocations.values())
            .filter(a => a.status === 'active').length;
        
        return {
            activeAllocations,
            totalReservations: this.metrics.totalReservations,
            approvedReservations: this.metrics.approvedReservations,
            rejectedReservations: this.metrics.rejectedReservations,
            totalCreditsUsed: this.metrics.totalCreditsUsed,
            globalCreditsRemaining: this.globalCredits,
        };
    }

    /**
     * Private helper methods
     */
    private checkUserCredits(
        userId: string,
        constraints: ExecutionConstraints,
    ): { allowed: boolean; reason?: string } {
        const userCredits = this.userCredits.get(userId) || this.config.defaultUserCredits;
        const estimatedCost = constraints.maxCost || 0.1;
        
        if (userCredits < estimatedCost) {
            return {
                allowed: false,
                reason: `Insufficient user credits: ${userCredits} < ${estimatedCost}`,
            };
        }
        
        // Reserve credits
        this.userCredits.set(userId, userCredits - estimatedCost);
        
        return { allowed: true };
    }

    private updateUserCredits(userId: string, cost: number): void {
        const currentCredits = this.userCredits.get(userId) || this.config.defaultUserCredits;
        // Credits were already reserved, so we just track actual vs reserved
        // In a real system, we'd reconcile the difference
    }

    private returnUserCredits(userId: string, amount: number): void {
        const currentCredits = this.userCredits.get(userId) || this.config.defaultUserCredits;
        this.userCredits.set(userId, currentCredits + amount);
    }

    private calculateUsageRatio(allocation: ResourceAllocation): number {
        const ratios: number[] = [];
        
        if (allocation.reserved.cost && allocation.used.cost) {
            ratios.push(allocation.used.cost / allocation.reserved.cost);
        }
        
        if (allocation.reserved.tokens && allocation.used.tokens) {
            ratios.push(allocation.used.tokens / allocation.reserved.tokens);
        }
        
        if (allocation.reserved.computeTime && allocation.used.computeTime) {
            ratios.push(allocation.used.computeTime / allocation.reserved.computeTime);
        }
        
        return ratios.length > 0 ? Math.max(...ratios) : 0;
    }

    private calculateEfficiency(allocation: ResourceAllocation): number {
        // Calculate efficiency score (0-1)
        const usageRatio = this.calculateUsageRatio(allocation);
        const timeEfficiency = allocation.reserved.computeTime && allocation.endTime
            ? (allocation.endTime - allocation.startTime) / allocation.reserved.computeTime
            : 1;
        
        // Ideal efficiency is using ~80% of reserved resources
        const targetRatio = 0.8;
        const efficiencyScore = 1 - Math.abs(usageRatio - targetRatio);
        
        return Math.max(0, Math.min(1, efficiencyScore * timeEfficiency));
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
