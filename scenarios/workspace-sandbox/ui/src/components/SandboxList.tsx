import { useState, useMemo, useCallback } from "react";
import {
  Box,
  ChevronDown,
  ChevronRight,
  Clock,
  HardDrive,
  Info,
  Loader2,
  Play,
  RefreshCw,
  Square,
  CheckCircle,
  XCircle,
  AlertCircle,
  FolderOpen,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import type { Sandbox, Status } from "../lib/api";
import { formatBytes, formatRelativeTime } from "../lib/api";
import { sandboxDisplayName, formatOwner, truncatePath } from "../lib/utils";
import { SELECTORS } from "../consts/selectors";

/** Max display length for paths in the sandbox list */
const PATH_MAX_LENGTH = 35;

/** Explanation shown when the user taps the info button on a mount warning */
const MOUNT_INFO_TEXT =
  "Sandboxes use OverlayFS to isolate changes. " +
  "Mounts are lost when the API restarts or the system reboots. " +
  "Your data is safe — stopping then starting the sandbox re-creates the mount.";

interface SandboxListProps {
  sandboxes: Sandbox[];
  selectedId?: string;
  onSelect: (sandbox: Sandbox) => void;
  isLoading: boolean;
  /** Called to stop then start a single sandbox (remount). */
  onRestartSandbox?: (sandboxId: string) => void;
  /** Called to stop then start all sandboxes whose mount is unhealthy. */
  onRestartUnhealthy?: () => void;
  /** Set of sandbox IDs currently being restarted. */
  restartingIds?: Set<string>;
}

interface SandboxGroupProps {
  title: string;
  status: Status;
  sandboxes: Sandbox[];
  selectedId?: string;
  onSelect: (sandbox: Sandbox) => void;
  icon: React.ReactNode;
  defaultExpanded?: boolean;
  /** Mount health messages that have been consolidated into the top banner */
  consolidatedMessages: Set<string>;
  /** Called to restart a single sandbox. */
  onRestartSandbox?: (sandboxId: string) => void;
  /** Set of sandbox IDs currently being restarted. */
  restartingIds?: Set<string>;
}

const STATUS_CONFIG: Record<Status, { icon: React.ReactNode; label: string }> = {
  creating: { icon: <Loader2 className="h-3.5 w-3.5 text-blue-400 animate-spin" />, label: "Creating" },
  active: { icon: <Play className="h-3.5 w-3.5 text-emerald-400" />, label: "Active" },
  stopped: { icon: <Square className="h-3.5 w-3.5 text-amber-400" />, label: "Stopped" },
  approved: { icon: <CheckCircle className="h-3.5 w-3.5 text-green-400" />, label: "Approved" },
  rejected: { icon: <XCircle className="h-3.5 w-3.5 text-red-400" />, label: "Rejected" },
  deleted: { icon: <Box className="h-3.5 w-3.5 text-slate-500" />, label: "Deleted" },
  error: { icon: <AlertCircle className="h-3.5 w-3.5 text-red-400" />, label: "Error" },
};

/** Compute the reserved path display string for a sandbox */
function reservedPathDisplay(sandbox: Sandbox): string {
  if (
    sandbox.noLock &&
    (!sandbox.reservedPaths || sandbox.reservedPaths.length === 0) &&
    !sandbox.reservedPath
  ) {
    return "No lock";
  }
  const reserved = sandbox.reservedPaths?.length
    ? sandbox.reservedPaths
    : [sandbox.reservedPath || sandbox.scopePath || "/"];
  const head = reserved[0] || "/";
  return reserved.length > 1 ? `${head} (+${reserved.length - 1})` : head;
}

function SandboxGroup({
  title,
  status,
  sandboxes,
  selectedId,
  onSelect,
  icon,
  defaultExpanded = true,
  consolidatedMessages,
  onRestartSandbox,
  restartingIds,
}: SandboxGroupProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const [inlineInfoId, setInlineInfoId] = useState<string | null>(null);

  const handleRestart = useCallback(
    (e: React.MouseEvent, sandboxId: string) => {
      e.stopPropagation();
      onRestartSandbox?.(sandboxId);
    },
    [onRestartSandbox],
  );

  const toggleInlineInfo = useCallback((e: React.MouseEvent, sandboxId: string) => {
    e.stopPropagation();
    setInlineInfoId((prev) => (prev === sandboxId ? null : sandboxId));
  }, []);

  if (sandboxes.length === 0) return null;

  return (
    <div className="mb-3" data-testid={SELECTORS.sandboxGroup(status)}>
      <button
        className="flex items-center gap-2 w-full text-left px-2 py-1.5 hover:bg-slate-800/50 rounded transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-slate-500" />
        ) : (
          <ChevronRight className="h-3 w-3 text-slate-500" />
        )}
        {icon}
        <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
          {title}
        </span>
        <span className="text-xs text-slate-600 ml-auto">{sandboxes.length}</span>
      </button>

      {expanded && (
        <ul className="mt-1 divide-y divide-slate-800/30">
          {sandboxes.map((sandbox) => {
            const isSelected = selectedId === sandbox.id;
            const displayName = sandboxDisplayName(sandbox);
            const pathStr = reservedPathDisplay(sandbox);
            const mountMsg = sandbox.mountHealth && !sandbox.mountHealth.healthy
              ? (sandbox.mountHealth.hint || sandbox.mountHealth.error || "Mount unhealthy")
              : null;
            const isMountConsolidated = mountMsg ? consolidatedMessages.has(mountMsg) : false;
            const isRestarting = restartingIds?.has(sandbox.id) ?? false;

            return (
              <li
                key={sandbox.id}
                className={`group flex flex-col px-2 py-2 rounded cursor-pointer transition-colors touch-target ${
                  isSelected
                    ? "bg-emerald-950/40 border-l-4 border-l-emerald-500"
                    : "hover:bg-slate-800/50 border-l-4 border-l-transparent"
                }`}
                data-testid={SELECTORS.sandboxItem}
                data-sandbox-id={sandbox.id}
                onClick={() => onSelect(sandbox)}
              >
                {/* Primary label — friendly name */}
                <div className="flex items-center gap-2 min-w-0">
                  <FolderOpen className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
                  <span className="text-xs font-medium text-slate-200 truncate">
                    {displayName}
                  </span>
                </div>

                {/* Secondary — reserved path (only if different from display name) */}
                {pathStr !== displayName && pathStr !== "No lock" && (
                  <div className="flex items-center gap-2 mt-0.5 min-w-0 pl-5">
                    <span
                      className="font-mono text-[10px] text-slate-500 truncate"
                      title={pathStr}
                    >
                      {truncatePath(pathStr, PATH_MAX_LENGTH)}
                    </span>
                  </div>
                )}

                {/* Metadata row */}
                <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-500">
                  {/* Owner */}
                  {sandbox.owner && (
                    <span className="truncate max-w-[100px]" title={sandbox.owner}>
                      {formatOwner(sandbox.owner, sandbox.ownerType)}
                    </span>
                  )}

                  {/* Size */}
                  <span className="flex items-center gap-1">
                    <HardDrive className="h-3 w-3" />
                    {formatBytes(sandbox.sizeBytes)}
                  </span>

                  {/* Created time */}
                  <span className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {formatRelativeTime(sandbox.createdAt)}
                  </span>
                </div>

                {/* Error message if present */}
                {sandbox.errorMessage && (
                  <div className="mt-1.5 text-xs text-red-400 truncate" title={sandbox.errorMessage}>
                    {sandbox.errorMessage}
                  </div>
                )}

                {/* Mount health warning — only if NOT consolidated into banner */}
                {mountMsg && !isMountConsolidated && (
                  <div
                    className="mt-1.5"
                    data-testid="mount-warning-inline"
                  >
                    <div className="flex items-center gap-1.5 text-xs text-amber-400">
                      <AlertCircle className="h-3 w-3 flex-shrink-0" />
                      <span className="truncate">{mountMsg}</span>
                      <button
                        className="ml-auto flex-shrink-0 p-0.5 rounded hover:bg-amber-900/40 transition-colors"
                        onClick={(e) => toggleInlineInfo(e, sandbox.id)}
                        data-testid="mount-info-button"
                        aria-label="Mount warning details"
                      >
                        <Info className="h-3 w-3" />
                      </button>
                      {onRestartSandbox && (
                        <button
                          className="flex-shrink-0 flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/40 hover:bg-amber-900/60 transition-colors disabled:opacity-50"
                          onClick={(e) => handleRestart(e, sandbox.id)}
                          disabled={isRestarting}
                          data-testid="mount-restart-button"
                        >
                          {isRestarting ? (
                            <Loader2 className="h-2.5 w-2.5 animate-spin" />
                          ) : (
                            <RefreshCw className="h-2.5 w-2.5" />
                          )}
                          Restart
                        </button>
                      )}
                    </div>
                    {inlineInfoId === sandbox.id && (
                      <p
                        className="mt-1 text-[10px] text-amber-300/70 leading-relaxed pl-4"
                        data-testid="mount-info-text"
                      >
                        {MOUNT_INFO_TEXT}
                      </p>
                    )}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

export function SandboxList({
  sandboxes,
  selectedId,
  onSelect,
  isLoading,
  onRestartSandbox,
  onRestartUnhealthy,
  restartingIds,
}: SandboxListProps) {
  const [bannerInfoOpen, setBannerInfoOpen] = useState(false);
  // Group sandboxes by status
  const grouped = useMemo(() => {
    const groups: Record<Status, Sandbox[]> = {
      creating: [],
      active: [],
      stopped: [],
      approved: [],
      rejected: [],
      deleted: [],
      error: [],
    };

    for (const sb of sandboxes) {
      if (groups[sb.status]) {
        groups[sb.status].push(sb);
      }
    }

    // Sort each group by createdAt descending
    for (const status of Object.keys(groups) as Status[]) {
      groups[status].sort((a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      );
    }

    return groups;
  }, [sandboxes]);

  // R2: Consolidate repeated mount health warnings into banners
  const { banners, consolidatedMessages } = useMemo(() => {
    const messageCounts = new Map<string, number>();
    for (const sb of sandboxes) {
      if (sb.mountHealth && !sb.mountHealth.healthy) {
        const msg = sb.mountHealth.hint || sb.mountHealth.error || "Mount unhealthy";
        messageCounts.set(msg, (messageCounts.get(msg) || 0) + 1);
      }
    }

    const consolidated = new Set<string>();
    const bannerList: Array<{ message: string; count: number }> = [];
    for (const [msg, count] of messageCounts) {
      if (count >= 2) {
        consolidated.add(msg);
        bannerList.push({ message: msg, count });
      }
    }

    return { banners: bannerList, consolidatedMessages: consolidated };
  }, [sandboxes]);

  const hasAnySandboxes = sandboxes.length > 0;

  return (
    <Card className="h-full flex flex-col" data-testid={SELECTORS.sandboxList}>
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3">
        <CardTitle className="flex items-center gap-2">
          <Box className="h-4 w-4 text-slate-500" />
          Sandboxes
        </CardTitle>
        {isLoading && (
          <Loader2 className="h-4 w-4 animate-spin text-slate-500" />
        )}
      </CardHeader>

      <CardContent className="flex-1 p-0 overflow-hidden">
        <ScrollArea className="h-full px-2 py-2">
          {/* Consolidated mount health banners */}
          {banners.map((banner) => (
            <div
              key={banner.message}
              className="mb-3 px-3 py-2 rounded-lg bg-amber-950/30 border border-amber-800/50"
              data-testid="mount-warning-banner"
            >
              <div className="flex items-start gap-2">
                <AlertCircle className="h-4 w-4 text-amber-400 flex-shrink-0 mt-0.5" />
                <div className="min-w-0 flex-1">
                  <p className="text-xs text-amber-300">{banner.message}</p>
                  <p className="text-[10px] text-amber-400/70 mt-0.5">
                    Affects {banner.count} sandboxes
                  </p>
                </div>
                <div className="flex items-center gap-1 flex-shrink-0">
                  <button
                    className="p-1 rounded hover:bg-amber-900/40 transition-colors text-amber-400"
                    onClick={() => setBannerInfoOpen((v) => !v)}
                    data-testid="banner-info-button"
                    aria-label="Mount warning details"
                  >
                    <Info className="h-3.5 w-3.5" />
                  </button>
                  {onRestartUnhealthy && (
                    <button
                      className="flex items-center gap-1 px-2 py-1 rounded text-[11px] font-medium text-amber-300 bg-amber-900/40 hover:bg-amber-900/60 transition-colors disabled:opacity-50"
                      onClick={onRestartUnhealthy}
                      disabled={restartingIds && restartingIds.size > 0}
                      data-testid="banner-restart-all-button"
                    >
                      {restartingIds && restartingIds.size > 0 ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <RefreshCw className="h-3 w-3" />
                      )}
                      Restart all
                    </button>
                  )}
                </div>
              </div>
              {bannerInfoOpen && (
                <p
                  className="mt-2 text-[10px] text-amber-300/70 leading-relaxed pl-6"
                  data-testid="banner-info-text"
                >
                  {MOUNT_INFO_TEXT}
                </p>
              )}
            </div>
          ))}

          {/* Active and Creating - most important */}
          <SandboxGroup
            title="Active"
            status="active"
            sandboxes={[...grouped.creating, ...grouped.active]}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.active.icon}
            defaultExpanded={true}
            consolidatedMessages={consolidatedMessages}
            onRestartSandbox={onRestartSandbox}
            restartingIds={restartingIds}
          />

          {/* Stopped - needs review */}
          <SandboxGroup
            title="Stopped"
            status="stopped"
            sandboxes={grouped.stopped}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.stopped.icon}
            defaultExpanded={true}
            consolidatedMessages={consolidatedMessages}
            onRestartSandbox={onRestartSandbox}
            restartingIds={restartingIds}
          />

          {/* Error - needs attention */}
          <SandboxGroup
            title="Error"
            status="error"
            sandboxes={grouped.error}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.error.icon}
            defaultExpanded={true}
            consolidatedMessages={consolidatedMessages}
            onRestartSandbox={onRestartSandbox}
            restartingIds={restartingIds}
          />

          {/* Approved - historical */}
          <SandboxGroup
            title="Approved"
            status="approved"
            sandboxes={grouped.approved}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.approved.icon}
            defaultExpanded={false}
            consolidatedMessages={consolidatedMessages}
            onRestartSandbox={onRestartSandbox}
            restartingIds={restartingIds}
          />

          {/* Rejected - historical */}
          <SandboxGroup
            title="Rejected"
            status="rejected"
            sandboxes={grouped.rejected}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.rejected.icon}
            defaultExpanded={false}
            consolidatedMessages={consolidatedMessages}
            onRestartSandbox={onRestartSandbox}
            restartingIds={restartingIds}
          />

          {/* Empty State */}
          {!isLoading && !hasAnySandboxes && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid={SELECTORS.emptyState}>
              <Box className="h-10 w-10 text-slate-700 mb-4" />
              <p className="text-sm text-slate-400">No sandboxes yet</p>
              <p className="text-xs text-slate-500 mt-1">
                Create a sandbox to get started
              </p>
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
