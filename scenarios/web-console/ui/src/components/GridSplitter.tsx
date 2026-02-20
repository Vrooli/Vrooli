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
    <button
      data-testid={`grid-splitter-${axis}`}
      type="button"
      aria-label={label}
      className="bg-transparent transition-colors hover:bg-[rgb(var(--wc-splitter-hover))] focus:outline-none"
      style={{
        gridColumn,
        gridRow,
        cursor: axis === "column" ? "col-resize" : "row-resize",
        minWidth: axis === "column" ? "8px" : undefined,
        minHeight: axis === "row" ? "8px" : undefined,
      }}
      onPointerDown={onPointerDown}
    />
  );
}
