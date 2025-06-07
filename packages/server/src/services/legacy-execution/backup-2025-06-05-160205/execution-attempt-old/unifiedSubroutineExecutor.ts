import { type EnhancedSubroutineExecutor, type ExecutionContext, type ExecutionDependencies, type ExecutionResult, type Location, type PassableLogger } from "@vrooli/shared";
import { ExecutionContextUtils, type ExecutionStrategy, type IUnifiedExecutionEngine, type StrategyFactory } from "./interfaces.js";

/**
 * Bridge between RunStateMachine and UnifiedExecutionEngine.
 * Implements the enhanced SubroutineExecutor interface to provide
 * strategy-aware execution within the routine execution flow.
 */
export class UnifiedSubroutineExecutor implements EnhancedSubroutineExecutor {
    private readonly logger: PassableLogger;

    constructor(
        private readonly unifiedEngine: IUnifiedExecutionEngine,
        private readonly strategyFactory: StrategyFactory,
        logger?: PassableLogger,
    ) {
        this.logger = logger || console as any; // Use console as fallback
    }

    /**
     * Executes a subroutine step using the unified execution engine.
     * Routes execution to the appropriate strategy based on routine configuration.
     */
    async executeStep(
        context: ExecutionContext,
        currentLocation: Location,
        _dependencies: ExecutionDependencies,
    ): Promise<ExecutionResult> {
        const startTime = Date.now();

        this.logger.info(`Executing step for subroutine ${context.subroutineInstanceId}`, {
            routine: context.routine.id,
            location: currentLocation.locationId,
            hasSwarmContext: !!context.parentSwarmContext,
            hasRoutineContext: !!context.parentRoutineContext,
        } as any);

        try {
            // Determine the execution strategy based on routine
            const strategy = this.selectExecutionStrategy(context);

            if ("info" in this.logger) {
                this.logger.info(`Selected execution strategy: ${strategy.name}`, {
                    routine: context.routine.id,
                    subroutine: context.subroutineInstanceId,
                } as any);
            }

            // Check execution limits before starting
            const limitCheck = ExecutionContextUtils.checkLimits(context, {
                creditsUsed: BigInt(0),
                timeElapsed: 0,
                toolCallsCount: 0,
            });

            if (limitCheck.exceeded) {
                throw new Error(limitCheck.reason);
            }

            // Execute using the UnifiedExecutionEngine
            const result = await this.unifiedEngine.executeWithStrategy(
                strategy,
                context,
            );

            // Convert to the expected ExecutionResult format
            const executionResult: ExecutionResult = {
                success: result.success,
                ioMapping: result.ioMapping || context.ioMapping,
                creditsUsed: result.creditsUsed,
                timeElapsed: Date.now() - startTime,
                toolCallsCount: result.toolCallsCount || 0,
                error: result.error,
                metadata: {
                    strategy: strategy.name,
                    ...result.metadata,
                },
            };

            this.logger.info(`Step execution completed for subroutine ${context.subroutineInstanceId}`, {
                success: executionResult.success,
                creditsUsed: executionResult.creditsUsed.toString(),
                timeElapsed: executionResult.timeElapsed,
                strategy: strategy.name,
            } as any);

            return executionResult;

        } catch (error) {
            this.logger.error(`Step execution failed for subroutine ${context.subroutineInstanceId}`, { error } as any);

            return {
                success: false,
                ioMapping: context.ioMapping,
                creditsUsed: BigInt(0),
                timeElapsed: Date.now() - startTime,
                toolCallsCount: 0,
                error: {
                    code: "EXECUTION_ERROR",
                    message: error instanceof Error ? error.message : "Unknown error",
                    details: error,
                },
                metadata: {
                    strategy: "unknown",
                },
            };
        }
    }

    /**
     * Estimates the cost of executing a subroutine.
     */
    async estimateCost(
        context: ExecutionContext,
        _location: Location,
    ): Promise<bigint> {
        try {
            // For now, return a default estimate based on context limits
            // In a full implementation, this would use strategy-specific estimation
            const baseEstimate = context.limits.maxCredits / BigInt(100); // Conservative estimate

            if ("info" in this.logger) {
                this.logger.info(`Cost estimate for subroutine ${context.subroutineInstanceId}`, {
                    routine: context.routine.id,
                    estimatedCost: baseEstimate.toString(),
                } as any);
            }

            return baseEstimate;

        } catch (error) {
            this.logger.error(`Cost estimation failed for subroutine ${context.subroutineInstanceId}`, { error } as any);

            // Return a conservative high estimate on error
            return context.limits.maxCredits;
        }
    }

    /**
     * Selects the appropriate execution strategy based on routine configuration.
     */
    private selectExecutionStrategy(context: ExecutionContext): ExecutionStrategy {
        // Use the existing StrategyFactory selectStrategy method
        return this.strategyFactory.selectStrategy(context.routine);
    }
} 
