/**
 * Tabs Component
 *
 * A simple, accessible tabs component for organizing content into panels.
 *
 * USAGE:
 * <Tabs value={activeTab} onValueChange={setActiveTab}>
 *   <TabsList>
 *     <TabsTrigger value="general">General</TabsTrigger>
 *     <TabsTrigger value="ai">AI</TabsTrigger>
 *   </TabsList>
 *   <TabsContent value="general">General content...</TabsContent>
 *   <TabsContent value="ai">AI content...</TabsContent>
 * </Tabs>
 */

import { createContext, useContext, type ReactNode } from "react";

// Context for sharing tab state
interface TabsContextValue {
  value: string;
  onValueChange: (value: string) => void;
}

const TabsContext = createContext<TabsContextValue | null>(null);

function useTabsContext() {
  const context = useContext(TabsContext);
  if (!context) {
    throw new Error("Tabs components must be used within a Tabs provider");
  }
  return context;
}

// Root Tabs component
interface TabsProps {
  value: string;
  onValueChange: (value: string) => void;
  children: ReactNode;
  className?: string;
}

export function Tabs({ value, onValueChange, children, className = "" }: TabsProps) {
  return (
    <TabsContext.Provider value={{ value, onValueChange }}>
      <div className={className} data-testid="tabs-root">
        {children}
      </div>
    </TabsContext.Provider>
  );
}

// Tab list container
interface TabsListProps {
  children: ReactNode;
  className?: string;
}

export function TabsList({ children, className = "" }: TabsListProps) {
  return (
    <div
      role="tablist"
      className={`flex gap-1 p-1 rounded-lg bg-white/5 ${className}`}
      data-testid="tabs-list"
    >
      {children}
    </div>
  );
}

// Individual tab trigger
interface TabsTriggerProps {
  value: string;
  children: ReactNode;
  className?: string;
  disabled?: boolean;
}

export function TabsTrigger({
  value,
  children,
  className = "",
  disabled = false,
}: TabsTriggerProps) {
  const { value: selectedValue, onValueChange } = useTabsContext();
  const isSelected = selectedValue === value;

  return (
    <button
      role="tab"
      aria-selected={isSelected}
      aria-controls={`tabpanel-${value}`}
      tabIndex={isSelected ? 0 : -1}
      disabled={disabled}
      onClick={() => onValueChange(value)}
      className={`
        flex-1 px-3 py-2 text-sm font-medium rounded-md transition-colors
        ${
          isSelected
            ? "bg-indigo-500/20 text-white"
            : "text-slate-400 hover:text-white hover:bg-white/5"
        }
        ${disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}
        ${className}
      `}
      data-testid={`tab-trigger-${value}`}
    >
      {children}
    </button>
  );
}

// Tab content panel
interface TabsContentProps {
  value: string;
  children: ReactNode;
  className?: string;
}

export function TabsContent({ value, children, className = "" }: TabsContentProps) {
  const { value: selectedValue } = useTabsContext();
  const isSelected = selectedValue === value;

  if (!isSelected) {
    return null;
  }

  return (
    <div
      role="tabpanel"
      id={`tabpanel-${value}`}
      aria-labelledby={`tab-trigger-${value}`}
      tabIndex={0}
      className={className}
      data-testid={`tab-content-${value}`}
    >
      {children}
    </div>
  );
}
