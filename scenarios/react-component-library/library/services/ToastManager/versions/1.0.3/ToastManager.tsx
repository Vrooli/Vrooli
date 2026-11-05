/**
 * @libraryId react-component-library:ToastManager
 * @displayName Toast Manager
 * @description The notification service that queues, deduplicates, updates, dismisses, limits, and announces transient feedback through one application surface.
 * @version 1.0.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ToastManager
 * @vrooliComponentSourceSlot services.toast-manager */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useAnnounce } from "@vrooli/react-component-library/useAnnounce/1";

export type ToastTone = "info" | "success" | "warning" | "error";

export interface ToastAction {
  label: string;
  onSelect: () => void;
}

export interface ToastInput {
  id?: string;
  dedupeKey?: string;
  tone?: ToastTone;
  title: string;
  message?: string;
  action?: ToastAction;
  durationMs?: number;
  dismissible?: boolean;
}

export interface ToastRecord extends Omit<ToastInput, "id"> {
  id: string;
  tone: ToastTone;
  createdAt: number;
  updatedAt: number;
}

/** The write side of the service. Stable across renders — see `useToastActions`. */
export interface ToastActions {
  push: (input: ToastInput) => string;
  update: (id: string, patch: Partial<ToastInput>) => void;
  dismiss: (id: string) => void;
  clear: () => void;
  /** Hold every running timer, e.g. while the reader is over the viewport. */
  pause: () => void;
  /** Resume held timers with the time each had left when it was paused. */
  resume: () => void;
}

export interface ToastManagerHandle extends ToastActions {
  toasts: ToastRecord[];
}

interface ToastManagerOptions {
  maxVisible?: number;
  defaultDurationMs?: number;
  initialToasts?: ToastInput[];
}

const ToastManagerContext = createContext<ToastManagerHandle | null>(null);
/**
 * The actions are published separately because the handle's identity changes with every
 * toast. A caller that pushes from an effect and depends on the handle re-runs that
 * effect on its own push, forever; taking the actions instead makes that impossible.
 */
const ToastActionsContext = createContext<ToastActions | null>(null);
let sequence = 0;

const makeId = () => `toast-${Date.now()}-${sequence++}`;
const normalizeDuration = (duration: number | undefined, fallback: number) =>
  duration === 0 ? 0 : Math.min(Math.max(duration ?? fallback, 1200), 30000);

function createRecord(input: ToastInput, defaultDurationMs: number): ToastRecord {
  const now = Date.now();
  return {
    ...input,
    id: input.id ?? makeId(),
    tone: input.tone ?? "info",
    durationMs: normalizeDuration(input.durationMs, defaultDurationMs),
    dismissible: input.dismissible ?? true,
    createdAt: now,
    updatedAt: now,
  };
}

/** What a toast says, for deciding whether a change is worth announcing again. */
const spokenText = (toast: { title?: string; message?: string }) =>
  [toast.title, toast.message].filter(Boolean).join(". ");

/**
 * Choose which toast loses its slot when the viewport is full.
 *
 * Dropping the oldest is wrong: a sticky failure is sticky precisely because the reader
 * has not dealt with it, and four routine confirmations would evict it unread. Anything
 * that expires on its own goes first, oldest of those first; a viewport made entirely of
 * notices that never expire drops the oldest, because something has to give.
 */
function chooseEviction(toasts: ToastRecord[]): string | null {
  if (toasts.length === 0) return null;
  const expiring = toasts.filter((toast) => toast.durationMs !== 0);
  const candidate = expiring[0] ?? toasts[0];
  return candidate?.id ?? null;
}

