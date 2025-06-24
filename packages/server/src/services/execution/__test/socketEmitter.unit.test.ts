import { describe, beforeEach, afterEach, test, expect, vi } from "vitest";
import { ExecutionSocketEventEmitter } from "../shared/SocketEventEmitter.js";

// Mock the dependencies to avoid complex setup
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

vi.mock("../../../events/logger.js", () => ({
    logger: {
        debug: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

describe("ExecutionSocketEventEmitter Unit Tests", () => {
    let socketEmitter: ExecutionSocketEventEmitter;
    let mockStateStore: any;

    beforeEach(() => {
        vi.clearAllMocks();
        
        socketEmitter = ExecutionSocketEventEmitter.get();
        
        mockStateStore = {
            getSwarm: vi.fn(),
        };
        
        socketEmitter.setStateStore(mockStateStore);
    });

    afterEach(() => {
        socketEmitter.clearCache();
    });

    test("should be a singleton", () => {
        const instance1 = ExecutionSocketEventEmitter.get();
        const instance2 = ExecutionSocketEventEmitter.get();
        
        expect(instance1).toBe(instance2);
    });

    test("should set state store successfully", () => {
        const newMockStore = { getSwarm: vi.fn() };
        socketEmitter.setStateStore(newMockStore);
        
        // This test passes if no errors are thrown
        expect(true).toBe(true);
    });

    test("should handle missing state store gracefully", async () => {
        const emitterWithoutStore = new (ExecutionSocketEventEmitter as any)();
        
        // Should not throw when state store is not set
        await expect(
            emitterWithoutStore.getChatIdForSwarm("test-swarm"),
        ).resolves.toBeNull();
    });

    test("should provide cache statistics", () => {
        const stats = socketEmitter.getCacheStats();
        
        expect(stats).toHaveProperty("size");
        expect(stats).toHaveProperty("hitRate");
        expect(typeof stats.size).toBe("number");
        expect(typeof stats.hitRate).toBe("number");
    });

    test("should clear cache successfully", () => {
        socketEmitter.clearCache();
        
        const stats = socketEmitter.getCacheStats();
        expect(stats.size).toBe(0);
    });

    test("should handle chatId override correctly", async () => {
        const swarmId = "test-swarm";
        const chatIdOverride = "override-chat-id";
        
        await socketEmitter.emitSwarmStateUpdate(
            swarmId,
            "RUNNING" as any,
            "Test message",
            chatIdOverride,
        );
        
        // Should not call state store when override is provided
        expect(mockStateStore.getSwarm).not.toHaveBeenCalled();
    });
});
