import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { RunStateMachine } from "../../tier2/orchestration/runStateMachine.js";
import { EventBus } from "../../cross-cutting/events/eventBus.js";
import { InMemoryRunStateStore } from "../../tier2/state/runStateStore.js";
import { NavigatorRegistry } from "../../tier2/navigation/navigatorRegistry.js";
import { BranchCoordinator } from "../../tier2/orchestration/branchCoordinator.js";
import { StepExecutor } from "../../tier2/orchestration/stepExecutor.js";
import { ContextManager } from "../../tier2/context/contextManager.js";
import { CheckpointManager } from "../../tier2/persistence/checkpointManager.js";
import { PerformanceMonitor } from "../../tier2/intelligence/performanceMonitor.js";
import { PathOptimizer } from "../../tier2/intelligence/pathOptimizer.js";
import { MOISEGate } from "../../tier2/validation/moiseGate.js";
import { RunState, type Run } from "@vrooli/shared";

describe("RunStateMachine", () => {
    let stateMachine: RunStateMachine;
    let logger: winston.Logger;
    let eventBus: EventBus;
    // Sandbox type removed for Vitest compatibility
    let stateStore: InMemoryRunStateStore;
    let navigatorRegistryStub: NavigatorRegistry;
    let branchCoordinatorStub: BranchCoordinator;
    let stepExecutorStub: StepExecutor;
    let contextManagerStub: ContextManager;
    let checkpointManagerStub: CheckpointManager;
    let performanceMonitorStub: PerformanceMonitor;
    let pathOptimizerStub: PathOptimizer;
    let moiseGateStub: MOISEGate;

    const mockRoutine = {
        id: "routine-v1",
        name: "Test Routine",
        description: "Test routine for unit tests",
        inputs: [
            { name: "input1", type: "string", required: true },
            { name: "input2", type: "number", required: false },
        ],
        outputs: [
            { name: "result", type: "object" },
        ],
        nodes: [
            {
                id: "node-1",
                type: "action",
                data: { action: "process" },
                next: ["node-2"],
            },
            {
                id: "node-2",
                type: "decision",
                data: { condition: "result > 0" },
                next: ["node-3", "node-4"],
            },
            {
                id: "node-3",
                type: "action",
                data: { action: "success" },
                next: [],
            },
            {
                id: "node-4",
                type: "action",
                data: { action: "failure" },
                next: [],
            },
        ],
    };

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        eventBus = new EventBus(logger);
        stateStore = new InMemoryRunStateStore(logger);

        // Create stubs
        navigatorRegistryStub = vi.mocked(new NavigatorRegistry() as any);
        branchCoordinatorStub = vi.mocked(new BranchCoordinator() as any);
        stepExecutorStub = vi.mocked(new StepExecutor() as any);
        contextManagerStub = vi.mocked(new ContextManager() as any);
        checkpointManagerStub = vi.mocked(new CheckpointManager() as any);
        performanceMonitorStub = vi.mocked(new PerformanceMonitor() as any);
        pathOptimizerStub = vi.mocked(new PathOptimizer() as any);
        moiseGateStub = vi.mocked(new MOISEGate() as any);

        stateMachine = new RunStateMachine(
            logger,
            eventBus,
            stateStore,
            navigatorRegistryStub,
            branchCoordinatorStub,
            stepExecutorStub,
            contextManagerStub,
            checkpointManagerStub,
            performanceMonitorStub,
            pathOptimizerStub,
            moiseGateStub,
        );
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("start", () => {
        it("should initialize run and transition to INITIALIZING", async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routine: mockRoutine,
                inputs: { input1: "test", input2: 42 },
                config: {
                    strategy: "reasoning",
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            moiseGateStub.validateOrganization.mockResolvedValue(true);
            contextManagerStub.initializeContext.mockResolvedValue({
                runId: "run-123",
                variables: { input1: "test", input2: 42 },
                outputs: {},
            });

            await stateMachine.start(runConfig);

            const run = await stateStore.getRun("run-123");
            expect(run).to.exist;
            expect(run?.state).toBe(RunState.INITIALIZING);
            expect(run?.routine).toEqual(mockRoutine);
        });

        it("should emit run started event", async () => {
            const eventSpy = vi.spyOn(eventBus, "publish");

            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routine: mockRoutine,
                inputs: {},
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            moiseGateStub.validateOrganization.mockResolvedValue(true);
            contextManagerStub.initializeContext.mockResolvedValue({
                runId: "run-123",
                variables: {},
                outputs: {},
            });

            await stateMachine.start(runConfig);

            expect(eventSpy).toHaveBeenCalledWith("run.state_transition", {
                runId: "run-123",
                from: RunState.UNINITIALIZED,
                to: RunState.INITIALIZING,
            });
        });

        it("should fail if MOISE validation fails", async () => {
            moiseGateStub.validateOrganization.mockResolvedValue(false);

            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routine: mockRoutine,
                inputs: {},
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            try {
                await stateMachine.start(runConfig);
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("MOISE organization validation failed");
            }
        });
    });

    describe("navigation", () => {
        beforeEach(async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routine: mockRoutine,
                inputs: { input1: "test" },
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            moiseGateStub.validateOrganization.mockResolvedValue(true);
            contextManagerStub.initializeContext.mockResolvedValue({
                runId: "run-123",
                variables: { input1: "test" },
                outputs: {},
            });

            await stateMachine.start(runConfig);
        });

        it("should navigate through routine nodes", async () => {
            navigatorRegistryStub.getNavigator.mockReturnValue({
                navigate: vi.fn().mockResolvedValue({
                    nextSteps: ["node-1"],
                    branches: [],
                    isComplete: false,
                }),
            } as any);

            pathOptimizerStub.optimizePath.mockResolvedValue({
                optimizedPath: ["node-1"],
                estimatedDuration: 1000,
                confidence: 0.9,
            });

            await stateMachine.transitionTo(RunState.NAVIGATING);

            expect(navigatorRegistryStub.getNavigator).toHaveBeenCalledWith("native");
            expect(pathOptimizerStub.optimizePath).toHaveBeenCalled();
        });

        it("should handle branch execution", async () => {
            navigatorRegistryStub.getNavigator.mockReturnValue({
                navigate: vi.fn().mockResolvedValue({
                    nextSteps: [],
                    branches: [
                        { id: "branch-1", steps: ["node-2", "node-3"] },
                        { id: "branch-2", steps: ["node-4"] },
                    ],
                    isComplete: false,
                }),
            } as any);

            branchCoordinatorStub.coordinateBranches.mockResolvedValue({
                completedBranches: ["branch-1", "branch-2"],
                results: {
                    "branch-1": { success: true, outputs: { value: 10 } },
                    "branch-2": { success: true, outputs: { value: 20 } },
                },
            });

            await stateMachine.transitionTo(RunState.BRANCH_EXECUTING);

            expect(branchCoordinatorStub.coordinateBranches).toHaveBeenCalled();
        });
    });

    describe("step execution", () => {
        beforeEach(async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routine: mockRoutine,
                inputs: { input1: "test" },
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            moiseGateStub.validateOrganization.mockResolvedValue(true);
            contextManagerStub.initializeContext.mockResolvedValue({
                runId: "run-123",
                variables: { input1: "test" },
                outputs: {},
            });

            await stateMachine.start(runConfig);
            await stateStore.updateRun("run-123", {
                state: RunState.EXECUTING,
                currentStep: "node-1",
            });
        });

        it("should execute steps successfully", async () => {
            stepExecutorStub.executeStep.mockResolvedValue({
                success: true,
                outputs: { result: "processed" },
                nextSteps: ["node-2"],
            });

            contextManagerStub.updateContext.mockResolvedValue();

            await stateMachine.handleStepCompletion("node-1", { result: "processed" });

            const run = await stateStore.getRun("run-123");
            expect(run?.currentStep).toBe("node-2");
            expect(contextManagerStub.updateContext).toHaveBeenCalledWith(
                "run-123",
                { result: "processed" }
            );
        });

        it("should handle step failure", async () => {
            await stateMachine.handleStepFailure("node-1", "Execution error");

            const run = await stateStore.getRun("run-123");
            expect(run?.state).toBe(RunState.ERROR_HANDLING);
            expect(run?.errors).to.include("Step node-1 failed: Execution error");
        });

        it("should create checkpoints periodically", async () => {
            checkpointManagerStub.shouldCreateCheckpoint.mockReturnValue(true);
            checkpointManagerStub.createCheckpoint.mockResolvedValue("checkpoint-1");

            stepExecutorStub.executeStep.mockResolvedValue({
                success: true,
                outputs: { result: "processed" },
                nextSteps: ["node-2"],
            });

            await stateMachine.handleStepCompletion("node-1", { result: "processed" });

            expect(checkpointManagerStub.createCheckpoint).toHaveBeenCalledWith(
                "run-123",
                expect.any(Object)
            );
        });
    });

    describe("completion", () => {
        beforeEach(async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routine: mockRoutine,
                inputs: { input1: "test" },
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            moiseGateStub.validateOrganization.mockResolvedValue(true);
            contextManagerStub.initializeContext.mockResolvedValue({
                runId: "run-123",
                variables: { input1: "test" },
                outputs: {},
            });

            await stateMachine.start(runConfig);
        });

        it("should complete run when navigation is done", async () => {
            navigatorRegistryStub.getNavigator.mockReturnValue({
                navigate: vi.fn().resolves({
                    nextSteps: [],
                    branches: [],
                    isComplete: true,
                }),
            } as any);

            performanceMonitorStub.generateReport.mockResolvedValue({
                totalDuration: 5000,
                stepsExecuted: 3,
                avgStepDuration: 1666,
                resourceUsage: { credits: 100, tokens: 1000 },
            });

            await stateMachine.transitionTo(RunState.FINALIZING);

            const run = await stateStore.getRun("run-123");
            expect(run?.state).toBe(RunState.COMPLETED);
            expect(performanceMonitorStub.generateReport).toHaveBeenCalled();
        });

        it("should emit completion event", async () => {
            const eventSpy = vi.spyOn(eventBus, "publish");

            await stateStore.updateRun("run-123", {
                state: RunState.FINALIZING,
                outputs: { finalResult: "success" },
            });

            await stateMachine.transitionTo(RunState.COMPLETED);

            expect(eventSpy).toHaveBeenCalledWith("run.completed", {
                runId: "run-123",
                outputs: { finalResult: "success" },
            });
        });
    });

    describe("cancel", () => {
        beforeEach(async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routine: mockRoutine,
                inputs: { input1: "test" },
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            moiseGateStub.validateOrganization.mockResolvedValue(true);
            contextManagerStub.initializeContext.mockResolvedValue({
                runId: "run-123",
                variables: { input1: "test" },
                outputs: {},
            });

            await stateMachine.start(runConfig);
        });

        it("should cancel run and clean up resources", async () => {
            checkpointManagerStub.cleanupCheckpoints.mockResolvedValue();

            await stateMachine.cancel("User requested cancellation");

            const run = await stateStore.getRun("run-123");
            expect(run?.state).toBe(RunState.CANCELLED);
            expect(checkpointManagerStub.cleanupCheckpoints).toHaveBeenCalledWith("run-123");
        });

        it("should emit cancellation event", async () => {
            const eventSpy = vi.spyOn(eventBus, "publish");

            await stateMachine.cancel("Timeout exceeded");

            expect(eventSpy).toHaveBeenCalledWith("run.cancelled", {
                runId: "run-123",
                reason: "Timeout exceeded",
            });
        });
    });

    describe("performance insights", () => {
        it("should handle performance optimization suggestions", async () => {
            const insight = {
                runId: "run-123",
                type: "optimization",
                suggestion: "Enable parallel execution for branches",
                impact: "30% reduction in execution time",
                confidence: 0.85,
            };

            // Create a run first
            await stateStore.createRun({
                id: "run-123",
                state: RunState.EXECUTING,
                routine: mockRoutine,
                config: { parallelBranches: false },
            } as any);

            await stateMachine.handlePerformanceInsight(insight);

            const run = await stateStore.getRun("run-123");
            expect(run?.config?.parallelBranches).toBe(true);
        });
    });
});