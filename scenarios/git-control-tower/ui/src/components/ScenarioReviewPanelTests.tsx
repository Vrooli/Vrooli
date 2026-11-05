import { useMemo, useState } from "react";
import { AlertTriangle, CheckCircle2, ChevronDown, ChevronRight, Loader2, Play, Search, XCircle } from "lucide-react";
import type { ArtifactRef, PhaseInfo, RunInfo, RunPhaseDescriptor } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import { Button } from "./ui/button";
import { useRun, useRuns, useStartRun } from "../lib/hooks-evidence";
import type { AgentContextItem } from "../lib/api";
import { AttachToAgentButton } from "./AgentTab";
import { runPhaseContextItem } from "../lib/agentContext";
import { MutationErrorBanner, ServiceUnavailableBanner, formatDuration, formatRelativeTime } from "./ScenarioReviewPanelShared";
import { SurfaceComparePanel } from "../features/baselines/SurfaceComparePanel";
import { SurfaceCaptureEmptyState } from "../features/baselines/SurfaceCaptureEmptyState";
import { useSurfaceBaselineModal } from "../features/baselines/useSurfaceBaselineModal";
import { ArtifactEvidenceRenderer } from "./ArtifactEvidenceRenderer";

const CLEAN_PAGE_SIZE = 20;

export function TestsTab({ scenarioSlug, repoId, testGenieAvailable, agentManagerAvailable, onAttachToAgent, onOpenBaselines, target }: {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  onOpenBaselines?: () => void;
  target?: { runId: string; phase: string } | null;
}) {
  const runsQuery = useRuns(scenarioSlug, { limit: 50 }, testGenieAvailable, repoId);
  const startRun = useStartRun(repoId);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(target?.runId ?? null);
  const [expandedPhase, setExpandedPhase] = useState<string | null>(target?.phase ?? null);
  const { openCaptureBaseline, baselineModal } = useSurfaceBaselineModal(scenarioSlug, repoId);

  if (!testGenieAvailable) return <ServiceUnavailableBanner name="Test Genie" message="Start the test-genie scenario to run automated tests" />;

  const runs = runsQuery.data?.runs ?? [];
  const selected = runs.find((run) => run.runId === selectedRunId) ?? runs[0];
  const runTests = () => startRun.mutate({ scenario: scenarioSlug });

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={startRun.error} onDismiss={() => startRun.reset()} />
      {baselineModal}
      {runsQuery.isLoading ? <LoadingState /> : runsQuery.error ? (
        <MutationErrorBanner error={runsQuery.error} />
      ) : !selected ? (
        <SurfaceCaptureEmptyState label="Tests" hasService onCaptureLoose={runTests} onCaptureBaseline={openCaptureBaseline} captureLabel="Run tests" isCapturing={startRun.isPending} />
      ) : (
        <RunContent
          key={selected.runId}
          run={selected}
          history={runs}
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          isStarting={startRun.isPending}
          runTests={runTests}
          onSelectRun={(runId) => { setSelectedRunId(runId); setExpandedPhase(null); }}
          openBaselines={onOpenBaselines ?? (() => {})}
          openCaptureBaseline={openCaptureBaseline}
          expandedPhase={expandedPhase}
          setExpandedPhase={setExpandedPhase}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={onAttachToAgent}
        />
      )}
    </div>
  );
}

function LoadingState() {
  return <div className="space-y-3" aria-label="Loading test runs"><div className="h-24 animate-pulse bg-slate-800 rounded" /><div className="h-16 animate-pulse bg-slate-800 rounded" /></div>;
}

