/**
 * Callback Emitter
 *
 * STABILITY: STABLE CORE
 *
 * This module emits navigation step events to the API callback URL.
 * The API uses these events for timeline updates, credit deduction, and WebSocket broadcasting.
 */

import type { StepEmitterInterface } from '../vision-agent/types';
import type { NavigationStep } from '../vision-agent/types';
import type {
  NavigationStepEvent,
  NavigationCompleteEvent,
  CallbackResponse,
} from './types';

/**
 * Configuration for CallbackEmitter.
 */
export interface CallbackEmitterConfig {
  /** Timeout for callback requests in ms */
  timeoutMs?: number;
  /** Number of retries on failure */
  retries?: number;
  /** Delay between retries in ms */
  retryDelayMs?: number;
}

/**
 * Default timeout for callback requests.
 */
const DEFAULT_TIMEOUT_MS = 5000;

/**
 * Default number of retries.
 */
const DEFAULT_RETRIES = 2;

/**
 * Default retry delay.
 */
const DEFAULT_RETRY_DELAY_MS = 500;

/**
 * Create a callback emitter.
 *
 * TESTING SEAM: This returns an interface that can be mocked.
 */
export function createCallbackEmitter(
  config: CallbackEmitterConfig = {}
): StepEmitterInterface {
  const timeoutMs = config.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const retries = config.retries ?? DEFAULT_RETRIES;
  const retryDelayMs = config.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS;

  return {
    async emit(step: NavigationStep, callbackUrl: string): Promise<void> {
      const event = stepToEvent(step);
      await sendWithRetry(callbackUrl, event, timeoutMs, retries, retryDelayMs);
    },
  };
}

/**
 * Convert NavigationStep to NavigationStepEvent for sending to API.
 */
function stepToEvent(step: NavigationStep): NavigationStepEvent {
  return {
    navigationId: step.navigationId,
    stepNumber: step.stepNumber,
    action: step.action,
    reasoning: step.reasoning,
    screenshot: step.screenshot.toString('base64'),
    annotatedScreenshot: step.annotatedScreenshot?.toString('base64'),
    currentUrl: step.currentUrl,
    tokensUsed: step.tokensUsed,
    durationMs: step.durationMs,
    goalAchieved: step.goalAchieved,
    error: step.error,
    elementLabels: step.elementLabels,
  };
}

/**
 * Send event with retry logic.
 */
async function sendWithRetry(
  url: string,
  event: NavigationStepEvent | NavigationCompleteEvent,
  timeoutMs: number,
  retries: number,
  retryDelayMs: number
): Promise<CallbackResponse> {
  let lastError: Error | undefined;

  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(event),
        signal: AbortSignal.timeout(timeoutMs),
      });

      if (!response.ok) {
        throw new Error(`Callback failed: ${response.status} ${response.statusText}`);
      }

      const result = (await response.json()) as CallbackResponse;
      return result;
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));

      if (attempt < retries) {
        await new Promise((resolve) => setTimeout(resolve, retryDelayMs));
      }
    }
  }

  throw new Error(`Callback failed after ${retries + 1} attempts: ${lastError?.message}`);
}

/**
 * Send navigation complete event.
 */
export async function emitNavigationComplete(
  callbackUrl: string,
  event: NavigationCompleteEvent,
  config: CallbackEmitterConfig = {}
): Promise<void> {
  const timeoutMs = config.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const retries = config.retries ?? DEFAULT_RETRIES;
  const retryDelayMs = config.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS;

  await sendWithRetry(callbackUrl, event, timeoutMs, retries, retryDelayMs);
}

/**
 * Create a mock emitter for testing.
 */
export function createMockEmitter(): StepEmitterInterface & {
  getEmittedSteps(): NavigationStep[];
  clearEmittedSteps(): void;
  setFailMode(shouldFail: boolean): void;
} {
  const emittedSteps: NavigationStep[] = [];
  let shouldFail = false;

  return {
    async emit(step: NavigationStep, _callbackUrl: string): Promise<void> {
      if (shouldFail) {
        throw new Error('Mock emitter failure');
      }
      emittedSteps.push(step);
    },

    getEmittedSteps(): NavigationStep[] {
      return [...emittedSteps];
    },

    clearEmittedSteps(): void {
      emittedSteps.length = 0;
    },

    setFailMode(fail: boolean): void {
      shouldFail = fail;
    },
  };
}
