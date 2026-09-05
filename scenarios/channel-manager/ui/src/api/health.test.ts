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

	it("surfaces the API envelope when health is unavailable", async () => {
		fetchSpy.mockResolvedValueOnce(new Response('{"message":"starting"}', { status: 503 }));
		await expect(fetchHealth()).rejects.toThrow("starting");
	});
});
