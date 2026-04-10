/**
 * Async Test Helpers
 *
 * DOC: docs/internal/SEAMS.md#async-helpers
 *
 * Provides utilities for testing asynchronous code.
 */

import { vi } from "vitest";

/**
 * Wait for a specified number of milliseconds.
 * Useful for tests that need to wait for debounced/throttled operations.
 */
export function delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Wait for the next event loop tick.
 * Useful for waiting for microtasks to complete.
 */
export function nextTick(): Promise<void> {
    return new Promise((resolve) => setImmediate(resolve));
}

/**
 * Wait for all pending promises to resolve.
 * Flushes the microtask queue.
 */
export async function flushPromises(): Promise<void> {
    await new Promise((resolve) => setImmediate(resolve));
}

/**
 * Create a deferred promise for testing async flows.
 */
export function createDeferred<T>(): {
    promise: Promise<T>;
    resolve: (value: T) => void;
    reject: (error: Error) => void;
} {
    let resolve!: (value: T) => void;
    let reject!: (error: Error) => void;

    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });

    return { promise, resolve, reject };
}

/**
 * Wait for a condition to become true.
 * Polls the condition at regular intervals.
 *
 * @param condition - Function that returns true when condition is met
 * @param options - Configuration options
 * @throws Error if timeout is reached before condition is met
 */
export async function waitFor(
    condition: () => boolean | Promise<boolean>,
    options: { timeout?: number; interval?: number } = {}
): Promise<void> {
    const { timeout = 5000, interval = 50 } = options;
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
        if (await condition()) {
            return;
        }
        await delay(interval);
    }

    throw new Error(`waitFor timed out after ${timeout}ms`);
}

/**
 * Create a mock timer control object.
 * Wraps Vitest's fake timers with a cleaner API.
 */
export function useFakeTimers() {
    vi.useFakeTimers();

    return {
        /**
         * Advance time by specified milliseconds.
         */
        advance: (ms: number) => vi.advanceTimersByTime(ms),

        /**
         * Run all pending timers.
         */
        runAll: () => vi.runAllTimers(),

        /**
         * Run only pending timers (not those scheduled by running timers).
         */
        runPending: () => vi.runOnlyPendingTimers(),

        /**
         * Set the current time.
         */
        setTime: (date: Date | number) => vi.setSystemTime(date),

        /**
         * Get the current mocked time.
         */
        getTime: () => Date.now(),

        /**
         * Restore real timers.
         */
        restore: () => vi.useRealTimers(),
    };
}

/**
 * Mock fetch responses for testing HTTP clients.
 */
export function mockFetchResponse(
    response: {
        ok?: boolean;
        status?: number;
        statusText?: string;
        json?: unknown;
        text?: string;
        headers?: Record<string, string>;
    } = {}
): Response {
    const {
        ok = true,
        status = 200,
        statusText = "OK",
        json = {},
        text = "",
        headers = {},
    } = response;

    return {
        ok,
        status,
        statusText,
        headers: new Headers(headers),
        json: vi.fn().mockResolvedValue(json),
        text: vi.fn().mockResolvedValue(text || JSON.stringify(json)),
        blob: vi.fn().mockResolvedValue(new Blob()),
        arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(0)),
        clone: vi.fn().mockReturnThis(),
        body: null,
        bodyUsed: false,
        redirected: false,
        type: "basic" as ResponseType,
        url: "",
        formData: vi.fn().mockResolvedValue(new FormData()),
        bytes: vi.fn().mockResolvedValue(new Uint8Array(0)),
    } as Response;
}

/**
 * Create a mock fetch function that returns configured responses.
 */
export function createMockFetch(
    responses: Record<string, Response | (() => Response)> = {}
) {
    const defaultResponse = mockFetchResponse();

    return vi.fn(async (url: string | URL, _init?: RequestInit): Promise<Response> => {
        const urlString = url.toString();

        for (const [pattern, response] of Object.entries(responses)) {
            if (urlString.includes(pattern)) {
                return typeof response === "function" ? response() : response;
            }
        }

        return defaultResponse;
    });
}

/**
 * Capture console output during test execution.
 */
export function captureConsole() {
    const logs: string[] = [];
    const warns: string[] = [];
    const errors: string[] = [];

    const originalLog = console.log;
    const originalWarn = console.warn;
    const originalError = console.error;

    console.log = (...args: unknown[]) => {
        logs.push(args.map(String).join(" "));
    };
    console.warn = (...args: unknown[]) => {
        warns.push(args.map(String).join(" "));
    };
    console.error = (...args: unknown[]) => {
        errors.push(args.map(String).join(" "));
    };

    return {
        logs,
        warns,
        errors,
        restore: () => {
            console.log = originalLog;
            console.warn = originalWarn;
            console.error = originalError;
        },
    };
}
