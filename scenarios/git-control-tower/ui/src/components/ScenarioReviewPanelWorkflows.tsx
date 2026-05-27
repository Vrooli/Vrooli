// Workflows tab (Plan B §4.4) — a focused, read-only view of test-genie
// playbooks-phase runs. It no longer owns baseline state (Decision 2): baselines
// are created and compared in the Baselines tab. Runs come through GCT's
// WorkflowReplayService proxy; videos stream from the GCT REST video route.

import { useState } from "react";
import { Play, CheckCircle2, XCircle, Minus, AlertTriangle, ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { Button } from "./ui/button";
import { MediaLightbox, MutationErrorBanner, formatDuration, formatRelativeTime, type LightboxItem } from "./ScenarioReviewPanelShared";
import { useRecentRuns, useRunDetail } from "../lib/hooks-workflowreplay";
import { useTriggerTestExecution } from "../lib/hooks";
import { workflowVideoUrl, type RunSummary } from "../lib/api-workflowreplay";
import { SurfaceComparePanel } from "../features/baselines/SurfaceComparePanel";
import { SurfaceCaptureEmptyState } from "../features/baselines/SurfaceCaptureEmptyState";
import { useSurfaceBaselineModal } from "../features/baselines/useSurfaceBaselineModal";

const STATUS_ICON: Record<string, typeof CheckCircle2> = {
  passed: CheckCircle2,
  failed: XCircle,
  skipped: Minus,
  aborted: AlertTriangle,
  in_progress: Loader2,
};

function statusColor(status: string): string {
  if (status === "passed") return "text-green-400";
  if (status === "failed" || status === "aborted") return "text-red-400";
  return "text-slate-500";
}

export function WorkflowsTab({
  scenarioSlug,
  repoId,
  testGenieAvailable,
  onOpenBaselines,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  onOpenBaselines: () => void;
}) {
  const runsQuery = useRecentRuns(scenarioSlug, { repoId, enabled: testGenieAvailable });
  const runs = runsQuery.data ?? [];
  const triggerRun = useTriggerTestExecution(repoId);
  const { openCaptureBaseline, baselineModal } = useSurfaceBaselineModal(scenarioSlug, "workflows", repoId);
  const isRunning = triggerRun.isPending;
  const runPlaybooks = () =>
    triggerRun.mutate(
      { scenarioName: scenarioSlug, phases: ["playbooks"] },
      { onSuccess: () => void runsQuery.refetch() },
    );

  if (!testGenieAvailable) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <Play className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">test-genie is not available</p>
        <p className="text-xs mt-1 text-slate-600">Start test-genie to see playbooks runs</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {baselineModal}
      <MutationErrorBanner error={triggerRun.error} onDismiss={() => triggerRun.reset()} />

      {runsQuery.isLoading ? (
        <div className="space-y-2">
          <div className="h-12 animate-pulse rounded bg-slate-800/60" />
          <div className="h-12 animate-pulse rounded bg-slate-800/60" />
        </div>
      ) : runsQuery.error ? (
        <MutationErrorBanner error={runsQuery.error} />
      ) : runs.length === 0 ? (
        <SurfaceCaptureEmptyState
          surface="workflows"
          hasService={testGenieAvailable}
          onCaptureLoose={runPlaybooks}
          onCaptureBaseline={openCaptureBaseline}
          captureLabel="Run playbooks"
          isCapturing={isRunning}
        />
      ) : (
        <>
          <SurfaceComparePanel
            scenario={scenarioSlug}
            surface="workflows"
            repoId={repoId}
            onOpenBaselines={onOpenBaselines}
            onCaptureBaseline={openCaptureBaseline}
            viewingLabel={`${runs.length} recent run${runs.length !== 1 ? "s" : ""}`}
          />

          <div className="flex items-center justify-between gap-2">
            <p className="text-xs text-slate-500">
              Workflow results from the latest test-genie playbooks runs.
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={runPlaybooks}
              disabled={isRunning}
              className="h-7 px-3 gap-1.5 shrink-0"
            >
              {isRunning ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}
              Run playbooks
            </Button>
          </div>

          <div className="space-y-2">
            {runs.map((run) => (
              <RunRow key={run.runId} run={run} scenario={scenarioSlug} repoId={repoId} />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function RunRow({ run, scenario, repoId }: { run: RunSummary; scenario: string; repoId?: string | null }) {
  const [expanded, setExpanded] = useState(false);
  const [lightboxIndex, setLightboxIndex] = useState(-1);
  const detail = useRunDetail(scenario, run.runId, { repoId, enabled: expanded });

  const Icon = STATUS_ICON[run.playbooksStatus || run.status] ?? AlertTriangle;
  const sha8 = run.gitSha ? run.gitSha.slice(0, 8) : "";

  const lightboxItems: LightboxItem[] = (detail.data?.videos ?? []).map((v): LightboxItem => ({
    label: v.workflow,
    sublabel: run.runId,
    type: "video",
    url: workflowVideoUrl(scenario, run.runId, v.relPath),
  }));

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-slate-800/30 rounded-lg"
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5 text-slate-500 shrink-0" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-slate-500 shrink-0" />
        )}
        <Icon className={`h-4 w-4 shrink-0 ${statusColor(run.playbooksStatus || run.status)}`} />
        <span className="text-xs text-slate-300 font-mono truncate">{run.runId}</span>
        <div className="ml-auto flex items-center gap-3 text-[11px] text-slate-500 shrink-0">
          {run.playbooksDurationSeconds > 0 && (
            <span>{formatDuration(Math.round(run.playbooksDurationSeconds))}</span>
          )}
          {sha8 && <span className="font-mono">{sha8}</span>}
          {run.gitDirty && <span className="text-amber-500">dirty</span>}
          {run.startedAt && <span>{formatRelativeTime(run.startedAt)}</span>}
        </div>
      </button>

      {expanded && (
        <div className="border-t border-slate-800/60 px-3 py-2 space-y-2">
          {detail.isLoading ? (
            <div className="flex items-center gap-2 text-xs text-slate-500">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Loading run detail…
            </div>
          ) : detail.error ? (
            <MutationErrorBanner error={detail.error} />
          ) : lightboxItems.length === 0 ? (
            <p className="text-xs text-slate-500">No videos recorded for this run.</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {lightboxItems.map((item, i) => (
                <button
                  key={`${item.label}-${i}`}
                  type="button"
                  onClick={() => setLightboxIndex(i)}
                  className="inline-flex items-center gap-1 rounded border border-slate-700 px-2 py-1 text-[11px] text-blue-400 hover:text-blue-300 hover:bg-slate-800/60"
                >
                  <Play className="h-3 w-3" />
                  {item.label}
                </button>
              ))}
            </div>
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
