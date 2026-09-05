import {
  ArtifactAccessCapability,
  ArtifactProvenance,
  type ArtifactRef,
  type RunInfo,
} from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import { BarChart3, Braces, File, FileText, Image, Network, Play, Route, ShieldAlert } from "lucide-react";
import type { AgentContextItem } from "../lib/api";
import { runArtifactUrl } from "../lib/api-evidence";
import { runArtifactContextItem } from "../lib/agentContext";
import { AttachToAgentButton } from "./AgentTab";

export type ArtifactRendererKind = "image" | "visual-diff" | "video" | "text" | "report" | "coverage" | "trace" | "generic";

const ARTIFACT_RENDERERS: Readonly<Record<string, ArtifactRendererKind>> = {
  screenshot: "image",
  image: "image",
  "visual.diff": "visual-diff",
  "workflow.video": "video",
  video: "video",
  "command.output": "text",
  text: "text",
  log: "text",
  console: "text",
  "console.log": "text",
  json: "report",
  report: "report",
  "findings.report": "report",
  "coverage.report": "coverage",
  coverage: "coverage",
  trace: "trace",
  "workflow.trace": "trace",
  har: "trace",
  "network.har": "trace",
  network: "trace",
};

export function artifactRendererKind(kind: string): ArtifactRendererKind {
  return ARTIFACT_RENDERERS[kind] ?? "generic";
}

export function isPreviewableArtifact(artifact: ArtifactRef): boolean {
  const renderer = artifactRendererKind(artifact.kind);
  return renderer === "image" || renderer === "visual-diff" || renderer === "video";
}

interface ArtifactEvidenceRendererProps {
  scenario: string;
  run: RunInfo;
  artifact: ArtifactRef;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  onPreview?: (artifact: ArtifactRef) => void;
  onOpenTests?: (runId: string, phase: string) => void;
  compact?: boolean;
}

