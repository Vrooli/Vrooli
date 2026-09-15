/* eslint-disable react-refresh/only-export-components -- ToastProvider, ToastHost, and useToast belong together; splitting them harms locality. */
import { AlertCircle, CheckCircle2, Info, X } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

export type ToastTone = "info" | "success" | "warn" | "error";

export interface ToastRecord {
  id: string;
  tone: ToastTone;
  title: string;
  description?: string;
  durationMs?: number;
}

interface ToastContextValue {
  toasts: ToastRecord[];
  push: (toast: Omit<ToastRecord, "id"> & { id?: string }) => string;
  dismiss: (id: string) => void;
}

const ToastContext = React.createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = React.useState<ToastRecord[]>([]);
  const timers = React.useRef<Map<string, number>>(new Map());

  const dismiss = React.useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
    const timer = timers.current.get(id);
    if (timer) {
      window.clearTimeout(timer);
      timers.current.delete(id);
    }
  }, []);

  const push = React.useCallback<ToastContextValue["push"]>(
    (toast) => {
      const id = toast.id ?? `toast-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      setToasts((prev) => [...prev, { ...toast, id }]);
      const duration = toast.durationMs ?? 5000;
      if (duration > 0) {
        const handle = window.setTimeout(() => dismiss(id), duration);
        timers.current.set(id, handle);
      }
      return id;
    },
    [dismiss],
  );

  React.useEffect(() => {
    const map = timers.current;
    return () => {
      map.forEach((handle) => window.clearTimeout(handle));
      map.clear();
    };
  }, []);

  const value = React.useMemo(() => ({ toasts, push, dismiss }), [toasts, push, dismiss]);
  return <ToastContext.Provider value={value}>{children}</ToastContext.Provider>;
}

export function useToast() {
  const ctx = React.useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used inside <ToastProvider>");
  return ctx;
}

const TONE_ICON: Record<ToastTone, React.ComponentType<{ className?: string }>> = {
  info: Info,
  success: CheckCircle2,
  warn: AlertCircle,
  error: AlertCircle,
};

const TONE_CLASS: Record<ToastTone, string> = {
  info: "border-app-info/40 bg-app-info/10 text-app-foreground",
  success: "border-app-success/40 bg-app-success/10 text-app-foreground",
  warn: "border-app-warning/40 bg-app-warning/10 text-app-foreground",
  error: "border-app-danger/40 bg-app-danger/10 text-app-foreground",
};

export function ToastHost({
  "data-testid": testId = "toast-host",
  dismissLabel,
}: {
  "data-testid"?: string;
  dismissLabel: string;
}) {
  const { toasts, dismiss } = useToast();
  return (
    <div
      data-testid={testId}
      aria-live="polite"
      aria-atomic="false"
      className="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2"
    >
      {toasts.map((toast) => {
        const Icon = TONE_ICON[toast.tone];
        return (
          <div
            key={toast.id}
            role={toast.tone === "error" ? "alert" : "status"}
            className={cn(
              "pointer-events-auto flex w-80 max-w-[calc(100%-2rem)] gap-3 rounded-panel border bg-app-surface p-3 text-sm shadow-lg",
              TONE_CLASS[toast.tone],
            )}
          >
            <Icon aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
            <div className="flex flex-1 flex-col gap-0.5">
              <strong className="font-medium">{toast.title}</strong>
              {toast.description ? (
                <span className="text-app-muted-foreground">{toast.description}</span>
              ) : null}
            </div>
            <button
              type="button"
              aria-label={dismissLabel}
              onClick={() => dismiss(toast.id)}
              className="text-app-muted-foreground hover:text-app-foreground"
            >
              <X aria-hidden className="h-4 w-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
