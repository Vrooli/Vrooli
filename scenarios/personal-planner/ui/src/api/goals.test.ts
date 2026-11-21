import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ listGoals: vi.fn(), createGoal: vi.fn(), updateGoalProgress: vi.fn(), listMilestones: vi.fn(), createMilestone: vi.fn(), updateMilestoneStatus: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { completeMilestone, createGoal, createMilestone, fetchGoals, fetchMilestones, updateGoalProgress, updateGoalTargetDate } from "./goals";
import { API_BASE } from "./client";

afterEach(() => { vi.clearAllMocks(); vi.unstubAllGlobals(); });

describe("goals API transport", () => {
  it("reads and creates generic goals through REST while explicit progress stays on Connect", async () => {
    const goal = { id: "g-1", revision: 1n } as never;
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ goals: [{ id: "g-1", title: "Ship", purpose: "Useful", status: "active", progress_method: "manual", progress_basis_points: 0, target_basis_points: 10000, revision: 1 }] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ goal: { id: "g-1", title: "Ship", purpose: "Useful", status: "active", progress_method: "manual", progress_basis_points: 0, target_basis_points: 10000, revision: 1 } }), { status: 201 })));
    client.updateGoalProgress.mockResolvedValue({ goal });
    expect((await fetchGoals())[0]).toMatchObject({ id: "g-1", progressBasisPoints: 0n });
    expect(await createGoal({ title: "Ship", purpose: "Useful", progressMethod: "manual", targetBasisPoints: 10000n })).toMatchObject({ id: "g-1" });
    expect(fetch).toHaveBeenNthCalledWith(2, `${API_BASE}/api/v1/goals`, expect.objectContaining({ method: "POST" }));
    expect(await updateGoalProgress(goal, 2500n)).toBe(goal);
    expect(client.updateGoalProgress).toHaveBeenCalledWith({ id: "g-1", progressBasisPoints: 2500n, expectedRevision: 1n });
  });

  it("round-trips one-time and recurring currency goal metadata without inventing progress", async () => {
    const payloads = [
      { id: "one", title: "First $1,000", purpose: "", status: "active", progress_method: "manual", progress_state: "unknown" as const, target_basis_points: 10000, revision: 1, target_date: "2026-11-30", measure: { unit: "currency" as const, target_amount_minor: 100000, currency_code: "USD", basis: "after_costs_before_tax" as const, recurrence: "one_time" as const, period: "once" as const } },
      { id: "monthly", title: "Recurring $1,000/month", purpose: "", status: "active", progress_method: "manual", progress_state: "unknown" as const, target_basis_points: 10000, revision: 1, target_date: "2026-12-31", measure: { unit: "currency" as const, target_amount_minor: 100000, currency_code: "USD", basis: "after_costs_before_tax" as const, recurrence: "recurring" as const, period: "month" as const } },
    ];
    const oneTimePayload = payloads[0];
    const monthlyPayload = payloads[1];
    if (!oneTimePayload || !monthlyPayload) throw new Error("Expected both payload fixtures");
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ goal: oneTimePayload }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ goal: monthlyPayload }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ goals: payloads }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const oneTime = await createGoal({ title: oneTimePayload.title, purpose: "", progressMethod: "manual", targetBasisPoints: 10000n, targetDate: "2026-11-30", measure: { unit: "currency", targetAmountMinor: 100000, currencyCode: "USD", basis: "after_costs_before_tax", recurrence: "one_time", period: "once" } });
    const monthly = await createGoal({ title: monthlyPayload.title, purpose: "", progressMethod: "manual", targetBasisPoints: 10000n, targetDate: "2026-12-31", measure: { unit: "currency", targetAmountMinor: 100000, currencyCode: "USD", basis: "after_costs_before_tax", recurrence: "recurring", period: "month" } });
    const readGoals = await fetchGoals();
    const readOneTime = readGoals[0];
    const readMonthly = readGoals[1];
    if (!readOneTime || !readMonthly) throw new Error("Expected both goal readbacks");
    expect(oneTime).toMatchObject({ targetDate: "2026-11-30", progressState: "unknown", measure: { recurrence: "one_time", period: "once", basis: "after_costs_before_tax" } });
    expect(monthly).toMatchObject({ targetDate: "2026-12-31", progressState: "unknown", measure: { recurrence: "recurring", period: "month", targetAmountMinor: 100000 } });
    expect(readOneTime).toMatchObject({ id: "one", progressState: "unknown", measure: { recurrence: "one_time" } });
    expect(readMonthly).toMatchObject({ id: "monthly", progressState: "unknown", measure: { recurrence: "recurring", period: "month" } });
  });

  it("maps milestone read and completion transport", async () => {
    const milestone = { id: "m-1", goalId: "g-1", revision: 2n } as never;
    client.listMilestones.mockResolvedValue({ milestones: [milestone] }); client.createMilestone.mockResolvedValue({ milestone }); client.updateMilestoneStatus.mockResolvedValue({ milestone });
    expect(await fetchMilestones("g-1")).toEqual([milestone]);
    expect(await createMilestone({ goalId: "g-1", title: "Proof", criteria: "Done", dueDate: "2026-10-01" })).toBe(milestone);
    expect(client.createMilestone).toHaveBeenCalledWith({ goalId: "g-1", title: "Proof", criteria: "Done", dueDate: "2026-10-01", linkedWorkItemId: "", prerequisiteMilestoneIds: [] });
    expect(await completeMilestone(milestone)).toBe(milestone);
    expect(client.updateMilestoneStatus).toHaveBeenCalledWith({ id: "m-1", status: "complete", expectedRevision: 2n });
  });

  it("rejects malformed mutation responses", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({}), { status: 201 })));
    await expect(createGoal({ title: "Ship", purpose: "Useful", progressMethod: "manual", targetBasisPoints: 10000n })).rejects.toThrow("no goal");
    client.updateGoalProgress.mockResolvedValue({});
    await expect(updateGoalProgress({ id: "g-1", revision: 1n } as never, 2500n)).rejects.toThrow("no goal");
  });

  it("persists an explicit target date through the REST seam", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 204 })));
    await updateGoalTargetDate({ id: "g/1", targetDate: "2026-10-01", reason: "Deadline clarified", expectedRevision: 7n });
    expect(fetch).toHaveBeenCalledWith(`${API_BASE}/api/v1/goals/g%2F1/target-date`, expect.objectContaining({ method: "POST", body: JSON.stringify({ target_date: "2026-10-01", reason: "Deadline clarified", expected_revision: 7 }) }));
  });

  it("reads ordered target date history from the goals owner", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ history: [{ goal_id: "g/1", previous_target_date: "2026-09-01", target_date: "2026-10-01", reason: "Deadline clarified", revision: 7, changed_at: "2026-09-20T12:00:00Z" }] }), { status: 200 })));
    await expect((await import("./goals")).fetchGoalTargetDateHistory("g/1")).resolves.toEqual([{ goalId: "g/1", previousTargetDate: "2026-09-01", targetDate: "2026-10-01", reason: "Deadline clarified", revision: 7, changedAt: "2026-09-20T12:00:00Z" }]);
    expect(fetch).toHaveBeenCalledWith(`${API_BASE}/api/v1/goals/g%2F1/target-date/history`);
  });
});
