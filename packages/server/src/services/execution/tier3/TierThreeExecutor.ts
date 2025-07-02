import {
    type ExecutionContext,
    type ExecutionResult,
    ExecutionStates,
    type ExecutionStatus,
    generatePK,
    isStepExecutionInput,
    isValidStepType,
    isValidStrategy,
    type StepExecutionInput,
    type SwarmId,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    type ValidStepType,
    type ValidStrategy,
} from "@vrooli/shared";
import { BaseTierExecutor } from "../shared/BaseTierExecutor.js";
// Socket events now handled through unified event system
import { logger } from "../../../events/logger.js";
import type { ConversationEngine } from "../../conversation/conversationEngine.js";
import { getEventBus } from "../../events/eventBus.js";
import { EventUtils } from "../../events/utils.js";
import type { ResponseService } from "../../response/responseService.js";
import { IOProcessor } from "./engine/ioProcessor.js";
import { ResourceManager } from "./engine/resourceManager.js";
import { SimpleStrategyProvider } from "./engine/simpleStrategyProvider.js";
import { ToolOrchestrator } from "./engine/toolOrchestrator.js";
import { UnifiedExecutor } from "./engine/unifiedExecutor.js";
import { ValidationEngine } from "./engine/validationEngine.js";

/**
 * Tier Three Executor
 * 
 * Main entry point for Tier 3 execution intelligence.
 * Manages step execution, strategy selection, and tool orchestration.
 */
export class TierThreeExecutor extends BaseTierExecutor implements TierCommunicationInterface {
    private readonly unifiedExecutor: UnifiedExecutor;

    // Track active executions for interface compliance
    private readonly activeExecutions: Map<SwarmId, { status: ExecutionStates; startTime: Date; context: ExecutionContext }> = new Map();

    constructor(conversationEngine?: ConversationEngine, responseService?: ResponseService) {
        super("TierThreeExecutor", "tier3");

        // Initialize components
        const toolOrchestrator = new ToolOrchestrator();
        const resourceManager = new ResourceManager();
        const validationEngine = new ValidationEngine();
        const ioProcessor = new IOProcessor();

        // Create simple strategy provider (enables agent-driven optimization)
        const strategyConfig = {
            defaultStrategy: "conversational" as const,
            fallbackChain: ["conversational", "reasoning", "deterministic"] as const,
            adaptationEnabled: true,
            learningRate: 0.1,
        };
        const strategyProvider = new SimpleStrategyProvider(strategyConfig, toolOrchestrator, validationEngine, conversationEngine, responseService);

        // Create unified executor
        this.unifiedExecutor = new UnifiedExecutor(
            strategyProvider,
            toolOrchestrator,
            resourceManager,
            validationEngine,
            ioProcessor,
        );

        // Setup event handlers
        this.setupEventHandlers();

        logger.info("[TierThreeExecutor] Initialized");
    }

