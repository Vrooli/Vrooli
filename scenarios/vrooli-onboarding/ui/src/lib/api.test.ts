import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:9999/api/v1",
  buildApiUrl: (path: string, options: { baseUrl: string }) => `${options.baseUrl}${path}`,
}));

import { applyOnboarding, fetchOperatorState, fetchV2Closure, fetchV2HostRequirements, fetchV2Resources, fetchV2Readiness, fetchV2Scenarios, provisionCredential, saveOperatorState } from "./api";

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

	it("reads the shared closure and grouped resources", async () => {
    mockFetch({ scenarios: [], resources: [] });
    await fetchV2Closure();
    await fetchV2Resources();
    expect(globalThis.fetch).toHaveBeenNthCalledWith(1, "http://localhost:9999/api/v2/closure", expect.objectContaining({ cache: "no-store" }));
    expect(globalThis.fetch).toHaveBeenNthCalledWith(2, "http://localhost:9999/api/v2/resources", expect.objectContaining({ cache: "no-store" }));
	});

	it("reads host requirements and sends credentials only to the authority", async () => {
		mockFetch({ tools: [], safeguards: [] });
		await fetchV2HostRequirements();
		expect(globalThis.fetch).toHaveBeenCalledWith("http://localhost:9999/api/v2/host-requirements", expect.objectContaining({ cache: "no-store" }));
		mockFetch({ status: "provisioned", logical_id: "demo", field: "key" });
		await provisionCredential({ logical_id: "demo", field: "key", value: "secret" });
		expect(globalThis.fetch).toHaveBeenLastCalledWith("http://localhost:9999/api/v2/credentials/provision", expect.objectContaining({ method: "POST", body: JSON.stringify({ logical_id: "demo", field: "key", value: "secret" }) }));
	});

	it("applies the committed selection through the V2 endpoint", async () => {
		mockFetch({ run_id: "apply-1", status: "applied", items: [] });
		await applyOnboarding();
		expect(globalThis.fetch).toHaveBeenCalledWith("http://localhost:9999/api/v2/apply", expect.objectContaining({ method: "POST", body: "{}" }));
	});

  it("persists operator state through its dedicated endpoint", async () => {
    const patch = { scenarios: { alpha: { enabled: true } } };
    mockFetch({ version: "1.0.0", updated_at: "now", ...patch });
    await saveOperatorState(patch);
    expect(globalThis.fetch).toHaveBeenCalledWith("http://localhost:9999/api/v2/operator-state", expect.objectContaining({ method: "PATCH", headers: { "Content-Type": "application/merge-patch+json" }, body: JSON.stringify(patch) }));
  });

  it("fails closed when the operator-state read fails", async () => {
    mockFetch(null, false, 503);
    await expect(fetchOperatorState()).rejects.toThrow("API request failed: 503");
  });
});
