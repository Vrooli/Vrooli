import { describe, it, expect, vi, beforeEach } from "vitest";
import { createInitiativeService, type IInitiativeService } from "./initiative-service";
import type { IApiClient } from "../lib/api-client";
import type { InitiativeWithRollup } from "../types";

describe("Initiative Service", () => {
  let mockApiClient: IApiClient;
  let service: IInitiativeService;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    service = createInitiativeService(mockApiClient);
  });

  it("lists initiatives with rollup", async () => {
    const mockData: InitiativeWithRollup[] = [
      {
        initiative: {
          name: "test-initiative",
          title: "Test Initiative",
          description: "A test",
          status: "active",
          mode: "item-level",
          acceptanceCriteria: [],
          priority: 0,
          dependsOn: [],
          items: ["execute/item-1"],
          created: "2026-03-28T00:00:00Z",
          updated: "2026-03-28T00:00:00Z",
        },
        rollup: { total: 1, completed: 0, inProgress: 1, failed: 0, pending: 0, archived: 0 },
      },
    ];

    vi.mocked(mockApiClient.get).mockResolvedValue(mockData);

    const result = await service.list();
    expect(mockApiClient.get).toHaveBeenCalledWith("/initiatives");
    expect(result).toEqual(mockData);
  });

  it("gets a single initiative with rollup", async () => {
    const mockData: InitiativeWithRollup = {
      initiative: {
        name: "my-initiative",
        title: "My Initiative",
        description: "Description",
          status: "completed",
          mode: "item-level",
          acceptanceCriteria: [],
          priority: 0,
        dependsOn: [],
        items: ["execute/a", "research/b"],
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
      rollup: { total: 2, completed: 2, inProgress: 0, failed: 0, pending: 0, archived: 0 },
    };

    vi.mocked(mockApiClient.get).mockResolvedValue(mockData);

    const result = await service.get("my-initiative");
    expect(mockApiClient.get).toHaveBeenCalledWith("/initiatives/my-initiative");
    expect(result).toEqual(mockData);
  });

  it("normalizes snake_case created_by attribution", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      initiative: {
        name: "session-created",
        title: "Session Created",
        description: "Created from a native session.",
        status: "active",
        created: "2026-05-01T12:00:00Z",
        updated: "2026-05-01T12:00:00Z",
        created_by: {
          type: "agent",
          run_id: "run-1",
          task_id: "task-1",
          profile_key: "swarm-manager/default",
          session_id: "sess_1",
          session_kind: "meta_orchestration",
          source: "session/sess_1",
        },
      },
      rollup: { total: 0, completed: 0, in_progress: 0, failed: 0, pending: 0, archived: 0 },
    });

    const result = await service.get("session-created");

    expect(result.initiative.createdBy).toEqual({
      type: "agent",
      runId: "run-1",
      taskId: "task-1",
      profileKey: "swarm-manager/default",
      sessionId: "sess_1",
      sessionKind: "meta_orchestration",
      source: "session/sess_1",
    });
  });

  it("updates acceptance criteria with backend field names", async () => {
    const mockData: InitiativeWithRollup = {
      initiative: {
        name: "my-initiative",
        title: "My Initiative",
        description: "Description",
        status: "active",
        mode: "holistic-loop",
        acceptanceCriteria: ["System passes review"],
        priority: 0,
        dependsOn: [],
        items: [],
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
      rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 },
    };

    vi.mocked(mockApiClient.put).mockResolvedValue(mockData);

    const result = await service.updateMetadata("my-initiative", {
      acceptanceCriteria: ["System passes review"],
    });

    expect(mockApiClient.put).toHaveBeenCalledWith("/initiatives/my-initiative", {
      acceptance_criteria: ["System passes review"],
    });
    expect(result.initiative.mode).toBe("holistic-loop");
    expect(result.initiative.acceptanceCriteria).toEqual(["System passes review"]);
  });

  it("propagates errors", async () => {
    vi.mocked(mockApiClient.get).mockRejectedValue(new Error("Network error"));
    await expect(service.get("bad")).rejects.toThrow("Network error");
  });
});
