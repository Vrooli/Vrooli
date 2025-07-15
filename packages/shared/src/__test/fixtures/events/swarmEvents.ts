/**
 * Swarm execution event fixtures for testing AI multi-agent systems
 * Enhanced with factory pattern for better type safety and maintainability
 */
import { type SwarmSocketEventPayloads } from "../../../consts/socketEvents.js";
import { type ChatConfigObject } from "../../../shape/configs/chat.js";
import { chatConfigFixtures } from "../config/chatConfigFixtures.js";
import { BaseEventFactory } from "./BaseEventFactory.js";
import { type BaseFixtureEvent } from "./types.js";

// Type definitions for swarm events
type SwarmConfigEvent = BaseFixtureEvent & {
    event: "swarmConfigUpdate";
    data: SwarmSocketEventPayloads["swarmConfigUpdate"];
};

type SwarmStateEvent = BaseFixtureEvent & {
    event: "swarmStateUpdate";
    data: SwarmSocketEventPayloads["swarmStateUpdate"];
};

type SwarmResourceEvent = BaseFixtureEvent & {
    event: "swarmResourceUpdate";
    data: SwarmSocketEventPayloads["swarmResourceUpdate"];
};

type SwarmTeamEvent = BaseFixtureEvent & {
    event: "swarmTeamUpdate";
    data: SwarmSocketEventPayloads["swarmTeamUpdate"];
};

/**
 * Factory for swarm configuration update events
 */
class SwarmConfigEventFactory extends BaseEventFactory<SwarmConfigEvent, SwarmSocketEventPayloads["swarmConfigUpdate"]> {
    constructor() {
        super("swarmConfigUpdate", {
            validation: (data) => {
                if (!data.chatId) return "chatId is required";
                if (!data.config) return "config is required";
                if (data.config.__typename && data.config.__typename !== "ChatConfigObject") {
                    return "Invalid __typename for config";
                }
                return true;
            },
            transform: (data) => ({
                ...data,
                config: {
                    ...data.config,
                    __typename: "ChatConfigObject" as const,
                },
            }),
        });
    }

    single: SwarmConfigEvent = {
        event: "swarmConfigUpdate",
        data: {
            chatId: "chat_minimal",
            config: {
                ...chatConfigFixtures.minimal,
                __typename: "ChatConfigObject" as const,
            },
        },
    };

    sequence: SwarmConfigEvent[] = [
        {
            event: "swarmConfigUpdate",
            data: {
                chatId: "chat_seq_1",
                config: {
                    ...chatConfigFixtures.minimal,
                    __typename: "ChatConfigObject" as const,
                },
            },
        },
        {
            event: "swarmConfigUpdate",
            data: {
                chatId: "chat_seq_1",
                config: {
                    preferredModel: "claude-3-sonnet",
                    __typename: "ChatConfigObject" as const,
                },
            },
        },
        {
            event: "swarmConfigUpdate",
            data: {
                chatId: "chat_seq_1",
                config: {
                    limits: {
                        maxToolCalls: 100,
                        maxCredits: "100000",
                        maxDurationMs: 600000,
                    },
                    __typename: "ChatConfigObject" as const,
                },
            },
        },
    ];

    variants: Record<string, SwarmConfigEvent | SwarmConfigEvent[]> = {
        fullConfig: {
            event: "swarmConfigUpdate" as const,
            data: {
                chatId: "chat_full",
                config: {
                    ...chatConfigFixtures.complete,
                    __typename: "ChatConfigObject" as const,
                },
            },
        },
        modelUpdate: {
            event: "swarmConfigUpdate" as const,
            data: {
                chatId: "chat_model",
                config: {
                    preferredModel: "gpt-4-turbo",
                    scheduling: {
                        defaultDelayMs: 1000,
                        requiresApprovalTools: ["delete", "payment"],
                        approvalTimeoutMs: 300000,
                    },
                    __typename: "ChatConfigObject" as const,
                },
            },
        },
        resourceLimits: {
            event: "swarmConfigUpdate" as const,
            data: {
                chatId: "chat_resources",
                config: {
                    limits: {
                        maxToolCalls: 50,
                        maxCredits: "50000",
                        maxDurationMs: 300000,
                        maxToolCallsPerBotResponse: 5,
                        maxCreditsPerBotResponse: "5000",
                        maxDurationPerBotResponseMs: 30000,
                    },
                    __typename: "ChatConfigObject" as const,
                },
            },
        },
        policyUpdate: {
            event: "swarmConfigUpdate" as const,
            data: {
                chatId: "chat_policy",
                config: {
                    policy: {
                        visibility: "restricted",
                        acl: ["bot_leader", "bot_analyst", "bot_researcher"],
                    },
                    __typename: "ChatConfigObject" as const,
                },
            },
        },
    };

