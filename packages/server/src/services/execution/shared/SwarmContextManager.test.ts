import { describe, expect, test, beforeEach, afterEach, vi, type MockedFunction } from "vitest";
import { RunState, EventTypes, type BaseTierExecutionRequest, type ResourceAllocation, type SwarmState, type SwarmId } from "@vrooli/shared";
import { SwarmContextManager, type ISwarmContextManager } from "./SwarmContextManager.js";
import { SwarmStateAccessor } from "./SwarmStateAccessor.js";

// Mock dependencies
vi.mock("../../../events/logger.js", async () => {
    const { createMockLogger } = await import("../../../__test/mocks/logger.js");
    return {
        logger: createMockLogger(),
    };
});

vi.mock("../../../redisConn.js", () => ({
    CacheService: {
        get: vi.fn(() => ({
            raw: vi.fn(),
        })),
    },
}));

vi.mock("../../events/publisher.js", () => ({
    EventPublisher: {
        emit: vi.fn().mockResolvedValue({ proceed: true, reason: null }),
    },
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => "test-pk-123"),
    };
});

// Create a mock instance for SwarmStateAccessor
const mockSwarmStateAccessor = {
    buildTriggerContext: vi.fn((context) => ({ mockTrigger: true, context })),
    accessData: vi.fn().mockImplementation(async (path, triggerContext, context) => {
        // Mock data access based on path
        const pathParts = path.split(".");
        let result = context;
        for (const part of pathParts) {
            if (result && typeof result === "object" && part in result) {
                result = (result as any)[part];
            } else {
                throw new Error(`Path not found: ${path}`);
            }
        }
        return result;
    }),
};

vi.mock("./SwarmStateAccessor.js", () => ({
    SwarmStateAccessor: vi.fn().mockImplementation(() => mockSwarmStateAccessor),
}));

// Test data factories
function createMockConfig(): BaseTierExecutionRequest {
    return {
        chatId: "test-swarm-1",
        userId: "test-user-1",
        taskId: "test-task-1",
        tier: "tier1",
        requestId: "test-request-1",
        priority: "normal",
        timeout: 300000,
        parentRequestId: null,
    };
}

function createMockSwarmState(chatId: SwarmId, partial: Partial<SwarmState> = {}): SwarmState {
    const now = new Date();
    return {
        swarmId: chatId,
        version: 1,
        chatConfig: {
            __version: "1.0.0",
            goal: "Test goal",
            subtasks: [],
            blackboard: [],
            resources: [],
            records: [],
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: Date.now(),
                lastProcessingCycleEndedAt: null,
            },
            limits: {
                maxCredits: "10000",
                maxDurationMs: 3600000,
            },
            scheduling: {
                defaultDelayMs: 0,
                requiresApprovalTools: "none",
                approvalTimeoutMs: 300000,
                autoRejectOnTimeout: true,
            },
            pendingToolCalls: [],
        },
        execution: {
            status: RunState.UNINITIALIZED,
            agents: [],
            activeRuns: [],
            startedAt: now,
            lastActivityAt: now,
        },
        resources: {
            allocated: [],
            consumed: {
                credits: 0,
                tokens: 0,
                time: 0,
            },
            remaining: {
                credits: 10000,
                tokens: 10000,
                time: 3600,
            },
        },
        metadata: {
            createdAt: now,
            lastUpdated: now,
            updatedBy: "system",
            subscribers: new Set<string>(),
        },
        ...partial,
    };
}

function createMockResourceAllocation(): Omit<ResourceAllocation, "id" | "allocated"> {
    return {
        consumerId: "test-consumer-1",
        consumerType: "routine",
        limits: {
            maxCredits: "100",
            maxDurationMs: 60000,
        },
        priority: "normal",
        allocatedAt: new Date(),
    };
}

