/**
 * Runtime Control Client Tests
 *
 * DOC: docs/internal/SEAMS.md#control-client-tests
 *
 * Tests for the runtime control API client.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import type {
    IRuntimeHttpClient,
    IRuntimeFileSystem,
    ITimer,
    RuntimeControlConfig,
} from "../types";
import { createRuntimeControlClient } from "../control-client";

// ===== Mock Factories =====

interface MockHttpClient {
    _requests: Array<{ url: string; method: string; headers: Record<string, string>; body: string | undefined }>;
    _response: { ok: boolean; status: number; text: string; json: unknown };
    request: IRuntimeHttpClient["request"];
}

function createMockHttpClient(): MockHttpClient {
    const requests: Array<{ url: string; method: string; headers: Record<string, string>; body: string | undefined }> = [];

    const mockClient: MockHttpClient = {
        _requests: requests,
        _response: { ok: true, status: 200, text: "", json: {} as unknown },
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        request: null as any,
    };

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockClient.request = vi.fn(async (
        url: string,
        opts?: { method?: string; headers?: Record<string, string>; body?: string }
    ) => {
        requests.push({
            url,
            method: opts?.method ?? "GET",
            headers: opts?.headers ?? {},
            body: opts?.body,
        });
        return {
            ok: mockClient._response.ok,
            status: mockClient._response.status,
            text: async () => mockClient._response.text,
            json: async () => mockClient._response.json,
        };
    }) as any;

    return mockClient;
}

function createMockFileSystem(): IRuntimeFileSystem & {
    _files: Map<string, string>;
} {
    const files = new Map<string, string>();
    return {
        _files: files,
        readFile: vi.fn(async (path: string, _encoding: "utf-8") => {
            const content = files.get(path);
            if (content === undefined) {
                throw new Error(`ENOENT: no such file: ${path}`);
            }
            return content;
        }),
        access: vi.fn(async (path: string) => {
            if (!files.has(path)) {
                throw new Error(`ENOENT: no such file: ${path}`);
            }
        }),
        stat: vi.fn(async (path: string) => {
            if (!files.has(path)) {
                throw new Error(`ENOENT: no such file: ${path}`);
            }
            return { isFile: () => true, isDirectory: () => false };
        }),
    };
}

function createMockTimer(): ITimer & { _currentTime: number; _setTime(ms: number): void } {
    let currentTime = 0;
    return {
        get _currentTime() { return currentTime; },
        _setTime(ms: number) { currentTime = ms; },
        now: vi.fn(() => currentTime),
        sleep: vi.fn(async (ms: number) => {
            currentTime += ms;
        }),
    };
}

function createTestConfig(overrides?: Partial<RuntimeControlConfig>): RuntimeControlConfig {
    return {
        enabled: true,
        host: "127.0.0.1",
        port: 47710,
        tokenPath: "/mock/token",
        telemetryUploadUrl: "https://example.com/telemetry",
        logLines: 100,
        ...overrides,
    };
}

// ===== Tests =====

describe("createRuntimeControlClient", () => {
    let http: MockHttpClient;
    let fs: ReturnType<typeof createMockFileSystem>;
    let timer: ReturnType<typeof createMockTimer>;
    let config: RuntimeControlConfig;

    beforeEach(() => {
        http = createMockHttpClient();
        fs = createMockFileSystem();
        timer = createMockTimer();
        config = createTestConfig();
        fs._files.set(config.tokenPath, "test-token-123");
    });

    describe("request", () => {
        it("builds correct URL", async () => {
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.request("/readyz");

            expect(http._requests[0]?.url).toBe("http://127.0.0.1:47710/readyz");
        });

        it("includes auth token in header", async () => {
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.request("/readyz");

            expect(http._requests[0]?.headers.Authorization).toBe("Bearer test-token-123");
        });

        it("skips auth header for healthz endpoint", async () => {
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.request("/healthz");

            expect(http._requests[0]?.headers.Authorization).toBeUndefined();
        });

        it("handles missing token gracefully", async () => {
            fs._files.delete(config.tokenPath);
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.request("/readyz");

            expect(http._requests[0]?.headers.Authorization).toBeUndefined();
        });

        it("sends POST with JSON body", async () => {
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.request("/secrets", { method: "POST", body: { key: "value" } });

            expect(http._requests[0]?.method).toBe("POST");
            expect(http._requests[0]?.headers["Content-Type"]).toBe("application/json");
            expect(http._requests[0]?.body).toBe('{"key":"value"}');
        });

        it("returns JSON response by default", async () => {
            http._response.json = { ready: true };
            const client = createRuntimeControlClient(http, fs, timer, config);

            const result = await client.request<{ ready: boolean }>("/readyz");

            expect(result).toEqual({ ready: true });
        });

        it("returns text response when expectText is true", async () => {
            http._response.text = "log line 1\nlog line 2";
            const client = createRuntimeControlClient(http, fs, timer, config);

            const result = await client.request("/logs", { expectText: true });

            expect(result).toBe("log line 1\nlog line 2");
        });

        it("throws when runtime control is disabled", async () => {
            config.enabled = false;
            const client = createRuntimeControlClient(http, fs, timer, config);

            await expect(client.request("/readyz")).rejects.toThrow("runtime control not enabled");
        });

        it("throws on HTTP error", async () => {
            http._response = { ok: false, status: 500, text: "Internal Server Error", json: {} };
            const client = createRuntimeControlClient(http, fs, timer, config);

            await expect(client.request("/readyz")).rejects.toThrow("runtime request failed (500)");
        });
    });

    describe("waitForHealth", () => {
        it("returns immediately when health check succeeds", async () => {
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.waitForHealth(5000);

            expect(http.request).toHaveBeenCalledTimes(1);
        });

        it("retries until health check succeeds", async () => {
            let attempts = 0;
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            http.request = vi.fn(async () => {
                attempts++;
                if (attempts < 3) {
                    throw new Error("Connection refused");
                }
                return { ok: true, status: 200, text: async () => "", json: async () => ({}) };
            }) as any;
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.waitForHealth(5000);

            expect(attempts).toBe(3);
        });

        it("throws on timeout", async () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            http.request = vi.fn(async () => {
                throw new Error("Connection refused");
            }) as any;
            // Make time advance past deadline on each sleep
            timer._setTime(0);
            timer.sleep = vi.fn(async () => {
                timer._setTime(timer._currentTime + 1000);
            });

            const client = createRuntimeControlClient(http, fs, timer, config);

            await expect(client.waitForHealth(500)).rejects.toThrow("did not respond before timeout");
        });

        it("skips waiting when runtime control is disabled", async () => {
            config.enabled = false;
            const client = createRuntimeControlClient(http, fs, timer, config);

            await client.waitForHealth(5000);

            expect(http.request).not.toHaveBeenCalled();
        });
    });

    describe("collectDiagnostics", () => {
        beforeEach(() => {
            // Set up default responses for diagnostic endpoints
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            http.request = vi.fn(async (url: string) => {
                http._requests.push({ url, method: "GET", headers: {}, body: undefined });

                if (url.includes("/readyz")) {
                    return {
                        ok: true,
                        status: 200,
                        text: async () => "",
                        json: async () => ({ ready: true, details: { svc1: { ready: true } } }),
                    };
                }
                if (url.includes("/ports")) {
                    return {
                        ok: true,
                        status: 200,
                        text: async () => "",
                        json: async () => ({ services: { svc1: { http: 8080 } } }),
                    };
                }
                if (url.includes("/telemetry")) {
                    return {
                        ok: true,
                        status: 200,
                        text: async () => "",
                        json: async () => ({ path: "/var/log/telemetry.jsonl", upload_url: "https://upload.example.com" }),
                    };
                }
                if (url.includes("/logs/tail")) {
                    return {
                        ok: true,
                        status: 200,
                        text: async () => "log line 1\nlog line 2",
                        json: async () => ({}),
                    };
                }
                return { ok: true, status: 200, text: async () => "", json: async () => ({}) };
            }) as any;
        });

        it("fetches readyz, ports, and telemetry", async () => {
            const client = createRuntimeControlClient(http, fs, timer, config);

            const diagnostics = await client.collectDiagnostics();

            expect(diagnostics.ready).toEqual({ ready: true, details: { svc1: { ready: true } } });
            expect(diagnostics.ports).toEqual({ services: { svc1: { http: 8080 } } });
            expect(diagnostics.telemetryPath).toBe("/var/log/telemetry.jsonl");
            expect(diagnostics.telemetryUploadUrl).toBe("https://upload.example.com");
        });

        it("fetches logs for discovered services", async () => {
            const client = createRuntimeControlClient(http, fs, timer, config);

            const diagnostics = await client.collectDiagnostics();

            // Should have fetched logs for svc1
            const logRequest = http._requests.find((r) => r.url.includes("/logs/tail"));
            expect(logRequest).toBeDefined();
            expect(logRequest?.url).toContain("serviceId=svc1");
            expect(diagnostics.logs.svc1).toBe("log line 1\nlog line 2");
        });
    });

    describe("validate", () => {
        it("returns validation response on success", async () => {
            http._response.json = { valid: true };
            const client = createRuntimeControlClient(http, fs, timer, config);

            const result = await client.validate();

            expect(result).toEqual({ valid: true });
        });

        it("returns null when runtime control is disabled", async () => {
            config.enabled = false;
            const client = createRuntimeControlClient(http, fs, timer, config);

            const result = await client.validate();

            expect(result).toBeNull();
        });

        it("returns null on request error", async () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            http.request = vi.fn(async () => {
                throw new Error("Connection refused");
            }) as any;
            const client = createRuntimeControlClient(http, fs, timer, config);

            const result = await client.validate();

            expect(result).toBeNull();
        });
    });
});
