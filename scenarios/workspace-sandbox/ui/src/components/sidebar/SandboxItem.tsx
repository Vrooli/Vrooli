/**
 * SandboxItem renders a single row in either tab's list.
 *
 * The row is intentionally read-only at this level; mutating actions
 * (restart, approve, reject, delete) live on the detail panel for the
 * selected sandbox. The Active tab still surfaces the inline mount-
 * health "Restart" affordance because that's the one mutation needed
 * before a user can even open the detail panel for a degraded mount.
 */

import { memo, useCallback, useState } from "react";
import {
  AlertCircle,
  Box,
  CheckCircle,
  Clock,
  FolderOpen,
  HardDrive,
  Info,
  Loader2,
  PauseCircle,
  Play,
  RefreshCw,
  Square,
  XCircle,
} from "lucide-react";

import type { Sandbox, Status } from "../../lib/api";
import { formatBytes, formatRelativeTime } from "../../lib/api";
import { formatOwner, sandboxDisplayName, truncatePath } from "../../lib/utils";
import { SELECTORS } from "../../consts/selectors";

const PATH_MAX_LENGTH = 35;

const MOUNT_INFO_TEXT =
  "Sandboxes use OverlayFS to isolate changes. " +
  "Mounts are lost when the API restarts or the system reboots. " +
  "Your data is safe — stopping then starting the sandbox re-creates the mount.";

const STATUS_ICON: Record<Status, React.ReactNode> = {
  creating: <Loader2 className="h-3.5 w-3.5 text-blue-400 animate-spin" />,
  active: <Play className="h-3.5 w-3.5 text-emerald-400" />,
  stopped: <Square className="h-3.5 w-3.5 text-amber-400" />,
  checkpointed: <PauseCircle className="h-3.5 w-3.5 text-cyan-400" />,
  approved: <CheckCircle className="h-3.5 w-3.5 text-green-400" />,
  rejected: <XCircle className="h-3.5 w-3.5 text-red-400" />,
  deleted: <Box className="h-3.5 w-3.5 text-slate-500" />,
  error: <AlertCircle className="h-3.5 w-3.5 text-red-400" />,
};

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

interface SandboxItemProps {
  sandbox: Sandbox;
  selected: boolean;
  onSelect: (sandbox: Sandbox) => void;
  /** When true, render the inline mount-health restart affordance.
   *  History items don't need this. */
  showMountActions?: boolean;
  /** Mount-health messages that have already been consolidated into a
   *  list-level banner; the inline warning is suppressed for these. */
  consolidatedMessages?: Set<string>;
  onRestartSandbox?: (sandboxId: string) => void;
  isRestarting?: boolean;
  /** Additional metadata pill — used by History items to show snapshot
   *  age separately from createdAt. */
  trailingMeta?: React.ReactNode;
}

export const SandboxItem = memo(SandboxItemInner);

function SandboxItemInner({
  sandbox,
  selected,
  onSelect,
  showMountActions = false,
  consolidatedMessages,
  onRestartSandbox,
  isRestarting = false,
  trailingMeta,
}: SandboxItemProps) {
  const [inlineInfoOpen, setInlineInfoOpen] = useState(false);

  const handleRestart = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onRestartSandbox?.(sandbox.id);
    },
    [onRestartSandbox, sandbox.id],
  );

  const toggleInlineInfo = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    setInlineInfoOpen((v) => !v);
  }, []);

  const displayName = sandboxDisplayName(sandbox);
  const pathStr = reservedPathDisplay(sandbox);
  const mountMsg =
    showMountActions && sandbox.mountHealth && !sandbox.mountHealth.healthy
      ? sandbox.mountHealth.hint || sandbox.mountHealth.error || "Mount unhealthy"
      : null;
  const isMountConsolidated = mountMsg && consolidatedMessages ? consolidatedMessages.has(mountMsg) : false;

  return (
    <li
      className={`group flex flex-col px-2 py-2 rounded cursor-pointer transition-colors touch-target ${
        selected
          ? "bg-emerald-950/40 border-l-4 border-l-emerald-500"
          : "hover:bg-slate-800/50 border-l-4 border-l-transparent"
      }`}
      data-testid={SELECTORS.sandboxItem}
      data-sandbox-id={sandbox.id}
      data-sandbox-status={sandbox.status}
      onClick={() => onSelect(sandbox)}
    >
      {/* Status + display name */}
      <div className="flex items-center gap-2 min-w-0">
        <span aria-hidden className="flex-shrink-0">
          {STATUS_ICON[sandbox.status]}
        </span>
        <FolderOpen className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
        <span className="text-xs font-medium text-slate-200 truncate">{displayName}</span>
      </div>

      {/* Reserved path (only if it differs from display name) */}
      {pathStr !== displayName && pathStr !== "No lock" && (
        <div className="flex items-center gap-2 mt-0.5 min-w-0 pl-5">
          <span className="font-mono text-[10px] text-slate-500 truncate" title={pathStr}>
            {truncatePath(pathStr, PATH_MAX_LENGTH)}
          </span>
        </div>
      )}

      {/* Metadata row */}
      <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-500">
        {sandbox.owner && (
          <span className="truncate max-w-[100px]" title={sandbox.owner}>
            {formatOwner(sandbox.owner, sandbox.ownerType)}
          </span>
        )}
        <span className="flex items-center gap-1">
          <HardDrive className="h-3 w-3" />
          {formatBytes(sandbox.sizeBytes)}
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {formatRelativeTime(sandbox.createdAt)}
        </span>
        {trailingMeta}
      </div>

      {sandbox.errorMessage && (
        <div className="mt-1.5 text-xs text-red-400 truncate" title={sandbox.errorMessage}>
          {sandbox.errorMessage}
        </div>
      )}

      {mountMsg && !isMountConsolidated && (
        <div className="mt-1.5" data-testid="mount-warning-inline">
          <div className="flex items-center gap-1.5 text-xs text-amber-400">
            <AlertCircle className="h-3 w-3 flex-shrink-0" />
            <span className="truncate">{mountMsg}</span>
            <button
              className="ml-auto flex-shrink-0 p-0.5 rounded hover:bg-amber-900/40 transition-colors"
              onClick={toggleInlineInfo}
              data-testid="mount-info-button"
              aria-label="Mount warning details"
            >
              <Info className="h-3 w-3" />
            </button>
            {onRestartSandbox && (
              <button
                className="flex-shrink-0 flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/40 hover:bg-amber-900/60 transition-colors disabled:opacity-50"
                onClick={handleRestart}
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
          {inlineInfoOpen && (
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
}

export { MOUNT_INFO_TEXT };