    // Helper method to create progressive configuration updates
    createProgressiveUpdates(chatId: string, updates: Array<Partial<ChatConfigObject>>): SwarmConfigEvent[] {
        return updates.map((update, index) => ({
            event: "swarmConfigUpdate" as const,
            data: {
                chatId,
                config: {
                    ...update,
                    __typename: "ChatConfigObject" as const,
                },
            },
        }));
    }
}

/**
 * Factory for swarm state update events
 */
class SwarmStateEventFactory extends BaseEventFactory<SwarmStateEvent, SwarmSocketEventPayloads["swarmStateUpdate"]> {
    constructor() {
        super("swarmStateUpdate", {
            validation: (data) => {
                if (!data.chatId) return "chatId is required";
                if (!data.swarmId) return "swarmId is required";
                if (!data.state) return "state is required";

                const validStates = ["UNINITIALIZED", "STARTING", "RUNNING", "PAUSED", "COMPLETED", "FAILED", "TERMINATED", "ARCHIVED"];
                if (!validStates.includes(data.state)) {
                    return `Invalid state: ${data.state}`;
                }

                return true;
            },
        });
    }

    single: SwarmStateEvent = {
        event: "swarmStateUpdate",
        data: {
            chatId: "chat_123",
            swarmId: "swarm_456",
            state: "UNINITIALIZED",
            message: "Swarm created, awaiting initialization",
        },
    };

    sequence: SwarmStateEvent[] = [
        {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_lifecycle",
                swarmId: "swarm_lifecycle",
                state: "UNINITIALIZED",
                message: "Swarm created",
            },
        },
        {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_lifecycle",
                swarmId: "swarm_lifecycle",
                state: "STARTING",
                message: "Initializing agents and allocating resources...",
            },
        },
        {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_lifecycle",
                swarmId: "swarm_lifecycle",
                state: "RUNNING",
                message: "Swarm execution in progress",
            },
        },
        {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_lifecycle",
                swarmId: "swarm_lifecycle",
                state: "COMPLETED",
                message: "All tasks completed successfully",
            },
        },
    ];

    variants: Record<string, SwarmStateEvent | SwarmStateEvent[]> = {
        paused: {
            event: "swarmStateUpdate" as const,
            data: {
                chatId: "chat_paused",
                swarmId: "swarm_paused",
                state: "PAUSED",
                message: "Execution paused by user request",
            },
        },
        failed: {
            event: "swarmStateUpdate" as const,
            data: {
                chatId: "chat_failed",
                swarmId: "swarm_failed",
                state: "FAILED",
                message: "Critical error: Agent coordination timeout",
            },
        },
        terminated: {
            event: "swarmStateUpdate" as const,
            data: {
                chatId: "chat_terminated",
                swarmId: "swarm_terminated",
                state: "TERMINATED",
                message: "Swarm terminated by system administrator",
            },
        },
        archived: {
            event: "swarmStateUpdate" as const,
            data: {
                chatId: "chat_archived",
                swarmId: "swarm_archived",
                state: "ARCHIVED",
                message: "Swarm execution archived for analysis",
            },
        },
        errorStates: [
            {
                event: "swarmStateUpdate" as const,
                data: {
                    chatId: "chat_error",
                    swarmId: "swarm_error_1",
                    state: "FAILED" as const,
                    message: "Resource allocation failed: Insufficient credits",
                },
            },
            {
                event: "swarmStateUpdate" as const,
                data: {
                    chatId: "chat_error",
                    swarmId: "swarm_error_2",
                    state: "FAILED" as const,
                    message: "Network partition detected between agents",
                },
            },
            {
                event: "swarmStateUpdate" as const,
                data: {
                    chatId: "chat_error",
                    swarmId: "swarm_error_3",
                    state: "FAILED" as const,
                    message: "Consensus timeout: Agents unable to agree on strategy",
                },
            },
        ],
    };

    // Helper to create state transition sequences
    createStateTransition(
        chatId: string,
        swarmId: string,
        fromState: SwarmSocketEventPayloads["swarmStateUpdate"]["state"],
        toState: SwarmSocketEventPayloads["swarmStateUpdate"]["state"],
        message?: string,
    ): SwarmStateEvent[] {
        const transitions: Record<string, SwarmSocketEventPayloads["swarmStateUpdate"]["state"][]> = {
            UNINITIALIZED: ["STARTING"],
            STARTING: ["RUNNING", "FAILED"],
            RUNNING: ["PAUSED", "COMPLETED", "FAILED", "TERMINATED"],
            PAUSED: ["RUNNING", "TERMINATED"],
            COMPLETED: ["ARCHIVED"],
            FAILED: ["STARTING", "TERMINATED", "ARCHIVED"],
            TERMINATED: ["ARCHIVED"],
            ARCHIVED: [],
        };

        if (!transitions[fromState]?.includes(toState)) {
            throw new Error(`Invalid state transition: ${fromState} -> ${toState}`);
        }

        return [
            {
                event: "swarmStateUpdate" as const,
                data: { chatId, swarmId, state: fromState },
            },
            {
                event: "swarmStateUpdate" as const,
                data: { chatId, swarmId, state: toState, message },
            },
        ];
    }

    // Apply state changes based on event
    protected applyEventToState(state: Record<string, unknown>, event: SwarmStateEvent): Record<string, unknown> {
        return {
            ...state,
            currentState: event.data.state,
            lastStateChange: Date.now(),
            stateHistory: [...(state.stateHistory as string[] || []), event.data.state],
        };
    }
}

