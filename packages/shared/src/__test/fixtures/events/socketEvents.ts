/**
 * Enhanced socket event fixtures using factory pattern for comprehensive testing
 * Provides realistic WebSocket connection lifecycle simulation
 */

import {
    type JoinChatRoomCallbackData,
    type JoinRunRoomCallbackData,
    type JoinUserRoomCallbackData,
    type LeaveChatRoomCallbackData,
    type LeaveRunRoomCallbackData,
    type LeaveUserRoomCallbackData,
} from "../../../consts/socketEvents.js";
import { BaseEventFactory } from "./BaseEventFactory.js";
import { generateCorrelationId } from "./eventUtils.js";
import {
    type SocketEventFixture,
} from "./types.js";

// Socket event data types
interface SocketConnectionData {
    socketId?: string;
    transport?: string;
    timestamp?: number;
}

interface SocketErrorData {
    code: string;
    message: string;
    type?: string;
    retryable?: boolean;
    details?: unknown;
}

interface SocketRoomData {
    roomType: "chat" | "run" | "user";
    roomId: string;
    userId?: string;
}

interface SocketReconnectData {
    attempt: number;
    delay?: number;
    maxAttempts?: number;
}

// Base socket event
type SocketEvent<T = unknown> = SocketEventFixture<T>;

/**
 * Factory for connection lifecycle events
 */
class ConnectionEventFactory extends BaseEventFactory<SocketEvent<SocketConnectionData>, SocketConnectionData> {
    constructor() {
        super("connection", {
            defaults: {
                socketId: "socket_123",
                transport: "websocket",
                timestamp: Date.now(),
            },
        });
    }

    single = {
        event: "connect",
        data: {
            socketId: "socket_123",
            transport: "websocket",
            timestamp: Date.now(),
        },
    } satisfies SocketEvent<SocketConnectionData>;

    sequence = [
        this.single,
        this.withDelay({
            event: "pong",
            data: { timestamp: Date.now() },
        }, 30000),
    ];

    variants = {
        polling: {
            event: "connect",
            data: {
                socketId: "socket_456",
                transport: "polling",
                timestamp: Date.now(),
            },
        },
        disconnect: {
            event: "disconnect",
            data: {
                reason: "transport close",
                timestamp: Date.now(),
            },
        },
        forceDisconnect: {
            event: "disconnect",
            data: {
                reason: "forced close",
                timestamp: Date.now(),
            },
        },
    };

    // Override to handle connection state
    protected applyEventToState(state: Record<string, unknown>, event: SocketEvent<SocketConnectionData>): Record<string, unknown> {
        return {
            ...state,
            connected: event.event === "connect",
            socketId: event.data.socketId || state.socketId,
            lastConnectionEvent: event.event,
            lastConnectionTime: Date.now(),
        };
    }
}

/**
 * Factory for error events
 */
class SocketErrorEventFactory extends BaseEventFactory<SocketEvent<SocketErrorData>, SocketErrorData> {
    constructor() {
        super("error", {
            validation: (data) => {
                if (!data.code || !data.message) {
                    return "Error events must have code and message";
                }
                return true;
            },
        });
    }

    single = {
        event: "error",
        data: {
            code: "UNKNOWN_ERROR",
            message: "An unknown error occurred",
            retryable: false,
        },
    } satisfies SocketEvent<SocketErrorData>;

    sequence = [
        this.create({ code: "NETWORK_ERROR", message: "Network unreachable", retryable: true }),
        this.withDelay(this.create({ code: "TIMEOUT", message: "Request timeout", retryable: true }), 5000),
        this.withDelay(this.create({ code: "SERVER_ERROR", message: "Internal server error", retryable: false }), 10000),
    ];

    variants = {
        unauthorized: {
            event: "error",
            data: {
                code: "UNAUTHORIZED",
                message: "Authentication required",
                type: "AuthError",
                retryable: false,
            },
        },
        rateLimit: {
            event: "error",
            data: {
                code: "RATE_LIMIT",
                message: "Too many requests",
                type: "RateLimitError",
                retryable: true,
                details: { retryAfter: 60, limit: 100, remaining: 0 },
            },
        },
        networkTimeout: {
            event: "connect_error",
            data: {
                code: "NETWORK_TIMEOUT",
                message: "Connection timeout",
                type: "TransportError",
                retryable: true,
            },
        },
        transportClosed: {
            event: "connect_error",
            data: {
                code: "TRANSPORT_CLOSED",
                message: "Transport closed",
                type: "TransportError",
                retryable: true,
            },
        },
    };
}

