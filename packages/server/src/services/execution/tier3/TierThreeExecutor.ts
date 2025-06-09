import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { UnifiedExecutor } from "./engine/unifiedExecutor.js";
import { StrategySelector } from "./engine/strategySelector.js";
import { ToolOrchestrator } from "./engine/toolOrchestrator.js";
import { ResourceManager } from "./engine/resourceManager.js";
import { ValidationEngine } from "./engine/validationEngine.js";
import { IOProcessor } from "./engine/ioProcessor.js";
import { TelemetryShim } from "../cross-cutting/monitoring/telemetryShim.js";
import { ExecutionRunContext } from "./context/runContext.js";
import { ContextExporter } from "./context/contextExporter.js";
import { ConversationalStrategy } from "./strategies/conversationalStrategy.js";
import { ReasoningStrategy } from "./strategies/reasoningStrategy.js";
import { DeterministicStrategy } from "./strategies/deterministicStrategy.js";
import {
    type ExecutionContext,
    type ExecutionId,
    type ExecutionResult,
    type ExecutionStatus,
    type StepExecutionInput,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    generatePk,
} from "@vrooli/shared";

/**
 * Tier Three Executor
 * 
 * Main entry point for Tier 3 execution intelligence.
 * Manages step execution, strategy selection, and tool orchestration.
 */
export class TierThreeExecutor implements TierCommunicationInterface {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly unifiedExecutor: UnifiedExecutor;
    private readonly contextExporter: ContextExporter;

    // Track active executions for interface compliance
    private readonly activeExecutions: Map<ExecutionId, { status: ExecutionStatus; startTime: Date; context: ExecutionContext }> = new Map();

    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
        
        // Initialize components
        const strategySelector = new StrategySelector(logger);
        const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
        const resourceManager = new ResourceManager(logger);
        const validationEngine = new ValidationEngine(logger);
        const ioProcessor = new IOProcessor(logger);
        const telemetryShim = new TelemetryShim(eventBus, 'tier3', { enabled: true });
        
        // Register strategies
        strategySelector.registerStrategy(new ConversationalStrategy(logger));
        strategySelector.registerStrategy(new ReasoningStrategy(logger));
        strategySelector.registerStrategy(new DeterministicStrategy(logger));
        
        // Create unified executor
        this.unifiedExecutor = new UnifiedExecutor(
            eventBus,
            logger,
            strategySelector,
            toolOrchestrator,
            resourceManager,
            validationEngine,
            ioProcessor,
            telemetryShim,
        );
        
        // Create context exporter
        this.contextExporter = new ContextExporter(logger);
        
        // Setup event handlers
        this.setupEventHandlers();
        
        this.logger.info("[TierThreeExecutor] Initialized");
    }

    /**
     * Executes a single step
     */
    async executeStep(context: ExecutionContext): Promise<ExecutionResult> {
        this.logger.info("[TierThreeExecutor] Executing step", {
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
                    routineId: context.routineId || '',
                    routineName: context.routineName || '',
                    currentStepId: context.stepId,
                    userData: context.userData || { id: '' },
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
            
            // Emit completion event
            if (result.success) {
                await this.eventBus.publish("step.completed", {
                    runId: context.runId,
                    stepId: context.stepId,
                    outputs: result.result,
                    metadata: result.metadata,
                });
            } else {
                await this.eventBus.publish("step.failed", {
                    runId: context.runId,
                    stepId: context.stepId,
                    error: result.error,
                    metadata: result.metadata,
                });
            }
            
            return result;
            
        } catch (error) {
            this.logger.error("[TierThreeExecutor] Step execution failed", {
                stepId: context.stepId,
                error: error instanceof Error ? error.message : String(error),
            });
            
            // Emit failure event
            await this.eventBus.publish("step.failed", {
                runId: context.runId,
                stepId: context.stepId,
                error: error instanceof Error ? error.message : "Unknown error",
            });
            
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
        this.logger.info("[TierThreeExecutor] Shutting down");
        // Currently no persistent state to clean up
    }

    /**
     * TierCommunicationInterface implementation
     */

    /**
     * Execute a step execution request
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
            context,
        });

        try {
            this.logger.info("[TierThreeExecutor] Starting tier execution", {
                executionId,
                stepId: input.stepId,
                stepType: input.stepType,
            });

            // Create execution context from request
            const executionContext: ExecutionContext = {
                ...context,
                stepId: input.stepId,
                stepType: input.stepType as any,
                strategy: input.strategy as any,
                inputs: input.parameters,
                config: {
                    toolName: input.toolName,
                    strategy: input.strategy,
                    ...options,
                },
            };

            // Execute the step
            const result = await this.executeStep(executionContext);

            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.COMPLETED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                context,
            });

            return result as ExecutionResult<TOutput>;

        } catch (error) {
            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.FAILED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                context,
            });

            this.logger.error("[TierThreeExecutor] Tier execution failed", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });

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
                    strategy: 'step_execution',
                    version: '1.0.0',
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

            this.logger.info("[TierThreeExecutor] Execution cancelled", { executionId });
        }
    }

    /**
     * Get tier capabilities
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: 'tier3',
            supportedInputTypes: ['StepExecutionInput'],
            supportedStrategies: ['conversational', 'reasoning', 'deterministic'],
            maxConcurrency: 20,
            estimatedLatency: {
                p50: 5000,
                p95: 30000,
                p99: 120000,
            },
            resourceLimits: {
                maxCredits: '10000',
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
        this.eventBus.on("tool.approval_required", async (event) => {
            // Forward to appropriate handlers (e.g., UI notification service)
            this.logger.info("[TierThreeExecutor] Tool approval required", event.data);
        });
        
        // Handle resource alerts
        this.eventBus.on("resources.exhausted", async (event) => {
            this.logger.warn("[TierThreeExecutor] Resources exhausted", event.data);
        });
        
        // Handle validation failures
        this.eventBus.on("validation.failed", async (event) => {
            this.logger.error("[TierThreeExecutor] Validation failed", event.data);
        });
    }
}