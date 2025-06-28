import {
    type AvailableResources,
    type BudgetReservation,
    type ExecutionConstraints,
    type ExecutionContext,
    type ExecutionResourceUsage,
    type RoutineId,
    type StepId,
    generatePK,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { type IEventBus } from "../../../events/types.js";
import { RunResourceAdapter } from "../../cross-cutting/resources/adapters.js";
import { type ResourceAmount } from "../../cross-cutting/resources/resourceManager.js";
import { BaseTierResourceManager } from "../../shared/BaseTierResourceManager.js";

/**
 * Resource allocation for a routine run
 */
interface RunResourceAllocation {
    runId: string;
    routineId: RoutineId;
    parentSwarmId?: string;

    // Reserved from parent (Tier 1)
    reserved: {
        credits: number;
        maxDurationMs: number;
        maxMemoryMB: number;
        maxConcurrentSteps: number;
        toolPermissions: string[];
    };

    // Currently allocated to child steps
    allocated: Map<StepId, StepResourceAllocation>;

    // Actual usage tracking
    usage: ExecutionResourceUsage;

    // Timestamps
    createdAt: Date;
    updatedAt: Date;
    completedAt?: Date;
}

/**
 * Resource allocation for a single step
 */
interface StepResourceAllocation {
    stepId: StepId;
    reserved: BudgetReservation;
    actualUsage?: ExecutionResourceUsage;
    status: "reserved" | "active" | "completed" | "failed";
    createdAt: Date;
    completedAt?: Date;
}

/**
 * Tier 2 ResourceManager
 * 
 * This component extends BaseTierResourceManager to provide run-specific
 * resource management functionality using the RunResourceAdapter.
 * 
 * Complex behaviors like path optimization and branch balancing emerge from
 * resource agents analyzing the events emitted by the unified manager.
 */
export class ResourceManager extends BaseTierResourceManager<RunResourceAdapter> {
    constructor(logger: Logger, eventBus: IEventBus) {
        super(logger, eventBus, 2);
    }

    /**
     * Create the run resource adapter
     */
    protected createAdapter(): RunResourceAdapter {
        return new RunResourceAdapter(this.unifiedManager);
    }

    /**
     * Reserve resources for a routine run from Tier 1
     * Maps legacy interface to unified manager
     */
    async reserveRunResources(
        context: ExecutionContext,
        parentAllocation: AvailableResources,
        constraints: ExecutionConstraints,
    ): Promise<RunResourceAllocation> {
        return this.withErrorHandling("reserve run resources", async () => {
            const runId = context.executionId;
            const routineId = context.routineId || generatePK();

            // Use adapter to reserve resources
            const allocation = await this.adapter.reserveForRun(
                runId,
                {
                    credits: Math.min(
                        parentAllocation.credits,
                        constraints.maxCost || parentAllocation.credits,
                    ),
                    time: Math.min(
                        constraints.maxTime || Number.MAX_SAFE_INTEGER,
                        constraints.maxExecutionTime || Number.MAX_SAFE_INTEGER,
                    ),
                    memory: parentAllocation.memoryMB * 1024 * 1024, // Convert MB to bytes
                    tokens: parentAllocation.credits * 10, // Estimate tokens from credits
                    apiCalls: 100, // Default API calls
                },
                context.parentSwarmId,
            );

            // Convert to legacy format
            const reserved = {
                credits: allocation.resources.credits || 0,
                maxDurationMs: allocation.resources.time || 0,
                maxMemoryMB: 2048, // Default 2GB per routine
                maxConcurrentSteps: 10, // Default concurrency limit
                toolPermissions: parentAllocation.tools.map(t => t.name),
            };

            // Return simplified allocation
            return {
                runId,
                routineId,
                parentSwarmId: context.parentSwarmId,
                reserved,
                allocated: new Map(),
                usage: {
                    creditsUsed: "0",
                    durationMs: 0,
                    memoryUsedMB: 0,
                    stepsExecuted: 0,
                },
                createdAt: new Date(),
                updatedAt: new Date(),
            };
        });
    }

    /**
     * Allocate resources to a step (for Tier 3)
     * Uses adapter to distribute resources
     */
    async allocateStepResources(
        runId: string,
        stepId: StepId,
        requirements: ExecutionConstraints,
    ): Promise<BudgetReservation> {
        return this.withErrorHandling("allocate step resources", async () => {
            // Use adapter to distribute resources to steps
            const stepAllocations = await this.adapter.distributeToSteps(
                runId,
                [stepId],
                "equal",
            );

            const allocation = stepAllocations.get(stepId);
            if (!allocation) {
                throw new Error("Failed to allocate resources for step");
            }

            // Convert to BudgetReservation format
            return {
                id: allocation.id,
                credits: allocation.resources.credits || 0,
                timeLimit: allocation.resources.time || 300000,
                memoryLimit: (allocation.resources.memory || 0) / (1024 * 1024), // Convert bytes to MB
                allocated: true,
                metadata: {
                    runId,
                    stepId,
                },
            };
        });
    }

    /**
     * Update step resource usage (from Tier 3)
     * Uses unified manager to track usage
     */
    async updateStepUsage(
        runId: string,
        stepId: StepId,
        usage: ExecutionResourceUsage,
        status: "active" | "completed" | "failed",
    ): Promise<void> {
        // Track usage for both run and step
        const resourceUsage: ResourceAmount = {
            credits: Number(usage.creditsUsed),
            time: usage.durationMs,
            memory: usage.memoryUsedMB * 1024 * 1024, // Convert MB to bytes
            tokens: 0, // Not tracked in legacy format
            apiCalls: 0, // Not tracked in legacy format
        };

        await this.unifiedManager.trackUsage(runId, resourceUsage);
        await this.unifiedManager.trackUsage(stepId, resourceUsage);
    }

    /**
     * Complete run and return unused resources to Tier 1
     * Uses adapter to return unused resources
     */
    async completeRun(runId: string): Promise<ExecutionResourceUsage> {
        // Return unused resources
        const unused = await this.adapter.returnUnused(runId);

        // Get final usage from unified manager
        const usage = this.unifiedManager.getUsage(runId);

        return {
            creditsUsed: String(usage.credits || 0),
            durationMs: usage.time || 0,
            memoryUsedMB: (usage.memory || 0) / (1024 * 1024),
            stepsExecuted: 0, // Not tracked by unified manager
        };
    }

    /**
     * Get current resource status for a run
     */
    async getRunStatus(runId: string): Promise<{
        reserved: ResourceAmount;
        used: ResourceAmount;
        available: ResourceAmount;
    }> {
        const allocations = this.unifiedManager.getAllocations(runId);
        const usage = this.unifiedManager.getUsage(runId);

        const allocated = allocations[0]?.resources || {};
        const available: ResourceAmount = {
            credits: (allocated.credits || 0) - (usage.credits || 0),
            time: (allocated.time || 0) - (usage.time || 0),
            memory: (allocated.memory || 0) - (usage.memory || 0),
            tokens: (allocated.tokens || 0) - (usage.tokens || 0),
            apiCalls: (allocated.apiCalls || 0) - (usage.apiCalls || 0),
        };

        return { reserved: allocated, used: usage, available };
    }

    /**
     * Clean up and shutdown
     */
    async shutdown(): Promise<void> {
        await this.cleanup();
    }
}