/**
 * Factory for swarm resource update events
 */
class SwarmResourceEventFactory extends BaseEventFactory<SwarmResourceEvent, SwarmSocketEventPayloads["swarmResourceUpdate"]> {
    constructor() {
        super("swarmResourceUpdate", {
            validation: (data) => {
                if (!data.chatId) return "chatId is required";
                if (!data.swarmId) return "swarmId is required";
                if (!data.resources) return "resources is required";

                const { allocated, consumed, remaining } = data.resources;
                if (allocated < 0) return "allocated must be non-negative";
                if (consumed < 0) return "consumed must be non-negative";
                if (remaining < 0) return "remaining must be non-negative";
                if (consumed > allocated) return "consumed cannot exceed allocated";
                if (remaining !== allocated - consumed) return "remaining must equal allocated - consumed";

                return true;
            },
        });
    }

    single: SwarmResourceEvent = {
        event: "swarmResourceUpdate",
        data: {
            chatId: "chat_123",
            swarmId: "swarm_456",
            resources: {
                allocated: 10000,
                consumed: 0,
                remaining: 10000,
            },
        },
    };

    sequence: SwarmResourceEvent[] = [
        {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_resource",
                swarmId: "swarm_resource",
                resources: {
                    allocated: 50000,
                    consumed: 0,
                    remaining: 50000,
                },
            },
        },
        {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_resource",
                swarmId: "swarm_resource",
                resources: {
                    allocated: 50000,
                    consumed: 12000,
                    remaining: 38000,
                },
            },
        },
        {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_resource",
                swarmId: "swarm_resource",
                resources: {
                    allocated: 50000,
                    consumed: 35000,
                    remaining: 15000,
                },
            },
        },
        {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_resource",
                swarmId: "swarm_resource",
                resources: {
                    allocated: 50000,
                    consumed: 48000,
                    remaining: 2000,
                },
            },
        },
    ];

    variants: Record<string, SwarmResourceEvent | SwarmResourceEvent[]> = {
        exhausted: {
            event: "swarmResourceUpdate" as const,
            data: {
                chatId: "chat_exhausted",
                swarmId: "swarm_exhausted",
                resources: {
                    allocated: 10000,
                    consumed: 10000,
                    remaining: 0,
                },
            },
        },
        increased: {
            event: "swarmResourceUpdate" as const,
            data: {
                chatId: "chat_increased",
                swarmId: "swarm_increased",
                resources: {
                    allocated: 100000,
                    consumed: 25000,
                    remaining: 75000,
                },
            },
        },
        warningLevel: {
            event: "swarmResourceUpdate" as const,
            data: {
                chatId: "chat_warning",
                swarmId: "swarm_warning",
                resources: {
                    allocated: 20000,
                    consumed: 18000,
                    remaining: 2000,
                },
            },
        },
        consumptionPatterns: [
            // Steady consumption
            {
                event: "swarmResourceUpdate" as const,
                data: {
                    chatId: "chat_steady",
                    swarmId: "swarm_steady",
                    resources: {
                        allocated: 30000,
                        consumed: 10000,
                        remaining: 20000,
                    },
                },
            },
            // Burst consumption
            {
                event: "swarmResourceUpdate" as const,
                data: {
                    chatId: "chat_burst",
                    swarmId: "swarm_burst",
                    resources: {
                        allocated: 30000,
                        consumed: 25000,
                        remaining: 5000,
                    },
                },
            },
            // Efficient consumption
            {
                event: "swarmResourceUpdate" as const,
                data: {
                    chatId: "chat_efficient",
                    swarmId: "swarm_efficient",
                    resources: {
                        allocated: 30000,
                        consumed: 5000,
                        remaining: 25000,
                    },
                },
            },
        ],
    };

    // Helper to create resource consumption over time
    createConsumptionCurve(
        chatId: string,
        swarmId: string,
        allocated: number,
        consumptionRates: number[],
    ): SwarmResourceEvent[] {
        let totalConsumed = 0;
        return consumptionRates.map(rate => {
            totalConsumed += rate;
            if (totalConsumed > allocated) {
                totalConsumed = allocated;
            }
            return {
                event: "swarmResourceUpdate" as const,
                data: {
                    chatId,
                    swarmId,
                    resources: {
                        allocated,
                        consumed: totalConsumed,
                        remaining: allocated - totalConsumed,
                    },
                },
            };
        });
    }

    // Apply resource changes to state
    protected applyEventToState(state: Record<string, unknown>, event: SwarmResourceEvent): Record<string, unknown> {
        const previousResources = state.resources as typeof event.data.resources || { allocated: 0, consumed: 0, remaining: 0 };
        const consumptionRate = event.data.resources.consumed - previousResources.consumed;

        return {
            ...state,
            resources: event.data.resources,
            resourceHistory: [...(state.resourceHistory as typeof event.data.resources[] || []), event.data.resources],
            consumptionRate,
            isResourceCritical: event.data.resources.remaining < event.data.resources.allocated * 0.1,
        };
    }
}

