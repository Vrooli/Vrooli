import { type ExecutionContext, type ExecutionResult } from "@vrooli/shared";

/**
 * Interface for the unified execution engine that coordinates all execution strategies.
 */
export interface IUnifiedExecutionEngine {
    /**
     * Executes a subroutine using the specified strategy.
     * 
     * @param strategy The execution strategy to use
     * @param context The execution context
     * @param abortSignal Optional abort signal for cancellation
     * @returns Promise resolving to execution result
     */
    executeWithStrategy(
        strategy: ExecutionStrategy,
        context: ExecutionContext,
        abortSignal?: AbortSignal,
    ): Promise<ExecutionResult>;

    /**
     * Estimates the cost of executing with a given strategy.
     * 
     * @param strategy The execution strategy
     * @param context The execution context
     * @returns Promise resolving to estimated cost in credits
     */
    estimateCost(
        strategy: ExecutionStrategy,
        context: ExecutionContext,
    ): Promise<bigint>;
}

/**
 * Base interface for execution strategies.
 */
export interface ExecutionStrategy {
    /** The name of the strategy */
    readonly name: string;

    /**
     * Executes the strategy with the given context.
     * 
     * @param dependencies The execution dependencies and tools
     * @returns Promise resolving to execution result
     */
    execute(dependencies: ExecutionDependencies): Promise<ExecutionResult>;

    /**
     * Strategy-specific routine execution.
     * Called by RoutineExecutor for actual work execution.
     * 
     * @param context The execution context
     * @param ioMapping The I/O mapping for the routine
     * @returns Promise resolving to execution result
     */
    executeRoutine(context: ExecutionContext, ioMapping: any): Promise<ExecutionResult>;

    /**
     * Generate missing inputs for a routine.
     * 
     * @param context The execution context
     * @param ioMapping The current I/O mapping
     * @param missingInputNames List of input names that need to be generated
     * @returns Promise resolving to execution result with generated inputs
     */
    generateMissingInputs(
        context: ExecutionContext,
        ioMapping: any,
        missingInputNames: string[],
    ): Promise<ExecutionResult>;

    /**
     * Estimates the cost of executing this strategy.
     * 
     * @param context The execution context
     * @returns Promise resolving to estimated cost in credits
     */
    estimateCost(context: ExecutionContext): Promise<bigint>;
}

/**
 * Dependencies and tools available to execution strategies.
 */
export interface ExecutionDependencies {
    /** The execution context */
    context: ExecutionContext;
    /** Reasoning engine for AI operations */
    reasoningEngine: any; // ReasoningEngine
    /** Tool runner for executing tools */
    toolRunner: any; // ToolRunner
    /** Context builder for managing conversation context */
    contextBuilder: any; // ContextBuilder
    /** Message store for conversation management */
    messageStore: any; // MessageStore
    /** Bot participant information */
    botParticipant: any; // BotParticipant
    /** Available tools for execution */
    availableTools: any[]; // Tool[]
    /** Optional abort signal for cancellation */
    abortSignal?: AbortSignal;
    /** Logger instance */
    logger: any; // Logger
}

/**
 * Factory for creating and selecting execution strategies.
 */
export interface StrategyFactory {
    /**
     * Selects the appropriate strategy for a routine.
     * 
     * @param routine The routine to execute
     * @returns The selected execution strategy
     */
    selectStrategy(routine: any): ExecutionStrategy;

    /**
     * Creates a specific strategy by name.
     * 
     * @param strategyName The name of the strategy to create
     * @param dependencies Optional dependencies for the strategy
     * @returns The created execution strategy
     */
    createStrategy(
        strategyName: string,
        dependencies?: any,
    ): ExecutionStrategy;
}

/**
 * Interface for synchronizing state between swarms and routines.
 */
export interface StateSynchronizer {
    /**
     * Gets the swarm context for a given run.
     * 
     * @param runId The run ID
     * @returns Promise resolving to swarm context or undefined
     */
    getSwarmContext(runId: string): Promise<any | undefined>; // SwarmContext

    /**
     * Synchronizes routine results back to the swarm.
     * 
     * @param routineContext The routine context with results
     * @returns Promise that resolves when synchronization is complete
     */
    syncRoutineResultsToSwarm(routineContext: any): Promise<void>; // RoutineContext

    /**
     * Updates swarm blackboard with routine outputs.
     * 
     * @param swarmId The swarm ID
     * @param outputs The outputs to add to the blackboard
     * @returns Promise that resolves when update is complete
     */
    updateSwarmBlackboard(
        swarmId: string,
        outputs: Record<string, unknown>,
    ): Promise<void>;
}

/**
 * Utility functions for working with execution contexts.
 */
export class ExecutionContextUtils {
    /**
     * Checks if execution limits have been exceeded.
     * 
     * @param context The execution context
     * @param usage Current usage statistics
     * @returns Object indicating if limits are exceeded and why
     */
    static checkLimits(
        context: ExecutionContext,
        usage: {
            creditsUsed: bigint;
            timeElapsed: number;
            toolCallsCount: number;
        },
    ): {
        exceeded: boolean;
        reason?: string;
    } {
        const { limits } = context;

        if (usage.creditsUsed >= limits.maxCredits) {
            return {
                exceeded: true,
                reason: `Credit limit exceeded: ${usage.creditsUsed} >= ${limits.maxCredits}`,
            };
        }

        if (usage.timeElapsed >= limits.maxTimeMs) {
            return {
                exceeded: true,
                reason: `Time limit exceeded: ${usage.timeElapsed}ms >= ${limits.maxTimeMs}ms`,
            };
        }

        if (usage.toolCallsCount >= limits.maxToolCalls) {
            return {
                exceeded: true,
                reason: `Tool call limit exceeded: ${usage.toolCallsCount} >= ${limits.maxToolCalls}`,
            };
        }

        return { exceeded: false };
    }

    /**
     * Merges parent context into child context.
     * 
     * @param parentContext The parent execution context
     * @param childContext The child execution context
     * @returns Merged execution context
     */
    static mergeContexts(
        parentContext: ExecutionContext,
        childContext: Partial<ExecutionContext>,
    ): ExecutionContext {
        return {
            ...parentContext,
            ...childContext,
            // Preserve parent swarm context if child doesn't have one
            parentSwarmContext: childContext.parentSwarmContext || parentContext.parentSwarmContext,
            // Merge limits (use more restrictive)
            limits: {
                maxCredits: childContext.limits?.maxCredits && childContext.limits.maxCredits < parentContext.limits.maxCredits
                    ? childContext.limits.maxCredits
                    : parentContext.limits.maxCredits,
                maxTimeMs: childContext.limits?.maxTimeMs && childContext.limits.maxTimeMs < parentContext.limits.maxTimeMs
                    ? childContext.limits.maxTimeMs
                    : parentContext.limits.maxTimeMs,
                maxToolCalls: childContext.limits?.maxToolCalls && childContext.limits.maxToolCalls < parentContext.limits.maxToolCalls
                    ? childContext.limits.maxToolCalls
                    : parentContext.limits.maxToolCalls,
                maxReasoningSteps: childContext.limits?.maxReasoningSteps && childContext.limits.maxReasoningSteps < parentContext.limits.maxReasoningSteps
                    ? childContext.limits.maxReasoningSteps
                    : parentContext.limits.maxReasoningSteps,
                strictLimits: parentContext.limits.strictLimits || (childContext.limits?.strictLimits ?? false),
            },
        };
    }
} 
