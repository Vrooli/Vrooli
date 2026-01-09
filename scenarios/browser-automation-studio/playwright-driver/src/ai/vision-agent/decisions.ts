/**
 * Vision Agent Decision Boundaries
 *
 * DECISION BOUNDARY EXTRACTION:
 * This module consolidates decision logic that was previously scattered
 * or embedded inline. Each function represents a named decision point
 * with clear semantics and documented rationale.
 *
 * ## Design Principles:
 * 1. Each function answers a YES/NO question about a situation
 * 2. Functions are pure - no side effects, just decision logic
 * 3. Related decisions are grouped together
 * 4. Complex conditionals are named for clarity
 *
 * ## Usage:
 * Import and use these in the main agent loop instead of inline conditionals.
 *
 * @see agent.ts - Main vision agent that uses these decisions
 * @see loop-detection.ts - Loop detection decisions (already extracted)
 */

// =============================================================================
// SCREENSHOT FAILURE CLASSIFICATION
// =============================================================================

/**
 * Error patterns that indicate screenshot capture was blocked by the page.
 * These patterns suggest CAPTCHA or verification challenges that intercept
 * the Chrome DevTools Protocol (CDP) screenshot command.
 *
 * RATIONALE:
 * - 'captureScreenshot': CDP method name that failed
 * - 'Unable to capture screenshot': Playwright error message
 * - 'Protocol error': Generic CDP communication failure
 */
const SCREENSHOT_BLOCK_PATTERNS = [
  'captureScreenshot',
  'Unable to capture screenshot',
  'Protocol error',
] as const;

/**
 * Determine if a screenshot failure indicates a potential CAPTCHA/verification.
 *
 * DECISION: Is this screenshot error caused by page-level blocking (CAPTCHA)
 * rather than a transient or infrastructure error?
 *
 * Background:
 * Some CAPTCHAs (especially iframe-based like Cloudflare) can block the CDP
 * screenshot command before we can detect them via DOM inspection. When this
 * happens, the screenshot fails with specific protocol errors.
 *
 * If TRUE: Trigger human intervention (show verification prompt)
 * If FALSE: Treat as regular error (fail the navigation)
 *
 * @param errorMessage - Error message from screenshot failure
 * @returns True if the error suggests CAPTCHA/verification blocking
 */
export function isScreenshotBlockedByCaptcha(errorMessage: string): boolean {
  return SCREENSHOT_BLOCK_PATTERNS.some(pattern =>
    errorMessage.includes(pattern)
  );
}

/**
 * Extend the blocked patterns list (for testing or customization).
 *
 * @param patterns - Additional patterns to check
 * @returns Updated decision function
 */
export function createScreenshotBlockDetector(
  additionalPatterns: string[] = []
): (errorMessage: string) => boolean {
  const allPatterns = [...SCREENSHOT_BLOCK_PATTERNS, ...additionalPatterns];
  return (errorMessage: string) =>
    allPatterns.some(pattern => errorMessage.includes(pattern));
}

// =============================================================================
// HUMAN INTERVENTION TRIGGERS
// =============================================================================

/**
 * Reasons why human intervention might be needed.
 */
export type InterventionTrigger =
  | 'captcha_detected'           // CAPTCHA found by DOM inspection
  | 'screenshot_blocked'         // Screenshot failed with protocol error
  | 'ai_requested'               // AI explicitly requested human help
  | 'loop_detected'              // Navigation stuck in a loop
  | 'max_retries_exceeded';      // Too many failed attempts

/**
 * Determine if an action result indicates the AI is stuck.
 *
 * DECISION: Has the AI exhausted reasonable attempts on an action?
 *
 * @param consecutiveFailures - Number of consecutive action failures
 * @param maxRetries - Maximum retries before giving up (default: 3)
 * @returns True if max retries exceeded
 */
export function isMaxRetriesExceeded(
  consecutiveFailures: number,
  maxRetries: number = 3
): boolean {
  return consecutiveFailures >= maxRetries;
}

// =============================================================================
// GOAL COMPLETION DETECTION
// =============================================================================

/**
 * Determine if the AI has achieved the navigation goal.
 *
 * DECISION: Should we stop navigation because the goal is complete?
 *
 * The AI signals completion by:
 * 1. Setting goalAchieved: true in its response
 * 2. Returning a 'done' action type
 *
 * @param goalAchieved - Boolean from AI analysis
 * @param actionType - Type of action returned by AI
 * @returns True if navigation should stop (goal achieved)
 */
export function isGoalAchieved(
  goalAchieved: boolean,
  actionType: string
): boolean {
  return goalAchieved || actionType === 'done';
}

/**
 * Determine if the AI is requesting human help.
 *
 * DECISION: Did the AI decide it needs human intervention?
 *
 * @param actionType - Type of action returned by AI
 * @returns True if AI requested human intervention
 */