/**
 * Factory for swarm team update events
 */
class SwarmTeamEventFactory extends BaseEventFactory<SwarmTeamEvent, SwarmSocketEventPayloads["swarmTeamUpdate"]> {
    constructor() {
        super("swarmTeamUpdate", {
            validation: (data) => {
                if (!data.chatId) return "chatId is required";
                if (!data.swarmId) return "swarmId is required";

                // At least one team-related field should be present
                if (!data.teamId && !data.swarmLeader && !data.subtaskLeaders) {
                    return "At least one of teamId, swarmLeader, or subtaskLeaders is required";
                }

                return true;
            },
        });
    }

    single: SwarmTeamEvent = {
        event: "swarmTeamUpdate",
        data: {
            chatId: "chat_123",
            swarmId: "swarm_456",
            teamId: "team_789",
            swarmLeader: "agent_leader",
        },
    };

    sequence: SwarmTeamEvent[] = [
        {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_team",
                swarmId: "swarm_team",
                teamId: "team_initial",
            },
        },
        {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_team",
                swarmId: "swarm_team",
                swarmLeader: "agent_coordinator",
            },
        },
        {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_team",
                swarmId: "swarm_team",
                subtaskLeaders: {
                    "research": "agent_researcher",
                    "analysis": "agent_analyst",
                },
            },
        },
        {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_team",
                swarmId: "swarm_team",
                subtaskLeaders: {
                    "research": "agent_researcher",
                    "analysis": "agent_senior_analyst",
                    "synthesis": "agent_synthesizer",
                },
            },
        },
    ];

    variants: Record<string, SwarmTeamEvent | SwarmTeamEvent[]> = {
        fullTeam: {
            event: "swarmTeamUpdate" as const,
            data: {
                chatId: "chat_full_team",
                swarmId: "swarm_full_team",
                teamId: "team_complete",
                swarmLeader: "agent_orchestrator",
                subtaskLeaders: {
                    "data_collection": "agent_collector",
                    "data_analysis": "agent_analyst",
                    "visualization": "agent_visualizer",
                    "reporting": "agent_reporter",
                    "validation": "agent_validator",
                },
            },
        },
        leadershipChange: {
            event: "swarmTeamUpdate" as const,
            data: {
                chatId: "chat_leadership",
                swarmId: "swarm_leadership",
                swarmLeader: "agent_new_leader",
            },
        },
        teamReassignment: {
            event: "swarmTeamUpdate" as const,
            data: {
                chatId: "chat_reassign",
                swarmId: "swarm_reassign",
                teamId: "team_new",
                swarmLeader: "agent_leader",
                subtaskLeaders: {},
            },
        },
        hierarchicalTeam: {
            event: "swarmTeamUpdate" as const,
            data: {
                chatId: "chat_hierarchy",
                swarmId: "swarm_hierarchy",
                teamId: "team_hierarchical",
                swarmLeader: "agent_ceo",
                subtaskLeaders: {
                    "strategic_planning": "agent_cto",
                    "operations": "agent_coo",
                    "quality_assurance": "agent_qa_lead",
                    "development_frontend": "agent_frontend_lead",
                    "development_backend": "agent_backend_lead",
                    "infrastructure": "agent_devops_lead",
                },
            },
        },
        dynamicTeamFormation: [
            {
                event: "swarmTeamUpdate" as const,
                data: {
                    chatId: "chat_dynamic",
                    swarmId: "swarm_dynamic",
                    teamId: "team_forming",
                },
            },
            {
                event: "swarmTeamUpdate" as const,
                data: {
                    chatId: "chat_dynamic",
                    swarmId: "swarm_dynamic",
                    swarmLeader: "agent_initiator",
                },
            },
            {
                event: "swarmTeamUpdate" as const,
                data: {
                    chatId: "chat_dynamic",
                    swarmId: "swarm_dynamic",
                    subtaskLeaders: {
                        "exploration": "agent_explorer",
                    },
                },
            },
            {
                event: "swarmTeamUpdate" as const,
                data: {
                    chatId: "chat_dynamic",
                    swarmId: "swarm_dynamic",
                    subtaskLeaders: {
                        "exploration": "agent_explorer",
                        "implementation": "agent_implementer",
                    },
                },
            },
        ],
    };

    // Helper to create team evolution over time
    createTeamEvolution(
        chatId: string,
        swarmId: string,
        stages: Array<{
            teamId?: string;
            swarmLeader?: string;
            subtaskLeaders?: Record<string, string>;
        }>,
    ): SwarmTeamEvent[] {
        return stages.map(stage => ({
            event: "swarmTeamUpdate" as const,
            data: {
                chatId,
                swarmId,
                ...stage,
            },
        }));
    }

    // Apply team changes to state
    protected applyEventToState(state: Record<string, unknown>, event: SwarmTeamEvent): Record<string, unknown> {
        const currentTeam = state.team as Partial<typeof event.data> || {};

        return {
            ...state,
            team: {
                ...currentTeam,
                ...event.data,
                subtaskLeaders: {
                    ...(currentTeam.subtaskLeaders || {}),
                    ...(event.data.subtaskLeaders || {}),
                },
            },
            teamHistory: [...(state.teamHistory as Array<typeof event.data> || []), event.data],
            lastTeamUpdate: Date.now(),
        };
    }
}

