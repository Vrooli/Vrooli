import type { PointerEvent as ReactPointerEvent } from "react";

interface GridSplitterProps {
  axis: "column" | "row";
  gridColumn: string;
  gridRow: string;
  onPointerDown: (e: ReactPointerEvent) => void;
  label: string;
}

export default function GridSplitter({
  axis,
  gridColumn,
  gridRow,
  onPointerDown,
  label,
}: GridSplitterProps) {
  return (
    <div
      data-testid={`grid-splitter-${axis}`}
      role="separator"
      aria-orientation={axis === "column" ? "vertical" : "horizontal"}
      aria-label={label}
      tabIndex={0}
      className="bg-transparent transition-colors hover:bg-[rgb(var(--wc-splitter-hover))] focus:outline-none"
      style={{
        gridColumn,
        gridRow,
        cursor: axis === "column" ? "col-resize" : "row-resize",
        minWidth: axis === "column" ? "8px" : undefined,
        minHeight: axis === "row" ? "8px" : undefined,
        // Column tracks span the scrollable grid so every pane stays aligned,
        // but the hit target only needs to cover the visible viewport. Keeping
        // it viewport-sized also prevents a tall scroll surface from being
        // announced as an enormous chrome control.
        height: axis === "column" ? "100dvh" : undefined,
        maxHeight: axis === "column" ? "100dvh" : undefined,
        alignSelf: axis === "column" ? "start" : undefined,
      }}
      onPointerDown={onPointerDown}
    />
  );
}
