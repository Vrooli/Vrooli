import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { UnifiedExecutor } from "./engine/unifiedExecutor.js";
import { StrategySelector } from "./engine/strategySelector.js";
import { ToolOrchestrator } from "./engine/toolOrchestrator.js";
import { ResourceManager } from "./engine/resourceManager.js";
import { ValidationEngine } from "./engine/validationEngine.js";
import { IOProcessor } from "./engine/ioProcessor.js";
import { TelemetryShim } from "./engine/telemetryShim.js";
import { ExecutionRunContext } from "./context/runContext.js";
import { ContextExporter } from "./context/contextExporter.js";
import { ConversationalStrategy } from "./strategies/conversationalStrategy.js";
import { ReasoningStrategy } from "./strategies/reasoningStrategy.js";
import { DeterministicStrategy } from "./strategies/deterministicStrategy.js";
import {
    type ExecutionContext,
    type ExecutionResult,
} from "@vrooli/shared";

/**
 * Tier Three Executor
 * 
 * Main entry point for Tier 3 execution intelligence.
 * Manages step execution, strategy selection, and tool orchestration.
 */
export class TierThreeExecutor {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly unifiedExecutor: UnifiedExecutor;
    private readonly contextExporter: ContextExporter;

    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
        
        // Initialize components
        const strategySelector = new StrategySelector(logger);
        const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
        const resourceManager = new ResourceManager(logger);
        const validationEngine = new ValidationEngine(logger);
        const ioProcessor = new IOProcessor(logger);
        const telemetryShim = new TelemetryShim(eventBus, logger);
        
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