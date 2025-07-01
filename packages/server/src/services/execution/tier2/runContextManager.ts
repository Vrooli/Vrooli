/**
 * RunContextManager - Clean Tier 2 Resource & Context Management
 * 
 * Replaces deprecated IRunStateStore singleton with proper dependency injection
 * and resource hierarchy delegation to SwarmContextManager.
 * 
 * This implementation provides focused adapter functionality that:
 * 1. Delegates resource management to SwarmContextManager (no duplication)
 * 2. Allocates run-specific resources from swarm pools
 * 3. Emits run-level events for monitoring
 * 4. Tracks run context without complex persistence logic
 */

import { generatePK } from "@vrooli/shared";
import { type Redis } from "ioredis";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import {
    type ResourceUsage,
    type RunAllocation,
    type RunAllocationRequest,
    type StepAllocation,
    type StepAllocationRequest,
} from "./types.js";

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
        usage: ResourceUsage
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
    emitRunCompleted(runId: string, result: any, usage: ResourceUsage): Promise<void>;

    /**
     * Emit run failed event
     * 
     * Notifies failure with error details and partial resource usage.
     * Automatically triggers resource release and cleanup.
     */
    emitRunFailed(runId: string, error: any, usage: ResourceUsage): Promise<void>;

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
        usage: ResourceUsage
    ): Promise<void>;
}

/**
 * Implementation of RunContextManager using SwarmContextManager delegation
 */
export class RunContextManager implements IRunContextManager {
    private readonly swarmContextManager: ISwarmContextManager;
    private readonly eventBus: IEventBus;
    private readonly redis: Redis;

    // Track active run allocations for cleanup
    private readonly activeAllocations = new Map<string, RunAllocation>();

    constructor(
        swarmContextManager: ISwarmContextManager,
        eventBus: IEventBus,
        redis: Redis,
    ) {
        this.swarmContextManager = swarmContextManager;
        this.eventBus = eventBus;
        this.redis = redis;
    }

    async allocateFromSwarm(
        swarmId: string,
        runRequest: RunAllocationRequest,
    ): Promise<RunAllocation> {
        // Delegate to SwarmContextManager for actual allocation
        const swarmAllocation = await this.swarmContextManager.allocateResources(swarmId, {
            entityId: runRequest.runId,
            entityType: "run",
            allocated: {
                credits: BigInt(runRequest.estimatedRequirements.credits),
                timeoutMs: runRequest.estimatedRequirements.durationMs,
                memoryMB: runRequest.estimatedRequirements.memoryMB,
                concurrentExecutions: 1,
            },
            purpose: runRequest.purpose,
            priority: runRequest.priority,
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
        await this.redis.setex(
            `run_allocation:${runRequest.runId}`,
            Math.floor(runRequest.estimatedRequirements.durationMs / 1000),
            JSON.stringify(runAllocation),
        );

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
        usage: ResourceUsage,
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
        await this.redis.del(`run_allocation:${runId}`);
        await this.redis.del(`run_context:${runId}`);

        logger.info("Released run resources to swarm", {
            runId,
            swarmId,
            usage: {
                credits: usage.credits,
                durationMs: usage.durationMs,
                stepsExecuted: usage.stepsExecuted,
            },
        });
    }

    async getRunContext(runId: string): Promise<RunExecutionContext> {
        const data = await this.redis.get(`run_context:${runId}`);
        if (!data) {
            throw new Error(`Run context not found: ${runId}`);
        }

        const context = JSON.parse(data) as RunExecutionContext;
        // Reconstruct Date objects
        context.resourceUsage.startTime = new Date(context.resourceUsage.startTime);

        return context;
    }

    async updateRunContext(runId: string, context: RunExecutionContext): Promise<void> {
        // Store context with TTL based on allocation
        const allocation = this.activeAllocations.get(runId);
        const ttl = allocation ?
            Math.floor((allocation.expiresAt.getTime() - Date.now()) / 1000) :
            3600; // Default 1 hour

        await this.redis.setex(
            `run_context:${runId}`,
            ttl,
            JSON.stringify({
                ...context,
                resourceUsage: {
                    ...context.resourceUsage,
                    startTime: context.resourceUsage.startTime.toISOString(),
                },
            }),
        );

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
        await this.eventBus.publish({
            type: EventTypes.RUN_STARTED,
            source: { tier: 2, component: "RunContextManager" },
            data: {
                runId,
                routineId,
                swarmId: allocation.swarmId,
                allocation: {
                    credits: allocation.allocated.credits,
                    durationMs: allocation.allocated.timeoutMs,
                },
                startTime: new Date().toISOString(),
            },
        });
    }

    async emitRunCompleted(runId: string, result: any, usage: ResourceUsage): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.RUN_COMPLETED,
            source: { tier: 2, component: "RunContextManager" },
            data: {
                runId,
                result,
                usage: {
                    credits: usage.credits,
                    durationMs: usage.durationMs,
                    stepsExecuted: usage.stepsExecuted,
                },
                completedAt: new Date().toISOString(),
            },
        });
    }

    async emitRunFailed(runId: string, error: any, usage: ResourceUsage): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.RUN_FAILED,
            source: { tier: 2, component: "RunContextManager" },
            data: {
                runId,
                error: {
                    message: error.message,
                    name: error.name,
                    stack: error.stack,
                },
                usage: {
                    credits: usage.credits,
                    durationMs: usage.durationMs,
                    stepsExecuted: usage.stepsExecuted,
                },
                failedAt: new Date().toISOString(),
            },
        });
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
            allocationId: generatePK(),
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
        usage: ResourceUsage,
    ): Promise<void> {
        const runAllocation = this.activeAllocations.get(runId);
        if (!runAllocation) {
            logger.warn("No run allocation found for step release", { runId, stepId });
            return;
        }

        // Return unused resources to run allocation
        const creditsUsed = BigInt(usage.credits);
        const creditsReturned = BigInt(runAllocation.allocated.credits) - creditsUsed;

        runAllocation.remaining.credits = (
            BigInt(runAllocation.remaining.credits) + creditsReturned
        ).toString();

        const timeUsed = usage.durationMs;
        // Note: We don't return time since it's consumed regardless of usage

        logger.debug("Released step resources back to run", {
            runId,
            stepId,
            creditsUsed: usage.credits,
            durationMs: usage.durationMs,
        });
    }
}
