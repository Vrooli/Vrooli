// Workflows tab — a workflow evidence-kind lens across arbitrary producer phases.

import { useMemo, useState } from "react";
import { AlertTriangle, CheckCircle2, ChevronDown, ChevronRight, Loader2, Minus, Play, XCircle } from "lucide-react";
import type { ArtifactRef, RunInfo } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import type { AgentContextItem } from "../lib/api";
import { Button } from "./ui/button";
import { MediaLightbox, MutationErrorBanner, formatDuration, formatRelativeTime, type LightboxItem } from "./ScenarioReviewPanelShared";
import { useEvidence, useStartRun } from "../lib/hooks-evidence";
import { runArtifactUrl } from "../lib/api-evidence";
import { ArtifactEvidenceRenderer, artifactRendererKind } from "./ArtifactEvidenceRenderer";
import { SurfaceComparePanel } from "../features/baselines/SurfaceComparePanel";
import { SurfaceCaptureEmptyState } from "../features/baselines/SurfaceCaptureEmptyState";
import { useSurfaceBaselineModal } from "../features/baselines/useSurfaceBaselineModal";

const WORKFLOW_KINDS = ["workflow.video", "video", "workflow.trace", "trace", "har", "network.har", "network", "console", "console.log", "log"];
const EVIDENCE_PAGE_SIZE = 40;

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

interface EvidenceRun {
  run: RunInfo;
  artifacts: ArtifactRef[];
}

export function WorkflowsTab({ scenarioSlug, repoId, testGenieAvailable, agentManagerAvailable, onAttachToAgent, onOpenBaselines, onOpenTests }: {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  onOpenBaselines: () => void;
  onOpenTests?: (runId: string, phase: string) => void;
}) {
  const [evidenceLimit, setEvidenceLimit] = useState(EVIDENCE_PAGE_SIZE);
  const evidenceQuery = useEvidence(scenarioSlug, { kinds: WORKFLOW_KINDS, limit: evidenceLimit, runLimit: 50 }, testGenieAvailable, repoId);
  const runs = useMemo<EvidenceRun[]>(() => groupEvidence(evidenceQuery.data?.items ?? []), [evidenceQuery.data?.items]);
  const triggerRun = useStartRun(repoId);
  const { openCaptureBaseline, baselineModal } = useSurfaceBaselineModal(scenarioSlug, repoId);
  const runWorkflows = () => triggerRun.mutate({ scenario: scenarioSlug, preset: "comprehensive" }, { onSuccess: () => void evidenceQuery.refetch() });

  if (!testGenieAvailable) {
    return <div className="flex flex-col items-center justify-center py-12 text-slate-500"><Play className="h-8 w-8 mb-3 opacity-50" /><p className="text-sm">test-genie is not available</p><p className="text-xs mt-1 text-slate-600">Start test-genie to see workflow evidence</p></div>;
  }

  return <div className="space-y-4">
    {baselineModal}
    <MutationErrorBanner error={triggerRun.error} onDismiss={() => triggerRun.reset()} />
    {evidenceQuery.isLoading ? <div className="space-y-2"><div className="h-12 animate-pulse rounded bg-slate-800/60" /><div className="h-12 animate-pulse rounded bg-slate-800/60" /></div>
      : evidenceQuery.error ? <MutationErrorBanner error={evidenceQuery.error} />
      : runs.length === 0 ? <SurfaceCaptureEmptyState label="Workflows" hasService={testGenieAvailable} onCaptureLoose={runWorkflows} onCaptureBaseline={openCaptureBaseline} isCapturing={triggerRun.isPending} />
      : <>
        <SurfaceComparePanel scenario={scenarioSlug} contextLabel="Workflows" repoId={repoId} onOpenBaselines={onOpenBaselines} onCaptureBaseline={openCaptureBaseline} viewingLabel={`${evidenceQuery.data?.total ?? 0} workflow artifact${evidenceQuery.data?.total === 1 ? "" : "s"} across ${runs.length} run${runs.length === 1 ? "" : "s"}`} />
        <div className="flex items-center justify-between gap-2"><p className="text-xs text-slate-500">Recordings, traces, logs, HAR, and network evidence are discovered by kind across every producing phase. Media loads only when previewed.</p><Button variant="outline" size="sm" onClick={runWorkflows} disabled={triggerRun.isPending} className="h-7 px-3 gap-1.5 shrink-0">{triggerRun.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}Capture workflows</Button></div>
        {evidenceQuery.data?.degradedReasons.map((reason) => <p key={reason} className="text-xs text-amber-400">Evidence unavailable: {reason}</p>)}
        <div className="space-y-2">{runs.map((entry) => <RunRow key={entry.run.runId} entry={entry} scenario={scenarioSlug} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} onOpenTests={onOpenTests} />)}</div>
        {evidenceQuery.data?.hasMore && <Button variant="outline" size="sm" onClick={() => setEvidenceLimit((limit) => limit + EVIDENCE_PAGE_SIZE)}>Load {Math.min(EVIDENCE_PAGE_SIZE, Math.max(0, evidenceQuery.data.total - evidenceLimit))} more workflow artifacts</Button>}
      </>}
  </div>;
}

