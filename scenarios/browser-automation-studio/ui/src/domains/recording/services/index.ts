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
 * - RetryService: Exponential backoff logic for session creation
 *
 * ## Related Files
 *
 * - hooks/useRecordingSession.ts: Uses RetryService for session creation
 * - types/timeline-unified.ts: AI reconciliation service (future extraction)
 */

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
