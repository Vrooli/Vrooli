// DOC: docs/reference/configuration.md#mobile-toolbar-layout
/**
 * Paints a `ToolbarLayout`. Holds no layout opinions of its own: every width,
 * height, and gap comes from the engine, so the live toolbar and the settings
 * preview differ only in the width they feed in.
 */
import type { ToolbarLayout } from "../../lib/toolbarLayout";
import { cn } from "../../lib/classnames";
import { renderToolbarControl, ToolbarDpad, type ToolbarControlContext } from "./toolbarControls";

export interface ToolbarSurfaceProps {
  layout: ToolbarLayout;
  ctx: ToolbarControlContext;
  testId?: string;
  className?: string;
  /** Focus-preservation: swallow the compatibility mouse events that slip
   *  through pointerdown on rapid double-taps. */
  onMouseDown?: (e: React.MouseEvent) => void;
}

export default function ToolbarSurface({
  layout,
  ctx,
  testId,
  className,
  onMouseDown,
}: ToolbarSurfaceProps) {
  const m = layout.metrics;
  if (layout.rowCount === 0) return null;

  // `inert` takes the whole subtree out of the focus order and the
  // accessibility tree, which a preview needs and `pointer-events: none`
  // cannot provide. React 18 has no typing for the attribute, so it is applied
  // as a plain DOM attribute alongside the aria fallback.
  const inertProps = ctx.inert
    ? ({ inert: "", "aria-hidden": true } as Record<string, unknown>)
    : {};

  return (
    <div
      data-testid={testId}
      {...inertProps}
      className={cn("flex items-start touch-manipulation select-none", className)}
      style={{ gap: m.gap, padding: m.padding }}
      onMouseDown={onMouseDown}
    >
      {/* Clipped, not scrollable: if a layout is ever computed against a stale
          width, a row must not be able to widen the app or start a page-level
          horizontal scroll. The overflow strip does its own scrolling inside. */}
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden" style={{ gap: m.gap }}>
        {layout.rows.map((row, rowIndex) => (
          <div
            key={rowIndex}
            data-testid={`toolbar-row-${String(rowIndex)}`}
            className="flex min-w-0 items-stretch"
            style={{ gap: m.gap, height: m.unit }}
          >
            {row.slots.map((slot) => (
              <div key={slot.id} className={cn("flex", slot.fill && "min-w-0 flex-1")}>
                {renderToolbarControl(slot, ctx, m)}
              </div>
            ))}

            {layout.strip?.rowIndex === rowIndex && (
              <div
                data-testid="toolbar-overflow-strip"
                // Overflowed controls stay one swipe away. The strip shares the
                // row it sits on, so it costs width and never height.
                className="flex min-w-0 flex-1 items-stretch overflow-x-auto"
                style={{ gap: m.gap }}
              >
                {layout.overflow.map((slot) => (
                  <div key={slot.id} className="flex shrink-0">
                    {renderToolbarControl({ ...slot, fill: false, width: m.unit }, ctx, m)}
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>

      {layout.dpad && <ToolbarDpad ctx={ctx} m={m} width={layout.dpad.width} />}
    </div>
  );
}
