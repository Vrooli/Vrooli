import { vi } from "vitest";
import type winston from "winston";
import { logger } from "../../events/logger.js";

/**
 * Utility to capture logs during tests
 * Useful when you need to verify that certain logs are generated
 * 
 * @example
 * const capture = new LogCapture();
 * capture.start();
 * 
 * // Your code that produces logs
 * logger.error('Test error');
 * 
 * const logs = capture.stop();
 * expect(logs).toHaveLength(1);
 * expect(logs[0].level).toBe('error');
 * expect(logs[0].message).toBe('Test error');
 */
export class LogCapture {
    private transport: winston.transport | null = null;
    private logs: Array<winston.LogEntry> = [];
    private originalLevel: string;

    constructor() {
        this.originalLevel = logger.level;
    }

    /**
     * Start capturing logs
     * @param level - Optional log level to capture (defaults to 'debug' to capture everything)
     */
    start(level = "debug"): void {
        // Save and set the logger level
        this.originalLevel = logger.level;
        logger.level = level;

        // Create a stream transport that captures logs
        this.transport = new (logger.transports[0].constructor as any)({
            stream: {
                write: (message: string) => {
                    try {
                        const parsed = JSON.parse(message);
                        this.logs.push(parsed);
                    } catch {
                        // If parsing fails, store raw message
                        this.logs.push({ message, level: "unknown" } as any);
                    }
                },
            },
        });

        // Add our capture transport
        if (this.transport) {
            logger.add(this.transport);
        }
    }

    /**
     * Stop capturing logs and return captured logs
     */
    stop(): Array<winston.LogEntry> {
        if (this.transport) {
            logger.remove(this.transport);
            this.transport = null;
        }

        // Restore original log level
        logger.level = this.originalLevel;

        return this.logs;
    }

    /**
     * Clear captured logs without stopping capture
     */
    clear(): void {
        this.logs = [];
    }

    /**
     * Get logs without stopping capture
     */
    getLogs(): Array<winston.LogEntry> {
        return [...this.logs];
    }

    /**
     * Find logs matching criteria
     */
    findLogs(criteria: {
        level?: string;
        message?: string | RegExp;
        service?: string;
        [key: string]: any;
    }): Array<winston.LogEntry> {
        return this.logs.filter(log => {
            if (criteria.level && log.level !== criteria.level) return false;

            if (criteria.message) {
                if (criteria.message instanceof RegExp) {
                    if (!criteria.message.test(log.message)) return false;
                } else {
                    if (!log.message.includes(criteria.message)) return false;
                }
            }

            // Check other criteria
            for (const [key, value] of Object.entries(criteria)) {
                if (key === "level" || key === "message") continue;
                if ((log as any)[key] !== value) return false;
            }

            return true;
        });
    }
}

/**
 * Temporarily suppress console methods during test execution
 * Useful for tests that produce expected errors or warnings
 * 
 * @example
 * await suppressConsole(async () => {
 *     // Code that produces console output
 *     console.error('This will be suppressed');
 * });
 */
export async function suppressConsole<T>(
    fn: () => T | Promise<T>,
    options: {
        suppressError?: boolean;
        suppressWarn?: boolean;
        suppressLog?: boolean;
        suppressInfo?: boolean;
    } = {},
): Promise<T> {
    const {
        suppressError = true,
        suppressWarn = true,
        suppressLog = true,
        suppressInfo = true,
    } = options;

    // Store original console methods
    const originalConsole = {
        error: console.error,
        warn: console.warn,
        log: console.log,
        info: console.info,
    };

    // Replace with no-op functions
    if (suppressError) console.error = () => { };
    if (suppressWarn) console.warn = () => { };
    if (suppressLog) console.log = () => { };
    if (suppressInfo) console.info = () => { };

    try {
        // Execute the function
        const result = await fn();
        return result;
    } finally {
        // Restore original console methods
        console.error = originalConsole.error;
        console.warn = originalConsole.warn;
        console.log = originalConsole.log;
        console.info = originalConsole.info;
    }
}

/**
 * Mock console methods and capture their output
 * 
 * @example
 * const consoleMock = mockConsole();
 * 
 * console.error('Test error');
 * console.warn('Test warning');
 * 
 * expect(consoleMock.error).toHaveBeenCalledWith('Test error');
 * expect(consoleMock.warn).toHaveBeenCalledWith('Test warning');
 * 
 * consoleMock.restore();
 */
export function mockConsole() {
    const mocks = {
        error: vi.fn(),
        warn: vi.fn(),
        log: vi.fn(),
        info: vi.fn(),
        debug: vi.fn(),
    };

    const originals = {
        error: console.error,
        warn: console.warn,
        log: console.log,
        info: console.info,
        debug: console.debug,
    };

    // Replace console methods with mocks
    console.error = mocks.error;
    console.warn = mocks.warn;
    console.log = mocks.log;
    console.info = mocks.info;
    console.debug = mocks.debug;

    return {
        ...mocks,
        restore: () => {
            console.error = originals.error;
            console.warn = originals.warn;
            console.log = originals.log;
            console.info = originals.info;
            console.debug = originals.debug;
        },
        getOutput: (method: keyof typeof mocks): string[] => {
            return mocks[method].mock.calls.map(call => call.map(arg => String(arg)).join(" "));
        },
    };
}
