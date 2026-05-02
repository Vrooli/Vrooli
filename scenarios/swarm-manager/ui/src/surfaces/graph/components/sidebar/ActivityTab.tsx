/**
 * ActivityTab - Renders the unified feed (captures + backlog items).
 *
 * Extracted from the original Sidebar component. Preserves the existing
 * FeedItemCard rendering and priority-based ordering.
 */

import { useState } from "react";
import { ChevronDown, ChevronRight, Clock, Inbox } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { useRecentlyViewedStore, type RecentlyViewedItem } from "../../../../stores/recently-viewed-store";
import { buildBacklogNodeId, buildExecutionNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import type { FeedItem } from "../../../../lib/feed";
import { SidebarEmptyState } from "./SidebarEmptyState";

interface ActivityTabProps {
  feed: FeedItem[];
  searchQuery: string;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
}

const statusColors: Record<string, string> = {
  running: "bg-cyan-500/20 text-cyan-300",
  in_progress: "bg-cyan-500/20 text-cyan-300",
  completed: "bg-green-500/20 text-green-300",
  classified: "bg-green-500/20 text-green-300",
  complete: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
  error: "bg-red-500/20 text-red-300",
  queued: "bg-amber-500/20 text-amber-300",
  needs_review: "bg-amber-500/20 text-amber-300",
  ready: "bg-emerald-500/20 text-emerald-300",
  classifying: "bg-blue-500/20 text-blue-300",
  researching: "bg-blue-500/20 text-blue-300",
};

function FeedItemCard({ item, onClick }: { item: FeedItem; onClick: () => void }) {
  const label =
    item.type === "capture"
      ? item.capture.text.slice(0, 60) + (item.capture.text.length > 60 ? "..." : "")
      : item.item.title || item.item.name;

  const timestamp = item.type === "capture" ? item.capture.created : item.item.updated;
  const statusBadge = item.type === "capture" ? item.capture.status : item.item.status;
  const statusColor = statusColors[statusBadge] ?? "bg-slate-700/60 text-slate-300";

  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
      data-testid="sidebar-feed-item"
    >
      <div className="flex items-start justify-between gap-2">
        <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">{label}</p>
        <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", statusColor)}>
          {statusBadge?.replace(/_/g, " ")}
        </span>
      </div>
      {item.type === "attention" && item.reasons.length > 0 && (
        <div className="mt-1.5 flex flex-wrap gap-1">
          {item.reasons.map((reason, index) => (
            <span
              key={`${reason.kind}-${index}`}
              className="rounded bg-amber-500/15 px-1.5 py-0.5 text-[10px] text-amber-300"
            >
              {reason.kind === "pending-decisions"
                ? `${reason.count} decision${reason.count > 1 ? "s" : ""}`
                : reason.kind.replace(/-/g, " ")}
            </span>
          ))}
        </div>
      )}
      <p className="mt-1 text-[11px] text-slate-500">{formatRelativeTime(timestamp)}</p>
    </button>
  );
}

const entityTypeColors: Record<string, string> = {
  backlog: "bg-blue-500/20 text-blue-300",
  execution: "bg-cyan-500/20 text-cyan-300",
  initiative: "bg-emerald-500/20 text-emerald-300",
  scenario: "bg-purple-500/20 text-purple-300",
};

function recentItemToNodeId(item: RecentlyViewedItem): string {
  switch (item.entityType) {
    case "backlog":
      return buildBacklogNodeId(item.kind ?? "feature", item.name ?? "");
    case "execution":
      return buildExecutionNodeId(item.identifier ?? "");
    case "initiative":
      return `initiative/${item.name}`;
    case "scenario":
      return `scenario/${item.name}`;
    default:
      return `${item.entityType}/${item.name}`;
  }
}

const COLLAPSED_COUNT = 5;

function RecentlyViewedSection({ onItemClick }: { onItemClick: (nodeId: string) => void }) {
  const items = useRecentlyViewedStore((s) => s.items);
  const [expanded, setExpanded] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

  if (items.length === 0) return null;

  const visible = expanded ? items : items.slice(0, COLLAPSED_COUNT);
  const hasMore = items.length > COLLAPSED_COUNT;

  return (
    <div className="mb-3">
      <button
        type="button"
        onClick={() => setCollapsed(!collapsed)}
        className="mb-1.5 flex w-full items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider text-slate-400 hover:text-slate-300"
      >
        {collapsed ? <ChevronRight className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
        <Clock className="h-3 w-3" />
        Recently Viewed
      </button>
      {!collapsed && (
        <div className="space-y-1">
          {visible.map((item) => {
            const nodeId = recentItemToNodeId(item);
            const typeColor = entityTypeColors[item.entityType] ?? "bg-slate-700/60 text-slate-300";
            return (
              <button
                key={nodeId}
                type="button"
                onClick={() => onItemClick(nodeId)}
                className="w-full rounded-lg border border-slate-800/60 bg-slate-900/30 p-2 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
              >
                <div className="flex items-start justify-between gap-2">
                  <p className="line-clamp-1 text-[13px] font-medium leading-snug text-slate-100">
                    {item.label}
                  </p>
                  <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", typeColor)}>
                    {item.entityType}
                  </span>
                </div>
                <p className="mt-0.5 text-[11px] text-slate-500">{formatRelativeTime(item.viewedAt)}</p>
              </button>
            );
          })}
          {hasMore && (
            <button
              type="button"
              onClick={() => setExpanded(!expanded)}
              className="w-full py-1 text-center text-[11px] text-slate-500 hover:text-slate-300"
            >
              {expanded ? "Show less" : `Show all (${items.length})`}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

function getSearchableText(item: FeedItem): [string, string | undefined] {
  if (item.type === "capture") {
    return [item.capture.text, undefined];
  }
  return [item.item.title || item.item.name, item.item.description];
}

export function ActivityTab({ feed, searchQuery, onItemClick, onClearSearch }: ActivityTabProps) {
  const filtered = searchQuery
    ? feed.filter((item) => {
        const [primary, secondary] = getSearchableText(item);
        return matchesSearch(searchQuery, primary, secondary);
      })
    : feed;

  if (filtered.length === 0) {
    return (
      <SidebarEmptyState
        icon={Inbox}
        title="No feed items available."
        hint="Captures, attention items, and recent backlog activity show up here."
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <div className="space-y-1.5">
      {!searchQuery && <RecentlyViewedSection onItemClick={onItemClick} />}
      {filtered.map((item, index) => {
        const nodeId =
          item.type === "capture"
            ? `capture/${item.capture.id}`
            : buildBacklogNodeId(item.item.kind, item.item.name);

        return (
          <FeedItemCard
            key={`${nodeId}-${index}`}
            item={item}
            onClick={() => onItemClick(nodeId)}
          />
        );
      })}
    </div>
  );
}
