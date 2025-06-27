/**
 * Tier-Specific Resource Adapters - Hierarchical Resource Management
 * 
 * These adapters implement the adapter pattern to add tier-specific behavior to the
 * unified ResourceManager without duplicating core allocation and tracking logic.
 * Each adapter focuses on the unique resource management needs of its tier while
 * maintaining consistency with the overall resource management philosophy.
 * 
 * ## Adapter Pattern Benefits
 * 
 * - **Single Responsibility**: Each adapter handles only its tier's specific needs
 * - **Code Reuse**: Core allocation logic remains in the unified manager
 * - **Consistent Events**: All adapters emit the same resource event types
 * - **Easy Testing**: Tier-specific behavior can be tested in isolation
 * - **Emergent Intelligence**: All adapters feed into the same event stream for AI analysis
 * 
 * ## Hierarchical Resource Flow
 * 
 * ```
 * SwarmResourceAdapter (Tier 1)
 *   ├── Swarm-level budgets and strategic allocation
 *   └── Team-level resource distribution
 *       ↓
 * RunResourceAdapter (Tier 2)  
 *   ├── Run-level execution resources
 *   └── Step-level resource distribution
 *       ↓
 * StepResourceAdapter (Tier 3)
 *   ├── User-specific tracking and limits
 *   └── Burst capacity management
 * ```
 * 
 * ## Emergent Optimization Philosophy
 * 
 * These adapters provide basic allocation strategies (equal distribution, simple budgeting)
 * while complex optimization emerges from AI agents analyzing the resource events.
 * Agents can propose improvements like:
 * - Dynamic resource reallocation based on step complexity
 * - Predictive burst capacity allocation
 * - Cost-optimized team composition strategies
 * - Performance-driven step resource tuning
 */

import { type ResourceManager, type ResourceAllocation, type ResourceAmount } from "./resourceManager.js";
import { logger } from "../../../../events/logger.js";
import type { SwarmResources } from "@vrooli/shared";

/**
 * Tier 1 Adapter - Swarm-level resource management
 * 
 * Adds hierarchical allocation for swarms and teams.
 * Optimization and prediction emerge from agent analysis.
 */
export class SwarmResourceAdapter {
    constructor(private readonly resourceManager: ResourceManager) {}

    /**
     * Allocates resources to a swarm with hierarchical budgets
     */
    async allocateToSwarm(
        swarmId: string,
        budget: SwarmResources,
    ): Promise<ResourceAllocation> {
        return await this.resourceManager.allocate(
            swarmId,
            "swarm",
            {
                credits: budget.totalBudget,
                time: budget.timeLimit,
                memory: budget.memoryLimit,
            },
        );
    }

    /**
     * Allocates resources to a team within a swarm
     */
    async allocateToTeam(
        teamId: string,
        swarmAllocationId: string,
        teamBudget: ResourceAmount,
    ): Promise<ResourceAllocation> {
        return await this.resourceManager.allocate(
            teamId,
            "team",
            teamBudget,
            swarmAllocationId,
        );
    }

    /**
     * Gets swarm resource summary
     */
    async getSwarmSummary(swarmId: string): Promise<{
        allocated: ResourceAmount;
        used: ResourceAmount;
        available: ResourceAmount;
    }> {
        const allocations = this.resourceManager.getAllocations(swarmId);
        const usage = this.resourceManager.getUsage(swarmId);
        
        const allocated = allocations[0]?.resources || {};
        const available: ResourceAmount = {
            credits: (allocated.credits || 0) - (usage.credits || 0),
            time: (allocated.time || 0) - (usage.time || 0),
            memory: (allocated.memory || 0) - (usage.memory || 0),
            tokens: (allocated.tokens || 0) - (usage.tokens || 0),
            apiCalls: (allocated.apiCalls || 0) - (usage.apiCalls || 0),
        };

        return { allocated, used: usage, available };
    }
}

/**
 * Tier 2 Adapter - Run-level resource management
 * 
 * Adds step distribution and branch handling.
 * Path optimization emerges from agent analysis.
 */
