import { describe, expect, it } from "vitest";
import { getItemActions, type ActionContext } from "./backlog-queue-utils";

const context = (overrides: Partial<ActionContext> = {}): ActionContext => ({
  item: { kind: "idea", name: "test-item", status: "ready", dependsOn: [] },
  blockingInfo: null,
  agentRunning: false,
  hasPendingDecisions: false,
  hasExecutionHistory: false,
  ...overrides,
});

describe("getItemActions", () => {
  it("offers Run for a queueable item without consulting a client readiness score", () => {
    const result = getItemActions(context());

    expect(result.primaryCta).toBe("run");
    expect(result.canRun).toBe(true);
  });

  it("disables Run while an agent is active", () => {
    const result = getItemActions(context({ agentRunning: true }));

    expect(result.primaryCta).toBe("run");
    expect(result.runDisabled).toBe(true);
    expect(result.disabledReason).toMatch(/already running/i);
  });

  it("holds execution behind unresolved Plan Workshop decisions", () => {
    const result = getItemActions(context({ hasPendingDecisions: true }));

    expect(result.showDecisionStepper).toBe(true);
    expect(result.primaryCta).toBeNull();
    expect(result.canRun).toBe(false);
  });

  it("shows follow-up and archive controls for completed work", () => {
    const result = getItemActions(context({
      item: { kind: "idea", name: "test-item", status: "completed", dependsOn: [] },
      hasExecutionHistory: true,
    }));

    expect(result.primaryCta).toBe("followUp");
    expect(result.canFollowUp).toBe(true);
    expect(result.canArchive).toBe(true);
  });

  it("keeps lifecycle-blocked items informational instead of launching an override", () => {
    const result = getItemActions(context({
      blockingInfo: { blocked: true, blockingDepKeys: ["idea/dependency"], allForceable: false },
    }));

    expect(result.blocked).toBe(true);
    expect(result.canRun).toBe(true);
  });
});
