import { createClient } from "@connectrpc/connect";
import { GoalsService, type Goal, type Milestone } from "@vrooli/proto-types/personal-planner/v1/goals/goals_pb";

import { transport } from "./client";

const goalsClient = createClient(GoalsService, transport);

export async function fetchGoals(): Promise<Goal[]> {
  const response = await goalsClient.listGoals({});
  return response.goals;
}

export async function createGoal(input: { title: string; purpose: string; progressMethod: string; targetBasisPoints: bigint }): Promise<Goal> {
	const response = await goalsClient.createGoal(input);
  if (!response.goal) throw new Error("The server returned no goal");
  return response.goal;
}

export async function updateGoalProgress(goal: Goal, progressBasisPoints: bigint): Promise<Goal> {
  const response = await goalsClient.updateGoalProgress({ id: goal.id, progressBasisPoints, expectedRevision: goal.revision });
  if (!response.goal) throw new Error("The server returned no goal");
  return response.goal;
}

export async function fetchMilestones(goalId: string): Promise<Milestone[]> {
  const response = await goalsClient.listMilestones({ goalId });
  return response.milestones;
}

export async function createMilestone(input: { goalId: string; title: string; criteria: string; dueDate: string; linkedWorkItemId?: string; prerequisiteMilestoneIds?: string[] }): Promise<Milestone> {
	const response = await goalsClient.createMilestone({ ...input, linkedWorkItemId: input.linkedWorkItemId ?? "", prerequisiteMilestoneIds: input.prerequisiteMilestoneIds ?? [] });
  if (!response.milestone) throw new Error("The server returned no milestone");
  return response.milestone;
}

export async function completeMilestone(milestone: Milestone): Promise<Milestone> {
  const response = await goalsClient.updateMilestoneStatus({ id: milestone.id, status: "complete", expectedRevision: milestone.revision });
  if (!response.milestone) throw new Error("The server returned no milestone");
  return response.milestone;
}
