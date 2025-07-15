/**
 * Chat-specific event fixtures for testing real-time messaging features
 * Enhanced with factory pattern for dynamic event creation
 */

import { type ChatMessage, type ChatParticipant, type User } from "../../../api/types.js";
import { type ChatSocketEventPayloads, type StreamErrorPayload } from "../../../consts/socketEvents.js";
import { chatMessageFixtures } from "../api-inputs/chatMessageFixtures.js";
import { userFixtures } from "../api-inputs/userFixtures.js";
import { BaseEventFactory } from "./BaseEventFactory.js";
import { type BaseFixtureEvent, type CorrelatedEvent, type StatefulEvent, type TimedEvent } from "./types.js";

// Type helpers for chat events
type ChatEvent<TPayload> = BaseFixtureEvent & {
    event: keyof ChatSocketEventPayloads;
    data: TPayload;
};

/**
 * Factory for message CRUD operation events
 */
class MessageEventFactory extends BaseEventFactory<ChatEvent<ChatSocketEventPayloads["messages"]>, ChatSocketEventPayloads["messages"]> {
    constructor() {
        super("messages", {
            defaults: {
                added: [{
                    __typename: "ChatMessage" as const,
                    id: "msg_default",
                    text: "Default message",
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    chat: {} as any, // Will be populated by server
                    user: {} as any, // Will be populated by server
                    language: "en",
                    score: 0,
                    sequence: 0,
                    versionIndex: 0,
                    reactionSummaries: [],
                    reports: [],
                    reportsCount: 0,
                    config: { __version: "1.0.0", resources: [] },
                    you: { __typename: "ChatMessageYou" as const, canDelete: true, canUpdate: true, canReply: true, canReport: true, canReact: true, reaction: null },
                }],
            },
            validation: (data) => {
                if (!data.added && !data.updated && !data.removed) {
                    return "Message event must have at least one operation";
                }
                return true;
            },
        });
    }

    // Helper to create a ChatMessage from ChatMessageCreateInput
    createChatMessage(input: any): ChatMessage {
        return {
            __typename: "ChatMessage" as const,
            id: input.id || `msg_${Date.now()}`,
            text: input.text || "",
            createdAt: input.createdAt || new Date().toISOString(),
            updatedAt: input.updatedAt || new Date().toISOString(),
            chat: input.chat || {} as any,
            user: input.user || {} as any,
            language: input.language || "en",
            score: input.score || 0,
            sequence: input.sequence || 0,
            versionIndex: input.versionIndex || 0,
            reactionSummaries: input.reactionSummaries || [],
            reports: input.reports || [],
            reportsCount: input.reportsCount || 0,
            config: input.config || { __version: "1.0.0", resources: [] },
            parent: input.parent,
            you: input.you || { __typename: "ChatMessageYou" as const, canDelete: true, canUpdate: true, canReply: true, canReport: true, canReact: true, reaction: null },
        };
    }

    single = {
        event: "messages" as const,
        data: {
            added: [this.createChatMessage(chatMessageFixtures.minimal.create)],
        },
    };

    sequence = [
        {
            event: "messages" as const,
            data: { added: [this.createChatMessage({ ...chatMessageFixtures.minimal.create, id: "msg_1", text: "First message" })] },
        },
        {
            event: "messages" as const,
            data: { added: [this.createChatMessage({ ...chatMessageFixtures.minimal.create, id: "msg_2", text: "Second message" })] },
        },
        {
            event: "messages" as const,
            data: { added: [this.createChatMessage({ ...chatMessageFixtures.minimal.create, id: "msg_3", text: "Third message" })] },
        },
    ];

    variants = {
        textMessage: {
            event: "messages" as const,
            data: {
                added: [this.createChatMessage(chatMessageFixtures.minimal.create)],
            },
        },
        messageWithAttachment: {
            event: "messages" as const,
            data: {
                added: [this.createChatMessage({
                    ...chatMessageFixtures.minimal.create,
                    id: "msg_with_attachment",
                    text: "Check out this file",
                    attachments: [{
                        id: "attach_123",
                        name: "document.pdf",
                        size: 1024000,
                        type: "application/pdf",
                        url: "https://example.com/files/document.pdf",
                    }],
                })],
            },
        },
        bulkMessages: {
            event: "messages" as const,
            data: {
                added: [
                    this.createChatMessage(chatMessageFixtures.minimal.create),
                    this.createChatMessage({ ...chatMessageFixtures.minimal.create, id: "msg_2", text: "Second message" }),
                    this.createChatMessage({ ...chatMessageFixtures.minimal.create, id: "msg_3", text: "Third message" }),
                ],
            },
        },
        messageUpdate: {
            event: "messages" as const,
            data: {
                updated: [{
                    id: "msg_123",
                    text: "Updated message content",
                    updatedAt: new Date().toISOString(),
                }],
            },
        },
        messageDelete: {
            event: "messages" as const,
            data: {
                removed: ["msg_123", "msg_456"],
            },
        },
        mixedOperations: {
            event: "messages" as const,
            data: {
                added: [this.createChatMessage(chatMessageFixtures.minimal.create)],
                updated: [{ id: "msg_789", text: "Edited" }],
                removed: ["msg_old"],
            },
        },
    };

