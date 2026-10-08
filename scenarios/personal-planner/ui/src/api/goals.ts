import { createClient } from "@connectrpc/connect";
import { GoalsService, type Goal, type Milestone } from "@vrooli/proto-types/personal-planner/v1/goals/goals_pb";

import { API_BASE, transport } from "./client";

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

export async function updateGoalTargetDate(input: { id: string; targetDate: string }): Promise<void> {
	const response = await fetch(`${API_BASE}/api/v1/goals/${encodeURIComponent(input.id)}/target-date`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ target_date: input.targetDate }) });
	if (!response.ok) throw new Error("The goal target date could not be saved");
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
