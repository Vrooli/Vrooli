import { type Logger } from "winston";
import { 
    type ExecutionContext, 
    type ExecutionStrategy, 
    type StrategyExecutionResult,
    type StrategyType,
    type ResourceUsage,
    type UnifiedExecutorConfig,
    type AvailableResources,
    type ExecutionConstraints,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    type ExecutionResult,
    type ExecutionStatus,
    type ExecutionId,
    type StepExecutionInput,
    type TierCapabilities,
    type ResourceAllocation,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { StrategySelector } from "./strategySelector.js";
import { ResourceManager } from "./resourceManager.js";
import { IOProcessor } from "./ioProcessor.js";
import { ToolOrchestrator } from "./toolOrchestrator.js";
import { ValidationEngine } from "./validationEngine.js";
import { TelemetryShim } from "../../cross-cutting/monitoring/telemetryShim.js";
import { type RunContext } from "../context/runContext.js";
import { ContextExporter } from "../context/contextExporter.js";
import { type IntegratedToolRegistry } from "../../integration/mcp/toolRegistry.js";
import { type RollingHistory } from "../../cross-cutting/monitoring/index.js";

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
 * - Telemetry emission for monitoring and learning
 * 
 * Implements TierCommunicationInterface for standardized inter-tier communication.
 */
export class UnifiedExecutor implements TierCommunicationInterface {
    private readonly strategySelector: StrategySelector;
    private readonly resourceManager: ResourceManager;
    private readonly ioProcessor: IOProcessor;
    private readonly toolOrchestrator: ToolOrchestrator;
    private readonly validationEngine: ValidationEngine;
    private readonly telemetryShim: TelemetryShim;
    private readonly contextExporter: ContextExporter;
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly rollingHistory?: RollingHistory;

    constructor(
        config: UnifiedExecutorConfig,
        eventBus: EventBus,
        logger: Logger,
        toolRegistry?: IntegratedToolRegistry,
        rollingHistory?: RollingHistory,
    ) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.rollingHistory = rollingHistory;

        // Initialize components
        this.toolOrchestrator = new ToolOrchestrator(eventBus, logger, toolRegistry);
        this.validationEngine = new ValidationEngine(logger);
        this.resourceManager = new ResourceManager(logger, eventBus);
        this.ioProcessor = new IOProcessor(logger);
        this.telemetryShim = new TelemetryShim(eventBus, config.telemetryEnabled);
        this.contextExporter = new ContextExporter(eventBus, logger);
        
        // Initialize strategy selector with enhanced configuration
        const strategyFactoryConfig = {
            ...config.strategyFactory,
            toolOrchestrator: this.toolOrchestrator,
            validationEngine: this.validationEngine,
        };
        this.strategySelector = new StrategySelector(strategyFactoryConfig, logger, this.toolOrchestrator, this.validationEngine);
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

        this.logger.info("[UnifiedExecutor] Starting step execution", {
            stepId,
            stepType: stepContext.stepType,
            routineId: runContext.routineId,
        });

        // Emit telemetry for step start
        await this.telemetryShim.emitStepStarted(stepId, {
            stepType: stepContext.stepType,
            strategy: stepContext.config?.strategy || StrategyType.CONVERSATIONAL,
            estimatedResources: stepContext.resources,
        });

        try {
            // 1. Choose execution strategy based on context
            const strategy = await this.strategySelector.selectStrategy(
                stepContext,
                runContext.usageHints,
            );

            // Inject shared services into the strategy if it supports them
            if ('setToolOrchestrator' in strategy && typeof strategy.setToolOrchestrator === 'function') {
                strategy.setToolOrchestrator(this.toolOrchestrator);
            }
            if ('setValidationEngine' in strategy && typeof strategy.setValidationEngine === 'function') {
                strategy.setValidationEngine(this.validationEngine);
            }

            this.logger.debug(`[UnifiedExecutor] Selected strategy: ${strategy.type}`, {
                stepId,
                strategyName: strategy.name,
            });

            // Emit strategy selection telemetry
            await this.telemetryShim.emitStrategySelected(stepId, {
                declared: stepContext.config?.strategy || StrategyType.CONVERSATIONAL,
                selected: strategy.type,
                reason: "Based on context and usage hints",
            });

            // 2. Reserve budget for this step
            const budgetReservation = await this.resourceManager.reserveBudget(
                stepContext.resources,
                stepContext.constraints,
                stepContext.userId,
                stepContext.swarmId,
            );

            // Emit resource allocation telemetry
            if (budgetReservation.approved) {
                await this.telemetryShim.emitResourceAllocated(stepId, {
                    credits: parseInt(budgetReservation.allocation.credits || "0"),
                    timeLimit: stepContext.constraints?.maxTime,
                    tools: budgetReservation.allocation.tools?.length || 0,
                    models: budgetReservation.allocation.models || [],
                });
            }

            if (!budgetReservation.approved) {
                const error = budgetReservation.reason || "Resource limit exceeded";
                await this.telemetryShim.emitLimitExceeded(stepId, stepContext.constraints);
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

            // Emit output generation telemetry
            await this.telemetryShim.emitOutputGenerated(stepId, {
                outputKeys: Object.keys(validatedOutput.data || {}),
                size: JSON.stringify(validatedOutput.data || {}).length,
                validationPassed: validatedOutput.valid,
            });

            if (!validatedOutput.valid) {
                const error = `Output validation failed: ${validatedOutput.errors.join(", ")}`;
                await this.telemetryShim.emitValidationFailure(stepId, validatedOutput.errors);
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

            // 8. Emit completion telemetry
            await this.telemetryShim.emitStepCompleted(stepId, {
                strategy: strategy.type,
                duration: Date.now() - startTime,
                resourceUsage: usageReport,
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
            this.logger.error("[UnifiedExecutor] Step execution failed", {
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            await this.telemetryShim.emitExecutionError(stepId, error);

            return this.createErrorResult(
                error instanceof Error ? error.message : "Unknown execution error",
                StrategyType.CONVERSATIONAL, // Default fallback
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
                user: context.userId ? { id: context.userId, languages: ["en"] } : undefined,
            },
        );

        // Set up tool call telemetry
        const originalExecuteTool = this.toolOrchestrator.executeTool?.bind(this.toolOrchestrator);
        if (originalExecuteTool) {
            this.toolOrchestrator.executeTool = async (toolName: string, params: any) => {
                const toolStartTime = Date.now();
                try {
                    const result = await originalExecuteTool(toolName, params);
                    await this.telemetryShim.emitToolCall(context.stepId, {
                        toolName,
                        duration: Date.now() - toolStartTime,
                        success: true,
                    });
                    return result;
                } catch (error) {
                    await this.telemetryShim.emitToolCall(context.stepId, {
                        toolName,
                        duration: Date.now() - toolStartTime,
                        success: false,
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
        strategyType: StrategyType,
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
    
    private readonly activeExecutions = new Map<ExecutionId, {
        status: ExecutionStatus;
        startTime: Date;
        promise?: Promise<ExecutionResult>;
    }>();

    /**
     * Execute a request via the standard tier communication interface
     * This is the primary method for Tier 2 to delegate step execution to Tier 3
     */
    async execute<TInput extends StepExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>> {
        const { context, input, allocation, options } = request;
        const executionId = context.executionId;

        // Track execution
        this.activeExecutions.set(executionId, {
            status: ExecutionStatus.RUNNING,
            startTime: new Date(),
        });

        try {
            this.logger.info("[UnifiedExecutor] Starting tier execution", {
                executionId,
                stepId: input.stepId,
                stepType: input.stepType,
                strategy: input.strategy,
            });

            // Track execution in rolling history if available
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier3.execution.started',
                    tier: 'tier3',
                    component: 'unified-executor',
                    data: {
                        executionId,
                        stepId: input.stepId,
                        stepType: input.stepType,
                        strategy: input.strategy,
                    },
                });
            }

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
                routineId: context.routineId || 'unknown',
                executionId,
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
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.COMPLETED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
            });

            this.logger.info("[UnifiedExecutor] Tier execution completed", {
                executionId,
                success: executionResult.success,
                duration: executionResult.duration,
            });

            // Track completion in rolling history
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier3.execution.completed',
                    tier: 'tier3',
                    component: 'unified-executor',
                    data: {
                        executionId,
                        success: executionResult.success,
                        duration: executionResult.duration,
                        resourcesUsed: executionResult.resourcesUsed,
                        confidence: executionResult.confidence,
                        performanceScore: executionResult.performanceScore,
                    },
                });
            }

            return executionResult;

        } catch (error) {
            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.FAILED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
            });

            this.logger.error("[UnifiedExecutor] Tier execution failed", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Track failure in rolling history
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier3.execution.failed',
                    tier: 'tier3',
                    component: 'unified-executor',
                    data: {
                        executionId,
                        error: error instanceof Error ? error.message : String(error),
                        errorType: error instanceof Error ? error.constructor.name : 'Error',
                    },
                });
            }

            // Return error result
            const errorResult: ExecutionResult<TOutput> = {
                success: false,
                error: {
                    code: 'TIER3_EXECUTION_FAILED',
                    message: error instanceof Error ? error.message : 'Unknown error',
                    tier: 'tier3',
                    type: error instanceof Error ? error.constructor.name : 'Error',
                },
                resourcesUsed: {
                    creditsUsed: '0',
                    durationMs: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                    memoryUsedMB: 0,
                    stepsExecuted: 0,
                },
                duration: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
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
    async getExecutionStatus(executionId: ExecutionId): Promise<ExecutionStatus> {
        const execution = this.activeExecutions.get(executionId);
        return execution?.status || ExecutionStatus.COMPLETED;
    }

    /**
     * Cancel a running execution
     */
    async cancelExecution(executionId: ExecutionId): Promise<void> {
        const execution = this.activeExecutions.get(executionId);
        if (execution) {
            this.activeExecutions.set(executionId, {
                ...execution,
                status: ExecutionStatus.CANCELLED,
            });

            this.logger.info("[UnifiedExecutor] Execution cancelled", { executionId });
        }
    }

    /**
     * Get tier capabilities for discovery
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: 'tier3',
            supportedInputTypes: ['StepExecutionInput'],
            supportedStrategies: ['conversational', 'reasoning', 'deterministic'],
            maxConcurrency: 100,
            estimatedLatency: {
                p50: 1000,
                p95: 5000,
                p99: 10000,
            },
            resourceLimits: {
                maxCredits: '10000',
                maxDurationMs: 300000,
                maxMemoryMB: 1024,
            },
        };
    }
}
