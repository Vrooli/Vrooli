import type { ComponentExperience } from "../api/components";
import { asVerdict } from "./verdicts";

export const EVIDENCE_STATUS = {
  covered: "covered",
  partial: "partial",
  unmeasured: "unmeasured",
  unavailable: "unavailable",
  notConfigured: "not-configured",
} as const;

export type EvidenceStatus = (typeof EVIDENCE_STATUS)[keyof typeof EVIDENCE_STATUS];

export interface StateCoverage {
  total: number;
  covered: number;
  status: EvidenceStatus;
  /** Deduplicated capture records, not raw sweep attempts. */
  captures: number;
  latest?: string;
  message?: string;
}

export interface ExperienceSummary {
  /** Null when the contract declares no states. */
  stateCoverage: StateCoverage | null;
  /** One status for the header and the drawer, even when no states exist. */
  status: EvidenceStatus;
}

/**
 * Derive coverage from the contract and its captures.
 *
 * Experience Manager reports transport health separately from state coverage.
 * Trusting its "evidenceStatus" string was the bug: it said "available" for a
 * reachable manager whose only capture was skipped, and the checks looked for
 * the never-emitted "fresh" value, so complete coverage could not pass.
 */
function coverageCounts(experience: ComponentExperience): { total: number; covered: number } {
  const total = experience.states.length;
  if (total === 0) return { total, covered: 0 };
  const passingStates = new Set(
    experience.evidence
      .filter((row) => asVerdict(row.verdict) === "pass")
      .map((row) => row.stateId)
      .filter(Boolean),
  );
  const covered = experience.states.filter((state) => passingStates.has(state.id)).length;
  return { total, covered };
}

export function evidenceStatusFor(experience: ComponentExperience | undefined): EvidenceStatus {
  if (!experience) return EVIDENCE_STATUS.notConfigured;
  if (experience.evidenceStatus === EVIDENCE_STATUS.unavailable) {
    return EVIDENCE_STATUS.unavailable;
  }
  if (experience.evidenceStatus === EVIDENCE_STATUS.notConfigured) {
    return EVIDENCE_STATUS.notConfigured;
  }
  const { total, covered } = coverageCounts(experience);
  if (total === 0) return EVIDENCE_STATUS.unmeasured;
  if (covered === total) return EVIDENCE_STATUS.covered;
  return covered > 0 ? EVIDENCE_STATUS.partial : EVIDENCE_STATUS.unmeasured;
}

/** One status tone, so a chip cannot drift between the header and the drawer. */
export function evidenceStatusTone(status: EvidenceStatus): "green" | "amber" | "red" | "ice" {
  switch (status) {
    case EVIDENCE_STATUS.covered:
      return "green";
    case EVIDENCE_STATUS.partial:
    case EVIDENCE_STATUS.unmeasured:
      return "amber";
    case EVIDENCE_STATUS.unavailable:
      return "red";
    default:
      return "ice";
  }
}

export function evidenceStatusCopy(status: EvidenceStatus): { key: string; defaultValue: string } {
  switch (status) {
    case EVIDENCE_STATUS.covered:
      return { key: "workbench.component.evidenceCovered", defaultValue: "Covered" };
    case EVIDENCE_STATUS.partial:
      return { key: "workbench.component.evidencePartial", defaultValue: "Partial" };
    case EVIDENCE_STATUS.unmeasured:
      return { key: "workbench.component.evidenceUnmeasured", defaultValue: "Unmeasured" };
    case EVIDENCE_STATUS.unavailable:
      return { key: "workbench.component.evidenceUnavailable", defaultValue: "Unavailable" };
    case EVIDENCE_STATUS.notConfigured:
      return { key: "workbench.component.evidenceNotConfigured", defaultValue: "No contract" };
    default:
      return { key: "workbench.component.evidenceUnmeasured", defaultValue: "Unmeasured" };
  }
}

/**
 * Projects one experience contract and its evidence stream into what the
 * workbench shows. A claim with several states takes its worst verdict: one
 * failed or uncaptured state must not be hidden by a passing sibling.
 */
export function summarizeExperience(
  experience: ComponentExperience | undefined,
): ExperienceSummary {
  if (!experience) {
    return {
      stateCoverage: null,
      status: EVIDENCE_STATUS.notConfigured,
    };
  }

  const status = evidenceStatusFor(experience);
  const evidence = experience.evidence;
  const { total, covered } = coverageCounts(experience);
  const latest = evidence
    .map((row) => row.checkedAt)
    .filter(Boolean)
    .sort()
    .at(-1);
  const message = experience.evidenceMessage || evidence.find((row) => row.message)?.message || "";
  const stateCoverage =
    total === 0
      ? null
      : {
          total,
          covered,
          status,
          captures: evidence.length,
          latest,
          message,
        };

  return { stateCoverage, status };
}
