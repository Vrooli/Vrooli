import { Bot, UserRound, Workflow } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useAgentSessionStore } from "../../stores";
import { sessionDetailPath } from "../../app/routes/route-paths";
import type { AgentSessionAttribution } from "../../types";
import { cn } from "../../lib/utils";

interface AttributionChipProps {
  attribution?: AgentSessionAttribution;
  labelPrefix?: string;
  className?: string;
}

export function AttributionChip({
  attribution,
  labelPrefix = "Created by",
  className,
}: AttributionChipProps) {
  const sessions = useAgentSessionStore((s) => s.sessions);
  const navigate = useNavigate();

  if (!attribution) return null;

  const sessionId = attribution.sessionId?.trim();
  const session = sessionId ? sessions.find((entry) => entry.id === sessionId) : undefined;
  const canOpenSession = sessionId != null && sessionId !== "";
  const label = attributionLabel(attribution, session?.title, labelPrefix);
  const Icon = sessionId ? Workflow : attribution.type === "agent" ? Bot : UserRound;

  const content = (
    <>
      <Icon className="h-3.5 w-3.5 shrink-0" />
      <span className="min-w-0 truncate">{label}</span>
    </>
  );

  const baseClassName = cn(
    "inline-flex h-6 max-w-[min(22rem,100%)] items-center gap-1.5 rounded-full border border-slate-700/80 bg-slate-900/80 px-2 text-xs font-medium text-slate-300",
    canOpenSession && "transition-colors hover:border-cyan-500/40 hover:bg-cyan-500/10 hover:text-cyan-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/40",
    className,
  );

  if (!canOpenSession) {
    return (
      <span className={baseClassName} title={label} data-testid="attribution-chip">
        {content}
      </span>
    );
  }

  return (
    <button
      type="button"
      className={baseClassName}
      title={label}
      onClick={() => {
        if (sessionId) navigate(sessionDetailPath(sessionId));
      }}
      data-testid="attribution-chip"
    >
      {content}
    </button>
  );
}

function attributionLabel(
  attribution: AgentSessionAttribution,
  sessionTitle: string | undefined,
  labelPrefix: string,
): string {
  if (attribution.sessionId) {
    return `${labelPrefix} ${sessionTitle || `session ${attribution.sessionId}`}`;
  }
  if (attribution.type === "agent") {
    const profile = attribution.profileKey || "agent";
    const run = attribution.runId || "unknown-run";
    return `${labelPrefix} agent:${profile}/${run}`;
  }
  return `${labelPrefix} operator`;
}