function RunRow({ entry, scenario, agentManagerAvailable, onAttachToAgent, onOpenTests }: { entry: EvidenceRun; scenario: string; agentManagerAvailable?: boolean; onAttachToAgent?: (item: AgentContextItem) => void; onOpenTests?: (runId: string, phase: string) => void }) {
  const [expanded, setExpanded] = useState(false);
  const [previewArtifact, setPreviewArtifact] = useState<ArtifactRef | null>(null);
  const { run, artifacts } = entry;
  const Icon = STATUS_ICON[run.status] ?? AlertTriangle;
  const sha8 = run.gitSha ? run.gitSha.slice(0, 8) : "";
  const recordings = artifacts.filter((artifact) => artifactRendererKind(artifact.kind) === "video").length;
  const lightboxItems: LightboxItem[] = previewArtifact ? [{ label: previewArtifact.label || previewArtifact.id, sublabel: `${previewArtifact.producingPhase || "unknown producer"} · ${run.runId}`, type: artifactRendererKind(previewArtifact.kind) === "video" ? "video" : "image", url: runArtifactUrl(scenario, run.runId, previewArtifact.id) }] : [];

  return <div className="rounded-lg border border-slate-800 bg-slate-900/40">
    <button type="button" onClick={() => setExpanded((value) => !value)} className="w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-slate-800/30 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500" aria-expanded={expanded}>
      {expanded ? <ChevronDown className="h-3.5 w-3.5 text-slate-500 shrink-0" /> : <ChevronRight className="h-3.5 w-3.5 text-slate-500 shrink-0" />}
      <Icon className={`h-4 w-4 shrink-0 ${statusColor(run.status)}`} />
      <span className="text-xs text-slate-300 font-mono truncate">{run.runId}</span>
      <span className="text-[11px] text-slate-500">{recordings} recording{recordings === 1 ? "" : "s"} · {artifacts.length} artifact{artifacts.length === 1 ? "" : "s"}</span>
      <div className="ml-auto flex items-center gap-3 text-[11px] text-slate-500 shrink-0">{run.startedAt && run.completedAt && <span>{formatDuration(Math.max(0, Math.round((Date.parse(run.completedAt) - Date.parse(run.startedAt)) / 1000)))}</span>}{sha8 && <span className="font-mono">{sha8}</span>}{run.gitDirty && <span className="text-amber-500">dirty</span>}{run.startedAt && <span>{formatRelativeTime(run.startedAt)}</span>}</div>
    </button>
    {expanded && <div className="border-t border-slate-800/60 px-3 py-3"><div className="grid grid-cols-1 gap-3 lg:grid-cols-2">{artifacts.map((artifact) => <ArtifactEvidenceRenderer key={artifact.id} scenario={scenario} run={run} artifact={artifact} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} onPreview={setPreviewArtifact} onOpenTests={onOpenTests} />)}</div></div>}
    <MediaLightbox items={lightboxItems} initialIndex={0} isOpen={previewArtifact !== null} onClose={() => setPreviewArtifact(null)} />
  </div>;
}

function groupEvidence(items: readonly { run?: RunInfo; artifact?: ArtifactRef }[]): EvidenceRun[] {
  const grouped = new Map<string, EvidenceRun>();
  for (const item of items) {
    if (!item.run || !item.artifact) continue;
    const entry = grouped.get(item.run.runId) ?? { run: item.run, artifacts: [] };
    if (!entry.artifacts.some((artifact) => artifact.id === item.artifact?.id)) entry.artifacts.push(item.artifact);
    grouped.set(item.run.runId, entry);
  }
  return [...grouped.values()];
}
