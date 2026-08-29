import type { ReactNode } from "react";

export interface FleetCardProps {
  testId?: string;
  title: string;
  meta?: string;
  state?: string;
  stateTone?: "accent" | "muted" | "warning" | "faint";
  silhouette: ReactNode;
  children?: ReactNode;
  actions?: ReactNode;
}

const stateTones: Record<NonNullable<FleetCardProps["stateTone"]>, string> = {
  accent: "border-cyan-400/30 bg-cyan-400/10 text-cyan-200",
  muted: "border-slate-400/25 bg-slate-400/10 text-slate-300",
  warning: "border-amber-400/30 bg-amber-400/10 text-amber-200",
  faint: "border-slate-500/20 bg-slate-500/5 text-slate-400",
};

export function FleetCard({ testId, title, meta, state, stateTone = "muted", silhouette, children, actions }: FleetCardProps) {
  return (
    <article
      data-testid={testId}
      className="flex w-[268px] shrink-0 flex-col overflow-hidden rounded-xl border border-wc-default bg-wc-surface-input p-4 shadow-sm transition hover:border-wc-accent/50"
    >
      <div className="relative h-28 w-full overflow-hidden rounded-lg bg-wc-surface-base sm:h-[120px]">
        {silhouette}
      </div>
      <div className="mt-3 flex min-h-12 items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate text-sm font-medium text-wc-text-primary">{title}</h3>
          {meta && <p className="mt-1 truncate text-xs text-wc-text-faint">{meta}</p>}
        </div>
        {state && <span className={`shrink-0 rounded-full border px-2 py-1 text-[10px] font-medium ${stateTones[stateTone]}`}>{state}</span>}
      </div>
      {children && <div className="mt-2 min-h-5 text-xs text-wc-text-secondary">{children}</div>}
      {actions && <div className="mt-3 flex min-h-11 items-center gap-2">{actions}</div>}
    </article>
  );
}

export default FleetCard;