// Create factory instances
const swarmConfigFactory = new SwarmConfigEventFactory();
const swarmStateFactory = new SwarmStateEventFactory();
const swarmResourceFactory = new SwarmResourceEventFactory();
const swarmTeamFactory = new SwarmTeamEventFactory();

// Export enhanced fixtures with backward compatibility
export const swarmEventFixtures = {
    // Original structure maintained for backward compatibility
    config: {
        initialConfig: swarmConfigFactory.single,
        configUpdate: swarmConfigFactory.variants.modelUpdate as SwarmConfigEvent,
        resourceUpdate: swarmConfigFactory.variants.resourceLimits as SwarmConfigEvent,
    },

    state: {
        uninitialized: swarmStateFactory.single,
        starting: swarmStateFactory.sequence[1],
        running: swarmStateFactory.sequence[2],
        paused: swarmStateFactory.variants.paused as SwarmStateEvent,
        completed: swarmStateFactory.sequence[3],
        failed: swarmStateFactory.variants.failed as SwarmStateEvent,
        terminated: swarmStateFactory.variants.terminated as SwarmStateEvent,
        archived: swarmStateFactory.variants.archived as SwarmStateEvent,
    },

    resources: {
        initialAllocation: swarmResourceFactory.single,
        consumptionUpdate: swarmResourceFactory.sequence[1],
        lowResources: swarmResourceFactory.variants.warningLevel as SwarmResourceEvent,
        exhausted: swarmResourceFactory.variants.exhausted as SwarmResourceEvent,
        increased: swarmResourceFactory.variants.increased as SwarmResourceEvent,
    },

    team: {
        initialTeam: swarmTeamFactory.variants.fullTeam as SwarmTeamEvent,
        leaderChange: swarmTeamFactory.variants.leadershipChange as SwarmTeamEvent,
        subtaskLeaderUpdate: swarmTeamFactory.sequence[3],
        teamReassignment: swarmTeamFactory.variants.teamReassignment as SwarmTeamEvent,
    },

    // Enhanced sequences with timing
    sequences: {
        // Basic swarm lifecycle with realistic timing
        basicLifecycle: [
            swarmStateFactory.withDelay(swarmStateFactory.single, 0),
            swarmStateFactory.withDelay(swarmStateFactory.sequence[1], 1000),
            swarmResourceFactory.withDelay(swarmResourceFactory.single, 2000),
            swarmTeamFactory.withDelay(swarmTeamFactory.single, 2100),
            swarmStateFactory.withDelay(swarmStateFactory.sequence[2], 2500),
            swarmResourceFactory.withDelay(swarmResourceFactory.sequence[1], 7500),
            swarmResourceFactory.withDelay(swarmResourceFactory.sequence[2], 12500),
            swarmStateFactory.withDelay(swarmStateFactory.sequence[3], 15000),
        ],

        // Resource exhaustion flow with correlation
        resourceExhaustionFlow: swarmResourceFactory.createCorrelated(
            "resource-exhaustion-001",
            [
                swarmStateFactory.variants.paused as SwarmStateEvent as any,
                ...swarmResourceFactory.createConsumptionCurve(
                    "chat_exhaustion",
                    "swarm_exhaustion",
                    20000,
                    [5000, 5000, 5000, 3000, 2000],
                ),
                {
                    event: "swarmStateUpdate" as const,
                    data: {
                        chatId: "chat_exhaustion",
                        swarmId: "swarm_exhaustion",
                        state: "PAUSED" as const,
                        message: "Paused due to resource exhaustion",
                    },
                } as any,
            ],
        ),

        // Team reorganization with state tracking
        teamReorganizationFlow: swarmTeamFactory.trackStateChanges([
            swarmTeamFactory.single,
            swarmTeamFactory.sequence[2],
            swarmTeamFactory.sequence[3],
            swarmTeamFactory.variants.leadershipChange as SwarmTeamEvent,
        ]),

        // Error recovery with jitter
        errorRecoveryFlow: swarmStateFactory.withJitter(
            [
                swarmStateFactory.sequence[2], // RUNNING
                swarmStateFactory.variants.failed as SwarmStateEvent,
                swarmStateFactory.sequence[1], // STARTING
                swarmTeamFactory.variants.teamReassignment as SwarmTeamEvent as any,
                swarmStateFactory.sequence[2], // RUNNING again
            ],
            2000,
            500,
        ),

        // Multi-agent coordination pattern
        multiAgentCoordination: [
            swarmConfigFactory.create({
                chatId: "chat_multi",
                config: {
                    __typename: "ChatConfigObject" as const,
                },
            }),
            ...swarmTeamFactory.createTeamEvolution("chat_multi", "swarm_multi", [
                { teamId: "team_alpha" },
                { swarmLeader: "agent_alpha_lead" },
                { subtaskLeaders: { "alpha_task": "agent_alpha_1" } },
                { teamId: "team_beta" },
                { swarmLeader: "agent_beta_lead" },
                { subtaskLeaders: { "beta_task": "agent_beta_1" } },
                {
                    swarmLeader: "agent_coordinator",
                    subtaskLeaders: {
                        "alpha_task": "agent_alpha_1",
                        "beta_task": "agent_beta_1",
                        "integration": "agent_integrator",
                    },
                },
            ]),
        ],
    },

    // Factory functions for dynamic event creation (backward compatible)
    factories: {
        createStateUpdate: (swarmId: string, state: SwarmSocketEventPayloads["swarmStateUpdate"]["state"], message?: string) =>
            swarmStateFactory.create({ chatId: "chat_123", swarmId, state, message }),

        createResourceUpdate: (swarmId: string, allocated: number, consumed: number) =>
            swarmResourceFactory.create({
                chatId: "chat_123",
                swarmId,
                resources: { allocated, consumed, remaining: allocated - consumed },
            }),

        createTeamUpdate: (swarmId: string, updates: Partial<SwarmSocketEventPayloads["swarmTeamUpdate"]>) =>
            swarmTeamFactory.create({ chatId: "chat_123", swarmId, ...updates }),

        createConfigUpdate: (chatId: string, config: Partial<ChatConfigObject>) =>
            swarmConfigFactory.create({ chatId, config }),
    },

    // Export the factory instances for advanced usage
    configFactory: swarmConfigFactory,
    stateFactory: swarmStateFactory,
    resourceFactory: swarmResourceFactory,
    teamFactory: swarmTeamFactory,

    // Advanced patterns for testing
    patterns: {
        // Simulate a complete swarm execution with realistic timing
        async simulateSwarmExecution(options?: {
            swarmId?: string;
            resourceLimit?: number;
            teamSize?: number;
        }) {
            const swarmId = options?.swarmId || "swarm_sim_" + Date.now();
            const resourceLimit = options?.resourceLimit || 50000;
            const teamSize = options?.teamSize || 5;

            const events = [
                swarmStateFactory.create({ chatId: "chat_sim", swarmId, state: "UNINITIALIZED" }),
                swarmStateFactory.create({ chatId: "chat_sim", swarmId, state: "STARTING" }),
                swarmResourceFactory.create({
                    chatId: "chat_sim",
                    swarmId,
                    resources: { allocated: resourceLimit, consumed: 0, remaining: resourceLimit },
                }),
                swarmTeamFactory.create({
                    chatId: "chat_sim",
                    swarmId,
                    teamId: "team_sim",
                    swarmLeader: "agent_lead",
                    subtaskLeaders: Object.fromEntries(
                        Array.from({ length: teamSize }, (_, i) => [`task_${i}`, `agent_${i}`]),
                    ),
                }),
                swarmStateFactory.create({ chatId: "chat_sim", swarmId, state: "RUNNING" }),
            ];

            // Add resource consumption updates
            const consumptionRates = Array.from({ length: 10 }, (_, i) =>
                Math.floor(resourceLimit / 10 * (1 + Math.random() * 0.5)),
            );
            events.push(...swarmResourceFactory.createConsumptionCurve(
                "chat_sim",
                swarmId,
                resourceLimit,
                consumptionRates,
            ));

            // Complete or fail based on resource consumption
            const finalConsumed = consumptionRates.reduce((a, b) => a + b, 0);
            if (finalConsumed < resourceLimit) {
                events.push(swarmStateFactory.create({
                    chatId: "chat_sim",
                    swarmId,
                    state: "COMPLETED",
                    message: "All tasks completed successfully",
                }));
            } else {
                events.push(swarmStateFactory.create({
                    chatId: "chat_sim",
                    swarmId,
                    state: "FAILED",
                    message: "Resource limit exceeded",
                }));
            }

            return swarmStateFactory.simulateEventFlow(events as any, {
                timing: "realtime",
                state: { swarmId, resourceLimit, teamSize },
            });
        },

        // Create a burst pattern for stress testing
        createBurstPattern(eventType: "config" | "state" | "resource" | "team", count = 100) {
            const factory = {
                config: swarmConfigFactory,
                state: swarmStateFactory,
                resource: swarmResourceFactory,
                team: swarmTeamFactory,
            }[eventType];

            return factory.createSequence("burst", { count });
        },

        // Create escalating resource consumption
        createEscalatingConsumption(swarmId: string, baseRate = 1000) {
            const rates = Array.from({ length: 10 }, (_, i) => baseRate * Math.pow(1.5, i));
            return swarmResourceFactory.createConsumptionCurve(
                "chat_escalate",
                swarmId,
                baseRate * 20,
                rates,
            );
        },
    },
};
