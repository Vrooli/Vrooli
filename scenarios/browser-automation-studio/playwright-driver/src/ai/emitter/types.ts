/**
 * Emitter Module Types
 *
 * STABILITY: STABLE CONTRACT
 *
 * Types for emitting navigation step events to the API.
 */

import type { BrowserAction } from '../action/types';
import type { TokenUsage, ElementLabel } from '../vision-client/types';

/**
 * Navigation step event sent to the API callback.
 */
export interface NavigationStepEvent {
  /** Navigation session ID */
  navigationId: string;
  /** Step number (1-indexed) */
  stepNumber: number;
  /** Action that was executed */
  action: BrowserAction;
  /** Model's reasoning for the action */
  reasoning: string;
  /** Base64-encoded screenshot */
  screenshot: string;
  /** Base64-encoded annotated screenshot (optional) */
  annotatedScreenshot?: string;
  /** Current URL after action */
  currentUrl: string;
  /** Token usage for this step */
  tokensUsed: TokenUsage;
  /** Duration of the step in milliseconds */
  durationMs: number;
  /** Whether the goal was achieved */
  goalAchieved: boolean;
  /** Error message if action failed */
  error?: string;
  /** Element labels visible in this step */
  elementLabels?: ElementLabel[];
}

/**
 * Navigation completion event sent to the API callback.
 */
export interface NavigationCompleteEvent {
  /** Navigation session ID */
  navigationId: string;
  /** Final status */
  status: 'completed' | 'failed' | 'aborted' | 'max_steps_reached' | 'loop_detected';
  /** Total steps executed */
  totalSteps: number;
  /** Total tokens used */
  totalTokens: number;
  /** Total duration in milliseconds */
  totalDurationMs: number;
  /** Final URL */
  finalUrl: string;
  /** Error message if failed */
  error?: string;
  /** Summary of what was accomplished */
  summary?: string;
}

/**
 * Callback response from the API.
 */
export interface CallbackResponse {
  /** Whether the callback was received */
  received: boolean;
  /** Whether to abort the navigation */
  abort?: boolean;
  /** Optional message from the API */
  message?: string;
}
