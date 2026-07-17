import { describe, it, expect } from "vitest";
import { getItemActions, type ActionContext, type ItemActions } from "./backlog-queue-utils";
import { applyWorkflowLegalActions, type WorkflowActionGate } from "./workflow-legal-actions";
import type { WorkflowDomainAction } from "../types/agent-operations";

function clientActions(overrides: Partial<ActionContext> = {}): ItemActions {
  const ctx: ActionContext = {
    item: { kind: "execute", name: "foo", status: "ready", dependsOn: [] },
    blockingInfo: null,
    readinessReady: true,
    pendingSynthesis: false,
    agentRunning: false,
    hasPendingDecisions: false,
    hasExecutionHistory: false,
    ...overrides,
  };
  return getItemActions(ctx);
}

function gate(legalActions: WorkflowDomainAction[], found = true): WorkflowActionGate {
  return { found, legalActions };
}

describe("applyWorkflowLegalActions — fallback rule", () => {
  it("returns client actions UNCHANGED when the projection is not loaded (null gate)", () => {
    const actions = clientActions();
    expect(applyWorkflowLegalActions(actions, null)).toBe(actions);
    expect(applyWorkflowLegalActions(actions, undefined)).toBe(actions);
  });

  it("returns client actions UNCHANGED when no workflow document exists (found=false)", () => {
    const actions = clientActions();
    const result = applyWorkflowLegalActions(actions, gate(["commit-workshop-round"], false));
    expect(result).toBe(actions);
    // Legacy funnel intact: ready item keeps run as primary.
    expect(result.canRun).toBe(true);
    expect(result.primaryCta).toBe("run");
  });
});

describe("applyWorkflowLegalActions — server authoritative when workflow exists", () => {
  it("enables only the CTAs whose domain actions are legal", () => {
    // Client funnel says "run" (ready item), but the server only allows a
    // workshop round — the projection wins.
    const result = applyWorkflowLegalActions(
      clientActions(),
      gate(["commit-workshop-round", "save-decisions"]),
    );
    expect(result.canRun).toBe(false);
    expect(result.runDisabled).toBe(false);
    expect(result.canWorkshop).toBe(true);
    expect(result.canFinalize).toBe(false);
    expect(result.canFollowUp).toBe(false);
    expect(result.primaryCta).toBe("workshop");
  });

  it("enables run when queue-plan-execution or start-execution is legal", () => {
    for (const action of ["queue-plan-execution", "start-execution"] as const) {
      const result = applyWorkflowLegalActions(clientActions(), gate([action]));
      expect(result.canRun).toBe(true);
      expect(result.primaryCta).toBe("run");
    }
  });

  it("enables run even when the client funnel would have hidden it", () => {
    // Client: readiness not met → workshop primary, no run. Server: execution
    // is legal → run is available (server projection is authoritative).
    const result = applyWorkflowLegalActions(
      clientActions({ readinessReady: false }),
      gate(["start-execution"]),
    );
    expect(result.canRun).toBe(true);
    expect(result.canWorkshop).toBe(false);
  });

  it("keeps agentRunning as a disable (live-run indicator), not a transition decision", () => {
    const result = applyWorkflowLegalActions(
      clientActions({ agentRunning: true }),
      gate(["start-execution", "commit-workshop-round"]),
    );
    expect(result.canRun).toBe(false);
    expect(result.runDisabled).toBe(true);
    expect(result.canWorkshop).toBe(false);
    expect(result.workshopDisabled).toBe(true);
  });

  it("maps finalize to bind-plan and follow-up to create-followup", () => {
    const result = applyWorkflowLegalActions(
      clientActions(),
      gate(["bind-plan", "create-followup"]),
    );
    expect(result.canFinalize).toBe(true);
    expect(result.canFollowUp).toBe(true);
    expect(result.canRun).toBe(false);
    expect(result.primaryCta).toBe("finalize");
  });

  it("gates retry on execution legality while keeping the client history requirement", () => {
    const withHistory = clientActions({
      item: { kind: "execute", name: "foo", status: "completed", dependsOn: [] },
      hasExecutionHistory: true,
      hasTerminalExecution: true,
    });
    expect(withHistory.canRetry).toBe(true);
    expect(applyWorkflowLegalActions(withHistory, gate(["start-execution"])).canRetry).toBe(true);
    expect(applyWorkflowLegalActions(withHistory, gate(["commit-review-round"])).canRetry).toBe(false);
  });

  it("vetoes the decision stepper when save-decisions is not legal", () => {
    const withDecisions = clientActions({ hasPendingDecisions: true });
    expect(withDecisions.showDecisionStepper).toBe(true);
    expect(
      applyWorkflowLegalActions(withDecisions, gate(["save-decisions"])).showDecisionStepper,
    ).toBe(true);
    expect(
      applyWorkflowLegalActions(withDecisions, gate(["commit-review-round"])).showDecisionStepper,
    ).toBe(false);
  });

  it("leaves non-workflow actions (archive) to client logic", () => {
    const terminal = clientActions({
      item: { kind: "execute", name: "foo", status: "completed", dependsOn: [] },
      hasExecutionHistory: true,
      hasTerminalExecution: true,
    });
    expect(terminal.canArchive).toBe(true);
    const result = applyWorkflowLegalActions(terminal, gate([]));
    expect(result.canArchive).toBe(true);
  });

  it("clears the primary CTA when nothing operation-backed is legal", () => {
    const result = applyWorkflowLegalActions(clientActions(), gate([]));
    expect(result.canRun).toBe(false);
    expect(result.canWorkshop).toBe(false);
    expect(result.canFinalize).toBe(false);
    expect(result.primaryCta).toBe(null);
  });
});
