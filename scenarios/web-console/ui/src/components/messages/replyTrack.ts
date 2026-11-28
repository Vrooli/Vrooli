import { isAgentReply } from "../../lib/conversationRows";

/**
 * The reader steps between replies — every message that is not the
 * operator's own. These helpers answer "where is the reader among them?" and
 * "where does a step of N land?" from the visible list alone, so tapping,
 * holding to scrub, and the position readout all agree.
 *
 * The reader may be showing one of the operator's own messages (it can be
 * opened on any row). That message sits *between* two replies: a step forward
 * lands on the next reply, a step back on the previous one, and each counts
 * as one move.
 */

/**
 * Indices into the visible list of every reply, ascending. A reply is the
 * agent's own message: the work rows, notices and subagent groups between
 * them are stepped over like the operator's messages.
 */
export function replyIndices(events: readonly Parameters<typeof isAgentReply>[0][]): number[] {
  const indices: number[] = [];
  events.forEach((event, index) => {
    if (isAgentReply(event)) indices.push(index);
  });
  return indices;
}

/** First position in `indices` whose value is at least `target`. */
function lowerBound(indices: readonly number[], target: number): number {
  let low = 0;
  let high = indices.length;
  while (low < high) {
    const middle = (low + high) >>> 1;
    if ((indices[middle] ?? Infinity) < target) low = middle + 1;
    else high = middle;
  }
  return low;
}

export interface ReplySeek {
  /** The visible-list index to show, or `null` when there is nowhere to go. */
  readonly index: number | null;
  /** How many replies the step actually covered; less than asked at an end. */
  readonly moved: number;
}

/** Step `delta` replies (negative for back) from the visible-list index `from`. */
export function seekReply(indices: readonly number[], from: number, delta: number): ReplySeek {
  if (delta === 0 || indices.length === 0) return { index: null, moved: 0 };
  const position = lowerBound(indices, from);
  const onReply = indices[position] === from;
  let target: number;
  let moved: number;
  if (delta > 0) {
    // Off a reply the reader sits just before `position`.
    const start = onReply ? position : position - 1;
    target = Math.min(start + delta, indices.length - 1);
    moved = target - start;
  } else {
    target = Math.max(position + delta, 0);
    moved = position - target;
  }
  if (moved <= 0) return { index: null, moved: 0 };
  return { index: indices[target] ?? null, moved };
}

export interface ReplyPosition {
  /** 1-based place of the reply shown; the replies before it otherwise. */
  readonly ordinal: number;
  readonly total: number;
}

export function replyPosition(indices: readonly number[], from: number): ReplyPosition {
  const position = lowerBound(indices, from);
  return { ordinal: indices[position] === from ? position + 1 : position, total: indices.length };
}