    /**
     * Executes a single step
     */
    async executeStep(context: ExecutionContext): Promise<ExecutionResult> {
        logger.info("[TierThreeExecutor] Executing step", {
            stepId: context.stepId,
            stepType: context.stepType,
            runId: context.runId,
        });

        try {
            // Execute through unified executor
            const result = await this.unifiedExecutor.execute(context);

            // Export context if needed
            if (context.config?.exportContext) {
                const runContext = new ExecutionRunContext({
                    runId: context.runId,
                    routineId: context.routineId || "",
                    routineName: context.routineName || "",
                    currentStepId: context.stepId,
                    userData: context.userData || { id: "" },
                });

                const exportedContext = await this.contextExporter.exportContext(
                    runContext,
                    {
                        outputs: result.result as Record<string, unknown>,
                        strategy: result.metadata?.strategy as any,
                        resourceUsage: result.metadata?.resourceUsage as any,
                    },
                );
                result.metadata.exportedContext = exportedContext;
            }

            // Emit completion event using unified event system
            if (result.success) {
                await this.emitStepEvent(EventTypes.STEP_COMPLETED, context, {
                    outputs: result.result,
                    metadata: result.metadata,
                });
            } else {
                await this.emitStepEvent(EventTypes.STEP_FAILED, context, {
                    error: result.error,
                    metadata: result.metadata,
                }, "reliable"); // Use reliable delivery for failures
            }

            return result;

        } catch (error) {
            logger.error("[TierThreeExecutor] Step execution failed", {
                stepId: context.stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Emit failure event using unified event system with reliable delivery
            await this.emitStepEvent(EventTypes.STEP_FAILED, context, {
                error: error instanceof Error ? error.message : "Unknown error",
            }, "reliable");

            throw error;
        }
    }

    /**
     * Gets available tools for the current context
     */
    async getAvailableTools(): Promise<Array<{ name: string; description: string }>> {
        const toolOrchestrator = this.unifiedExecutor.getToolOrchestrator();
        const tools = await toolOrchestrator.listTools();

        return tools.map(tool => ({
            name: tool.name,
            description: tool.description,
        }));
    }

    /**
     * Processes tool approval
     */
    async processToolApproval(
        approvalId: string,
        approved: boolean,
        approvedBy?: string,
    ): Promise<void> {
        const toolOrchestrator = this.unifiedExecutor.getToolOrchestrator();
        await toolOrchestrator.processApproval(approvalId, approved, approvedBy);
    }

    /**
     * Shuts down the executor
     */
    async shutdown(): Promise<void> {
        logger.info("[TierThreeExecutor] Shutting down");
        // Currently no persistent state to clean up
    }

    /**
     * TierCommunicationInterface implementation
     */

    /**
     * Execute a step execution request with type-safe input validation
     */
    async execute<TInput extends StepExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        return this.executeWithErrorHandling(request, async (req) => {
            return this.executeImpl<TInput, TOutput>(req);
        });
    }

    /**
     * Implementation of tier-specific execution logic
     */
    protected async executeImpl<TInput extends StepExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        const { context, input, allocation, options } = request;
        const swarmId = context.swarmId;

        // Validate input type (additional validation beyond base class)
        if (!isStepExecutionInput(input)) {
            throw new Error(`Invalid input type for Tier 3. Expected StepExecutionInput, got ${typeof input}`);
        }

        // Track execution for interface compliance
        this.activeExecutions.set(swarmId, {
            status: ExecutionStates.RUNNING,
            startTime: new Date(),
            context,
        });

        logger.info("[TierThreeExecutor] Starting tier execution", {
            swarmId,
            stepId: input.stepId,
            stepType: input.stepType,
            strategy: input.strategy,
            toolName: input.toolName,
        });

        // Create execution context with type-safe transformation
        const executionContext = this.createExecutionContext(request);

        // Emit step started event
        await this.emitStepEvent(EventTypes.STEP_STARTED, executionContext, {
            strategy: input.strategy,
            toolName: input.toolName,
            parameters: input.parameters,
        });

        let result: ExecutionResult<TOutput>;
        try {
            // Execute the step
            result = await this.executeStep(executionContext);

            // Update execution status
            this.activeExecutions.set(swarmId, {
                status: ExecutionStates.COMPLETED,
                startTime: this.activeExecutions.get(swarmId)?.startTime || new Date(),
                context,
            });

            // Emit step completed event
            await this.emitStepEvent(EventTypes.STEP_COMPLETED, executionContext, {
                strategy: input.strategy,
                toolName: input.toolName,
                parameters: input.parameters,
                result: {
                    status: result.status,
                    output: result.output,
                    creditsUsed: result.creditsUsed,
                    duration: result.duration,
                },
            });

        } catch (executionError) {
            // Update execution status to failed
            this.activeExecutions.set(swarmId, {
                status: ExecutionStates.FAILED,
                startTime: this.activeExecutions.get(swarmId)?.startTime || new Date(),
                context,
            });

            // Emit step failed event (reliable delivery for errors)
            await this.emitStepEvent(EventTypes.STEP_FAILED, executionContext, {
                strategy: input.strategy,
                toolName: input.toolName,
                parameters: input.parameters,
                error: {
                    name: (executionError as Error).name,
                    message: (executionError as Error).message,
                    stack: (executionError as Error).stack,
                },
            }, "reliable");

            // Re-throw to maintain existing error handling
            throw executionError;
        }

        // Emit config update through unified event system (if swarmId is available)
        if (context.swarmId) {
            await this.publishEvent(EventTypes.CONFIG_SWARM_UPDATED, {
                entityType: "swarm",
                entityId: context.swarmId,
                config: {
                    records: [{
                        id: generatePK().toString(),
                        routine_id: input.stepId,
                        routine_name: input.toolName || `${input.stepType} execution`,
                        params: input.parameters,
                        output_resource_ids: [], // TODO: Extract from result
                        caller_bot_id: context.userData.id,
                        created_at: new Date().toISOString(),
                    }],
                    stats: {
                        totalToolCalls: 1, // Increment would need to be tracked
                        totalCredits: allocation.maxCredits,
                        lastProcessingCycleEndedAt: Date.now(),
                    },
                },
            }, {
                deliveryGuarantee: "fire-and-forget",
                priority: "medium",
                conversationId: context.conversationId || context.swarmId,
            });
        }

        return result as ExecutionResult<TOutput>;
    }

