/**
 * Toast Provider & Viewport
 *
 * Renders the app's feedback channel. See `hooks/useToast.ts` for the headless
 * contract and the reasoning behind it.
 *
 * Behaviour that matters:
 * - Errors are sticky. Successes auto-dismiss. An operator who looked away
 *   must still be able to find out what failed.
 * - `key` de-duplicates in place, so a double-tap or a retry loop replaces the
 *   existing toast instead of stacking a wall of identical messages.
 * - Errors announce assertively (`role="alert"`); everything else is polite,
 *   so a success toast never interrupts a screen reader mid-sentence.
 */

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { AlertTriangle, CheckCircle2, Info, Loader2, X } from "lucide-react";
import {
  ToastContext,
  type Toast,
  type ToastContextValue,
  type ToastInput,
  type ToastKind,
} from "../../hooks/useToast";

/** Beyond this the viewport becomes noise; oldest non-error toasts are evicted. */
const MAX_VISIBLE = 4;
const DEFAULT_DURATION_MS = 5000;

/**
 * Toast ids come from a local counter rather than the shared id helper in
 * `lib/error-utils`.
 *
 * That helper transitively imports `lib/api-client`, which resolves the API
 * base at module load. Because this provider is part of the shared test
 * harness, importing it would drag the HTTP layer into the module graph of
 * every test that renders anything — breaking suites that partially mock
 * `@vrooli/api-base` or `lib/api-client`. A counter is also deterministic,
 * which makes toast ids stable across runs.
 */
let toastSequence = 0;
function nextToastId(): string {
  toastSequence += 1;
  return `toast-${toastSequence}`;
}

const KIND_ICON: Record<ToastKind, React.ComponentType<{ className?: string }>> = {
  success: CheckCircle2,
  error: AlertTriangle,
  info: Info,
  progress: Loader2,
};

const KIND_STYLES: Record<ToastKind, { container: string; icon: string }> = {
  success: { container: "border-emerald-500/40 bg-emerald-950/85", icon: "text-emerald-300" },
  error: { container: "border-rose-500/50 bg-rose-950/85", icon: "text-rose-300" },
  info: { container: "border-slate-600/60 bg-slate-900/90", icon: "text-slate-300" },
  progress: { container: "border-cyan-500/40 bg-cyan-950/85", icon: "text-cyan-300" },
};

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timers = useRef(new Map<string, ReturnType<typeof setTimeout>>());

  const clearTimer = useCallback((id: string) => {
    const handle = timers.current.get(id);
    if (handle !== undefined) {
      clearTimeout(handle);
      timers.current.delete(id);
    }
  }, []);

  const dismiss = useCallback((id: string) => {
    clearTimer(id);
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, [clearTimer]);

  const dismissAll = useCallback(() => {
    for (const handle of timers.current.values()) clearTimeout(handle);
    timers.current.clear();
    setToasts([]);
  }, []);

  const notify = useCallback((input: ToastInput): string => {
    const id = nextToastId();
    const toast: Toast = { ...input, id, createdAt: Date.now() };

    setToasts((current) => {
      // Same-key replacement keeps the slot but restarts the lifecycle, so a
      // retry reads as one evolving notification rather than a pile.
      let next = input.key
        ? current.filter((existing) => {
          if (existing.key !== input.key) return true;
          clearTimer(existing.id);
          return false;
        })
        : current;

      next = [...next, toast];

      if (next.length > MAX_VISIBLE) {
        // Evict the oldest dismissible entry first. Errors are the reason the
        // operator is looking at this viewport at all, so they outrank the
        // successes that would otherwise push them out.
        const evictable = next.find((candidate) => candidate.kind !== "error") ?? next[0];
        if (evictable) {
          clearTimer(evictable.id);
          next = next.filter((candidate) => candidate.id !== evictable.id);
        }
      }
      return next;
    });

    const duration = input.durationMs ?? (input.kind === "error" ? undefined : DEFAULT_DURATION_MS);
    if (duration !== undefined && duration > 0) {
      timers.current.set(id, setTimeout(() => dismiss(id), duration));
    }
    return id;
  }, [clearTimer, dismiss]);

  // Snapshot the map itself: `timers.current` may point at a different Map by
  // cleanup time, which would leak every timer created after the swap.
  const timerMap = timers.current;
  useEffect(() => () => {
    for (const handle of timerMap.values()) clearTimeout(handle);
    timerMap.clear();
  }, [timerMap]);

  // Every member is stable, so this value is created once and never changes:
  // sending a toast re-renders the viewport, not the whole consumer tree.
  const value = useMemo<ToastContextValue>(
    () => ({ notify, dismiss, dismissAll }),
    [notify, dismiss, dismissAll],
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastViewport toasts={toasts} onDismiss={dismiss} />
    </ToastContext.Provider>
  );
}

function ToastViewport({ toasts, onDismiss }: { toasts: readonly Toast[]; onDismiss: (id: string) => void }) {
  if (typeof document === "undefined") return null;

  return createPortal(
    <div
      // pointer-events-none on the stack, auto on each card: the viewport spans
      // the corner but must not swallow clicks on the page beneath it.
      className="pointer-events-none fixed inset-x-0 bottom-0 z-[60] flex flex-col items-center gap-2 p-4 pb-[max(1rem,env(safe-area-inset-bottom))] sm:inset-x-auto sm:right-0 sm:items-end"
      data-testid="toast-viewport"
    >
      {toasts.map((toast) => (
        <ToastCard key={toast.id} toast={toast} onDismiss={onDismiss} />
      ))}
    </div>,
    document.body,
  );
}

function ToastCard({ toast, onDismiss }: { toast: Toast; onDismiss: (id: string) => void }) {
  const Icon = KIND_ICON[toast.kind];
  const styles = KIND_STYLES[toast.kind];
  const isError = toast.kind === "error";

  return (
    <div
      // Errors interrupt; everything else waits its turn.
      role={isError ? "alert" : "status"}
      aria-live={isError ? "assertive" : "polite"}
      data-testid="toast"
      data-toast-kind={toast.kind}
      className={`pointer-events-auto w-full max-w-sm rounded-xl border px-4 py-3 text-sm shadow-lg shadow-black/40 backdrop-blur ${styles.container}`}
    >
      <div className="flex items-start gap-3">
        <Icon
          className={`mt-0.5 h-4 w-4 shrink-0 ${styles.icon} ${toast.kind === "progress" ? "animate-spin motion-reduce:animate-none" : ""}`}
          aria-hidden
        />
        <div className="min-w-0 flex-1">
          <p className="font-medium leading-5 text-slate-50">{toast.message}</p>
          {toast.description ? (
            <p className="mt-1 text-xs leading-5 text-slate-300">{toast.description}</p>
          ) : null}
          {toast.action ? (
            <button
              type="button"
              data-testid="toast-action"
              onClick={() => {
                toast.action?.onClick();
                onDismiss(toast.id);
              }}
              className="mt-2 rounded font-medium text-cyan-300 underline-offset-2 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-400"
            >
              {toast.action.label}
            </button>
          ) : null}
        </div>
        <button
          type="button"
          onClick={() => onDismiss(toast.id)}
          aria-label={`Dismiss: ${toast.message}`}
          data-testid="toast-dismiss"
          className="-mr-1 -mt-1 rounded p-1 text-slate-400 transition-colors hover:bg-white/10 hover:text-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-400"
        >
          <X className="h-3.5 w-3.5" aria-hidden />
        </button>
      </div>
    </div>
  );
}
