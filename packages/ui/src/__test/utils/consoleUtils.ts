import { vi } from "vitest";

/**
 * Utility for suppressing expected console errors in tests.
 * Useful for testing error boundaries and other error scenarios.
 */
export class ConsoleSuppressor {
    private originalError: typeof console.error;
    private originalWarn: typeof console.warn;
    private originalLog: typeof console.log;
    private suppressedPatterns: RegExp[];
    private capturedMessages: { type: string; message: string }[] = [];

    constructor(patterns: RegExp[] = []) {
        this.originalError = console.error;
        this.originalWarn = console.warn;
        this.originalLog = console.log;
        this.suppressedPatterns = patterns;
    }

    /**
     * Start suppressing console messages matching the patterns
     */
    suppress() {
        console.error = (...args: any[]) => {
            const message = args.join(" ");
            const shouldSuppress = this.suppressedPatterns.some(pattern => pattern.test(message));
            
            if (shouldSuppress) {
                this.capturedMessages.push({ type: "error", message });
            } else {
                this.originalError(...args);
            }
        };

        console.warn = (...args: any[]) => {
            const message = args.join(" ");
            const shouldSuppress = this.suppressedPatterns.some(pattern => pattern.test(message));
            
            if (shouldSuppress) {
                this.capturedMessages.push({ type: "warn", message });
            } else {
                this.originalWarn(...args);
            }
        };

        console.log = (...args: any[]) => {
            const message = args.join(" ");
            const shouldSuppress = this.suppressedPatterns.some(pattern => pattern.test(message));
            
            if (shouldSuppress) {
                this.capturedMessages.push({ type: "log", message });
            } else {
                this.originalLog(...args);
            }
        };
    }

    /**
     * Restore original console methods
     */
    restore() {
        console.error = this.originalError;
        console.warn = this.originalWarn;
        console.log = this.originalLog;
    }

    /**
     * Get all captured messages
     */
    getCapturedMessages() {
        return this.capturedMessages;
    }

    /**
     * Clear captured messages
     */
    clearCapturedMessages() {
        this.capturedMessages = [];
    }

    /**
     * Add additional patterns to suppress
     */
    addPatterns(patterns: RegExp[]) {
        this.suppressedPatterns.push(...patterns);
    }
}

/**
 * Helper function to suppress console errors for a specific test
 * @param testFn The test function to run
 * @param patterns Optional patterns to suppress (in addition to defaults)
 */
export async function withSuppressedConsole(
    testFn: () => void | Promise<void>,
    patterns: RegExp[] = [],
) {
    const defaultPatterns = [
        /Error caught by Error Boundary:/,
        /The above error occurred in the <.*> component:/,
        /React will try to recreate this component tree/,
        /componentStack:/,
        /Unsupported input type:/,
        /Cannot read properties of null \(reading 'placeholder'\)/,
        /Cannot read properties of undefined \(reading 'placeholder'\)/,
    ];

    const suppressor = new ConsoleSuppressor([...defaultPatterns, ...patterns]);
    
    try {
        suppressor.suppress();
        await testFn();
    } finally {
        suppressor.restore();
    }
}

/**
 * Mock console methods for a test suite
 * @param patterns Patterns to suppress
 * @returns Object with setup and teardown functions
 */
export function mockConsoleForSuite(patterns: RegExp[] = []) {
    let suppressor: ConsoleSuppressor;

    return {
        setup: () => {
            suppressor = new ConsoleSuppressor(patterns);
            suppressor.suppress();
        },
        teardown: () => {
            if (suppressor) {
                suppressor.restore();
            }
        },
        getSuppressor: () => suppressor,
    };
}

/**
 * Temporarily mock console.error to expect certain error messages
 * @param expectedErrors Array of expected error patterns
 * @returns Object with mock and verification functions
 */
export function expectConsoleErrors(expectedErrors: RegExp[]) {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    
    return {
        verify: () => {
            const calls = errorSpy.mock.calls;
            expectedErrors.forEach(pattern => {
                const found = calls.some(call => {
                    const message = call.join(" ");
                    return pattern.test(message);
                });
                
                if (!found) {
                    throw new Error(`Expected console.error to be called with pattern: ${pattern}`);
                }
            });
        },
        restore: () => {
            errorSpy.mockRestore();
        },
        getCallCount: () => errorSpy.mock.calls.length,
        getCalls: () => errorSpy.mock.calls,
    };
}