    /**
     * Create execution context from tier request with type-safe validation
     */
    private createExecutionContext(
        request: TierExecutionRequest<StepExecutionInput>,
    ): ExecutionContext {
        const { context, input } = request;

        // Validate and transform strategy
        const strategy = this.validateStrategy(input.strategy);

        // Validate and transform step type
        const stepType = this.validateStepType(input.stepType);

        return {
            ...context,
            stepId: input.stepId,
            stepType,
            strategy,
            inputs: input.parameters,
            config: {
                toolName: input.toolName,
                strategy: input.strategy,
                ...request.options,
            },
        };
    }

    /**
     * Emit step events using the unified event system (simplified interface)
     */
    private async emitStepEvent(
        eventType: string,
        context: ExecutionContext,
        data: Record<string, unknown>,
        deliveryGuarantee: "fire-and-forget" | "reliable" = "fire-and-forget",
    ): Promise<void> {
        try {
            const event = EventUtils.createBaseEvent(
                eventType,
                {
                    runId: context.runId,
                    stepId: context.stepId,
                    stepType: context.stepType,
                    routineId: context.routineId,
                    userId: context.userData.id,
                    ...data,
                },
                EventUtils.createEventSource(3, "TierThreeExecutor", context.swarmId),
                EventUtils.createEventMetadata(
                    deliveryGuarantee,
                    deliveryGuarantee === "reliable" ? "high" : "medium",
                    {
                        conversationId: context.swarmId,
                        userId: context.userData.id,
                        tags: ["tier3", "execution", context.stepType || "unknown"],
                    },
                ),
            );

            await getEventBus().publish(event);

            logger.debug("[TierThreeExecutor] Emitted step event", {
                eventType,
                stepId: context.stepId,
                runId: context.runId,
                deliveryGuarantee,
            });

        } catch (eventError) {
            logger.error("[TierThreeExecutor] Failed to emit step event", {
                eventType,
                stepId: context.stepId,
                runId: context.runId,
                error: eventError instanceof Error ? eventError.message : String(eventError),
            });
        }
    }


    /**
     * Override to provide tier-specific input type name
     */
    protected getInputTypeName(input: unknown): string {
        if (isStepExecutionInput(input)) {
            return `StepExecutionInput(${input.stepType})`;
        }
        return super.getInputTypeName(input);
    }

