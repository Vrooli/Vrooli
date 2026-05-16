import * as React from "react";
import { cn } from "../../lib/utils";

/**
 * Accessible tabs with arrow-key navigation. State is uncontrolled by default
 * with an optional `value`/`onValueChange` for controlled usage.
 */
export interface TabItem {
  value: string;
  label: React.ReactNode;
  disabled?: boolean;
}

export interface TabsProps {
  items: TabItem[];
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  className?: string;
  /** Accessible label for the tablist (required when there's no visible heading). */
  ariaLabel?: string;
  children: (active: string) => React.ReactNode;
}

export function Tabs({ items, value, defaultValue, onValueChange, className, ariaLabel, children }: TabsProps) {
  const [internal, setInternal] = React.useState(defaultValue ?? items[0]?.value ?? "");
  const active = value ?? internal;

  const select = (next: string) => {
    if (value === undefined) setInternal(next);
    onValueChange?.(next);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLButtonElement>) => {
    const idx = items.findIndex((i) => i.value === active);
    if (idx < 0) return;
    if (e.key === "ArrowRight") {
      e.preventDefault();
      for (let i = 1; i <= items.length; i++) {
        const cand = items[(idx + i) % items.length];
        if (cand && !cand.disabled) {
          select(cand.value);
          break;
        }
      }
    } else if (e.key === "ArrowLeft") {
      e.preventDefault();
      for (let i = 1; i <= items.length; i++) {
        const cand = items[(idx - i + items.length) % items.length];
        if (cand && !cand.disabled) {
          select(cand.value);
          break;
        }
      }
    }
  };

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      <div
        role="tablist"
        aria-label={ariaLabel}
        className="inline-flex flex-wrap items-center gap-1 rounded-control border border-app-border bg-app-surface-muted p-1"
      >
        {items.map((item) => {
          const isActive = item.value === active;
          return (
            <button
              key={item.value}
              type="button"
              role="tab"
              aria-selected={isActive}
              aria-controls={`tab-panel-${item.value}`}
              id={`tab-${item.value}`}
              tabIndex={isActive ? 0 : -1}
              disabled={item.disabled}
              onClick={() => select(item.value)}
              onKeyDown={onKeyDown}
              className={cn(
                "rounded-control px-3 py-1.5 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50",
                isActive
                  ? "bg-app-surface text-app-foreground shadow-sm"
                  : "text-app-muted-foreground hover:text-app-foreground",
              )}
            >
              {item.label}
            </button>
          );
        })}
      </div>
      <div
        role="tabpanel"
        id={`tab-panel-${active}`}
        aria-labelledby={`tab-${active}`}
        className="min-h-[1px]"
      >
        {children(active)}
      </div>
    </div>
  );
}
