import { buildUrl, throwIfNotOk } from "./client";
import type {
  AgentManagerStatus,
  CreateTaskRequest,
  CreateTaskResponse,
  GetTaskResponse,
  Investigation,
  InvestigationSummary,
  ListTasksResponse,
} from "../../types/investigation";

export async function getAgentManagerStatus(): Promise<AgentManagerStatus> {
  const response = await fetch(buildUrl("/agent-manager/status"));
  await throwIfNotOk(response);
  return await response.json() as AgentManagerStatus;
}

export async function createTask(
  pipelineId: string,
  request: CreateTaskRequest
): Promise<Investigation> {
  const response = await fetch(buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  await throwIfNotOk(response);
  const data = await response.json() as CreateTaskResponse;
  return data.task;
}

export async function listTasks(
  pipelineId: string,
  limit?: number
): Promise<InvestigationSummary[]> {
  const params = new URLSearchParams();
  if (limit) {
    params.set("limit", String(limit));
  }
  const url = buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks`) +
    (params.toString() ? `?${params.toString()}` : "");
  const response = await fetch(url);
  await throwIfNotOk(response);
  const data = await response.json() as ListTasksResponse;
  return data.tasks;
}

export async function getTask(
  pipelineId: string,
  taskId: string
): Promise<Investigation> {
  const response = await fetch(
    buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks/${encodeURIComponent(taskId)}`)
  );
  await throwIfNotOk(response);
  const data = await response.json() as GetTaskResponse;
  return data.task;
}

export async function stopTask(
  pipelineId: string,
  taskId: string
): Promise<void> {
  const response = await fetch(
    buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks/${encodeURIComponent(taskId)}/stop`),
    { method: "POST" }
  );
  await throwIfNotOk(response);
  // Note: We don't need to check data.success since throwIfNotOk already handles errors
}
