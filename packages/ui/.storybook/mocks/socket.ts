// Mock Socket.io for Storybook to prevent connection errors

// Mock socket instance
export const mockSocket = {
    connect: () => {},
    disconnect: () => {},
    on: () => {},
    off: () => {},
    emit: () => {},
    once: () => {},
    removeAllListeners: () => {},
    connected: false,
    id: 'mock-socket-id',
};

// Mock io function
export const io = () => {
    console.info('Socket.io mock: Returning mocked socket in Storybook');
    return mockSocket;
};

// Export default for ESM imports
export default io;