export function isHumanInterventionRequested(actionType: string): boolean {
  return actionType === 'request_human';
}

// =============================================================================
// NAVIGATION STATE DECISIONS
// =============================================================================

/**
 * Determine if navigation should continue to the next step.
 *
 * DECISION: Should the navigation loop continue?
 *
 * Navigation continues unless:
 * - Goal has been achieved
 * - Maximum steps reached
 * - Navigation was aborted
 * - Fatal error occurred
 *
 * @param status - Current navigation status
 * @returns True if navigation should continue
 */
export function shouldContinueNavigation(
  status: 'in_progress' | 'completed' | 'failed' | 'aborted' | 'max_steps_reached' | 'paused'
): boolean {
  return status === 'in_progress';
}

/**
 * Determine if a step should be skipped (resume after intervention).
 *
 * DECISION: Should we skip normal processing after resuming from pause?
 *
 * After human intervention, we typically want to restart the observe phase
 * rather than continuing with stale data.
 *
 * @param wasJustResumed - Whether we just resumed from a pause
 * @returns True if we should restart observation
 */
export function shouldRestartObservation(wasJustResumed: boolean): boolean {
  return wasJustResumed;
}

// =============================================================================
// CONTEXT WINDOW MANAGEMENT
// =============================================================================

/**
 * Determine if conversation history needs trimming.
 *
 * DECISION: Is the conversation history too long for the context window?
 *
 * Long histories consume tokens and can cause context overflow.
 * This decision helps manage memory usage.
 *
 * @param historyLength - Current number of messages in history
 * @param maxMessages - Maximum allowed messages
 * @returns True if history should be trimmed
 */
export function shouldTrimHistory(
  historyLength: number,
  maxMessages: number
): boolean {
  return historyLength > maxMessages;
}

/**
 * Calculate how many messages to keep after trimming.
 *
 * DECISION: How much history should we preserve?
 *
 * Strategy: Keep the most recent messages as they're most relevant
 * for the current navigation context.
 *
 * @param currentLength - Current history length
 * @param targetLength - Desired history length
 * @returns Number of messages to remove from the start
 */
export function calculateHistoryTrimCount(
  currentLength: number,
  targetLength: number
): number {
  if (currentLength <= targetLength) return 0;
  return currentLength - targetLength;
}

// =============================================================================
// ERROR CLASSIFICATION
// =============================================================================

/**
 * Error categories for navigation failures.
 */
export type NavigationErrorCategory =
  | 'captcha'           // CAPTCHA or verification challenge
  | 'network'           // Network connectivity issues
  | 'timeout'           // Operation timed out
  | 'vision_model'      // Vision API error
  | 'action_failed'     // Browser action execution failed
  | 'page_crashed'      // Page became unresponsive
  | 'unknown';          // Unclassified error

/**
 * Classify a navigation error for appropriate handling.
 *
 * DECISION: What type of error is this?
 *
 * @param errorMessage - Error message to classify
 * @returns Error category
 */
export function classifyNavigationError(errorMessage: string): NavigationErrorCategory {
  const msg = errorMessage.toLowerCase();

  // CAPTCHA indicators
  if (isScreenshotBlockedByCaptcha(errorMessage)) {
    return 'captcha';
  }

  // Network errors
  if (msg.includes('net::') ||
      msg.includes('network') ||
      msg.includes('econnrefused') ||
      msg.includes('dns')) {
    return 'network';
  }

  // Timeouts
  if (msg.includes('timeout') || msg.includes('timed out')) {
    return 'timeout';
  }

  // Vision model errors
  if (msg.includes('vision') ||
      msg.includes('openai') ||
      msg.includes('claude') ||
      msg.includes('anthropic') ||
      msg.includes('api key') ||
      msg.includes('rate limit')) {
    return 'vision_model';
  }

  // Action failures
  if (msg.includes('element not found') ||
      msg.includes('not clickable') ||
      msg.includes('selector')) {
    return 'action_failed';
  }

  // Page crashes
  if (msg.includes('target closed') ||
      msg.includes('page closed') ||
      msg.includes('context destroyed')) {
    return 'page_crashed';
  }

  return 'unknown';
}

/**
 * Determine if an error is recoverable.
 *
 * DECISION: Should we retry after this error?
 *
 * @param category - Error category
 * @returns True if error is potentially recoverable with retry
 */
export function isRecoverableError(category: NavigationErrorCategory): boolean {
  switch (category) {
    case 'captcha':
      return true;  // Human can solve CAPTCHA
    case 'network':
      return true;  // Network issues can be transient
    case 'timeout':
      return true;  // Timeouts can be transient
    case 'action_failed':
      return true;  // AI can try different approach
    case 'vision_model':
      return false; // API issues usually require intervention
    case 'page_crashed':
      return false; // Page is gone
    case 'unknown':
      return false; // Don't retry unknown errors
  }
}
