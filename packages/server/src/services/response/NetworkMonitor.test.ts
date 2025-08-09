import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { NetworkMonitor } from "./NetworkMonitor.js";
import * as fetch from "node-fetch";
import dns from "dns";

// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-08-05

// Mock fetch
vi.mock("node-fetch", () => ({
    default: vi.fn(),
}));

// Mock dns
vi.mock("dns", async (importOriginal) => {
    const actual = await importOriginal();
    return {
        ...actual,
        default: {
            resolve4: vi.fn((hostname, callback) => {
                // Default mock implementation for callback style
                callback(null, ["8.8.8.8"]);
            }),
        },
        resolve4: vi.fn((hostname, callback) => {
            // Also mock the named export for direct access pattern
            callback(null, ["8.8.8.8"]);
        }),
    };
});

// Mock logger
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
    },
}));

describe("NetworkMonitor", () => {
    let networkMonitor: NetworkMonitor;

    beforeEach(() => {
        // Reset singleton instance
        (NetworkMonitor as any).instance = undefined;
        networkMonitor = NetworkMonitor.getInstance();
        vi.clearAllMocks();
    });

    afterEach(() => {
        networkMonitor.stop();
        
        // Reset internal state to prevent cross-test contamination
        (networkMonitor as any).state = {
            isOnline: true,
            connectivity: "online",
            lastChecked: new Date(),
            cloudServicesReachable: true,
            localServicesReachable: true,
        };
        (networkMonitor as any).lastUpdateTime = 0;
        
        vi.clearAllMocks();
    });

    describe("getInstance", () => {
        it("should return singleton instance", () => {
            const instance1 = NetworkMonitor.getInstance();
            const instance2 = NetworkMonitor.getInstance();
            expect(instance1).toBe(instance2);
        });
    });

    describe("start/stop", () => {
        it("should start monitoring", () => {
            networkMonitor.start();
            expect(networkMonitor["updateInterval"]).toBeDefined();
        });

        it("should stop monitoring", () => {
            networkMonitor.start();
            networkMonitor.stop();
            expect(networkMonitor["updateInterval"]).toBeNull();
        });

        it("should not start multiple intervals", () => {
            networkMonitor.start();
            const firstInterval = networkMonitor["updateInterval"];
            networkMonitor.start();
            expect(networkMonitor["updateInterval"]).toBe(firstInterval);
        });
    });

    describe("getState", () => {
        it("should return initial state", async () => {
            // Setup successful mocks before calling getState()
            vi.mocked(fetch.default)
                .mockResolvedValueOnce({ ok: true, status: 200 } as any) // OpenRouter
                .mockResolvedValueOnce({ ok: true, status: 200 } as any) // Cloudflare  
                .mockResolvedValueOnce({ ok: true, status: 200 } as any); // Ollama

            const state = await networkMonitor.getState();
            expect(state).toEqual({
                isOnline: true,
                connectivity: "online",
                lastChecked: expect.any(Date),
                cloudServicesReachable: true,
                localServicesReachable: true,
            });
        });

        it("should use cached state when fresh", async () => {
            // Set a fresh update time
            networkMonitor["lastUpdateTime"] = Date.now();
            
            const state1 = await networkMonitor.getState();
            const state2 = await networkMonitor.getState();
            
            expect(state1).toBe(state2);
        });
    });

    describe("forceUpdate", () => {
        it("should force an immediate update", async () => {
            // Mock successful DNS resolution
            vi.mocked(dns.resolve4).mockImplementation((hostname, callback) => {
                callback(null, ["8.8.8.8"]);
            });
            
            // Mock successful fetch responses
            vi.mocked(fetch.default).mockResolvedValue({
                ok: true,
                status: 200,
            } as any);

            const state = await networkMonitor.forceUpdate();
            
            expect(state.isOnline).toBe(true);
            expect(state.connectivity).toBe("online");
        });
    });

    describe("checkEndpoint", () => {
        it("should return true for successful endpoint", async () => {
            vi.mocked(fetch.default).mockResolvedValue({
                ok: true,
                status: 200,
            } as any);

            const result = await networkMonitor["checkEndpoint"]({
                name: "test",
                url: "https://example.com",
                timeout: 5000,
            });

            expect(result).toBe(true);
        });

        it("should return false for failed endpoint", async () => {
            vi.mocked(fetch.default).mockRejectedValue(new Error("Network error"));

            const result = await networkMonitor["checkEndpoint"]({
                name: "test",
                url: "https://example.com",
                timeout: 5000,
            });

            expect(result).toBe(false);
        });

        it("should return true for server errors below threshold", async () => {
            vi.mocked(fetch.default).mockResolvedValue({
                ok: false,
                status: 404,
            } as any);

            const result = await networkMonitor["checkEndpoint"]({
                name: "test",
                url: "https://example.com",
                timeout: 5000,
            });

            expect(result).toBe(true);
        });

        it("should return false for server errors above threshold", async () => {
            vi.mocked(fetch.default).mockResolvedValue({
                ok: false,
                status: 500,
            } as any);

            const result = await networkMonitor["checkEndpoint"]({
                name: "test",
                url: "https://example.com",
                timeout: 5000,
            });

            expect(result).toBe(false);
        });
    });

    describe("checkDns", () => {
        it("should return true for successful DNS resolution", async () => {
            vi.mocked(dns.resolve4).mockImplementation((hostname, callback) => {
                callback(null, ["8.8.8.8"]);
            });

            const result = await networkMonitor["checkDns"]();
            expect(result).toBe(true);
        });

        it("should return false for failed DNS resolution", async () => {
            vi.mocked(dns.resolve4).mockImplementation((hostname, callback) => {
                callback(new Error("DNS error"), null);
            });

            const result = await networkMonitor["checkDns"]();
            expect(result).toBe(false);
        });
    });

    describe("updateState", () => {
        beforeEach(() => {
            vi.mocked(dns.resolve4).mockImplementation((hostname, callback) => {
                callback(null, ["8.8.8.8"]);
            });
        });

        it("should set offline state when DNS fails", async () => {
            vi.mocked(dns.resolve4).mockImplementation((hostname, callback) => {
                callback(new Error("DNS error"), null);
            });

            await networkMonitor["updateState"]();

            expect(networkMonitor["state"]).toEqual({
                isOnline: false,
                connectivity: "offline",
                lastChecked: expect.any(Date),
                cloudServicesReachable: false,
                localServicesReachable: false,
            });
        });

        it("should set online state when all services are reachable", async () => {
            vi.mocked(fetch.default).mockResolvedValue({
                ok: true,
                status: 200,
            } as any);

            await networkMonitor["updateState"]();

            expect(networkMonitor["state"]).toEqual({
                isOnline: true,
                connectivity: "online",
                lastChecked: expect.any(Date),
                cloudServicesReachable: true,
                localServicesReachable: true,
            });
        });

        it("should set degraded state when only cloud services are reachable", async () => {
            vi.mocked(fetch.default)
                .mockResolvedValueOnce({ ok: true, status: 200 } as any) // OpenRouter
                .mockResolvedValueOnce({ ok: true, status: 200 } as any) // Cloudflare
                .mockRejectedValueOnce(new Error("Connection refused")); // Ollama

            await networkMonitor["updateState"]();

            expect(networkMonitor["state"]).toEqual({
                isOnline: true,
                connectivity: "degraded",
                lastChecked: expect.any(Date),
                cloudServicesReachable: true,
                localServicesReachable: false,
            });
        });

        it("should set degraded state when only local services are reachable", async () => {
            vi.mocked(fetch.default)
                .mockRejectedValueOnce(new Error("Connection refused")) // OpenRouter
                .mockRejectedValueOnce(new Error("Connection refused")) // Cloudflare
                .mockResolvedValueOnce({ ok: true, status: 200 } as any); // Ollama

            await networkMonitor["updateState"]();

            expect(networkMonitor["state"]).toEqual({
                isOnline: true,
                connectivity: "degraded",
                lastChecked: expect.any(Date),
                cloudServicesReachable: false,
                localServicesReachable: true,
            });
        });
    });

    describe("utility methods", () => {
        it("shouldUseLocalModels should return true when cloud is unreachable", () => {
            networkMonitor["state"] = {
                isOnline: true,
                connectivity: "degraded",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: true,
            };

            expect(networkMonitor.shouldUseLocalModels()).toBe(true);
        });

        it("shouldUseLocalModels should return false when cloud is reachable", () => {
            networkMonitor["state"] = {
                isOnline: true,
                connectivity: "online",
                lastChecked: new Date(),
                cloudServicesReachable: true,
                localServicesReachable: true,
            };

            expect(networkMonitor.shouldUseLocalModels()).toBe(false);
        });

        it("isOffline should return true when not online", () => {
            networkMonitor["state"] = {
                isOnline: false,
                connectivity: "offline",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: false,
            };

            expect(networkMonitor.isOffline()).toBe(true);
        });

        it("isOffline should return false when online", () => {
            networkMonitor["state"] = {
                isOnline: true,
                connectivity: "online",
                lastChecked: new Date(),
                cloudServicesReachable: true,
                localServicesReachable: true,
            };

            expect(networkMonitor.isOffline()).toBe(false);
        });
    });
});
