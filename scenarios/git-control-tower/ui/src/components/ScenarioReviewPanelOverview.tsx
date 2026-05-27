import { useState, useEffect, useCallback } from "react";
import { RefreshCw, Loader2, CheckCircle2, XCircle, AlertTriangle, Plus, Minus, Camera, ExternalLink } from "lucide-react";
import { Button } from "./ui/button";
import { useTestExecutions, useTidinessScore, useTidinessStaleness, useScenarios, useReviewSummary, useTriggerReviewRun, useReviewJobStatus } from "../lib/hooks";
import { fetchExternalUrl } from "../lib/api-internals";
import type { SnapshotSetMeta, SnapshotStalenessInfo, TestExecutionResult, RepoFileStats, AgentContextItem, Readiness } from "../lib/api";
import { aggregateFileStats, formatNetLines } from "../lib/metrics";
import { AttachToAgentButton } from "./AgentTab";
import { testFailureContextItems, changeSummaryContextItem, scenarioQualityContextItem } from "../lib/agentContext";
import { formatDuration, formatRelativeTime, formatStalenessMessage } from "./ScenarioReviewPanelShared";
import { BaselineDriftCallout } from "../features/baselines/BaselineDriftCallout";

export function OverviewTab({
  capture,
  captureStaleness,
  scenarioSlug,
  repoId,
  basAvailable,
  testGenieAvailable,
  tidinessAvailable,
  isCapturing,
  onCapture,
  fileStats,
  agentManagerAvailable,
  onAttachToAgent,
  onOpenBaselines,
}: {
  capture?: SnapshotSetMeta;
  captureStaleness?: SnapshotStalenessInfo;
  scenarioSlug: string;
  repoId?: string | null;
  basAvailable: boolean;
  testGenieAvailable: boolean;
  tidinessAvailable: boolean;
  isCapturing: boolean;
  onCapture: () => void;
  fileStats?: RepoFileStats;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  onOpenBaselines?: () => void;
}) {
  const testExecutions = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const latestTest = testExecutions.data?.items?.[0] as TestExecutionResult | undefined;
  const tidinessScore = useTidinessScore(scenarioSlug, tidinessAvailable, repoId);
  const tidinessStaleness = useTidinessStaleness(scenarioSlug, tidinessAvailable, repoId);
  const scenarios = useScenarios();
  const scenarioInfo = scenarios.data?.find(s => s.name === scenarioSlug);
  const [proxyUrl, setProxyUrl] = useState(`/embedded/${encodeURIComponent(scenarioSlug)}/`);

  // Unified review summary from server
  const reviewSummary = useReviewSummary(scenarioSlug, repoId);
  const triggerReview = useTriggerReviewRun(repoId);
  const [reviewJobId, setReviewJobId] = useState<string | null>(null);
  const reviewJob = useReviewJobStatus(reviewJobId, repoId);

  useEffect(() => {
    let cancelled = false;
    fetchExternalUrl(`/embedded/${encodeURIComponent(scenarioSlug)}/external-url`)
      .then((url) => {
        if (!cancelled && url) {
          setProxyUrl(url);
        }
      })
      .catch(() => { /* keep fallback */ });
    return () => { cancelled = true; };
  }, [scenarioSlug]);

  // Used by both readiness fallback and visual status card
  const latestSnapshot = capture;

  // Use server-side readiness when available, fall back to client-side calculation.
  // Client-side fallback — the server-side calculation (review_readiness.go) is authoritative
  // and also includes standards/blocking-violations checks that this fallback cannot replicate.
  const hasBeenScanned = Boolean(tidinessStaleness.data?.last_scan_at) ||
    (tidinessStaleness.data ? !tidinessStaleness.data.stale_reason?.includes("no scans") : false);

  const readiness: Readiness = reviewSummary.data?.readiness ?? (() => {
    const hasScreenshots = latestSnapshot && latestSnapshot.screenshotCount > 0;
    const hasTests = Boolean(latestTest);
    const testsPass = latestTest?.success ?? false;
    const qualityScore = tidinessScore.data?.score ?? null;
    const qualityOk = hasBeenScanned && qualityScore !== null && qualityScore >= 60;
    if (hasScreenshots && hasTests && testsPass && qualityOk) return "green" as Readiness;
    if (hasScreenshots || hasTests || qualityOk) return "yellow" as Readiness;
    return "red" as Readiness;
  })();

  const readinessColors = {
    green: "bg-emerald-500",
    yellow: "bg-amber-500",
    red: "bg-red-500",
  };
  const readinessLabels = {
    green: "Ready",
    yellow: "Incomplete",
    red: "No data",
  };

  const handleRerunAll = useCallback(() => {
    triggerReview.mutate({ scenarioName: scenarioSlug }, {
      onSuccess: (data) => setReviewJobId(data.jobId),
    });
  }, [triggerReview, scenarioSlug]);

  const isRerunning = triggerReview.isPending || (reviewJob.data?.status === "running");

  return (
    <div className="space-y-4">
      {onOpenBaselines && (
        <BaselineDriftCallout scenario={scenarioSlug} repoId={repoId} onOpenBaselines={onOpenBaselines} />
      )}
      {/* Scenario Info Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-xs font-medium text-slate-400">Scenario</h3>
          <div className="flex items-center gap-2">
            <div className={`h-2 w-2 rounded-full ${scenarioInfo?.status === "running" ? "bg-emerald-500" : "bg-slate-500"}`} />
            <span className={`text-[11px] ${scenarioInfo?.status === "running" ? "text-emerald-400" : "text-slate-500"}`}>
              {scenarioInfo?.status ?? "unknown"}
            </span>
          </div>
        </div>
        <p className="text-sm font-medium text-slate-200 mb-2">
          {scenarioInfo?.display_name || scenarioSlug}
        </p>
        <div className="flex items-center gap-2">
          <span className="text-[11px] text-slate-500 truncate flex-1 font-mono">{proxyUrl}</span>
          <a
            href={proxyUrl}
            target="_blank"
            rel="noopener"
            className={`inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded border transition-colors shrink-0 ${
              scenarioInfo?.status === "running"
                ? "text-blue-400 border-blue-800 hover:bg-blue-950/50"
                : "text-slate-600 border-slate-700 pointer-events-none opacity-50"
            }`}
            aria-label="Open scenario in new tab"
          >
            <ExternalLink className="h-3 w-3" />
            Open
          </a>
        </div>
        {scenarioInfo?.health_status && scenarioInfo.health_status !== "healthy" && (
          <div className="flex items-center gap-2 mt-2 text-[11px] text-amber-400">
            <AlertTriangle className="h-3 w-3" />
            {scenarioInfo.health_status}
          </div>
        )}
      </div>

      {/* Readiness indicator */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className={`h-3 w-3 rounded-full ${readinessColors[readiness]}`} />
          <span className="text-sm font-medium text-slate-200">{readinessLabels[readiness]}</span>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleRerunAll}
          disabled={isRerunning}
          className="h-7 text-xs gap-1"
        >
          {isRerunning ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
          Rerun All Checks
        </Button>
      </div>
      {isRerunning && reviewJob.data && (
        <div className="mb-4 rounded-lg border border-slate-800 bg-slate-900/50 p-3">
          <p className="text-xs text-slate-400 mb-2">Review run in progress...</p>
          <div className="flex flex-wrap gap-2">
            {Object.entries(reviewJob.data.checks).map(([check, status]) => (
              <span key={check} className={`text-[11px] px-2 py-0.5 rounded-full ${
                status === "completed" ? "bg-emerald-900/50 text-emerald-400" :
                status === "running" ? "bg-blue-900/50 text-blue-400" :
                status === "failed" ? "bg-red-900/50 text-red-400" :
                status === "skipped" ? "bg-slate-800 text-slate-500" :
                "bg-slate-800 text-slate-400"
              }`}>
                {check}: {status}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Change Summary Card */}
      {fileStats && (() => {
        const agg = aggregateFileStats(fileStats);
        if (!agg || agg.totalFiles === 0) return null;
        return (
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-xs font-medium text-slate-400">Change Summary</h3>
              {agentManagerAvailable && onAttachToAgent && (
                <AttachToAgentButton onClick={() => onAttachToAgent(changeSummaryContextItem(fileStats))} />
              )}
            </div>
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1 text-emerald-500 text-sm font-medium">
                <Plus className="h-3.5 w-3.5" />
                {agg.totalAdditions}
              </span>
              <span className="flex items-center gap-1 text-red-500 text-sm font-medium">
                <Minus className="h-3.5 w-3.5" />
                {agg.totalDeletions}
              </span>
              <span className="text-sm font-medium text-blue-400">
                net {formatNetLines(agg.totalNetLines)}
              </span>
            </div>
            <div className="mt-2 text-xs text-slate-400">
              {agg.totalFiles} file{agg.totalFiles !== 1 ? "s" : ""} changed
            </div>
          </div>
        );
      })()}

      {/* Visual Status Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xs font-medium text-slate-400">Visual Status</h3>
          {captureStaleness?.isStale && (
            <span className="text-[11px] text-amber-400 flex items-center gap-1">
              <AlertTriangle className="h-3 w-3" />
              Stale
            </span>
          )}
        </div>
        {!basAvailable ? (
          <p className="text-xs text-slate-500">Start browser-automation-studio to enable visual captures</p>
        ) : !latestSnapshot ? (
          <div className="space-y-2">
            <p className="text-xs text-slate-500">No screenshots captured yet</p>
            <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="h-7 text-xs gap-1">
              {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
              Capture screenshots
            </Button>
          </div>
        ) : (
          <div className="space-y-2">
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Captured{captureStaleness?.isStale ? " (stale)" : ""}</span>
              <span className="text-slate-200">{new Date(latestSnapshot.createdAt).toLocaleString()}</span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Screenshots</span>
              <span className="text-slate-200">{latestSnapshot.screenshotCount}</span>
            </div>
            {latestSnapshot.pageDiscoveryMethod === "fallback" && (
              <div className="flex items-start gap-2 mt-2 p-2 rounded bg-amber-950/30 border border-amber-900/40">
                <AlertTriangle className="h-3.5 w-3.5 text-amber-400 mt-0.5 shrink-0" />
                <p className="text-[11px] text-amber-300">
                  Pages discovered via fallback (root only). Add <code className="bg-slate-800 px-1 rounded">.vrooli/lighthouse.json</code> to capture all pages.
                </p>
              </div>
            )}
            <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="h-7 text-xs gap-1 mt-2">
              {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
              Re-capture
            </Button>
          </div>
        )}
      </div>

      {/* Test Status Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xs font-medium text-slate-400">Test Status</h3>
          {agentManagerAvailable && onAttachToAgent && latestTest && !latestTest.success && (
            <AttachToAgentButton onClick={() => {
              const failedPhases = latestTest.phases.filter(p => p.status === "failed");
              for (const item of testFailureContextItems(failedPhases, scenarioSlug)) {
                onAttachToAgent(item);
              }
            }} />
          )}
        </div>
        {!testGenieAvailable ? (
          <p className="text-xs text-slate-500">Start test-genie to enable automated tests</p>
        ) : testExecutions.isLoading ? (
          <div className="h-12 animate-pulse bg-slate-800 rounded" />
        ) : !latestTest ? (
          <p className="text-xs text-slate-500">No tests run yet</p>
        ) : (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              {latestTest.success ? (
                <CheckCircle2 className="h-4 w-4 text-emerald-400" />
              ) : (
                <XCircle className="h-4 w-4 text-red-400" />
              )}
              <span className={`text-xs font-medium ${latestTest.success ? "text-emerald-300" : "text-red-300"}`}>
                {latestTest.success ? "Passed" : "Failed"}
              </span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Phases</span>
              <span className="text-slate-200">
                {latestTest.phaseSummary.passed}/{latestTest.phaseSummary.total} passed
              </span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Duration</span>
              <span className="text-slate-200">{formatDuration(latestTest.phaseSummary.durationSeconds)}</span>
            </div>
            {latestTest.completedAt && (
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Completed</span>
                <span className="text-slate-200">{new Date(latestTest.completedAt).toLocaleString()}</span>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Code Quality Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xs font-medium text-slate-400">Code Quality</h3>
          {agentManagerAvailable && onAttachToAgent && tidinessScore.data && hasBeenScanned && tidinessScore.data.violations > 0 && (
            <AttachToAgentButton onClick={() => onAttachToAgent(scenarioQualityContextItem(tidinessScore.data as { score: number; violations: number }))} />
          )}
        </div>
        {!tidinessAvailable ? (
          <p className="text-xs text-slate-500">Start tidiness-manager to view code quality data</p>
        ) : tidinessScore.isLoading ? (
          <div className="h-12 animate-pulse bg-slate-800 rounded" />
        ) : tidinessScore.error ? (
          <p className="text-xs text-slate-500">No quality data available</p>
        ) : tidinessScore.data ? (
          !hasBeenScanned ? (
            <div className="space-y-2">
              <p className="text-xs text-slate-500">Not yet scanned</p>
              <p className="text-[11px] text-slate-600">Open the Code Quality tab to run a scan</p>
            </div>
          ) : (
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <div className={`h-4 w-4 rounded-full flex items-center justify-center ${
                  tidinessScore.data.score >= 70 ? "bg-emerald-500" :
                  tidinessScore.data.score >= 40 ? "bg-amber-500" : "bg-red-500"
                }`}>
                  <span className="text-[8px] font-bold text-white">
                    {Math.round(tidinessScore.data.score)}
                  </span>
                </div>
                <span className={`text-xs font-medium ${
                  tidinessScore.data.score >= 70 ? "text-emerald-300" :
                  tidinessScore.data.score >= 40 ? "text-amber-300" : "text-red-300"
                }`}>
                  {tidinessScore.data.score >= 70 ? "Good" :
                   tidinessScore.data.score >= 40 ? "Fair" : "Poor"}
                </span>
              </div>
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Score</span>
                <span className="text-slate-200">{Math.round(tidinessScore.data.score)}/100</span>
              </div>
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Violations</span>
                <span className="text-slate-200">{tidinessScore.data.violations}</span>
              </div>
              {tidinessStaleness.data?.last_scan_at && (
                <div className="flex justify-between text-xs">
                  <span className="text-slate-400">Last scan</span>
                  <span className="text-slate-200">{formatRelativeTime(tidinessStaleness.data.last_scan_at)}</span>
                </div>
              )}
              {tidinessStaleness.data?.is_stale && (
                <div className="flex items-start gap-2 mt-2 p-2 rounded bg-amber-950/30 border border-amber-900/40">
                  <AlertTriangle className="h-3.5 w-3.5 text-amber-400 mt-0.5 shrink-0" />
                  <p className="text-[11px] text-amber-300">
                    {formatStalenessMessage(tidinessStaleness.data)}
                  </p>
                </div>
              )}
            </div>
          )
        ) : (
          <p className="text-xs text-slate-500">No quality data available</p>
        )}
      </div>
    </div>
  );
}
