/* eslint-disable no-magic-numbers */
/**
 * Scenario Factory
 * 
 * Orchestrates full multi-agent test scenarios
 */

import type { BotParticipant, SwarmId } from "@vrooli/shared";
import { RunState } from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { EventInterceptor } from "../../../../services/events/EventInterceptor.js";
import { InMemoryLockService } from "../../../../services/events/LockService.js";
import type { ISwarmContextManager } from "../../../../services/execution/shared/SwarmContextManager.js";
import { SwarmContextManager } from "../../../../services/execution/shared/SwarmContextManager.js";
import { StepExecutor } from "../../../../services/execution/tier3/stepExecutor.js";
import { AgentFactory } from "../agent/AgentFactory.js";
import { RoutineFactory } from "../routine/RoutineFactory.js";
import { SwarmFactory } from "../swarm/SwarmFactory.js";
import { MockController } from "./MockController.js";
import type { ScenarioContext, ScenarioDefinition } from "./types.js";

export class ScenarioFactory {
    private routineFactory: RoutineFactory;
    private agentFactory: AgentFactory;
    private swarmFactory: SwarmFactory;
    private mockController: MockController;
    private eventInterceptor: EventInterceptor;
    private contextManager: ISwarmContextManager;
    private stepExecutor: StepExecutor;
    private lockService: InMemoryLockService;

    constructor() {
        this.routineFactory = new RoutineFactory();
        this.agentFactory = new AgentFactory();
        this.swarmFactory = new SwarmFactory();
        this.mockController = new MockController();

        // Initialize execution services
        this.lockService = new InMemoryLockService();

        // Create mock config for SwarmContextManager
        const mockConfig = {
            chatId: "test-scenario-chat",
            userId: "test-user",
            taskId: "test-task",
            tier: "tier1",
            requestId: "test-request",
            priority: "normal",
            timeout: 300000,
            parentRequestId: null,
            allocation: {
                maxCredits: "1000",
                maxDurationMs: 300000,
                maxMemoryMB: 512,
                maxConcurrentSteps: 5,
            },
        };
        this.contextManager = new SwarmContextManager(mockConfig);

        this.stepExecutor = new StepExecutor(); // StepExecutor takes optional SwarmTools, not contextManager
        // Mock RoutineExecutor factory for tests
        const routineExecutorFactory = async () => ({
            start: async () => undefined,
            stop: async () => ({ success: true }),
            getState: () => "READY" as any,
        }) as any;

        this.eventInterceptor = new EventInterceptor(
            this.lockService,
            this.contextManager,
            routineExecutorFactory,
        );
    }

    async setupScenario(definition: ScenarioDefinition): Promise<ScenarioContext> {
        // 1. Create all routines
        const routines = await this.routineFactory.createBatch(
            definition.schemas.routines,
            { saveToDb: true, mockResponses: true },
        );

        // 2. Create all agents
        const agents = await this.agentFactory.createBatch(
            definition.schemas.agents,
            { saveToDb: true, mockBehaviors: true },
        );

        // 3. Create swarms/teams
        const swarms = await this.swarmFactory.createBatch(
            definition.schemas.swarms,
            { saveToDb: true, mockState: true },
        );

        // 4. Configure mocks
        await this.mockController.configure(definition.mockConfig);

        // 5. Convert agents to BotParticipant format
        const botParticipants: any[] = [];
        for (const agent of agents) {
            try {
                const botParticipant = this.convertAgentToBotParticipant(agent);
                botParticipants.push(botParticipant);

                // Register directly with EventInterceptor for test execution
                this.eventInterceptor.registerBot(botParticipant);
                logger.info("[ScenarioFactory] Registered test agent with EventInterceptor", {
                    agentId: agent.id,
                    agentName: agent.name,
                    subscriptions: agent.subscriptions,
                });
            } catch (error) {
                logger.error("[ScenarioFactory] Failed to register test agent", {
                    agentId: agent.id,
                    agentName: agent.name,
                    error: error instanceof Error ? error.message : String(error),
                });
                throw error;
            }
        }

        // 6. Initialize blackboard
        const blackboard = await this.initializeBlackboard(
            definition.expectations.initialBlackboard,
        );

        // 7. Create initial SwarmState with agents already populated
        const initialSwarmState = {
            swarmId: definition.name,
            version: 1,
            chatConfig: {
                __version: "1.0",
                goal: definition.description,
                blackboard: [],
                stats: {
                    totalToolCalls: 0,
                    totalCredits: "0",
                    startedAt: null,
                    lastProcessingCycleEndedAt: null,
                },
            },
            execution: {
                status: RunState.READY,
                agents: botParticipants, // Add the converted BotParticipants here
                activeRuns: [],
                startedAt: new Date(),
                lastActivityAt: new Date(),
            },
            resources: {
                allocated: [],
                consumed: { credits: 0, tokens: 0, time: 0 },
                remaining: { credits: 1000, tokens: 10000, time: 3600 },
            },
            metadata: {
                createdAt: new Date(),
                lastUpdated: new Date(),
                updatedBy: "test-scenario",
                subscribers: new Set<string>(),
            },
        };

        // Create context with agents already in it
        await this.contextManager.createContext(definition.name, initialSwarmState);

        return {
            name: definition.name,
            description: definition.description,
            routines,
            agents: botParticipants, // Return BotParticipants instead of raw agents
            swarms,
            blackboard,
            mockController: this.mockController,
            expectations: definition.expectations,
            events: [],
            routineCalls: [],
            startTime: new Date(),
            // Add execution services to context
            eventInterceptor: this.eventInterceptor,
            contextManager: this.contextManager,
            stepExecutor: this.stepExecutor,
        };
    }

