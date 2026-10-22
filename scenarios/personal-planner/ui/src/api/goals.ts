import { createClient } from "@connectrpc/connect";
import { GoalsService, type Goal, type Milestone } from "@vrooli/proto-types/personal-planner/v1/goals/goals_pb";

import { API_BASE, transport } from "./client";

const goalsClient = createClient(GoalsService, transport);

export type GoalMeasure = { unit: "currency"; targetAmountMinor: number; currencyCode: string; basis: "after_costs_before_tax"; recurrence: "one_time" | "recurring"; period: "once" | "month" };
export type PlannerGoal = Goal & { targetDate?: string; measure?: GoalMeasure; progressState?: "unknown" };
type GoalPayload = { id: string; title: string; purpose: string; status: string; progress_method: string; progress_basis_points?: number; progress_state?: "unknown"; target_basis_points: number; revision: number; target_date?: string; measure?: { unit: "currency"; target_amount_minor: number; currency_code: string; basis: "after_costs_before_tax"; recurrence: "one_time" | "recurring"; period: "once" | "month" } };

function fromPayload(goal: GoalPayload): PlannerGoal {
	return { id: goal.id, title: goal.title, purpose: goal.purpose, status: goal.status, progressMethod: goal.progress_method, progressBasisPoints: BigInt(goal.progress_basis_points ?? 0), targetBasisPoints: BigInt(goal.target_basis_points), revision: BigInt(goal.revision), targetDate: goal.target_date, progressState: goal.progress_state, measure: goal.measure && { unit: goal.measure.unit, targetAmountMinor: goal.measure.target_amount_minor, currencyCode: goal.measure.currency_code, basis: goal.measure.basis, recurrence: goal.measure.recurrence, period: goal.measure.period } } as PlannerGoal;
}

export type GoalTargetDateHistory = { goalId: string; previousTargetDate: string; targetDate: string; reason: string; revision: number; changedAt: string };

export async function fetchGoals(): Promise<PlannerGoal[]> {
	const response = await fetch(`${API_BASE}/api/v1/goals`);
	if (!response.ok) throw new Error("Goals are unavailable");
	const payload = await response.json() as { goals?: GoalPayload[] };
	return (payload.goals ?? []).map(fromPayload);
}

export async function createGoal(input: { title: string; purpose: string; progressMethod: string; targetBasisPoints: bigint; targetDate?: string; measure?: GoalMeasure }): Promise<PlannerGoal> {
	const response = await fetch(`${API_BASE}/api/v1/goals`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ title: input.title, purpose: input.purpose, progress_method: input.progressMethod, target_basis_points: Number(input.targetBasisPoints), target_date: input.targetDate ?? "", measure: input.measure && { unit: input.measure.unit, target_amount_minor: input.measure.targetAmountMinor, currency_code: input.measure.currencyCode, basis: input.measure.basis, recurrence: input.measure.recurrence, period: input.measure.period } }) });
	if (!response.ok) throw new Error("The goal could not be saved");
	const payload = await response.json() as { goal?: GoalPayload };
	if (!payload.goal) throw new Error("The server returned no goal");
	return fromPayload(payload.goal);
}

export async function updateGoalProgress(goal: Goal, progressBasisPoints: bigint): Promise<Goal> {
  const response = await goalsClient.updateGoalProgress({ id: goal.id, progressBasisPoints, expectedRevision: goal.revision });
  if (!response.goal) throw new Error("The server returned no goal");
  return response.goal;
}

export async function updateGoalTargetDate(input: { id: string; targetDate: string; reason: string; expectedRevision: bigint }): Promise<void> {
	const response = await fetch(`${API_BASE}/api/v1/goals/${encodeURIComponent(input.id)}/target-date`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ target_date: input.targetDate, reason: input.reason, expected_revision: Number(input.expectedRevision) }) });
	if (!response.ok) throw new Error("The goal target date could not be saved");
}

export async function fetchGoalTargetDateHistory(goalId: string): Promise<GoalTargetDateHistory[]> {
	const response = await fetch(`${API_BASE}/api/v1/goals/${encodeURIComponent(goalId)}/target-date/history`);
	if (!response.ok) throw new Error("Goal target date history is unavailable");
	const payload = await response.json() as { history?: Array<{ goal_id: string; previous_target_date: string; target_date: string; reason: string; revision: number; changed_at: string }> };
	return (payload.history ?? []).map((item) => ({ goalId: item.goal_id, previousTargetDate: item.previous_target_date, targetDate: item.target_date, reason: item.reason, revision: item.revision, changedAt: item.changed_at }));
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
