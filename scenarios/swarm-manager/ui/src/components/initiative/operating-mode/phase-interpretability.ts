import type {
  OperatingModeCatalogPhase,
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

export interface WorkedExample {
  title: string;
  result: Record<string, unknown>;
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

export function workedExampleForPhase(phase: OperatingModeCatalogPhase): WorkedExample {
  const contract = phase.outputContract;
  if (contract.requiresBacklogSync) {
    return {
      title: "Backlog reconcile example",
      result: {
        backlog_sync: {
          completed_items: ["item-123"],
          created_items: ["follow-up-456"],
          rationale: "The implementation shipped and left one scoped follow-up.",
        },
      },
    };
  }
  if (contract.requiresProgress) {
    return {
      title: "Progress decision example",
      result: {
        progress: {
          decision: "continue",
          completed_phases: ["phase-1"],
          current_phase: "phase-2",
          rationale: "The previous slice is complete and the next contiguous slice is ready.",
        },
      },
    };
  }
  if (contract.requiresVerdict) {
    return {
      title: "Review verdict example",
      result: {
        verdict: "accepted",
      },
    };
  }
  if (contract.requiresHandoff) {
    return {
      title: "Execution handoff example",
      result: {
        handoff: {
          summary: "Completed the next drainable plan slice.",
          completed_phases: ["Phase 2"],
          changed_files: ["scenarios/swarm-manager/api/internal/example.go"],
          tests: ["go test ./..."],
          blockers: [],
          next_step: "classify_progress",
        },
      },
    };
  }
  if ((phase.outputArtifacts?.length ?? 0) > 0) {
    const artifact = phase.outputArtifacts?.[0];
    return {
      title: "Artifact output example",
      result: {
        artifacts: [
          {
            path: artifact?.path ?? "modes/example/output.md",
            content_type: artifact?.contentType ?? "text/markdown",
            content: "# Findings\n\nKey facts and recommended next step.",
          },
        ],
      },
    };
  }
  if (phase.samplesReplanRate) {
    return {
      title: "Execution routing example",
      result: {
        replan_needed: false,
      },
    };
  }
  return {
    title: "Structured result example",
    result: {
      readiness: {
        dimensions: [{ key: "scope_defined", score: 0.85, rationale: "The work boundary is clear." }],
        overall_score: 0.85,
        ready: true,
      },
    },
  };
}

export function formatTransition(transition: OperatingModePhaseTransition): string {
  const condition = transition.label === "always" ? "always" : transition.label;
  return `if ${condition}, go to ${transition.to}`;
}
