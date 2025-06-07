import { API_CREDITS_MULTIPLIER, type BlackboardItem, type ChatConfigObject, type ResourceVersion, type RunConfig, type SessionUser, type SubroutineIOMapping, type SwarmResource, type SwarmSubTask, type TeamConfigObject, type ToolCallRecord } from "@vrooli/shared";

// Constants
const DEFAULT_CREDIT_DOLLARS = 10; // $10 default limit
const DEFAULT_MAX_CREDITS = API_CREDITS_MULTIPLIER * BigInt(DEFAULT_CREDIT_DOLLARS);
const DEFAULT_MAX_TIME_MS = 300000; // 5 minutes
const DEFAULT_MAX_TOOL_CALLS = 50;
const DEFAULT_MAX_REASONING_STEPS = 10;

/**
 * Core execution context that contains all data needed to execute a subroutine.
 * This context can be inherited from parent swarms or routines.
 */
export interface ExecutionContext {
    // Core execution data
    /** Unique identifier for this subroutine instance */
    subroutineInstanceId: string;
    /** The routine/subroutine to execute */
    routine: ResourceVersion;
    /** Input/output mapping for the subroutine */
    ioMapping: SubroutineIOMapping;

    // Parent context (for inheritance)
    /** Context from parent swarm if this routine was initiated by a swarm */
    parentSwarmContext?: SwarmContext;
    /** Context from parent routine if this is a nested subroutine */
    parentRoutineContext?: RoutineContext;

    // Execution environment
    /** User session data */
    userData: SessionUser;
    /** Run configuration (limits, bot config, etc.) */
    config: RunConfig;
    /** Execution limits specific to this subroutine */
    limits: ExecutionLimits;

    // Execution metadata
    /** The bot executing this subroutine (if from a swarm) */
    executingBotId?: string;
    /** Timestamp when execution started */
    startedAt: Date;
    /** Current execution status */
    status: ExecutionStatus;
}

/**
 * Swarm context that can be passed down to routines.
 * Contains the current state of the swarm including goals, tasks, and shared memory.
 */
export interface SwarmContext {
    /** Unique conversation/swarm ID */
    conversationId: string;
    /** The swarm's primary goal */
    goal: string;
    /** Current subtasks in the swarm */
    subtasks: SwarmSubTask[];
    /** Shared scratchpad/blackboard for the swarm */
    blackboard: BlackboardItem[];
    /** Resources created or referenced by the swarm */
    resources: SwarmResource[];
    /** Historical record of tool calls */
    records: ToolCallRecord[];
    /** Team configuration if the swarm is team-based */
    teamConfig?: TeamConfigObject;
    /** Full chat configuration */
    chatConfig: ChatConfigObject;
    /** The bot that's currently leading the swarm */
    swarmLeader?: string;
    /** Map of subtask IDs to assigned bot IDs */
    subtaskLeaders?: Record<string, string>;
}

/**
 * Routine context for nested routine execution.
 * Contains state from a parent routine that can be inherited by subroutines.
 */
export interface RoutineContext {
    /** The ID of the parent routine run */
    runId: string;
    /** The ID of the parent routine */
    routineId: string;
    /** Current step in the parent routine */
    currentStep?: string;
    /** Accumulated context data from the parent routine */
    contextData: Record<string, unknown>;
    /** Parent routine's configuration */
    parentConfig: RunConfig;
    /** Credits used by the parent routine so far */
    creditsUsed: bigint;
    /** Time elapsed in the parent routine */
    timeElapsed: number;
}

/**
 * Execution limits for a subroutine.
 * These can be inherited from parent context or overridden.
 */
export interface ExecutionLimits {
    /** Maximum credits this subroutine can use */
    maxCredits: bigint;
    /** Maximum execution time in milliseconds */
    maxTimeMs: number;
    /** Maximum number of tool calls */
    maxToolCalls: number;
    /** Maximum number of AI reasoning steps */
    maxReasoningSteps: number;
    /** Whether to enforce strict limits or allow slight overages */
    strictLimits: boolean;
}

