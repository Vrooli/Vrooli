import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "../../lib/utils";
import { Tabs, TabsList, TabsTrigger } from "./tabs";

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
  "data-testid"?: string;
  /**
   * Collapse tabs to icon-only below the `sm` breakpoint. Only applies to
   * items that have an icon — a label with no icon always stays visible.
   */
  hideLabelsOnMobile?: boolean;
}

export function CompactTabBar<TValue extends string>({
  items,
  activeValue,
  onValueChange,
  "aria-label": ariaLabel,
  className,
  tabTestIdPrefix,
  "data-testid": testId,
  hideLabelsOnMobile = false,
}: CompactTabBarProps<TValue>) {
  return (
    <Tabs value={activeValue} onValueChange={(value) => onValueChange(value as TValue)} data-testid={testId}>
      <TabsList className={className} aria-label={ariaLabel}>
      {items.map((item) => {
        const isActive = activeValue === item.value;
        const Icon = item.icon;
        return (
          <TabsTrigger
            key={item.value}
            value={item.value}
            onClick={() => onValueChange(item.value)}
            className={cn(
              isActive ? "border-cyan-400 text-cyan-300" : "border-transparent text-slate-400",
            )}
            data-testid={tabTestIdPrefix ? `${tabTestIdPrefix}-${item.value}` : undefined}
          >
            <span className="inline-flex items-center gap-1.5">
              {Icon ? <Icon aria-hidden="true" className="h-3.5 w-3.5" /> : null}
              {/* sr-only (not hidden) below sm so screen readers keep the tab name */}
              <span className={cn(hideLabelsOnMobile && Icon && "sr-only sm:not-sr-only")}>{item.label}</span>
            </span>
            {typeof item.count === "number" && (
              <span className="ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-slate-800 px-1 text-[10px] font-semibold text-slate-300">
                {item.count}
              </span>
            )}
            {item.badge}
          </TabsTrigger>
        );
      })}
      </TabsList>
    </Tabs>
  );
}