function RunContent({ run, history, scenarioSlug, repoId, isStarting, runTests, onSelectRun, openBaselines, openCaptureBaseline, expandedPhase, setExpandedPhase, agentManagerAvailable, onAttachToAgent }: {
  run: RunInfo;
  history: RunInfo[];
  scenarioSlug: string;
  repoId?: string | null;
  isStarting: boolean;
  runTests: () => void;
  onSelectRun: (runId: string) => void;
  openBaselines: () => void;
  openCaptureBaseline: () => void;
  expandedPhase: string | null;
  setExpandedPhase: (phase: string | null) => void;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  const descriptors = useMemo(() => new Map(run.descriptorSnapshot?.phases.map((descriptor) => [descriptor.phase, descriptor]) ?? []), [run.descriptorSnapshot]);
  const [filters, setFilters] = useState({ search: "", status: "all", provider: "all", phaseClass: "all", dimension: "all", maturity: "all" });
  const [showClean, setShowClean] = useState(Boolean(expandedPhase));
  const [cleanLimit, setCleanLimit] = useState(CLEAN_PAGE_SIZE);
  const filterActive = Object.entries(filters).some(([key, value]) => key === "search" ? Boolean(value.trim()) : value !== "all");
  const filtered = useMemo(() => run.phases.filter((phase) => phaseMatches(phase, descriptors.get(phase.name), filters)), [descriptors, filters, run.phases]);
  const needsAttention = filtered.filter((phase) => !isCleanPhase(phase));
  const clean = filtered.filter(isCleanPhase);
  const visibleClean = (showClean || filterActive) ? clean.slice(0, cleanLimit) : [];
  const passed = run.phases.filter((phase) => phase.status === "passed").length;
  const failed = run.phases.filter((phase) => phase.status === "failed").length;
  const duration = run.phases.reduce((total, phase) => total + phase.durationSeconds, 0);
  const success = run.status === "passed";

  return <>
    <SurfaceComparePanel scenario={scenarioSlug} contextLabel="Tests" repoId={repoId} onOpenBaselines={openBaselines} onCaptureBaseline={openCaptureBaseline} viewingLabel={run.completedAt ? `run ${run.runId} · ${formatRelativeTime(run.completedAt)}` : `run ${run.runId}`} />
    <div className="flex items-center justify-between gap-3">
      <h3 className="text-xs font-medium text-slate-400">Test Runs</h3>
      <Button variant="outline" size="sm" onClick={runTests} disabled={isStarting} className="h-7 text-xs gap-1">
        {isStarting ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}Run Tests
      </Button>
    </div>
    {isStarting && <div role="status" className="flex items-center gap-2 px-3 py-2 bg-blue-950/50 border border-blue-900/50 rounded-lg text-blue-300 text-xs"><Loader2 className="h-3.5 w-3.5 animate-spin" />Run accepted by Test Genie…</div>}
    <RunSummary run={run} success={success} passed={passed} failed={failed} duration={duration} />
    <RunFilters phases={run.phases} descriptors={descriptors} filters={filters} onChange={(next) => { setFilters(next); setCleanLimit(CLEAN_PAGE_SIZE); }} />
    {filtered.length === 0 ? (
      <div className="rounded-lg border border-dashed border-slate-800 px-4 py-8 text-center text-xs text-slate-500">{run.phases.length === 0 ? "No phase records were captured for this run." : "No phases match these filters."}</div>
    ) : <div className="space-y-3">
      <PhaseGroup title="Needs attention" count={needsAttention.length} tone="attention">
        {needsAttention.map((phase) => <PhaseRow key={phase.name} run={run} phase={phase} descriptor={descriptors.get(phase.name)} scenario={scenarioSlug} repoId={repoId} expanded={expandedPhase === phase.name} onToggle={() => setExpandedPhase(expandedPhase === phase.name ? null : phase.name)} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} />)}
      </PhaseGroup>
      <section className="space-y-2" aria-labelledby="clean-phases-heading">
        <button id="clean-phases-heading" type="button" className="flex w-full items-center justify-between rounded border border-slate-800/70 bg-slate-900/30 px-3 py-2 text-left text-xs text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500" onClick={() => setShowClean((value) => !value)} aria-expanded={showClean || filterActive}>
          <span className="flex items-center gap-2"><CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />Clean phases</span><span>{clean.length}</span>
        </button>
        {(showClean || filterActive) && visibleClean.map((phase) => <PhaseRow key={phase.name} run={run} phase={phase} descriptor={descriptors.get(phase.name)} scenario={scenarioSlug} repoId={repoId} expanded={expandedPhase === phase.name} onToggle={() => setExpandedPhase(expandedPhase === phase.name ? null : phase.name)} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} />)}
        {(showClean || filterActive) && cleanLimit < clean.length && <Button variant="outline" size="sm" onClick={() => setCleanLimit((limit) => limit + CLEAN_PAGE_SIZE)}>Show {Math.min(CLEAN_PAGE_SIZE, clean.length - cleanLimit)} more clean phases</Button>}
      </section>
    </div>}
    <RunHistory runs={history} selectedRunId={run.runId} onSelect={onSelectRun} />
  </>;
}

