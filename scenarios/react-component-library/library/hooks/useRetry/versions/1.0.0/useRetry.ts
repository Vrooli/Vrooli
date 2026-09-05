/** @vrooliComponentSource hooks.use-retry */
import { useCallback, useEffect, useRef, useState } from "react";

export type RetryStatus =
  | "idle"
  | "running"
  | "success"
  | "error"
  | "cancelled";

export interface RetryContext {
  attempt: number;
  signal: AbortSignal;
}

export interface UseRetryOptions {
  maxAttempts?: number;
  backoff?: number | ((attempt: number) => number);
  isRetryable?: (error: unknown, attempt: number) => boolean;
}

export interface RetryState<T> {
  status: RetryStatus;
  attempt: number;
  value?: T;
  error?: unknown;
}

export interface RetryController<T> extends RetryState<T> {
  run: (task: (context: RetryContext) => Promise<T>) => Promise<T>;
  cancel: () => void;
  reset: () => void;
}

const defaultBackoff = (attempt: number) =>
  Math.min(1000, 150 * 2 ** Math.max(0, attempt - 1));
const defaultIsRetryable = () => true;

const abortError = () => new DOMException("Retry cancelled", "AbortError");

const wait = (duration: number, signal: AbortSignal) =>
  new Promise<void>((resolve, reject) => {
    if (signal.aborted) {
      reject(abortError());
      return;
    }
    const timer = globalThis.setTimeout(
      () => {
        signal.removeEventListener("abort", abort);
        resolve();
      },
      Math.max(0, duration),
    );
    const abort = () => {
      globalThis.clearTimeout(timer);
      reject(abortError());
    };
    signal.addEventListener("abort", abort, { once: true });
  });

export function useRetry<T>({
  maxAttempts = 3,
  backoff = defaultBackoff,
  isRetryable = defaultIsRetryable,
}: UseRetryOptions = {}): RetryController<T> {
  const controller = useRef<AbortController | null>(null);
  const [state, setState] = useState<RetryState<T>>({
    status: "idle",
    attempt: 0,
  });

  const cancel = useCallback(() => {
    controller.current?.abort();
    controller.current = null;
    setState((previous) =>
      previous.status === "running"
        ? { ...previous, status: "cancelled" }
        : previous,
    );
  }, []);

  const reset = useCallback(() => {
    cancel();
    setState({ status: "idle", attempt: 0 });
  }, [cancel]);

  const run = useCallback(
    async (task: (context: RetryContext) => Promise<T>) => {
      cancel();
      const localController = new AbortController();
      controller.current = localController;
      const attempts = Math.max(1, Math.floor(maxAttempts));
      for (let attempt = 1; attempt <= attempts; attempt += 1) {
        if (controller.current !== localController) throw abortError();
        setState({ status: "running", attempt });
        try {
          const value = await task({ attempt, signal: localController.signal });
          if (controller.current !== localController) throw abortError();
          setState({ status: "success", attempt, value });
          controller.current = null;
          return value;
        } catch (error) {
          if (controller.current !== localController) {
            setState({ status: "cancelled", attempt });
            throw error;
          }
          if (attempt >= attempts || !isRetryable(error, attempt)) {
            setState({ status: "error", attempt, error });
            controller.current = null;
            throw error;
          }
          const duration =
            typeof backoff === "function" ? backoff(attempt) : backoff;
          await wait(duration, localController.signal);
        }
      }
      throw new Error("Retry exhausted without a result");
    },
    [backoff, cancel, isRetryable, maxAttempts],
  );

  useEffect(
    () => () => {
      controller.current?.abort();
      controller.current = null;
    },
    [],
  );
  return { ...state, run, cancel, reset };
}
