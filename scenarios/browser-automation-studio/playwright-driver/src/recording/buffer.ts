import type { RecordedAction } from './types';
import { MAX_RECORDING_BUFFER_SIZE, logger } from '../utils';

/**
 * Recording Buffer
 *
 * In-memory action buffers keyed by session ID for record-mode endpoints.
 *
 * Hardened assumptions:
 * - Buffers have a maximum size to prevent memory exhaustion
 * - Old actions are evicted when buffer is full (FIFO)
 * - Buffer operations are O(1) or O(n) where n is bounded
 */

// In-memory action buffers keyed by session ID
const actionBuffers = new Map<string, RecordedAction[]>();

// Track eviction counts for observability
const evictionCounts = new Map<string, number>();

export function initRecordingBuffer(sessionId: string): void {
  actionBuffers.set(sessionId, []);
  evictionCounts.set(sessionId, 0);
}

/**
 * Buffer a recorded action.
 *
 * Hardened: Enforces maximum buffer size with FIFO eviction.
 * Logs a warning when eviction occurs to signal potential issues.
 */
export function bufferRecordedAction(sessionId: string, action: RecordedAction): void {
  let buffer = actionBuffers.get(sessionId);

  if (!buffer) {
    buffer = [];
    actionBuffers.set(sessionId, buffer);
    evictionCounts.set(sessionId, 0);
  }

  // Hardened: Enforce maximum buffer size
  if (buffer.length >= MAX_RECORDING_BUFFER_SIZE) {
    // Evict oldest action (FIFO)
    buffer.shift();

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

  buffer.push(action);
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
}

export function removeRecordedActions(sessionId: string): void {
  actionBuffers.delete(sessionId);
  evictionCounts.delete(sessionId);
}
