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
 * Resource allocation tracking with enhanced metrics
 */
interface ResourceAllocation {
    reservationId: string;
    stepId: string;
    userId: string;
    swarmId?: string;
    allocated: AvailableResources;
    reserved: ResourceUsage;
    used: ResourceUsage;
    startTime: number;
    endTime?: number;
    status: 'active' | 'completed' | 'exceeded' | 'cancelled';
    warnings: string[];
}

/**
 * Rate limit tracking with burst support
 */
interface RateLimit {
    resource: string;
    limit: number;
    burstLimit: number;
    window: number; // milliseconds
    current: number;
    burstCurrent: number;
    resetTime: number;
    violations: number;
}

/**
 * User-specific resource tracking
 */
interface UserResourceState {
    userId: string;
    creditsUsed: number;
    creditsLimit: number;
    dailyUsage: number;
    monthlyUsage: number;
    rateLimits: Map<string, RateLimit>;
    lastActivity: number;
}

/**
 * Resource pool for distributed management
 */
interface ResourcePool {
    name: string;
    totalCapacity: number;
    availableCapacity: number;
    reservedCapacity: number;
    allocations: Set<string>; // reservation IDs
}

/**
 * EnhancedResourceManager - Production-ready resource management
 * 
 * Enhanced features:
 * - User-specific credit tracking and limits
 * - Distributed resource pools for scaling
 * - Sophisticated rate limiting with burst support
 * - Real-time usage monitoring and alerts
 * - Automatic cleanup and resource recovery
 * - Integration with event bus for monitoring
 * - Predictive resource allocation
 * - Cost optimization strategies
 */
export class EnhancedResourceManager {
    private readonly allocations: Map<string, ResourceAllocation>;
    private readonly userStates: Map<string, UserResourceState>;
    private readonly resourcePools: Map<string, ResourcePool>;
    private readonly globalRateLimits: Map<string, RateLimit>;
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    
    // Configuration (would be loaded from config/database)
    private readonly config = {
        defaultUserCredits: 10000,
        warningThreshold: 0.8, // Warn when 80% of resources used
        criticalThreshold: 0.95, // Critical when 95% used
        cleanupInterval: 60000, // 1 minute
        staleAllocationTimeout: 300000, // 5 minutes
        burstMultiplier: 1.5, // Allow 50% burst capacity
    };

    // Metrics tracking
    private metrics = {
        totalAllocations: 0,
        successfulAllocations: 0,
        failedAllocations: 0,
        totalCreditsUsed: 0,
        averageExecutionTime: 0,
    };

    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.allocations = new Map();
        this.userStates = new Map();
        this.resourcePools = new Map();
        this.globalRateLimits = new Map();
        
