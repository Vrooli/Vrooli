import type { ReactNode } from "react";
import { EdgeFade } from "@vrooli/react-component-library/EdgeFade/1";

/**
 * One shelf of the fleet drawer.
 *
 * The drawer reads on two axes and each one carries a different meaning: you
 * scroll *down* through categories and *across* within one. That is why a
 * shelf is a horizontal rail rather than a list — and it is the structure a
 * third category slots into without redesigning anything.
 *
 * Three details are what make a rail read as a rail rather than as a row that
 * happens to be cut off:
 *
 *   The peek. The rail's trailing padding is smaller than its leading padding,
 *   so the next card is partly visible instead of stopping flush at the
 *   container edge. A row that ends exactly at the edge looks complete when it
 *   is not, and nothing else on screen says otherwise.
 *
 *   The fade over that peek, so the cut edge reads as "continues" rather than
 *   as a clipped card.
 *
 *   Cards of one height, which is the card's job rather than the rail's — see
 *   `FleetCard`. A shelf is scanned across one line at a time, so a neighbour
 *   that is taller breaks the scan and drags the shelf's height with it.
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
      <div className="relative mt-3 min-w-0">
        <div
          data-testid={testId ? `${testId}-scroller` : undefined}
          className="flex min-w-0 items-stretch gap-3 overflow-x-auto ps-5 pe-10 pb-2 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
        >
          {children}
        </div>
        {/* The fade is painted in the drawer's own surface, so it reads as the
            shelf running under the edge rather than as a grey wash over it. */}
        <EdgeFade
          side="inline-end"
          style={{
            ["--edge-fade-width" as string]: "2.5rem",
            ["--edge-fade" as string]:
              "linear-gradient(to left, rgb(var(--wc-surface-raised)) 35%, rgb(var(--wc-surface-raised) / 0) 100%)",
            insetBlockEnd: "0.5rem",
          }}
        />
      </div>
    </section>
  );
}

export default FleetRail;
