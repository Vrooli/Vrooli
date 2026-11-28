import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";
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

  it("throws typed API errors on unhealthy responses", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response('{"code":"unhealthy","message":"database unavailable"}', {
        status: 503,
      }),
    );

    try {
      await fetchHealth();
      throw new Error("expected fetchHealth to reject");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect(err).toMatchObject({ code: "unhealthy", status: 503 });
      expect((err as Error).message).toContain("database unavailable");
    }
  });
});