/**
 * Execution status for tracking progress.
 */
export enum ExecutionStatus {
    /** Not yet started */
    Pending = "pending",
    /** Currently executing */
    Running = "running",
    /** Execution completed successfully */
    Completed = "completed",
    /** Execution failed with error */
    Failed = "failed",
    /** Execution was cancelled */
    Cancelled = "cancelled",
    /** Execution timed out */
    TimedOut = "timed_out",
    /** Execution hit credit limit */
    CreditLimitExceeded = "credit_limit_exceeded",
}

/**
 * Result of executing a subroutine.
 */
export interface ExecutionResult {
    /** Whether execution succeeded */
    success: boolean;
    /** Updated I/O mapping with outputs filled */
    ioMapping: SubroutineIOMapping;
    /** Total credits used */
    creditsUsed: bigint;
    /** Total time elapsed in milliseconds */
    timeElapsed: number;
    /** Number of tool calls made */
    toolCallsCount: number;
    /** Any error that occurred */
    error?: {
        code: string;
        message: string;
        details?: unknown;
    };
    /** Metadata about the execution */
    metadata?: {
        /** Which execution strategy was used */
        strategy: string;
        /** Any warnings during execution */
        warnings?: string[];
        /** Performance metrics */
        metrics?: Record<string, number>;
    };
}

/**
 * Builder class for creating execution contexts with proper inheritance.
 */
export class ExecutionContextBuilder {
    private context: Partial<ExecutionContext> = {};

    /**
     * Sets the core execution data.
     */
    withCoreData(
        subroutineInstanceId: string,
        routine: ResourceVersion,
        ioMapping: SubroutineIOMapping,
    ): this {
        this.context.subroutineInstanceId = subroutineInstanceId;
        this.context.routine = routine;
        this.context.ioMapping = ioMapping;
        return this;
    }

    /**
     * Inherits context from a parent swarm.
     */
    withSwarmContext(swarmContext: SwarmContext): this {
        this.context.parentSwarmContext = swarmContext;
        // Inherit the executing bot from the swarm if not already set
        if (!this.context.executingBotId && swarmContext.swarmLeader) {
            this.context.executingBotId = swarmContext.swarmLeader;
        }
        return this;
    }

    /**
     * Inherits context from a parent routine.
     */
    withRoutineContext(routineContext: RoutineContext): this {
        this.context.parentRoutineContext = routineContext;
        return this;
    }

    /**
     * Sets the execution environment.
     */
    withEnvironment(userData: SessionUser, config: RunConfig): this {
        this.context.userData = userData;
        this.context.config = config;
        return this;
    }

    /**
     * Sets execution limits, potentially inheriting from parent contexts.
     */
    withLimits(limits?: Partial<ExecutionLimits>): this {
        // Start with defaults
        const defaultLimits: ExecutionLimits = {
            maxCredits: DEFAULT_MAX_CREDITS,
            maxTimeMs: DEFAULT_MAX_TIME_MS,
            maxToolCalls: DEFAULT_MAX_TOOL_CALLS,
            maxReasoningSteps: DEFAULT_MAX_REASONING_STEPS,
            strictLimits: true,
        };

        // Apply parent limits if available
        if (this.context.parentSwarmContext?.chatConfig.limits) {
            const swarmLimits = this.context.parentSwarmContext.chatConfig.limits;
            if (swarmLimits.maxCredits) {
                defaultLimits.maxCredits = BigInt(swarmLimits.maxCredits);
            }
            if (swarmLimits.maxDurationMs) {
                defaultLimits.maxTimeMs = swarmLimits.maxDurationMs;
            }
        }

        // Apply any specific overrides
        this.context.limits = {
            ...defaultLimits,
            ...limits,
        };

        return this;
    }

