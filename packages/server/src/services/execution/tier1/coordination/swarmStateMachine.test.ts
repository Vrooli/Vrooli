import { describe, it, expect, vi, beforeEach } from "vitest";
import { SwarmStateMachine } from "./swarmStateMachine.js";
import { EventBus } from "../../cross-cutting/events/eventBus.js";
import { mockLogger } from "../../../../__test/logger.mock.js";
import { mockSwarmCoordination, mockTeamFormation } from "../../../../__test/fixtures/execution/swarmFixtures.js";
import { mockEvents } from "../../../../__test/fixtures/execution/eventFixtures.js";
import { type SwarmMetadata, type SwarmConfig } from "@vrooli/shared";

describe("SwarmStateMachine", () => {
    let swarmStateMachine: SwarmStateMachine;
    let eventBus: EventBus;
    let mockTierTwo: any;

    beforeEach(() => {
        vi.clearAllMocks();
        eventBus = new EventBus(mockLogger);
        mockTierTwo = {
            canExecuteRoutine: vi.fn().mockResolvedValue(true),
            startRun: vi.fn().mockResolvedValue({ runId: "test-run-123" }),
        };
    });

    describe("initialization", () => {
        it("should initialize with correct swarm ID and metadata", () => {
            const mockSwarm = mockSwarmCoordination.basicSwarm;
            const metadata: SwarmMetadata = {
                goal: mockSwarm.goal,
                resources: mockSwarm.resources,
                agents: mockSwarm.agents.map(a => ({ id: a.id, role: a.role })),
                startTime: new Date().toISOString(),
            };

            swarmStateMachine = new SwarmStateMachine(
                mockSwarm.id,
                metadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );

            expect(swarmStateMachine.getSwarmId()).toBe(mockSwarm.id);
        });

        it("should start in UNINITIALIZED state", () => {
            const mockSwarm = mockSwarmCoordination.basicSwarm;
            swarmStateMachine = new SwarmStateMachine(
                mockSwarm.id,
                {} as SwarmMetadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );

            // Note: Using BaseStates from actual implementation, not SwarmState from fixtures
            expect(swarmStateMachine.getCurrentState()).toBe("UNINITIALIZED");
        });
    });

    describe("state transitions", () => {
        beforeEach(() => {
            const mockSwarm = mockSwarmCoordination.basicSwarm;
            const metadata: SwarmMetadata = {
                goal: mockSwarm.goal,
                resources: mockSwarm.resources,
                agents: [],
                startTime: new Date().toISOString(),
            };
            swarmStateMachine = new SwarmStateMachine(
                mockSwarm.id,
                metadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );
        });

        it("should transition from UNINITIALIZED to RUNNING on start", async () => {
            const result = await swarmStateMachine.start();

            expect(result.swarmId).toBe(swarmStateMachine.getSwarmId());
            expect(swarmStateMachine.getCurrentState()).toBe("RUNNING");
        });

        it("should handle pause request", async () => {
            await swarmStateMachine.start();
            const paused = await swarmStateMachine.requestPause();

            expect(paused).toBe(true);
            expect(swarmStateMachine.getCurrentState()).toBe("PAUSED");
        });

        it("should handle stop request", async () => {
            await swarmStateMachine.start();
            const stopped = await swarmStateMachine.requestStop("Test stop");

            expect(stopped).toBe(true);
            expect(swarmStateMachine.getCurrentState()).toBe("STOPPED");
        });
    });

    describe("event handling", () => {
        beforeEach(() => {
            const mockSwarm = mockSwarmCoordination.basicSwarm;
            swarmStateMachine = new SwarmStateMachine(
                mockSwarm.id,
                {} as SwarmMetadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );
        });

        it("should process external message created event", async () => {
            await swarmStateMachine.start();
            
            // Using actual event type from implementation, not fixture
            const event = {
                type: "external_message_created",
                data: {
                    message: "Test message",
                    sender: "user-123",
                },
            };

            const result = await swarmStateMachine.processEvent(event);
            expect(result).toBeDefined();
            expect(result.message).toContain("external_message_created");
        });

        it("should emit events to event bus", async () => {
            const emitSpy = vi.spyOn(eventBus, "emit");
            
            await swarmStateMachine.start();

            // Check that swarm_started event was emitted
            expect(emitSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "swarm_started",
                    source: "tier1.swarm",
                    data: expect.objectContaining({
                        swarmId: swarmStateMachine.getSwarmId(),
                    }),
                }),
            );
        });
    });

    describe("tool execution", () => {
        beforeEach(() => {
            const mockSwarm = mockSwarmCoordination.basicSwarm;
            swarmStateMachine = new SwarmStateMachine(
                mockSwarm.id,
                {} as SwarmMetadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );
        });

        it("should handle tool approval response", async () => {
            await swarmStateMachine.start();

            const event = {
                type: "tool_approval_response",
                data: {
                    toolName: "test-tool",
                    approved: true,
                    reason: "Test approval",
                },
            };

            const result = await swarmStateMachine.processEvent(event);
            expect(result).toBeDefined();
            expect(result.message).toContain("tool_approval_response");
        });
    });

    describe("team formation", () => {
        it("should handle team formation from fixtures", async () => {
            const mockTeam = mockTeamFormation.basicTeam;
            const metadata: SwarmMetadata = {
                goal: mockTeam.goal,
                resources: mockTeam.resourceRequirements,
                agents: mockTeam.agents.map(a => ({ 
                    id: a.id, 
                    role: a.role,
                })),
                startTime: new Date().toISOString(),
            };

            swarmStateMachine = new SwarmStateMachine(
                mockTeam.id,
                metadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );

            await swarmStateMachine.start();
            expect(swarmStateMachine.getCurrentState()).toBe("RUNNING");
        });
    });

    describe("error handling", () => {
        beforeEach(() => {
            swarmStateMachine = new SwarmStateMachine(
                "error-test-swarm",
                {} as SwarmMetadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );
        });

        it("should transition to FAILED state on critical error", async () => {
            await swarmStateMachine.start();

            // Simulate a critical error
            const errorEvent = {
                type: "error",
                data: {
                    error: new Error("Critical failure"),
                    severity: "critical",
                },
            };

            await swarmStateMachine.processEvent(errorEvent);
            expect(swarmStateMachine.getCurrentState()).toBe("FAILED");
        });
    });

    describe("cross-tier communication", () => {
        it("should coordinate with Tier 2 for routine execution", async () => {
            const mockSwarm = mockSwarmCoordination.basicSwarm;
            swarmStateMachine = new SwarmStateMachine(
                mockSwarm.id,
                {} as SwarmMetadata,
                {} as SwarmConfig,
                mockLogger,
                eventBus,
                mockTierTwo,
            );

            await swarmStateMachine.start();

            // Request routine execution
            const routineRequest = {
                type: "execute_routine",
                data: {
                    routineId: "test-routine-123",
                    inputs: { test: "data" },
                },
            };

            await swarmStateMachine.processEvent(routineRequest);

            // Verify Tier 2 was called
            expect(mockTierTwo.canExecuteRoutine).toHaveBeenCalledWith("test-routine-123");
            expect(mockTierTwo.startRun).toHaveBeenCalledWith(
                expect.objectContaining({
                    routineVersionId: "test-routine-123",
                    inputs: { test: "data" },
                }),
            );
        });
    });
});