    /**
     * Override to provide tier-specific additional error context
     */
    protected getAdditionalErrorContext(
        request: TierExecutionRequest<any>,
        error: unknown,
    ): Record<string, unknown> {
        const baseContext = super.getAdditionalErrorContext(request, error);

        if (isStepExecutionInput(request.input)) {
            return {
                ...baseContext,
                stepId: request.input.stepId,
                stepType: request.input.stepType,
                strategy: request.input.strategy,
                toolName: request.input.toolName,
                hasValidStrategy: isValidStrategy(request.input.strategy),
                hasValidStepType: isValidStepType(request.input.stepType),
            };
        }

        return baseContext;
    }

    /**
     * Override to provide tier-specific strategy extraction
     */
    protected getStrategyFromRequest(request: TierExecutionRequest<any>): string {
        if (isStepExecutionInput(request.input)) {
            return request.input.strategy;
        }
        return super.getStrategyFromRequest(request);
    }

    /**
     * Validate strategy with proper type checking
     */
    private validateStrategy(strategy: string): ValidStrategy {
        if (!isValidStrategy(strategy)) {
            throw new Error(
                `Invalid strategy: ${strategy}. Valid strategies are: ${["conversational", "reasoning", "deterministic", "routing"].join(", ")}`,
            );
        }
        return strategy;
    }

    /**
     * Validate step type with proper type checking
     */
    private validateStepType(stepType: string): ValidStepType {
        if (!isValidStepType(stepType)) {
            throw new Error(
                `Invalid step type: ${stepType}. Valid step types are: ${["tool_call", "api_request", "data_transform", "decision_point", "loop_iteration", "conditional_branch", "parallel_execution", "subroutine_call"].join(", ")}`,
            );
        }
        return stepType;
    }

    /**
     * Get execution status for monitoring
     */
    async getExecutionStatus(swarmId: SwarmId): Promise<ExecutionStatus> {
        const execution = this.activeExecutions.get(swarmId);
        const status = execution?.status || ExecutionStates.COMPLETED;

        return {
            swarmId,
            status,
            progress: status === ExecutionStates.COMPLETED ? 100 : status === ExecutionStates.RUNNING ? 50 : 0,
            metadata: {
                currentPhase: status === ExecutionStates.RUNNING ? "executing" : "idle",
                activeRuns: status === ExecutionStates.RUNNING ? 1 : 0,
                completedRuns: status === ExecutionStates.COMPLETED ? 1 : 0,
                startTime: execution?.startTime?.toISOString(),
            },
        };
    }

    /**
     * Cancel a running execution
     */
    async cancelExecution(swarmId: SwarmId): Promise<void> {
        const execution = this.activeExecutions.get(swarmId);
        if (execution) {
            this.activeExecutions.set(swarmId, {
                ...execution,
                status: ExecutionStates.CANCELLED,
            });

            logger.info("[TierThreeExecutor] Execution cancelled", { swarmId });
        }
    }

    /**
     * Get tier capabilities
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: "tier3",
            maxConcurrency: 20,
            resourceLimits: {
                maxCredits: "10000",
                maxDurationMs: 1800000, // 30 minutes
                maxMemoryMB: 2048,
            },
        };
    }

    /**
     * Private helper methods
     */
    private setupEventHandlers(): void {
        // Handle tool approval requests
        getEventBus().on("tool.approval_required", async (event) => {
            // Forward to appropriate handlers (e.g., UI notification service)
            logger.info("[TierThreeExecutor] Tool approval required", event.data);
        });

        // Handle resource alerts
        getEventBus().on("resources.exhausted", async (event) => {
            logger.warn("[TierThreeExecutor] Resources exhausted", event.data);
        });

        // Handle validation failures
        getEventBus().on("validation.failed", async (event) => {
            logger.error("[TierThreeExecutor] Validation failed", event.data);
        });
    }
}
