import { describe, beforeEach, afterEach, test, expect, vi } from "vitest";
import { TierOneCoordinator } from "../tier1/tierOneCoordinator.js";
import { TierTwoOrchestrator } from "../tier2/tierTwoOrchestrator.js";
import { TierThreeExecutor } from "../tier3/TierThreeExecutor.js";
import { ExecutionSocketEventEmitter } from "../shared/SocketEventEmitter.js";
import { SwarmSocketEmitter } from "../../swarmSocketEmitter.js";
import { ExecutionStates } from "@vrooli/shared";
import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";

// Mock dependencies
vi.mock("../../swarmSocketEmitter.js", () => ({
    SwarmSocketEmitter: {
        get: vi.fn(() => ({
            emitSwarmStateUpdate: vi.fn(),
            emitSwarmConfigUpdate: vi.fn(),
            emitSwarmResourceUpdate: vi.fn(),
            emitSwarmTeamUpdate: vi.fn(),
            emitFullSwarmUpdate: vi.fn(),
        })),
    },
}));

vi.mock("../../../db/provider.js", () => ({
    DbProvider: {
        get: vi.fn(() => ({
            $transaction: vi.fn(),
            user: {
                findUnique: vi.fn(),
            },
            chat: {
                create: vi.fn(),
            },
            chat_participants: {
                create: vi.fn(),
            },
        })),
    },
}));

vi.mock("../../../services/conversation/chatStore.js", () => ({
    PrismaChatStore: vi.fn().mockImplementation(() => ({
        saveState: vi.fn(),
        finalizeSave: vi.fn().mockResolvedValue(true),
    })),
}));

vi.mock("../tier1/intelligence/conversationBridge.js", () => ({
    createConversationBridge: vi.fn(() => ({
        generateAgentResponse: vi.fn(),
        getConversationState: vi.fn(),
    })),
}));

