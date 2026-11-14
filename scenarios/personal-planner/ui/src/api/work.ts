import { createClient } from "@connectrpc/connect";
import { WorkService, type GetTodayPlanResponse, type WorkItem } from "@vrooli/proto-types/personal-planner/v1/work/work_pb";

import { API_BASE, transport } from "./client";

export const workClient = createClient(WorkService, transport);

export async function fetchWorkItems(): Promise<WorkItem[]> {
  const response = await workClient.listWorkItems({});
  return response.workItems;
}

export async function createWorkItem(input: { title: string; description?: string; remainingMinutes?: number; sourceLabel?: string }): Promise<WorkItem> {
  const response = await workClient.createWorkItem({
    title: input.title,
    description: input.description ?? "",
    remainingMinutes: input.remainingMinutes ?? 0,
    sourceLabel: input.sourceLabel ?? "",
  });
  if (!response.workItem) throw new Error("work item was not returned");
  return response.workItem;
}

export async function fetchTodayPlan(): Promise<GetTodayPlanResponse> {
  return workClient.getTodayPlan({});
}

export async function snoozeWorkItem(input: { id: string; until: string; reason?: string }): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/work/${encodeURIComponent(input.id)}/snooze`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ until: input.until, reason: input.reason ?? "not_now" }),
  });
  if (!response.ok) throw new Error("The work item could not be deferred");
}

export async function completeWorkItem(id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/work/${encodeURIComponent(id)}/complete`, { method: "POST" });
  if (!response.ok) throw new Error("The work item could not be completed");
}

export async function updateWorkEstimate(input: { id: string; remainingMinutes: number; reason?: string }): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/work/${encodeURIComponent(input.id)}/estimate`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ remaining_minutes: input.remainingMinutes, reason: input.reason ?? "" }),
  });
  if (!response.ok) throw new Error("The work estimate could not be updated");
}

export type { WorkItem };
