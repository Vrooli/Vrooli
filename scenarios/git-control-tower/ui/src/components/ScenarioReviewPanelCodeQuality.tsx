import { useState, useCallback, useMemo } from "react";
import { RefreshCw, Loader2, Play, AlertTriangle } from "lucide-react";
import { Button } from "./ui/button";
import { useTidinessScore, useTidinessIssues, useTidinessStaleness, useTriggerTidinessScan } from "../lib/hooks";
import type { RepoFileStats, TidinessIssue, AgentContextItem } from "../lib/api";
import { CodeQualityPickerModal } from "./CodeQualityPickerModal";
import { MutationErrorBanner, ServiceUnavailableBanner, formatRelativeTime, formatStalenessMessage, ScanResultSummary } from "./ScenarioReviewPanelShared";
import { ChangedFilesView, ScenarioWideView } from "./ScenarioReviewPanelRulesViews";

export function CodeQualityTab({
  scenarioSlug,
  repoId,
  tidinessAvailable,
  fileStats,
  agentManagerAvailable,
  onAttachToAgent,
  initialView,
  onViewChange,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  tidinessAvailable: boolean;
  fileStats?: RepoFileStats;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  initialView?: "changed" | "scenario";
  onViewChange?: (view: "changed" | "scenario") => void;
}) {
  const [view, setViewInternal] = useState<"changed" | "scenario">(initialView ?? "changed");
  const [pickerOpen, setPickerOpen] = useState(false);
  const setView = useCallback((v: "changed" | "scenario") => {
    setViewInternal(v);
    onViewChange?.(v);
  }, [onViewChange]);
  const tidinessScore = useTidinessScore(scenarioSlug, tidinessAvailable, repoId);
  const tidinessIssues = useTidinessIssues(scenarioSlug, { enabled: tidinessAvailable, repoId });
  const tidinessStaleness = useTidinessStaleness(scenarioSlug, tidinessAvailable, repoId);
  const triggerScan = useTriggerTidinessScan(repoId);

  // Get changed file paths (scenario-relative)
  const changedFiles = useMemo(() => {
    if (!fileStats) return [];
    const prefix = `scenarios/${scenarioSlug}/`;
    return Object.keys(fileStats).map(p =>
      p.startsWith(prefix) ? p.slice(prefix.length) : p
    );
  }, [fileStats, scenarioSlug]);

  // Filter issues to changed files
  const changedFileIssues = useMemo(() => {
    if (!tidinessIssues.data || changedFiles.length === 0) return [];
    return tidinessIssues.data.filter(issue =>
      changedFiles.includes(issue.file_path)
    );
  }, [tidinessIssues.data, changedFiles]);

  // Group issues by file
  const issuesByFile = useMemo(() => {
    const map = new Map<string, TidinessIssue[]>();
    for (const issue of changedFileIssues) {
      const existing = map.get(issue.file_path) ?? [];
      existing.push(issue);
      map.set(issue.file_path, existing);
    }
    return map;
  }, [changedFileIssues]);

  if (!tidinessAvailable) {
    return <ServiceUnavailableBanner name="Tidiness Manager" message="Start the tidiness-manager scenario to view code quality data" />;
  }

  // Use staleness as source of truth for "has been scanned" — more reliable than score.last_scan
  const stalenessLoaded = !tidinessStaleness.isLoading;
  const neverScanned = stalenessLoaded && (
    !tidinessStaleness.data?.last_scan_at &&
    tidinessStaleness.data?.stale_reason?.includes("no scans")
  );
  const isScanning = triggerScan.isPending;
  const scanResult = triggerScan.data;

  // Never-scanned state: show a prominent CTA
  if (neverScanned) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm font-medium text-slate-400">No scan data yet</p>
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
          Run a scan to analyze code quality for this scenario
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => triggerScan.mutate({
            scenarioName: scenarioSlug,
            incremental: false,
          })}
          disabled={isScanning}
          className="mt-4 gap-1.5"
        >
          {isScanning ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Play className="h-3.5 w-3.5" />
          )}
          {isScanning ? "Scanning..." : "Run Scan"}
        </Button>
        {scanResult && (
          <ScanResultSummary result={scanResult} />
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={triggerScan.error} onDismiss={() => triggerScan.reset()} />
      {/* Staleness banner */}
      {tidinessStaleness.data?.is_stale && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            {formatStalenessMessage(tidinessStaleness.data)}
          </p>
        </div>
      )}

      {/* Last scan info */}
      {tidinessStaleness.data?.last_scan_at && !tidinessStaleness.data?.is_stale && (
        <p className="text-[11px] text-slate-500">
          Last scanned {formatRelativeTime(tidinessStaleness.data.last_scan_at)}
          {tidinessScore.data?.metrics?.total_files ? ` · ${tidinessScore.data.metrics.total_files} files` : ""}
        </p>
      )}

      {/* View toggle + scan button */}
      <div className="flex items-center justify-between">
        <div className="flex gap-1">
          <button
            type="button"
            onClick={() => setView("changed")}
            className={`px-2.5 py-1 rounded-full text-xs transition-colors ${
              view === "changed"
                ? "bg-blue-600 text-white"
                : "bg-slate-800 text-slate-400 hover:text-slate-200"
            }`}
          >
            Changed Files
          </button>
          <button
            type="button"
            onClick={() => setView("scenario")}
            className={`px-2.5 py-1 rounded-full text-xs transition-colors ${
              view === "scenario"
                ? "bg-blue-600 text-white"
                : "bg-slate-800 text-slate-400 hover:text-slate-200"
            }`}
          >
            Full Scenario
          </button>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => triggerScan.mutate({
            scenarioName: scenarioSlug,
            incremental: true,
          })}
          disabled={isScanning}
          className="h-7 text-xs gap-1"
        >
          {isScanning ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCw className="h-3 w-3" />
          )}
          Scan
        </Button>
      </div>

      {agentManagerAvailable && onAttachToAgent && (
        <CodeQualityPickerModal
          isOpen={pickerOpen}
          onClose={() => setPickerOpen(false)}
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          onAttachItems={(items) => { for (const item of items) onAttachToAgent(item); }}
        />
      )}

      {/* In-progress banner */}
      {isScanning && (
        <div className="flex items-center gap-2 px-3 py-2 bg-blue-950/50 border border-blue-900/50 rounded-lg text-blue-300 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Scanning...
        </div>
      )}

      {/* Scan result summary */}
      {scanResult && !isScanning && (
        <ScanResultSummary result={scanResult} />
      )}

      {view === "changed" ? (
        <ChangedFilesView
          issues={changedFileIssues}
          issuesByFile={issuesByFile}
          changedFiles={changedFiles}
          scoreData={tidinessScore.data}
          isLoading={tidinessIssues.isLoading}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={onAttachToAgent}
          scenarioSlug={scenarioSlug}
        />
      ) : (
        <ScenarioWideView
          scoreData={tidinessScore.data}
          isLoading={tidinessScore.isLoading}
          agentManagerAvailable={agentManagerAvailable}
          onOpenPicker={() => setPickerOpen(true)}
        />
      )}
    </div>
  );
}

// ============================================================================
// Rules Tab
