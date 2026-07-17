/**
 * Workflow legal-action gating for backlog item CTAs.
 *
 * When the canonical workflow projection EXISTS for an item (found=true), the
 * server's `legal_actions` set is authoritative for which operation-backed
 * CTAs are available — the client makes NO precedence/transition decisions of
 * its own. When no workflow document exists (found=false) the item simply has
 * not run an operation yet — a workflow document is created on the first
 * operation start — and the default client-side funnel in
 * backlog-queue-utils.ts applies UNCHANGED.
 *
 * Actions with no counterpart in the closed domain-action registry (edit,
 * archive, delete, reset-workshop) always keep their client-side logic —
 * they are domain-entity operations, not workflow operations.
 */

import type { ItemActions } from "./backlog-queue-utils";
import type { WorkflowDomainAction, WorkflowProjection } from "../types/agent-operations";

/** The projection fields the gate consumes (subset of WorkflowProjection). */
export type WorkflowActionGate = Pick<WorkflowProjection, "found" | "legalActions">;

const RUN_ACTIONS: readonly WorkflowDomainAction[] = ["queue-plan-execution", "start-execution"];

function hasAny(legal: ReadonlySet<WorkflowDomainAction>, actions: readonly WorkflowDomainAction[]): boolean {
  return actions.some((a) => legal.has(a));
}

/**
 * Gate client-computed ItemActions by the canonical projection's legal
 * actions.
 *
 * Fallback rule: a `null`/`undefined` gate (projection not loaded) or
 * `found === false` (no workflow yet — the item has not run an operation)
 * returns the client actions UNCHANGED. Otherwise, for every operation-backed CTA the server's
 * legal_actions decides availability; `agentRunning` still disables buttons
 * (it is a live-run indicator, not a transition decision).
 */
export function applyWorkflowLegalActions(
  actions: ItemActions,
  gate: WorkflowActionGate | null | undefined,
): ItemActions {
  if (!gate || !gate.found) return actions;

  const legal = new Set<WorkflowDomainAction>(gate.legalActions);
  const busy = actions.agentRunning;

  const runLegal = hasAny(legal, RUN_ACTIONS);
  const workshopLegal = legal.has("commit-workshop-round");
  const finalizeLegal = legal.has("bind-plan");
  const followUpLegal = legal.has("create-followup");

  const gated: ItemActions = {
    ...actions,
    canRun: runLegal && !busy,
    runDisabled: runLegal && busy,
    canWorkshop: workshopLegal && !busy,
    workshopDisabled: workshopLegal && busy,
    canFinalize: finalizeLegal && !busy,
    finalizeDisabled: finalizeLegal && busy,
    canFollowUp: followUpLegal,
    // Retry re-dispatches the prior attempt — same workflow operation as run.
    canRetry: actions.canRetry && runLegal,
    // The stepper needs client-side pending decisions to render anything;
    // the server gate only vetoes it when saving decisions is not legal.
    showDecisionStepper: actions.showDecisionStepper && legal.has("save-decisions"),
  };

  // Keep the client-picked primary CTA when it survived gating; otherwise
  // fall back to the first available operation-backed CTA. This is
  // presentation-order only — availability itself came from the server.
  gated.primaryCta = resolvePrimaryCta(gated, actions.primaryCta);

  return gated;
}

function ctaAvailable(actions: ItemActions, cta: ItemActions["primaryCta"]): boolean {
  switch (cta) {
    case "run":
      return actions.canRun || actions.runDisabled;
    case "workshop":
      return actions.canWorkshop || actions.workshopDisabled;
    case "finalize":
      return actions.canFinalize || actions.finalizeDisabled;
    case "followUp":
      return actions.canFollowUp;
    case "archive":
      return actions.canArchive;
    case "answer":
      return actions.showDecisionStepper;
    default:
      return false;
  }
}

const CTA_FALLBACK_ORDER: ItemActions["primaryCta"][] = [
  "finalize",
  "workshop",
  "run",
  "followUp",
  "archive",
];

function resolvePrimaryCta(
  gated: ItemActions,
  clientPrimary: ItemActions["primaryCta"],
): ItemActions["primaryCta"] {
  if (clientPrimary && ctaAvailable(gated, clientPrimary)) return clientPrimary;
  for (const cta of CTA_FALLBACK_ORDER) {
    if (ctaAvailable(gated, cta)) return cta;
  }
  return null;
}
