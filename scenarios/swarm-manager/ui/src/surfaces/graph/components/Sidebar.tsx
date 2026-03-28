/**
 * Sidebar - Activity-first unified feed with entity-type filter toggles.
 *
 * Fully collapsible to 0px with a floating toggle button.
 * On mobile: overlays as a panel below the header (top-14).
 * Collapse state persisted in localStorage via graph-ui-store.
 */

import { useMemo } from "react";
import { PanelLeft, X, Lightbulb, Package, Zap, MessageSquare, Activity, Target } from "lucide-react";
import { cn } from "../../../lib/utils";
import { formatRelativeTime } from "../../../lib/format-utils";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { EntityType } from "../stores/graph-data-store";
import type { FeedItem } from "../../../lib/feed";

interface SidebarProps {
  feed: FeedItem[];
  onItemClick: (nodeId: string) => void;
}

const ENTITY_FILTER_CONFIG: Array<{ type: EntityType; label: string; icon: React.ElementType }> = [
  { type: "backlog", label: "Backlog", icon: Lightbulb },
  { type: "scenario", label: "Scenarios", icon: Package },
  { type: "execution", label: "Execution", icon: Zap },
  { type: "capture", label: "Captures", icon: MessageSquare },
  { type: "agent-run", label: "Runs", icon: Activity },
  { type: "initiative", label: "Initiatives", icon: Target },
];

function FeedItemCard({ item, onClick }: { item: FeedItem; onClick: () => void }) {
  const label =
    item.type === "capture"
      ? item.capture.text.slice(0, 60) + (item.capture.text.length > 60 ? "..." : "")
      : item.item.title || item.item.name;

  const timestamp =
    item.type === "capture"
      ? item.capture.created
      : item.item.updated;

  const statusBadge: string =
    item.type === "capture"
      ? item.capture.status
      : item.item.status;

  const STATUS_COLORS: Record<string, string> = {
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

  const statusColor = STATUS_COLORS[statusBadge] ?? "bg-slate-700/60 text-slate-300";

  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full text-left rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 hover:bg-slate-800/60 hover:border-slate-700/80 transition-colors"
      data-testid="sidebar-feed-item"
    >
      <div className="flex items-start justify-between gap-2">
        <p className="text-[13px] font-medium text-slate-100 leading-snug line-clamp-2">{label}</p>
        <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", statusColor)}>
          {statusBadge?.replace(/_/g, " ")}
        </span>
      </div>
      {item.type === "attention" && item.reasons.length > 0 && (
        <div className="mt-1.5 flex flex-wrap gap-1">
          {item.reasons.map((reason, i) => (
            <span
              key={i}
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
  const entityFilters = useGraphDataStore((s) => s.entityFilters);
  const toggleEntityFilter = useGraphDataStore((s) => s.toggleEntityFilter);
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  // Filter feed items by active entity filters.
  const filteredFeed = useMemo(() => {
    return feed.filter((item) => {
      if (item.type === "capture") return entityFilters.capture;
      return entityFilters.backlog;
    });
  }, [feed, entityFilters]);

  if (sidebarCollapsed) {
    return (
      <button
        type="button"
        onClick={toggleSidebar}
        className="fixed left-3 top-[4.25rem] z-20 rounded-lg border border-slate-700/80 bg-slate-900/90 p-2 text-slate-400 hover:text-slate-200 hover:bg-slate-800/70 backdrop-blur-sm transition-colors shadow-lg"
        aria-label="Open sidebar"
        data-testid="sidebar-toggle-open"
      >
        <PanelLeft className="h-4 w-4" />
      </button>
    );
  }

  return (
    <>
      {/* Mobile backdrop */}
      <button
        type="button"
        className="md:hidden fixed inset-0 top-14 z-20 bg-black/40 backdrop-blur-[2px]"
        aria-label="Close sidebar"
        onClick={toggleSidebar}
      />

      <aside
        className={cn(
          "flex flex-col border-r border-slate-200/20 bg-slate-950",
          // Desktop: part of layout flow, full height of body area
          "md:relative md:w-80 md:shrink-0",
          // Mobile: fixed overlay below header, full width up to 320px
          "fixed top-14 bottom-0 left-0 z-30 w-[85vw] max-w-[320px]",
          "shadow-2xl md:shadow-none",
        )}
        data-testid="sidebar"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200/20 px-3 py-2.5 shrink-0">
          <h2 className="text-sm font-semibold text-slate-100">Activity Feed</h2>
          <button
            type="button"
            onClick={toggleSidebar}
            className="rounded p-1 text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
            aria-label="Collapse sidebar"
            data-testid="sidebar-toggle-close"
          >
            <X className="h-4 w-4 md:hidden" />
            <PanelLeft className="h-4 w-4 hidden md:block" />
          </button>
        </div>

        {/* Entity type filter toggles */}
        <div className="flex flex-wrap gap-1 border-b border-slate-200/20 px-2.5 py-2 shrink-0">
          {ENTITY_FILTER_CONFIG.map(({ type, label, icon: Icon }) => (
            <button
              key={type}
              type="button"
              onClick={() => toggleEntityFilter(type)}
              className={cn(
                "flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium transition-colors",
                entityFilters[type]
                  ? "bg-cyan-500/15 text-cyan-300 border border-cyan-500/30"
                  : "bg-slate-800/50 text-slate-500 border border-slate-700/50",
              )}
              data-testid={`filter-toggle-${type}`}
            >
              <Icon className="h-3 w-3" />
              {label}
            </button>
          ))}
        </div>

        {/* Feed items */}
        <div className="flex-1 overflow-y-auto p-2.5 space-y-1.5">
          {filteredFeed.length === 0 ? (
            <p className="text-center text-sm text-slate-500 py-8">
              No items match your filters.
            </p>
          ) : (
            filteredFeed.map((item, idx) => {
              const nodeId =
                item.type === "capture"
                  ? `capture/${item.capture.id}`
                  : `${item.item.kind}/${item.item.name}`;
              return (
                <FeedItemCard
                  key={nodeId + idx}
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
