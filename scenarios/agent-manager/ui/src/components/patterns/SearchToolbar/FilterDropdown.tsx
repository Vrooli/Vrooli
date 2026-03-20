import * as React from "react";
import { ChevronDown, Check } from "lucide-react";
import { cn } from "../../../lib/utils";

export interface FilterOption {
  value: string;
  label: string;
}

interface FilterDropdownProps {
  value: string;
  onChange: (value: string) => void;
  options: FilterOption[];
  label: string;
  allLabel?: string;
  className?: string;
}

export function FilterDropdown({
  value,
  onChange,
  options,
  label,
  allLabel = "All",
  className,
}: FilterDropdownProps) {
  const [open, setOpen] = React.useState(false);
  const containerRef = React.useRef<HTMLDivElement>(null);

  const allOptions: FilterOption[] = [
    { value: "all", label: allLabel },
    ...options,
  ];

  const selectedLabel =
    allOptions.find((o) => o.value === value)?.label ?? allLabel;

  // Close on click outside
  React.useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (!containerRef.current?.contains(e.target as Node)) {
        setOpen(false);
      }
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

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        aria-label={label}
        aria-expanded={open}
        className={cn(
          "flex items-center justify-between w-full h-9 rounded-md border border-input bg-background px-3 text-sm",
          "hover:bg-accent/50 transition-colors",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        )}
      >
        <span className="truncate">{selectedLabel}</span>
        <ChevronDown
          className={cn(
            "h-3.5 w-3.5 ml-2 shrink-0 text-muted-foreground transition-transform",
            open && "rotate-180"
          )}
        />
      </button>
      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-popover shadow-md shadow-black/30 py-1 animate-in fade-in-0 zoom-in-95">
          {allOptions.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => {
                onChange(option.value);
                setOpen(false);
              }}
              className={cn(
                "flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors",
                "hover:bg-accent/50",
                option.value === value
                  ? "text-primary"
                  : "text-popover-foreground"
              )}
            >
              <Check
                className={cn(
                  "h-3 w-3 shrink-0",
                  option.value === value ? "opacity-100" : "opacity-0"
                )}
              />
              {option.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