export function ToastManagerProvider({
  children,
  defaultDurationMs = 5000,
  initialToasts = [],
  maxVisible = 4,
}: ToastManagerOptions & { children?: ReactNode }) {
  const announce = useAnnounce();
  const [toasts, setToasts] = useState<ToastRecord[]>(() =>
    initialToasts
      .slice(-Math.max(1, maxVisible))
      .map((toast) => createRecord(toast, defaultDurationMs)),
  );
  const toastsRef = useRef(toasts);
  const timers = useRef(new Map<string, ReturnType<typeof setTimeout>>());
  /** When each running timer is due, so a pause can resume with the remainder. */
  const deadlines = useRef(new Map<string, number>());
  /** Time left on each timer while paused. */
  const held = useRef(new Map<string, number>());
  const [paused, setPaused] = useState(false);
  toastsRef.current = toasts;

  const clearTimer = useCallback((id: string) => {
    const timer = timers.current.get(id);
    if (timer) clearTimeout(timer);
    timers.current.delete(id);
    deadlines.current.delete(id);
    held.current.delete(id);
  }, []);

  const dismiss = useCallback(
    (id: string) => {
      clearTimer(id);
      setToasts((current) => current.filter((toast) => toast.id !== id));
    },
    [clearTimer],
  );

  const push = useCallback(
    (input: ToastInput) => {
      const existing = toastsRef.current.find(
        (toast) =>
          (input.id && toast.id === input.id) ||
          (input.dedupeKey && toast.dedupeKey === input.dedupeKey),
      );
      if (existing) {
        clearTimer(existing.id);
        const merged = {
          ...existing,
          ...input,
          id: existing.id,
          tone: input.tone ?? existing.tone,
          durationMs: normalizeDuration(input.durationMs, defaultDurationMs),
          updatedAt: Date.now(),
        };
        if (spokenText(merged) !== spokenText(existing)) {
          announce(spokenText(merged), {
            priority: merged.tone === "error" ? "assertive" : "polite",
          });
        }
        setToasts((current) => current.map((toast) => (toast.id === existing.id ? merged : toast)));
        return existing.id;
      }
      const record = createRecord(input, defaultDurationMs);
      announce(spokenText(record), {
        priority: record.tone === "error" ? "assertive" : "polite",
      });
      setToasts((current) => {
        const next = [...current, record];
        const limit = Math.max(1, maxVisible);
        while (next.length > limit) {
          const evicted = chooseEviction(next.slice(0, -1));
          const index = next.findIndex((toast) => toast.id === evicted);
          if (index < 0) break;
          const evictedToast = next[index];
          if (!evictedToast) break;
          clearTimer(evictedToast.id);
          next.splice(index, 1);
        }
        return next;
      });
      return record.id;
    },
    [announce, clearTimer, defaultDurationMs, maxVisible],
  );

  /**
   * Patch a live toast in place.
   *
   * The timer is torn down whenever the patch touches the schedule. Leaving it running —
   * which this did — means a notice patched from a five-second progress report into a
   * failure that must not expire still disappears on the progress report's clock.
   */
  const update = useCallback(
    (id: string, patch: Partial<ToastInput>) => {
      if ("durationMs" in patch) clearTimer(id);
      setToasts((current) =>
        current.map((toast) => {
          if (toast.id !== id) return toast;
          const merged = {
            ...toast,
            ...patch,
            id,
            tone: patch.tone ?? toast.tone,
            durationMs:
              "durationMs" in patch
                ? normalizeDuration(patch.durationMs, defaultDurationMs)
                : toast.durationMs,
            updatedAt: Date.now(),
          };
          if (spokenText(merged) !== spokenText(toast)) {
            announce(spokenText(merged), {
              priority: merged.tone === "error" ? "assertive" : "polite",
            });
          }
          return merged;
        }),
      );
    },
    [announce, clearTimer, defaultDurationMs],
  );

  const clear = useCallback(() => {
    timers.current.forEach((timer) => clearTimeout(timer));
    timers.current.clear();
    deadlines.current.clear();
    held.current.clear();
    setToasts([]);
  }, []);

  const pause = useCallback(() => {
    const now = Date.now();
    timers.current.forEach((timer, id) => {
      clearTimeout(timer);
      const deadline = deadlines.current.get(id);
      if (deadline !== undefined) held.current.set(id, Math.max(0, deadline - now));
    });
    timers.current.clear();
    deadlines.current.clear();
    setPaused(true);
  }, []);

  const resume = useCallback(() => setPaused(false), []);

  useEffect(() => {
    const activeIds = new Set(toasts.map((toast) => toast.id));
    timers.current.forEach((timer, id) => {
      if (!activeIds.has(id)) {
        clearTimeout(timer);
        timers.current.delete(id);
        deadlines.current.delete(id);
        held.current.delete(id);
      }
    });
    if (paused) return undefined;
    toasts.forEach((toast) => {
      if (toast.durationMs === 0 || timers.current.has(toast.id)) return;
      const remaining = held.current.get(toast.id) ?? toast.durationMs ?? 0;
      held.current.delete(toast.id);
      deadlines.current.set(toast.id, Date.now() + remaining);
      timers.current.set(
        toast.id,
        setTimeout(() => dismiss(toast.id), remaining),
      );
    });
    return undefined;
  }, [dismiss, paused, toasts]);

  useEffect(() => () => clear(), [clear]);

  const actions = useMemo<ToastActions>(
    () => ({ push, update, dismiss, clear, pause, resume }),
    [clear, dismiss, pause, push, resume, update],
  );
  const handle = useMemo<ToastManagerHandle>(() => ({ ...actions, toasts }), [actions, toasts]);

  return (
    <ToastActionsContext.Provider value={actions}>
      <ToastManagerContext.Provider value={handle}>{children}</ToastManagerContext.Provider>
    </ToastActionsContext.Provider>
  );
}

export function useToastManager(): ToastManagerHandle {
  const manager = useContext(ToastManagerContext);
  if (!manager) throw new Error("useToastManager must be used within ToastManagerProvider");
  return manager;
}

/**
 * The write side only. Prefer this wherever the toast list itself is not rendered: its
 * identity is stable, so it is safe to depend on from an effect that pushes a toast.
 */
export function useToastActions(): ToastActions {
  const actions = useContext(ToastActionsContext);
  if (!actions) throw new Error("useToastActions must be used within ToastManagerProvider");
  return actions;
}
