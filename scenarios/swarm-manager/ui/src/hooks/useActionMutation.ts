/**
 * useActionMutation - one way to run an operator-triggered async action
 *
 * Every mutation in this app owes the operator four things, and before this
 * hook each call site decided independently which of them to provide:
 *
 *   1. a pending signal while it runs,
 *   2. visible confirmation when it succeeds,
 *   3. a readable reason when it fails,
 *   4. fresh data afterwards.
 *
 * The common failure was (3): `useMutation({ mutationFn, onSuccess:
 * invalidate })` with no `onError`, so a rejected request stopped the spinner
 * and changed nothing on screen. The operator cannot distinguish that from a
 * button that was never wired up.
 *
 * This hook makes all four the default. It is a thin wrapper over
 * `useMutation` and returns the real mutation result, so anything TanStack
 * offers stays available.
 *
 * DOC: docs/internal/ERROR-SEMANTICS.md
 */

import { useCallback, useMemo, useRef } from "react";
import {
  useMutation,
  useQueryClient,
  type UseMutationResult,
} from "@tanstack/react-query";
import { useToast, type ToastKind } from "./useToast";
import { describeError, logError, type ErrorDescription } from "../lib/error-utils";

export interface ActionMutationOptions<TData, TVariables> {
  mutationFn: (variables: TVariables) => Promise<TData>;

  /**
   * Headline shown when the action fails, e.g. "Couldn't start the milestone
   * review". Required: an action worth a button is worth a sentence saying it
   * failed. The specific reason is derived from the error and shown beneath.
   */
  errorMessage: string;

  /**
   * Headline shown when the action succeeds. Omit for actions whose result is
   * already obvious on screen (an inline rename, a list that visibly
   * reorders) — a toast for those is noise.
   */
  successMessage?: string | ((data: TData, variables: TVariables) => string);

  /**
   * Severity of the success toast. Use "progress" for operations that only
   * *start* work — dispatching a workflow returns when the run is queued, and
   * claiming "done" would be a lie.
   */
  successKind?: Extract<ToastKind, "success" | "info" | "progress">;

  /** Optional second line on the success toast. */
  successDescription?: string | ((data: TData, variables: TVariables) => string | undefined);

  /** Query keys invalidated after a successful run. */
  invalidateKeys?: readonly (readonly unknown[])[];

  /**
   * Offer a "Retry" action on the error toast when the failure is retryable
   * (network, timeout, 5xx). Defaults to true. Turn it off for
   * non-idempotent actions where a blind repeat could double-apply.
   */
  allowRetry?: boolean;

  /**
   * De-duplication key for the toasts this action produces. Defaults to
   * `errorMessage`, which is already action-specific, so repeated failures of
   * the same button replace rather than stack.
   */
  toastKey?: string;

  /**
   * Suppress the automatic error toast. Only for call sites that render the
   * failure inline — a dialog that must stay open and show the reason in
   * place. `errorDescription` is still populated.
   */
  silentError?: boolean;

  /** Label used in structured logs to locate the failing call site. */
  source?: string;

  onSuccess?: (data: TData, variables: TVariables) => void;
  onError?: (error: unknown, variables: TVariables) => void;
}

/**
 * The TanStack result plus this hook's additions.
 *
 * Written as an intersection rather than an extending interface on purpose:
 * `UseMutationResult` is a discriminated union over the mutation's status, and
 * an interface cannot extend a union. The intersection keeps the narrowing
 * (`if (result.isSuccess)` still refines `data`) that an interface would flatten.
 */
export type ActionMutationResult<TData, TVariables> =
  UseMutationResult<TData, unknown, TVariables> & {
    /**
     * The failure, described for a human. Non-null only while the mutation is
     * in its error state. Use for inline error rendering.
     */
    errorDescription: ErrorDescription | null;
    /**
     * Fire-and-forget trigger. Unlike `mutateAsync` this never rejects, so it
     * can be handed straight to `onClick` without a floating-promise lint
     * error or an unhandled rejection.
     */
    run: (variables: TVariables) => void;
  };

/**
 * A mutation that always reports what happened.
 */
export function useActionMutation<TData = unknown, TVariables = void>(
  options: ActionMutationOptions<TData, TVariables>,
): ActionMutationResult<TData, TVariables> {
  const {
    mutationFn,
    errorMessage,
    successMessage,
    successKind = "success",
    successDescription,
    invalidateKeys,
    allowRetry = true,
    toastKey,
    silentError = false,
    source = "useActionMutation",
    onSuccess,
    onError,
  } = options;

  const queryClient = useQueryClient();
  const { notify } = useToast();

  // Retry re-runs the original variables. Reached through a ref because the
  // toast's callback outlives the render that created it.
  const mutationRef = useRef<UseMutationResult<TData, unknown, TVariables> | null>(null);

  const mutation = useMutation<TData, unknown, TVariables>({
    mutationFn,
    onSuccess: (data, variables) => {
      if (invalidateKeys) {
        for (const queryKey of invalidateKeys) {
          void queryClient.invalidateQueries({ queryKey });
        }
      }

      if (successMessage) {
        const message = typeof successMessage === "function"
          ? successMessage(data, variables)
          : successMessage;
        const description = typeof successDescription === "function"
          ? successDescription(data, variables)
          : successDescription;
        if (message) {
          notify({ kind: successKind, message, description, key: toastKey ?? errorMessage });
        }
      }

      onSuccess?.(data, variables);
    },
    onError: (error, variables) => {
      // Log before notifying: a correlation id in the console is what makes a
      // screenshot of a toast diagnosable later.
      logError(error, source);

      if (!silentError) {
        const described = describeError(error);
        const retryable = allowRetry && described.canRetry;
        notify({
          kind: "error",
          message: errorMessage,
          description: described.message,
          key: toastKey ?? errorMessage,
          action: retryable
            ? { label: "Retry", onClick: () => mutationRef.current?.mutate(variables) }
            : undefined,
        });
      }

      onError?.(error, variables);
    },
  });

  mutationRef.current = mutation;

  const run = useCallback((variables: TVariables) => {
    mutation.mutate(variables);
  }, [mutation]);

  const errorDescription = useMemo(
    () => (mutation.error === null || mutation.error === undefined ? null : describeError(mutation.error)),
    [mutation.error],
  );

  return useMemo(
    () => ({ ...mutation, errorDescription, run }),
    [mutation, errorDescription, run],
  );
}
