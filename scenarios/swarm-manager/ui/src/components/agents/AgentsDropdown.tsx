/**
 * AgentsDropdown — Reusable dropdown showing running agent activities.
 *
 * Two display variants:
 * - "button": Full-size HUD button with icon + count badge (used in graph HUD bar)
 * - "badge": Compact pill with pulsing icon + count (used in sidebar header)
 *
 * On desktop the panel opens as a popover anchored to the trigger and clamped
 * to the viewport. On mobile it opens as a full-width bottom sheet.
 */

import { useState, useRef, useEffect, useCallback } from "react";
import { Activity, Square, X } from "lucide-react";
import { cn } from "../../lib/utils";
import { formatRelativeTime } from "../../lib";
import { buildBacklogNodeId } from "../../surfaces/graph/lib/node-id-parser";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { BottomSheet } from "../ui/bottom-sheet";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface AgentsDropdownProps {
  activities: AgentActivityRecord[];
  onViewActivity: (activityId: string) => void;
  onViewBacklog: (nodeId: string) => void;
  onStopRun: (runId: string) => void;
  /** "button" = HUD-style, "badge" = compact sidebar pill. Default: "button". */
  variant?: "badge" | "button";
  /** Max concurrent executions from governance settings. When set, shows N/M format. */
  maxConcurrent?: number;
  className?: string;
}

interface ActivityListProps {
  activities: AgentActivityRecord[];
  onViewActivity: (activityId: string) => void;
  onViewBacklog: (nodeId: string) => void;
  onStopRun: (runId: string) => void;
  onItemAction: () => void;
}