export class RunResourceAdapter {
    constructor(private readonly resourceManager: ResourceManager) {}

    /**
     * Reserves resources for a run
     */
    async reserveForRun(
        runId: string,
        requested: ResourceAmount,
        swarmAllocationId?: string,
    ): Promise<ResourceAllocation> {
        return await this.resourceManager.allocate(
            runId,
            "run",
            requested,
            swarmAllocationId,
        );
    }

    /**
     * Distributes resources to parallel branches
     */
    async distributeToSteps(
        runId: string,
        stepIds: string[],
        strategy: "equal" | "weighted" = "equal",
    ): Promise<Map<string, ResourceAllocation>> {
        const runAllocations = this.resourceManager.getAllocations(runId);
        if (runAllocations.length === 0) {
            throw new Error(`No allocation found for run ${runId}`);
        }

        const runAllocation = runAllocations[0];
        const stepAllocations = new Map<string, ResourceAllocation>();

        // Simple equal distribution
        const perStepResources: ResourceAmount = {
            credits: (runAllocation.resources.credits || 0) / stepIds.length,
            tokens: (runAllocation.resources.tokens || 0) / stepIds.length,
            apiCalls: (runAllocation.resources.apiCalls || 0) / stepIds.length,
            time: runAllocation.resources.time, // Time is shared
            memory: runAllocation.resources.memory, // Memory is shared
        };

        for (const stepId of stepIds) {
            const allocation = await this.resourceManager.allocate(
                stepId,
                "step",
                perStepResources,
                runAllocation.id,
            );
            stepAllocations.set(stepId, allocation);
        }

        return stepAllocations;
    }

    /**
     * Returns unused resources from run
     */
    async returnUnused(runId: string): Promise<ResourceAmount> {
        const allocations = this.resourceManager.getAllocations(runId);
        const usage = this.resourceManager.getUsage(runId);

        const unused: ResourceAmount = {};
        for (const allocation of allocations) {
            unused.credits = (allocation.resources.credits || 0) - (usage.credits || 0);
            unused.tokens = (allocation.resources.tokens || 0) - (usage.tokens || 0);
            unused.apiCalls = (allocation.resources.apiCalls || 0) - (usage.apiCalls || 0);
            
            await this.resourceManager.release(allocation.id);
        }

        logger.info("[RunResourceAdapter] Returned unused resources", {
            runId,
            unused,
        });

        return unused;
    }
}

/**
 * Tier 3 Adapter - Step-level resource management
 * 
 * Adds user-specific tracking and burst handling.
 * Efficiency optimization emerges from agent analysis.
 */
export class StepResourceAdapter {
    constructor(private readonly resourceManager: ResourceManager) {}

    /**
     * Checks if user can execute step
     */
    async canUserExecute(
        userId: string,
        stepRequirements: ResourceAmount,
    ): Promise<boolean> {
        return await this.resourceManager.canUse(userId, stepRequirements);
    }

    /**
     * Tracks step execution for user
     */
    async trackStepExecution(
        userId: string,
        stepId: string,
        used: ResourceAmount,
    ): Promise<void> {
        // Track for both user and step
        await this.resourceManager.trackUsage(userId, used);
        await this.resourceManager.trackUsage(stepId, used);
    }

    /**
     * Gets user resource summary
     */
    getUserSummary(userId: string): {
        usage: ResourceAmount;
        allocations: ResourceAllocation[];
    } {
        return {
            usage: this.resourceManager.getUsage(userId),
            allocations: this.resourceManager.getAllocations(userId),
        };
    }

    /**
     * Allocates burst capacity for user
     */
    async allocateBurst(
        userId: string,
        burstAmount: ResourceAmount,
        duration: number, // milliseconds
    ): Promise<ResourceAllocation> {
        const allocation = await this.resourceManager.allocate(
            userId,
            "user",
            burstAmount,
        );

        // Set expiration
        allocation.expires = new Date(Date.now() + duration);

        return allocation;
    }
}
