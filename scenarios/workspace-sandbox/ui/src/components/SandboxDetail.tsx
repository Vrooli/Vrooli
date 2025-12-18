import { useState } from "react";
import {
  Box,
  Clock,
  HardDrive,
  User,
  FolderOpen,
  Server,
  AlertCircle,
  CheckCircle,
  XCircle,
  Square,
  Play,
  Loader2,
  Copy,
  Check,
  Trash2,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "./ui/card";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { DiffViewer } from "./DiffViewer";
import type { Sandbox, DiffResult, Status } from "../lib/api";
import { formatBytes, formatRelativeTime } from "../lib/api";
import { SELECTORS } from "../consts/selectors";

interface SandboxDetailProps {
  sandbox?: Sandbox;
  diff?: DiffResult;
  isDiffLoading: boolean;
  diffError?: Error | null;
  onStop: () => void;
  onStart: () => void;
  onApprove: () => void;
  onReject: () => void;
  onDelete: () => void;
  onDiscardFile?: (fileId: string) => void;
  isApproving: boolean;
  isRejecting: boolean;
  isStopping: boolean;
  isStarting: boolean;
  isDeleting: boolean;
  isDiscarding?: boolean;
}

const STATUS_CONFIG: Record<Status, { icon: React.ReactNode; label: string; variant: Status }> = {
  creating: {
    icon: <Loader2 className="h-4 w-4 animate-spin" />,
    label: "Creating",
    variant: "creating",
  },
  active: {
    icon: <Play className="h-4 w-4" />,
    label: "Active",
    variant: "active",
  },
  stopped: {
    icon: <Square className="h-4 w-4" />,
    label: "Stopped",
    variant: "stopped",
  },
  approved: {
    icon: <CheckCircle className="h-4 w-4" />,
    label: "Approved",
    variant: "approved",
  },
  rejected: {
    icon: <XCircle className="h-4 w-4" />,
    label: "Rejected",
    variant: "rejected",
  },
  deleted: {
    icon: <Box className="h-4 w-4" />,
    label: "Deleted",
    variant: "deleted",
  },
  error: {
    icon: <AlertCircle className="h-4 w-4" />,
    label: "Error",
    variant: "error",
  },
};

function MetadataRow({
  icon,
  label,
  value,
  copyable = false,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  copyable?: boolean;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex items-start gap-3 py-2 border-b border-slate-800/50 last:border-b-0">
      <div className="flex items-center gap-2 text-slate-500 w-24 flex-shrink-0">
        {icon}
        <span className="text-xs">{label}</span>
      </div>
      <div className="flex-1 min-w-0 flex items-center gap-2">
        <span className="text-sm text-slate-200 font-mono truncate">{value}</span>
        {copyable && (
          <button
            onClick={handleCopy}
            className="p-1 rounded hover:bg-slate-800 transition-colors flex-shrink-0"
            title="Copy to clipboard"
          >
            {copied ? (
              <Check className="h-3 w-3 text-emerald-400" />
            ) : (
              <Copy className="h-3 w-3 text-slate-500" />
            )}
          </button>
        )}
      </div>
    </div>
  );
}

export function SandboxDetail({
  sandbox,
  diff,
  isDiffLoading,
  diffError,
  onStop,
  onStart,
  onApprove,
  onReject,
  onDelete,
  onDiscardFile,
  isApproving,
  isRejecting,
  isStopping,
  isStarting,
  isDeleting,
  isDiscarding,
}: SandboxDetailProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showApproveConfirm, setShowApproveConfirm] = useState(false);
  const [showRejectConfirm, setShowRejectConfirm] = useState(false);

  // Empty state
  if (!sandbox) {
    return (
      <div
        className="h-full flex flex-col items-center justify-center text-center"
        data-testid={SELECTORS.detailEmpty}
      >
        <Box className="h-12 w-12 text-slate-700 mb-4" />
        <p className="text-lg text-slate-400">No sandbox selected</p>
        <p className="text-sm text-slate-500 mt-1">
          Select a sandbox from the list to view details
        </p>
      </div>
    );
  }

  const statusConfig = STATUS_CONFIG[sandbox.status];
  const canStop = sandbox.status === "active";
  const canStart = sandbox.status === "stopped";
  const canApproveReject = sandbox.status === "active" || sandbox.status === "stopped";
  const isTerminal =
    sandbox.status === "approved" ||
    sandbox.status === "rejected" ||
    sandbox.status === "deleted";

  return (
    <div className="h-full flex flex-col gap-4 p-4" data-testid={SELECTORS.detailPanel}>
      {/* Header Card */}
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <FolderOpen className="h-4 w-4" />
                {sandbox.scopePath || "/"}
              </CardTitle>
              <CardDescription className="font-mono text-xs mt-1">
                {sandbox.id}
              </CardDescription>
            </div>
            <Badge variant={statusConfig.variant}>
              <span className="flex items-center gap-1.5">
                {statusConfig.icon}
                {statusConfig.label}
              </span>
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="pt-0">
          {/* Metadata */}
          <div className="mt-2">
            <MetadataRow
              icon={<FolderOpen className="h-3.5 w-3.5" />}
              label="Project"
              value={sandbox.projectRoot || "Not specified"}
              copyable={!!sandbox.projectRoot}
            />
            <MetadataRow
              icon={<User className="h-3.5 w-3.5" />}
              label="Owner"
              value={sandbox.owner || "Unknown"}
            />
            <MetadataRow
              icon={<HardDrive className="h-3.5 w-3.5" />}
              label="Size"
              value={`${formatBytes(sandbox.sizeBytes)} (${sandbox.fileCount} files)`}
            />
            <MetadataRow
              icon={<Clock className="h-3.5 w-3.5" />}
              label="Created"
              value={formatRelativeTime(sandbox.createdAt)}
            />
            <MetadataRow
              icon={<Server className="h-3.5 w-3.5" />}
              label="Driver"
              value={`${sandbox.driver} v${sandbox.driverVersion}`}
            />
            {sandbox.mergedDir && sandbox.status === "active" && (
              <MetadataRow
                icon={<FolderOpen className="h-3.5 w-3.5" />}
                label="Workspace"
                value={sandbox.mergedDir}
                copyable
              />
            )}
          </div>

          {/* Error message */}
          {sandbox.errorMessage && (
            <div className="mt-3 p-3 rounded-lg bg-red-950/30 border border-red-800/50">
              <div className="flex items-start gap-2">
                <AlertCircle className="h-4 w-4 text-red-400 flex-shrink-0 mt-0.5" />
                <p className="text-sm text-red-300">{sandbox.errorMessage}</p>
              </div>
            </div>
          )}

          {/* Mount health warning */}
          {sandbox.mountHealth && !sandbox.mountHealth.healthy && (
            <div className="mt-3 p-3 rounded-lg bg-amber-950/30 border border-amber-800/50">
              <div className="flex items-start gap-2">
                <AlertCircle className="h-4 w-4 text-amber-400 flex-shrink-0 mt-0.5" />
                <div>
                  <p className="text-sm text-amber-300 font-medium">Mount Unhealthy</p>
                  {sandbox.mountHealth.error && (
                    <p className="text-xs text-amber-400/80 mt-1">{sandbox.mountHealth.error}</p>
                  )}
                  {sandbox.mountHealth.hint && (
                    <p className="text-xs text-amber-200 mt-1">{sandbox.mountHealth.hint}</p>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Actions - show workflow actions for non-terminal, delete for non-deleted */}
          {(sandbox.status !== "deleted") && (
            <div className="mt-4 flex flex-wrap gap-2">
              {canStop && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onStop}
                  disabled={isStopping}
                  data-testid={SELECTORS.stopButton}
                >
                  {isStopping ? (
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                  ) : (
                    <Square className="h-3.5 w-3.5 mr-1.5" />
                  )}
                  Stop
                </Button>
              )}

              {canStart && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onStart}
                  disabled={isStarting}
                  data-testid={SELECTORS.startButton}
                >
                  {isStarting ? (
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                  ) : (
                    <Play className="h-3.5 w-3.5 mr-1.5" />
                  )}
                  Start
                </Button>
              )}

              {canApproveReject && (
                <>
                  {/* Approve button with confirmation */}
                  {showApproveConfirm ? (
                    <div className="flex items-center gap-1">
                      <span className="text-xs text-slate-400 mr-1">Apply changes?</span>
                      <Button
                        variant="success"
                        size="sm"
                        onClick={() => {
                          onApprove();
                          setShowApproveConfirm(false);
                        }}
                        disabled={isApproving}
                        data-testid={SELECTORS.confirmApprove}
                      >
                        {isApproving ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          "Yes"
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowApproveConfirm(false)}
                        data-testid={SELECTORS.cancelAction}
                      >
                        No
                      </Button>
                    </div>
                  ) : (
                    <Button
                      variant="success"
                      size="sm"
                      onClick={() => setShowApproveConfirm(true)}
                      disabled={isApproving}
                      data-testid={SELECTORS.approveButton}
                    >
                      <CheckCircle className="h-3.5 w-3.5 mr-1.5" />
                      Approve
                    </Button>
                  )}

                  {/* Reject button with confirmation */}
                  {showRejectConfirm ? (
                    <div className="flex items-center gap-1">
                      <span className="text-xs text-slate-400 mr-1">Discard changes?</span>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => {
                          onReject();
                          setShowRejectConfirm(false);
                        }}
                        disabled={isRejecting}
                        data-testid={SELECTORS.confirmReject}
                      >
                        {isRejecting ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          "Yes"
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowRejectConfirm(false)}
                        data-testid={SELECTORS.cancelAction}
                      >
                        No
                      </Button>
                    </div>
                  ) : (
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setShowRejectConfirm(true)}
                      disabled={isRejecting}
                      data-testid={SELECTORS.rejectButton}
                    >
                      <XCircle className="h-3.5 w-3.5 mr-1.5" />
                      Reject
                    </Button>
                  )}
                </>
              )}

              {/* Delete button with confirmation - available for all non-deleted sandboxes */}
              <div className="ml-auto">
                {showDeleteConfirm ? (
                  <div className="flex items-center gap-1">
                    <span className="text-xs text-red-400 mr-1">Delete sandbox?</span>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => {
                        onDelete();
                        setShowDeleteConfirm(false);
                      }}
                      disabled={isDeleting}
                    >
                      {isDeleting ? (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      ) : (
                        "Yes"
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowDeleteConfirm(false)}
                    >
                      No
                    </Button>
                  </div>
                ) : (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowDeleteConfirm(true)}
                    disabled={isDeleting}
                    data-testid={SELECTORS.deleteButton}
                  >
                    <Trash2 className="h-3.5 w-3.5 mr-1.5" />
                    Delete
                  </Button>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Diff Viewer */}
      <div className="flex-1 min-h-0">
        <DiffViewer
          diff={diff}
          isLoading={isDiffLoading}
          error={diffError}
          showFileActions={canApproveReject && !!onDiscardFile}
          onRejectFile={onDiscardFile}
        />
      </div>
    </div>
  );
}
