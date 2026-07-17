/**
 * Decision-flow config
 *
 * Static, hand-authored question tree that guides operators to a recommended
 * operating mode. Validated against the catalog at render time — modes
 * referenced here that aren't registered surface a visible error chip rather
 * than silently failing. Adding a new mode requires extending this config
 * with at least one terminal-question path that selects it (enforced by the
 * authoring skill).
 *
 * Each question has a yes / no path. A `NodeRef` either points at another
 * question (`{ kind: "question", id }`) or terminates with a recommended
 * mode (`{ kind: "mode", mode }`).
 */

import type { InitiativeOperatingMode } from "../../../types";
import { MEMBER_ITEM_STRATEGY_WIRE_VALUE } from "../../../lib/member-item-strategy";

export type DecisionFlowNodeRef =
  | { kind: "question"; id: string }
  | { kind: "mode"; mode: InitiativeOperatingMode };

export interface DecisionFlowQuestion {
  id: string;
  question: string;
  /** Optional sub-text rendered under the question (clarifying examples). */
  hint?: string;
  yes: DecisionFlowNodeRef;
  no: DecisionFlowNodeRef;
}

export const DECISION_FLOW_ROOT_ID = "items-coupled";

export const DECISION_FLOW: DecisionFlowQuestion[] = [
  {
    id: "items-coupled",
    question: "Are the items coupled — does completing one item invalidate or break others?",
    hint: "If multiple items touch the same substrate (auth, data model, shared infrastructure), they are coupled.",
    yes: { kind: "question", id: "plan-stable" },
    no: { kind: "question", id: "items-stable" },
  },
  {
    id: "items-stable",
    question: "Are the items stable — will their scope and shape stay constant during execution?",
    hint: "If items are stable and decoupled, they don't need a methodology loop — each item runs its own workflow and the initiative only provides strategy configuration (the member-item workflow).",
    // Terminal ref keeps the legacy wire value so accepting the
    // recommendation flows through the EXISTING switch-mode mutation;
    // presentation maps to "Member-item workflow" (lib/member-item-strategy).
    yes: { kind: "mode", mode: MEMBER_ITEM_STRATEGY_WIRE_VALUE },
    no: { kind: "question", id: "plan-stable" },
  },
  {
    id: "plan-stable",
    question: "Can a single multi-phase plan be prepared up front and remain stable as agents drain it?",
    hint: "If the plan needs to revise itself between rounds based on what execution reveals, it is not stable enough to drain.",
    yes: { kind: "mode", mode: "phased-plan-drain" },
    no: { kind: "mode", mode: "holistic-loop" },
  },
];

export function findQuestion(id: string): DecisionFlowQuestion | undefined {
  return DECISION_FLOW.find((q) => q.id === id);
}
