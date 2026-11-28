import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
import { ActivityIndicator } from "./ConsolePrimitives";
import { cn } from "../../../lib/utils";

export interface DestructiveActionDialogProps {
  title: string;
  /** What the action does, in one or two sentences. */
  description: string;
  /** Target key the effect lands on (machine:… or host:…). */
  target: string;
  /** Data bindings the action touches; an empty list is rendered as "no declared data". */
  affectedData: string[];
  /** Extra preview rows (downtime, recovery strategy). */
  details?: { label: string; value: ReactNode }[];
  /** The text the operator must type; the short deployment id by convention. */
  confirmText: string;
  confirmLabel: string;
  isPending: boolean;
  error?: string | null;
  onConfirm: () => void;
  onCancel: () => void;
}

const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * DestructiveActionDialog is the single review step every destructive action
 * goes through. It names the target and the affected data, requires typed
 * confirmation, traps focus while open, closes on Escape and returns focus
 * to the control that opened it. The typed text is local state and survives
 * remote observation refreshes.
 */
export function DestructiveActionDialog({
  title,
  description,
  target,
  affectedData,
  details = [],
  confirmText,
  confirmLabel,
  isPending,
  error,
  onConfirm,
  onCancel,
}: DestructiveActionDialogProps) {
  const [typed, setTyped] = useState("");
  const dialogRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const openerRef = useRef<HTMLElement | null>(null);
  const titleId = useId();
  const descId = useId();
  const canConfirm = typed === confirmText && !isPending;

  useEffect(() => {
    openerRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    inputRef.current?.focus();
    return () => {
      openerRef.current?.focus();
    };
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !isPending) {
        event.preventDefault();
        onCancel();
        return;
      }
      if (event.key !== "Tab" || !dialogRef.current) return;
      const nodes = Array.from(dialogRef.current.querySelectorAll<HTMLElement>(FOCUSABLE));
      const first = nodes[0];
      const last = nodes[nodes.length - 1];
      if (!first || !last) return;
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [isPending, onCancel]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" data-testid="console-destructive-dialog-backdrop">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descId}
        data-testid="console-destructive-dialog"
        className="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-lg border border-red-500/40 bg-slate-900 p-5 shadow-xl"
      >
        <div className="flex items-start gap-3">
          <span className="rounded-lg bg-red-500/20 p-2" aria-hidden="true">
            <AlertTriangle className="h-5 w-5 text-red-300" />
          </span>
          <div className="min-w-0">
            <h2 id={titleId} className="text-lg font-semibold text-red-200">
              {title}
            </h2>
            <p id={descId} className="mt-1 text-sm text-slate-200">
              {description}
            </p>
          </div>
        </div>

        <dl className="mt-4 space-y-2 rounded-md border border-white/10 bg-slate-950/60 p-3 text-sm">
          <div className="flex flex-wrap gap-x-3">
            <dt className="text-slate-300 w-32 shrink-0">Target</dt>
            <dd className="font-mono text-xs text-slate-100 break-all" data-testid="console-destructive-target">
              {target}
            </dd>
          </div>
          <div className="flex flex-wrap gap-x-3">
            <dt className="text-slate-300 w-32 shrink-0">Affected data</dt>
            <dd className="text-slate-100 min-w-0" data-testid="console-destructive-data">
              {affectedData.length === 0 ? (
                <span>No declared data bindings</span>
              ) : (
                <ul className="list-disc pl-4">
                  {affectedData.map((binding) => (
                    <li key={binding} className="font-mono text-xs break-all">
                      {binding}
                    </li>
                  ))}
                </ul>
              )}
            </dd>
          </div>
          {details.map((row) => (
            <div key={row.label} className="flex flex-wrap gap-x-3">
              <dt className="text-slate-300 w-32 shrink-0">{row.label}</dt>
              <dd className="text-slate-100 min-w-0">{row.value}</dd>
            </div>
          ))}
        </dl>

        <label className="mt-4 block text-sm text-slate-200" htmlFor={`${titleId}-confirm`}>
          Type <code className="rounded bg-slate-800 px-1.5 py-0.5 font-mono text-amber-200">{confirmText}</code> to confirm
        </label>
        <input
          id={`${titleId}-confirm`}
          ref={inputRef}
          type="text"
          value={typed}
          onChange={(event) => setTyped(event.target.value)}
          disabled={isPending}
          autoComplete="off"
          data-testid="console-destructive-confirm-input"
          className="mt-1 w-full rounded-lg border border-white/15 bg-slate-800 px-3 py-2 font-mono text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-400"
        />

        {error && (
          <p role="alert" className="mt-3 text-sm text-red-200" data-testid="console-destructive-error">
            {error}
          </p>
        )}

        <div className="mt-5 flex flex-wrap justify-end gap-3">
          <button
            type="button"
            onClick={onCancel}
            disabled={isPending}
            data-testid="console-destructive-cancel"
            className="rounded-lg px-4 py-2 text-sm font-medium text-slate-200 hover:bg-white/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 disabled:opacity-60"
          >
            Keep everything as is
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={!canConfirm}
            data-testid="console-destructive-confirm"
            className={cn(
              "inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400",
              "bg-red-600 hover:bg-red-500 disabled:opacity-60 disabled:cursor-not-allowed",
            )}
          >
            {isPending ? <ActivityIndicator label="Working" /> : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
