/**
 * GCT's vocabulary over the library's toast service.
 *
 * Everything here goes through `react-component-library:ToastManager`, which owns the
 * viewport, stacking, timeout policy and screen-reader announcement. This module only
 * decides what GCT has to say and how long it should stay.
 *
 * Writes go through `useToastActions`, whose identity is stable. The full handle is
 * rebuilt whenever the toast list changes, so an effect that pushes a toast and depends
 * on the handle re-runs itself forever.
 */

import { useCallback, useEffect, useRef } from "react";
import { useToastActions, type ToastInput } from "@vrooli/react-component-library/ToastManager/1";

/**
 * The sync lifecycle owns one toast for its whole duration: preflight, transfer, and
 * result all re-`push` under this id so a push reads as one notice that changes rather
 * than three that pile up.
 *
 * It must be `push` and not `update`: `update` leaves the existing timer running, so a
 * progress notice that becomes a sticky failure would still disappear on the progress
 * notice's original schedule.
 */
export const SYNC_TOAST_ID = "gct.sync";

/** Notices the reader must acknowledge — failures — never expire on their own. */
export const STICKY = 0;

/** Long enough to read a confirmation without holding the corner hostage. */
export const TRANSIENT_MS = 5000;

export interface NotifyOptions extends Omit<ToastInput, "title"> {
  title: string;
}

export function useNotifications() {
  const actions = useToastActions();

  const notify = useCallback((options: NotifyOptions) => actions.push(options), [actions]);

  const notifySync = useCallback(
    (options: Omit<NotifyOptions, "id">) => actions.push({ ...options, id: SYNC_TOAST_ID }),
    [actions]
  );

  const dismissSync = useCallback(() => actions.dismiss(SYNC_TOAST_ID), [actions]);

  return { notify, notifySync, dismissSync };
}

/** A react-query mutation reduced to what a notice needs from it. */
export interface WatchedMutation {
  /** Sentence-leading label, e.g. "Stage files" — becomes "Stage files failed". */
  label: string;
  error: Error | null;
  reset: () => void;
}

/**
 * Report mutation failures as toasts.
 *
 * The error is carried into the toast and the mutation is reset immediately, because
 * react-query holds a failure until something clears it: left alone, the last error
 * stays on the mutation and a retry of a *different* path reports the stale one.
 *
 * Each (mutation, message) pair is reported once. Resetting is the caller's contract,
 * not this hook's guarantee — a mutation whose reset does not clear the error must not
 * be able to push the same notice on every render.
 */
export function useMutationErrorToasts(mutations: WatchedMutation[]): void {
  const actions = useToastActions();
  const mutationsRef = useRef(mutations);
  mutationsRef.current = mutations;
  const reported = useRef(new Map<string, string>());

  const failures = mutations
    .map((mutation) => `${mutation.label}:${mutation.error?.message ?? ""}`)
    .join("|");

  useEffect(() => {
    for (const mutation of mutationsRef.current) {
      const key = `gct.mutation.${mutation.label}`;
      if (!mutation.error) {
        reported.current.delete(key);
        continue;
      }
      if (reported.current.get(key) === mutation.error.message) continue;
      reported.current.set(key, mutation.error.message);
      actions.push({
        dedupeKey: key,
        tone: "error",
        title: `${mutation.label} failed`,
        message: mutation.error.message,
        durationMs: STICKY
      });
      mutation.reset();
    }
  }, [actions, failures]);
}
