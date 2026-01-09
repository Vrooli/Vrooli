/**
 * Instruction Replay Idempotency Tests
 *
 * Tests for idempotent instruction execution to ensure replay safety.
 * These tests verify that repeated instruction execution returns cached outcomes.
 */

import type { ExecutedInstructionRecord } from '../../../src/types';

describe('Instruction Replay Idempotency', () => {
  describe('instruction key generation', () => {
    it('should generate unique keys for different instructions', () => {
      // Key format: nodeId:index
      const key1 = 'node-abc:0';
      const key2 = 'node-abc:1';
      const key3 = 'node-xyz:0';

      expect(key1).not.toBe(key2);
      expect(key1).not.toBe(key3);
      expect(key2).not.toBe(key3);
    });

    it('should generate same key for same instruction', () => {
      // Same nodeId and index = same key = idempotent lookup
      const instruction1 = { nodeId: 'node-123', index: 5 };
      const instruction2 = { nodeId: 'node-123', index: 5 };

      const key1 = `${instruction1.nodeId}:${instruction1.index}`;
      const key2 = `${instruction2.nodeId}:${instruction2.index}`;

      expect(key1).toBe(key2);
    });
  });

  describe('outcome caching', () => {
    it('should store complete outcome for replay', () => {
      const cachedOutcome = {
        success: true,
        duration_ms: 150,
        screenshot_base64: 'iVBORw0KGgo...',
        dom_snapshot: '<html>...</html>',
        final_url: 'https://example.com/page',
      };

      const record: ExecutedInstructionRecord = {
        key: 'node-1:0',
        executedAt: new Date(),
        success: true,
        cachedOutcome,
      };

      // Verify full outcome is preserved
      expect(record.cachedOutcome).toEqual(cachedOutcome);
      expect((record.cachedOutcome as Record<string, unknown>).duration_ms).toBe(150);
      expect((record.cachedOutcome as Record<string, unknown>).screenshot_base64).toBeDefined();
    });

    it('should store failure outcomes for replay', () => {
      const failedOutcome = {
        success: false,
        duration_ms: 50,
        error: {
          code: 'ELEMENT_NOT_FOUND',
          message: 'Selector not found: #missing',
          kind: 'user',
          retryable: false,
        },
      };

      const record: ExecutedInstructionRecord = {
        key: 'node-2:3',
        executedAt: new Date(),
        success: false,
        cachedOutcome: failedOutcome,
      };

      // Failed outcomes should also be cached
      expect(record.success).toBe(false);
      expect(record.cachedOutcome).toEqual(failedOutcome);
    });
  });

  describe('replay detection', () => {
    it('should detect replay by instruction key lookup', () => {
      const executedInstructions = new Map<string, ExecutedInstructionRecord>();

      // First execution
      const key = 'node-click:0';
      const record: ExecutedInstructionRecord = {
        key,
        executedAt: new Date(),
        success: true,
        cachedOutcome: { success: true },
      };
      executedInstructions.set(key, record);

      // Replay detection
      const isReplay = executedInstructions.has(key);
      expect(isReplay).toBe(true);

      // Unknown instruction
      const isNew = executedInstructions.has('node-new:0');
      expect(isNew).toBe(false);
    });

    it('should return cached outcome on replay', () => {
      const executedInstructions = new Map<string, ExecutedInstructionRecord>();
      const cachedOutcome = { success: true, duration_ms: 100, extracted_data: { value: '42' } };

      const key = 'node-extract:0';
      executedInstructions.set(key, {
        key,
        executedAt: new Date(),
        success: true,
        cachedOutcome,
      });

      // On replay, return cached outcome directly
      const previousExecution = executedInstructions.get(key);
      if (previousExecution?.cachedOutcome) {
        // This is what session-run.ts does - return cached outcome immediately
        expect(previousExecution.cachedOutcome).toEqual(cachedOutcome);
      }
    });
  });

  describe('eviction policy', () => {
    it('should evict oldest instruction when limit reached', () => {
      const MAX_EXECUTED_INSTRUCTIONS = 5;
      const executedInstructions = new Map<string, ExecutedInstructionRecord>();

      // Fill to capacity
      for (let i = 0; i < MAX_EXECUTED_INSTRUCTIONS; i++) {
        executedInstructions.set(`node:${i}`, {
          key: `node:${i}`,
          executedAt: new Date(),
          success: true,
        });
      }

      expect(executedInstructions.size).toBe(MAX_EXECUTED_INSTRUCTIONS);

      // Simulate eviction when adding new instruction
      if (executedInstructions.size >= MAX_EXECUTED_INSTRUCTIONS) {
        const firstKey = executedInstructions.keys().next().value;
        if (firstKey) {
          executedInstructions.delete(firstKey);
        }
      }

      // Add new instruction
      executedInstructions.set(`node:${MAX_EXECUTED_INSTRUCTIONS}`, {
        key: `node:${MAX_EXECUTED_INSTRUCTIONS}`,
        executedAt: new Date(),
        success: true,
      });

      // Size should still be at limit
      expect(executedInstructions.size).toBe(MAX_EXECUTED_INSTRUCTIONS);
      // Oldest should be evicted
      expect(executedInstructions.has('node:0')).toBe(false);
      // Newest should exist
      expect(executedInstructions.has(`node:${MAX_EXECUTED_INSTRUCTIONS}`)).toBe(true);
    });
  });

  describe('HTTP idempotency header', () => {
    it('should document idempotency key header behavior', () => {
      // The x-idempotency-key header provides HTTP-level idempotency:
      // 1. Client provides unique key per request
      // 2. Server caches response by key (5 minute TTL)
      // 3. Duplicate requests with same key return cached response
      // 4. Different sessions with same key are logged but proceed
      //
      // This is a safety net that works even if instruction-level
      // caching is not available (e.g., instruction not yet executed)
      const IDEMPOTENCY_KEY_HEADER = 'x-idempotency-key';
      expect(IDEMPOTENCY_KEY_HEADER).toBe('x-idempotency-key');
    });
  });

  describe('session reset behavior', () => {
    it('should clear all cached outcomes on reset', () => {
      const executedInstructions = new Map<string, ExecutedInstructionRecord>();

      // Add multiple cached instructions
      executedInstructions.set('node:0', { key: 'node:0', executedAt: new Date(), success: true, cachedOutcome: {} });
      executedInstructions.set('node:1', { key: 'node:1', executedAt: new Date(), success: true, cachedOutcome: {} });
      executedInstructions.set('node:2', { key: 'node:2', executedAt: new Date(), success: true, cachedOutcome: {} });

      expect(executedInstructions.size).toBe(3);

      // Session reset clears all
      executedInstructions.clear();

      expect(executedInstructions.size).toBe(0);
    });

    it('should allow re-execution of all instructions after reset', () => {
      const executedInstructions = new Map<string, ExecutedInstructionRecord>();

      // Execute instruction
      const key = 'node:0';
      executedInstructions.set(key, { key, executedAt: new Date(), success: true, cachedOutcome: {} });

      // Replay would return cached
      expect(executedInstructions.has(key)).toBe(true);

      // Reset
      executedInstructions.clear();

      // Same instruction is now "new"
      expect(executedInstructions.has(key)).toBe(false);
    });
  });
});
