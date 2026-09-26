import * as React from "react";

import { cn } from "../../lib/utils";

export interface TabItem<V extends string = string> {
  value: V;
  label: React.ReactNode;
  disabled?: boolean;
}

export interface TabsProps<V extends string = string> {
  value: V;
  onChange: (next: V) => void;
  items: ReadonlyArray<TabItem<V>>;
  ariaLabel: string;
  className?: string;
  "data-testid"?: string;
}

export function Tabs<V extends string>({
  value,
  onChange,
  items,
  ariaLabel,
  className,
  "data-testid": testId,
}: TabsProps<V>) {
  const refs = React.useRef<Map<V, HTMLButtonElement | null>>(new Map());

  const onKeyDown = (e: React.KeyboardEvent<HTMLButtonElement>, current: V) => {
    const enabled = items.filter((t) => !t.disabled);
    if (enabled.length === 0) return;
    const idx = enabled.findIndex((t) => t.value === current);
    if (idx < 0) return;
    let nextItem: TabItem<V> | undefined;
    if (e.key === "ArrowRight") nextItem = enabled[(idx + 1) % enabled.length];
    else if (e.key === "ArrowLeft") nextItem = enabled[(idx - 1 + enabled.length) % enabled.length];
    else if (e.key === "Home") nextItem = enabled[0];
    else if (e.key === "End") nextItem = enabled[enabled.length - 1];
    if (nextItem) {
      e.preventDefault();
      onChange(nextItem.value);
      refs.current.get(nextItem.value)?.focus();
    }
  };

  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      data-testid={testId}
      className={cn("flex gap-1 border-b border-app-border", className)}
    >
      {items.map((item) => {
        const selected = item.value === value;
        return (
          <button
            key={item.value}
            ref={(el) => {
              refs.current.set(item.value, el);
            }}
            role="tab"
            type="button"
            aria-selected={selected}
            aria-disabled={item.disabled || undefined}
            tabIndex={selected ? 0 : -1}
            disabled={item.disabled}
            onClick={() => onChange(item.value)}
            onKeyDown={(e) => onKeyDown(e, item.value)}
            className={cn(
              "relative inline-flex min-h-touch items-center gap-2 px-3 py-2 text-sm font-medium transition-colors",
              selected
                ? "text-app-foreground after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-app-primary"
                : "text-app-muted-foreground hover:text-app-foreground",
              item.disabled && "opacity-50",
            )}
          >
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
