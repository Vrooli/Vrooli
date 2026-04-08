import {
  RUN_STATUS,
  type AgentRun,
  type AgentRunEvent,
  type AgentRunStatus,
  type AgentRunDiffFile,
  type AgentRunSummary,
  type AgentRunActions,
} from "../lib/api";

// ── Chat message types ──────────────────────────────────────────────

export type ChatMessage =
  | { type: "user"; text: string; contextCount: number; timestamp: string }
  | { type: "agent-message"; text: string; timestamp: string }
  | { type: "tool-group"; tools: { name: string; result?: string }[]; timestamp: string }
  | { type: "status"; status: AgentRunStatus; phase?: string; timestamp: string }
  | { type: "error"; text: string; timestamp: string }
  | { type: "summary"; summary: AgentRunSummary; run?: AgentRun; timestamp: string }
  | { type: "diff"; files: AgentRunDiffFile[]; runId: string; actions: AgentRunActions }
  | { type: "action-prompt"; actions: AgentRunActions; runId: string };

// ── Build chat messages from events ─────────────────────────────────

export function buildChatMessages(
  events: AgentRunEvent[],
  activeRun: AgentRun | null,
  diffFiles: AgentRunDiffFile[] | null,
  runId: string | null,
  promptPreview?: string,
): ChatMessage[] {
  const messages: ChatMessage[] = [];

  // Walk events
  let i = 0;
  while (i < events.length) {
    const evt = events[i] as typeof events[number];
    const data = evt.data as Record<string, unknown> | undefined;

    if (evt.eventType === "message") {
      const text = (data?.content ?? "") as string;
      const role = (data?.role ?? "") as string;
      if (text) {
        if (role === "user") {
          messages.push({ type: "user", text, contextCount: 0, timestamp: evt.timestamp });
        } else {
          messages.push({ type: "agent-message", text, timestamp: evt.timestamp });
        }
      }
    } else if (evt.eventType === "tool_call") {
      // Group consecutive tool_call/tool_result pairs
      const tools: { name: string; result?: string }[] = [];
      while (i < events.length) {
        const e = events[i] as typeof events[number];
        if (e.eventType !== "tool_call" && e.eventType !== "tool_result") break;
        const d = e.data as Record<string, unknown> | undefined;
        if (e.eventType === "tool_call") {
          tools.push({ name: (d?.name ?? "tool") as string });
        } else if (e.eventType === "tool_result" && tools.length > 0) {
          const last = tools[tools.length - 1] as typeof tools[number];
          if (!last.result) {
            last.result = ((d?.result ?? d?.content ?? "") as string).slice(0, 500);
          }
        }
        i++;
      }
      if (tools.length > 0) {
        messages.push({ type: "tool-group", tools, timestamp: evt.timestamp });
      }
      continue; // already advanced i
    } else if (evt.eventType === "error") {
      const text = (data?.message ?? data?.error ?? "") as string;
      if (text) {
        messages.push({ type: "error", text, timestamp: evt.timestamp });
      }
    } else if (evt.eventType === "status_change") {
      const status = (data?.status ?? data?.newStatus) as AgentRunStatus | undefined;
      if (status) {
        messages.push({ type: "status", status, phase: data?.phase as string | undefined, timestamp: evt.timestamp });
      }
    }
    i++;
  }

  // Append error from run
  if (activeRun?.errorMsg && !messages.some((m) => m.type === "error" && m.text === activeRun.errorMsg)) {
    messages.push({ type: "error", text: activeRun.errorMsg, timestamp: new Date().toISOString() });
  }

  // If no user/agent messages were produced from events, fall back to
  // showing the run's promptPreview as a synthetic user message so the
  // chat never appears empty for a run that clearly had interaction.
  const preview = activeRun?.promptPreview || promptPreview;
  if (
    preview &&
    !messages.some((m) => m.type === "user" || m.type === "agent-message")
  ) {
    messages.unshift({
      type: "user",
      text: preview,
      contextCount: 0,
      timestamp: activeRun?.createdAt ?? new Date().toISOString(),
    });
  }

  // Append summary
  if (activeRun?.summary) {
    messages.push({ type: "summary", summary: activeRun.summary, run: activeRun, timestamp: new Date().toISOString() });
  }

  // Append diff + action-prompt when needs_review
  if (activeRun?.status === RUN_STATUS.NEEDS_REVIEW && activeRun.actions && runId) {
    if (diffFiles && diffFiles.length > 0) {
      messages.push({ type: "diff", files: diffFiles, runId, actions: activeRun.actions });
    }
    messages.push({ type: "action-prompt", actions: activeRun.actions, runId });
  }

  return messages;
}
