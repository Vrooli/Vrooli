import { useState, useMemo, useCallback } from "react";
import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";

export type SortDir = "asc" | "desc";

/** Shared page sizes for mobile/desktop table pagination. */
export const MOBILE_PAGE_SIZE = 20;
export const DESKTOP_PAGE_SIZE = 25;

/**
 * Generic sortable column header button with visual sort indicators.
 * Works with any field type string union.
 */
export function SortHeader<F extends string>({
  field,
  label,
  current,
  dir,
  onToggle,
}: {
  field: F;
  label: string;
  current: F;
  dir: SortDir;
  onToggle: (f: F) => void;
}) {
  const isActive = current === field;
  return (
    <button
      type="button"
      onClick={() => onToggle(field)}
      className="inline-flex items-center gap-1 rounded px-2 py-1 -mx-2 min-h-[36px] text-left hover:text-slate-200 focus-visible:ring-2 focus-visible:ring-blue-500/50 transition-colors"
      aria-sort={isActive ? (dir === "asc" ? "ascending" : "descending") : "none"}
    >
      {label}
      {isActive ? (
        dir === "asc" ? <ArrowUp className="h-3 w-3" aria-hidden="true" /> : <ArrowDown className="h-3 w-3" aria-hidden="true" />
      ) : (
        <ArrowUpDown className="h-3 w-3 opacity-40" aria-hidden="true" />
      )}
    </button>
  );
}

/**
 * Generic hook for sortable table state with optional default direction per field.
 *
 * @param defaultField - Initial sort field
 * @param defaultDir - Initial sort direction (default: "asc")
 * @param compareFn - Comparison function: (a, b, field) => number
 * @param data - Array to sort
 */
export function useSort<T, F extends string>(
  data: T[] | undefined,
  defaultField: F,
  compareFn: (a: T, b: T, field: F) => number,
  defaultDir: SortDir = "asc",
) {
  const [sortField, setSortField] = useState<F>(defaultField);
  const [sortDir, setSortDir] = useState<SortDir>(defaultDir);

  const sorted = useMemo(() => {
    if (!data) return [];
    return [...data].sort((a, b) => {
      const cmp = compareFn(a, b, sortField);
      return sortDir === "desc" ? -cmp : cmp;
    });
  }, [data, sortField, sortDir, compareFn]);

  const toggleSort = useCallback((field: F) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortDir((d) => (d === "asc" ? "desc" : "asc"));
        return prev;
      }
      setSortDir("asc");
      return field;
    });
  }, []);

  return { sorted, sortField, sortDir, toggleSort };
}
