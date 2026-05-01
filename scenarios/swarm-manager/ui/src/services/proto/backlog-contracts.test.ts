import { describe, expect, it } from "vitest";
import type { BacklogItem as ProtoBacklogItem } from "@vrooli/proto-types/swarm-manager/v1/domain/backlog_pb";
import { mapProtoBacklogItem } from "./backlog-contracts";

describe("mapProtoBacklogItem", () => {
  it("maps created_by attribution into the UI domain shape", () => {
    const item = mapProtoBacklogItem({
      name: "session-created",
      title: "Session Created",
      description: "Created from a native session.",
      status: "backlog",
      priority: 1,
      tags: [],
      created: "2026-05-01T12:00:00Z",
      updated: "2026-05-01T12:00:00Z",
      kind: "execute",
      suggestedSkills: [],
      createdBy: {
        type: "agent",
        runId: "run-1",
        taskId: "task-1",
        profileKey: "swarm-manager/default",
        sessionId: "sess_1",
        sessionKind: "meta_orchestration",
        source: "session/sess_1",
      },
    } as unknown as ProtoBacklogItem);

    expect(item.createdBy).toEqual({
      type: "agent",
      runId: "run-1",
      taskId: "task-1",
      profileKey: "swarm-manager/default",
      sessionId: "sess_1",
      sessionKind: "meta_orchestration",
      source: "session/sess_1",
    });
  });
});
