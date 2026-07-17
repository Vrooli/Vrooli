import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import {
  AgentOpsBindingLayer,
  AgentOpsDomainAction,
  AgentOpsWorkflowState,
} from "@vrooli/proto-types/swarm-manager/v1/domain/agent_operations_pb";
import { OperatingModeTargetKind } from "@vrooli/proto-types/swarm-manager/v1/domain/operating_mode_pb";
import {
  AgentOpsGetMigrationStatusResponseSchema,
  AgentOpsGetResolvedBindingsResponseSchema,
  AgentOpsGetWorkflowProjectionResponseSchema,
  AgentOpsListExecutionHistoryResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/agent_operations_pb";
import {
  mapBindingLayer,
  mapDomainAction,
  mapExecutionHistory,
  mapMigrationState,
  mapMigrationStatus,
  mapOperationState,
  mapResolvedBindings,
  mapWorkflowProjection,
  mapWorkflowState,
  targetKindToProto,
} from "./agent-operations-contracts";

// ---------------------------------------------------------------------------
// Enum exhaustiveness — every generated enum member must map to a non-numeric
// kebab-case domain string, and only UNSPECIFIED may map to "unspecified".
// ---------------------------------------------------------------------------

function numericMembers<E extends number>(protoEnum: Record<string, string | number>): E[] {
  return Object.values(protoEnum).filter((v): v is E => typeof v === "number");
}

describe("agent-operations enum mappings", () => {
  it("maps every AgentOpsBindingLayer member exhaustively", () => {
    for (const value of numericMembers<AgentOpsBindingLayer>(AgentOpsBindingLayer)) {
      const mapped = mapBindingLayer(value);
      expect(typeof mapped).toBe("string");
      if (value !== AgentOpsBindingLayer.UNSPECIFIED) {
        expect(mapped).not.toBe("unspecified");
      }
    }
    expect(mapBindingLayer(AgentOpsBindingLayer.SYSTEM_DEFAULT)).toBe("system-default");
    expect(mapBindingLayer(AgentOpsBindingLayer.INITIATIVE_OVERRIDE)).toBe("initiative-override");
    expect(mapBindingLayer(AgentOpsBindingLayer.BACKLOG_ITEM_OVERRIDE)).toBe("backlog-item-override");
    expect(mapBindingLayer(AgentOpsBindingLayer.AUTHORIZED_INVOCATION)).toBe("authorized-invocation");
    expect(mapBindingLayer(undefined)).toBe("unspecified");
    // Unknown wire value from a newer server degrades safely.
    expect(mapBindingLayer(999 as AgentOpsBindingLayer)).toBe("unspecified");
  });

  it("maps every AgentOpsWorkflowState member exhaustively", () => {
    for (const value of numericMembers<AgentOpsWorkflowState>(AgentOpsWorkflowState)) {
      const mapped = mapWorkflowState(value);
      expect(typeof mapped).toBe("string");
      if (value !== AgentOpsWorkflowState.UNSPECIFIED) {
        expect(mapped).not.toBe("unspecified");
      }
    }
    expect(mapWorkflowState(AgentOpsWorkflowState.AWAITING_DECISION)).toBe("awaiting-decision");
    expect(mapWorkflowState(AgentOpsWorkflowState.TERMINAL_COMPLETE)).toBe("terminal-complete");
    expect(mapWorkflowState(undefined)).toBe("unspecified");
  });

  it("maps every AgentOpsDomainAction member exhaustively", () => {
    const seen = new Set<string>();
    for (const value of numericMembers<AgentOpsDomainAction>(AgentOpsDomainAction)) {
      const mapped = mapDomainAction(value);
      expect(typeof mapped).toBe("string");
      if (value !== AgentOpsDomainAction.UNSPECIFIED) {
        expect(mapped).not.toBe("unspecified");
        // Every non-unspecified action maps to a DISTINCT domain string.
        expect(seen.has(mapped)).toBe(false);
        seen.add(mapped);
      }
    }
    expect(mapDomainAction(AgentOpsDomainAction.QUEUE_PLAN_EXECUTION)).toBe("queue-plan-execution");
    expect(mapDomainAction(AgentOpsDomainAction.COMMIT_WORKSHOP_ROUND)).toBe("commit-workshop-round");
    expect(mapDomainAction(AgentOpsDomainAction.MARK_INITIATIVE_REVIEWED)).toBe("mark-initiative-reviewed");
  });

  it("maps target kinds to the proto selector enum", () => {
    expect(targetKindToProto("backlog-item")).toBe(OperatingModeTargetKind.BACKLOG_ITEM);
    expect(targetKindToProto("initiative")).toBe(OperatingModeTargetKind.INITIATIVE);
  });

  it("normalizes operation and migration state strings", () => {
    expect(mapOperationState("running")).toBe("running");
    expect(mapOperationState("needs-attention")).toBe("needs-attention");
    expect(mapOperationState("bogus")).toBe("");
    expect(mapOperationState(undefined)).toBe("");
    expect(mapMigrationState("staged")).toBe("staged");
    expect(mapMigrationState("quarantined")).toBe("quarantined");
    expect(mapMigrationState("")).toBe("not-started");
    expect(mapMigrationState("weird")).toBe("not-started");
  });
});

// ---------------------------------------------------------------------------
// Message mappings
// ---------------------------------------------------------------------------

describe("mapWorkflowProjection", () => {
  it("maps the full projection: operations, decisions, timers, legal actions, policy pin", () => {
    const resp = create(AgentOpsGetWorkflowProjectionResponseSchema, {
      found: true,
      policyId: "backlog-item-policy",
      policyRevision: "sha256:deadbeef",
      workflow: {
        instanceId: "wf-1",
        domainKind: "backlog-item",
        domainId: "feature/foo",
        state: AgentOpsWorkflowState.AWAITING_DECISION,
        version: 7,
        decisions: [{ decision: "approve-plan", actor: "matt", atVersion: 5, note: "lgtm" }],
        timers: [
          {
            intent: "auto-advance",
            action: AgentOpsDomainAction.COMMIT_WORKSHOP_ROUND,
            notBefore: "2026-07-15T00:00:00Z",
          },
        ],
        legalActions: [
          AgentOpsDomainAction.SAVE_DECISIONS,
          AgentOpsDomainAction.COMMIT_WORKSHOP_ROUND,
        ],
      },
      operations: [
        {
          operation: "workshop-round",
          operationVersion: "1.0.0",
          executionId: "exec-1",
          runId: "run-1",
          state: "completed",
          outcome: "success",
          idempotencyKey: "idem-1",
          provenanceDigest: "sha256:aaaa",
          mode: "workshop-loop",
          modeRevision: "sha256:bbbb",
          bindingLayer: AgentOpsBindingLayer.INITIATIVE_OVERRIDE,
          bindingOwnerKind: "initiative",
          bindingOwnerId: "init-a",
          recordedAt: "2026-07-14T12:00:00Z",
          snapshotFound: true,
          attempt: 2,
          priorExecutionId: "exec-0",
        },
      ],
    });

    const projection = mapWorkflowProjection(resp);

    expect(projection.found).toBe(true);
    expect(projection.instanceId).toBe("wf-1");
    expect(projection.state).toBe("awaiting-decision");
    expect(projection.version).toBe(7);
    expect(projection.policyId).toBe("backlog-item-policy");
    expect(projection.policyRevision).toBe("sha256:deadbeef");

    expect(projection.decisions).toEqual([
      { decision: "approve-plan", actor: "matt", atVersion: 5, note: "lgtm" },
    ]);
    expect(projection.timers).toEqual([
      { intent: "auto-advance", action: "commit-workshop-round", notBefore: "2026-07-15T00:00:00Z" },
    ]);
    expect(projection.legalActions).toEqual(["save-decisions", "commit-workshop-round"]);

    const op = projection.operations[0];
    expect(op).toMatchObject({
      operation: "workshop-round",
      operationVersion: "1.0.0",
      executionId: "exec-1",
      runId: "run-1",
      state: "completed",
      outcome: "success",
      mode: "workshop-loop",
      modeRevision: "sha256:bbbb",
      bindingLayer: "initiative-override",
      bindingOwnerKind: "initiative",
      bindingOwnerId: "init-a",
      snapshotFound: true,
      attempt: 2,
      priorExecutionId: "exec-0",
    });
  });

  it("maps a not-found projection to found=false with empty collections", () => {
    const projection = mapWorkflowProjection(
      create(AgentOpsGetWorkflowProjectionResponseSchema, { found: false }),
    );
    expect(projection.found).toBe(false);
    expect(projection.operations).toEqual([]);
    expect(projection.legalActions).toEqual([]);
    expect(projection.decisions).toEqual([]);
    expect(projection.timers).toEqual([]);
  });
});

describe("mapExecutionHistory", () => {
  it("maps summaries with digests and the reproducible flag", () => {
    const resp = create(AgentOpsListExecutionHistoryResponseSchema, {
      executions: [
        {
          executionId: "exec-9",
          operation: "plan-execution",
          operationVersion: "2.1.0",
          mode: "execute-loop",
          modeRevision: "sha256:cccc",
          bindingLayer: AgentOpsBindingLayer.SYSTEM_DEFAULT,
          compiledModeDigest: "sha256:dddd",
          promptCatalogDigest: "sha256:eeee",
          callerInputDigest: "sha256:ffff",
          outcome: "success",
          reproducible: true,
          recordedAt: "2026-07-14T10:00:00Z",
        },
      ],
    });
    expect(mapExecutionHistory(resp)).toEqual([
      {
        executionId: "exec-9",
        operation: "plan-execution",
        operationVersion: "2.1.0",
        mode: "execute-loop",
        modeRevision: "sha256:cccc",
        bindingLayer: "system-default",
        compiledModeDigest: "sha256:dddd",
        promptCatalogDigest: "sha256:eeee",
        callerInputDigest: "sha256:ffff",
        outcome: "success",
        reproducible: true,
        legacyImport: false,
        recordedAt: "2026-07-14T10:00:00Z",
      },
    ]);
  });
});

describe("mapResolvedBindings", () => {
  it("maps typed per-operation results including fail-closed errors and contributions", () => {
    const resp = create(AgentOpsGetResolvedBindingsResponseSchema, {
      operations: [
        {
          operation: "workshop-round",
          operationVersion: "1.0.0",
          resolved: true,
          binding: {
            operation: "workshop-round",
            operationVersion: "1.0.0",
            layer: AgentOpsBindingLayer.BACKLOG_ITEM_OVERRIDE,
            owner: { kind: "backlog-item", id: "feature/foo" },
            mode: "workshop-loop",
            modeRevision: "sha256:1111",
            disabled: false,
          },
          policyId: "p",
          policyRevision: "r",
          contributions: [
            {
              binding: {
                operation: "workshop-round",
                layer: AgentOpsBindingLayer.SYSTEM_DEFAULT,
                owner: { kind: "system", id: "" },
                mode: "workshop-loop",
                modeRevision: "sha256:0000",
              },
              winning: false,
            },
          ],
        },
        {
          operation: "review-round",
          resolved: false,
          error: "no-binding",
          errorMessage: "no binding in scope",
        },
      ],
    });

    const [resolved, failed] = mapResolvedBindings(resp);
    expect(resolved?.resolved).toBe(true);
    expect(resolved?.binding).toMatchObject({
      layer: "backlog-item-override",
      ownerKind: "backlog-item",
      ownerId: "feature/foo",
      mode: "workshop-loop",
    });
    expect(resolved?.contributions).toHaveLength(1);
    expect(resolved?.contributions[0]).toMatchObject({
      winning: false,
      binding: { layer: "system-default" },
    });
    expect(failed).toMatchObject({
      operation: "review-round",
      resolved: false,
      binding: null,
      error: "no-binding",
      errorMessage: "no binding in scope",
    });
  });
});

describe("mapMigrationStatus", () => {
  it("maps the document fields and normalizes the state", () => {
    const resp = create(AgentOpsGetMigrationStatusResponseSchema, {
      state: "staged",
      epoch: 3,
      stagedCount: 12,
      promotedCount: 0,
      quarantinedCount: 1,
      startedAt: "2026-07-14T09:00:00Z",
      updatedAt: "2026-07-14T09:05:00Z",
      reportPath: "/reports/epoch-3.json",
      documentFound: true,
    });
    expect(mapMigrationStatus(resp)).toEqual({
      state: "staged",
      epoch: 3,
      stagedCount: 12,
      promotedCount: 0,
      quarantinedCount: 1,
      startedAt: "2026-07-14T09:00:00Z",
      updatedAt: "2026-07-14T09:05:00Z",
      reportPath: "/reports/epoch-3.json",
      documentFound: true,
    });
  });

  it("treats an absent document as not-started", () => {
    const status = mapMigrationStatus(
      create(AgentOpsGetMigrationStatusResponseSchema, { documentFound: false }),
    );
    expect(status.state).toBe("not-started");
    expect(status.documentFound).toBe(false);
  });
});