    // Additional factory methods
    createConversation(messages: Array<{ userId: string; text: string; delay?: number }>): TimedEvent<ChatSocketEventPayloads["messages"]>[] {
        return messages.map((msg, index) => {
            const messageData = this.createChatMessage({
                ...chatMessageFixtures.minimal.create,
                id: `msg_conv_${index}`,
                user: { __typename: "User", id: msg.userId, handle: `user_${msg.userId}`, name: `User ${msg.userId}` } as User,
                text: msg.text,
                createdAt: new Date(Date.now() + (msg.delay || 0)).toISOString(),
            });

            return this.withDelay(
                this.create({ added: [messageData] }),
                msg.delay || index * 1000,
            );
        });
    }

    createSystemMessage(content: string, type?: "info" | "warning" | "error"): ChatEvent<ChatSocketEventPayloads["messages"]> {
        return this.create({
            added: [this.createChatMessage({
                ...chatMessageFixtures.minimal.create,
                id: `msg_system_${Date.now()}`,
                text: content,
                user: { __typename: "User", id: "system", handle: "system", name: "System" } as User,
                config: {
                    __version: "1.0.0",
                    resources: [],
                    metadata: type ? { type, isSystem: true } : { isSystem: true },
                },
            })],
        });
    }

    createBulkDelete(messageIds: string[]): ChatEvent<ChatSocketEventPayloads["messages"]> {
        return this.create({
            removed: messageIds,
        });
    }
}

/**
 * Factory for AI response streaming events
 */
class ResponseStreamEventFactory extends BaseEventFactory<ChatEvent<ChatSocketEventPayloads["responseStream"]>, ChatSocketEventPayloads["responseStream"]> {
    constructor() {
        super("responseStream", {
            defaults: {
                __type: "stream",
                botId: "bot_default",
                chunk: "",
            },
        });
    }

    single = {
        event: "responseStream" as const,
        data: {
            __type: "stream" as const,
            botId: "bot_123",
            chunk: "Hello",
        },
    };

    sequence = [
        {
            event: "responseStream" as const,
            data: { __type: "stream" as const, botId: "bot_123", chunk: "I'm thinking about" },
        },
        {
            event: "responseStream" as const,
            data: { __type: "stream" as const, botId: "bot_123", chunk: " your question..." },
        },
        {
            event: "responseStream" as const,
            data: { __type: "end" as const, botId: "bot_123", finalMessage: "I'm thinking about your question... Here's my answer." },
        },
    ];

    variants = {
        streamStart: {
            event: "responseStream" as const,
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: "I'm thinking about",
            },
        },
        streamChunk: {
            event: "responseStream" as const,
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: " your question...",
            },
        },
        streamEnd: {
            event: "responseStream" as const,
            data: {
                __type: "end" as const,
                botId: "bot_123",
                finalMessage: "I'm thinking about your question... Here's my answer.",
            },
        },
        streamError: {
            event: "responseStream" as const,
            data: {
                __type: "error" as const,
                botId: "bot_123",
                error: {
                    message: "Failed to generate response",
                    code: "LLM_ERROR",
                    details: "Model timeout after 30 seconds",
                    retryable: true,
                },
            },
        },
    };

    // Additional factory methods
    createStreamSequence(botId: string, text: string, chunkSize = 20, delay = 100): TimedEvent<ChatSocketEventPayloads["responseStream"]>[] {
        const chunks: TimedEvent<ChatSocketEventPayloads["responseStream"]>[] = [];

        // Split text into chunks
        for (let i = 0; i < text.length; i += chunkSize) {
            const chunk = text.slice(i, i + chunkSize);
            chunks.push(
                this.withDelay(
                    this.create({ __type: "stream", botId, chunk }),
                    (chunks.length) * delay,
                ),
            );
        }

        // Add end event
        chunks.push(
            this.withDelay(
                this.create({ __type: "end", botId, finalMessage: text }),
                chunks.length * delay,
            ),
        );

        return chunks;
    }

    createErrorStream(botId: string, error: StreamErrorPayload): ChatEvent<ChatSocketEventPayloads["responseStream"]> {
        return this.create({
            __type: "error",
            botId,
            error,
        });
    }

    createTypingStream(botId: string, duration = 3000): TimedEvent<ChatSocketEventPayloads["responseStream"]>[] {
        const dots = [".", "..", "...", ""];
        return dots.map((dot, index) =>
            this.withDelay(
                this.create({ __type: "stream", botId, chunk: `Typing${dot}` }),
                (duration / dots.length) * index,
            ),
        );
    }
}

/**
 * Factory for model reasoning stream events
 */
class ModelReasoningEventFactory extends BaseEventFactory<ChatEvent<ChatSocketEventPayloads["modelReasoningStream"]>, ChatSocketEventPayloads["modelReasoningStream"]> {
    constructor() {
        super("modelReasoningStream", {
            defaults: {
                __type: "stream",
                botId: "bot_default",
                chunk: "",
            },
        });
    }