function RunSummary({ run, success, passed, failed, duration }: { run: RunInfo; success: boolean; passed: number; failed: number; duration: number }) {
  const inProgress = run.status === "in_progress";
  return <section aria-label="Selected run summary" className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
    <div className="flex items-center justify-between gap-3"><div className="flex items-center gap-2">{success ? <CheckCircle2 className="h-4 w-4 text-emerald-400" /> : inProgress ? <Loader2 className="h-4 w-4 animate-spin text-blue-400" /> : <XCircle className="h-4 w-4 text-red-400" />}<span className={`text-sm font-medium ${success ? "text-emerald-300" : inProgress ? "text-blue-300" : "text-red-300"}`}>{success ? "All Passed" : inProgress ? "In Progress" : run.status === "aborted" ? "Aborted" : "Failed"}</span></div><span className="text-[11px] text-slate-500">{run.completedAt ? new Date(run.completedAt).toLocaleString() : run.startedAt}</span></div>
    <div className="flex flex-wrap gap-4 text-xs text-slate-400"><span>{run.phases.length} total</span><span className="text-emerald-400">{passed} passed</span>{failed > 0 && <span className="text-red-400">{failed} failed</span>}<span>{formatDuration(duration)}</span></div>
    <div className="flex flex-wrap gap-2 text-[11px] text-slate-500"><span className="font-mono">{run.runId}</span>{run.gitSha && <span className="font-mono">sha={run.gitSha.slice(0, 8)}</span>}{run.treeDigest && <span className="font-mono">tree={run.treeDigest.slice(0, 12)}</span>}{run.preset && <span>Preset: {run.preset}</span>}{run.descriptorSnapshot?.digest ? <span className="font-mono">catalog={run.descriptorSnapshot.digest.slice(0, 16)}</span> : <span className="text-amber-400">legacy catalog metadata unavailable</span>}{run.gitDirty && <span className="text-amber-400">dirty tree</span>}</div>
  </section>;
}

type Filters = { search: string; status: string; provider: string; phaseClass: string; dimension: string; maturity: string };

function RunFilters({ phases, descriptors, filters, onChange }: { phases: PhaseInfo[]; descriptors: Map<string, RunPhaseDescriptor>; filters: Filters; onChange: (filters: Filters) => void }) {
  const values = (pick: (phase: PhaseInfo, descriptor?: RunPhaseDescriptor) => string[]) => Array.from(new Set(phases.flatMap((phase) => pick(phase, descriptors.get(phase.name))).filter(Boolean))).sort();
  const select = (label: string, key: keyof Filters, options: string[]) => <label className="space-y-1 text-[10px] uppercase tracking-wide text-slate-500"><span>{label}</span><select aria-label={label} value={filters[key]} onChange={(event) => onChange({ ...filters, [key]: event.target.value })} className="block h-8 w-full rounded border border-slate-700 bg-slate-950 px-2 text-xs normal-case tracking-normal text-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"><option value="all">All</option>{options.map((option) => <option key={option} value={option}>{option || "Unknown"}</option>)}</select></label>;
  return <section aria-label="Phase filters" className="grid gap-2 rounded-lg border border-slate-800 bg-slate-900/30 p-3 sm:grid-cols-2 xl:grid-cols-6">
    <label className="relative sm:col-span-2 xl:col-span-1"><span className="sr-only">Search phases</span><Search className="pointer-events-none absolute left-2 top-2.5 h-3.5 w-3.5 text-slate-500" /><input aria-label="Search phases" value={filters.search} onChange={(event) => onChange({ ...filters, search: event.target.value })} placeholder="Search phases" className="h-8 w-full rounded border border-slate-700 bg-slate-950 pl-7 pr-2 text-xs text-slate-300 placeholder:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500" /></label>
    {select("Status", "status", values((phase) => [phase.status]))}
    {select("Provider", "provider", values((_phase, descriptor) => [descriptor?.provider || "legacy/unknown"]))}
    {select("Class", "phaseClass", values((_phase, descriptor) => [descriptor?.phaseClass || "unknown"]))}
    {select("Dimension", "dimension", values((_phase, descriptor) => descriptor?.dimensions ?? []))}
    {select("Maturity", "maturity", values((phase) => [phase.maturityStanding?.currentLevel || "none"]))}
  </section>;
}

