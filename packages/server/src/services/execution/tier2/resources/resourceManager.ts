import { type Logger } from "winston";
import {
    type ExecutionContext,
    type AvailableResources,
    type ExecutionConstraints,
    type ExecutionResourceUsage,
    type BudgetReservation,
    type StepId,
    type RoutineId,
    generatePk,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";

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
 * Manages resources for routine runs, implementing the reserve-and-return pattern:
 * 1. Receives resource allocation from Tier 1 (Swarm level)
 * 2. Distributes resources to individual steps via Tier 3
 * 3. Tracks aggregate usage across all steps
 * 4. Returns unused resources to Tier 1 upon completion
 * 
 * Key responsibilities:
 * - Manage resource distribution to parallel branches
 * - Enforce routine-level resource limits
 * - Aggregate step-level usage for reporting
 * - Handle resource allocation for sub-routines
 */
export class ResourceManager {
    private readonly logger: Logger;
    private readonly eventBus?: EventBus;
    private readonly allocations: Map<string, RunResourceAllocation> = new Map();
    
    constructor(logger: Logger, eventBus?: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
    }
    
    /**
     * Reserve resources for a routine run from Tier 1
     */
    async reserveRunResources(
        context: ExecutionContext,
        parentAllocation: AvailableResources,
        constraints: ExecutionConstraints,
    ): Promise<RunResourceAllocation> {
        const runId = context.executionId;
        const routineId = context.routineId || generatePk();
        
        // Calculate resource reservation based on routine requirements
        const reserved = {
            credits: Math.min(
                parentAllocation.credits,
                constraints.maxCost || parentAllocation.credits,
            ),
            maxDurationMs: Math.min(
                constraints.maxTime || Number.MAX_SAFE_INTEGER,
                constraints.maxExecutionTime || Number.MAX_SAFE_INTEGER,
            ),
            maxMemoryMB: 2048, // Default 2GB per routine
            maxConcurrentSteps: 10, // Default concurrency limit
            toolPermissions: parentAllocation.tools.map(t => t.name),
        };
        
        const allocation: RunResourceAllocation = {
            runId,
            routineId,
            parentSwarmId: context.swarmId,
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
        
        this.allocations.set(runId, allocation);
        
        // Emit reservation event
        if (this.eventBus) {
            await this.eventBus.publish("run.resources.reserved", {
                runId,
                routineId,
                reserved,
                timestamp: new Date(),
            });
        }
        
        this.logger.debug("[Tier2 ResourceManager] Reserved run resources", {
            runId,
            routineId,
            reserved,
        });
        
        return allocation;
    }
    
    /**
     * Allocate resources to a step (for Tier 3)
     */
    async allocateStepResources(
        runId: string,
        stepId: StepId,
        requirements: ExecutionConstraints,
    ): Promise<BudgetReservation> {
        const allocation = this.allocations.get(runId);
        if (!allocation) {
            throw new Error(`No resource allocation found for run ${runId}`);
        }
        
        // Check if we have enough resources
        const currentUsage = this.calculateCurrentUsage(allocation);
        const availableCredits = allocation.reserved.credits - Number(currentUsage.creditsUsed);
        
        if (availableCredits <= 0) {
            throw new Error("Insufficient credits for step execution");
        }
        
        // Check concurrent step limit
        const activeSteps = Array.from(allocation.allocated.values())
            .filter(s => s.status === "active").length;
        
        if (activeSteps >= allocation.reserved.maxConcurrentSteps) {
            throw new Error("Maximum concurrent steps reached");
        }
        
        // Create step allocation
        const stepReservation: BudgetReservation = {
            id: generatePk(),
            credits: Math.min(availableCredits, requirements.maxCost || availableCredits / 10),
            timeLimit: requirements.maxTime || 300000, // 5 min default
            memoryLimit: 512, // 512MB per step default
            allocated: true,
            metadata: {
                runId,
                stepId,
                routineId: allocation.routineId,
            },
        };
        
        const stepAllocation: StepResourceAllocation = {
            stepId,
            reserved: stepReservation,
            status: "reserved",
            createdAt: new Date(),
        };
        
        allocation.allocated.set(stepId, stepAllocation);
        allocation.updatedAt = new Date();
        
        // Emit allocation event
        if (this.eventBus) {
            await this.eventBus.publish("run.step.allocated", {
                runId,
                stepId,
                reservation: stepReservation,
                timestamp: new Date(),
            });
        }
        
        this.logger.debug("[Tier2 ResourceManager] Allocated step resources", {
            runId,
            stepId,
            reservation: stepReservation,
        });
        
        return stepReservation;
    }
    
    /**
     * Update step resource usage (from Tier 3)
     */
    async updateStepUsage(
        runId: string,
        stepId: StepId,
        usage: ExecutionResourceUsage,
        status: "active" | "completed" | "failed",
    ): Promise<void> {
        const allocation = this.allocations.get(runId);
        if (!allocation) {
            throw new Error(`No resource allocation found for run ${runId}`);
        }
        
        const stepAllocation = allocation.allocated.get(stepId);
        if (!stepAllocation) {
            throw new Error(`No step allocation found for step ${stepId}`);
        }
        
        // Update step allocation
        stepAllocation.actualUsage = usage;
        stepAllocation.status = status;
        if (status === "completed" || status === "failed") {
            stepAllocation.completedAt = new Date();
        }
        
        // Update run-level usage
        allocation.usage = this.calculateCurrentUsage(allocation);
        allocation.updatedAt = new Date();
        
        // Emit usage update event
        if (this.eventBus) {
            await this.eventBus.publish("run.step.usage", {
                runId,
                stepId,
                usage,
                status,
                timestamp: new Date(),
            });
        }
        
        this.logger.debug("[Tier2 ResourceManager] Updated step usage", {
            runId,
            stepId,
            usage,
            status,
        });
    }
    
    /**
     * Complete run and return unused resources to Tier 1
     */
    async completeRun(runId: string): Promise<ExecutionResourceUsage> {
        const allocation = this.allocations.get(runId);
        if (!allocation) {
            throw new Error(`No resource allocation found for run ${runId}`);
        }
        
        // Calculate final usage
        const finalUsage = this.calculateCurrentUsage(allocation);
        allocation.usage = finalUsage;
        allocation.completedAt = new Date();
        allocation.updatedAt = new Date();
        
        // Calculate returned resources
        const returned = {
            credits: allocation.reserved.credits - Number(finalUsage.creditsUsed),
            duration: allocation.reserved.maxDurationMs - finalUsage.durationMs,
        };
        
        // Emit completion event
        if (this.eventBus) {
            await this.eventBus.publish("run.resources.completed", {
                runId,
                routineId: allocation.routineId,
                finalUsage,
                returned,
                timestamp: new Date(),
            });
        }
        
        this.logger.info("[Tier2 ResourceManager] Completed run", {
            runId,
            finalUsage,
            returned,
        });
        
        // Clean up allocation after some time
        setTimeout(() => {
            this.allocations.delete(runId);
        }, 300000); // Keep for 5 minutes for debugging
        
        return finalUsage;
    }
    
    /**
     * Get current resource state for a run
     */
    async getRunResourceState(runId: string): Promise<RunResourceAllocation | null> {
        return this.allocations.get(runId) || null;
    }
    
    /**
     * Handle resource exhaustion scenarios
     */
    async handleResourceExhaustion(
        runId: string,
        resourceType: "credits" | "time" | "memory",
    ): Promise<void> {
        const allocation = this.allocations.get(runId);
        if (!allocation) {
            return;
        }
        
        // Mark all active steps as failed
        for (const [stepId, stepAlloc] of allocation.allocated) {
            if (stepAlloc.status === "active") {
                stepAlloc.status = "failed";
                stepAlloc.completedAt = new Date();
            }
        }
        
        // Emit exhaustion event
        if (this.eventBus) {
            await this.eventBus.publish("run.resources.exhausted", {
                runId,
                routineId: allocation.routineId,
                resourceType,
                usage: allocation.usage,
                timestamp: new Date(),
            });
        }
        
        this.logger.error("[Tier2 ResourceManager] Resource exhausted", {
            runId,
            resourceType,
            usage: allocation.usage,
        });
    }
    
    /**
     * Calculate current aggregate usage across all steps
     */
    private calculateCurrentUsage(allocation: RunResourceAllocation): ExecutionResourceUsage {
        let totalCredits = 0;
        let maxDuration = 0;
        let peakMemory = 0;
        let stepsExecuted = 0;
        
        for (const stepAlloc of allocation.allocated.values()) {
            if (stepAlloc.actualUsage) {
                totalCredits += Number(stepAlloc.actualUsage.creditsUsed);
                maxDuration = Math.max(maxDuration, stepAlloc.actualUsage.durationMs);
                peakMemory = Math.max(peakMemory, stepAlloc.actualUsage.memoryUsedMB);
                
                if (stepAlloc.status === "completed") {
                    stepsExecuted++;
                }
            }
        }
        
        // Calculate actual run duration
        const runDuration = allocation.completedAt
            ? allocation.completedAt.getTime() - allocation.createdAt.getTime()
            : Date.now() - allocation.createdAt.getTime();
        
        return {
            creditsUsed: totalCredits.toString(),
            durationMs: runDuration,
            memoryUsedMB: peakMemory,
            stepsExecuted,
            tokens: 0, // Aggregated from steps if available
            apiCalls: 0, // Aggregated from steps if available
        };
    }
    
    /**
     * Distribute resources among parallel branches
     */
    async distributeToParallelBranches(
        runId: string,
        branchCount: number,
        strategy: "equal" | "weighted" = "equal",
        weights?: number[],
    ): Promise<BudgetReservation[]> {
        const allocation = this.allocations.get(runId);
        if (!allocation) {
            throw new Error(`No resource allocation found for run ${runId}`);
        }
        
        const currentUsage = this.calculateCurrentUsage(allocation);
        const availableCredits = allocation.reserved.credits - Number(currentUsage.creditsUsed);
        
        const branchReservations: BudgetReservation[] = [];
        
        if (strategy === "equal") {
            const creditsPerBranch = availableCredits / branchCount;
            for (let i = 0; i < branchCount; i++) {
                branchReservations.push({
                    id: generatePk(),
                    credits: creditsPerBranch,
                    timeLimit: allocation.reserved.maxDurationMs,
                    memoryLimit: allocation.reserved.maxMemoryMB / branchCount,
                    allocated: true,
                    metadata: { runId, branchIndex: i },
                });
            }
        } else if (strategy === "weighted" && weights) {
            const totalWeight = weights.reduce((sum, w) => sum + w, 0);
            for (let i = 0; i < branchCount; i++) {
                const weight = weights[i] || 1;
                branchReservations.push({
                    id: generatePk(),
                    credits: (availableCredits * weight) / totalWeight,
                    timeLimit: allocation.reserved.maxDurationMs,
                    memoryLimit: (allocation.reserved.maxMemoryMB * weight) / totalWeight,
                    allocated: true,
                    metadata: { runId, branchIndex: i, weight },
                });
            }
        }
        
        this.logger.debug("[Tier2 ResourceManager] Distributed to branches", {
            runId,
            branchCount,
            strategy,
            totalCredits: availableCredits,
        });
        
        return branchReservations;
    }
}