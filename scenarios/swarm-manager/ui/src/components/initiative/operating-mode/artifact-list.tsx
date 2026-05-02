import { useState } from "react";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeArtifactSnapshot } from "../../../types/operating-mode";
import { ArtifactViewerDialog } from "./artifact-viewer-dialog";

export function ArtifactList({ artifacts }: { artifacts: OperatingModeArtifactSnapshot[] }) {
  const [selected, setSelected] = useState<OperatingModeArtifactSnapshot | null>(null);

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-slate-100">Artifacts</h3>
      {artifacts.length === 0 ? (
        <p className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-500">
          No declared artifacts for this mode.
        </p>
      ) : (
        artifacts.map((artifact) => {
          const hasContent = Boolean(artifact.content);
          return (
            <button
              key={artifact.path}
              type="button"
              onClick={() => setSelected(artifact)}
              disabled={!hasContent}
              className={`block w-full rounded-lg border border-slate-800/80 bg-slate-900/55 p-3 text-left transition-colors ${
                hasContent
                  ? "cursor-pointer hover:border-cyan-400/60 hover:bg-slate-800/80"
                  : "cursor-default opacity-80"
              }`}
              data-testid={selectors.initiativeDetails.artifactRow}
              data-artifact-path={artifact.path}
              title={hasContent ? `Open ${artifact.path}` : "Artifact not created yet"}
            >
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="break-all text-sm font-medium text-slate-100">{artifact.path}</p>
                <span className="text-[11px] text-slate-500">
                  {artifact.sizeBytes ? `${artifact.sizeBytes} bytes` : artifact.required ? "required" : "optional"}
                </span>
              </div>
              {artifact.content ? (
                <p className="mt-2 line-clamp-3 whitespace-pre-wrap text-xs leading-relaxed text-slate-400">
                  {artifact.content}
                </p>
              ) : (
                <p className="mt-2 text-sm text-slate-500">Not created yet.</p>
              )}
            </button>
          );
        })
      )}
      <ArtifactViewerDialog
        artifact={selected}
        isOpen={selected !== null}
        onClose={() => setSelected(null)}
      />
    </div>
  );
}
