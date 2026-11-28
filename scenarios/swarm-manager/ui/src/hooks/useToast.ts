/**
 * Toast Notifications - the app's feedback channel
 *
 * Every asynchronous operation an operator triggers must produce visible
 * feedback. Before this existed, a failed mutation resolved into silence: the
 * spinner stopped, nothing changed, and the operator could not tell whether the
 * action had succeeded, failed, or never fired at all.
 *
 * This module is the headless half — types, context, and the consumer hook.
 * The provider and the rendered viewport live in
 * `components/ui/toast-provider.tsx` so this file stays free of components and
 * can be imported from non-render code paths.
 *
 * DOC: docs/internal/ERROR-SEMANTICS.md
 */

import { createContext, useContext } from "react";

/**
 * Toast severities.
 *
 * `progress` is for operations that outlive the click — starting a workflow
 * returns as soon as the run is queued, not when it finishes, so the toast has
 * to say "started" rather than "done".
 */
export type ToastKind = "success" | "error" | "info" | "progress";

/** An action rendered inside the toast, e.g. "Retry" or "View run". */
export interface ToastAction {
  label: string;
  onClick: () => void;
}

export interface ToastInput {
  kind: ToastKind;
  /** The headline. One short sentence, past tense for outcomes. */
  message: string;
  /** Optional second line: recovery guidance, or a detail worth keeping. */
  description?: string;
  /**
   * Replaces any existing toast with the same key instead of stacking.
   * Use the operation identity (e.g. `goal:archive:release-1`) so a
   * double-click or a retry loop cannot flood the viewport.
   */
  key?: string;
  /** Optional inline action. Dismisses the toast when invoked. */
  action?: ToastAction;
  /**
   * Milliseconds before auto-dismiss. Defaults to 5000 for non-error kinds.
   * Errors never auto-dismiss unless this is set explicitly — an operator who
   * looked away must still be able to find out what failed.
   */
  durationMs?: number;
}

export interface Toast extends ToastInput {
  id: string;
  createdAt: number;
}

/**
 * Controls only — deliberately no `toasts` array.
 *
 * The visible list lives in the provider's local state and is handed to the
 * viewport as a prop. Publishing it through context would change the context
 * value on every toast, re-rendering every component that merely wants to
 * *send* one. Keeping the value to three stable callbacks makes it constant
 * for the life of the app, so `useToast()` costs a consumer nothing.
 */
export interface ToastContextValue {
  /** Shows a toast and returns its id. */
  notify: (input: ToastInput) => string;
  dismiss: (id: string) => void;
  dismissAll: () => void;
}

export const ToastContext = createContext<ToastContextValue | null>(null);

let warnedAboutMissingProvider = false;

/**
 * A context value that swallows notifications.
 *
 * A missing provider must never take a page down. Losing feedback is bad;
 * white-screening the route the operator was working in is worse. The warning
 * fires once so the wiring mistake is still visible in the console.
 */
const NOOP_TOASTS: ToastContextValue = {
  notify: () => {
    if (!warnedAboutMissingProvider) {
      warnedAboutMissingProvider = true;
      console.warn("[toast] useToast called outside <ToastProvider>; feedback was dropped.");
    }
    return "";
  },
  dismiss: () => undefined,
  dismissAll: () => undefined,
};

/**
 * Returns the toast controls.
 *
 * Safe to call anywhere — outside a provider it degrades to a no-op rather
 * than throwing.
 */
export function useToast(): ToastContextValue {
  return useContext(ToastContext) ?? NOOP_TOASTS;
}

/** Test seam: lets a suite assert the warn-once behaviour more than once. */
export function resetToastProviderWarning(): void {
  warnedAboutMissingProvider = false;
}
