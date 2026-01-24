/**
 * Action Merge Service
 *
 * Pure domain service for deduplicating and cleaning up raw recorded actions.
 * Extracted from utils/mergeActions.ts for better testability.
 *
 * ## Problem
 *
 * Browser recording captures low-level events that are noisy:
 * - Typing "hello" → 5 separate input events
 * - Scrolling → dozens of events per second
 * - Focus before typing → redundant (implicit in the input)
 *
 * ## Solution
 *
 * Single-pass greedy merging that:
 * 1. Removes redundant focus events (implicit in following input)
 * 2. Concatenates consecutive inputs on the same selector
 * 3. Collapses consecutive scrolls to final position
 * 4. Collapses redirect chains to final URL
 *
 * ## Design Decisions
 *
 * - **Client-side**: Instant preview as users record (WYSIWYG)
 * - **Mirrors backend**: Same logic as api/handlers/record_mode.go
 * - **Metadata preservation**: Tracks original action IDs for undo capability
 */

import type { RecordedAction, SelectorSet } from '../types/types';

/**
 * Metadata about merged actions for UI display.
 */
export interface MergedActionMeta {
  /** Number of original actions merged into this one */
  mergedCount: number;
  /** Original action IDs that were merged */
  mergedIds: string[];
  /** Type of merge applied */
  mergeType: 'input' | 'scroll' | 'navigate' | 'focus-removed' | null;
}

/**
 * RecordedAction extended with merge metadata.
 */
export type MergedAction = RecordedAction & {
  _merged?: MergedActionMeta;
};

/**
 * Check if two SelectorSets refer to the same element.
 */
function selectorsMatch(a?: SelectorSet, b?: SelectorSet): boolean {
  if (!a || !b) return false;
  return a.primary === b.primary;
}

/**
 * Merge consecutive actions to create a cleaner timeline.
 *
 * This mirrors the backend logic in api/handlers/record_mode.go:mergeConsecutiveActions
 * to ensure "what you see = what you get" in the workflow.
 *
 * @param actions - Raw recorded actions
 * @returns Merged actions with metadata about merges applied
 */
