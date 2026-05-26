import { useState, useCallback, useEffect, useMemo } from "react";
import { Loader2, Play, CheckCircle2, XCircle, AlertTriangle, ChevronDown, ChevronRight, Minus, Anchor, Camera } from "lucide-react";
import { Button } from "./ui/button";
import { buildWorkflowVideoUrl } from "../lib/api";
import type { ExecutionMode, WorkflowCaptureResult } from "../lib/api";
import { MediaLightbox, MutationErrorBanner, sanitizePagePath, formatDuration, type LightboxItem } from "./ScenarioReviewPanelShared";
import { IsolationBadge } from "./IsolationBadge";
import { useScenarioIsolation, type ScenarioIsolation } from "../hooks/useScenarioIsolation";

export const EXECUTION_MODE_COLORS: Record<ExecutionMode, string> = {
  observer: "bg-green-900/50 text-green-300 border-green-700/50",
  mutating: "bg-yellow-900/50 text-yellow-300 border-yellow-700/50",
  destructive: "bg-red-900/50 text-red-300 border-red-700/50",
};

export const STATUS_ICONS: Record<string, typeof CheckCircle2> = {
  passed: CheckCircle2,
  failed: XCircle,
  skipped: Minus,
  error: AlertTriangle,
};

export function WorkflowsTab({
  baseline,
  capture,
  captureStaleness,
  scenarioSlug,
  basAvailable,
  isRunning,
  onBaseline,
  onCapture,
  mutationError,
  onDismissError,
  initialSelectedModes,
  initialViewRole,
  onSelectedModesChange,
  onViewRoleChange,
}: {
  baseline?: WorkflowCaptureResult;
  capture?: WorkflowCaptureResult;
  captureStaleness?: import("../lib/api").SnapshotStalenessInfo;
  scenarioSlug: string;
  basAvailable: boolean;
  isRunning: boolean;
  onBaseline: (executionModes: ExecutionMode[]) => void;
  onCapture: (executionModes: ExecutionMode[]) => void;
  mutationError?: Error | null;
  onDismissError?: () => void;
  initialSelectedModes?: ExecutionMode[];
  initialViewRole?: "baseline" | "capture";
  onSelectedModesChange?: (modes: ExecutionMode[]) => void;
  onViewRoleChange?: (role: "baseline" | "capture") => void;
}) {
  const isolation = useScenarioIsolation(scenarioSlug);
  const isolationDefaultModes = useMemo<ExecutionMode[]>(
    () => (isolation.status === "routed" ? ["observer", "mutating", "destructive"] : ["observer"]),
    [isolation.status],
  );
  const [selectedModes, setSelectedModesInternal] = useState<Set<ExecutionMode>>(
    () => new Set(initialSelectedModes ?? isolationDefaultModes),
  );
  const [hasManualModeSelection, setHasManualModeSelection] = useState<boolean>(!!initialSelectedModes && initialSelectedModes.length > 0);
  const [modeSelectorOpen, setModeSelectorOpen] = useState<boolean>(isolation.status !== "routed");

  // When isolation resolves, sync defaults if the user hasn't explicitly
  // picked modes yet. We do not overwrite an explicit selection — only the
  // initial implicit default tracks the badge.
  useEffect(() => {
    if (isolation.status === "loading" || hasManualModeSelection) return;
    setSelectedModesInternal(new Set(isolationDefaultModes));
    setModeSelectorOpen(isolation.status !== "routed");
  }, [isolation.status, isolationDefaultModes, hasManualModeSelection]);
  const [lightboxIndex, setLightboxIndex] = useState(-1);
  // Which role's results to show in the table ("capture" by default, toggle to "baseline")
  const [viewRole, setViewRoleInternal] = useState<"baseline" | "capture">(initialViewRole ?? "capture");
  // Which rows are expanded to show error details
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());

  const setSelectedModes = useCallback((updater: Set<ExecutionMode> | ((prev: Set<ExecutionMode>) => Set<ExecutionMode>)) => {
    setSelectedModesInternal(prev => {
      const next = typeof updater === "function" ? updater(prev) : updater;
      onSelectedModesChange?.(Array.from(next));
      return next;
    });
    setHasManualModeSelection(true);
  }, [onSelectedModesChange]);

  const setViewRole = useCallback((role: "baseline" | "capture") => {
    setViewRoleInternal(role);
    onViewRoleChange?.(role);
  }, [onViewRoleChange]);

  const toggleMode = useCallback((mode: ExecutionMode) => {
    setSelectedModes(prev => {
      const next = new Set(prev);
      if (next.has(mode)) next.delete(mode);
      else next.add(mode);
      return next;
    });
  }, [setSelectedModes]);

  const toggleExpanded = useCallback((idx: number) => {
    setExpandedRows(prev => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx);
      else next.add(idx);
      return next;
    });
  }, []);

  const modesArray = Array.from(selectedModes);

  // Which result to display: user-selected role, falling back to whatever exists
  const viewedResult = viewRole === "capture" ? (capture ?? baseline) : (baseline ?? capture);

  // Build lightbox items from viewed result videos
  const lightboxItems: LightboxItem[] = [];
  if (viewedResult) {
    for (const wfResult of viewedResult.workflowResults) {
      if (wfResult.videoCount > 0 && wfResult.executionId) {
        for (let i = 0; i < wfResult.videoCount; i++) {
          const filename = `${sanitizePagePath(wfResult.workflowName)}_${i}.webm`;
          lightboxItems.push({
            label: wfResult.workflowName,
            sublabel: `${wfResult.executionMode} - ${wfResult.status}`,
            type: "video",
            url: buildWorkflowVideoUrl(viewedResult.id, scenarioSlug, filename),
          });
        }
      }
    }
  }

  // Summary counts for a given result
  const summarize = (result: WorkflowCaptureResult) => ({
    passed: result.workflowResults.filter(r => r.status === "passed").length,
    failed: result.workflowResults.filter(r => r.status === "failed" || r.status === "error").length,
    skipped: result.workflowResults.filter(r => r.status === "skipped").length,
  });

  // No captures at all — empty state
  if (!baseline && !capture) {
    return (
      <div className="space-y-3 py-6">
        <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
        <IsolationBadge isolation={isolation} />
        <div className="flex flex-col items-center justify-center text-slate-500 pt-2">
          <Play className="h-8 w-8 mb-3 opacity-50" />
          <p className="text-sm">No workflow captures yet</p>
          <p className="text-xs mt-1 mb-3 text-slate-600">Set a baseline to start comparing workflow results</p>
          {basAvailable ? (
            <>
              <ModeFilterDisclosure
                isolation={isolation}
                selectedModes={selectedModes}
                onToggle={toggleMode}
                open={modeSelectorOpen}
                onOpenChange={setModeSelectorOpen}
              />
              <Button
                variant="outline"
                size="sm"
                onClick={() => onBaseline(modesArray)}
                disabled={isRunning || selectedModes.size === 0}
                className="h-7 text-xs gap-1 mt-2"
              >
                {isRunning ? <Loader2 className="h-3 w-3 animate-spin" /> : <Anchor className="h-3 w-3" />}
                Set Baseline
              </Button>
            </>
          ) : (
            <p className="text-xs">Start browser-automation-studio to enable workflow captures</p>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
      <IsolationBadge isolation={isolation} />
      {/* Action buttons + execution mode selector (demoted to a disclosure) */}
      <div className="flex items-center gap-2 flex-wrap">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onBaseline(modesArray)}
          disabled={isRunning || selectedModes.size === 0}
          className="h-7 text-xs gap-1"
        >
          {isRunning ? <Loader2 className="h-3 w-3 animate-spin" /> : <Anchor className="h-3 w-3" />}
          {baseline ? "Reset Baseline" : "Set Baseline"}
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onCapture(modesArray)}
          disabled={isRunning || selectedModes.size === 0}
          className="h-7 text-xs gap-1"
        >
          {isRunning ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
          {capture ? "Re-capture" : "Capture"}
        </Button>
        <div className="ml-auto">
          <ModeFilterDisclosure
            isolation={isolation}
            selectedModes={selectedModes}
            onToggle={toggleMode}
            open={modeSelectorOpen}
            onOpenChange={setModeSelectorOpen}
          />
        </div>
      </div>

      {/* Staleness warning */}
      {captureStaleness?.isStale && capture && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Files have changed since this capture. Re-capture to see the latest workflow results.
          </p>
        </div>
      )}

      {/* Status message when only baseline exists */}
      {baseline && !capture && (
        <div className="text-xs text-slate-500 bg-slate-900/50 rounded px-3 py-2">
          Baseline set. Capture to compare workflow results against it.
        </div>
      )}

      {/* Summary bars — clickable to toggle which role's detail is shown */}
      {baseline && capture ? (
        <div className="space-y-1">
          {[capture, baseline].map((result) => {
            const role = result === capture ? "capture" : "baseline";
            const s = summarize(result);
            const isViewed = viewedResult === result;
            return (
              <button
                key={role}
                type="button"
                onClick={() => setViewRole(role)}
                className={`w-full flex items-center gap-4 px-3 py-2 rounded text-xs transition-colors text-left ${
                  isViewed ? "bg-slate-800/50 ring-1 ring-slate-600" : "bg-slate-800/30 hover:bg-slate-800/40"
                }`}
              >
                <span className="text-slate-300 font-medium capitalize">{role}</span>
                {role === "capture" && captureStaleness?.isStale && (
                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/50 text-amber-300">
                    Stale
                  </span>
                )}
                {result.status === "failed" && (
                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-red-900/50 text-red-300">
                    Failed
                  </span>
                )}
                <span className="text-green-400">{s.passed} passed</span>
                <span className="text-red-400">{s.failed} failed</span>
                <span className="text-slate-400">{s.skipped} skipped</span>
                <span className="text-slate-500 ml-auto">
                  {new Date(result.createdAt).toLocaleString()}
                </span>
                {isViewed && <ChevronRight className="h-3 w-3 text-blue-400" />}
              </button>
            );
          })}
        </div>
      ) : viewedResult && (() => {
        const s = summarize(viewedResult);
        return (
          <div className="flex items-center gap-4 px-3 py-2 bg-slate-800/50 rounded text-xs">
            <span className="text-slate-300 font-medium capitalize">{viewedResult.role}</span>
            {viewedResult.status === "failed" && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-red-900/50 text-red-300">
                Failed
              </span>
            )}
            <span className="text-green-400">{s.passed} passed</span>
            <span className="text-red-400">{s.failed} failed</span>
            <span className="text-slate-400">{s.skipped} skipped</span>
            <span className="text-slate-500 ml-auto">
              {new Date(viewedResult.createdAt).toLocaleString()}
            </span>
          </div>
        );
      })()}

      {/* Overall capture error */}
      {viewedResult?.error && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-red-950/30 border border-red-900/40">
          <XCircle className="h-4 w-4 text-red-400 mt-0.5 shrink-0" />
          <div className="min-w-0">
            <p className="text-xs font-medium text-red-300 mb-1">Capture error</p>
            <pre className="text-[11px] text-red-200/80 whitespace-pre-wrap break-words font-mono">{viewedResult.error}</pre>
          </div>
        </div>
      )}

      {/* Results table — shows whichever role is selected */}
      {viewedResult && viewedResult.workflowResults.length > 0 && (
        <div className="border border-slate-800 rounded-lg overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-slate-800/50">
                <th className="w-5 px-1 py-2"></th>
                <th className="text-left px-3 py-2 text-slate-400 font-medium">Workflow</th>
                <th className="text-left px-3 py-2 text-slate-400 font-medium">Mode</th>
                <th className="text-center px-3 py-2 text-slate-400 font-medium">Status</th>
                <th className="text-right px-3 py-2 text-slate-400 font-medium">Duration</th>
                <th className="text-center px-3 py-2 text-slate-400 font-medium">Video</th>
              </tr>
            </thead>
            <tbody>
              {viewedResult.workflowResults.map((wfr, idx) => {
                const StatusIcon = STATUS_ICONS[wfr.status] ?? AlertTriangle;
                const statusColor = wfr.status === "passed" ? "text-green-400"
                  : wfr.status === "failed" || wfr.status === "error" ? "text-red-400"
                  : "text-slate-500";
                const hasError = !!wfr.error;
                const isExpanded = expandedRows.has(idx);

                let videoLightboxIdx = -1;
                if (wfr.videoCount > 0) {
                  videoLightboxIdx = lightboxItems.findIndex(
                    item => item.label === wfr.workflowName
                  );
                }

                return (
                  <tr key={idx} className={`border-t border-slate-800/50 ${hasError ? "cursor-pointer" : ""} hover:bg-slate-800/30`} onClick={hasError ? () => toggleExpanded(idx) : undefined}>
                    <td className="px-1 py-2 text-center">
                      {hasError && (isExpanded ? <ChevronDown className="h-3 w-3 text-slate-500 inline" /> : <ChevronRight className="h-3 w-3 text-slate-500 inline" />)}
                    </td>
                    <td className="px-3 py-2 text-slate-200 max-w-[200px] truncate" title={wfr.workflowName}>
                      {wfr.workflowName}
                    </td>
                    <td className="px-3 py-2">
                      <span className={`px-1.5 py-0.5 rounded border text-[10px] ${EXECUTION_MODE_COLORS[wfr.executionMode as ExecutionMode] ?? "text-slate-400"}`}>
                        {wfr.executionMode}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-center">
                      <StatusIcon className={`h-3.5 w-3.5 inline ${statusColor}`} />
                    </td>
                    <td className="px-3 py-2 text-right text-slate-400">
                      {wfr.durationMs > 0 ? formatDuration(Math.round(wfr.durationMs / 1000)) : "-"}
                    </td>
                    <td className="px-3 py-2 text-center">
                      {wfr.videoCount > 0 && videoLightboxIdx >= 0 ? (
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); setLightboxIndex(videoLightboxIdx); }}
                          className="text-blue-400 hover:text-blue-300 text-[10px] underline"
                        >
                          Watch
                        </button>
                      ) : (
                        <span className="text-slate-600">-</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {/* Expanded error details rendered outside table for proper layout */}
          {viewedResult.workflowResults.map((wfr, idx) =>
            expandedRows.has(idx) && wfr.error ? (
              <div key={`err-${idx}`} className="px-4 py-2 bg-red-950/20 border-t border-red-900/30">
                <p className="text-[10px] text-slate-400 mb-1">{wfr.workflowName} — error details</p>
                <pre className="text-[11px] text-red-200/80 whitespace-pre-wrap break-words font-mono max-h-48 overflow-y-auto">{wfr.error}</pre>
              </div>
            ) : null
          )}
        </div>
      )}

      <MediaLightbox
        items={lightboxItems}
        initialIndex={lightboxIndex}
        isOpen={lightboxIndex >= 0}
        onClose={() => setLightboxIndex(-1)}
      />
    </div>
  );
}

function ExecutionModeSelector({ selectedModes, onToggle }: { selectedModes: Set<ExecutionMode>; onToggle: (mode: ExecutionMode) => void }) {
  return (
    <div className="flex items-center gap-2 flex-wrap">
      <span className="text-xs text-slate-400">Filter:</span>
      {(["observer", "mutating", "destructive"] as ExecutionMode[]).map(mode => (
        <label
          key={mode}
          className="flex items-center gap-1 text-xs cursor-pointer"
          title={`Include workflows tagged execution_mode=${mode}`}
        >
          <input
            type="checkbox"
            checked={selectedModes.has(mode)}
            onChange={() => onToggle(mode)}
            className="rounded border-slate-600"
          />
          <span className={`px-1.5 py-0.5 rounded border text-[10px] ${EXECUTION_MODE_COLORS[mode]}`}>
            {mode}
          </span>
        </label>
      ))}
    </div>
  );
}

// ModeFilterDisclosure wraps ExecutionModeSelector in a collapsible
// disclosure. The disclosure is collapsed-by-default when isolation is
// confirmed routed (no reason to filter for safety) and expanded otherwise.
// The legend distinguishes filter-as-filter from any data-safety claim.
function ModeFilterDisclosure({
  isolation,
  selectedModes,
  onToggle,
  open,
  onOpenChange,
}: {
  isolation: ScenarioIsolation;
  selectedModes: Set<ExecutionMode>;
  onToggle: (mode: ExecutionMode) => void;
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const summary = Array.from(selectedModes).join(", ") || "none";
  return (
    <div className="flex items-center gap-2 flex-wrap text-xs">
      <button
        type="button"
        onClick={() => onOpenChange(!open)}
        className="inline-flex items-center gap-1 text-slate-400 hover:text-slate-200"
        aria-expanded={open}
      >
        {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        Filter modes
        <span className="text-slate-500">({summary})</span>
      </button>
      {open && <ExecutionModeSelector selectedModes={selectedModes} onToggle={onToggle} />}
      {open && isolation.status === "not_routed" && (
        <span className="text-[10px] text-amber-300/80">
          Selecting non-observer modes will run against the scenario's primary database.
        </span>
      )}
    </div>
  );
}
