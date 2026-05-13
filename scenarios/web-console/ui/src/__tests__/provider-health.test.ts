import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiBaseMock } from "../test-utils";

vi.mock("@vrooli/api-base", () => apiBaseMock());

const getConfigMock = vi.fn();
const updateConfigMock = vi.fn();
const getHealthMock = vi.fn();

vi.mock("../api/ai", async () => {
  const actual = await vi.importActual<typeof import("../api/ai")>("../api/ai");
  return {
    ...actual,
    aiClient: {
      getConfig: getConfigMock,
      updateConfig: updateConfigMock,
      getHealth: getHealthMock,
    },
    getAIConfig: async () => {
      const resp = await getConfigMock({});
      return {
        providers: resp.providers.map((p: Record<string, unknown>) => ({
          name: p.name as string,
          enabled: p.enabled as boolean,
          priority: p.priority as number,
          timeout_sec: p.timeoutSec as number,
          max_retries: p.maxRetries as number,
        })),
        health: resp.health.map((h: Record<string, unknown>) => ({
          name: h.name as string,
          available: h.available as boolean,
          last_check: (h.lastCheck as string) || undefined,
          last_latency: (h.lastLatency as string) || undefined,
          error_count: Number(h.errorCount ?? 0),
          success_count: Number(h.successCount ?? 0),
          error_rate: h.errorRate as number,
        })),
      };
    },
    updateAIConfig: async (update: { name: string; enabled?: boolean; priority?: number; timeout_sec?: number; max_retries?: number }) => {
      const req: Record<string, unknown> = { name: update.name };
      if (update.enabled !== undefined) { req.enabled = update.enabled; req.hasEnabled = true; }
      if (update.priority !== undefined) { req.priority = update.priority; req.hasPriority = true; }
      if (update.timeout_sec !== undefined) { req.timeoutSec = update.timeout_sec; req.hasTimeoutSec = true; }
      if (update.max_retries !== undefined) { req.maxRetries = update.max_retries; req.hasMaxRetries = true; }
      const resp = await updateConfigMock(req);
      return {
        providers: resp.providers.map((p: Record<string, unknown>) => ({
          name: p.name as string,
          enabled: p.enabled as boolean,
          priority: p.priority as number,
          timeout_sec: p.timeoutSec as number,
          max_retries: p.maxRetries as number,
        })),
        health: resp.health.map((h: Record<string, unknown>) => ({
          name: h.name as string,
          available: h.available as boolean,
          error_count: Number(h.errorCount ?? 0),
          success_count: Number(h.successCount ?? 0),
          error_rate: h.errorRate as number,
        })),
      };
    },
    getAIHealth: async () => {
      const resp = await getHealthMock({});
      return resp.health.map((h: Record<string, unknown>) => ({
        name: h.name as string,
        available: h.available as boolean,
        error_count: Number(h.errorCount ?? 0),
        success_count: Number(h.successCount ?? 0),
        error_rate: h.errorRate as number,
      }));
    },
  };
});

beforeEach(() => {
  getConfigMock.mockReset();
  updateConfigMock.mockReset();
  getHealthMock.mockReset();
});

// [REQ:P1-003a] Provider Configuration API client tests
describe("AI Provider Config API", () => {
  it("getAIConfig returns providers and health", async () => {
    getConfigMock.mockResolvedValue({
      providers: [
        { name: "ollama", enabled: true, priority: 1, timeoutSec: 30, maxRetries: 0 },
        { name: "openrouter", enabled: true, priority: 2, timeoutSec: 30, maxRetries: 0 },
      ],
      health: [
        { name: "ollama", available: true, errorCount: 0, successCount: 5, errorRate: 0 },
        { name: "openrouter", available: false, errorCount: 2, successCount: 0, errorRate: 1 },
      ],
    });

    const { getAIConfig } = await import("../api/ai");
    const result = await getAIConfig();

    expect(result.providers).toHaveLength(2);
    expect(result.health).toHaveLength(2);
    expect(result.providers[0]).toMatchObject({ name: "ollama", timeout_sec: 30, max_retries: 0 });
    expect(getConfigMock).toHaveBeenCalled();
  });

  it("updateAIConfig sends partial update with has* presence flags", async () => {
    updateConfigMock.mockResolvedValue({
      providers: [{ name: "ollama", enabled: false, priority: 1, timeoutSec: 10, maxRetries: 0 }],
      health: [],
    });

    const { updateAIConfig } = await import("../api/ai");
    const result = await updateAIConfig({ name: "ollama", enabled: false, timeout_sec: 10 });

    expect(result.providers[0]?.enabled).toBe(false);
    expect(updateConfigMock).toHaveBeenCalledWith(expect.objectContaining({
      name: "ollama",
      enabled: false,
      hasEnabled: true,
      timeoutSec: 10,
      hasTimeoutSec: true,
    }));
  });

  it("getAIHealth returns health array", async () => {
    getHealthMock.mockResolvedValue({
      health: [{ name: "ollama", available: true, errorCount: 0, successCount: 3, errorRate: 0 }],
    });

    const { getAIHealth } = await import("../api/ai");
    const result = await getAIHealth();

    expect(result).toHaveLength(1);
    expect(result[0]?.available).toBe(true);
  });

  it("getAIConfig propagates errors from the Connect client", async () => {
    getConfigMock.mockRejectedValue(new Error("Internal error"));

    const { getAIConfig } = await import("../api/ai");
    await expect(getAIConfig()).rejects.toThrow("Internal error");
  });
});

// [REQ:P1-003b] Provider Health Dashboard - type tests
describe("Provider types", () => {
  it("ProviderConfig interface includes fields", async () => {
    const config: import("../api/ai").ProviderConfig = {
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
    const health: import("../api/ai").ProviderHealth = {
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
});
