import { describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createPlanService } from "./plan-service";

function mockClient(): IApiClient {
  return {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  };
}

describe("plan service Create-Work-From-Plan methods", () => {
  it("lists canonical plan summaries from the picker proxy", async () => {
    const api = mockClient();
    vi.mocked(api.get).mockResolvedValue({
      plans: [{
        id: "plan-1",
        slug: "alpha-plan",
        title: "Alpha",
        status: "READY",
        updated_at: "2026-07-08T12:00:00Z",
        phase_count: 3,
      }],
    });

    const plans = await createPlanService(api).listCanonicalPlans();

    expect(api.get).toHaveBeenCalledWith("/plan-import/plans", { signal: undefined });
    expect(plans).toEqual([{
      id: "plan-1",
      slug: "alpha-plan",
      title: "Alpha",
      status: "READY",
      updatedAt: "2026-07-08T12:00:00Z",
      createdAt: undefined,
      phaseCount: 3,
    }]);
  });

  it("posts an import request and maps item and initiative results", async () => {
    const api = mockClient();
    vi.mocked(api.post).mockResolvedValue({
      slug: "alpha-plan",
      plan_id: "plan-1",
      container: "initiative",
      items: [{ kind: "execute", name: "alpha-phase-1", title: "Build", action: "created" }],
      initiative: { name: "alpha", title: "Alpha", mode: "phased-plan-drain", action: "created" },
      count: 1,
      created: 1,
      linked: 0,
      updated: 0,
    });

    const result = await createPlanService(api).importPlan({
      planId: "plan-1",
      container: { type: "initiative", name: "alpha", mode: "phased-plan-drain" },
    });

    expect(api.post).toHaveBeenCalledWith("/plan-import", {
      plan_id: "plan-1",
      source_path: undefined,
      markdown: undefined,
      title: undefined,
      slug: undefined,
      container: { type: "initiative", name: "alpha", mode: "phased-plan-drain" },
    }, { signal: undefined });
    expect(result.initiative?.name).toBe("alpha");
    expect(result.items[0]?.name).toBe("alpha-phase-1");
  });
});
