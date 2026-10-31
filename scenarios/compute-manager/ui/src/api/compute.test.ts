import { beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  instances: { listInstances: vi.fn(), getInstance: vi.fn(), requestInstance: vi.fn() },
  reconcile: { listFindings: vi.fn() },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn()
    .mockReturnValueOnce(clients.instances)
    .mockReturnValueOnce(clients.reconcile),
}));
vi.mock("./client", () => ({ transport: {} }));

import { fetchInstance, fetchInstances, fetchOpenFindings, requestInstance } from "./compute";

describe("compute API wrappers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("forwards instance and finding queries", () => {
    fetchInstances();
    fetchOpenFindings();
    fetchInstance("i-1");

    expect(clients.instances.listInstances).toHaveBeenCalledWith({});
    expect(clients.reconcile.listFindings).toHaveBeenCalledWith({ status: "open" });
    expect(clients.instances.getInstance).toHaveBeenCalledWith({ id: "i-1" });
  });

  it("forwards a capacity request unchanged", () => {
    const input = { idempotencyKey: "k", provider: "hetzner", region: "fsn1", size: "cx22", lifetimeSeconds: 60n };
    requestInstance(input);
    expect(clients.instances.requestInstance).toHaveBeenCalledWith(input);
  });

  it("returns generated-client inventory responses unchanged", async () => { // [REQ:COMPUTEM-P1-005]
    const instances = { instances: [{ id: "instance-1" }] };
    const findings = { findings: [{ id: "finding-1", kind: 1 }] };
    clients.instances.listInstances.mockResolvedValueOnce(instances);
    clients.reconcile.listFindings.mockResolvedValueOnce(findings);

    await expect(fetchInstances()).resolves.toEqual(instances);
    await expect(fetchOpenFindings()).resolves.toEqual(findings);
  });
});
