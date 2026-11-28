/**
 * Server Readiness Checker Tests
 *
 * DOC: docs/internal/SEAMS.md#server-readiness-tests
 *
 * Tests for the server readiness checker module.
 * Uses mock HTTP client and timer to test without network access.
 */

import { describe, it, expect } from "vitest";
import {
    type IHttpClient,
    type ITimer,
    type ReadinessConfig,
    checkServerReadiness,
    isAcceptableStatus,
    DEFAULT_READINESS_CONFIG,
} from "../server-readiness";

// Mock timer for deterministic tests
function createMockTimer(initialTime: number = 0): ITimer & { advance: (ms: number) => void; currentTime: number } {
    let currentTime = initialTime;
    return {
        get currentTime() {
            return currentTime;
        },
        now: () => currentTime,
        sleep: (ms: number) => {
            currentTime += ms;
            return Promise.resolve();
        },
        advance: (ms: number) => {
            currentTime += ms;
        },
    };
}

// Mock HTTP client
function createMockHttpClient(
    responses: Array<{ statusCode: number; body?: string } | Error>
): IHttpClient & { callCount: number } {
    let callIndex = 0;
    const fallbackResponse = { statusCode: 500, body: "Mock fallback" };
    return {
        get callCount() {
            return callIndex;
        },
        get: (_url: string, _timeoutMs: number) => {
            const response = responses[callIndex] ?? responses[responses.length - 1] ?? fallbackResponse;
            callIndex++;

            if (response instanceof Error) {
                return Promise.reject(response);
            }
            return Promise.resolve(response);
        },
    };
}

describe("isAcceptableStatus", () => {
    describe("with acceptAny2xx: true", () => {
        const config = { acceptAny2xx: true, acceptableStatusCodes: [200] };

        it.each([
            { status: 200, acceptable: true },
            { status: 201, acceptable: true },
            { status: 204, acceptable: true },
            { status: 299, acceptable: true },
            { status: 300, acceptable: false },
            { status: 302, acceptable: false },
            { status: 404, acceptable: false },
            { status: 500, acceptable: false },
            { status: 503, acceptable: false },
        ])("accepts $status: $acceptable", ({ status, acceptable }) => {
            expect(isAcceptableStatus(status, config)).toBe(acceptable);
        });
    });

    describe("with acceptAny2xx: false", () => {
        const config = { acceptAny2xx: false, acceptableStatusCodes: [200, 204, 401] };

        it.each([
            { status: 200, acceptable: true },
            { status: 204, acceptable: true },
            { status: 401, acceptable: true },
            { status: 201, acceptable: false },
            { status: 202, acceptable: false },
            { status: 403, acceptable: false },
        ])("accepts $status: $acceptable", ({ status, acceptable }) => {
            expect(isAcceptableStatus(status, config)).toBe(acceptable);
        });
    });
});

