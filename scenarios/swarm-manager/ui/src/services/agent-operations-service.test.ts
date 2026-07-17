// [REQ:REQ-P1-002-UI-OPERATIONS-PARITY]
import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  AgentOpsBindingLayer,
  AgentOpsDomainAction,
  AgentOpsWorkflowState,
} from "@vrooli/proto-types/swarm-manager/v1/domain/agent_operations_pb";
import { OperatingModeTargetKind } from "@vrooli/proto-types/swarm-manager/v1/domain/operating_mode_pb";
import {
  createAgentOperationsService,
  type AgentOperationsClient,
  type IAgentOperationsService,
} from "./agent-operations-service";

// The service consumes the generated AgentOperationsService Connect client and
// projects its proto messages onto the domain types. These tests drive it
// through a mock client (initiative-mode-service test pattern): each RPC is a
// vi.fn() returning a proto-shaped response, so we assert both the request the
// service builds and the mapped domain shape.
const RPCS = [
  "resolveBinding",
  "validateInvocation",
  "inspectWorkflow",
  "inspectExecution",
  "listOperationCatalog",
  "listCompatibleModes",
  "getResolvedBindings",
  "listBindingOverrides",
  "putBindingOverride",
  "deleteBindingOverride",
  "getWorkflowProjection",
  "listExecutionHistory",
  "getMigrationStatus",
] as const;

type MockClient = Record<(typeof RPCS)[number], ReturnType<typeof vi.fn>>;

function makeClient(): MockClient {
  const client = {} as MockClient;
  for (const rpc of RPCS) client[rpc] = vi.fn();
  return client;
}