    async teardownScenario(context: ScenarioContext): Promise<void> {
        // Clean up mocks
        this.mockController.clear();

        // Delete the swarm context from Redis
        try {
            await this.contextManager.deleteContext(context.name as SwarmId);
            logger.info("[ScenarioFactory] Deleted swarm context", {
                swarmId: context.name,
            });
        } catch (error) {
            logger.error("[ScenarioFactory] Failed to delete swarm context", {
                swarmId: context.name,
                error: error instanceof Error ? error.message : String(error),
            });
        }

        // Unregister all test agents from EventInterceptor
        if (context.agents && Array.isArray(context.agents)) {
            for (const agent of context.agents) {
                try {
                    this.eventInterceptor.unregisterBot(agent.id);
                    logger.info("[ScenarioFactory] Unregistered test agent from EventInterceptor", {
                        agentId: agent.id,
                        agentName: agent.name,
                    });
                } catch (error) {
                    logger.error("[ScenarioFactory] Failed to unregister test agent", {
                        agentId: agent.id,
                        error: error instanceof Error ? error.message : String(error),
                    });
                }
            }
        }

        // Clean up test data if needed
        // Note: In test environments, database cleanup is usually handled by test framework
    }

    private async initializeBlackboard(
        initialState?: Record<string, any>,
    ): Promise<Map<string, any>> {
        const blackboard = new Map<string, any>();

        if (initialState) {
            for (const [key, value] of Object.entries(initialState)) {
                blackboard.set(key, value);
            }
        }

        return blackboard;
    }

    /**
     * Convert schema-defined agent to BotParticipant for EventInterceptor registration
     */
    private convertAgentToBotParticipant(agent: any): BotParticipant {
        if (!agent.behaviors || !Array.isArray(agent.behaviors)) {
            logger.warn("[ScenarioFactory] Agent has no behaviors, creating empty behavior list", {
                agentId: agent.id,
                agentName: agent.name,
            });
        }

        return {
            id: agent.id,
            name: agent.name,
            config: {
                agentSpec: {
                    goal: agent.goal,
                    role: agent.role || "specialist", // Default role
                    subscriptions: agent.subscriptions || [],
                    behaviors: (agent.behaviors || []).map((behavior: any) => ({
                        trigger: {
                            topic: behavior.trigger.topic,
                            when: behavior.trigger.when,
                            progression: behavior.trigger.progression,
                        },
                        action: {
                            type: behavior.action.type,
                            routine: behavior.action.routine || behavior.action.label,
                            input: behavior.action.input,
                            outputOperations: behavior.action.outputOperations,
                            topic: behavior.action.topic, // For emit actions
                        },
                    })),
                    resources: [],
                },
                model: agent.resources?.preferredModel || "gpt-4",
                maxTokens: agent.resources?.maxTokens || 4096,
            },
            state: "ready",
            role: agent.role || "specialist",
            priority: agent.priority || 1,
        };
    }
}
