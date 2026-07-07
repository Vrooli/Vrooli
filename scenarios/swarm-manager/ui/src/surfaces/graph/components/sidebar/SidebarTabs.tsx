/**
 * SidebarTabs - Horizontal scrollable tab bar for sidebar navigation.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { CompactTabBar, type CompactTabItem } from "../../../../components/ui/compact-tab-bar";
import { groupActionItems } from "../../../../lib/command-post-utils";
import type { FeedbackItem, MaturityItem } from "../../../../lib/feed";
import { backlogService } from "../../../../services";
import { useAgentSessionStore, useBacklogStore, useCaptureStore, useExecutionStore } from "../../../../stores";
import { useSnoozedKeys } from "../../../../stores/snooze-store";
import type { AgentSession } from "../../../../types";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { SIDEBAR_TABS, TAB_LABELS, type SidebarTab } from "./types";

interface SidebarTabsProps {
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
}

const SESSION_ATTENTION_STATUSES = new Set<AgentSession["status"]>([
  "waiting_for_user",
  "proposal_ready",
]);

function makeBadge(count: number, className: string) {
  if (count <= 0) return null;
  return (
    <span className={`ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-semibold ${className}`}>
      {count > 99 ? "99+" : count}
    </span>
  );
}

export function SidebarTabs({ activeTab, onTabChange }: SidebarTabsProps) {
  const backlogItems = useBacklogStore((s) => s.items);
  const captures = useCaptureStore((s) => s.captures);
  const executions = useExecutionStore((s) => s.items);
  const sessions = useAgentSessionStore((s) => s.sessions);
  const snoozedKeys = useSnoozedKeys();

  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
  });

  const feedbackMap = useMemo(() => {
    const map = new Map<string, FeedbackItem>();
    for (const item of summaryQuery.data?.feedback?.items ?? []) {
      map.set(`${item.kind}/${item.name}`, {
        kind: item.kind,
        name: item.name,
        pendingDecisions: item.pending_decisions ?? 0,
      });
    }
    return map;
  }, [summaryQuery.data?.feedback]);

  const maturityMap = useMemo(() => {
    const map = new Map<string, MaturityItem>();
    for (const item of summaryQuery.data?.maturity?.items ?? []) {
      map.set(`${item.kind}/${item.name}`, {
        kind: item.kind,
        name: item.name,
        ready: item.ready ?? false,
        pendingItems: item.pending_items ?? 0,
      });
    }
    return map;
  }, [summaryQuery.data?.maturity]);

  const badgeCounts = useMemo(() => {
    const groups = groupActionItems(backlogItems, executions, captures, feedbackMap, maturityMap, snoozedKeys);
    const groupById = new Map(groups.map((group) => [group.id, group]));
    const needsReview = groupById.get("needs-review");

    return {
      backlog:
        (groupById.get("needs-workshop")?.count ?? 0)
        + (groupById.get("ready-to-run")?.count ?? 0)
        + (groupById.get("pending-decisions")?.count ?? 0)
        + (needsReview?.items.filter((item) => item.type === "backlog").length ?? 0),
      captures: groupById.get("needs-classification")?.items.filter((item) => item.type === "capture").length ?? 0,
      executions: needsReview?.items.filter((item) => item.type === "execution").length ?? 0,
      sessions: sessions.filter((session) => SESSION_ATTENTION_STATUSES.has(session.status)).length,
    };
  }, [backlogItems, captures, executions, feedbackMap, maturityMap, sessions, snoozedKeys]);

  const items: CompactTabItem<SidebarTab>[] = SIDEBAR_TABS.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    icon: SIDEBAR_TAB_ICONS[tab],
    badge: tab === "backlog"
      ? makeBadge(badgeCounts.backlog, "bg-amber-500/20 text-amber-300")
      : tab === "captures"
        ? makeBadge(badgeCounts.captures, "bg-violet-500/30 text-violet-300")
        : tab === "executions"
          ? makeBadge(badgeCounts.executions, "bg-orange-500/20 text-orange-300")
          : tab === "sessions"
            ? makeBadge(badgeCounts.sessions, "bg-cyan-500/25 text-cyan-300")
            : null,
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
