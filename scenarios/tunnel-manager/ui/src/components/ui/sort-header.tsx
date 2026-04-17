import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";
import type { SortDir } from "../../lib/utils";

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
