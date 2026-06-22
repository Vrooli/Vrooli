import { describe, expect, it, vi, beforeEach } from "vitest";

/**
 * Records every Connect RPC the storageClient wrappers make. `createClient` is
 * mocked to return a Proxy whose every property is a recording function, so we
 * can assert the wrappers forward the exact request fields (including the
 * defaults the UI fills in) without a live transport.
 */
const calls: { method: string; arg: unknown }[] = [];

vi.mock("@connectrpc/connect", () => ({
  createClient: () =>
    new Proxy(
      {},
      {
        get: (_t, method: string) => (arg: unknown) => {
          calls.push({ method, arg });
          return Promise.resolve({ ok: true });
        },
      },
    ),
}));

// Avoid constructing a real transport during import.
vi.mock("./client", () => ({ transport: {} }));

beforeEach(() => {
  calls.length = 0;
});

const lastFor = (method: string) =>
  calls.filter((c) => c.method === method).at(-1)?.arg as Record<string, unknown> | undefined;

describe("storageClient wrappers forward typed requests", () => {
  it("scanFleet defaults scenarios to an empty list", async () => {
    const { storageClient } = await import("./storage");
    await storageClient.scanFleet();
    expect(lastFor("scanFleet")).toEqual({ scenarios: [] });
    await storageClient.scanFleet({ scenarios: ["a"] });
    expect(lastFor("scanFleet")).toEqual({ scenarios: ["a"] });
  });

  it("getInventory forwards an empty request", async () => {
    const { storageClient } = await import("./storage");
    await storageClient.getInventory();
    expect(lastFor("getInventory")).toEqual({});
  });

  it("advisor wrappers default scenarios to an empty list", async () => {
    const { storageClient } = await import("./storage");
    await storageClient.adviseEngines();
    expect(lastFor("adviseEngines")).toEqual({ scenarios: [] });
    await storageClient.adviseEngines({ scenarios: ["x"] });
    expect(lastFor("adviseEngines")).toEqual({ scenarios: ["x"] });
    await storageClient.analyzeMigrations();
    expect(lastFor("analyzeMigrations")).toEqual({ scenarios: [] });
    await storageClient.analyzeMigrations({ scenarios: ["y"] });
    expect(lastFor("analyzeMigrations")).toEqual({ scenarios: ["y"] });
  });

  it("validateScenario forwards the scenario name", async () => {
    const { storageClient } = await import("./storage");
    await storageClient.validateScenario({ scenario: "demo" });
    expect(lastFor("validateScenario")).toEqual({ scenario: "demo" });
  });

  it("fix wrappers default ruleIds and forward explicit ids", async () => {
    const { storageClient } = await import("./storage");
    await storageClient.previewFix({ scenario: "demo" });
    expect(lastFor("previewFix")).toEqual({ scenario: "demo", ruleIds: [] });
    await storageClient.applyFix({ scenario: "demo", ruleIds: ["r1"] });
    expect(lastFor("applyFix")).toEqual({ scenario: "demo", ruleIds: ["r1"] });
  });
});
