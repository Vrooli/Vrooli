import { X } from "lucide-react";
import type { PushNotice, WarningNotice } from "./App.types";

interface MutationError {
  error: Error | null;
  reset: () => void;
}

export interface ErrorToastProps {
  mutations: MutationError[];
  /** CSS position classes (e.g. "fixed bottom-4 right-4 max-w-md" for desktop) */
  positionClass: string;
}

export function ErrorToast({ mutations, positionClass }: ErrorToastProps) {
  const firstError = mutations.find((m) => m.error)?.error;
  if (!firstError) return null;

  return (
    <div
      className={`${positionClass} px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm shadow-lg`}
      data-testid="error-toast"
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="font-medium">Operation failed</p>
          <p className="text-xs mt-1 text-red-300">{firstError.message}</p>
        </div>
        <button
          type="button"
          onClick={() => mutations.forEach((m) => m.reset())}
          className="text-red-400 hover:text-red-200 p-1"
          aria-label="Dismiss"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

export interface WarningToastProps {
  notice: WarningNotice | null;
  onDismiss: () => void;
  positionClass: string;
}

export function WarningToast({ notice, onDismiss, positionClass }: WarningToastProps) {
  if (!notice) return null;
  return (
    <div
      className={`${positionClass} px-4 py-3 rounded-lg bg-amber-950 border border-amber-800 text-amber-200 text-sm shadow-lg`}
      data-testid="warning-toast"
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="font-medium">{notice.message}</p>
          {notice.details && (
            <p className="text-xs mt-1 text-amber-300 whitespace-pre-wrap max-h-32 overflow-y-auto">
              {notice.details}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={onDismiss}
          className="text-amber-400 hover:text-amber-200 p-1"
          aria-label="Dismiss"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

export interface PushToastProps {
  notice: PushNotice | null;
  positionClass: string;
}

export function PushToast({ notice, positionClass }: PushToastProps) {
  if (!notice) return null;

  const tone =
    notice.tone === "warning"
      ? "bg-amber-950 border-amber-800 text-amber-200"
      : notice.tone === "info"
        ? "bg-sky-950 border-sky-800 text-sky-200"
        : "bg-emerald-950 border-emerald-800 text-emerald-200";
  const title =
    notice.tone === "warning" ? "Push verification warning" : "Push status";

  return (
    <div
      className={`${positionClass} px-4 py-3 rounded-lg border text-sm ${tone}`}
      data-testid="push-toast"
    >
      <p className="font-medium">{title}</p>
      <p className="text-xs mt-1">{notice.message}</p>
    </div>
  );
}
