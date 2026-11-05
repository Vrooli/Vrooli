import type { KeyboardEventHandler, ReactNode } from "react";
import { EdgeFade } from "@vrooli/react-component-library/EdgeFade/1";
import { cn } from "../lib/classnames";

/**
 * A row of same-height cards that scrolls across, shared by the fleet's shelves
 * and the launch drawer's agents.
 *
 * Three details make it read as a rail rather than as a row that happens to be
 * cut off. The peek: trailing padding is larger than a card's gap, so the next
 * card shows instead of stopping flush at the edge. The fade over that peek, so
 * the cut edge reads as "continues". The snap, so a swipe settles on a card.
 * Cards of one height are the card's job.
 */
export interface HorizontalRailProps {
  testId?: string;
  /** Spacing between cards; the fleet's roomier shelves keep their own. */
  gapClassName?: string;
  /** Leading gutter and matching scroll padding, aligned with what sits above. */
  gutterClassName?: string;
  role?: string;
  "aria-label"?: string;
  onKeyDown?: KeyboardEventHandler<HTMLDivElement>;
  children: ReactNode;
}

export function HorizontalRail({
  testId,
  gapClassName = "gap-3",
  gutterClassName = "ps-5 scroll-ps-5",
  role,
  "aria-label": ariaLabel,
  onKeyDown,
  children,
}: HorizontalRailProps) {
  return (
    <div className="relative min-w-0">
      <div
        data-testid={testId}
        role={role}
        aria-label={ariaLabel}
        onKeyDown={onKeyDown}
        className={cn(
          "flex min-w-0 snap-x snap-proximity items-stretch overflow-x-auto pe-10 pb-2 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden [&>*]:snap-start",
          gapClassName,
          gutterClassName,
        )}
      >
        {children}
      </div>
      {/* The fade is painted in the surface beneath, so it reads as the rail
          running under the edge rather than as a grey wash over it. */}
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
  );
}
