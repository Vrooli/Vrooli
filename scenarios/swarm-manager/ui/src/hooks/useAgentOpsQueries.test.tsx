import { describe, it, expect, vi, beforeEach } from "vitest";
import { waitFor } from "@testing-library/react";
import { renderHookWithProviders } from "../test-utils";
import {
  backlogItemTarget,
  initiativeTarget,
  useWorkflowProjectionQuery,
} from "./useAgentOpsQueries";
import { agentOperationsService } from "../services";
import { applyWorkflowLegalActions, getItemActions } from "../lib";
import type { ActionContext } from "../lib";
import type { WorkflowProjection } from "../types/agent-operations";

vi.mock("../services", () => ({
  agentOperationsService: {
    getWorkflowProjection: vi.fn(),
    listExecutionHistory: vi.fn(),
    getResolvedBindings: vi.fn(),
    getMigrationStatus: vi.fn(),
  },
}));

const mockedGetProjection = vi.mocked(agentOperationsService.getWorkflowProjection);

function projection(overrides: Partial<WorkflowProjection> = {}): WorkflowProjection {
  return {
    found: true,
    instanceId: "wf-1",
    domainKind: "backlog-item",
    domainId: "execute/foo",
    state: "open",
    version: 1,
    operations: [],
    decisions: [],
    timers: [],
    legalActions: [],
    policyId: "",
    policyRevision: "",
    ...overrides,
  };
}

const readyItemContext: ActionContext = {
  item: { kind: "execute", name: "foo", status: "ready", dependsOn: [] },
  blockingInfo: null,
  readinessReady: true,
  pendingSynthesis: false,
  agentRunning: false,
  hasPendingDecisions: false,
  hasExecutionHistory: false,
};

describe("target selectors", () => {
  it("builds a backlog-item target from kind/name and null when incomplete", () => {
    expect(backlogItemTarget("execute", "foo")).toEqual({
      kind: "backlog-item",
      id: "execute/foo",
    });
    expect(backlogItemTarget(null, "foo")).toBeNull();
    expect(backlogItemTarget("execute", undefined)).toBeNull();
    expect(initiativeTarget("init-a")).toEqual({ kind: "initiative", id: "init-a" });
    expect(initiativeTarget(undefined)).toBeNull();
  });
});

describe("useWorkflowProjectionQuery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches the projection for the target", async () => {
    mockedGetProjection.mockResolvedValue(projection({ legalActions: ["start-execution"] }));

    const { result } = renderHookWithProviders(() =>
      useWorkflowProjectionQuery({ kind: "backlog-item", id: "execute/foo" }),
    );

    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(mockedGetProjection).toHaveBeenCalledWith({ kind: "backlog-item", id: "execute/foo" });
    expect(result.current.data?.legalActions).toEqual(["start-execution"]);
  });

  it("does not fetch without a target", () => {
    renderHookWithProviders(() => useWorkflowProjectionQuery(null));
    expect(mockedGetProjection).not.toHaveBeenCalled();
  });

  it("workflow FOUND → the projection's legal actions win over the client funnel", async () => {
    // Server: only a workshop round is legal, even though the client funnel
    // would offer Run for a ready item.
    mockedGetProjection.mockResolvedValue(
      projection({ legalActions: ["commit-workshop-round"] }),
    );

    const { result } = renderHookWithProviders(() =>
      useWorkflowProjectionQuery({ kind: "backlog-item", id: "execute/foo" }),
    );
    await waitFor(() => expect(result.current.data).toBeDefined());

    const gated = applyWorkflowLegalActions(getItemActions(readyItemContext), result.current.data);
    expect(gated.canRun).toBe(false);
    expect(gated.canWorkshop).toBe(true);
    expect(gated.primaryCta).toBe("workshop");
  });

  it("query ERROR → surfaces the error and the null-gate rule keeps client actions unchanged", async () => {
    mockedGetProjection.mockRejectedValue(new Error("agent-operations unavailable"));

    const { result } = renderHookWithProviders(() =>
      useWorkflowProjectionQuery({ kind: "backlog-item", id: "execute/foo" }),
    );
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
    expect(result.current.data).toBeUndefined();

    // Documented fallback: no gate (undefined data) → client funnel unchanged.
    const clientActions = getItemActions(readyItemContext);
    const gated = applyWorkflowLegalActions(clientActions, result.current.data);
    expect(gated).toBe(clientActions);
    expect(gated.canRun).toBe(true);
  });

  it("workflow NOT found → the legacy client funnel wins unchanged", async () => {
    mockedGetProjection.mockResolvedValue(projection({ found: false, legalActions: [] }));

    const { result } = renderHookWithProviders(() =>
      useWorkflowProjectionQuery({ kind: "backlog-item", id: "execute/foo" }),
    );
    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data?.found).toBe(false);

    const clientActions = getItemActions(readyItemContext);
    const gated = applyWorkflowLegalActions(clientActions, result.current.data);
    expect(gated).toBe(clientActions);
    expect(gated.canRun).toBe(true);
    expect(gated.primaryCta).toBe("run");
  });
});
