import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock api-base before importing api module
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:17085/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) =>
    `${opts.baseUrl}${path}`,
  resolveWsBase: () => "ws://localhost:29349/api/v1",
  buildWsUrl: (path: string, opts: { baseUrl: string }) =>
    `${opts.baseUrl}${path}`,
}));

// [REQ:P0-005a] AI Command Generation API - client tests
// [REQ:P0-005b] AI Input UI Component - API integration tests
describe("generateAICommand", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("sends POST with prompt and returns command/provider", async () => {
    const mockResponse = {
      command: "find . -name '*.go'",
      provider: "ollama",
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    }) as typeof fetch;

    const { generateAICommand } = await import("../lib/api");
    const result = await generateAICommand("find all Go files");

    expect(result).toEqual(mockResponse);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/ai/generate"),
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("find all Go files") as string,
      }),
    );
  });

  it("includes context when provided", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: "ls /tmp", provider: "ollama" }),
    }) as typeof fetch;

    const { generateAICommand } = await import("../lib/api");
    await generateAICommand("list temp files", "cwd: /home/user");

    const calls = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls;
    const firstCall = calls[0] as [string, RequestInit];
    const callBody = JSON.parse(firstCall[1].body as string) as Record<string, unknown>;
    expect(callBody.context).toBe("cwd: /home/user");
  });

  it("throws APIError on failure", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      json: () =>
        Promise.resolve({
          error: "AI unavailable",
          code: "ai_provider_unavailable",
          category: "dependency",
          retry: true,
        }),
    }) as typeof fetch;

    const { generateAICommand, APIError } = await import("../lib/api");
    try {
      await generateAICommand("test prompt");
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(APIError);
      const apiErr = err as InstanceType<typeof APIError>;
      expect(apiErr.code).toBe("ai_provider_unavailable");
      expect(apiErr.retry).toBe(true);
    }
  });

  it("omits context field when not provided", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: "pwd", provider: "openrouter" }),
    }) as typeof fetch;

    const { generateAICommand } = await import("../lib/api");
    await generateAICommand("current directory");

    const calls = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls;
    const firstCall = calls[0] as [string, RequestInit];
    const callBody = JSON.parse(firstCall[1].body as string) as Record<string, unknown>;
    expect(callBody.prompt).toBe("current directory");
  });
});
