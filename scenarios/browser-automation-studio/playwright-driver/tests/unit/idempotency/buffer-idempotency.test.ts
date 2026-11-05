/**
 * Recording Buffer Idempotency Tests
 *
 * Tests for idempotent action buffering to ensure replay safety.
 * These tests verify that duplicate actions are handled correctly.
 */

import {
  bufferTimelineEntry,
  acknowledgeTimelineEntries,
  assertRecordingAcknowledged,
  deliverTimelineEntry,
  flushRecordingDeliveries,
  getBufferStats,
  getTimelineEntries,
  initRecordingBuffer,
  isEntryBuffered,
  removeRecordingBuffer,
} from '../../../src/recording/io/buffer';
import { TimelineEntrySchema, type TimelineEntry } from '../../../src/recording/types';
import { MAX_RECORDING_BUFFER_SIZE } from '../../../src/utils';
import { create } from '@bufbuild/protobuf';

function createTestEntry(overrides: Partial<TimelineEntry> = {}): TimelineEntry {
  return create(TimelineEntrySchema, {
    id: `entry-${Math.random().toString(36).slice(2)}`,
    sequenceNum: 0,
    ...overrides,
  });
}

describe('Recording Buffer Idempotency', () => {
  const sessionId = 'idempotency-test-session';

  beforeEach(() => {
    initRecordingBuffer(sessionId);
  });

  afterEach(() => {
    acknowledgeTimelineEntries(sessionId, getTimelineEntries(sessionId).map((entry) => entry.id));
    removeRecordingBuffer(sessionId);
  });

  describe('bufferTimelineEntry', () => {
    it('buffers new entry and returns true', () => {
      const entry = createTestEntry({ id: 'unique-entry-1', sequenceNum: 0 });

      const result = bufferTimelineEntry(sessionId, entry);

      expect(result).toBe(true);
      expect(getTimelineEntries(sessionId)).toHaveLength(1);
    });

    it('rejects duplicate entry ID and returns false', () => {
      const entry = createTestEntry({ id: 'duplicate-entry', sequenceNum: 0 });

      // Buffer first time
      const result1 = bufferTimelineEntry(sessionId, entry);
      expect(result1).toBe(true);

      // Buffer same entry again - should be rejected
      const result2 = bufferTimelineEntry(sessionId, entry);
      expect(result2).toBe(false);

      // Buffer should still have only one entry
      expect(getTimelineEntries(sessionId)).toHaveLength(1);
    });

    it('handles multiple buffering of same entry ID gracefully', () => {
      const entry = createTestEntry({ id: 'multi-duplicate', sequenceNum: 0 });

      // Buffer multiple times
      bufferTimelineEntry(sessionId, entry);
      bufferTimelineEntry(sessionId, entry);
      bufferTimelineEntry(sessionId, entry);
      bufferTimelineEntry(sessionId, entry);

      // Only one entry should be in buffer
      expect(getTimelineEntries(sessionId)).toHaveLength(1);
    });

    it('allows different entry IDs with same sequence number', () => {
      const entry1 = createTestEntry({ id: 'entry-a', sequenceNum: 0 });
      const entry2 = createTestEntry({ id: 'entry-b', sequenceNum: 0 });

      bufferTimelineEntry(sessionId, entry1);
      bufferTimelineEntry(sessionId, entry2);

      expect(getTimelineEntries(sessionId)).toHaveLength(2);
    });

    it('buffers entries in insertion order regardless of sequence numbers', () => {
      const entry1 = createTestEntry({ id: 'entry-1', sequenceNum: 2 });
      const entry2 = createTestEntry({ id: 'entry-2', sequenceNum: 1 });
      const entry3 = createTestEntry({ id: 'entry-3', sequenceNum: 0 });

      bufferTimelineEntry(sessionId, entry1);
      bufferTimelineEntry(sessionId, entry2);
      bufferTimelineEntry(sessionId, entry3);

      const entries = getTimelineEntries(sessionId);
      expect(entries).toHaveLength(3);
      const [firstEntry, secondEntry, thirdEntry] = entries;
      if (!firstEntry || !secondEntry || !thirdEntry) {
        throw new Error('Expected three buffered entries');
      }
      expect(firstEntry.id).toBe('entry-1');
      expect(secondEntry.id).toBe('entry-2');
      expect(thirdEntry.id).toBe('entry-3');
    });

    it('creates new buffer if session buffer does not exist', () => {
      const newSessionId = 'new-session-for-test';
      const entry = createTestEntry({ id: 'new-entry' });

      const result = bufferTimelineEntry(newSessionId, entry);

      expect(result).toBe(true);
      expect(getTimelineEntries(newSessionId)).toHaveLength(1);

      // Cleanup
      acknowledgeTimelineEntries(newSessionId, [entry.id]);
      removeRecordingBuffer(newSessionId);
    });
  });

  describe('isEntryBuffered', () => {
    it('returns true for buffered entry ID', () => {
      const entry = createTestEntry({ id: 'check-buffered' });
      bufferTimelineEntry(sessionId, entry);

      expect(isEntryBuffered(sessionId, 'check-buffered')).toBe(true);
    });

    it('returns false for non-buffered entry ID', () => {
      expect(isEntryBuffered(sessionId, 'never-buffered')).toBe(false);
    });

    it('returns false for non-existent session', () => {
      expect(isEntryBuffered('non-existent-session', 'some-entry')).toBe(false);
    });
  });

  describe('acknowledgement and capacity', () => {
    it('refuses replacement and removal while observations are pending', () => {
      const entry = createTestEntry({ id: 'pending' });
      bufferTimelineEntry(sessionId, entry);
      expect(() => initRecordingBuffer(sessionId)).toThrow('pending');
      expect(() => removeRecordingBuffer(sessionId)).toThrow('pending');
      expect(getTimelineEntries(sessionId)).toEqual([entry]);
    });

    it('preserves replay identity when committed entries are cleared', () => {
      const entry = createTestEntry({ id: 'committed' });
      bufferTimelineEntry(sessionId, entry);
      acknowledgeTimelineEntries(sessionId, [entry.id], true);
      expect(getTimelineEntries(sessionId)).toEqual([]);
      expect(bufferTimelineEntry(sessionId, entry)).toBe(false);
      expect(isEntryBuffered(sessionId, entry.id)).toBe(true);
      expect(getTimelineEntries(sessionId)).toEqual([]);
      expect(() => bufferTimelineEntry(sessionId, createTestEntry({ id: entry.id, sequenceNum: 99 }))).toThrow('conflicts');
    });

    it('rejects an unknown acknowledgement without clearing any pending observation', () => {
      const entry = createTestEntry({ id: 'pending' });
      bufferTimelineEntry(sessionId, entry);
      expect(() => acknowledgeTimelineEntries(sessionId, [entry.id, 'unknown'], true)).toThrow('Unknown');
      expect(getBufferStats(sessionId).pendingCount).toBe(1);
      expect(getTimelineEntries(sessionId)).toEqual([entry]);
    });

    it('never evicts pending entries at capacity; committed entries may make room', () => {
      for (let i = 0; i < MAX_RECORDING_BUFFER_SIZE; i++) bufferTimelineEntry(sessionId, createTestEntry({ id: `bounded-${i}` }));
      const next = createTestEntry({ id: 'next' });
      expect(() => bufferTimelineEntry(sessionId, next)).toThrow('capacity');
      expect(getBufferStats(sessionId).entryCount).toBe(MAX_RECORDING_BUFFER_SIZE);
      expect(isEntryBuffered(sessionId, 'bounded-0')).toBe(true);
      acknowledgeTimelineEntries(sessionId, ['bounded-0']);
      expect(bufferTimelineEntry(sessionId, next)).toBe(true);
      expect(getBufferStats(sessionId).entryCount).toBe(MAX_RECORDING_BUFFER_SIZE);
      expect(isEntryBuffered(sessionId, 'bounded-1')).toBe(true);
    });

    it('joins concurrent delivery and retries a failed head before later entries', async () => {
      const first = createTestEntry({ id: 'first' });
      const second = createTestEntry({ id: 'second' });
      bufferTimelineEntry(sessionId, first);
      bufferTimelineEntry(sessionId, second);
      let release!: () => void;
      const held = new Promise<void>((resolve) => { release = resolve; });
      let fail = true;
      const calls: string[] = [];
      const deliverFirst = async () => { calls.push('first'); await held; if (fail) throw new Error('disk full'); };
      const a = deliverTimelineEntry(sessionId, first, deliverFirst);
      const duplicate = deliverTimelineEntry(sessionId, first, deliverFirst);
      const b = deliverTimelineEntry(sessionId, second, async () => { calls.push('second'); });
      const outcomes = Promise.allSettled([a, duplicate, b]);
      expect(calls).toEqual(['first']);
      expect(() => assertRecordingAcknowledged(sessionId)).toThrow('pending');
      release();
      expect((await outcomes).every((result) => result.status === 'rejected')).toBe(true);
      expect(calls).toEqual(['first']);
      fail = false;
      await flushRecordingDeliveries(sessionId);
      expect(calls).toEqual(['first', 'first', 'second']);
      expect(() => assertRecordingAcknowledged(sessionId)).not.toThrow();
    });

    it('retains pull-mode entries until the consumer acknowledges its commit', async () => {
      const entry = createTestEntry({ id: 'pull' });
      bufferTimelineEntry(sessionId, entry);
      await deliverTimelineEntry(sessionId, entry, async () => {}, false);
      expect(() => removeRecordingBuffer(sessionId)).toThrow('pending');
      acknowledgeTimelineEntries(sessionId, [entry.id], true);
      expect(() => assertRecordingAcknowledged(sessionId)).not.toThrow();
    });
  });

  describe('sequence number validation', () => {
    it('should warn but still accept out-of-order sequence numbers', () => {
      // Actions arriving out of order (network reordering)
      const entry5 = createTestEntry({ id: 'seq-5', sequenceNum: 5 });
      const entry3 = createTestEntry({ id: 'seq-3', sequenceNum: 3 });
      const entry7 = createTestEntry({ id: 'seq-7', sequenceNum: 7 });

      // All should be buffered despite being out of order
      expect(bufferTimelineEntry(sessionId, entry5)).toBe(true);
      expect(bufferTimelineEntry(sessionId, entry3)).toBe(true);
      expect(bufferTimelineEntry(sessionId, entry7)).toBe(true);

      expect(getTimelineEntries(sessionId)).toHaveLength(3);
    });
  });

  describe('buffer stats', () => {
    it('should track buffer size correctly with deduplication', () => {
      const entry1 = createTestEntry({ id: 'stats-1', sequenceNum: 0 });
      const entry2 = createTestEntry({ id: 'stats-2', sequenceNum: 1 });

      bufferTimelineEntry(sessionId, entry1);
      bufferTimelineEntry(sessionId, entry1); // Duplicate - should not increase count
      bufferTimelineEntry(sessionId, entry2);

      const stats = getBufferStats(sessionId);
      expect(stats.entryCount).toBe(2);
    });
  });

  describe('replay safety', () => {
    it('should handle rapid replay of same action safely', () => {
      const entry = createTestEntry({ id: 'rapid-replay', sequenceNum: 0 });

      // Simulate rapid retries (e.g., network issues causing multiple sends)
      const results = Array(10).fill(null).map(() =>
        bufferTimelineEntry(sessionId, entry)
      );

      // Only first should succeed
      expect(results[0]).toBe(true);
      expect(results.slice(1).every(r => r === false)).toBe(true);

      // Only one entry in buffer
      expect(getTimelineEntries(sessionId)).toHaveLength(1);
    });

    it('should maintain consistency after multiple operations', () => {
      const entries = Array(100).fill(null).map((_, i) =>
        createTestEntry({ id: `consistency-${i}`, sequenceNum: i })
      );

      // Buffer all actions
      entries.forEach((entry) => bufferTimelineEntry(sessionId, entry));

      // Try to buffer all again - should all fail
      const replayResults = entries.map((entry) =>
        bufferTimelineEntry(sessionId, entry)
      );

      expect(replayResults.every(r => r === false)).toBe(true);
      expect(getTimelineEntries(sessionId)).toHaveLength(100);
    });
  });
});
