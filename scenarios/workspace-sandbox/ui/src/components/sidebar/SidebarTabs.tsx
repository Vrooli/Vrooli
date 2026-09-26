/**
 * SidebarTabs renders the two top-level tabs (Active, History) and an
 * unobtrusive count badge per tab.
 */

import { SIDEBAR_TABS, TAB_LABELS, type SidebarTab } from "./types";

interface SidebarTabsProps {
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
  /** Per-tab counts surfaced as a small badge; pass undefined to hide. */
  counts?: Partial<Record<SidebarTab, number | undefined>>;
}

export function SidebarTabs({ activeTab, onTabChange, counts }: SidebarTabsProps) {
  return (
    <div
      className="flex border-b border-slate-800 px-2 pt-1"
      role="tablist"
      data-testid="sidebar-tabs"
    >
      {SIDEBAR_TABS.map((tab) => {
        const count = counts?.[tab];
        const isActive = activeTab === tab;
        return (
          <button
            key={tab}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onTabChange(tab)}
            className={`shrink-0 px-3 py-2 text-xs font-medium transition-colors border-b-2 -mb-px ${
              isActive
                ? "border-emerald-500 text-emerald-300"
                : "border-transparent text-slate-400 hover:text-slate-200"
            }`}
            data-testid={`sidebar-tab-${tab}`}
          >
            <span>{TAB_LABELS[tab]}</span>
            {typeof count === "number" && (
              <span
                className={`ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-semibold ${
                  isActive ? "bg-emerald-500/20 text-emerald-200" : "bg-slate-800 text-slate-400"
                }`}
                data-testid={`sidebar-tab-${tab}-count`}
              >
                {count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
