import { useMemo, useState } from "react";
import { AlertTriangle, Camera, Loader2 } from "lucide-react";
import type { ArtifactRef, RunInfo } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import type { AgentContextItem } from "../lib/api";
import { useEvidence, useStartRun } from "../lib/hooks-evidence";
import { runArtifactUrl } from "../lib/api-evidence";
import { ArtifactEvidenceRenderer, artifactRendererKind } from "./ArtifactEvidenceRenderer";
import { MediaLightbox, MutationErrorBanner, formatRelativeTime, type LightboxItem } from "./ScenarioReviewPanelShared";
import { SurfaceComparePanel } from "../features/baselines/SurfaceComparePanel";
import { SurfaceCaptureEmptyState } from "../features/baselines/SurfaceCaptureEmptyState";
import { useSurfaceBaselineModal } from "../features/baselines/useSurfaceBaselineModal";
import { Button } from "./ui/button";

const SCREENSHOT_KINDS = ["screenshot", "image", "visual.diff"];
const EVIDENCE_PAGE_SIZE = 40;

interface VisualRun {
  run: RunInfo;
  artifacts: ArtifactRef[];
}

export function ScreenshotsTab({ scenarioSlug, repoId, testGenieAvailable, agentManagerAvailable, onAttachToAgent, onOpenBaselines, onOpenTests }: {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  onOpenBaselines: () => void;
  onOpenTests?: (runId: string, phase: string) => void;
}) {
  const [evidenceLimit, setEvidenceLimit] = useState(EVIDENCE_PAGE_SIZE);
  const evidenceQuery = useEvidence(scenarioSlug, { kinds: SCREENSHOT_KINDS, limit: evidenceLimit, runLimit: 50 }, testGenieAvailable, repoId);
  const startRun = useStartRun(repoId);
  const { openCaptureBaseline, baselineModal } = useSurfaceBaselineModal(scenarioSlug, repoId);
  const runs = useMemo<VisualRun[]>(() => groupEvidence(evidenceQuery.data?.items ?? []), [evidenceQuery.data?.items]);
  const capture = () => startRun.mutate({ scenario: scenarioSlug, preset: "comprehensive", captureProfile: "baseline" });

  return <div className="space-y-4">
    {baselineModal}
    <MutationErrorBanner error={startRun.error} onDismiss={() => startRun.reset()} />
    {evidenceQuery.isLoading ? <div className="grid grid-cols-1 gap-3 sm:grid-cols-2"><div className="h-32 animate-pulse rounded bg-slate-800" /><div className="h-32 animate-pulse rounded bg-slate-800" /></div>
      : evidenceQuery.error ? <MutationErrorBanner error={evidenceQuery.error} />
      : runs.length === 0 ? <SurfaceCaptureEmptyState label="Visuals" hasService={testGenieAvailable} onCaptureLoose={capture} onCaptureBaseline={openCaptureBaseline} captureLabel="Capture screenshots" isCapturing={startRun.isPending} serviceMessage="Start test-genie to capture screenshot evidence" icon={<Camera className="h-8 w-8 mb-3 opacity-50" />} />
      : <>
        <SurfaceComparePanel scenario={scenarioSlug} contextLabel="Screenshots" repoId={repoId} onOpenBaselines={onOpenBaselines} onCaptureBaseline={openCaptureBaseline} viewingLabel={`${evidenceQuery.data?.total ?? 0} visual artifact${evidenceQuery.data?.total === 1 ? "" : "s"} across ${runs.length} run${runs.length === 1 ? "" : "s"}`} />
        <p className="text-xs text-slate-500">Visual evidence is grouped by run and capture context. Previews load only when opened; comparisons are advisory.</p>
        {startRun.isPending && <div role="status" className="flex items-center gap-2 text-xs text-blue-300"><Loader2 className="h-3.5 w-3.5 animate-spin" />Screenshot run accepted by Test Genie…</div>}
        {evidenceQuery.data?.degradedReasons.map((reason) => <p key={reason} className="text-xs text-amber-400"><AlertTriangle className="inline h-3 w-3 mr-1" />{reason}</p>)}
        {runs.map((entry) => <VisualRunGallery key={entry.run.runId} entry={entry} scenario={scenarioSlug} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} onOpenTests={onOpenTests} />)}
        {evidenceQuery.data?.hasMore && <Button variant="outline" size="sm" onClick={() => setEvidenceLimit((limit) => limit + EVIDENCE_PAGE_SIZE)}>Load {Math.min(EVIDENCE_PAGE_SIZE, Math.max(0, evidenceQuery.data.total - evidenceLimit))} more visual artifacts</Button>}
      </>}
  </div>;
}

