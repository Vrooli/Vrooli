import type {
  OperatingModeCatalogPhase,
  OperatingModeHandoff,
  OperatingModePhaseTransition,
} from "../../../types/operating-mode";

export interface PhaseReadSpec {
  key: string;
  label: string;
  meaning: string;
}

export interface PhaseEmitSpec {
  field: string;
  label: string;
  meaning: string;
  required: boolean;
}

export const PHASE_READS: PhaseReadSpec[] = [
  {
    key: "PRIOR_ROUNDS_JSON",
    label: "Prior rounds",
    meaning: "Completed rounds, handoffs, payloads, and errors already recorded for this mode.",
  },
  {
    key: "MEMBER_ITEMS_JSON",
    label: "Member items",
    meaning: "The initiative's current backlog scope: refs, titles, status, priority, and effort.",
  },
  {
    key: "MODE_ARTIFACTS_JSON",
    label: "Mode artifacts",
    meaning: "Durable files previously produced under the mode artifact root.",
  },
  {
    key: "ACCEPTANCE_CRITERIA",
    label: "Acceptance criteria",
    meaning: "The operator-defined criteria review phases must evaluate against.",
  },
];

const EMIT_MEANINGS = {
  artifacts: "Files the phase writes into the mode artifact store.",
  handoff: "A single execution handoff with completed phases, changed files, tests, blockers, and next step.",
  handoffs: "Multiple execution handoffs when a phase drains more than one slice.",
  readiness: "A scored readiness report for plan or initiative quality.",
  progress: "The continue, blocked, replan, or complete decision used by phased plan routing.",
  verdict: "The acceptance review outcome consumed by review metrics.",
  replan_needed: "A boolean signal that routes exploratory execution back to investigation.",
  backlog_sync: "A proposed backlog mutation plan for reconcile phases.",
} as const;

export const PHASE_RESULT_FIELDS = Object.keys(EMIT_MEANINGS);

export function phaseEmitSchema(phase: OperatingModeCatalogPhase): PhaseEmitSpec[] {
  const contract = phase.outputContract;
  const requiredArtifacts = contract.requiredArtifactCount > 0;
  return [
    {
      field: "artifacts",
      label: "artifacts[]",
      meaning: EMIT_MEANINGS.artifacts,
      required: requiredArtifacts,
    },
    {
      field: "handoff",
      label: "handoff",
      meaning: EMIT_MEANINGS.handoff,
      required: contract.requiresHandoff,
    },
    {
      field: "handoffs",
      label: "handoffs[]",
      meaning: EMIT_MEANINGS.handoffs,
      required: false,
    },
    {
      field: "readiness",
      label: "readiness",
      meaning: EMIT_MEANINGS.readiness,
      required: false,
    },
    {
      field: "progress",
      label: "progress",
      meaning: EMIT_MEANINGS.progress,
      required: contract.requiresProgress,
    },
    {
      field: "verdict",
      label: "verdict",
      meaning: EMIT_MEANINGS.verdict,
      required: contract.requiresVerdict,
    },
    {
      field: "replan_needed",
      label: "replan_needed",
      meaning: EMIT_MEANINGS.replan_needed,
      required: false,
    },
    {
      field: "backlog_sync",
      label: "backlog_sync",
      meaning: EMIT_MEANINGS.backlog_sync,
      required: contract.requiresBacklogSync,
    },
  ];
}

export function formatTransition(transition: OperatingModePhaseTransition): string {
  const condition = transition.label === "always" ? "always" : transition.label;
  return `if ${condition}, go to ${transition.to}`;
}

/**
 * The four semantic categories a phase reads from. Each maps a raw context
 * blob onto operator language shared between the static phase cards and the
 * live/simulation Flow trace, so the two surfaces never drift.
 */
export type PhaseReadCategory = "items" | "artifacts" | "priorRounds" | "acceptanceCriteria";

export interface PhaseReadCategorySpec {
  key: PhaseReadCategory;
  label: string;
  meaning: string;
  /** The prompt template variable this read fills, e.g. MEMBER_ITEMS_JSON. */
  variable: string;
}

const READ_MEANING = (key: string): string =>
  PHASE_READS.find((read) => read.key === key)?.meaning ?? "";

export const PHASE_READ_CATEGORIES: PhaseReadCategorySpec[] = [
  { key: "items", label: "Member items", meaning: READ_MEANING("MEMBER_ITEMS_JSON"), variable: "MEMBER_ITEMS_JSON" },
  { key: "artifacts", label: "Mode artifacts", meaning: READ_MEANING("MODE_ARTIFACTS_JSON"), variable: "MODE_ARTIFACTS_JSON" },
  { key: "priorRounds", label: "Prior rounds", meaning: READ_MEANING("PRIOR_ROUNDS_JSON"), variable: "PRIOR_ROUNDS_JSON" },
  { key: "acceptanceCriteria", label: "Acceptance criteria", meaning: READ_MEANING("ACCEPTANCE_CRITERIA"), variable: "ACCEPTANCE_CRITERIA" },
];

export interface TransitionExplanation {
  /** Short headline, e.g. "execute → review" or "terminal". */
  headline: string;
  /** One-sentence reason the next phase was (or was not) selected. */
  reason: string;
  /** Visual tone hint for the caller. */
  tone: "route" | "terminal" | "pending" | "blocked";
}

export interface PhaseTraceTransitionInput {
  from: string;
  to?: string;
  conditionKind?: string;
  label?: string;
  payloadKey?: string;
  progressDecision?: string;
}

