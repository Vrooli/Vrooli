import { describe, expect, it } from "vitest";
import { livePhaseView } from "./phase-view";
import type {
  OperatingModePhaseTransition,
  OperatingModeRound,
  OperatingModeWorkspace,
} from "../../../types/operating-mode";

// These oracles pin the generic field-predicate guard evaluation that
// livePhaseView uses to pick which declared outgoing edge a completed live
// round fired. They guard against the Phase-4A regression where the UI read
// the deleted payload_key/progress_decision fields instead of the generic
// {conditionKind (op), field, value} shape now on the wire.

function round(overrides: Partial<OperatingModeRound> = {}): OperatingModeRound {
  return {
    round: 2,
    mode: "holistic-loop",
    scopeKind: "initiative",
    scopeId: "demo",
    phase: "execute",
    runStrategy: "sequential_handoff",
    agentProfileKey: "swarm-manager/execution",
    generatedAt: "2026-07-08T00:00:00Z",
    status: "completed",
    items: [],
    artifactUpdates: [],
    handoffs: [],
    payload: {},
    ...overrides,
  };
}

function workspace(rounds: OperatingModeRound[]): OperatingModeWorkspace {
  return {
    initiativeName: "demo",
    mode: "holistic-loop",
    definition: {
      mode: "holistic-loop",
      label: "Holistic Loop",
      scopeKind: "initiative",
      capabilities: {
        supportsPhases: true,
        canStartPhases: true,
        canCompleteItems: false,
        canApplyBacklogSyncProposals: false,
        requiresAcceptanceCriteria: false,
        supportsArtifacts: true,
        supportsHandoffs: true,
        usesItemExecutionFlow: false,
      },
      phases: [
        {
          phase: "execute",
          phaseKind: "execute",
          activityPurpose: "",
          profileKey: "swarm-manager/execution",
          writesRepo: true,
          startable: true,
        },
      ],
      terminal: ["review"],
      transitions: {},
      runStrategy: "sequential_handoff",
    },
    artifacts: [],
    rounds,
  };
}

const TRANSITIONS: OperatingModePhaseTransition[] = [
  { from: "execute", to: "investigate", conditionKind: "eq", label: "on replan_needed = true", field: "replan_needed", value: "true" },
  { from: "execute", to: "review", conditionKind: "always", label: "always" },
];

describe("livePhaseView generic guard selection", () => {
  it("fires the eq guard when the payload field matches the rendered value", () => {
    const r = round({ payload: { replan_needed: true } });
    const view = livePhaseView(r, workspace([r]), TRANSITIONS, "demo");
    expect(view.firedTransition?.to).toBe("investigate");
    expect(view.firedTransition?.conditionKind).toBe("eq");
  });

  it("falls through to the always edge when the eq guard does not match", () => {
    const r = round({ payload: { replan_needed: false } });
    const view = livePhaseView(r, workspace([r]), TRANSITIONS, "demo");
    expect(view.firedTransition?.to).toBe("review");
    expect(view.firedTransition?.conditionKind).toBe("always");
  });

  it("resolves a dotted field path for the guard", () => {
    const transitions: OperatingModePhaseTransition[] = [
      { from: "execute", to: "review", conditionKind: "eq", label: "on progress.decision = complete", field: "progress.decision", value: "complete" },
    ];
    const r = round({ payload: { progress: { decision: "complete" } } });
    const view = livePhaseView(r, workspace([r]), transitions, "demo");
    expect(view.firedTransition?.to).toBe("review");
  });

  it("treats a guarded edge with no target as a guarded stop (no fired transition)", () => {
    const transitions: OperatingModePhaseTransition[] = [
      { from: "execute", to: "", conditionKind: "exists", label: "when blocked is set", field: "blocked" },
    ];
    const r = round({ payload: { blocked: "needs operator" } });
    const view = livePhaseView(r, workspace([r]), transitions, "demo");
    // The guard matches but has no downstream target, so there is a fired
    // transition record whose `to` is undefined (a guarded stop), not a route.
    expect(view.firedTransition?.to).toBeUndefined();
    expect(view.firedTransition?.field).toBe("blocked");
  });

  it("selects no transition when no guard matches and there is no always edge", () => {
    const transitions: OperatingModePhaseTransition[] = [
      { from: "execute", to: "investigate", conditionKind: "eq", label: "on replan_needed = true", field: "replan_needed", value: "true" },
    ];
    const r = round({ payload: { replan_needed: false } });
    const view = livePhaseView(r, workspace([r]), transitions, "demo");
    expect(view.firedTransition).toBeUndefined();
    expect(view.terminal).toBe(true);
  });
});
