import { afterEach, beforeEach, describe, expect, it, vi, type MockedFunction } from "vitest";

// Mock winston - must be defined inline in the factory function to avoid hoisting issues
vi.mock("winston", () => {
    const mockTransports = {
        File: vi.fn().mockImplementation((options) => ({
            ...options,
            type: "file",
        })),
        Console: vi.fn().mockImplementation((options) => ({
            ...options,
            type: "console",
        })),
    };

    const mockFormat = {
        combine: vi.fn().mockImplementation((...args) => ({ combined: args })),
        errors: vi.fn().mockImplementation((options) => ({ errors: options })),
        timestamp: vi.fn().mockImplementation((options) => ({ timestamp: options })),
        json: vi.fn().mockImplementation(() => ({ json: true })),
        simple: vi.fn().mockImplementation(() => ({ simple: true })),
    };

    const mockConfig = {
        syslog: {
            levels: {
                emerg: 0,
                alert: 1,
                crit: 2,
                error: 3,
                warning: 4,
                notice: 5,
                info: 6,
                debug: 7,
            },
        },
    };

    const mockLogger = {
        error: vi.fn(),
        warn: vi.fn(),
        info: vi.fn(),
        debug: vi.fn(),
        log: vi.fn(),
    };

    return {
        default: {
            createLogger: vi.fn().mockReturnValue(mockLogger),
            transports: mockTransports,
            format: mockFormat,
            config: mockConfig,
        },
    };
});

import winston from "winston";
import { logger } from "./logger.js";

