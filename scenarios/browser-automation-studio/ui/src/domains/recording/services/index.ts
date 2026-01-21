/**
 * Recording Services
 *
 * This module exports domain services for the recording reconciliation system.
 * These services contain pure business logic extracted from React hooks for
 * better testability and reusability.
 *
 * ## Architecture
 *
 * Services are pure functions or stateless classes that:
 * - Have no React dependencies (no hooks, no components)
 * - Are easily unit testable without rendering
 * - Can be reused across hooks and components
 *
 * ## Services
 *
 * - ActionMergeService: Deduplicates and cleans up raw recorded actions
 * - RetryService: Exponential backoff logic for session creation
 *
 * ## Related Files
 *
 * - utils/mergeActions.ts: Re-exports from ActionMergeService for compatibility
 * - hooks/useRecordingSession.ts: Uses RetryService for session creation
 * - types/timeline-unified.ts: AI reconciliation service (future extraction)
 */

// Action merge service for deduplicating recorded actions
export {
  mergeConsecutiveActions,
  getMergeDescription,
  type MergedAction,
  type MergedActionMeta,
} from './ActionMergeService';

// Retry service for exponential backoff
export {
  calculateRetryDelay,
  getNextRetryState,
  canRetry,
  createInitialRetryState,
  getRemainingCooldown,
  createSuccessState,
  createManualRetryState,
  type RetryConfig,
  type RetryState,
  DEFAULT_RETRY_CONFIG,
} from './RetryService';
