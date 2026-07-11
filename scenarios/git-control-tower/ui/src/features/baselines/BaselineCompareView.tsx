import { useState } from "react";
import { AlertTriangle, ArrowLeft, CheckCircle2, ChevronDown, ChevronRight, GitCompare, Loader2 } from "lucide-react";
import { Button } from "../../components/ui/button";
import { AttachToAgentButton } from "../../components/AgentTab";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useCompareOnDemand } from "../../lib/hooks-baselines";
import { baselineComparisonContextItem } from "../../lib/agentContext";
import { phaseDiffNeedsAttention, summarizePhaseDiffs, verdictMeta } from "./model";
import { PhaseDiffCard, VerdictBadge } from "./parts";
import { BaselineDetailView } from "./BaselineDetailView";
import type { AgentContextItem } from "../../lib/api";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

interface BaselineCompareViewProps {
  scenario: string;
  baseline: BaselineManifest;
  repoId?: string | null;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  onBack: () => void;
}

export function BaselineCompareView({ scenario, baseline, repoId, agentManagerAvailable, onAttachToAgent, onBack }: BaselineCompareViewProps) {
  const compare = useCompareOnDemand(scenario, { baselineName: baseline.name, branch: baseline.branch, repoId });
  const [showClean, setShowClean] = useState(false);
  const evidence = compare.diff?.evidence;
  const phases = compare.diff?.phases ?? [];
  const summary = summarizePhaseDiffs(phases);
  const attention = phases.filter(phaseDiffNeedsAttention);
  const clean = phases.filter((phase) => !phaseDiffNeedsAttention(phase));
  const advisory = evidence?.visualDeltas.filter((delta) => delta.status !== "identical").length ?? 0;
  const catalogChanged = summary.catalogAdded > 0 || summary.catalogRetired > 0;

  return <div className="space-y-4">
    <div className="flex items-center gap-2"><button type="button" onClick={onBack} className="inline-flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"><ArrowLeft className="h-3.5 w-3.5" /> Back</button><span className="text-sm font-medium text-slate-200">{baseline.name}</span><span className="text-xs text-slate-500">vs. working tree</span></div>
    <BaselineDetailView baseline={baseline} />
    {!compare.comparing ? (
      <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-4 text-center space-y-3"><p className="text-xs text-slate-400">Comparison runs one comprehensive suite and preserves Test Genie&apos;s complete dynamic phase result.</p><Button size="sm" onClick={compare.start} className="gap-1.5"><GitCompare className="h-4 w-4" />Compare against working tree</Button></div>
    ) : compare.isRunning ? (
      <div role="status" className="flex items-center gap-2 text-xs text-slate-400 py-8 justify-center"><Loader2 className="h-4 w-4 animate-spin" />Running comprehensive comparison…</div>
    ) : compare.error ? (
      <MutationErrorBanner error={compare.error} />
    ) : compare.diff ? (
      <div className="space-y-3">
        <section aria-label="Comparison summary" className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/40 p-3">
          <div className="flex flex-wrap items-center justify-between gap-2"><div className="flex items-center gap-2"><span className="text-xs text-slate-500">Overall</span><VerdictBadge verdict={compare.diff.verdict} /></div>{agentManagerAvailable && onAttachToAgent && <AttachToAgentButton onClick={() => { if (compare.diff) onAttachToAgent(baselineComparisonContextItem(compare.diff)); }} />}</div>
          <dl className="grid gap-2 text-xs sm:grid-cols-2 xl:grid-cols-4">
            <Identity label="Base run" value={baseline.run?.runId || "unavailable"} />
            <Identity label="Current run" value={evidence?.currentRunId || "unavailable"} />
            <Identity label="Base SHA" value={baseline.git?.sha || "unavailable"} />
            <Identity label="Current SHA" value={compare.diff.currentGit?.sha || "unavailable"} />
          </dl>
          <div className="flex flex-wrap gap-2 text-[11px]"><Count label="Regressions" value={summary.regressions} tone="red" /><Count label="New failures" value={summary.newFailures} tone="amber" /><Count label="Preexisting" value={summary.preexisting} /><Count label="Cleared" value={summary.cleared} tone="green" /><Count label="Advisory" value={advisory} tone="blue" /><Count label="Not comparable" value={summary.notComparable} tone="amber" /></div>
          <div className="flex flex-wrap gap-3 text-[11px] text-slate-500">{compare.diff.staleness?.likelyStale && <span className="text-amber-400">Baseline likely stale · {compare.diff.staleness.commitsSince} commits / {compare.diff.staleness.filesChanged} files</span>}{compare.diff.currentGit?.dirty && <span className="text-amber-400">Current tree is dirty</span>}{catalogChanged ? <span className="text-blue-300">Catalog changed: {summary.catalogAdded} new / {summary.catalogRetired} retired</span> : <span>Catalog shape unchanged</span>}</div>
        </section>
        {compare.diff.dirtyWarning && <div className="rounded-lg border border-amber-900/40 bg-amber-950/30 p-3 text-xs text-amber-300">{compare.diff.dirtyWarning}</div>}
        {evidence && <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 text-xs text-slate-400"><p>Typed evidence: {evidence.baseCatalog?.artifacts.length ?? 0} baseline / {evidence.currentCatalog?.artifacts.length ?? 0} current artifacts</p>{evidence.degradedReasons.map((reason) => <p key={reason} className="text-amber-400">⚠ {reason}</p>)}{evidence.visualDeltas.filter((delta) => delta.status !== "identical").map((delta) => <p key={`${delta.page}-${delta.status}`} className="text-blue-300">Visual advisory: {delta.page} {delta.status}{delta.changedFraction > 0 ? ` · ${(delta.changedFraction * 100).toFixed(2)}% changed` : ""}</p>)}</div>}
        {phases.length === 0 ? <p className="rounded-lg border border-dashed border-slate-800 px-3 py-6 text-center text-xs text-slate-500">No phase comparison records were available.</p> : <>
          {attention.length > 0 && <section className="space-y-2" aria-labelledby="comparison-attention-heading"><h3 id="comparison-attention-heading" className="flex items-center gap-2 text-xs font-medium text-red-300"><AlertTriangle className="h-3.5 w-3.5" />Needs attention <span className="text-slate-500">{attention.length}</span></h3>{attention.map((phase) => <PhaseDiffCard key={phase.phase} diff={phase} />)}</section>}
          <section className="space-y-2"><button type="button" aria-expanded={showClean} onClick={() => setShowClean((value) => !value)} className="flex w-full items-center justify-between rounded border border-slate-800/70 bg-slate-900/30 px-3 py-2 text-xs text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"><span className="flex items-center gap-2">{showClean ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}<CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />Clean phases</span><span>{clean.length}</span></button>{showClean && clean.map((phase) => <PhaseDiffCard key={phase.phase} diff={phase} />)}</section>
        </>}
        <p className="text-[11px] text-slate-600">{verdictMeta("regression").label}: your change broke something. {verdictMeta("preexisting").label}: already failing at capture time.</p>
      </div>
    ) : null}
  </div>;
}

function Identity({ label, value }: { label: string; value: string }) {
  return <div><dt className="text-slate-500">{label}</dt><dd className="truncate font-mono text-slate-300" title={value}>{value}</dd></div>;
}

function Count({ label, value, tone = "slate" }: { label: string; value: number; tone?: "slate" | "red" | "amber" | "green" | "blue" }) {
  const tones = { slate: "border-slate-700 text-slate-300", red: "border-red-900/60 text-red-300", amber: "border-amber-900/60 text-amber-300", green: "border-emerald-900/60 text-emerald-300", blue: "border-blue-900/60 text-blue-300" };
  return <span className={`rounded border px-2 py-1 ${tones[tone]}`}>{label}: {value}</span>;
}
