import { ExternalLink } from "lucide-react";
import { useRunArtifacts } from "../../hooks/useRunArtifacts";
import { buildOpaqueArtifactUrl } from "../../lib/api";

interface RunArtifactCatalogProps {
  scenarioName: string;
  runId?: string;
}

function artifactSize(size?: string | number): string | undefined {
  const bytes = Number(size);
  if (!Number.isFinite(bytes) || bytes < 0) return undefined;
  if (bytes < 1024) return `${bytes} B`;
  return `${(bytes / 1024).toFixed(1)} KiB`;
}

export function RunArtifactCatalog({ scenarioName, runId }: RunArtifactCatalogProps) {
  const catalog = useRunArtifacts(scenarioName, runId);
  if (!runId) return null;
  if (catalog.isLoading) return <section className="h-24 animate-pulse rounded-xl bg-white/5" aria-label="Loading run evidence" />;
  if (catalog.isError) return <section className="rounded-xl border border-amber-400/30 bg-amber-400/10 p-4 text-sm text-amber-100">Artifact catalog is unavailable for this run.</section>;
  const data = catalog.data;
  if (!data) return null;
  return <section className="rounded-xl border border-white/10 bg-black/20 p-4" aria-label="Run evidence catalog">
    <div className="flex flex-wrap items-baseline justify-between gap-2">
      <div><p className="text-xs uppercase tracking-wide text-slate-400">Run evidence</p><h3 className="mt-1 font-medium">Typed artifact catalog</h3></div>
      <span className="text-xs text-slate-400">{data.artifacts.length} artifact{data.artifacts.length === 1 ? "" : "s"}</span>
    </div>
    {(data.legacyDiscovered || (data.degradedReasons?.length ?? 0) > 0) && <div className="mt-3 rounded-lg border border-amber-400/30 bg-amber-400/10 p-3 text-sm text-amber-100"><strong>Evidence needs attention.</strong>{data.degradedReasons?.map((reason) => <p key={reason} className="mt-1">{reason}</p>)}</div>}
    {data.artifacts.length === 0 ? <p className="mt-3 text-sm text-slate-400">This run has no cataloged evidence artifacts.</p> : <ul className="mt-3 grid gap-2">
      {data.artifacts.map((artifact) => {
        const accessURL = artifact.accessPath ? buildOpaqueArtifactUrl(artifact.accessPath) : undefined;
        return <li key={artifact.id} className="rounded-lg border border-white/10 p-3 text-sm">
          <div className="flex flex-wrap items-start justify-between gap-2"><div><strong>{artifact.label || artifact.kind || "Unknown evidence"}</strong><p className="mt-1 text-xs text-slate-400">{artifact.kind || "unknown kind"}{artifact.producingPhase ? ` · ${artifact.producingPhase}` : ""}{artifact.mediaType ? ` · ${artifact.mediaType}` : ""}{artifactSize(artifact.sizeBytes) ? ` · ${artifactSize(artifact.sizeBytes)}` : ""}</p><p className="mt-1 break-all text-xs text-slate-500">Artifact ID: {artifact.id}</p></div>{accessURL && <a href={accessURL} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 rounded border border-white/20 px-2 py-1 text-xs text-cyan-200 hover:bg-white/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-300">Open <ExternalLink className="h-3 w-3" /></a>}</div>
          {(artifact.relationships?.length ?? 0) > 0 && <p className="mt-2 text-xs text-slate-400">{artifact.relationships?.length} catalog relationship{artifact.relationships?.length === 1 ? "" : "s"}</p>}
        </li>;
      })}
    </ul>}
  </section>;
}
