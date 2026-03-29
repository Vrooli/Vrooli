/**
 * ActivityTab - Renders the unified feed (captures + backlog items).
 *
 * Extracted from the original Sidebar component. Preserves the existing
 * FeedItemCard rendering and priority-based ordering.
 */

import { Inbox } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { buildBacklogNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import type { FeedItem } from "../../../../lib/feed";

interface ActivityTabProps {
  feed: FeedItem[];
  searchQuery: string;
  onItemClick: (nodeId: string) => void;
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

function getSearchableText(item: FeedItem): [string, string | undefined] {
  if (item.type === "capture") {
    return [item.capture.text, undefined];
  }
  return [item.item.title || item.item.name, item.item.description];
}

export function ActivityTab({ feed, searchQuery, onItemClick }: ActivityTabProps) {
  const filtered = searchQuery
    ? feed.filter((item) => {
        const [primary, secondary] = getSearchableText(item);
        return matchesSearch(searchQuery, primary, secondary);
      })
    : feed;

  if (filtered.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <Inbox className="mb-2 h-8 w-8" />
        <p className="text-sm">{searchQuery ? "No items match your search." : "No feed items available."}</p>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
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
