import { fetchHealth } from "./api";

// [REQ:GCT-OT-P0-001] Health check endpoint

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:18700/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) => `${opts.baseUrl}${path}`
}));

test("fetchHealth returns parsed JSON on success", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = vi.fn(async () => {
    return new Response(JSON.stringify({ status: "healthy", service: "x", timestamp: "t" }), {
      status: 200,
      headers: { "Content-Type": "application/json" }
    });
  }) as unknown as typeof fetch;

  try {
    const result = await fetchHealth();
    expect(result.status).toBe("healthy");
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("fetchHealth throws when API returns non-200", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = vi.fn(async () => {
    return new Response("Service unavailable", { status: 503 });
  }) as unknown as typeof fetch;

  try {
    await expect(fetchHealth()).rejects.toThrow(/Service unavailable|Request failed: 503/);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