    single = {
        event: "modelReasoningStream" as const,
        data: {
            __type: "stream" as const,
            botId: "bot_123",
            chunk: "Analyzing the request...",
        },
    };

    sequence = [
        {
            event: "modelReasoningStream" as const,
            data: { __type: "stream" as const, botId: "bot_123", chunk: "Let me analyze this step by step:\n1. First," },
        },
        {
            event: "modelReasoningStream" as const,
            data: { __type: "stream" as const, botId: "bot_123", chunk: " I need to understand the context..." },
        },
        {
            event: "modelReasoningStream" as const,
            data: { __type: "end" as const, botId: "bot_123" },
        },
    ];

    variants = {
        reasoningStart: {
            event: "modelReasoningStream" as const,
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: "Let me analyze this step by step:\n1. First,",
            },
        },
        reasoningChunk: {
            event: "modelReasoningStream" as const,
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: " I need to understand the context...",
            },
        },
        reasoningEnd: {
            event: "modelReasoningStream" as const,
            data: {
                __type: "end" as const,
                botId: "bot_123",
            },
        },
        reasoningError: {
            event: "modelReasoningStream" as const,
            data: {
                __type: "error" as const,
                botId: "bot_123",
                error: {
                    message: "Reasoning generation failed",
                    code: "REASONING_ERROR",
                },
            },
        },
    };

    // Additional factory methods
    createStepByStepReasoning(botId: string, steps: string[]): TimedEvent<ChatSocketEventPayloads["modelReasoningStream"]>[] {
        const events: TimedEvent<ChatSocketEventPayloads["modelReasoningStream"]>[] = [];
        let delay = 0;

        steps.forEach((step, index) => {
            const formattedStep = `${index + 1}. ${step}\n`;
            events.push(
                this.withDelay(
                    this.create({ __type: "stream", botId, chunk: formattedStep }),
                    delay,
                ),
            );
            delay += 500 + Math.random() * 1000; // Variable delay for realism
        });

        events.push(
            this.withDelay(
                this.create({ __type: "end", botId }),
                delay,
            ),
        );

        return events;
    }
}

/**
 * Factory for typing indicator events
 */
class TypingEventFactory extends BaseEventFactory<ChatEvent<ChatSocketEventPayloads["typing"]>, ChatSocketEventPayloads["typing"]> {
    constructor() {
        super("typing", {
            defaults: {
                starting: [],
                stopping: [],
            },
        });
    }

    single = {
        event: "typing" as const,
        data: {
            starting: ["user_123"],
        },
    };

    sequence = [
        {
            event: "typing" as const,
            data: { starting: ["user_123"] },
        },
        {
            event: "typing" as const,
            data: { stopping: ["user_123"] },
        },
    ];

    variants = {
        userStartTyping: {
            event: "typing" as const,
            data: {
                starting: ["user_123"],
            },
        },
        multipleUsersTyping: {
            event: "typing" as const,
            data: {
                starting: ["user_123", "user_456"],
            },
        },
        userStopTyping: {
            event: "typing" as const,
            data: {
                stopping: ["user_123"],
            },
        },
        mixedTyping: {
            event: "typing" as const,
            data: {
                starting: ["user_789"],
                stopping: ["user_123", "user_456"],
            },
        },
    };

    // Additional factory methods
    createTypingSession(userId: string, duration = 3000): TimedEvent<ChatSocketEventPayloads["typing"]>[] {
        return [
            this.withDelay(
                this.create({ starting: [userId] }),
                0,
            ),
            this.withDelay(
                this.create({ stopping: [userId] }),
                duration,
            ),
        ];
    }

    createMultiUserTyping(userIds: string[], pattern: "sequential" | "overlapping" | "simultaneous" = "overlapping"): TimedEvent<ChatSocketEventPayloads["typing"]>[] {
        const events: TimedEvent<ChatSocketEventPayloads["typing"]>[] = [];

        switch (pattern) {
            case "sequential":
                userIds.forEach((userId, index) => {
                    const startDelay = index * 3000;
                    events.push(
                        this.withDelay(this.create({ starting: [userId] }), startDelay),
                        this.withDelay(this.create({ stopping: [userId] }), startDelay + 2000),
                    );
                });
                break;

            case "overlapping":
                userIds.forEach((userId, index) => {
                    const startDelay = index * 1000;
                    events.push(
                        this.withDelay(this.create({ starting: [userId] }), startDelay),
                        this.withDelay(this.create({ stopping: [userId] }), startDelay + 3000),
                    );
                });
                break;

            case "simultaneous":
                events.push(
                    this.withDelay(this.create({ starting: userIds }), 0),
                    this.withDelay(this.create({ stopping: userIds }), 3000),
                );
                break;
        }

        return events;
    }
}

/**
 * Factory for participant change events
 */
class ParticipantEventFactory extends BaseEventFactory<ChatEvent<ChatSocketEventPayloads["participants"]>, ChatSocketEventPayloads["participants"]> {
    constructor() {
        super("participants", {
            defaults: {
                joining: [],
                leaving: [],
            },
        });
    }

