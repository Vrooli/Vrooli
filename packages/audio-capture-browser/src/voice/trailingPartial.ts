/**
 * Turn-end uncommitted-tail recovery — the committed-length cursor.
 *
 * Streaming STT (kyutai) delivers durable text as `segment-final` events; the
 * final message carries only the un-segmented tail, and for kyutai it is empty
 * once any segment committed. If the last words of a turn were still an
 * uncommitted partial when the turn ended — because a teardown race dropped the
 * server flush — that text used to be wiped, a silent tail-loss.
 *
 * The recovery rule is a committed-length cursor rather than a single
 * overwritten slot: it promotes exactly the remainder of the latest partial
 * that lies BEYOND what the durable segment-finals already committed. This is
 * robust to both partial shapes the durability contract permits:
 *   - a per-segment tail (kyutai today: the partial is already only the
 *     uncommitted segment), and
 *   - a coalesced running FULL hypothesis (the natural shape of a
 *     "coalesce-to-latest" partial), where the already-committed prefix must be
 *     stripped so it is not double-appended.
 *
 * Kept pure so the promotion rule is directly testable; the hook owns the
 * stateful cursor (the committed transcript this turn and the latest partial).
 */
export interface UncommittedRemainderInput {
  /** The durable transcript committed this turn (segment-finals joined). */
  committedText: string;
  /** Latest interim hypothesis for the uncommitted tail. */
  latestPartial: string;
}

/**
 * Returns the durable text delta to append for the uncommitted tail beyond the
 * committed transcript, or null when there is nothing new to promote (empty
 * tail, or the partial adds nothing past what segment-finals already committed).
 */
export function uncommittedRemainder(input: UncommittedRemainderInput): string | null {
  const partial = input.latestPartial.trim();
  if (!partial) return null;
  const committed = input.committedText.trim();

  // If the partial is the running FULL hypothesis, strip the committed prefix;
  // otherwise it is already a pure current-segment tail.
  let tail = partial;
  if (committed && partial.startsWith(committed)) {
    tail = partial.slice(committed.length).trim();
  }
  if (!tail) return null;

  // Space-separate from a non-empty committed transcript unless the tail opens
  // with closing punctuation. Mirrors handleSegmentFinal's join rule.
  const needsSpace = committed.length > 0 && !/^[\s,.!?;:]/.test(tail);
  return needsSpace ? ` ${tail}` : tail;
}
