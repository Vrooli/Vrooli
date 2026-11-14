import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchHealth } from "./health";
import { ApiError } from "./client";

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

  it("throws a typed ApiError on a non-2xx response", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "unavailable", message: "down" }), { status: 503 }),
    );

    await expect(fetchHealth()).rejects.toBeInstanceOf(ApiError);
  });

  it("parses the body into a typed HealthResponse on success", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response('{"status":"healthy","service":"image-tools","timestamp":"t","readiness":true}', {
        status: 200,
      }),
    );

    const out = await fetchHealth();

    expect(out.status).toBe("healthy");
    expect(out.service).toBe("image-tools");
    expect(out.readiness).toBe(true);
  });
});
