/**
 * Sidebar header showing pipeline info and debug button.
 */

import { useState } from "react";
import { Bug, Copy, Check } from "lucide-react";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { DebugJsonModal } from "./DebugJsonModal";
import { usePipelineStore, selectProgress, selectCurrentStage } from "../../store";
import { cn } from "../../lib/utils";
import { writeToClipboard } from "../../lib/browser";
import { getPipelineStatusDisplay, formatStageName } from "../../lib/status-display";

export function SidebarHeader() {
  const [debugModalOpen, setDebugModalOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const pipelineId = usePipelineStore((s) => s.pipelineId);
  const pipelineStatus = usePipelineStore((s) => s.pipelineStatus);
  const runStatus = usePipelineStore((s) => s.runStatus);
  const scenarioName = usePipelineStore((s) => s.scenarioName);
  const progress = usePipelineStore(selectProgress);
  const currentStage = usePipelineStore(selectCurrentStage);

  const status = pipelineStatus?.status ?? (runStatus !== "idle" ? runStatus : null);
  const { label, icon: StatusIcon, className } = getPipelineStatusDisplay(status);

  const isRunning = status === "running" || status === "starting";
  const progressPercent = Math.round(progress * 100);

  const handleCopyId = async () => {
    if (pipelineId) {
      const result = await writeToClipboard(pipelineId);
      if (result.success) {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }
    }
  };

  return (
    <>
      <div className="border-b border-white/10 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-slate-200">Pipeline</h2>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setDebugModalOpen(true)}
            className="h-7 w-7 p-0 text-slate-400 hover:text-slate-200"
            title="Debug JSON"
          >
            <Bug className="h-4 w-4" />
          </Button>
        </div>

        {/* Scenario name */}
        {scenarioName && (
          <div className="space-y-1">
            <span className="text-xs text-slate-500">Scenario</span>
            <p className="text-sm text-slate-200 truncate">{scenarioName}</p>
          </div>
        )}

        {/* Pipeline ID */}
        <div className="space-y-1">
          <span className="text-xs text-slate-500">Pipeline ID</span>
          <div className="flex items-center gap-2">
            <code className="flex-1 truncate rounded bg-slate-900/50 px-2 py-1 text-xs text-slate-300 font-mono">
              {pipelineId ?? "No active pipeline"}
            </code>
            {pipelineId && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleCopyId}
                className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
                title="Copy ID"
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
          <Badge variant="outline" className={cn("flex w-fit items-center gap-1.5", className)}>
            <StatusIcon
              className={cn("h-3 w-3", isRunning && "animate-spin")}
            />
            {label}
          </Badge>
        </div>

        {/* Progress bar when running */}
        {isRunning && (
          <div className="space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-400">
                {currentStage ? formatStageName(currentStage) : "Starting..."}
              </span>
              <span className="text-slate-500">{progressPercent}%</span>
            </div>
            <div className="h-1.5 w-full rounded-full bg-slate-800 overflow-hidden">
              <div
                className="h-full bg-blue-500 transition-all duration-300 ease-out"
                style={{ width: `${progressPercent}%` }}
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

      <DebugJsonModal open={debugModalOpen} onClose={() => setDebugModalOpen(false)} />
    </>
  );
}