/**
 * Factory for room management events
 */
class RoomEventFactory extends BaseEventFactory<SocketEvent<SocketRoomData>, SocketRoomData> {
    constructor() {
        super("room");
    }

    single = {
        event: "joinChatRoom",
        data: {
            roomType: "chat" as const,
            roomId: "chat_123",
            userId: "user_123",
        },
        room: "chat:chat_123",
    } satisfies SocketEvent<SocketRoomData>;

    sequence = [
        this.create({ roomType: "user", roomId: "user_123" }),
        this.withDelay(this.create({ roomType: "chat", roomId: "chat_456" }), 100),
        this.withDelay(this.create({ roomType: "run", roomId: "run_789" }), 200),
    ];

    variants = {
        // Chat room variants
        joinChatSuccess: {
            event: "joinChatRoom",
            data: { roomType: "chat" as const, roomId: "chat_123" },
            room: "chat:chat_123",
            acknowledgment: { success: true } satisfies JoinChatRoomCallbackData,
        },
        joinChatFailure: {
            event: "joinChatRoom",
            data: { roomType: "chat" as const, roomId: "chat_invalid" },
            acknowledgment: { success: false, error: "ChatNotFound" } satisfies JoinChatRoomCallbackData,
        },
        leaveChatSuccess: {
            event: "leaveChatRoom",
            data: { roomType: "chat" as const, roomId: "chat_123" },
            acknowledgment: { success: true } satisfies LeaveChatRoomCallbackData,
        },

        // Run room variants
        joinRunSuccess: {
            event: "joinRunRoom",
            data: { roomType: "run" as const, roomId: "run_123" },
            room: "run:run_123",
            acknowledgment: { success: true } satisfies JoinRunRoomCallbackData,
        },
        joinRunFailure: {
            event: "joinRunRoom",
            data: { roomType: "run" as const, roomId: "run_invalid" },
            acknowledgment: { success: false, error: "RunNotFound" } satisfies JoinRunRoomCallbackData,
        },

        // User room variants
        joinUserSuccess: {
            event: "joinUserRoom",
            data: { roomType: "user" as const, roomId: "user_123", userId: "user_123" },
            room: "user:user_123",
            acknowledgment: { success: true } satisfies JoinUserRoomCallbackData,
        },
        joinUserUnauthorized: {
            event: "joinUserRoom",
            data: { roomType: "user" as const, roomId: "user_456", userId: "user_123" },
            acknowledgment: { success: false, error: "Unauthorized" } satisfies JoinUserRoomCallbackData,
        },
    };

    // Factory method override to handle room-specific event names
    create(overrides?: Partial<SocketRoomData>): SocketEvent<SocketRoomData> {
        const data = { ...this.single.data, ...overrides };
        const eventName = this.getRoomEventName(data.roomType, "join");
        const roomName = `${data.roomType}:${data.roomId}`;

        return {
            event: eventName,
            data,
            room: roomName,
        };
    }

    private getRoomEventName(roomType: string, action: "join" | "leave"): string {
        const capitalizedType = roomType.charAt(0).toUpperCase() + roomType.slice(1);
        return `${action}${capitalizedType}Room`;
    }
}

/**
 * Factory for reconnection events
 */
class ReconnectionEventFactory extends BaseEventFactory<SocketEvent<SocketReconnectData>, SocketReconnectData> {
    constructor() {
        super("reconnect");
    }

    single = {
        event: "reconnect_attempt",
        data: {
            attempt: 1,
            delay: 1000,
            maxAttempts: 5,
        },
    } satisfies SocketEvent<SocketReconnectData>;

    sequence = this.createSequence("escalating", { count: 5, interval: 1000 });

    variants = {
        firstAttempt: this.single,
        subsequentAttempt: {
            event: "reconnect_attempt",
            data: { attempt: 3, delay: 4000, maxAttempts: 5 },
        },
        success: {
            event: "reconnect",
            data: { attempt: 3, delay: 4000 },
        },
        failure: {
            event: "reconnect_failed",
            data: { attempt: 5, delay: 16000, maxAttempts: 5 },
        },
        backoffSequence: this.withJitter(
            [
                this.create({ attempt: 1, delay: 1000 }),
                this.create({ attempt: 2, delay: 2000 }),
                this.create({ attempt: 3, delay: 4000 }),
                this.create({ attempt: 4, delay: 8000 }),
                this.create({ attempt: 5, delay: 16000 }),
            ],
            0,
            500,
        ),
    };

