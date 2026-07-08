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

export const FLOW_GUIDE_EXPLAINER: ConceptExplainer = {
  title: "Reading the Flow tab",
  intro:
    "Flow shows how a mode moves through its phases through one shared phase viewer with four tabs — Instructions, Reads, Emits, Transition. A data-source control swaps the fill without changing the view: the Contract template, a Simulation preset, or a Live round.",
  sections: [
    {
      heading: "Data sources",
      label: "Contract",
      body: "The phase's static contract. Instructions shows the agent skill template with its {{VARIABLE}} slots still unfilled, and Reads/Emits/Transition show what the phase is defined to consume, produce, and route to — no initiative data substituted.",
    },
    {
      heading: "Data sources",
      label: "Simulation preset",
      body: "A deterministic, in-memory walk of the phase graph against illustrative data. Presets seed different phase outputs to exercise real branches (replan, continue, blocked, review, reconcile) but never run agents, acquire locks, or touch real initiative state. Instructions shows the literal prompt with the preset's data substituted.",
    },
    {
      heading: "Data sources",
      label: "Live round",
      body: "The actual rounds recorded for a linked initiative. Pick an initiative to render its most recent active or completed round. This is real data — the prompt, reads, emits, and transition come from what agents actually produced.",
    },
    {
      heading: "Tabs",
      label: "Instructions",
      body: "The literal agent prompt for the selected source, plus the agent profile that would run it. Contract shows the unfilled template; Simulation and Live show the prompt with data substituted — byte-identical to what an agent actually receives.",
    },
    {
      heading: "Vocabulary",
      label: "Round",
      body: "One agent run of a single phase. Rounds accumulate as the mode advances; each round records the phase it ran, the items in scope, and the structured result it emitted.",
    },
    {
      heading: "Vocabulary",
      label: "Phase",
      body: "A named step in the mode's graph (investigate, plan, execute, review, reconcile, …). Each phase has a fixed contract for what it reads and must emit.",
    },
    {
      heading: "Vocabulary",
      label: "Reads",
      body: "The context a phase consumes, one card per prompt variable: member items (MEMBER_ITEMS_JSON), durable mode artifacts (MODE_ARTIFACTS_JSON), prior rounds/handoffs (PRIOR_ROUNDS_JSON), and acceptance criteria (ACCEPTANCE_CRITERIA).",
    },
    {
      heading: "Vocabulary",
      label: "Emits",
      body: "The structured result a phase produces: artifacts, a handoff, a progress decision, an acceptance verdict, a replan signal, a readiness report, or a backlog-sync proposal.",
    },
    {
      heading: "Vocabulary",
      label: "Transition",
      body: "Why the next phase was selected. Transitions are decided by backend guards — an unconditional 'always', a boolean payload signal (e.g. replan_needed), or a progress decision (continue / replan / complete / blocked). A blocked decision has no downstream target and ends the cycle.",
    },
    {
      heading: "Vocabulary",
      label: "Artifact",
      body: "A durable file a phase writes under modes/<mode>/. Later phases read these to maintain continuity across rounds.",
    },
    {
      heading: "Vocabulary",
      label: "Handoff",
      body: "A summary one round leaves for the next: what was completed, changed files, tests, blockers, and the intended next step. Sequential modes rely on handoffs for continuity.",
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
