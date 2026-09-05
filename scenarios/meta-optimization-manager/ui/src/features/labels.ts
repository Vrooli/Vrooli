import { Projection, CellStatus, DenominatorConfidence, GapAxis } from "@vrooli/proto-types/meta-optimization-manager/v1/shared/model_pb";
import {
  FitnessTier,
  ReferenceEligibility,
} from "@vrooli/proto-types/meta-optimization-manager/v1/convergence/convergence_pb";
import { TrialVerdict } from "@vrooli/proto-types/meta-optimization-manager/v1/trials/trials_pb";

/**
 * Stable, locale-independent labels for the readiness-model enums. These are
 * short machine-ish tokens (the same vocabulary the CLI prints), intentionally
 * NOT translated — they are domain identifiers, not prose. User-facing prose
 * lives in the i18n catalog; these label the data points themselves.
 */

export function projectionLabel(p: Projection): string {
  switch (p) {
    case Projection.ANSWER:
      return "answer";
    case Projection.VALIDATE:
      return "validate";
    case Projection.GUIDE:
      return "guide";
    case Projection.ACT:
      return "act";
    default:
      return "cross-cutting";
  }
}

export function cellStatusLabel(s: CellStatus): string {
  switch (s) {
    case CellStatus.NOW:
      return "now";
    case CellStatus.IN_REACH:
      return "in_reach";
    case CellStatus.MISSING:
      return "missing";
    default:
      return "?";
  }
}

export function gapAxisLabel(axis: GapAxis): string {
  switch (axis) {
    case GapAxis.COVERAGE:
      return "coverage";
    case GapAxis.EMPIRICAL:
      return "empirical";
    default:
      return "?";
  }
}

export function confidenceLabel(c: DenominatorConfidence): string {
  switch (c) {
    case DenominatorConfidence.AUTHORITATIVE:
      return "authoritative";
    case DenominatorConfidence.PARTIAL:
      return "partial";
    case DenominatorConfidence.SKETCH:
      return "sketch";
    default:
      return "unspecified";
  }
}

export function tierLabel(t: FitnessTier): string {
  switch (t) {
    case FitnessTier.STRONG:
      return "strong";
    case FitnessTier.FAIR:
      return "fair";
    case FitnessTier.WEAK:
      return "weak";
    default:
      return "?";
  }
}

export function eligibilityLabel(e: ReferenceEligibility): string {
  switch (e) {
    case ReferenceEligibility.ELIGIBLE:
      return "eligible";
    case ReferenceEligibility.CANDIDATE:
      return "candidate";
    case ReferenceEligibility.INELIGIBLE:
      return "ineligible";
    default:
      return "?";
  }
}

export function verdictLabel(v: TrialVerdict): string {
  switch (v) {
    case TrialVerdict.PASS:
      return "pass";
    case TrialVerdict.FAIL:
      return "fail";
    case TrialVerdict.ERROR:
      return "error";
    default:
      return "?";
  }
}

/** pct formats a 0..1 ratio as a whole-number percentage string. */
export function pct(ratio: number): string {
  return `${Math.round(ratio * 100)}`;
}
