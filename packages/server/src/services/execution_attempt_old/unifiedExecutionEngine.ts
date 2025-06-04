import { ExecutionStatus, type BotConfigObject, type ExecutionContext, type ExecutionResult } from "@local/shared";
import { type Logger } from "winston";
import { logger } from "../../events/logger.js";
import { type ContextBuilder } from "../conversation/contextBuilder.js";
import { type MessageStore } from "../conversation/messageStore.js";
import { type ReasoningEngine } from "../conversation/responseEngine.js";
import { type ToolRunner } from "../conversation/toolRunner.js";
import { type BotParticipant, type MessageState } from "../conversation/types.js";
import { type ToolRegistry } from "../mcp/registry.js";
import { type Tool } from "../mcp/types.js";
import { type ExecutionDependencies, type ExecutionStrategy, type IUnifiedExecutionEngine } from "./interfaces.js";

/**
 * Request to execute a subroutine.
 */
export interface SubroutineExecutionRequest {
    /** The execution context */
    context: ExecutionContext;
    /** The execution strategy to use */
    strategy: ExecutionStrategy;
    /** Optional abort signal for cancellation */
    abortSignal?: AbortSignal;
}

/**
 * Result of executing a subroutine, extends the base ExecutionResult.
 */
export interface SubroutineExecutionResult extends ExecutionResult {
    /** Messages generated during execution (for conversational strategies) */
    messages?: MessageState[];
    /** Resources created during execution */
    createdResources?: string[];
}

/**
 * Configuration for the unified execution engine.
 */
export interface UnifiedExecutionEngineConfig {
    /** Logger instance */
    logger?: Logger;
    /** Whether to emit progress events */
    emitProgressEvents?: boolean;
    /** Whether to enforce strict credit limits */
    strictCreditLimits?: boolean;
}

/**
 * Unified execution engine that coordinates all AI-based executions.
 * 
 * This engine:
 * - Routes execution requests to appropriate strategies
 * - Manages context inheritance from swarms/routines
 * - Tracks credits and resource usage
 * - Provides consistent execution interface across all tiers
 */
export class UnifiedExecutionEngine implements IUnifiedExecutionEngine {
    private readonly logger: Logger;
    private readonly config: UnifiedExecutionEngineConfig;
    private readonly activeExecutions = new Map<string, ExecutionContext>();

    constructor(
        private readonly reasoningEngine: ReasoningEngine,
        private readonly toolRunner: ToolRunner,
        private readonly contextBuilder: ContextBuilder,
        private readonly messageStore: MessageStore,
        private readonly toolRegistry: ToolRegistry,
        config?: UnifiedExecutionEngineConfig,
    ) {
        this.config = {
            emitProgressEvents: true,
            strictCreditLimits: true,
            ...config,
        };
        this.logger = config?.logger || logger;
    }

