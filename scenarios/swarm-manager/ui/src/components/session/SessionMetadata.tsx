import { memo } from "react";
import { ExternalLink } from "lucide-react";
import { formatRelativeTime } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import {
  useAgentProfileUrl,
  useAgentRunUrl,
  useAgentTaskUrl,
  useSkillUrl,
} from "../../services/external-links";
import type { AgentSession } from "../../types";

interface SessionMetadataProps {
  session: AgentSession;
  variant?: "panel" | "plain";
}

function SessionMetadataImpl({ session, variant = "panel" }: SessionMetadataProps) {
  const skillUrl = useSkillUrl(session.skillId);
  const runUrl = useAgentRunUrl(session.runId);
  const taskUrl = useAgentTaskUrl(session.taskId);
  const profileUrl = useAgentProfileUrl(session.profileKey);
  const contextCount = session.messages.reduce((count, message) => count + (message.context?.length ?? 0), 0);

  return (
    <section className={cn(variant === "panel" && "rounded-lg border border-white/10 bg-slate-950/30 p-3")} data-testid="agent-session-details">
      <h4 className="text-xs font-medium text-slate-300">Session details</h4>
      <dl className="mt-3 space-y-1 text-[11px] text-slate-400">
        <RunDetail label="Session ID" value={session.id} />
        <RunDetail label="Skill" value={session.skillId} href={skillUrl} testId="agent-session-skill-link" />
        <RunDetail label="Run" value={session.runId} href={runUrl} testId="agent-session-run-link" />
        <RunDetail label="Task" value={session.taskId} href={taskUrl} testId="agent-session-task-link" />
        <RunDetail label="Profile" value={session.profileKey} href={profileUrl} testId="agent-session-profile-link" />
        <RunDetail label="Context" value={contextCount > 0 ? String(contextCount) : undefined} />
        <RunDetail label="Images" value={(session.attachments?.length ?? 0) > 0 ? String(session.attachments?.length) : undefined} />
        <RunDetail label="Created" value={formatRelativeTime(session.createdAt)} />
      </dl>
    </section>
  );
}

function RunDetail({
  label,
  value,
  href,
  testId,
}: {
  label: string;
  value?: string;
  href?: string | null;
  testId?: string;
}) {
  if (!value) return null;
  const valueContent = href ? (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      aria-label={`Open ${label.toLowerCase()} in a new tab`}
      data-testid={testId}
      className="inline-flex min-w-0 items-center justify-end gap-1 rounded border border-cyan-500/20 bg-cyan-500/5 px-1.5 py-0.5 text-cyan-300 transition-colors hover:border-cyan-400/40 hover:bg-cyan-500/10 hover:text-cyan-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
    >
      <span className="min-w-0 truncate">{value}</span>
      <ExternalLink className="h-3 w-3 shrink-0" aria-hidden />
    </a>
  ) : (
    value
  );

  return (
    <div className="flex min-w-0 justify-between gap-2">
      <dt className="shrink-0 text-slate-500">{label}</dt>
      <dd className="min-w-0 truncate text-right text-slate-300">{valueContent}</dd>
    </div>
  );
}

/**
 * Memoized: Session details are static for the life of a session object, so they should re-render only when that object actually changes.
 * Its props are stabilised at the call site in SessionDetailsPage.
 */
export const SessionMetadata = memo(SessionMetadataImpl);
