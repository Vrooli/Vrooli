/**
 * Self-tests for makeApiMocks.
 *
 * Builders that ship without their own tests are how mocks drift
 * silently — see the package header in `mocks/spatial.test.ts` for the
 * full rationale. The contract pinned here:
 *
 *   - each call returns a fresh surface (no shared mutable state)
 *   - default `fetchHealth` resolves to a sane HealthResponse
 *   - the returned function is a vi.fn so per-test overrides via
 *     mockResolvedValueOnce / mockRejectedValueOnce work
 */
import { describe, expect, it, vi } from "vitest";

import { makeApiMocks } from "./api";

describe("makeApiMocks", () => {
  it("returns a fresh surface on every call (no shared state)", () => {
    const a = makeApiMocks();
    const b = makeApiMocks();
    expect(a).not.toBe(b);
    expect(a.fetchHealth).not.toBe(b.fetchHealth);
  });

  it("default fetchHealth resolves to a healthy HealthResponse", async () => {
    const { fetchHealth } = makeApiMocks();
    const r = await fetchHealth();
    expect(r.status).toBe("healthy");
    expect(r.readiness).toBe(true);
  });

  it("fetchHealth is a vi.fn so per-test overrides work", async () => {
    const { fetchHealth } = makeApiMocks();
    expect(vi.isMockFunction(fetchHealth)).toBe(true);

    fetchHealth.mockRejectedValueOnce(new Error("simulated 5xx"));
    await expect(fetchHealth()).rejects.toThrow("simulated 5xx");
    // Subsequent calls fall back to the default resolved value.
    await expect(fetchHealth()).resolves.toMatchObject({ status: "healthy" });
  });
});
