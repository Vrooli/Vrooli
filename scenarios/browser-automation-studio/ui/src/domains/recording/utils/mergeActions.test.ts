import { describe, it, expect } from 'vitest';
import { mergeConsecutiveActions, getMergeDescription, type MergedAction } from './mergeActions';
import type { RecordedAction } from '../types/types';

/**
 * Test suite for action merging utilities.
 *
 * These tests verify the core reconciliation logic that transforms raw browser
 * events into a clean, user-friendly timeline. This is critical for WYSIWYG
 * recording - what users see must match what gets saved to the workflow.
 */

// Helper to create test actions
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

describe('mergeConsecutiveActions', () => {
  describe('empty and single action cases', () => {
    it('handles empty action array', () => {
      const result = mergeConsecutiveActions([]);
      expect(result).toEqual([]);
    });

    it('handles single action unchanged', () => {
      const action = createAction({ id: '1', actionType: 'click' });
      const result = mergeConsecutiveActions([action]);

      expect(result).toHaveLength(1);
      expect(result[0].id).toBe('1');
      expect(result[0].actionType).toBe('click');
      expect(result[0]._merged).toBeUndefined();
    });
  });

  describe('focus removal', () => {
    it('removes focus events followed by input on same element', () => {
      const selector = { primary: 'input#email' };
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'focus', selector }),
        createAction({ id: '2', actionType: 'input', selector, payload: { text: 'hello' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      // Focus should be removed, only input remains
      expect(result).toHaveLength(1);
      expect(result[0].actionType).toBe('input');
      expect(result[0].id).toBe('2');
    });

    it('preserves focus events NOT followed by input', () => {
      const selector = { primary: 'input#email' };
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'focus', selector }),
        createAction({ id: '2', actionType: 'click', selector: { primary: 'button#submit' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      // Both actions should be preserved
      expect(result).toHaveLength(2);
      expect(result[0].actionType).toBe('focus');
      expect(result[1].actionType).toBe('click');
    });

    it('preserves focus events when next input is on different element', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'focus', selector: { primary: 'input#email' } }),
        createAction({ id: '2', actionType: 'input', selector: { primary: 'input#password' }, payload: { text: 'secret' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      // Both actions should be preserved (different selectors)
      expect(result).toHaveLength(2);
      expect(result[0].actionType).toBe('focus');
      expect(result[1].actionType).toBe('input');
    });

    it('preserves focus event at end of array', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'click', selector: { primary: 'button' } }),
        createAction({ id: '2', actionType: 'focus', selector: { primary: 'input#email' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(2);
      expect(result[1].actionType).toBe('focus');
    });
  });

  describe('input merging', () => {
    it('merges consecutive inputs on same selector into single action', () => {
      const selector = { primary: 'input#email' };
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'input', selector, payload: { text: 'h' } }),
        createAction({ id: '2', actionType: 'input', selector, payload: { text: 'e' } }),
        createAction({ id: '3', actionType: 'input', selector, payload: { text: 'l' } }),
        createAction({ id: '4', actionType: 'input', selector, payload: { text: 'l' } }),
        createAction({ id: '5', actionType: 'input', selector, payload: { text: 'o' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(1);
      expect(result[0].payload?.text).toBe('hello');
    });

    it('preserves merged IDs in _merged metadata', () => {
      const selector = { primary: 'input#email' };
      const actions: RecordedAction[] = [
        createAction({ id: 'a', actionType: 'input', selector, payload: { text: 'h' } }),
        createAction({ id: 'b', actionType: 'input', selector, payload: { text: 'i' } }),
      ];

      const result = mergeConsecutiveActions(actions) as MergedAction[];

      expect(result).toHaveLength(1);
      expect(result[0]._merged).toBeDefined();
      expect(result[0]._merged?.mergedCount).toBe(2);
      expect(result[0]._merged?.mergedIds).toEqual(['a', 'b']);
      expect(result[0]._merged?.mergeType).toBe('input');
    });

    it('stops merging when selector changes', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'input', selector: { primary: 'input#email' }, payload: { text: 'ab' } }),
        createAction({ id: '2', actionType: 'input', selector: { primary: 'input#email' }, payload: { text: 'c' } }),
        createAction({ id: '3', actionType: 'input', selector: { primary: 'input#password' }, payload: { text: 'xyz' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(2);
      expect(result[0].payload?.text).toBe('abc');
      expect(result[0].selector?.primary).toBe('input#email');
      expect(result[1].payload?.text).toBe('xyz');
      expect(result[1].selector?.primary).toBe('input#password');
    });

    it('stops merging when action type changes', () => {
      const selector = { primary: 'input#email' };
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'input', selector, payload: { text: 'ab' } }),
        createAction({ id: '2', actionType: 'click', selector }),
        createAction({ id: '3', actionType: 'input', selector, payload: { text: 'cd' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(3);
      expect(result[0].payload?.text).toBe('ab');
      expect(result[1].actionType).toBe('click');
      expect(result[2].payload?.text).toBe('cd');
    });

    it('handles input with empty text', () => {
      const selector = { primary: 'input#email' };
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'input', selector, payload: { text: '' } }),
        createAction({ id: '2', actionType: 'input', selector, payload: { text: 'hello' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(1);
      expect(result[0].payload?.text).toBe('hello');
    });

    it('handles input without payload', () => {
      const selector = { primary: 'input#email' };
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'input', selector }),
        createAction({ id: '2', actionType: 'input', selector, payload: { text: 'hello' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(1);
      expect(result[0].payload?.text).toBe('hello');
    });
  });

  describe('scroll merging', () => {
    it('collapses consecutive scrolls to final position', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'scroll', payload: { scrollX: 0, scrollY: 100, deltaX: 0, deltaY: 100 } }),
        createAction({ id: '2', actionType: 'scroll', payload: { scrollX: 0, scrollY: 200, deltaX: 0, deltaY: 100 } }),
        createAction({ id: '3', actionType: 'scroll', payload: { scrollX: 0, scrollY: 500, deltaX: 0, deltaY: 300 } }),
      ];

      const result = mergeConsecutiveActions(actions) as MergedAction[];

      expect(result).toHaveLength(1);
      expect(result[0].payload?.scrollY).toBe(500); // Final position
      expect(result[0].payload?.deltaY).toBe(500); // Sum of deltas
      expect(result[0]._merged?.mergedCount).toBe(3);
      expect(result[0]._merged?.mergeType).toBe('scroll');
    });

    it('preserves scroll when interrupted by other action', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'scroll', payload: { scrollY: 100 } }),
        createAction({ id: '2', actionType: 'click', selector: { primary: 'button' } }),
        createAction({ id: '3', actionType: 'scroll', payload: { scrollY: 200 } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(3);
      expect(result[0].actionType).toBe('scroll');
      expect(result[1].actionType).toBe('click');
      expect(result[2].actionType).toBe('scroll');
    });

    it('accumulates horizontal and vertical scroll deltas', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'scroll', payload: { scrollX: 50, scrollY: 100, deltaX: 50, deltaY: 100 } }),
        createAction({ id: '2', actionType: 'scroll', payload: { scrollX: 150, scrollY: 250, deltaX: 100, deltaY: 150 } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(1);
      expect(result[0].payload?.scrollX).toBe(150);
      expect(result[0].payload?.scrollY).toBe(250);
      expect(result[0].payload?.deltaX).toBe(150);
      expect(result[0].payload?.deltaY).toBe(250);
    });
  });

  describe('navigate merging', () => {
    it('collapses redirect chains to final URL', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'navigate', payload: { targetUrl: 'https://example.com/login' } }),
        createAction({ id: '2', actionType: 'navigate', payload: { targetUrl: 'https://example.com/oauth' } }),
        createAction({ id: '3', actionType: 'navigate', payload: { targetUrl: 'https://example.com/dashboard' } }),
      ];

      const result = mergeConsecutiveActions(actions) as MergedAction[];

      expect(result).toHaveLength(1);
      expect(result[0].payload?.targetUrl).toBe('https://example.com/dashboard');
      expect(result[0].url).toBe('https://example.com/dashboard');
      expect(result[0]._merged?.mergedCount).toBe(3);
      expect(result[0]._merged?.mergeType).toBe('navigate');
    });

    it('preserves navigate when interrupted by other action', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'navigate', payload: { targetUrl: 'https://a.com' } }),
        createAction({ id: '2', actionType: 'click', selector: { primary: 'button' } }),
        createAction({ id: '3', actionType: 'navigate', payload: { targetUrl: 'https://b.com' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(3);
    });
  });

  describe('complex scenarios', () => {
    it('handles mixed action types correctly', () => {
      const emailSelector = { primary: 'input#email' };
      const passwordSelector = { primary: 'input#password' };

      const actions: RecordedAction[] = [
        // Navigate and redirect
        createAction({ id: '1', actionType: 'navigate', payload: { targetUrl: 'https://login.example.com' } }),
        createAction({ id: '2', actionType: 'navigate', payload: { targetUrl: 'https://app.example.com/login' } }),
        // Focus and type email
        createAction({ id: '3', actionType: 'focus', selector: emailSelector }),
        createAction({ id: '4', actionType: 'input', selector: emailSelector, payload: { text: 't' } }),
        createAction({ id: '5', actionType: 'input', selector: emailSelector, payload: { text: 'e' } }),
        createAction({ id: '6', actionType: 'input', selector: emailSelector, payload: { text: 's' } }),
        createAction({ id: '7', actionType: 'input', selector: emailSelector, payload: { text: 't' } }),
        // Click password field and type
        createAction({ id: '8', actionType: 'click', selector: passwordSelector }),
        createAction({ id: '9', actionType: 'input', selector: passwordSelector, payload: { text: 'pass' } }),
        // Scroll and click submit
        createAction({ id: '10', actionType: 'scroll', payload: { scrollY: 100 } }),
        createAction({ id: '11', actionType: 'scroll', payload: { scrollY: 200 } }),
        createAction({ id: '12', actionType: 'click', selector: { primary: 'button#submit' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      // Expected: navigate(merged), input(merged), click, input, scroll(merged), click
      expect(result).toHaveLength(6);

      // Navigate merged
      expect(result[0].actionType).toBe('navigate');
      expect(result[0].payload?.targetUrl).toBe('https://app.example.com/login');

      // Input merged (focus removed)
      expect(result[1].actionType).toBe('input');
      expect(result[1].payload?.text).toBe('test');

      // Click password
      expect(result[2].actionType).toBe('click');

      // Input password
      expect(result[3].actionType).toBe('input');
      expect(result[3].payload?.text).toBe('pass');

      // Scroll merged
      expect(result[4].actionType).toBe('scroll');
      expect(result[4].payload?.scrollY).toBe(200);

      // Click submit
      expect(result[5].actionType).toBe('click');
    });

    it('preserves action order after merging', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'click', selector: { primary: 'a' }, sequenceNum: 1 }),
        createAction({ id: '2', actionType: 'input', selector: { primary: 'input' }, payload: { text: 'h' }, sequenceNum: 2 }),
        createAction({ id: '3', actionType: 'input', selector: { primary: 'input' }, payload: { text: 'i' }, sequenceNum: 3 }),
        createAction({ id: '4', actionType: 'click', selector: { primary: 'b' }, sequenceNum: 4 }),
      ];

      const result = mergeConsecutiveActions(actions);

      expect(result).toHaveLength(3);
      expect(result[0].sequenceNum).toBe(1);
      expect(result[1].sequenceNum).toBe(2); // Merged input keeps first sequence
      expect(result[2].sequenceNum).toBe(4);
    });
  });

  describe('edge cases', () => {
    it('handles actions without selectors', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'input', payload: { text: 'a' } }),
        createAction({ id: '2', actionType: 'input', payload: { text: 'b' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      // Without selectors, they can't be merged (no matching)
      expect(result).toHaveLength(2);
    });

    it('handles focus without selector', () => {
      const actions: RecordedAction[] = [
        createAction({ id: '1', actionType: 'focus' }),
        createAction({ id: '2', actionType: 'input', selector: { primary: 'input' }, payload: { text: 'test' } }),
      ];

      const result = mergeConsecutiveActions(actions);

      // Focus without selector can't be removed (no matching)
      expect(result).toHaveLength(2);
    });

    it('returns copies, not original references', () => {
      const originalAction = createAction({ id: '1', actionType: 'click' });
      const result = mergeConsecutiveActions([originalAction]);

      expect(result[0]).not.toBe(originalAction);
      expect(result[0]).toEqual(originalAction);
    });
  });
});

