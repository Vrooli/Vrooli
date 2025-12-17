/**
 * Replay Preview Service
 *
 * Executes recorded timeline entries for preview/testing before saving.
 *
 * RESPONSIBILITY:
 * This service owns the replay execution logic, extracted from RecordModeController
 * to provide cleaner separation of concerns:
 * - RecordModeController: Recording lifecycle and event capture
 * - ReplayPreviewService: Replay execution and result collection
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ WHAT THIS SERVICE DOES:                                                 │
 * │                                                                         │
 * │ 1. REPLAY EXECUTION - Executes TimelineEntry actions against a page     │
 * │ 2. RESULT COLLECTION - Collects success/failure for each action         │
 * │ 3. ERROR SCREENSHOTS - Captures screenshots on action failures          │
 * │ 4. IDEMPOTENCY - Deduplicates concurrent replay requests                │
 * │                                                                         │
 * │ WHAT IT DOES NOT DO:                                                    │
 * │                                                                         │
 * │ - Recording lifecycle (controller)                                      │
 * │ - Event capture/conversion (controller)                                 │
 * │ - Session management (session manager)                                  │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * DESIGN DECISIONS:
 *
 * 1. Page-bound service - Created per page, holds page reference
 * 2. Idempotent - Deduplicates concurrent identical replay requests
 * 3. Configurable - Supports stopOnFailure, timeout, limits
 * 4. Proto-native - Works directly with TimelineEntry (proto type)
 *
 * @module recording/replay-service
 */

import type { Page } from 'playwright';
import {
  executeTimelineEntry,
  type ExecutorContext,
  type ActionReplayResult,
} from './action-executor';
import { validateSelectorOnPage, type SelectorValidation } from './selector-service';
import type { TimelineEntry } from '../proto/recording';

// =============================================================================
// Types
// =============================================================================

/**
 * Request for replay preview.
 */
export interface ReplayPreviewRequest {
  /** Timeline entries to replay */
  entries: TimelineEntry[];
  /** Maximum number of entries to replay (default: all) */
  limit?: number;
  /** Stop replay on first failure (default: true) */
  stopOnFailure?: boolean;
  /** Timeout for each action in milliseconds (default: 10000) */
  actionTimeout?: number;
}

/**
 * Response from replay preview.
 */
export interface ReplayPreviewResponse {
  /** Whether all actions succeeded */
  success: boolean;
  /** Total number of actions executed */
  totalActions: number;
  /** Number of actions that passed */
  passedActions: number;
  /** Number of actions that failed */
  failedActions: number;
  /** Individual results for each action */
  results: ActionReplayResult[];
  /** Total execution time in milliseconds */
  totalDurationMs: number;
  /** Whether replay stopped early due to failure */
  stoppedEarly: boolean;
}

// Re-export types for convenience
export type { ActionReplayResult, SelectorValidation };

// =============================================================================
// ReplayPreviewService
// =============================================================================

/**
 * Service for replaying recorded timeline entries.
 *
 * Usage:
 * ```typescript
 * const replayService = new ReplayPreviewService(page);
 *
 * const response = await replayService.replayPreview({
 *   entries: timelineEntries,
 *   stopOnFailure: true,
 *   actionTimeout: 10000,
 * });
 *
 * if (response.success) {
 *   console.log(`All ${response.totalActions} actions passed`);
 * } else {
 *   console.log(`${response.failedActions} actions failed`);
 * }
 * ```
 */
export class ReplayPreviewService {
  private readonly page: Page;
  private pendingReplays: Map<string, Promise<ReplayPreviewResponse>> = new Map();

  constructor(page: Page) {
    this.page = page;
  }

  /**
   * Replay recorded entries for preview/testing.
   *
   * Supports idempotency - concurrent calls with the same entries
   * will return the same promise.
   *
   * @param request - Replay configuration
   * @returns Replay results
   */
  async replayPreview(request: ReplayPreviewRequest): Promise<ReplayPreviewResponse> {
    const {
      entries,
      limit,
      stopOnFailure = true,
      actionTimeout = 10000,
    } = request;

    const entriesToReplay = limit ? entries.slice(0, limit) : entries;
    const replayKey = this.generateReplayKey(entriesToReplay);

    // Check for in-flight replay with same entries
    const pendingReplay = this.pendingReplays.get(replayKey);
    if (pendingReplay) {
      return pendingReplay;
    }

    // Start new replay
    const replayPromise = this.executeReplay(entriesToReplay, stopOnFailure, actionTimeout);
    this.pendingReplays.set(replayKey, replayPromise);

    try {
      return await replayPromise;
    } finally {
      this.pendingReplays.delete(replayKey);
    }
  }

  /**
   * Validate a selector on the current page.
   *
   * Useful for checking if recorded selectors still work.
   *
   * @param selector - CSS selector to validate
   * @returns Validation result with match count
   */
  async validateSelector(selector: string): Promise<SelectorValidation> {
    return validateSelectorOnPage(this.page, selector);
  }

  /**
   * Generate a stable key for replay idempotency tracking.
   */
  private generateReplayKey(entries: TimelineEntry[]): string {
    return entries.map((e) => `${e.id}:${e.sequenceNum}`).join('|');
  }

  /**
   * Internal replay execution logic.
   */
  private async executeReplay(
    entriesToReplay: TimelineEntry[],
    stopOnFailure: boolean,
    actionTimeout: number
  ): Promise<ReplayPreviewResponse> {
    const results: ActionReplayResult[] = [];
    let stoppedEarly = false;
    const startTime = Date.now();

    // Create executor context once for all entries
    const context: ExecutorContext = {
      page: this.page,
      timeout: actionTimeout,
      validateSelector: (sel: string) => this.validateSelector(sel),
    };

    for (const entry of entriesToReplay) {
      // Execute using proto-native executor
      const result = await executeTimelineEntry(entry, context);

      // Capture screenshot on error
      if (!result.success) {
        try {
          const screenshot = await this.page.screenshot({ type: 'png' });
          result.screenshotOnError = screenshot.toString('base64');
        } catch {
          // Ignore screenshot errors
        }
      }

      results.push(result);

      if (!result.success && stopOnFailure) {
        stoppedEarly = true;
        break;
      }
    }

    const passedActions = results.filter((r) => r.success).length;
    const failedActions = results.filter((r) => !r.success).length;

    return {
      success: failedActions === 0,
      totalActions: results.length,
      passedActions,
      failedActions,
      results,
      totalDurationMs: Date.now() - startTime,
      stoppedEarly,
    };
  }
}

/**
 * Create a ReplayPreviewService for a page.
 *
 * @param page - Playwright page to replay actions against
 * @returns New ReplayPreviewService instance
 */
export function createReplayPreviewService(page: Page): ReplayPreviewService {
  return new ReplayPreviewService(page);
}
