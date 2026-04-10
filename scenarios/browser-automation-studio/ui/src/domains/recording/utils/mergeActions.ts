/**
 * Action Merging Utilities
 *
 * This module deduplicates and cleans up raw recorded actions for display.
 * It's one of three reconciliation systems in browser-automation-studio.
 *
 * ## Reconciliation Context
 *
 * See docs/architecture/reconciliation.md for the full architecture covering
 * all three systems (backend sync, action merge, AI correlation).
 *
 * ## Implementation
 *
 * The actual implementation lives in services/ActionMergeService.ts.
 * This file re-exports for backward compatibility with existing imports.
 *
 * @module
 * @see services/ActionMergeService.ts - Implementation
 * @see docs/architecture/reconciliation.md - Architecture docs
 */

// Re-export everything from the service for backward compatibility
export {
  mergeConsecutiveActions,
  getMergeDescription,
  type MergedAction,
  type MergedActionMeta,
} from '../services/ActionMergeService';
