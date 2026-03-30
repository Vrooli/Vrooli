/**
 * SidebarTabs - Horizontal scrollable tab bar for sidebar navigation.
 */

import { cn } from "../../../../lib/utils";
import { useCaptureStore } from "../../../../stores";
import { SIDEBAR_TABS, TAB_LABELS, type SidebarTab } from "./types";

interface SidebarTabsProps {
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
}

export function SidebarTabs({ activeTab, onTabChange }: SidebarTabsProps) {
  const captures = useCaptureStore((s) => s.captures);
  const pendingCount = captures.filter((c) => c.status === "classified").length;

  return (
    <div className="flex overflow-x-auto border-b border-slate-200/20 scrollbar-none" role="tablist">
      {SIDEBAR_TABS.map((tab) => (
        <button
          key={tab}
          type="button"
          role="tab"
          aria-selected={activeTab === tab}
          onClick={() => onTabChange(tab)}
          className={cn(
            "shrink-0 px-3 py-2 text-xs font-medium transition-colors",
            activeTab === tab
              ? "border-b-2 border-cyan-400 text-cyan-300"
              : "text-slate-400 hover:text-slate-200",
          )}
          data-testid={`sidebar-tab-${tab}`}
        >
          {TAB_LABELS[tab]}
          {tab === "captures" && pendingCount > 0 && (
            <span className="ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-violet-500/30 px-1 text-[10px] font-semibold text-violet-300">
              {pendingCount}
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
