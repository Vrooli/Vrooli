import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { RedisSwarmStateStore } from "../../tier1/state/redisSwarmStateStore.js";
import { redis } from "../../../../services/redisConn.js";
import { SwarmState, type Swarm } from "@vrooli/shared";

describe("RedisSwarmStateStore", () => {
    let store: RedisSwarmStateStore;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility
    let redisStub: any;

    const mockSwarm: Swarm = {
        id: "swarm-123",
        state: SwarmState.ACTIVE,
        config: {
            name: "Test Swarm",
            description: "Test description",
            goal: "Test goal",
            model: "gpt-4o-mini",
            temperature: 0.7,
            autoApproveTools: false,
            parallelExecutionLimit: 5,
        },
        team: {
            agents: [],
            capabilities: [],
            activeMembers: 0,
        },
        resources: {
            allocated: {
                maxCredits: 10000,
                maxTokens: 100000,
                maxTime: 3600000,
            },
            consumed: {
                credits: 1000,
                tokens: 10000,
                time: 600000,
            },
            remaining: {
                maxCredits: 9000,
                maxTokens: 90000,
                maxTime: 3000000,
            },
        },
        metrics: {
            tasksCompleted: 5,
            tasksFailed: 1,
            avgTaskDuration: 120000,
            resourceEfficiency: 0.85,
        },
        createdAt: new Date("2024-01-01"),
        updatedAt: new Date("2024-01-02"),
        metadata: {
            userId: "user-123",
            organizationId: "org-123",
            version: "2.0.0",
        },
    };

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });

        // Stub Redis methods
        redisStub = {
            set: vi.fn().resolves("OK"),
            get: vi.fn(),
            del: vi.fn().resolves(1),
            expire: vi.fn().resolves(1),
            sadd: vi.fn().resolves(1),
            srem: vi.fn().resolves(1),
            smembers: vi.fn().resolves([]),
        };

        // Replace redis module
        sandbox.stub(redis, "set").callsFake(redisStub.set);
        sandbox.stub(redis, "get").callsFake(redisStub.get);
        sandbox.stub(redis, "del").callsFake(redisStub.del);
        sandbox.stub(redis, "expire").callsFake(redisStub.expire);
        sandbox.stub(redis, "sadd").callsFake(redisStub.sadd);
        sandbox.stub(redis, "srem").callsFake(redisStub.srem);
        sandbox.stub(redis, "smembers").callsFake(redisStub.smembers);

        store = new RedisSwarmStateStore(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("createSwarm", () => {
        it("should store swarm data in Redis with TTL", async () => {
            await store.createSwarm("swarm-123", mockSwarm);

            expect(redisStub.set).toHaveBeenCalledWith(
                "swarm:swarm-123",
                JSON.stringify(mockSwarm);,
                "EX",
                86400 * 7 // 7 days
            );
        });

        it("should update indexes when creating swarm", async () => {
            await store.createSwarm("swarm-123", mockSwarm);

            // Should add to state index
            expect(redisStub.sadd).toHaveBeenCalledWith(
                "swarm_index:state:ACTIVE",
                "swarm-123"
            );

            // Should add to user index
            expect(redisStub.sadd).toHaveBeenCalledWith(
                "swarm_index:user:user-123",
                "swarm-123"
            );
        });

        it("should handle Redis errors", async () => {
            redisStub.set.mockRejectedValue(new Error("Redis connection failed"););

            try {
                await store.createSwarm("swarm-123", mockSwarm);
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("Redis connection failed");
            }
        });
    });

    describe("getSwarm", () => {
        it("should retrieve and parse swarm data", async () => {
            redisStub.get.mockResolvedValue(JSON.stringify(mockSwarm););

            const result = await store.getSwarm("swarm-123");

            expect(result).toEqual(mockSwarm);
            expect(redisStub.get).toHaveBeenCalledWith("swarm:swarm-123");
        });

        it("should refresh TTL on access", async () => {
            redisStub.get.mockResolvedValue(JSON.stringify(mockSwarm););

            await store.getSwarm("swarm-123");

            expect(redisStub.expire).toHaveBeenCalledWith(
                "swarm:swarm-123",
                86400 * 7
            );
        });

        it("should return null if swarm not found", async () => {
            redisStub.get.mockResolvedValue(null);

            const result = await store.getSwarm("swarm-123");

            expect(result).to.be.null;
        });

        it("should handle JSON parse errors gracefully", async () => {
            redisStub.get.mockResolvedValue("invalid json");

            const result = await store.getSwarm("swarm-123");

            expect(result).to.be.null;
        });
    });

    describe("updateSwarm", () => {
        beforeEach(() => {
            redisStub.get.mockResolvedValue(JSON.stringify(mockSwarm););
        });

        it("should update swarm with new data", async () => {
            const updates = {
                state: SwarmState.STOPPED,
                metrics: { tasksCompleted: 10 },
            };

            await store.updateSwarm("swarm-123", updates);

            const savedData = JSON.parse(redisStub.set.firstCall.args[1]);
            expect(savedData.state).toBe(SwarmState.STOPPED);
            expect(savedData.metrics.tasksCompleted).toBe(10);
            expect(savedData.updatedAt).to.exist;
        });

        it("should update indexes when state changes", async () => {
            await store.updateSwarm("swarm-123", { state: SwarmState.STOPPED });

            // Should remove from all state indexes
            expect(redisStub.srem).toHaveBeenCalledWith(
                "swarm_index:state:ACTIVE",
                "swarm-123"
            );

            // Should add to new state index
            expect(redisStub.sadd).toHaveBeenCalledWith(
                "swarm_index:state:STOPPED",
                "swarm-123"
            );
        });

        it("should throw error if swarm not found", async () => {
            redisStub.get.mockResolvedValue(null);

            try {
                await store.updateSwarm("swarm-123", { state: SwarmState.STOPPED });
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("Swarm swarm-123 not found");
            }
        });
    });

    describe("deleteSwarm", () => {
        beforeEach(() => {
            redisStub.get.mockResolvedValue(JSON.stringify(mockSwarm););
        });

        it("should delete swarm and remove from indexes", async () => {
            await store.deleteSwarm("swarm-123");

            expect(redisStub.del).toHaveBeenCalledWith("swarm:swarm-123");
            expect(redisStub.srem).toHaveBeenCalledWith(
                "swarm_index:state:ACTIVE",
                "swarm-123"
            );
            expect(redisStub.srem).toHaveBeenCalledWith(
                "swarm_index:user:user-123",
                "swarm-123"
            );
        });

        it("should not throw if swarm doesn't exist", async () => {
            redisStub.get.mockResolvedValue(null);

            await store.deleteSwarm("swarm-123");
            // Should complete without throwing
        });
    });

    describe("listActiveSwarms", () => {
        it("should return swarms from all active states", async () => {
            redisStub.smembers
                .withArgs("swarm_index:state:INITIALIZING").resolves(["swarm-1"])
                .withArgs("swarm_index:state:ACTIVE").resolves(["swarm-2", "swarm-3"])
                .withArgs("swarm_index:state:READY").resolves(["swarm-4"]);

            const swarms = {
                "swarm-1": { ...mockSwarm, id: "swarm-1", state: SwarmState.INITIALIZING },
                "swarm-2": { ...mockSwarm, id: "swarm-2", state: SwarmState.ACTIVE },
                "swarm-3": { ...mockSwarm, id: "swarm-3", state: SwarmState.ACTIVE },
                "swarm-4": { ...mockSwarm, id: "swarm-4", state: SwarmState.READY },
            };

            redisStub.get.callsFake((key: string) => {
                const swarmId = key.replace("swarm:", "");
                return Promise.resolve(swarms[swarmId] ? JSON.stringify(swarms[swarmId]) : null);
            });

            const result = await store.listActiveSwarms();

            expect(result).toHaveLength(4);
            expect(result).to.include("swarm-1");
            expect(result).to.include("swarm-2");
            expect(result).to.include("swarm-3");
            expect(result).to.include("swarm-4");
        });

        it("should remove duplicates", async () => {
            redisStub.smembers
                .withArgs("swarm_index:state:ACTIVE").resolves(["swarm-1"])
                .withArgs("swarm_index:state:READY").resolves(["swarm-1"]); // Duplicate

            redisStub.get.mockResolvedValue(JSON.stringify({ ...mockSwarm, id: "swarm-1" }););

            const result = await store.listActiveSwarms();

            expect(result).toEqual(["swarm-1"]);
        });
    });

    describe("getSwarmsByState", () => {
        it("should return swarms in specific state", async () => {
            redisStub.smembers.mockResolvedValue(["swarm-1", "swarm-2"]);
            
            redisStub.get
                .withArgs("swarm:swarm-1").resolves(JSON.stringify({ ...mockSwarm, id: "swarm-1" }))
                .withArgs("swarm:swarm-2").resolves(JSON.stringify({ ...mockSwarm, id: "swarm-2" }));

            const result = await store.getSwarmsByState(SwarmState.ACTIVE);

            expect(result).toEqual(["swarm-1", "swarm-2"]);
        });

        it("should clean up stale index entries", async () => {
            redisStub.smembers.mockResolvedValue(["swarm-1", "swarm-2", "swarm-stale"]);
            
            redisStub.get
                .withArgs("swarm:swarm-1").resolves(JSON.stringify({ ...mockSwarm, id: "swarm-1" }))
                .withArgs("swarm:swarm-2").resolves(JSON.stringify({ ...mockSwarm, id: "swarm-2" }))
                .withArgs("swarm:swarm-stale").resolves(null); // Doesn't exist

            const result = await store.getSwarmsByState(SwarmState.ACTIVE);

            expect(result).toEqual(["swarm-1", "swarm-2"]);
            expect(redisStub.srem).toHaveBeenCalledWith(
                "swarm_index:state:ACTIVE",
                "swarm-stale"
            );
        });
    });

    describe("getSwarmsByUser", () => {
        it("should return swarms for specific user", async () => {
            redisStub.smembers.mockResolvedValue(["swarm-1", "swarm-2"]);
            
            const swarm1 = { ...mockSwarm, id: "swarm-1" };
            const swarm2 = { ...mockSwarm, id: "swarm-2" };
            
            redisStub.get
                .withArgs("swarm:swarm-1").resolves(JSON.stringify(swarm1))
                .withArgs("swarm:swarm-2").resolves(JSON.stringify(swarm2));

            const result = await store.getSwarmsByUser("user-123");

            expect(result).toEqual(["swarm-1", "swarm-2"]);
        });

        it("should handle Redis errors gracefully", async () => {
            redisStub.smembers.mockRejectedValue(new Error("Redis error"););

            const result = await store.getSwarmsByUser("user-123");

            expect(result).toEqual([]);
        });
    });
});