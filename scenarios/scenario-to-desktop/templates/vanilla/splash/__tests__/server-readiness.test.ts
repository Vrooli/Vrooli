/**
 * Server Readiness Checker Tests
 *
 * DOC: docs/internal/SEAMS.md#server-readiness-tests
 *
 * Tests for the server readiness checker module.
 * Uses mock HTTP client and timer to test without network access.
 */

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
        sleep: async (ms: number) => {
            currentTime += ms;
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
    return {
        get callCount() {
            return callIndex;
        },
        get: async (_url: string, _timeoutMs: number) => {
            const response = responses[callIndex] ?? responses[responses.length - 1];
            callIndex++;

            if (response instanceof Error) {
                throw response;
            }
            return response;
        },
    };
}

describe("isAcceptableStatus", () => {
    describe("with acceptAny2xx: true", () => {
        const config = { acceptAny2xx: true, acceptableStatusCodes: [200] };

        it("accepts 200", () => {
            expect(isAcceptableStatus(200, config)).toBe(true);
        });

        it("accepts 201", () => {
            expect(isAcceptableStatus(201, config)).toBe(true);
        });

        it("accepts 204", () => {
            expect(isAcceptableStatus(204, config)).toBe(true);
        });

        it("accepts 299", () => {
            expect(isAcceptableStatus(299, config)).toBe(true);
        });

        it("rejects 300", () => {
            expect(isAcceptableStatus(300, config)).toBe(false);
        });

        it("rejects 404", () => {
            expect(isAcceptableStatus(404, config)).toBe(false);
        });

        it("rejects 500", () => {
            expect(isAcceptableStatus(500, config)).toBe(false);
        });

        it("rejects 503", () => {
            expect(isAcceptableStatus(503, config)).toBe(false);
        });
    });

    describe("with acceptAny2xx: false", () => {
        const config = { acceptAny2xx: false, acceptableStatusCodes: [200, 204] };

        it("accepts status codes in the list", () => {
            expect(isAcceptableStatus(200, config)).toBe(true);
            expect(isAcceptableStatus(204, config)).toBe(true);
        });

        it("rejects status codes not in the list", () => {
            expect(isAcceptableStatus(201, config)).toBe(false);
            expect(isAcceptableStatus(202, config)).toBe(false);
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
            expect(progressCalls[0].attempt).toBe(1);
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

describe("regression: 404 responses should not be considered ready", () => {
    /**
     * This test documents the bug that was fixed:
     * Previously, ANY HTTP response (including 404) was considered "ready"
     * because the old checkServerReady only checked for a response, not the status code.
     */
    it("rejects 404 even though the server responded", async () => {
        const httpClient = createMockHttpClient([{ statusCode: 404, body: "Not Found" }]);
        const timer = createMockTimer();

        const result = await checkServerReadiness(httpClient, {
            ...DEFAULT_READINESS_CONFIG,
            url: "http://localhost:8080",
            timeoutMs: 50,
        }, timer);

        // This is the key assertion: 404 is NOT ready, even though the server responded
        expect(result.ready).toBe(false);
        expect(result.statusCode).toBe(404);
        expect(result.error).toContain("404");
    });
});
