import { ArrowLeft, GitCompare, Loader2 } from "lucide-react";
import { Button } from "../../components/ui/button";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useCompareOnDemand } from "../../lib/hooks-baselines";
import { verdictMeta } from "./model";
import { PhaseDiffCard, VerdictBadge } from "./parts";
import { BaselineDetailView } from "./BaselineDetailView";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

interface BaselineCompareViewProps {
  scenario: string;
  baseline: BaselineManifest;
  repoId?: string | null;
  onBack: () => void;
}

export function BaselineCompareView({ scenario, baseline, repoId, onBack }: BaselineCompareViewProps) {
  const compare = useCompareOnDemand(scenario, { baselineName: baseline.name, branch: baseline.branch, repoId });
  const evidence = compare.diff?.evidence;
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <button type="button" onClick={onBack} className="inline-flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200">
          <ArrowLeft className="h-3.5 w-3.5" /> Back
        </button>
        <span className="text-sm font-medium text-slate-200">{baseline.name}</span>
        <span className="text-xs text-slate-500">vs. working tree</span>
      </div>
      <BaselineDetailView baseline={baseline} />
      {!compare.comparing ? (
        <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-4 text-center space-y-3">
          <p className="text-xs text-slate-400">Comparison runs one comprehensive suite and preserves Test Genie&apos;s complete dynamic phase result.</p>
          <Button size="sm" onClick={compare.start} className="gap-1.5"><GitCompare className="h-4 w-4" />Compare against working tree</Button>
        </div>
      ) : compare.isRunning ? (
        <div className="flex items-center gap-2 text-xs text-slate-400 py-8 justify-center"><Loader2 className="h-4 w-4 animate-spin" />Running comprehensive comparison…</div>
      ) : compare.error ? (
        <MutationErrorBanner error={compare.error} />
      ) : compare.diff ? (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2"><span className="text-xs text-slate-500">Overall</span><VerdictBadge verdict={compare.diff.verdict} /></div>
          {compare.diff.dirtyWarning && <div className="rounded-lg border border-amber-900/40 bg-amber-950/30 p-3 text-xs text-amber-300">{compare.diff.dirtyWarning}</div>}
          {evidence && (
            <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 text-xs text-slate-400">
              <p>Typed evidence: {evidence.baseCatalog?.artifacts.length ?? 0} baseline / {evidence.currentCatalog?.artifacts.length ?? 0} current artifacts</p>
              {evidence.degradedReasons.map((reason) => <p key={reason} className="text-amber-400">⚠ {reason}</p>)}
              {evidence.visualDeltas.filter((delta) => delta.status !== "identical").map((delta) => (
                <p key={`${delta.page}-${delta.status}`} className="text-blue-300">Visual advisory: {delta.page} {delta.status}</p>
              ))}
            </div>
          )}
          {compare.diff.phases.length === 0 ? <p className="text-xs text-slate-500">No phase comparison records were available.</p> : compare.diff.phases.map((phase) => <PhaseDiffCard key={phase.phase} diff={phase} />)}
          <p className="text-[11px] text-slate-600">{verdictMeta("regression").label}: your change broke something. {verdictMeta("preexisting").label}: already failing at capture time.</p>
        </div>
      ) : null}
    </div>
  );
}
