import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { RunStateMachine } from "./orchestration/runStateMachine.js";
import { NavigatorRegistry } from "./navigation/navigatorRegistry.js";
import { BranchCoordinator } from "./orchestration/branchCoordinator.js";
import { StepExecutor } from "./orchestration/stepExecutor.js";
import { ContextManager } from "./context/contextManager.js";
import { CheckpointManager } from "./persistence/checkpointManager.js";
import { PerformanceMonitor } from "./intelligence/performanceMonitor.js";
import { PathOptimizer } from "./intelligence/pathOptimizer.js";
import { MOISEGate } from "./validation/moiseGate.js";
import { type InMemoryRunStateStore } from "./state/runStateStore.js";
import {
    type RunStatus,
    type Run,
    RunState,
} from "@vrooli/shared";

/**
 * Tier Two Orchestrator
 * 
 * Main entry point for Tier 2 process intelligence.
 * Manages run lifecycle, routine navigation, and orchestration.
 */
export class TierTwoOrchestrator {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly runMachines: Map<string, RunStateMachine> = new Map();
    private readonly navigatorRegistry: NavigatorRegistry;
    private readonly branchCoordinator: BranchCoordinator;
    private readonly stepExecutor: StepExecutor;
    private readonly contextManager: ContextManager;
    private readonly checkpointManager: CheckpointManager;
    private readonly performanceMonitor: PerformanceMonitor;
    private readonly pathOptimizer: PathOptimizer;
    private readonly moiseGate: MOISEGate;
    private readonly stateStore: InMemoryRunStateStore;

    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
        
        // Initialize components
        this.stateStore = new InMemoryRunStateStore(logger);
        this.navigatorRegistry = new NavigatorRegistry(logger);
        this.branchCoordinator = new BranchCoordinator(logger, eventBus);
        this.stepExecutor = new StepExecutor(logger, eventBus);
        this.contextManager = new ContextManager(logger);
        this.checkpointManager = new CheckpointManager(logger);
        this.performanceMonitor = new PerformanceMonitor(logger, eventBus);
        this.pathOptimizer = new PathOptimizer(logger);
        this.moiseGate = new MOISEGate(logger);
        
        // Setup event handlers
        this.setupEventHandlers();
        
        this.logger.info("[TierTwoOrchestrator] Initialized");
    }

    /**
     * Starts a new run
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
        this.logger.info("[TierTwoOrchestrator] Starting run", {
            runId: config.runId,
            routineVersionId: config.routineVersionId,
        });

        try {
            // Create run state machine
            const stateMachine = new RunStateMachine(
                this.logger,
                this.eventBus,
                this.stateStore,
                this.navigatorRegistry,
                this.branchCoordinator,
                this.stepExecutor,
                this.contextManager,
                this.checkpointManager,
                this.performanceMonitor,
                this.pathOptimizer,
                this.moiseGate,
            );

            this.runMachines.set(config.runId, stateMachine);

            // Start the run
            await stateMachine.start({
                runId: config.runId,
                swarmId: config.swarmId,
                routine: config.routine,
                inputs: config.inputs,
                config: config.config,
                userId: config.userId,
            });

            // Emit run started event
            await this.eventBus.publish("run.started", {
                runId: config.runId,
                routineVersionId: config.routineVersionId,
                swarmId: config.swarmId,
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

            const stateMachine = this.runMachines.get(runId);
            const currentStep = stateMachine?.getCurrentStep();

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
        const stateMachine = this.runMachines.get(runId);
        if (!stateMachine) {
            throw new Error(`Run ${runId} not found`);
        }

        await stateMachine.cancel(reason || "User cancelled");
        
        // Remove from active machines
        this.runMachines.delete(runId);
        
        // Emit cancellation event
        await this.eventBus.publish("run.cancelled", {
            runId,
            reason,
        });
    }

    /**
     * Shuts down the orchestrator
     */
    async shutdown(): Promise<void> {
        this.logger.info("[TierTwoOrchestrator] Shutting down");
        
        // Cancel all active runs
        for (const [runId, stateMachine] of this.runMachines) {
            try {
                await stateMachine.cancel("System shutdown");
            } catch (error) {
                this.logger.error("[TierTwoOrchestrator] Error cancelling run during shutdown", {
                    runId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
        
        this.runMachines.clear();
    }

    /**
     * Private helper methods
     */
    private setupEventHandlers(): void {
        // Handle step completion from Tier 3
        this.eventBus.on("step.completed", async (event) => {
            const { runId, stepId, outputs } = event.data;
            const stateMachine = this.runMachines.get(runId);
            if (stateMachine) {
                await stateMachine.handleStepCompletion(stepId, outputs);
            }
        });

        // Handle step failure from Tier 3
        this.eventBus.on("step.failed", async (event) => {
            const { runId, stepId, error } = event.data;
            const stateMachine = this.runMachines.get(runId);
            if (stateMachine) {
                await stateMachine.handleStepFailure(stepId, error);
            }
        });

        // Handle performance insights
        this.eventBus.on("performance.insight", async (event) => {
            const { runId } = event.data;
            const stateMachine = this.runMachines.get(runId);
            if (stateMachine) {
                await stateMachine.handlePerformanceInsight(event.data);
            }
        });
    }

    private calculateProgress(run: Run): number {
        if (!run.metrics) return 0;
        
        const { stepsCompleted = 0, totalSteps = 1 } = run.metrics;
        return Math.min((stepsCompleted / totalSteps) * 100, 100);
    }
}