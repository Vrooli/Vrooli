import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchHealth } from "./health";

describe("api/health.fetchHealth", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("requests /health with cache: 'no-store'", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response('{"status":"healthy","service":"x","timestamp":"t","readiness":true}', {
        status: 200,
      }),
    );

    await fetchHealth();

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/health$/);
    expect(init).toMatchObject({
      method: "GET",
      cache: "no-store",
    });
  });

  it("rejects a non-success response with its typed API error", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "unavailable", message: "database unavailable" }), {
        status: 503,
      }),
    );

    await expect(fetchHealth()).rejects.toMatchObject({
      name: "ApiError",
      code: "unavailable",
      status: 503,
    });
  });
});
