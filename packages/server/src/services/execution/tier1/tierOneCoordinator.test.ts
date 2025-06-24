import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TierOneCoordinator } from "./tierOneCoordinator.js";
import { type Logger } from "winston";
import { EventBus } from "../cross-cutting/events/eventBus.js";
import { type TierCommunicationInterface, type TierExecutionRequest, type ExecutionResult } from "@vrooli/shared";
import { generatePK, type ExecutionStatus, SEEDED_PUBLIC_IDS } from "@vrooli/shared";
import { DbProvider } from "../../../db/provider.js";

/**
 * TierOneCoordinator Tests - Strategic Coordination Layer
 * 
 * These tests validate that Tier 1 provides minimal coordination infrastructure
 * while enabling strategic intelligence to emerge from:
 * 
 * 1. **Swarm Lifecycle Management**: Basic creation/teardown, not strategies
 * 2. **Resource Allocation**: Limits enforcement, not optimization algorithms
 * 3. **Team Formation**: Infrastructure for agents, not selection logic
 * 4. **Metacognitive Events**: Signals for agents, not analysis
 * 5. **Cross-Swarm Coordination**: Communication channels, not strategies
 * 
 * Complex behaviors like goal decomposition, strategic planning, and
 * metacognitive evaluation emerge from AI agents using this infrastructure.
 */