function ActivityList({
  activities,
  onViewActivity,
  onViewBacklog,
  onStopRun,
  onItemAction,
}: ActivityListProps) {
  if (activities.length === 0) {
    return (
      <p className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-6 text-center text-sm text-slate-400">
        No agents are currently running.
      </p>
    );
  }

  return (
    <div className="space-y-2">
      {activities.map((activity) => {
        const backlogNodeId =
          activity.ownerType === "backlog" &&
          typeof activity.ownerKind === "string" &&
          typeof activity.ownerName === "string"
            ? buildBacklogNodeId(activity.ownerKind, activity.ownerName)
            : null;

        return (
          <div
            key={activity.activityId}
            className="rounded-lg border border-slate-800 bg-slate-900/45 p-3"
          >
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-slate-100">
                  {activity.ownerTitle ??
                    `${activity.ownerType}/${activity.ownerName}`}
                </p>
                {activity.runId && (
                  <p className="font-mono text-xs text-cyan-300">
                    {activity.runId}
                  </p>
                )}
              </div>
              <span className="rounded-full bg-cyan-500/15 px-2 py-0.5 text-[11px] text-cyan-200">
                {activity.status.replace("_", " ")}
              </span>
            </div>
            <p className="mt-1 text-xs text-slate-400">
              {activity.purpose.replace("_", " ")} • Requested{" "}
              {formatRelativeTime(activity.requestedAt)}
            </p>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <button
                type="button"
                className="rounded border border-slate-700/80 bg-slate-900/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800/70"
                onClick={() => {
                  onViewActivity(activity.activityId);
                  onItemAction();
                }}
              >
                View Activity
              </button>
              {backlogNodeId && (
                <button
                  type="button"
                  className="rounded border border-slate-700/80 bg-slate-900/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800/70"
                  onClick={() => {
                    onViewBacklog(backlogNodeId);
                    onItemAction();
                  }}
                >
                  View Backlog
                </button>
              )}
              {activity.runId && (
                <button
                  type="button"
                  className="rounded border border-red-500/40 bg-red-500/10 px-2 py-1 text-xs text-red-200 hover:bg-red-500/20"
                  onClick={() => onStopRun(activity.runId ?? "")}
                  disabled={activity.isStopping}
                >
                  <Square className="mr-1 inline h-3 w-3" />
                  {activity.isStopping ? "Stopping..." : "Stop"}
                </button>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

export function AgentsDropdown({
  activities,
  onViewActivity,
  onViewBacklog,
  onStopRun,
  variant = "button",
  maxConcurrent,
  className,
}: AgentsDropdownProps) {
  const [open, setOpen] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const isMobile = useIsMobile();
  const count = activities.length;
  const countLabel = maxConcurrent != null ? `${count}/${maxConcurrent}` : `${count}`;
  const close = useCallback(() => setOpen(false), []);

  // Clamp desktop popover panel to viewport bounds after it renders.
  const clampToViewport = useCallback(() => {
    const el = panelRef.current;
    if (!el) return;

    // Reset any previous clamp styles before re-measuring.
    el.style.transform = "";
    el.style.top = "";
    el.style.bottom = "";
    el.style.marginTop = "";
    el.style.marginBottom = "";

    const rect = el.getBoundingClientRect();
    const margin = 8;
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    // Horizontal clamp: shift right if overflowing left, shift left if overflowing right.
    let shiftX = 0;
    if (rect.left < margin) {
      shiftX = margin - rect.left;
    } else if (rect.right > viewportWidth - margin) {
      shiftX = viewportWidth - margin - rect.right;
    }
    if (shiftX !== 0) {
      el.style.transform = `translateX(${shiftX}px)`;
    }

    // Vertical clamp: flip above trigger if overflowing bottom.
    if (rect.bottom > viewportHeight - margin) {
      el.style.top = "auto";
      el.style.bottom = "100%";
      el.style.marginTop = "0";
      el.style.marginBottom = "4px";
    }
  }, []);

  useEffect(() => {
    if (!open || isMobile) return;
    clampToViewport();
    const handle = () => clampToViewport();
    window.addEventListener("resize", handle);
    window.addEventListener("scroll", handle, true);
    return () => {
      window.removeEventListener("resize", handle);
      window.removeEventListener("scroll", handle, true);
    };
  }, [open, isMobile, clampToViewport, count]);

  const trigger =
    variant === "badge" ? (
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium transition-colors",
          count > 0
            ? "bg-emerald-500/15 text-emerald-400 hover:bg-emerald-500/25"
            : "bg-slate-800/60 text-slate-400 hover:bg-slate-700/60",
        )}
        title={`${count} running agent${count !== 1 ? "s" : ""}`}
        data-testid="agents-badge"
      >
        <Activity className={cn("h-3.5 w-3.5", count > 0 && "animate-pulse")} />
        <span>{countLabel}</span>
      </button>
    ) : (
      <button
        type="button"
        className="flex items-center gap-1.5 rounded-lg border border-slate-700/60 bg-slate-900/80 px-2.5 py-1.5 text-sm text-slate-100 hover:bg-slate-800/80"
        onClick={() => setOpen((prev) => !prev)}
        data-testid="graph-agents-toggle"
      >
        <Activity className="h-4 w-4 text-cyan-300" />
        <span className="rounded-full bg-cyan-500/20 px-1.5 py-0.5 text-xs text-cyan-200">
          {countLabel}
        </span>
      </button>
    );

  const sheetDescription = `${count} active activity item${count !== 1 ? "s" : ""}`;

  return (
    <div className={cn("relative", className)}>
      {trigger}

      {/* Mobile: bottom sheet */}
      {isMobile && (
        <BottomSheet
          isOpen={open}
          onClose={close}
          title="Agents running"
          description={sheetDescription}
          data-testid="graph-agents-dropdown"
          contentClassName="max-h-[70vh]"
        >
          <ActivityList
            activities={activities}
            onViewActivity={onViewActivity}
            onViewBacklog={onViewBacklog}
            onStopRun={onStopRun}
            onItemAction={close}
          />
        </BottomSheet>
      )}

      {/* Desktop: anchored popover */}
      {!isMobile && open && (
        <>
          {/* Click-outside backdrop */}
          <button
            type="button"
            className="fixed inset-0 z-40 cursor-default bg-transparent"
            aria-label="Close agents dropdown"
            onClick={close}
          />

          <div
            ref={panelRef}
            className="absolute right-0 top-full z-50 mt-1 w-[360px] max-w-[calc(100vw-1rem)] rounded-lg border border-slate-700/80 bg-slate-950 shadow-xl"
            data-testid="graph-agents-dropdown"
          >
            <div className="flex items-center justify-between border-b border-slate-800 px-3 py-2">
              <div>
                <p className="text-sm font-semibold text-slate-100">Agents running</p>
                <p className="text-xs text-slate-400">{sheetDescription}</p>
              </div>
              <button
                type="button"
                className="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
                onClick={close}
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="max-h-80 overflow-y-auto p-2">
              <ActivityList
                activities={activities}
                onViewActivity={onViewActivity}
                onViewBacklog={onViewBacklog}
                onStopRun={onStopRun}
                onItemAction={close}
              />
            </div>
          </div>
        </>
      )}
    </div>
  );
}
