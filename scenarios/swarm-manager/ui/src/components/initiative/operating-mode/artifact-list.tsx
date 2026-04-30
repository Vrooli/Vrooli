import { selectors } from "../../../consts/selectors";
import type { OperatingModeArtifactSnapshot } from "../../../types/operating-mode";

export function ArtifactList({ artifacts }: { artifacts: OperatingModeArtifactSnapshot[] }) {
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-slate-100">Artifacts</h3>
      {artifacts.length === 0 ? (
        <p className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-500">No declared artifacts for this mode.</p>
      ) : (
        artifacts.map((artifact) => (
          <div
            key={artifact.path}
            className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-3"
            data-testid={selectors.initiativeDetails.artifactCard}
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="break-all text-sm font-medium text-slate-100">{artifact.path}</p>
              <span className="text-[11px] text-slate-500">{artifact.sizeBytes ? `${artifact.sizeBytes} bytes` : artifact.required ? "required" : "optional"}</span>
            </div>
            {artifact.content ? (
              <pre className="mt-2 max-h-56 overflow-auto whitespace-pre-wrap rounded-lg bg-slate-950/70 p-3 text-xs leading-relaxed text-slate-300">
                {artifact.content}
              </pre>
            ) : (
              <p className="mt-2 text-sm text-slate-500">Not created yet.</p>
            )}
          </div>
        ))
      )}
    </div>
  );
}
