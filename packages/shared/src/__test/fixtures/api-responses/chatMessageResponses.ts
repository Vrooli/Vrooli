/* c8 ignore start */
/**
 * Chat Message API Response Fixtures
 * 
 * Comprehensive fixtures for chat message management including
 * message creation, search, threading, and assistant responses.
 */

import type {
    ChatMessage,
    ChatMessageCreateInput,
    ChatMessageUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import { chatResponseFactory } from "./chatResponses.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_MESSAGE_LENGTH = 10000;
const MAX_SEQUENCE = 999999;
const RATE_LIMIT_PER_HOUR = 100;

// Message roles
const MESSAGE_ROLES = ["user", "assistant", "system"] as const;

/**
 * Chat Message API response factory
 */
export class ChatMessageResponseFactory extends BaseAPIResponseFactory<
    ChatMessage,
    ChatMessageCreateInput,
    ChatMessageUpdateInput
> {
    protected readonly entityName = "chat_message";

    /**
     * Create mock chat message data
     */
    createMockData(options?: MockDataOptions): ChatMessage {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const messageId = options?.overrides?.id || generatePK().toString();

        const baseChatMessage: ChatMessage = {
            __typename: "ChatMessage",
            id: messageId,
            createdAt: now,
            updatedAt: now,
            config: {
                __version: "1.0.0",
                role: "user",
                resources: [],
            },
            language: "en",
            text: "Hello, this is a test message!",
            sequence: 1,
            versionIndex: 0,
            score: 0,
            reportsCount: 0,
            chat: chatResponseFactory.createMockData(),
            user: userResponseFactory.createMockData(),
            parent: null,
            reactionSummaries: [],
            reports: [],
            translations: [],
            translationsCount: 0,
            you: {
                canDelete: true,
                canReact: true,
                canReply: true,
                canReport: false,
                canUpdate: true,
                reaction: null,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseChatMessage,
                text: scenario === "edge-case"
                    ? "A".repeat(MAX_MESSAGE_LENGTH) // Maximum length message
                    : "This is a comprehensive chat message with all possible features including tool calls and resource references.",
                config: {
                    __version: "1.0.0",
                    role: scenario === "edge-case" ? "system" : "assistant",
                    resources: [
                        {
                            id: generatePK().toString(),
                            name: "Test Resource",
                            type: "document",
                        },
                    ],
                    toolCalls: scenario === "complete" ? [{
                        id: "tool_call_1",
                        function: {
                            name: "search_web",
                            arguments: JSON.stringify({ query: "test query" }),
                        },
                        result: {
                            success: true,
                            output: "Search completed successfully",
                        },
                    }] : undefined,
                    respondingBots: scenario === "complete" ? ["@assistant"] : undefined,
                },
                sequence: scenario === "edge-case" ? MAX_SEQUENCE : 5,
                versionIndex: scenario === "edge-case" ? 3 : 1,
                score: scenario === "complete" ? 15 : 0,
                chat: chatResponseFactory.createMockData({ scenario: "complete" }),
                user: scenario === "complete"
                    ? userResponseFactory.createMockData({ scenario: "complete" })
                    : userResponseFactory.createMockData({
                        overrides: {
                            isBot: true,
                            name: "System",
                            handle: "system",
                        },
                    }),
                parent: scenario === "edge-case" ? {
                    id: generatePK().toString(),
                    createdAt: new Date(Date.now().toISOString() - (60 * 60 * 1000)).toISOString(),
                } : null,
                reactionSummaries: scenario === "complete" ? [
                    {
                        emotion: "👍",
                        count: 5,
                    },
                    {
                        emotion: "❤️",
                        count: 2,
                    },
                ] : [],
                you: {
                    canDelete: scenario !== "edge-case",
                    canReact: true,
                    canReply: true,
                    canReport: scenario !== "edge-case",
                    canUpdate: scenario !== "edge-case",
                    reaction: scenario === "complete" ? "👍" : null,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseChatMessage,
            ...options?.overrides,
        };
    }

    /**
     * Create chat message from input
     */
    createFromInput(input: ChatMessageCreateInput): ChatMessage {
        const now = new Date().toISOString();
        const messageId = generatePK().toString();

        return {
            __typename: "ChatMessage",
            id: messageId,
            createdAt: now,
            updatedAt: now,
            config: input.config || {
                __version: "1.0.0",
                role: "user",
                resources: [],
            },
            language: input.language || "en",
            text: input.text,
            sequence: input.sequence || 1,
            versionIndex: input.versionIndex || 0,
            score: 0,
            reportsCount: 0,
            chat: chatResponseFactory.createMockData({ overrides: { id: input.chatConnect } }),
            user: userResponseFactory.createMockData({ overrides: { id: input.userConnect } }),
            parent: input.parentConnect ? {
                id: input.parentConnect,
                createdAt: new Date(Date.now().toISOString() - (60 * 60 * 1000)).toISOString(),
            } : null,
            reactionSummaries: [],
            reports: [],
            translations: [],
            translationsCount: 0,
            you: {
                canDelete: true,
                canReact: true,
                canReply: true,
                canReport: false,
                canUpdate: true,
                reaction: null,
            },
        };
    }

    /**
     * Update chat message from input
     */
    updateFromInput(existing: ChatMessage, input: ChatMessageUpdateInput): ChatMessage {
        const updates: Partial<ChatMessage> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.text !== undefined) updates.text = input.text;
        if (input.config !== undefined) updates.config = input.config;
        if (input.language !== undefined) updates.language = input.language;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ChatMessageCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.text || input.text.trim().length === 0) {
            errors.text = "Message text is required";
        } else if (input.text.length > MAX_MESSAGE_LENGTH) {
            errors.text = `Message must be ${MAX_MESSAGE_LENGTH} characters or less`;
        }

        if (!input.chatConnect) {
            errors.chatConnect = "Chat ID is required";
        }

        if (!input.userConnect) {
            errors.userConnect = "User ID is required";
        }

        if (input.sequence !== undefined && (input.sequence < 1 || input.sequence > MAX_SEQUENCE)) {
            errors.sequence = `Sequence must be between 1 and ${MAX_SEQUENCE}`;
        }

        if (input.versionIndex !== undefined && input.versionIndex < 0) {
            errors.versionIndex = "Version index cannot be negative";
        }

        if (input.language && !/^[a-z]{2}(-[A-Z]{2})?$/.test(input.language)) {
            errors.language = "Language must be a valid language code (e.g., 'en', 'es-ES')";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ChatMessageUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.text !== undefined) {
            if (!input.text || input.text.trim().length === 0) {
                errors.text = "Message text cannot be empty";
            } else if (input.text.length > MAX_MESSAGE_LENGTH) {
                errors.text = `Message must be ${MAX_MESSAGE_LENGTH} characters or less`;
            }
        }

        if (input.language !== undefined && !/^[a-z]{2}(-[A-Z]{2})?$/.test(input.language)) {
            errors.language = "Language must be a valid language code (e.g., 'en', 'es-ES')";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create messages for different roles
     */
    createMessagesForAllRoles(): ChatMessage[] {
        return MESSAGE_ROLES.map((role, index) =>
            this.createMockData({
                overrides: {
                    id: `message_${role}_${index}`,
                    config: {
                        __version: "1.0.0",
                        role,
                        resources: [],
                    },
                    text: role === "user"
                        ? "Hello, can you help me with a question?"
                        : role === "assistant"
                            ? "Of course! I'd be happy to help you with your question."
                            : "Chat session has been initialized successfully.",
                    sequence: index + 1,
                    user: role === "system" || role === "assistant"
                        ? userResponseFactory.createMockData({
                            overrides: {
                                isBot: true,
                                name: role === "system" ? "System" : "AI Assistant",
                                handle: role,
                            },
                        })
                        : userResponseFactory.createMockData(),
                },
            }),
        );
    }

    /**
     * Create a conversation thread
     */
    createConversationThread(count = 5): ChatMessage[] {
        const chatId = generatePK().toString();
        const userId = generatePK().toString();
        const assistantId = generatePK().toString();

        const chat = chatResponseFactory.createMockData({ overrides: { id: chatId } });
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });
        const assistant = userResponseFactory.createMockData({
            overrides: {
                id: assistantId,
                isBot: true,
                name: "AI Assistant",
                handle: "assistant",
            },
        });

        return Array.from({ length: count }, (_, index) => {
            const isUserMessage = index % 2 === 0;
            const messageTime = new Date(Date.now().toISOString() - ((count - index) * 5 * 60 * 1000)); // 5 minutes apart

            return this.createMockData({
                overrides: {
                    id: `thread_message_${index}`,
                    chat,
                    user: isUserMessage ? user : assistant,
                    text: isUserMessage
                        ? `User message ${Math.floor(index / 2) + 1}: This is what I'm thinking about...`
                        : `Assistant response ${Math.floor(index / 2) + 1}: Here's my response to your question.`,
                    config: {
                        __version: "1.0.0",
                        role: isUserMessage ? "user" : "assistant",
                        resources: [],
                        ...((!isUserMessage && index > 0) && {
                            toolCalls: [{
                                id: `tool_${index}`,
                                function: {
                                    name: "analyze_context",
                                    arguments: JSON.stringify({ context: "user_question" }),
                                },
                                result: {
                                    success: true,
                                    output: "Context analyzed successfully",
                                },
                            }],
                        }),
                    },
                    sequence: index + 1,
                    createdAt: messageTime.toISOString(),
                    updatedAt: messageTime.toISOString(),
                },
            });
        });
    }

    /**
     * Create threaded messages (replies)
     */
    createThreadedMessages(): ChatMessage[] {
        const parentMessage = this.createMockData({
            overrides: {
                id: "parent_thread_message",
                text: "What's the best way to implement authentication?",
                sequence: 1,
            },
        });

        const replies = Array.from({ length: 3 }, (_, index) =>
            this.createMockData({
                overrides: {
                    id: `reply_message_${index}`,
                    text: `Reply ${index + 1}: Here's one approach to authentication...`,
                    sequence: index + 2,
                    parent: {
                        id: parentMessage.id,
                        createdAt: parentMessage.created_at,
                    },
                    createdAt: new Date(Date.now().toISOString() + ((index + 1) * 30 * 60 * 1000)).toISOString(), // 30 minutes apart
                },
            }),
        );

        return [parentMessage, ...replies];
    }

    /**
     * Create messages with different content types
     */
    createMessagesWithVariedContent(): ChatMessage[] {
        return [
            // Simple text message
            this.createMockData({
                overrides: {
                    id: "simple_text_message",
                    text: "Hello, how are you today?",
                    config: {
                        __version: "1.0.0",
                        role: "user",
                        resources: [],
                    },
                },
            }),

            // Message with tool calls
            this.createMockData({
                overrides: {
                    id: "tool_call_message",
                    text: "Let me search for that information.",
                    config: {
                        __version: "1.0.0",
                        role: "assistant",
                        resources: [],
                        toolCalls: [{
                            id: "search_tool",
                            function: {
                                name: "web_search",
                                arguments: JSON.stringify({
                                    query: "machine learning algorithms",
                                    max_results: 5,
                                }),
                            },
                            result: {
                                success: true,
                                output: "Found 5 relevant articles about machine learning algorithms",
                            },
                        }],
                    },
                    user: userResponseFactory.createMockData({
                        overrides: {
                            isBot: true,
                            name: "AI Assistant",
                            handle: "assistant",
                        },
                    }),
                },
            }),

            // Message with resources
            this.createMockData({
                overrides: {
                    id: "resource_message",
                    text: "I found these helpful resources for you.",
                    config: {
                        __version: "1.0.0",
                        role: "assistant",
                        resources: [
                            {
                                id: generatePK().toString(),
                                name: "Authentication Best Practices",
                                type: "document",
                            },
                            {
                                id: generatePK().toString(),
                                name: "Security Guidelines",
                                type: "guide",
                            },
                        ],
                    },
                    user: userResponseFactory.createMockData({
                        overrides: {
                            isBot: true,
                            name: "AI Assistant",
                            handle: "assistant",
                        },
                    }),
                },
            }),

            // Long message
            this.createMockData({
                overrides: {
                    id: "long_message",
                    text: "This is a very long message that contains a lot of information. ".repeat(50),
                    config: {
                        __version: "1.0.0",
                        role: "user",
                        resources: [],
                    },
                },
            }),

            // Message with reactions
            this.createMockData({
                overrides: {
                    id: "reacted_message",
                    text: "This message has received some reactions!",
                    reactionSummaries: [
                        { emotion: "👍", count: 8 },
                        { emotion: "❤️", count: 3 },
                        { emotion: "😊", count: 5 },
                    ],
                    you: {
                        canDelete: true,
                        canReact: true,
                        canReply: true,
                        canReport: false,
                        canUpdate: true,
                        reaction: "👍",
                    },
                },
            }),
        ];
    }

    /**
     * Create search result response
     */
    createSearchResultResponse(messages: ChatMessage[], hasNextPage = false) {
        return this.createSuccessResponse({
            __typename: "ChatMessageSearchResult",
            edges: messages.map((message, index) => ({
                __typename: "ChatMessageEdge" as const,
                cursor: `cursor_${index}`,
                node: message,
            })),
            pageInfo: {
                __typename: "PageInfo",
                hasNextPage,
                hasPreviousPage: false,
                startCursor: messages.length > 0 ? "cursor_0" : null,
                endCursor: messages.length > 0 ? `cursor_${messages.length - 1}` : null,
            },
        });
    }

    /**
     * Create tree result response
     */
    createTreeResultResponse(messages: ChatMessage[], hasMoreUp = false, hasMoreDown = false) {
        return this.createSuccessResponse({
            __typename: "ChatMessageSearchTreeResult",
            messages,
            hasMoreUp,
            hasMoreDown,
        });
    }

    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse() {
        return this.createBusinessErrorResponse("rate_limit", {
            resource: "chat_message",
            limit: RATE_LIMIT_PER_HOUR,
            window: "1 hour",
            retryAfter: 3600, // 1 hour
            message: `Rate limit exceeded. Maximum ${RATE_LIMIT_PER_HOUR} messages per hour.`,
        });
    }

    /**
     * Create message too long error response
     */
    createMessageTooLongErrorResponse(length: number) {
        return this.createValidationErrorResponse({
            text: `Message is ${length} characters. Maximum allowed is ${MAX_MESSAGE_LENGTH} characters.`,
        });
    }

    /**
     * Create parent not found error response
     */
    createParentNotFoundErrorResponse(parentId: string) {
        return this.createBusinessErrorResponse("parent_not_found", {
            resource: "chat_message",
            parentId,
            message: `Parent message with ID ${parentId} not found or not accessible`,
        });
    }

    /**
     * Create chat closed error response
     */
    createChatClosedErrorResponse(chatId: string) {
        return this.createBusinessErrorResponse("chat_closed", {
            resource: "chat_message",
            chatId,
            message: "Cannot send messages to a closed chat",
        });
    }
}

/**
 * Pre-configured chat message response scenarios
 */
export const chatMessageResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ChatMessageCreateInput>) => {
        const factory = new ChatMessageResponseFactory();
        const defaultInput: ChatMessageCreateInput = {
            text: "Hello, this is a test message!",
            chatConnect: generatePK().toString(),
            userConnect: generatePK().toString(),
            language: "en",
            config: {
                __version: "1.0.0",
                role: "user",
                resources: [],
            },
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (chatMessage?: ChatMessage) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createSuccessResponse(
            chatMessage || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: ChatMessage, updates?: Partial<ChatMessageUpdateInput>) => {
        const factory = new ChatMessageResponseFactory();
        const message = existing || factory.createMockData({ scenario: "complete" });
        const input: ChatMessageUpdateInput = {
            id: message.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(message, input),
        );
    },

    listSuccess: (chatMessages?: ChatMessage[]) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createPaginatedResponse(
            chatMessages || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: chatMessages?.length || DEFAULT_COUNT },
        );
    },

    allRolesSuccess: () => {
        const factory = new ChatMessageResponseFactory();
        const messages = factory.createMessagesForAllRoles();
        return factory.createPaginatedResponse(
            messages,
            { page: 1, totalCount: messages.length },
        );
    },

    conversationSuccess: (count?: number) => {
        const factory = new ChatMessageResponseFactory();
        const messages = factory.createConversationThread(count);
        return factory.createPaginatedResponse(
            messages,
            { page: 1, totalCount: messages.length },
        );
    },

    threadedSuccess: () => {
        const factory = new ChatMessageResponseFactory();
        const messages = factory.createThreadedMessages();
        return factory.createPaginatedResponse(
            messages,
            { page: 1, totalCount: messages.length },
        );
    },

    variedContentSuccess: () => {
        const factory = new ChatMessageResponseFactory();
        const messages = factory.createMessagesWithVariedContent();
        return factory.createPaginatedResponse(
            messages,
            { page: 1, totalCount: messages.length },
        );
    },

    searchSuccess: (messages?: ChatMessage[], hasNextPage = false) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createSearchResultResponse(
            messages || factory.createConversationThread(),
            hasNextPage,
        );
    },

    treeSuccess: (messages?: ChatMessage[], hasMoreUp = false, hasMoreDown = false) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createTreeResultResponse(
            messages || factory.createThreadedMessages(),
            hasMoreUp,
            hasMoreDown,
        );
    },

    assistantResponseSuccess: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    text: "I'm here to help! What would you like to know?",
                    config: {
                        __version: "1.0.0",
                        role: "assistant",
                        resources: [],
                    },
                    user: userResponseFactory.createMockData({
                        overrides: {
                            isBot: true,
                            name: "AI Assistant",
                            handle: "assistant",
                        },
                    }),
                },
            }),
        );
    },

    toolCallSuccess: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    text: "Let me search for that information.",
                    config: {
                        __version: "1.0.0",
                        role: "assistant",
                        resources: [],
                        toolCalls: [{
                            id: "search_call",
                            function: {
                                name: "web_search",
                                arguments: JSON.stringify({ query: "test query" }),
                            },
                            result: {
                                success: true,
                                output: "Search completed successfully",
                            },
                        }],
                    },
                    user: userResponseFactory.createMockData({
                        overrides: {
                            isBot: true,
                            name: "AI Assistant",
                            handle: "assistant",
                        },
                    }),
                },
            }),
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createValidationErrorResponse({
            text: "Message text is required",
            chatConnect: "Chat ID is required",
            userConnect: "User ID is required",
        });
    },

    notFoundError: (messageId?: string) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createNotFoundErrorResponse(
            messageId || "non-existent-message",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["chat:write"],
        );
    },

    rateLimitError: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createRateLimitErrorResponse();
    },

    messageTooLongError: (length = MAX_MESSAGE_LENGTH + 1) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createMessageTooLongErrorResponse(length);
    },

    parentNotFoundError: (parentId?: string) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createParentNotFoundErrorResponse(parentId || generatePK().toString());
    },

    chatClosedError: (chatId?: string) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createChatClosedErrorResponse(chatId || generatePK().toString());
    },

    emptyMessageError: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createValidationErrorResponse({
            text: "Message text cannot be empty",
        });
    },

    invalidLanguageError: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createValidationErrorResponse({
            language: "Language must be a valid language code (e.g., 'en', 'es-ES')",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ChatMessageResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ChatMessageResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ChatMessageResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const chatMessageResponseFactory = new ChatMessageResponseFactory();
