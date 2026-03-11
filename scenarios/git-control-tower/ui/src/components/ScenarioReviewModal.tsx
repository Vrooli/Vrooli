import { useState } from "react";
import { X, ClipboardCheck, RefreshCw, Loader2, Play, CheckCircle2, XCircle, AlertTriangle, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";
import { useVisualCaptures, useTriggerVisualCapture, useCapabilities, useTestExecutions, useTriggerTestExecution } from "../lib/hooks";
import { buildCaptureScreenshotUrl, buildCaptureVideoUrl } from "../lib/api";
import type { SnapshotSetMeta, TestExecutionResult, TestPhaseResult } from "../lib/api";

type Tab = "overview" | "screenshots" | "videos" | "tests";

interface ScenarioReviewModalProps {
  isOpen: boolean;
  onClose: () => void;
  scenarioSlug: string;
  repoId?: string | null;
}

export function ScenarioReviewModal({ isOpen, onClose, scenarioSlug, repoId }: ScenarioReviewModalProps) {
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

  if (!isOpen) return null;

  const snapshots = capturesQuery.data?.snapshots ?? [];
  const completeSnapshots = snapshots.filter(s => s.status === "complete");
  const after = completeSnapshots[0] as SnapshotSetMeta | undefined;
  const before = completeSnapshots[1] as SnapshotSetMeta | undefined;

  const isCapturing = triggerCapture.isPending;

  const tabLabels: Record<Tab, string> = {
    overview: "Overview",
    screenshots: "Screenshots",
    videos: "Videos",
    tests: "Tests",
  };

  const content = (
    <>
      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
        <div className="flex items-center gap-2">
          <ClipboardCheck className="h-4 w-4 text-slate-400" />
          <div>
            <h2 className={`font-semibold text-slate-100 ${isMobile ? "text-base" : "text-sm"}`}>
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
            className={`inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 ${
              isMobile ? "h-11 w-11" : "h-8 w-8"
            }`}
            onClick={onClose}
            aria-label="Close"
          >
            <X className={isMobile ? "h-5 w-5" : "h-4 w-4"} />
          </button>
        </div>
      </div>

      {/* Capture in progress banner */}
      {isCapturing && (
        <div className="flex items-center gap-2 px-4 py-2 bg-blue-950/50 border-b border-blue-900/50 text-blue-300 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Capturing screenshots...
        </div>
      )}

      {/* Tab Navigation */}
      <div className="flex border-b border-slate-800 px-4">
        {(Object.keys(tabLabels) as Tab[]).map((tab) => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-xs font-medium border-b-2 transition-colors ${
              activeTab === tab
                ? "text-blue-400 border-blue-400"
                : "text-slate-400 border-transparent hover:text-slate-200"
            }`}
          >
            {tabLabels[tab]}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-4 py-4">
        {activeTab === "overview" ? (
          <OverviewTab
            after={after}
            scenarioSlug={scenarioSlug}
            repoId={repoId}
            basAvailable={basAvailable}
            testGenieAvailable={testGenieAvailable}
          />
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
            />
          )
        ) : activeTab === "videos" ? (
          capturesQuery.isLoading ? (
            <div className="h-48 animate-pulse bg-slate-800 rounded" />
          ) : capturesQuery.error ? (
            <p className="text-red-400 text-sm">{capturesQuery.error.message}</p>
          ) : (
            <VideosTab before={before} after={after} scenarioSlug={scenarioSlug} />
          )
        ) : (
          <TestsTab
            scenarioSlug={scenarioSlug}
            repoId={repoId}
            testGenieAvailable={testGenieAvailable}
          />
        )}
      </div>
    </>
  );

  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Scenario Review"
      >
        {content}
        <div className="border-t border-slate-800 px-4 py-4 pb-safe">
          <Button variant="default" size="sm" onClick={onClose} className="w-full h-12 text-sm">
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
        {content}
      </div>
    </div>
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
}: {
  after?: SnapshotSetMeta;
  scenarioSlug: string;
  repoId?: string | null;
  basAvailable: boolean;
  testGenieAvailable: boolean;
}) {
  const testExecutions = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const latestTest = testExecutions.data?.items?.[0] as TestExecutionResult | undefined;

  // Readiness logic
  const hasScreenshots = after && after.screenshotCount > 0;
  const hasTests = Boolean(latestTest);
  const testsPass = latestTest?.success ?? false;

  let readiness: "green" | "yellow" | "red" = "red";
  if (hasScreenshots && hasTests && testsPass) {
    readiness = "green";
  } else if (hasScreenshots || hasTests) {
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

      {/* Visual Status Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <h3 className="text-xs font-medium text-slate-400 mb-3">Visual Status</h3>
        {!basAvailable ? (
          <p className="text-xs text-slate-500">Browser Automation Studio not available</p>
        ) : !after ? (
          <p className="text-xs text-slate-500">No captures yet. Click the capture button to take screenshots.</p>
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
}: {
  before?: SnapshotSetMeta;
  after?: SnapshotSetMeta;
  scenarioSlug: string;
  isMobile: boolean;
}) {
  const [selectedPage, setSelectedPage] = useState(0);

  if (!after) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <ClipboardCheck className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">No captures yet</p>
        <p className="text-xs mt-1">Click the capture button to take screenshots</p>
      </div>
    );
  }

  const afterPages = after.pages ?? [];
  const currentPage = afterPages[selectedPage] ?? "/";

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
            <ScreenshotImage captureId={before.id} scenarioSlug={scenarioSlug} pagePath={currentPage} />
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
          <ScreenshotImage captureId={after.id} scenarioSlug={scenarioSlug} pagePath={currentPage} />
        </div>
      </div>
    </div>
  );
}

function ScreenshotImage({
  captureId,
  scenarioSlug,
  pagePath,
}: {
  captureId: string;
  scenarioSlug: string;
  pagePath: string;
}) {
  const filename = sanitizePagePath(pagePath) + ".png";
  const url = buildCaptureScreenshotUrl(captureId, scenarioSlug, filename);

  return (
    <div className="rounded-lg border border-slate-800 overflow-hidden bg-slate-900">
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
}: {
  before?: SnapshotSetMeta;
  after?: SnapshotSetMeta;
  scenarioSlug: string;
}) {
  if (!after || after.videoCount === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm">No video recordings available</p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-4">
      {before && before.videoCount > 0 && (
        <div>
          <p className="text-xs font-medium text-slate-400 mb-2">Before</p>
          <video
            controls
            src={buildCaptureVideoUrl(before.id, scenarioSlug, "recording.webm")}
            className="w-full rounded-lg border border-slate-800"
          />
        </div>
      )}
      <div>
        <p className="text-xs font-medium text-slate-400 mb-2">
          {before ? "After" : "Current"}
        </p>
        <video
          controls
          src={buildCaptureVideoUrl(after.id, scenarioSlug, "recording.webm")}
          className="w-full rounded-lg border border-slate-800"
        />
      </div>
    </div>
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
