import { expect, describe, it, beforeEach, afterEach, beforeAll, afterAll } from "vitest";
import winston from "winston";
import { RedisSwarmStateStore } from "../../tier1/state/redisSwarmStateStore.js";
import { SwarmState, type Swarm } from "@vrooli/shared";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import IORedis from "ioredis";

describe("RedisSwarmStateStore", () => {
    let store: RedisSwarmStateStore;
    let logger: winston.Logger;
    let redisContainer: StartedTestContainer;
    let testRedis: IORedis;

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

    before(async function() {
        this.timeout(60000); // Container startup can take time
        
        // Start Redis container
        redisContainer = await new GenericContainer("redis:7-alpine")
            .withExposedPorts(6379)
            .start();
            
        const redisHost = redisContainer.getHost();
        const redisPort = redisContainer.getMappedPort(6379);
        
        // Create test Redis client
        testRedis = new IORedis({
            host: redisHost,
            port: redisPort,
        });
        
        // Wait for Redis to be ready
        await testRedis.ping();
    });

    after(async () => {
        if (testRedis) {
            await testRedis.quit();
        }
        if (redisContainer) {
            await redisContainer.stop();
        }
    });

    beforeEach(async () => {
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });

        // Clear Redis before each test
        await testRedis.flushall();

        // Create store with test Redis connection
        store = new RedisSwarmStateStore(logger, testRedis);
    });

    afterEach(async () => {
        // Clean up test data
        await testRedis.flushall();
    });

    describe("createSwarm", () => {
        it("should store swarm data in Redis with TTL", async () => {
            await store.createSwarm("swarm-123", mockSwarm);

            // Verify the swarm was stored
            const storedData = await testRedis.get("swarm:swarm-123");
            expect(storedData).to.not.be.null;
            
            const parsedSwarm = JSON.parse(storedData!);
            expect(parsedSwarm.id).toBe("swarm-123");
            expect(parsedSwarm.state).toBe(SwarmState.ACTIVE);
            
            // Check TTL was set
            const ttl = await testRedis.ttl("swarm:swarm-123");
            expect(ttl).toBeGreaterThan(0);
            expect(ttl).toBeLessThanOrEqual(86400); // 24 hours
        });

        it("should add swarm to active set if in ACTIVE state", async () => {
            await store.createSwarm("swarm-123", mockSwarm);

            const activeSwarms = await testRedis.smembers("swarms:active");
            expect(activeSwarms).toContain("swarm-123");
        });

        it("should add swarm to user's swarm set", async () => {
            await store.createSwarm("swarm-123", mockSwarm);

            const userSwarms = await testRedis.smembers("user:user-123:swarms");
            expect(userSwarms).toContain("swarm-123");
        });
    });

    describe("getSwarm", () => {
        it("should retrieve swarm from Redis", async () => {
            // Store swarm first
            await testRedis.set("swarm:swarm-123", JSON.stringify(mockSwarm));

            const retrievedSwarm = await store.getSwarm("swarm-123");
            
            expect(retrievedSwarm).to.not.be.null;
            expect(retrievedSwarm!.id).toBe("swarm-123");
            expect(retrievedSwarm!.config.name).toBe("Test Swarm");
        });

        it("should return null for non-existent swarm", async () => {
            const retrievedSwarm = await store.getSwarm("non-existent");
            expect(retrievedSwarm).toBeNull();
        });

        it("should parse dates correctly", async () => {
            await testRedis.set("swarm:swarm-123", JSON.stringify(mockSwarm));

            const retrievedSwarm = await store.getSwarm("swarm-123");
            
            expect(retrievedSwarm!.createdAt).to.be.instanceOf(Date);
            expect(retrievedSwarm!.updatedAt).to.be.instanceOf(Date);
        });
    });

    describe("updateSwarm", () => {
        beforeEach(async () => {
            // Pre-store the swarm
            await store.createSwarm("swarm-123", mockSwarm);
        });

        it("should update swarm data", async () => {
            const updatedSwarm = {
                ...mockSwarm,
                config: {
                    ...mockSwarm.config,
                    name: "Updated Swarm Name",
                },
            };

            await store.updateSwarm("swarm-123", updatedSwarm);

            const retrieved = await store.getSwarm("swarm-123");
            expect(retrieved!.config.name).toBe("Updated Swarm Name");
        });

        it("should update active set when state changes", async () => {
            const inactiveSwarm = {
                ...mockSwarm,
                state: SwarmState.COMPLETED,
            };

            await store.updateSwarm("swarm-123", inactiveSwarm);

            const activeSwarms = await testRedis.smembers("swarms:active");
            expect(activeSwarms).to.not.include("swarm-123");
        });
    });

    describe("deleteSwarm", () => {
        beforeEach(async () => {
            await store.createSwarm("swarm-123", mockSwarm);
        });

        it("should remove swarm from Redis", async () => {
            await store.deleteSwarm("swarm-123");

            const retrieved = await testRedis.get("swarm:swarm-123");
            expect(retrieved).toBeNull();
        });

        it("should remove from active set", async () => {
            await store.deleteSwarm("swarm-123");

            const activeSwarms = await testRedis.smembers("swarms:active");
            expect(activeSwarms).to.not.include("swarm-123");
        });

        it("should remove from user's swarm set", async () => {
            await store.deleteSwarm("swarm-123");

            const userSwarms = await testRedis.smembers("user:user-123:swarms");
            expect(userSwarms).to.not.include("swarm-123");
        });
    });

    describe("state operations", () => {
        beforeEach(async () => {
            await store.createSwarm("swarm-123", mockSwarm);
        });

        it("should get current swarm state", async () => {
            const state = await store.getSwarmState("swarm-123");
            expect(state).toBe(SwarmState.ACTIVE);
        });

        it("should update swarm state", async () => {
            await store.updateSwarmState("swarm-123", SwarmState.PAUSED);

            const state = await store.getSwarmState("swarm-123");
            expect(state).toBe(SwarmState.PAUSED);
        });
    });

    describe("list operations", () => {
        beforeEach(async () => {
            // Create multiple swarms
            await store.createSwarm("swarm-1", { ...mockSwarm, id: "swarm-1", state: SwarmState.ACTIVE });
            await store.createSwarm("swarm-2", { ...mockSwarm, id: "swarm-2", state: SwarmState.PAUSED });
            await store.createSwarm("swarm-3", { ...mockSwarm, id: "swarm-3", state: SwarmState.ACTIVE });
        });

        it("should list active swarms", async () => {
            const activeSwarms = await store.listActiveSwarms();
            
            expect(activeSwarms).toHaveLength(2);
            expect(activeSwarms.map(s => s.id)).toContain.members(["swarm-1", "swarm-3"]);
        });

        it("should get swarms by state", async () => {
            const pausedSwarms = await store.getSwarmsByState(SwarmState.PAUSED);
            
            expect(pausedSwarms).toHaveLength(1);
            expect(pausedSwarms[0].id).toBe("swarm-2");
        });

        it("should get swarms by user", async () => {
            const userSwarms = await store.getSwarmsByUser("user-123");
            
            expect(userSwarms).toHaveLength(3);
            expect(userSwarms.map(s => s.id)).toContain.members(["swarm-1", "swarm-2", "swarm-3"]);
        });
    });
});