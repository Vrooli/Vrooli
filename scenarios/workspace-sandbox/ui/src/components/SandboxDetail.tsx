import { useState, useRef, useEffect } from "react";
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
  Terminal,
  MousePointerClick,
  ChevronDown,
  ChevronRight,
  Tag,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { ScrollArea } from "./ui/scroll-area";
import { DiffViewer, type HunkSelection } from "./DiffViewer";
import type { Sandbox, DiffResult, Status, ViewMode } from "../lib/api";
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
  onOverrideAcceptance?: () => void;
  onReject: () => void;
  onDelete: () => void;
  onDiscardFile?: (fileId: string) => void;
  onLaunchAgent?: () => void;
  onApproveSelected?: (options: {
    hunkRanges: Array<{ fileId: string; startLine: number; endLine: number }>;
  }) => void;
  isApproving: boolean;
  isRejecting: boolean;
  isStopping: boolean;
  isStarting: boolean;
  isDeleting: boolean;
  isDiscarding?: boolean;
  // Review mode state (lifted to parent)
  isReviewMode: boolean;
  onReviewModeChange: (enabled: boolean) => void;
  selectedFileIds: string[];
  onSelectedFileIdsChange: (ids: string[]) => void;
  selectedHunks: HunkSelection[];
  onSelectedHunksChange: (hunks: HunkSelection[]) => void;
  // View mode props
  viewMode?: ViewMode;
  onViewModeChange?: (mode: ViewMode) => void;
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
  onOverrideAcceptance,
  onReject,
  onDelete,
  onDiscardFile,
  onLaunchAgent,
  onApproveSelected,
  isApproving,
  isRejecting,
  isStopping,
  isStarting,
  isDeleting,
  isDiscarding,
  isReviewMode,
  onReviewModeChange,
  selectedFileIds,
  onSelectedFileIdsChange,
  selectedHunks,
  onSelectedHunksChange,
  viewMode,
  onViewModeChange,
}: SandboxDetailProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showApproveConfirm, setShowApproveConfirm] = useState(false);
  const [showApproveAllConfirm, setShowApproveAllConfirm] = useState(false);
  const [showRejectConfirm, setShowRejectConfirm] = useState(false);

  // Check if any hunks are selected (file selection is now derived from hunk selection)
  const hasSelection = selectedHunks.length > 0;

  // Collapsed state for details section (persisted in localStorage)
  const [isDetailsCollapsed, setIsDetailsCollapsed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem("wsb.detailsCollapsed") === "true";
  });

  // Persist collapsed state to localStorage
  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("wsb.detailsCollapsed", String(isDetailsCollapsed));
  }, [isDetailsCollapsed]);

  // Header resize state
  const HEADER_MIN_HEIGHT = 200;
  const DIFF_MIN_HEIGHT = 200;
  const [headerHeight, setHeaderHeight] = useState(() => {
    if (typeof window === "undefined") return 400;
    const stored = Number(localStorage.getItem("wsb.detailsHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 400;
  });
  const [isResizingHeader, setIsResizingHeader] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const headerResize = useRef<{ top: number; height: number } | null>(null);

  // Header resize handler
  const handleHeaderResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!containerRef.current) return;

    const rect = containerRef.current.getBoundingClientRect();
    headerResize.current = {
      top: rect.top,
      height: rect.height,
    };
    setIsResizingHeader(true);
  };

  // Header resize mouse events
  useEffect(() => {
    if (!isResizingHeader) return;

    const handleMove = (event: MouseEvent) => {
      if (!headerResize.current) return;
      const nextHeight = event.clientY - headerResize.current.top;
      const maxHeight = headerResize.current.height - DIFF_MIN_HEIGHT;
      const clampedHeight = Math.max(HEADER_MIN_HEIGHT, Math.min(maxHeight, nextHeight));
      setHeaderHeight(clampedHeight);
    };

    const handleUp = () => {
      setIsResizingHeader(false);
      headerResize.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "row-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizingHeader]);

  // Persist header height to localStorage
  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("wsb.detailsHeight", String(headerHeight));
  }, [headerHeight]);

  // Constrain header height when container shrinks
  useEffect(() => {
    if (!containerRef.current || typeof ResizeObserver === "undefined") return;

    const clamp = () => {
      if (!containerRef.current) return;
      const height = containerRef.current.clientHeight;
      const maxHeader = Math.max(HEADER_MIN_HEIGHT, height - DIFF_MIN_HEIGHT);
      if (headerHeight > maxHeader) {
        setHeaderHeight(maxHeader);
      }
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, [headerHeight]);

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
  // When noLock is true, acceptance rules don't apply - show simplified "Approve All" button
  const isNoLock = sandbox.noLock === true;

  return (
    <div className="h-full flex flex-col" data-testid={SELECTORS.detailPanel} ref={containerRef}>
      {/* Details Panel */}
      <div
        className="flex-shrink-0"
        style={isDetailsCollapsed ? undefined : { height: headerHeight }}
      >
        <Card className={isDetailsCollapsed ? "" : "h-full flex flex-col"}>
          <CardHeader
            className="flex-row items-center justify-between space-y-0 py-3 cursor-pointer hover:bg-slate-800/30 transition-colors"
            onClick={() => setIsDetailsCollapsed(!isDetailsCollapsed)}
            data-testid={SELECTORS.detailsCollapseToggle}
          >
            <CardTitle className="flex items-center gap-2">
              {isDetailsCollapsed ? (
                <ChevronRight className="h-4 w-4 text-slate-500" />
              ) : (
                <ChevronDown className="h-4 w-4 text-slate-500" />
              )}
              <Box className="h-4 w-4 text-slate-500" />
              Details
            </CardTitle>
            <div className="flex items-center gap-2">
              <Badge variant={statusConfig.variant}>
                <span className="flex items-center gap-1.5">
                  {statusConfig.icon}
                  {statusConfig.label}
                </span>
              </Badge>
              {/* Show sandbox path summary in header when collapsed */}
              {isDetailsCollapsed && (
                <span className="text-xs text-slate-500 font-mono truncate max-w-[200px]">
                  {(() => {
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
                    return reserved[0] || "/";
                  })()}
                </span>
              )}
            </div>
          </CardHeader>

          {!isDetailsCollapsed && (
          <CardContent className="flex-1 p-0 overflow-hidden">
            <ScrollArea className="h-full px-3 py-3">
              {/* Sandbox Path & ID */}
              <div className="mb-3 pb-3 border-b border-slate-800">
                <div className="flex items-center gap-2 text-sm text-slate-200">
                  <FolderOpen className="h-4 w-4 text-slate-500" />
                  <span className="font-medium truncate">
                    {(() => {
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
                    })()}
                  </span>
                </div>
                <div className="mt-1 font-mono text-xs text-slate-500 pl-6">
                  {sandbox.id}
                </div>
              </div>

              {/* Metadata */}
              <div>
            <MetadataRow
              icon={<FolderOpen className="h-3.5 w-3.5" />}
              label="Reserved"
              value={(() => {
                if (
                  sandbox.noLock &&
                  (!sandbox.reservedPaths || sandbox.reservedPaths.length === 0) &&
                  !sandbox.reservedPath
                ) {
                  return "No lock";
                }
                const reserved = sandbox.reservedPaths?.length
                  ? sandbox.reservedPaths
                  : sandbox.reservedPath
                  ? [sandbox.reservedPath]
                  : sandbox.scopePath
                  ? [sandbox.scopePath]
                  : [];
                if (reserved.length === 0) return "Not specified";
                return reserved.join(", ");
              })()}
              copyable={
                !(
                  sandbox.noLock &&
                  (!sandbox.reservedPaths || sandbox.reservedPaths.length === 0) &&
                  !sandbox.reservedPath
                ) &&
                !!(
                  (sandbox.reservedPaths && sandbox.reservedPaths.length > 0) ||
                  sandbox.reservedPath ||
                  sandbox.scopePath
                )
              }
            />
            {sandbox.name && (
              <MetadataRow
                icon={<Tag className="h-3.5 w-3.5" />}
                label="Name"
                value={sandbox.name}
              />
            )}
            <MetadataRow
              icon={<FolderOpen className="h-3.5 w-3.5" />}
              label="Scope"
              value={sandbox.scopePath || "Not specified"}
              copyable={!!sandbox.scopePath}
            />
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
            </ScrollArea>
          </CardContent>
          )}

          {/* Actions - always visible, even when collapsed */}
          {(sandbox.status !== "deleted") && (
            <div
              className="px-3 py-3 border-t border-slate-800 flex flex-wrap gap-2"
              onClick={(e) => e.stopPropagation()}
            >
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

              {sandbox.status === "active" && onLaunchAgent && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onLaunchAgent}
                  data-testid={SELECTORS.launchAgentButton}
                >
                  <Terminal className="h-3.5 w-3.5 mr-1.5" />
                  Launch Agent
                </Button>
              )}

              {/* Review mode toggle */}
              {canApproveReject && onApproveSelected && (
                <Button
                  variant={isReviewMode ? "default" : "outline"}
                  size="sm"
                  onClick={() => {
                    const newMode = !isReviewMode;
                    onReviewModeChange(newMode);
                    if (!newMode) {
                      // Clear selections when exiting review mode
                      onSelectedFileIdsChange([]);
                      onSelectedHunksChange([]);
                    }
                  }}
                  data-testid="selection-mode-toggle"
                >
                  <MousePointerClick className="h-3.5 w-3.5 mr-1.5" />
                  {isReviewMode ? "Exit Review" : "Review"}
                </Button>
              )}

              {/* Approve Selected button - shows when hunks are selected */}
              {canApproveReject && isReviewMode && hasSelection && onApproveSelected && (
                <Button
                  variant="success"
                  size="sm"
                  onClick={() => {
                    onApproveSelected({
                      hunkRanges: selectedHunks.map((h) => ({
                        fileId: h.fileId,
                        startLine: h.startLine,
                        endLine: h.endLine,
                      })),
                    });
                    // Clear selections after approval
                    onSelectedFileIdsChange([]);
                    onSelectedHunksChange([]);
                  }}
                  disabled={isApproving}
                  data-testid="approve-selected-button"
                >
                  {isApproving ? (
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                  ) : (
                    <CheckCircle className="h-3.5 w-3.5 mr-1.5" />
                  )}
                  Approve Selected ({selectedHunks.length} {selectedHunks.length === 1 ? "hunk" : "hunks"})
                </Button>
              )}

              {canApproveReject && (
                <>
                  {/* When noLock is true, show single "Approve All" button */}
                  {isNoLock ? (
                    showApproveAllConfirm ? (
                      <div className="flex items-center gap-1">
                        <span className="text-xs text-slate-400 mr-1">Approve all changes?</span>
                        <Button
                          variant="success"
                          size="sm"
                          onClick={() => {
                            (onOverrideAcceptance || onApprove)();
                            setShowApproveAllConfirm(false);
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
                          onClick={() => setShowApproveAllConfirm(false)}
                          data-testid={SELECTORS.cancelAction}
                        >
                          No
                        </Button>
                      </div>
                    ) : (
                      <Button
                        variant="success"
                        size="sm"
                        onClick={() => setShowApproveAllConfirm(true)}
                        disabled={isApproving}
                        data-testid={SELECTORS.approveButton}
                      >
                        <CheckCircle className="h-3.5 w-3.5 mr-1.5" />
                        Approve All
                      </Button>
                    )
                  ) : (
                    <>
                      {/* Approve reserved changes (default) */}
                      {showApproveConfirm ? (
                        <div className="flex items-center gap-1">
                          <span className="text-xs text-slate-400 mr-1">Approve accepted changes?</span>
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
                          onClick={() => {
                            setShowApproveAllConfirm(false);
                            setShowApproveConfirm(true);
                          }}
                          disabled={isApproving}
                          data-testid={SELECTORS.approveButton}
                        >
                          <CheckCircle className="h-3.5 w-3.5 mr-1.5" />
                          Approve Accepted
                        </Button>
                      )}

                      {/* Override acceptance rules */}
                      {onOverrideAcceptance &&
                        (showApproveAllConfirm ? (
                          <div className="flex items-center gap-1">
                            <span className="text-xs text-slate-400 mr-1">Override acceptance rules?</span>
                            <Button
                              variant="success"
                              size="sm"
                              onClick={() => {
                                onOverrideAcceptance();
                                setShowApproveAllConfirm(false);
                              }}
                              disabled={isApproving}
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
                              onClick={() => setShowApproveAllConfirm(false)}
                            >
                              No
                            </Button>
                          </div>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              setShowApproveConfirm(false);
                              setShowApproveAllConfirm(true);
                            }}
                            disabled={isApproving}
                          >
                            <CheckCircle className="h-3.5 w-3.5 mr-1.5" />
                            Override Acceptance
                          </Button>
                        ))}
                    </>
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
        </Card>
      </div>

      {/* Header/Diff Resize Handle - only show when details expanded */}
      {!isDetailsCollapsed && (
        <div
          className="h-1.5 bg-slate-900 hover:bg-slate-700 cursor-row-resize flex-shrink-0"
          onMouseDown={handleHeaderResizeStart}
        />
      )}

      {/* Diff Viewer */}
      <div className="flex-1 min-h-0">
        <DiffViewer
          diff={diff}
          isLoading={isDiffLoading}
          error={diffError}
          showFileActions={canApproveReject && !!onDiscardFile}
          onRejectFile={onDiscardFile}
          // File selection props for partial approval
          showFileSelection={isReviewMode && canApproveReject}
          selectedFiles={selectedFileIds}
          onFileSelectionChange={onSelectedFileIdsChange}
          // Hunk selection props for partial approval
          showHunkSelection={isReviewMode && canApproveReject}
          selectedHunks={selectedHunks}
          onHunkSelectionChange={onSelectedHunksChange}
          // View mode props
          viewMode={viewMode}
          onViewModeChange={onViewModeChange}
        />
      </div>
    </div>
  );
}
