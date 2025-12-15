/**
 * Recording Buffer - In-Memory TimelineEntry Storage
 *
 * Stores proto TimelineEntry objects in memory during a recording session.
 * Each session has its own buffer keyed by session ID.
 *
 * PROTO-FIRST ARCHITECTURE:
 * This buffer stores TimelineEntry directly (the canonical proto format),
 * eliminating the need for intermediate types like RecordedAction.
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ BUFFER BEHAVIOR:                                                        │
 * │                                                                         │
 * │   bufferTimelineEntry()                                                 │
 * │        │                                                                │
 * │        ├──▶ Duplicate? (same entry.id) ──▶ Return false (no-op)         │
 * │        │                                                                │
 * │        ├──▶ Buffer full? ──▶ Evict oldest (FIFO), then add              │
 * │        │                                                                │
 * │        └──▶ Add to buffer, return true                                  │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * IDEMPOTENCY: Same entry ID inserted twice is a no-op (safe for retries)
 * MEMORY SAFETY: Buffer size is capped, oldest entries evicted when full
 */

import type { TimelineEntry } from '../proto/recording';
import { MAX_RECORDING_BUFFER_SIZE, logger } from '../utils';

// In-memory entry buffers keyed by session ID
const entryBuffers = new Map<string, TimelineEntry[]>();

// Track eviction counts for observability
const evictionCounts = new Map<string, number>();

// Track seen entry IDs per session for deduplication
const seenEntryIds = new Map<string, Set<string>>();

// Track last sequence number per session for ordering validation
const lastSequenceNums = new Map<string, number>();

export function initRecordingBuffer(sessionId: string): void {
  entryBuffers.set(sessionId, []);
  evictionCounts.set(sessionId, 0);
  seenEntryIds.set(sessionId, new Set());
  lastSequenceNums.set(sessionId, -1);
}

/**
 * Buffer a TimelineEntry.
 *
 * Hardened: Enforces maximum buffer size with FIFO eviction.
 * Logs a warning when eviction occurs to signal potential issues.
 *
 * Idempotency: Entries with the same ID are deduplicated (no-op on second insert).
 * This makes buffering safe for replay scenarios where callbacks might fire twice.
 *
 * @returns true if entry was buffered, false if it was a duplicate (already seen)
 */
export function bufferTimelineEntry(sessionId: string, entry: TimelineEntry): boolean {
  let buffer = entryBuffers.get(sessionId);

  if (!buffer) {
    buffer = [];
    entryBuffers.set(sessionId, buffer);
    evictionCounts.set(sessionId, 0);
    seenEntryIds.set(sessionId, new Set());
    lastSequenceNums.set(sessionId, -1);
  }

  // Idempotency: Check if we've already seen this entry ID
  const seen = seenEntryIds.get(sessionId);
  if (seen?.has(entry.id)) {
    logger.debug('Recording buffer: duplicate entry ignored', {
      sessionId,
      entryId: entry.id,
      sequenceNum: entry.sequenceNum,
      hint: 'Entry already buffered, treating as idempotent retry',
    });
    return false;
  }

  // Validate sequence ordering (warn but don't reject - network timing can cause reordering)
  const lastSeq = lastSequenceNums.get(sessionId) ?? -1;
  if (entry.sequenceNum <= lastSeq) {
    logger.warn('Recording buffer: out-of-order entry received', {
      sessionId,
      entryId: entry.id,
      sequenceNum: entry.sequenceNum,
      lastSequenceNum: lastSeq,
      hint: 'Entry arrived out of order, may indicate network issues',
    });
  }

  // Hardened: Enforce maximum buffer size
  if (buffer.length >= MAX_RECORDING_BUFFER_SIZE) {
    // Evict oldest entry (FIFO)
    const evictedEntry = buffer.shift();

    // Also remove from seen set to allow memory cleanup
    if (evictedEntry) {
      seen?.delete(evictedEntry.id);
    }

    // Track eviction count
    const evictions = (evictionCounts.get(sessionId) || 0) + 1;
    evictionCounts.set(sessionId, evictions);

    // Log warning on first eviction and periodically thereafter
    if (evictions === 1 || evictions % 100 === 0) {
      logger.warn('Recording buffer full, evicting old entries', {
        sessionId,
        maxSize: MAX_RECORDING_BUFFER_SIZE,
        totalEvictions: evictions,
        hint: 'Consider stopping recording or increasing buffer size',
      });
    }
  }

  // Add entry to buffer and track it
  buffer.push(entry);
  seen?.add(entry.id);
  lastSequenceNums.set(sessionId, Math.max(lastSeq, entry.sequenceNum));

  return true;
}

/**
 * Get all buffered TimelineEntry objects for a session.
 */
export function getTimelineEntries(sessionId: string): TimelineEntry[] {
  return entryBuffers.get(sessionId) || [];
}

/**
 * Get the count of buffered entries.
 */
export function getTimelineEntryCount(sessionId: string): number {
  return getTimelineEntries(sessionId).length;
}

/**
 * Get buffer statistics for monitoring.
 */
export function getBufferStats(sessionId: string): {
  entryCount: number;
  evictionCount: number;
  maxSize: number;
} {
  return {
    entryCount: getTimelineEntryCount(sessionId),
    evictionCount: evictionCounts.get(sessionId) || 0,
    maxSize: MAX_RECORDING_BUFFER_SIZE,
  };
}

/**
 * Clear all buffered entries for a session but keep the buffer initialized.
 */
export function clearTimelineEntries(sessionId: string): void {
  entryBuffers.set(sessionId, []);
  evictionCounts.set(sessionId, 0);
  seenEntryIds.set(sessionId, new Set());
  lastSequenceNums.set(sessionId, -1);
}

/**
 * Remove the buffer entirely for a session (cleanup on session end).
 */
export function removeRecordingBuffer(sessionId: string): void {
  entryBuffers.delete(sessionId);
  evictionCounts.delete(sessionId);
  seenEntryIds.delete(sessionId);
  lastSequenceNums.delete(sessionId);
}

/**
 * Check if an entry ID has already been buffered.
 * Useful for external code to check before attempting to buffer.
 */
export function isEntryBuffered(sessionId: string, entryId: string): boolean {
  const seen = seenEntryIds.get(sessionId);
  return seen?.has(entryId) ?? false;
}

