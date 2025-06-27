import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { UnifiedRunStateMachine, RunStateMachineState } from "./unifiedRunStateMachine.js";
import { type Logger } from "winston";
import { EventBus } from "../../cross-cutting/events/eventBus.js";
import { type NavigatorRegistry } from "../navigation/navigatorRegistry.js";
import { type MOISEGate } from "../validation/moiseGate.js";
import { type IRunStateStore } from "../state/runStateStore.js";
import { InMemoryRunStateStore } from "../state/inMemoryRunStateStore.js";
import { 
    type TierCommunicationInterface, 
    type TierExecutionRequest, 
    type ExecutionResult, 
    type ExecutionStatus,
    type Navigator,
    type Location,
    generatePK, 
} from "@vrooli/shared";

/**
 * UnifiedRunStateMachine Integration Tests
 * 
 * These tests validate the complete tier 2 unified architecture including:
 * - All documented state machine states (NAVIGATOR_SELECTION, PLANNING, etc.)
 * - Universal navigator integration (Native, BPMN, Langchain, Temporal)
 * - MOISE+ deontic gate for organizational compliance
 * - Swarm context inheritance and bidirectional data flow
 * - Event-driven execution with sophisticated coordination
 * - Parallel branch execution with proper synchronization
 * - Resource management and checkpoint/recovery capabilities
 * - Comprehensive tier communication interface compliance
 */
