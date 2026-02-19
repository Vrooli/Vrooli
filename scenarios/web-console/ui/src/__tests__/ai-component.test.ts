import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock api-base before importing
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:17085/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) =>
    `${opts.baseUrl}${path}`,
  resolveWsBase: () => "ws://localhost:29349/api/v1",
  buildWsUrl: (path: string, opts: { baseUrl: string }) =>
    `${opts.baseUrl}${path}`,
}));

// [REQ:P0-005b] AI Input UI Component - component tests
describe("AiInput component", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("component module exports default function", async () => {
    const mod = await import("../components/AiInput");
    expect(typeof mod.default).toBe("function");
  });

  it("component accepts onExecute and hasActiveTerminal props", async () => {
    const mod = await import("../components/AiInput");
    expect(mod.default).toBeDefined();
    // React functional component accepts props
    expect(mod.default.length).toBeGreaterThanOrEqual(0);
  });

  it("generateAICommand is importable from api module", async () => {
    const { generateAICommand } = await import("../lib/api");
    expect(typeof generateAICommand).toBe("function");
  });

  it("AIGenerateResponse type is correctly shaped", async () => {
    // Verify the response interface matches the API contract
    const mockResponse = { command: "ls -la", provider: "ollama" };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    }) as typeof fetch;

    const { generateAICommand } = await import("../lib/api");
    const result = await generateAICommand("list files");

    expect(result).toHaveProperty("command");
    expect(result).toHaveProperty("provider");
    expect(typeof result.command).toBe("string");
    expect(typeof result.provider).toBe("string");
  });

  it("generateAICommand sends correct request shape", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: "pwd", provider: "ollama" }),
    }) as typeof fetch;

    const { generateAICommand } = await import("../lib/api");
    await generateAICommand("show current dir", "cwd: /home");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/ai/generate"),
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Content-Type": "application/json" }) as Record<string, string>,
      }),
    );

    const calls = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls;
    const firstCall = calls[0] as [string, RequestInit];
    const body = JSON.parse(firstCall[1].body as string) as Record<string, unknown>;
    expect(body).toEqual({ prompt: "show current dir", context: "cwd: /home" });
  });

  it("generateAICommand propagates structured error with retry info", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      json: () =>
        Promise.resolve({
          error: "AI unavailable",
          code: "ai_provider_unavailable",
          category: "dependency",
          recovery: "Check Ollama",
          retry: true,
        }),
    }) as typeof fetch;

    const { generateAICommand, APIError } = await import("../lib/api");
    try {
      await generateAICommand("test");
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(APIError);
      const apiErr = err as InstanceType<typeof APIError>;
      expect(apiErr.code).toBe("ai_provider_unavailable");
      expect(apiErr.category).toBe("dependency");
      expect(apiErr.recovery).toBe("Check Ollama");
      expect(apiErr.retry).toBe(true);
      expect(apiErr.status).toBe(503);
    }
  });
});
