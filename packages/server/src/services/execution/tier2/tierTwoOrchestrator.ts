// AI_CHECK: STARTUP_ERRORS=4 | LAST: 2025-06-25 | FIXED: Import/export mismatch causing module not found error
import { type Logger } from "winston";
import { type IEventBus } from "../../events/types.js";
import { BaseComponent } from "../shared/BaseComponent.js";
// Socket events now handled through unified event system
import { EventTypes } from "../../events/index.js";
import { NavigatorRegistry } from "./navigation/navigatorRegistry.js";
import { UnifiedRunStateMachine } from "./orchestration/unifiedRunStateMachine.js";
// Legacy components removed - functionality consolidated in UnifiedRunStateMachine
import {
    type ExecutionResult,
    type RoutineExecutionInput,
    type Run,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
} from "@vrooli/shared";
import { type ISwarmContextManager } from "../shared/SwarmContextManager.js";
import { type IRunStateStore, getRunStateStore } from "./state/runStateStore.js";
import { MOISEGate } from "./validation/moiseGate.js";

/**
 * Tier Two Orchestrator - Main Entry Point for Tier 2 Process Intelligence
 * 
 * This class serves as the public API and facade for Tier 2 in the three-tier execution architecture.
 * It implements the TierCommunicationInterface to enable cross-tier communication with Tier 1 
 * (Swarm Coordination) and Tier 3 (Tool Execution).
 * 
 * ## Architecture Position:
 * ```
 * Tier 1 (Swarm) → TierTwoOrchestrator → Tier 3 (Execution)
 *                          ↓
 *                  UnifiedRunStateMachine (Internal Implementation)
 * ```
 * 
 * ## Key Responsibilities:
 * 1. **Initialization**: Sets up all Tier 2 dependencies (state store, navigators, MOISE+ gate)
 * 2. **API Gateway**: Provides stable public interface for Tier 1 to execute routines
 * 3. **Delegation**: Delegates all complex execution logic to UnifiedRunStateMachine
 * 4. **Event Coordination**: Manages cross-tier event subscriptions and emissions
 * 5. **Resource Management**: Initializes and manages shared resources
 * 
 * ## Relationship to Other Components:
 * - **UnifiedRunStateMachine**: The actual implementation of all Tier 2 logic. TierTwoOrchestrator
 *   creates and manages a single instance of this state machine.
 * - **RunOrchestrator**: NOT directly used. UnifiedRunStateMachine provides its own run management
 *   implementation but exposes a compatible interface via getRunOrchestrator().
 * - **NavigatorRegistry**: Shared dependency that manages routine navigation strategies
 * - **MOISEGate**: Shared dependency for organizational permission validation
 * 
 * ## Design Rationale:
 * This thin orchestration layer exists to:
 * - Provide a stable public API that won't change even if internal implementation does
 * - Handle initialization complexity that shouldn't be in the state machine
 * - Enable easier testing by separating concerns
 * - Follow the Facade pattern for simplified external interaction
 * 
 * @see UnifiedRunStateMachine - The comprehensive state machine implementation
 * @see {@link https://github.com/Vrooli/Vrooli/blob/main/docs/architecture/execution/tiers/tier2-process-intelligence} - Tier 2 Architecture Documentation
 */
export class TierTwoOrchestrator extends BaseComponent implements TierCommunicationInterface {
    private readonly tier3Executor: TierCommunicationInterface;

    // Core unified architecture - single comprehensive state machine
    private readonly unifiedStateMachine: UnifiedRunStateMachine;

    // Shared dependencies used by unified state machine
    private readonly navigatorRegistry: NavigatorRegistry;
    private readonly moiseGate: MOISEGate;
    private readonly stateStore: IRunStateStore;
    private readonly contextManager?: ISwarmContextManager;


    constructor(logger: Logger, eventBus: IEventBus, tier3Executor: TierCommunicationInterface, contextManager?: ISwarmContextManager) {
        super(logger, eventBus, "TierTwoOrchestrator");
        this.tier3Executor = tier3Executor;
        this.contextManager = contextManager;

        // Initialize shared dependencies
        this.stateStore = getRunStateStore();
        this.initializeStateStore();
        this.navigatorRegistry = new NavigatorRegistry(logger);
        this.moiseGate = new MOISEGate(logger);

        // Initialize unified state machine with all dependencies
        this.unifiedStateMachine = new UnifiedRunStateMachine(
            logger,
            eventBus,
            this.navigatorRegistry,
            this.moiseGate,
            this.stateStore,
            tier3Executor,
            contextManager,
        );

        // Setup event handlers
        this.setupEventHandlers();

        this.logger.info("[TierTwoOrchestrator] Initialized with unified architecture");
    }

    /**
     * Initialize the state store asynchronously
     */
    private async initializeStateStore(): Promise<void> {
        try {
            await this.stateStore.initialize();
            this.logger.info("[TierTwoOrchestrator] State store initialized");
        } catch (error) {
            this.logger.error("[TierTwoOrchestrator] Failed to initialize state store", { error });
            throw error;
        }
    }

