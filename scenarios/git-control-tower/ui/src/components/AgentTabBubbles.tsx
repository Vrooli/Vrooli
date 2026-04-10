import { useState } from "react";
import {
  Loader2,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  Bot,
  Wrench,
  ExternalLink,
} from "lucide-react";
import { Button } from "./ui/button";
import type {
  AgentRun,
  AgentRunDiffFile,
  AgentRunSummary,
  AgentRunActions,
  AgentRunStatus,
} from "../lib/api";
import { useApproveAgentRun, useRejectAgentRun } from "../lib/hooks";

// ── Helpers ─────────────────────────────────────────────────────────

export function humanizeNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 10_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toLocaleString();
}

export function formatDuration(start?: string, end?: string): string | null {
  if (!start || !end) return null;
  const ms = new Date(end).getTime() - new Date(start).getTime();
  if (ms < 0) return null;
  const secs = Math.floor(ms / 1000);
  const mins = Math.floor(secs / 60);
  const rem = secs % 60;
  if (mins === 0) return `${rem}s`;
  return `${mins}m ${rem}s`;
}

// ── Status helpers ──────────────────────────────────────────────────

export const STATUS_COLORS: Record<AgentRunStatus, string> = {
  pending: "bg-slate-500",
  starting: "bg-blue-500",
  running: "bg-blue-500 animate-pulse",
  needs_review: "bg-amber-500",
  complete: "bg-emerald-500",
  failed: "bg-red-500",
  cancelled: "bg-slate-600",
};

export const STATUS_LABELS: Record<AgentRunStatus, string> = {
  pending: "Pending",
  starting: "Starting",
  running: "Running",
  needs_review: "Needs Review",
  complete: "Complete",
  failed: "Failed",
  cancelled: "Cancelled",
};

export function StatusBadge({ status }: { status: AgentRunStatus }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`h-2 w-2 rounded-full ${STATUS_COLORS[status]}`} />
      <span className="text-xs text-slate-300">{STATUS_LABELS[status]}</span>
    </span>
  );
}

// ── Bubble components ───────────────────────────────────────────────

export function UserBubble({ text, contextCount, timestamp }: { text: string; contextCount: number; timestamp: string }) {
  return (
    <div className="flex justify-end">
      <div className="max-w-[80%] rounded-lg px-3 py-2 bg-blue-900/40 border border-blue-800/50">
        <p className="text-sm text-slate-200 whitespace-pre-wrap">{text}</p>
        <div className="flex items-center justify-end gap-2 mt-1">
          {contextCount > 0 && (
            <span className="text-[10px] text-blue-400/70">{contextCount} context item{contextCount > 1 ? "s" : ""}</span>
          )}
          <span className="text-[10px] text-slate-500">{new Date(timestamp).toLocaleTimeString()}</span>
        </div>
      </div>
    </div>
  );
}

export function AgentBubble({ text, timestamp }: { text: string; timestamp: string }) {
  return (
    <div className="flex justify-start gap-2">
      <div className="h-6 w-6 rounded-full bg-slate-700 flex items-center justify-center shrink-0 mt-1">
        <Bot className="h-3.5 w-3.5 text-slate-400" />
      </div>
      <div className="max-w-[80%] rounded-lg px-3 py-2 bg-slate-800/60 border border-slate-700">
        <p className="text-sm text-slate-300 whitespace-pre-wrap">{text}</p>
        <span className="text-[10px] text-slate-500 block mt-1">{new Date(timestamp).toLocaleTimeString()}</span>
      </div>
    </div>
  );
}

