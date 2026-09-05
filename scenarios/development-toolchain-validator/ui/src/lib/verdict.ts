import {
  TupleKind,
  Verdict,
} from "../api/validationRecord";
import type { TupleVerdict } from "../api/report";
import type { VerdictKind } from "../shared/ui/composites/VerdictCell";

/**
 * Map a proto Verdict + stale flag to the visual VerdictKind the
 * composite expects. `stale` always wins over a pass — the user needs
 * to see the staleness before any positive verdict.
 */
export function verdictToKind(verdict: Verdict, stale: boolean): VerdictKind {
  if (stale) return "stale";
  switch (verdict) {
    case Verdict.PASS:
      return "pass";
    case Verdict.UNEXPECTED_MUTATION:
      return "unexpected";
    case Verdict.RUN_FAILURE:
    case Verdict.TOOL_FAILURE:
      return "failure";
    case Verdict.UNSPECIFIED:
    default:
      return "neutral";
  }
}

/**
 * Map a TupleKind to a short URL segment used in the
 * `/goldens/:slug/:tupleKind/:subjectId` route.
 */
export function tupleKindToSegment(kind: TupleKind): "skill" | "tool" {
  return kind === TupleKind.TOOL ? "tool" : "skill";
}

export function segmentToTupleKind(segment: string): TupleKind {
  return segment === "tool" ? TupleKind.TOOL : TupleKind.SKILL;
}

export interface VerdictSummaryCounts {
  pass: number;
  stale: number;
  unexpected: number;
  failure: number;
  total: number;
}

export function summarizeVerdicts(rows: readonly TupleVerdict[]): VerdictSummaryCounts {
  const counts: VerdictSummaryCounts = {
    pass: 0,
    stale: 0,
    unexpected: 0,
    failure: 0,
    total: rows.length,
  };
  for (const row of rows) {
    const kind = verdictToKind(row.latestVerdict, row.stale);
    if (kind === "pass") counts.pass += 1;
    else if (kind === "stale") counts.stale += 1;
    else if (kind === "unexpected") counts.unexpected += 1;
    else if (kind === "failure") counts.failure += 1;
  }
  return counts;
}

/**
 * Pick a top-level Badge variant for an aggregate summary chip. Stale
 * and unexpected dominate; pure-pass is verdict-pass; empty is neutral.
 */
export function summaryToVariant(
  counts: VerdictSummaryCounts,
): VerdictKind {
  if (counts.total === 0) return "neutral";
  if (counts.failure > 0) return "failure";
  if (counts.unexpected > 0) return "unexpected";
  if (counts.stale > 0) return "stale";
  if (counts.pass > 0) return "pass";
  return "neutral";
}