        this.initializeResourcePools();
        this.initializeRateLimits();
        this.startCleanupTimer();
    }

    /**
     * Reserves budget for step execution with enhanced validation
     */
    async reserveBudget(
        available: AvailableResources,
        constraints: ExecutionConstraints,
        userId: string,
        swarmId?: string,
    ): Promise<BudgetReservation> {
        const reservationId = nanoid();
        this.metrics.totalAllocations++;

        try {
            // 1. Get or create user state
            const userState = this.getOrCreateUserState(userId);
            
            // 2. Check user-specific limits
            const userCheck = await this.checkUserLimits(userState, constraints);
            if (!userCheck.allowed) {
                this.metrics.failedAllocations++;
                return this.createRejection(reservationId, available, userCheck.reason);
            }

            // 3. Check global resource pools
            const poolCheck = await this.checkResourcePools(constraints);
            if (!poolCheck.allowed) {
                this.metrics.failedAllocations++;
                return this.createRejection(reservationId, available, poolCheck.reason);
            }

            // 4. Check rate limits with burst support
            const rateLimitCheck = await this.checkRateLimitsWithBurst(userState, constraints);
            if (!rateLimitCheck.allowed) {
                this.metrics.failedAllocations++;
                return this.createRejection(reservationId, available, rateLimitCheck.reason);
            }

            // 5. Calculate optimal resource allocation
            const optimizedAllocation = await this.optimizeResourceAllocation(
                available,
                constraints,
                userState,
            );

            // 6. Create and store allocation
            const allocation: ResourceAllocation = {
                reservationId,
                stepId: "", // Set by executor
                userId,
                swarmId,
                allocated: optimizedAllocation,
                reserved: this.estimateResourceUsage(constraints, userState),
                used: {},
                startTime: Date.now(),
                status: 'active',
                warnings: [],
            };

            this.allocations.set(reservationId, allocation);
            
            // 7. Update resource pools
            await this.updateResourcePools(allocation);
            
            // 8. Emit reservation event
            await this.eventBus.emit({
                type: 'resource.reserved',
                timestamp: new Date(),
                data: {
                    reservationId,
                    userId,
                    swarmId,
                    credits: optimizedAllocation.credits,
                    timeLimit: optimizedAllocation.timeLimit,
                },
            });

            this.logger.info("[EnhancedResourceManager] Budget reserved", {
                reservationId,
                userId,
                credits: optimizedAllocation.credits,
                timeLimit: optimizedAllocation.timeLimit,
            });

            this.metrics.successfulAllocations++;
            return {
                approved: true,
                reservationId,
                allocation: optimizedAllocation,
            };

        } catch (error) {
            this.logger.error("[EnhancedResourceManager] Reservation failed", {
                error: error instanceof Error ? error.message : String(error),
                userId,
            });
            
            this.metrics.failedAllocations++;
            return this.createRejection(
                reservationId,
                available,
                "Internal error during reservation",
            );
        }
    }

    /**
     * Tracks actual resource usage with threshold monitoring
     */
    async trackUsage(
        stepId: string,
        usage: ResourceUsage,
    ): Promise<void> {
        const allocation = this.findAllocationByStepId(stepId);
        if (!allocation) {
            this.logger.warn(`[EnhancedResourceManager] No allocation found for step ${stepId}`);
            return;
        }

        // Update usage
        const previousUsage = { ...allocation.used };
        allocation.used = this.mergeResourceUsage(allocation.used, usage);

        // Check thresholds
        const usageRatio = this.calculateUsageRatio(allocation);
        
        if (usageRatio >= this.config.criticalThreshold) {
            allocation.status = 'exceeded';
            allocation.warnings.push(`Critical usage: ${Math.round(usageRatio * 100)}%`);
            
            // Emit critical event
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
        } else if (usageRatio >= this.config.warningThreshold) {
            allocation.warnings.push(`High usage: ${Math.round(usageRatio * 100)}%`);
            
            // Emit warning event
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

        // Update user state
        const userState = this.userStates.get(allocation.userId);
        if (userState && usage.cost) {
            userState.creditsUsed += usage.cost;
            userState.dailyUsage += usage.cost;
            userState.monthlyUsage += usage.cost;
            userState.lastActivity = Date.now();
        }
    }

    /**
     * Finalizes resource usage with comprehensive reporting
     */
    async finalizeUsage(
        reservationId: string,
        actualUsage: ResourceUsage,
    ): Promise<ResourceUsage> {
        const allocation = this.allocations.get(reservationId);
        if (!allocation) {
            this.logger.warn(`[EnhancedResourceManager] No allocation found for ${reservationId}`);
            return actualUsage;
        }

        try {
            // Update final usage
            allocation.used = actualUsage;
            allocation.endTime = Date.now();
            allocation.status = 'completed';

            // Calculate execution metrics
            const executionTime = allocation.endTime - allocation.startTime;
            this.updateExecutionMetrics(executionTime);

            // Update user state
            const userState = this.userStates.get(allocation.userId);
            if (userState) {
                await this.updateUserResourceState(userState, actualUsage);
            }

            // Release resource pools
            await this.releaseResourcePools(allocation);

            // Generate comprehensive report
            const report: ResourceUsage = {
                ...actualUsage,
                computeTime: executionTime,
                // Add efficiency metrics
                efficiency: this.calculateEfficiency(allocation),
            };

            // Emit completion event
            await this.eventBus.emit({
                type: 'resource.completed',
                timestamp: new Date(),
                data: {
                    reservationId,
                    userId: allocation.userId,
                    duration: executionTime,
                    cost: report.cost,
                    efficiency: report.efficiency,
                    warnings: allocation.warnings,
                },
            });

            this.logger.info("[EnhancedResourceManager] Finalized usage", {
                reservationId,
                duration: executionTime,
                cost: report.cost,
                efficiency: report.efficiency,
            });

            // Schedule cleanup
            this.scheduleAllocationCleanup(reservationId);

            return report;

        } catch (error) {
            this.logger.error("[EnhancedResourceManager] Finalization failed", {
                reservationId,
                error: error instanceof Error ? error.message : String(error),
            });
            return actualUsage;
        }
    }

    /**
     * Releases a reservation without finalizing
     */
    async releaseReservation(reservationId: string): Promise<void> {
        const allocation = this.allocations.get(reservationId);
        if (!allocation) {
            return;
        }

        allocation.status = 'cancelled';
        allocation.endTime = Date.now();
        
        // Release resource pools
        await this.releaseResourcePools(allocation);
        
        // Clean up immediately
        this.allocations.delete(reservationId);
        
        this.logger.debug("[EnhancedResourceManager] Reservation released", {
            reservationId,
        });
    }

    /**
     * Gets current resource state for monitoring
     */
    getResourceState(): {
        activeAllocations: number;
        totalCreditsAvailable: number;
        poolUtilization: Record<string, number>;
        metrics: typeof this.metrics;
    } {
        const activeAllocations = Array.from(this.allocations.values())
            .filter(a => a.status === 'active').length;
        
        const totalCreditsAvailable = Array.from(this.userStates.values())
            .reduce((sum, state) => sum + (state.creditsLimit - state.creditsUsed), 0);
        
        const poolUtilization: Record<string, number> = {};
        for (const [name, pool] of this.resourcePools) {
            poolUtilization[name] = 1 - (pool.availableCapacity / pool.totalCapacity);
        }
        
        return {
            activeAllocations,
            totalCreditsAvailable,
            poolUtilization,
            metrics: { ...this.metrics },
        };
    }

    /**
     * Private helper methods
     */
    private initializeResourcePools(): void {
        // Initialize resource pools for different resource types
        this.resourcePools.set('compute', {
            name: 'compute',
            totalCapacity: 1000, // Compute units
            availableCapacity: 1000,
            reservedCapacity: 0,
            allocations: new Set(),
        });
        
        this.resourcePools.set('memory', {
            name: 'memory',
            totalCapacity: 16384, // MB
            availableCapacity: 16384,
            reservedCapacity: 0,
            allocations: new Set(),
        });
        
        this.resourcePools.set('api_quota', {
            name: 'api_quota',
            totalCapacity: 10000, // API calls
            availableCapacity: 10000,
            reservedCapacity: 0,
            allocations: new Set(),
        });
    }

    private initializeRateLimits(): void {
        // OpenAI rate limits with burst support
        this.globalRateLimits.set("openai", {
            resource: "openai",
            limit: 3000,
            burstLimit: 4500, // 50% burst
            window: 60000, // 1 minute
            current: 0,
            burstCurrent: 0,
            resetTime: Date.now() + 60000,
            violations: 0,
        });

        // Anthropic rate limits
        this.globalRateLimits.set("anthropic", {
            resource: "anthropic",
            limit: 1000,
            burstLimit: 1500,
            window: 60000,
            current: 0,
            burstCurrent: 0,
            resetTime: Date.now() + 60000,
            violations: 0,
        });
        
        // Tool execution rate limits
        this.globalRateLimits.set("tools", {
            resource: "tools",
            limit: 100,
            burstLimit: 150,
            window: 60000,
            current: 0,
            burstCurrent: 0,
            resetTime: Date.now() + 60000,
            violations: 0,
        });
    }

    private startCleanupTimer(): void {
        setInterval(() => {
            this.cleanupStaleAllocations();
            this.resetDailyUsage();
        }, this.config.cleanupInterval);
    }

    private cleanupStaleAllocations(): void {
        const now = Date.now();
        const staleTimeout = this.config.staleAllocationTimeout;
        
        for (const [id, allocation] of this.allocations) {
            if (allocation.status === 'active' && 
                now - allocation.startTime > staleTimeout) {
                this.logger.warn("[EnhancedResourceManager] Cleaning stale allocation", {
                    reservationId: id,
                    age: now - allocation.startTime,
                });
                
                allocation.status = 'exceeded';
                allocation.endTime = now;
                this.releaseResourcePools(allocation);
                this.allocations.delete(id);
            }
        }
    }

    private resetDailyUsage(): void {
        const now = new Date();
        const isNewDay = now.getHours() === 0 && now.getMinutes() === 0;
        
        if (isNewDay) {
            for (const userState of this.userStates.values()) {
                userState.dailyUsage = 0;
            }
        }
    }

    private getOrCreateUserState(userId: string): UserResourceState {
        let state = this.userStates.get(userId);
        if (!state) {
            state = {
                userId,
                creditsUsed: 0,
                creditsLimit: this.config.defaultUserCredits,
                dailyUsage: 0,
                monthlyUsage: 0,
                rateLimits: new Map(),
                lastActivity: Date.now(),
            };
            this.userStates.set(userId, state);
        }
        return state;
    }

    private async checkUserLimits(
        userState: UserResourceState,
        constraints: ExecutionConstraints,
    ): Promise<{ allowed: boolean; reason?: string }> {
        const estimatedCost = constraints.maxCost || 0.1;
        const remainingCredits = userState.creditsLimit - userState.creditsUsed;
        
        if (remainingCredits < estimatedCost) {
            return {
                allowed: false,
                reason: `Insufficient user credits: ${remainingCredits} < ${estimatedCost}`,
            };
        }
        
        // Check daily limits
        const dailyLimit = userState.creditsLimit * 0.5; // 50% daily limit
        if (userState.dailyUsage + estimatedCost > dailyLimit) {
            return {
                allowed: false,
                reason: `Daily limit exceeded: ${userState.dailyUsage + estimatedCost} > ${dailyLimit}`,
            };
        }
        
        return { allowed: true };
    }

    private async checkResourcePools(
        constraints: ExecutionConstraints,
    ): Promise<{ allowed: boolean; reason?: string }> {
        // Check compute pool
        const computePool = this.resourcePools.get('compute');
        if (computePool) {
            const requiredCompute = Math.ceil((constraints.maxTime || 30000) / 100); // Rough estimate
            if (computePool.availableCapacity < requiredCompute) {
                return {
                    allowed: false,
                    reason: `Insufficient compute capacity: ${computePool.availableCapacity} < ${requiredCompute}`,
                };
            }
        }
        
        // Check memory pool
        const memoryPool = this.resourcePools.get('memory');
        if (memoryPool) {
            const requiredMemory = 512; // Default 512MB per execution
            if (memoryPool.availableCapacity < requiredMemory) {
                return {
                    allowed: false,
                    reason: `Insufficient memory: ${memoryPool.availableCapacity}MB < ${requiredMemory}MB`,
                };
            }
        }
        
        return { allowed: true };
    }

    private async checkRateLimitsWithBurst(
        userState: UserResourceState,
        constraints: ExecutionConstraints,
    ): Promise<{ allowed: boolean; reason?: string }> {
        // Check global rate limits
        for (const [resource, limit] of this.globalRateLimits) {
            if (!this.isWithinRateLimitWithBurst(limit)) {
                return {
                    allowed: false,
                    reason: `Rate limit exceeded for ${resource}`,
                };
            }
        }
        
        // Check user-specific rate limits
        // TODO: Implement user-specific rate limits
        
        return { allowed: true };
    }

    private isWithinRateLimitWithBurst(limit: RateLimit): boolean {
        const now = Date.now();
        
        // Reset if window passed
        if (now > limit.resetTime) {
            limit.current = 0;
            limit.burstCurrent = 0;
            limit.resetTime = now + limit.window;
            limit.violations = 0;
        }
        
        // Check normal limit
        if (limit.current < limit.limit) {
            return true;
        }
        
        // Check burst limit
        if (limit.burstCurrent < limit.burstLimit) {
            limit.violations++;
            return true;
        }
        
        return false;
    }

    private async optimizeResourceAllocation(
        available: AvailableResources,
        constraints: ExecutionConstraints,
        userState: UserResourceState,
    ): Promise<AvailableResources> {
        // Apply smart limits based on user history and constraints
        const optimized = { ...available };
        
        // Limit credits based on user's remaining balance
        const remainingCredits = userState.creditsLimit - userState.creditsUsed;
        optimized.credits = Math.min(
            available.credits,
            remainingCredits,
            constraints.maxCost || Infinity,
        );
        
        // Optimize time limit
        if (constraints.maxTime && available.timeLimit) {
            optimized.timeLimit = Math.min(available.timeLimit, constraints.maxTime);
        }
        
        // Filter models based on cost efficiency
        if (optimized.models.length > 0) {
            optimized.models = this.selectCostEffectiveModels(
                available.models,
                constraints,
            );
        }
        
        return optimized;
    }

    private selectCostEffectiveModels(
        models: AvailableResources['models'],
        constraints: ExecutionConstraints,
    ): AvailableResources['models'] {
        // Sort by cost and select most appropriate
        return models
            .filter(m => m.available)
            .sort((a, b) => a.cost - b.cost)
            .slice(0, 3); // Keep top 3 most cost-effective
    }

    private estimateResourceUsage(
        constraints: ExecutionConstraints,
        userState: UserResourceState,
    ): ResourceUsage {
        // Use historical data to improve estimates
        const baseEstimate = {
            tokens: constraints.maxTokens || 1000,
            apiCalls: 5,
            computeTime: constraints.maxTime || 30000,
            memory: 512, // MB
            cost: constraints.maxCost || 0.1,
        };
        
        // Adjust based on user's average usage patterns
        if (this.metrics.totalAllocations > 0) {
            const avgMultiplier = userState.creditsUsed / 
                (this.metrics.totalAllocations * 0.1);
            
            baseEstimate.cost *= Math.max(0.5, Math.min(2, avgMultiplier));
        }
        
        return baseEstimate;
    }

    private async updateResourcePools(allocation: ResourceAllocation): Promise<void> {
        // Update compute pool
        const computePool = this.resourcePools.get('compute');
        if (computePool) {
            const computeUnits = Math.ceil((allocation.reserved.computeTime || 30000) / 100);
            computePool.availableCapacity -= computeUnits;
            computePool.reservedCapacity += computeUnits;
            computePool.allocations.add(allocation.reservationId);
        }
        
        // Update memory pool
        const memoryPool = this.resourcePools.get('memory');
        if (memoryPool) {
            const memoryUnits = allocation.reserved.memory || 512;
            memoryPool.availableCapacity -= memoryUnits;
            memoryPool.reservedCapacity += memoryUnits;
            memoryPool.allocations.add(allocation.reservationId);
        }
    }

    private async releaseResourcePools(allocation: ResourceAllocation): Promise<void> {
        // Release compute pool
        const computePool = this.resourcePools.get('compute');
        if (computePool && computePool.allocations.has(allocation.reservationId)) {
            const computeUnits = Math.ceil((allocation.reserved.computeTime || 30000) / 100);
            computePool.availableCapacity += computeUnits;
            computePool.reservedCapacity -= computeUnits;
            computePool.allocations.delete(allocation.reservationId);
        }
        
        // Release memory pool
        const memoryPool = this.resourcePools.get('memory');
        if (memoryPool && memoryPool.allocations.has(allocation.reservationId)) {
            const memoryUnits = allocation.reserved.memory || 512;
            memoryPool.availableCapacity += memoryUnits;
            memoryPool.reservedCapacity -= memoryUnits;
            memoryPool.allocations.delete(allocation.reservationId);
        }
    }

    private findAllocationByStepId(stepId: string): ResourceAllocation | undefined {
        return Array.from(this.allocations.values())
            .find(a => a.stepId === stepId);
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
        const timeEfficiency = allocation.reserved.computeTime 
            ? (allocation.endTime! - allocation.startTime) / allocation.reserved.computeTime
            : 1;
        
        // Ideal efficiency is using ~80% of reserved resources
        const targetRatio = 0.8;
        const efficiencyScore = 1 - Math.abs(usageRatio - targetRatio);
        
        return Math.max(0, Math.min(1, efficiencyScore * timeEfficiency));
    }

    private updateExecutionMetrics(executionTime: number): void {
        const currentAvg = this.metrics.averageExecutionTime;
        const totalExecutions = this.metrics.successfulAllocations;
        
        this.metrics.averageExecutionTime = 
            (currentAvg * (totalExecutions - 1) + executionTime) / totalExecutions;
    }

    private async updateUserResourceState(
        userState: UserResourceState,
        usage: ResourceUsage,
    ): Promise<void> {
        if (usage.cost) {
            userState.creditsUsed += usage.cost;
            userState.dailyUsage += usage.cost;
            userState.monthlyUsage += usage.cost;
            this.metrics.totalCreditsUsed += usage.cost;
        }
        
        userState.lastActivity = Date.now();
        
        // Update rate limits
        if (usage.apiCalls) {
            for (const [resource, globalLimit] of this.globalRateLimits) {
                if (globalLimit.current < globalLimit.limit) {
                    globalLimit.current += usage.apiCalls;
                } else {
                    globalLimit.burstCurrent += usage.apiCalls;
                }
            }
        }
    }

    private scheduleAllocationCleanup(reservationId: string): void {
        setTimeout(() => {
            const allocation = this.allocations.get(reservationId);
            if (allocation && allocation.status === 'completed') {
                this.allocations.delete(reservationId);
                this.logger.debug("[EnhancedResourceManager] Cleaned up allocation", {
                    reservationId,
                });
            }
        }, 60000); // Clean up after 1 minute
    }

    private createRejection(
        reservationId: string,
        available: AvailableResources,
        reason: string,
    ): BudgetReservation {
        this.logger.warn("[EnhancedResourceManager] Budget rejected", {
            reservationId,
            reason,
        });
        
        return {
            approved: false,
            reservationId,
            allocation: available,
            reason,
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
            efficiency: current.efficiency, // Keep existing efficiency
        };
    }
}