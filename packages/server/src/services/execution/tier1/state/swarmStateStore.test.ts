import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { InMemorySwarmStateStore } from "./swarmStateStore.js";
import { RedisSwarmStateStore } from "./redisSwarmStateStore.js";
import { type Logger } from "winston";
import {
    type Swarm,
    ExecutionStates,
    type ExecutionState,
    type TeamFormation,
    type SwarmAgent,
    type SwarmTeam,
    type BlackboardItem,
    type SwarmResource,
    generatePK,
} from "@vrooli/shared";

/**
 * SwarmStateStore Tests - Distributed State Infrastructure
 * 
 * These tests validate that the state store provides minimal infrastructure
 * for swarm coordination while enabling emergent intelligence through:
 * 
 * 1. **State Persistence**: Reliable storage without optimization logic
 * 2. **Team Management**: Infrastructure for agent coordination
 * 3. **Blackboard Pattern**: Shared memory without communication strategies
 * 4. **Resource Tracking**: Allocation without optimization algorithms
 * 5. **Query Interface**: Data access without analysis logic
 * 
 * The store provides data infrastructure - coordination intelligence
 * emerges from how agents use this shared state.
 */

describe("SwarmStateStore - State Management Infrastructure", () => {
    let logger: Logger;
    let store: InMemorySwarmStateStore;
    let redisStore: RedisSwarmStateStore;

    beforeEach(async () => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;
        
        // Test both in-memory and Redis implementations
        store = new InMemorySwarmStateStore(logger);
        
        // RedisSwarmStateStore uses CacheService internally
        redisStore = new RedisSwarmStateStore(logger);
        
        // Note: EventBus is not needed for the state store itself
        // If tests need EventBus, they should create it separately
    });

    afterEach(async () => {
        // RedisSwarmStateStore will handle its own cleanup
        // We don't need to manually clear Redis data as each test
        // should use unique IDs to avoid conflicts
    });

    describe("Swarm Lifecycle Management", () => {
        it("should manage basic swarm CRUD operations", async () => {
            const swarmId = generatePK();
            const swarm: Swarm = {
                id: swarmId,
                name: "Test Swarm",
                description: "Testing swarm operations",
                goal: "Validate state management",
                state: ExecutionStates.UNINITIALIZED,
                userId: generatePK(),
                config: {
                    maxCredits: 1000,
                    maxTokens: 50000,
                    maxTime: 3600000,
                },
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            // Test both implementations
            for (const storeImpl of [store, redisStore]) {
                // Create swarm
                await storeImpl.createSwarm(swarmId, swarm);
                
                // Retrieve swarm
                const retrieved = await storeImpl.getSwarm(swarmId);
                expect(retrieved).toEqual(swarm);
                
                // Update swarm
                const updates = { name: "Updated Swarm", goal: "New goal" };
                await storeImpl.updateSwarm(swarmId, updates);
                
                const updated = await storeImpl.getSwarm(swarmId);
                expect(updated?.name).toBe("Updated Swarm");
                expect(updated?.goal).toBe("New goal");
                
                // Clean up for next test
                await storeImpl.deleteSwarm(swarmId);
                const deleted = await storeImpl.getSwarm(swarmId);
                expect(deleted).toBeNull();
            }
        });

        it("should manage state transitions without validation logic", async () => {
            const swarmId = generatePK();
            const swarm: Swarm = {
                id: swarmId,
                name: "State Test Swarm",
                description: "Testing state transitions",
                goal: "Transition through states",
                state: ExecutionStates.UNINITIALIZED,
                userId: generatePK(),
                config: {},
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await store.createSwarm(swarmId, swarm);
            
            // Transition through states
            const states = [
                ExecutionStates.STARTING,
                ExecutionStates.RUNNING,
                ExecutionStates.PAUSED,
                ExecutionStates.RUNNING,
                ExecutionStates.STOPPED,
            ];
            
            for (const state of states) {
                await store.updateSwarmState(swarmId, state);
                const currentState = await store.getSwarmState(swarmId);
                expect(currentState).toBe(state);
            }
            
            // No validation logic - agents determine valid transitions
        });
    });

    describe("Team Management Infrastructure", () => {
        it("should manage team formation without organization logic", async () => {
            const swarmId = generatePK();
            const team: TeamFormation = {
                leadership: {
                    leaderId: generatePK(),
                    structure: "hierarchical",
                },
                roles: [
                    { id: "analyst", name: "Data Analyst", capabilities: ["sql", "python"] },
                    { id: "architect", name: "System Architect", capabilities: ["design", "planning"] },
                ],
                coordination: {
                    communicationPattern: "hub_spoke",
                    decisionMaking: "consensus",
                },
            };

            await store.updateTeam(swarmId, team);
            const retrieved = await store.getTeam(swarmId);
            
            expect(retrieved).toEqual(team);
            expect(retrieved?.roles).toHaveLength(2);
            expect(retrieved?.leadership.structure).toBe("hierarchical");
            
            // Team structure data without organizational intelligence
        });

        it("should manage individual swarm teams", async () => {
            const swarmId = generatePK();
            const teams: SwarmTeam[] = [
                {
                    id: "frontend-team",
                    name: "Frontend Team",
                    purpose: "UI development",
                    members: [generatePK(), generatePK()],
                    capabilities: ["react", "typescript", "css"],
                },
                {
                    id: "backend-team",
                    name: "Backend Team",
                    purpose: "API development",
                    members: [generatePK()],
                    capabilities: ["node", "database", "api_design"],
                },
            ];

            // Create teams
            for (const team of teams) {
                await store.createTeam!(swarmId, team);
            }
            
            // List all teams
            const allTeams = await store.listTeams!(swarmId);
            expect(allTeams).toHaveLength(2);
            
            // Get specific team
            const frontendTeam = await store.getTeamById!(swarmId, "frontend-team");
            expect(frontendTeam?.name).toBe("Frontend Team");
            expect(frontendTeam?.capabilities).toContain("react");
            
            // Update team
            await store.updateTeamById!(swarmId, "frontend-team", {
                capabilities: [...(frontendTeam?.capabilities || []), "nextjs"],
            });
            
            const updatedTeam = await store.getTeamById!(swarmId, "frontend-team");
            expect(updatedTeam?.capabilities).toContain("nextjs");
        });
    });

    describe("Agent Registration and Management", () => {
        it("should register and manage agents without assignment logic", async () => {
            const swarmId = generatePK();
            const agents: SwarmAgent[] = [
                {
                    id: generatePK(),
                    name: "DataAnalyzer",
                    role: "analyst",
                    capabilities: ["data_processing", "visualization"],
                    status: "active",
                    teamId: "analytics-team",
                },
                {
                    id: generatePK(),
                    name: "CodeReviewer",
                    role: "reviewer",
                    capabilities: ["code_analysis", "security_audit"],
                    status: "active",
                    teamId: "quality-team",
                },
            ];

            // Register agents
            for (const agent of agents) {
                await store.registerAgent!(swarmId, agent);
            }
            
            // List all agents
            const allAgents = await store.listAgents!(swarmId);
            expect(allAgents).toHaveLength(2);
            
            // Get specific agent
            const dataAgent = await store.getAgent!(swarmId, agents[0].id);
            expect(dataAgent?.name).toBe("DataAnalyzer");
            expect(dataAgent?.capabilities).toContain("data_processing");
            
            // Update agent status
            await store.updateAgent!(swarmId, agents[0].id, { status: "busy" });
            
            const updatedAgent = await store.getAgent!(swarmId, agents[0].id);
            expect(updatedAgent?.status).toBe("busy");
            
            // Unregister agent
            await store.unregisterAgent!(swarmId, agents[1].id);
            
            const remainingAgents = await store.listAgents!(swarmId);
            expect(remainingAgents).toHaveLength(1);
        });
    });

    describe("Blackboard Pattern Implementation", () => {
        it("should implement shared memory without communication logic", async () => {
            const swarmId = generatePK();
            const blackboardItems: BlackboardItem[] = [
                {
                    id: generatePK(),
                    type: "goal",
                    content: "Build recommendation system",
                    authorId: generatePK(),
                    timestamp: new Date(),
                    priority: "high",
                },
                {
                    id: generatePK(),
                    type: "constraint",
                    content: "Must use existing database schema",
                    authorId: generatePK(),
                    timestamp: new Date(),
                    priority: "medium",
                },
                {
                    id: generatePK(),
                    type: "insight",
                    content: "User preference patterns show strong seasonality",
                    authorId: generatePK(),
                    timestamp: new Date(),
                    priority: "low",
                },
            ];

            // Add items to blackboard
            for (const item of blackboardItems) {
                await store.addBlackboardItem!(swarmId, item);
            }
            
            // Get all items
            const allItems = await store.getBlackboardItems!(swarmId);
            expect(allItems).toHaveLength(3);
            
            // Filter by type
            const goalItems = await store.getBlackboardItems!(swarmId, 
                item => item.type === "goal",
            );
            expect(goalItems).toHaveLength(1);
            expect(goalItems[0].content).toBe("Build recommendation system");
            
            // Update item
            const insightItem = blackboardItems[2];
            await store.updateBlackboardItem!(swarmId, insightItem.id, {
                priority: "high",
                content: "Updated: User preferences show strong seasonal patterns",
            });
            
            const updatedItems = await store.getBlackboardItems!(swarmId);
            const updated = updatedItems.find(item => item.id === insightItem.id);
            expect(updated?.priority).toBe("high");
            expect(updated?.content).toContain("Updated:");
            
            // Remove item
            await store.removeBlackboardItem!(swarmId, blackboardItems[1].id);
            
            const remainingItems = await store.getBlackboardItems!(swarmId);
            expect(remainingItems).toHaveLength(2);
        });
    });

    describe("Resource Allocation Infrastructure", () => {
        it("should track resource allocation without optimization", async () => {
            const swarmId = generatePK();
            const resources: SwarmResource[] = [
                {
                    id: "gpu-cluster-1",
                    type: "compute",
                    capacity: 8,
                    metadata: { location: "us-east", gpuType: "A100" },
                },
                {
                    id: "database-primary",
                    type: "storage",
                    capacity: 1,
                    metadata: { readonly: false, sharded: true },
                },
            ];
            
            const consumers = [generatePK(), generatePK(), generatePK()];

            // Allocate resources
            await store.allocateResource!(swarmId, resources[0], consumers[0]);
            await store.allocateResource!(swarmId, resources[0], consumers[1]);
            await store.allocateResource!(swarmId, resources[1], consumers[2]);
            
            // Check allocations
            const gpuConsumers = await store.getResourceAllocation!(swarmId, "gpu-cluster-1");
            expect(gpuConsumers).toHaveLength(2);
            expect(gpuConsumers).toContain(consumers[0]);
            expect(gpuConsumers).toContain(consumers[1]);
            
            const dbConsumers = await store.getResourceAllocation!(swarmId, "database-primary");
            expect(dbConsumers).toHaveLength(1);
            expect(dbConsumers).toContain(consumers[2]);
            
            // Release resource
            await store.releaseResource!(swarmId, "gpu-cluster-1", consumers[0]);
            
            const remainingGpuConsumers = await store.getResourceAllocation!(swarmId, "gpu-cluster-1");
            expect(remainingGpuConsumers).toHaveLength(1);
            expect(remainingGpuConsumers).toContain(consumers[1]);
            
            // Resource allocation without optimization logic
        });
    });

    describe("Query Operations for Analysis", () => {
        it("should provide query interface for agent analysis", async () => {
            const swarms = [
                {
                    id: generatePK(),
                    name: "Active Swarm 1",
                    state: ExecutionStates.RUNNING,
                    userId: "user-1",
                },
                {
                    id: generatePK(),
                    name: "Completed Swarm",
                    state: ExecutionStates.STOPPED,
                    userId: "user-1",
                },
                {
                    id: generatePK(),
                    name: "Active Swarm 2",
                    state: ExecutionStates.RUNNING,
                    userId: "user-2",
                },
                {
                    id: generatePK(),
                    name: "Paused Swarm",
                    state: ExecutionStates.PAUSED,
                    userId: "user-1",
                },
            ];

            // Create test swarms
            for (const swarmData of swarms) {
                const swarm: Swarm = {
                    ...swarmData,
                    description: "Test swarm",
                    goal: "Test goal",
                    config: {},
                    createdAt: new Date(),
                    updatedAt: new Date(),
                };
                await store.createSwarm(swarm.id, swarm);
            }
            
            // Query by state
            const runningSwarms = await store.getSwarmsByState(ExecutionStates.RUNNING);
            expect(runningSwarms).toHaveLength(2);
            
            const completedSwarms = await store.getSwarmsByState(ExecutionStates.STOPPED);
            expect(completedSwarms).toHaveLength(1);
            
            // Query by user
            const user1Swarms = await store.getSwarmsByUser("user-1");
            expect(user1Swarms).toHaveLength(3);
            
            const user2Swarms = await store.getSwarmsByUser("user-2");
            expect(user2Swarms).toHaveLength(1);
            
            // List active swarms
            const activeSwarms = await store.listActiveSwarms();
            expect(activeSwarms.length).toBeGreaterThanOrEqual(2); // RUNNING + PAUSED
            
            // Query interface enables agent analysis without built-in analytics
        });
    });

    describe("Distributed State Consistency", () => {
        it("should handle concurrent state updates", async () => {
            const swarmId = generatePK();
            const swarm: Swarm = {
                id: swarmId,
                name: "Concurrent Test Swarm",
                description: "Testing concurrent updates",
                goal: "Handle concurrency",
                state: ExecutionStates.UNINITIALIZED,
                userId: generatePK(),
                config: { counter: 0 },
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await redisStore.createSwarm(swarmId, swarm);
            
            // Simulate concurrent updates
            const updatePromises = Array.from({ length: 10 }, (_, i) => 
                redisStore.updateSwarm(swarmId, {
                    config: { counter: i, timestamp: Date.now() },
                }),
            );
            
            await Promise.all(updatePromises);
            
            // State should be consistent (last update wins)
            const finalSwarm = await redisStore.getSwarm(swarmId);
            expect(finalSwarm?.config).toHaveProperty("counter");
            expect(finalSwarm?.config).toHaveProperty("timestamp");
        });

        it("should handle store failures gracefully", async () => {
            const swarmId = generatePK();
            
            // Attempt to get non-existent swarm
            const nonExistent = await store.getSwarm("non-existent-id");
            expect(nonExistent).toBeNull();
            
            // Attempt to update non-existent swarm
            await store.updateSwarm("non-existent-id", { name: "New Name" });
            // Should not throw - resilient infrastructure
            
            // Store provides infrastructure, resilience strategies emerge
        });
    });
});
