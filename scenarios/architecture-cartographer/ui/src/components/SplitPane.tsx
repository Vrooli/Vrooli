import * as React from "react";
import { cn } from "../lib/utils";
import { selectors } from "../consts/selectors";
import { useResizableSplit } from "../hooks/useResizableSplit";

export interface SplitPaneProps {
  primary: React.ReactNode;
  secondary: React.ReactNode;
  /** Pre-translated label for the resize handle. */
  handleLabel: string;
  /** Initial percent for the primary pane. Defaults to 50. */
  initialPercent?: number;
  /** Defaults to horizontal (primary on the left in LTR). */
  orientation?: "horizontal" | "vertical";
  className?: string;
}

/**
 * SplitPane — resizable two-pane container.
 *
 * On screens narrower than the `md` breakpoint, both panes stack and the
 * resize handle hides — callers that want a stepwise-tab mobile experience
 * should compose with `<Tabs>` separately.
 */
export function SplitPane({
  primary,
  secondary,
  handleLabel,
  initialPercent = 50,
  orientation = "horizontal",
  className,
}: SplitPaneProps) {
  const { percent, beginDrag, setPercent } = useResizableSplit({
    initialPercent,
    orientation,
  });

  const onKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    const step = e.shiftKey ? 10 : 2;
    if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
      e.preventDefault();
      setPercent(percent - step);
    } else if (e.key === "ArrowRight" || e.key === "ArrowDown") {
      e.preventDefault();
      setPercent(percent + step);
    } else if (e.key === "Home") {
      e.preventDefault();
      setPercent(0);
    } else if (e.key === "End") {
      e.preventDefault();
      setPercent(100);
    }
  };

  const isHorizontal = orientation === "horizontal";

  return (
    <div
      data-testid={selectors.shared.splitPane.root}
      className={cn(
        "flex w-full h-full min-h-0 flex-col md:flex-row",
        !isHorizontal && "md:flex-col",
        className,
      )}
    >
      <div
        data-testid={selectors.shared.splitPane.primary}
        className="min-h-0 min-w-0 overflow-auto"
        style={{ flexBasis: `${percent}%`, flexGrow: 0, flexShrink: 0 }}
      >
        {primary}
      </div>
      {/* eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions -- ARIA separator widget needs pointer + key handlers to be resizable */}
      <div
        role="separator"
        aria-orientation={isHorizontal ? "vertical" : "horizontal"}
        aria-label={handleLabel}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(percent)}
        // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex -- separator widget must be keyboard-reachable
        tabIndex={0}
        data-testid={selectors.shared.splitPane.handle}
        onPointerDown={beginDrag}
        onKeyDown={onKeyDown}
        className={cn(
          "hidden md:block bg-app-border hover:bg-app-primary/40 focus:bg-app-primary/40 outline-none",
          isHorizontal ? "w-1 cursor-col-resize h-full" : "h-1 cursor-row-resize w-full",
        )}
      />
      <div
        data-testid={selectors.shared.splitPane.secondary}
        className="min-h-0 min-w-0 overflow-auto flex-1"
      >
        {secondary}
      </div>
    </div>
  );
}
