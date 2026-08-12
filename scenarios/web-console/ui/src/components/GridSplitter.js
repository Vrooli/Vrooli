import { jsx as _jsx } from "react/jsx-runtime";
export default function GridSplitter({ axis, gridColumn, gridRow, onPointerDown, label, }) {
    return (_jsx("button", { "data-testid": `grid-splitter-${axis}`, type: "button", "aria-label": label, className: "bg-transparent transition-colors hover:bg-[rgb(var(--wc-splitter-hover))] focus:outline-none", style: {
            gridColumn,
            gridRow,
            cursor: axis === "column" ? "col-resize" : "row-resize",
            minWidth: axis === "column" ? "8px" : undefined,
            minHeight: axis === "row" ? "8px" : undefined,
        }, onPointerDown: onPointerDown }));
}
