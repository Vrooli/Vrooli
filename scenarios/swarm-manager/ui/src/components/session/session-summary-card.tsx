/**
 * SessionSummaryCard — the compact agent-session card shared by the sidebar
 * SessionsTab and the SessionContextPicker.
 *
 * - Sidebar mode (no `selection`): a button that opens the session, with an
 *   optional bulk-selection checkbox. Behavior is identical to the previous
 *   inlined markup.
 * - Pick mode (`selection.selectionMode`): renders inside PickModeRow — a
 *   single toggle with checkbox + selected ring, no navigation.
 */
import { memo } from "react";
import { Archive, Gauge, GitPullRequestArrow, Layers3, MessageSquareMore, Workflow } from "lucide-react";
import { cn } from "../../lib/utils";
import { formatRelativeTime } from "../../lib/format-utils";
import { isActiveAgentSession } from "../../stores";
import type { AgentSession } from "../../types";
import { PickModeRow } from "./context/selectable-card";
import type { CardSelection } from "./context/selectable";

const STATUS_COLORS: Record<AgentSession["status"], string> = {
  draft: "bg-slate-700/60 text-slate-300",
  starting: "bg-blue-500/20 text-blue-300",
  running: "bg-cyan-500/20 text-cyan-300",
  waiting_for_user: "bg-amber-500/20 text-amber-300",
  proposal_ready: "bg-violet-500/20 text-violet-300",
  applying: "bg-blue-500/20 text-blue-300",
  complete: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
  canceled: "bg-slate-700/40 text-slate-500",
};

const KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Archived mode authoring",
  swarm_operations: "Swarm operations",
	workflow_authoring: "Workflow authoring",
};

const KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: Archive,
  swarm_operations: Gauge,
	workflow_authoring: Workflow,
};

export interface SessionSummaryCardProps {
  session: AgentSession;
  // Sidebar mode
  onOpen?: (id: string) => void;
  batchMode?: boolean;
  batchSelected?: boolean;
  onBatchToggle?: () => void;
  // Picker pick mode
  selection?: CardSelection;
}

function SessionCardBody({ session }: { session: AgentSession }) {
  const Icon = KIND_ICONS[session.kind];
  const active = isActiveAgentSession(session);
  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-start gap-2">
          <Icon className="mt-0.5 h-3.5 w-3.5 shrink-0 text-cyan-300" />
          <p className="line-clamp-2 min-w-0 text-[13px] font-medium leading-snug text-slate-100">
            {session.title}
          </p>
        </div>
        <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[session.status])}>
          {session.status.replace(/_/g, " ")}
        </span>
      </div>

      <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-slate-500">
        <span>{KIND_LABELS[session.kind]}</span>
        <span>{formatRelativeTime(session.updatedAt)}</span>
        {active && (
          <span className="inline-flex items-center gap-1 text-cyan-300">
            <span className="h-1.5 w-1.5 rounded-full bg-cyan-300" />
            active
          </span>
        )}
      </div>

      <div className="mt-1 flex flex-wrap items-center gap-1.5 text-[10px] text-slate-400">
        <span className="inline-flex items-center gap-1 rounded-full bg-slate-800/70 px-1.5 py-0.5">
          <MessageSquareMore className="h-3 w-3" />
          {session.messages.length}
        </span>
        <span className="inline-flex items-center gap-1 rounded-full bg-slate-800/70 px-1.5 py-0.5">
          <GitPullRequestArrow className="h-3 w-3" />
          {session.proposals.length}
        </span>
        <span className="inline-flex items-center gap-1 rounded-full bg-slate-800/70 px-1.5 py-0.5">
          <Layers3 className="h-3 w-3" />
          {session.artifacts.length}
        </span>
      </div>
    </>
  );
}

function SessionSummaryCardImpl({
  session,
  onOpen,
  batchMode = false,
  batchSelected = false,
  onBatchToggle,
  selection,
}: SessionSummaryCardProps) {
  if (selection?.selectionMode) {
    return (
      <PickModeRow selection={selection}>
        <SessionCardBody session={session} />
      </PickModeRow>
    );
  }

  return (
    <button
      type="button"
      onClick={() => onOpen?.(session.id)}
      className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
      data-testid="sidebar-session-item"
    >
      <div className="flex items-start gap-2">
        {batchMode && (
          <input
            type="checkbox"
            aria-label={`${batchSelected ? "Deselect" : "Select"} ${session.title}`}
            checked={batchSelected}
            onClick={(event) => event.stopPropagation()}
            onChange={(event) => {
              event.stopPropagation();
              onBatchToggle?.();
            }}
            className="mt-0.5"
          />
        )}
        <div className="min-w-0 flex-1">
          <SessionCardBody session={session} />
        </div>
      </div>
    </button>
  );
}

export const SessionSummaryCard = memo(SessionSummaryCardImpl);