    /**
     * Builds the final execution context.
     */
    build(): ExecutionContext {
        if (!this.context.subroutineInstanceId || !this.context.routine || !this.context.ioMapping) {
            throw new Error("Core execution data (subroutineInstanceId, routine, ioMapping) must be provided");
        }
        if (!this.context.userData || !this.context.config) {
            throw new Error("Execution environment (userData, config) must be provided");
        }
        if (!this.context.limits) {
            this.withLimits(); // Apply defaults if not set
        }

        return {
            subroutineInstanceId: this.context.subroutineInstanceId,
            routine: this.context.routine,
            ioMapping: this.context.ioMapping,
            parentSwarmContext: this.context.parentSwarmContext,
            parentRoutineContext: this.context.parentRoutineContext,
            userData: this.context.userData,
            config: this.context.config,
            limits: this.context.limits!,
            executingBotId: this.context.executingBotId,
            startedAt: new Date(),
            status: ExecutionStatus.Pending,
        };
    }
}

/**
 * Utilities for working with execution contexts.
 */
export class ExecutionContextUtils {
    /**
     * Extracts relevant context data for AI reasoning from the execution context.
     */
    static extractReasoningContext(context: ExecutionContext): Record<string, unknown> {
        const reasoningContext: Record<string, unknown> = {
            currentTask: {
                name: context.routine.translations?.[0]?.name || "Unknown Task",
                description: context.routine.translations?.[0]?.description || "",
                instructions: context.routine.translations?.[0]?.instructions || "",
            },
            inputs: context.ioMapping.inputs,
        };

        // Add swarm context if available
        if (context.parentSwarmContext) {
            reasoningContext.swarmGoal = context.parentSwarmContext.goal;
            reasoningContext.swarmSubtasks = context.parentSwarmContext.subtasks;
            reasoningContext.swarmBlackboard = context.parentSwarmContext.blackboard;
            reasoningContext.swarmResources = context.parentSwarmContext.resources;
        }

        // Add parent routine context if available
        if (context.parentRoutineContext) {
            reasoningContext.parentRoutine = {
                id: context.parentRoutineContext.routineId,
                currentStep: context.parentRoutineContext.currentStep,
                contextData: context.parentRoutineContext.contextData,
            };
        }

        return reasoningContext;
    }

    /**
     * Checks if execution limits have been exceeded.
     */
    static checkLimits(
        context: ExecutionContext,
        usage: {
            creditsUsed: bigint;
            timeElapsed: number;
            toolCallsCount: number;
            reasoningSteps?: number;
        },
    ): { exceeded: boolean; reason?: string } {
        const { limits } = context;

        if (usage.creditsUsed > limits.maxCredits) {
            return { exceeded: true, reason: `Credit limit exceeded: ${usage.creditsUsed} > ${limits.maxCredits}` };
        }

        if (usage.timeElapsed > limits.maxTimeMs) {
            return { exceeded: true, reason: `Time limit exceeded: ${usage.timeElapsed}ms > ${limits.maxTimeMs}ms` };
        }

        if (usage.toolCallsCount > limits.maxToolCalls) {
            return { exceeded: true, reason: `Tool call limit exceeded: ${usage.toolCallsCount} > ${limits.maxToolCalls}` };
        }

        if (usage.reasoningSteps && usage.reasoningSteps > limits.maxReasoningSteps) {
            return { exceeded: true, reason: `Reasoning step limit exceeded: ${usage.reasoningSteps} > ${limits.maxReasoningSteps}` };
        }

        return { exceeded: false };
    }

    /**
     * Merges parent context data with current context for inheritance.
     */
    static mergeContextData(
        parentData: Record<string, unknown>,
        currentData: Record<string, unknown>,
    ): Record<string, unknown> {
        const result: Record<string, unknown> = {};

        // Copy parent data
        for (const key in parentData) {
            result[key] = parentData[key];
        }

        // Override with current data
        for (const key in currentData) {
            result[key] = currentData[key];
        }

        // Special handling for arrays - concatenate rather than replace
        if (parentData.resources && currentData.resources && Array.isArray(parentData.resources) && Array.isArray(currentData.resources)) {
            result.resources = [...parentData.resources, ...currentData.resources];
        }

        return result;
    }
} 
