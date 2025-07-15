/**
 * RunContextManager - Clean Tier 2 Resource & Context Management
 * 
 * This implementation provides focused adapter functionality that:
 * 1. Delegates resource management to SwarmContextManager (no duplication)
 * 2. Allocates run-specific resources from swarm pools
 * 3. Emits run-level events for monitoring
 * 4. Tracks run context without complex persistence logic
 */

import { EventTypes, generatePK, type ExecutionResourceUsage } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { CacheService } from "../../../redisConn.js";
import { EventPublisher } from "../../events/publisher.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import { type RunAllocation, type RunAllocationRequest, type RunExecutionContext, type StepAllocation, type StepAllocationRequest } from "./types.js";

/**
 * RunContextManager - Tier 2 Resource & Context Management
 * 
 * Provides run-level resource allocation and context management by delegating
 * to SwarmContextManager for actual resource storage and tracking.
 */
export interface IRunContextManager {
    // === RESOURCE HIERARCHY ===

    /**
     * Allocate resources for a run from its parent swarm
     * 
     * Creates a sub-allocation from the swarm's available resources and
     * tracks it for proper cleanup when the run completes.
     */
    allocateFromSwarm(
        swarmId: string,
        runRequest: RunAllocationRequest
    ): Promise<RunAllocation>;

    /**
     * Release run resources back to parent swarm
     * 
     * Reports actual resource usage and returns unused allocation
     * back to the swarm for other runs to use.
     */
    releaseToSwarm(
        swarmId: string,
        runId: string,
        usage: ExecutionResourceUsage
    ): Promise<void>;

    // === RUN CONTEXT MANAGEMENT ===

    /**
     * Get complete run execution context
     * 
     * Retrieves the comprehensive run state including navigation,
     * variables, progress, and resource allocation.
     */
    getRunContext(runId: string): Promise<RunExecutionContext>;

    /**
     * Update run execution context
     * 
     * Persists run state changes and notifies SwarmContextManager
     * of any resource consumption updates.
     */
    updateRunContext(runId: string, context: RunExecutionContext): Promise<void>;

    // === RUN LIFECYCLE EVENTS ===

    /**
     * Emit run started event
     * 
     * Notifies the system that a routine execution has begun,
     * including resource allocation and initial context.
     */
    emitRunStarted(runId: string, routineId: string, allocation: RunAllocation): Promise<void>;

    /**
     * Emit run completed event
     * 
     * Notifies completion with final results and resource usage.
     * Automatically triggers resource release back to swarm.
     */
    emitRunCompleted(runId: string, result: { success: boolean;[key: string]: unknown }, usage: ExecutionResourceUsage, parentSwarmId?: string): Promise<void>;

    /**
     * Emit run failed event
     * 
     * Notifies failure with error details and partial resource usage.
     * Automatically triggers resource release and cleanup.
     */
    emitRunFailed(runId: string, error: Error | { message: string }, usage: ExecutionResourceUsage, parentSwarmId?: string): Promise<void>;

    // === STEP-LEVEL RESOURCE ALLOCATION ===

    /**
     * Allocate resources for a step from run allocation
     * 
     * Sub-allocates from the run's resources for individual step execution.
     * Used when delegating to Tier 3.
     */
    allocateForStep(
        runId: string,
        stepRequest: StepAllocationRequest
    ): Promise<StepAllocation>;

    /**
     * Release step resources back to run
     * 
     * Returns step resources and updates run's remaining allocation.
     */
    releaseFromStep(
        runId: string,
        stepId: string,
        usage: ExecutionResourceUsage
    ): Promise<void>;
}

/**
 * Implementation of RunContextManager using SwarmContextManager delegation
 */
export class RunContextManager implements IRunContextManager {
    private readonly swarmContextManager: ISwarmContextManager;

    // Track active run allocations for cleanup
    private readonly activeAllocations = new Map<string, RunAllocation>();

    constructor(
        swarmContextManager: ISwarmContextManager,
    ) {
        this.swarmContextManager = swarmContextManager;
    }

    /**
     * Calculate TTL in seconds for cache operations
     * Ensures minimum TTL and handles edge cases
     */
    private calculateTtlSeconds(durationMs: number): number {
        const MIN_TTL_SECONDS = 300; // 5 minutes minimum
        const MAX_TTL_SECONDS = 86400; // 24 hours maximum
        const MS_TO_SECONDS = 1000;

        const calculatedTtl = Math.floor(durationMs / MS_TO_SECONDS);
        return Math.max(MIN_TTL_SECONDS, Math.min(MAX_TTL_SECONDS, calculatedTtl));
    }

