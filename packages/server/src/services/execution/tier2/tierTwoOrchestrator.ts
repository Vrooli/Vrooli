import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { RunStateMachine } from "./orchestration/runStateMachine.js";
import { NavigatorRegistry } from "./navigation/navigatorRegistry.js";
import { BranchCoordinator } from "./orchestration/branchCoordinator.js";
import { StepExecutor } from "./orchestration/stepExecutor.js";
import { ContextManager } from "./context/contextManager.js";
import { CheckpointManager } from "./persistence/checkpointManager.js";
import { PerformanceMonitorAdapter as PerformanceMonitor } from "../monitoring/adapters/PerformanceMonitorAdapter.js";
import { PathOptimizer } from "./intelligence/pathOptimizer.js";
import { MOISEGate } from "./validation/moiseGate.js";
import { type IRunStateStore, getRunStateStore } from "./state/runStateStore.js";
import {
    type ExecutionId,
    type ExecutionResult,
    type ExecutionStatus,
    type ResourceAllocation,
    type RoutineExecutionInput,
    type RunStatus,
    type Run,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    RunState,
    generatePk,
} from "@vrooli/shared";

/**
 * Tier Two Orchestrator
 * 
 * Main entry point for Tier 2 process intelligence.
 * Manages run lifecycle, routine navigation, and orchestration.
 */
export class TierTwoOrchestrator implements TierCommunicationInterface {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly tier3Executor: TierCommunicationInterface;
    private readonly runMachines: Map<string, RunStateMachine> = new Map();
    private readonly navigatorRegistry: NavigatorRegistry;
    private readonly branchCoordinator: BranchCoordinator;
    private readonly stepExecutor: StepExecutor;
    private readonly contextManager: ContextManager;
    private readonly checkpointManager: CheckpointManager;
    private readonly performanceMonitor: PerformanceMonitor;
    private readonly pathOptimizer: PathOptimizer;
    private readonly moiseGate: MOISEGate;
    private readonly stateStore: IRunStateStore;

    // Track active executions for interface compliance
    private readonly activeExecutions: Map<ExecutionId, { status: ExecutionStatus; startTime: Date; runId: string }> = new Map();

    constructor(logger: Logger, eventBus: EventBus, tier3Executor: TierCommunicationInterface) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.tier3Executor = tier3Executor;
        
        // Initialize components
        this.stateStore = getRunStateStore();
        this.initializeStateStore();
        this.navigatorRegistry = new NavigatorRegistry(logger);
        this.branchCoordinator = new BranchCoordinator(eventBus, logger, this.stateStore);
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

    /**
     * TierCommunicationInterface implementation
     */

    /**
     * Execute a routine execution request
     */
    async execute<TInput extends RoutineExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>> {
        const { context, input, allocation, options } = request;
        const executionId = context.executionId;

        // Track execution
        this.activeExecutions.set(executionId, {
            status: ExecutionStatus.RUNNING,
            startTime: new Date(),
            runId: input.routineId,
        });

        try {
            this.logger.info("[TierTwoOrchestrator] Starting tier execution", {
                executionId,
                routineId: input.routineId,
            });

            // Create a run ID for this execution
            const runId = generatePk();

            // Start routine execution through the existing startRun method
            // Note: This is a simplified implementation - in practice we'd need to load the routine data
            await this.startRun({
                runId,
                swarmId: context.swarmId,
                routineVersionId: input.routineId,
                routine: { workflow: input.workflow }, // Simplified routine data
                inputs: input.parameters,
                config: {
                    strategy: options?.strategy || 'reasoning',
                    model: 'gpt-4',
                    maxSteps: 50,
                    timeout: parseInt(allocation.maxDurationMs?.toString() || '300000'),
                },
                userId: context.userId || 'system',
            });

            // Wait for completion (simplified - in practice this would be more sophisticated)
            const result = await this.waitForCompletion(runId, parseInt(allocation.maxDurationMs?.toString() || '300000'));

            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.COMPLETED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                runId,
            });

            const executionResult: ExecutionResult<TOutput> = {
                success: true,
                result: result as TOutput,
                outputs: result as Record<string, unknown>,
                resourcesUsed: {
                    creditsUsed: '10', // Simplified
                    durationMs: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                    memoryUsedMB: 64,
                    stepsExecuted: 1,
                },
                duration: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                context,
                metadata: {
                    strategy: 'routine_execution',
                    version: '1.0.0',
                    timestamp: new Date().toISOString(),
                },
                confidence: 0.85,
                performanceScore: 0.80,
            };

            return executionResult;

        } catch (error) {
            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.FAILED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                runId: this.activeExecutions.get(executionId)?.runId || 'unknown',
            });

            this.logger.error("[TierTwoOrchestrator] Tier execution failed", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });

            const errorResult: ExecutionResult<TOutput> = {
                success: false,
                error: {
                    code: 'TIER2_EXECUTION_FAILED',
                    message: error instanceof Error ? error.message : 'Unknown error',
                    tier: 'tier2',
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
                    strategy: 'routine_execution',
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

            // Cancel the associated run
            await this.cancelRun(execution.runId, 'Execution cancelled');

            this.logger.info("[TierTwoOrchestrator] Execution cancelled", { executionId });
        }
    }

    /**
     * Get tier capabilities
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: 'tier2',
            supportedInputTypes: ['RoutineExecutionInput'],
            supportedStrategies: ['reasoning', 'deterministic', 'conversational'],
            maxConcurrency: 5,
            estimatedLatency: {
                p50: 15000,
                p95: 90000,
                p99: 300000,
            },
            resourceLimits: {
                maxCredits: '100000',
                maxDurationMs: 3600000, // 1 hour
                maxMemoryMB: 4096,
            },
        };
    }

    /**
     * Wait for run completion (simplified implementation)
     */
    private async waitForCompletion(runId: string, timeoutMs: number): Promise<Record<string, unknown>> {
        const startTime = Date.now();
        const checkInterval = 1000; // Check every second

        return new Promise((resolve, reject) => {
            const checkStatus = async () => {
                try {
                    const run = await this.stateStore.getRun(runId);
                    if (!run) {
                        reject(new Error(`Run ${runId} not found`));
                        return;
                    }

                    if (run.state === RunState.COMPLETED) {
                        resolve(run.outputs || {});
                        return;
                    }

                    if (run.state === RunState.FAILED) {
                        reject(new Error(`Run ${runId} failed: ${run.errors?.join(', ')}`));
                        return;
                    }

                    if (Date.now() - startTime > timeoutMs) {
                        reject(new Error(`Run ${runId} timed out after ${timeoutMs}ms`));
                        return;
                    }

                    // Continue checking
                    setTimeout(checkStatus, checkInterval);

                } catch (error) {
                    reject(error);
                }
            };

            checkStatus();
        });
    }
}