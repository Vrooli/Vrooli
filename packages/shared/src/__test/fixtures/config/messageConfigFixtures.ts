import { type ChatMessageRunConfig, type MessageConfigObject, type ToolFunctionCall } from "../../../shape/configs/message.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

// Constants for random ID generation
const RADIX_BASE_36 = 36;
const RANDOM_ID_LENGTH = 9;

// Counter for generating unique snowflake IDs in tests
let snowflakeCounter = 623456789012345678n;

/**
 * Message configuration fixtures for testing chat message metadata and tool usage
 */
export const messageConfigFixtures: ConfigTestFixtures<MessageConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
        resources: [], // Required field - empty array for minimal config
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        contextHints: [
            "Previous conversation mentioned project deadlines",
            "User prefers detailed explanations",
        ],
        eventTopic: "project-update",
        respondingBots: ["assistant-1", "research-bot"],
        role: "assistant",
        turnId: 42,
        toolCalls: [
            {
                id: "123456789012345678", // Valid Snowflake ID
                function: {
                    name: "search_documents",
                    arguments: JSON.stringify({ query: "project requirements", limit: 5 }),
                },
                result: {
                    success: true,
                    output: [
                        { id: "doc_1", title: "Requirements Doc", relevance: 0.95 },
                        { id: "doc_2", title: "Technical Spec", relevance: 0.87 },
                    ],
                },
            },
        ],
        runs: [
            {
                runId: "run_456",
                resourceVersionId: "routine_v_789",
                resourceVersionName: "Data Processing Routine v2",
                taskId: "task_012",
                runStatus: "completed",
                createdAt: "2024-01-01T10:00:00Z",
                completedAt: "2024-01-01T10:05:00Z",
            },
        ],
        resources: [{
            link: "https://example.com/context-doc",
            usedFor: "Context",
            translations: [{
                language: "en",
                name: "Context Document",
                description: "Additional context for message",
            }],
        }],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        contextHints: [],
        respondingBots: [],
        role: "user",
        turnId: null,
        toolCalls: [],
        runs: [],
        resources: [], // Required field
    },

    invalid: {
        missingVersion: {
            // Missing __version
            role: "user",
            turnId: 1,
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            role: "assistant",
            resources: [], // Still include required field to test version independently
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            toolCalls: "string instead of array", // Wrong type
            resources: [], // Still include required field to test toolCalls type independently
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            // @ts-expect-error - Intentionally invalid type for testing
            turnId: "not a number", // Should be number or null
            // @ts-expect-error - Intentionally invalid type for testing
            role: "admin", // Invalid role
            resources: [], // Still include required field to test other type errors independently
            toolCalls: [
                {
                    // Missing required fields
                    id: "523456789012345678", // Valid Snowflake ID
                    function: {
                        name: "test",
                        arguments: "", // Empty arguments for invalid test case
                    },
                },
            ],
        },
    },

    variants: {
        userMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "user",
            turnId: 1,
            contextHints: ["User is asking about API documentation"],
            respondingBots: ["@all"],
            resources: [], // Required field
        },

        assistantWithTools: {
            __version: LATEST_CONFIG_VERSION,
            role: "assistant",
            turnId: 2,
            resources: [], // Required field
            toolCalls: [
                {
                    id: "223456789012345678", // Valid Snowflake ID
                    function: {
                        name: "web_search",
                        arguments: JSON.stringify({
                            query: "Vrooli API documentation",
                            maxResults: 10,
                        }),
                    },
                    result: {
                        success: true,
                        output: {
                            results: [
                                { url: "https://docs.vrooli.com/api", title: "API Reference" },
                            ],
                        },
                    },
                },
                {
                    id: "323456789012345678", // Valid Snowflake ID
                    function: {
                        name: "calculator",
                        arguments: JSON.stringify({
                            expression: "2 + 2",
                        }),
                    },
                    result: {
                        success: true,
                        output: 4,
                    },
                },
            ],
        },

        systemMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "system",
            contextHints: [
                "System maintenance scheduled",
                "Increased rate limits applied",
            ],
            eventTopic: "system-notification",
            resources: [], // Required field
        },

        toolErrorMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "tool",
            turnId: 3,
            resources: [], // Required field
            toolCalls: [
                {
                    id: "423456789012345678", // Valid Snowflake ID
                    function: {
                        name: "database_query",
                        arguments: JSON.stringify({
                            table: "users",
                            query: "SELECT * FROM users WHERE invalid_column = ?",
                        }),
                    },
                    result: {
                        success: false,
                        error: {
                            code: "INVALID_COLUMN",
                            message: "Column 'invalid_column' does not exist in table 'users'",
                        },
                    },
                },
            ],
        },

        messageWithMultipleRuns: {
            __version: LATEST_CONFIG_VERSION,
            role: "assistant",
            turnId: 10,
            resources: [], // Required field
            runs: [
                {
                    runId: "run_parallel_1",
                    resourceVersionId: "routine_v_100",
                    resourceVersionName: "Data Import Routine",
                    taskId: "task_import_1",
                    runStatus: "running",
                    createdAt: "2024-01-01T12:00:00Z",
                },
                {
                    runId: "run_parallel_2",
                    resourceVersionId: "routine_v_200",
                    resourceVersionName: "Data Validation Routine",
                    taskId: "task_validate_1",
                    runStatus: "queued",
                    createdAt: "2024-01-01T12:00:05Z",
                },
                {
                    runId: "run_parallel_3",
                    resourceVersionId: "routine_v_300",
                    taskId: "task_transform_1",
                    runStatus: "failed",
                    createdAt: "2024-01-01T12:00:10Z",
                    completedAt: "2024-01-01T12:00:15Z",
                },
            ],
        },

        broadcastMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "user",
            turnId: 5,
            respondingBots: ["@all"],
            eventTopic: "broadcast-request",
            contextHints: ["Urgent request", "Requires all bots to respond"],
            resources: [], // Required field
        },

        eventDrivenMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "system",
            eventTopic: "payment-succeeded",
            contextHints: ["Payment of $99.99 processed", "Premium subscription activated"],
            respondingBots: ["billing-bot", "notification-bot"],
            resources: [], // Required field
        },
    },
};