    // Helper to create a participant object without chat reference
    createParticipant(userId: string, participantId: string, userName?: string): Omit<ChatParticipant, "chat"> {
        return {
            __typename: "ChatParticipant",
            id: participantId,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            user: {
                __typename: "User",
                id: userId,
                handle: userName || `user_${userId}`,
                name: userName || `User ${userId}`,
                ...userFixtures.minimal.create,
            } as User,
        } as Omit<ChatParticipant, "chat">;
    }

    single = {
        event: "participants" as const,
        data: {
            joining: [this.createParticipant("user_123", "participant_123")],
        },
    };

    sequence = [
        {
            event: "participants" as const,
            data: { joining: [this.createParticipant("user_123", "participant_123")] },
        },
        {
            event: "participants" as const,
            data: { leaving: ["user_123"] },
        },
    ];

    variants = {
        userJoining: {
            event: "participants" as const,
            data: {
                joining: [this.createParticipant("user_123", "participant_123")],
            },
        },
        multipleUsersJoining: {
            event: "participants" as const,
            data: {
                joining: [
                    this.createParticipant("user_123", "participant_123"),
                    this.createParticipant("user_456", "participant_456", "Another User"),
                ],
            },
        },
        userLeaving: {
            event: "participants" as const,
            data: {
                leaving: ["user_123"],
            },
        },
        mixedParticipants: {
            event: "participants" as const,
            data: {
                joining: [this.createParticipant("user_new", "participant_new")],
                leaving: ["user_456", "user_789"],
            },
        },
    };

    // Additional factory methods
    createMassJoin(count: number): ChatEvent<ChatSocketEventPayloads["participants"]> {
        const participants: Omit<ChatParticipant, "chat">[] = [];
        for (let i = 0; i < count; i++) {
            participants.push(this.createParticipant(`user_${i}`, `participant_${i}`, `User ${i}`));
        }
        return this.create({ joining: participants });
    }

    createMassLeave(userIds: string[]): ChatEvent<ChatSocketEventPayloads["participants"]> {
        return this.create({ leaving: userIds });
    }

    createChurn(joinCount: number, leaveCount: number): ChatEvent<ChatSocketEventPayloads["participants"]> {
        const joining: Omit<ChatParticipant, "chat">[] = [];
        const leaving: string[] = [];

        for (let i = 0; i < joinCount; i++) {
            joining.push(this.createParticipant(`user_join_${i}`, `participant_join_${i}`, `New User ${i}`));
        }

        for (let i = 0; i < leaveCount; i++) {
            leaving.push(`user_leave_${i}`);
        }

        return this.create({ joining, leaving });
    }
}

/**
 * Factory for bot status update events
 */
class BotStatusEventFactory extends BaseEventFactory<ChatEvent<ChatSocketEventPayloads["botStatusUpdate"]>, ChatSocketEventPayloads["botStatusUpdate"]> {
    constructor() {
        super("botStatusUpdate", {
            defaults: {
                chatId: "chat_default",
                botId: "bot_default",
                status: "thinking",
            },
        });
    }

    single = {
        event: "botStatusUpdate" as const,
        data: {
            chatId: "chat_123",
            botId: "bot_123",
            status: "thinking" as const,
            message: "Processing your request...",
        },
    };

    sequence = [
        {
            event: "botStatusUpdate" as const,
            data: { chatId: "chat_123", botId: "bot_123", status: "thinking" as const },
        },
        {
            event: "botStatusUpdate" as const,
            data: { chatId: "chat_123", botId: "bot_123", status: "tool_calling" as const },
        },
        {
            event: "botStatusUpdate" as const,
            data: { chatId: "chat_123", botId: "bot_123", status: "processing_complete" as const },
        },
    ];