export function ArtifactEvidenceRenderer({
  scenario,
  run,
  artifact,
  agentManagerAvailable,
  onAttachToAgent,
  onPreview,
  onOpenTests,
  compact = false,
}: ArtifactEvidenceRendererProps) {
  const renderer = artifactRendererKind(artifact.kind);
  const presentation = RENDERER_PRESENTATION[renderer];
  const Icon = presentation.icon;
  const canOpen = artifact.accessCapability === undefined || artifact.accessCapability === ArtifactAccessCapability.STREAM || Boolean(artifact.accessPath);
  const artifactMetadata = artifact.metadata ?? {};
  const metadata = Object.entries(artifactMetadata).sort(([left], [right]) => left.localeCompare(right));
  const comparisonMetadata = Object.entries(artifact.comparison?.metadata ?? {}).sort(([left], [right]) => left.localeCompare(right));
  const changedFraction = artifactMetadata.changed_fraction ?? artifact.comparison?.metadata.changed_fraction;
  const provenance = artifact.provenance === ArtifactProvenance.LEGACY_DISCOVERY ? "Legacy discovery" : artifact.provenance === ArtifactProvenance.CATALOG ? "Catalog" : "Unspecified";

  return (
    <article className="rounded border border-slate-800 bg-slate-950/50 p-3 space-y-2" data-renderer={renderer}>
      <div className="flex items-start gap-2">
        <Icon className="mt-0.5 h-4 w-4 shrink-0 text-slate-400" />
        <div className="min-w-0 flex-1">
          <p className="truncate text-xs text-slate-200">{artifact.label || artifact.id}</p>
          <p className="truncate text-[10px] text-slate-500">{presentation.label} · {artifact.kind}</p>
        </div>
        {agentManagerAvailable && onAttachToAgent && <AttachToAgentButton onClick={() => onAttachToAgent(runArtifactContextItem(run, artifact))} />}
      </div>

      <div className="flex flex-wrap gap-x-3 gap-y-1 text-[10px] text-slate-500">
        <span>Run <span className="font-mono">{run.runId}</span></span>
        <span>Phase <span className="font-mono">{artifact.producingPhase || "unknown"}</span></span>
        {artifact.mediaType && <span>{artifact.mediaType}</span>}
        {artifact.sizeBytes > 0n && <span>{formatBytes(artifact.sizeBytes)}</span>}
        <span>{provenance}</span>
      </div>

      {renderer === "visual-diff" && (
        <div className="rounded border border-blue-900/50 bg-blue-950/20 px-2 py-1.5 text-[10px] text-blue-200">
          Advisory visual comparison{changedFraction ? ` · ${formatChangedFraction(changedFraction)} changed` : ""}. Visual changes do not alter the test verdict.
        </div>
      )}

      {!compact && (metadata.length > 0 || comparisonMetadata.length > 0) && (
        <dl className="grid gap-x-3 gap-y-1 text-[10px] sm:grid-cols-2">
          {[...metadata, ...comparisonMetadata.map(([key, value]) => [`comparison.${key}`, value] as const)].map(([key, value]) => (
            <div key={key} className="flex min-w-0 gap-1"><dt className="shrink-0 text-slate-600">{key}:</dt><dd className="truncate text-slate-400" title={value}>{value}</dd></div>
          ))}
        </dl>
      )}

      {!compact && (artifact.relationships?.length ?? 0) > 0 && (
        <div className="space-y-1 text-[10px] text-slate-500">
          <p className="text-slate-600">Relationships</p>
          {(artifact.relationships ?? []).map((relationship, index) => (
            <p key={`${relationship.type}-${relationship.targetArtifactId}-${index}`}><span>{relationship.type || "related"}</span> → <span className="font-mono">{relationship.targetArtifactId}</span></p>
          ))}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2 pt-1">
        {isPreviewableArtifact(artifact) && canOpen && onPreview && (
          <button type="button" onClick={() => onPreview(artifact)} className="inline-flex items-center gap-1 rounded border border-blue-800 px-2 py-1 text-[11px] text-blue-300 hover:bg-blue-950/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500">
            {renderer === "video" ? <Play className="h-3 w-3" /> : <Image className="h-3 w-3" />}Preview
          </button>
        )}
        {canOpen && (
          <a href={runArtifactUrl(scenario, run.runId, artifact.id)} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 rounded border border-slate-700 px-2 py-1 text-[11px] text-slate-300 hover:bg-slate-800/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500">
            <File className="h-3 w-3" />Open evidence
          </a>
        )}
        {!canOpen && <span className="inline-flex items-center gap-1 text-[10px] text-amber-400"><ShieldAlert className="h-3 w-3" />Artifact bytes unavailable</span>}
        {artifact.producingPhase && onOpenTests && (
          <button type="button" onClick={() => onOpenTests(run.runId, artifact.producingPhase)} className="inline-flex items-center gap-1 rounded px-2 py-1 text-[11px] text-slate-400 hover:bg-slate-800/60 hover:text-slate-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500">
            <Route className="h-3 w-3" />Open exact test phase
          </button>
        )}
      </div>
    </article>
  );
}

const RENDERER_PRESENTATION: Record<ArtifactRendererKind, { label: string; icon: typeof File }> = {
  image: { label: "Image", icon: Image },
  "visual-diff": { label: "Visual comparison", icon: Image },
  video: { label: "Recording", icon: Play },
  text: { label: "Text or log", icon: FileText },
  report: { label: "JSON or report", icon: Braces },
  coverage: { label: "Coverage report", icon: BarChart3 },
  trace: { label: "Trace, HAR, or network evidence", icon: Network },
  generic: { label: "Artifact", icon: File },
};

function formatBytes(bytes: bigint): string {
  const value = Number(bytes);
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MiB`;
}

function formatChangedFraction(value: string): string {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return value;
  return `${(parsed * 100).toFixed(parsed < 0.01 ? 2 : 1)}%`;
}
