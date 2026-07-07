import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface CompactTabItem<TValue extends string> {
  value: TValue;
  label: string;
  icon?: LucideIcon;
  count?: number;
  badge?: ReactNode;
}

interface CompactTabBarProps<TValue extends string> {
  items: CompactTabItem<TValue>[];
  activeValue: TValue;
  onValueChange: (value: TValue) => void;
  "aria-label": string;
  className?: string;
  tabTestIdPrefix?: string;
}

export function CompactTabBar<TValue extends string>({
  items,
  activeValue,
  onValueChange,
  "aria-label": ariaLabel,
  className,
  tabTestIdPrefix,
}: CompactTabBarProps<TValue>) {
  return (
    <div className={cn("flex overflow-x-auto scrollbar-none", className)} role="tablist" aria-label={ariaLabel}>
      {items.map((item) => {
        const isActive = activeValue === item.value;
        const Icon = item.icon;
        return (
          <button
            key={item.value}
            type="button"
            role="tab"
            aria-selected={isActive}
            data-state={isActive ? "active" : "inactive"}
            onClick={() => onValueChange(item.value)}
            className={cn(
              "shrink-0 px-3 py-2 text-xs font-medium transition-colors",
              isActive
                ? "border-b-2 border-cyan-400 text-cyan-300"
                : "border-b-2 border-transparent text-slate-400 hover:text-slate-200",
            )}
            data-testid={tabTestIdPrefix ? `${tabTestIdPrefix}-${item.value}` : undefined}
          >
            <span className="inline-flex items-center gap-1.5">
              {Icon ? <Icon aria-hidden="true" className="h-3.5 w-3.5" /> : null}
              <span>{item.label}</span>
            </span>
            {typeof item.count === "number" && (
              <span className="ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-slate-800 px-1 text-[10px] font-semibold text-slate-300">
                {item.count}
              </span>
            )}
            {item.badge}
          </button>
        );
      })}
    </div>
  );
}