export function mergeConsecutiveActions(actions: RecordedAction[]): MergedAction[] {
  if (actions.length <= 1) {
    return actions.map((a) => ({ ...a }));
  }

  const merged: MergedAction[] = [];

  for (let i = 0; i < actions.length; i++) {
    const currentAction = actions[i];
    if (!currentAction) continue;
    const action = { ...currentAction } as MergedAction;
    const mergedIds: string[] = [action.id];

    // Rule 1: Focus Removal
    // Why: When you click an input field and type, the browser fires focus then input events.
    // The focus is implicit - including it would make workflows verbose and fragile.
    // We skip focus if the very next action is an input on the same element.
    if (action.actionType === 'focus' && i + 1 < actions.length) {
      const next = actions[i + 1];
      if (next && next.actionType === 'input' && selectorsMatch(action.selector, next.selector)) {
        continue; // Skip this focus, it's implied by the following input
      }
    }

    // Rule 2: Input Merge
    // Why: Typing "hello" creates 5 input events ('h', 'e', 'l', 'l', 'o').
    // Users expect to see a single "type 'hello'" action, not 5 separate keystrokes.
    // We concatenate consecutive inputs on the same element into one action.
    if (action.actionType === 'input' && action.selector) {
      let mergedText = '';
      if (action.payload?.text) {
        mergedText = String(action.payload.text);
      }

      // Look ahead for more input actions on same element
      while (i + 1 < actions.length) {
        const next = actions[i + 1];
        if (!next || next.actionType !== 'input' || !selectorsMatch(action.selector, next.selector)) {
          break;
        }
        // Merge the text
        if (next.payload?.text) {
          mergedText += String(next.payload.text);
        }
        mergedIds.push(next.id);
        i++; // Skip this action, we've merged it
      }

      // Update the action with merged text
      if (mergedIds.length > 1 || mergedText !== String(action.payload?.text || '')) {
        action.payload = { ...action.payload, text: mergedText };
        action._merged = {
          mergedCount: mergedIds.length,
          mergedIds,
          mergeType: 'input',
        };
      }
    }

    // Rule 3: Scroll Merge
    // Why: Scrolling fires dozens of events per second as the user drags.
    // What matters is the final position, not each intermediate step.
    // We keep only the final scroll position and sum the total deltas.
    if (action.actionType === 'scroll') {
      let finalScrollX = action.payload?.scrollX as number | undefined;
      let finalScrollY = action.payload?.scrollY as number | undefined;
      let totalDeltaX = (action.payload?.deltaX as number) || 0;
      let totalDeltaY = (action.payload?.deltaY as number) || 0;

      // Look ahead for more scroll actions
      while (i + 1 < actions.length) {
        const next = actions[i + 1];
        if (!next || next.actionType !== 'scroll') {
          break;
        }
        // Accumulate deltas and use final position
        if (next.payload?.scrollX !== undefined) {
          finalScrollX = next.payload.scrollX as number;
        }
        if (next.payload?.scrollY !== undefined) {
          finalScrollY = next.payload.scrollY as number;
        }
        totalDeltaX += (next.payload?.deltaX as number) || 0;
        totalDeltaY += (next.payload?.deltaY as number) || 0;
        mergedIds.push(next.id);
        i++; // Skip this action, we've merged it
      }

      // Update the action with final scroll position and total delta
      if (mergedIds.length > 1) {
        action.payload = {
          ...action.payload,
          scrollX: finalScrollX,
          scrollY: finalScrollY,
          deltaX: totalDeltaX,
          deltaY: totalDeltaY,
        };
        action._merged = {
          mergedCount: mergedIds.length,
          mergedIds,
          mergeType: 'scroll',
        };
      }
    }

    // Rule 4: Navigate Merge
    // Why: HTTP redirects create chains (e.g., /login → /oauth → /dashboard).
    // Users care about the final destination, not the redirect chain.
    // We collapse consecutive navigates to keep only the final URL.
    if (action.actionType === 'navigate') {
      let finalUrl = action.payload?.targetUrl as string | undefined;

      // Look ahead for more navigate actions
      while (i + 1 < actions.length) {
        const next = actions[i + 1];
        if (!next || next.actionType !== 'navigate') {
          break;
        }
        // Use the final URL from the chain
        if (next.payload?.targetUrl) {
          finalUrl = next.payload.targetUrl as string;
        }
        mergedIds.push(next.id);
        i++; // Skip this action, we've merged it
      }

      // Update the action with final URL
      if (mergedIds.length > 1) {
        action.payload = {
          ...action.payload,
          targetUrl: finalUrl,
        };
        action.url = finalUrl || action.url;
        action._merged = {
          mergedCount: mergedIds.length,
          mergedIds,
          mergeType: 'navigate',
        };
      }
    }

    // Check if the previous action was a focus that should be noted
    const prevAction = i > 0 ? actions[i - 1] : undefined;
    if (
      merged.length === 0 &&
      prevAction &&
      prevAction.actionType === 'focus' &&
      action.actionType === 'input' &&
      selectorsMatch(prevAction.selector, action.selector)
    ) {
      // This input action had a focus removed before it
      if (!action._merged) {
        action._merged = {
          mergedCount: 1,
          mergedIds: [action.id],
          mergeType: 'focus-removed',
        };
      }
    }

    merged.push(action);
  }

  return merged;
}

/**
 * Get a human-readable description of the merge applied.
 */
export function getMergeDescription(meta?: MergedActionMeta): string | null {
  if (!meta || meta.mergedCount <= 1) return null;

  switch (meta.mergeType) {
    case 'input':
      return `Merged ${meta.mergedCount} keystrokes`;
    case 'scroll':
      return `Merged ${meta.mergedCount} scroll events`;
    case 'navigate':
      return `Merged ${meta.mergedCount} navigation events`;
    case 'focus-removed':
      return 'Focus event removed (implicit)';
    default:
      return `Merged ${meta.mergedCount} actions`;
  }
}
