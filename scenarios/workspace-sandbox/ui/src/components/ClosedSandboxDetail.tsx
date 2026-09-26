/**
 * ClosedSandboxDetail — read-only detail view for History selections.
 *
 * Renders the archive metadata, snapshot timestamp, and the same
 * `DiffViewer` used by the Active tab — but fed from the durable
 * archive (`/sandboxes/{id}/diff` resolves to the archive when the
 * sandbox status is terminal). No action buttons appear here; History
 * sandboxes are state-machine-terminal and cannot be restarted,
 * approved, or rejected.
 *
 * When `archiveState === "not_captured"` (e.g. Error → Deleted), the
 * file list is empty and we render an explicit explanation rather than
 * the generic "No changes detected" empty state.
 */

import {
  AlertCircle,
  Archive,
  Box,
  CheckCircle,
  Clock,
  FolderOpen,
  HardDrive,
  Hash,
  Tag,
  User,
  XCircle,
} from "lucide-react";

import type { DiffArchive, DiffResult } from "../lib/api";
import { formatBytes, formatRelativeTime } from "../lib/api";
import { sandboxDisplayName, formatOwner } from "../lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Badge } from "./ui/badge";
import { DiffViewer } from "./DiffViewer";
import { SELECTORS } from "../consts/selectors";

interface ClosedSandboxDetailProps {
  archive: DiffArchive;
  diff?: DiffResult;
  isDiffLoading: boolean;
  diffError?: Error | null;
}

const STATUS_ICON: Record<string, React.ReactNode> = {
  approved: <CheckCircle className="h-4 w-4" />,
  rejected: <XCircle className="h-4 w-4" />,
  deleted: <Box className="h-4 w-4" />,
};

const STATUS_LABEL: Record<string, string> = {
  approved: "Approved",
  rejected: "Rejected",
  deleted: "Deleted",
};

export function ClosedSandboxDetail({
  archive,
  diff,
  isDiffLoading,
  diffError,
}: ClosedSandboxDetailProps) {
  const displayName = sandboxDisplayName({
    id: archive.sandboxId,
    scopePath: archive.projectRoot,
  });
  const isNotCaptured = archive.archiveState === "not_captured";

  return (
    <div className="h-full flex flex-col" data-testid={SELECTORS.detailPanel}>
      {/* Metadata header */}
      <Card className="flex-shrink-0">
        <CardHeader className="flex-row items-center justify-between space-y-0 py-3">
          <CardTitle className="flex items-center gap-2">
            <Archive className="h-4 w-4 text-slate-500" />
            Archive
          </CardTitle>
          <Badge variant={archive.sandboxStatus}>
            <span className="flex items-center gap-1.5">
              {STATUS_ICON[archive.sandboxStatus]}
              {STATUS_LABEL[archive.sandboxStatus] ?? archive.sandboxStatus}
            </span>
          </Badge>
        </CardHeader>
        <CardContent className="p-0">
          <ScrollArea className="max-h-[40vh]">
            <div className="px-3 pb-3">
              <div className="mb-3 pb-3 border-b border-slate-800">
                <div className="flex items-center gap-2 text-sm text-slate-200">
                  <FolderOpen className="h-4 w-4 text-slate-500" />
                  <span className="font-medium truncate" title={archive.projectRoot}>
                    {displayName}
                  </span>
                </div>
                <div className="mt-1 font-mono text-xs text-slate-500 pl-6 break-all">
                  {archive.sandboxId}
                </div>
              </div>

              <MetadataRow
                icon={<Clock className="h-3.5 w-3.5" />}
                label="Snapshot"
                value={`${formatRelativeTime(archive.snapshotAt)} (${archive.snapshotAt})`}
              />
              {archive.owner && (
                <MetadataRow
                  icon={<User className="h-3.5 w-3.5" />}
                  label="Owner"
                  value={formatOwner(archive.owner)}
                />
              )}
              <MetadataRow
                icon={<FolderOpen className="h-3.5 w-3.5" />}
                label="Project"
                value={archive.projectRoot || "Not specified"}
                mono
              />
              <MetadataRow
                icon={<HardDrive className="h-3.5 w-3.5" />}
                label="Archive size"
                value={`${formatBytes(archive.totalBlobBytes)} (${archive.files?.length ?? 0} files)`}
              />
              {archive.agentManagerRunId && (
                <MetadataRow
                  icon={<Tag className="h-3.5 w-3.5" />}
                  label="Run ID"
                  value={archive.agentManagerRunId}
                  mono
                />
              )}
              {archive.unifiedDiffSha256 && (
                <MetadataRow
                  icon={<Hash className="h-3.5 w-3.5" />}
                  label="Diff hash"
                  value={archive.unifiedDiffSha256.slice(0, 12) + "…"}
                  mono
                />
              )}

              {isNotCaptured && (
                <div
                  className="mt-3 p-3 rounded-lg bg-slate-800/50 border border-slate-700"
                  data-testid="archive-not-captured"
                >
                  <div className="flex items-start gap-2">
                    <AlertCircle className="h-4 w-4 text-slate-400 flex-shrink-0 mt-0.5" />
                    <div className="text-xs text-slate-300">
                      <p className="font-medium">No diff captured</p>
                      <p className="text-slate-400 mt-1">
                        This sandbox transitioned terminal without a usable overlay (typically
                        Error → Deleted). The archive row exists for audit, but no file content
                        was retained.
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      {/* Diff viewer (read-only) */}
      <div className="flex-1 min-h-0 mt-2">
        <DiffViewer
          diff={diff}
          isLoading={isDiffLoading}
          error={diffError}
          showFileActions={false}
          showFileSelection={false}
          showHunkSelection={false}
        />
      </div>
    </div>
  );
}

function MetadataRow({
  icon,
  label,
  value,
  mono = false,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="flex items-start gap-3 py-2 border-b border-slate-800/50 last:border-b-0">
      <div className="flex items-center gap-2 text-slate-500 w-24 flex-shrink-0">
        {icon}
        <span className="text-xs">{label}</span>
      </div>
      <div className="flex-1 min-w-0">
        <span
          className={`text-sm text-slate-200 truncate block ${mono ? "font-mono text-xs" : ""}`}
          title={value}
        >
          {value}
        </span>
      </div>
    </div>
  );
}
