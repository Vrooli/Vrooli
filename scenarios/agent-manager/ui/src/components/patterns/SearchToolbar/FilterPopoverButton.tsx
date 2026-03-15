import * as React from "react";
import { createPortal } from "react-dom";
import { SlidersHorizontal } from "lucide-react";
import { cn } from "../../../lib/utils";
import { FilterDropdown, type FilterOption } from "./FilterDropdown";
import { SortDropdown, type SortOption } from "./SortDropdown";

export interface FilterPopoverFilterConfig {
  id: string;
  label: string;
  value: string;
  options: FilterOption[];
  onChange: (value: string) => void;
  allLabel?: string;
  defaultValue?: string;
}

interface FilterPopoverButtonProps {
  filters?: FilterPopoverFilterConfig[];
  sortOptions?: SortOption[];
  currentSort?: string;
  onSortChange?: (value: string) => void;
  defaultSort?: string;
}

export function FilterPopoverButton({
  filters,
  sortOptions,
  currentSort,
  onSortChange,
  defaultSort,
}: FilterPopoverButtonProps) {
  const [open, setOpen] = React.useState(false);
  const buttonRef = React.useRef<HTMLButtonElement>(null);
  const panelRef = React.useRef<HTMLDivElement>(null);
  const [panelStyle, setPanelStyle] = React.useState<React.CSSProperties>({});

  const resolvedDefaultSort = defaultSort ?? sortOptions?.[0]?.value;

  const hasActiveFilters =
    (filters?.some((f) => f.value !== (f.defaultValue ?? "all")) ?? false) ||
    (currentSort !== undefined &&
      resolvedDefaultSort !== undefined &&
      currentSort !== resolvedDefaultSort);

  // Position the portal-rendered panel below the button
  React.useEffect(() => {
    if (!open || !buttonRef.current) return;
    const rect = buttonRef.current.getBoundingClientRect();
    setPanelStyle({
      position: "fixed",
      top: rect.bottom + 8,
      right: window.innerWidth - rect.right,
      zIndex: 50,
    });
  }, [open]);

  // Close on click outside
  React.useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      const target = e.target as Node;
      if (
        buttonRef.current?.contains(target) ||
        panelRef.current?.contains(target)
      ) {
        return;
      }
      setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  // Close on Escape
  React.useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open]);

  function handleReset() {
    filters?.forEach((f) => f.onChange(f.defaultValue ?? "all"));
    if (onSortChange && resolvedDefaultSort) {
      onSortChange(resolvedDefaultSort);
    }
  }

  return (
    <div className="shrink-0">
      <button
        ref={buttonRef}
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        aria-label="Filter and sort options"
        aria-expanded={open}
        className={cn(
          "relative inline-flex items-center justify-center h-9 w-9 rounded-md border border-input bg-background text-sm",
          "hover:bg-accent hover:text-accent-foreground transition-colors",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        )}
      >
        <SlidersHorizontal className="h-4 w-4" />
        {hasActiveFilters && (
          <span className="absolute -top-1 -right-1 h-2.5 w-2.5 rounded-full bg-primary" />
        )}
      </button>

      {open &&
        createPortal(
          <div
            ref={panelRef}
            style={panelStyle}
            className={cn(
              "w-56 rounded-md border border-border bg-muted p-3 shadow-lg",
              "animate-in fade-in-0 zoom-in-95"
            )}
          >
            <div className="space-y-3">
              {filters?.map((filter) => (
                <div key={filter.id} className="space-y-1">
                  <label className="text-xs font-medium text-muted-foreground">
                    {filter.label}
                  </label>
                  <FilterDropdown
                    value={filter.value}
                    onChange={filter.onChange}
                    options={filter.options}
                    label={filter.label}
                    allLabel={filter.allLabel}
                    className="w-full"
                  />
                </div>
              ))}

              {sortOptions &&
                sortOptions.length > 0 &&
                currentSort &&
                onSortChange && (
                  <>
                    {filters && filters.length > 0 && (
                      <div className="border-t border-border" />
                    )}
                    <div className="space-y-1">
                      <label className="text-xs font-medium text-muted-foreground">
                        Sort by
                      </label>
                      <SortDropdown
                        value={currentSort}
                        onChange={onSortChange}
                        options={sortOptions}
                        className="w-full"
                      />
                    </div>
                  </>
                )}

              {hasActiveFilters && (
                <>
                  <div className="border-t border-border" />
                  <button
                    type="button"
                    onClick={handleReset}
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors"
                  >
                    Reset filters
                  </button>
                </>
              )}
            </div>
          </div>,
          document.body
        )}
    </div>
  );
}