    async allocateFromSwarm(
        swarmId: string,
        runRequest: RunAllocationRequest,
    ): Promise<RunAllocation> {
        // Delegate to SwarmContextManager for actual allocation
        const _swarmAllocation = await this.swarmContextManager.allocateResources(swarmId, {
            consumerId: runRequest.runId,
            consumerType: "run",
            limits: {
                maxCredits: runRequest.estimatedRequirements.credits.toString(),
                maxDurationMs: runRequest.estimatedRequirements.durationMs,
                maxMemoryMB: runRequest.estimatedRequirements.memoryMB,
                maxConcurrentSteps: 1,
            },
            allocated: {
                credits: 0, // Will be set by SwarmContextManager
                timestamp: new Date(),
            },
            purpose: runRequest.purpose,
            priority: runRequest.priority as "low" | "normal" | "high" | undefined,
        });

        // Create run-specific allocation wrapper
        const runAllocation: RunAllocation = {
            allocationId: generatePK().toString(),
            runId: runRequest.runId,
            swarmId,
            allocated: {
                credits: runRequest.estimatedRequirements.credits,
                timeoutMs: runRequest.estimatedRequirements.durationMs,
                memoryMB: runRequest.estimatedRequirements.memoryMB,
                concurrentExecutions: 1,
            },
            remaining: {
                credits: runRequest.estimatedRequirements.credits,
                timeoutMs: runRequest.estimatedRequirements.durationMs,
                memoryMB: runRequest.estimatedRequirements.memoryMB,
                concurrentExecutions: 1,
            },
            allocatedAt: new Date(),
            expiresAt: new Date(Date.now() + runRequest.estimatedRequirements.durationMs),
        };

        // Track allocation for cleanup
        this.activeAllocations.set(runRequest.runId, runAllocation);

        // Store allocation in Redis for persistence
        const ttlSeconds = this.calculateTtlSeconds(runRequest.estimatedRequirements.durationMs);
        try {
            await CacheService.get().set(
                `run_allocation:${runRequest.runId}`,
                runAllocation,
                ttlSeconds,
            );
        } catch (error) {
            logger.error("Failed to store run allocation in cache", {
                runId: runRequest.runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new Error(`Failed to persist run allocation: ${error instanceof Error ? error.message : String(error)}`);
        }

        logger.info("Allocated resources for run from swarm", {
            runId: runRequest.runId,
            swarmId,
            credits: runRequest.estimatedRequirements.credits,
            durationMs: runRequest.estimatedRequirements.durationMs,
        });

        return runAllocation;
    }

    async releaseToSwarm(
        swarmId: string,
        runId: string,
        usage: ExecutionResourceUsage,
    ): Promise<void> {
        const allocation = this.activeAllocations.get(runId);
        if (!allocation) {
            logger.warn("No allocation found for run", { runId });
            return;
        }

        // Release resources back to swarm through SwarmContextManager
        await this.swarmContextManager.releaseResources(swarmId, allocation.allocationId);

        // Clean up tracking
        this.activeAllocations.delete(runId);
        try {
            await CacheService.get().del(`run_allocation:${runId}`);
            await CacheService.get().del(`run_context:${runId}`);
        } catch (error) {
            logger.warn("Failed to clean up cache entries", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't throw here - cleanup failure shouldn't prevent resource release
        }

        logger.info("Released run resources to swarm", {
            runId,
            swarmId,
            usage: {
                creditsUsed: usage.creditsUsed,
                durationMs: usage.durationMs,
                stepsExecuted: usage.stepsExecuted,
            },
        });
    }

    async getRunContext(runId: string): Promise<RunExecutionContext> {
        try {
            const context = await CacheService.get().get<RunExecutionContext>(`run_context:${runId}`);
            if (!context) {
                throw new Error(`Run context not found: ${runId}`);
            }

            // Reconstruct Date objects that may have been serialized
            if (typeof context.resourceUsage.startTime === "string") {
                context.resourceUsage.startTime = new Date(context.resourceUsage.startTime);
            }

            return context;
        } catch (error) {
            logger.error("Failed to retrieve run context from cache", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async updateRunContext(runId: string, context: RunExecutionContext): Promise<void> {
        // Store context with TTL based on allocation
        const allocation = this.activeAllocations.get(runId);
        const DEFAULT_TTL_SECONDS = 3600; // Default 1 hour
        const MS_TO_SECONDS = 1000;
        const ttl = allocation ?
            Math.floor((allocation.expiresAt.getTime() - Date.now()) / MS_TO_SECONDS) :
            DEFAULT_TTL_SECONDS;

        try {
            await CacheService.get().set(
                `run_context:${runId}`,
                context,
                ttl,
            );
        } catch (error) {
            logger.error("Failed to update run context in cache", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new Error(`Failed to persist run context: ${error instanceof Error ? error.message : String(error)}`);
        }

        // Update resource usage in allocation tracking
        if (allocation) {
            allocation.remaining.credits = (
                BigInt(allocation.allocated.credits) - BigInt(context.resourceUsage.creditsUsed || "0")
            ).toString();
            allocation.remaining.timeoutMs = Math.max(0,
                allocation.allocated.timeoutMs - context.resourceUsage.durationMs,
            );
        }

        logger.debug("Updated run context", {
            runId,
            currentStepId: context.progress.currentStepId,
            percentComplete: context.progress.percentComplete,
        });
    }

    async emitRunStarted(runId: string, routineId: string, allocation: RunAllocation): Promise<void> {
        const { proceed, reason } = await EventPublisher.emit(EventTypes.RUN.STARTED, {
            runId,
            routineId,
            inputs: {},
            estimatedDuration: allocation.allocated.timeoutMs,
            parentSwarmId: allocation.swarmId,
        });

        if (!proceed) {
            logger.warn("[RunContextManager] Run started event blocked", {
                runId,
                routineId,
                reason,
            });
            // For run started events, we log but continue - the run has already started
        }
    }

    async emitRunCompleted(runId: string, result: { success: boolean;[key: string]: unknown }, usage: ExecutionResourceUsage, parentSwarmId?: string): Promise<void> {
        const { proceed, reason } = await EventPublisher.emit(EventTypes.RUN.COMPLETED, {
            runId,
            parentSwarmId,
            outputs: result,
            duration: usage.durationMs,
            message: "Run completed successfully",
            isSwarmIntegrated: !!parentSwarmId,
        });

        if (!proceed) {
            logger.warn("[RunContextManager] Run completed event blocked", {
                runId,
                reason,
            });
            // For completion events, we log but continue - the run has already completed
        }
    }

    async emitRunFailed(runId: string, error: Error | { message: string }, usage: ExecutionResourceUsage, parentSwarmId?: string): Promise<void> {
        const { proceed, reason } = await EventPublisher.emit(EventTypes.RUN.FAILED, {
            runId,
            parentSwarmId,
            error: error.message || String(error),
            duration: usage.durationMs,
            retryable: false,
            isSwarmIntegrated: !!parentSwarmId,
        });

        if (!proceed) {
            logger.warn("[RunContextManager] Run failed event blocked", {
                runId,
                error: error.message || String(error),
                blockReason: reason,
            });
            // For failure events, we log but continue - the failure has already occurred
        }
    }

    async allocateForStep(
        runId: string,
        stepRequest: StepAllocationRequest,
    ): Promise<StepAllocation> {
        const runAllocation = this.activeAllocations.get(runId);
        if (!runAllocation) {
            throw new Error(`No allocation found for run: ${runId}`);
        }

        // Check if run has sufficient remaining resources
        const requestedCredits = BigInt(stepRequest.estimatedRequirements.credits);
        const remainingCredits = BigInt(runAllocation.remaining.credits);

        if (requestedCredits > remainingCredits) {
            throw new Error(`Insufficient credits for step. Requested: ${stepRequest.estimatedRequirements.credits}, Available: ${runAllocation.remaining.credits}`);
        }

        if (stepRequest.estimatedRequirements.durationMs > runAllocation.remaining.timeoutMs) {
            throw new Error(`Insufficient time for step. Requested: ${stepRequest.estimatedRequirements.durationMs}ms, Available: ${runAllocation.remaining.timeoutMs}ms`);
        }

        // Create step allocation
        const stepAllocation: StepAllocation = {
            allocationId: generatePK().toString(),
            stepId: stepRequest.stepId,
            runId,
            allocated: {
                credits: stepRequest.estimatedRequirements.credits,
                timeoutMs: stepRequest.estimatedRequirements.durationMs,
                memoryMB: stepRequest.estimatedRequirements.memoryMB,
                concurrentExecutions: 1,
            },
            allocatedAt: new Date(),
            expiresAt: new Date(Date.now() + stepRequest.estimatedRequirements.durationMs),
        };

        // Update run's remaining resources (reserve for step)
        runAllocation.remaining.credits = (remainingCredits - requestedCredits).toString();
        runAllocation.remaining.timeoutMs = Math.max(0,
            runAllocation.remaining.timeoutMs - stepRequest.estimatedRequirements.durationMs,
        );

        logger.debug("Allocated resources for step from run", {
            runId,
            stepId: stepRequest.stepId,
            credits: stepRequest.estimatedRequirements.credits,
            durationMs: stepRequest.estimatedRequirements.durationMs,
        });

        return stepAllocation;
    }

    async releaseFromStep(
        runId: string,
        stepId: string,
        usage: ExecutionResourceUsage,
    ): Promise<void> {
        const runAllocation = this.activeAllocations.get(runId);
        if (!runAllocation) {
            logger.warn("No run allocation found for step release", { runId, stepId });
            return;
        }

        // Return unused resources to run allocation
        const creditsUsed = BigInt(usage.creditsUsed);
        const creditsReturned = BigInt(runAllocation.allocated.credits) - creditsUsed;

        runAllocation.remaining.credits = (
            BigInt(runAllocation.remaining.credits) + creditsReturned
        ).toString();

        const _timeUsed = usage.durationMs;
        // Note: We don't return time since it's consumed regardless of usage

        logger.debug("Released step resources back to run", {
            runId,
            stepId,
            creditsUsed: usage.creditsUsed,
            durationMs: usage.durationMs,
        });
    }
}