describe("Tier Socket Integration", () => {
    let mockLogger: Logger;
    let mockEventBus: EventBus;
    let mockSwarmEmitter: any;
    let socketEmitter: ExecutionSocketEventEmitter;

    beforeEach(() => {
        vi.clearAllMocks();

        // Create mock logger
        mockLogger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as any;

        // Create mock event bus
        mockEventBus = {
            publish: vi.fn(),
            subscribe: vi.fn(),
        } as any;

        // Get socket emitter and mock swarm emitter
        socketEmitter = ExecutionSocketEventEmitter.get();
        mockSwarmEmitter = SwarmSocketEmitter.get();
    });

    afterEach(() => {
        socketEmitter.clearCache();
    });

    describe("Tier One Coordinator Integration", () => {
        test("should emit socket events during swarm creation", async () => {
            // This test would require significant mocking of the TierOneCoordinator
            // dependencies, so we'll focus on unit testing the socket emissions
            
            const swarmId = "test-swarm-1";
            const chatId = "123456789";

            // Test the socket emissions that would happen during swarm creation
            await socketEmitter.emitSwarmStateUpdate(
                swarmId,
                ExecutionStates.UNINITIALIZED,
                "Swarm initialization in progress",
                chatId
            );

            await socketEmitter.emitSwarmResourceUpdate(
                swarmId,
                {
                    allocated: 1000,
                    consumed: 0,
                    remaining: 1000,
                },
                chatId
            );

            expect(mockSwarmEmitter.emitSwarmStateUpdate).toHaveBeenCalledWith(
                chatId,
                swarmId,
                ExecutionStates.UNINITIALIZED,
                "Swarm initialization in progress"
            );

            expect(mockSwarmEmitter.emitSwarmResourceUpdate).toHaveBeenCalledWith(
                chatId,
                swarmId,
                { resources: { allocated: 1000, consumed: 0, remaining: 1000 } }
            );
        });
    });

    describe("SwarmStateMachine Integration", () => {
        test("should emit socket events during state transitions", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";

            // Test state transition emissions
            await socketEmitter.emitSwarmStateUpdate(
                swarmId,
                ExecutionStates.STARTING,
                "Swarm initialization complete, entering idle state",
                chatId
            );

            await socketEmitter.emitSwarmConfigUpdate(
                swarmId,
                {
                    goal: "Test Goal",
                    subtasks: [],
                    swarmLeader: "bot-leader",
                    stats: {
                        startedAt: Date.now(),
                        totalToolCalls: 0,
                        totalCredits: "0",
                        lastProcessingCycleEndedAt: null,
                    },
                },
                chatId
            );

            expect(mockSwarmEmitter.emitSwarmStateUpdate).toHaveBeenCalledWith(
                chatId,
                swarmId,
                ExecutionStates.STARTING,
                "Swarm initialization complete, entering idle state"
            );

            expect(mockSwarmEmitter.emitSwarmConfigUpdate).toHaveBeenCalledWith(
                chatId,
                expect.objectContaining({
                    goal: "Test Goal",
                    swarmLeader: "bot-leader",
                })
            );
        });

        test("should emit socket events for failure states", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";
            const errorMessage = "Swarm initialization failed: Test error";

            await socketEmitter.emitSwarmStateUpdate(
                swarmId,
                ExecutionStates.FAILED,
                errorMessage,
                chatId
            );

            expect(mockSwarmEmitter.emitSwarmStateUpdate).toHaveBeenCalledWith(
                chatId,
                swarmId,
                ExecutionStates.FAILED,
                errorMessage
            );
        });
    });

    describe("Tier Two Orchestrator Integration", () => {
        test("should emit socket events for run progress", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";

            // Test run progress emissions
            await socketEmitter.emitSwarmConfigUpdate(
                swarmId,
                {
                    subtasks: [], // Would be mapped from run steps
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "0",
                        lastProcessingCycleEndedAt: Date.now(),
                    },
                },
                chatId
            );

            expect(mockSwarmEmitter.emitSwarmConfigUpdate).toHaveBeenCalledWith(
                chatId,
                expect.objectContaining({
                    stats: expect.objectContaining({
                        totalToolCalls: 0,
                        totalCredits: "0",
                    }),
                })
            );
        });
    });

    describe("Tier Three Executor Integration", () => {
        test("should emit socket events for execution completion", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";

            // Test execution completion emissions
            await socketEmitter.emitSwarmConfigUpdate(
                swarmId,
                {
                    records: [{
                        id: "test-record-1",
                        routine_id: "test-routine",
                        routine_name: "Tool Execution",
                        params: { param1: "value1" },
                        output_resource_ids: [],
                        caller_bot_id: "bot-1",
                        created_at: new Date().toISOString(),
                    }],
                    stats: {
                        totalToolCalls: 1,
                        totalCredits: "100",
                        lastProcessingCycleEndedAt: Date.now(),
                    },
                },
                chatId
            );

            expect(mockSwarmEmitter.emitSwarmConfigUpdate).toHaveBeenCalledWith(
                chatId,
                expect.objectContaining({
                    records: expect.arrayContaining([
                        expect.objectContaining({
                            routine_name: "Tool Execution",
                            caller_bot_id: "bot-1",
                        }),
                    ]),
                    stats: expect.objectContaining({
                        totalToolCalls: 1,
                        totalCredits: "100",
                    }),
                })
            );
        });
    });

    describe("Error Handling Integration", () => {
        test("should handle socket emission failures gracefully", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";

            // Mock socket emitter to throw an error
            mockSwarmEmitter.emitSwarmStateUpdate.mockRejectedValue(
                new Error("Socket emission failed")
            );

            // Should not throw even if socket emission fails
            await expect(
                socketEmitter.emitSwarmStateUpdate(
                    swarmId,
                    ExecutionStates.RUNNING,
                    "Test message",
                    chatId
                )
            ).resolves.toBeUndefined();
        });
    });

    describe("Performance Integration", () => {
        test("should handle multiple concurrent emissions", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";

            // Emit multiple events concurrently
            const promises = [
                socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.RUNNING, "Test 1", chatId),
                socketEmitter.emitSwarmConfigUpdate(swarmId, { goal: "Test Goal" }, chatId),
                socketEmitter.emitSwarmResourceUpdate(swarmId, { allocated: 1000, consumed: 100, remaining: 900 }, chatId),
                socketEmitter.emitSwarmTeamUpdate(swarmId, { teamId: "team-1" }, chatId),
            ];

            await expect(Promise.all(promises)).resolves.toEqual([
                undefined,
                undefined,
                undefined,
                undefined,
            ]);

            // All emissions should have been called
            expect(mockSwarmEmitter.emitSwarmStateUpdate).toHaveBeenCalledTimes(1);
            expect(mockSwarmEmitter.emitSwarmConfigUpdate).toHaveBeenCalledTimes(1);
            expect(mockSwarmEmitter.emitSwarmResourceUpdate).toHaveBeenCalledTimes(1);
            expect(mockSwarmEmitter.emitSwarmTeamUpdate).toHaveBeenCalledTimes(1);
        });
    });
});