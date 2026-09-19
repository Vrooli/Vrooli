import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ listGoals: vi.fn(), createGoal: vi.fn(), updateGoalProgress: vi.fn(), listMilestones: vi.fn(), createMilestone: vi.fn(), updateMilestoneStatus: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { completeMilestone, createGoal, createMilestone, fetchGoals, fetchMilestones, updateGoalProgress } from "./goals";

afterEach(() => vi.clearAllMocks());

describe("goals API transport", () => {
  it("maps list, create, and explicit progress through Connect", async () => {
    const goal = { id: "g-1", revision: 1n } as never;
    client.listGoals.mockResolvedValue({ goals: [goal] }); client.createGoal.mockResolvedValue({ goal }); client.updateGoalProgress.mockResolvedValue({ goal });
    expect(await fetchGoals()).toEqual([goal]);
    expect(await createGoal({ title: "Ship", purpose: "Useful", progressMethod: "manual", targetBasisPoints: 10000n })).toBe(goal);
    expect(await updateGoalProgress(goal, 2500n)).toBe(goal);
    expect(client.updateGoalProgress).toHaveBeenCalledWith({ id: "g-1", progressBasisPoints: 2500n, expectedRevision: 1n });
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
    client.createGoal.mockResolvedValue({});
    await expect(createGoal({ title: "Ship", purpose: "Useful", progressMethod: "manual", targetBasisPoints: 10000n })).rejects.toThrow("no goal");
    client.updateGoalProgress.mockResolvedValue({});
    await expect(updateGoalProgress({ id: "g-1", revision: 1n } as never, 2500n)).rejects.toThrow("no goal");
  });
});
