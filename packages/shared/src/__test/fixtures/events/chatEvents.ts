/**
 * Chat-specific event fixtures for testing real-time messaging features
 */

import { type ChatMessage, type ChatParticipant } from "../../../api/types.js";
import { type ChatSocketEventPayloads, type StreamErrorPayload } from "../../../consts/socketEvents.js";
import { chatMessageFixtures } from "../api/chatMessageFixtures.js";
import { chatParticipantFixtures } from "../api/chatParticipantFixtures.js";
import { userFixtures } from "../api/userFixtures.js";

export const chatEventFixtures = {
    messages: {
        // Simple text message event
        textMessage: {
            event: "messages",
            data: {
                added: [chatMessageFixtures.minimal.find],
            } satisfies ChatSocketEventPayloads["messages"],
        },

        // Message with attachment
        messageWithAttachment: {
            event: "messages",
            data: {
                added: [{
                    ...chatMessageFixtures.minimal.find,
                    id: "msg_with_attachment",
                    content: "Check out this file",
                    attachments: [{
                        id: "attach_123",
                        name: "document.pdf",
                        size: 1024000,
                        type: "application/pdf",
                        url: "https://example.com/files/document.pdf",
                    }],
                }],
            } satisfies ChatSocketEventPayloads["messages"],
        },

        // Multiple messages at once
        bulkMessages: {
            event: "messages",
            data: {
                added: [
                    chatMessageFixtures.minimal.find,
                    { ...chatMessageFixtures.minimal.find, id: "msg_2", content: "Second message" },
                    { ...chatMessageFixtures.minimal.find, id: "msg_3", content: "Third message" },
                ],
            } satisfies ChatSocketEventPayloads["messages"],
        },

        // Message update
        messageUpdate: {
            event: "messages",
            data: {
                updated: [{
                    id: "msg_123",
                    content: "Updated message content",
                    updatedAt: new Date().toISOString(),
                }],
            } satisfies ChatSocketEventPayloads["messages"],
        },

        // Message deletion
        messageDelete: {
            event: "messages",
            data: {
                removed: ["msg_123", "msg_456"],
            } satisfies ChatSocketEventPayloads["messages"],
        },

        // Combined operations
        mixedOperations: {
            event: "messages",
            data: {
                added: [chatMessageFixtures.minimal.find],
                updated: [{ id: "msg_789", content: "Edited" }],
                removed: ["msg_old"],
            } satisfies ChatSocketEventPayloads["messages"],
        },
    },

    responseStream: {
        // Stream start
        streamStart: {
            event: "responseStream",
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: "I'm thinking about",
            } satisfies ChatSocketEventPayloads["responseStream"],
        },

        // Stream continuation
        streamChunk: {
            event: "responseStream",
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: " your question...",
            } satisfies ChatSocketEventPayloads["responseStream"],
        },

        // Stream end
        streamEnd: {
            event: "responseStream",
            data: {
                __type: "end" as const,
                botId: "bot_123",
                finalMessage: "I'm thinking about your question... Here's my answer.",
            } satisfies ChatSocketEventPayloads["responseStream"],
        },

        // Stream error
        streamError: {
            event: "responseStream",
            data: {
                __type: "error" as const,
                botId: "bot_123",
                error: {
                    message: "Failed to generate response",
                    code: "LLM_ERROR",
                    details: "Model timeout after 30 seconds",
                    retryable: true,
                } satisfies StreamErrorPayload,
            } satisfies ChatSocketEventPayloads["responseStream"],
        },
    },

    modelReasoning: {
        // Reasoning stream start
        reasoningStart: {
            event: "modelReasoningStream",
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: "Let me analyze this step by step:\n1. First,",
            } satisfies ChatSocketEventPayloads["modelReasoningStream"],
        },

        // Reasoning continuation
        reasoningChunk: {
            event: "modelReasoningStream",
            data: {
                __type: "stream" as const,
                botId: "bot_123",
                chunk: " I need to understand the context...",
            } satisfies ChatSocketEventPayloads["modelReasoningStream"],
        },

        // Reasoning end
        reasoningEnd: {
            event: "modelReasoningStream",
            data: {
                __type: "end" as const,
                botId: "bot_123",
            } satisfies ChatSocketEventPayloads["modelReasoningStream"],
        },

        // Reasoning error
        reasoningError: {
            event: "modelReasoningStream",
            data: {
                __type: "error" as const,
                botId: "bot_123",
                error: {
                    message: "Reasoning generation failed",
                    code: "REASONING_ERROR",
                } satisfies StreamErrorPayload,
            } satisfies ChatSocketEventPayloads["modelReasoningStream"],
        },
    },

    typing: {
        // Single user starts typing
        userStartTyping: {
            event: "typing",
            data: {
                starting: ["user_123"],
            } satisfies ChatSocketEventPayloads["typing"],
        },

        // Multiple users typing
        multipleUsersTyping: {
            event: "typing",
            data: {
                starting: ["user_123", "user_456"],
            } satisfies ChatSocketEventPayloads["typing"],
        },

        // User stops typing
        userStopTyping: {
            event: "typing",
            data: {
                stopping: ["user_123"],
            } satisfies ChatSocketEventPayloads["typing"],
        },

        // Mixed typing states
        mixedTyping: {
            event: "typing",
            data: {
                starting: ["user_789"],
                stopping: ["user_123", "user_456"],
            } satisfies ChatSocketEventPayloads["typing"],
        },
    },

    participants: {
        // User joins chat
        userJoining: {
            event: "participants",
            data: {
                joining: [chatParticipantFixtures.minimal.find as Omit<ChatParticipant, "chat">],
            } satisfies ChatSocketEventPayloads["participants"],
        },

        // Multiple users join
        multipleUsersJoining: {
            event: "participants",
            data: {
                joining: [
                    chatParticipantFixtures.minimal.find as Omit<ChatParticipant, "chat">,
                    { ...chatParticipantFixtures.minimal.find, id: "participant_2", user: userFixtures.complete.find } as Omit<ChatParticipant, "chat">,
                ],
            } satisfies ChatSocketEventPayloads["participants"],
        },

        // User leaves chat
        userLeaving: {
            event: "participants",
            data: {
                leaving: ["user_123"],
            } satisfies ChatSocketEventPayloads["participants"],
        },

        // Mixed participant changes
        mixedParticipants: {
            event: "participants",
            data: {
                joining: [chatParticipantFixtures.minimal.find as Omit<ChatParticipant, "chat">],
                leaving: ["user_456", "user_789"],
            } satisfies ChatSocketEventPayloads["participants"],
        },
    },

    botStatus: {
        thinking: {
            event: "botStatusUpdate",
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "thinking" as const,
                message: "Processing your request...",
            } satisfies ChatSocketEventPayloads["botStatusUpdate"],
        },

        toolCalling: {
            event: "botStatusUpdate",
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
            } satisfies ChatSocketEventPayloads["botStatusUpdate"],
        },

        toolCompleted: {
            event: "botStatusUpdate",
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "tool_completed" as const,
                toolInfo: {
                    callId: "call_123",
                    name: "search",
                    result: JSON.stringify({ results: ["Paper 1", "Paper 2"] }),
                },
            } satisfies ChatSocketEventPayloads["botStatusUpdate"],
        },

        toolFailed: {
            event: "botStatusUpdate",
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "tool_failed" as const,
                toolInfo: {
                    callId: "call_123",
                    name: "search",
                    error: "Search service unavailable",
                },
            } satisfies ChatSocketEventPayloads["botStatusUpdate"],
        },

        processingComplete: {
            event: "botStatusUpdate",
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "processing_complete" as const,
            } satisfies ChatSocketEventPayloads["botStatusUpdate"],
        },

        internalError: {
            event: "botStatusUpdate",
            data: {
                chatId: "chat_123",
                botId: "bot_123",
                status: "error_internal" as const,
                error: {
                    message: "An unexpected error occurred",
                    code: "INTERNAL_ERROR",
                    retryable: false,
                },
            } satisfies ChatSocketEventPayloads["botStatusUpdate"],
        },
    },

    toolApproval: {
        required: {
            event: "tool_approval_required",
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
            } satisfies ChatSocketEventPayloads["tool_approval_required"],
        },

        rejected: {
            event: "tool_approval_rejected",
            data: {
                pendingId: "pending_123",
                toolCallId: "call_456",
                toolName: "execute_code",
                reason: "User declined code execution",
                callerBotId: "bot_123",
            } satisfies ChatSocketEventPayloads["tool_approval_rejected"],
        },
    },

    // Event sequences for testing chat flows
    sequences: {
        // Typical message flow
        messageFlow: [
            { event: "typing", data: chatEventFixtures.typing.userStartTyping.data },
            { delay: 2000 },
            { event: "typing", data: chatEventFixtures.typing.userStopTyping.data },
            { delay: 100 },
            { event: "messages", data: chatEventFixtures.messages.textMessage.data },
        ],

        // Bot response flow
        botResponseFlow: [
            { event: "botStatusUpdate", data: chatEventFixtures.botStatus.thinking.data },
            { delay: 1000 },
            { event: "responseStream", data: chatEventFixtures.responseStream.streamStart.data },
            { delay: 500 },
            { event: "responseStream", data: chatEventFixtures.responseStream.streamChunk.data },
            { delay: 500 },
            { event: "responseStream", data: chatEventFixtures.responseStream.streamEnd.data },
            { event: "botStatusUpdate", data: chatEventFixtures.botStatus.processingComplete.data },
        ],

        // Tool execution flow
        toolExecutionFlow: [
            { event: "botStatusUpdate", data: chatEventFixtures.botStatus.thinking.data },
            { delay: 500 },
            { event: "botStatusUpdate", data: chatEventFixtures.botStatus.toolCalling.data },
            { delay: 2000 },
            { event: "botStatusUpdate", data: chatEventFixtures.botStatus.toolCompleted.data },
            { delay: 500 },
            { event: "responseStream", data: chatEventFixtures.responseStream.streamStart.data },
        ],

        // Tool approval flow
        toolApprovalFlow: [
            { event: "botStatusUpdate", data: chatEventFixtures.botStatus.thinking.data },
            { delay: 500 },
            { event: "tool_approval_required", data: chatEventFixtures.toolApproval.required.data },
            { delay: 5000 }, // User decision time
            { event: "tool_approval_rejected", data: chatEventFixtures.toolApproval.rejected.data },
            { event: "botStatusUpdate", data: { ...chatEventFixtures.botStatus.thinking.data, message: "Adjusting approach..." } },
        ],

        // Error recovery flow
        errorRecoveryFlow: [
            { event: "responseStream", data: chatEventFixtures.responseStream.streamStart.data },
            { delay: 1000 },
            { event: "responseStream", data: chatEventFixtures.responseStream.streamError.data },
            { delay: 2000 },
            { event: "botStatusUpdate", data: { ...chatEventFixtures.botStatus.thinking.data, message: "Retrying..." } },
            { delay: 1000 },
            { event: "responseStream", data: chatEventFixtures.responseStream.streamStart.data },
        ],

        // Participant change flow
        participantChangeFlow: [
            { event: "participants", data: chatEventFixtures.participants.userJoining.data },
            { delay: 100 },
            { event: "messages", data: { added: [{ ...chatMessageFixtures.minimal.find, content: "User joined the chat", isSystem: true }] } },
            { delay: 30000 },
            { event: "participants", data: chatEventFixtures.participants.userLeaving.data },
            { delay: 100 },
            { event: "messages", data: { added: [{ ...chatMessageFixtures.minimal.find, content: "User left the chat", isSystem: true }] } },
        ],
    },

    // Factory functions for dynamic event creation
    factories: {
        createMessageEvent: (message: Partial<ChatMessage>) => ({
            event: "messages",
            data: {
                added: [{
                    ...chatMessageFixtures.minimal.find,
                    ...message,
                }],
            } satisfies ChatSocketEventPayloads["messages"],
        }),

        createStreamChunk: (botId: string, chunk: string) => ({
            event: "responseStream",
            data: {
                __type: "stream" as const,
                botId,
                chunk,
            } satisfies ChatSocketEventPayloads["responseStream"],
        }),

        createTypingEvent: (userIds: string[], isTyping: boolean) => ({
            event: "typing",
            data: isTyping
                ? { starting: userIds }
                : { stopping: userIds },
        } satisfies ChatSocketEventPayloads["typing"]),

        createBotStatusEvent: (chatId: string, botId: string, status: ChatSocketEventPayloads["botStatusUpdate"]["status"], message?: string) => ({
            event: "botStatusUpdate",
            data: {
                chatId,
                botId,
                status,
                message,
            } satisfies ChatSocketEventPayloads["botStatusUpdate"],
        }),
    },
};