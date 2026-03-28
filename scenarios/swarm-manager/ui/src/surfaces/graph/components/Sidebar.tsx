/**
 * Sidebar - Activity feed for captures and backlog items.
 *
 * Graph configuration lives in the graph controls panel; the sidebar is kept
 * focused on feed/navigation so the feed and graph concerns do not compete.
 */

import { PanelLeft, X } from "lucide-react";
import { cn } from "../../../lib/utils";
import { formatRelativeTime } from "../../../lib/format-utils";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { buildBacklogNodeId } from "../lib/node-id-parser";
import type { FeedItem } from "../../../lib/feed";

interface SidebarProps {
  feed: FeedItem[];
  onItemClick: (nodeId: string) => void;
}

function FeedItemCard({ item, onClick }: { item: FeedItem; onClick: () => void }) {
  const label =
    item.type === "capture"
      ? item.capture.text.slice(0, 60) + (item.capture.text.length > 60 ? "..." : "")
      : item.item.title || item.item.name;

  const timestamp = item.type === "capture" ? item.capture.created : item.item.updated;
  const statusBadge = item.type === "capture" ? item.capture.status : item.item.status;

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

export function Sidebar({ feed, onItemClick }: SidebarProps) {
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  if (sidebarCollapsed) {
    return (
      <button
        type="button"
        onClick={toggleSidebar}
        className="fixed left-3 top-[4.25rem] z-20 rounded-lg border border-slate-700/80 bg-slate-900/90 p-2 text-slate-400 shadow-lg backdrop-blur-sm transition-colors hover:bg-slate-800/70 hover:text-slate-200"
        aria-label="Open sidebar"
        data-testid="sidebar-toggle-open"
      >
        <PanelLeft className="h-4 w-4" />
      </button>
    );
  }

  return (
    <>
      <button
        type="button"
        className="fixed inset-0 top-14 z-20 bg-black/40 backdrop-blur-[2px] md:hidden"
        aria-label="Close sidebar"
        onClick={toggleSidebar}
      />

      <aside
        className={cn(
          "fixed bottom-0 left-0 top-14 z-30 flex w-[85vw] max-w-[320px] flex-col border-r border-slate-200/20 bg-slate-950 shadow-2xl md:relative md:w-80 md:shrink-0 md:shadow-none",
        )}
        data-testid="sidebar"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slate-200/20 px-3 py-2.5">
          <div>
            <h2 className="text-sm font-semibold text-slate-100">Activity Feed</h2>
            <p className="text-xs text-slate-500">Captures and backlog work that need attention.</p>
          </div>
          <button
            type="button"
            onClick={toggleSidebar}
            className="rounded p-1 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
            aria-label="Collapse sidebar"
            data-testid="sidebar-toggle-close"
          >
            <X className="h-4 w-4 md:hidden" />
            <PanelLeft className="hidden h-4 w-4 md:block" />
          </button>
        </div>

        <div className="flex-1 space-y-1.5 overflow-y-auto p-2.5">
          {feed.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-500">No feed items available.</p>
          ) : (
            feed.map((item, index) => {
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
            })
          )}
        </div>
      </aside>
    </>
  );
}
