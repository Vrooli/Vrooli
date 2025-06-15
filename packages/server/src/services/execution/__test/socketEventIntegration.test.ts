import { describe, beforeEach, afterEach, test, expect, vi } from "vitest";
import { ExecutionSocketEventEmitter } from "../shared/SocketEventEmitter.js";
import { SwarmSocketEmitter } from "../../swarmSocketEmitter.js";
import { type ISwarmStateStore } from "../tier1/state/swarmStateStore.js";
import { ExecutionStates, type Swarm } from "@vrooli/shared";

// Mock the SwarmSocketEmitter
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

describe("Socket Event Integration", () => {
    let socketEmitter: ExecutionSocketEventEmitter;
    let mockSwarmEmitter: any;
    let mockStateStore: ISwarmStateStore;

    beforeEach(() => {
        // Reset mocks
        vi.clearAllMocks();
        
        // Create mock state store
        mockStateStore = {
            getSwarm: vi.fn(),
            createSwarm: vi.fn(),
            updateSwarm: vi.fn(),
            deleteSwarm: vi.fn(),
            getSwarmState: vi.fn(),
            updateSwarmState: vi.fn(),
            getTeam: vi.fn(),
            updateTeam: vi.fn(),
            listActiveSwarms: vi.fn(),
            getSwarmsByState: vi.fn(),
            getSwarmsByUser: vi.fn(),
        };

        // Get the socket emitter instance
        socketEmitter = ExecutionSocketEventEmitter.get();
        socketEmitter.setStateStore(mockStateStore);

        // Get the mocked swarm emitter
        mockSwarmEmitter = SwarmSocketEmitter.get();
    });

    afterEach(() => {
        socketEmitter.clearCache();
    });

    describe("ChatId Resolution", () => {
        test("should resolve chatId from swarm metadata", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";
            
            const mockSwarm: Swarm = {
                id: swarmId,
                name: "Test Swarm",
                state: ExecutionStates.RUNNING,
                metadata: {
                    conversationId: chatId,
                    userId: "user1",
                },
                resources: {
                    allocated: { credits: 1000, tokens: 10000, time: 300000 },
                    consumed: { credits: 100, tokens: 1000, time: 30000 },
                    remaining: { credits: 900, tokens: 9000, time: 270000 },
                    reservedByChildren: { credits: 0, tokens: 0, time: 0 },
                    childReservations: [],
                },
                metrics: { tasksCompleted: 0, tasksFailed: 0, avgTaskDuration: 0 },
                createdAt: new Date(),
                updatedAt: new Date(),
            } as Swarm;

            mockStateStore.getSwarm = vi.fn().mockResolvedValue(mockSwarm);

            await socketEmitter.emitSwarmStateUpdate(
                swarmId,
                ExecutionStates.RUNNING,
                "Test message"
            );

            expect(mockStateStore.getSwarm).toHaveBeenCalledWith(swarmId);
            expect(mockSwarmEmitter.emitSwarmStateUpdate).toHaveBeenCalledWith(
                chatId,
                swarmId,
                ExecutionStates.RUNNING,
                "Test message"
            );
        });

        test("should use chatId override when provided", async () => {
            const swarmId = "test-swarm-1";
            const chatIdOverride = "override-chat-id";

            await socketEmitter.emitSwarmStateUpdate(
                swarmId,
                ExecutionStates.RUNNING,
                "Test message",
                chatIdOverride
            );

            // Should not call state store when override is provided
            expect(mockStateStore.getSwarm).not.toHaveBeenCalled();
            expect(mockSwarmEmitter.emitSwarmStateUpdate).toHaveBeenCalledWith(
                chatIdOverride,
                swarmId,
                ExecutionStates.RUNNING,
                "Test message"
            );
        });

        test("should cache chatId for subsequent calls", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";
            
            const mockSwarm: Swarm = {
                id: swarmId,
                metadata: { conversationId: chatId },
            } as Swarm;

            mockStateStore.getSwarm = vi.fn().mockResolvedValue(mockSwarm);

            // First call
            await socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.RUNNING);
            
            // Second call
            await socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.PAUSED);

            // State store should only be called once due to caching
            expect(mockStateStore.getSwarm).toHaveBeenCalledTimes(1);
            expect(mockSwarmEmitter.emitSwarmStateUpdate).toHaveBeenCalledTimes(2);
        });

        test("should handle missing swarm gracefully", async () => {
            const swarmId = "nonexistent-swarm";
            
            mockStateStore.getSwarm = vi.fn().mockResolvedValue(null);

            await socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.FAILED);

            expect(mockStateStore.getSwarm).toHaveBeenCalledWith(swarmId);
            // Should not emit when swarm is not found
            expect(mockSwarmEmitter.emitSwarmStateUpdate).not.toHaveBeenCalled();
        });

        test("should handle missing conversationId gracefully", async () => {
            const swarmId = "test-swarm-1";
            
            const mockSwarm: Swarm = {
                id: swarmId,
                metadata: {}, // No conversationId
            } as Swarm;

            mockStateStore.getSwarm = vi.fn().mockResolvedValue(mockSwarm);

            await socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.FAILED);

            expect(mockStateStore.getSwarm).toHaveBeenCalledWith(swarmId);
            // Should not emit when conversationId is missing
            expect(mockSwarmEmitter.emitSwarmStateUpdate).not.toHaveBeenCalled();
        });
    });

    describe("Event Emission", () => {
        test("should emit swarm resource updates", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";
            const resources = {
                allocated: 1000,
                consumed: 100,
                remaining: 900,
            };

            await socketEmitter.emitSwarmResourceUpdate(swarmId, resources, chatId);

            expect(mockSwarmEmitter.emitSwarmResourceUpdate).toHaveBeenCalledWith(
                chatId,
                swarmId,
                { resources }
            );
        });

        test("should emit swarm config updates", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";
            const configUpdate = {
                goal: "Updated goal",
                subtasks: [],
                stats: {
                    totalToolCalls: 5,
                    totalCredits: "100",
                    lastProcessingCycleEndedAt: Date.now(),
                },
            };

            await socketEmitter.emitSwarmConfigUpdate(swarmId, configUpdate, chatId);

            expect(mockSwarmEmitter.emitSwarmConfigUpdate).toHaveBeenCalledWith(
                chatId,
                configUpdate
            );
        });

        test("should emit swarm team updates", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";
            const teamUpdate = {
                teamId: "team-1",
                swarmLeader: "bot-leader",
                subtaskLeaders: {
                    "task-1": "bot-1",
                    "task-2": "bot-2",
                },
            };

            await socketEmitter.emitSwarmTeamUpdate(swarmId, teamUpdate, chatId);

            expect(mockSwarmEmitter.emitSwarmTeamUpdate).toHaveBeenCalledWith(
                chatId,
                swarmId,
                teamUpdate.teamId,
                teamUpdate.swarmLeader,
                teamUpdate.subtaskLeaders
            );
        });
    });

    describe("Error Handling", () => {
        test("should handle state store errors gracefully", async () => {
            const swarmId = "test-swarm-1";
            
            mockStateStore.getSwarm = vi.fn().mockRejectedValue(new Error("Database error"));

            // Should not throw
            await expect(
                socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.FAILED)
            ).resolves.toBeUndefined();

            // Should not emit on error
            expect(mockSwarmEmitter.emitSwarmStateUpdate).not.toHaveBeenCalled();
        });
    });

    describe("Cache Management", () => {
        test("should provide cache statistics", () => {
            const stats = socketEmitter.getCacheStats();
            
            expect(stats).toHaveProperty("size");
            expect(stats).toHaveProperty("hitRate");
            expect(typeof stats.size).toBe("number");
            expect(typeof stats.hitRate).toBe("number");
        });

        test("should clear cache when requested", async () => {
            const swarmId = "test-swarm-1";
            const chatId = "123456789";
            
            const mockSwarm: Swarm = {
                id: swarmId,
                metadata: { conversationId: chatId },
            } as Swarm;

            mockStateStore.getSwarm = vi.fn().mockResolvedValue(mockSwarm);

            // First call to populate cache
            await socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.RUNNING);
            
            // Clear cache
            socketEmitter.clearCache();
            
            // Second call should hit state store again
            await socketEmitter.emitSwarmStateUpdate(swarmId, ExecutionStates.PAUSED);

            expect(mockStateStore.getSwarm).toHaveBeenCalledTimes(2);
        });
    });
});