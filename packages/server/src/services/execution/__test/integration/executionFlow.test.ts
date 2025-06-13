import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { SwarmExecutionService } from "../../swarmExecutionService.js";
import { EventBus } from "../../cross-cutting/events/eventBus.js";
import { redis } from "../../../../redisConn.js";
import { SwarmStatus, RunStatus } from "@vrooli/shared";

describe("Execution Flow Integration Tests", () => {
    let service: SwarmExecutionService;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility
    let eventBus: EventBus;

    // Mock routine data
    const mockRoutine = {
        id: "routine-v1",
        name: "Integration Test Routine",
        root: {
            id: "root-1",
            name: "Test Root",
            ownerId: "owner-123",
        },
        nodes: [
            {
                id: "start",
                type: "start",
                data: {},
                translations: [{ language: "en", name: "Start", description: "Begin execution" }],
            },
            {
                id: "process",
                type: "action",
                data: {
                    action: "process_data",
                    inputs: ["inputData"],
                    outputs: ["processedData"],
                },
                translations: [{ language: "en", name: "Process", description: "Process input data" }],
            },
            {
                id: "decision",
                type: "decision",
                data: {
                    condition: "processedData.value > 50",
                },
                translations: [{ language: "en", name: "Check Value", description: "Check if value exceeds threshold" }],
            },
            {
                id: "success",
                type: "action",
                data: {
                    action: "report_success",
                    inputs: ["processedData"],
                    outputs: ["result"],
                },
                translations: [{ language: "en", name: "Success", description: "Report successful processing" }],
            },
            {
                id: "failure",
                type: "action",
                data: {
                    action: "report_failure",
                    inputs: ["processedData"],
                    outputs: ["error"],
                },
                translations: [{ language: "en", name: "Failure", description: "Report processing failure" }],
            },
            {
                id: "end",
                type: "end",
                data: {},
                translations: [{ language: "en", name: "End", description: "Complete execution" }],
            },
        ],
        edges: [
            { from: "start", to: "process" },
            { from: "process", to: "decision" },
            { from: "decision", to: "success", condition: "true" },
            { from: "decision", to: "failure", condition: "false" },
            { from: "success", to: "end" },
            { from: "failure", to: "end" },
        ],
    };

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: process.env.NODE_ENV === "test" ? "error" : "debug",
            transports: [new winston.transports.Console()],
        });

        // Set up test environment
        process.env.SWARM_STATE_STORE = "memory"; // Use in-memory store for tests

        // Mock Redis to prevent actual connections
        sandbox.stub(redis, "set").resolves("OK");
        sandbox.stub(redis, "get").resolves(null);
        sandbox.stub(redis, "del").resolves(1);
        sandbox.stub(redis, "expire").resolves(1);
        sandbox.stub(redis, "sadd").resolves(1);
        sandbox.stub(redis, "srem").resolves(1);
        sandbox.stub(redis, "smembers").resolves([]);

        // Create service instance
        service = new SwarmExecutionService(logger);
        eventBus = (service as any).eventBus;
    });

    afterEach(async () => {
        await service.shutdown();
        vi.restoreAllMocks();
    });

    describe("End-to-End Swarm and Run Execution", () => {
        it("should execute a complete swarm with a routine run", async () => {
            // Mock auth service
            const authService = (service as any).authService;
            sandbox.stub(authService, "checkPermission").resolves(true);

            // Mock routine loading
            const routineService = (service as any).routineService;
            sandbox.stub(routineService, "loadRoutine").resolves(mockRoutine);

            // Mock run persistence
            const persistenceService = (service as any).persistenceService;
            const createRunStub = sandbox.stub(persistenceService, "createRun").resolves({
                id: "run-123",
                routineVersionId: "routine-v1",
                userId: "user-123",
                name: "Test Run",
                status: RunStatus.InProgress,
                inputs: { inputData: { value: 75 } },
                completedComplexity: 0,
                contextSwitches: 0,
                timeElapsed: "0",
            });
            const updateRunStub = sandbox.stub(persistenceService, "updateRunStatus").resolves();
            const completeRunStub = sandbox.stub(persistenceService, "completeRun").resolves();
            const getRunStub = sandbox.stub(persistenceService, "getRun").resolves({
                id: "run-123",
                status: RunStatus.InProgress,
                outputs: {},
            });

            // Track events
            const events: Array<{ type: string; data: any }> = [];
            eventBus.on("*", (event) => {
                events.push({ type: event.type, data: event.data });
            });

            // Step 1: Start a swarm
            const swarmConfig = {
                swarmId: "swarm-test-123",
                name: "Integration Test Swarm",
                description: "Testing complete execution flow",
                goal: "Execute test routine successfully",
                resources: {
                    maxCredits: 10000,
                    maxTokens: 100000,
                    maxTime: 3600000,
                    tools: [],
                },
                config: {
                    model: "gpt-4o-mini",
                    temperature: 0.3,
                    autoApproveTools: true,
                    parallelExecutionLimit: 5,
                },
                userId: "user-123",
                organizationId: "org-123",
            };

            const swarmResult = await service.startSwarm(swarmConfig);
            expect(swarmResult.swarmId).toBe("swarm-test-123");

            // Verify swarm started event
            const swarmStartedEvent = events.find(e => e.type === "swarm.started");
            expect(swarmStartedEvent).toBeDefined();
            expect(swarmStartedEvent?.data.swarmId).toBe("swarm-test-123");

            // Step 2: Start a run within the swarm
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-test-123",
                routineVersionId: "routine-v1",
                inputs: { inputData: { value: 75 } },
                config: {
                    strategy: "reasoning",
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            const runResult = await service.startRun(runConfig);
            expect(runResult.runId).toBe("run-123");

            // Verify run creation
            expect(createRunStub).toHaveBeenCalled();
            expect(routineService.loadRoutine).toHaveBeenCalledWith("routine-v1");

            // Step 3: Check swarm status
            const swarmStatus = await service.getSwarmStatus("swarm-test-123");
            expect(swarmStatus.status).toBeDefined();
            expect([SwarmStatus.Pending, SwarmStatus.Running]).toContain(swarmStatus.status);

            // Step 4: Check run status
            const runStatus = await service.getRunStatus("run-123");
            expect(runStatus.status).toBe(RunStatus.InProgress);

            // Step 5: Simulate step execution events
            await eventBus.publish("step.completed", {
                runId: "run-123",
                stepId: "process",
                outputs: { processedData: { value: 75, processed: true } },
            });

            await eventBus.publish("step.completed", {
                runId: "run-123",
                stepId: "decision",
                outputs: { decision: true },
            });

            await eventBus.publish("step.completed", {
                runId: "run-123",
                stepId: "success",
                outputs: { result: "Successfully processed value: 75" },
            });

            // Step 6: Complete the run
            await eventBus.publish("run.completed", {
                runId: "run-123",
                outputs: { finalResult: "Successfully processed value: 75" },
            });

            // Verify run completion
            expect(completeRunStub).toHaveBeenCalledWith("run-123", {
                finalResult: "Successfully processed value: 75",
            });

            // Step 7: Cancel the swarm
            const cancelResult = await service.cancelSwarm("swarm-test-123", "user-123", "Test complete");
            expect(cancelResult.success).toBe(true);

            // Verify cancellation event
            const cancelEvent = events.find(e => e.type === "swarm.cancelled");
            expect(cancelEvent).toBeDefined();
            expect(cancelEvent?.data.swarmId).toBe("swarm-test-123");
        });

        it("should handle run failures gracefully", async () => {
            // Mock auth service
            const authService = (service as any).authService;
            sandbox.stub(authService, "checkPermission").resolves(true);

            // Mock routine loading to fail
            const routineService = (service as any).routineService;
            sandbox.stub(routineService, "loadRoutine").resolves(null);

            // Mock run persistence
            const persistenceService = (service as any).persistenceService;
            const updateRunStatusStub = sandbox.stub(persistenceService, "updateRunStatus").resolves();

            // Start a swarm
            const swarmConfig = {
                swarmId: "swarm-fail-123",
                name: "Failure Test Swarm",
                description: "Testing failure handling",
                goal: "Test error scenarios",
                resources: {
                    maxCredits: 1000,
                    maxTokens: 10000,
                    maxTime: 60000,
                    tools: [],
                },
                config: {
                    model: "gpt-4o-mini",
                    temperature: 0.3,
                    autoApproveTools: false,
                    parallelExecutionLimit: 1,
                },
                userId: "user-123",
            };

            await service.startSwarm(swarmConfig);

            // Try to start a run with non-existent routine
            const runConfig = {
                runId: "run-fail-123",
                swarmId: "swarm-fail-123",
                routineVersionId: "non-existent",
                inputs: {},
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 10,
                    timeout: 30000,
                },
                userId: "user-123",
            };

            try {
                await service.startRun(runConfig);
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("Routine version not found: non-existent");
            }

            // Verify run status was updated to failed
            expect(updateRunStatusStub).toHaveBeenCalledWith("run-fail-123", RunStatus.Failed);
        });

        it("should handle resource exhaustion", async () => {
            // Mock auth service
            const authService = (service as any).authService;
            sandbox.stub(authService, "checkPermission").resolves(true);

            // Start a swarm with limited resources
            const swarmConfig = {
                swarmId: "swarm-limited-123",
                name: "Limited Resource Swarm",
                description: "Testing resource limits",
                goal: "Test resource exhaustion",
                resources: {
                    maxCredits: 10, // Very limited credits
                    maxTokens: 100,
                    maxTime: 5000,
                    tools: [],
                },
                config: {
                    model: "gpt-4o-mini",
                    temperature: 0.3,
                    autoApproveTools: false,
                    parallelExecutionLimit: 1,
                },
                userId: "user-123",
            };

            await service.startSwarm(swarmConfig);

            // Emit resource exhaustion event
            await eventBus.publish("resources.low", {
                swarmId: "swarm-limited-123",
                resourceType: "credits",
                remaining: 0,
                threshold: 10,
                severity: "critical",
            });

            // Check swarm status
            const status = await service.getSwarmStatus("swarm-limited-123");
            expect(status.status).toBeDefined();
        });

        it("should handle concurrent runs in a swarm", async () => {
            // Mock services
            const authService = (service as any).authService;
            sandbox.stub(authService, "checkPermission").resolves(true);

            const routineService = (service as any).routineService;
            sandbox.stub(routineService, "loadRoutine").resolves(mockRoutine);

            const persistenceService = (service as any).persistenceService;
            let runCounter = 0;
            sandbox.stub(persistenceService, "createRun").callsFake(async (runData) => ({
                ...runData,
                id: runData.id,
                status: RunStatus.InProgress,
            }));
            sandbox.stub(persistenceService, "getRun").callsFake(async (runId) => ({
                id: runId,
                status: RunStatus.InProgress,
                outputs: {},
            }));

            // Start a swarm
            const swarmConfig = {
                swarmId: "swarm-concurrent-123",
                name: "Concurrent Execution Swarm",
                description: "Testing parallel run execution",
                goal: "Execute multiple runs concurrently",
                resources: {
                    maxCredits: 50000,
                    maxTokens: 500000,
                    maxTime: 3600000,
                    tools: [],
                },
                config: {
                    model: "gpt-4o-mini",
                    temperature: 0.3,
                    autoApproveTools: true,
                    parallelExecutionLimit: 10,
                },
                userId: "user-123",
            };

            await service.startSwarm(swarmConfig);

            // Start multiple runs concurrently
            const runPromises = [];
            for (let i = 0; i < 5; i++) {
                const runConfig = {
                    runId: `run-concurrent-${i}`,
                    swarmId: "swarm-concurrent-123",
                    routineVersionId: "routine-v1",
                    inputs: { inputData: { value: 50 + i * 10 } },
                    config: {
                        strategy: i % 2 === 0 ? "reasoning" : "deterministic",
                        model: "gpt-4o-mini",
                        maxSteps: 50,
                        timeout: 60000,
                    },
                    userId: "user-123",
                };
                runPromises.push(service.startRun(runConfig));
            }

            const results = await Promise.all(runPromises);

            // Verify all runs started successfully
            expect(results).toHaveLength(5);
            results.forEach((result, index) => {
                expect(result.runId).toBe(`run-concurrent-${index}`);
            });
        });
    });

    describe("Event-Driven Coordination", () => {
        it("should coordinate between tiers via events", async () => {
            const authService = (service as any).authService;
            sandbox.stub(authService, "checkPermission").resolves(true);

            // Track cross-tier events
            const crossTierEvents: string[] = [];
            
            eventBus.on("swarm.run_requested", () => crossTierEvents.push("tier1->tier2"));
            eventBus.on("step.execution_requested", () => crossTierEvents.push("tier2->tier3"));
            eventBus.on("step.completed", () => crossTierEvents.push("tier3->tier2"));
            eventBus.on("run.completed", () => crossTierEvents.push("tier2->tier1"));

            // Start a swarm
            await service.startSwarm({
                swarmId: "swarm-events-123",
                name: "Event Test Swarm",
                description: "Testing event coordination",
                goal: "Verify event flow",
                resources: {
                    maxCredits: 1000,
                    maxTokens: 10000,
                    maxTime: 60000,
                    tools: [],
                },
                config: {
                    model: "gpt-4o-mini",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 3,
                },
                userId: "user-123",
            });

            // Simulate event flow
            await eventBus.publish("swarm.run_requested", {
                swarmId: "swarm-events-123",
                runId: "run-events-123",
            });

            await eventBus.publish("step.execution_requested", {
                runId: "run-events-123",
                stepId: "step-1",
            });

            await eventBus.publish("step.completed", {
                runId: "run-events-123",
                stepId: "step-1",
                outputs: { result: "success" },
            });

            await eventBus.publish("run.completed", {
                runId: "run-events-123",
                outputs: { finalResult: "success" },
            });

            // Verify event flow
            expect(crossTierEvents).toContain("tier1->tier2");
            expect(crossTierEvents).toContain("tier2->tier3");
            expect(crossTierEvents).toContain("tier3->tier2");
            expect(crossTierEvents).toContain("tier2->tier1");
        });
    });
});