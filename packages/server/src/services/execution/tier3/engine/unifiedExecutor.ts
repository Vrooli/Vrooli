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
} from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/eventBus.js";
import { StrategySelector } from "./strategySelector.js";
import { ResourceManager } from "./resourceManager.js";
import { IOProcessor } from "./ioProcessor.js";
import { ToolOrchestrator } from "./toolOrchestrator.js";
import { ValidationEngine } from "./validationEngine.js";
import { TelemetryShim } from "./telemetryShim.js";
import { RunContext } from "../context/runContext.js";
import { ContextExporter } from "../context/contextExporter.js";

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
 */
export class UnifiedExecutor {
    private readonly strategySelector: StrategySelector;
    private readonly resourceManager: ResourceManager;
    private readonly ioProcessor: IOProcessor;
    private readonly toolOrchestrator: ToolOrchestrator;
    private readonly validationEngine: ValidationEngine;
    private readonly telemetryShim: TelemetryShim;
    private readonly contextExporter: ContextExporter;
    private readonly eventBus: EventBus;
    private readonly logger: Logger;

    constructor(
        config: UnifiedExecutorConfig,
        eventBus: EventBus,
        logger: Logger,
    ) {
        this.eventBus = eventBus;
        this.logger = logger;

        // Initialize components
        this.strategySelector = new StrategySelector(config.strategyFactory, logger);
        this.resourceManager = new ResourceManager(logger);
        this.ioProcessor = new IOProcessor(logger);
        this.toolOrchestrator = new ToolOrchestrator(eventBus, logger);
        this.validationEngine = new ValidationEngine(logger);
        this.telemetryShim = new TelemetryShim(eventBus, config.telemetryEnabled);
        this.contextExporter = new ContextExporter(eventBus, logger);
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

        this.logger.info(`[UnifiedExecutor] Starting step execution`, {
            stepId,
            stepType: stepContext.stepType,
            routineId: runContext.routineId,
        });

        try {
            // 1. Choose execution strategy based on context
            const strategy = await this.strategySelector.selectStrategy(
                stepContext,
                runContext.usageHints,
            );

            this.logger.debug(`[UnifiedExecutor] Selected strategy: ${strategy.type}`, {
                stepId,
                strategyName: strategy.name,
            });

            // 2. Reserve budget for this step
            const budgetReservation = await this.resourceManager.reserveBudget(
                stepContext.resources,
                stepContext.constraints,
            );

            if (!budgetReservation.approved) {
                const error = "Resource limit exceeded";
                await this.telemetryShim.emitLimitExceeded(stepId, stepContext.constraints);
                return this.createErrorResult(error, strategy.type, startTime);
            }

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
            this.logger.error(`[UnifiedExecutor] Step execution failed`, {
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
        );

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
}