    // Override to generate exponential backoff
    createSequence(pattern: "escalating", options?: { count?: number }): SocketEvent<SocketReconnectData>[] {
        const count = options?.count || 5;
        const events: SocketEvent<SocketReconnectData>[] = [];

        for (let i = 0; i < count; i++) {
            const delay = Math.min(1000 * Math.pow(2, i), 30000); // Cap at 30s
            events.push({
                event: "reconnect_attempt",
                data: {
                    attempt: i + 1,
                    delay,
                    maxAttempts: count,
                },
            });
        }

        return events;
    }
}

// Create factory instances
const connectionFactory = new ConnectionEventFactory();
const errorFactory = new SocketErrorEventFactory();
const roomFactory = new RoomEventFactory();
const reconnectionFactory = new ReconnectionEventFactory();

// Export the enhanced fixtures with backward compatibility
export const socketEventFixtures = {
    // Connection events
    connection: {
        connected: connectionFactory.single,
        disconnected: connectionFactory.variants.disconnect as SocketEvent,
        reconnecting: reconnectionFactory.single,
        reconnected: reconnectionFactory.variants.success as SocketEvent,
        error: errorFactory.variants.networkTimeout as SocketEvent,

        // Additional variants
        polling: connectionFactory.variants.polling as SocketEvent,
        forceDisconnect: connectionFactory.variants.forceDisconnect as SocketEvent,
    },

    // Room management events
    room: {
        // Chat room events - backward compatible
        joinChatSuccess: {
            event: "joinChatRoom",
            data: { chatId: "chat_123" },
            callback: (roomFactory.variants.joinChatSuccess as SocketEvent).acknowledgment,
        },
        joinChatFailure: {
            event: "joinChatRoom",
            data: { chatId: "chat_invalid" },
            callback: (roomFactory.variants.joinChatFailure as SocketEvent).acknowledgment,
        },
        leaveChatSuccess: {
            event: "leaveChatRoom",
            data: { chatId: "chat_123" },
            callback: (roomFactory.variants.leaveChatSuccess as SocketEvent).acknowledgment,
        },

        // Run room events - backward compatible
        joinRunSuccess: {
            event: "joinRunRoom",
            data: { runId: "run_123" },
            callback: (roomFactory.variants.joinRunSuccess as SocketEvent).acknowledgment,
        },
        joinRunFailure: {
            event: "joinRunRoom",
            data: { runId: "run_invalid" },
            callback: (roomFactory.variants.joinRunFailure as SocketEvent).acknowledgment,
        },
        leaveRunSuccess: {
            event: "leaveRunRoom",
            data: { runId: "run_123" },
            callback: { success: true } satisfies LeaveRunRoomCallbackData,
        },

        // User room events - backward compatible
        joinUserSuccess: {
            event: "joinUserRoom",
            data: { userId: "user_123" },
            callback: (roomFactory.variants.joinUserSuccess as SocketEvent).acknowledgment,
        },
        joinUserFailure: {
            event: "joinUserRoom",
            data: { userId: "user_invalid" },
            callback: { success: false, error: "Unauthorized" } satisfies JoinUserRoomCallbackData,
        },
        leaveUserSuccess: {
            event: "leaveUserRoom",
            data: { userId: "user_123" },
            callback: { success: true } satisfies LeaveUserRoomCallbackData,
        },
    },

    // Error events
    errors: {
        unauthorized: errorFactory.variants.unauthorized as SocketEvent,
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

        // Additional error types
        networkTimeout: errorFactory.variants.networkTimeout as SocketEvent,
        transportClosed: errorFactory.variants.transportClosed as SocketEvent,
    },

    // Event sequences for testing connection flows
    sequences: {
        // Basic connection flow
        connectionFlow: [
            connectionFactory.single,
            { delay: 100 },
            { event: "joinUserRoom", data: { userId: "user_123" } },
            { delay: 50 },
            { event: "joinChatRoom", data: { chatId: "chat_456" } },
        ],

        // Reconnection with exponential backoff
        reconnectionFlow: reconnectionFactory.variants.backoffSequence,

        // Authentication failure flow
        authenticationFlow: [
            connectionFactory.single,
            { delay: 100 },
            { event: "authenticate", data: { token: "invalid" } },
            { delay: 50 },
            errorFactory.variants.unauthorized,
            { delay: 100 },
            connectionFactory.variants.disconnect,
        ],

        // Enhanced sequences
        stableConnection: [
            connectionFactory.single,
            { delay: 100 },
            roomFactory.variants.joinUserSuccess,
            { delay: 50 },
            roomFactory.variants.joinChatSuccess,
            { delay: 30000 },
            { event: "ping", data: {} },
            { delay: 100 },
            { event: "pong", data: {} },
        ],

        unstableNetwork: connectionFactory.withJitter([
            connectionFactory.single,
            connectionFactory.variants.disconnect,
            reconnectionFactory.single,
            connectionFactory.single,
            connectionFactory.variants.disconnect,
        ], 5000, 2000),

        roomJoinFailures: [
            connectionFactory.single,
            { delay: 100 },
            roomFactory.variants.joinChatFailure,
            { delay: 1000 },
            roomFactory.variants.joinRunFailure,
            { delay: 1000 },
            roomFactory.variants.joinUserUnauthorized,
        ],

        // Complex multi-user scenario
        multiUserJoins: [
            { event: "participants", data: { joining: [{ userId: "user_1" }] } },
            { delay: 500 },
            { event: "participants", data: { joining: [{ userId: "user_2" }] } },
            { delay: 1000 },
            { event: "participants", data: { joining: [{ userId: "user_3" }] } },
            { delay: 2000 },
            { event: "participants", data: { leaving: ["user_1"] } },
        ],
    },

    // Enhanced factory functions
    factories: {
        // Connection factory
        connection: connectionFactory,

        // Error factory
        error: errorFactory,

        // Room factory
        room: roomFactory,

        // Reconnection factory
        reconnection: reconnectionFactory,

        // Legacy factory functions for backward compatibility
        createConnectionEvent: (connected: boolean) =>
            connected ? connectionFactory.single : connectionFactory.variants.disconnect,

        createRoomJoinEvent: (roomType: "chat" | "run" | "user", id: string) =>
            roomFactory.create({ roomType, roomId: id }),

        createErrorEvent: (code: string, message: string, details?: unknown) =>
            errorFactory.create({ code, message, ...details as SocketErrorData }),

        // New factory methods
        createReconnectSequence: (attempts = 5) =>
            reconnectionFactory.createSequence("escalating", { count: attempts }),

        createRoomSequence: (rooms: Array<{ type: "chat" | "run" | "user"; id: string }>) =>
            rooms.map((room, i) =>
                reconnectionFactory.withDelay(
                    roomFactory.create({ roomType: room.type, roomId: room.id }),
                    i * 100,
                ),
            ),

        createCorrelatedConnection: (correlationId: string = generateCorrelationId()) =>
            connectionFactory.createCorrelated(correlationId, [
                connectionFactory.single,
                roomFactory.create({ roomType: "user", roomId: "user_123" }),
                roomFactory.create({ roomType: "chat", roomId: "chat_456" }),
            ]),
    },

    // Scenarios for comprehensive testing
    scenarios: {
        // Network condition testing
        perfectNetwork: {
            sequence: connectionFactory.sequence,
            networkCondition: { latency: 5, jitter: 1, loss: 0 },
        },

        poorNetwork: {
            sequence: reconnectionFactory.variants.backoffSequence,
            networkCondition: { latency: 200, jitter: 100, loss: 5 },
        },

        offlineRecovery: {
            sequence: [
                connectionFactory.single,
                { delay: 5000 },
                connectionFactory.variants.disconnect,
                { delay: 10000 },
                ...reconnectionFactory.createSequence("escalating", { count: 3 }),
                connectionFactory.single,
            ],
            networkCondition: { latency: 0, jitter: 0, loss: 100 },
        },

        // Authentication scenarios
        successfulAuth: {
            sequence: [
                connectionFactory.single,
                { event: "authenticate", data: { token: "valid_token" } },
                { event: "authenticated", data: { userId: "user_123" } },
                roomFactory.variants.joinUserSuccess,
            ],
        },

        tokenExpiry: {
            sequence: [
                connectionFactory.single,
                { delay: 3600000 }, // 1 hour
                errorFactory.create({ code: "TOKEN_EXPIRED", message: "Authentication token expired" }),
                connectionFactory.variants.disconnect,
            ],
        },
    },
};

// Export factory instances for direct use
export {
    connectionFactory,
    errorFactory, reconnectionFactory, roomFactory,
};

// Re-export types for convenience
export type { EventSequenceItem } from "./types.js";

