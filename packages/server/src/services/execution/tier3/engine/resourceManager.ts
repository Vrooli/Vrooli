import { type Logger } from "winston";
import {
    type AvailableResources,
    type ExecutionConstraints,
    type ExecutionResourceUsage,
    type BudgetReservation,
    generatePk,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type ResourceAmount } from "../../cross-cutting/resources/resourceManager.js";
import { StepResourceAdapter } from "../../cross-cutting/resources/adapters.js";
import { BaseTierResourceManager } from "../../shared/BaseTierResourceManager.js";

/**
 * Budget reservation result for a step execution
 */
export interface BudgetReservationResult {
    approved: boolean;
    reservation?: BudgetReservation;
    reason?: string;
}

/**
 * ResourceManager - Tier 3 Step Resource Management
 * 
 * This component extends BaseTierResourceManager to provide step-specific
 * resource management functionality using the StepResourceAdapter.
 * 
 * Complex behaviors like burst handling and user-specific optimization
 * emerge from resource agents analyzing the events emitted by the unified manager.
 */
export class ResourceManager extends BaseTierResourceManager<StepResourceAdapter> {
    constructor(logger: Logger, eventBus: EventBus) {
        super(logger, eventBus, 3);
    }

    /**
     * Create the step resource adapter
     */
    protected createAdapter(): StepResourceAdapter {
        return new StepResourceAdapter(this.unifiedManager);
    }

    /**
     * Reserves budget for step execution
     * Uses adapter to check user limits and allocate
     */
    async reserveBudget(
        available: AvailableResources,
        constraints: ExecutionConstraints,
        userId: string,
        swarmId?: string,
    ): Promise<BudgetReservationResult> {
        return this.withErrorHandling("reserve budget", async () => {
            // Check if user can execute
            const canExecute = await this.adapter.canUserExecute(
                userId,
                {
                    credits: constraints.maxCost || available.credits,
                    time: constraints.maxTime || 300000,
                    memory: 512 * 1024 * 1024, // 512MB default
                    tokens: available.credits * 10, // Estimate
                    apiCalls: 100,
                },
            );
            
            if (!canExecute) {
                return {
                    approved: false,
                    reason: "Insufficient resources or rate limit exceeded",
                };
            }
            
            // Allocate resources for user
            const allocation = await this.unifiedManager.allocate(
                userId,
                "user",
                {
                    credits: constraints.maxCost || available.credits,
                    time: constraints.maxTime || 300000,
                    memory: 512 * 1024 * 1024,
                },
            );
            
            // Convert to BudgetReservation
            const budgetReservation: BudgetReservation = {
                id: allocation.id,
                credits: allocation.resources.credits || 0,
                timeLimit: allocation.resources.time || 300000,
                memoryLimit: (allocation.resources.memory || 0) / (1024 * 1024),
                allocated: true,
                metadata: {
                    userId,
                    swarmId,
                },
            };
            
            return {
                approved: true,
                reservation: budgetReservation,
            };
        }, () => ({
            approved: false,
            reason: "Internal error during reservation",
        }));
    }

    /**
     * Tracks actual resource usage
     * Uses adapter to track for both user and step
     */
    async trackUsage(
        stepId: string,
        usage: ExecutionResourceUsage,
    ): Promise<void> {
        // Get userId from step metadata or use a default
        const userId = "system"; // Should be passed from context
        
        await this.adapter.trackStepExecution(
            userId,
            stepId,
            {
                credits: Number(usage.creditsUsed),
                time: usage.durationMs,
                memory: usage.memoryUsedMB * 1024 * 1024,
                tokens: usage.tokens || 0,
                apiCalls: usage.apiCalls || 0,
            },
        );
    }

    /**
     * Finalizes resource usage
     * Uses unified manager to track final usage
     */
    async finalizeUsage(
        reservationId: string,
        actualUsage: ExecutionResourceUsage,
    ): Promise<ExecutionResourceUsage> {
        await this.unifiedManager.release(reservationId);
        return actualUsage;
    }

    /**
     * Releases a reservation without finalizing
     */
    async releaseReservation(reservationId: string): Promise<void> {
        await this.unifiedManager.release(reservationId);
    }

    /**
     * Gets user resource summary
     */
    getUserResourceSummary(userId: string) {
        return this.adapter.getUserSummary(userId);
    }

    /**
     * Allocates burst capacity for high-priority operations
     */
    async allocateBurstCapacity(
        userId: string,
        amount: number,
        durationMs: number,
    ): Promise<BudgetReservation | null> {
        return this.withErrorHandling("allocate burst capacity", async () => {
            const allocation = await this.adapter.allocateBurst(
                userId,
                { credits: amount },
                durationMs,
            );
            
            return {
                id: allocation.id,
                credits: allocation.resources.credits || 0,
                timeLimit: durationMs,
                memoryLimit: 512,
                allocated: true,
                metadata: { userId, burst: true },
            };
        }, () => null);
    }

    /**
     * Clean up and shutdown
     */
    async shutdown(): Promise<void> {
        await this.cleanup();
    }
}
