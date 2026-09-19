import { createClient } from "@connectrpc/connect";
import { WorkService, type GetTodayPlanResponse, type WorkItem } from "@vrooli/proto-types/personal-planner/v1/work/work_pb";

import { transport } from "./client";

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

export type { WorkItem };