    variants = {
        thinking: {
            event: "botStatusUpdate" as const,
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "thinking" as const,
                message: "Processing your request...",
            },
        },
        toolCalling: {
            event: "botStatusUpdate" as const,
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "tool_calling" as const,
                message: "Searching for information...",
                toolInfo: {
                    callId: "call_123",
                    name: "search",
                    args: JSON.stringify({ query: "AI research papers" }),
                },
            },
        },
        toolCompleted: {
            event: "botStatusUpdate" as const,
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "tool_completed" as const,
                toolInfo: {
                    callId: "call_123",
                    name: "search",
                    result: JSON.stringify({ results: ["Paper 1", "Paper 2"] }),
                },
            },
        },
        toolFailed: {
            event: "botStatusUpdate" as const,
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "tool_failed" as const,
                toolInfo: {
                    callId: "call_123",
                    name: "search",
                    error: "Search service unavailable",
                },
            },
        },
        processingComplete: {
            event: "botStatusUpdate" as const,
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "processing_complete" as const,
            },
        },
        internalError: {
            event: "botStatusUpdate" as const,
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "error_internal" as const,
                error: {
                    message: "An unexpected error occurred",
                    code: "INTERNAL_ERROR",
                    retryable: false,
                },
            },
        },
    };

    // Additional factory methods
    createToolFlow(chatId: string, botId: string, toolName: string, args: Record<string, unknown>, result?: unknown, error?: string): TimedEvent<ChatSocketEventPayloads["botStatusUpdate"]>[] {
        const callId = `call_${Date.now()}`;
        const events: TimedEvent<ChatSocketEventPayloads["botStatusUpdate"]>[] = [];

        // Start thinking
        events.push(
            this.withDelay(
                this.create({ chatId, botId, status: "thinking", message: "Analyzing request..." }),
                0,
            ),
        );

        // Tool calling
        events.push(
            this.withDelay(
                this.create({
                    chatId,
                    botId,
                    status: "tool_calling",
                    message: `Using ${toolName}...`,
                    toolInfo: { callId, name: toolName, args: JSON.stringify(args) },
                }),
                1000,
            ),
        );

        // Tool result or error
        if (error) {
            events.push(
                this.withDelay(
                    this.create({
                        chatId,
                        botId,
                        status: "tool_failed",
                        toolInfo: { callId, name: toolName, error },
                    }),
                    3000,
                ),
            );
        } else {
            events.push(
                this.withDelay(
                    this.create({
                        chatId,
                        botId,
                        status: "tool_completed",
                        toolInfo: { callId, name: toolName, result: JSON.stringify(result || {}) },
                    }),
                    3000,
                ),
            );
        }

        // Processing complete
        events.push(
            this.withDelay(
                this.create({ chatId, botId, status: "processing_complete" }),
                4000,
            ),
        );

        return events;
    }

    createErrorFlow(chatId: string, botId: string, error: StreamErrorPayload): ChatEvent<ChatSocketEventPayloads["botStatusUpdate"]>[] {
        return [
            this.create({ chatId, botId, status: "thinking" }),
            this.create({ chatId, botId, status: "error_internal", error }),
        ];
    }
}

/**
 * Factory for tool approval flow events
 */
class ToolApprovalEventFactory extends BaseEventFactory<
    ChatEvent<ChatSocketEventPayloads["tool_approval_required"] | ChatSocketEventPayloads["tool_approval_rejected"]>,
    ChatSocketEventPayloads["tool_approval_required"] | ChatSocketEventPayloads["tool_approval_rejected"]
> {
    constructor() {
        super("tool_approval", {
            defaults: {
                pendingId: "pending_default",
                toolCallId: "call_default",
                toolName: "default_tool",
                callerBotId: "bot_default",
            } as Partial<ChatSocketEventPayloads["tool_approval_required"]>,
        });
    }

    single = {
        event: "tool_approval_required" as const,
        data: {
            pendingId: "pending_123",
            toolCallId: "call_456",
            toolName: "execute_code",
            toolArguments: {
                language: "python",
                code: "print('Hello, World!')",
            },
            callerBotId: "bot_123",
            callerBotName: "Assistant",
            approvalTimeoutAt: Date.now() + 60000,
            estimatedCost: "0.001",
        } as ChatSocketEventPayloads["tool_approval_required"],
    };

    sequence = [
        {
            event: "tool_approval_required" as const,
            data: {
                pendingId: "pending_123",
                toolCallId: "call_456",
                toolName: "execute_code",
                toolArguments: { language: "python", code: "print('Hello')" },
                callerBotId: "bot_123",
            } as ChatSocketEventPayloads["tool_approval_required"],
        },
        {
            event: "tool_approval_rejected" as const,
            data: {
                pendingId: "pending_123",
                toolCallId: "call_456",
                toolName: "execute_code",
                reason: "User declined",
                callerBotId: "bot_123",
            } as ChatSocketEventPayloads["tool_approval_rejected"],
        },
    ];

    variants = {
        required: {
            event: "tool_approval_required" as const,
            data: {
                pendingId: "pending_123",
                toolCallId: "call_456",
                toolName: "execute_code",
                toolArguments: {
                    language: "python",
                    code: "print('Hello, World!')",
                },
                callerBotId: "bot_123",
                callerBotName: "Assistant",
                approvalTimeoutAt: Date.now() + 60000,
                estimatedCost: "0.001",
            } as ChatSocketEventPayloads["tool_approval_required"],
        },
        rejected: {
            event: "tool_approval_rejected" as const,
            data: {
                pendingId: "pending_123",
                toolCallId: "call_456",
                toolName: "execute_code",
                reason: "User declined code execution",
                callerBotId: "bot_123",
            } as ChatSocketEventPayloads["tool_approval_rejected"],
        },
    };

    // Additional factory methods
    createApprovalFlow(
        pendingId: string,
        toolCallId: string,
        toolName: string,
        toolArguments: Record<string, unknown>,
        approved: boolean,
        botId = "bot_123",
    ): TimedEvent<unknown>[] {
        const events: TimedEvent<unknown>[] = [];

        // Approval required event
        const requiredEvent = {
            event: "tool_approval_required" as const,
            data: {
                pendingId,
                toolCallId,
                toolName,
                toolArguments,
                callerBotId: botId,
                approvalTimeoutAt: Date.now() + 60000,
            } as ChatSocketEventPayloads["tool_approval_required"],
        };

        events.push({
            event: requiredEvent.event,
            data: requiredEvent.data,
            timing: {
                delay: 0,
                timestamp: Date.now(),
            },
        });

        // User decision (after delay)
        if (!approved) {
            const rejectedEvent = {
                event: "tool_approval_rejected" as const,
                data: {
                    pendingId,
                    toolCallId,
                    toolName,
                    reason: "User declined tool execution",
                    callerBotId: botId,
                } as ChatSocketEventPayloads["tool_approval_rejected"],
            };

            events.push({
                event: rejectedEvent.event,
                data: rejectedEvent.data,
                timing: {
                    delay: 5000,
                    timestamp: Date.now() + 5000,
                },
            });
        }

        return events;
    }

    createHighCostApproval(toolName: string, estimatedCost: string): ChatEvent<ChatSocketEventPayloads["tool_approval_required"]> {
        return {
            event: "tool_approval_required",
            data: {
                pendingId: `pending_${Date.now()}`,
                toolCallId: `call_${Date.now()}`,
                toolName,
                toolArguments: { operation: "expensive_operation" },
                callerBotId: "bot_123",
                callerBotName: "Assistant",
                approvalTimeoutAt: Date.now() + 120000, // 2 minute timeout for high cost
                estimatedCost,
            },
        };
    }
}

