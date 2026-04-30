/**
 * HistoryTab renders archived sandboxes (approved, rejected, deleted).
 * Source: GET /sandboxes/history with full filter set and server-side
 * pagination.
 *
 * Each row maps a `DiffArchive` to a `Sandbox`-shaped object so it can
 * reuse `SandboxItem`. The shape adapter never invents data — fields
 * the archive doesn't carry are left empty/zero so the UI shows a blank
 * cell rather than misleading content.
 */

import { useMemo } from "react";
import { Archive, Loader2 } from "lucide-react";

import type { DiffArchive, Sandbox } from "../../lib/api";
import { formatBytes, formatRelativeTime } from "../../lib/api";
import {
  type HistoryFilters,
  type HistorySortField,
  type SortConfig,
} from "./types";
import { SandboxItem } from "./SandboxItem";
import { SELECTORS } from "../../consts/selectors";

interface HistoryTabProps {
  archives: DiffArchive[];
  selectedId?: string;
  onSelect: (archive: DiffArchive, asSandbox: Sandbox) => void;
  isLoading: boolean;
  isError: boolean;
  errorMessage?: string;
  filters: HistoryFilters;
  sort: SortConfig<HistorySortField>;
  totalCount: number;
}

/** Adapt a `DiffArchive` to a `Sandbox`-shaped object for SandboxItem. */
function archiveAsSandbox(archive: DiffArchive): Sandbox {
  return {
    id: archive.sandboxId,
    name: undefined,
    scopePath: archive.projectRoot,
    reservedPath: archive.projectRoot,
    reservedPaths: [],
    noLock: false,
    projectRoot: archive.projectRoot,
    owner: archive.owner,
    ownerType: "agent",
    status: archive.sandboxStatus,
    errorMessage: undefined,
    createdAt: archive.snapshotAt,
    lastUsedAt: archive.snapshotAt,
    driverId: "",
    driverVersion: "",
    sizeBytes: archive.totalBlobBytes,
    fileCount: archive.files?.length ?? 0,
    activePids: [],
    sessionCount: 0,
  };
}

export function HistoryTab({
  archives,
  selectedId,
  onSelect,
  isLoading,
  isError,
  errorMessage,
  totalCount,
}: HistoryTabProps) {
  const items = useMemo(
    () => archives.map((a) => ({ archive: a, sandbox: archiveAsSandbox(a) })),
    [archives],
  );

  if (isLoading && items.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-slate-500" data-testid="sidebar-history-loading">
        <Loader2 className="h-4 w-4 animate-spin mr-2" />
        <span className="text-xs">Loading history...</span>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="px-3 py-6 text-center" data-testid="sidebar-history-error">
        <p className="text-sm text-red-400">Failed to load history</p>
        {errorMessage && <p className="text-xs text-slate-500 mt-1">{errorMessage}</p>}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div
        className="flex flex-col items-center justify-center py-12 text-center"
        data-testid={SELECTORS.emptyState}
      >
        <Archive className="h-10 w-10 text-slate-700 mb-4" />
        <p className="text-sm text-slate-400">No archived sandboxes</p>
        <p className="text-xs text-slate-500 mt-1">
          Approved, rejected, and deleted sandboxes appear here.
        </p>
      </div>
    );
  }

  return (
    <div className="px-2 py-2" data-testid="sidebar-history-tab">
      <ul className="divide-y divide-slate-800/30" role="list" data-testid="sidebar-history-list">
        {items.map(({ archive, sandbox }) => (
          <SandboxItem
            key={archive.sandboxId}
            sandbox={sandbox}
            selected={selectedId === archive.sandboxId}
            onSelect={() => onSelect(archive, sandbox)}
            trailingMeta={
              <span className="flex items-center gap-1 text-slate-500" title={archive.snapshotAt}>
                <Archive className="h-3 w-3" />
                {formatRelativeTime(archive.snapshotAt)} · {formatBytes(archive.totalBlobBytes)}
              </span>
            }
          />
        ))}
      </ul>
      {totalCount > items.length && (
        <p className="text-[10px] text-slate-500 text-center pt-2" data-testid="sidebar-history-truncated">
          Showing {items.length} of {totalCount}. Use filters to narrow results.
        </p>
      )}
    </div>
  );
}
