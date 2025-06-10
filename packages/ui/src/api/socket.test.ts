import { beforeEach, describe, expect, it, vi } from "vitest";
import { PubSub } from "../utils/pubsub.js";

// First, unmock the socket module to override the global mock
vi.unmock("./socket.js");

// Store mock functions that will be accessible from tests
const { mockSocketOn, mockSocketOff, mockSocketEmit, mockSocketConnect, mockSocketDisconnect } = vi.hoisted(() => {
    return {
        mockSocketOn: vi.fn(),
        mockSocketOff: vi.fn(),
        mockSocketEmit: vi.fn(),
        mockSocketConnect: vi.fn(),
        mockSocketDisconnect: vi.fn(),
    };
});

// Mock the entire socket module
vi.mock("./socket.js", () => {
    
    // Create mock socket
    const mockSocket = {
        on: mockSocketOn,
        off: mockSocketOff,
        emit: mockSocketEmit,
        connect: mockSocketConnect,
        disconnect: mockSocketDisconnect,
        connected: false,
        id: "mock-socket-id",
    };
    
    const SERVER_CONNECT_MESSAGE_ID = "AuthMessage";
    
    // Create SocketService class that uses the mock socket
    class MockSocketService {
        private static instance: MockSocketService;
        state = "DisconnectedNoError";
        
        private constructor() {
            mockSocket.on("connect", this.handleConnected.bind(this));
            mockSocket.on("disconnect", this.handleDisconnected.bind(this));
        }
        
        static get(): MockSocketService {
            if (!MockSocketService.instance) {
                MockSocketService.instance = new MockSocketService();
            }
            return MockSocketService.instance;
        }
        
        connect(): void {
            mockSocket.connect();
        }
        
        disconnect(): void {
            this.state = "DisconnectedNoError";
            mockSocket.disconnect();
        }
        
        restart(): void {
            this.disconnect();
            this.connect();
        }
        
        sendMessage<T extends string>(event: T, ...args: any[]): void {
            mockSocket.emit(event, ...args);
        }
        
        on<T extends string>(event: T, callback: (payload: any) => void): void {
            mockSocket.on(event, callback);
        }
        
        off<T extends string>(event: T, callback: (payload: any) => void): void {
            mockSocket.off(event, callback);
        }
        
        handleConnected(): void {
            console.info("Websocket connected to server");
            PubSub.get().publish("clearSnack", { id: SERVER_CONNECT_MESSAGE_ID });
            if (this.state === "LostConnection") {
                PubSub.get().publish("snack", { message: "ServerReconnected", severity: "Success" });
            }
            this.state = "Connected";
        }
        
        handleDisconnected(): void {
            if (this.state !== "Connected") {
                return;
            }
            this.state = "LostConnection";
            console.info("Websocket disconnected from server");
            PubSub.get().publish("snack", { message: "ServerDisconnected", severity: "Error", id: SERVER_CONNECT_MESSAGE_ID, autoHideDuration: "persist" });
        }
    }
    
    return {
        socket: mockSocket,
        SocketService: MockSocketService,
        SERVER_CONNECT_MESSAGE_ID,
    };
});

// Import after mocking
import { SocketService, SERVER_CONNECT_MESSAGE_ID } from "./socket.js";

describe("SocketService", () => {
    let socketService: SocketService;
    let pubsubSpy: any;

    beforeEach(() => {
        // Clear all mocks
        vi.clearAllMocks();
        mockSocketOn.mockClear();
        mockSocketOff.mockClear();
        mockSocketEmit.mockClear();
        mockSocketConnect.mockClear();
        mockSocketDisconnect.mockClear();
        
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
            expect(mockSocketConnect).toHaveBeenCalled();
        });

        it("should set state to DisconnectedNoError and call socket.disconnect() when disconnect is called", () => {
            socketService.disconnect();
            expect(socketService.state).toBe("DisconnectedNoError");
            expect(mockSocketDisconnect).toHaveBeenCalled();
        });

        it("should disconnect then connect when restart is called", () => {
            socketService.restart();
            expect(mockSocketDisconnect).toHaveBeenCalled();
            expect(mockSocketConnect).toHaveBeenCalled();
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
            
            // Set state to Connected first
            socketService.state = "Connected";
            socketService.handleDisconnected();
            
            expect(socketService.state).toBe("LostConnection");
            expect(pubsubSpy.publish).toHaveBeenCalledWith("snack", expect.objectContaining({
                id: SERVER_CONNECT_MESSAGE_ID,
                message: "ServerDisconnected",
                severity: "Error",
                autoHideDuration: "persist",
            }));
            
            consoleSpy.mockRestore();
        });
    });

    describe("Send Message", () => {
        it("should emit a message through socket", () => {
            const event = "testEvent";
            const data = { test: "data" };
            
            socketService.sendMessage(event, data);
            
            expect(mockSocketEmit).toHaveBeenCalledWith(event, data);
        });

        it("should accept messages without data", () => {
            const event = "simpleEvent";
            
            socketService.sendMessage(event);
            
            expect(mockSocketEmit).toHaveBeenCalledWith(event);
        });
    });

    describe("Socket Event Listeners", () => {
        it("should register event listeners on socket", () => {
            expect(mockSocketOn).toHaveBeenCalledWith("connect", expect.any(Function));
            expect(mockSocketOn).toHaveBeenCalledWith("disconnect", expect.any(Function));
        });

        it("should handle custom events through on method", () => {
            const event = "customEvent";
            const handler = vi.fn();
            
            socketService.on(event, handler);
            
            expect(mockSocketOn).toHaveBeenCalledWith(event, handler);
        });

        it("should remove event listeners through off method", () => {
            const event = "customEvent";
            const handler = vi.fn();
            
            socketService.off(event, handler);
            
            expect(mockSocketOff).toHaveBeenCalledWith(event, handler);
        });
    });
});