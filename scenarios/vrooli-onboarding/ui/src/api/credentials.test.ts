import { describe, expect, it, vi } from "vitest";

const calls = vi.hoisted(() => ({ provision: vi.fn() }));
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:9999/api/v1",
  createScenarioConnectTransport: () => ({})
}));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    listCredentials: vi.fn(),
    diagnoseCredentials: vi.fn(),
    provisionCredential: calls.provision,
  }),
}));

import { provisionCredential } from "./credentials";

describe("typed credential client", () => {
  it("sends the value only in the provision request and returns metadata", async () => {
    calls.provision.mockResolvedValueOnce({ status: "provisioned", logicalId: "demo", field: "key" });
    await expect(provisionCredential({ logical_id: "demo", field: "key", value: "secret" })).resolves.toEqual({ status: "provisioned", logicalId: "demo", field: "key" });
    expect(calls.provision).toHaveBeenCalledWith({ target: "local", logicalId: "demo", field: "key", value: "secret" });
    expect(JSON.stringify(calls.provision.mock.results[0])).not.toContain("secret");
  });
});
