import { beforeEach, describe, expect, it, vi } from "vitest";
import { PubSub } from "../utils/pubsub.js";

// Create mock functions
const mockSocket = {
    connect: vi.fn(),
    disconnect: vi.fn(),
    emit: vi.fn(),
    on: vi.fn(),
    off: vi.fn(),
    connected: false,
    id: "mock-socket-id",
};

// Mock socket.io-client
vi.mock("socket.io-client", () => ({
    io: () => mockSocket,
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

// Import after mocking
const { SocketService, SERVER_CONNECT_MESSAGE_ID } = await import("./socket.js");

describe("SocketService", () => {
    let socketService: SocketService;
    let pubsubSpy: any;

    beforeEach(() => {
        // Reset the singleton instance
        (SocketService as any).instance = undefined;
        
        // Get fresh socket service instance
        socketService = SocketService.get();
        
        // Spy on PubSub methods
        pubsubSpy = {
            publish: vi.spyOn(PubSub.get(), "publish"),
            subscribe: vi.spyOn(PubSub.get(), "subscribe"),
        };
        
        // Clear all mocks
        vi.clearAllMocks();
    });

    describe("Singleton Pattern", () => {
        it("should return the same instance when called multiple times", () => {
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
            expect(mockSocket.connect).toHaveBeenCalled();
        });

        it("should set state to DisconnectedNoError and call socket.disconnect() when disconnect is called", () => {
            socketService.disconnect();
            expect(socketService.state).toBe("DisconnectedNoError");
            expect(mockSocket.disconnect).toHaveBeenCalled();
        });

        it("should disconnect then connect when restart is called", () => {
            socketService.restart();
            expect(mockSocket.disconnect).toHaveBeenCalled();
            expect(mockSocket.connect).toHaveBeenCalled();
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
            
            socketService.state = "Connected";
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
            
            expect(mockSocket.emit).toHaveBeenCalledWith(testEvent, testPayload);
        });

        it("should emit event with payload and callback when callback provided", () => {
            const testEvent = "joinChatRoom" as const;
            const testPayload = { chatId: "test-chat-id" };
            const testCallback = vi.fn();
            
            socketService.emitEvent(testEvent, testPayload, testCallback);
            
            expect(mockSocket.emit).toHaveBeenCalledWith(testEvent, testPayload, testCallback);
        });
    });

    describe("Event Listening", () => {
        it("should register event listener and return unsubscribe function", () => {
            const testEvent = "messages" as const;
            const testHandler = vi.fn();
            
            const unsubscribe = socketService.onEvent(testEvent, testHandler);
            
            expect(mockSocket.on).toHaveBeenCalledWith(testEvent, testHandler);
            expect(typeof unsubscribe).toBe("function");
        });

        it("should unsubscribe when returned function is called", () => {
            const testEvent = "responseStream" as const;
            const testHandler = vi.fn();
            
            const unsubscribe = socketService.onEvent(testEvent, testHandler);
            unsubscribe();
            
            expect(mockSocket.off).toHaveBeenCalledWith(testEvent, testHandler);
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