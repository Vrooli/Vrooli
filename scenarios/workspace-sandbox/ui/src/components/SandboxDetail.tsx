import { Profiler, useState, useRef, useEffect } from "react";
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
  PauseCircle,
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
import { onProfilerRender } from "../lib/profiler";
import type { Sandbox, DiffResult, Status, ViewMode } from "../lib/api";
import { formatBytes, formatRelativeTime } from "../lib/api";
import { cn, truncatePath, formatOwner, sandboxDisplayName } from "../lib/utils";
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
  /** When true, hide the DiffViewer section (used on mobile where diff has its own tab) */
  hideDiffViewer?: boolean;
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
  checkpointing: {
    icon: <PauseCircle className="h-4 w-4" />,
    label: "Checkpointing",
    variant: "checkpointed",
  },
  checkpointed: {
    icon: <PauseCircle className="h-4 w-4" />,
    label: "Checkpointed",
    variant: "checkpointed",
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

/** R9: MetadataRow with optional monospace font */
function MetadataRow({
  icon,
  label,
  value,
  copyable = false,
  mono = false,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  copyable?: boolean;
  /** Use monospace font for the value (for paths, IDs) */
  mono?: boolean;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex items-start gap-3 py-2 border-b border-slate-800/50 last:border-b-0" data-testid="metadata-row">
      <div className="flex items-center gap-2 text-slate-500 w-24 flex-shrink-0">
        {icon}
        <span className="text-xs">{label}</span>
      </div>
      <div className="flex-1 min-w-0 flex items-center gap-2">
        <span className={cn("text-sm text-slate-200 truncate", mono && "font-mono")} title={value}>
          {value}
        </span>
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

/** Compute the reserved path display string */
function reservedPathValue(sandbox: Sandbox): { display: string; copyable: boolean } {
  if (
    sandbox.noLock &&
    (!sandbox.reservedPaths || sandbox.reservedPaths.length === 0) &&
    !sandbox.reservedPath
  ) {
    return { display: "No lock", copyable: false };
  }
  const reserved = sandbox.reservedPaths?.length
    ? sandbox.reservedPaths
    : sandbox.reservedPath
    ? [sandbox.reservedPath]
    : sandbox.scopePath
    ? [sandbox.scopePath]
    : [];
  if (reserved.length === 0) return { display: "Not specified", copyable: false };
  return { display: reserved.join(", "), copyable: true };
}

export function SandboxDetail(props: SandboxDetailProps) {
  return (
    <Profiler id="SandboxDetail" onRender={onProfilerRender}>
      <SandboxDetailImpl {...props} />
    </Profiler>
  );
}

function SandboxDetailImpl({
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
  isDiscarding: _isDiscarding,
  isReviewMode,
  onReviewModeChange,
  selectedFileIds,
  onSelectedFileIdsChange,
  selectedHunks,
  onSelectedHunksChange,
  viewMode,
  onViewModeChange,
  hideDiffViewer,
}: SandboxDetailProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showApproveConfirm, setShowApproveConfirm] = useState(false);
  const [showApproveAllConfirm, setShowApproveAllConfirm] = useState(false);
  const [showRejectConfirm, setShowRejectConfirm] = useState(false);

  // R4: Show more toggle for secondary metadata
  const [showMore, setShowMore] = useState(false);

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

  // Header resize state. During a drag we write the height directly to the
  // header element via RAF; React state and localStorage update only on
  // mouseup. See docs/perf/2026-05-03-history-fileviewer-resize.md F1/F2/F8.
  const HEADER_MIN_HEIGHT = 200;
  const DIFF_MIN_HEIGHT = 200;
  const [headerHeight, setHeaderHeight] = useState(() => {
    if (typeof window === "undefined") return 400;
    const stored = Number(localStorage.getItem("wsb.detailsHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 400;
  });
  const headerHeightRef = useRef(headerHeight);
  headerHeightRef.current = headerHeight;
  const containerRef = useRef<HTMLDivElement | null>(null);
  const headerPaneRef = useRef<HTMLDivElement | null>(null);
  const headerResize = useRef<
    | { top: number; height: number; current: number }
    | null
  >(null);

  const handleHeaderResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!containerRef.current || !headerPaneRef.current) return;

    const rect = containerRef.current.getBoundingClientRect();
    const startHeight = headerPaneRef.current.getBoundingClientRect().height;
    headerResize.current = {
      top: rect.top,
      height: rect.height,
      current: startHeight,
    };

    let rafId = 0;
    let pendingHeight = startHeight;

    const applyHeight = () => {
      rafId = 0;
      if (!headerPaneRef.current || !headerResize.current) return;
      headerPaneRef.current.style.height = `${pendingHeight}px`;
      headerResize.current.current = pendingHeight;
    };

    const handleMove = (e: MouseEvent) => {
      if (!headerResize.current) return;
      const nextHeight = e.clientY - headerResize.current.top;
      const maxHeight = headerResize.current.height - DIFF_MIN_HEIGHT;
      pendingHeight = Math.max(HEADER_MIN_HEIGHT, Math.min(maxHeight, nextHeight));
      if (rafId === 0) rafId = requestAnimationFrame(applyHeight);
    };

    const handleUp = () => {
      if (rafId !== 0) {
        cancelAnimationFrame(rafId);
        applyHeight();
      }
      const settled = headerResize.current?.current ?? startHeight;
      headerResize.current = null;
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
      setHeaderHeight(settled);
      try {
        localStorage.setItem("wsb.detailsHeight", String(settled));
      } catch {
        // see App.tsx — silently skip persistence if localStorage is unusable
      }
    };

    document.body.style.cursor = "row-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);
  };

  // Constrain header height when container shrinks. Observer attaches once;
  // latest height is read via ref so we don't tear down/rebuild on each drag
  // step (F8).
  useEffect(() => {
    if (!containerRef.current || typeof ResizeObserver === "undefined") return;

    const clamp = () => {
      if (!containerRef.current) return;
      const height = containerRef.current.clientHeight;
      const maxHeader = Math.max(HEADER_MIN_HEIGHT, height - DIFF_MIN_HEIGHT);
      if (headerHeightRef.current > maxHeader) {
        setHeaderHeight(maxHeader);
      }
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, []);

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
  const canStart = sandbox.status === "stopped" || sandbox.status === "checkpointed";
  const canApproveReject = sandbox.status === "active" || sandbox.status === "stopped";
  // When noLock is true, acceptance rules don't apply - show simplified "Approve All" button
  const isNoLock = sandbox.noLock === true;
  const reserved = reservedPathValue(sandbox);
  const displayName = sandboxDisplayName(sandbox);

  // R1: Determine if we need the visual divider between lifecycle and review groups
  const hasLifecycleButtons = canStop || canStart || (sandbox.status === "active" && onLaunchAgent);
  const hasReviewButtons = canApproveReject && onApproveSelected;

  return (
    <div className="h-full flex flex-col" data-testid={SELECTORS.detailPanel} ref={containerRef}>
      {/* Details Panel */}
      <div
        ref={headerPaneRef}
        className={hideDiffViewer ? "flex-1 min-h-0" : "flex-shrink-0"}
        style={hideDiffViewer || isDetailsCollapsed ? undefined : { height: headerHeight }}
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
              {/* Show sandbox name/path summary in header when collapsed */}
              {isDetailsCollapsed && (
                <span className="text-xs text-slate-500 font-mono truncate max-w-[200px]">
                  {truncatePath(reserved.display, 30)}
                </span>
              )}
            </div>
          </CardHeader>

          {!isDetailsCollapsed && (
          <CardContent className="flex-1 p-0 overflow-hidden">
            <ScrollArea className="h-full px-3 py-3">
              {/* Sandbox display name & ID */}
              <div className="mb-3 pb-3 border-b border-slate-800">
                <div className="flex items-center gap-2 text-sm text-slate-200">
                  <FolderOpen className="h-4 w-4 text-slate-500" />
                  <span className="font-medium truncate" title={reserved.display}>
                    {displayName}
                  </span>
                </div>
                <div className="mt-1 font-mono text-xs text-slate-500 pl-6">
                  {sandbox.id}
                </div>
              </div>

              {/* Primary metadata — always visible */}
              <div>
                <MetadataRow
                  icon={<FolderOpen className="h-3.5 w-3.5" />}
                  label="Reserved"
                  value={reserved.display}
                  copyable={reserved.copyable}
                  mono
                />
                {sandbox.name && (
                  <MetadataRow
                    icon={<Tag className="h-3.5 w-3.5" />}
                    label="Name"
                    value={sandbox.name}
                  />
                )}
                <MetadataRow
                  icon={<User className="h-3.5 w-3.5" />}
                  label="Owner"
                  value={formatOwner(sandbox.owner, sandbox.ownerType)}
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
              </div>

              {/* R4: Show more toggle */}
              <button
                className="flex items-center gap-1 mt-2 mb-1 text-xs text-slate-500 hover:text-slate-300 transition-colors"
                onClick={(e) => {
                  e.stopPropagation();
                  setShowMore(!showMore);
                }}
                data-testid="show-more-toggle"
              >
                {showMore ? (
                  <ChevronDown className="h-3 w-3" />
                ) : (
                  <ChevronRight className="h-3 w-3" />
                )}
                {showMore ? "Show less" : "Show more"}
              </button>

              {/* Secondary metadata — behind toggle */}
              {showMore && (
                <div data-testid="secondary-metadata">
                  <MetadataRow
                    icon={<FolderOpen className="h-3.5 w-3.5" />}
                    label="Scope"
                    value={sandbox.scopePath || "Not specified"}
                    copyable={!!sandbox.scopePath}
                    mono
                  />
                  <MetadataRow
                    icon={<FolderOpen className="h-3.5 w-3.5" />}
                    label="Project"
                    value={sandbox.projectRoot || "Not specified"}
                    copyable={!!sandbox.projectRoot}
                    mono
                  />
                  <MetadataRow
                    icon={<Server className="h-3.5 w-3.5" />}
                    label="Driver"
                    value={`${sandbox.driverId} v${sandbox.driverVersion}`}
                  />
                  {sandbox.mergedDir && sandbox.status === "active" && (
                    <MetadataRow
                      icon={<FolderOpen className="h-3.5 w-3.5" />}
                      label="Workspace"
                      value={sandbox.mergedDir}
                      copyable
                      mono
                    />
                  )}
                </div>
              )}

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
              className="px-3 py-3 border-t border-slate-800 flex flex-wrap items-center gap-2"
              onClick={(e) => e.stopPropagation()}
              data-testid="action-buttons"
            >
              {/* R1: Lifecycle group — Stop/Start, Launch Agent */}
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
                  {sandbox.status === "checkpointed" ? "Resume" : "Start"}
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

              {/* R1: Visual divider between lifecycle and review/approval groups (desktop only) */}
              {!hideDiffViewer && hasLifecycleButtons && hasReviewButtons && (
                <div className="w-px h-6 bg-slate-700 self-center" data-testid="action-divider" />
              )}

              {/* R1: Review/approval group (desktop only — on mobile these live on the Changes tab) */}
              {/* Review mode toggle */}
              {!hideDiffViewer && canApproveReject && onApproveSelected && (
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

              {/* Approve Selected button - shows when hunks are selected (desktop only) */}
              {!hideDiffViewer && canApproveReject && isReviewMode && hasSelection && onApproveSelected && (
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

      {/* Header/Diff Resize Handle - only show when details expanded and diff visible */}
      {!hideDiffViewer && !isDetailsCollapsed && (
        <div
          data-testid="header-resize-handle"
          className="h-1.5 bg-slate-900 hover:bg-slate-700 cursor-row-resize flex-shrink-0"
          onMouseDown={handleHeaderResizeStart}
        />
      )}

      {/* Diff Viewer - hidden on mobile (separate tab) */}
      {!hideDiffViewer && (
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
      )}
    </div>
  );
}