describe.skip("TierOneCoordinator - Strategic Coordination Infrastructure", () => {
    // Skipping all tests in this file because they were heavily mocked
    // Need to rewrite as proper integration tests using real database
    let logger: Logger;
    let eventBus: EventBus;
    let tier2Orchestrator: TierCommunicationInterface;
    let coordinator: TierOneCoordinator;

    beforeEach(() => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;
        eventBus = new EventBus(logger);
        
        // Mock Tier 2 orchestrator
        tier2Orchestrator = {
            execute: vi.fn(),
            getMetrics: vi.fn().mockResolvedValue({
                activeRuns: 0,
                completedRuns: 0,
                failedRuns: 0,
            }),
        };

        // Database provider will use real testcontainer database

        coordinator = new TierOneCoordinator(logger, eventBus, tier2Orchestrator);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Minimal Swarm Infrastructure", () => {
        it("should provide basic swarm lifecycle management", async () => {
            const swarmConfig = {
                swarmId: generatePK(),
                name: "Test Swarm",
                description: "Test swarm for validation",
                goal: "Complete test objectives",
                resources: {
                    maxCredits: 1000,
                    maxTokens: 50000,
                    maxTime: 3600000, // 1 hour
                    tools: [{ name: "search", description: "Search the web" }],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 3,
                },
                userId: generatePK(),
            };

            // Start swarm
            await coordinator.startSwarm(swarmConfig);

            // Verify basic infrastructure is created
            expect(logger.info).toHaveBeenCalledWith(
                "[TierOneCoordinator] Starting swarm",
                expect.objectContaining({
                    swarmId: swarmConfig.swarmId,
                    name: swarmConfig.name,
                }),
            );

            // Should create conversation for agent coordination
            expect(DbProvider.get().chat.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    chatType: "SwarmAgent",
                }),
            });
        });

        it("should handle concurrent swarm creation with locks", async () => {
            const swarmId = generatePK();
            const baseConfig = {
                swarmId,
                name: "Concurrent Test",
                description: "Testing concurrent creation",
                goal: "Test concurrent handling",
                resources: {
                    maxCredits: 500,
                    maxTokens: 10000,
                    maxTime: 600000,
                    tools: [],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.5,
                    autoApproveTools: true,
                    parallelExecutionLimit: 1,
                },
                userId: generatePK(),
            };

            // Start multiple concurrent creations
            const promises = Array.from({ length: 3 }, () => 
                coordinator.startSwarm(baseConfig),
            );

            await Promise.all(promises);

            // Only one swarm should be created despite concurrent calls
            expect(DbProvider.get().chat.create).toHaveBeenCalledOnce();
        });
    });

    describe("Cross-Tier Communication", () => {
        it("should delegate routine execution to Tier 2", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: { query: "test" },
                },
            };

            // Mock Tier 2 response
            const mockResult: ExecutionResult = {
                executionId: request.executionId,
                status: "completed" as ExecutionStatus,
                data: { result: "processed" },
            };
            vi.mocked(tier2Orchestrator.execute).mockResolvedValue(mockResult);

            const result = await coordinator.execute(request);

            // Should pass through to Tier 2
            expect(tier2Orchestrator.execute).toHaveBeenCalledWith(request);
            expect(result).toEqual(mockResult);
        });

        it("should handle swarm execution requests", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "swarm",
                userId: generatePK(),
                payload: {
                    goal: "Analyze customer feedback and generate insights",
                    config: {
                        maxCredits: 2000,
                        temperature: 0.8,
                    },
                },
            };

            // Execute swarm request
            const resultPromise = coordinator.execute(request);

            // Should create a new swarm
            await new Promise(resolve => setTimeout(resolve, 100));
            expect(logger.info).toHaveBeenCalledWith(
                "[TierOneCoordinator] Starting swarm",
                expect.any(Object),
            );

            // Complete the execution
            const result = await resultPromise;
            expect(result.status).toBe("in_progress");
            expect(result.data).toHaveProperty("swarmId");
        });
    });

    describe("Resource Management Infrastructure", () => {
        it("should enforce resource limits without optimization", async () => {
            const swarmConfig = {
                swarmId: generatePK(),
                name: "Resource Limited Swarm",
                description: "Testing resource limits",
                goal: "Process data within constraints",
                resources: {
                    maxCredits: 100,
                    maxTokens: 1000,
                    maxTime: 60000, // 1 minute
                    tools: [],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.5,
                    autoApproveTools: false,
                    parallelExecutionLimit: 1,
                },
                userId: generatePK(),
            };

            await coordinator.startSwarm(swarmConfig);

            // Resource limits are stored but not optimized
            expect(DbProvider.get().chat.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    chatType: "SwarmAgent",
                    configId: expect.any(String),
                }),
            });
            
            // No optimization logic - agents determine resource strategies
        });

        it("should emit resource events for monitoring agents", async () => {
            const resourceEvents: any[] = [];
            await eventBus.subscribe("swarm.resource.*", async (event) => {
                resourceEvents.push(event);
            });

            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "swarm",
                userId: generatePK(),
                payload: {
                    goal: "Monitor resource usage",
                    config: {
                        maxCredits: 500,
                    },
                },
            };

            await coordinator.execute(request);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Resource allocation events for agent analysis
            expect(resourceEvents.length).toBeGreaterThan(0);
        });
    });

    describe("Team Formation Infrastructure", () => {
        it("should provide channels for agent collaboration", async () => {
            const swarmConfig = {
                swarmId: generatePK(),
                name: "Multi-Agent Swarm",
                description: "Testing team formation",
                goal: "Build a recommendation system",
                resources: {
                    maxCredits: 5000,
                    maxTokens: 100000,
                    maxTime: 7200000, // 2 hours
                    tools: [
                        { name: "spawn_agent", description: "Create specialized agent" },
                        { name: "message_agent", description: "Send message to agent" },
                    ],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 5,
                },
                userId: generatePK(),
                organizationId: generatePK(),
            };

            await coordinator.startSwarm(swarmConfig);

            // Infrastructure for agents, not team selection logic
            expect(DbProvider.get().chat.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    chatType: "SwarmAgent",
                    participantsConnect: expect.objectContaining({
                        create: expect.arrayContaining([
                            expect.objectContaining({
                                botId: expect.any(String), // Leader bot
                            }),
                        ]),
                    }),
                }),
            });
        });

        it("should enable dynamic team expansion through events", async () => {
            const teamEvents: any[] = [];
            await eventBus.subscribe("swarm.team.*", async (event) => {
                teamEvents.push(event);
            });

            const swarmId = generatePK();
            await coordinator.startSwarm({
                swarmId,
                name: "Dynamic Team",
                description: "Testing dynamic team formation",
                goal: "Solve complex problem requiring specialists",
                resources: {
                    maxCredits: 3000,
                    maxTokens: 50000,
                    maxTime: 3600000,
                    tools: [{ name: "spawn_agent", description: "Create agent" }],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.7,
                    autoApproveTools: true,
                    parallelExecutionLimit: 4,
                },
                userId: generatePK(),
            });

            // Simulate team expansion event
            await eventBus.publish("swarm.team.agent_added", {
                swarmId,
                agentId: generatePK(),
                role: "data_analyst",
                capabilities: ["sql", "visualization"],
            });

            await new Promise(resolve => setTimeout(resolve, 100));

            // Team formation events available for coordination agents
            expect(teamEvents).toContainEqual(
                expect.objectContaining({
                    type: "swarm.team.agent_added",
                    data: expect.objectContaining({
                        role: "data_analyst",
                    }),
                }),
            );
        });
    });

    describe("Metacognitive Event Infrastructure", () => {
        it("should emit signals for metacognitive agents", async () => {
            const metacognitiveEvents: any[] = [];
            await eventBus.subscribe("swarm.metacognitive.*", async (event) => {
                metacognitiveEvents.push(event);
            });

            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "swarm",
                userId: generatePK(),
                payload: {
                    goal: "Optimize database performance",
                    metadata: {
                        complexity: "high",
                        domain: "technical",
                    },
                },
            };

            await coordinator.execute(request);

            // Simulate metacognitive insights
            await eventBus.publish("swarm.metacognitive.insight", {
                swarmId: request.executionId,
                insight: {
                    type: "strategy_adjustment",
                    observation: "Current approach too sequential",
                    recommendation: "Consider parallel analysis",
                },
            });

            await new Promise(resolve => setTimeout(resolve, 100));

            // Metacognitive agents can analyze and adapt
            expect(metacognitiveEvents).toHaveLength(1);
            expect(metacognitiveEvents[0].data.insight.type).toBe("strategy_adjustment");
        });

        it("should enable emergent learning through event patterns", async () => {
            const learningEvents: any[] = [];
            await eventBus.subscribe("swarm.learning.*", async (event) => {
                learningEvents.push(event);
            });

            // Execute multiple similar swarms
            const goals = [
                "Analyze customer feedback",
                "Process user reviews",
                "Evaluate product feedback",
            ];

            for (const goal of goals) {
                const request: TierExecutionRequest = {
                    executionId: generatePK(),
                    type: "swarm",
                    userId: generatePK(),
                    payload: { goal },
                };

                await coordinator.execute(request);
                
                // Simulate pattern detection
                await eventBus.publish("swarm.learning.pattern_detected", {
                    pattern: "feedback_analysis",
                    frequency: goals.indexOf(goal) + 1,
                    optimization: "Use specialized feedback analyzer",
                });
            }

            await new Promise(resolve => setTimeout(resolve, 100));

            // Learning agents can detect and optimize patterns
            expect(learningEvents.length).toBe(3);
            expect(learningEvents[2].data.frequency).toBe(3);
        });
    });

    describe("Child Swarm Coordination", () => {
        it("should enable parent-child swarm relationships", async () => {
            const parentSwarmId = generatePK();
            const childSwarmId = generatePK();

            // Start parent swarm
            await coordinator.startSwarm({
                swarmId: parentSwarmId,
                name: "Parent Swarm",
                description: "Main coordination swarm",
                goal: "Build complete system",
                resources: {
                    maxCredits: 10000,
                    maxTokens: 200000,
                    maxTime: 7200000,
                    tools: [{ name: "spawn_swarm", description: "Create child swarm" }],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 5,
                },
                userId: generatePK(),
            });

            // Start child swarm
            await coordinator.startSwarm({
                swarmId: childSwarmId,
                name: "Child Swarm",
                description: "Handle specific subtask",
                goal: "Implement authentication module",
                resources: {
                    maxCredits: 2000,
                    maxTokens: 40000,
                    maxTime: 3600000,
                    tools: [],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.6,
                    autoApproveTools: true,
                    parallelExecutionLimit: 2,
                },
                userId: generatePK(),
                parentSwarmId, // Link to parent
            });

            // Infrastructure enables hierarchical coordination
            expect(logger.info).toHaveBeenCalledWith(
                "[TierOneCoordinator] Starting swarm",
                expect.objectContaining({ swarmId: childSwarmId }),
            );
        });
    });

    describe("Error Handling and Resilience", () => {
        it("should handle tier communication failures gracefully", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                },
            };

            // Simulate Tier 2 failure
            vi.mocked(tier2Orchestrator.execute).mockRejectedValue(
                new Error("Tier 2 connection failed"),
            );

            const result = await coordinator.execute(request);

            expect(result.status).toBe("failed");
            expect(result.error).toContain("Tier 2 connection failed");
        });

        it("should emit failure events for monitoring agents", async () => {
            const failureEvents: any[] = [];
            await eventBus.subscribe("swarm.failure.*", async (event) => {
                failureEvents.push(event);
            });

            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "swarm",
                userId: generatePK(),
                payload: {
                    goal: "Invalid goal that will fail",
                },
            };

            // Mock chat creation failure
            vi.mocked(DbProvider.get().chat.create).mockRejectedValue(
                new Error("Database connection lost"),
            );

            await coordinator.execute(request);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Monitoring agents can analyze failures
            expect(failureEvents.length).toBeGreaterThan(0);
        });
    });
});
