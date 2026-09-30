import { describe, it, expect, vi, beforeEach } from "vitest";
import { createGoalsService, type IGoalsService } from "./goals-service";
import type { IApiClient } from "../lib/api-client";

describe("Goals Service", () => {
  let mockApiClient: IApiClient;
  let service: IGoalsService;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    service = createGoalsService(mockApiClient);
  });

  it("normalizes snake_case goal + scope + eta from the API", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      goal: {
        name: "monetization-v1",
        title: "Monetization v1",
        status: "active",
        priority: 7,
        targets: ["execute/foo"],
        milestones: [{
          name: "foundation",
          title: "Foundation",
          items: ["execute/dep"],
          acceptance_criteria: ["Dependencies are complete"],
          depends_on: [],
        }],
        seeded: true,
        scope_history: [{ at: "t0", target_count: 2, closure_size: 9, completed: 3 }],
        created: "c",
        updated: "u",
      },
      scope: {
        closure: ["execute/foo", "execute/dep"],
        completed: ["execute/dep"],
        total: 2,
        completed_count: 1,
        blocked_count: 0,
        progress_pct: 50,
      },
      eta: { p50_hours: 120, p80_hours: 240, p50_label: "~5 days", p80_label: "~10 days", basis: "live", basis_label: "27 samples", confidence: "high", remaining_items: 1, lane_capacity: 3 },
    });

    const res = await service.get("monetization-v1");
    expect(res.goal.priority).toBe(7);
    expect(res.goal.seeded).toBe(true);
    expect(res.goal.scopeHistory[0]).toEqual({ at: "t0", targetCount: 2, closureSize: 9, completed: 3 });
    expect(res.goal.milestones).toEqual([{
      name: "foundation",
      title: "Foundation",
      description: "",
      items: ["execute/dep"],
      acceptanceCriteria: ["Dependencies are complete"],
      dependsOn: [],
    }]);
    expect(res.scope.completedCount).toBe(1);
    expect(res.scope.progressPct).toBe(50);
    expect(res.eta?.basisLabel).toBe("27 samples");
    expect(res.eta?.laneCapacity).toBe(3);
  });

  it("unwraps the `items` list envelope emitted by the API", async () => {
    // The Go handler responds with {"items": [...]} — the real wire shape.
    vi.mocked(mockApiClient.get).mockResolvedValueOnce({ items: [{ goal: { name: "a", title: "A", status: "active" }, scope: {} }] });
    const wrapped = await service.list();
    expect(wrapped).toHaveLength(1);
    expect(wrapped[0]?.goal.name).toBe("a");
    expect(wrapped[0]?.eta).toBeNull();
  });

  it("preserves the operator-terminal achieved status", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValueOnce({ items: [{ goal: { name: "done", title: "Done", status: "achieved" }, scope: {} }] });
    const goals = await service.list();
    expect(goals[0]?.goal.status).toBe("achieved");
  });

  it("tolerates a legacy `goals` envelope and a bare array", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValueOnce({ goals: [{ goal: { name: "g", title: "G", status: "active" }, scope: {} }] });
    const legacy = await service.list();
    expect(legacy[0]?.goal.name).toBe("g");

    vi.mocked(mockApiClient.get).mockResolvedValueOnce([{ goal: { name: "b", title: "B", status: "active" }, scope: {} }]);
    const bare = await service.list();
    expect(bare[0]?.goal.name).toBe("b");
  });

  it("create posts only the provided fields", async () => {
    vi.mocked(mockApiClient.post).mockResolvedValue({ goal: { name: "g", title: "G", status: "active" }, scope: {} });
    await service.create({ title: "G", targets: ["execute/x"] });
    expect(mockApiClient.post).toHaveBeenCalledWith("/goals", { title: "G", targets: ["execute/x"] });
  });

  it("setPriority routes through update with the priority field", async () => {
    vi.mocked(mockApiClient.put).mockResolvedValue({ goal: { name: "g", title: "G", status: "active", priority: 4 }, scope: {} });
    await service.setPriority("g", 4);
    expect(mockApiClient.put).toHaveBeenCalledWith("/goals/g", { priority: 4 });
  });

  it("addTargets / removeTargets hit the targets endpoint with a body", async () => {
    vi.mocked(mockApiClient.post).mockResolvedValue({ goal: { name: "g", title: "G", status: "active" }, scope: {} });
    vi.mocked(mockApiClient.delete).mockResolvedValue({ goal: { name: "g", title: "G", status: "active" }, scope: {} });
    await service.addTargets("g", ["execute/x"]);
    expect(mockApiClient.post).toHaveBeenCalledWith("/goals/g/targets", { targets: ["execute/x"] });
    await service.removeTargets("g", ["execute/x"]);
    expect(mockApiClient.delete).toHaveBeenCalledWith("/goals/g/targets", { targets: ["execute/x"] });
  });
});
