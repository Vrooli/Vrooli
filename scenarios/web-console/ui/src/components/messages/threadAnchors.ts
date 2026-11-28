import type { ConversationEvent, ConversationThreadSummary } from "../../api/conversation";
import { coversSequence } from "../../lib/conversationRows";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#agent-threads

/**
 * Places each agent thread in the main conversation: at the loaded row whose
 * range covers the thread's first row (a work row, since another thread's
 * rows only ever sit inside one). A thread that began outside the loaded
 * window is not placed until that page loads. Rows must be in sequence order.
 */
export function threadAnchors(
  events: readonly ConversationEvent[],
  threads: readonly ConversationThreadSummary[],
): ReadonlyMap<string, ConversationThreadSummary[]> {
  const anchors = new Map<string, ConversationThreadSummary[]>();
  if (threads.length === 0 || events.length === 0) return anchors;
  for (const thread of [...threads].sort((left, right) => left.firstSequence - right.firstSequence)) {
    const anchor = findCovering(events, thread.firstSequence);
    if (!anchor) continue;
    const placed = anchors.get(anchor.id);
    if (placed) placed.push(thread);
    else anchors.set(anchor.id, [thread]);
  }
  return anchors;
}

function findCovering(events: readonly ConversationEvent[], sequence: number): ConversationEvent | undefined {
  let low = 0;
  let high = events.length - 1;
  while (low <= high) {
    const middle = (low + high) >>> 1;
    const event = events[middle];
    if (!event) return undefined;
    if (coversSequence(event, sequence)) return event;
    if (event.sequence < sequence) low = middle + 1;
    else high = middle - 1;
  }
  return undefined;
}