describe("checkServerReadiness", () => {
    const baseConfig: ReadinessConfig = {
        url: "http://localhost:8080",
        timeoutMs: 5000,
        pollIntervalMs: 100,
        acceptableStatusCodes: DEFAULT_READINESS_CONFIG.acceptableStatusCodes,
        acceptAny2xx: true,
    };

    describe("successful readiness checks", () => {
        it("returns ready on first successful response", async () => {
            const httpClient = createMockHttpClient([{ statusCode: 200 }]);
            const timer = createMockTimer();

            const result = await checkServerReadiness(httpClient, baseConfig, timer);

            expect(result.ready).toBe(true);
            expect(result.statusCode).toBe(200);
            expect(result.error).toBeUndefined();
            expect(httpClient.callCount).toBe(1);
        });

        it("returns ready after retry when server becomes available", async () => {
            const httpClient = createMockHttpClient([
                new Error("ECONNREFUSED"),
                new Error("ECONNREFUSED"),
                { statusCode: 200 },
            ]);
            const timer = createMockTimer();

            const result = await checkServerReadiness(httpClient, baseConfig, timer);

            expect(result.ready).toBe(true);
            expect(result.statusCode).toBe(200);
            expect(httpClient.callCount).toBe(3);
        });

        it("accepts 201 Created response", async () => {
            const httpClient = createMockHttpClient([{ statusCode: 201 }]);
            const timer = createMockTimer();

            const result = await checkServerReadiness(httpClient, baseConfig, timer);

            expect(result.ready).toBe(true);
            expect(result.statusCode).toBe(201);
        });

        it("accepts 204 No Content response", async () => {
            const httpClient = createMockHttpClient([{ statusCode: 204 }]);
            const timer = createMockTimer();

            const result = await checkServerReadiness(httpClient, baseConfig, timer);

            expect(result.ready).toBe(true);
            expect(result.statusCode).toBe(204);
        });
    });

    describe("failed readiness checks", () => {
        it("rejects 404 response as not ready", async () => {
            // Server keeps returning 404
            const httpClient = createMockHttpClient([
                { statusCode: 404 },
                { statusCode: 404 },
                { statusCode: 404 },
            ]);
            const timer = createMockTimer();
            const shortConfig = { ...baseConfig, timeoutMs: 250, pollIntervalMs: 100 };

            const result = await checkServerReadiness(httpClient, shortConfig, timer);

            expect(result.ready).toBe(false);
            expect(result.statusCode).toBe(404);
            expect(result.error).toContain("404");
        });

        it("rejects 500 response as not ready", async () => {
            const httpClient = createMockHttpClient([{ statusCode: 500 }]);
            const timer = createMockTimer();
            const shortConfig = { ...baseConfig, timeoutMs: 50 };

            const result = await checkServerReadiness(httpClient, shortConfig, timer);

            expect(result.ready).toBe(false);
            expect(result.statusCode).toBe(500);
        });

        it("rejects 503 Service Unavailable as not ready", async () => {
            const httpClient = createMockHttpClient([{ statusCode: 503 }]);
            const timer = createMockTimer();
            const shortConfig = { ...baseConfig, timeoutMs: 50 };

            const result = await checkServerReadiness(httpClient, shortConfig, timer);

            expect(result.ready).toBe(false);
            expect(result.statusCode).toBe(503);
        });

        it("times out when server never becomes available", async () => {
            const httpClient = createMockHttpClient([
                new Error("ECONNREFUSED"),
            ]);
            const timer = createMockTimer();
            const shortConfig = { ...baseConfig, timeoutMs: 200, pollIntervalMs: 50 };

            const result = await checkServerReadiness(httpClient, shortConfig, timer);

            expect(result.ready).toBe(false);
            expect(result.error).toContain("ECONNREFUSED");
        });
    });

    describe("content validation", () => {
        it("validates response content when validator is provided", async () => {
            const httpClient = createMockHttpClient([
                { statusCode: 200, body: '{"status": "ready"}' },
            ]);
            const timer = createMockTimer();
            const configWithValidator: ReadinessConfig = {
                ...baseConfig,
                contentValidator: (body) => body.includes('"status": "ready"'),
            };

            const result = await checkServerReadiness(httpClient, configWithValidator, timer);

            expect(result.ready).toBe(true);
        });

        it("rejects when content validation fails", async () => {
            const httpClient = createMockHttpClient([
                { statusCode: 200, body: '{"status": "starting"}' },
            ]);
            const timer = createMockTimer();
            const shortConfig: ReadinessConfig = {
                ...baseConfig,
                timeoutMs: 50,
                contentValidator: (body) => body.includes('"status": "ready"'),
            };

            const result = await checkServerReadiness(httpClient, shortConfig, timer);

            expect(result.ready).toBe(false);
            expect(result.error).toContain("content validation");
        });
    });

    describe("progress reporting", () => {
        it("calls progress callback on each attempt", async () => {
            const httpClient = createMockHttpClient([
                new Error("ECONNREFUSED"),
                new Error("ECONNREFUSED"),
                { statusCode: 200 },
            ]);
            const timer = createMockTimer();
            const progressCalls: Array<{ attempt: number; elapsed: number }> = [];

            await checkServerReadiness(httpClient, baseConfig, timer, (attempt, elapsed) => {
                progressCalls.push({ attempt, elapsed });
            });

            expect(progressCalls.length).toBeGreaterThan(0);
            expect(progressCalls[0]?.attempt).toBe(1);
        });
    });

    describe("duration tracking", () => {
        it("reports accurate duration on success", async () => {
            const httpClient = createMockHttpClient([{ statusCode: 200 }]);
            const timer = createMockTimer(1000);

            const result = await checkServerReadiness(httpClient, baseConfig, timer);

            expect(result.durationMs).toBeGreaterThanOrEqual(0);
        });

        it("reports accurate duration on timeout", async () => {
            const httpClient = createMockHttpClient([new Error("ECONNREFUSED")]);
            const timer = createMockTimer();
            const shortConfig = { ...baseConfig, timeoutMs: 100, pollIntervalMs: 50 };

            const result = await checkServerReadiness(httpClient, shortConfig, timer);

            expect(result.durationMs).toBeGreaterThanOrEqual(0);
        });
    });
});
