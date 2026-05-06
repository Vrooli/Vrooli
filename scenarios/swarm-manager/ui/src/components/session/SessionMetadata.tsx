import { formatRelativeTime } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import type { AgentSession } from "../../types";

interface SessionMetadataProps {
  session: AgentSession;
  variant?: "panel" | "plain";
}

export function SessionMetadata({ session, variant = "panel" }: SessionMetadataProps) {
  return (
    <section className={cn(variant === "panel" && "rounded-lg border border-white/10 bg-slate-950/30 p-3")} data-testid="agent-session-details">
      <h4 className="text-xs font-medium text-slate-300">Session details</h4>
      <dl className="mt-3 space-y-1 text-[11px] text-slate-400">
        <RunDetail label="Session ID" value={session.id} />
        <RunDetail label="Skill" value={session.skillId} />
        <RunDetail label="Run" value={session.runId} />
        <RunDetail label="Task" value={session.taskId} />
        <RunDetail label="Profile" value={session.profileKey} />
        <RunDetail label="Created" value={formatRelativeTime(session.createdAt)} />
      </dl>
    </section>
  );
}

function RunDetail({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div className="flex min-w-0 justify-between gap-2">
      <dt className="shrink-0 text-slate-500">{label}</dt>
      <dd className="min-w-0 truncate text-right text-slate-300">{value}</dd>
    </div>
  );
}
