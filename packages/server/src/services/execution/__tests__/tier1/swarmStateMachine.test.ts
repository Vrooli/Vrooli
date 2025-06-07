import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { SwarmStateMachine } from "../../tier1/coordination/swarmStateMachine.js";
import { MockEventBus } from "../mocks/eventBus.js";
import { type ISwarmStateStore } from "../../tier1/state/swarmStateStore.js";
import { TeamManager } from "../../tier1/organization/teamManager.js";
import { ResourceManager } from "../../tier1/organization/resourceManager.js";
import { StrategyEngine } from "../../tier1/intelligence/strategyEngine.js";
import { MetacognitiveMonitor } from "../../tier1/intelligence/metacognitiveMonitor.js";
import { SwarmState, type Swarm } from "@vrooli/shared";

describe("SwarmStateMachine", () => {
    let stateMachine: SwarmStateMachine;
    let logger: winston.Logger;
    let eventBus: MockEventBus;
    let stateStoreStub: ISwarmStateStore;
    let teamManagerStub: TeamManager;
    let resourceManagerStub: ResourceManager;
    let strategyEngineStub: StrategyEngine;
    let metacognitiveMonitorStub: MetacognitiveMonitor;

    const mockSwarm: Swarm = {
        id: "swarm-123",
        state: SwarmState.UNINITIALIZED,
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
                credits: 0,
                tokens: 0,
                time: 0,
            },
            remaining: {
                maxCredits: 10000,
                maxTokens: 100000,
                maxTime: 3600000,
            },
        },
        metrics: {
            tasksCompleted: 0,
            tasksFailed: 0,
            avgTaskDuration: 0,
            resourceEfficiency: 0,
        },
        createdAt: new Date(),
        updatedAt: new Date(),
        metadata: {
            userId: "user-123",
            version: "2.0.0",
        },
    };

    beforeEach(() => {
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        eventBus = new MockEventBus();

        // Create mocks
        stateStoreStub = {
            createSwarm: vi.fn(),
            getSwarm: vi.fn(),
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

        teamManagerStub = vi.mocked(new TeamManager(logger, eventBus, stateStoreStub as any));
        resourceManagerStub = vi.mocked(new ResourceManager(logger, eventBus, stateStoreStub as any));
        strategyEngineStub = vi.mocked(new StrategyEngine(logger, eventBus));
        metacognitiveMonitorStub = vi.mocked(new MetacognitiveMonitor(logger, eventBus));

        stateMachine = new SwarmStateMachine(
            logger,
            eventBus,
            stateStoreStub,
            teamManagerStub,
            resourceManagerStub,
            strategyEngineStub,
            metacognitiveMonitorStub,
        );
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("start", () => {
        it("should transition from UNINITIALIZED to INITIALIZING", async () => {
            stateStoreStub.getSwarm.mockResolvedValue(mockSwarm);
            stateStoreStub.updateSwarmState.mockResolvedValue(undefined);

            await stateMachine.start("swarm-123");

            expect(stateStoreStub.getSwarm).toHaveBeenCalledWith("swarm-123");
            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.INITIALIZING
            );
        });

        it("should emit state transition event", async () => {
            stateStoreStub.getSwarm.mockResolvedValue(mockSwarm);
            stateStoreStub.updateSwarmState.mockResolvedValue();

            const eventSpy = vi.spyOn(eventBus, "publish");

            await stateMachine.start("swarm-123");

            expect(eventSpy).toHaveBeenCalledWith("swarm.state_transition", {
                swarmId: "swarm-123",
                from: SwarmState.UNINITIALIZED,
                to: SwarmState.INITIALIZING,
            });
        });

        it("should throw error if swarm not found", async () => {
            stateStoreStub.getSwarm.mockResolvedValue(null);

            try {
                await stateMachine.start("swarm-123");
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("Swarm swarm-123 not found");
            }
        });

        it("should throw error if swarm already started", async () => {
            const activeSwarm = { ...mockSwarm, state: SwarmState.ACTIVE };
            stateStoreStub.getSwarm.mockResolvedValue(activeSwarm);

            try {
                await stateMachine.start("swarm-123");
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("Swarm swarm-123 is already started");
            }
        });
    });

    describe("state transitions", () => {
        beforeEach(async () => {
            stateStoreStub.getSwarm.mockResolvedValue(mockSwarm);
            stateStoreStub.updateSwarmState.mockResolvedValue();
            await stateMachine.start("swarm-123");
        });

        it("should transition through initialization phases", async () => {
            // Mock successful strategy selection
            strategyEngineStub.selectStrategy.mockResolvedValue({
                strategy: "exploration",
                confidence: 0.8,
                reasoning: "Goal requires exploration",
            });

            // Mock successful resource allocation
            resourceManagerStub.allocateInitialResources.mockResolvedValue({
                credits: 1000,
                tokens: 10000,
                time: 3600000,
                tools: [],
            });

            // Mock successful team formation
            teamManagerStub.formTeam.mockResolvedValue({
                agents: [
                    {
                        id: "agent-1",
                        name: "Explorer",
                        role: "explorer",
                        capabilities: ["search", "analyze"],
                        status: "active",
                    },
                ],
                capabilities: ["search", "analyze"],
                activeMembers: 1,
            });

            // Progress through states
            await stateMachine.transitionTo(SwarmState.STRATEGIZING);
            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.STRATEGIZING
            );

            await stateMachine.transitionTo(SwarmState.RESOURCE_ALLOCATION);
            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.RESOURCE_ALLOCATION
            );

            await stateMachine.transitionTo(SwarmState.TEAM_FORMING);
            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.TEAM_FORMING
            );

            await stateMachine.transitionTo(SwarmState.READY);
            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.READY
            );
        });

        it("should handle invalid state transitions", async () => {
            // Try to jump directly from INITIALIZING to ACTIVE
            try {
                await stateMachine.transitionTo(SwarmState.ACTIVE);
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toMatch(/Invalid state transition/);
            }
        });
    });

    describe("run execution", () => {
        beforeEach(async () => {
            const activeSwarm = { ...mockSwarm, state: SwarmState.ACTIVE };
            stateStoreStub.getSwarm.mockResolvedValue(activeSwarm);
            stateStoreStub.updateSwarmState.mockResolvedValue();
        });

        it("should request run execution when in ACTIVE state", async () => {
            const eventSpy = vi.spyOn(eventBus, "publish");

            const runRequest = {
                swarmId: "swarm-123",
                runId: "run-123",
                routineVersionId: "routine-v1",
                inputs: { test: "input" },
                config: {
                    strategy: "reasoning",
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
            };

            await stateMachine.requestRunExecution(runRequest);

            expect(eventSpy).toHaveBeenCalledWith("swarm.run_requested", runRequest);
        });

        it("should track run completion", async () => {
            stateStoreStub.updateSwarm.mockResolvedValue();

            await stateMachine.handleRunCompletion("run-123");

            expect(stateStoreStub.updateSwarm).toHaveBeenCalledWith(
                "swarm-123",
                expect.objectContaining({
                    metrics: expect.objectContaining({
                        tasksCompleted: 1,
                    }),
                })
            );
        });
    });

    describe("stop", () => {
        beforeEach(async () => {
            const activeSwarm = { ...mockSwarm, state: SwarmState.ACTIVE };
            stateStoreStub.getSwarm.mockResolvedValue(activeSwarm);
            stateStoreStub.updateSwarmState.mockResolvedValue();
        });

        it("should transition to STOPPED state", async () => {
            await stateMachine.stop("User requested stop");

            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.STOPPED
            );
        });

        it("should emit stop event", async () => {
            const eventSpy = vi.spyOn(eventBus, "publish");

            await stateMachine.stop("System shutdown");

            expect(eventSpy).toHaveBeenCalledWith("swarm.stopped", {
                swarmId: "swarm-123",
                reason: "System shutdown",
            });
        });
    });

    describe("metacognitive insights", () => {
        it("should handle adaptation suggestions", async () => {
            const activeSwarm = { ...mockSwarm, state: SwarmState.ACTIVE };
            stateStoreStub.getSwarm.mockResolvedValue(activeSwarm);
            stateStoreStub.updateSwarmState.mockResolvedValue();

            const insight = {
                swarmId: "swarm-123",
                type: "adaptation",
                suggestion: "Switch to deterministic strategy",
                confidence: 0.9,
                reasoning: "Pattern detected in recent executions",
            };

            await stateMachine.handleMetacognitiveInsight(insight);

            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.ADAPTING
            );
        });
    });

    describe("resource alerts", () => {
        it("should handle low resource alerts", async () => {
            const activeSwarm = { ...mockSwarm, state: SwarmState.ACTIVE };
            stateStoreStub.getSwarm.mockResolvedValue(activeSwarm);
            resourceManagerStub.handleResourceAlert.mockResolvedValue({
                action: "optimize",
                adjustments: { reduceParallelism: true },
            });

            const alert = {
                swarmId: "swarm-123",
                resourceType: "credits",
                remaining: 100,
                threshold: 1000,
                severity: "warning" as const,
            };

            await stateMachine.handleResourceAlert(alert);

            expect(resourceManagerStub.handleResourceAlert).toHaveBeenCalledWith(alert);
        });

        it("should stop swarm on critical resource depletion", async () => {
            const activeSwarm = { ...mockSwarm, state: SwarmState.ACTIVE };
            stateStoreStub.getSwarm.mockResolvedValue(activeSwarm);
            stateStoreStub.updateSwarmState.mockResolvedValue();
            resourceManagerStub.handleResourceAlert.mockResolvedValue({
                action: "stop",
                reason: "Credits exhausted",
            });

            const alert = {
                swarmId: "swarm-123",
                resourceType: "credits",
                remaining: 0,
                threshold: 1000,
                severity: "critical" as const,
            };

            await stateMachine.handleResourceAlert(alert);

            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.STOPPED
            );
        });
    });
});