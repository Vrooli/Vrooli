/**
 * Additional coverage for api/health.ts — lines 14-15 (error branch).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchHealth } from "./health";
import { ApiError } from "./client";

describe("api/health.fetchHealth — error path (lines 14-15)", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("throws an ApiError when the response is not ok (500)", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "internal", message: "server error" }), {
        status: 500,
      }),
    );
    await expect(fetchHealth()).rejects.toBeInstanceOf(ApiError);
  });

  it("throws an ApiError with code 'not_found' for a 404", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "not_found", message: "endpoint gone" }), {
        status: 404,
      }),
    );
    const err = await fetchHealth().catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(404);
    expect((err as ApiError).code).toBe("not_found");
  });

  it("throws a generic ApiError when the 500 body is unparseable JSON", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response("not json at all", { status: 503 }),
    );
    const err = await fetchHealth().catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(503);
    // Falls back to "internal" code when envelope is missing
    expect((err as ApiError).code).toBe("internal");
  });
});
