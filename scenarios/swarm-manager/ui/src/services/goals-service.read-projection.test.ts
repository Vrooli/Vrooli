import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createGoalsService, type IGoalsService } from "./goals-service";
import type { IApiClient } from "../lib/api-client";

// Actual read adapter with an injected upstream boundary. Owner-returned read
// projections are synthetic and do not establish verified delivery or admission.
const richWire = {
  goal: {
    name: "fixture-projection", title: "Retained goal", description: "Owner description", status: "archived", priority: 4,
    targets: ["execute/target"], seeded: true, created: "2026-10-01T00:00:00Z", updated: "2026-10-02T00:00:00Z", archived_at: "2026-10-03T00:00:00Z",
    verified_delivered_at: "2026-10-02T12:00:00Z",
    milestones: [{ name: "proof", title: "Proof", description: "Retained boundary", items: ["execute/target"], acceptance_criteria: ["Human review retained"], depends_on: ["foundation"], archived_at: "2026-10-03T00:00:00Z" }],
    scope_history: [{ at: "snapshot-at", target_count: 1, closure_size: 3, completed: 2 }],
  },
  scope: { targets: ["execute/target"], closure: ["execute/target", "fix/dependency", "idea/blocked"], completed: ["fix/dependency"], ready: ["execute/target"], blocked: ["idea/blocked"], total: 3, completed_count: 1, blocked_count: 1, progress_pct: 33 },
  eta: { p50_hours: 4, p80_hours: 8, p50_label: "4 h", p80_label: "8 h", basis: "retained", basis_label: "Owner samples", confidence: "bounded", remaining_items: 2, lane_capacity: 1 },
  scope_entities: { items: { "execute/target": { name: "target", title: "Exact owner target", description: "Retained task", status: "ready", priority: 2, tags: ["owner"], created: "created-at", updated: "updated-at", kind: "execute", depends_on: ["fix/dependency"], milestone: "fixture-projection/proof", effort: "fixture-effort", note: "Owner note", archived_at: "archive-at" } } },
};
const richCanonical = {
  goal: { name: "fixture-projection", title: "Retained goal", description: "Owner description", status: "archived", priority: 4, targets: ["execute/target"], seeded: true, created: "2026-10-01T00:00:00Z", updated: "2026-10-02T00:00:00Z", archivedAt: "2026-10-03T00:00:00Z", verifiedDeliveredAt: "2026-10-02T12:00:00Z", milestones: [{ name: "proof", title: "Proof", description: "Retained boundary", items: ["execute/target"], acceptanceCriteria: ["Human review retained"], dependsOn: ["foundation"], archivedAt: "2026-10-03T00:00:00Z" }], scopeHistory: [{ at: "snapshot-at", targetCount: 1, closureSize: 3, completed: 2 }] },
  scope: { targets: ["execute/target"], closure: ["execute/target", "fix/dependency", "idea/blocked"], completed: ["fix/dependency"], ready: ["execute/target"], blocked: ["idea/blocked"], total: 3, completedCount: 1, blockedCount: 1, progressPct: 33 },
  eta: { p50Hours: 4, p80Hours: 8, p50Label: "4 h", p80Label: "8 h", basis: "retained", basisLabel: "Owner samples", confidence: "bounded", remainingItems: 2, laneCapacity: 1 },
  scopeEntities: { items: { "execute/target": { name: "target", title: "Exact owner target", description: "Retained task", status: "ready", priority: 2, tags: ["owner"], created: "created-at", updated: "updated-at", kind: "execute", dependsOn: ["fix/dependency"], milestone: "fixture-projection/proof", effort: "fixture-effort", note: "Owner note", archivedAt: "archive-at" } } },
};
let client: IApiClient;
let service: IGoalsService;
let expectedReads: unknown[][];
beforeEach(() => {
  client = { get: vi.fn().mockRejectedValue(new Error("Unconfigured owner read")), post: vi.fn().mockRejectedValue(new Error("Forbidden post")), put: vi.fn().mockRejectedValue(new Error("Forbidden put")), patch: vi.fn().mockRejectedValue(new Error("Forbidden patch")), delete: vi.fn().mockRejectedValue(new Error("Forbidden delete")) };
  service = createGoalsService(client); expectedReads = [];
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Forbidden alternate transport")));
});
afterEach(() => {
  try {
    expect(vi.mocked(client.get).mock.calls).toEqual(expectedReads);
    for (const method of ["post", "put", "patch", "delete"] as const) expect(client[method]).not.toHaveBeenCalled();
    expect(fetch).not.toHaveBeenCalled();
  } finally { vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});
describe("Goals actual owner read projection compatibility", () => {
  it("retains the full literal snake-case read including history, milestone, hydration and ETA", async () => {
    expectedReads = [["/goals/fixture-projection"]]; vi.mocked(client.get).mockResolvedValue(richWire);
    expect(await service.get("fixture-projection")).toEqual(richCanonical);
  });
  it("honors explicit zero, empty and camel-case owner fields ahead of legacy aliases", async () => {
    expectedReads = [["/goals/fixture-camel"]];
    vi.mocked(client.get).mockResolvedValue({
      goal: { name: "fixture-camel", title: "", description: "", status: "achieved", priority: 0, seeded: false, targets: [], archivedAt: "camel-archive", archived_at: "legacy-archive", verifiedDeliveredAt: "camel-delivery", verified_delivered_at: "legacy-delivery", scopeHistory: [{ at: "", targetCount: 0, target_count: 7, closureSize: 0, closure_size: 9, completed: 0 }], scope_history: [{ at: "old", completed: 8 }], milestones: [{ name: "zero", title: "", description: "", items: [], acceptanceCriteria: [], acceptance_criteria: ["legacy"], dependsOn: [], depends_on: ["legacy"], archivedAt: "camel-milestone", archived_at: "legacy-milestone" }] },
      scope: { completedCount: 0, completed_count: 7, blockedCount: 0, blocked_count: 8, progressPct: 0, progress_pct: 99 },
      eta: { p50Hours: 0, p50_hours: 9, p80Hours: 0, p80_hours: 12, p50Label: "", p50_label: "legacy", p80Label: "", p80_label: "legacy", basis: "", basisLabel: "", basis_label: "legacy", confidence: "", remainingItems: 0, remaining_items: 7, laneCapacity: 0, lane_capacity: 9 },
      scopeEntities: { items: { "fix/selected": { name: "selected", title: "", description: "", status: "backlog", priority: 0, kind: "fix", tags: [], created: "", updated: "", dependsOn: [], depends_on: ["legacy"], archivedAt: "camel-item", archived_at: "legacy-item" } } },
      scope_entities: { items: { "execute/ignored": { name: "ignored" } } },
    });
    expect(await service.get("fixture-camel")).toEqual({
      goal: { name: "fixture-camel", title: "", description: "", status: "achieved", priority: 0, seeded: false, targets: [], created: "", updated: "", archivedAt: "camel-archive", verifiedDeliveredAt: "camel-delivery", scopeHistory: [{ at: "", targetCount: 0, closureSize: 0, completed: 0 }], milestones: [{ name: "zero", title: "", description: "", items: [], acceptanceCriteria: [], dependsOn: [], archivedAt: "camel-milestone" }] },
      scope: { targets: [], closure: [], completed: [], ready: [], blocked: [], total: 0, completedCount: 0, blockedCount: 0, progressPct: 0 },
      eta: { p50Hours: 0, p80Hours: 0, p50Label: "", p80Label: "", basis: "", basisLabel: "", confidence: "", remainingItems: 0, laneCapacity: 0 },
      scopeEntities: { items: { "fix/selected": { name: "selected", title: "", description: "", status: "backlog", priority: 0, kind: "fix", tags: [], created: "", updated: "", dependsOn: [], archivedAt: "camel-item" } } },
    });
  });
  it("keeps omitted optional hydration and ETA absent with stable display defaults", async () => {
    expectedReads = [["/goals/fixture-minimal"]]; vi.mocked(client.get).mockResolvedValue({ goal: { name: "fixture-minimal" }, scope_entities: { items: {} }, eta: null });
    expect(await service.get("fixture-minimal")).toEqual({ goal: { name: "fixture-minimal", title: "fixture-minimal", description: "", status: "active", priority: 0, targets: [], milestones: [], seeded: false, scopeHistory: [], created: "", updated: "" }, scope: { targets: [], closure: [], completed: [], ready: [], blocked: [], total: 0, completedCount: 0, blockedCount: 0, progressPct: 0 }, eta: null });
  });
  it("fills minimal hydrated rows and milestone history without inventing extra owner refs", async () => {
    expectedReads = [["/goals/fixture-minimal-rows"]];
    vi.mocked(client.get).mockResolvedValue({ goal: { name: "fixture-minimal-rows", milestones: [{ name: "first" }], scope_history: [{}] }, eta: {}, scope_entities: { items: { "execute/only-owner-ref": { name: "only-owner-ref" } } } });
    const result = await service.get("fixture-minimal-rows");
    expect(result.goal.milestones).toEqual([{ name: "first", title: "first", description: "", items: [], acceptanceCriteria: [], dependsOn: [] }]);
    expect(result.goal.scopeHistory).toEqual([{ at: "", targetCount: 0, closureSize: 0, completed: 0 }]);
    expect(result.eta).toEqual({ p50Hours: 0, p80Hours: 0, p50Label: "", p80Label: "", basis: "", basisLabel: "", confidence: "", remainingItems: 0, laneCapacity: 0 });
    expect(result.scopeEntities).toEqual({ items: { "execute/only-owner-ref": { name: "only-owner-ref", title: "only-owner-ref", description: "", status: "backlog", priority: 0, tags: [], created: "", updated: "", kind: "execute", dependsOn: [] } } });
  });
  it("preserves the same complete owner projection through the canonical list envelope", async () => {
    expectedReads = [["/goals"]]; vi.mocked(client.get).mockResolvedValue({ items: [richWire] });
    expect(await service.list()).toEqual([richCanonical]);
  });
  it("propagates the exact read refusal without an alternate read or repair write", async () => {
    expectedReads = [["/goals/fixture-refused"]]; const error = new Error("Owner read unavailable"); vi.mocked(client.get).mockRejectedValue(error);
    await expect(service.get("fixture-refused")).rejects.toBe(error);
  });
  it("does not turn a non-array list projection into a successful empty list", async () => {
    expectedReads = [["/goals"]]; vi.mocked(client.get).mockResolvedValue({ items: { unsupported: "not an owner list" } });
    await expect(service.list()).rejects.toThrow();
  });
  it("rejects malformed typed file sizes on the exact owner read without an alternate route", async () => {
    expectedReads = [["/goals/fixture-files/files"]]; vi.mocked(client.get).mockResolvedValue({ files: [{ name: "retained.md", path: "retained.md", type: "file", size: "not-an-int64" }] });
    await expect(service.getFiles("fixture-files")).rejects.toThrow();
  });
});
