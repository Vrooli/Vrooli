const mockLogger = {
    error: jest.fn(),
    warn: jest.fn(),
    info: jest.fn(),
    http: jest.fn(),
    verbose: jest.fn(),
    debug: jest.fn(),
    silly: jest.fn(),
    log: jest.fn(), // This is a generic logging function provided by winston
    add: jest.fn(), // Mocking the `add` method for adding new transports
};

export const logger = mockLogger;
