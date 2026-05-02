/**
 * Static configs for the operating-mode concept explainers.
 *
 * Each entry feeds <ConceptExplainerDialog> through a thin component or
 * inline opener on the details page and the picker. The labels mirror the
 * registry vocabulary (scope kinds, run strategies, capability flags) so
 * adding a new value server-side surfaces here too.
 */

import type { ConceptExplainerSection } from "../../ui/concept-explainer-dialog";

export interface ConceptExplainer {
  title: string;
  intro?: string;
  sections: ConceptExplainerSection[];
}

export const SCOPE_KIND_EXPLAINER: ConceptExplainer = {
  title: "Scope",
  intro:
    "Scope names the unit a mode operates on per agent run. It determines what the agent reads, writes, and is reviewed against.",
  sections: [
    {
      label: "Backlog item",
      body: "One backlog item per agent run. Each item is workshopped, executed, and reviewed in isolation; the workshop loop is the refinement primitive. Best when items are right-sized and stable.",
    },
    {
      label: "Initiative",
      body: "The whole initiative is one unit of work. The agent reads cross-item state, writes initiative-level artifacts, and is reviewed against the initiative's acceptance criteria. Best when items are coupled or the right item shape is only knowable after investigation.",
    },
  ],
};

export const RUN_STRATEGY_EXPLAINER: ConceptExplainer = {
  title: "Run strategy",
  intro:
    "Run strategy names how successive agent runs relate to each other inside one mode. It controls whether runs continue from prior handoffs, whether the operator gates phases, and whether multiple runs can be in flight.",
  sections: [
    {
      label: "Existing item flow",
      body: "Each backlog item runs through the standard generator/improver/review pipeline. The mode does not introduce new orchestration on top of the existing item flow.",
    },
    {
      label: "Single phase run",
      body: "Each phase is one agent run. There is no gated loop and no sequential handoff between runs.",
    },
    {
      label: "Sequential handoff",
      body: "Runs are sequenced. Each run completes the earliest contiguous slice it can fully finish and emits a handoff that the next run reads. Best when continuity across handoffs matters more than parallelism.",
    },
    {
      label: "Operator-gated loop",
      body: "Phases run as an investigate → plan → execute → review loop, with the operator gating each phase start. Execute can either advance to review or loop back to investigate when the run reports replan_needed.",
    },
  ],
};

export const DEFAULT_FLAG_EXPLAINER: ConceptExplainer = {
  title: "Default mode",
  intro:
    "Exactly one mode is the default. New initiatives start in the default mode; the operator can switch later through the operating-mode switch endpoint.",
  sections: [
    {
      label: "yes",
      body: "This mode is the registered default. New initiatives are created in this mode. Today, item-level is the default — its assumption that backlog items are the right unit of execution is the safest starting point.",
    },
    {
      label: "no",
      body: "This mode is registered but not the default. Initiatives reach it through an explicit operator-driven switch.",
    },
  ],
};

export const CAPABILITY_EXPLAINER: ConceptExplainer = {
  title: "Capabilities",
  intro:
    "Each mode declares a set of capability flags. The UI gates panels and controls on these flags rather than checking mode names — adding a new mode that sets the right capabilities surfaces the right controls automatically.",
  sections: [
    {
      label: "Phase graph",
      body: "The mode runs as a graph of named phases (investigate, plan, execute, review). Without this flag, the mode has no phase composer or graph view.",
    },
    {
      label: "Phase start controls",
      body: "The operator can start phases through the panel. Disabled when phases are supported but not operator-startable (some modes auto-advance).",
    },
    {
      label: "Mark items complete from rounds",
      body: "Round-level completion can promote member backlog items to completed via the run-id-validated reconciliation endpoint.",
    },
    {
      label: "Apply backlog sync proposals",
      body: "The mode can emit and apply structured proposals that create, update, or follow-up backlog items as part of the round result.",
    },
    {
      label: "Requires acceptance criteria",
      body: "The mode's review phase needs the initiative to have at least one acceptance criterion before review can start.",
    },
    {
      label: "Phase artifacts",
      body: "Phases write durable artifacts under modes/<mode>/. The artifact list panel renders these between rounds.",
    },
    {
      label: "Round handoffs",
      body: "Each round produces a handoff summary that the next run reads. Sequential modes use this to maintain continuity.",
    },
    {
      label: "Existing item execution flow",
      body: "The mode bridges to the standard backlog item execution pipeline — used by item-level. The panel surfaces a useful empty state instead of operating-mode controls.",
    },
  ],
};
