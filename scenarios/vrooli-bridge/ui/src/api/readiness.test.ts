import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { authedFetch, decodeApiError } = vi.hoisted(() => ({ authedFetch: vi.fn(), decodeApiError: vi.fn() }));

vi.mock("./client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./client")>();
  return { ...actual, authedFetch, decodeApiError };
});

import { fetchBridgeReadiness, performBridgeFirewallAction } from "./readiness";

describe("bridge readiness API", () => {
  beforeEach(() => { decodeApiError.mockResolvedValue(new Error("readiness failed")); });
  afterEach(() => { vi.clearAllMocks(); });

  it("reads the non-cacheable readiness projection", async () => {
    authedFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve({ status: "ready" }) });
    await expect(fetchBridgeReadiness()).resolves.toMatchObject({ status: "ready" });
    expect(authedFetch).toHaveBeenCalledWith(expect.stringMatching(/\/readiness$/), { cache: "no-store" });
  });

  it("posts an explicit firewall action and maps failed responses", async () => {
    authedFetch.mockResolvedValueOnce({ ok: false }).mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ status: "allowed", changed: true }) });
    await expect(fetchBridgeReadiness()).rejects.toThrow("readiness failed");
    await expect(performBridgeFirewallAction("allow", "192.0.2.5", true)).resolves.toMatchObject({ status: "allowed", changed: true });
    expect(authedFetch).toHaveBeenLastCalledWith(expect.stringMatching(/\/readiness\/firewall$/), expect.objectContaining({ method: "POST", body: JSON.stringify({ action: "allow", candidate_ip: "192.0.2.5", confirm: true }) }));
  });

  it("rejects a failed firewall action response", async () => {
    authedFetch.mockResolvedValue({ ok: false });
    await expect(performBridgeFirewallAction("inspect", "192.0.2.10")).rejects.toThrow("readiness failed");
  });
});
