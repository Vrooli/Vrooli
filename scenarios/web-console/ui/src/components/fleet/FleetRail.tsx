import type { ReactNode } from "react";

export interface FleetRailProps {
  testId?: string;
  eyebrow: string;
  description: string;
  count: number;
  children: ReactNode;
}

export function FleetRail({ testId, eyebrow, description, count, children }: FleetRailProps) {
  return (
    <section data-testid={testId} className="min-w-0">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 px-5">
        <div>
          <h2 className="text-sm font-semibold uppercase tracking-[0.16em] text-wc-text-primary">{eyebrow}</h2>
          <p className="mt-1 text-xs text-wc-text-muted">{description}</p>
        </div>
        <span className="text-xs text-wc-text-faint">{count}</span>
      </div>
      <div className="mt-3 flex min-w-0 gap-3 overflow-x-auto px-5 pb-2 [scrollbar-width:thin]">
        {children}
      </div>
    </section>
  );
}

export default FleetRail;
