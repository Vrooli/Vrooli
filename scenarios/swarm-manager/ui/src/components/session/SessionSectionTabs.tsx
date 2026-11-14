import type { ReactNode } from "react";
import { CompactTabBar } from "../ui/compact-tab-bar";
import { cn } from "../../lib/utils";
import type { SessionInspectorSection } from "./session-view-model";

export type SessionSectionValue = SessionInspectorSection | "conversation";

export interface SessionSectionConfig {
  value: SessionSectionValue;
  label: string;
  count?: number;
  content: ReactNode;
}

interface SessionSectionTabsProps {
  sections: SessionSectionConfig[];
  activeValue: SessionSectionValue;
  onValueChange: (value: SessionSectionValue) => void;
  listLabel: string;
  className?: string;
  tabBarClassName?: string;
  contentClassName?: string;
}

export function SessionSectionTabs({
  sections,
  activeValue,
  onValueChange,
  listLabel,
  className,
  tabBarClassName,
  contentClassName,
}: SessionSectionTabsProps) {
  if (sections.length === 0) return null;
  const firstSection = sections[0];
  if (!firstSection) return null;
  const activeSection = sections.find((section) => section.value === activeValue) ?? firstSection;

  return (
    <div className={cn("flex min-h-0 flex-col", className)}>
      <CompactTabBar
        items={sections.map(({ value, label, count }) => ({ value, label, count }))}
        activeValue={activeSection.value}
        onValueChange={onValueChange}
        aria-label={listLabel}
        className={tabBarClassName}
        tabTestIdPrefix="session-tab"
      />
      <div className={cn("min-h-0 flex-1", contentClassName)}>{activeSection.content}</div>
    </div>
  );
}
