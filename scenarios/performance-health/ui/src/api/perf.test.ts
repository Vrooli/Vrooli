import { describe, expect, it, vi, beforeEach } from "vitest";

/**
 * Records every Connect RPC the perfClient wrappers make. `createClient` is
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

describe("perfClient wrappers forward typed requests", () => {
  it("scanFleet defaults scenarios to an empty list", async () => {
    const { perfClient } = await import("./perf");
    await perfClient.scanFleet();
    expect(lastFor("scanFleet")).toEqual({ scenarios: [] });
    await perfClient.scanFleet({ scenarios: ["a"] });
    expect(lastFor("scanFleet")).toEqual({ scenarios: ["a"] });
  });

  it("readiness wrappers default ruleIds", async () => {
    const { perfClient } = await import("./perf");
    await perfClient.validateReadiness({ scenario: "x" });
    expect(lastFor("validateReadiness")).toEqual({ scenario: "x" });
    await perfClient.previewReadinessFix({ scenario: "x" });
    expect(lastFor("previewReadinessFix")).toEqual({ scenario: "x", ruleIds: [] });
    await perfClient.applyReadinessFix({ scenario: "x", ruleIds: ["r1"] });
    expect(lastFor("applyReadinessFix")).toEqual({ scenario: "x", ruleIds: ["r1"] });
  });

  it("runAudit defaults the workflow to empty", async () => {
    const { perfClient } = await import("./perf");
    await perfClient.runAudit({ scenario: "x" });
    expect(lastFor("runAudit")).toEqual({ scenario: "x", workflow: "" });
  });

  it("trend wrappers default the limit to 30", async () => {
    const { perfClient } = await import("./perf");
    await perfClient.getTrend({ scenario: "x" });
    expect(lastFor("getTrend")).toEqual({ scenario: "x", limit: 30 });
    await perfClient.getStartupTrend({ scenario: "x", limit: 5 });
    expect(lastFor("getStartupTrend")).toEqual({ scenario: "x", limit: 5 });
  });

  it("budget wrappers forward scenario + budget payloads", async () => {
    const { perfClient } = await import("./perf");
    await perfClient.getBudget({ scenario: "x" });
    expect(lastFor("getBudget")).toEqual({ scenario: "x" });
    const budget = { scenario: "x" } as never;
    await perfClient.setBudget({ budget });
    expect(lastFor("setBudget")).toEqual({ budget });
    await perfClient.checkBudget({ scenario: "x" });
    expect(lastFor("checkBudget")).toEqual({ scenario: "x" });
  });

  it("analysis wrappers forward artifact handles", async () => {
    const { perfClient } = await import("./perf");
    await perfClient.analyzeTrace({ scenario: "x", traceArtifact: "/a.json" });
    expect(lastFor("analyzeTrace")).toEqual({ scenario: "x", traceArtifact: "/a.json" });
    await perfClient.compareTraces({
      scenario: "x",
      baselineArtifact: "/b.json",
      candidateArtifact: "/c.json",
    });
    expect(lastFor("compareTraces")).toEqual({
      scenario: "x",
      baselineArtifact: "/b.json",
      candidateArtifact: "/c.json",
    });
  });
});
