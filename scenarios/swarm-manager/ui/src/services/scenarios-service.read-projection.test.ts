import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createScenariosService, type IScenariosService } from "./scenarios-service";
import type { IApiClient } from "../lib/api-client";

// Public service read methods only. Complete synthetic owner projections are
// display/read compatibility evidence, not verified quality or admission.
let client: IApiClient;
let service: IScenariosService;
let expectedReads: unknown[][];
beforeEach(() => {
  client = { get: vi.fn().mockRejectedValue(new Error("Unconfigured scenario read")), post: vi.fn().mockRejectedValue(new Error("Forbidden post")), put: vi.fn().mockRejectedValue(new Error("Forbidden put")), patch: vi.fn().mockRejectedValue(new Error("Forbidden patch")), delete: vi.fn().mockRejectedValue(new Error("Forbidden delete")) };
  service = createScenariosService(client); expectedReads = [];
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Forbidden alternate transport")));
});
afterEach(() => {
  try { expect(vi.mocked(client.get).mock.calls).toEqual(expectedReads); for (const method of ["post", "put", "patch", "delete"] as const) expect(client[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); }
  finally { vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});
describe("Scenario owner read projection compatibility", () => {
  it("retains complete literal goal, orphan and active/archived fix coverage from the owner", async () => {
    expectedReads = [["/scenarios/fixture-coverage/context"]];
    vi.mocked(client.get).mockResolvedValue({ scenario_name: "fixture-coverage", goals: [{ name: "fixture-goal", title: "Retained goal", status: "achieved", priority: 3, scope: { total: 7, completed: 2, in_progress: 1, failed: 1, pending: 2, archived: 1 } }], orphan_items: [{ kind: "fix", name: "orphan", title: "Retained orphan", status: "failed", priority: 4, archived_at: "archive-at" }], rollup: { total: 9, completed: 3, in_progress: 1, failed: 2, pending: 2, archived: 1 }, fixes: { active: [{ name: "active", title: "Active fix", status: "ready", priority: 2, goal: "fixture-goal", updated: "updated-at", path: "fix/active" }], archived: [{ name: "past", title: "Archived fix", status: "completed", priority: 1, updated: "past-at", archived_at: "archived-at", path: "fix/past" }] } });
    expect(await service.getContext("fixture-coverage")).toEqual({ scenarioName: "fixture-coverage", goals: [{ name: "fixture-goal", title: "Retained goal", status: "achieved", priority: 3, rollup: { total: 7, completed: 2, inProgress: 1, failed: 1, pending: 2, archived: 1 } }], orphanItems: [{ kind: "fix", name: "orphan", title: "Retained orphan", status: "failed", priority: 4, archivedAt: "archive-at" }], rollup: { total: 9, completed: 3, inProgress: 1, failed: 2, pending: 2, archived: 1 }, fixes: { active: [{ name: "active", title: "Active fix", status: "ready", priority: 2, goal: "fixture-goal", updated: "updated-at", archivedAt: undefined, path: "fix/active" }], archived: [{ name: "past", title: "Archived fix", status: "completed", priority: 1, goal: undefined, updated: "past-at", archivedAt: "archived-at", path: "fix/past" }] } });
  });
  it("preserves explicit camel-case empty names and zero metrics ahead of legacy aliases", async () => {
    expectedReads = [["/scenarios/fixture-camel/context"]];
    vi.mocked(client.get).mockResolvedValue({ scenarioName: "", scenario_name: "legacy-name", goals: [{ name: "", title: "", status: "", priority: 0, scope: { inProgress: 0, in_progress: 7 } }], orphanItems: [{ kind: "", name: "", title: "", status: "", priority: 0, archivedAt: "camel-orphan", archived_at: "legacy-orphan" }], orphan_items: [{ name: "ignored-orphan" }], rollup: { total: 0, completed: 0, inProgress: 0, in_progress: 8, failed: 0, pending: 0, archived: 0 }, fixes: { archived: [{ name: "", title: "", status: "", priority: 0, goal: "", updated: "", archivedAt: "camel-fix", archived_at: "legacy-fix", path: "" }] } });
    expect(await service.getContext("fixture-camel")).toEqual({ scenarioName: "", goals: [{ name: "", title: "", status: "", priority: 0, rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 } }], orphanItems: [{ kind: "", name: "", title: "", status: "", priority: 0, archivedAt: "camel-orphan" }], rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 }, fixes: { active: [], archived: [{ name: "", title: "", status: "", priority: 0, goal: "", updated: "", archivedAt: "camel-fix", path: "" }] } });
  });
  it("renders absent coverage as empty/default data rather than inventing a goal or fix", async () => {
    expectedReads = [["/scenarios/fixture-absent/context"]]; vi.mocked(client.get).mockResolvedValue({});
    expect(await service.getContext("fixture-absent")).toEqual({ scenarioName: "", goals: [], orphanItems: [], rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 }, fixes: { active: [], archived: [] } });
  });
  it("supplies display defaults for minimal retained owner rows without inventing extra rows", async () => {
    expectedReads = [["/scenarios/fixture-minimal/context"]]; vi.mocked(client.get).mockResolvedValue({ goals: [{}], orphan_items: [{}], fixes: { active: [{}] } });
    expect(await service.getContext("fixture-minimal")).toEqual({ scenarioName: "", goals: [{ name: "", title: "", status: "", priority: 0, rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 } }], orphanItems: [{ kind: "", name: "", title: "", status: "", priority: 0, archivedAt: undefined }], rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 }, fixes: { active: [{ name: "", title: "", status: "", priority: 0, goal: undefined, updated: undefined, archivedAt: undefined, path: "" }], archived: [] } });
  });
  it("preserves the exact context read refusal without a lifecycle or remediation fallback", async () => {
    expectedReads = [["/scenarios/fixture-refused/context"]]; const refusal = new Error("Owner coverage read unavailable"); vi.mocked(client.get).mockRejectedValue(refusal);
    await expect(service.getContext("fixture-refused")).rejects.toBe(refusal);
  });
  it("fails on non-array owner coverage instead of reporting a successful empty coverage", async () => {
    expectedReads = [["/scenarios/fixture-malformed/context"]]; vi.mocked(client.get).mockResolvedValue({ goals: { unsupported: true } });
    await expect(service.getContext("fixture-malformed")).rejects.toThrow();
  });
  it("retains nested typed file sizes through the actual owner file contract", async () => {
    expectedReads = [["/scenarios/fixture-files/files"]];
    vi.mocked(client.get).mockResolvedValue({ files: [{ name: "docs", path: "docs", type: "directory", size: "0", children: [{ name: "retained.md", path: "docs/retained.md", type: "file", size: "42" }] }] });
    expect(await service.getFiles("fixture-files")).toEqual([{ name: "docs", path: "docs", type: "directory", size: 0, children: [{ name: "retained.md", path: "docs/retained.md", type: "file", size: 42 }] }]);
  });
  it("rejects an unsupported typed file kind rather than forcing an unreachable display fallback", async () => {
    expectedReads = [["/scenarios/fixture-kind/files"]]; vi.mocked(client.get).mockResolvedValue({ files: [{ name: "unfamiliar", path: "unfamiliar", type: "future-file-kind" }] });
    await expect(service.getFiles("fixture-kind")).rejects.toThrow();
  });
  it("refuses a detail response with no typed scenario without an alternate endpoint", async () => {
    expectedReads = [["/scenarios/fixture-missing"]]; vi.mocked(client.get).mockResolvedValue({});
    await expect(service.get("fixture-missing")).rejects.toThrow();
  });
});
