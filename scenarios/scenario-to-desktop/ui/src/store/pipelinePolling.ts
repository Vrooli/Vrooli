/**
 * Polling and subscription utilities for pipeline status management.
 */

import type { VerbosePipelineStatus } from "../lib/api";

export interface PollingController {
  /** Clear the current polling timeout */
  clearPollingTimeout: () => void;

  /** Set a new polling timeout */
  setPollingTimeout: (callback: () => void, delayMs: number) => void;
}

/**
 * Creates a polling controller that manages a single polling timeout.
 */
export function createPollingController(): PollingController {
  let pollingTimeoutId: ReturnType<typeof setTimeout> | null = null;

  const clearPollingTimeout = () => {
    if (pollingTimeoutId) {
      clearTimeout(pollingTimeoutId);
      pollingTimeoutId = null;
    }
  };

  const setPollingTimeout = (callback: () => void, delayMs: number) => {
    pollingTimeoutId = setTimeout(callback, delayMs);
  };

  return { clearPollingTimeout, setPollingTimeout };
}

export interface StatusSubscriberManager {
  /** Add a subscriber and return an unsubscribe function */
  subscribe: (
    callback: (status: VerbosePipelineStatus | null) => void,
  ) => () => void;

  /** Notify all subscribers with the given status */
  notifyAll: (status: VerbosePipelineStatus | null) => void;
}

/**
 * Creates a subscriber manager for pipeline status notifications.
 */
export function createStatusSubscriberManager(): StatusSubscriberManager {
  const subscribers = new Set<(status: VerbosePipelineStatus | null) => void>();

  const subscribe = (
    callback: (status: VerbosePipelineStatus | null) => void,
  ) => {
    subscribers.add(callback);
    return () => {
      subscribers.delete(callback);
    };
  };

  const notifyAll = (status: VerbosePipelineStatus | null) => {
    subscribers.forEach((callback) => {
      try {
        callback(status);
      } catch (err) {
        console.error("Error in status subscriber:", err);
      }
    });
  };

  return { subscribe, notifyAll };
}

/**
 * Debounce guard for preventing rapid repeated load calls.
 */
export interface LoadDebounceGuard {
  /** Check if a load attempt should proceed (returns true if OK) */
  shouldProceed: () => boolean;

  /** The minimum time between load attempts in ms */
  readonly debounceMs: number;
}

/**
 * Creates a debounce guard for loadActivePipeline to prevent rapid repeated calls.
 */
export function createLoadDebounceGuard(debounceMs = 500): LoadDebounceGuard {
  let lastLoadAttemptTime = 0;

  const shouldProceed = (): boolean => {
    const now = Date.now();
    if (now - lastLoadAttemptTime < debounceMs) {
      console.debug(
        "[pipelineStore] loadActivePipeline skipped - debounce active",
        {
          timeSinceLast: now - lastLoadAttemptTime,
          debounceMs,
        },
      );
      return false;
    }
    lastLoadAttemptTime = now;
    return true;
  };

  return { shouldProceed, debounceMs };
}
