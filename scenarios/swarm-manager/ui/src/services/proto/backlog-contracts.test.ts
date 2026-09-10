import { describe, expect, it } from "vitest";
import type { BacklogItem as ProtoBacklogItem } from "@vrooli/proto-types/swarm-manager/v1/domain/backlog_pb";
import { backlogItemResponseSchema, mapProtoBacklogItem } from "./backlog-contracts";

describe("mapProtoBacklogItem", () => {
  it("preserves the reviewed strategy, approval and grant limits from API JSON", () => {
    const parsed = backlogItemResponseSchema.parse({ item: {
      name: "develop-audio", title: "Develop Audio", status: "backlog", kind: "execute", priority: 1,
      created: "2026-09-10T12:00:00Z", updated: "2026-09-10T12:00:00Z",
      execution_strategy: "adaptive-improvement",
      plan_acceptance: { actor: "operator", accepted_at: "2026-09-10T12:00:00Z", plan_content_hash: "a".repeat(64), subject_version: "b".repeat(64) },
      execution_limits: { max_slices: 128, max_tokens: "2000000", max_wall_seconds: "604800", max_turns: 2400, max_charge_micro_usd: "250000000", max_children: 512, max_node_attempts: 512, max_retries: 128 },
    } });
    const item = mapProtoBacklogItem(parsed.item!);
    expect(item.executionStrategy).toBe("adaptive-improvement");
    expect(item.planAcceptance).toEqual({ actor: "operator", acceptedAt: "2026-09-10T12:00:00Z", planContentHash: "a".repeat(64), subjectVersion: "b".repeat(64) });
    expect(item.executionLimits).toEqual({ maxSlices: 128, maxTokens: 2000000, maxWallSeconds: 604800, maxTurns: 2400, maxChargeMicroUsd: 250000000, maxChildren: 512, maxNodeAttempts: 512, maxRetries: 128 });
  });

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