// Create factory instances
const messageEventFactory = new MessageEventFactory();
const responseStreamEventFactory = new ResponseStreamEventFactory();
const modelReasoningEventFactory = new ModelReasoningEventFactory();
const typingEventFactory = new TypingEventFactory();
const participantEventFactory = new ParticipantEventFactory();
const botStatusEventFactory = new BotStatusEventFactory();
const toolApprovalEventFactory = new ToolApprovalEventFactory();

// Enhanced sequences for complex chat flows
const enhancedSequences = {
    // Complete conversation flow with multiple participants
    fullConversation: () => {
        const correlationId = `conv_${Date.now()}`;
        const events: CorrelatedEvent<unknown>[] = [];

        // User 1 joins
        events.push(...participantEventFactory.createCorrelated(correlationId, [
            participantEventFactory.createMassJoin(1),
        ]));

        // User 1 types and sends message
        const typingSession = typingEventFactory.createTypingSession("user_1", 2000);
        events.push(...typingEventFactory.createCorrelated(correlationId,
            typingSession.map(timedEvent => ({ event: timedEvent.event, data: timedEvent.data } as ChatEvent<ChatSocketEventPayloads["typing"]>)),
        ));

        events.push(...messageEventFactory.createCorrelated(correlationId, [
            messageEventFactory.create({
                added: [messageEventFactory.createChatMessage({
                    ...chatMessageFixtures.minimal.create,
                    id: "msg_user1_1",
                    user: { __typename: "User", id: "user_1", handle: "user_1", name: "User 1" } as User,
                    text: "Hello, I need help with a coding problem",
                })],
            }),
        ]));

        // Bot responds with reasoning
        events.push(...botStatusEventFactory.createCorrelated(correlationId, [
            botStatusEventFactory.create({
                chatId: "chat_123",
                botId: "bot_123",
                status: "thinking",
                message: "Analyzing your request...",
            }),
        ]));

        // Bot reasoning stream
        const reasoningSteps = modelReasoningEventFactory.createStepByStepReasoning("bot_123", [
            "Understanding the user's coding problem",
            "Identifying the best approach to help",
            "Preparing a detailed response",
        ]);
        events.push(...modelReasoningEventFactory.createCorrelated(correlationId,
            reasoningSteps.map(timedEvent => ({ event: timedEvent.event, data: timedEvent.data } as ChatEvent<ChatSocketEventPayloads["modelReasoningStream"]>)),
        ));

        // Bot response stream
        const responseStream = responseStreamEventFactory.createStreamSequence(
            "bot_123",
            "I'd be happy to help with your coding problem. Could you please provide more details about what you're trying to achieve?",
            30,
            150,
        );
        events.push(...responseStreamEventFactory.createCorrelated(correlationId,
            responseStream.map(timedEvent => ({ event: timedEvent.event, data: timedEvent.data } as ChatEvent<ChatSocketEventPayloads["responseStream"]>)),
        ));

        return events;
    },

    // Bot interaction with tool usage
    botWithTools: () => {
        const events: TimedEvent<unknown>[] = [];
        let currentDelay = 0;

        // User message
        events.push(
            messageEventFactory.withDelay(
                messageEventFactory.create({
                    added: [messageEventFactory.createChatMessage({
                        ...chatMessageFixtures.minimal.create,
                        user: { __typename: "User", id: "user_123", handle: "user_123", name: "User 123" } as User,
                        text: "Can you search for the latest AI research papers?",
                    })],
                }),
                currentDelay,
            ),
        );
        currentDelay += 500;

        // Bot starts processing with tool flow
        const toolFlow = botStatusEventFactory.createToolFlow(
            "chat_123",
            "bot_123",
            "search",
            { query: "latest AI research papers", limit: 10 },
            { results: ["Paper 1: GPT-4 Architecture", "Paper 2: Multimodal Learning"] },
        );

        toolFlow.forEach(event => {
            events.push({
                ...event,
                timing: { ...event.timing, delay: currentDelay + (event.timing?.delay || 0) },
            });
        });
        currentDelay += 5000;

        // Bot responds with results
        const response = responseStreamEventFactory.createStreamSequence(
            "bot_123",
            "I found 2 recent AI research papers:\n\n1. GPT-4 Architecture - This paper discusses...\n2. Multimodal Learning - This research explores...",
            40,
            100,
        );

        response.forEach(event => {
            events.push({
                ...event,
                timing: { ...event.timing, delay: currentDelay + (event.timing?.delay || 0) },
            });
        });

        return events;
    },

    // Tool approval workflow
    toolApprovalWorkflow: () => {
        const events: TimedEvent<unknown>[] = [];

        // User requests code execution
        events.push(
            messageEventFactory.withDelay(
                messageEventFactory.create({
                    added: [messageEventFactory.createChatMessage({
                        ...chatMessageFixtures.minimal.create,
                        user: { __typename: "User", id: "user_123", handle: "user_123", name: "User 123" } as User,
                        text: "Can you write and run a Python script to analyze this data?",
                    })],
                }),
                0,
            ),
        );

        // Bot requests approval
        const approvalFlow = toolApprovalEventFactory.createApprovalFlow(
            "pending_exec_123",
            "call_exec_456",
            "execute_code",
            {
                language: "python",
                code: "import pandas as pd\ndf = pd.read_csv('data.csv')\nprint(df.describe())",
            },
            false, // User rejects
            "bot_123",
        );

        approvalFlow.forEach((event, index) => {
            events.push({
                ...event,
                timing: { ...event.timing, delay: 1000 + (event.timing?.delay || 0) },
            });
        });

        // Bot adjusts approach
        events.push(
            botStatusEventFactory.withDelay(
                botStatusEventFactory.create({
                    chatId: "chat_123",
                    botId: "bot_123",
                    status: "thinking",
                    message: "I'll provide the code for you to run locally instead...",
                }),
                7000,
            ),
        );

        return events;
    },

    // Error and recovery scenario
    errorRecovery: () => {
        const events: StatefulEvent<unknown>[] = [];
        const state = { messageCount: 0, lastError: null as string | null, retryCount: 0 };

        // Initial message
        const msg1 = messageEventFactory.withState(
            messageEventFactory.create({
                added: [messageEventFactory.createChatMessage({
                    ...chatMessageFixtures.minimal.create,
                    text: "Generate a complex analysis",
                })],
            }),
            {
                before: { ...state },
                after: { ...state, messageCount: 1 },
            },
        );
        events.push(msg1);

        // First attempt - fails
        const error1 = responseStreamEventFactory.withState(
            responseStreamEventFactory.createErrorStream("bot_123", {
                message: "Model timeout",
                code: "TIMEOUT",
                retryable: true,
            }),
            {
                before: { ...state, messageCount: 1 },
                after: { ...state, messageCount: 1, lastError: "TIMEOUT", retryCount: 1 },
            },
        );
        events.push(error1);

        // Retry attempt - succeeds
        const success = responseStreamEventFactory.withState(
            responseStreamEventFactory.create({
                __type: "end",
                botId: "bot_123",
                finalMessage: "Analysis completed successfully after retry",
            }),
            {
                before: { ...state, messageCount: 1, lastError: "TIMEOUT", retryCount: 1 },
                after: { ...state, messageCount: 2, lastError: null, retryCount: 1 },
            },
        );
        events.push(success);

        return events;
    },

    // High-volume chat scenario
    highVolumeChat: () => {
        const events: TimedEvent<unknown>[] = [];

        // Multiple users join
        events.push(
            participantEventFactory.withDelay(
                participantEventFactory.createMassJoin(10),
                0,
            ),
        );

        // Rapid message exchange
        for (let i = 0; i < 20; i++) {
            const userId = `user_${i % 10}`;
            const delay = i * 200; // 200ms between messages

            // Some users type
            if (i % 3 === 0) {
                events.push(
                    typingEventFactory.withDelay(
                        typingEventFactory.create({ starting: [userId] }),
                        delay - 100,
                    ),
                );
            }

            // Send message
            events.push(
                messageEventFactory.withDelay(
                    messageEventFactory.create({
                        added: [messageEventFactory.createChatMessage({
                            ...chatMessageFixtures.minimal.create,
                            id: `msg_rapid_${i}`,
                            user: { __typename: "User", id: userId, handle: userId, name: userId } as User,
                            text: `Message ${i} from ${userId}`,
                        })],
                    }),
                    delay,
                ),
            );

            // Stop typing
            if (i % 3 === 0) {
                events.push(
                    typingEventFactory.withDelay(
                        typingEventFactory.create({ stopping: [userId] }),
                        delay + 50,
                    ),
                );
            }
        }

        // Some users leave
        events.push(
            participantEventFactory.withDelay(
                participantEventFactory.createMassLeave(["user_2", "user_5", "user_8"]),
                4500,
            ),
        );

        return events;
    },
};