/**
 * Explain why a completed phase routes (or does not) to the next phase. The
 * text is derived entirely from backend-provided transition data — React never
 * re-derives routing logic, it only phrases what the guard decided.
 */
export function describeTransition(
  transition: PhaseTraceTransitionInput | undefined,
  terminal: boolean | undefined,
): TransitionExplanation {
  if (transition && transition.to) {
    return {
      headline: `${transition.from} → ${transition.to}`,
      reason: transitionReason(transition),
      tone: "route",
    };
  }
  if (transition && !transition.to) {
    // A guard matched but has no downstream target (e.g. a "blocked" decision):
    // the cycle stops here even though a condition was evaluated.
    return {
      headline: "cycle ends here",
      reason: `${transitionReason(transition)} No downstream phase is defined for this outcome, so the cycle stops for operator intervention.`,
      tone: "blocked",
    };
  }
  if (terminal) {
    return {
      headline: "terminal phase",
      reason: "This is the mode's terminal phase — reaching it closes the current cycle.",
      tone: "terminal",
    };
  }
  return {
    headline: "pending",
    reason: "This phase has not completed yet, so no transition has been evaluated.",
    tone: "pending",
  };
}

function transitionReason(transition: PhaseTraceTransitionInput): string {
  switch (transition.conditionKind) {
    case "payload_bool":
      return `The phase output set ${transition.payloadKey ?? "a boolean signal"} = true, which fires this transition.`;
    case "progress_decision":
      return `The phase reported a progress decision of "${transition.progressDecision ?? "?"}", which routes here.`;
    case "always":
      return "This is an unconditional transition — a clean completion always advances here.";
    default:
      return transition.label ? `Guard: ${transition.label}.` : "The transition guard selected this path.";
  }
}

export interface PhaseTraceEmitBacklogSync {
  completedItems: string[];
  createdItems: string[];
  updatedItems: string[];
  rationale?: string;
}

/**
 * Normalized, operator-facing projection of whatever a phase emitted. It
 * tolerates both the camelCase simulation shape (OperatingModePhaseResult) and
 * the snake_case live round payload, so one set of emit cards renders both.
 */
export interface PhaseTraceEmits {
  artifacts: Array<{ path: string; contentType?: string }>;
  handoffs: OperatingModeHandoff[];
  progress?: { decision: string; rationale?: string };
  verdict?: string;
  replanNeeded: boolean;
  readiness?: Record<string, unknown>;
  backlogSync?: PhaseTraceEmitBacklogSync;
  hasContent: boolean;
}

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() !== "" ? value : undefined;
}

function asStringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
}

function normalizeHandoff(raw: unknown): OperatingModeHandoff {
  const handoff = asRecord(raw);
  return {
    summary: asString(handoff.summary),
    completedPhases: asStringArray(handoff.completed_phases ?? handoff.completedPhases),
    changedFiles: asStringArray(handoff.changed_files ?? handoff.changedFiles),
    tests: asStringArray(handoff.tests),
    blockers: asStringArray(handoff.blockers),
    nextStep: asString(handoff.next_step ?? handoff.nextStep),
  };
}

/**
 * Build the shared emit view-model from a phase's raw output. Reads both key
 * spellings so simulation (camelCase) and live payloads (snake_case) collapse
 * onto the same semantic cards.
 */
export function phaseTraceEmits(output: unknown): PhaseTraceEmits {
  const record = asRecord(output);
  const artifacts = Array.isArray(record.artifacts)
    ? record.artifacts.map((item) => {
        const artifact = asRecord(item);
        return {
          path: asString(artifact.path) ?? "",
          contentType: asString(artifact.content_type ?? artifact.contentType),
        };
      }).filter((artifact) => artifact.path !== "")
    : [];

  const handoffs: OperatingModeHandoff[] = [];
  if (record.handoff && Object.keys(asRecord(record.handoff)).length > 0) {
    handoffs.push(normalizeHandoff(record.handoff));
  }
  if (Array.isArray(record.handoffs)) {
    for (const item of record.handoffs) handoffs.push(normalizeHandoff(item));
  }

  const progressRecord = asRecord(record.progress);
  const progressDecision = asString(progressRecord.decision);
  const progress = progressDecision
    ? { decision: progressDecision, rationale: asString(progressRecord.rationale) }
    : undefined;

  const verdict = asString(record.verdict);
  const replanNeeded = record.replan_needed === true || record.replanNeeded === true;
  const readinessRecord = asRecord(record.readiness);
  const readiness = Object.keys(readinessRecord).length > 0 ? readinessRecord : undefined;

  const backlogRecord = asRecord(record.backlog_sync ?? record.backlogSync);
  const backlogSync = Object.keys(backlogRecord).length > 0
    ? {
        completedItems: asStringArray(backlogRecord.completed_items ?? backlogRecord.completedItems),
        createdItems: asStringArray(backlogRecord.created_items ?? backlogRecord.createdItems),
        updatedItems: asStringArray(backlogRecord.updated_items ?? backlogRecord.updatedItems),
        rationale: asString(backlogRecord.rationale),
      }
    : undefined;

  const hasContent =
    artifacts.length > 0 ||
    handoffs.length > 0 ||
    Boolean(progress) ||
    Boolean(verdict) ||
    replanNeeded ||
    Boolean(readiness) ||
    Boolean(backlogSync);

  return { artifacts, handoffs, progress, verdict, replanNeeded, readiness, backlogSync, hasContent };
}
