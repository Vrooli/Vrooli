import * as React from "react";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";

interface TabsContextValue {
  value: string;
  setValue: (next: string) => void;
  baseId: string;
}

const TabsContext = React.createContext<TabsContextValue | null>(null);

const useTabsContext = (): TabsContextValue => {
  const ctx = React.useContext(TabsContext);
  if (!ctx) throw new Error("Tabs.* must be used inside <Tabs>");
  return ctx;
};

export interface TabsProps {
  value?: string;
  defaultValue?: string;
  onValueChange?: (next: string) => void;
  children: React.ReactNode;
  className?: string;
  /** Accessible label for the tablist; required for SR users. */
  ariaLabel: string;
}

let TABS_ID_COUNTER = 0;

export function Tabs({
  value: controlled,
  defaultValue,
  onValueChange,
  children,
  className,
  ariaLabel,
}: TabsProps) {
  const [internal, setInternal] = React.useState<string>(defaultValue ?? "");
  const value = controlled ?? internal;
  const baseId = React.useMemo(() => `tabs-${++TABS_ID_COUNTER}`, []);

  const setValue = React.useCallback(
    (next: string) => {
      if (controlled === undefined) setInternal(next);
      onValueChange?.(next);
    },
    [controlled, onValueChange],
  );

  return (
    <TabsContext.Provider value={{ value, setValue, baseId }}>
      <div
        data-testid={selectors.ui.tabs.root}
        data-tabs-label={ariaLabel}
        className={cn("flex flex-col gap-3", className)}
      >
        {children}
      </div>
    </TabsContext.Provider>
  );
}

export interface TabsListProps {
  children: React.ReactNode;
  className?: string;
}

export function TabsList({ children, className }: TabsListProps) {
  return (
    <div
      data-testid={selectors.ui.tabs.list}
      role="tablist"
      className={cn("inline-flex items-center gap-1 rounded-control border border-app-border bg-app-surface-muted p-1", className)}
    >
      {children}
    </div>
  );
}

export interface TabsTriggerProps {
  value: string;
  children: React.ReactNode;
  className?: string;
  disabled?: boolean;
}

export function TabsTrigger({ value, children, className, disabled }: TabsTriggerProps) {
  const { value: active, setValue, baseId } = useTabsContext();
  const isActive = active === value;
  return (
    <button
      type="button"
      role="tab"
      aria-selected={isActive}
      aria-controls={`${baseId}-panel-${value}`}
      id={`${baseId}-trigger-${value}`}
      data-testid={selectors.ui.tabs.trigger({ value })}
      disabled={disabled}
      tabIndex={isActive ? 0 : -1}
      onClick={() => setValue(value)}
      className={cn(
        "rounded-control px-3 py-1.5 text-sm font-medium transition-colors",
        isActive
          ? "bg-app-surface text-app-foreground shadow-sm"
          : "text-app-muted-foreground hover:text-app-foreground",
        className,
      )}
    >
      {children}
    </button>
  );
}

export interface TabsPanelProps {
  value: string;
  children: React.ReactNode;
  className?: string;
}

export function TabsPanel({ value, children, className }: TabsPanelProps) {
  const { value: active, baseId } = useTabsContext();
  if (active !== value) return null;
  return (
    <div
      role="tabpanel"
      id={`${baseId}-panel-${value}`}
      aria-labelledby={`${baseId}-trigger-${value}`}
      data-testid={selectors.ui.tabs.panel({ value })}
      className={cn("focus:outline-none", className)}
    >
      {children}
    </div>
  );
}
