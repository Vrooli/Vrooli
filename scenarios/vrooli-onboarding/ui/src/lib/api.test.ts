import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:9999/api/v1",
  buildApiUrl: (path: string, options: { baseUrl: string }) => `${options.baseUrl}${path}`,
}));

import { fetchOperatorState, fetchV2Readiness, fetchV2Scenarios, saveOperatorState } from "./api";

function mockFetch(body: unknown, ok = true, status = 200) {
  globalThis.fetch = vi.fn().mockResolvedValue({ ok, status, json: () => Promise.resolve(body) });
}

describe("onboarding V2 API client", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("reads the manifest-derived scenario model", async () => {
    mockFetch({ scenarios: [], count: 0 });
    await fetchV2Scenarios();
    expect(globalThis.fetch).toHaveBeenCalledWith("http://localhost:9999/api/v2/scenarios", expect.objectContaining({ cache: "no-store" }));
  });

  it("reads readiness without accepting a credential value", async () => {
    mockFetch({ status: "ready", scenarios: [], resources: [], credentials: [], checked_at: "now" });
    await fetchV2Readiness();
    expect(globalThis.fetch).toHaveBeenCalledWith("http://localhost:9999/api/v2/readiness", expect.objectContaining({ cache: "no-store" }));
  });

  it("persists operator state through its dedicated endpoint", async () => {
    const state = { version: "1.0.0", updated_at: "now", scenarios: { alpha: { enabled: true } } };
    mockFetch(state);
    await saveOperatorState(state);
    expect(globalThis.fetch).toHaveBeenCalledWith("http://localhost:9999/api/v1/operator-state", expect.objectContaining({ method: "PUT", body: JSON.stringify(state) }));
  });

  it("fails closed when the operator-state read fails", async () => {
    mockFetch(null, false, 503);
    await expect(fetchOperatorState()).rejects.toThrow("API request failed: 503");
  });
});
