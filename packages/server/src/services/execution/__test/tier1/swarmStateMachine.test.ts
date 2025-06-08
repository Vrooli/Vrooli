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
    let teamManagerStub: any;
    let resourceManagerStub: any;
    let strategyEngineStub: any;
    let metacognitiveMonitorStub: any;

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

        // Create mock instances
        teamManagerStub = {
            formTeam: vi.fn(),
            updateTeam: vi.fn(),
            disbandTeam: vi.fn(),
        };
        
        resourceManagerStub = {
            allocateInitialResources: vi.fn(),
            updateResourceUsage: vi.fn(),
            releaseResources: vi.fn(),
        };
        
        strategyEngineStub = {
            selectStrategy: vi.fn(),
            evaluateStrategy: vi.fn(),
        };
        
        metacognitiveMonitorStub = {
            startMonitoring: vi.fn(),
            stopMonitoring: vi.fn(),
            getInsights: vi.fn(),
        };

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
            vi.mocked(stateStoreStub.getSwarm).mockResolvedValue(mockSwarm);
            vi.mocked(stateStoreStub.updateSwarmState).mockResolvedValue(undefined);

            await stateMachine.start("swarm-123");

            expect(stateStoreStub.getSwarm).toHaveBeenCalledWith("swarm-123");
            expect(stateStoreStub.updateSwarmState).toHaveBeenCalledWith(
                "swarm-123",
                SwarmState.INITIALIZING
            );
        });

        it("should emit state transition event", async () => {
            vi.mocked(stateStoreStub.getSwarm).mockResolvedValue(mockSwarm);
            vi.mocked(stateStoreStub.updateSwarmState).mockResolvedValue();

            const eventSpy = vi.spyOn(eventBus, "publish");

            await stateMachine.start("swarm-123");

            expect(eventSpy).toHaveBeenCalledWith("swarm.state_transition", {
                swarmId: "swarm-123",
                from: SwarmState.UNINITIALIZED,
                to: SwarmState.INITIALIZING,
            });
        });

        it("should throw error if swarm not found", async () => {
            vi.mocked(stateStoreStub.getSwarm).mockResolvedValue(null);

            await expect(stateMachine.start("swarm-123")).rejects.toThrow("Swarm swarm-123 not found");
        });

        it("should throw error if swarm already started", async () => {
            const activeSwarm = { ...mockSwarm, state: SwarmState.ACTIVE };
            vi.mocked(stateStoreStub.getSwarm).mockResolvedValue(activeSwarm);

            await expect(stateMachine.start("swarm-123")).rejects.toThrow("Swarm swarm-123 is already started");
        });
    });

    describe("state transitions", () => {
        beforeEach(async () => {
            vi.mocked(stateStoreStub.getSwarm).mockResolvedValue(mockSwarm);
            vi.mocked(stateStoreStub.updateSwarmState).mockResolvedValue();
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

            // Continue with initialization
            await stateMachine.initialize("swarm-123");

            // Verify the expected method calls
            expect(strategyEngineStub.selectStrategy).toHaveBeenCalled();
            expect(resourceManagerStub.allocateInitialResources).toHaveBeenCalled();
            expect(teamManagerStub.formTeam).toHaveBeenCalled();
        });
    });
});