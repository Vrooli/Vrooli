import type { RecordedAction } from './types';
import { MAX_RECORDING_BUFFER_SIZE, logger } from '../utils';

/**
 * Recording Buffer - In-Memory Action Storage
 *
 * Stores recorded actions in memory during a recording session.
 * Each session has its own buffer keyed by session ID.
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ BUFFER BEHAVIOR:                                                        │
 * │                                                                         │
 * │   bufferRecordedAction()                                                │
 * │        │                                                                │
 * │        ├──▶ Duplicate? (same action.id) ──▶ Return false (no-op)        │
 * │        │                                                                │
 * │        ├──▶ Buffer full? ──▶ Evict oldest (FIFO), then add              │
 * │        │                                                                │
 * │        └──▶ Add to buffer, return true                                  │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * IDEMPOTENCY: Same action ID inserted twice is a no-op (safe for retries)
 * MEMORY SAFETY: Buffer size is capped, oldest actions evicted when full
 */

// In-memory action buffers keyed by session ID
const actionBuffers = new Map<string, RecordedAction[]>();

// Track eviction counts for observability
const evictionCounts = new Map<string, number>();

// Track seen action IDs per session for deduplication
const seenActionIds = new Map<string, Set<string>>();

// Track last sequence number per session for ordering validation
const lastSequenceNums = new Map<string, number>();

export function initRecordingBuffer(sessionId: string): void {
  actionBuffers.set(sessionId, []);
  evictionCounts.set(sessionId, 0);
  seenActionIds.set(sessionId, new Set());
  lastSequenceNums.set(sessionId, -1);
}

/**
 * Buffer a recorded action.
 *
 * Hardened: Enforces maximum buffer size with FIFO eviction.
 * Logs a warning when eviction occurs to signal potential issues.
 *
 * Idempotency: Actions with the same ID are deduplicated (no-op on second insert).
 * This makes buffering safe for replay scenarios where callbacks might fire twice.
 *
 * @returns true if action was buffered, false if it was a duplicate (already seen)
 */
export function bufferRecordedAction(sessionId: string, action: RecordedAction): boolean {
  let buffer = actionBuffers.get(sessionId);

  if (!buffer) {
    buffer = [];
    actionBuffers.set(sessionId, buffer);
    evictionCounts.set(sessionId, 0);
    seenActionIds.set(sessionId, new Set());
    lastSequenceNums.set(sessionId, -1);
  }

  // Idempotency: Check if we've already seen this action ID
  const seen = seenActionIds.get(sessionId);
  if (seen?.has(action.id)) {
    logger.debug('Recording buffer: duplicate action ignored', {
      sessionId,
      actionId: action.id,
      sequenceNum: action.sequenceNum,
      hint: 'Action already buffered, treating as idempotent retry',
    });
    return false;
  }

  // Validate sequence ordering (warn but don't reject - network timing can cause reordering)
  const lastSeq = lastSequenceNums.get(sessionId) ?? -1;
  if (action.sequenceNum <= lastSeq) {
    logger.warn('Recording buffer: out-of-order action received', {
      sessionId,
      actionId: action.id,
      sequenceNum: action.sequenceNum,
      lastSequenceNum: lastSeq,
      hint: 'Action arrived out of order, may indicate network issues',
    });
  }

  // Hardened: Enforce maximum buffer size
  if (buffer.length >= MAX_RECORDING_BUFFER_SIZE) {
    // Evict oldest action (FIFO)
    const evictedAction = buffer.shift();

    // Also remove from seen set to allow memory cleanup
    // (though unlikely to see same ID again after eviction)
    if (evictedAction) {
      seen?.delete(evictedAction.id);
    }

    // Track eviction count
    const evictions = (evictionCounts.get(sessionId) || 0) + 1;
    evictionCounts.set(sessionId, evictions);

    // Log warning on first eviction and periodically thereafter
    if (evictions === 1 || evictions % 100 === 0) {
      logger.warn('Recording buffer full, evicting old actions', {
        sessionId,
        maxSize: MAX_RECORDING_BUFFER_SIZE,
        totalEvictions: evictions,
        hint: 'Consider stopping recording or increasing buffer size',
      });
    }
  }

  // Add action to buffer and track it
  buffer.push(action);
  seen?.add(action.id);
  lastSequenceNums.set(sessionId, Math.max(lastSeq, action.sequenceNum));

  return true;
}

export function getRecordedActions(sessionId: string): RecordedAction[] {
  return actionBuffers.get(sessionId) || [];
}

export function getRecordedActionCount(sessionId: string): number {
  return getRecordedActions(sessionId).length;
}

/**
 * Get buffer statistics for monitoring.
 */
export function getBufferStats(sessionId: string): {
  actionCount: number;
  evictionCount: number;
  maxSize: number;
} {
  return {
    actionCount: getRecordedActionCount(sessionId),
    evictionCount: evictionCounts.get(sessionId) || 0,
    maxSize: MAX_RECORDING_BUFFER_SIZE,
  };
}

export function clearRecordedActions(sessionId: string): void {
  actionBuffers.set(sessionId, []);
  evictionCounts.set(sessionId, 0);
  seenActionIds.set(sessionId, new Set());
  lastSequenceNums.set(sessionId, -1);
}

export function removeRecordedActions(sessionId: string): void {
  actionBuffers.delete(sessionId);
  evictionCounts.delete(sessionId);
  seenActionIds.delete(sessionId);
  lastSequenceNums.delete(sessionId);
}

/**
 * Check if an action ID has already been buffered.
 * Useful for external code to check before attempting to buffer.
 */
export function isActionBuffered(sessionId: string, actionId: string): boolean {
  const seen = seenActionIds.get(sessionId);
  return seen?.has(actionId) ?? false;
}
