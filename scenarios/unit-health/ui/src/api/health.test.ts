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

    const health = await fetchHealth();

    expect(health).toMatchObject({ status: "healthy", service: "x", readiness: true });

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/health$/);
    expect(init).toMatchObject({
      method: "GET",
      cache: "no-store",
    });
  });

  it("rejects an unsuccessful response with the server's structured error", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ code: "unavailable", message: "store down" }), { status: 503 }));

    const error = await fetchHealth().catch((error: unknown) => error);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ code: "unavailable", status: 503, message: "unavailable: store down" });
    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it("preserves transport failures instead of returning a healthy response", async () => {
    const failure = new TypeError("connection closed");
    fetchSpy.mockRejectedValueOnce(failure);
    await expect(fetchHealth()).rejects.toBe(failure);
  });

  it("rejects malformed successful JSON instead of manufacturing health", async () => {
    fetchSpy.mockResolvedValueOnce(new Response("not json", { status: 200 }));
    await expect(fetchHealth()).rejects.toBeInstanceOf(SyntaxError);
  });
});
