import { describe, it, expect } from 'vitest';
import {
  mergeConsecutiveActions,
  getMergeDescription,
  type MergedActionMeta,
} from './ActionMergeService';
import type { RecordedAction, SelectorSet } from '../types/types';

/**
 * Helper to create a minimal RecordedAction for testing.
 */
function createAction(
  overrides: Partial<RecordedAction> & { id: string; actionType: RecordedAction['actionType'] }
): RecordedAction {
  return {
    sessionId: 'test-session',
    sequenceNum: 1,
    timestamp: new Date().toISOString(),
    confidence: 1.0,
    url: 'https://example.com',
    ...overrides,
  };
}

/**
 * Helper to create a selector set.
 */
function createSelector(primary: string): SelectorSet {
  return {
    primary,
    candidates: [{ selector: primary, type: 'css', score: 1.0 }],
  };
}

describe('ActionMergeService', () => {
  describe('mergeConsecutiveActions', () => {
    describe('empty and single action cases', () => {
      it('returns empty array for empty input', () => {
        const result = mergeConsecutiveActions([]);
        expect(result).toEqual([]);
      });

      it('returns single action unchanged (without mutation)', () => {
        const action = createAction({
          id: '1',
          actionType: 'click',
        });
        const result = mergeConsecutiveActions([action]);

        expect(result).toHaveLength(1);
        expect(result[0]).toEqual(action);
        // Verify no mutation
        expect(result[0]).not.toBe(action);
      });

      it('preserves all action properties', () => {
        const action = createAction({
          id: '1',
          actionType: 'click',
          selector: createSelector('#button'),
          payload: { clientX: 100, clientY: 200 },
          durationMs: 50,
        });
        const result = mergeConsecutiveActions([action]);

        expect(result[0]?.selector).toEqual(action.selector);
        expect(result[0]?.payload).toEqual(action.payload);
        expect(result[0]?.durationMs).toBe(50);
      });
    });

    describe('Rule 1: Focus Removal', () => {
      it('removes focus when followed by input on same element', () => {
        const selector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'focus', selector }),
          createAction({ id: '2', actionType: 'input', selector, payload: { text: 'a' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.actionType).toBe('input');
        expect(result[0]?.id).toBe('2');
      });

      it('keeps focus when followed by input on different element', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'focus', selector: createSelector('#input1') }),
          createAction({ id: '2', actionType: 'input', selector: createSelector('#input2'), payload: { text: 'a' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(2);
        expect(result[0]?.actionType).toBe('focus');
        expect(result[1]?.actionType).toBe('input');
      });

      it('keeps focus when followed by non-input action', () => {
        const selector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'focus', selector }),
          createAction({ id: '2', actionType: 'click', selector }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(2);
        expect(result[0]?.actionType).toBe('focus');
        expect(result[1]?.actionType).toBe('click');
      });

      it('keeps focus when it is the last action', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'click' }),
          createAction({ id: '2', actionType: 'focus', selector: createSelector('#input') }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(2);
        expect(result[1]?.actionType).toBe('focus');
      });

      it('keeps focus when input has no selector', () => {
        const selector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'focus', selector }),
          createAction({ id: '2', actionType: 'input', payload: { text: 'a' } }), // No selector
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(2);
      });
    });

    describe('Rule 2: Input Merge', () => {
      it('merges consecutive inputs on same element', () => {
        const selector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', selector, payload: { text: 'h' } }),
          createAction({ id: '2', actionType: 'input', selector, payload: { text: 'e' } }),
          createAction({ id: '3', actionType: 'input', selector, payload: { text: 'l' } }),
          createAction({ id: '4', actionType: 'input', selector, payload: { text: 'l' } }),
          createAction({ id: '5', actionType: 'input', selector, payload: { text: 'o' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.text).toBe('hello');
        expect(result[0]?._merged?.mergedCount).toBe(5);
        expect(result[0]?._merged?.mergedIds).toEqual(['1', '2', '3', '4', '5']);
        expect(result[0]?._merged?.mergeType).toBe('input');
      });

      it('does not merge inputs on different elements', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', selector: createSelector('#input1'), payload: { text: 'a' } }),
          createAction({ id: '2', actionType: 'input', selector: createSelector('#input2'), payload: { text: 'b' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(2);
        expect(result[0]?.payload?.text).toBe('a');
        expect(result[1]?.payload?.text).toBe('b');
      });

      it('stops merging when different action type is encountered', () => {
        const selector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', selector, payload: { text: 'a' } }),
          createAction({ id: '2', actionType: 'input', selector, payload: { text: 'b' } }),
          createAction({ id: '3', actionType: 'click', selector }),
          createAction({ id: '4', actionType: 'input', selector, payload: { text: 'c' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(3);
        expect(result[0]?.payload?.text).toBe('ab');
        expect(result[1]?.actionType).toBe('click');
        expect(result[2]?.payload?.text).toBe('c');
      });

      it('handles empty text in payloads', () => {
        const selector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', selector, payload: { text: 'a' } }),
          createAction({ id: '2', actionType: 'input', selector, payload: {} }),
          createAction({ id: '3', actionType: 'input', selector, payload: { text: 'b' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.text).toBe('ab');
      });

      it('handles input without selector (edge case)', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', payload: { text: 'a' } }),
          createAction({ id: '2', actionType: 'input', payload: { text: 'b' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        // Without selectors, inputs cannot be matched and won't merge
        expect(result).toHaveLength(2);
      });
    });

    describe('Rule 3: Scroll Merge', () => {
      it('merges consecutive scrolls into final position', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'scroll', payload: { scrollX: 0, scrollY: 100, deltaY: 100 } }),
          createAction({ id: '2', actionType: 'scroll', payload: { scrollX: 0, scrollY: 200, deltaY: 100 } }),
          createAction({ id: '3', actionType: 'scroll', payload: { scrollX: 0, scrollY: 300, deltaY: 100 } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.scrollY).toBe(300);
        expect(result[0]?.payload?.deltaY).toBe(300);
        expect(result[0]?._merged?.mergedCount).toBe(3);
        expect(result[0]?._merged?.mergeType).toBe('scroll');
      });

      it('accumulates both X and Y deltas', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'scroll', payload: { scrollX: 50, scrollY: 100, deltaX: 50, deltaY: 100 } }),
          createAction({ id: '2', actionType: 'scroll', payload: { scrollX: 100, scrollY: 200, deltaX: 50, deltaY: 100 } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.scrollX).toBe(100);
        expect(result[0]?.payload?.scrollY).toBe(200);
        expect(result[0]?.payload?.deltaX).toBe(100);
        expect(result[0]?.payload?.deltaY).toBe(200);
      });

      it('stops merging at non-scroll action', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'scroll', payload: { scrollY: 100, deltaY: 100 } }),
          createAction({ id: '2', actionType: 'scroll', payload: { scrollY: 200, deltaY: 100 } }),
          createAction({ id: '3', actionType: 'click' }),
          createAction({ id: '4', actionType: 'scroll', payload: { scrollY: 50, deltaY: 50 } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(3);
        expect(result[0]?.payload?.scrollY).toBe(200);
        expect(result[0]?._merged?.mergedCount).toBe(2);
        expect(result[1]?.actionType).toBe('click');
        expect(result[2]?.payload?.scrollY).toBe(50);
      });

      it('handles missing delta values', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'scroll', payload: { scrollY: 100 } }),
          createAction({ id: '2', actionType: 'scroll', payload: { scrollY: 200, deltaY: 100 } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.deltaY).toBe(100);
      });
    });

    describe('Rule 4: Navigate Merge', () => {
      it('merges redirect chain to final URL', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'navigate', payload: { targetUrl: 'https://example.com/login' } }),
          createAction({ id: '2', actionType: 'navigate', payload: { targetUrl: 'https://oauth.example.com/auth' } }),
          createAction({ id: '3', actionType: 'navigate', payload: { targetUrl: 'https://example.com/dashboard' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.targetUrl).toBe('https://example.com/dashboard');
        expect(result[0]?.url).toBe('https://example.com/dashboard');
        expect(result[0]?._merged?.mergedCount).toBe(3);
        expect(result[0]?._merged?.mergeType).toBe('navigate');
      });

      it('stops merging at non-navigate action', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'navigate', payload: { targetUrl: 'https://page1.com' } }),
          createAction({ id: '2', actionType: 'navigate', payload: { targetUrl: 'https://page2.com' } }),
          createAction({ id: '3', actionType: 'click' }),
          createAction({ id: '4', actionType: 'navigate', payload: { targetUrl: 'https://page3.com' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(3);
        expect(result[0]?.payload?.targetUrl).toBe('https://page2.com');
        expect(result[1]?.actionType).toBe('click');
        expect(result[2]?.payload?.targetUrl).toBe('https://page3.com');
      });

      it('handles navigate without targetUrl', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'navigate', url: 'https://page1.com', payload: {} }),
          createAction({ id: '2', actionType: 'navigate', payload: { targetUrl: 'https://page2.com' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.targetUrl).toBe('https://page2.com');
      });
    });

    describe('complex sequences', () => {
      it('handles real-world sequence: focus + type + click + navigate', () => {
        const inputSelector = createSelector('#search');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'navigate', payload: { targetUrl: 'https://search.com' } }),
          createAction({ id: '2', actionType: 'focus', selector: inputSelector }),
          createAction({ id: '3', actionType: 'input', selector: inputSelector, payload: { text: 'q' } }),
          createAction({ id: '4', actionType: 'input', selector: inputSelector, payload: { text: 'u' } }),
          createAction({ id: '5', actionType: 'input', selector: inputSelector, payload: { text: 'e' } }),
          createAction({ id: '6', actionType: 'input', selector: inputSelector, payload: { text: 'r' } }),
          createAction({ id: '7', actionType: 'input', selector: inputSelector, payload: { text: 'y' } }),
          createAction({ id: '8', actionType: 'click', selector: createSelector('#submit') }),
          createAction({ id: '9', actionType: 'navigate', payload: { targetUrl: 'https://search.com/results' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        // Expected: navigate, input (merged), click, navigate
        expect(result).toHaveLength(4);
        expect(result[0]?.actionType).toBe('navigate');
        expect(result[1]?.actionType).toBe('input');
        expect(result[1]?.payload?.text).toBe('query');
        expect(result[2]?.actionType).toBe('click');
        expect(result[3]?.actionType).toBe('navigate');
      });

      it('handles sequence with scrolls between inputs', () => {
        const inputSelector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', selector: inputSelector, payload: { text: 'a' } }),
          createAction({ id: '2', actionType: 'scroll', payload: { scrollY: 100, deltaY: 100 } }),
          createAction({ id: '3', actionType: 'scroll', payload: { scrollY: 200, deltaY: 100 } }),
          createAction({ id: '4', actionType: 'input', selector: inputSelector, payload: { text: 'b' } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(3);
        expect(result[0]?.payload?.text).toBe('a');
        expect(result[1]?.payload?.scrollY).toBe(200);
        expect(result[1]?._merged?.mergedCount).toBe(2);
        expect(result[2]?.payload?.text).toBe('b');
      });

      it('preserves unrelated action types', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'click', selector: createSelector('#btn1') }),
          createAction({ id: '2', actionType: 'hover' as RecordedAction['actionType'], selector: createSelector('#menu') }),
          createAction({ id: '3', actionType: 'click', selector: createSelector('#btn2') }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(3);
        expect(result[0]?.actionType).toBe('click');
        expect(result[1]?.actionType).toBe('hover');
        expect(result[2]?.actionType).toBe('click');
      });
    });

    describe('edge cases', () => {
      it('does not mutate original actions', () => {
        const original: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', selector: createSelector('#input'), payload: { text: 'a' } }),
          createAction({ id: '2', actionType: 'input', selector: createSelector('#input'), payload: { text: 'b' } }),
        ];
        const originalJson = JSON.stringify(original);

        mergeConsecutiveActions(original);

        expect(JSON.stringify(original)).toBe(originalJson);
      });

      it('handles actions with undefined selector', () => {
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'navigate', payload: { targetUrl: 'https://example.com' } }),
          createAction({ id: '2', actionType: 'click' }), // No selector
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(2);
      });

      it('handles numeric text values', () => {
        const selector = createSelector('#input');
        const actions: RecordedAction[] = [
          createAction({ id: '1', actionType: 'input', selector, payload: { text: 1 as unknown as string } }),
          createAction({ id: '2', actionType: 'input', selector, payload: { text: 2 as unknown as string } }),
        ];

        const result = mergeConsecutiveActions(actions);

        expect(result).toHaveLength(1);
        expect(result[0]?.payload?.text).toBe('12');
      });
    });
  });

  describe('getMergeDescription', () => {
    it('returns null for undefined meta', () => {
      expect(getMergeDescription(undefined)).toBeNull();
    });

    it('returns null for single action', () => {
      const meta: MergedActionMeta = {
        mergedCount: 1,
        mergedIds: ['1'],
        mergeType: 'input',
      };
      expect(getMergeDescription(meta)).toBeNull();
    });

    it('returns input merge description', () => {
      const meta: MergedActionMeta = {
        mergedCount: 5,
        mergedIds: ['1', '2', '3', '4', '5'],
        mergeType: 'input',
      };
      expect(getMergeDescription(meta)).toBe('Merged 5 keystrokes');
    });

    it('returns scroll merge description', () => {
      const meta: MergedActionMeta = {
        mergedCount: 10,
        mergedIds: [],
        mergeType: 'scroll',
      };
      expect(getMergeDescription(meta)).toBe('Merged 10 scroll events');
    });

    it('returns navigate merge description', () => {
      const meta: MergedActionMeta = {
        mergedCount: 3,
        mergedIds: [],
        mergeType: 'navigate',
      };
      expect(getMergeDescription(meta)).toBe('Merged 3 navigation events');
    });

    it('returns focus-removed description', () => {
      const meta: MergedActionMeta = {
        mergedCount: 2,
        mergedIds: [],
        mergeType: 'focus-removed',
      };
      expect(getMergeDescription(meta)).toBe('Focus event removed (implicit)');
    });

    it('returns generic description for null merge type', () => {
      const meta: MergedActionMeta = {
        mergedCount: 4,
        mergedIds: [],
        mergeType: null,
      };
      expect(getMergeDescription(meta)).toBe('Merged 4 actions');
    });
  });
});
