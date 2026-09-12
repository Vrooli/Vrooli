/**
 * SidebarTabs - Horizontal scrollable tab bar for sidebar navigation.
 *
 * A tab's badge counts only the items that are waiting on the operator, so a
 * tab with nothing to address shows no badge. Hovering the badge explains
 * what it counts.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { CompactTabBar, type CompactTabItem } from "../../../../components/ui/compact-tab-bar";
import { Tooltip } from "../../../../components/ui/tooltip";
import { groupActionItems } from "../../../../lib/command-post-utils";
import { nextActionService } from "../../../../services/next-action-service";
import type { FeedbackItem, MaturityItem } from "../../../../lib/attention";
import { useAgentSessionStore, useBacklogStore, useCaptureStore, useExecutionStore } from "../../../../stores";
import { useSnoozedKeys } from "../../../../stores/snooze-store";
import type { AgentSession } from "../../../../types";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { SIDEBAR_TABS, TAB_LABELS, type SidebarTab } from "./types";

interface SidebarTabsProps {
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
}

type BadgedTab = "backlog" | "captures" | "executions" | "sessions";

const SESSION_ATTENTION_STATUSES = new Set<AgentSession["status"]>([
  "waiting_for_user",
  "proposal_ready",
]);
const EMPTY_FEEDBACK = new Map<string, FeedbackItem>();
const EMPTY_MATURITY = new Map<string, MaturityItem>();

const plural = (count: number, one: string, many: string) => (count === 1 ? one : many);

const BADGES: Record<BadgedTab, { className: string; describe: (count: number) => string }> = {
  backlog: {
    className: "bg-amber-500/20 text-amber-300",
    describe: (n) => `${n} backlog ${plural(n, "item has", "items have")} a next step for you`,
  },
  captures: {
    className: "bg-violet-500/30 text-violet-300",
    describe: (n) => `${n} ${plural(n, "capture has", "captures have")} proposals to review`,
  },
  executions: {
    className: "bg-orange-500/20 text-orange-300",
    describe: (n) => `${n} ${plural(n, "execution needs", "executions need")} your review`,
  },
  sessions: {
    className: "bg-cyan-500/25 text-cyan-300",
    describe: (n) => `${n} ${plural(n, "session is", "sessions are")} waiting on you`,
  },
};

function AttentionBadge({ tab, count }: { tab: BadgedTab; count: number }) {
  if (count <= 0) return null;
  const { className, describe } = BADGES[tab];
  const description = describe(count);
  return (
    <Tooltip content={description}>
      <span
        className={`ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-semibold ${className}`}
        data-testid={`sidebar-tab-${tab}-badge`}
      >
        <span aria-hidden="true">{count > 99 ? "99+" : count}</span>
        <span className="sr-only">{description}</span>
      </span>
    </Tooltip>
  );
}

export function SidebarTabs({ activeTab, onTabChange }: SidebarTabsProps) {
  const backlogItems = useBacklogStore((s) => s.items);
  const captures = useCaptureStore((s) => s.captures);
  const executions = useExecutionStore((s) => s.items);
  const sessions = useAgentSessionStore((s) => s.sessions);
  const snoozedKeys = useSnoozedKeys();
  const { data: nextActionFeed } = useQuery({
    queryKey: ["next-actions-feed"],
    queryFn: () => nextActionService.getFeed(),
    staleTime: 15_000,
  });

  // Grouping walks every backlog item, so it runs only when its inputs change,
  // not on each session poll.
  const actionCounts = useMemo(() => {
    const groups = groupActionItems(backlogItems, executions, captures, EMPTY_FEEDBACK, EMPTY_MATURITY, snoozedKeys);
    const groupById = new Map(groups.map((group) => [group.id, group]));
    return {
      // Captures still classifying are machine work, not the operator's; only
      // those with proposals waiting for review count.
      captures:
        groupById.get("needs-classification")?.items.filter((item) => item.type === "capture" && item.primaryCta === "review").length ?? 0,
      executions: groupById.get("needs-review")?.items.filter((item) => item.type === "execution").length ?? 0,
    };
  }, [backlogItems, captures, executions, snoozedKeys]);

  const badgeCounts: Record<BadgedTab, number> = {
    ...actionCounts,
    backlog: nextActionFeed?.entries.filter((entry) => entry.entity_kind === "backlog_item").length ?? 0,
    sessions: sessions.filter((session) => SESSION_ATTENTION_STATUSES.has(session.status)).length,
  };

  const items: CompactTabItem<SidebarTab>[] = SIDEBAR_TABS.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    icon: SIDEBAR_TAB_ICONS[tab],
    badge: tab in BADGES ? <AttentionBadge tab={tab as BadgedTab} count={badgeCounts[tab as BadgedTab]} /> : null,
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
