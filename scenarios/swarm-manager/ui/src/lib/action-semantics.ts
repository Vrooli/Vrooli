/**
 * Action semantics — what happens when the operator presses this?
 *
 * The decision stream puts very different operations behind visually identical
 * buttons. "Accept suggestion" flips a status field; "Start review" dispatches
 * an autonomous agent that will run for minutes and cost tokens. Both rendered
 * as a cyan button with a label, so the only way to learn which was which was
 * to press one.
 *
 * The classification is NOT invented here; this module only maps server
 * declarations onto operator-facing meaning. There are two, and they are
 * complementary:
 *
 *   - The next-action feed's `effect`, from `internal/nextaction`. This is the
 *     primary signal and covers every action, including `run`, `retry` and
 *     `review`, whose transitions take an execution id or a review round —
 *     subjects the client never holds, which is why their `transition_key` is
 *     empty by design.
 *   - A transition's `kind` from the registry, for controls wired straight to
 *     a transition (Plan goal, Discover, Classify) rather than driven by a
 *     feed entry.
 *
 * Neither is a list maintained here, and that is the point: adding a workflow
 * or a next-action gives its button the right affordance with no UI change.
 */

import { TransitionKind } from "@vrooli/proto-types/swarm-manager/v1/domain/transition_pb";
import type { NextActionEffect } from "../services/next-action-service";

/**
 * What pressing the control actually does, ordered roughly by how much the
 * operator should think first.
 */
export type ConsequenceClass =
  /** Dispatches an autonomous agent run. Costs tokens, takes minutes. */
  | "agent_workflow"
  /** Opens an interactive agent session the operator will converse with. */
  | "agent_session"
  /** Removes or interrupts state. Confirm first. */
  | "destructive"
  /** Server-side state change, immediate and cheap. */
  | "state_change"
  /** Only moves the operator somewhere else. No side effect. */
  | "navigation";

export interface ConsequenceMeta {
  /** Short phrase for a tooltip or a hint line. Sentence case, no period. */
  hint: string;
  /** True when the operator is about to spend agent time. */
  spawnsAgent: boolean;
  /** True when a confirmation step is warranted. */
  confirms: boolean;
}

export const CONSEQUENCE_META: Record<ConsequenceClass, ConsequenceMeta> = {
  agent_workflow: {
    hint: "Dispatches an agent run — this takes minutes and consumes tokens",
    spawnsAgent: true,
    confirms: false,
  },
  agent_session: {
    hint: "Opens an interactive agent session",
    spawnsAgent: true,
    confirms: false,
  },
  destructive: {
    hint: "Removes state — you'll be asked to confirm",
    spawnsAgent: false,
    confirms: true,
  },
  state_change: {
    hint: "Applies immediately",
    spawnsAgent: false,
    confirms: false,
  },
  navigation: {
    hint: "Opens the full view",
    spawnsAgent: false,
    confirms: false,
  },
};

/** Maps the server's declared transition kind onto operator meaning. */
export function consequenceOfTransitionKind(kind: TransitionKind | undefined): ConsequenceClass | undefined {
  switch (kind) {
    case TransitionKind.WORKFLOW:
      return "agent_workflow";
    case TransitionKind.SESSION:
      return "agent_session";
    case TransitionKind.DETERMINISTIC:
      return "state_change";
    default:
      return undefined;
  }
}

/** Maps the next-action feed's declared effect onto operator meaning. */
export function consequenceOfEffect(effect: NextActionEffect | undefined): ConsequenceClass | undefined {
  switch (effect) {
    case "agent_run":
      return "agent_workflow";
    case "agent_session":
      return "agent_session";
    case "state_change":
      return "state_change";
    case "none":
      return "navigation";
    default:
      return undefined;
  }
}

export interface ConsequenceInput {
  /** The next-action id. Used only for logging context, never to classify. */
  actionId?: string;
  /**
   * The effect the next-action feed declared for this action. This is the
   * primary signal — the server knows which actions dispatch agent work,
   * including the ones whose transition subject the client never holds.
   */
  effect?: NextActionEffect;
  /**
   * The transition the action starts, when the caller has resolved one from
   * the registry. Used for controls that are wired to a transition directly
   * rather than driven by a feed entry.
   */
  transitionKind?: TransitionKind;
  /** The server's destructive flag, or a caller that already knows. */
  destructive?: boolean;
}

/**
 * Classifies an action.
 *
 * Precedence: the destructive flag wins, because it is a safety property and
 * orthogonal to cost. Then the feed's declared effect, then a transition kind
 * resolved from the registry.
 *
 * There is deliberately no fallback keyed on the action id. An earlier version
 * kept one, and it was both redundant (four of its seven entries already had
 * transitions) and wrong (`dispatch_followup` is a *deterministic* transition,
 * so marking it an agent run mislabelled the button whenever the catalog had
 * not loaded). Classification now has exactly one source: the server.
 *
 * An unclassifiable action falls back to `state_change`, never `navigation` —
 * under-promising a side effect is safer than implying there is none.
 */
export function consequenceOf({ effect, transitionKind, destructive }: ConsequenceInput): ConsequenceClass {
  if (destructive) return "destructive";
  return consequenceOfEffect(effect)
    ?? consequenceOfTransitionKind(transitionKind)
    ?? "state_change";
}

/** Convenience: does pressing this put work on an agent? */
export function spawnsAgent(input: ConsequenceInput): boolean {
  return CONSEQUENCE_META[consequenceOf(input)].spawnsAgent;
}

/**
 * The toast severity an action's success should use.
 *
 * Agent work only *starts* on success — the run is queued, not finished — so
 * claiming "done" would be a lie. This is the single place that decision is
 * made, replacing a per-call-site `successKind` judgement.
 */
export function successKindFor(input: ConsequenceInput): "success" | "progress" {
  return spawnsAgent(input) ? "progress" : "success";
}
