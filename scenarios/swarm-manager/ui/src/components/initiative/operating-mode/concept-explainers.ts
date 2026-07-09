/**
 * Static configs for the operating-mode concept explainers.
 *
 * Each entry feeds <ConceptExplainerDialog> through a thin component or
 * inline opener on the details page and the picker. The labels mirror the
 * registry vocabulary (target kinds, run strategies, capability flags) so
 * adding a new value server-side surfaces here too.
 */

import type { ConceptExplainerSection } from "../../ui/concept-explainer-dialog";

export interface ConceptExplainer {
  title: string;
  intro?: string;
  docLink?: { href: string; label: string };
  sections: ConceptExplainerSection[];
}

export const TARGET_KIND_EXPLAINER: ConceptExplainer = {
  title: "Target",
  intro:
    "A mode's target is its unit of work — the thing one run of the loop operates on. Each target kind has an adapter that supplies target-specific reads, artifact roots, and locking identity. The engine's run substrate is generic; initiative is one adapter among several, not the base case.",
  docLink: {
    href: "/docs/swarm-manager/concepts/EXECUTION-MODES.md",
    label: "Read the canonical operating modes doc",
  },
  sections: [
    {
      label: "Plan-manager plan",
      body: "A canonical plan-manager plan (execution id / slug). The plan already says what to do; the loop drains it slice by slice with accumulated handoffs — no initiative created, no backlog ceremony.",
    },
    {
      label: "Plan reference",
      body: "A plan file or reference not imported into swarm-manager (a repo-relative plan path). The loop reads the plan content directly and drains it, with a ref-derived artifact root and lock identity.",
    },
    {
      label: "Initiative",
      body: "A swarm-manager initiative and its member items. The initiative adapter layers on member-item context, acceptance criteria, and backlog reconciliation. Best when items are coupled or the right shape is only knowable after investigation.",
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
      body: "A phase's declared, composed input contract, rendered from the mode data — never a fixed category list. Reads are grouped by supplying provider: the generic base provider (mode/phase identity, round number, operator note, prior handoffs, artifacts) and the mode's target adapter (initiative reads like member items and acceptance criteria, or plan reads like the plan path and plan context). A plan-target phase and an initiative-target phase show different, scope-appropriate groups — the symmetric twin of Emits.",
    },
    {
      heading: "Vocabulary",
      label: "Emits",
      body: "The structured result a phase is contracted to produce (declared_output): a handoff, a progress decision, an acceptance verdict, a readiness report, artifacts, or a backlog-sync proposal. Reads and Emits are symmetric declared contracts.",
    },
    {
      heading: "Vocabulary",
      label: "Transition",
      body: "Why the next phase was selected. Transitions are decided by backend guards — a generic field-predicate over the phase's structured output. When the routing field a guard needs is not emitted directly, the transition declares a classification-on-transition contract that derives it from the handoff via the resolution ladder — a built-in step, not an agent phase — abstaining to needs_attention rather than fabricating a route. A matched guard with no downstream target ends the cycle for operator intervention.",
    },
    {
      heading: "Vocabulary",
      label: "Composition",
      body: "A phase can delegate to a sub-mode (executed_by): the engine runs the sub-mode's loop as that phase, one level deep. The composed sub-mode's phases render inline, marked delegated; the backend engine remains the single source of truth for routing.",
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

/**
 * The top-level "what is an operating mode" explainer. Reachable from the
 * operating-modes list surface without drilling into any one mode, so an
 * operator who has never seen operating modes can understand the concept —
 * including the resolution ladder — before opening a mode's detail page.
 */
export const OPERATING_MODE_INTRO_EXPLAINER: ConceptExplainer = {
  title: "What is an operating mode?",
  intro:
    "An operating mode is a generic, composable, plan-first, data-defined methodology loop for driving coding agents — the repeatable way a human works with agents to get software built. Each mode is data (a target, a phase graph, per-phase Reads and Emits contracts, prompt skill pointers, and example runs) interpreted by one generic engine, so a new methodology is authored and simulated as data with no code change.",
  docLink: {
    href: "/docs/swarm-manager/concepts/EXECUTION-MODES.md",
    label: "Read the canonical operating modes doc",
  },
  sections: [
    {
      heading: "Concept",
      label: "Target",
      body: "A mode declares its unit of work — a plan-manager plan, a plan reference, or an initiative. The engine runs on a generic run context; a target adapter supplies the target-specific reads and identity. Initiative is one adapter among several, so the simplest useful run is 'point the loop at a plan and drain it', with no initiative ceremony.",
    },
    {
      heading: "Concept",
      label: "Phase graph",
      body: "A mode runs as a graph of named phases (investigate, plan, execute, review, reconcile). Each phase has its own resolved prompt-manager SkillID, agent profile, a declared Reads contract, and a declared output (Emits) contract.",
    },
    {
      heading: "Concept",
      label: "Guard & classification",
      body: "Branching is a generic field-predicate over a phase's structured output — 'always', a leaf comparison/presence/membership check, or a composite (all/any/not). When the routing field is not emitted directly, the transition declares a classification-on-transition contract that derives it from the handoff via the resolution ladder — a built-in step, not an agent phase.",
    },
    {
      heading: "Concept",
      label: "Composition",
      body: "A phase may delegate to a sub-mode (executed_by): the engine runs the sub-mode's loop as that phase, one level deep. Initiative modes compose the generic plan drain instead of duplicating execution — the composed graph renders inline, and the backend stays the routing source of truth.",
    },
    {
      heading: "Concept",
      label: "Slice & handoff",
      body: "A slice is one comprehensively-completable unit of work — a whole phase or the remainder of one. A round advances the frontier by one slice and leaves a handoff (what was completed, changed files, tests, blockers, next step) so a fresh agent continues correctly.",
    },
    {
      heading: "Resolution ladder",
      label: "L0 · True final message",
      body: "Agents often emit a 'final' message, then a subagent appends more. L0 scans the last few messages for the phase's declared output shape to find the real answer instead of trusting the chronologically-last message.",
    },
    {
      heading: "Resolution ladder",
      label: "L1 · Deterministic extraction",
      body: "The declared output schema (field names, types, required, enum, bounds) is extracted directly from the message — no model call — when the output is well-formed.",
    },
    {
      heading: "Resolution ladder",
      label: "L2 · Classifier fallback",
      body: "When deterministic extraction can't cleanly recover the fields, the raw output plus the declared schema are routed through an LLM classifier that reconstructs the declared scalar fields — and abstains rather than guessing.",
    },
    {
      heading: "Resolution ladder",
      label: "L3 · Contract validation",
      body: "The resolved result is validated against the declared output contract. Malformed or trailing-subagent output resolves to the correct structured result or an honest abstain — the phase no longer fails outright on imperfect model output.",
    },
  ],
};
