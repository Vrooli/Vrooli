import { createContext, useContext, useId, type ReactNode } from "react";
import { cn } from "../../lib/utils";

/**
 * Tabs primitive — controlled, value-driven.
 *
 * Consumer owns the active value. Shape mirrors Radix Tabs minus the
 * uncontrolled mode (DTV surfaces always know which tab they want).
 */
interface TabsContextValue {
  value: string;
  onValueChange: (next: string) => void;
  rootId: string;
}

const TabsContext = createContext<TabsContextValue | null>(null);

const useTabsContext = (caller: string) => {
  const ctx = useContext(TabsContext);
  if (!ctx) {
    throw new Error(`${caller} must be rendered inside <Tabs>`);
  }
  return ctx;
};

export interface TabsProps {
  value: string;
  onValueChange: (next: string) => void;
  children: ReactNode;
  className?: string;
}

export function Tabs({ value, onValueChange, children, className }: TabsProps) {
  const rootId = useId();
  return (
    <TabsContext.Provider value={{ value, onValueChange, rootId }}>
      <div className={cn("flex flex-col gap-3", className)}>{children}</div>
    </TabsContext.Provider>
  );
}

export function TabsList({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      role="tablist"
      className={cn(
        "inline-flex items-center gap-1 rounded-control border border-app-border bg-app-surface-muted p-1",
        className,
      )}
    >
      {children}
    </div>
  );
}

export interface TabsTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  value: string;
}

export function TabsTrigger({ value, className, children, ...props }: TabsTriggerProps) {
  const ctx = useTabsContext("TabsTrigger");
  const isActive = ctx.value === value;
  return (
    <button
      type="button"
      role="tab"
      aria-selected={isActive}
      aria-controls={`${ctx.rootId}-panel-${value}`}
      id={`${ctx.rootId}-trigger-${value}`}
      tabIndex={isActive ? 0 : -1}
      onClick={() => ctx.onValueChange(value)}
      className={cn(
        "rounded-control px-3 py-1.5 text-xs font-medium transition-colors",
        isActive
          ? "bg-app-surface text-app-foreground shadow-sm"
          : "text-app-muted-foreground hover:text-app-foreground",
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}

export interface TabsContentProps {
  value: string;
  children: ReactNode;
  className?: string;
}

export function TabsContent({ value, children, className }: TabsContentProps) {
  const ctx = useTabsContext("TabsContent");
  if (ctx.value !== value) return null;
  return (
    <div
      role="tabpanel"
      id={`${ctx.rootId}-panel-${value}`}
      aria-labelledby={`${ctx.rootId}-trigger-${value}`}
      className={className}
    >
      {children}
    </div>
  );
}
