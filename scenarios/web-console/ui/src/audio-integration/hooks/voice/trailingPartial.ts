/**
 * Turn-end trailing-partial promotion.
 *
 * Streaming STT (kyutai) delivers durable text as `segment-final` events; the
 * final message carries only the un-segmented tail, and for kyutai it is empty
 * once any segment committed. If the last words of a turn were still an
 * uncommitted partial when the turn ended — because a teardown race dropped the
 * server flush — that text used to be wiped, a silent tail-loss.
 *
 * This helper decides the durable delta to append for such a trailing partial.
 * It is kept pure so the promotion rule is directly testable; the hook owns the
 * stateful dedup (it clears the tracked partial when a segment-final commits or
 * a non-empty final delivers, so a non-empty `trailingPartial` reaching here is
 * genuinely undelivered).
 */
export interface TrailingPartialInput {
  /** Latest interim hypothesis for the uncommitted tail. */
  trailingPartial: string;
  /** True when the server's final/segment path already delivered this turn's
   *  tail (non-empty final, or a segment-final that cleared the tracked
   *  partial). When true, nothing is promoted — avoids double-appending. */
  finalDelivered: boolean;
  /** True when at least one segment already committed this turn, so the
   *  promoted tail needs a leading space to not run into the prior segment. */
  hasSegments: boolean;
}

/**
 * Returns the durable text delta to append for a trailing partial, or null when
 * there is nothing to promote (empty tail, or the tail was already delivered).
 */
export function trailingPartialDelta(input: TrailingPartialInput): string | null {
  const tail = input.trailingPartial.trim();
  if (!tail || input.finalDelivered) return null;
  // Space-separate from a prior committed segment unless the tail opens with
  // closing punctuation. Mirrors handleSegmentFinal's join rule.
  return input.hasSegments && !/^[\s,.!?;:]/.test(tail) ? ` ${tail}` : tail;
}
