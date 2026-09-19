import type { MessageInitShape } from "@bufbuild/protobuf";
import {
  CreateTaskRequestSchema,
  type AgentManagerStatusResponse,
  type Investigation,
  type InvestigationSummary,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/tasks_pb";
import { taskConnectClient } from "./connect";

export type CreateTaskInput = Omit<
  MessageInitShape<typeof CreateTaskRequestSchema>,
  "pipelineId" | "$typeName"
>;

export async function getAgentManagerStatus(): Promise<AgentManagerStatusResponse> {
  return taskConnectClient.getAgentManagerStatus({});
}

export async function createTask(
  pipelineId: string,
  request: CreateTaskInput,
): Promise<Investigation> {
  const response = await taskConnectClient.createTask({
    pipelineId,
    ...request,
  });
  if (!response.task) throw new Error("Task service returned no investigation");
  return response.task;
}

export async function listTasks(
  pipelineId: string,
  limit?: number,
): Promise<InvestigationSummary[]> {
  const response = await taskConnectClient.listTasks({ pipelineId, limit });
  return response.tasks;
}

export async function getTask(
  pipelineId: string,
  taskId: string,
): Promise<Investigation> {
  const response = await taskConnectClient.getTask({ pipelineId, taskId });
  if (!response.task) throw new Error("Task service returned no investigation");
  return response.task;
}

export async function stopTask(
  pipelineId: string,
  taskId: string,
): Promise<void> {
  await taskConnectClient.stopTask({ pipelineId, taskId });
}