function VisualRunGallery({ entry, scenario, agentManagerAvailable, onAttachToAgent, onOpenTests }: { entry: VisualRun; scenario: string; agentManagerAvailable?: boolean; onAttachToAgent?: (item: AgentContextItem) => void; onOpenTests?: (runId: string, phase: string) => void }) {
  const [previewArtifact, setPreviewArtifact] = useState<ArtifactRef | null>(null);
  const screenshots = entry.artifacts.filter((artifact) => artifactRendererKind(artifact.kind) === "image");
  const diffs = entry.artifacts.filter((artifact) => artifactRendererKind(artifact.kind) === "visual-diff");
  const captureContexts = new Set(entry.artifacts.map((artifact) => artifact.metadata.page || artifact.metadata.route || artifact.metadata.url).filter(Boolean));
  const lightboxItems = previewArtifact ? [toLightboxItem(entry.run, previewArtifact, scenario)] : [];

  return <section className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 space-y-3">
    <div className="flex flex-wrap items-center justify-between gap-2"><div><p className="text-xs font-mono text-slate-300">{entry.run.runId}</p><p className="text-[11px] text-slate-500">{entry.run.startedAt ? formatRelativeTime(entry.run.startedAt) : "Unknown time"} · {screenshots.length} capture{screenshots.length === 1 ? "" : "s"} · {diffs.length} comparison{diffs.length === 1 ? "" : "s"}{captureContexts.size > 0 ? ` · ${captureContexts.size} page context${captureContexts.size === 1 ? "" : "s"}` : ""}</p></div><span className="text-[11px] text-slate-500">{entry.run.status}</span></div>
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-2 xl:grid-cols-3">
      {entry.artifacts.map((artifact) => <ArtifactEvidenceRenderer key={artifact.id} scenario={scenario} run={entry.run} artifact={artifact} agentManagerAvailable={agentManagerAvailable} onAttachToAgent={onAttachToAgent} onPreview={setPreviewArtifact} onOpenTests={onOpenTests} />)}
    </div>
    <MediaLightbox items={lightboxItems} initialIndex={0} isOpen={previewArtifact !== null} onClose={() => setPreviewArtifact(null)} />
  </section>;
}

function groupEvidence(items: readonly { run?: RunInfo; artifact?: ArtifactRef }[]): VisualRun[] {
  const grouped = new Map<string, VisualRun>();
  for (const item of items) {
    if (!item.run || !item.artifact) continue;
    const entry = grouped.get(item.run.runId) ?? { run: item.run, artifacts: [] };
    if (!entry.artifacts.some((artifact) => artifact.id === item.artifact?.id)) entry.artifacts.push(item.artifact);
    grouped.set(item.run.runId, entry);
  }
  return [...grouped.values()];
}

function toLightboxItem(run: RunInfo, artifact: ArtifactRef, scenario: string): LightboxItem {
  return {
    label: artifact.label || artifact.id,
    sublabel: `${artifact.metadata.page || artifact.metadata.route || "capture context unavailable"} · ${artifact.producingPhase || "unknown producer"} · ${run.runId}`,
    type: artifactRendererKind(artifact.kind) === "video" ? "video" : "image",
    url: runArtifactUrl(scenario, run.runId, artifact.id),
  };
}
