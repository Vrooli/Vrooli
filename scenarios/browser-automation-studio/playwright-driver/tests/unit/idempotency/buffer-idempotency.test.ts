/**
 * Recording Buffer Idempotency Tests
 *
 * Tests for idempotent action buffering to ensure replay safety.
 * These tests verify that duplicate actions are handled correctly.
 */

import {
  initRecordingBuffer,
  bufferRecordedAction,
  getRecordedActions,
  clearRecordedActions,
  removeRecordedActions,
  isActionBuffered,
  getBufferStats,
} from '../../../src/recording/buffer';
import type { RecordedAction } from '../../../src/recording/types';

// Helper to create a test action
function createTestAction(overrides: Partial<RecordedAction> = {}): RecordedAction {
  return {
    id: `action-${Math.random().toString(36).slice(2)}`,
    sessionId: 'test-session',
    sequenceNum: 0,
    timestamp: new Date().toISOString(),
    actionType: 'click',
    confidence: 0.9,
    url: 'https://example.com',
    selector: {
      primary: 'button#test',
      candidates: [],
    },
    ...overrides,
  };
}

describe('Recording Buffer Idempotency', () => {
  const sessionId = 'idempotency-test-session';

  beforeEach(() => {
    initRecordingBuffer(sessionId);
  });

  afterEach(() => {
    removeRecordedActions(sessionId);
  });

  describe('bufferRecordedAction', () => {
    it('should buffer new action and return true', () => {
      const action = createTestAction({ id: 'unique-action-1', sequenceNum: 0 });

      const result = bufferRecordedAction(sessionId, action);

      expect(result).toBe(true);
      expect(getRecordedActions(sessionId)).toHaveLength(1);
    });

    it('should reject duplicate action ID and return false', () => {
      const action = createTestAction({ id: 'duplicate-action', sequenceNum: 0 });

      // Buffer first time
      const result1 = bufferRecordedAction(sessionId, action);
      expect(result1).toBe(true);

      // Buffer same action again - should be rejected
      const result2 = bufferRecordedAction(sessionId, action);
      expect(result2).toBe(false);

      // Buffer should still have only one action
      expect(getRecordedActions(sessionId)).toHaveLength(1);
    });

    it('should handle multiple buffering of same action ID gracefully', () => {
      const action = createTestAction({ id: 'multi-duplicate', sequenceNum: 0 });

      // Buffer multiple times
      bufferRecordedAction(sessionId, action);
      bufferRecordedAction(sessionId, action);
      bufferRecordedAction(sessionId, action);
      bufferRecordedAction(sessionId, action);

      // Only one action should be in buffer
      expect(getRecordedActions(sessionId)).toHaveLength(1);
    });

    it('should allow different action IDs with same sequence number', () => {
      const action1 = createTestAction({ id: 'action-a', sequenceNum: 0 });
      const action2 = createTestAction({ id: 'action-b', sequenceNum: 0 });

      bufferRecordedAction(sessionId, action1);
      bufferRecordedAction(sessionId, action2);

      expect(getRecordedActions(sessionId)).toHaveLength(2);
    });

    it('should buffer actions in order regardless of sequence numbers', () => {
      const action1 = createTestAction({ id: 'action-1', sequenceNum: 2 });
      const action2 = createTestAction({ id: 'action-2', sequenceNum: 1 });
      const action3 = createTestAction({ id: 'action-3', sequenceNum: 0 });

      bufferRecordedAction(sessionId, action1);
      bufferRecordedAction(sessionId, action2);
      bufferRecordedAction(sessionId, action3);

      const actions = getRecordedActions(sessionId);
      expect(actions).toHaveLength(3);
      // Actions should be in insertion order, not sequence order
      expect(actions[0].id).toBe('action-1');
      expect(actions[1].id).toBe('action-2');
      expect(actions[2].id).toBe('action-3');
    });

    it('should create new buffer if session buffer does not exist', () => {
      const newSessionId = 'new-session-for-test';
      const action = createTestAction({ sessionId: newSessionId, id: 'new-action' });

      const result = bufferRecordedAction(newSessionId, action);

      expect(result).toBe(true);
      expect(getRecordedActions(newSessionId)).toHaveLength(1);

      // Cleanup
      removeRecordedActions(newSessionId);
    });
  });

  describe('isActionBuffered', () => {
    it('should return true for buffered action ID', () => {
      const action = createTestAction({ id: 'check-buffered' });
      bufferRecordedAction(sessionId, action);

      expect(isActionBuffered(sessionId, 'check-buffered')).toBe(true);
    });

    it('should return false for non-buffered action ID', () => {
      expect(isActionBuffered(sessionId, 'never-buffered')).toBe(false);
    });

    it('should return false for non-existent session', () => {
      expect(isActionBuffered('non-existent-session', 'some-action')).toBe(false);
    });
  });

  describe('clearRecordedActions', () => {
    it('should clear action tracking state', () => {
      const action1 = createTestAction({ id: 'to-clear-1', sequenceNum: 0 });
      const action2 = createTestAction({ id: 'to-clear-2', sequenceNum: 1 });

      bufferRecordedAction(sessionId, action1);
      bufferRecordedAction(sessionId, action2);

      expect(getRecordedActions(sessionId)).toHaveLength(2);
      expect(isActionBuffered(sessionId, 'to-clear-1')).toBe(true);

      clearRecordedActions(sessionId);

      expect(getRecordedActions(sessionId)).toHaveLength(0);
      expect(isActionBuffered(sessionId, 'to-clear-1')).toBe(false);
    });

    it('should allow rebuffering after clear', () => {
      const action = createTestAction({ id: 'rebuffer-test', sequenceNum: 0 });

      bufferRecordedAction(sessionId, action);
      clearRecordedActions(sessionId);

      // Should be able to buffer same action again after clear
      const result = bufferRecordedAction(sessionId, action);
      expect(result).toBe(true);
      expect(getRecordedActions(sessionId)).toHaveLength(1);
    });
  });

  describe('sequence number validation', () => {
    it('should warn but still accept out-of-order sequence numbers', () => {
      // Actions arriving out of order (network reordering)
      const action5 = createTestAction({ id: 'seq-5', sequenceNum: 5 });
      const action3 = createTestAction({ id: 'seq-3', sequenceNum: 3 });
      const action7 = createTestAction({ id: 'seq-7', sequenceNum: 7 });

      // All should be buffered despite being out of order
      expect(bufferRecordedAction(sessionId, action5)).toBe(true);
      expect(bufferRecordedAction(sessionId, action3)).toBe(true);
      expect(bufferRecordedAction(sessionId, action7)).toBe(true);

      expect(getRecordedActions(sessionId)).toHaveLength(3);
    });
  });

  describe('buffer stats', () => {
    it('should track buffer size correctly with deduplication', () => {
      const action1 = createTestAction({ id: 'stats-1', sequenceNum: 0 });
      const action2 = createTestAction({ id: 'stats-2', sequenceNum: 1 });

      bufferRecordedAction(sessionId, action1);
      bufferRecordedAction(sessionId, action1); // Duplicate - should not increase count
      bufferRecordedAction(sessionId, action2);

      const stats = getBufferStats(sessionId);
      expect(stats.actionCount).toBe(2);
    });
  });

  describe('replay safety', () => {
    it('should handle rapid replay of same action safely', () => {
      const action = createTestAction({ id: 'rapid-replay', sequenceNum: 0 });

      // Simulate rapid retries (e.g., network issues causing multiple sends)
      const results = Array(10).fill(null).map(() =>
        bufferRecordedAction(sessionId, action)
      );

      // Only first should succeed
      expect(results[0]).toBe(true);
      expect(results.slice(1).every(r => r === false)).toBe(true);

      // Only one action in buffer
      expect(getRecordedActions(sessionId)).toHaveLength(1);
    });

    it('should maintain consistency after multiple operations', () => {
      const actions = Array(100).fill(null).map((_, i) =>
        createTestAction({ id: `consistency-${i}`, sequenceNum: i })
      );

      // Buffer all actions
      actions.forEach(action => bufferRecordedAction(sessionId, action));

      // Try to buffer all again - should all fail
      const replayResults = actions.map(action =>
        bufferRecordedAction(sessionId, action)
      );

      expect(replayResults.every(r => r === false)).toBe(true);
      expect(getRecordedActions(sessionId)).toHaveLength(100);
    });
  });
});