    /**
     * Starts a new run using UnifiedRunStateMachine for comprehensive execution
     */
    async startRun(config: {
        runId: string;
        swarmId: string;
        routineVersionId: string;
        routine: any; // Routine data from database
        inputs: Record<string, unknown>;
        config: {
            strategy?: string;
            model: string;
            maxSteps: number;
            timeout: number;
        };
        userId: string;
    }): Promise<void> {
        this.logger.info("[TierTwoOrchestrator] Starting run with unified architecture", {
            runId: config.runId,
            routineVersionId: config.routineVersionId,
        });

        try {
            // Use unified state machine for comprehensive execution
            const runOrchestrator = this.unifiedStateMachine.getRunOrchestrator();

            // Create and start the run
            const run = await runOrchestrator.createRun({
                routineId: config.routineVersionId,
                userId: config.userId,
                inputs: config.inputs,
                config: {
                    maxRetries: 3,
                    timeout: config.config.timeout,
                    parallel: true,
                    strategy: config.config.strategy,
                    maxSteps: config.config.maxSteps,
                },
                swarmId: config.swarmId,
            });

            // Start the run
            await runOrchestrator.startRun(run.id);

            // Emit run started event using unified event system
            await this.publishUnifiedEvent(EventTypes.ROUTINE_STARTED, {
                runId: config.runId,
                routineVersionId: config.routineVersionId,
                swarmId: config.swarmId,
                routineName: run.routineName,
                userId: run.userId,
            }, {
                deliveryGuarantee: "reliable",
                priority: "high",
                tags: ["routine", "orchestration"],
                conversationId: config.swarmId,
            });

            // Emit config update through unified event system
            await this.publishUnifiedEvent(EventTypes.CONFIG_SWARM_UPDATED, {
                entityType: "swarm",
                entityId: config.swarmId,
                config: {
                    // Update task progress - this would be mapped from the run
                    subtasks: [], // TODO: Map run steps to subtasks for display
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "0",
                        lastProcessingCycleEndedAt: Date.now(),
                    },
                },
            }, {
                deliveryGuarantee: "fire-and-forget",
                priority: "medium",
                conversationId: config.swarmId,
            });

        } catch (error) {
            this.logger.error("[TierTwoOrchestrator] Failed to start run", {
                runId: config.runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Gets run status
     */
    async getRunStatus(runId: string): Promise<{
        progress?: number;
        currentStep?: string;
        errors?: string[];
    } | null> {
        try {
            const run = await this.stateStore.getRun(runId);
            if (!run) {
                return null;
            }

            // NEW: Use unified state machine run orchestrator
            const runOrchestrator = this.unifiedStateMachine.getRunOrchestrator();
            const currentStep = run.progress?.currentStepId;

            return {
                progress: this.calculateProgress(run),
                currentStep,
                errors: run.errors,
            };

        } catch (error) {
            this.logger.error("[TierTwoOrchestrator] Failed to get run status", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    /**
     * Cancels a run
     */
    async cancelRun(runId: string, reason?: string): Promise<void> {
        // Use unified state machine run orchestrator
        const runOrchestrator = this.unifiedStateMachine.getRunOrchestrator();

        // Check if run exists
        const run = await runOrchestrator.getRunState(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        await runOrchestrator.cancelRun(runId, reason || "User cancelled");

        // Emit cancellation event
        await this.publishUnifiedEvent(
            "run.cancelled",
            {
                runId,
                reason,
            },
            {
                priority: "high",
                deliveryGuarantee: "reliable",
            },
        );
    }

    /**
     * Shuts down the orchestrator
     */
    async shutdown(): Promise<void> {
        this.logger.info("[TierTwoOrchestrator] Shutting down");

        try {
            // Stop the unified state machine (this will cancel all active runs)
            await this.unifiedStateMachine.stop();
        } catch (error) {
            this.logger.error("[TierTwoOrchestrator] Error during shutdown", {
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Private helper methods
     */
    private setupEventHandlers(): void {
        // Handle step completion from Tier 3
        this.eventBus.on("step.completed", async (event) => {
            const { runId, stepId, outputs } = event.data;
            // Use unified state machine run orchestrator
            const runOrchestrator = this.unifiedStateMachine.getRunOrchestrator();

            // Update run progress
            await runOrchestrator.updateProgress(runId, {
                currentStepId: stepId,
                completedSteps: [...(await runOrchestrator.getRunState(runId))?.progress?.completedSteps || [], stepId],
            });
        });

        // Handle step failure from Tier 3
        this.eventBus.on("step.failed", async (event) => {
            const { runId, stepId, error } = event.data;
            // Use unified state machine run orchestrator
            const runOrchestrator = this.unifiedStateMachine.getRunOrchestrator();

            // Fail the run
            await runOrchestrator.failRun(runId, `Step ${stepId} failed: ${error}`);
        });

        // Handle performance insights
        this.eventBus.on("performance.insight", async (event) => {
            const { runId } = event.data;
            // Performance insights can be logged or used for optimization
            this.logger.info("[TierTwoOrchestrator] Performance insight received", {
                runId,
                insight: event.data,
            });
        });
    }

    private calculateProgress(run: Run): number {
        if (!run.metrics) return 0;

        const { stepsCompleted = 0, totalSteps = 1 } = run.metrics;
        return Math.min((stepsCompleted / totalSteps) * 100, 100);
    }

    /**
     * TierCommunicationInterface implementation
     */

    /**
     * Execute a routine execution request using UnifiedRunStateMachine
     */
    async execute<TInput extends RoutineExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        // Delegate to unified state machine for complete execution
        return await this.unifiedStateMachine.execute(request) as ExecutionResult<TOutput>;
    }

    /**
     * Get tier capabilities from UnifiedRunStateMachine
     */
    async getCapabilities(): Promise<TierCapabilities> {
        // Get capabilities from unified state machine
        return await this.unifiedStateMachine.getCapabilities();
    }

    /**
     * Get tier status information from UnifiedRunStateMachine
     */
    getTierStatus() {
        // Get status from unified state machine
        return this.unifiedStateMachine.getTierStatus();
    }

}
