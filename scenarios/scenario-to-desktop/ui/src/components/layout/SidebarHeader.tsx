/**
 * Sidebar header showing pipeline info, timestamps, and controls.
 * Consolidates all pipeline metadata with "New Pipeline" button and history access.
 */

import { useState } from "react";
import {
  Bug,
  Copy,
  Check,
  Plus,
  History,
  Clock,
  GitCommit,
} from "lucide-react";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { DebugJsonModal } from "./DebugJsonModal";
import { PipelineHistoryDropdown } from "./PipelineHistoryDropdown";
import {
  usePipelineStore,
  selectProgress,
  selectCurrentStage,
  selectIsRunning,
} from "../../store";
import { cn } from "../../lib/utils";
import { writeToClipboard } from "../../lib/browser";
import {
  getPipelineStatusDisplay,
  formatStageName,
} from "../../lib/status-display";
import { StageStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

/** Format Unix timestamp to locale string */
function formatTimestamp(
  value: { seconds: bigint; nanos: number } | undefined,
): string {
  if (!value) return "-";
  return new Date(
    Number(value.seconds) * 1000 + value.nanos / 1_000_000,
  ).toLocaleString();
}

/** Format Unix timestamp to short time-only string (for same-day display) */
function _formatTimestampShort(unix: number | undefined): string {
  if (!unix) return "-";
  const date = new Date(unix * 1000);
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();
  if (isToday) {
    return date.toLocaleTimeString(undefined, {
      hour: "numeric",
      minute: "2-digit",
    });
  }
  return (
    date.toLocaleDateString(undefined, { month: "short", day: "numeric" }) +
    " " +
    date.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" })
  );
}

export function SidebarHeader() {
  const [debugModalOpen, setDebugModalOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const pipelineId = usePipelineStore((s) => s.pipelineId);
  const pipelineStatus = usePipelineStore((s) => s.pipelineStatus);
  const runStatus = usePipelineStore((s) => s.runStatus);
  const scenarioName = usePipelineStore((s) => s.scenarioName);
  const progress = usePipelineStore(selectProgress);
  const currentStage = usePipelineStore(selectCurrentStage);
  const isRunningStore = usePipelineStore(selectIsRunning);
  const isLoadingActivePipeline = usePipelineStore(
    (s) => s.isLoadingActivePipeline,
  );
  const createNewPipelineForScenario = usePipelineStore(
    (s) => s.createNewPipelineForScenario,
  );

  // Use server-side status if available, otherwise use local runStatus
  // "idle" is a valid status (created but not started), so don't filter it out
  const status = pipelineStatus?.status ?? runStatus;
  const {
    label,
    icon: StatusIcon,
    className,
  } = getPipelineStatusDisplay(status);

  // Only consider pipeline "running" if it's actively executing (not idle)
  const isRunning =
    status === StageStatus.RUNNING ||
    status === "starting" ||
    status === StageStatus.PENDING ||
    isRunningStore;
  const progressPercent = Math.round(progress * 100);

  // Get timestamps from pipeline status
  const startedAt = pipelineStatus?.startedAt;
  const completedAt = pipelineStatus?.completedAt;

  const handleCreateNewPipeline = async () => {
    if (isRunning || isCreating) return;
    setIsCreating(true);
    try {
      await createNewPipelineForScenario();
    } catch (err) {
      console.error("Failed to create new pipeline:", err);
    } finally {
      setIsCreating(false);
    }
  };

  const handleCopyId = async () => {
    if (pipelineId) {
      const result = await writeToClipboard(pipelineId);
      if (result.success) {
        setCopied(true);
        setTimeout(() => {
          setCopied(false);
        }, 2000);
      }
    }
  };

  return (
    <>
      <div className="border-b border-white/10 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-slate-200">Pipeline</h2>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setHistoryOpen(true);
              }}
              className="h-7 w-7 p-0 text-slate-400 hover:text-slate-200"
              title="Pipeline History"
              aria-label="Open pipeline history"
            >
              <History className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setDebugModalOpen(true);
              }}
              className="h-7 w-7 p-0 text-slate-400 hover:text-slate-200"
              title="Debug JSON"
              aria-label="Open pipeline debug JSON"
            >
              <Bug className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Scenario name */}
        {scenarioName && (
          <div className="space-y-1">
            <span className="text-xs text-slate-500">Scenario</span>
            <p className="text-sm text-slate-200 truncate">{scenarioName}</p>
          </div>
        )}

        {/* Pipeline ID with New Pipeline button */}
        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-500">Pipeline ID</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                void handleCreateNewPipeline();
              }}
              disabled={
                isRunning ||
                isCreating ||
                isLoadingActivePipeline ||
                !scenarioName
              }
              className="h-6 px-2 text-xs text-slate-400 hover:text-slate-200 disabled:opacity-50"
              title={
                isRunning
                  ? "Cannot create while running"
                  : "Create new pipeline"
              }
            >
              <Plus className="h-3 w-3 mr-1" />
              New
            </Button>
          </div>
          <div className="flex items-center gap-2">
            <code className="flex-1 truncate rounded bg-slate-900/50 px-2 py-1 text-xs text-slate-300 font-mono">
              {isLoadingActivePipeline
                ? "Loading..."
                : (pipelineId ?? "No active pipeline")}
            </code>
            {pipelineId && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  void handleCopyId();
                }}
                className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
                title="Copy ID"
                aria-label={copied ? "Pipeline ID copied" : "Copy pipeline ID"}
              >
                {copied ? (
                  <Check className="h-3 w-3 text-green-400" />
                ) : (
                  <Copy className="h-3 w-3" />
                )}
              </Button>
            )}
          </div>
        </div>

        {/* Pipeline Status */}
        <div className="space-y-1">
          <span className="text-xs text-slate-500">Status</span>
          <Badge
            variant="outline"
            className={cn("flex w-fit items-center gap-1.5", className)}
          >
            <StatusIcon
              className={cn("h-3 w-3", isRunning && "animate-spin")}
            />
            {label}
          </Badge>
        </div>

        {/* Build Provenance */}
        {pipelineStatus?.startedAt ? (
          <div className="flex items-center gap-2 text-xs text-slate-500">
            <GitCommit className="h-3 w-3" />
            <span>No provenance — run a new pipeline to capture git info</span>
          </div>
        ) : null}

        {/* Timestamps */}
        {(startedAt || completedAt) && (
          <div className="space-y-1.5 text-xs">
            {startedAt && (
              <div className="flex items-center gap-2 text-slate-400">
                <Clock className="h-3 w-3" />
                <span>Started: {formatTimestamp(startedAt)}</span>
              </div>
            )}
            {completedAt && (
              <div className="flex items-center gap-2 text-slate-400">
                <Check className="h-3 w-3" />
                <span>Completed: {formatTimestamp(completedAt)}</span>
              </div>
            )}
          </div>
        )}

        {/* Progress bar when running */}
        {isRunning && (
          <div className="space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-400">
                {currentStage ? formatStageName(currentStage) : "Starting..."}
              </span>
              <span className="text-slate-500">{String(progressPercent)}%</span>
            </div>
            <div className="h-1.5 w-full rounded-full bg-slate-800 overflow-hidden">
              <div
                className="h-full bg-blue-500 transition-all duration-300 ease-out"
                style={{ width: `${String(progressPercent)}%` }}
              />
            </div>
          </div>
        )}

        {/* Current stage info when running */}
        {isRunning && currentStage && (
          <p className="text-xs text-blue-400">
            Running {formatStageName(currentStage)} stage...
          </p>
        )}
      </div>

      <DebugJsonModal
        open={debugModalOpen}
        onClose={() => {
          setDebugModalOpen(false);
        }}
      />
      <PipelineHistoryDropdown
        open={historyOpen}
        onClose={() => {
          setHistoryOpen(false);
        }}
      />
    </>
  );
}
