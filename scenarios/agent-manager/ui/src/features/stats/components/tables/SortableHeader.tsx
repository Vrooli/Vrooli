// Sortable column headers shared by the Details tables: one sort state and one
// header cell with a full-size hit target (44 px touch, 32 px pointer).

import { useState } from "react";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";

export type SortDirection = "asc" | "desc";

export interface TableSort<F extends string> {
  field: F;
  direction: SortDirection;
  /** Same column flips the direction; a new text column starts ascending, a new numeric column descending. */
  toggle: (field: F) => void;
}

export interface TableSortOptions<F extends string> {
  /** Direction the first column opens in; defaults to descending. */
  direction?: SortDirection;
  /** Columns that hold text; every other column is numeric. */
  textFields?: readonly F[];
}

export function useTableSort<F extends string>(initial: F, options: TableSortOptions<F> = {}): TableSort<F> {
  const [sort, setSort] = useState<{ field: F; direction: SortDirection }>({ field: initial, direction: options.direction ?? "desc" });
  const toggle = (field: F) =>
    setSort((current) => {
      if (current.field === field) return { field, direction: current.direction === "asc" ? "desc" : "asc" };
      return { field, direction: options.textFields?.includes(field) ? "asc" : "desc" };
    });
  return { ...sort, toggle };
}

const ALIGN_TEXT = { start: "text-left", center: "text-center", end: "text-right" } as const;
const ALIGN_FLEX = { start: "justify-start", center: "justify-center", end: "justify-end" } as const;

export function SortHeaderCell<F extends string>({ field, label, sort, last = false, align = "start", className }: { field: F; label: string; sort: TableSort<F>; last?: boolean; align?: "start" | "center" | "end"; /** Replaces the default cell padding. */ className?: string }) {
  const active = sort.field === field;
  const Icon = !active ? ArrowUpDown : sort.direction === "asc" ? ArrowUp : ArrowDown;
  return (
    <th className={`${className ?? (last ? "pb-1" : "pb-1 pr-4")} ${ALIGN_TEXT[align]}`} aria-sort={active ? (sort.direction === "asc" ? "ascending" : "descending") : "none"}>
      <button
        type="button"
        onClick={() => sort.toggle(field)}
        className={`flex min-h-11 w-full items-center gap-1 ${ALIGN_FLEX[align]} text-xs font-semibold text-muted-foreground hover:text-foreground sm:min-h-8`}
      >
        {label} <Icon aria-hidden="true" className="h-3 w-3" />
      </button>
    </th>
  );
}

/** Strings compare by locale, numbers numerically, in the sort's direction. */
export function compareSortValues(a: number | string, b: number | string, direction: SortDirection): number {
  const order = typeof a === "string" && typeof b === "string" ? a.localeCompare(b) : Number(a) - Number(b);
  return direction === "asc" ? order : -order;
}
