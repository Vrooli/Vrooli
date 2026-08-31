/**
 * @libraryId react-component-library:ToastManager
 * @displayName Toast Manager
 * @description A scoped transient-feedback service for queueing, deduping, updating, dismissing, and announcing toasts.
 * @version 1.0.1
 * @tags ["runtime","feedback","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource services.toast-manager */
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

export interface ToastManagerHandle {
  toasts: ToastRecord[];
  push: (input: ToastInput) => string;
  update: (id: string, patch: Partial<ToastInput>) => void;
  dismiss: (id: string) => void;
  clear: () => void;
}

interface ToastManagerOptions {
  maxVisible?: number;
  defaultDurationMs?: number;
  initialToasts?: ToastInput[];
}

const ToastManagerContext = createContext<ToastManagerHandle | null>(null);
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
  toastsRef.current = toasts;

  const dismiss = useCallback((id: string) => {
    const timer = timers.current.get(id);
    if (timer) clearTimeout(timer);
    timers.current.delete(id);
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const push = useCallback(
    (input: ToastInput) => {
      const existing = toastsRef.current.find(
        (toast) =>
          (input.id && toast.id === input.id) ||
          (input.dedupeKey && toast.dedupeKey === input.dedupeKey),
      );
      if (existing) {
        const timer = timers.current.get(existing.id);
        if (timer) clearTimeout(timer);
        timers.current.delete(existing.id);
        setToasts((current) =>
          current.map((toast) =>
            toast.id === existing.id
              ? {
                  ...toast,
                  ...input,
                  id: toast.id,
                  tone: input.tone ?? toast.tone,
                  durationMs: normalizeDuration(input.durationMs, defaultDurationMs),
                  updatedAt: Date.now(),
                }
              : toast,
          ),
        );
        return existing.id;
      }
      const record = createRecord(input, defaultDurationMs);
      announce([record.title, record.message].filter(Boolean).join(". "), {
        priority: record.tone === "error" ? "assertive" : "polite",
      });
      setToasts((current) => [...current, record].slice(-Math.max(1, maxVisible)));
      const resultId = record.id;
      return resultId;
    },
    [announce, defaultDurationMs, maxVisible],
  );

  const update = useCallback((id: string, patch: Partial<ToastInput>) => {
    setToasts((current) =>
      current.map((toast) =>
        toast.id === id
          ? {
              ...toast,
              ...patch,
              id,
              tone: patch.tone ?? toast.tone,
              updatedAt: Date.now(),
            }
          : toast,
      ),
    );
  }, []);

  const clear = useCallback(() => {
    timers.current.forEach((timer) => clearTimeout(timer));
    timers.current.clear();
    setToasts([]);
  }, []);

  useEffect(() => {
    const activeIds = new Set(toasts.map((toast) => toast.id));
    timers.current.forEach((timer, id) => {
      if (!activeIds.has(id)) {
        clearTimeout(timer);
        timers.current.delete(id);
      }
    });
    toasts.forEach((toast) => {
      if (toast.durationMs === 0 || timers.current.has(toast.id)) return;
      timers.current.set(
        toast.id,
        setTimeout(() => dismiss(toast.id), toast.durationMs),
      );
    });
    return () => undefined;
  }, [dismiss, toasts]);

  useEffect(() => () => clear(), [clear]);

  const handle = useMemo<ToastManagerHandle>(
    () => ({ toasts, push, update, dismiss, clear }),
    [clear, dismiss, push, toasts, update],
  );

  return <ToastManagerContext.Provider value={handle}>{children}</ToastManagerContext.Provider>;
}

export function useToastManager(): ToastManagerHandle {
  const manager = useContext(ToastManagerContext);
  if (!manager) throw new Error("useToastManager must be used within ToastManagerProvider");
  return manager;
}