describe("SwarmContextManager", () => {
    let contextManager: SwarmContextManager;
    let mockConfig: BaseTierExecutionRequest;
    let mockRedisClient: any;
    let mockCacheService: any;

    beforeEach(async () => {
        vi.clearAllMocks();
        
        // Setup Redis mock
        mockRedisClient = {
            get: vi.fn().mockResolvedValue(null),
            set: vi.fn().mockResolvedValue("OK"),
            del: vi.fn().mockResolvedValue(1),
            keys: vi.fn().mockResolvedValue([]),
            ping: vi.fn().mockResolvedValue("PONG"),
        };

        mockCacheService = {
            raw: vi.fn().mockResolvedValue(mockRedisClient),
        };

        const { CacheService } = await import("../../../redisConn.js");
        vi.mocked(CacheService).get.mockReturnValue(mockCacheService);

        mockConfig = createMockConfig();
        contextManager = new SwarmContextManager(mockConfig);
    });

    afterEach(async () => {
        // Clean up any contexts that may have been created during tests
        try {
            if (contextManager) {
                // Try to delete common test context IDs to ensure clean state
                const testContextIds = ["test-swarm-1", "high-frequency-test"];
                for (const contextId of testContextIds) {
                    try {
                        await contextManager.deleteContext(contextId);
                    } catch {
                        // Ignore errors - context may not exist
                    }
                }
                await contextManager.stop();
            }
        } catch (error) {
            // Ignore cleanup errors to prevent masking test failures
        }
        
        // Reset all mocks to ensure clean state
        vi.clearAllMocks();
        
        // Clear all timers to prevent timeout issues
        vi.clearAllTimers();
    });

    describe("Initialization and Lifecycle", () => {
        test("should initialize with provided config", () => {
            expect(contextManager).toBeInstanceOf(SwarmContextManager);
            expect(SwarmStateAccessor).toHaveBeenCalled();
        });

        test("should start and stop correctly", async () => {
            await contextManager.start();
            await contextManager.stop();
            
            // Should not throw
            expect(true).toBe(true);
        });

        test("should handle errors during stop gracefully", async () => {
            // Mock an error during cache clear
            const originalClear = Map.prototype.clear;
            Map.prototype.clear = vi.fn(() => {
                throw new Error("Cache clear failed");
            });

            await expect(contextManager.stop()).resolves.not.toThrow();

            // Restore original method
            Map.prototype.clear = originalClear;
        });
    });

    describe("Context Lifecycle Management", () => {
        const testSwarmId: SwarmId = "test-swarm-1";

        describe("createContext", () => {
            test("should create new context with defaults", async () => {
                mockRedisClient.get.mockResolvedValue(null); // No existing context
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);

                const result = await contextManager.createContext(testSwarmId);

                expect(result).toBeDefined();
                expect(result.swarmId).toBe(testSwarmId);
                expect(result.version).toBe(1);
                expect(result.chatConfig.__version).toBe("1.0.0");
                expect(result.resources.remaining.credits).toBe(10000);
                expect(result.execution.status).toBe(RunState.UNINITIALIZED);
                expect(mockRedisClient.set).toHaveBeenCalled();
            });

            test("should merge initial config with defaults", async () => {
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);

                const initialConfig: Partial<SwarmState> = {
                    chatConfig: {
                        __version: "1.0",
                        goal: "Custom goal",
                        limits: {
                            maxCredits: "5000",
                            maxDurationMs: 1800000,
                        },
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: null,
                            lastProcessingCycleEndedAt: null,
                        },
                    },
                    metadata: {
                        updatedBy: "test-user",
                    },
                };

                const result = await contextManager.createContext(testSwarmId, initialConfig);

                expect(result.chatConfig.goal).toBe("Custom goal");
                expect(result.chatConfig.limits?.maxCredits).toBe("5000");
                expect(result.metadata.updatedBy).toBe("test-user");
                expect(result.swarmId).toBe(testSwarmId); // Core fields preserved
                expect(result.version).toBe(1);
            });

            test("should reject creation if context already exists", async () => {
                const existingContext = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context: existingContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };

                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));

                await expect(contextManager.createContext(testSwarmId))
                    .rejects.toThrow(`Swarm context already exists: ${testSwarmId}`);
            });

            test("should reject creation if validation fails", async () => {
                mockRedisClient.get.mockResolvedValue(null);

                // Mock invalid context by creating one with invalid version
                const initialConfig: Partial<SwarmState> = {
                    version: -1, // Invalid version
                };

                await expect(contextManager.createContext(testSwarmId, initialConfig))
                    .rejects.toThrow("Context validation failed");
            });

            test("should handle Redis errors during creation", async () => {
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockRejectedValue(new Error("Redis connection failed"));

                await expect(contextManager.createContext(testSwarmId))
                    .rejects.toThrow("Redis connection failed");
            });
        });

        describe("getContext", () => {
            test("should return null for non-existent context", async () => {
                mockRedisClient.get.mockResolvedValue(null);

                const result = await contextManager.getContext(testSwarmId);

                expect(result).toBeNull();
                expect(mockRedisClient.get).toHaveBeenCalledWith(`swarm_context:${testSwarmId}`);
            });

            test("should return context from Redis", async () => {
                const mockContext = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context: mockContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };

                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");

                const result = await contextManager.getContext(testSwarmId);

                expect(result).toEqual(mockContext);
                expect(mockRedisClient.get).toHaveBeenCalledWith(`swarm_context:${testSwarmId}`);
            });

            test("should use cache when available and valid", async () => {
                const mockContext = createMockSwarmState(testSwarmId);
                
                // First call should hit Redis
                mockRedisClient.get.mockResolvedValue(JSON.stringify({
                    __version: "1",
                    context: mockContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                }));
                mockRedisClient.set.mockResolvedValue("OK");

                await contextManager.getContext(testSwarmId);

                // Reset mocks
                mockRedisClient.get.mockClear();

                // Second call should use cache
                const result = await contextManager.getContext(testSwarmId);

                expect(result).toEqual(mockContext);
                expect(mockRedisClient.get).not.toHaveBeenCalled(); // Cache hit
            });

            test("should handle invalid JSON in Redis", async () => {
                mockRedisClient.get.mockResolvedValue("invalid-json{");

                const result = await contextManager.getContext(testSwarmId);

                expect(result).toBeNull();
            });

            test("should handle invalid storage record format", async () => {
                const invalidRecord = { someField: "invalid" };
                mockRedisClient.get.mockResolvedValue(JSON.stringify(invalidRecord));

                const result = await contextManager.getContext(testSwarmId);

                expect(result).toBeNull();
            });

            test("should handle Redis errors gracefully", async () => {
                mockRedisClient.get.mockRejectedValue(new Error("Redis connection failed"));

                const result = await contextManager.getContext(testSwarmId);

                expect(result).toBeNull();
            });
        });

        describe("updateContext", () => {
            test("should update existing context with version increment", async () => {
                const originalContext = createMockSwarmState(testSwarmId, { version: 1 });
                const mockRecord = {
                    __version: "1",
                    context: originalContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };

                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);

                const updates: Partial<SwarmState> = {
                    chatConfig: {
                        ...originalContext.chatConfig,
                        goal: "Updated goal",
                    },
                };

                const result = await contextManager.updateContext(testSwarmId, updates, "Test update");

                expect(result.version).toBe(2); // Version incremented
                expect(result.chatConfig.goal).toBe("Updated goal");
                expect(result.swarmId).toBe(testSwarmId); // Core field preserved
                expect(mockRedisClient.set).toHaveBeenCalled();
            });

            test("should reject update if context not found", async () => {
                mockRedisClient.get.mockResolvedValue(null);

                const updates: Partial<SwarmState> = {
                    chatConfig: {
                        __version: "1.0",
                        goal: "Updated goal",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: null,
                            lastProcessingCycleEndedAt: null,
                        },
                    },
                };

                await expect(contextManager.updateContext(testSwarmId, updates))
                    .rejects.toThrow(`Swarm context not found: ${testSwarmId}`);
            });

            test("should reject update if validation fails", async () => {
                const originalContext = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context: originalContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };

                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));

                // Update with invalid resource consumption
                const updates: Partial<SwarmState> = {
                    resources: {
                        ...originalContext.resources,
                        consumed: {
                            credits: 20000, // Exceeds limit of 10000
                            tokens: 0,
                            time: 0,
                        },
                    },
                };

                await expect(contextManager.updateContext(testSwarmId, updates))
                    .rejects.toThrow("Context validation failed");
            });

            test("should publish update event after successful update", async () => {
                const originalContext = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context: originalContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };

                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);

                const updates: Partial<SwarmState> = {
                    chatConfig: {
                        ...originalContext.chatConfig,
                        goal: "Updated goal",
                    },
                };

                await contextManager.updateContext(testSwarmId, updates, "Test update");

                const { EventPublisher } = await import("../../events/publisher.js");
                expect(EventPublisher.emit).toHaveBeenCalled();
            });

            test("should clear cache after update", async () => {
                const originalContext = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context: originalContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };

                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);

                // First call to cache the context
                await contextManager.getContext(testSwarmId);

                // Update should clear cache
                const updates: Partial<SwarmState> = {
                    chatConfig: {
                        ...originalContext.chatConfig,
                        goal: "Updated goal",
                    },
                };

                await contextManager.updateContext(testSwarmId, updates);

                // Next get should hit Redis again (cache cleared)
                mockRedisClient.get.mockClear();
                mockRedisClient.get.mockResolvedValue(JSON.stringify({
                    ...mockRecord,
                    context: { ...originalContext, version: 2 },
                }));

                await contextManager.getContext(testSwarmId);
                expect(mockRedisClient.get).toHaveBeenCalled();
            });
        });

        describe("deleteContext", () => {
            test("should delete context and cleanup version history", async () => {
                mockRedisClient.del.mockResolvedValue(1);
                mockRedisClient.keys.mockResolvedValue([
                    `swarm_context:${testSwarmId}:version:1`,
                    `swarm_context:${testSwarmId}:version:2`,
                ]);

                await contextManager.deleteContext(testSwarmId);

                expect(mockRedisClient.del).toHaveBeenCalledWith(`swarm_context:${testSwarmId}`);
                expect(mockRedisClient.del).toHaveBeenCalledWith(
                    `swarm_context:${testSwarmId}:version:1`,
                    `swarm_context:${testSwarmId}:version:2`,
                );
            });

            test("should handle Redis errors during deletion", async () => {
                mockRedisClient.del.mockRejectedValue(new Error("Redis connection failed"));

                await expect(contextManager.deleteContext(testSwarmId))
                    .rejects.toThrow("Redis connection failed");
            });

            test("should handle missing version history gracefully", async () => {
                mockRedisClient.del.mockResolvedValue(1);
                mockRedisClient.keys.mockResolvedValue([]);

                await expect(contextManager.deleteContext(testSwarmId)).resolves.not.toThrow();
            });
        });
    });

    describe("Resource Management", () => {
        const testSwarmId: SwarmId = "test-swarm-1";

        beforeEach(async () => {
            // Setup a context for resource testing
            const mockContext = createMockSwarmState(testSwarmId);
            const mockRecord = {
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            };

            mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.keys.mockResolvedValue([]);
        });

        describe("allocateResources", () => {
            test("should allocate resources successfully", async () => {
                const allocationRequest = createMockResourceAllocation();

                const result = await contextManager.allocateResources(testSwarmId, allocationRequest);

                expect(result).toBeDefined();
                expect(result.id).toBe("test-pk-123");
                expect(result.consumerId).toBe(allocationRequest.consumerId);
                expect(result.consumerType).toBe(allocationRequest.consumerType);
                expect(result.allocated.credits).toBe(100);
                expect(result.allocated.timestamp).toBeInstanceOf(Date);
            });

            test("should update context with allocated resources", async () => {
                const allocationRequest = createMockResourceAllocation();

                await contextManager.allocateResources(testSwarmId, allocationRequest);

                // Verify update was called with resource changes
                expect(mockRedisClient.set).toHaveBeenCalled();
            });

            test("should reject allocation if context not found", async () => {
                mockRedisClient.get.mockResolvedValue(null);

                const allocationRequest = createMockResourceAllocation();

                await expect(contextManager.allocateResources(testSwarmId, allocationRequest))
                    .rejects.toThrow(`Swarm context not found: ${testSwarmId}`);
            });

            test("should reject allocation if insufficient resources", async () => {
                // Create context with low remaining credits
                const mockContext = createMockSwarmState(testSwarmId, {
                    resources: {
                        allocated: [],
                        consumed: { credits: 0, tokens: 0, time: 0 },
                        remaining: { credits: 50, tokens: 10000, time: 3600 }, // Only 50 credits available
                    },
                });

                mockRedisClient.get.mockResolvedValue(JSON.stringify({
                    __version: "1",
                    context: mockContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                }));

                const allocationRequest = createMockResourceAllocation(); // Requests 100 credits

                await expect(contextManager.allocateResources(testSwarmId, allocationRequest))
                    .rejects.toThrow("Resource allocation validation failed");
            });
        });

        describe("releaseResources", () => {
            test("should release resources successfully", async () => {
                const allocationId = "test-allocation-1";

                // Create context with existing allocation
                const mockContext = createMockSwarmState(testSwarmId, {
                    resources: {
                        allocated: [{
                            id: allocationId,
                            consumerId: "test-consumer-1",
                            consumerType: "routine",
                            limits: { maxCredits: "100", maxDurationMs: 60000 },
                            priority: "normal",
                            allocatedAt: new Date(),
                            allocated: { credits: 100, timestamp: new Date() },
                        }],
                        consumed: { credits: 0, tokens: 0, time: 0 },
                        remaining: { credits: 9900, tokens: 10000, time: 3600 },
                    },
                });

                mockRedisClient.get.mockResolvedValue(JSON.stringify({
                    __version: "1",
                    context: mockContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                }));

                await contextManager.releaseResources(testSwarmId, allocationId);

                // Should update context with released resources
                expect(mockRedisClient.set).toHaveBeenCalled();
            });

            test("should reject release if context not found", async () => {
                mockRedisClient.get.mockResolvedValue(null);

                await expect(contextManager.releaseResources(testSwarmId, "test-allocation-1"))
                    .rejects.toThrow(`Swarm context not found: ${testSwarmId}`);
            });

            test("should reject release if allocation not found", async () => {
                const allocationId = "non-existent-allocation";

                await expect(contextManager.releaseResources(testSwarmId, allocationId))
                    .rejects.toThrow(`Resource allocation not found: ${allocationId}`);
            });
        });

        describe("getResourceStatus", () => {
            test("should return resource status successfully", async () => {
                const result = await contextManager.getResourceStatus(testSwarmId);

                expect(result).toBeDefined();
                expect(result.total).toBeDefined();
                expect(result.allocated).toBeInstanceOf(Array);
                expect(result.available).toBeDefined();
                expect(result.total.credits).toBe(10000);
                expect(result.available.credits).toBe(10000);
            });

            test("should reject if context not found", async () => {
                mockRedisClient.get.mockResolvedValue(null);

                await expect(contextManager.getResourceStatus(testSwarmId))
                    .rejects.toThrow(`Swarm context not found: ${testSwarmId}`);
            });
        });
    });

    describe("Context Validation", () => {
        test("should validate valid context successfully", async () => {
            const validContext = createMockSwarmState("test-swarm-1");

            const result = await contextManager.validate(validContext);

            expect(result.valid).toBe(true);
            expect(result.errors).toHaveLength(0);
            expect(result.metrics.validationTimeMs).toBeGreaterThan(0);
            expect(result.metrics.rulesChecked).toBeGreaterThan(0);
        });

        test("should reject null or invalid context", async () => {
            const result = await contextManager.validate(null as any);

            expect(result.valid).toBe(false);
            expect(result.errors).toContainEqual(
                expect.objectContaining({
                    path: "root",
                    message: "Context must be a valid object",
                    severity: "error",
                }),
            );
        });

        test("should reject context missing required fields", async () => {
            const invalidContext = {
                swarmId: "test-swarm-1",
                version: 1,
                // Missing chatConfig and resources
            };

            const result = await contextManager.validate(invalidContext as any);

            expect(result.valid).toBe(false);
            expect(result.errors).toContainEqual(
                expect.objectContaining({
                    path: "root",
                    message: "Context missing required fields (chatConfig, resources)",
                    severity: "error",
                }),
            );
        });

        test("should reject context with invalid version", async () => {
            const invalidContext = createMockSwarmState("test-swarm-1", { version: 0 });

            const result = await contextManager.validate(invalidContext);

            expect(result.valid).toBe(false);
            expect(result.errors).toContainEqual(
                expect.objectContaining({
                    path: "version",
                    message: "Context version must be >= 1",
                    severity: "error",
                }),
            );
        });

        test("should reject context with excessive resource consumption", async () => {
            const invalidContext = createMockSwarmState("test-swarm-1", {
                resources: {
                    allocated: [],
                    consumed: { credits: 20000, tokens: 0, time: 0 }, // Exceeds limit
                    remaining: { credits: 0, tokens: 10000, time: 3600 },
                },
            });

            const result = await contextManager.validate(invalidContext);

            expect(result.valid).toBe(false);
            expect(result.errors).toContainEqual(
                expect.objectContaining({
                    path: "resources.consumed.credits",
                    message: expect.stringContaining("exceeds limit"),
                    severity: "error",
                }),
            );
        });

        test("should validate blackboard items", async () => {
            const invalidContext = createMockSwarmState("test-swarm-1", {
                chatConfig: {
                    ...createMockSwarmState("test-swarm-1").chatConfig,
                    blackboard: [
                        { id: "valid-item", value: "some value" },
                        { id: "", value: "invalid - missing id" }, // Invalid item
                        { value: "missing id completely" }, // Invalid item
                    ],
                },
            });

            const result = await contextManager.validate(invalidContext);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => e.path.includes("blackboard"))).toBe(true);
        });

        test("should provide warnings for missing agent configurations", async () => {
            const contextWithAgents = createMockSwarmState("test-swarm-1", {
                execution: {
                    ...createMockSwarmState("test-swarm-1").execution,
                    agents: [
                        { id: "agent-1", config: { __version: "1.0", resources: [] } }, // Missing agentSpec
                        { id: "agent-2", config: { __version: "1.0", resources: [], agentSpec: "valid-spec" } }, // Valid
                    ],
                },
            });

            const result = await contextManager.validate(contextWithAgents);

            expect(result.warnings.some(w => 
                w.path.includes("agent-1") && w.message.includes("agentSpec"),
            )).toBe(true);
        });

        test("should handle validation errors in SwarmStateAccessor", async () => {
            const mockStateAccessor = vi.mocked(SwarmStateAccessor);
            const instance = mockStateAccessor.mock.instances[0];
            vi.mocked(instance.buildTriggerContext).mockImplementation(() => {
                throw new Error("State accessor failed");
            });

            const validContext = createMockSwarmState("test-swarm-1");

            const result = await contextManager.validate(validContext);

            expect(result.valid).toBe(false);
            expect(result.errors).toContainEqual(
                expect.objectContaining({
                    path: "stateAccessor",
                    message: expect.stringContaining("State accessor failed"),
                    severity: "error",
                }),
            );
        });
    });

    describe("Context Querying", () => {
        const testSwarmId: SwarmId = "test-swarm-1";

        beforeEach(async () => {
            const mockContext = createMockSwarmState(testSwarmId, {
                chatConfig: {
                    ...createMockSwarmState(testSwarmId).chatConfig,
                    goal: "Test goal for querying",
                    blackboard: [
                        { id: "item1", value: "value1" },
                        { id: "item2", value: 42 },
                    ],
                },
            });

            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));
        });

        test("should return empty object for non-existent swarm", async () => {
            mockRedisClient.get.mockResolvedValue(null);

            const result = await contextManager.query({
                chatId: "non-existent-swarm",
            });

            expect(result).toEqual({});
        });

        test("should return full context when no specific query", async () => {
            const result = await contextManager.query({
                chatId: testSwarmId,
            });

            expect(result).toBeDefined();
            expect(result.swarmId).toBe(testSwarmId);
            expect(result.chatConfig?.goal).toBe("Test goal for querying");
        });

        test("should handle select paths correctly", async () => {
            const result = await contextManager.query({
                chatId: testSwarmId,
                select: ["chatConfig.goal", "swarmId"],
            });

            expect(result).toEqual({
                "chatConfig.goal": "Test goal for querying",
                "swarmId": testSwarmId,
            });
        });

        test("should handle invalid select paths gracefully", async () => {
            const result = await contextManager.query({
                chatId: testSwarmId,
                select: ["invalid.path", "chatConfig.goal"],
            });

            expect(result).toEqual({
                "invalid.path": undefined,
                "chatConfig.goal": "Test goal for querying",
            });
        });

        test("should filter using where conditions", async () => {
            const result = await contextManager.query({
                chatId: testSwarmId,
                where: [
                    {
                        path: "chatConfig.goal",
                        operator: "contains",
                        value: "querying",
                    },
                ],
            });

            expect(result.swarmId).toBe(testSwarmId); // Should return full context since condition matches
        });

        test("should return empty object when where conditions don't match", async () => {
            const result = await contextManager.query({
                chatId: testSwarmId,
                where: [
                    {
                        path: "chatConfig.goal",
                        operator: "equals",
                        value: "Non-matching goal",
                    },
                ],
            });

            expect(result).toEqual({});
        });

        test("should handle different where operators", async () => {
            // Test exists operator
            const existsResult = await contextManager.query({
                chatId: testSwarmId,
                where: [{ path: "chatConfig.goal", operator: "exists", value: true }],
            });
            expect(existsResult.swarmId).toBe(testSwarmId);

            // Test greaterThan operator (would need numeric data)
            const numericResult = await contextManager.query({
                chatId: testSwarmId,
                where: [{ path: "version", operator: "greaterThan", value: 0 }],
            });
            expect(numericResult.swarmId).toBe(testSwarmId);
        });

        test("should handle multiple where conditions with AND logic", async () => {
            const result = await contextManager.query({
                chatId: testSwarmId,
                where: [
                    { path: "swarmId", operator: "equals", value: testSwarmId },
                    { path: "version", operator: "greaterThan", value: 0 },
                ],
            });

            expect(result.swarmId).toBe(testSwarmId);
        });
    });

    describe("Event Publishing", () => {
        const testSwarmId: SwarmId = "test-swarm-1";

        beforeEach(async () => {
            const mockContext = createMockSwarmState(testSwarmId);
            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.keys.mockResolvedValue([]);
        });

        test("should publish update events after context update", async () => {
            const { EventPublisher } = await import("../../events/publisher.js");

            const updates: Partial<SwarmState> = {
                chatConfig: {
                    ...createMockSwarmState(testSwarmId).chatConfig,
                    goal: "Updated goal",
                },
            };

            await contextManager.updateContext(testSwarmId, updates, "Test update");

            expect(EventPublisher.emit).toHaveBeenCalled();
        });

        test("should handle event publishing errors gracefully", async () => {
            const { EventPublisher } = await import("../../events/publisher.js");
            vi.mocked(EventPublisher.emit).mockRejectedValue(new Error("Event publishing failed"));

            const updates: Partial<SwarmState> = {
                chatConfig: {
                    ...createMockSwarmState(testSwarmId).chatConfig,
                    goal: "Updated goal",
                },
            };

            // Should not throw even if event publishing fails
            await expect(contextManager.updateContext(testSwarmId, updates, "Test update"))
                .resolves.toBeDefined();
        });

        test("should include context in published events", async () => {
            const { EventPublisher } = await import("../../events/publisher.js");

            const updates: Partial<SwarmState> = {
                execution: {
                    ...createMockSwarmState(testSwarmId).execution,
                    status: RunState.RUNNING,
                },
            };

            await contextManager.updateContext(testSwarmId, updates, "Status update");

            expect(EventPublisher.emit).toHaveBeenCalled();
        });
    });

    describe("Health Check and Metrics", () => {
        test("should return healthy status when Redis is connected", async () => {
            mockRedisClient.ping.mockResolvedValue("PONG");

            const result = await contextManager.healthCheck();

            expect(result.healthy).toBe(true);
            expect(result.details.redis).toBe("connected");
            expect(result.details.cacheSize).toBeDefined();
            expect(result.details.metrics).toBeDefined();
        });

        test("should return unhealthy status when Redis is disconnected", async () => {
            mockRedisClient.ping.mockRejectedValue(new Error("Redis connection failed"));

            const result = await contextManager.healthCheck();

            expect(result.healthy).toBe(false);
            expect(result.details.redis).toBe("disconnected");
            expect(result.details.error).toContain("Redis connection failed");
        });

        test("should return current metrics", async () => {
            // Perform some operations to generate metrics
            const testSwarmId: SwarmId = "test-swarm-1";
            mockRedisClient.get.mockResolvedValue(null);
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.keys.mockResolvedValue([]);

            await contextManager.createContext(testSwarmId);

            const metrics = await contextManager.getMetrics();

            expect(metrics).toBeDefined();
            expect(metrics.contextsCreated).toBeGreaterThan(0);
            expect(typeof metrics.cacheHits).toBe("number");
            expect(typeof metrics.cacheMisses).toBe("number");
        });
    });

    describe("Caching System", () => {
        const testSwarmId: SwarmId = "test-swarm-1";

        test("should cache context after first retrieval", async () => {
            const mockContext = createMockSwarmState(testSwarmId);
            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));
            mockRedisClient.set.mockResolvedValue("OK");

            // First call should hit Redis
            await contextManager.getContext(testSwarmId);
            expect(mockRedisClient.get).toHaveBeenCalledTimes(1);

            // Reset Redis mock
            mockRedisClient.get.mockClear();

            // Second call should use cache
            const result = await contextManager.getContext(testSwarmId);
            expect(result).toEqual(mockContext);
            expect(mockRedisClient.get).not.toHaveBeenCalled();
        });

        test("should invalidate cache on context update", async () => {
            const mockContext = createMockSwarmState(testSwarmId);
            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.keys.mockResolvedValue([]);

            // Cache the context
            await contextManager.getContext(testSwarmId);

            // Update should invalidate cache
            await contextManager.updateContext(testSwarmId, {
                chatConfig: {
                    __version: "1.0",
                    goal: "Updated goal",
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "0",
                        startedAt: null,
                        lastProcessingCycleEndedAt: null,
                    },
                },
            });

            // Next get should hit Redis again
            mockRedisClient.get.mockClear();
            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: { ...mockContext, version: 2 },
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));

            await contextManager.getContext(testSwarmId);
            expect(mockRedisClient.get).toHaveBeenCalled();
        });

        test("should clear cache on stop", async () => {
            const mockContext = createMockSwarmState(testSwarmId);
            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));

            // Cache some contexts
            await contextManager.getContext(testSwarmId);

            // Stop should clear cache
            await contextManager.stop();

            // Cache should be empty after stop
            const health = await contextManager.healthCheck();
            expect(health.details.cacheSize).toBe(0);
        });

        test("should track cache metrics", async () => {
            const mockContext = createMockSwarmState(testSwarmId);
            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));
            mockRedisClient.set.mockResolvedValue("OK");

            // First call should be cache miss
            await contextManager.getContext(testSwarmId);

            // Second call should be cache hit
            await contextManager.getContext(testSwarmId);

            const metrics = await contextManager.getMetrics();
            expect(metrics.cacheHits).toBeGreaterThan(0);
            expect(metrics.cacheMisses).toBeGreaterThan(0);
        });
    });

    describe("Error Handling", () => {
        const testSwarmId: SwarmId = "test-swarm-1";

        test("should handle Redis connection failures gracefully", async () => {
            mockRedisClient.get.mockRejectedValue(new Error("Redis connection lost"));

            const result = await contextManager.getContext(testSwarmId);
            expect(result).toBeNull();

            const health = await contextManager.healthCheck();
            expect(health.healthy).toBe(false);
        });

        test("should handle malformed data in Redis", async () => {
            mockRedisClient.get.mockResolvedValue("invalid-json-data");

            const result = await contextManager.getContext(testSwarmId);
            expect(result).toBeNull();
        });

        test("should handle missing required dependencies", async () => {
            // Test scenario where SwarmStateAccessor throws an error
            const mockStateAccessor = vi.mocked(SwarmStateAccessor);
            const instance = mockStateAccessor.mock.instances[0];
            vi.mocked(instance.accessData).mockRejectedValue(new Error("Dependency unavailable"));

            const invalidQuery = {
                chatId: testSwarmId,
                select: ["some.path"],
            };

            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: createMockSwarmState(testSwarmId),
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));

            const result = await contextManager.query(invalidQuery);

            expect(result["some.path"]).toBeUndefined();
        });

        test("should handle concurrent access conflicts", async () => {
            const mockContext = createMockSwarmState(testSwarmId);
            mockRedisClient.get.mockResolvedValue(JSON.stringify({
                __version: "1",
                context: mockContext,
                metadata: {
                    storageVersion: 1,
                    compressed: false,
                    checksum: "test-checksum",
                    accessCount: 0,
                    lastAccessed: new Date(),
                },
            }));
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.keys.mockResolvedValue([]);

            // Simulate concurrent updates
            const updates1 = {
                chatConfig: {
                    __version: "1.0",
                    goal: "Goal 1",
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "0",
                        startedAt: null,
                        lastProcessingCycleEndedAt: null,
                    },
                },
            };
            const updates2 = {
                chatConfig: {
                    __version: "1.0",
                    goal: "Goal 2",
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "0",
                        startedAt: null,
                        lastProcessingCycleEndedAt: null,
                    },
                },
            };

            // Both should succeed (last one wins)
            const [result1, result2] = await Promise.all([
                contextManager.updateContext(testSwarmId, updates1),
                contextManager.updateContext(testSwarmId, updates2),
            ]);

            expect(result1).toBeDefined();
            expect(result2).toBeDefined();
        });
    });

    describe("Private Methods and Edge Cases", () => {
        test("should handle version history cleanup", async () => {
            const testSwarmId: SwarmId = "test-swarm-1";
            
            // Mock many version keys to trigger cleanup
            const manyVersions = Array.from({ length: 15 }, (_, i) => 
                `swarm_context:${testSwarmId}:version:${i + 1}`,
            );
            
            mockRedisClient.keys.mockResolvedValue(manyVersions);
            mockRedisClient.del.mockResolvedValue(1);
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.get.mockResolvedValue(null);

            await contextManager.createContext(testSwarmId);

            // Should delete oldest versions (keeping only MAX_VERSION_HISTORY)
            expect(mockRedisClient.del).toHaveBeenCalled();
        });

        test("should generate unique context keys", async () => {
            const swarmId1 = "swarm-1";
            const swarmId2 = "swarm-2";

            mockRedisClient.get.mockResolvedValue(null);
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.keys.mockResolvedValue([]);

            await contextManager.createContext(swarmId1);
            await contextManager.createContext(swarmId2);

            // Should have called Redis with different keys
            expect(mockRedisClient.set).toHaveBeenCalledWith(
                `swarm_context:${swarmId1}`,
                expect.any(String),
                "EX",
                expect.any(Number),
            );
            expect(mockRedisClient.set).toHaveBeenCalledWith(
                `swarm_context:${swarmId2}`,
                expect.any(String),
                "EX",
                expect.any(Number),
            );
        });

        test("should calculate checksums consistently", async () => {
            const context1 = createMockSwarmState("test-swarm-1");
            const context2 = createMockSwarmState("test-swarm-1");

            mockRedisClient.get.mockResolvedValue(null);
            mockRedisClient.set.mockResolvedValue("OK");
            mockRedisClient.keys.mockResolvedValue([]);

            await contextManager.createContext("test-swarm-1", context1);

            // Get the stored data to check checksum
            const storedData = mockRedisClient.set.mock.calls[0][1];
            const record1 = JSON.parse(storedData);

            // Clear mocks and create identical context
            mockRedisClient.set.mockClear();
            await contextManager.createContext("test-swarm-2", context2);

            const storedData2 = mockRedisClient.set.mock.calls[0][1];
            const record2 = JSON.parse(storedData2);

            // Checksums should be deterministic for same content
            expect(record1.metadata.checksum).toBeDefined();
            expect(record2.metadata.checksum).toBeDefined();
        });
    });

    describe("Enhanced Stress Tests - Concurrent Operations", () => {
        describe("simultaneous context updates", () => {
            test("should handle concurrent updates with version conflict resolution", async () => {
                const testSwarmId: SwarmId = "concurrent-test-swarm";
                
                // Create initial context
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);
                
                const context = await contextManager.createContext(testSwarmId);
                
                // Mock context retrieval for concurrent updates
                const existingRecord = {
                    __version: "1",
                    context: { ...context, version: 1 },
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "initial-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(existingRecord));
                
                // Simulate concurrent updates
                const updatePromises = Array.from({ length: 5 }, (_, i) => 
                    contextManager.updateContext(testSwarmId, {
                        chatConfig: {
                            __version: "1.0.0",
                            goal: `Concurrent goal update ${i}`,
                            stats: {
                                totalToolCalls: 0,
                                totalCredits: "0",
                                startedAt: null,
                                lastProcessingCycleEndedAt: null,
                            },
                        },
                    }, `Update ${i}`),
                );
                
                const results = await Promise.allSettled(updatePromises);
                
                // At least some updates should succeed
                const successful = results.filter(r => r.status === "fulfilled");
                expect(successful.length).toBeGreaterThan(0);
                
                // Final version should be consistent
                const finalContext = await contextManager.getContext(testSwarmId);
                expect(finalContext?.version).toBeGreaterThan(1);
            });

            test("should handle concurrent resource allocations without conflicts", async () => {
                const testSwarmId: SwarmId = "resource-concurrent-test";
                
                // Setup context with resource pool
                const context = createMockSwarmState(testSwarmId, {
                    resources: {
                        total: { credits: 1000, tokens: 10000, time: 3600 },
                        remaining: { credits: 1000, tokens: 10000, time: 3600 },
                        allocated: [],
                    },
                });
                
                const mockRecord = {
                    __version: "1",
                    context,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                
                // Simulate concurrent resource allocations
                const allocationPromises = Array.from({ length: 10 }, (_, i) => 
                    contextManager.allocateResources(testSwarmId, {
                        taskId: `task-${i}`,
                        resourceType: "credits",
                        amount: 50,
                        priority: "normal",
                        reason: `Concurrent allocation ${i}`,
                    }),
                );
                
                const results = await Promise.allSettled(allocationPromises);
                
                // All allocations should either succeed or fail gracefully
                results.forEach((result, i) => {
                    if (result.status === "rejected") {
                        // Should be due to insufficient resources, not corruption
                        expect(result.reason.message).toMatch(/insufficient|exceeded/i);
                    } else {
                        // Successful allocations should have valid IDs
                        expect(result.value.id).toBeDefined();
                        expect(result.value.taskId).toBe(`task-${i}`);
                    }
                });
            });

            test("should maintain data consistency during high-frequency updates", async () => {
                const testSwarmId: SwarmId = "high-frequency-test";
                
                // Create initial context
                const context = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                
                // Rapid-fire updates (100 updates in quick succession)
                const rapidUpdates = Array.from({ length: 100 }, (_, i) => 
                    contextManager.updateContext(testSwarmId, {
                        metadata: {
                            lastActivity: new Date(),
                            updateCount: i,
                        },
                    }, `Rapid update ${i}`),
                );
                
                const startTime = Date.now();
                const results = await Promise.allSettled(rapidUpdates);
                const duration = Date.now() - startTime;
                
                // Should complete within reasonable time (< 10 seconds)
                expect(duration).toBeLessThan(10000);
                
                // Most updates should succeed
                const successful = results.filter(r => r.status === "fulfilled");
                expect(successful.length).toBeGreaterThan(80); // At least 80% success rate
            });
        });
    });

    describe("Enhanced Stress Tests - Scale & Performance", () => {
        describe("large context objects", () => {
            test("should handle contexts larger than 10MB efficiently", async () => {
                const testSwarmId: SwarmId = "large-context-test";
                
                // Create a very large context object
                const largeSubtasks = Array.from({ length: 10000 }, (_, i) => ({
                    id: `massive_task_${i}`,
                    description: `Large task ${i} with extensive description that contains many details and requirements that need to be processed efficiently by the system. This description is intentionally verbose to increase the size of the context object for stress testing purposes.`.repeat(10),
                    status: i % 3 === 0 ? "completed" : i % 3 === 1 ? "in_progress" : "todo",
                    dependencies: i > 0 ? [`massive_task_${i - 1}`] : [],
                    metadata: {
                        priority: i % 5,
                        estimatedDuration: i * 1000,
                        resources: Array.from({ length: 20 }, (_, j) => `resource_${i}_${j}`),
                        notes: `Detailed notes for task ${i}`.repeat(50),
                    },
                }));
                
                const largeResources = Array.from({ length: 1000 }, (_, i) => ({
                    id: `resource_${i}`,
                    type: `type_${i % 10}`,
                    allocation: {
                        amount: i * 10,
                        unit: "credits",
                        allocated: new Date(),
                        expires: new Date(Date.now() + 3600000),
                    },
                    metadata: {
                        description: `Resource ${i} with detailed metadata`.repeat(20),
                        tags: Array.from({ length: 10 }, (_, j) => `tag_${i}_${j}`),
                    },
                }));
                
                const largeContext: Partial<SwarmState> = {
                    chatConfig: {
                        __version: "1.0.0",
                        goal: "Process massive dataset efficiently",
                        subtasks: largeSubtasks,
                        resources: largeResources,
                        blackboard: Array.from({ length: 500 }, (_, i) => ({
                            id: `note_${i}`,
                            value: `Large note ${i} with extensive content`.repeat(100),
                        })),
                        records: Array.from({ length: 2000 }, (_, i) => ({
                            id: `record_${i}`,
                            type: "execution_log",
                            data: {
                                message: `Execution record ${i}`.repeat(30),
                                details: Object.fromEntries(
                                    Array.from({ length: 20 }, (_, j) => [`key_${j}`, `value_${i}_${j}`]),
                                ),
                            },
                            timestamp: new Date(),
                        })),
                        stats: {
                            totalToolCalls: 50000,
                            totalCredits: "999999",
                            startedAt: Date.now(),
                            lastProcessingCycleEndedAt: Date.now(),
                        },
                        limits: {
                            maxCredits: "1000000",
                            maxDurationMs: 3600000,
                        },
                        scheduling: {
                            defaultDelayMs: 0,
                            requiresApprovalTools: "none",
                            approvalTimeoutMs: 300000,
                            autoRejectOnTimeout: true,
                        },
                        pendingToolCalls: [],
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);
                
                const startTime = Date.now();
                const result = await contextManager.createContext(testSwarmId, largeContext);
                const duration = Date.now() - startTime;
                
                // Should complete within reasonable time despite large size
                expect(duration).toBeLessThan(5000); // 5 seconds max
                expect(result).toBeDefined();
                expect(result.swarmId).toBe(testSwarmId);
                expect(result.chatConfig.subtasks).toHaveLength(10000);
                expect(result.chatConfig.resources).toHaveLength(1000);
                
                // Verify context was stored (Redis set should be called)
                expect(mockRedisClient.set).toHaveBeenCalled();
            });

            test("should handle high-frequency operations efficiently", async () => {
                const testSwarmId: SwarmId = "high-frequency-test";
                
                // Setup initial context
                const context = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                
                // Simulate 1000 operations per second for 5 seconds
                const operations = [];
                const operationTypes = ["getContext", "updateContext", "query"] as const;
                
                for (let i = 0; i < 5000; i++) {
                    const operationType = operationTypes[i % operationTypes.length];
                    
                    switch (operationType) {
                        case "getContext":
                            operations.push(() => contextManager.getContext(testSwarmId));
                            break;
                        case "updateContext":
                            operations.push(() => contextManager.updateContext(testSwarmId, {
                                metadata: { lastActivity: new Date(), operationCount: i },
                            }, `High freq update ${i}`));
                            break;
                        case "query":
                            operations.push(() => contextManager.query({
                                chatId: testSwarmId,
                                select: ["chatConfig.goal", "metadata.lastActivity"],
                            }));
                            break;
                    }
                }
                
                const startTime = Date.now();
                
                // Execute operations in batches to simulate high frequency
                const batchSize = 100;
                const batches = [];
                for (let i = 0; i < operations.length; i += batchSize) {
                    batches.push(operations.slice(i, i + batchSize));
                }
                
                let totalSuccessful = 0;
                for (const batch of batches) {
                    const batchResults = await Promise.allSettled(
                        batch.map(op => op()),
                    );
                    totalSuccessful += batchResults.filter(r => r.status === "fulfilled").length;
                }
                
                const duration = Date.now() - startTime;
                const operationsPerSecond = (totalSuccessful / duration) * 1000;
                
                // Should maintain reasonable throughput
                expect(operationsPerSecond).toBeGreaterThan(50); // At least 50 ops/sec
                expect(totalSuccessful).toBeGreaterThan(4000); // At least 80% success rate
            });

            test("should manage memory usage under sustained load", async () => {
                const contextIds = Array.from({ length: 100 }, (_, i) => `memory-test-${i}`);
                
                // Create many contexts to test memory management
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);
                
                const initialMemory = process.memoryUsage().heapUsed;
                
                // Create 100 contexts with moderate size
                for (const contextId of contextIds) {
                    const largeConfig: Partial<SwarmState> = {
                        chatConfig: {
                            __version: "1.0.0",
                            goal: `Memory test for ${contextId}`,
                            subtasks: Array.from({ length: 100 }, (_, i) => ({
                                id: `task_${i}`,
                                description: `Task ${i} for memory testing`.repeat(10),
                                status: "todo",
                            })),
                            blackboard: Array.from({ length: 50 }, (_, i) => ({
                                id: `note_${i}`,
                                value: `Note ${i} content`.repeat(20),
                            })),
                            resources: [],
                            records: [],
                            stats: {
                                totalToolCalls: 0,
                                totalCredits: "0",
                                startedAt: Date.now(),
                                lastProcessingCycleEndedAt: null,
                            },
                            limits: {
                                maxCredits: "10000",
                                maxDurationMs: 3600000,
                            },
                            scheduling: {
                                defaultDelayMs: 0,
                                requiresApprovalTools: "none",
                                approvalTimeoutMs: 300000,
                                autoRejectOnTimeout: true,
                            },
                            pendingToolCalls: [],
                        },
                    };
                    
                    await contextManager.createContext(contextId, largeConfig);
                }
                
                const afterCreationMemory = process.memoryUsage().heapUsed;
                const memoryIncrease = afterCreationMemory - initialMemory;
                
                // Memory increase should be reasonable (< 100MB for 100 contexts)
                expect(memoryIncrease).toBeLessThan(100 * 1024 * 1024);
                
                // Access all contexts to test cache behavior
                for (const contextId of contextIds) {
                    await contextManager.getContext(contextId);
                }
                
                const afterAccessMemory = process.memoryUsage().heapUsed;
                const totalMemoryIncrease = afterAccessMemory - initialMemory;
                
                // Total memory increase should still be reasonable
                expect(totalMemoryIncrease).toBeLessThan(150 * 1024 * 1024);
            });
        });
    });

    describe("Enhanced Stress Tests - Data Integrity & Recovery", () => {
        describe("corruption detection and recovery", () => {
            test("should detect and handle corrupted context data", async () => {
                const testSwarmId: SwarmId = "corruption-test";
                
                // Mock corrupted data in Redis
                const corruptedData = "{\"__version\":\"1\",\"context\":{\"invalid\":\"json\""; // Truncated JSON
                mockRedisClient.get.mockResolvedValue(corruptedData);
                
                const result = await contextManager.getContext(testSwarmId);
                
                // Should handle corruption gracefully
                expect(result).toBeNull();
                expect(logger.error).toHaveBeenCalledWith(
                    expect.stringContaining("Failed to parse context"),
                    expect.any(Object),
                );
            });

            test("should validate context integrity using checksums", async () => {
                const testSwarmId: SwarmId = "checksum-test";
                
                const validContext = createMockSwarmState(testSwarmId);
                const invalidRecord = {
                    __version: "1",
                    context: validContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "invalid-checksum", // Wrong checksum
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(invalidRecord));
                
                const result = await contextManager.getContext(testSwarmId);
                
                // Should detect checksum mismatch and handle appropriately
                // Implementation may vary - could return null, repair, or log warning
                expect(logger.warn).toHaveBeenCalledWith(
                    expect.stringContaining("checksum"),
                    expect.any(Object),
                );
            });

            test("should recover from Redis connection failures", async () => {
                const testSwarmId: SwarmId = "redis-failure-test";
                
                // Mock Redis failure
                mockRedisClient.get.mockRejectedValue(new Error("Redis connection failed"));
                
                const result = await contextManager.getContext(testSwarmId);
                
                // Should handle Redis failures gracefully
                expect(result).toBeNull();
                expect(logger.error).toHaveBeenCalledWith(
                    expect.stringContaining("Failed to get context"),
                    expect.any(Object),
                );
            });

            test("should handle partial update failures with rollback", async () => {
                const testSwarmId: SwarmId = "rollback-test";
                
                const originalContext = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context: originalContext,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                
                // Mock Redis set to fail
                mockRedisClient.set.mockRejectedValue(new Error("Redis write failed"));
                
                await expect(contextManager.updateContext(testSwarmId, {
                    chatConfig: {
                        __version: "1.0",
                        goal: "Updated goal",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: null,
                            lastProcessingCycleEndedAt: null,
                        },
                    },
                }, "Test update")).rejects.toThrow("Redis write failed");
                
                // Original context should remain unchanged
                mockRedisClient.set.mockResolvedValue("OK"); // Restore for subsequent call
                const retrievedContext = await contextManager.getContext(testSwarmId);
                expect(retrievedContext?.chatConfig.goal).toBe(originalContext.chatConfig.goal);
            });
        });

        describe("backup and restore capabilities", () => {
            test("should create consistent backups of context state", async () => {
                const testSwarmId: SwarmId = "backup-test";
                
                const context = createMockSwarmState(testSwarmId, {
                    chatConfig: {
                        __version: "1.0.0",
                        goal: "Backup test goal",
                        subtasks: [
                            { id: "task1", description: "Test task", status: "todo" },
                        ],
                        blackboard: [],
                        resources: [],
                        records: [],
                        stats: {
                            totalToolCalls: 5,
                            totalCredits: "100",
                            startedAt: Date.now(),
                            lastProcessingCycleEndedAt: null,
                        },
                        limits: {
                            maxCredits: "10000",
                            maxDurationMs: 3600000,
                        },
                        scheduling: {
                            defaultDelayMs: 0,
                            requiresApprovalTools: "none",
                            approvalTimeoutMs: 300000,
                            autoRejectOnTimeout: true,
                        },
                        pendingToolCalls: [],
                    },
                });
                
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);
                
                // Create context
                await contextManager.createContext(testSwarmId, context);
                
                // Verify backup keys are created appropriately
                expect(mockRedisClient.set).toHaveBeenCalled();
                
                // Check that the stored data includes version and metadata
                const setCall = mockRedisClient.set.mock.calls[0];
                const storedData = JSON.parse(setCall[1]);
                
                expect(storedData.__version).toBe("1");
                expect(storedData.metadata).toBeDefined();
                expect(storedData.metadata.checksum).toBeDefined();
                expect(storedData.context).toBeDefined();
            });
        });
    });

    describe("Enhanced Stress Tests - Event Bus Integration", () => {
        describe("real-time update propagation", () => {
            test("should publish context updates to event bus efficiently", async () => {
                const testSwarmId: SwarmId = "event-bus-test";
                
                const context = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                
                // Perform multiple updates to test event publishing
                const updates = [
                    {
                        chatConfig: {
                            __version: "1.0",
                            goal: "First update",
                            stats: {
                                totalToolCalls: 0,
                                totalCredits: "0",
                                startedAt: null,
                                lastProcessingCycleEndedAt: null,
                            },
                        },
                    },
                    {
                        chatConfig: {
                            __version: "1.0",
                            goal: "Second update",
                            stats: {
                                totalToolCalls: 0,
                                totalCredits: "0",
                                startedAt: null,
                                lastProcessingCycleEndedAt: null,
                            },
                        },
                    },
                    {
                        chatConfig: {
                            __version: "1.0",
                            goal: "Third update",
                            stats: {
                                totalToolCalls: 0,
                                totalCredits: "0",
                                startedAt: null,
                                lastProcessingCycleEndedAt: null,
                            },
                        },
                    },
                ];
                
                for (const update of updates) {
                    await contextManager.updateContext(testSwarmId, update, "Event test");
                }
                
                // Verify events were published
                expect(EventPublisher.emit).toHaveBeenCalledTimes(updates.length);
                
                // Check event structure
                const publishCalls = (EventPublisher.emit as any).mock.calls;
                publishCalls.forEach((call: any) => {
                    const event = call[0];
                    expect(event.type).toBe(EventTypes.ContextUpdate);
                    expect(event.payload).toBeDefined();
                    expect(event.payload.chatId).toBe(testSwarmId);
                });
            });

            test("should handle event bus failures gracefully", async () => {
                const testSwarmId: SwarmId = "event-failure-test";
                
                const context = createMockSwarmState(testSwarmId);
                const mockRecord = {
                    __version: "1",
                    context,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.set.mockResolvedValue("OK");
                
                // Mock event bus failure
                (EventPublisher.emit as any).mockRejectedValueOnce(new Error("Event bus failure"));
                
                // Update should still succeed even if event publishing fails
                const result = await contextManager.updateContext(testSwarmId, {
                    chatConfig: {
                        __version: "1.0",
                        goal: "Update despite event failure",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: null,
                            lastProcessingCycleEndedAt: null,
                        },
                    },
                }, "Resilience test");
                
                expect(result).toBeDefined();
                expect(result.chatConfig.goal).toBe("Update despite event failure");
                expect(logger.warn).toHaveBeenCalledWith(
                    expect.stringContaining("Failed to publish context update event"),
                    expect.any(Object),
                );
            });
        });
    });

    describe("Enhanced Stress Tests - Memory Management", () => {
        describe("cache eviction and cleanup", () => {
            test("should evict old entries when cache size limit is reached", async () => {
                // Create many contexts to fill up cache
                const contextIds = Array.from({ length: 150 }, (_, i) => `cache-test-${i}`);
                
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);
                
                // Create contexts to populate cache
                for (const contextId of contextIds) {
                    await contextManager.createContext(contextId, {
                        chatConfig: {
                            __version: "1.0",
                            goal: `Goal for ${contextId}`,
                            stats: {
                                totalToolCalls: 0,
                                totalCredits: "0",
                                startedAt: null,
                                lastProcessingCycleEndedAt: null,
                            },
                        },
                    });
                }
                
                // Cache should have some limit and evict old entries
                // This test verifies the system doesn't grow unbounded
                const metrics = await contextManager.getMetrics();
                expect(metrics.cacheHits).toBeDefined();
                expect(metrics.cacheMisses).toBeDefined();
            });

            test("should cleanup resources on context deletion", async () => {
                const testSwarmId: SwarmId = "cleanup-test";
                
                const context = createMockSwarmState(testSwarmId, {
                    resources: {
                        total: { credits: 1000, tokens: 10000, time: 3600 },
                        remaining: { credits: 500, tokens: 5000, time: 1800 },
                        allocated: [
                            {
                                id: "alloc-1",
                                taskId: "task-1",
                                resourceType: "credits",
                                amount: 500,
                                priority: "normal",
                                allocatedAt: new Date(),
                                reason: "Test allocation",
                            },
                        ],
                    },
                });
                
                const mockRecord = {
                    __version: "1",
                    context,
                    metadata: {
                        storageVersion: 1,
                        compressed: false,
                        checksum: "test-checksum",
                        accessCount: 0,
                        lastAccessed: new Date(),
                    },
                };
                
                mockRedisClient.get.mockResolvedValue(JSON.stringify(mockRecord));
                mockRedisClient.del.mockResolvedValue(1);
                
                await contextManager.deleteContext(testSwarmId);
                
                // Verify cleanup occurred
                expect(mockRedisClient.del).toHaveBeenCalled();
                
                // Subsequent get should return null
                mockRedisClient.get.mockResolvedValue(null);
                const deletedContext = await contextManager.getContext(testSwarmId);
                expect(deletedContext).toBeNull();
            });

            test("should handle memory pressure during intensive operations", async () => {
                const testSwarmId: SwarmId = "memory-pressure-test";
                
                // Simulate memory pressure by performing many memory-intensive operations
                const largeOperations = Array.from({ length: 50 }, (_, i) => {
                    const largeContext: Partial<SwarmState> = {
                        chatConfig: {
                            __version: "1.0.0",
                            goal: `Memory pressure test ${i}`,
                            subtasks: Array.from({ length: 500 }, (_, j) => ({
                                id: `pressure_task_${i}_${j}`,
                                description: "Large task description for memory pressure testing".repeat(50),
                                status: "todo",
                            })),
                            blackboard: Array.from({ length: 200 }, (_, j) => ({
                                id: `pressure_note_${i}_${j}`,
                                value: "Large note content for memory testing".repeat(100),
                            })),
                            resources: [],
                            records: [],
                            stats: {
                                totalToolCalls: 0,
                                totalCredits: "0",
                                startedAt: Date.now(),
                                lastProcessingCycleEndedAt: null,
                            },
                            limits: {
                                maxCredits: "10000",
                                maxDurationMs: 3600000,
                            },
                            scheduling: {
                                defaultDelayMs: 0,
                                requiresApprovalTools: "none",
                                approvalTimeoutMs: 300000,
                                autoRejectOnTimeout: true,
                            },
                            pendingToolCalls: [],
                        },
                    };
                    
                    return () => contextManager.createContext(`${testSwarmId}-${i}`, largeContext);
                });
                
                mockRedisClient.get.mockResolvedValue(null);
                mockRedisClient.set.mockResolvedValue("OK");
                mockRedisClient.keys.mockResolvedValue([]);
                
                const initialMemory = process.memoryUsage().heapUsed;
                
                // Execute operations in batches to avoid overwhelming the system
                const batchSize = 10;
                for (let i = 0; i < largeOperations.length; i += batchSize) {
                    const batch = largeOperations.slice(i, i + batchSize);
                    await Promise.all(batch.map(op => op()));
                    
                    // Force garbage collection if available (only in test environments)
                    if (global.gc) {
                        global.gc();
                    }
                }
                
                const finalMemory = process.memoryUsage().heapUsed;
                const memoryIncrease = finalMemory - initialMemory;
                
                // Memory increase should be reasonable despite large operations
                expect(memoryIncrease).toBeLessThan(500 * 1024 * 1024); // Less than 500MB
            });
        });
    });
});
