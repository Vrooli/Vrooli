// Sortable column headers shared by the Details tables: one sort state and one
// header cell with a full-size hit target (44 px touch, 32 px pointer).

import { useState } from "react";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";

export type SortDirection = "asc" | "desc";

export interface TableSort<F extends string> {
  field: F;
  direction: SortDirection;
  /** Same column flips the direction; a new column starts descending. */
  toggle: (field: F) => void;
}

export function useTableSort<F extends string>(initial: F): TableSort<F> {
  const [sort, setSort] = useState<{ field: F; direction: SortDirection }>({ field: initial, direction: "desc" });
  const toggle = (field: F) =>
    setSort((current) => (current.field === field ? { field, direction: current.direction === "asc" ? "desc" : "asc" } : { field, direction: "desc" }));
  return { ...sort, toggle };
}

export function SortHeaderCell<F extends string>({ field, label, sort, last = false }: { field: F; label: string; sort: TableSort<F>; last?: boolean }) {
  const active = sort.field === field;
  const Icon = !active ? ArrowUpDown : sort.direction === "asc" ? ArrowUp : ArrowDown;
  return (
    <th className={last ? "pb-1" : "pb-1 pr-4"} aria-sort={active ? (sort.direction === "asc" ? "ascending" : "descending") : "none"}>
      <button
        type="button"
        onClick={() => sort.toggle(field)}
        className="flex min-h-11 items-center gap-1 text-xs font-semibold text-muted-foreground hover:text-foreground sm:min-h-8"
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
