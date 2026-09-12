import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from "react";
import { Check, Copy, Loader2 } from "lucide-react";
import { cn } from "../../../lib/utils";

export type PanelState = "empty" | "loading" | "ready" | "degraded" | "denied" | "interrupted" | "failed-recovery" | "needs-input" | "no-op";

interface PanelProps {
  testId: string;
  title: string;
  state: PanelState;
  description?: string;
  actions?: ReactNode;
  headingRef?: React.Ref<HTMLHeadingElement>;
  className?: string;
  children: ReactNode;
}

/** Panel is one labelled region of the console; its state is machine-readable. */
export function Panel({ testId, title, state, description, actions, headingRef, className, children }: PanelProps) {
  const headingId = useId();
  return (
    <section
      data-testid={testId}
      data-state={state}
      aria-labelledby={headingId}
      aria-busy={state === "loading" || undefined}
      className={cn(
        "rounded-lg border bg-slate-900/60 p-4 min-w-0",
        state === "denied" || state === "failed-recovery" ? "border-red-500/40" : state === "degraded" || state === "interrupted" ? "border-amber-500/40" : "border-white/10",
        className,
      )}
    >
      <div className="flex flex-wrap items-start justify-between gap-2 mb-3">
        <div className="min-w-0">
          <h2 id={headingId} ref={headingRef} tabIndex={-1} className="text-sm font-semibold text-slate-100 outline-none focus-visible:ring-2 focus-visible:ring-blue-400 rounded">
            {title}
          </h2>
          {description && <p className="text-xs text-slate-300 mt-0.5">{description}</p>}
        </div>
        {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
      </div>
      {children}
    </section>
  );
}

export function KeyValue({ label, children, mono = false, testId }: { label: string; children: ReactNode; mono?: boolean; testId?: string }) {
  return (
    <div className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5 text-sm min-w-0" data-testid={testId}>
      <dt className="text-slate-300 w-32 shrink-0">{label}</dt>
      <dd className={cn("text-slate-100 min-w-0 break-all", mono && "font-mono text-xs")}>{children}</dd>
    </div>
  );
}

export function CopyButton({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<number | null>(null);
  useEffect(() => () => {
    if (timer.current) window.clearTimeout(timer.current);
  }, []);
  const onCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      timer.current = window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  }, [value]);
  return (
    <button
      type="button"
      onClick={onCopy}
      aria-label={copied ? `${label} copied` : `Copy ${label}`}
      className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs text-slate-300 hover:text-white hover:bg-white/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
    >
      {copied ? <Check className="h-3.5 w-3.5" aria-hidden="true" /> : <Copy className="h-3.5 w-3.5" aria-hidden="true" />}
      <span>{copied ? "Copied" : "Copy"}</span>
    </button>
  );
}

export type Tone = "neutral" | "success" | "warning" | "danger" | "info";

const toneClasses: Record<Tone, string> = {
  neutral: "bg-slate-500/15 text-slate-200 border-slate-400/30",
  success: "bg-emerald-500/15 text-emerald-200 border-emerald-400/30",
  warning: "bg-amber-500/15 text-amber-200 border-amber-400/30",
  danger: "bg-red-500/15 text-red-200 border-red-400/30",
  info: "bg-blue-500/15 text-blue-200 border-blue-400/30",
};

export function StatusPill({ tone, children, testId }: { tone: Tone; children: ReactNode; testId?: string }) {
  return (
    <span data-testid={testId} data-tone={tone} className={cn("inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium", toneClasses[tone])}>
      {children}
    </span>
  );
}

export function SkeletonRows({ rows = 3 }: { rows?: number }) {
  return (
    <div className="space-y-2" aria-hidden="true">
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="h-4 rounded bg-white/10" style={{ width: `${70 - i * 12}%` }} />
      ))}
    </div>
  );
}

/** Spinner that respects prefers-reduced-motion. */
export function ActivityIndicator({ label }: { label: string }) {
  return (
    <span className="inline-flex items-center gap-2 text-sm text-slate-200">
      <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden="true" />
      <span>{label}</span>
    </span>
  );
}

/** Visually hidden live region; assertive for operation state, polite otherwise. */
export function LiveRegion({ message, assertive = false, testId }: { message: string; assertive?: boolean; testId?: string }) {
  return (
    <div data-testid={testId} role="status" aria-live={assertive ? "assertive" : "polite"} aria-atomic="true" className="sr-only">
      {message}
    </div>
  );
}

export function ActionButton({
  available,
  reason,
  children,
  tone = "primary",
  testId,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { available: boolean; reason?: string; tone?: "primary" | "danger" | "secondary"; testId?: string }) {
  const reasonId = useId();
  const base =
    tone === "danger"
      ? "bg-red-600 text-white hover:bg-red-500"
      : tone === "secondary"
        ? "border border-white/20 text-slate-100 hover:bg-white/10"
        : "bg-blue-600 text-white hover:bg-blue-500";
  return (
    <span className="inline-flex flex-col items-start gap-1 min-w-0">
      <button
        type="button"
        data-testid={testId}
        disabled={!available || props.disabled}
        aria-describedby={!available && reason ? reasonId : undefined}
        className={cn("inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 disabled:opacity-60 disabled:cursor-not-allowed", base)}
        {...props}
      >
        {children}
      </button>
      {!available && reason && (
        <span id={reasonId} className="text-xs text-amber-200">
          {reason}
        </span>
      )}
    </span>
  );
}
