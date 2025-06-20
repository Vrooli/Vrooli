import { type ChatConfigObject, PendingToolCallStatus } from "../../../shape/configs/chat.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";

/**
 * Chat configuration fixtures for testing swarm chat behavior settings
 */
export interface ChatConfigTestFixtures {
    minimal: ChatConfigObject;
    complete: ChatConfigObject;
    withDefaults: ChatConfigObject;
    invalid: {
        missingVersion?: Partial<ChatConfigObject>;
        invalidVersion?: ChatConfigObject;
        malformedStructure?: any;
        invalidTypes?: Partial<ChatConfigObject>;
    };
    variants: Record<string, ChatConfigObject>;
}

export const chatConfigFixtures: ChatConfigTestFixtures = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
        stats: {
            totalToolCalls: 0,
            totalCredits: "0",
            startedAt: null,
            lastProcessingCycleEndedAt: null,
        },
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        goal: "Complete the user's tasks efficiently and collaboratively",
        preferredModel: "claude-3-opus",
        subtasks: [{
            id: "task-1",
            description: "Initialize the conversation",
            status: "done",
            created_at: new Date().toISOString(),
            priority: "medium",
        }],
        swarmLeader: "bot-leader-1",
        subtaskLeaders: {
            "task-1": "bot-leader-1",
        },
        teamId: "team-123",
        blackboard: [{
            id: "note-1",
            value: "Initial conversation setup complete",
            created_at: new Date().toISOString(),
        }],
        resources: [{
            id: "resource-1",
            kind: "Note",
            creator_bot_id: "bot-leader-1",
            created_at: new Date().toISOString(),
        }],
        records: [{
            id: "call-1",
            routine_id: "routine-123",
            routine_name: "SearchWeb",
            params: { query: "test" },
            output_resource_ids: ["resource-1"],
            caller_bot_id: "bot-leader-1",
            created_at: new Date().toISOString(),
        }],
        eventSubscriptions: {
            "swarm/subtask/+": ["bot-leader-1"],
            "swarm/resource/+": ["bot-leader-1"],
        },
        policy: {
            visibility: "private",
            acl: ["bot-leader-1", "user-123"],
        },
        stats: {
            totalToolCalls: 5,
            totalCredits: "1000",
            startedAt: Date.now(),
            lastProcessingCycleEndedAt: Date.now(),
        },
        limits: {
            maxToolCallsPerBotResponse: 10,
            maxToolCalls: 50,
            maxCreditsPerBotResponse: "100",
            maxCredits: "1000",
            maxDurationPerBotResponseMs: 60000,
            maxDurationMs: 600000,
            delayBetweenProcessingCyclesMs: 1000,
        },
        scheduling: {
            defaultDelayMs: 0,
            toolSpecificDelays: {
                "web_search": 2000,
                "file_upload": 5000,
            },
            requiresApprovalTools: ["web_search"],
            approvalTimeoutMs: 300000,
            autoRejectOnTimeout: true,
        },
        pendingToolCalls: [{
            pendingId: "pending-1",
            toolCallId: "call-123",
            toolName: "web_search",
            toolArguments: "{\"query\": \"test\"}",
            callerBotId: "bot-leader-1",
            conversationId: "chat-123",
            requestedAt: Date.now(),
            status: PendingToolCallStatus.PENDING_APPROVAL,
            executionAttempts: 0,
        }],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        stats: {
            totalToolCalls: 0,
            totalCredits: "0",
            startedAt: Date.now(),
            lastProcessingCycleEndedAt: null,
        },
    },

    invalid: {
        missingVersion: {
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: null,
                lastProcessingCycleEndedAt: null,
            },
        },
        invalidVersion: {
            __version: "0.1",
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: null,
                lastProcessingCycleEndedAt: null,
            },
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            stats: "not an object",
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: null,
                lastProcessingCycleEndedAt: null,
            },
            limits: {
                maxToolCalls: 0,
                maxCredits: "0",
            },
        },
    },

    variants: {
        publicSwarm: {
            __version: LATEST_CONFIG_VERSION,
            goal: "Collaborate on public tasks",
            policy: {
                visibility: "open",
            },
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: Date.now(),
                lastProcessingCycleEndedAt: null,
            },
        },

        restrictedTeamSwarm: {
            __version: LATEST_CONFIG_VERSION,
            goal: "Team collaboration with restricted access",
            teamId: "team-456",
            policy: {
                visibility: "restricted",
                acl: ["team-456"],
            },
            limits: {
                maxToolCalls: 100,
                maxCredits: "5000",
            },
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: Date.now(),
                lastProcessingCycleEndedAt: null,
            },
        },

        highLimitSwarm: {
            __version: LATEST_CONFIG_VERSION,
            goal: "Complex task requiring high limits",
            limits: {
                maxToolCallsPerBotResponse: 25,
                maxToolCalls: 500,
                maxCreditsPerBotResponse: "500",
                maxCredits: "10000",
                maxDurationPerBotResponseMs: 300000,
                maxDurationMs: 3600000,
            },
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: Date.now(),
                lastProcessingCycleEndedAt: null,
            },
        },
    },
};

/**
 * Create a chat config with specific swarm settings
 */
export function createChatConfigWithGoal(
    goal: string,
    teamId?: string,
): ChatConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        goal,
        teamId,
        stats: {
            totalToolCalls: 0,
            totalCredits: "0",
            startedAt: Date.now(),
            lastProcessingCycleEndedAt: null,
        },
    };
}

/**
 * Create a chat config with specific limits
 */
export function createChatConfigWithLimits(
    limits: ChatConfigObject["limits"],
): ChatConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        limits,
        stats: {
            totalToolCalls: 0,
            totalCredits: "0",
            startedAt: Date.now(),
            lastProcessingCycleEndedAt: null,
        },
    };
}
