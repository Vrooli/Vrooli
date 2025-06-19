/**
 * Base socket event fixtures for testing WebSocket connections and lifecycle
 */

import { type JoinChatRoomCallbackData, type LeaveChatRoomCallbackData, type JoinRunRoomCallbackData, type LeaveRunRoomCallbackData, type JoinUserRoomCallbackData, type LeaveUserRoomCallbackData } from "../../../consts/socketEvents.js";

export const socketEventFixtures = {
    connection: {
        connected: {
            event: "connect",
            data: undefined,
        },
        disconnected: {
            event: "disconnect",
            data: undefined,
        },
        reconnecting: {
            event: "reconnect_attempt",
            data: {
                attempt: 1,
                delay: 1000,
            },
        },
        reconnected: {
            event: "reconnect",
            data: {
                attempt: 3,
            },
        },
        error: {
            event: "connect_error",
            data: {
                message: "Connection refused",
                type: "TransportError",
            },
        },
    },

    room: {
        // Chat room events
        joinChatSuccess: {
            event: "joinChatRoom",
            data: { chatId: "chat_123" },
            callback: {
                success: true,
            } satisfies JoinChatRoomCallbackData,
        },
        joinChatFailure: {
            event: "joinChatRoom",
            data: { chatId: "chat_invalid" },
            callback: {
                success: false,
                error: "ChatNotFound",
            } satisfies JoinChatRoomCallbackData,
        },
        leaveChatSuccess: {
            event: "leaveChatRoom",
            data: { chatId: "chat_123" },
            callback: {
                success: true,
            } satisfies LeaveChatRoomCallbackData,
        },

        // Run room events
        joinRunSuccess: {
            event: "joinRunRoom",
            data: { runId: "run_123" },
            callback: {
                success: true,
            } satisfies JoinRunRoomCallbackData,
        },
        joinRunFailure: {
            event: "joinRunRoom",
            data: { runId: "run_invalid" },
            callback: {
                success: false,
                error: "RunNotFound",
            } satisfies JoinRunRoomCallbackData,
        },
        leaveRunSuccess: {
            event: "leaveRunRoom",
            data: { runId: "run_123" },
            callback: {
                success: true,
            } satisfies LeaveRunRoomCallbackData,
        },

        // User room events
        joinUserSuccess: {
            event: "joinUserRoom",
            data: { userId: "user_123" },
            callback: {
                success: true,
            } satisfies JoinUserRoomCallbackData,
        },
        joinUserFailure: {
            event: "joinUserRoom",
            data: { userId: "user_invalid" },
            callback: {
                success: false,
                error: "Unauthorized",
            } satisfies JoinUserRoomCallbackData,
        },
        leaveUserSuccess: {
            event: "leaveUserRoom",
            data: { userId: "user_123" },
            callback: {
                success: true,
            } satisfies LeaveUserRoomCallbackData,
        },
    },

    errors: {
        unauthorized: {
            event: "error",
            data: {
                code: "UNAUTHORIZED",
                message: "Authentication required",
            },
        },
        rateLimit: {
            event: "error",
            data: {
                code: "RATE_LIMIT",
                message: "Too many requests",
                retryAfter: 60,
            },
        },
        serverError: {
            event: "error",
            data: {
                code: "INTERNAL_ERROR",
                message: "An unexpected error occurred",
            },
        },
    },

    // Event sequences for testing connection flows
    sequences: {
        connectionFlow: [
            { event: "connect", data: undefined },
            { delay: 100 },
            { event: "joinUserRoom", data: { userId: "user_123" } },
            { delay: 50 },
            { event: "joinChatRoom", data: { chatId: "chat_456" } },
        ],
        reconnectionFlow: [
            { event: "disconnect", data: undefined },
            { delay: 1000 },
            { event: "reconnect_attempt", data: { attempt: 1 } },
            { delay: 2000 },
            { event: "reconnect_attempt", data: { attempt: 2 } },
            { delay: 3000 },
            { event: "reconnect", data: { attempt: 3 } },
        ],
        authenticationFlow: [
            { event: "connect", data: undefined },
            { delay: 100 },
            { event: "authenticate", data: { token: "invalid" } },
            { delay: 50 },
            { event: "error", data: { code: "UNAUTHORIZED" } },
            { delay: 100 },
            { event: "disconnect", data: undefined },
        ],
    },

    // Factory functions for dynamic event creation
    factories: {
        createConnectionEvent: (connected: boolean) => ({
            event: connected ? "connect" : "disconnect",
            data: undefined,
        }),
        createRoomJoinEvent: (roomType: "chat" | "run" | "user", id: string) => ({
            event: `join${roomType.charAt(0).toUpperCase() + roomType.slice(1)}Room`,
            data: { [`${roomType}Id`]: id },
        }),
        createErrorEvent: (code: string, message: string, details?: any) => ({
            event: "error",
            data: { code, message, ...details },
        }),
    },
};

// Type for event sequences (used in mock utilities)
export interface EventSequenceItem {
    event: string;
    data?: any;
    delay?: number;
}