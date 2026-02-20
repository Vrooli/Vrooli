import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiBaseMock } from "../test-utils";

// Mock api-base before any imports
vi.mock("@vrooli/api-base", () => apiBaseMock());

beforeEach(() => {
  vi.restoreAllMocks();
});

// [REQ:P1-003a] Provider Configuration API client tests
describe("AI Provider Config API", () => {
  it("getAIConfig returns providers and health", async () => {
    const mockResp = {
      providers: [
        { name: "ollama", enabled: true, priority: 1, timeout_sec: 30, max_retries: 0 },
        { name: "openrouter", enabled: true, priority: 2, timeout_sec: 30, max_retries: 0 },
      ],
      health: [
        { name: "ollama", available: true, error_count: 0, success_count: 5, error_rate: 0 },
        { name: "openrouter", available: false, error_count: 2, success_count: 0, error_rate: 1 },
      ],
    };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockResp),
    }) as typeof fetch;

    const { getAIConfig } = await import("../lib/api");
    const result = await getAIConfig();

    expect(result.providers).toHaveLength(2);
    expect(result.health).toHaveLength(2);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/ai/config"),
      expect.objectContaining({ cache: "no-store" }),
    );
  });

  it("updateAIConfig sends PUT with partial update", async () => {
    const mockResp = {
      providers: [
        { name: "ollama", enabled: false, priority: 1, timeout_sec: 10, max_retries: 0 },
      ],
      health: [],
    };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockResp),
    }) as typeof fetch;

    const { updateAIConfig } = await import("../lib/api");
    const result = await updateAIConfig({ name: "ollama", enabled: false, timeout_sec: 10 });

    expect(result.providers[0]?.enabled).toBe(false);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/ai/config"),
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("getAIHealth returns health array", async () => {
    const mockHealth = [
      { name: "ollama", available: true, error_count: 0, success_count: 3, error_rate: 0 },
    ];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockHealth),
    }) as typeof fetch;

    const { getAIHealth } = await import("../lib/api");
    const result = await getAIHealth();

    expect(result).toHaveLength(1);
    expect(result[0]?.available).toBe(true);
  });

  it("getAIConfig throws on error", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: "Internal error" }),
    }) as typeof fetch;

    const { getAIConfig, APIError } = await import("../lib/api");

    try {
      await getAIConfig();
      expect.unreachable("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(APIError);
    }
  });
});

// [REQ:P1-003b] Provider Health Dashboard - type tests
describe("Provider types", () => {
  it("ProviderConfig interface includes fields", async () => {
    const config: import("../lib/api").ProviderConfig = {
      name: "ollama",
      enabled: true,
      priority: 1,
      timeout_sec: 30,
      max_retries: 0,
    };
    expect(config.name).toBe("ollama");
    expect(config.enabled).toBe(true);
  });

  it("ProviderHealth interface includes availability and metrics", async () => {
    const health: import("../lib/api").ProviderHealth = {
      name: "ollama",
      available: true,
      last_check: "2026-01-01T00:00:00Z",
      last_latency: "100ms",
      error_count: 0,
      success_count: 10,
      error_rate: 0,
    };
    expect(health.available).toBe(true);
    expect(health.error_rate).toBe(0);
  });

  it("ProviderHealthPanel module exports default function", async () => {
    const mod = await import("../components/ProviderHealthPanel");
    expect(typeof mod.default).toBe("function");
  });
});
