import { vi } from "vitest";

/**
 * Factory function to create a mock socket instance
 */
export function createMockSocket(overrides?: Partial<any>) {
    const mockSocket = {
        on: vi.fn(),
        off: vi.fn(),
        emit: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        connected: false,
        id: "mock-socket-id",
        ...overrides,
    };
    
    return mockSocket;
}

/**
 * Default mock for socket.io-client
 */
export const socketIOClientMock = {
    io: vi.fn(() => createMockSocket()),
};

// For tests that need to control the socket instance
export const mockSocket = createMockSocket();