function phaseMatches(phase: PhaseInfo, descriptor: RunPhaseDescriptor | undefined, filters: Filters) {
  const query = filters.search.trim().toLowerCase();
  const searchable = [phase.name, phase.status, descriptor?.displayName, descriptor?.description, descriptor?.provider, descriptor?.phaseClass, ...(descriptor?.dimensions ?? [])].filter(Boolean).join(" ").toLowerCase();
  return (!query || searchable.includes(query)) && (filters.status === "all" || phase.status === filters.status) && (filters.provider === "all" || (descriptor?.provider || "legacy/unknown") === filters.provider) && (filters.phaseClass === "all" || (descriptor?.phaseClass || "unknown") === filters.phaseClass) && (filters.dimension === "all" || descriptor?.dimensions.includes(filters.dimension)) && (filters.maturity === "all" || (phase.maturityStanding?.currentLevel || "none") === filters.maturity);
}

function isCleanPhase(phase: PhaseInfo) { return phase.status === "passed" && (phase.findingsSummary?.total ?? 0) === 0; }

function PhaseGroup({ title, count, tone, children }: { title: string; count: number; tone: "attention"; children: React.ReactNode }) {
  if (count === 0) return null;
  return <section className="space-y-2" aria-labelledby={`${tone}-phases-heading`}><h4 id={`${tone}-phases-heading`} className="flex items-center gap-2 text-xs font-medium text-red-300"><AlertTriangle className="h-3.5 w-3.5" />{title}<span className="text-slate-500">{count}</span></h4>{children}</section>;
}

function PhaseRow({ run, phase, descriptor, scenario, repoId, expanded, onToggle, agentManagerAvailable, onAttachToAgent }: { run: RunInfo; phase: PhaseInfo; descriptor?: RunPhaseDescriptor; scenario: string; repoId?: string | null; expanded: boolean; onToggle: () => void; agentManagerAvailable?: boolean; onAttachToAgent?: (item: AgentContextItem) => void }) {
  const detail = useRun(scenario, run.runId, { limit: 100 }, expanded, repoId);
  const artifacts = (detail.data?.artifacts ?? []).filter((artifact) => !artifact.producingPhase || artifact.producingPhase === phase.name);
  const findings = phase.findingsSummary;
  const displayName = descriptor?.displayName || phase.name;
  const statusColor = phase.status === "passed" ? "bg-emerald-500" : phase.status === "skipped" ? "bg-slate-500" : phase.status === "in_progress" ? "bg-blue-500" : "bg-red-500";
  return <div className="flex items-start gap-1"><article className="min-w-0 flex-1 rounded border border-slate-800/50 bg-slate-900/30">
    <button type="button" onClick={onToggle} aria-expanded={expanded} className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-xs hover:bg-slate-800/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500"><span className="flex min-w-0 items-center gap-2">{expanded ? <ChevronDown className="h-3 w-3 shrink-0 text-slate-500" /> : <ChevronRight className="h-3 w-3 shrink-0 text-slate-500" />}<span className={`h-2 w-2 shrink-0 rounded-full ${statusColor}`} /><span className="truncate text-slate-200">{displayName}</span><span className="hidden text-[10px] text-slate-500 sm:inline">{descriptor?.provider || "legacy/unknown"}</span></span><span className="shrink-0 text-slate-500">{formatDuration(Math.round(phase.durationSeconds))}</span></button>
    {expanded && <div className="space-y-3 border-t border-slate-800/30 px-3 pb-3 pt-2 text-[11px] text-slate-400">
      <p className="font-mono text-slate-500">{phase.name}</p>
      {descriptor?.description && <p>{descriptor.description}</p>}
      <div className="flex flex-wrap gap-2">{descriptor?.phaseClass && <span>Class: {descriptor.phaseClass}</span>}{descriptor?.runtimeClass && <span>Runtime: {descriptor.runtimeClass}</span>}{descriptor?.dimensions.map((dimension) => <span key={dimension} className="rounded bg-slate-800 px-1.5 py-0.5">{dimension}</span>)}</div>
      {descriptor?.policy && <p>Policy: selection {descriptor.policy.selection || "unknown"} · gating {descriptor.policy.resultGating || "unknown"} · unavailable {descriptor.policy.unavailable || "unknown"}</p>}
      {descriptor?.applicability && descriptor.applicability.status !== "applies" && <p className="text-amber-300">Applicability: {descriptor.applicability.status || "unknown"}{descriptor.applicability.reasons.length > 0 ? ` — ${descriptor.applicability.reasons.join("; ")}` : ""}</p>}
      {findings && findings.total > 0 && <p className="text-amber-300"><AlertTriangle className="mr-1 inline h-3 w-3" />{findings.total} finding{findings.total !== 1 ? "s" : ""} ({findings.blockers} blockers, {findings.errors} errors, {findings.warnings} warnings)</p>}
      {phase.maturityStanding?.currentLevel && <p>Maturity: {phase.maturityStanding.currentLevelLabel || phase.maturityStanding.currentLevel}{phase.maturityStanding.nextMove ? ` · Next: ${phase.maturityStanding.nextMove}` : ""}</p>}
      {descriptor?.docsPath && <a href={descriptor.docsPath} className="text-blue-400 hover:text-blue-300">Provider documentation</a>}
      <PhaseArtifacts loading={detail.isLoading} error={detail.error} artifacts={artifacts} scenario={scenario} run={run} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} />
    </div>}
  </article>{agentManagerAvailable && onAttachToAgent && <div className="mt-2 shrink-0"><AttachToAgentButton onClick={() => onAttachToAgent(runPhaseContextItem(run, phase, descriptor, scenario))} /></div>}</div>;
}

