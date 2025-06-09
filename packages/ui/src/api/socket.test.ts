import { beforeEach, describe, expect, it, vi } from "vitest";

// First, unmock the socket module to override the global mock
vi.unmock("./socket.js");

// Mock socket.io-client before importing socket module
vi.mock("socket.io-client", () => ({
    io: () => ({
        connect: vi.fn(),
        disconnect: vi.fn(),
        emit: vi.fn(),
        on: vi.fn(),
        off: vi.fn(),
        connected: false,
        id: "mock-socket-id",
    }),
}));

// Mock i18next
vi.mock("i18next", () => ({
    default: {
        t: (key: string) => key,
    },
}));

// Mock the webSocketUrlBase constant
vi.mock("../utils/consts.js", () => ({
    webSocketUrlBase: "ws://localhost:3000",
}));

// Import after all mocks are set up
import { SocketService, SERVER_CONNECT_MESSAGE_ID, socket } from "./socket.js";
import { PubSub } from "../utils/pubsub.js";

describe("SocketService", () => {
    let socketService: SocketService;
    let pubsubSpy: any;

    beforeEach(() => {
        // Clear all mocks
        vi.clearAllMocks();
        
        // Reset the singleton instance
        (SocketService as any).instance = undefined;
        
        // Get fresh socket service instance
        socketService = SocketService.get();
        
        // Spy on PubSub methods
        const pubsubInstance = PubSub.get();
        pubsubSpy = {
            publish: vi.spyOn(pubsubInstance, "publish"),
            subscribe: vi.spyOn(pubsubInstance, "subscribe"),
        };
    });

    describe("Singleton Pattern", () => {
        it("should return the same instance when called multiple times", () => {
            // Reset instance for this specific test
            (SocketService as any).instance = undefined;
            
            const instance1 = SocketService.get();
            const instance2 = SocketService.get();
            expect(instance1).toBe(instance2);
        });

        it("should have initial state as DisconnectedNoError", () => {
            expect(socketService.state).toBe("DisconnectedNoError");
        });
    });

    describe("Connection Management", () => {
        it("should call socket.connect() when connect is called", () => {
            socketService.connect();
            expect(socket.connect).toHaveBeenCalled();
        });

        it("should set state to DisconnectedNoError and call socket.disconnect() when disconnect is called", () => {
            socketService.disconnect();
            expect(socketService.state).toBe("DisconnectedNoError");
            expect(socket.disconnect).toHaveBeenCalled();
        });

        it("should disconnect then connect when restart is called", () => {
            socketService.restart();
            expect(socket.disconnect).toHaveBeenCalled();
            expect(socket.connect).toHaveBeenCalled();
        });
    });

    describe("Connection Event Handlers", () => {
        it("should update state to Connected when handleConnected is called", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            socketService.handleConnected();
            
            expect(socketService.state).toBe("Connected");
            expect(pubsubSpy.publish).toHaveBeenCalledWith("clearSnack", { id: SERVER_CONNECT_MESSAGE_ID });
            
            consoleSpy.mockRestore();
        });

        it("should update state to LostConnection when handleDisconnected is called from Connected state", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            // First set to connected state by calling handleConnected
            socketService.handleConnected();
            expect(socketService.state).toBe("Connected");
            
            // Then disconnect
            socketService.handleDisconnected();
            
            expect(socketService.state).toBe("LostConnection");
            expect(pubsubSpy.publish).toHaveBeenCalledWith("snack", expect.objectContaining({
                severity: "Error"
            }));
            
            consoleSpy.mockRestore();
        });

        it("should not change state when handleDisconnected is called from non-Connected state", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            socketService.state = "DisconnectedNoError";
            socketService.handleDisconnected();
            
            expect(socketService.state).toBe("DisconnectedNoError");
            
            consoleSpy.mockRestore();
        });
    });

    describe("Event Emission", () => {
        it("should emit event with payload when no callback provided", () => {
            const testEvent = "joinChatRoom" as const;
            const testPayload = { chatId: "test-chat-id" };
            
            socketService.emitEvent(testEvent, testPayload);
            
            expect(socket.emit).toHaveBeenCalledWith(testEvent, testPayload);
        });

        it("should emit event with payload and callback when callback provided", () => {
            const testEvent = "joinChatRoom" as const;
            const testPayload = { chatId: "test-chat-id" };
            const testCallback = vi.fn();
            
            socketService.emitEvent(testEvent, testPayload, testCallback);
            
            expect(socket.emit).toHaveBeenCalledWith(testEvent, testPayload, testCallback);
        });
    });

    describe("Event Listening", () => {
        it("should register event listener and return unsubscribe function", () => {
            const testEvent = "messages" as const;
            const testHandler = vi.fn();
            
            const unsubscribe = socketService.onEvent(testEvent, testHandler);
            
            expect(socket.on).toHaveBeenCalledWith(testEvent, testHandler);
            expect(typeof unsubscribe).toBe("function");
        });

        it("should unsubscribe when returned function is called", () => {
            const testEvent = "responseStream" as const;
            const testHandler = vi.fn();
            
            const unsubscribe = socketService.onEvent(testEvent, testHandler);
            unsubscribe();
            
            expect(socket.off).toHaveBeenCalledWith(testEvent, testHandler);
        });
    });

    describe("State Management", () => {
        it("should track state transitions correctly", () => {
            // Initial state
            expect(socketService.state).toBe("DisconnectedNoError");
            
            // Connect
            socketService.handleConnected();
            expect(socketService.state).toBe("Connected");
            
            // Disconnect (simulating lost connection)
            socketService.handleDisconnected();
            expect(socketService.state).toBe("LostConnection");
            
            // Manual disconnect
            socketService.disconnect();
            expect(socketService.state).toBe("DisconnectedNoError");
        });
    });
});