// Export the enhanced chat event fixtures with backward compatibility
export const chatEventFixtures = {
    // Preserve original structure
    messages: messageEventFactory.variants,
    responseStream: responseStreamEventFactory.variants,
    modelReasoning: modelReasoningEventFactory.variants,
    typing: typingEventFactory.variants,
    participants: participantEventFactory.variants,
    botStatus: botStatusEventFactory.variants,
    toolApproval: toolApprovalEventFactory.variants,

    // Original sequences (preserved for backward compatibility)
    sequences: {
        messageFlow: [
            { event: "typing", data: typingEventFactory.variants.userStartTyping.data },
            { delay: 2000 },
            { event: "typing", data: typingEventFactory.variants.userStopTyping.data },
            { delay: 100 },
            { event: "messages", data: messageEventFactory.variants.textMessage.data },
        ],
        botResponseFlow: [
            { event: "botStatusUpdate", data: botStatusEventFactory.variants.thinking.data },
            { delay: 1000 },
            { event: "responseStream", data: responseStreamEventFactory.variants.streamStart.data },
            { delay: 500 },
            { event: "responseStream", data: responseStreamEventFactory.variants.streamChunk.data },
            { delay: 500 },
            { event: "responseStream", data: responseStreamEventFactory.variants.streamEnd.data },
            { event: "botStatusUpdate", data: botStatusEventFactory.variants.processingComplete.data },
        ],
        toolExecutionFlow: [
            { event: "botStatusUpdate", data: botStatusEventFactory.variants.thinking.data },
            { delay: 500 },
            { event: "botStatusUpdate", data: botStatusEventFactory.variants.toolCalling.data },
            { delay: 2000 },
            { event: "botStatusUpdate", data: botStatusEventFactory.variants.toolCompleted.data },
            { delay: 500 },
            { event: "responseStream", data: responseStreamEventFactory.variants.streamStart.data },
        ],
        toolApprovalFlow: [
            { event: "botStatusUpdate", data: botStatusEventFactory.variants.thinking.data },
            { delay: 500 },
            { event: "tool_approval_required", data: toolApprovalEventFactory.variants.required.data },
            { delay: 5000 },
            { event: "tool_approval_rejected", data: toolApprovalEventFactory.variants.rejected.data },
            { event: "botStatusUpdate", data: { ...botStatusEventFactory.variants.thinking.data, message: "Adjusting approach..." } },
        ],
        errorRecoveryFlow: [
            { event: "responseStream", data: responseStreamEventFactory.variants.streamStart.data },
            { delay: 1000 },
            { event: "responseStream", data: responseStreamEventFactory.variants.streamError.data },
            { delay: 2000 },
            { event: "botStatusUpdate", data: { ...botStatusEventFactory.variants.thinking.data, message: "Retrying..." } },
            { delay: 1000 },
            { event: "responseStream", data: responseStreamEventFactory.variants.streamStart.data },
        ],
        participantChangeFlow: [
            { event: "participants", data: participantEventFactory.variants.userJoining.data },
            { delay: 100 },
            { event: "messages", data: messageEventFactory.createSystemMessage("User joined the chat").data },
            { delay: 30000 },
            { event: "participants", data: participantEventFactory.variants.userLeaving.data },
            { delay: 100 },
            { event: "messages", data: messageEventFactory.createSystemMessage("User left the chat").data },
        ],
    },

    // Original factory functions (preserved for backward compatibility)
    factories: {
        createMessageEvent: (message: Partial<ChatMessage>) => messageEventFactory.create({ added: [messageEventFactory.createChatMessage({ ...chatMessageFixtures.minimal.create, ...message })] }),
        createStreamChunk: (botId: string, chunk: string) => responseStreamEventFactory.create({ __type: "stream", botId, chunk }),
        createTypingEvent: (userIds: string[], isTyping: boolean) => typingEventFactory.create(isTyping ? { starting: userIds } : { stopping: userIds }),
        createBotStatusEvent: (chatId: string, botId: string, status: ChatSocketEventPayloads["botStatusUpdate"]["status"], message?: string) =>
            botStatusEventFactory.create({ chatId, botId, status, message }),
    },

    // New factory instances for advanced usage
    eventFactories: {
        message: messageEventFactory,
        responseStream: responseStreamEventFactory,
        modelReasoning: modelReasoningEventFactory,
        typing: typingEventFactory,
        participant: participantEventFactory,
        botStatus: botStatusEventFactory,
        toolApproval: toolApprovalEventFactory,
    },

    // Enhanced sequences
    enhancedSequences,
};

// Export factory types for external use
export type {
    BotStatusEventFactory, MessageEventFactory, ModelReasoningEventFactory, ParticipantEventFactory, ResponseStreamEventFactory, ToolApprovalEventFactory, TypingEventFactory,
};