export function ToolGroupBubble({ tools, timestamp }: { tools: { name: string; result?: string }[]; timestamp: string }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="flex justify-start gap-2">
      <div className="h-6 w-6 rounded-full bg-slate-700 flex items-center justify-center shrink-0 mt-1">
        <Wrench className="h-3.5 w-3.5 text-amber-400" />
      </div>
      <div className="max-w-[80%] rounded-lg bg-slate-800/40 border border-slate-700/50">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="w-full flex items-center gap-2 px-3 py-2 text-xs text-slate-400 hover:text-slate-300"
        >
          {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          Used {tools.length} tool{tools.length > 1 ? "s" : ""}
          <span className="text-[10px] text-slate-600 ml-auto">{new Date(timestamp).toLocaleTimeString()}</span>
        </button>
        {expanded && (
          <div className="px-3 pb-2 space-y-1.5 border-t border-slate-700/50 pt-2">
            {tools.map((tool, i) => (
              <div key={i} className="text-[11px]">
                <span className="text-amber-400/80 font-mono">{tool.name}</span>
                {tool.result && (
                  <pre className="mt-0.5 text-slate-500 overflow-hidden text-ellipsis whitespace-nowrap max-w-full">
                    {tool.result.slice(0, 200)}
                  </pre>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export function ErrorBubble({ text }: { text: string }) {
  return (
    <div className="flex justify-start gap-2">
      <div className="h-6 w-6 rounded-full bg-red-950/50 flex items-center justify-center shrink-0 mt-1">
        <AlertTriangle className="h-3.5 w-3.5 text-red-400" />
      </div>
      <div className="max-w-[80%] rounded-lg px-3 py-2 bg-red-950/30 border border-red-900/40">
        <p className="text-xs text-red-300">{text}</p>
      </div>
    </div>
  );
}

export function SummaryCard({ summary, run, sandboxReviewUrl }: { summary: AgentRunSummary; run?: AgentRun; sandboxReviewUrl?: string }) {
  const duration = run ? formatDuration(run.startedAt, run.endedAt) : null;
  const totalFiles = (summary.filesModified?.length ?? 0) + (summary.filesCreated?.length ?? 0) + (summary.filesDeleted?.length ?? 0);

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-3 space-y-2">
      <h4 className="text-xs font-medium text-slate-400">Summary</h4>
      {run?.promptPreview && (
        <p className="text-[11px] text-slate-500 truncate">{run.promptPreview}</p>
      )}
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-slate-400">
        {summary.tokensUsed != null && (
          <span>{humanizeNumber(summary.tokensUsed)} tokens</span>
        )}
        {summary.turnsUsed != null && (
          <span>{summary.turnsUsed} turns</span>
        )}
        {summary.costEstimate != null && summary.costEstimate > 0 && (
          <span>${summary.costEstimate.toFixed(2)}</span>
        )}
        {duration && <span>{duration}</span>}
      </div>
      {totalFiles > 0 && (
        <div className="flex gap-4 text-[11px] text-slate-400">
          {(summary.filesModified?.length ?? 0) > 0 && <span>Modified: {summary.filesModified?.length}</span>}
          {(summary.filesCreated?.length ?? 0) > 0 && <span>Created: {summary.filesCreated?.length}</span>}
          {(summary.filesDeleted?.length ?? 0) > 0 && <span>Deleted: {summary.filesDeleted?.length}</span>}
        </div>
      )}
      {totalFiles === 0 && (
        <div className="text-[11px] text-slate-500">No file changes</div>
      )}
      {sandboxReviewUrl && (
        <a
          href={sandboxReviewUrl}
          target="_blank"
          rel="noopener"
          className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded border transition-colors text-amber-400 border-amber-800 hover:bg-amber-950/50"
        >
          <ExternalLink className="h-3 w-3" />
          Review in Sandbox
        </a>
      )}
    </div>
  );
}

export function DiffSection({ files, isLoading }: { files: AgentRunDiffFile[]; isLoading: boolean }) {
  if (isLoading) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Loading diff...
        </div>
      </div>
    );
  }

  if (!files.length) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <p className="text-xs text-slate-500">No file changes</p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 divide-y divide-slate-800">
      {files.map((file) => (
        <div key={file.path} className="p-3">
          <div className="flex items-center justify-between mb-1">
            <code className="text-xs text-slate-300">{file.path}</code>
            <div className="flex items-center gap-2 text-[11px]">
              <span className={`${file.changeType === "added" ? "text-emerald-400" : file.changeType === "deleted" ? "text-red-400" : "text-blue-400"}`}>
                {file.changeType}
              </span>
              <span className="text-emerald-500">+{file.additions}</span>
              <span className="text-red-500">-{file.deletions}</span>
            </div>
          </div>
          {file.patch && (
            <pre className="mt-2 p-2 bg-slate-950 rounded text-[11px] text-slate-400 overflow-x-auto max-h-48">{file.patch}</pre>
          )}
        </div>
      ))}
    </div>
  );
}

export function ActionButtons({
  runId,
  actions,
  approve,
  reject,
}: {
  runId: string;
  actions: AgentRunActions;
  approve: ReturnType<typeof useApproveAgentRun>;
  reject: ReturnType<typeof useRejectAgentRun>;
}) {
  return (
    <div className="flex items-center gap-2 justify-center py-2">
      {actions.canApprove && (
        <>
          <Button
            variant="default"
            size="sm"
            onClick={() => approve.mutate({ runId, request: {} })}
            disabled={approve.isPending}
            className="h-8 text-xs gap-1 bg-emerald-700 hover:bg-emerald-600"
          >
            {approve.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <CheckCircle2 className="h-3 w-3" />
            )}
            Approve
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => reject.mutate({ runId, request: {} })}
            disabled={reject.isPending}
            className="h-8 text-xs gap-1 text-red-400 border-red-800 hover:bg-red-950/50"
          >
            {reject.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <XCircle className="h-3 w-3" />
            )}
            Reject
          </Button>
        </>
      )}
    </div>
  );
}
