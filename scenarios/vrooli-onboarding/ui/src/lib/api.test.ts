import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:9999/api/v1",
  buildApiUrl: (path: string, options: { baseUrl: string }) => `${options.baseUrl}${path}`,
}));

import { applyOnboarding, applyCapability, fetchCapabilities, fetchHealth, fetchGlossary, fetchOperatorInputs, fetchOperatorState, fetchResourceHealth, fetchResources, fetchV2ApplyPlan, fetchV2ApplyStatus, fetchV2Closure, fetchV2HostRequirements, fetchV2Recommendation, fetchV2Resources, fetchV2Readiness, fetchV2Scenarios, fetchV2Session, previewCapability, provisionCredential, resolveOperatorInputs, saveOperatorState, saveV2SessionStep } from "./api";

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

	it("reads and applies provider-defined capabilities through the generic endpoints", async () => {
		mockFetch({ capabilities: [], count: 0 });
		await fetchCapabilities();
		expect(globalThis.fetch).toHaveBeenCalledWith("http://localhost:9999/api/v2/capabilities", expect.objectContaining({ cache: "no-store" }));
		mockFetch({ capability_id: "demo", plan_id: "plan", state: "ready_to_preview" });
		await previewCapability({ capability_id: "demo", confirm: false, inputs: { destination: "/mnt/demo" } });
		mockFetch({ capability_id: "demo", state: "ready", outcome: "complete", retryable: true });
		await applyCapability({ capability_id: "demo", confirm: true, inputs: { destination: "/mnt/demo" } });
		expect(globalThis.fetch).toHaveBeenLastCalledWith("http://localhost:9999/api/v2/capabilities/apply", expect.objectContaining({ method: "POST", body: JSON.stringify({ capability_id: "demo", confirm: true, inputs: { destination: "/mnt/demo" } }) }));
	});

  it("covers the read and mutation surfaces without exposing secrets", async () => {
    mockFetch({ status: "ok", service: "onboarding", timestamp: "now" });
    await fetchHealth();
    mockFetch({ scenarios: [], resources: [] });
    await fetchV2Recommendation();
    mockFetch({ run_id: "apply-1", status: "applied", items: [] });
    await fetchV2ApplyPlan();
    await fetchV2ApplyStatus("run/1");
    mockFetch({ version: 1, current_step: 1 });
    await fetchV2Session();
    await saveV2SessionStep(2);
    mockFetch({ version: 1, updated_at: "now", requests: [] });
    await fetchOperatorInputs();
    mockFetch({ status: "resolved", configuration_pending: false });
    await resolveOperatorInputs([{ request_id: "choice", value: "yes" }]);
    mockFetch({ resources: [] });
    await fetchResources();
    mockFetch([]);
    await fetchResources();
    mockFetch({ resources: [] });
    await fetchResourceHealth();
    mockFetch({ entries: [] });
    await fetchGlossary("credential store");
    expect(globalThis.fetch).toHaveBeenLastCalledWith("http://localhost:9999/api/v1/glossary?q=credential%20store", expect.objectContaining({ cache: "no-store" }));
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
