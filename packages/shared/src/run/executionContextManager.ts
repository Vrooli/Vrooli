import { type ExecutionContext, type ExecutionLimits, type ExecutionResult, type ExecutionStatus, type RoutineContext, type SwarmContext } from "./executor.js";
import { BranchStatus, type BranchProgress, type RunConfig, type RunProgress, type RunTriggeredBy, type SubroutineContext, type SubroutineIOMapping } from "./types.js";

// API_CREDITS_MULTIPLIER from shared would be ideal, but we'll use a default
const API_CREDITS_MULTIPLIER = BigInt(1000000); // 1M credits per dollar
const DEFAULT_CREDIT_DOLLARS = 10; // $10 default limit
const DEFAULT_MAX_CREDITS = API_CREDITS_MULTIPLIER * BigInt(DEFAULT_CREDIT_DOLLARS);
const DEFAULT_MAX_TIME_MS = 300000; // 5 minutes
const DEFAULT_MAX_TOOL_CALLS = 50;
const DEFAULT_MAX_REASONING_STEPS = 10;

/**
 * Manages the creation and integration of execution contexts for the unified execution system.
 * Provides seamless conversion between routine contexts and execution contexts.
 */
export class ExecutionContextManager {
    /**
     * Creates a RoutineContext from RunStateMachine data that integrates with ExecutionContext.
     * 
     * @param run The current run progress
     * @param branch The current branch being executed
     * @param subroutineContext The subroutine context data
     * @param swarmContext Optional swarm context for swarm integration
     * @returns RoutineContext that extends ExecutionContext
     */
    static createRoutineContext(
        run: RunProgress,
        branch: BranchProgress,
        subroutineContext: SubroutineContext,
        swarmContext?: SwarmContext,
    ): RoutineContextImpl {
        const currentLocation = branch.locationStack[branch.locationStack.length - 1];
        if (!currentLocation) {
            throw new Error("Branch has no current location");
        }

        // Build execution limits from run config
        const limits: ExecutionLimits = this.buildExecutionLimits(run.config, swarmContext);

        // Create compatible userData from run owner
        const userData: RunTriggeredBy = {
            id: run.owner.id,
            // Default values for missing properties - these would normally come from the session
            hasPremium: false,
            languages: ["en"],
            // Add timeZone if it exists in the owner data
            ...(run.owner as any).timeZone ? { timeZone: (run.owner as any).timeZone } : {},
        };

        // Create the base execution context data
        const baseContext: Omit<ExecutionContext, "routine" | "ioMapping"> = {
            subroutineInstanceId: branch.subroutineInstanceId,
            currentLocation,
            parentSwarmContext: swarmContext,
            userData,
            config: run.config,
            limits,
            status: this.mapBranchStatusToExecutionStatus(branch.status),
            startedAt: branch.nodeStartTimeMs ? new Date(branch.nodeStartTimeMs) : new Date(),
            executingBotId: swarmContext?.swarmLeader,
        };

        // Create routine context with accumulated data
        return new RoutineContextImpl(
            baseContext,
            run.runId,
            run,
            branch,
            branch.subroutineInstanceId,
            subroutineContext,
        );
    }

    /**
     * Updates a SubroutineContext with execution results.
     * 
     * @param context The routine context
     * @param result The execution result
     * @returns Updated SubroutineContext
     */
    static updateSubroutineContext(
        context: RoutineContextImpl,
        result: ExecutionResult,
    ): SubroutineContext {
        const updatedContext = { ...context.getSubroutineContext() };

        // Update inputs and outputs
        if (result.ioMapping?.inputs) {
            for (const [key, input] of Object.entries(result.ioMapping.inputs)) {
                if (input.value !== undefined) {
                    updatedContext.allInputsMap[key] = input.value;
                    updatedContext.allInputsList.push({ key, value: input.value });
                }
            }
        }

        if (result.ioMapping?.outputs) {
            for (const [key, output] of Object.entries(result.ioMapping.outputs)) {
                if (output.value !== undefined) {
                    updatedContext.allOutputsMap[key] = output.value;
                    updatedContext.allOutputsList.push({ key, value: output.value });
                }
            }
        }

        return updatedContext;
    }

