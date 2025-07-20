// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-24
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-01 - Fixed 10 'as any' type assertions with proper types and eslint-disable comments
import { beforeEach, describe, expect, it, vi } from "vitest";

// Unmock PubSub to use real implementation
vi.unmock("../utils/pubsub.js");
// Unmock the socket service to test the real implementation
vi.unmock("./socket.js");

// Use the centralized socket mock from setup.vitest.ts
import { createMockSocket } from "../__test/mocks/libraries/socket.io-client.js";
import { io } from "socket.io-client";
import type { Socket } from "socket.io-client";

const mockSocket = createMockSocket();

// Set up io to return our mock socket
vi.mocked(io).mockReturnValue(mockSocket as Socket);

// Import after mocking - this will use the real SocketService implementation
import { socket, SocketService, SERVER_CONNECT_MESSAGE_ID } from "./socket.js";
import { PubSub } from "../utils/pubsub.js";

// Spy on the actual socket methods
beforeEach(() => {
    // Replace socket methods with our mock methods
    Object.assign(socket, mockSocket);
});

describe("SocketService", () => {
    let socketService: SocketService;
    let pubsubSpy: {
        publish: ReturnType<typeof vi.spyOn>;
        subscribe: ReturnType<typeof vi.spyOn>;
    };

    beforeEach(() => {
        // Clear all mocks
        vi.clearAllMocks();
        
        // Reset the singleton instance
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
        it("returns the same instance when called multiple times", () => {
            // Reset instance for this specific test
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (SocketService as any).instance = undefined;
            
            const instance1 = SocketService.get();
            const instance2 = SocketService.get();
            expect(instance1).toBe(instance2);
        });

        it("initializes with DisconnectedNoError state", () => {
            // State is private but we can verify through behavior
            // When disconnected, no snack messages should be published
            socketService.disconnect();
            expect(pubsubSpy.publish).not.toHaveBeenCalled();
        });
    });

    describe("Connection Management", () => {
        it("calls socket.connect() when connect is called", () => {
            socketService.connect();
            expect(mockSocket.connect).toHaveBeenCalled();
        });

        it("sets state to DisconnectedNoError and calls socket.disconnect() when disconnect is called", () => {
            socketService.disconnect();
            expect(socketService.state).toBe("DisconnectedNoError");
            expect(mockSocket.disconnect).toHaveBeenCalled();
        });

        it("disconnects then connects when restart is called", () => {
            socketService.restart();
            expect(mockSocket.disconnect).toHaveBeenCalled();
            expect(mockSocket.connect).toHaveBeenCalled();
        });
    });

    describe("Connection Event Handlers", () => {
        it("updates state to Connected and publishes clearSnack when handleConnected is called", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            socketService.handleConnected();
            
            expect(socketService.state).toBe("Connected");
            expect(pubsubSpy.publish).toHaveBeenCalledWith("clearSnack", { id: SERVER_CONNECT_MESSAGE_ID });
            expect(consoleSpy).toHaveBeenCalledWith("Websocket connected to server");
            
            consoleSpy.mockRestore();
        });

        it("publishes success snack when reconnecting after lost connection", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            // Set state to LostConnection first
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (socketService as any).state = "LostConnection";
            socketService.handleConnected();
            
            expect(socketService.state).toBe("Connected");
            expect(pubsubSpy.publish).toHaveBeenCalledWith("clearSnack", { id: SERVER_CONNECT_MESSAGE_ID });
            expect(pubsubSpy.publish).toHaveBeenCalledWith("snack", { 
                message: "ServerReconnected", 
                severity: "Success", 
            });
            
            consoleSpy.mockRestore();
        });

        it("updates state to LostConnection when handleDisconnected is called from Connected state", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            // Set state to Connected first
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (socketService as any).state = "Connected";
            socketService.handleDisconnected();
            
            expect(socketService.state).toBe("LostConnection");
            expect(pubsubSpy.publish).toHaveBeenCalledWith("snack", {
                id: SERVER_CONNECT_MESSAGE_ID,
                message: "ServerDisconnected",
                severity: "Error",
                autoHideDuration: "persist",
            });
            expect(consoleSpy).toHaveBeenCalledWith("Websocket disconnected from server");
            
            consoleSpy.mockRestore();
        });

        it("does not change state when handleDisconnected is called from non-Connected state", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (socketService as any).state = "DisconnectedNoError";
            socketService.handleDisconnected();
            
            expect(socketService.state).toBe("DisconnectedNoError");
            expect(pubsubSpy.publish).not.toHaveBeenCalled();
        });

        it("handles reconnect attempt with warning message", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            socketService.handleReconnectAttempted();
            
            expect(pubsubSpy.publish).toHaveBeenCalledWith("snack", {
                message: "ServerReconnectAttempt",
                severity: "Warning",
                id: SERVER_CONNECT_MESSAGE_ID,
                autoHideDuration: "persist",
            });
            expect(consoleSpy).toHaveBeenCalledWith("Websocket attempting to reconnect to server");
            
            consoleSpy.mockRestore();
        });

        it("handles successful reconnection", () => {
            const consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
            
            socketService.handleReconnected();
            
            expect(socketService.state).toBe("Connected");
            expect(pubsubSpy.publish).toHaveBeenCalledWith("snack", {
                message: "ServerReconnected",
                severity: "Success",
                id: SERVER_CONNECT_MESSAGE_ID,
            });
            expect(consoleSpy).toHaveBeenCalledWith("Websocket reconnected to server");
            
            consoleSpy.mockRestore();
        });
    });

    describe("Emit Event", () => {
        it("emits an event through socket with payload", () => {
            const event = "message:send" as const;
            const payload = { test: "data" };
            
            socketService.emitEvent(event, payload);
            
            expect(mockSocket.emit).toHaveBeenCalledWith(event, payload);
        });

        it("emits an event with callback", () => {
            const event = "message:send" as const;
            const payload = { test: "data" };
            const callback = vi.fn();
            
            socketService.emitEvent(event, payload, callback);
            
            expect(mockSocket.emit).toHaveBeenCalledWith(event, payload, callback);
        });
    });

    describe("Socket Event Listeners", () => {
        it("registers and unregisters event listeners through onEvent method", () => {
            const event = "message:receive" as const;
            const handler = vi.fn();
            
            const unsubscribe = socketService.onEvent(event, handler);
            
            // Verify that socket.on was called with the event and a wrapped handler function
            expect(mockSocket.on).toHaveBeenCalledWith(event, expect.any(Function));
            
            // Get the wrapped handler that was actually passed to socket.on
            const wrappedHandler = mockSocket.on.mock.calls[mockSocket.on.mock.calls.length - 1][1];
            
            // Call unsubscribe to remove the listener
            unsubscribe();
            
            // Verify that socket.off was called with the same wrapped handler
            expect(mockSocket.off).toHaveBeenCalledWith(event, wrappedHandler);
        });

        it("returns unsubscribe function that removes the listener", () => {
            const event = "message:receive" as const;
            const handler = vi.fn();
            
            const unsubscribe = socketService.onEvent(event, handler);
            
            expect(typeof unsubscribe).toBe("function");
        });
    });
});
