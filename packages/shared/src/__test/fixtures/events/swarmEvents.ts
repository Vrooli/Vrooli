/**
 * Swarm execution event fixtures for testing AI multi-agent systems
 */

import { type SwarmSocketEventPayloads } from "../../../consts/socketEvents.js";
import { type ChatConfigObject } from "../../../shape/configs/chat.js";
import { chatConfigFixtures } from "../config/chatConfigFixtures.js";

export const swarmEventFixtures = {
    config: {
        // Initial swarm configuration
        initialConfig: {
            event: "swarmConfigUpdate",
            data: {
                chatId: "chat_123",
                config: {
                    ...chatConfigFixtures.complete,
                    __typename: "ChatConfigObject" as const,
                } satisfies Partial<ChatConfigObject> & { __typename?: "ChatConfigObject" },
            } satisfies SwarmSocketEventPayloads["swarmConfigUpdate"],
        },

        // Configuration update
        configUpdate: {
            event: "swarmConfigUpdate",
            data: {
                chatId: "chat_123",
                config: {
                    autonomy: "semi_autonomous",
                    agentGroups: {
                        enabled: true,
                        maxAgentsPerGroup: 5,
                        groupCoordinationStrategy: "consensus",
                    },
                    __typename: "ChatConfigObject" as const,
                },
            } satisfies SwarmSocketEventPayloads["swarmConfigUpdate"],
        },

        // Resource limit update
        resourceUpdate: {
            event: "swarmConfigUpdate",
            data: {
                chatId: "chat_123",
                config: {
                    resourceAllocation: {
                        maxTokens: 50000,
                        maxExecutionTime: 300000,
                        memoryLimit: 1024,
                    },
                    __typename: "ChatConfigObject" as const,
                },
            } satisfies SwarmSocketEventPayloads["swarmConfigUpdate"],
        },
    },

    state: {
        // Swarm initialization
        uninitialized: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "UNINITIALIZED" as const,
                message: "Swarm created, awaiting initialization",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },

        // Swarm starting
        starting: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "STARTING" as const,
                message: "Initializing agents and resources...",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },

        // Swarm running
        running: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "RUNNING" as const,
                message: "Swarm execution in progress",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },

        // Swarm paused
        paused: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "PAUSED" as const,
                message: "Execution paused by user request",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },

        // Swarm completed
        completed: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "COMPLETED" as const,
                message: "All tasks completed successfully",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },

        // Swarm failed
        failed: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "FAILED" as const,
                message: "Critical error: Agent coordination failure",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },

        // Swarm terminated
        terminated: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "TERMINATED" as const,
                message: "Swarm terminated by system administrator",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },

        // Swarm archived
        archived: {
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                state: "ARCHIVED" as const,
                message: "Swarm execution archived for review",
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        },
    },

    resources: {
        // Initial resource allocation
        initialAllocation: {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                resources: {
                    allocated: 10000,
                    consumed: 0,
                    remaining: 10000,
                },
            } satisfies SwarmSocketEventPayloads["swarmResourceUpdate"],
        },

        // Resource consumption update
        consumptionUpdate: {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                resources: {
                    allocated: 10000,
                    consumed: 3500,
                    remaining: 6500,
                },
            } satisfies SwarmSocketEventPayloads["swarmResourceUpdate"],
        },

        // Resource warning
        lowResources: {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                resources: {
                    allocated: 10000,
                    consumed: 9000,
                    remaining: 1000,
                },
            } satisfies SwarmSocketEventPayloads["swarmResourceUpdate"],
        },

        // Resources exhausted
        exhausted: {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                resources: {
                    allocated: 10000,
                    consumed: 10000,
                    remaining: 0,
                },
            } satisfies SwarmSocketEventPayloads["swarmResourceUpdate"],
        },

        // Resource increase
        increased: {
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                resources: {
                    allocated: 20000,
                    consumed: 10000,
                    remaining: 10000,
                },
            } satisfies SwarmSocketEventPayloads["swarmResourceUpdate"],
        },
    },

    team: {
        // Initial team assignment
        initialTeam: {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                teamId: "team_789",
                swarmLeader: "agent_leader",
                subtaskLeaders: {
                    "research": "agent_researcher",
                    "analysis": "agent_analyst",
                    "synthesis": "agent_synthesizer",
                },
            } satisfies SwarmSocketEventPayloads["swarmTeamUpdate"],
        },

        // Leader change
        leaderChange: {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                swarmLeader: "agent_new_leader",
            } satisfies SwarmSocketEventPayloads["swarmTeamUpdate"],
        },

        // Subtask leader update
        subtaskLeaderUpdate: {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                subtaskLeaders: {
                    "research": "agent_researcher",
                    "analysis": "agent_new_analyst",
                    "synthesis": "agent_synthesizer",
                    "validation": "agent_validator", // New subtask
                },
            } satisfies SwarmSocketEventPayloads["swarmTeamUpdate"],
        },

        // Team reassignment
        teamReassignment: {
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_123",
                swarmId: "swarm_456",
                teamId: "team_new",
                swarmLeader: "agent_leader",
                subtaskLeaders: {},
            } satisfies SwarmSocketEventPayloads["swarmTeamUpdate"],
        },
    },

    // Event sequences for testing swarm execution flows
    sequences: {
        // Basic swarm lifecycle
        basicLifecycle: [
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.uninitialized.data },
            { delay: 1000 },
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.starting.data },
            { delay: 2000 },
            { event: "swarmResourceUpdate", data: swarmEventFixtures.resources.initialAllocation.data },
            { event: "swarmTeamUpdate", data: swarmEventFixtures.team.initialTeam.data },
            { delay: 500 },
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.running.data },
            { delay: 5000 },
            { event: "swarmResourceUpdate", data: swarmEventFixtures.resources.consumptionUpdate.data },
            { delay: 5000 },
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.completed.data },
        ],

        // Resource exhaustion flow
        resourceExhaustionFlow: [
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.running.data },
            { event: "swarmResourceUpdate", data: swarmEventFixtures.resources.consumptionUpdate.data },
            { delay: 2000 },
            { event: "swarmResourceUpdate", data: swarmEventFixtures.resources.lowResources.data },
            { delay: 1000 },
            { event: "swarmResourceUpdate", data: swarmEventFixtures.resources.exhausted.data },
            { event: "swarmStateUpdate", data: { ...swarmEventFixtures.state.paused.data, message: "Paused due to resource exhaustion" } },
        ],

        // Team reorganization flow
        teamReorganizationFlow: [
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.running.data },
            { event: "swarmTeamUpdate", data: swarmEventFixtures.team.initialTeam.data },
            { delay: 3000 },
            { event: "swarmTeamUpdate", data: swarmEventFixtures.team.subtaskLeaderUpdate.data },
            { delay: 2000 },
            { event: "swarmTeamUpdate", data: swarmEventFixtures.team.leaderChange.data },
        ],

        // Error recovery flow
        errorRecoveryFlow: [
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.running.data },
            { delay: 3000 },
            { event: "swarmStateUpdate", data: { ...swarmEventFixtures.state.failed.data, message: "Agent communication timeout" } },
            { delay: 2000 },
            { event: "swarmStateUpdate", data: { ...swarmEventFixtures.state.starting.data, message: "Attempting recovery..." } },
            { delay: 1000 },
            { event: "swarmTeamUpdate", data: swarmEventFixtures.team.teamReassignment.data },
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.running.data },
        ],

        // Configuration update flow
        configurationUpdateFlow: [
            { event: "swarmConfigUpdate", data: swarmEventFixtures.config.initialConfig.data },
            { event: "swarmStateUpdate", data: swarmEventFixtures.state.running.data },
            { delay: 2000 },
            { event: "swarmConfigUpdate", data: swarmEventFixtures.config.configUpdate.data },
            { delay: 1000 },
            { event: "swarmConfigUpdate", data: swarmEventFixtures.config.resourceUpdate.data },
            { event: "swarmResourceUpdate", data: swarmEventFixtures.resources.increased.data },
        ],
    },

    // Factory functions for dynamic event creation
    factories: {
        createStateUpdate: (swarmId: string, state: SwarmSocketEventPayloads["swarmStateUpdate"]["state"], message?: string) => ({
            event: "swarmStateUpdate",
            data: {
                chatId: "chat_123",
                swarmId,
                state,
                message,
            } satisfies SwarmSocketEventPayloads["swarmStateUpdate"],
        }),

        createResourceUpdate: (swarmId: string, allocated: number, consumed: number) => ({
            event: "swarmResourceUpdate",
            data: {
                chatId: "chat_123",
                swarmId,
                resources: {
                    allocated,
                    consumed,
                    remaining: allocated - consumed,
                },
            } satisfies SwarmSocketEventPayloads["swarmResourceUpdate"],
        }),

        createTeamUpdate: (swarmId: string, updates: Partial<SwarmSocketEventPayloads["swarmTeamUpdate"]>) => ({
            event: "swarmTeamUpdate",
            data: {
                chatId: "chat_123",
                swarmId,
                ...updates,
            } satisfies SwarmSocketEventPayloads["swarmTeamUpdate"],
        }),

        createConfigUpdate: (chatId: string, config: Partial<ChatConfigObject>) => ({
            event: "swarmConfigUpdate",
            data: {
                chatId,
                config: {
                    ...config,
                    __typename: "ChatConfigObject" as const,
                },
            } satisfies SwarmSocketEventPayloads["swarmConfigUpdate"],
        }),
    },
};