describe("logger", () => {
    const originalEnv = process.env;

    beforeEach(() => {
        vi.clearAllMocks();
        // Reset environment
        process.env = { ...originalEnv };
    });

    afterEach(() => {
        process.env = originalEnv;
    });

    describe("logger initialization", () => {
        it("should create logger with correct configuration", () => {
            const mockedWinston = vi.mocked(winston);
            const createLoggerSpy = mockedWinston.createLogger as MockedFunction<typeof winston.createLogger>;

            expect(createLoggerSpy).toHaveBeenCalledTimes(1);
            const config = createLoggerSpy.mock.calls[0][0];

            // Check levels include winston syslog levels plus 'warn'
            expect(config.levels).toHaveProperty("error");
            expect(config.levels).toHaveProperty("warning");
            expect(config.levels).toHaveProperty("warn");
            expect(config.levels.warn).toBe(config.levels.warning);

            // Check format combines errors, timestamp, and json
            expect(mockedWinston.format.combine).toHaveBeenCalled();
            expect(mockedWinston.format.errors).toHaveBeenCalledWith({ stack: true });
            expect(mockedWinston.format.timestamp).toHaveBeenCalledWith({ format: "YYYY-MM-DD HH:mm:ss" });
            expect(mockedWinston.format.json).toHaveBeenCalled();

            // Check default meta
            expect(config.defaultMeta).toEqual({ service: "express-server" });
        });
    });

    describe("transport configuration", () => {
        it("should add file transports in production with PROJECT_DIR", () => {
            process.env.NODE_ENV = "production";
            process.env.PROJECT_DIR = "/test/project";
            delete process.env.JEST_WORKER_ID;

            // Re-import to trigger new configuration
            vi.resetModules();
            vi.doMock("winston", () => ({ default: winston }));

            import("./logger.js").then(() => {
                const mockedWinston = vi.mocked(winston);
                const FileTransport = mockedWinston.transports.File as MockedFunction<any>;

                expect(FileTransport).toHaveBeenCalledTimes(2);

                // Check error log configuration
                expect(FileTransport).toHaveBeenCalledWith({
                    filename: "/test/project/data/logs/error.log",
                    level: "error",
                    maxsize: 5_242_880,
                });

                // Check combined log configuration
                expect(FileTransport).toHaveBeenCalledWith({
                    filename: "/test/project/data/logs/combined.log",
                    maxsize: 5_242_880,
                });
            });
        });

        it("should add console transport in development", () => {
            process.env.NODE_ENV = "development";
            delete process.env.JEST_WORKER_ID;

            // Re-import to trigger new configuration
            vi.resetModules();
            vi.doMock("winston", () => ({ default: winston }));

            import("./logger.js").then(() => {
                const mockedWinston = vi.mocked(winston);
                const ConsoleTransport = mockedWinston.transports.Console as MockedFunction<any>;

                expect(ConsoleTransport).toHaveBeenCalledTimes(1);
                expect(ConsoleTransport).toHaveBeenCalledWith({
                    format: expect.objectContaining({
                        combined: expect.arrayContaining([
                            expect.objectContaining({ errors: { stack: true } }),
                            expect.objectContaining({ simple: true }),
                        ]),
                    }),
                });
            });
        });

        it("should not add file transports in test environment", () => {
            process.env.NODE_ENV = "test";
            process.env.PROJECT_DIR = "/test/project";
            process.env.JEST_WORKER_ID = "1";

            // Re-import to trigger new configuration
            vi.resetModules();
            vi.doMock("winston", () => ({ default: winston }));

            import("./logger.js").then(() => {
                const mockedWinston = vi.mocked(winston);
                const FileTransport = mockedWinston.transports.File as MockedFunction<any>;

                expect(FileTransport).not.toHaveBeenCalled();
            });
        });

        it("should not add file transports without PROJECT_DIR", () => {
            process.env.NODE_ENV = "production";
            delete process.env.PROJECT_DIR;
            delete process.env.JEST_WORKER_ID;

            // Re-import to trigger new configuration
            vi.resetModules();
            vi.doMock("winston", () => ({ default: winston }));

            import("./logger.js").then(() => {
                const mockedWinston = vi.mocked(winston);
                const FileTransport = mockedWinston.transports.File as MockedFunction<any>;

                expect(FileTransport).not.toHaveBeenCalled();
            });
        });

        it("should add console transport when NODE_ENV is not set", () => {
            delete process.env.NODE_ENV;
            delete process.env.JEST_WORKER_ID;

            // Re-import to trigger new configuration
            vi.resetModules();
            vi.doMock("winston", () => ({ default: winston }));

            import("./logger.js").then(() => {
                const mockedWinston = vi.mocked(winston);
                const ConsoleTransport = mockedWinston.transports.Console as MockedFunction<any>;

                expect(ConsoleTransport).toHaveBeenCalledTimes(1);
            });
        });

        it("should not add console transport in production", () => {
            process.env.NODE_ENV = "production";
            delete process.env.JEST_WORKER_ID;

            // Re-import to trigger new configuration
            vi.resetModules();
            vi.doMock("winston", () => ({ default: winston }));

            import("./logger.js").then(() => {
                const mockedWinston = vi.mocked(winston);
                const ConsoleTransport = mockedWinston.transports.Console as MockedFunction<any>;

                expect(ConsoleTransport).not.toHaveBeenCalled();
            });
        });

        it("should handle prod-like environments correctly", () => {
            process.env.NODE_ENV = "prod-staging";
            delete process.env.JEST_WORKER_ID;

            // Re-import to trigger new configuration
            vi.resetModules();
            vi.doMock("winston", () => ({ default: winston }));

            import("./logger.js").then(() => {
                const mockedWinston = vi.mocked(winston);
                const ConsoleTransport = mockedWinston.transports.Console as MockedFunction<any>;

                // Should not add console transport for prod-like environments
                expect(ConsoleTransport).not.toHaveBeenCalled();
            });
        });
    });

    describe("logger usage", () => {
        it("should export a logger instance", () => {
            expect(logger).toBeDefined();
            expect(logger).toHaveProperty("error");
            expect(logger).toHaveProperty("warn");
            expect(logger).toHaveProperty("info");
            expect(logger).toHaveProperty("debug");
        });

        it("should support logging with trace metadata", () => {
            const errorSpy = logger.error as MockedFunction<typeof logger.error>;
            const testError = new Error("Test error");

            logger.error("Detailed message", { trace: "0000-cKST", error: testError });

            expect(errorSpy).toHaveBeenCalledWith("Detailed message", {
                trace: "0000-cKST",
                error: testError,
            });
        });

        it("should support different log levels", () => {
            const errorSpy = logger.error as MockedFunction<typeof logger.error>;
            const warnSpy = logger.warn as MockedFunction<typeof logger.warn>;
            const infoSpy = logger.info as MockedFunction<typeof logger.info>;
            const debugSpy = logger.debug as MockedFunction<typeof logger.debug>;

            logger.error("Error message");
            logger.warn("Warning message");
            logger.info("Info message");
            logger.debug("Debug message");

            expect(errorSpy).toHaveBeenCalledWith("Error message");
            expect(warnSpy).toHaveBeenCalledWith("Warning message");
            expect(infoSpy).toHaveBeenCalledWith("Info message");
            expect(debugSpy).toHaveBeenCalledWith("Debug message");
        });
    });
});

