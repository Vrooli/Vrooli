import { useEffect, useRef, useCallback } from 'react';

type TimeoutCallback = () => void;

/**
 * Custom hook for managing timeouts with automatic cleanup.
 * Prevents memory leaks by ensuring timeouts are always cleared on unmount or dependency changes.
 *
 * @param callback - The function to execute when the timeout expires
 * @param delay - Delay in milliseconds. Set to null to disable the timeout
 * @returns Object with methods to manually clear or reset the timeout
 *
 * @example
 * ```tsx
 * const { clear, reset } = useTimeout(() => {
 *   console.log('Timeout fired!');
 * }, 1000);
 *
 * // Manually clear the timeout
 * clear();
 *
 * // Reset the timeout (restart the countdown)
 * reset();
 * ```
 */
export const useTimeout = (callback: TimeoutCallback, delay: number | null) => {
  const timeoutRef = useRef<number | null>(null);
  const callbackRef = useRef<TimeoutCallback>(callback);

  // Keep callback ref up to date
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  const clear = useCallback(() => {
    if (timeoutRef.current !== null && typeof window !== 'undefined') {
      window.clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  }, []);

  const reset = useCallback(() => {
    clear();
    if (delay !== null && typeof window !== 'undefined') {
      timeoutRef.current = window.setTimeout(() => {
        callbackRef.current();
      }, delay);
    }
  }, [delay, clear]);

  // Set up timeout
  useEffect(() => {
    if (delay === null) {
      return undefined;
    }

    if (typeof window === 'undefined') {
      return undefined;
    }

    timeoutRef.current = window.setTimeout(() => {
      callbackRef.current();
    }, delay);

    return clear;
  }, [delay, clear]);

  return { clear, reset };
};

/**
 * Custom hook for managing a single timeout that can be scheduled imperatively.
 * Unlike useTimeout, this doesn't automatically start on mount.
 *
 * @returns Object with schedule and clear methods
 *
 * @example
 * ```tsx
 * const { schedule, clear } = useScheduledTimeout();
 *
 * const handleClick = () => {
 *   schedule(() => {
 *     console.log('Delayed action');
 *   }, 1000);
 * };
 * ```
 */
export const useScheduledTimeout = () => {
  const timeoutRef = useRef<number | null>(null);

  const clear = useCallback(() => {
    if (timeoutRef.current !== null && typeof window !== 'undefined') {
      window.clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  }, []);

  const schedule = useCallback((callback: TimeoutCallback, delay: number) => {
    clear();
    if (typeof window === 'undefined') {
      return;
    }
    timeoutRef.current = window.setTimeout(() => {
      callback();
      timeoutRef.current = null;
    }, delay);
  }, [clear]);

  useEffect(() => {
    return clear;
  }, [clear]);

  return { schedule, clear };
};
