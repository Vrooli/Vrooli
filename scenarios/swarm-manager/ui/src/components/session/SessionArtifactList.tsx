import { ExternalLink, FileText, Layers3 } from "lucide-react";
import { formatDisplayText } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import type { AgentSessionArtifact } from "../../types";
import { nodeIdForSessionArtifact } from "./session-artifact-routing";

interface SessionArtifactListProps {
  artifacts: AgentSessionArtifact[];
  onOpenArtifact: (artifact: AgentSessionArtifact) => void;
  variant?: "panel" | "plain";
}

export function SessionArtifactList({ artifacts, onOpenArtifact, variant = "panel" }: SessionArtifactListProps) {
  return (
    <section className={cn(variant === "panel" && "rounded-lg border border-white/10 bg-slate-950/30 p-3")} data-testid="agent-session-artifacts">
      <div className="flex items-center gap-2">
        <Layers3 className="h-4 w-4 text-cyan-300" />
        <h4 className="text-xs font-medium text-slate-300">Artifacts</h4>
        <span className="ml-auto text-[11px] text-slate-500">{artifacts.length}</span>
      </div>
      <div className="mt-3 space-y-2">
        {artifacts.length > 0 ? (
          artifacts.map((artifact) => {
            const canOpen = nodeIdForSessionArtifact(artifact) !== null;
            return (
              <button
                key={artifact.id}
                type="button"
                onClick={() => onOpenArtifact(artifact)}
                className="w-full rounded-md border border-white/10 bg-slate-900/70 p-2 text-left transition-colors hover:border-slate-700 hover:bg-slate-800/70 disabled:pointer-events-none disabled:opacity-60"
                disabled={!canOpen}
                data-testid="agent-session-artifact"
              >
                <div className="flex items-start gap-2">
                  <FileText className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-400" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-medium text-slate-100">{artifact.title || artifact.entityRef}</p>
                    <p className="mt-0.5 text-[11px] text-slate-500">
                      {formatDisplayText(artifact.action)} {formatDisplayText(artifact.artifactType)}
                    </p>
                  </div>
                  {canOpen && <ExternalLink className="h-3.5 w-3.5 shrink-0 text-slate-500" />}
                </div>
              </button>
            );
          })
        ) : (
          <p className="py-8 text-center text-xs text-slate-500">No artifacts linked yet.</p>
        )}
      </div>
    </section>
  );
}
