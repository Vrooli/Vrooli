/**
 * Mock SocketService for tests
 * Provides minimal functionality to prevent errors when SocketService is used in tests
 */

import { logger } from "../../events/logger.js";
import { type Socket } from "socket.io";

export class MockSocketService {
    public userSockets: Map<string, Set<string>> = new Map();
    public sessionSockets: Map<string, Set<string>> = new Map();
    public io: any = {
        emit: () => { /* no-op */ },
        to: () => ({ emit: () => { /* no-op */ } }),
        close: () => { /* no-op */ },
    };

    /**
     * Mock version always returns false - no users are connected in tests
     */
    public roomHasOpenConnections(roomId: string): boolean {
        return false;
    }

    /**
     * Mock implementation of getHealthDetails
     */
    public getHealthDetails() {
        return {
            connectedClients: 0,
            activeRooms: 0,
            namespaces: 0,
        };
    }

    /**
     * Mock implementation of addSocket
     */
    public addSocket(socket: Socket): void {
        // No-op in tests
    }

    /**
     * Mock implementation of removeSocket
     */
    public removeSocket(socket: Socket): void {
        // No-op in tests
    }

    /**
     * Mock implementation for closing user sockets
     */
    public async closeUserSockets(userId: string): Promise<void> {
        // No-op in tests
    }

    /**
     * Mock implementation for closing session sockets
     */
    public async closeSessionSockets(sessionId: string): Promise<void> {
        // No-op in tests
    }

    /**
     * Mock implementation for emitting to a room
     */
    public emitToRoom(room: string, event: string, data: any): void {
        // No-op in tests
    }
}

/**
 * Initialize the mock SocketService for tests
 * This replaces the real SocketService.get() method with one that returns a mock
 */
export async function initializeSocketServiceMock() {
    const mockInstance = new MockSocketService();
    
    // Import the real SocketService to override its static methods
    const { SocketService } = await import("../../sockets/io.js");
    
    // Override the get() method to return our mock
    (SocketService as any).get = () => mockInstance;
    
    // Override the init() method to return the mock without actually initializing
    (SocketService as any).init = async () => {
        logger.debug("SocketService mock initialized for tests");
        return mockInstance;
    };
    
    // Override the reset/shutdown methods to be no-ops
    (SocketService as any).reset = async () => {
        logger.debug("SocketService mock reset called");
    };
    
    (SocketService as any).shutdown = async () => {
        logger.debug("SocketService mock shutdown called");
    };
    
    logger.debug("SocketService mocked for test environment");
    
    return mockInstance;
}