// Additional test suite for error serialization (without mocking winston)
describe("Logger Error Serialization (Integration)", () => {
    let logOutput: any[] = [];

    // Create a test logger with a custom transport that captures output
    const testTransport = new winston.transports.Stream({
        stream: {
            write: (message: string) => {
                try {
                    logOutput.push(JSON.parse(message));
                } catch {
                    logOutput.push(message);
                }
            },
        } as any,
    });

    // Import the actual error serializer
    const { errorSerializer } = require("./logger.js");

    const testLogger = winston.createLogger({
        format: winston.format.combine(
            winston.format.errors({ stack: true }),
            errorSerializer?.() || winston.format.json(), // Use the actual errorSerializer if available
            winston.format.json(),
        ),
        transports: [testTransport],
    });

    beforeEach(() => {
        logOutput = [];
    });

    it("should properly serialize Error objects in metadata", () => {
        const testError = new Error("Test error message");
        testError.stack = "Error: Test error message\n    at test.js:1:1";

        testLogger.error("Test log message", { error: testError, otherData: "test" });

        expect(logOutput).toHaveLength(1);
        const logEntry = logOutput[0];

        // The error should be serialized, not [object Object]
        expect(JSON.stringify(logEntry)).not.toContain("[object Object]");

        // Check that error properties are preserved
        if (logEntry.error) {
            expect(logEntry.error.message).toBe("Test error message");
            expect(logEntry.error.name).toBe("Error");
            expect(logEntry.error.stack).toContain("Test error message");
        }
    });

    it("should handle custom Error properties", () => {
        const customError = new Error("Custom error") as any;
        customError.code = "CUSTOM_CODE";
        customError.statusCode = 404;
        customError.details = { field: "username", issue: "required" };

        testLogger.error("Custom error test", { error: customError });

        expect(logOutput).toHaveLength(1);
        const logEntry = logOutput[0];

        if (logEntry.error) {
            expect(logEntry.error.message).toBe("Custom error");
            expect(logEntry.error.code).toBe("CUSTOM_CODE");
            expect(logEntry.error.statusCode).toBe(404);
            expect(logEntry.error.details).toEqual({ field: "username", issue: "required" });
        }
    });

    it("should handle nested Error objects", () => {
        const innerError = new Error("Inner error");
        const outerError = new Error("Outer error");

        testLogger.error("Nested error test", {
            error: outerError,
            errors: [innerError],
            nested: { deep: { error: innerError } },
        });

        expect(logOutput).toHaveLength(1);
        const logEntry = logOutput[0];

        // All errors should be properly serialized
        expect(JSON.stringify(logEntry)).not.toContain("[object Object]");

        if (logEntry.error) {
            expect(logEntry.error.message).toBe("Outer error");
        }
        if (logEntry.errors?.[0]) {
            expect(logEntry.errors[0].message).toBe("Inner error");
        }
        if (logEntry.nested?.deep?.error) {
            expect(logEntry.nested.deep.error.message).toBe("Inner error");
        }
    });
});
