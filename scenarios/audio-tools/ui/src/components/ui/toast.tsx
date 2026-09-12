import { useEffect, useState } from "react";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

/**
 * Minimal in-process toast bus. Created here because audio-tools has no
 * pre-existing notification primitive; we keep it tiny on purpose and
 * confined to <Toaster /> so it stays trivial to replace if a richer
 * primitive lands in a shared package later.
 *
 * The pushToast/dismissToast non-component exports live alongside the
 * <Toaster /> component on purpose: this is a single self-contained
 * primitive. Splitting it across files would multiply imports for every
 * call site. HMR for this file is acceptable to break.
 */
/* eslint-disable react-refresh/only-export-components */

export interface Toast {
  id: string;
  title: string;
  body?: string;
  href?: string;
  hrefLabel?: string;
}

type Listener = (toasts: Toast[]) => void;

const listeners = new Set<Listener>();
let toasts: Toast[] = [];

function emit() {
  for (const l of listeners) l(toasts);
}

export function pushToast(t: Omit<Toast, "id"> & { id?: string }): string {
  const id = t.id ?? `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  toasts = [...toasts, { ...t, id }];
  emit();
  // Auto-dismiss after 8s.
  setTimeout(() => dismissToast(id), 8000);
  return id;
}

export function dismissToast(id: string): void {
  toasts = toasts.filter((t) => t.id !== id);
  emit();
}

export function Toaster() {
  const { t } = useTranslation();
  const [items, setItems] = useState<Toast[]>(toasts);
  useEffect(() => {
    listeners.add(setItems);
    return () => {
      listeners.delete(setItems);
    };
  }, []);
  return (
    <div
      aria-live="polite"
      role="region"
      className="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2"
    >
      {items.map((toast) => (
        <div
          key={toast.id}
          role="status"
          className="pointer-events-auto max-w-sm rounded-md border border-app-border bg-app-surface px-3 py-2 text-sm shadow-md"
        >
          <div className="flex items-start justify-between gap-2">
            <div>
              <div className="font-medium text-app-foreground">{toast.title}</div>
              {toast.body ? <div className="text-app-muted-foreground">{toast.body}</div> : null}
              {toast.href ? (
                <a className="mt-1 inline-block text-app-primary underline" href={toast.href}>
                  {toast.hrefLabel ?? t(strings.toast.viewDefault)} →
                </a>
              ) : null}
            </div>
            <button
              aria-label={t(strings.toast.dismiss)}
              className="text-app-muted-foreground hover:text-app-foreground"
              onClick={() => dismissToast(toast.id)}
            >
              ×
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
