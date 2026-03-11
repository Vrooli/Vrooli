import { useState, useEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import { X, ClipboardCheck, RefreshCw, Loader2, Play, CheckCircle2, XCircle, AlertTriangle, ChevronDown, ChevronRight, ChevronLeft, Plus, Minus } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";
import { useVisualCaptures, useTriggerVisualCapture, useCapabilities, useTestExecutions, useTriggerTestExecution, useTidinessScore, useTidinessIssues, useTidinessStaleness, useTriggerTidinessScan } from "../lib/hooks";
import { buildCaptureScreenshotUrl, buildCaptureVideoUrl } from "../lib/api";
import type { SnapshotSetMeta, TestExecutionResult, TestPhaseResult, RepoFileStats, TidinessIssue, TidinessLightScanResult, TidinessStalenessInfo } from "../lib/api";
import { AggregateMetricsContent } from "./ChangeMetricsModal";
import { aggregateFileStats, formatNetLines } from "../lib/metrics";

type Tab = "overview" | "metrics" | "screenshots" | "videos" | "tests" | "code-quality";

interface ScenarioReviewModalProps {
  isOpen: boolean;
  onClose: () => void;
  scenarioSlug: string;
  repoId?: string | null;
  fileStats?: RepoFileStats;
}

export function ScenarioReviewModal({ isOpen, onClose, scenarioSlug, repoId, fileStats }: ScenarioReviewModalProps) {
  const isMobile = useIsMobile();
  const [activeTab, setActiveTab] = useState<Tab>("overview");
  const capturesQuery = useVisualCaptures(scenarioSlug, isOpen, repoId);
  const triggerCapture = useTriggerVisualCapture(repoId);
  const capabilities = useCapabilities();

  const basAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "browser-automation-studio" && c.status === "available"
  ) ?? false;
  const testGenieAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "test-genie" && c.status === "available"
  ) ?? false;
  const tidinessAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "tidiness-manager" && c.status === "available"
  ) ?? false;

  if (!isOpen) return null;

  const snapshots = capturesQuery.data?.snapshots ?? [];
  const completeSnapshots = snapshots.filter(s => s.status === "complete");
  const after = completeSnapshots[0] as SnapshotSetMeta | undefined;
  const before = completeSnapshots[1] as SnapshotSetMeta | undefined;

  const isCapturing = triggerCapture.isPending;

  const tabLabels: Record<Tab, string> = {
    overview: "Overview",
    metrics: "Metrics",
    screenshots: "Screenshots",
    videos: "Videos",
    tests: "Tests",
    "code-quality": "Code Quality",
  };

  const visibleTabs = (Object.keys(tabLabels) as Tab[]).filter(
    tab => {
      if (tab === "metrics") return Boolean(fileStats);
      if (tab === "code-quality") return tidinessAvailable;
      return true;
    }
  );

  const captureBanner = isCapturing && (
    <div className="flex items-center gap-2 px-4 py-2 bg-blue-950/50 border-b border-blue-900/50 text-blue-300 text-xs">
      <Loader2 className="h-3.5 w-3.5 animate-spin" />
      Capturing screenshots...
    </div>
  );

  const tabNav = (mobile: boolean) => (
    <div className="flex border-b border-slate-800 px-4">
      {visibleTabs.map((tab) => (
        <button
          key={tab}
          type="button"
          onClick={() => setActiveTab(tab)}
          className={`px-4 ${mobile ? "py-3 text-sm" : "py-2 text-xs"} font-medium border-b-2 transition-colors ${
            activeTab === tab
              ? "text-blue-400 border-blue-400"
              : "text-slate-400 border-transparent hover:text-slate-200"
          }`}
        >
          {tabLabels[tab]}
        </button>
      ))}
    </div>
  );

  const tabContent = (
    <div className="flex-1 overflow-y-auto px-4 py-4">
      {activeTab === "overview" ? (
        <OverviewTab
          after={after}
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          basAvailable={basAvailable}
          testGenieAvailable={testGenieAvailable}
          tidinessAvailable={tidinessAvailable}
          isCapturing={isCapturing}
          onCapture={() => triggerCapture.mutate(scenarioSlug)}
          fileStats={fileStats}
        />
      ) : activeTab === "metrics" ? (
        fileStats ? <AggregateMetricsContent fileStats={fileStats} /> : null
      ) : activeTab === "screenshots" ? (
        capturesQuery.isLoading ? (
          <div className="space-y-4">
            <div className="h-48 animate-pulse bg-slate-800 rounded" />
            <div className="h-48 animate-pulse bg-slate-800 rounded" />
          </div>
        ) : capturesQuery.error ? (
          <p className="text-red-400 text-sm">{capturesQuery.error.message}</p>
        ) : (
          <ScreenshotsTab
            before={before}
            after={after}
            scenarioSlug={scenarioSlug}
            isMobile={isMobile}
            basAvailable={basAvailable}
            isCapturing={isCapturing}
            onCapture={() => triggerCapture.mutate(scenarioSlug)}
          />
        )
      ) : activeTab === "videos" ? (
        capturesQuery.isLoading ? (
          <div className="h-48 animate-pulse bg-slate-800 rounded" />
        ) : capturesQuery.error ? (
          <p className="text-red-400 text-sm">{capturesQuery.error.message}</p>
        ) : (
          <VideosTab
            before={before}
            after={after}
            scenarioSlug={scenarioSlug}
            basAvailable={basAvailable}
            isCapturing={isCapturing}
            onCapture={() => triggerCapture.mutate(scenarioSlug)}
          />
        )
      ) : activeTab === "tests" ? (
        <TestsTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          testGenieAvailable={testGenieAvailable}
        />
      ) : (
        <CodeQualityTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          tidinessAvailable={tidinessAvailable}
          fileStats={fileStats}
        />
      )}
    </div>
  );

  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Scenario Review"
      >
        {/* Mobile header with pt-safe and touch-friendly close */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <div className="flex items-center gap-2">
            <ClipboardCheck className="h-4 w-4 text-slate-400" />
            <div>
              <h2 className="font-semibold text-slate-100 text-base">
                {scenarioSlug}
              </h2>
              {after && (
                <p className="text-[11px] text-slate-500 mt-0.5">
                  Last captured: {new Date(after.createdAt).toLocaleString()}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {basAvailable && (
              <button
                type="button"
                className="h-11 w-11 inline-flex items-center justify-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
                onClick={() => triggerCapture.mutate(scenarioSlug)}
                disabled={isCapturing}
                title="Re-capture"
              >
                {isCapturing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
              </button>
            )}
            <button
              type="button"
              className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
              onClick={onClose}
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>
        {captureBanner}
        {tabNav(true)}
        {tabContent}
        <div className="border-t border-slate-800 px-4 py-4 pb-safe">
          <Button variant="default" size="sm" onClick={onClose} className="w-full h-12 text-sm touch-target">
            Done
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Scenario Review"
    >
      <div className="w-full max-w-4xl max-h-[85vh] flex flex-col rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        {/* Desktop header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <div className="flex items-center gap-2">
            <ClipboardCheck className="h-4 w-4 text-slate-400" />
            <div>
              <h2 className="font-semibold text-slate-100 text-sm">
                {scenarioSlug}
              </h2>
              {after && (
                <p className="text-[11px] text-slate-500 mt-0.5">
                  Last captured: {new Date(after.createdAt).toLocaleString()}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {basAvailable && (
              <button
                type="button"
                className="h-8 px-2 inline-flex items-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
                onClick={() => triggerCapture.mutate(scenarioSlug)}
                disabled={isCapturing}
                title="Re-capture"
              >
                {isCapturing ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <RefreshCw className="h-3.5 w-3.5" />
                )}
              </button>
            )}
            <button
              type="button"
              className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
              onClick={onClose}
              aria-label="Close"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
        {captureBanner}
        {tabNav(false)}
        {tabContent}
      </div>
    </div>
  );
}

// ============================================================================
// Media Lightbox with navigation
// ============================================================================

interface LightboxItem {
  label: string;
  sublabel?: string;
  type: "image" | "video";
  url: string;
}

function MediaLightbox({
  items,
  initialIndex,
  isOpen,
  onClose,
}: {
  items: LightboxItem[];
  initialIndex: number;
  isOpen: boolean;
  onClose: () => void;
}) {
  const [index, setIndex] = useState(initialIndex);

  // Reset index when opening with a new initialIndex
  useEffect(() => {
    if (isOpen) setIndex(initialIndex);
  }, [isOpen, initialIndex]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "ArrowLeft") setIndex(i => Math.max(0, i - 1));
    if (e.key === "ArrowRight") setIndex(i => Math.min(items.length - 1, i + 1));
  }, [onClose, items.length]);

  useEffect(() => {
    if (!isOpen) return;
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, handleKeyDown]);

  if (!isOpen || items.length === 0) return null;

  const clampedIndex = Math.max(0, Math.min(index, items.length - 1));
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- clampedIndex is always in bounds after the length check above
  const current = items[clampedIndex]!;
  const hasPrev = clampedIndex > 0;
  const hasNext = clampedIndex < items.length - 1;

  return createPortal(
    <div
      className="fixed inset-0 z-[60] flex flex-col bg-black/95"
      onClick={onClose}
    >
      {/* Top info bar */}
      <div
        className="flex items-center justify-between px-4 py-3 bg-black/80 border-b border-slate-800/50"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-200 truncate">{current.label}</p>
          {current.sublabel && (
            <p className="text-[11px] text-slate-500 truncate">{current.sublabel}</p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0 ml-4">
          {items.length > 1 && (
            <span className="text-xs text-slate-500">{clampedIndex + 1} / {items.length}</span>
          )}
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Media area */}
      <div
        className="flex-1 flex items-center justify-center p-4 min-h-0"
        onClick={(e) => e.stopPropagation()}
      >
        {current.type === "image" ? (
          <img
            key={current.url}
            src={current.url}
            alt={current.label}
            className="max-w-full max-h-full object-contain rounded-lg"
          />
        ) : (
          <video
            key={current.url}
            controls
            autoPlay
            src={current.url}
            className="max-w-full max-h-full rounded-lg"
          />
        )}
      </div>

      {/* Bottom nav bar */}
      {items.length > 1 && (
        <div
          className="flex items-center justify-center gap-4 px-4 py-3 bg-black/80 border-t border-slate-800/50"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            disabled={!hasPrev}
            onClick={() => setIndex(i => i - 1)}
            className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Previous"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <div className="flex gap-1.5">
            {items.map((_, i) => (
              <button
                key={i}
                type="button"
                onClick={() => setIndex(i)}
                className={`h-2 rounded-full transition-all ${
                  i === clampedIndex ? "w-6 bg-blue-400" : "w-2 bg-slate-600 hover:bg-slate-500"
                }`}
                aria-label={`Go to item ${i + 1}`}
              />
            ))}
          </div>
          <button
            type="button"
            disabled={!hasNext}
            onClick={() => setIndex(i => i + 1)}
            className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Next"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
        </div>
      )}
    </div>,
    document.body,
  );
}

// ============================================================================
// Overview Tab
// ============================================================================

function OverviewTab({
  after,
  scenarioSlug,
  repoId,
  basAvailable,
  testGenieAvailable,
  tidinessAvailable,
  isCapturing,
  onCapture,
  fileStats,
}: {
  after?: SnapshotSetMeta;
  scenarioSlug: string;
  repoId?: string | null;
  basAvailable: boolean;
  testGenieAvailable: boolean;
  tidinessAvailable: boolean;
  isCapturing: boolean;
  onCapture: () => void;
  fileStats?: RepoFileStats;
}) {
  const testExecutions = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const latestTest = testExecutions.data?.items?.[0] as TestExecutionResult | undefined;
  const tidinessScore = useTidinessScore(scenarioSlug, tidinessAvailable, repoId);
  const tidinessStaleness = useTidinessStaleness(scenarioSlug, tidinessAvailable, repoId);

  // Readiness logic
  const hasScreenshots = after && after.screenshotCount > 0;
  const hasTests = Boolean(latestTest);
  const testsPass = latestTest?.success ?? false;
  const qualityScore = tidinessScore.data?.score ?? null;
  const hasBeenScanned = Boolean(tidinessStaleness.data?.last_scan_at) ||
    (tidinessStaleness.data ? !tidinessStaleness.data.stale_reason?.includes("no scans") : false);
  const qualityOk = hasBeenScanned && qualityScore !== null && qualityScore >= 60;

  let readiness: "green" | "yellow" | "red" = "red";
  if (hasScreenshots && hasTests && testsPass && qualityOk) {
    readiness = "green";
  } else if (hasScreenshots || hasTests || qualityOk) {
    readiness = "yellow";
  }

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

  return (
    <div className="space-y-4">
      {/* Readiness indicator */}
      <div className="flex items-center gap-2 mb-4">
        <div className={`h-3 w-3 rounded-full ${readinessColors[readiness]}`} />
        <span className="text-sm font-medium text-slate-200">{readinessLabels[readiness]}</span>
      </div>

      {/* Change Summary Card */}
      {fileStats && (() => {
        const agg = aggregateFileStats(fileStats);
        if (!agg || agg.totalFiles === 0) return null;
        return (
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
            <h3 className="text-xs font-medium text-slate-400 mb-3">Change Summary</h3>
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
        <h3 className="text-xs font-medium text-slate-400 mb-3">Visual Status</h3>
        {!basAvailable ? (
          <p className="text-xs text-slate-500">Browser Automation Studio not available</p>
        ) : !after ? (
          <div className="space-y-2">
            <p className="text-xs text-slate-500">No captures yet</p>
            <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="h-7 text-xs gap-1">
              {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
              Capture Screenshots
            </Button>
          </div>
        ) : (
          <div className="space-y-2">
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Last capture</span>
              <span className="text-slate-200">{new Date(after.createdAt).toLocaleString()}</span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Screenshots</span>
              <span className="text-slate-200">{after.screenshotCount}</span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Pages captured</span>
              <span className="text-slate-200">{after.pages?.length ?? 0}</span>
            </div>
            {after.pageDiscoveryMethod === "fallback" && (
              <div className="flex items-start gap-2 mt-2 p-2 rounded bg-amber-950/30 border border-amber-900/40">
                <AlertTriangle className="h-3.5 w-3.5 text-amber-400 mt-0.5 shrink-0" />
                <p className="text-[11px] text-amber-300">
                  Pages discovered via fallback (root only). Add <code className="bg-slate-800 px-1 rounded">.vrooli/lighthouse.json</code> to capture all pages.
                </p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Test Status Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <h3 className="text-xs font-medium text-slate-400 mb-3">Test Status</h3>
        {!testGenieAvailable ? (
          <p className="text-xs text-slate-500">Test Genie not available</p>
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
        <h3 className="text-xs font-medium text-slate-400 mb-3">Code Quality</h3>
        {!tidinessAvailable ? (
          <p className="text-xs text-slate-500">Tidiness Manager not available</p>
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

// ============================================================================
// Screenshots Tab
// ============================================================================

function ScreenshotsTab({
  before,
  after,
  scenarioSlug,
  isMobile,
  basAvailable,
  isCapturing,
  onCapture,
}: {
  before?: SnapshotSetMeta;
  after?: SnapshotSetMeta;
  scenarioSlug: string;
  isMobile: boolean;
  basAvailable: boolean;
  isCapturing: boolean;
  onCapture: () => void;
}) {
  const [selectedPage, setSelectedPage] = useState(0);
  const [lightboxIndex, setLightboxIndex] = useState(-1);

  if (!after) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <ClipboardCheck className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">No captures yet</p>
        {basAvailable ? (
          <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="mt-3 h-7 text-xs gap-1">
            {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
            Capture Screenshots
          </Button>
        ) : (
          <p className="text-xs mt-1">Browser Automation Studio is not available</p>
        )}
      </div>
    );
  }

  const afterPages = after.pages ?? [];
  const currentPage = afterPages[selectedPage] ?? "/";

  // Build lightbox items: "before" first (matching left side), then "after" (matching right side)
  const lightboxItems: LightboxItem[] = [];
  const beforePages = before?.pages ?? [];
  if (before) {
    for (const page of beforePages) {
      const filename = sanitizePagePath(page) + ".png";
      lightboxItems.push({
        label: `Before: ${page === "/" ? "/ (Home)" : page}`,
        sublabel: new Date(before.createdAt).toLocaleString(),
        type: "image",
        url: buildCaptureScreenshotUrl(before.id, scenarioSlug, filename),
      });
    }
  }
  for (const page of afterPages) {
    const filename = sanitizePagePath(page) + ".png";
    lightboxItems.push({
      label: before ? `After: ${page === "/" ? "/ (Home)" : page}` : page === "/" ? "/ (Home)" : page,
      sublabel: new Date(after.createdAt).toLocaleString(),
      type: "image",
      url: buildCaptureScreenshotUrl(after.id, scenarioSlug, filename),
    });
  }

  // Map click targets to lightbox indices
  const beforeIndex = (pageIdx: number) => pageIdx;
  const afterIndex = (pageIdx: number) => beforePages.length + pageIdx;

  return (
    <div className="space-y-4">
      {/* Fallback warning */}
      {after.pageDiscoveryMethod === "fallback" && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Pages discovered via fallback (root only). Add <code className="bg-slate-800 px-1 rounded">.vrooli/lighthouse.json</code> to capture all pages.
          </p>
        </div>
      )}

      {/* Page selector */}
      {afterPages.length > 1 && (
        <div className="flex gap-1.5 overflow-x-auto pb-2">
          {afterPages.map((page, i) => (
            <button
              key={page}
              type="button"
              onClick={() => setSelectedPage(i)}
              className={`px-2.5 py-1 rounded-full text-xs whitespace-nowrap transition-colors ${
                i === selectedPage
                  ? "bg-blue-600 text-white"
                  : "bg-slate-800 text-slate-400 hover:text-slate-200"
              }`}
            >
              {page === "/" ? "/ (Home)" : page}
            </button>
          ))}
        </div>
      )}

      {/* Current page URL label */}
      <div className="text-xs text-slate-500">
        Page: <code className="bg-slate-800 px-1.5 py-0.5 rounded text-slate-300">{currentPage}</code>
        {currentPage === "/" && " (Home)"}
      </div>

      {/* Comparison status */}
      {!before && (
        <div className="text-xs text-slate-500 bg-slate-900/50 rounded px-3 py-2">
          Capture again to compare before/after
        </div>
      )}

      {/* Side-by-side or stacked screenshots */}
      <div className={`gap-4 ${isMobile ? "space-y-4" : "grid grid-cols-2"}`}>
        {before && (
          <div>
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs font-medium text-slate-400">Before</span>
              <span className="text-[10px] text-slate-600">
                {new Date(before.createdAt).toLocaleString()}
              </span>
            </div>
            <ScreenshotImage
              captureId={before.id}
              scenarioSlug={scenarioSlug}
              pagePath={currentPage}
              onClick={() => setLightboxIndex(beforeIndex(selectedPage))}
            />
          </div>
        )}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-medium text-slate-400">{before ? "After" : "Current"}</span>
            {before && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/50 text-amber-300">
                Changed
              </span>
            )}
          </div>
          <ScreenshotImage
            captureId={after.id}
            scenarioSlug={scenarioSlug}
            pagePath={currentPage}
            onClick={() => setLightboxIndex(afterIndex(selectedPage))}
          />
        </div>
      </div>

      <MediaLightbox
        items={lightboxItems}
        initialIndex={lightboxIndex}
        isOpen={lightboxIndex >= 0}
        onClose={() => setLightboxIndex(-1)}
      />
    </div>
  );
}

function ScreenshotImage({
  captureId,
  scenarioSlug,
  pagePath,
  onClick,
}: {
  captureId: string;
  scenarioSlug: string;
  pagePath: string;
  onClick: () => void;
}) {
  const filename = sanitizePagePath(pagePath) + ".png";
  const url = buildCaptureScreenshotUrl(captureId, scenarioSlug, filename);

  return (
    <div
      className="rounded-lg border border-slate-800 overflow-hidden bg-slate-900 cursor-pointer hover:ring-2 hover:ring-blue-500/50 transition-shadow"
      onClick={onClick}
    >
      <img
        src={url}
        alt={`Screenshot of ${pagePath}`}
        className="max-w-full object-contain"
        loading="lazy"
      />
    </div>
  );
}

// ============================================================================
// Videos Tab
// ============================================================================

function VideosTab({
  before,
  after,
  scenarioSlug,
  basAvailable,
  isCapturing,
  onCapture,
}: {
  before?: SnapshotSetMeta;
  after?: SnapshotSetMeta;
  scenarioSlug: string;
  basAvailable: boolean;
  isCapturing: boolean;
  onCapture: () => void;
}) {
  const [lightboxIndex, setLightboxIndex] = useState(-1);

  if (!after || after.videoCount === 0) {
    const reason = after?.videoStatus === "not_implemented"
      ? "Video recording is not yet supported. Screenshots are available."
      : after?.videoStatus === "failed"
      ? "Video recording failed during capture."
      : !after
      ? "No captures yet"
      : "No videos were recorded for this capture.";

    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm">{reason}</p>
        {basAvailable && !after && (
          <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="mt-3 h-7 text-xs gap-1">
            {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
            Capture Screenshots
          </Button>
        )}
      </div>
    );
  }

  const lightboxItems: LightboxItem[] = [];
  const beforeVideoUrl = before && before.videoCount > 0
    ? buildCaptureVideoUrl(before.id, scenarioSlug, "recording.webm")
    : null;
  if (before && beforeVideoUrl) {
    lightboxItems.push({
      label: "Before",
      sublabel: new Date(before.createdAt).toLocaleString(),
      type: "video",
      url: beforeVideoUrl,
    });
  }
  const afterVideoUrl = buildCaptureVideoUrl(after.id, scenarioSlug, "recording.webm");
  lightboxItems.push({
    label: before ? "After" : "Current",
    sublabel: new Date(after.createdAt).toLocaleString(),
    type: "video",
    url: afterVideoUrl,
  });

  const afterLightboxIndex = beforeVideoUrl ? 1 : 0;

  return (
    <>
      <div className="grid grid-cols-2 gap-4">
        {beforeVideoUrl && (
          <div>
            <p className="text-xs font-medium text-slate-400 mb-2">Before</p>
            <div
              className="cursor-pointer hover:ring-2 hover:ring-blue-500/50 rounded-lg transition-shadow"
              onClick={() => setLightboxIndex(0)}
            >
              <video
                controls
                src={beforeVideoUrl}
                className="w-full rounded-lg border border-slate-800"
              />
            </div>
          </div>
        )}
        <div>
          <p className="text-xs font-medium text-slate-400 mb-2">
            {before ? "After" : "Current"}
          </p>
          <div
            className="cursor-pointer hover:ring-2 hover:ring-blue-500/50 rounded-lg transition-shadow"
            onClick={() => setLightboxIndex(afterLightboxIndex)}
          >
            <video
              controls
              src={afterVideoUrl}
              className="w-full rounded-lg border border-slate-800"
            />
          </div>
        </div>
      </div>
      <MediaLightbox
        items={lightboxItems}
        initialIndex={lightboxIndex}
        isOpen={lightboxIndex >= 0}
        onClose={() => setLightboxIndex(-1)}
      />
    </>
  );
}

// ============================================================================
// Tests Tab
// ============================================================================

function TestsTab({
  scenarioSlug,
  repoId,
  testGenieAvailable,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
}) {
  const testExecutions = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const triggerTest = useTriggerTestExecution(repoId);
  const [expandedPhase, setExpandedPhase] = useState<string | null>(null);

  if (!testGenieAvailable) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm">Test Genie is not available</p>
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
          Start the test-genie scenario to run automated tests
        </p>
      </div>
    );
  }

  const isRunning = triggerTest.isPending;
  const executions = testExecutions.data?.items ?? [];
  const latest = executions[0] as TestExecutionResult | undefined;
  const history = executions.slice(1, 6);

  return (
    <div className="space-y-4">
      {/* Run Tests button */}
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-medium text-slate-400">Test Execution</h3>
        <Button
          variant="outline"
          size="sm"
          onClick={() => triggerTest.mutate({ scenarioName: scenarioSlug })}
          disabled={isRunning}
          className="h-7 text-xs gap-1"
        >
          {isRunning ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <Play className="h-3 w-3" />
          )}
          Run Tests
        </Button>
      </div>

      {/* In-progress banner */}
      {isRunning && (
        <div className="flex items-center gap-2 px-3 py-2 bg-blue-950/50 border border-blue-900/50 rounded-lg text-blue-300 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Running tests...
        </div>
      )}

      {/* Loading state */}
      {testExecutions.isLoading ? (
        <div className="space-y-3">
          <div className="h-24 animate-pulse bg-slate-800 rounded" />
          <div className="h-16 animate-pulse bg-slate-800 rounded" />
        </div>
      ) : !latest ? (
        <div className="flex flex-col items-center justify-center py-8 text-slate-500">
          <p className="text-sm">No test results yet</p>
          <p className="text-xs mt-1">Run tests to see results here</p>
        </div>
      ) : (
        <>
          {/* Latest execution card */}
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {latest.success ? (
                  <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                ) : (
                  <XCircle className="h-4 w-4 text-red-400" />
                )}
                <span className={`text-sm font-medium ${latest.success ? "text-emerald-300" : "text-red-300"}`}>
                  {latest.success ? "All Passed" : "Failed"}
                </span>
              </div>
              <span className="text-[11px] text-slate-500">
                {latest.completedAt ? new Date(latest.completedAt).toLocaleString() : latest.startedAt}
              </span>
            </div>
            <div className="flex gap-4 text-xs text-slate-400">
              <span>{latest.phaseSummary.total} total</span>
              <span className="text-emerald-400">{latest.phaseSummary.passed} passed</span>
              {latest.phaseSummary.failed > 0 && (
                <span className="text-red-400">{latest.phaseSummary.failed} failed</span>
              )}
              <span>{formatDuration(latest.phaseSummary.durationSeconds)}</span>
            </div>
            {latest.preset && (
              <div className="text-[11px] text-slate-500">Preset: {latest.preset}</div>
            )}
            {latest.warnings && latest.warnings.length > 0 && (
              <div className="space-y-1">
                {latest.warnings.map((w, i) => (
                  <div key={i} className="flex items-start gap-1.5 text-[11px] text-amber-400">
                    <AlertTriangle className="h-3 w-3 mt-0.5 shrink-0" />
                    {w}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Phase list */}
          <div className="space-y-1">
            <h4 className="text-xs font-medium text-slate-400 mb-2">Phases</h4>
            {latest.phases.map((phase) => (
              <PhaseRow
                key={phase.name}
                phase={phase}
                expanded={expandedPhase === phase.name}
                onToggle={() => setExpandedPhase(expandedPhase === phase.name ? null : phase.name)}
              />
            ))}
          </div>

          {/* History */}
          {history.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-slate-400 mb-2">Recent History</h4>
              <div className="space-y-1">
                {history.map((exec) => (
                  <div
                    key={exec.executionId}
                    className="flex items-center justify-between px-3 py-2 rounded bg-slate-900/50 border border-slate-800/50"
                  >
                    <div className="flex items-center gap-2">
                      {exec.success ? (
                        <div className="h-2 w-2 rounded-full bg-emerald-500" />
                      ) : (
                        <div className="h-2 w-2 rounded-full bg-red-500" />
                      )}
                      <span className="text-[11px] text-slate-400">
                        {exec.completedAt ? new Date(exec.completedAt).toLocaleString() : exec.startedAt}
                      </span>
                    </div>
                    <div className="flex gap-3 text-[11px] text-slate-500">
                      {exec.preset && <span>{exec.preset}</span>}
                      <span>{formatDuration(exec.phaseSummary.durationSeconds)}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ============================================================================
// Phase Row
// ============================================================================

function PhaseRow({
  phase,
  expanded,
  onToggle,
}: {
  phase: TestPhaseResult;
  expanded: boolean;
  onToggle: () => void;
}) {
  const hasDetails = Boolean(phase.error || phase.remediation || (phase.observations && phase.observations.length > 0));

  return (
    <div className="rounded border border-slate-800/50 bg-slate-900/30">
      <button
        type="button"
        onClick={hasDetails ? onToggle : undefined}
        className={`w-full flex items-center justify-between px-3 py-2 text-xs ${hasDetails ? "cursor-pointer hover:bg-slate-800/30" : "cursor-default"}`}
      >
        <div className="flex items-center gap-2">
          {hasDetails ? (
            expanded ? <ChevronDown className="h-3 w-3 text-slate-500" /> : <ChevronRight className="h-3 w-3 text-slate-500" />
          ) : (
            <div className="w-3" />
          )}
          <div className={`h-2 w-2 rounded-full ${phase.status === "passed" ? "bg-emerald-500" : "bg-red-500"}`} />
          <span className="text-slate-200">{phase.name}</span>
        </div>
        <span className="text-slate-500">{formatDuration(phase.durationSeconds)}</span>
      </button>

      {expanded && hasDetails && (
        <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-2">
          {phase.error && (
            <div className="text-[11px] text-red-400 bg-red-950/30 rounded px-2 py-1.5">
              {phase.error}
            </div>
          )}
          {phase.classification && (
            <div className="text-[11px] text-slate-400">
              <span className="text-slate-500">Classification:</span> {phase.classification}
            </div>
          )}
          {phase.remediation && (
            <div className="text-[11px] text-amber-300 bg-amber-950/20 rounded px-2 py-1.5">
              {phase.remediation}
            </div>
          )}
          {phase.observations && phase.observations.length > 0 && (
            <div className="space-y-1">
              {phase.observations.map((obs, i) => (
                <div key={i} className="text-[11px] text-slate-400 flex gap-1.5">
                  {obs.icon && <span>{obs.icon}</span>}
                  {obs.prefix && (
                    <span className={`font-medium ${
                      obs.prefix === "ERROR" ? "text-red-400" :
                      obs.prefix === "WARNING" ? "text-amber-400" :
                      obs.prefix === "SUCCESS" ? "text-emerald-400" : "text-slate-400"
                    }`}>
                      {obs.prefix}:
                    </span>
                  )}
                  <span>{obs.text}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Code Quality Tab
// ============================================================================

function CodeQualityTab({
  scenarioSlug,
  repoId,
  tidinessAvailable,
  fileStats,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  tidinessAvailable: boolean;
  fileStats?: RepoFileStats;
}) {
  const [view, setView] = useState<"changed" | "scenario">("changed");
  const tidinessScore = useTidinessScore(scenarioSlug, tidinessAvailable, repoId);
  const tidinessIssues = useTidinessIssues(scenarioSlug, undefined, tidinessAvailable, repoId);
  const tidinessStaleness = useTidinessStaleness(scenarioSlug, tidinessAvailable, repoId);
  const triggerScan = useTriggerTidinessScan(repoId);

  if (!tidinessAvailable) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm">Tidiness Manager is not available</p>
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
          Start the tidiness-manager scenario to view code quality data
        </p>
      </div>
    );
  }

  // Use staleness as source of truth for "has been scanned" — more reliable than score.last_scan
  const stalenessLoaded = !tidinessStaleness.isLoading;
  const neverScanned = stalenessLoaded && (
    !tidinessStaleness.data?.last_scan_at &&
    tidinessStaleness.data?.stale_reason?.includes("no scans")
  );
  const isScanning = triggerScan.isPending;
  const scanResult = triggerScan.data;

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
        />
      ) : (
        <ScenarioWideView
          scoreData={tidinessScore.data}
          isLoading={tidinessScore.isLoading}
        />
      )}
    </div>
  );
}

function ChangedFilesView({
  issues,
  issuesByFile,
  changedFiles,
  scoreData,
  isLoading,
}: {
  issues: TidinessIssue[];
  issuesByFile: Map<string, TidinessIssue[]>;
  changedFiles: string[];
  scoreData?: { score: number; violations: number } | null;
  isLoading: boolean;
}) {
  const [expandedFile, setExpandedFile] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="h-16 animate-pulse bg-slate-800 rounded" />
        <div className="h-16 animate-pulse bg-slate-800 rounded" />
      </div>
    );
  }

  if (changedFiles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-slate-500">
        <p className="text-sm">No changed files to analyze</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Scenario-wide score badge for context */}
      {scoreData && (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span>Scenario score:</span>
          <span className={`font-medium ${
            scoreData.score >= 70 ? "text-emerald-400" :
            scoreData.score >= 40 ? "text-amber-400" : "text-red-400"
          }`}>
            {Math.round(scoreData.score)}/100
          </span>
        </div>
      )}

      {issues.length === 0 ? (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
            <span className="text-xs text-emerald-300 font-medium">No issues in changed files</span>
          </div>
        </div>
      ) : (
        <div className="space-y-1">
          {Array.from(issuesByFile.entries()).map(([filePath, fileIssues]) => (
            <div key={filePath} className="rounded border border-slate-800/50 bg-slate-900/30">
              <button
                type="button"
                onClick={() => setExpandedFile(expandedFile === filePath ? null : filePath)}
                className="w-full flex items-center justify-between px-3 py-2 text-xs cursor-pointer hover:bg-slate-800/30"
              >
                <div className="flex items-center gap-2">
                  {expandedFile === filePath ? (
                    <ChevronDown className="h-3 w-3 text-slate-500" />
                  ) : (
                    <ChevronRight className="h-3 w-3 text-slate-500" />
                  )}
                  <code className="text-slate-200">{filePath}</code>
                  <span className="text-slate-500">({fileIssues.length} issue{fileIssues.length !== 1 ? "s" : ""})</span>
                </div>
              </button>
              {expandedFile === filePath && (
                <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-1.5">
                  {fileIssues.map(issue => (
                    <div key={issue.id} className="flex items-start gap-2 text-[11px]">
                      <div className={`h-1.5 w-1.5 rounded-full mt-1.5 shrink-0 ${
                        issue.severity === "critical" || issue.severity === "high" ? "bg-red-500" :
                        issue.severity === "medium" ? "bg-amber-500" : "bg-blue-500"
                      }`} />
                      <div className="min-w-0">
                        <span className="text-slate-500">{issue.category}:</span>{" "}
                        <span className="text-slate-300">{issue.title}</span>
                        {issue.line_number != null && (
                          <span className="text-slate-600 ml-1">L:{issue.line_number}</span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ScenarioWideView({
  scoreData,
  isLoading,
}: {
  scoreData?: {
    score: number;
    violations: number;
    breakdown?: {
      lint_issues: number;
      type_issues: number;
      long_files: number;
      complex_functions: number;
      tech_debt_markers: number;
      duplication_issues: number;
    };
    metrics?: {
      total_files: number;
      total_lines: number;
      avg_file_length: number;
      max_complexity: number;
      avg_complexity: number;
      duplication_pct: number;
    };
  } | null;
  isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="h-24 animate-pulse bg-slate-800 rounded" />
        <div className="h-32 animate-pulse bg-slate-800 rounded" />
      </div>
    );
  }

  if (!scoreData) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-slate-500">
        <p className="text-sm">No quality data available</p>
        <p className="text-xs mt-1">Run a scan to generate quality metrics</p>
      </div>
    );
  }

  const scoreColor = scoreData.score >= 70 ? "bg-emerald-500" :
    scoreData.score >= 40 ? "bg-amber-500" : "bg-red-500";
  const scoreTextColor = scoreData.score >= 70 ? "text-emerald-300" :
    scoreData.score >= 40 ? "text-amber-300" : "text-red-300";

  return (
    <div className="space-y-4">
      {/* Score bar */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <span className={`text-2xl font-bold ${scoreTextColor}`}>
            {Math.round(scoreData.score)}
          </span>
          <span className="text-xs text-slate-500">/100</span>
        </div>
        <div className="w-full h-2 bg-slate-800 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${scoreColor}`}
            style={{ width: `${Math.min(100, Math.max(0, scoreData.score))}%` }}
          />
        </div>
        <div className="flex justify-between text-xs">
          <span className="text-slate-400">Violations</span>
          <span className="text-slate-200">{scoreData.violations}</span>
        </div>
      </div>

      {/* Breakdown */}
      {scoreData.breakdown && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <h4 className="text-xs font-medium text-slate-400 mb-3">Breakdown</h4>
          <div className="space-y-2">
            {([
              ["Lint issues", scoreData.breakdown.lint_issues],
              ["Type issues", scoreData.breakdown.type_issues],
              ["Long files", scoreData.breakdown.long_files],
              ["Complex functions", scoreData.breakdown.complex_functions],
              ["Tech debt markers", scoreData.breakdown.tech_debt_markers],
              ["Duplication", scoreData.breakdown.duplication_issues],
            ] as const).map(([label, value]) => (
              <div key={label} className="flex justify-between text-xs">
                <span className="text-slate-400">{label}</span>
                <span className={value > 0 ? "text-slate-200" : "text-slate-600"}>{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Metrics */}
      {scoreData.metrics && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <h4 className="text-xs font-medium text-slate-400 mb-3">Metrics</h4>
          <div className="space-y-2">
            {([
              ["Total files", String(scoreData.metrics.total_files)],
              ["Total lines", scoreData.metrics.total_lines.toLocaleString()],
              ["Avg file length", String(Math.round(scoreData.metrics.avg_file_length))],
              ["Max complexity", String(scoreData.metrics.max_complexity)],
              ["Avg complexity", scoreData.metrics.avg_complexity.toFixed(1)],
              ["Duplication", `${scoreData.metrics.duplication_pct.toFixed(1)}%`],
            ] as const).map(([label, value]) => (
              <div key={label} className="flex justify-between text-xs">
                <span className="text-slate-400">{label}</span>
                <span className="text-slate-200">{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Helpers
// ============================================================================

function sanitizePagePath(pagePath: string): string {
  if (pagePath === "/" || pagePath === "") return "_root_";
  let s = pagePath.startsWith("/") ? pagePath.slice(1) : pagePath;
  s = s.endsWith("/") ? s.slice(0, -1) : s;
  s = s.replace(/\//g, "_");
  return "_" + s + "_";
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
}

function formatRelativeTime(isoString: string): string {
  const date = new Date(isoString);
  const now = Date.now();
  const diffMs = now - date.getTime();
  if (diffMs < 0) return "just now";
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return "just now";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

function formatStalenessMessage(staleness: TidinessStalenessInfo): string {
  const parts: string[] = [];
  if (staleness.stale_reason) {
    parts.push(staleness.stale_reason);
  } else {
    parts.push("Quality data may be stale");
  }
  if (staleness.last_scan_at) {
    parts.push(`Last scan: ${formatRelativeTime(staleness.last_scan_at)}`);
  }
  if (staleness.modified_files && staleness.modified_files > 0 && !staleness.stale_reason?.includes("file")) {
    parts.push(`${staleness.modified_files} file${staleness.modified_files !== 1 ? "s" : ""} changed`);
  }
  return parts.join(" · ");
}

function ScanResultSummary({ result }: { result: TidinessLightScanResult }) {
  const durationSec = (result.duration_ms / 1000).toFixed(1);
  const totalIssues = result.lint_issues + result.type_issues;
  return (
    <div className="flex items-center gap-2 px-3 py-2 bg-slate-800/50 border border-slate-700/50 rounded-lg text-xs text-slate-300 mt-2">
      <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
      <span>
        Scanned {result.total_files} files ({result.total_lines.toLocaleString()} lines) in {durationSec}s
        {" — "}
        {totalIssues === 0 ? (
          <span className="text-emerald-400">no issues found</span>
        ) : (
          <span className="text-amber-400">
            {result.lint_issues} lint, {result.type_issues} type, {result.long_files_count} long file{result.long_files_count !== 1 ? "s" : ""}
          </span>
        )}
      </span>
    </div>
  );
}
