import { type MessageConfigObject, type ToolFunctionCall, type ToolFunctionCallResult, type ChatMessageRunConfig } from "../../../shape/configs/message.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Message configuration fixtures for testing chat message metadata and tool usage
 */
export const messageConfigFixtures: ConfigTestFixtures<MessageConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },
    
    complete: {
        __version: LATEST_CONFIG_VERSION,
        contextHints: [
            "Previous conversation mentioned project deadlines",
            "User prefers detailed explanations"
        ],
        eventTopic: "project-update",
        respondingBots: ["assistant-1", "research-bot"],
        role: "assistant",
        turnId: 42,
        toolCalls: [
            {
                id: "call_123",
                function: {
                    name: "search_documents",
                    arguments: JSON.stringify({ query: "project requirements", limit: 5 })
                },
                result: {
                    success: true,
                    output: [
                        { id: "doc_1", title: "Requirements Doc", relevance: 0.95 },
                        { id: "doc_2", title: "Technical Spec", relevance: 0.87 }
                    ]
                }
            }
        ],
        runs: [
            {
                runId: "run_456",
                resourceVersionId: "routine_v_789",
                resourceVersionName: "Data Processing Routine v2",
                taskId: "task_012",
                runStatus: "completed",
                createdAt: "2024-01-01T10:00:00Z",
                completedAt: "2024-01-01T10:05:00Z"
            }
        ],
        resources: [{
            link: "https://example.com/context-doc",
            usedFor: "Context",
            translations: [{
                language: "en",
                name: "Context Document",
                description: "Additional context for message"
            }]
        }]
    },
    
    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        contextHints: [],
        respondingBots: [],
        role: "user",
        turnId: null,
        toolCalls: [],
        runs: [],
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
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            toolCalls: "string instead of array", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            turnId: "not a number" as any, // Should be number or null
            role: "admin" as any, // Invalid role
            toolCalls: [
                {
                    // Missing required fields
                    id: "missing_args",
                    function: {
                        name: "test"
                        // Missing arguments
                    } as any
                }
            ] as any
        }
    },
    
    variants: {
        userMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "user",
            turnId: 1,
            contextHints: ["User is asking about API documentation"],
            respondingBots: ["@all"]
        },
        
        assistantWithTools: {
            __version: LATEST_CONFIG_VERSION,
            role: "assistant",
            turnId: 2,
            toolCalls: [
                {
                    id: "call_search_1",
                    function: {
                        name: "web_search",
                        arguments: JSON.stringify({ 
                            query: "Vrooli API documentation",
                            maxResults: 10 
                        })
                    },
                    result: {
                        success: true,
                        output: {
                            results: [
                                { url: "https://docs.vrooli.com/api", title: "API Reference" }
                            ]
                        }
                    }
                },
                {
                    id: "call_calc_1",
                    function: {
                        name: "calculator",
                        arguments: JSON.stringify({ 
                            expression: "2 + 2" 
                        })
                    },
                    result: {
                        success: true,
                        output: 4
                    }
                }
            ]
        },
        
        systemMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "system",
            contextHints: [
                "System maintenance scheduled",
                "Increased rate limits applied"
            ],
            eventTopic: "system-notification"
        },
        
        toolErrorMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "tool",
            turnId: 3,
            toolCalls: [
                {
                    id: "call_failed_1",
                    function: {
                        name: "database_query",
                        arguments: JSON.stringify({ 
                            table: "users",
                            query: "SELECT * FROM users WHERE invalid_column = ?" 
                        })
                    },
                    result: {
                        success: false,
                        error: {
                            code: "INVALID_COLUMN",
                            message: "Column 'invalid_column' does not exist in table 'users'"
                        }
                    }
                }
            ]
        },
        
        messageWithMultipleRuns: {
            __version: LATEST_CONFIG_VERSION,
            role: "assistant",
            turnId: 10,
            runs: [
                {
                    runId: "run_parallel_1",
                    resourceVersionId: "routine_v_100",
                    resourceVersionName: "Data Import Routine",
                    taskId: "task_import_1",
                    runStatus: "running",
                    createdAt: "2024-01-01T12:00:00Z"
                },
                {
                    runId: "run_parallel_2",
                    resourceVersionId: "routine_v_200",
                    resourceVersionName: "Data Validation Routine",
                    taskId: "task_validate_1",
                    runStatus: "queued",
                    createdAt: "2024-01-01T12:00:05Z"
                },
                {
                    runId: "run_parallel_3",
                    resourceVersionId: "routine_v_300",
                    taskId: "task_transform_1",
                    runStatus: "failed",
                    createdAt: "2024-01-01T12:00:10Z",
                    completedAt: "2024-01-01T12:00:15Z"
                }
            ]
        },
        
        broadcastMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "user",
            turnId: 5,
            respondingBots: ["@all"],
            eventTopic: "broadcast-request",
            contextHints: ["Urgent request", "Requires all bots to respond"]
        },
        
        eventDrivenMessage: {
            __version: LATEST_CONFIG_VERSION,
            role: "system",
            eventTopic: "payment-succeeded",
            contextHints: ["Payment of $99.99 processed", "Premium subscription activated"],
            respondingBots: ["billing-bot", "notification-bot"]
        }
    }
};

/**
 * Create a message config with specific tool calls
 */
export function createMessageConfigWithToolCalls(
    toolCalls: ToolFunctionCall[],
    role: MessageConfigObject["role"] = "assistant"
): MessageConfigObject {
    return mergeWithBaseDefaults<MessageConfigObject>({
        role,
        toolCalls,
        turnId: 1
    });
}

/**
 * Create a successful tool call result
 */
export function createSuccessfulToolCall(
    functionName: string,
    args: Record<string, any>,
    output: unknown
): ToolFunctionCall {
    return {
        id: `call_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
        function: {
            name: functionName,
            arguments: JSON.stringify(args)
        },
        result: {
            success: true,
            output
        }
    };
}

/**
 * Create a failed tool call result
 */
export function createFailedToolCall(
    functionName: string,
    args: Record<string, any>,
    errorCode: string,
    errorMessage: string
): ToolFunctionCall {
    return {
        id: `call_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
        function: {
            name: functionName,
            arguments: JSON.stringify(args)
        },
        result: {
            success: false,
            error: {
                code: errorCode,
                message: errorMessage
            }
        }
    };
}

/**
 * Create a message config with runs
 */
export function createMessageConfigWithRuns(
    runs: ChatMessageRunConfig[],
    role: MessageConfigObject["role"] = "assistant"
): MessageConfigObject {
    return mergeWithBaseDefaults<MessageConfigObject>({
        role,
        runs,
        turnId: 1
    });
}

/**
 * Create a run configuration
 */
export function createRunConfig(
    overrides: Partial<ChatMessageRunConfig> = {}
): ChatMessageRunConfig {
    const baseRun: ChatMessageRunConfig = {
        runId: `run_${Date.now()}`,
        resourceVersionId: `routine_v_${Date.now()}`,
        taskId: `task_${Date.now()}`,
        runStatus: "pending",
        createdAt: new Date().toISOString()
    };
    
    return {
        ...baseRun,
        ...overrides
    };
}