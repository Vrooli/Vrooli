import type { ReactNode } from "react";
import { HorizontalRail } from "../HorizontalRail";

/**
 * One shelf of the fleet drawer.
 *
 * The drawer reads on two axes and each one carries a different meaning: you
 * scroll *down* through categories and *across* within one. That is why a
 * shelf is a horizontal rail rather than a list — and it is the structure a
 * third category slots into without redesigning anything.
 *
 * The scroller is `HorizontalRail`. Cards of one height are the card's job — see
 * `FleetCard`. A shelf is scanned across one line at a time, so a neighbour that
 * is taller breaks the scan and drags the shelf's height with it.
 *
 * The header stays put while the row scrolls, because the count beside it is
 * the only thing that says how much of the row you have not seen.
 */

export interface FleetRailProps {
  testId?: string;
  eyebrow: string;
  description: string;
  count: number;
  /**
   * The shelf's own controls — add, refresh. They belong in the header, not in
   * the rail: a control panel parked in the first card slot stretches to the
   * tallest card's height and leaves a column of empty surface where the eye
   * expects the first item.
   */
  actions?: ReactNode;
  children: ReactNode;
}

export function FleetRail({ testId, eyebrow, description, count, actions, children }: FleetRailProps) {
  return (
    <section data-testid={testId} className="min-w-0">
      <div className="flex flex-wrap items-end justify-between gap-x-3 gap-y-2 px-5">
        <div className="min-w-0">
          <div className="flex items-baseline gap-2">
            <h2 className="text-sm font-semibold uppercase tracking-[0.16em] text-wc-text-primary">{eyebrow}</h2>
            <span className="text-xs tabular-nums text-wc-text-faint">{count}</span>
          </div>
          <p className="mt-1 text-xs text-wc-text-muted">{description}</p>
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
      <div className="mt-3">
        <HorizontalRail testId={testId ? `${testId}-scroller` : undefined}>{children}</HorizontalRail>
      </div>
    </section>
  );
}

export default FleetRail;
