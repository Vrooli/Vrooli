import { describe, expect, it, vi } from "vitest";
import {
  InvestigationStatus,
  TaskType,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/tasks_pb";

const { client } = vi.hoisted(() => ({
  client: {
    createTask: vi.fn(),
    getTask: vi.fn(),
    listTasks: vi.fn(),
    stopTask: vi.fn(),
    getAgentManagerStatus: vi.fn(),
  },
}));
vi.mock("./connect", () => ({ taskConnectClient: client }));

import {
  createTask,
  getAgentManagerStatus,
  getTask,
  listTasks,
  stopTask,
} from "./tasks";

describe("task Connect client", () => {
  it("maps a fix request to the typed TaskService request", async () => {
    client.createTask.mockResolvedValue({
      task: {
        id: "task-1",
        pipelineId: "pipe-1",
        status: InvestigationStatus.PENDING,
        progress: 0,
        createdAt: { seconds: 0n, nanos: 0 },
        updatedAt: { seconds: 0n, nanos: 0 },
      },
    });
    await createTask("pipe-1", {
      taskType: TaskType.FIX,
      focus: { harness: true, subject: false },
      permissions: { immediate: true, permanent: false, prevention: false },
      maxIterations: 3,
    });
    expect(client.createTask).toHaveBeenCalledWith(
      expect.objectContaining({
        pipelineId: "pipe-1",
        taskType: TaskType.FIX,
        maxIterations: 3,
      }),
    );
  });

  it("returns generated agent availability and investigation summaries", async () => {
    client.getAgentManagerStatus.mockResolvedValue({
      available: true,
      url: "http://agent",
    });
    await expect(getAgentManagerStatus()).resolves.toEqual({
      available: true,
      url: "http://agent",
    });
    client.listTasks.mockResolvedValue({
      tasks: [
        {
          id: "task-1",
          pipelineId: "pipe-1",
          status: InvestigationStatus.COMPLETED,
          progress: 100,
          createdAt: { seconds: 0n, nanos: 0 },
        },
      ],
    });
    await expect(listTasks("pipe-1")).resolves.toMatchObject([
      { id: "task-1", status: InvestigationStatus.COMPLETED, progress: 100 },
    ]);
  });

  it("retrieves and stops a typed investigation, while rejecting absent task payloads", async () => {
    const task = {
      id: "task-2",
      pipelineId: "pipe-1",
      status: InvestigationStatus.RUNNING,
      progress: 25,
      createdAt: { seconds: 0n, nanos: 0 },
      updatedAt: { seconds: 0n, nanos: 0 },
    };
    client.getTask.mockResolvedValueOnce({ task }).mockResolvedValueOnce({});

    await expect(getTask("pipe-1", "task-2")).resolves.toEqual(task);
    await expect(stopTask("pipe-1", "task-2")).resolves.toBeUndefined();
    await expect(getTask("pipe-1", "task-missing")).rejects.toThrow(
      "Task service returned no investigation",
    );
    expect(client.getTask).toHaveBeenCalledWith({
      pipelineId: "pipe-1",
      taskId: "task-2",
    });
    expect(client.stopTask).toHaveBeenCalledWith({
      pipelineId: "pipe-1",
      taskId: "task-2",
    });
  });

  it("rejects a create response that omits its required investigation", async () => {
    client.createTask.mockResolvedValue({});

    await expect(
      createTask("pipe-1", {
        taskType: TaskType.FIX,
        focus: { harness: true, subject: false },
        permissions: { immediate: true, permanent: false, prevention: false },
        maxIterations: 1,
      }),
    ).rejects.toThrow("Task service returned no investigation");
  });
});