describe("Agent Operations Service", () => {
  let client: MockClient;
  let service: IAgentOperationsService;

  beforeEach(() => {
    client = makeClient();
    service = createAgentOperationsService(client as unknown as AgentOperationsClient);
  });

  it("builds a backlog-item selector and maps the workflow projection", async () => {
    client.getWorkflowProjection.mockResolvedValue({
      found: true,
      policyId: "policy",
      policyRevision: "rev",
      workflow: {
        instanceId: "wf-1",
        domainKind: "backlog-item",
        domainId: "feature/foo",
        state: AgentOpsWorkflowState.OPEN,
        version: 1,
        decisions: [],
        timers: [],
        legalActions: [AgentOpsDomainAction.START_EXECUTION],
      },
      operations: [],
    });

    const projection = await service.getWorkflowProjection({
      kind: "backlog-item",
      id: "feature/foo",
    });

    expect(client.getWorkflowProjection).toHaveBeenCalledWith({
      target: { kind: OperatingModeTargetKind.BACKLOG_ITEM, id: "feature/foo" },
    });
    expect(projection.found).toBe(true);
    expect(projection.state).toBe("open");
    expect(projection.legalActions).toEqual(["start-execution"]);
  });

  it("maps found=false projections without throwing", async () => {
    client.getWorkflowProjection.mockResolvedValue({ found: false });
    const projection = await service.getWorkflowProjection({ kind: "initiative", id: "init-a" });
    expect(client.getWorkflowProjection).toHaveBeenCalledWith({
      target: { kind: OperatingModeTargetKind.INITIATIVE, id: "init-a" },
    });
    expect(projection.found).toBe(false);
    expect(projection.operations).toEqual([]);
  });

  it("lists execution history newest-first as mapped summaries", async () => {
    client.listExecutionHistory.mockResolvedValue({
      executions: [
        {
          executionId: "exec-2",
          operation: "plan-execution",
          operationVersion: "1.0.0",
          mode: "execute-loop",
          modeRevision: "sha256:aa",
          bindingLayer: AgentOpsBindingLayer.SYSTEM_DEFAULT,
          compiledModeDigest: "sha256:bb",
          promptCatalogDigest: "sha256:cc",
          callerInputDigest: "sha256:dd",
          outcome: "success",
          reproducible: true,
          recordedAt: "2026-07-14T10:00:00Z",
        },
      ],
    });

    const history = await service.listExecutionHistory({ kind: "backlog-item", id: "feature/foo" }, 5);

    expect(client.listExecutionHistory).toHaveBeenCalledWith({
      target: { kind: OperatingModeTargetKind.BACKLOG_ITEM, id: "feature/foo" },
      limit: 5,
    });
    expect(history).toHaveLength(1);
    expect(history[0]).toMatchObject({
      executionId: "exec-2",
      bindingLayer: "system-default",
      reproducible: true,
    });
  });

  it("maps resolved bindings including contributions", async () => {
    client.getResolvedBindings.mockResolvedValue({
      operations: [
        {
          operation: "workshop-round",
          operationVersion: "1.0.0",
          resolved: true,
          binding: {
            operation: "workshop-round",
            layer: AgentOpsBindingLayer.INITIATIVE_OVERRIDE,
            owner: { kind: "initiative", id: "init-a" },
            mode: "workshop-loop",
            modeRevision: "sha256:11",
          },
          contributions: [],
        },
      ],
    });

    const bindings = await service.getResolvedBindings({ kind: "backlog-item", id: "feature/foo" });
    expect(bindings[0]).toMatchObject({
      resolved: true,
      binding: { layer: "initiative-override", ownerId: "init-a" },
    });
  });

  it("maps migration status", async () => {
    client.getMigrationStatus.mockResolvedValue({
      state: "quarantined",
      epoch: 2,
      quarantinedCount: 4,
      documentFound: true,
    });
    const status = await service.getMigrationStatus();
    expect(client.getMigrationStatus).toHaveBeenCalledWith({});
    expect(status.state).toBe("quarantined");
    expect(status.quarantinedCount).toBe(4);
  });

  it("puts a binding override with the owner-derived selector", async () => {
    client.putBindingOverride.mockResolvedValue({
      stored: {
        operation: "workshop-round",
        operationVersion: "1.0.0",
        layer: AgentOpsBindingLayer.BACKLOG_ITEM_OVERRIDE,
        owner: { kind: "backlog-item", id: "feature/foo" },
        mode: "workshop-loop",
        modeRevision: "sha256:22",
        disabled: false,
      },
      file: "workshop-round.json",
      revision: "sha256:33",
    });

    const result = await service.putBindingOverride({
      owner: { kind: "backlog-item", id: "feature/foo" },
      operation: "workshop-round",
      operationVersion: "1.0.0",
      mode: "workshop-loop",
      modeRevision: "sha256:22",
    });

    expect(client.putBindingOverride).toHaveBeenCalledWith({
      owner: { kind: OperatingModeTargetKind.BACKLOG_ITEM, id: "feature/foo" },
      operation: "workshop-round",
      operationVersion: "1.0.0",
      mode: "workshop-loop",
      modeRevision: "sha256:22",
      disabled: false,
    });
    expect(result.file).toBe("workshop-round.json");
    expect(result.stored).toMatchObject({ layer: "backlog-item-override" });
  });

  it("deletes a binding override and reports found=false as data, not an error", async () => {
    client.deleteBindingOverride.mockResolvedValue({ found: false });
    const result = await service.deleteBindingOverride(
      { kind: "initiative", id: "init-a" },
      "workshop-round",
    );
    expect(client.deleteBindingOverride).toHaveBeenCalledWith({
      owner: { kind: OperatingModeTargetKind.INITIATIVE, id: "init-a" },
      operation: "workshop-round",
      operationVersion: "",
    });
    expect(result.found).toBe(false);
  });

  it("propagates transport errors unchanged", async () => {
    const failure = new Error("unavailable");
    client.getWorkflowProjection.mockRejectedValue(failure);
    await expect(
      service.getWorkflowProjection({ kind: "backlog-item", id: "feature/foo" }),
    ).rejects.toBe(failure);
  });
});