/**
 * Create a message config with specific tool calls
 */
export function createMessageConfigWithToolCalls(
    toolCalls: ToolFunctionCall[],
    role: MessageConfigObject["role"] = "assistant",
): MessageConfigObject {
    return mergeWithBaseDefaults<MessageConfigObject>({
        role,
        toolCalls,
        turnId: 1,
    });
}

/**
 * Create a successful tool call result
 */
export function createSuccessfulToolCall(
    functionName: string,
    args: Record<string, any>,
    output: unknown,
): ToolFunctionCall {
    snowflakeCounter++;
    return {
        id: snowflakeCounter.toString(), // Valid Snowflake ID
        function: {
            name: functionName,
            arguments: JSON.stringify(args),
        },
        result: {
            success: true,
            output,
        },
    };
}

// Constants - using already declared RANDOM_ID_LENGTH from top of file

/**
 * Create a failed tool call result
 */
export function createFailedToolCall(
    functionName: string,
    args: Record<string, any>,
    errorCode: string,
    errorMessage: string,
): ToolFunctionCall {
    snowflakeCounter++;
    return {
        id: snowflakeCounter.toString(), // Valid Snowflake ID
        function: {
            name: functionName,
            arguments: JSON.stringify(args),
        },
        result: {
            success: false,
            error: {
                code: errorCode,
                message: errorMessage,
            },
        },
    };
}

/**
 * Create a message config with runs
 */
export function createMessageConfigWithRuns(
    runs: ChatMessageRunConfig[],
    role: MessageConfigObject["role"] = "assistant",
): MessageConfigObject {
    return mergeWithBaseDefaults<MessageConfigObject>({
        role,
        runs,
        turnId: 1,
    });
}

/**
 * Create a run configuration
 */
export function createRunConfig(
    overrides: Partial<ChatMessageRunConfig> = {},
): ChatMessageRunConfig {
    const baseRun: ChatMessageRunConfig = {
        runId: `run_${Date.now()}`,
        resourceVersionId: `routine_v_${Date.now()}`,
        taskId: `task_${Date.now()}`,
        runStatus: "pending",
        createdAt: new Date().toISOString(),
    };

    return {
        ...baseRun,
        ...overrides,
    };
}
