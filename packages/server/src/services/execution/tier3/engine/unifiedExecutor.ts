import {
    type AvailableResources,
    EventTypes,
    type ExecutionContext,
    type ExecutionResult,
    ExecutionStates,
    type ExecutionStatus,
    type ExecutionStrategy,
    type RunContext,
    type StepExecutionInput,
    type StrategyExecutionResult,
    type SwarmId,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { getEventBus } from "../../../events/eventBus.js";
import { EventUtils } from "../../../events/utils.js";
import { type IOProcessor } from "./ioProcessor.js";
import { type ResourceManager } from "./resourceManager.js";
import { type SimpleStrategyProvider } from "./simpleStrategyProvider.js";
import { type ToolOrchestrator } from "./toolOrchestrator.js";
import { type ValidationEngine } from "./validationEngine.js";

/**
 * UnifiedExecutor - The heart of Tier 3 Execution Intelligence
 * 
 * This class represents the culmination of Vrooli's execution intelligence,
 * where individual routine steps are executed with strategy-aware adaptation
 * that evolves based on routine characteristics, usage patterns, and performance metrics.
 * 
 * Key capabilities:
 * - Dynamic strategy selection (Conversational, Reasoning, Deterministic)
 * - Resource management with credit and time tracking
 * - Tool orchestration through MCP protocol
 * - Output validation with schema enforcement
 * - Event emission for emergent monitoring and learning
 * 
 * Implements TierCommunicationInterface for standardized inter-tier communication.
 */
export class UnifiedExecutor implements TierCommunicationInterface {
    private readonly strategySelector: SimpleStrategyProvider;
    private readonly resourceManager: ResourceManager;
    private readonly ioProcessor: IOProcessor;
    private readonly toolOrchestrator: ToolOrchestrator;
    private readonly validationEngine: ValidationEngine;

    constructor(
        strategySelector: SimpleStrategyProvider,
        toolOrchestrator: ToolOrchestrator,
        resourceManager: ResourceManager,
        validationEngine: ValidationEngine,
        ioProcessor: IOProcessor,
    ) {
        this.strategySelector = strategySelector;
        this.toolOrchestrator = toolOrchestrator;
        this.resourceManager = resourceManager;
        this.validationEngine = validationEngine;
        this.ioProcessor = ioProcessor;
    }

    /**
     * Executes a single step with strategy-aware adaptation
     * 
     * This method is called by Tier 2 (RunStateMachine) to execute individual routine steps.
     * It dynamically selects the optimal execution strategy based on context and constraints.
     * 
     * @param stepContext The execution context for this step
     * @param runContext The broader run context from Tier 2
     * @returns The execution result with outputs, resource usage, and metadata
     */
    async executeStep(
        stepContext: ExecutionContext,
        runContext: RunContext,
    ): Promise<StrategyExecutionResult> {
        const startTime = Date.now();
        const stepId = stepContext.stepId;

        logger.info("[UnifiedExecutor] Starting step execution", {
            stepId,
            stepType: stepContext.stepType,
            routineId: runContext.routineId,
        });

        // Emit event for step start
        await getEventBus().publish({
            ...EventUtils.createBaseEvent(
                EventTypes.STEP_STARTED,
                {
                    stepId,
                    stepType: stepContext.stepType,
                    strategy: stepContext.config?.strategy || "conversational",
                    estimatedResources: stepContext.resources,
                    runId: runContext.runId,
                    routineId: runContext.routineId,
                },
                EventUtils.createEventSource(3, "unified-executor"),
            ),
            metadata: EventUtils.createEventMetadata("fire-and-forget", "medium", {
                conversationId: runContext.conversationId,
                userId: stepContext.userData.id,
            }),
        });

        try {
            // 1. Choose execution strategy based on context
            const strategy = await this.strategySelector.getStrategy(
                stepContext,
                runContext.usageHints,
            );

            // Inject shared services into the strategy if it supports them
            if ("setToolOrchestrator" in strategy && typeof strategy.setToolOrchestrator === "function") {
                strategy.setToolOrchestrator(this.toolOrchestrator);
            }
            if ("setValidationEngine" in strategy && typeof strategy.setValidationEngine === "function") {
                strategy.setValidationEngine(this.validationEngine);
            }

            // 2. Reserve budget for this step
            const budgetReservation = await this.resourceManager.reserveBudget(
                stepContext.resources,
                stepContext.constraints,
                stepContext.userData.id,
                stepContext.swarmId,
            );

            // Emit resource allocation event
            if (budgetReservation.approved) {
                await getEventBus().publish({
                    ...EventUtils.createBaseEvent(
                        EventTypes.RESOURCE_ALLOCATED,
                        {
                            stepId,
                            credits: parseInt(budgetReservation.allocation.credits || "0"),
                            timeLimit: stepContext.constraints?.maxTime,
                            tools: budgetReservation.allocation.tools?.length || 0,
                            models: budgetReservation.allocation.models || [],
                            reservationId: budgetReservation.reservationId,
                        },
                        EventUtils.createEventSource(3, "resource-manager"),
                    ),
                    metadata: EventUtils.createEventMetadata("reliable", "medium", {
                        conversationId: runContext.conversationId,
                        userId: stepContext.userData.id,
                    }),
                });
            }

            if (!budgetReservation.approved) {
                const error = budgetReservation.reason || "Resource limit exceeded";
                await getEventBus().publish({
                    ...EventUtils.createBaseEvent(
                        EventTypes.RESOURCE_EXHAUSTED,
                        {
                            stepId,
                            constraints: stepContext.constraints,
                            reason: error,
                            resourceType: "credits",
                            attempted: stepContext.resources,
                        },
                        EventUtils.createEventSource(3, "resource-manager"),
                    ),
                    metadata: EventUtils.createEventMetadata("reliable", "high", {
                        conversationId: runContext.conversationId,
                        userId: stepContext.userData.id,
                    }),
                });
                return this.createErrorResult(error, strategy.type, startTime);
            }

            // Set the stepId on the reservation
            this.resourceManager.setStepId(budgetReservation.reservationId, stepId);

            // 3. Build input payload
            const preparedInputs = await this.ioProcessor.buildInputPayload(
                stepContext.inputs,
                runContext,
            );

            // 4. Execute step with selected strategy
            const executionResult = await this.executeWithStrategy(
                strategy,
                stepContext,
                preparedInputs,
                budgetReservation.allocation,
            );

            // 5. Validate outputs
            const validatedOutput = await this.validationEngine.validate(
                executionResult.result,
                stepContext.config.outputSchema,
            );

            // Emit output generation event  
            await getEventBus().publish({
                ...EventUtils.createBaseEvent(
                    EventTypes.STEP_COMPLETED,
                    {
                        stepId,
                        outputKeys: Object.keys(validatedOutput.data || {}),
                        outputSize: JSON.stringify(validatedOutput.data || {}).length,
                        validationPassed: validatedOutput.valid,
                        strategy: strategy.type,
                        duration: Date.now() - startTime,
                    },
                    EventUtils.createEventSource(3, "io-processor"),
                ),
                metadata: EventUtils.createEventMetadata("fire-and-forget", "low", {
                    conversationId: runContext.conversationId,
                    userId: stepContext.userData.id,
                }),
            });

            if (!validatedOutput.valid) {
                const error = `Output validation failed: ${validatedOutput.errors.join(", ")}`;
                await getEventBus().publish({
                    ...EventUtils.createBaseEvent(
                        EventTypes.STEP_FAILED,
                        {
                            stepId,
                            errors: validatedOutput.errors,
                            failureReason: "validation_failed",
                            strategy: strategy.type,
                            duration: Date.now() - startTime,
                        },
                        EventUtils.createEventSource(3, "validation-engine"),
                    ),
                    metadata: EventUtils.createEventMetadata("reliable", "high", {
                        conversationId: runContext.conversationId,
                        userId: stepContext.userData.id,
                    }),
                });
                return this.createErrorResult(error, strategy.type, startTime);
            }

            // 6. Finalize resource usage
            const usageReport = await this.resourceManager.finalizeUsage(
                budgetReservation.reservationId,
                executionResult.metadata.resourceUsage,
            );

            // 7. Export context for cross-tier synchronization
            await this.contextExporter.exportContext(stepId, {
                strategy: strategy.type,
                resourceUsage: usageReport,
                outputs: validatedOutput.data,
            });

            // 8. Emit completion event
            await getEventBus().publish({
                ...EventUtils.createBaseEvent(
                    EventTypes.STEP_COMPLETED,
                    {
                        stepId,
                        strategy: strategy.type,
                        duration: Date.now() - startTime,
                        resourceUsage: usageReport,
                        success: true,
                        runId: runContext.runId,
                        routineId: runContext.routineId,
                    },
                    EventUtils.createEventSource(3, "unified-executor"),
                ),
                metadata: EventUtils.createEventMetadata("reliable", "medium", {
                    conversationId: runContext.conversationId,
                    userId: stepContext.userData.id,
                }),
            });

            // 9. Return successful result
            return {
                success: true,
                result: validatedOutput.data,
                metadata: {
                    strategyType: strategy.type,
                    executionTime: Date.now() - startTime,
                    resourceUsage: usageReport,
                    confidence: executionResult.metadata.confidence,
                    fallbackUsed: executionResult.metadata.fallbackUsed,
                },
                feedback: {
                    outcome: "success",
                    performanceScore: this.calculatePerformanceScore(
                        executionResult,
                        Date.now() - startTime,
                    ),
                },
            };

        } catch (error) {
            logger.error("[UnifiedExecutor] Step execution failed", {
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            await getEventBus().publish({
                ...EventUtils.createBaseEvent(
                    EventTypes.EXECUTION_ERROR_OCCURRED,
                    {
                        stepId,
                        error: error instanceof Error ? error.message : String(error),
                        errorType: error instanceof Error ? error.constructor.name : "Error",
                        stack: error instanceof Error ? error.stack : undefined,
                        context: {
                            executionTime: Date.now() - startTime,
                            stepType: stepContext.stepType,
                        },
                    },
                    EventUtils.createEventSource(3, "unified-executor"),
                ),
                metadata: EventUtils.createEventMetadata("reliable", "critical", {
                    conversationId: runContext.conversationId,
                    userId: stepContext.userData.id,
                }),
            });

            return this.createErrorResult(
                error instanceof Error ? error.message : "Unknown execution error",
                "conversational", // Default fallback
                startTime,
            );
        }
    }

    /**
     * Executes step with the selected strategy
     */
    private async executeWithStrategy(
        strategy: ExecutionStrategy,
        context: ExecutionContext,
        preparedInputs: Record<string, unknown>,
        resourceAllocation: AvailableResources,
    ): Promise<StrategyExecutionResult> {
        // Create an execution context with prepared inputs
        const executionContext: ExecutionContext = {
            ...context,
            inputs: preparedInputs,
            resources: resourceAllocation,
        };

        // Configure tool orchestrator for this execution
        this.toolOrchestrator.configureForExecution(
            context.stepId,
            resourceAllocation.tools,
            this.resourceManager,
            {
                runId: context.runId,
                swarmId: context.swarmId,
                conversationId: context.conversationId,
                user: context.userData ? { id: context.userData.id, languages: ["en"] } : undefined,
            },
        );

        // Set up tool call event emission
        const originalExecuteTool = this.toolOrchestrator.executeTool?.bind(this.toolOrchestrator);
        if (originalExecuteTool) {
            this.toolOrchestrator.executeTool = async (toolName: string, params: any) => {
                const toolStartTime = Date.now();
                try {
                    const result = await originalExecuteTool(toolName, params);
                    await getEventBus().publish("tool.executed", {
                        stepId: context.stepId,
                        toolName,
                        duration: Date.now() - toolStartTime,
                        success: true,
                        timestamp: new Date().toISOString(),
                    });
                    return result;
                } catch (error) {
                    await getEventBus().publish("tool.executed", {
                        stepId: context.stepId,
                        toolName,
                        duration: Date.now() - toolStartTime,
                        success: false,
                        error: error instanceof Error ? error.message : String(error),
                        timestamp: new Date().toISOString(),
                    });
                    throw error;
                }
            };
        }

        // Execute with strategy
        const result = await strategy.execute(executionContext);

        // Update resource tracking with actual usage
        if (result.metadata.resourceUsage) {
            await this.resourceManager.trackUsage(
                context.stepId,
                result.metadata.resourceUsage,
            );
        }

        return result;
    }

    /**
     * Creates an error result
     */
    private createErrorResult(
        error: string,
        strategyType: ExecutionStrategy,
        startTime: number,
    ): StrategyExecutionResult {
        return {
            success: false,
            error,
            metadata: {
                strategyType,
                executionTime: Date.now() - startTime,
                resourceUsage: {},
                confidence: 0,
                fallbackUsed: false,
            },
            feedback: {
                outcome: "failure",
                performanceScore: 0,
                issues: [error],
            },
        };
    }

    /**
     * Calculates performance score based on execution results
     */
    private calculatePerformanceScore(
        result: StrategyExecutionResult,
        duration: number,
    ): number {
        let score = result.success ? 0.8 : 0.2;

        // Adjust for confidence
        score += result.metadata.confidence * 0.1;

        // Penalize for excessive duration
        const targetDuration = 30000; // 30 seconds
        if (duration < targetDuration) {
            score += 0.1;
        } else if (duration > targetDuration * 2) {
            score -= 0.1;
        }

        // Penalize for fallback usage
        if (result.metadata.fallbackUsed) {
            score -= 0.05;
        }

        return Math.max(0, Math.min(1, score));
    }

    /**
     * Provides tool orchestrator for direct tool access
     * Used by strategies that need to execute tools
     */
    getToolOrchestrator(): ToolOrchestrator {
        return this.toolOrchestrator;
    }

    /**
     * Provides resource manager for strategies
     */
    getResourceManager(): ResourceManager {
        return this.resourceManager;
    }

    // ===== TierCommunicationInterface Implementation =====

    private readonly activeExecutions = new Map<SwarmId, {
        status: ExecutionStates;
        startTime: Date;
        promise?: Promise<ExecutionResult>;
    }>();

    /**
     * Execute a request via the standard tier communication interface
     * This is the primary method for Tier 2 to delegate step execution to Tier 3
     */
    async execute<TInput extends StepExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        const { context, input, allocation, options } = request;
        const swarmId = context.swarmId;

        // Track execution
        this.activeExecutions.set(swarmId, {
            status: ExecutionStates.RUNNING,
            startTime: new Date(),
        });

        try {
            logger.info("[UnifiedExecutor] Starting tier execution", {
                swarmId,
                stepId: input.stepId,
                stepType: input.stepType,
                strategy: input.strategy,
            });

            // Emit execution started event
            await getEventBus().publish("tier3.execution.started", {
                swarmId,
                stepId: input.stepId,
                stepType: input.stepType,
                strategy: input.strategy,
                timestamp: new Date().toISOString(),
            });

            // Build enhanced context for step execution
            const stepContext: ExecutionContext = {
                ...context,
                stepId: input.stepId,
                stepType: input.stepType,
                inputs: input.parameters,
                config: {
                    strategy: input.strategy,
                    toolName: input.toolName,
                    ...context.config,
                },
                resources: context.resources,
                constraints: {
                    maxCredits: parseInt(allocation.maxCredits),
                    maxTime: allocation.maxDurationMs,
                    ...context.constraints,
                },
            };

            // Create a simplified run context for this execution
            const runContext: RunContext = {
                routineId: context.routineId || "unknown",
                swarmId,
                usageHints: {},
                variables: new Map(),
                state: {},
            };

            // Execute the step using our existing method
            const strategyResult = await this.executeStep(stepContext, runContext);

            // Convert StrategyExecutionResult to ExecutionResult
            const executionResult: ExecutionResult<TOutput> = {
                success: strategyResult.success,
                result: strategyResult.result as TOutput,
                outputs: strategyResult.result as Record<string, unknown>,
                resourcesUsed: {
                    creditsUsed: allocation.maxCredits, // Will be updated with actual usage
                    durationMs: strategyResult.metadata?.executionTime || 0,
                    memoryUsedMB: 0,
                    stepsExecuted: 1,
                    tokens: strategyResult.metadata?.resourceUsage?.tokens,
                    apiCalls: strategyResult.metadata?.resourceUsage?.apiCalls,
                    computeTime: strategyResult.metadata?.executionTime || 0,
                    cost: strategyResult.metadata?.resourceUsage?.cost,
                },
                duration: strategyResult.metadata?.executionTime || 0,
                context: stepContext,
                metadata: {
                    strategy: strategyResult.metadata?.strategyType,
                    version: "3.0.0",
                    performance: {
                        executionTime: strategyResult.metadata?.executionTime,
                        resourceUsage: strategyResult.metadata?.resourceUsage,
                        confidence: strategyResult.metadata?.confidence,
                    },
                    timestamp: new Date().toISOString(),
                },
                confidence: strategyResult.metadata?.confidence,
                performanceScore: strategyResult.feedback?.performanceScore,
            };

            // Update execution status
            this.activeExecutions.set(swarmId, {
                status: ExecutionStates.COMPLETED,
                startTime: this.activeExecutions.get(swarmId)!.startTime,
            });

            logger.info("[UnifiedExecutor] Tier execution completed", {
                swarmId,
                success: executionResult.success,
                duration: executionResult.duration,
            });

            // Emit execution completed event
            await getEventBus().publish("tier3.execution.completed", {
                swarmId,
                success: executionResult.success,
                duration: executionResult.duration,
                resourcesUsed: executionResult.resourcesUsed,
                confidence: executionResult.confidence,
                performanceScore: executionResult.performanceScore,
                timestamp: new Date().toISOString(),
            });

            return executionResult;

        } catch (error) {
            // Update execution status
            this.activeExecutions.set(swarmId, {
                status: ExecutionStates.FAILED,
                startTime: this.activeExecutions.get(swarmId)!.startTime,
            });

            logger.error("[UnifiedExecutor] Tier execution failed", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Emit execution failed event
            await getEventBus().publish("tier3.execution.failed", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
                errorType: error instanceof Error ? error.constructor.name : "Error",
                timestamp: new Date().toISOString(),
            });

            // Return error result
            const errorResult: ExecutionResult<TOutput> = {
                success: false,
                error: {
                    code: "TIER3_EXECUTION_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error",
                    tier: "tier3",
                    type: error instanceof Error ? error.constructor.name : "Error",
                },
                resourcesUsed: {
                    creditsUsed: "0",
                    durationMs: Date.now() - this.activeExecutions.get(swarmId)!.startTime.getTime(),
                    memoryUsedMB: 0,
                    stepsExecuted: 0,
                },
                duration: Date.now() - this.activeExecutions.get(swarmId)!.startTime.getTime(),
                context,
                metadata: {
                    strategy: input.strategy,
                    version: "3.0.0",
                    timestamp: new Date().toISOString(),
                },
                confidence: 0.0,
                performanceScore: 0.0,
            };

            return errorResult;
        }
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

            logger.info("[UnifiedExecutor] Execution cancelled", { swarmId });
        }
    }

    /**
     * Get tier capabilities for discovery
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: "tier3",
            maxConcurrency: 100,
            resourceLimits: {
                maxCredits: "10000",
                maxDurationMs: 300000,
                maxMemoryMB: 1024,
            },
        };
    }
}