describe('getMergeDescription', () => {
  it('returns null for undefined metadata', () => {
    expect(getMergeDescription(undefined)).toBeNull();
  });

  it('returns null for single action (mergedCount <= 1)', () => {
    expect(getMergeDescription({
      mergedCount: 1,
      mergedIds: ['a'],
      mergeType: 'input',
    })).toBeNull();
  });

  it('returns correct description for input merge', () => {
    expect(getMergeDescription({
      mergedCount: 5,
      mergedIds: ['a', 'b', 'c', 'd', 'e'],
      mergeType: 'input',
    })).toBe('Merged 5 keystrokes');
  });

  it('returns correct description for scroll merge', () => {
    expect(getMergeDescription({
      mergedCount: 10,
      mergedIds: Array(10).fill('x'),
      mergeType: 'scroll',
    })).toBe('Merged 10 scroll events');
  });

  it('returns correct description for navigate merge', () => {
    expect(getMergeDescription({
      mergedCount: 3,
      mergedIds: ['a', 'b', 'c'],
      mergeType: 'navigate',
    })).toBe('Merged 3 navigation events');
  });

  it('returns correct description for focus-removed', () => {
    expect(getMergeDescription({
      mergedCount: 2,
      mergedIds: ['a', 'b'],
      mergeType: 'focus-removed',
    })).toBe('Focus event removed (implicit)');
  });

  it('returns generic description for unknown merge type', () => {
    expect(getMergeDescription({
      mergedCount: 4,
      mergedIds: ['a', 'b', 'c', 'd'],
      mergeType: null,
    })).toBe('Merged 4 actions');
  });
});
