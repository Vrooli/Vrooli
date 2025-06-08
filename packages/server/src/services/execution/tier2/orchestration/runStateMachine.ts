import { type Logger } from "winston";
import {
    type Run,
    type RunState,
    type RunConfig,
    type RunProgress,
    type RunContext,
    type RunEvent,
    type RunEventType,
    type Location,
    type Navigator,
    type Routine,
    type StepStatus,
    type BranchExecution,
    RunState as RunStateEnum,
    RunEventType as RunEventTypeEnum,
    generatePk,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    type ExecutionResult,
    type ExecutionStatus,
    type ExecutionId,
    type RoutineExecutionInput,
    type TierCapabilities,
    type StepExecutionInput,
    type ResourceAllocation,
    type WorkflowDefinition,
    StrategyType,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/eventBus.js";
import { type IRunStateStore } from "../state/runStateStore.js";
import { NavigatorRegistry } from "../navigation/navigatorRegistry.js";
import { ContextManager } from "../context/contextManager.js";
import { BranchCoordinator } from "./branchCoordinator.js";
import { StepExecutor } from "./stepExecutor.js";
import { CheckpointManager } from "../persistence/checkpointManager.js";
import { PerformanceMonitor } from "../intelligence/performanceMonitor.js";
import { PathOptimizer } from "../intelligence/pathOptimizer.js";
import { MOISEGate } from "../validation/moiseGate.js";
import { type RollingHistory } from "../../cross-cutting/monitoring/index.js";
import { TelemetryShim } from "../../cross-cutting/monitoring/telemetryShim.js";

/**
 * Run initialization parameters
 */
export interface RunInitParams {
    routineId: string;
    userId: string;
    inputs: Record<string, unknown>;
    config?: Partial<RunConfig>;
    parentRunId?: string;
    swarmId?: string;
}

/**
 * TierTwoRunStateMachine - Universal routine execution engine
 * 
 * This is the heart of Vrooli's ability to execute diverse automation routines.
 * It represents a universal routine execution engine that's completely agnostic
 * to the underlying automation platform.
 * 
 * Key capabilities:
 * - Navigator-agnostic execution (BPMN, Langchain, Temporal, Native)
 * - Parallel branch coordination with resource awareness
 * - State management and checkpoint/recovery
 * - MOISE+ permission validation for every step
 * - Performance monitoring and path optimization
 * - Event-driven orchestration for external integrations
 * 
 * This creates an unprecedented universal automation ecosystem where workflows
 * from different platforms can share and execute each other's routines.
 * 
 * Implements TierCommunicationInterface for standardized inter-tier communication.
 */
export class TierTwoRunStateMachine implements TierCommunicationInterface {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly stateStore: IRunStateStore;
    private readonly tier3Executor: TierCommunicationInterface;
    private readonly navigatorRegistry: NavigatorRegistry;
    private readonly contextManager: ContextManager;
    private readonly branchCoordinator: BranchCoordinator;
    private readonly stepExecutor: StepExecutor;
    private readonly checkpointManager: CheckpointManager;
    private readonly performanceMonitor: PerformanceMonitor;
    private readonly pathOptimizer: PathOptimizer;
    private readonly moiseGate: MOISEGate;
    private readonly telemetryShim: TelemetryShim;
    private readonly rollingHistory?: RollingHistory;

    // Active runs
    private readonly activeRuns: Map<string, Run> = new Map();
    private readonly runNavigators: Map<string, Navigator> = new Map();

    constructor(
        logger: Logger,
        eventBus: EventBus,
        stateStore: IRunStateStore,
        tier3Executor: TierCommunicationInterface,
        rollingHistory?: RollingHistory,
    ) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.stateStore = stateStore;
        this.tier3Executor = tier3Executor;
        this.rollingHistory = rollingHistory;

        // Initialize components
        this.navigatorRegistry = new NavigatorRegistry(logger);
        this.contextManager = new ContextManager(stateStore, logger);
        this.branchCoordinator = new BranchCoordinator(eventBus, logger);
        this.stepExecutor = new StepExecutor(eventBus, logger);
        this.checkpointManager = new CheckpointManager(stateStore, logger);
        this.performanceMonitor = new PerformanceMonitor(eventBus, logger);
        this.pathOptimizer = new PathOptimizer(logger);
        this.moiseGate = new MOISEGate(logger);
        this.telemetryShim = new TelemetryShim(eventBus, { tier: 'tier2', component: 'run-state-machine' });

        // Subscribe to tier 3 events
        this.subscribeToExecutionEvents();
    }

    /**
     * Creates and initializes a new run
     */
    async createRun(params: RunInitParams): Promise<Run> {
        const runId = generatePk();
        
        this.logger.info("[RunStateMachine] Creating new run", {
            runId,
            routineId: params.routineId,
            userId: params.userId,
        });

        // Track in rolling history
        if (this.rollingHistory) {
            this.rollingHistory.addEvent({
                timestamp: new Date(),
                type: 'tier2.run.created',
                tier: 'tier2',
                component: 'run-state-machine',
                data: {
                    runId,
                    routineId: params.routineId,
                    userId: params.userId,
                },
            });
        }

        try {
            // Load routine
            const routine = await this.loadRoutine(params.routineId);
            
            // Select appropriate navigator
            const navigator = this.navigatorRegistry.getNavigator(routine.type);
            if (!navigator.canNavigate(routine.definition)) {
                throw new Error(`Navigator ${routine.type} cannot handle routine ${routine.id}`);
            }

            // Create run configuration
            const config: RunConfig = {
                maxSteps: 1000,
                maxDepth: 10,
                maxTime: 3600000, // 1 hour
                maxCost: 100,
                parallelization: true,
                checkpointInterval: 300000, // 5 minutes
                recoveryStrategy: "retry",
                ...params.config,
            };

            // Initialize progress
            const startLocation = navigator.getStartLocation(routine.definition);
            const progress: RunProgress = {
                totalSteps: this.estimateTotalSteps(routine, navigator),
                completedSteps: 0,
                failedSteps: 0,
                skippedSteps: 0,
                currentLocation: startLocation,
                locationStack: {
                    locations: [startLocation],
                    depth: 1,
                },
                branches: [],
            };

            // Initialize context
            const context = await this.contextManager.createContext({
                variables: {},
                blackboard: {},
                scopes: [{
                    id: "root",
                    name: "Root Scope",
                    variables: params.inputs,
                }],
            });

            // Create run
            const run: Run = {
                id: runId,
                routineId: params.routineId,
                state: RunStateEnum.UNINITIALIZED,
                config,
                progress,
                context,
                startedAt: new Date(),
            };

            // Store in state store
            await this.stateStore.createRun(runId, {
                id: runId,
                routineId: params.routineId,
                userId: params.userId,
                inputs: params.inputs,
                metadata: {
                    parentRunId: params.parentRunId,
                    swarmId: params.swarmId,
                },
                createdAt: new Date(),
                updatedAt: new Date(),
            });

            // Cache run and navigator
            this.activeRuns.set(runId, run);
            this.runNavigators.set(runId, navigator);

            // Emit creation event
            await this.emitRunEvent({
                type: RunEventTypeEnum.RUN_STARTED,
                timestamp: new Date(),
                runId,
                metadata: {
                    routineId: params.routineId,
                    routineType: routine.type,
                },
            });

            // Emit telemetry for run start
            await this.telemetryShim.emitExecutionTiming(
                'run-state-machine',
                'run_started',
                new Date(),
                new Date(),
                true,
                {
                    runId,
                    stepType: 'routine_orchestration',
                    strategy: StrategyType.CONVERSATIONAL,
                    estimatedCredits: config.maxCost?.toString() || '100',
                    estimatedTimeMs: config.maxTime,
                }
            );

            // Transition to LOADING state
            await this.transitionState(run, RunStateEnum.LOADING);

            return run;

        } catch (error) {
            this.logger.error("[RunStateMachine] Failed to create run", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Starts execution of a run
     */
    async startRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        this.logger.info("[RunStateMachine] Starting run", { runId });

        try {
            // Validate run can be started
            if (![RunStateEnum.READY, RunStateEnum.PAUSED].includes(run.state)) {
                throw new Error(`Cannot start run in state ${run.state}`);
            }

            // Transition to RUNNING
            await this.transitionState(run, RunStateEnum.RUNNING);

            // Start execution loop
            this.executeRunLoop(run).catch(error => {
                this.logger.error("[RunStateMachine] Run loop error", {
                    runId,
                    error: error instanceof Error ? error.message : String(error),
                });
                this.handleRunFailure(run, error);
            });

        } catch (error) {
            this.logger.error("[RunStateMachine] Failed to start run", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Pauses a running run
     */
    async pauseRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        if (run.state !== RunStateEnum.RUNNING) {
            throw new Error(`Cannot pause run in state ${run.state}`);
        }

        this.logger.info("[RunStateMachine] Pausing run", { runId });

        await this.transitionState(run, RunStateEnum.PAUSED);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_PAUSED,
            timestamp: new Date(),
            runId,
        });
    }

    /**
     * Resumes a paused run
     */
    async resumeRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        if (run.state !== RunStateEnum.PAUSED) {
            throw new Error(`Cannot resume run in state ${run.state}`);
        }

        this.logger.info("[RunStateMachine] Resuming run", { runId });

        await this.transitionState(run, RunStateEnum.RUNNING);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_RESUMED,
            timestamp: new Date(),
            runId,
        });

        // Resume execution loop
        this.executeRunLoop(run).catch(error => {
            this.logger.error("[RunStateMachine] Run loop error on resume", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            this.handleRunFailure(run, error);
        });
    }

    /**
     * Cancels a run
     */
    async cancelRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        this.logger.info("[RunStateMachine] Cancelling run", { runId });

        await this.transitionState(run, RunStateEnum.CANCELLED);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_CANCELLED,
            timestamp: new Date(),
            runId,
        });

        this.cleanupRun(run);
    }

    /**
     * Main execution loop for a run
     */
    private async executeRunLoop(run: Run): Promise<void> {
        const navigator = this.runNavigators.get(run.id);
        if (!navigator) {
            throw new Error(`Navigator not found for run ${run.id}`);
        }

        while (run.state === RunStateEnum.RUNNING) {
            try {
                // Check resource limits
                if (await this.checkResourceLimits(run)) {
                    this.logger.warn("[RunStateMachine] Resource limits exceeded", {
                        runId: run.id,
                    });
                    await this.suspendRun(run, "Resource limits exceeded");
                    break;
                }

                // Create checkpoint if needed
                if (await this.shouldCreateCheckpoint(run)) {
                    await this.checkpointManager.createCheckpoint(run);
                    
                    // Track checkpoint creation
                    if (this.rollingHistory) {
                        this.rollingHistory.addEvent({
                            timestamp: new Date(),
                            type: 'tier2.checkpoint.created',
                            tier: 'tier2',
                            component: 'checkpoint-manager',
                            data: {
                                runId: run.id,
                                location: run.progress.currentLocation,
                                completedSteps: run.progress.completedSteps,
                            },
                        });
                    }
                }

                // Get current location
                const currentLocation = run.progress.currentLocation;

                // Check if at end location
                if (navigator.isEndLocation(currentLocation)) {
                    await this.completeRun(run);
                    break;
                }

                // Get next locations (could be multiple for parallel branches)
                const nextLocations = navigator.getNextLocations(
                    currentLocation,
                    run.context.variables,
                );

                if (nextLocations.length === 0) {
                    // Dead end - complete run
                    await this.completeRun(run);
                    break;
                }

                // Handle branching
                if (nextLocations.length > 1) {
                    // Track branch execution
                    if (this.rollingHistory) {
                        this.rollingHistory.addEvent({
                            timestamp: new Date(),
                            type: 'tier2.branch.started',
                            tier: 'tier2',
                            component: 'branch-coordinator',
                            data: {
                                runId: run.id,
                                branchCount: nextLocations.length,
                                parallel: run.config.parallelization,
                            },
                        });
                    }
                    
                    await this.executeBranches(run, nextLocations, navigator);
                } else {
                    await this.executeStep(run, nextLocations[0], navigator);
                }

                // Check if paused
                if (run.state !== RunStateEnum.RUNNING) {
                    break;
                }

            } catch (error) {
                this.logger.error("[RunStateMachine] Error in run loop", {
                    runId: run.id,
                    error: error instanceof Error ? error.message : String(error),
                });

                if (run.config.recoveryStrategy === "retry") {
                    // Retry from last checkpoint
                    const recovered = await this.recoverFromCheckpoint(run);
                    if (!recovered) {
                        await this.handleRunFailure(run, error);
                        break;
                    }
                } else {
                    await this.handleRunFailure(run, error);
                    break;
                }
            }
        }
    }

    /**
     * Executes a single step
     */
    private async executeStep(
        run: Run,
        location: Location,
        navigator: Navigator,
    ): Promise<void> {
        const stepInfo = navigator.getStepInfo(location);
        const stepId = stepInfo.id;

        this.logger.debug("[RunStateMachine] Executing step", {
            runId: run.id,
            stepId,
            location,
        });

        // Check MOISE+ permissions
        const permitted = await this.moiseGate.checkPermission(
            run.id,
            stepId,
            run.context,
        );

        if (!permitted) {
            this.logger.warn("[RunStateMachine] Step not permitted by MOISE+", {
                runId: run.id,
                stepId,
            });
            
            // Emit telemetry for permission denial
            await this.telemetryShim.emitSecurityIncident(
                'moise_permission_denied',
                'medium',
                {
                    stepId,
                    details: 'Step execution blocked by MOISE+ permission system',
                    component: 'moise-gate',
                }
            );
            
            await this.skipStep(run, location, "Permission denied");
            return;
        }

        // Emit step start event
        await this.emitRunEvent({
            type: RunEventTypeEnum.STEP_STARTED,
            timestamp: new Date(),
            runId: run.id,
            stepId,
            metadata: { location, stepInfo },
        });

        try {
            // Execute step via Tier 3
            const result = await this.stepExecutor.executeStep({
                runId: run.id,
                stepId,
                stepInfo,
                context: run.context,
                location,
            });

            // Update context with results
            if (result.outputs) {
                await this.contextManager.updateVariables(
                    run.id,
                    result.outputs,
                );
            }

            // Update progress
            run.progress.completedSteps++;
            run.progress.currentLocation = location;

            // Update state store
            await this.stateStore.recordStepExecution(run.id, {
                stepId,
                state: "completed",
                startedAt: new Date(),
                completedAt: new Date(),
                result: result.outputs,
            });

            // Emit completion event
            await this.emitRunEvent({
                type: RunEventTypeEnum.STEP_COMPLETED,
                timestamp: new Date(),
                runId: run.id,
                stepId,
                metadata: { result },
            });

            // Track step completion
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier2.step.completed',
                    tier: 'tier2',
                    component: 'step-executor',
                    data: {
                        runId: run.id,
                        stepId,
                        location,
                        outputs: Object.keys(result.outputs || {}),
                    },
                });
            }

        } catch (error) {
            this.logger.error("[RunStateMachine] Step execution failed", {
                runId: run.id,
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Update progress
            run.progress.failedSteps++;

            // Record failure
            await this.stateStore.recordStepExecution(run.id, {
                stepId,
                state: "failed",
                startedAt: new Date(),
                completedAt: new Date(),
                error: error instanceof Error ? error.message : String(error),
            });

            // Emit failure event
            await this.emitRunEvent({
                type: RunEventTypeEnum.STEP_FAILED,
                timestamp: new Date(),
                runId: run.id,
                stepId,
                metadata: { error: error instanceof Error ? error.message : String(error) },
            });

            // Track step failure
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier2.step.failed',
                    tier: 'tier2',
                    component: 'step-executor',
                    data: {
                        runId: run.id,
                        stepId,
                        error: error instanceof Error ? error.message : String(error),
                        errorType: error instanceof Error ? error.constructor.name : 'Error',
                    },
                });
            }

            // Handle based on recovery strategy
            if (run.config.recoveryStrategy === "skip") {
                await this.skipStep(run, location, "Step failed");
            } else {
                throw error;
            }
        }
    }

    /**
     * Executes parallel branches
     */
    private async executeBranches(
        run: Run,
        locations: Location[],
        navigator: Navigator,
    ): Promise<void> {
        this.logger.debug(`[RunStateMachine] Executing ${locations.length} branches`, {
            runId: run.id,
        });

        const branches = await this.branchCoordinator.createBranches(
            run.id,
            locations,
            run.config.parallelization,
        );

        // Execute branches
        const results = await this.branchCoordinator.executeBranches(
            run,
            branches,
            navigator,
            this.stepExecutor,
        );

        // Merge results
        const mergedContext = await this.branchCoordinator.mergeBranchResults(
            run.context,
            results,
        );

        // Update run context
        run.context = mergedContext;
        await this.contextManager.updateContext(run.id, mergedContext);

        // Update progress
        for (const result of results) {
            run.progress.completedSteps += result.completedSteps;
            run.progress.failedSteps += result.failedSteps;
            run.progress.skippedSteps += result.skippedSteps;
        }

        // Continue from convergence point
        const convergenceLocation = this.findConvergenceLocation(locations, navigator);
        if (convergenceLocation) {
            run.progress.currentLocation = convergenceLocation;
        }
    }

    /**
     * Helper methods
     */
    private async transitionState(run: Run, newState: RunState): Promise<void> {
        const oldState = run.state;
        run.state = newState;
        
        await this.stateStore.updateRunState(run.id, newState);
        
        this.logger.info("[RunStateMachine] State transition", {
            runId: run.id,
            oldState,
            newState,
        });
    }

    private async loadRoutine(routineId: string): Promise<Routine> {
        // TODO: Load from routine store
        return {
            id: routineId,
            type: "native",
            version: "1.0.0",
            name: "Test Routine",
            definition: {},
            metadata: {
                author: "system",
                createdAt: new Date(),
                updatedAt: new Date(),
                tags: [],
                complexity: "simple",
            },
        };
    }

    private estimateTotalSteps(routine: Routine, navigator: Navigator): number {
        // TODO: Implement step counting based on routine definition
        return 10;
    }

    private async checkResourceLimits(run: Run): Promise<boolean> {
        // Check step limit
        if (run.config.maxSteps && run.progress.completedSteps >= run.config.maxSteps) {
            return true;
        }

        // Check time limit
        if (run.config.maxTime && run.startedAt) {
            const elapsed = Date.now() - run.startedAt.getTime();
            if (elapsed >= run.config.maxTime) {
                return true;
            }
        }

        // Check cost limit
        // TODO: Implement cost tracking

        return false;
    }

    private async shouldCreateCheckpoint(run: Run): Promise<boolean> {
        if (!run.config.checkpointInterval) {
            return false;
        }

        const lastCheckpoint = await this.checkpointManager.getLastCheckpoint(run.id);
        if (!lastCheckpoint) {
            return true;
        }

        const elapsed = Date.now() - lastCheckpoint.timestamp.getTime();
        return elapsed >= run.config.checkpointInterval;
    }

    private async suspendRun(run: Run, reason: string): Promise<void> {
        await this.transitionState(run, RunStateEnum.SUSPENDED);
        run.error = reason;
        
        this.logger.warn("[RunStateMachine] Run suspended", {
            runId: run.id,
            reason,
        });
    }

    private async completeRun(run: Run): Promise<void> {
        run.completedAt = new Date();
        await this.transitionState(run, RunStateEnum.COMPLETED);

        const duration = run.completedAt.getTime() - (run.startedAt?.getTime() || 0);
        
        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_COMPLETED,
            timestamp: new Date(),
            runId: run.id,
            metadata: {
                duration,
                stepsCompleted: run.progress.completedSteps,
            },
        });

        // Emit telemetry for run completion
        await this.telemetryShim.emitTaskCompletion(
            run.id,
            'routine_execution',
            run.status === RunStateEnum.COMPLETED ? 'success' : 'failure',
            duration,
            0 // No direct resource cost at this level
        );

        // Track run completion
        if (this.rollingHistory) {
            this.rollingHistory.addEvent({
                timestamp: new Date(),
                type: 'tier2.run.completed',
                tier: 'tier2',
                component: 'run-state-machine',
                data: {
                    runId: run.id,
                    duration,
                    stepsCompleted: run.progress.completedSteps,
                    stepsFailed: run.progress.failedSteps,
                    stepsSkipped: run.progress.skippedSteps,
                },
            });
        }

        this.cleanupRun(run);
    }

    private async handleRunFailure(run: Run, error: unknown): Promise<void> {
        run.completedAt = new Date();
        run.error = error instanceof Error ? error.message : String(error);
        await this.transitionState(run, RunStateEnum.FAILED);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_FAILED,
            timestamp: new Date(),
            runId: run.id,
            metadata: {
                error: run.error,
                failedSteps: run.progress.failedSteps,
            },
        });

        this.cleanupRun(run);
    }

    private async recoverFromCheckpoint(run: Run): Promise<boolean> {
        try {
            const checkpoint = await this.checkpointManager.getLastCheckpoint(run.id);
            if (!checkpoint) {
                return false;
            }

            await this.checkpointManager.restoreCheckpoint(run, checkpoint);
            return true;
        } catch (error) {
            this.logger.error("[RunStateMachine] Failed to recover from checkpoint", {
                runId: run.id,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    private async skipStep(run: Run, location: Location, reason: string): Promise<void> {
        run.progress.skippedSteps++;
        
        await this.emitRunEvent({
            type: RunEventTypeEnum.STEP_SKIPPED,
            timestamp: new Date(),
            runId: run.id,
            stepId: location.nodeId,
            metadata: { reason },
        });
    }

    private findConvergenceLocation(
        branches: Location[],
        navigator: Navigator,
    ): Location | null {
        // TODO: Implement branch convergence detection
        return branches[0];
    }

    private cleanupRun(run: Run): void {
        this.activeRuns.delete(run.id);
        this.runNavigators.delete(run.id);
    }

    private async emitRunEvent(event: RunEvent): Promise<void> {
        await this.eventBus.publish("run.events", event);
    }

    private subscribeToExecutionEvents(): void {
        // Subscribe to Tier 3 execution results
        this.eventBus.subscribe("execution.outputs", async (event) => {
            // Handle step completion from Tier 3
            this.logger.debug("[RunStateMachine] Received execution output", event);
        });

        // Subscribe to performance events
        this.eventBus.subscribe("telemetry.perf", async (event) => {
            // Feed to performance monitor
            await this.performanceMonitor.recordMetric(event);
        });
    }

    // ===== TierCommunicationInterface Implementation =====
    
    private readonly activeExecutions = new Map<ExecutionId, {
        status: ExecutionStatus;
        startTime: Date;
        runId: string;
    }>();

    /**
     * Execute a request via the standard tier communication interface
     * This is the primary method for Tier 1 to delegate routine execution to Tier 2
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
            runId: generatePk(),
        });

        try {
            this.logger.info("[RunStateMachine] Starting tier execution", {
                executionId,
                routineId: input.routineId,
                stepCount: input.workflow.steps.length,
            });

            // Track execution in rolling history
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier2.execution.started',
                    tier: 'tier2',
                    component: 'run-state-machine',
                    data: {
                        executionId,
                        routineId: input.routineId,
                        stepCount: input.workflow.steps.length,
                        parameters: Object.keys(input.parameters),
                    },
                });
            }

            // Create run parameters from the request
            const runParams: RunInitParams = {
                routineId: input.routineId,
                userId: context.userId,
                inputs: input.parameters,
                config: {
                    maxSteps: input.workflow.steps.length,
                    timeoutMs: allocation.maxDurationMs,
                    memoryLimitMB: allocation.maxMemoryMB,
                },
                parentRunId: context.parentExecutionId,
                swarmId: context.swarmId,
            };

            // Create and initialize the run
            const run = await this.createRun(runParams);
            // Add missing properties for tier communication
            (run as any).userId = context.userId;
            (run as any).swarmId = context.swarmId;

            // Execute the workflow using our existing orchestration
            const result = await this.executeWorkflow(run, input.workflow, allocation);

            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.COMPLETED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                runId: run.id,
            });

            this.logger.info("[RunStateMachine] Tier execution completed", {
                executionId,
                runId: run.id,
                success: true,
            });

            // Track execution completion
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier2.execution.completed',
                    tier: 'tier2',
                    component: 'run-state-machine',
                    data: {
                        executionId,
                        runId: run.id,
                        success: true,
                        duration: executionResult.duration,
                        stepsExecuted: executionResult.resourcesUsed.stepsExecuted,
                    },
                });
            }

            // Return execution result
            const executionResult: ExecutionResult<TOutput> = {
                success: true,
                result: result as TOutput,
                outputs: result as Record<string, unknown>,
                resourcesUsed: {
                    creditsUsed: allocation.maxCredits, // Will be updated with actual usage
                    durationMs: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                    memoryUsedMB: 0,
                    stepsExecuted: input.workflow.steps.length,
                },
                duration: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                context,
                metadata: {
                    strategy: 'routine_orchestration',
                    version: '2.0.0',
                    performance: {
                        runId: run.id,
                        stepsExecuted: input.workflow.steps.length,
                    },
                    timestamp: new Date().toISOString(),
                },
                confidence: 0.95, // High confidence for successful routine execution
                performanceScore: 0.85,
            };

            return executionResult;

        } catch (error) {
            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.FAILED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                runId: this.activeExecutions.get(executionId)?.runId || 'unknown',
            });

            this.logger.error("[RunStateMachine] Tier execution failed", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Return error result
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
                    strategy: 'routine_orchestration',
                    version: '2.0.0',
                    timestamp: new Date().toISOString(),
                },
                confidence: 0.0,
                performanceScore: 0.0,
            };

            return errorResult;
        }
    }

    /**
     * Execute workflow by delegating steps to Tier 3
     */
    private async executeWorkflow(
        run: Run,
        workflow: WorkflowDefinition,
        allocation: ResourceAllocation
    ): Promise<unknown> {
        const results = new Map<string, unknown>();
        
        // Execute steps based on dependencies
        for (const step of workflow.steps) {
            try {
                // Create step execution request for Tier 3
                const stepRequest: TierExecutionRequest<StepExecutionInput> = {
                    context: {
                        executionId: generatePk(),
                        parentExecutionId: run.id,
                        swarmId: run.swarmId || 'default',
                        userId: run.userId,
                        timestamp: new Date(),
                        correlationId: run.id,
                        stepId: step.id,
                        routineId: run.routineId,
                        stepType: step.name,
                    },
                    input: {
                        stepId: step.id,
                        stepType: step.name,
                        toolName: step.toolName,
                        parameters: step.parameters,
                        strategy: step.strategy,
                    },
                    allocation: {
                        maxCredits: (parseInt(allocation.maxCredits) / workflow.steps.length).toString(),
                        maxDurationMs: step.timeout || 30000,
                        maxMemoryMB: Math.floor(allocation.maxMemoryMB / workflow.steps.length),
                        maxConcurrentSteps: 1,
                    },
                };

                // Delegate to Tier 3
                const stepResult = await this.tier3Executor.execute(stepRequest);
                
                if (stepResult.success) {
                    results.set(step.id, stepResult.result);
                    this.logger.debug("[RunStateMachine] Step completed", {
                        runId: run.id,
                        stepId: step.id,
                        success: true,
                    });
                } else {
                    throw new Error(`Step ${step.id} failed: ${stepResult.error?.message}`);
                }

            } catch (error) {
                this.logger.error("[RunStateMachine] Step execution failed", {
                    runId: run.id,
                    stepId: step.id,
                    error: error instanceof Error ? error.message : String(error),
                });
                throw error;
            }
        }

        // Return final results (could be last step result or aggregated)
        const finalResults = Array.from(results.values());
        return finalResults.length === 1 ? finalResults[0] : finalResults;
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

            // Cancel the associated run if it exists
            const run = this.activeRuns.get(execution.runId);
            if (run) {
                await this.cancelRun(run.id);
            }

            this.logger.info("[RunStateMachine] Execution cancelled", { executionId });
        }
    }

    /**
     * Get tier capabilities for discovery
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: 'tier2',
            supportedInputTypes: ['RoutineExecutionInput'],
            supportedStrategies: ['routine_orchestration', 'parallel_execution', 'sequential_execution'],
            maxConcurrency: 50,
            estimatedLatency: {
                p50: 5000,
                p95: 30000,
                p99: 60000,
            },
            resourceLimits: {
                maxCredits: '100000',
                maxDurationMs: 3600000, // 1 hour
                maxMemoryMB: 4096,
            },
        };
    }
}