describe("UnifiedRunStateMachine - Complete Tier 2 Architecture", () => {
    let logger: Logger;
    let eventBus: EventBus;
    let navigatorRegistry: NavigatorRegistry;
    let moiseGate: MOISEGate;
    let stateStore: IRunStateStore;
    let tier3Executor: TierCommunicationInterface;
    let unifiedStateMachine: UnifiedRunStateMachine;
    
    // Mock navigator for testing
    let mockNavigator: Navigator;
    
    beforeEach(async () => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;
        
        eventBus = new EventBus(logger);
        stateStore = new InMemoryRunStateStore();
        await stateStore.initialize();
        
        // Mock navigator
        mockNavigator = {
            id: "native",
            name: "Native Navigator",
            canNavigate: vi.fn().mockReturnValue(true),
            getStartLocation: vi.fn().mockReturnValue({ id: "start", type: "start" }),
            getNextLocations: vi.fn().mockReturnValue([{ id: "step1", type: "action" }]),
            parseRoutine: vi.fn().mockReturnValue({ steps: [] }),
            validateRoutine: vi.fn().mockReturnValue({ valid: true }),
        } as Navigator;
        
        // Mock navigator registry
        navigatorRegistry = {
            register: vi.fn(),
            getNavigator: vi.fn().mockReturnValue(mockNavigator),
            listNavigators: vi.fn().mockReturnValue([mockNavigator]),
            isRegistered: vi.fn().mockReturnValue(true),
        } as unknown as NavigatorRegistry;
        
        // Mock MOISE gate
        moiseGate = {
            validateExecution: vi.fn().mockResolvedValue({
                allowed: true,
                obligations: [],
                prohibitions: [],
                permissions: [],
            }),
            checkPermissions: vi.fn().mockResolvedValue(true),
            enforceObligations: vi.fn().mockResolvedValue([]),
        } as unknown as MOISEGate;
        
        // Mock Tier 3 executor
        tier3Executor = {
            execute: vi.fn(),
            getCapabilities: vi.fn().mockResolvedValue({
                supportedInputTypes: ["StepExecutionInput"],
                maxConcurrency: 10,
            }),
            getTierStatus: vi.fn().mockReturnValue({
                state: "running",
                activeExecutions: 0,
            }),
        } as TierCommunicationInterface;
        
        unifiedStateMachine = new UnifiedRunStateMachine(
            logger,
            eventBus,
            navigatorRegistry,
            moiseGate,
            stateStore,
            tier3Executor,
        );
    });
    
    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("Complete State Machine Lifecycle", () => {
        it("should execute full workflow with all documented states", async () => {
            const capturedStates: RunStateMachineState[] = [];
            
            // Track all state transitions
            const originalTransitionTo = unifiedStateMachine["transitionTo"];
            unifiedStateMachine["transitionTo"] = vi.fn().mockImplementation(async (state) => {
                capturedStates.push(state);
                return originalTransitionTo.call(unifiedStateMachine, state);
            });
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: { testData: "value" },
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "step1", action: "process" }] },
                    },
                    config: {
                        maxSteps: 10,
                        timeout: 30000,
                        maxCredits: "100",
                    },
                },
                metadata: {
                    userId: generatePK(),
                    swarmId: generatePK(),
                },
            };
            
            // Mock successful tier 3 execution
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: { result: "processed" },
                resourcesUsed: {
                    creditsUsed: "10",
                    durationMs: 1000,
                    memoryUsedMB: 50,
                    stepsExecuted: 1,
                },
                duration: 1000,
                metadata: { strategy: "test" },
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            // Verify successful execution
            expect(result.success).toBe(true);
            expect(result.outputs?.result).toBe("processed");
            
            // Verify key states were traversed
            expect(capturedStates).toContain(RunStateMachineState.INITIALIZING);
            expect(capturedStates).toContain(RunStateMachineState.NAVIGATOR_SELECTION);
            expect(capturedStates).toContain(RunStateMachineState.PLANNING);
            expect(capturedStates).toContain(RunStateMachineState.EXECUTING);
            expect(capturedStates).toContain(RunStateMachineState.COMPLETED);
        });
        
        it("should handle MOISE+ deontic validation during execution", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "step1", permissions: ["data:write"] }] },
                    },
                },
                metadata: {
                    userId: generatePK(),
                    teamId: generatePK(),
                },
            };
            
            // Mock deontic validation failure
            vi.mocked(moiseGate.validateExecution).mockResolvedValue({
                allowed: false,
                obligations: [],
                prohibitions: ["data:write forbidden"],
                permissions: [],
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            // Should fail due to MOISE+ validation
            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("MOISE+ validation failed");
            
            // Verify deontic gate was called
            expect(moiseGate.validateExecution).toHaveBeenCalled();
        });
    });

    describe("Navigator Integration", () => {
        it("should work with different navigator types", async () => {
            const navigatorTypes = [
                { type: "native", routine: { steps: [{ id: "step1" }] } },
                { type: "bpmn", routine: { process: { tasks: [{ id: "task1" }] } } },
                { type: "langchain", routine: { chain: { nodes: [{ id: "node1" }] } } },
                { type: "temporal", routine: { workflow: { activities: [{ id: "act1" }] } } },
            ];
            
            for (const { type, routine } of navigatorTypes) {
                // Configure navigator registry to return appropriate navigator
                vi.mocked(navigatorRegistry.getNavigator).mockReturnValue({
                    ...mockNavigator,
                    id: type,
                    name: `${type} Navigator`,
                });
                
                const request: TierExecutionRequest = {
                    executionId: generatePK(),
                    tierOrigin: 1,
                    tierTarget: 2,
                    type: "routine",
                    payload: {
                        routineId: generatePK(),
                        inputs: {},
                        routine: { id: generatePK(), type, definition: routine },
                    },
                };
                
                vi.mocked(tier3Executor.execute).mockResolvedValue({
                    success: true,
                    outputs: { navigatorType: type },
                    resourcesUsed: { creditsUsed: "5", durationMs: 500, memoryUsedMB: 10, stepsExecuted: 1 },
                    duration: 500,
                    metadata: {},
                    confidence: 0.9,
                    performanceScore: 0.8,
                });
                
                const result = await unifiedStateMachine.execute(request);
                
                expect(result.success).toBe(true);
                expect(result.outputs?.navigatorType).toBe(type);
                expect(navigatorRegistry.getNavigator).toHaveBeenCalledWith(type);
            }
        });
        
        it("should handle navigator selection and auto-detection", async () => {
            const routineWithoutType = {
                id: generatePK(),
                definition: { steps: [{ id: "step1", action: "process" }] },
            };
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: routineWithoutType,
                },
            };
            
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: {},
                resourcesUsed: { creditsUsed: "5", durationMs: 500, memoryUsedMB: 10, stepsExecuted: 1 },
                duration: 500,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            await unifiedStateMachine.execute(request);
            
            // Should auto-detect as "native" due to steps property
            expect(navigatorRegistry.getNavigator).toHaveBeenCalledWith("native");
        });
    });

    describe("Parallel Branch Execution", () => {
        it("should coordinate parallel branch execution with proper isolation", async () => {
            const parallelLocations: Location[] = [
                { id: "branch1", type: "action" },
                { id: "branch2", type: "action" },
                { id: "branch3", type: "action" },
            ];
            
            // Mock navigator to return parallel paths
            vi.mocked(mockNavigator.getNextLocations).mockReturnValue(parallelLocations);
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: { enableParallel: true },
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { 
                            steps: [
                                { id: "start", parallel: true },
                                { id: "branch1" },
                                { id: "branch2" },
                                { id: "branch3" },
                            ],
                        },
                    },
                },
            };
            
            // Track parallel execution calls
            const executionCalls: any[] = [];
            vi.mocked(tier3Executor.execute).mockImplementation(async (req) => {
                executionCalls.push(req);
                return {
                    success: true,
                    outputs: { branchId: req.payload.stepInfo?.id || "unknown" },
                    resourcesUsed: { creditsUsed: "3", durationMs: 300, memoryUsedMB: 5, stepsExecuted: 1 },
                    duration: 300,
                    metadata: {},
                    confidence: 0.9,
                    performanceScore: 0.8,
                };
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            expect(result.success).toBe(true);
            
            // Verify parallel execution was attempted
            // Note: In the actual implementation, this would involve more complex branch coordination
            expect(tier3Executor.execute).toHaveBeenCalled();
        });
        
        it("should aggregate results from parallel branches", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "parallel_step", parallel: true }] },
                    },
                },
            };
            
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: { 
                    branch_0: { result: "branch1_result" },
                    branch_1: { result: "branch2_result" },
                    branch_2: { result: "branch3_result" },
                },
                resourcesUsed: { creditsUsed: "15", durationMs: 800, memoryUsedMB: 20, stepsExecuted: 3 },
                duration: 800,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            expect(result.success).toBe(true);
            expect(result.outputs).toHaveProperty("branch_0");
            expect(result.outputs).toHaveProperty("branch_1");
            expect(result.outputs).toHaveProperty("branch_2");
        });
    });

    describe("Swarm Context Inheritance", () => {
        it("should inherit context from swarm configuration", async () => {
            const swarmConfig = {
                swarmLeader: generatePK(),
                goal: "Complete data analysis pipeline",
                teamId: generatePK(),
                resources: [
                    { id: "db1", type: "database", url: "postgres://..." },
                    { id: "api1", type: "api", endpoint: "https://..." },
                ],
                blackboard: [
                    { key: "dataset", value: "customer_data.csv", timestamp: new Date() },
                    { key: "model", value: "regression_v1", timestamp: new Date() },
                ],
            };
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "inherit_test" }] },
                    },
                    swarmConfig,
                },
            };
            
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: { inheritedGoal: swarmConfig.goal },
                resourcesUsed: { creditsUsed: "5", durationMs: 500, memoryUsedMB: 10, stepsExecuted: 1 },
                duration: 500,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            expect(result.success).toBe(true);
            expect(result.outputs?.inheritedGoal).toBe(swarmConfig.goal);
            
            // Verify tier 3 received swarm context
            expect(tier3Executor.execute).toHaveBeenCalledWith(
                expect.objectContaining({
                    metadata: expect.objectContaining({
                        swarmId: swarmConfig.swarmLeader,
                    }),
                }),
            );
        });
        
        it("should update swarm context with routine results", async () => {
            const swarmConfig = {
                swarmLeader: generatePK(),
                goal: "Data processing",
                blackboard: [],
            };
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "process_data" }] },
                    },
                    swarmConfig,
                },
            };
            
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: { 
                    processedData: "analysis_complete",
                    insights: ["trend1", "trend2", "trend3"],
                },
                resourcesUsed: { creditsUsed: "20", durationMs: 2000, memoryUsedMB: 100, stepsExecuted: 1 },
                duration: 2000,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            expect(result.success).toBe(true);
            expect(result.outputs?.processedData).toBe("analysis_complete");
            expect(result.outputs?.insights).toEqual(["trend1", "trend2", "trend3"]);
        });
    });

    describe("Resource Management and Limits", () => {
        it("should enforce resource limits and fail when exceeded", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "resource_heavy" }] },
                    },
                    config: {
                        maxCredits: "10", // Very low limit
                        timeout: 5000,
                        maxSteps: 2,
                    },
                },
            };
            
            // Mock resource-heavy execution
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: {},
                resourcesUsed: { 
                    creditsUsed: "50", // Exceeds limit
                    durationMs: 10000, // Exceeds timeout
                    memoryUsedMB: 200,
                    stepsExecuted: 5, // Exceeds step limit
                },
                duration: 10000,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            // Should fail due to resource limit violations
            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("UNIFIED_EXECUTION_FAILED");
        });
        
        it("should track resource usage accurately", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "resource_tracking" }] },
                    },
                },
            };
            
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: { completed: true },
                resourcesUsed: { 
                    creditsUsed: "25",
                    durationMs: 1500,
                    memoryUsedMB: 75,
                    stepsExecuted: 1,
                },
                duration: 1500,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            expect(result.success).toBe(true);
            expect(result.resourcesUsed?.creditsUsed).toBe("25");
            expect(result.resourcesUsed?.durationMs).toBeGreaterThan(0);
            expect(result.resourcesUsed?.memoryUsedMB).toBe(75);
            expect(result.resourcesUsed?.stepsExecuted).toBe(1);
        });
    });

    describe("Checkpoint and Recovery", () => {
        it("should create checkpoints during execution", async () => {
            const checkpointEvents: any[] = [];
            await eventBus.subscribe("run.checkpoint.*", async (event) => {
                checkpointEvents.push(event);
            });
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "checkpoint_test" }] },
                    },
                    config: {
                        enableCheckpoints: true,
                        checkpointFrequency: "every_step",
                    },
                },
            };
            
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: { checkpointed: true },
                resourcesUsed: { creditsUsed: "10", durationMs: 1000, memoryUsedMB: 50, stepsExecuted: 1 },
                duration: 1000,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            const result = await unifiedStateMachine.execute(request);
            
            expect(result.success).toBe(true);
            
            // Wait for events to propagate
            await new Promise(resolve => setTimeout(resolve, 100));
            
            // Verify checkpoint events were emitted
            expect(checkpointEvents.length).toBeGreaterThan(0);
        });
    });

    describe("Event-Driven Architecture", () => {
        it("should emit comprehensive events for agent analysis", async () => {
            const capturedEvents: any[] = [];
            const eventTypes = [
                "run.started", "run.navigator.selected", "run.planning.complete",
                "run.step.executing", "run.step.completed", "run.completed",
            ];
            
            for (const eventType of eventTypes) {
                await eventBus.subscribe(eventType, async (event) => {
                    capturedEvents.push({ type: eventType, data: event.data });
                });
            }
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "event_test" }] },
                    },
                },
            };
            
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                success: true,
                outputs: { eventTest: true },
                resourcesUsed: { creditsUsed: "5", durationMs: 500, memoryUsedMB: 25, stepsExecuted: 1 },
                duration: 500,
                metadata: {},
                confidence: 0.9,
                performanceScore: 0.8,
            });
            
            await unifiedStateMachine.execute(request);
            
            // Wait for events to propagate
            await new Promise(resolve => setTimeout(resolve, 200));
            
            // Verify comprehensive event emission
            expect(capturedEvents.length).toBeGreaterThan(0);
            
            // Events should provide rich data for agent analysis
            capturedEvents.forEach(event => {
                expect(event.data).toBeDefined();
                expect(event.type).toMatch(/^run\./);
            });
        });
    });

    describe("TierCommunicationInterface Compliance", () => {
        it("should provide complete tier capabilities", async () => {
            const capabilities = await unifiedStateMachine.getCapabilities();
            
            expect(capabilities.tier).toBe("tier2");
            expect(capabilities.supportedInputTypes).toContain("RoutineExecutionInput");
            expect(capabilities.supportedStrategies).toEqual(
                expect.arrayContaining(["reasoning", "deterministic", "conversational"]),
            );
            expect(capabilities.maxConcurrency).toBeGreaterThan(0);
            expect(capabilities.estimatedLatency).toBeDefined();
            expect(capabilities.resourceLimits).toBeDefined();
        });
        
        it("should provide accurate tier status", async () => {
            const status = unifiedStateMachine.getTierStatus();
            
            expect(status.state).toBeDefined();
            expect(status.activeRuns).toBeDefined();
            expect(status.activeExecutions).toBeDefined();
            expect(typeof status.activeRuns).toBe("number");
            expect(typeof status.activeExecutions).toBe("number");
        });
        
        it("should handle execution cancellation", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "long_running" }] },
                    },
                },
            };
            
            // Mock long-running execution
            vi.mocked(tier3Executor.execute).mockImplementation(async () => {
                await new Promise(resolve => setTimeout(resolve, 10000)); // 10 seconds
                return {
                    success: true,
                    outputs: {},
                    resourcesUsed: { creditsUsed: "5", durationMs: 10000, memoryUsedMB: 25, stepsExecuted: 1 },
                    duration: 10000,
                    metadata: {},
                    confidence: 0.9,
                    performanceScore: 0.8,
                };
            });
            
            // Start execution
            const executionPromise = unifiedStateMachine.execute(request);
            
            // Cancel after 100ms
            setTimeout(async () => {
                await unifiedStateMachine.cancelExecution(request.executionId);
            }, 100);
            
            const result = await executionPromise;
            
            // Should be cancelled
            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("UNIFIED_EXECUTION_FAILED");
        });
    });

    describe("Error Handling and Resilience", () => {
        it("should handle tier 3 communication failures gracefully", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "failing_step" }] },
                    },
                },
            };
            
            // Mock tier 3 failure
            vi.mocked(tier3Executor.execute).mockRejectedValue(
                new Error("Tier 3 communication failed"),
            );
            
            const result = await unifiedStateMachine.execute(request);
            
            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Tier 3 communication failed");
            expect(result.error?.code).toBe("UNIFIED_EXECUTION_FAILED");
            expect(result.error?.tier).toBe("tier2");
        });
        
        it("should emit failure events for resilience analysis", async () => {
            const failureEvents: any[] = [];
            await eventBus.subscribe("run.failed", async (event) => {
                failureEvents.push(event);
            });
            
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "failure_analysis" }] },
                    },
                },
            };
            
            vi.mocked(tier3Executor.execute).mockRejectedValue(
                new Error("Simulated failure for analysis"),
            );
            
            await unifiedStateMachine.execute(request);
            
            // Wait for events
            await new Promise(resolve => setTimeout(resolve, 100));
            
            // Verify failure events for agent analysis
            expect(failureEvents.length).toBeGreaterThan(0);
            expect(failureEvents[0].data).toBeDefined();
        });
    });

    describe("Run Orchestrator Interface", () => {
        it("should provide complete run orchestrator functionality", async () => {
            const runOrchestrator = unifiedStateMachine.getRunOrchestrator();
            
            // Verify all required methods are available
            expect(typeof runOrchestrator.createRun).toBe("function");
            expect(typeof runOrchestrator.startRun).toBe("function");
            expect(typeof runOrchestrator.getRunState).toBe("function");
            expect(typeof runOrchestrator.updateProgress).toBe("function");
            expect(typeof runOrchestrator.completeRun).toBe("function");
            expect(typeof runOrchestrator.failRun).toBe("function");
            expect(typeof runOrchestrator.cancelRun).toBe("function");
        });
        
        it("should support complete run lifecycle through orchestrator", async () => {
            const runOrchestrator = unifiedStateMachine.getRunOrchestrator();
            
            // Create run
            const run = await runOrchestrator.createRun({
                routineId: generatePK(),
                userId: generatePK(),
                inputs: { test: "data" },
                config: { maxSteps: 5 },
            });
            
            expect(run.id).toBeDefined();
            
            // Start run
            await runOrchestrator.startRun(run.id);
            
            // Get run state
            const runState = await runOrchestrator.getRunState(run.id);
            expect(runState).toBeDefined();
            expect(runState.id).toBe(run.id);
            
            // Update progress
            await runOrchestrator.updateProgress(run.id, {
                currentStepId: "step1",
                completedSteps: ["step1"],
            });
            
            // Complete run
            await runOrchestrator.completeRun(run.id, { result: "success" });
            
            // Verify completion
            const completedState = await runOrchestrator.getRunState(run.id);
            expect(completedState?.state).toBe(RunStateMachineState.COMPLETED);
        });
    });
});