function PhaseArtifacts({ loading, error, artifacts, scenario, run, agentManagerAvailable, onAttachToAgent }: { loading: boolean; error: Error | null; artifacts: ArtifactRef[]; scenario: string; run: RunInfo; agentManagerAvailable?: boolean; onAttachToAgent?: (item: AgentContextItem) => void }) {
  if (loading) return <p role="status" className="text-slate-500">Loading phase evidence…</p>;
  if (error) return <p className="text-amber-400">Phase evidence is temporarily unavailable.</p>;
  if (artifacts.length === 0) return <p className="text-slate-600">No typed artifacts were recorded for this phase.</p>;
  return <div className="space-y-2"><p className="font-medium text-slate-500">Evidence ({artifacts.length})</p><div className="grid gap-2 lg:grid-cols-2">{artifacts.map((artifact) => <ArtifactEvidenceRenderer key={artifact.id} scenario={scenario} run={run} artifact={artifact} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} compact />)}</div></div>;
}

function RunHistory({ runs, selectedRunId, onSelect }: { runs: RunInfo[]; selectedRunId: string; onSelect: (runId: string) => void }) {
  if (runs.length <= 1) return null;
  return <section aria-labelledby="run-history-heading" className="space-y-2"><h4 id="run-history-heading" className="text-xs font-medium text-slate-400">Run history</h4><div className="max-h-64 space-y-1 overflow-y-auto">{runs.map((run) => <button key={run.runId} type="button" aria-current={run.runId === selectedRunId ? "true" : undefined} onClick={() => onSelect(run.runId)} className={`flex w-full items-center justify-between gap-3 rounded border px-3 py-2 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 ${run.runId === selectedRunId ? "border-blue-700 bg-blue-950/20" : "border-slate-800/50 bg-slate-900/50 hover:bg-slate-800/50"}`}><span className="flex min-w-0 items-center gap-2"><span className={`h-2 w-2 shrink-0 rounded-full ${run.status === "passed" ? "bg-emerald-500" : run.status === "in_progress" ? "bg-blue-500" : "bg-red-500"}`} /><span className="truncate font-mono text-[11px] text-slate-400">{run.runId}</span></span><span className="shrink-0 text-[11px] text-slate-500">{run.completedAt ? new Date(run.completedAt).toLocaleString() : "in progress"} · {run.phases.length} phases</span></button>)}</div></section>;
}
