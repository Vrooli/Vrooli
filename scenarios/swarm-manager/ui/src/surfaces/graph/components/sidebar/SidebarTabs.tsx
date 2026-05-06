/**
 * SidebarTabs - Horizontal scrollable tab bar for sidebar navigation.
 */

import { CompactTabBar, type CompactTabItem } from "../../../../components/ui/compact-tab-bar";
import { isActiveAgentSession, useAgentSessionStore, useCaptureStore } from "../../../../stores";
import { SIDEBAR_TABS, TAB_LABELS, type SidebarTab } from "./types";

interface SidebarTabsProps {
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
}

export function SidebarTabs({ activeTab, onTabChange }: SidebarTabsProps) {
  const captures = useCaptureStore((s) => s.captures);
  const pendingCount = captures.filter((c) => c.status === "classified").length;
  const activeSessionCount = useAgentSessionStore((s) => s.sessions.reduce((count, session) => count + (isActiveAgentSession(session) ? 1 : 0), 0));
  const items: CompactTabItem<SidebarTab>[] = SIDEBAR_TABS.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    badge:
      tab === "captures" && pendingCount > 0 ? (
        <span className="ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-violet-500/30 px-1 text-[10px] font-semibold text-violet-300">
          {pendingCount}
        </span>
      ) : tab === "sessions" && activeSessionCount > 0 ? (
        <span className="ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-cyan-500/25 px-1 text-[10px] font-semibold text-cyan-300">
          {activeSessionCount}
        </span>
      ) : null,
  }));

  return (
    <CompactTabBar
      items={items}
      activeValue={activeTab}
      onValueChange={onTabChange}
      aria-label="Sidebar sections"
      className="border-b border-slate-200/20"
      tabTestIdPrefix="sidebar-tab"
    />
  );
}
