/**
 * Shared logger mock configuration for all test files
 * This provides better error logging by properly stringifying objects
 */
import { vi } from "vitest";

// Helper function to format log output
function formatLogOutput(level: string, message: any, ...args: any[]) {
    const timestamp = new Date().toISOString();
    const formattedArgs = args.map(arg => {
        if (arg === null) return "null";
        if (arg === undefined) return "undefined";
        if (typeof arg === "object") {
            try {
                // Handle circular references and format objects nicely
                return JSON.stringify(arg, (key, value) => {
                    if (typeof value === "object" && value !== null) {
                        // Handle circular references
                        const seen = new WeakSet();
                        return JSON.parse(JSON.stringify(value, (k, v) => {
                            if (typeof v === "object" && v !== null) {
                                if (seen.has(v)) return "[Circular Reference]";
                                seen.add(v);
                            }
                            return v;
                        }));
                    }
                    return value;
                }, 2);
            } catch (e) {
                // Fallback for objects that can't be stringified
                return Object.prototype.toString.call(arg);
            }
        }
        return arg;
    });

    // Only output in test if log level allows it
    const testLogLevel = process.env.TEST_LOG_LEVEL || "ERROR";
    const shouldLog = {
        error: ["ERROR", "WARN", "INFO", "DEBUG"].includes(testLogLevel),
        warn: ["WARN", "INFO", "DEBUG"].includes(testLogLevel),
        info: ["INFO", "DEBUG"].includes(testLogLevel),
        debug: ["DEBUG"].includes(testLogLevel),
    };

    if (shouldLog[level as keyof typeof shouldLog]) {
        console.log(`[${timestamp}] [${level.toUpperCase()}] ${message}`, ...formattedArgs.filter(arg => arg !== ""));
    }
}

// Create the mock logger object
export function createMockLogger() {
    return {
        error: vi.fn((message: any, ...args: any[]) => formatLogOutput("error", message, ...args)),
        warn: vi.fn((message: any, ...args: any[]) => formatLogOutput("warn", message, ...args)),
        warning: vi.fn((message: any, ...args: any[]) => formatLogOutput("warn", message, ...args)),
        info: vi.fn((message: any, ...args: any[]) => formatLogOutput("info", message, ...args)),
        debug: vi.fn((message: any, ...args: any[]) => formatLogOutput("debug", message, ...args)),
    };
}

// Export a singleton mock logger for use in vi.mock
export const mockLogger = createMockLogger();
