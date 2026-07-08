/**
 * PhaseView — the source-agnostic view-model behind the shared PhaseViewer.
 *
 * One phase's story is the same four concerns — Instructions, Reads, Emits,
 * Transition — regardless of whether the data comes from the static contract,
 * a deterministic simulation preset, or a real live round. This module builds a
 * PhaseView from each of those three sources so the viewer never forks by
 * surface; only the fill differs.
 */

import type {
  OperatingModeArtifactSnapshot,
  OperatingModeCatalogPhase,
  OperatingModePhaseTransition,
  OperatingModeRound,
  OperatingModeRoundItem,
  OperatingModeSimulationStep,
  OperatingModeWorkspace,
} from "../../../types/operating-mode";
import {
  phaseEmitSchema,
  type PhaseEmitSpec,
  type PhaseTraceTransitionInput,
} from "./phase-interpretability";

export type PhaseViewSource = "contract" | "simulation" | "live";

export interface PhaseViewReads {
  items: OperatingModeRoundItem[];
  artifacts: OperatingModeArtifactSnapshot[];
  priorRounds: OperatingModeRound[];
  acceptanceCriteria: string[];
}

/**
 * How the Instructions tab should fetch the prompt for this view. A
 * discriminated union so the lazy render hook can dispatch without the viewer
 * knowing the mechanics: `contract` fetches the raw skill template (unfilled
 * {{VARIABLE}} slots), the other two call the server render endpoints.
 */
export type PhasePromptRequest =
  | {
      source: "contract";
      skillId: string;
      profileKey: string;
    }
  | {
      source: "simulation";
      mode: string;
      preset: string;
      stepIndex: number;
      skillId: string;
      profileKey: string;
      /** Fallback substitution map from the step if the render call fails. */
      variables: Record<string, string>;
    }
  | {
      source: "live";
      initiative: string;
      phase: string;
      round?: number;
      skillId: string;
      profileKey: string;
      variables: Record<string, string>;
    };

export interface PhaseView {
  source: PhaseViewSource;
  /** snake_case phase id. */
  phase: string;
  /** Operator-facing headline. */
  label: string;
  skillId: string;
  profileKey: string;
  status?: string;
  terminal?: boolean;
  /** Fixture/live data behind each Reads card; absent for the contract source. */
  reads?: PhaseViewReads;
  /** Actual emitted output for simulation/live; drives the Emits cards. */
  output?: unknown;
  /** Declared emit schema for the contract source. */
  emitSchema?: PhaseEmitSpec[];
  /** The single fired transition for simulation/live. */
  firedTransition?: PhaseTraceTransitionInput;
  /** All declared outgoing transitions for the contract source. */
  declaredTransitions?: OperatingModePhaseTransition[];
  /** How to render the Instructions tab. */
  prompt: PhasePromptRequest;
}

export function contractPhaseView(
  phase: OperatingModeCatalogPhase,
  transitions: OperatingModePhaseTransition[] = [],
): PhaseView {
  const outgoing = transitions.filter((transition) => transition.from === phase.phase);
  return {
    source: "contract",
    phase: phase.phase,
    label: phase.label || phase.title || phase.phase,
    skillId: phase.skillId,
    profileKey: phase.profileKey,
    terminal: phase.isTerminal,
    emitSchema: phaseEmitSchema(phase),
    declaredTransitions: outgoing,
    prompt: {
      source: "contract",
      skillId: phase.skillId,
      profileKey: phase.profileKey,
    },
  };
}

export function simulationPhaseView(
  step: OperatingModeSimulationStep,
  mode: string,
  preset: string,
): PhaseView {
  const skillId = step.skillId ?? "";
  const profileKey = step.profileKey ?? step.round.agentProfileKey ?? "";
  return {
    source: "simulation",
    phase: step.phase,
    label: step.phase,
    skillId,
    profileKey,
    status: step.round.status,
    terminal: step.terminal,
    reads: {
      items: step.inputs.items,
      artifacts: step.inputs.artifacts,
      priorRounds: step.inputs.priorRounds,
      acceptanceCriteria: step.inputs.acceptanceCriteria,
    },
    output: step.output,
    firedTransition: step.transition
      ? {
          from: step.transition.from,
          to: step.transition.to ?? "",
          conditionKind: step.transition.conditionKind,
          label: step.transition.label,
          payloadKey: step.transition.payloadKey,
          progressDecision: step.transition.progressDecision,
        }
      : undefined,
    prompt: {
      source: "simulation",
      mode,
      preset,
      stepIndex: step.index,
      skillId,
      profileKey,
      variables: step.promptVariables ?? {},
    },
  };
}

export function livePhaseView(
  round: OperatingModeRound,
  workspace: OperatingModeWorkspace,
  transitions: OperatingModePhaseTransition[],
  initiative: string,
): PhaseView {
  const priorRounds = workspace.rounds.filter((candidate) => candidate.round < round.round);
  const firedTransition = round.status === "completed"
    ? selectLiveTransition(round, transitions)
    : undefined;
  const workspacePhase = workspace.definition.phases.find((phase) => phase.phase === round.phase);
  return {
    source: "live",
    phase: round.phase,
    label: round.phase,
    skillId: skillIdForLiveRound(round, workspace),
    profileKey: round.agentProfileKey || workspacePhase?.profileKey || "",
    status: round.status,
    terminal: round.status === "completed" && !firedTransition,
    reads: {
      items: round.items ?? [],
      artifacts: workspace.artifacts,
      priorRounds,
      acceptanceCriteria: stringArrayFromRecord(round.payload, "acceptance_criteria"),
    },
    output: round.payload,
    firedTransition: firedTransition
      ? {
          from: firedTransition.from,
          to: firedTransition.to || undefined,
          conditionKind: firedTransition.conditionKind,
          label: firedTransition.label,
          payloadKey: firedTransition.payloadKey,
          progressDecision: firedTransition.progressDecision,
        }
      : undefined,
    prompt: {
      source: "live",
      initiative,
      phase: round.phase,
      round: round.round,
      skillId: skillIdForLiveRound(round, workspace),
      profileKey: round.agentProfileKey || workspacePhase?.profileKey || "",
      variables: {},
    },
  };
}

// The live round payload records the skill id at start; fall back to nothing so
// the Instructions tab still renders (the render endpoint resolves the skill
// server-side regardless).
function skillIdForLiveRound(round: OperatingModeRound, _workspace: OperatingModeWorkspace): string {
  const payload = round.payload ?? {};
  const skillId = payload.skill_id;
  return typeof skillId === "string" ? skillId : "";
}

function selectLiveTransition(
  round: OperatingModeRound,
  transitions: OperatingModePhaseTransition[],
): OperatingModePhaseTransition | undefined {
  const outgoing = transitions.filter((transition) => transition.from === round.phase);
  return outgoing.find((transition) => transitionMatchesPayload(transition, round.payload ?? {}));
}

function transitionMatchesPayload(
  transition: OperatingModePhaseTransition,
  payload: Record<string, unknown>,
): boolean {
  switch (transition.conditionKind) {
    case "payload_bool":
      return Boolean(transition.payloadKey && payload[transition.payloadKey] === true);
    case "progress_decision": {
      const progress = objectFromRecord(payload, "progress");
      return progress.decision === transition.progressDecision;
    }
    case "always":
      return true;
  }
}

function objectFromRecord(record: Record<string, unknown> | undefined, key: string): Record<string, unknown> {
  const value = record?.[key];
  return value && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : {};
}

function stringArrayFromRecord(record: Record<string, unknown> | undefined, key: string): string[] {
  const value = record?.[key];
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
}
