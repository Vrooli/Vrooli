/**
 * useAsyncAction - the imperative sibling of `useActionMutation`
 *
 * Not every async operation belongs in TanStack Query. Some are one-shot
 * imperative calls inside a component that already owns its own refresh — a
 * decision card that advances to the next entry, a drawer that closes itself.
 * Those were each hand-rolling the same four pieces of state:
 *
 *     const [pending, setPending] = useState(false);
 *     const [error, setError] = useState<string | null>(null);
 *     ... setPending(true); setError(null);
 *     catch (cause) { setError(cause instanceof Error ? cause.message : "…"); }
 *     finally { setPending(false); }
 *
 * That shape appears throughout the app and gets three things wrong every
 * time: it drops `ApiError`'s structured server message on the floor, it
 * writes state after unmount, and it lets a double-click run the operation
 * twice. This hook fixes all three in one place.
 *
 * Use `useActionMutation` when the result should invalidate cached queries.
 * Use this when the caller owns what happens next.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useToast, type ToastKind } from "./useToast";
import { describeError, logError, RECOVERY_PATHS, type ErrorDescription } from "../lib/error-utils";

export interface RunOptions {
  /** Headline for the failure, e.g. "Couldn't start the milestone review". */
  errorMessage: string;
  /** Shown on success. Omit when the visible result speaks for itself. */
  successMessage?: string;
  successKind?: Extract<ToastKind, "success" | "info" | "progress">;
  successDescription?: string;
}

export interface AsyncActionOptions {
  /**
   * Also raise an error toast. Defaults to true so that forgetting to render
   * `error` cannot silently swallow a failure. Set false where the component
   * displays `error` inline and a toast would report the same event twice.
   */
  toastOnError?: boolean;
  /** Label used in structured logs to locate the failing call site. */
  source?: string;
}

export interface AsyncActionState {
  pending: boolean;
  /** User-facing failure sentence, or null. */
  error: string | null;
  /** The same failure with its category and recovery guidance. */
  errorDescription: ErrorDescription | null;
  /** Clears the error without running anything. */
  reset: () => void;
  /**
   * Reports a failure that was detected before any request went out — a
   * precondition the operator can see is unmet. Shares the single error
   * channel so a component never has to maintain two.
   */
  fail: (message: string) => void;
  /**
   * Runs `fn`, managing pending/error around it.
   *
   * Resolves true when `fn` completed, false when it threw or was ignored
   * because a run was already in flight. Never rejects — callers branch on
   * the boolean instead of wrapping this in another try/catch.
   */
  run: (fn: () => Promise<unknown>, options: RunOptions) => Promise<boolean>;
}

export function useAsyncAction({ toastOnError = true, source = "useAsyncAction" }: AsyncActionOptions = {}): AsyncActionState {
  const [pending, setPending] = useState(false);
  const [errorDescription, setErrorDescription] = useState<ErrorDescription | null>(null);
  const { notify } = useToast();

  // A rejected request that resolves after the operator has navigated away
  // must not write state into an unmounted tree.
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; };
  }, []);

  // `pending` state lags a render behind, so it cannot guard re-entry on its
  // own; a double-tap fires twice before React re-renders.
  const inFlight = useRef(false);

  const reset = useCallback(() => setErrorDescription(null), []);

  const fail = useCallback((message: string) => {
    // VALIDATION carries canRetry: false, which is correct here — repeating
    // the same click cannot satisfy a precondition the operator must fix.
    setErrorDescription({
      category: "VALIDATION",
      message,
      recovery: RECOVERY_PATHS.VALIDATION.action,
      canRetry: false,
      code: "",
    });
    if (toastOnError) notify({ kind: "error", message, key: message });
  }, [notify, toastOnError]);

  const run = useCallback(async (fn: () => Promise<unknown>, options: RunOptions): Promise<boolean> => {
    if (inFlight.current) return false;
    inFlight.current = true;
    setPending(true);
    setErrorDescription(null);

    try {
      await fn();
      if (mounted.current && options.successMessage) {
        notify({
          kind: options.successKind ?? "success",
          message: options.successMessage,
          description: options.successDescription,
          key: options.errorMessage,
        });
      }
      return true;
    } catch (cause) {
      logError(cause, source);
      const described = describeError(cause);
      if (mounted.current) {
        setErrorDescription(described);
        if (toastOnError) {
          notify({
            kind: "error",
            message: options.errorMessage,
            description: described.message,
            key: options.errorMessage,
          });
        }
      }
      return false;
    } finally {
      inFlight.current = false;
      if (mounted.current) setPending(false);
    }
  }, [notify, source, toastOnError]);

  return {
    pending,
    error: errorDescription?.message ?? null,
    errorDescription,
    reset,
    fail,
    run,
  };
}