    /**
     * Builds execution limits from run configuration and optional swarm context.
     */
    private static buildExecutionLimits(config: RunConfig, swarmContext?: SwarmContext): ExecutionLimits {
        // Start with defaults
        const limits: ExecutionLimits = {
            maxCredits: DEFAULT_MAX_CREDITS,
            maxTimeMs: DEFAULT_MAX_TIME_MS,
            maxToolCalls: DEFAULT_MAX_TOOL_CALLS,
            maxReasoningSteps: DEFAULT_MAX_REASONING_STEPS,
            strictLimits: true,
        };

        // Apply run config limits
        if (config.limits.maxCredits) {
            limits.maxCredits = BigInt(config.limits.maxCredits);
        }
        if (config.limits.maxTime) {
            limits.maxTimeMs = config.limits.maxTime;
        }

        // Apply swarm limits if available and more restrictive
        if (swarmContext?.chatConfig?.limits) {
            const swarmLimits = swarmContext.chatConfig.limits;
            if (swarmLimits.maxCredits && BigInt(swarmLimits.maxCredits) < limits.maxCredits) {
                limits.maxCredits = BigInt(swarmLimits.maxCredits);
            }
            if (swarmLimits.maxDurationMs && swarmLimits.maxDurationMs < limits.maxTimeMs) {
                limits.maxTimeMs = swarmLimits.maxDurationMs;
            }
        }

        return limits;
    }

    /**
     * Maps BranchStatus to ExecutionStatus.
     */
    private static mapBranchStatusToExecutionStatus(branchStatus: BranchStatus): ExecutionStatus {
        switch (branchStatus) {
            case BranchStatus.Active:
                return "running" as ExecutionStatus;
            case BranchStatus.Completed:
                return "completed" as ExecutionStatus;
            case BranchStatus.Failed:
                return "failed" as ExecutionStatus;
            case BranchStatus.Waiting:
                return "pending" as ExecutionStatus;
            default:
                return "pending" as ExecutionStatus;
        }
    }
}

/**
 * Implementation of RoutineContext that extends ExecutionContext.
 * Provides routine-specific data while maintaining compatibility with the unified execution system.
 */
export class RoutineContextImpl implements ExecutionContext {
    // ExecutionContext properties
    public readonly subroutineInstanceId: string;
    public readonly routine: any; // Will be set when we have the routine
    public readonly ioMapping: SubroutineIOMapping = { inputs: {}, outputs: {} }; // Will be updated
    public readonly currentLocation: any;
    public readonly parentSwarmContext?: SwarmContext;
    public readonly parentRoutineContext?: RoutineContext;
    public readonly userData: RunTriggeredBy;
    public readonly config: RunConfig;
    public readonly limits: ExecutionLimits;
    public status: ExecutionStatus;
    public readonly startedAt: Date;
    public readonly executingBotId?: string;

    // Routine-specific properties
    public readonly runId: string;
    public readonly runProgress: RunProgress;
    public readonly currentBranch: BranchProgress;
    private readonly subroutineContext: SubroutineContext;

    constructor(
        baseContext: Omit<ExecutionContext, "routine" | "ioMapping">,
        runId: string,
        runProgress: RunProgress,
        currentBranch: BranchProgress,
        subroutineInstanceId: string,
        subroutineContext: SubroutineContext,
    ) {
        // Initialize ExecutionContext properties from baseContext
        this.subroutineInstanceId = baseContext.subroutineInstanceId;
        this.currentLocation = baseContext.currentLocation;
        this.parentSwarmContext = baseContext.parentSwarmContext;
        this.userData = baseContext.userData;
        this.config = baseContext.config;
        this.limits = baseContext.limits;
        this.status = baseContext.status;
        this.startedAt = baseContext.startedAt;
        this.executingBotId = baseContext.executingBotId;

        // Set routine-specific properties
        this.runId = runId;
        this.runProgress = runProgress;
        this.currentBranch = currentBranch;
        this.subroutineContext = subroutineContext;
    }

    /**
     * Sets the routine and IO mapping for this context.
     */
    setRoutineAndIO(routine: any, ioMapping: SubroutineIOMapping): void {
        (this as any).routine = routine;
        (this as any).ioMapping = ioMapping;
    }

    /**
     * Gets the routine-specific context data.
     */
    getSubroutineContext(): SubroutineContext {
        return this.subroutineContext;
    }

    /**
     * Gets branch progress information.
     */
    getBranchProgress(): {
        branchId: string;
        status: BranchStatus;
        creditsSpent: string;
        nodeStartTimeMs: number | null;
    } {
        return {
            branchId: this.currentBranch.branchId,
            status: this.currentBranch.status,
            creditsSpent: this.currentBranch.creditsSpent,
            nodeStartTimeMs: this.currentBranch.nodeStartTimeMs,
        };
    }

    /**
     * Updates the branch status.
     */
    updateBranchStatus(status: BranchStatus): void {
        this.currentBranch.status = status;
        this.status = ExecutionContextManager["mapBranchStatusToExecutionStatus"](status);
    }

    /**
     * Adds a step result to the routine progress.
     */
    addStepResult(stepId: string, result: ExecutionResult): void {
        // In a full implementation, this would update the run progress
        // For now, we'll just update the context
        const updatedContext = ExecutionContextManager.updateSubroutineContext(this, result);
        Object.assign(this.subroutineContext, updatedContext);
    }
} 