    /**
     * Executes a subroutine using the specified strategy.
     */
    async execute(request: SubroutineExecutionRequest): Promise<SubroutineExecutionResult> {
        const { context, strategy, abortSignal } = request;
        const startTime = Date.now();

        // Track active execution
        this.activeExecutions.set(context.subroutineInstanceId, context);
        context.status = ExecutionStatus.Running;

        this.logger.info(`Starting execution of subroutine ${context.subroutineInstanceId}`, {
            routine: context.routine.id,
            strategy: strategy.name,
            hasSwarmContext: !!context.parentSwarmContext,
            hasRoutineContext: !!context.parentRoutineContext,
        });

        try {
            // Check if we should abort before starting
            if (abortSignal?.aborted) {
                throw new Error(`Execution aborted before start: ${abortSignal.reason}`);
            }

            // Create a bot participant for AI-based strategies
            const botParticipant = await this.createBotParticipant(context);

            // Get available tools
            const availableTools = await this.getAvailableTools(context);

            // Create execution dependencies
            const dependencies: ExecutionDependencies = {
                context,
                reasoningEngine: this.reasoningEngine,
                toolRunner: this.toolRunner,
                contextBuilder: this.contextBuilder,
                messageStore: this.messageStore,
                botParticipant,
                availableTools,
                abortSignal,
                logger: this.logger,
            };

            // Execute using the strategy
            const result = await strategy.execute(dependencies);

            // Update context status
            context.status = result.success ? ExecutionStatus.Completed : ExecutionStatus.Failed;

            // Emit progress if configured
            if (this.config.emitProgressEvents) {
                await this.emitExecutionComplete(context, result as SubroutineExecutionResult);
            }

            return result as SubroutineExecutionResult;

        } catch (error) {
            this.logger.error(`Execution failed for subroutine ${context.subroutineInstanceId}`, error);

            // Determine status based on error type
            if (error instanceof Error) {
                if (error.message.includes("abort")) {
                    context.status = ExecutionStatus.Cancelled;
                } else if (error.message.includes("timeout")) {
                    context.status = ExecutionStatus.TimedOut;
                } else if (error.message.includes("credit limit")) {
                    context.status = ExecutionStatus.CreditLimitExceeded;
                } else {
                    context.status = ExecutionStatus.Failed;
                }
            } else {
                context.status = ExecutionStatus.Failed;
            }

            const errorResult: SubroutineExecutionResult = {
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
                    strategy: strategy.name,
                },
            };

            // Emit error event if configured
            if (this.config.emitProgressEvents) {
                await this.emitExecutionError(context, errorResult.error!);
            }

            return errorResult;

        } finally {
            // Clean up active execution tracking
            this.activeExecutions.delete(context.subroutineInstanceId);
        }
    }

    /**
     * Executes a subroutine with a specific strategy.
     * Convenience method that combines strategy selection with execution.
     * Implements IUnifiedExecutionEngine.executeWithStrategy
     */
    async executeWithStrategy(
        strategy: ExecutionStrategy,
        context: ExecutionContext,
        abortSignal?: AbortSignal,
    ): Promise<ExecutionResult> {
        const result = await this.execute({ context, strategy, abortSignal });
        return result; // SubroutineExecutionResult extends ExecutionResult
    }

    /**
     * Estimates the cost of executing with a given strategy.
     * Implements IUnifiedExecutionEngine.estimateCost
     */
    async estimateCost(
        strategy: ExecutionStrategy,
        context: ExecutionContext,
    ): Promise<bigint> {
        try {
            // Delegate to the strategy's cost estimation
            return await strategy.estimateCost(context);
        } catch (error) {
            this.logger.warn(`Failed to estimate cost for strategy ${strategy.name}`, error);
            // Return a default cost estimate if strategy estimation fails
            return BigInt(100); // Default cost estimate
        }
    }

    /**
     * Creates a bot participant for AI-based execution strategies.
     */
    private async createBotParticipant(context: ExecutionContext): Promise<BotParticipant> {
        let botConfig: BotConfigObject;
        const botName = "Assistant";
        const botId = context.executingBotId || "system";

        // Try to get bot config from parent swarm context
        if (context.parentSwarmContext && context.executingBotId) {
            // In a real implementation, you would fetch the bot config from the database
            // For now, we'll use a default config
            botConfig = {
                __version: "1.0",
                model: "gpt-4",
            };
        } else if (context.config.botConfig && context.config.botConfig.model) {
            // Use run config if available, ensuring it has the required fields
            botConfig = {
                __version: "1.0",
                model: String(context.config.botConfig.model), // Convert to string
            };
        } else {
            // Use default config
            botConfig = {
                __version: "1.0",
                model: "gpt-4",
            };
        }

        return {
            id: botId,
            name: botName,
            config: botConfig,
            meta: {
                role: context.parentSwarmContext ? "worker" : "assistant",
            },
        };
    }

    /**
     * Gets available tools for the execution context.
     */
    private async getAvailableTools(context: ExecutionContext): Promise<Tool[]> {
        const tools: Tool[] = [];

        // Add built-in tools
        const builtInTools = this.toolRegistry.getBuiltInDefinitions();
        tools.push(...builtInTools);

        // Add swarm tools if in swarm context
        if (context.parentSwarmContext) {
            const swarmTools = this.toolRegistry.getSwarmToolDefinitions();
            tools.push(...swarmTools);
        }

        // Filter tools based on routine configuration
        // TODO: Implement tool filtering based on routine config

        return tools;
    }

    /**
     * Emits an execution complete event.
     */
    private async emitExecutionComplete(
        context: ExecutionContext,
        result: SubroutineExecutionResult,
    ): Promise<void> {
        // TODO: Implement proper event publishing when event types are defined
        this.logger.info(`Execution completed for subroutine ${context.subroutineInstanceId}`, {
            routineId: context.routine.id,
            success: result.success,
            creditsUsed: result.creditsUsed.toString(),
            timeElapsed: result.timeElapsed,
        });
    }

    /**
     * Emits an execution error event.
     */
    private async emitExecutionError(
        context: ExecutionContext,
        error: NonNullable<ExecutionResult["error"]>,
    ): Promise<void> {
        // TODO: Implement proper event publishing when event types are defined
        this.logger.error(`Execution error for subroutine ${context.subroutineInstanceId}`, {
            routineId: context.routine.id,
            error,
        });
    }

    /**
     * Gets the status of an active execution.
     */
    getExecutionStatus(subroutineInstanceId: string): ExecutionStatus | null {
        const context = this.activeExecutions.get(subroutineInstanceId);
        return context?.status || null;
    }

    /**
     * Cancels an active execution.
     */
    async cancelExecution(subroutineInstanceId: string, reason?: string): Promise<boolean> {
        const context = this.activeExecutions.get(subroutineInstanceId);
        if (!context) {
            return false;
        }

        context.status = ExecutionStatus.Cancelled;

        // Emit cancellation event
        if (this.config.emitProgressEvents) {
            await this.emitExecutionError(context, {
                code: "EXECUTION_CANCELLED",
                message: reason || "Execution cancelled by user",
            });
        }

        return true;
    }

    /**
     * Gets statistics about current executions.
     */
    getExecutionStats(): {
        activeCount: number;
        byStatus: Record<ExecutionStatus, number>;
        byStrategy: Record<string, number>;
    } {
        const stats = {
            activeCount: this.activeExecutions.size,
            byStatus: {} as Record<ExecutionStatus, number>,
            byStrategy: {} as Record<string, number>,
        };

        // Initialize status counts
        for (const status of Object.values(ExecutionStatus)) {
            stats.byStatus[status as ExecutionStatus] = 0;
        }

        // Count executions
        for (const context of this.activeExecutions.values()) {
            stats.byStatus[context.status]++;
            // Strategy counting would require tracking in context
        }

        return stats;
    }
} 
