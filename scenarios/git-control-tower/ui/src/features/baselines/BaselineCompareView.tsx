// BaselineCompareView (Plan B §4.2/§4.3) — "<baseline> vs. working tree".
//
// Running a diff re-executes the surfaces server-side (BAS + test-genie +
// auditor), which takes minutes, so the comparison is an explicit action, not
// an on-mount fetch. Once run, each surface renders through its focused diff
// component.

import { useState } from "react";
import { ArrowLeft, GitCompare, Loader2 } from "lucide-react";
import { Button } from "../../components/ui/button";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useBaselineDiff } from "../../lib/hooks-baselines";
import { verdictMeta } from "./model";
import { VerdictBadge } from "./parts";
import { BaselineDetailView } from "./BaselineDetailView";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import { WorkflowsDiff } from "./diffs/WorkflowsDiff";
import { TestsDiff } from "./diffs/TestsDiff";
import { StructureDiff } from "./diffs/StructureDiff";
import { VisualsDiff } from "./diffs/VisualsDiff";
import { RulesDiff } from "./diffs/RulesDiff";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

interface BaselineCompareViewProps {
  scenario: string;
  baseline: BaselineManifest;
  repoId?: string | null;
  onBack: () => void;
  onOpenWorkflows?: () => void;
  onOpenTests?: () => void;
  onOpenScreenshots?: () => void;
}

function renderSurface(
  diff: SurfaceDiff,
  cbs: Pick<BaselineCompareViewProps, "onOpenWorkflows" | "onOpenTests" | "onOpenScreenshots">,
) {
  switch (diff.surfaceId) {
    case "workflows":
      return <WorkflowsDiff key="workflows" diff={diff} onOpenWorkflows={cbs.onOpenWorkflows} />;
    case "tests":
      return <TestsDiff key="tests" diff={diff} onOpenTests={cbs.onOpenTests} />;
    case "structure":
      return <StructureDiff key="structure" diff={diff} />;
    case "visuals":
      return <VisualsDiff key="visuals" diff={diff} onOpenScreenshots={cbs.onOpenScreenshots} />;
    case "rules":
      return <RulesDiff key="rules" diff={diff} />;
    default:
      return null;
  }
}

export function BaselineCompareView({
  scenario,
  baseline,
  repoId,
  onBack,
  onOpenWorkflows,
  onOpenTests,
  onOpenScreenshots,
}: BaselineCompareViewProps) {
  const [started, setStarted] = useState(false);
  const diffQuery = useBaselineDiff(scenario, baseline.name, baseline.branch, { enabled: started, repoId });

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Back
        </button>
        <span className="text-sm font-medium text-slate-200">{baseline.name}</span>
        <span className="text-xs text-slate-500">vs. working tree</span>
      </div>

      <BaselineDetailView baseline={baseline} />

      {!started ? (
        <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-4 text-center space-y-3">
          <p className="text-xs text-slate-400">
            Comparing re-runs each captured surface against your current working tree. This can take
            a few minutes.
          </p>
          <Button size="sm" onClick={() => setStarted(true)} className="gap-1.5">
            <GitCompare className="h-4 w-4" />
            Compare against working tree
          </Button>
        </div>
      ) : diffQuery.isLoading || diffQuery.isFetching ? (
        <div className="flex items-center gap-2 text-xs text-slate-400 py-8 justify-center">
          <Loader2 className="h-4 w-4 animate-spin" />
          Running surfaces — this can take a few minutes.
        </div>
      ) : diffQuery.error ? (
        <MutationErrorBanner error={diffQuery.error} />
      ) : diffQuery.data ? (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-slate-500">Overall</span>
            <VerdictBadge verdict={diffQuery.data.verdict} />
            {diffQuery.data.staleness?.likelyStale && (
              <span className="text-xs text-amber-400">
                {diffQuery.data.staleness.commitsSince} commits / {diffQuery.data.staleness.filesChanged} files
                since baseline
              </span>
            )}
          </div>
          {diffQuery.data.dirtyWarning && (
            <div className="rounded-lg border border-amber-900/40 bg-amber-950/30 p-3 text-xs text-amber-300">
              {diffQuery.data.dirtyWarning}
            </div>
          )}
          {diffQuery.data.surfaces.length === 0 ? (
            <p className="text-xs text-slate-500">This baseline pinned no comparable surfaces.</p>
          ) : (
            diffQuery.data.surfaces.map((s) =>
              renderSurface(s, { onOpenWorkflows, onOpenTests, onOpenScreenshots }),
            )
          )}
          {/* Verdict legend uses the same vocabulary as the per-surface badges. */}
          <p className="text-[11px] text-slate-600">
            {verdictMeta("regression").label}: your change broke something.{" "}
            {verdictMeta("preexisting").label}: already failing at capture time.
          </p>
        </div>
      ) : null}
    </div>
  );
}
