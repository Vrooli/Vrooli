/** The reader's place in the Messages list, as restore needs it. */
export interface TopPosition {
  topEventId: string;
  topSequence: number;
  /** How far the message's top sits above the viewport top, in px (>= 0). */
  offsetPx: number;
}

/**
 * Finds the first rendered message row whose bottom is below the scroll
 * container's top: the message the reader is looking at. Rows carry
 * `data-event-id` and `data-sequence`; DOM order is list order.
 */
export function captureTopPosition(container: HTMLElement): TopPosition | null {
  const containerTop = container.getBoundingClientRect().top;
  for (const row of container.querySelectorAll<HTMLElement>("[data-event-id]")) {
    const rect = row.getBoundingClientRect();
    if (rect.bottom <= containerTop) continue;
    const topEventId = row.dataset.eventId;
    const topSequence = Number(row.dataset.sequence);
    if (!topEventId || !Number.isFinite(topSequence)) return null;
    return { topEventId, topSequence, offsetPx: Math.max(0, containerTop - rect.top) };
  }
  return null;
}
