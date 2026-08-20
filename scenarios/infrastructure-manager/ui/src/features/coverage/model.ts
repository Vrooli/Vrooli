import {
  ConfidenceLevel,
  Projection,
  type ProjectionCoverage,
} from "@vrooli/proto-types/infrastructure-manager/v1/coverage/coverage_pb";

import { strings } from "../../consts/strings.generated";
import type { ConfidenceLevel as ConfidenceName, SignalState } from "../../theme/instrument";

/**
 * Coverage-space model helpers.
 *
 * These live beside the coverage feature rather than inside its page because
 * the board reads the same spaces the detail page does, and two independent
 * derivations of "which spaces answered" is how two surfaces start disagreeing
 * about the same read.
 */

/** Catalog keys for the closed denominator-confidence vocabulary. */
export const CONFIDENCE_KEYS = {
  AUTHORITATIVE: strings.instrument.confidence.authoritative,
  PARTIAL: strings.instrument.confidence.partial,
  SKETCH: strings.instrument.confidence.sketch,
} as const satisfies Record<ConfidenceName, string>;

/** Weakest denominator first: a board is only as sound as its worst claim. */
const CONFIDENCE_RANK: Record<ConfidenceName, number> = { SKETCH: 0, PARTIAL: 1, AUTHORITATIVE: 2 };

/**
 * The weakest denominator claim among the spaces that ANSWERED.
 *
 * `null` when no space answered — there is no confidence to report, and
 * reporting the strongest available level (or none, read as fine) would let an
 * outage present as a well-founded board.
 */
export function weakestConfidence(read: readonly ProjectionCoverage[]): ConfidenceName | null {
  let weakest: ConfidenceName | null = null;
  for (const projection of read) {
    const level = confidenceName(projection.confidence?.level);
    if (weakest === null || CONFIDENCE_RANK[level] < CONFIDENCE_RANK[weakest]) {
      weakest = level;
    }
  }
  return weakest;
}

/**
 * The lamp state for one space.
 *
 * A space that did not answer is SOURCE_DOWN — a fact about the instrument,
 * rendered in a visibly different KIND from every plant-side state beside it,
 * so it can never be read as a coverage collapse. A space that answered but
 * returned no ratio is UNMEASURABLE, which carries its reason; it is not
 * blindness, because the space is declared and the instrument knows why it
 * could not grade it.
 */
export function projectionSignal(projection: ProjectionCoverage): SignalState {
  if (!projection.available) return "SOURCE_DOWN";
  if (!projection.ratio) return "UNMEASURABLE";
  if (projection.ratio.value >= 1) return "COVERED";
  if (projection.ratio.value > 0) return "PARTIAL";
  return "BLIND";
}

/** Why a space's lamp reads the way it does, for the lamp's accessible name. */
export function reasonFor(projection: ProjectionCoverage): string | undefined {
  if (projection.available) return projection.confidence?.rationale || undefined;
  return projection.unavailableReason || undefined;
}

/** The rationale a ratio travels with; a space that did not answer states why. */
export function rationaleFor(projection: ProjectionCoverage, fallback: string): string {
  return projection.confidence?.rationale || projection.unavailableReason || fallback;
}

/**
 * The space's own identifier, not a translated label.
 *
 * These are cell-space references the CLI and the setpoint use verbatim, so
 * translating them would break the one property that makes them useful: a
 * reader can paste what they see into `vrooli` and get the same answer.
 */
export function projectionName(value: number): string {
  return (Projection[value] || "").replace("PROJECTION_", "").toLowerCase().replace(/_/g, "-");
}

/**
 * A missing or unrecognised confidence level is SKETCH, never AUTHORITATIVE.
 * The unspecified default on the wire means the owner did not make a claim,
 * and reading "no claim" as "the strongest claim" is how an unfounded ratio
 * acquires authority it was never given.
 */
export function confidenceName(value: ConfidenceLevel | undefined): ConfidenceName {
  return value === ConfidenceLevel.AUTHORITATIVE
    ? "AUTHORITATIVE"
    : value === ConfidenceLevel.PARTIAL
      ? "PARTIAL"
      : "SKETCH";
}
