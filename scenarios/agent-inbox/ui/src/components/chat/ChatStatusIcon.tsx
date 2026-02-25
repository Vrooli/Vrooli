/**
 * ChatStatusIcon — compact status indicator for the chat header.
 *
 * In LLM mode: a MessageSquare icon with a popover showing model info.
 * In Agent mode: a Bot icon surrounded by a circular progress ring
 * whose colour and fill reflect the current run status. The popover
 * shows detailed status, metrics, errors, and a stop button.
 */

import { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { Bot, MessageSquare, StopCircle, Zap } from "lucide-react";
import type { AgentRunStatus, AgentModeStatus, Model } from "../../lib/api";
import type { AgentMetric } from "./agent/AgentEventList";

interface ChatStatusIconProps {
  chatMode: "llm" | "agent";
  // LLM info
  model: string;
  models: Model[];
  // Agent info
  isAgentActive: boolean;
  agentStatus?: AgentModeStatus | null;
  agentMetrics?: AgentMetric[];
  agentError?: { message: string; recovery?: string } | null;
  onStopAgent?: () => void;
}

/* ── Circular‑progress constants ─────────────────────────────────── */
const RING_SIZE = 28;
const STROKE_WIDTH = 2.5;
const RADIUS = (RING_SIZE - STROKE_WIDTH) / 2;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

const RING_COLORS: Record<string, string> = {
  pending: "stroke-blue-400",
  starting: "stroke-blue-400",
  running: "stroke-blue-500",
  needs_review: "stroke-yellow-500",
  complete: "stroke-green-500",
  failed: "stroke-red-500",
  cancelled: "stroke-slate-500",
};

/* ── Main component ──────────────────────────────────────────────── */

export function ChatStatusIcon({
  chatMode,
  model,
  models,
  isAgentActive,
  agentStatus,
  agentMetrics,
  agentError,
  onStopAgent,
}: ChatStatusIconProps) {
  const [isOpen, setIsOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const [popoverPos, setPopoverPos] = useState<{ top: number; left: number } | null>(null);

  // Position popover below the trigger, clamped to viewport
  useEffect(() => {
    if (!isOpen || !triggerRef.current) {
      setPopoverPos(null);
      return;
    }
    const rect = triggerRef.current.getBoundingClientRect();
    const popoverWidth = chatMode === "agent" ? 288 : 256; // w-72 / w-64
    const margin = 8;
    let left = rect.left;
    if (left + popoverWidth > window.innerWidth - margin) {
      left = window.innerWidth - popoverWidth - margin;
    }
    if (left < margin) left = margin;
    setPopoverPos({ top: rect.bottom + margin, left });
  }, [isOpen, chatMode]);

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;
    function handleClick(e: MouseEvent) {
      const target = e.target as Node;
      if (triggerRef.current?.contains(target) || popoverRef.current?.contains(target)) return;
      setIsOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [isOpen]);

  // Close on Escape
  useEffect(() => {
    if (!isOpen) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") setIsOpen(false);
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [isOpen]);

  /* ── LLM mode ──────────────────────────────────────────────────── */

  if (chatMode === "llm") {
    const currentModel = models.find((m) => m.id === model);
    return (
      <div className="relative">
        <button
          ref={triggerRef}
          onClick={() => setIsOpen(!isOpen)}
          className="p-1 rounded-md text-slate-400 hover:text-white hover:bg-white/10 transition-colors"
          title="Chat info"
          data-testid="chat-status-icon"
        >
          <MessageSquare className="h-4 w-4" />
        </button>

        {isOpen && popoverPos && createPortal(
          <div
            ref={popoverRef}
            className="fixed z-[9999] w-64 bg-slate-900 border border-white/10 rounded-lg shadow-xl p-3 animate-in fade-in-0 zoom-in-95 duration-100"
            style={{ top: popoverPos.top, left: popoverPos.left }}
          >
            <div className="text-xs text-slate-500 mb-2">LLM Mode</div>
            <div className="flex items-center gap-2">
              <Bot className="h-4 w-4 text-indigo-400 flex-shrink-0" />
              <div className="min-w-0">
                <div className="text-sm font-medium text-white truncate">
                  {currentModel?.name || model}
                </div>
                {currentModel?.description && (
                  <div className="text-xs text-slate-400 mt-0.5 line-clamp-2">
                    {currentModel.description}
                  </div>
                )}
              </div>
            </div>
          </div>,
          document.body,
        )}
      </div>
    );
  }

  /* ── Agent mode ────────────────────────────────────────────────── */

  const status = agentStatus?.status;
  const isRunning = status === "running" || status === "starting" || status === "pending";
  const hasProgress = isRunning && agentStatus?.progress_percent != null && agentStatus.progress_percent > 0;
  const progress = agentStatus?.progress_percent ?? 0;
  const ringColor = status ? (RING_COLORS[status] || "stroke-slate-600") : "stroke-slate-600";

  // Dash-offset controls how much of the ring is visible.
  let dashOffset: number;
  if (!status || !isAgentActive) {
    dashOffset = CIRCUMFERENCE; // fully hidden — no active run
  } else if (isRunning) {
    dashOffset = hasProgress
      ? CIRCUMFERENCE - (progress / 100) * CIRCUMFERENCE
      : CIRCUMFERENCE * 0.75; // ~25% arc for indeterminate
  } else {
    dashOffset = 0; // full ring for terminal states
  }

  const hasError = !!agentError || (status === "failed" && !!agentStatus?.error_msg);

  const iconColor = isRunning
    ? "text-blue-400"
    : status === "complete"
      ? "text-green-400"
      : status === "failed" || status === "cancelled"
        ? "text-red-400"
        : status === "needs_review"
          ? "text-yellow-400"
          : "text-slate-400";

  return (
    <div className="relative">
      <button
        ref={triggerRef}
        onClick={() => setIsOpen(!isOpen)}
        className={`relative p-0.5 rounded-md hover:bg-white/10 transition-colors ${hasError ? "ring-1 ring-red-500/50" : ""}`}
        title={status ? `Agent: ${statusLabel(status)}` : "Agent mode"}
        data-testid="chat-status-icon"
      >
        <div className="relative" style={{ width: RING_SIZE, height: RING_SIZE }}>
          {/* SVG ring */}
          {status && isAgentActive && (
            <svg
              width={RING_SIZE}
              height={RING_SIZE}
              className={`absolute inset-0 ${isRunning && !hasProgress ? "animate-spin" : ""}`}
              style={isRunning && !hasProgress ? { animationDuration: "1.5s" } : undefined}
            >
              {/* Track */}
              <circle
                cx={RING_SIZE / 2}
                cy={RING_SIZE / 2}
                r={RADIUS}
                fill="none"
                strokeWidth={STROKE_WIDTH}
                className="stroke-white/10"
              />
              {/* Progress arc */}
              <circle
                cx={RING_SIZE / 2}
                cy={RING_SIZE / 2}
                r={RADIUS}
                fill="none"
                strokeWidth={STROKE_WIDTH}
                strokeLinecap="round"
                className={ringColor}
                style={{
                  strokeDasharray: CIRCUMFERENCE,
                  strokeDashoffset: dashOffset,
                  transform: "rotate(-90deg)",
                  transformOrigin: "center",
                  transition: hasProgress ? "stroke-dashoffset 0.3s ease" : undefined,
                }}
              />
            </svg>
          )}
          {/* Bot icon — centred inside the ring */}
          <Bot className={`absolute inset-0 m-auto h-4 w-4 ${iconColor}`} />
        </div>
      </button>

      {/* Popover — portalled to escape overflow clips */}
      {isOpen && popoverPos && createPortal(
        <div
          ref={popoverRef}
          className="fixed z-[9999] w-72 bg-slate-900 border border-white/10 rounded-lg shadow-xl animate-in fade-in-0 zoom-in-95 duration-100"
          style={{ top: popoverPos.top, left: popoverPos.left }}
        >
          <div className="p-3 space-y-3">
            {/* Header row */}
            <div className="flex items-center justify-between">
              <div className="text-xs text-slate-500">Agent Mode</div>
              {status && (
                <span className={`text-xs font-medium ${statusTextColor(status)}`}>
                  {statusLabel(status)}
                </span>
              )}
            </div>

            {/* Phase */}
            {agentStatus?.phase && isRunning && (
              <div className="text-xs text-slate-400">{agentStatus.phase}</div>
            )}

            {/* Progress bar */}
            {isRunning && hasProgress && (
              <div>
                <div className="h-1.5 bg-zinc-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 transition-all duration-300 rounded-full"
                    style={{ width: `${Math.min(progress, 100)}%` }}
                  />
                </div>
                <div className="text-xs text-slate-500 mt-1">{Math.round(progress)}%</div>
              </div>
            )}

            {/* Metrics */}
            {agentMetrics && agentMetrics.length > 0 && (
              <MetricsDisplay metrics={agentMetrics} />
            )}

            {/* Agent API error */}
            {agentError && (
              <div className="text-xs text-red-400 bg-red-500/10 rounded p-2">
                {agentError.message}
                {agentError.recovery && (
                  <span className="text-slate-400 block mt-1">{agentError.recovery}</span>
                )}
              </div>
            )}

            {/* Run error from status */}
            {status === "failed" && agentStatus?.error_msg && !agentError && (
              <div className="text-xs text-red-400 bg-red-500/10 rounded p-2 break-words">
                {agentStatus.error_msg}
              </div>
            )}

            {/* Stop button */}
            {isRunning && onStopAgent && (
              <button
                onClick={() => { onStopAgent(); setIsOpen(false); }}
                className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs text-red-400 hover:text-red-300 bg-red-500/10 hover:bg-red-500/20 transition-colors"
              >
                <StopCircle className="h-3.5 w-3.5" />
                Stop Agent
              </button>
            )}
          </div>
        </div>,
        document.body,
      )}
    </div>
  );
}

/* ── Helpers ──────────────────────────────────────────────────────── */

function statusTextColor(status: AgentRunStatus): string {
  switch (status) {
    case "running": case "starting": case "pending": return "text-blue-400";
    case "complete": return "text-green-400";
    case "failed": case "cancelled": return "text-red-400";
    case "needs_review": return "text-yellow-400";
    default: return "text-slate-400";
  }
}

function statusLabel(status: AgentRunStatus): string {
  switch (status) {
    case "pending": return "Pending";
    case "starting": return "Starting";
    case "running": return "Running";
    case "needs_review": return "Needs Review";
    case "complete": return "Completed";
    case "failed": return "Failed";
    case "cancelled": return "Stopped";
    default: return status;
  }
}

/* ── Metric chips (mirrors AgentStatusIndicator logic) ────────── */

function MetricsDisplay({ metrics }: { metrics: AgentMetric[] }) {
  const totals = new Map<string, { value: number; unit: string }>();
  for (const m of metrics) {
    const existing = totals.get(m.name);
    if (existing) {
      existing.value += m.value;
    } else {
      totals.set(m.name, { value: m.value, unit: m.unit });
    }
  }

  if (totals.size === 0) return null;

  const chips: { key: string; label: string; tooltip: string }[] = [];
  for (const [name, { value, unit }] of totals) {
    chips.push({
      key: name,
      label: `${formatValue(value, unit)} ${unit || name}`,
      tooltip: `${name}: ${value} ${unit}`,
    });
  }

  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <Zap className="h-3 w-3 text-zinc-500 flex-shrink-0" />
      {chips.map((c) => (
        <span
          key={c.key}
          className="px-1.5 py-0.5 text-xs rounded bg-zinc-700/50 text-zinc-400"
          title={c.tooltip}
        >
          {c.label}
        </span>
      ))}
    </div>
  );
}

function formatValue(value: number, unit: string): string {
  if (unit === "tokens" || unit === "bytes") {
    if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
    if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
  }
  if (unit === "ms") {
    if (value >= 1_000) return `${(value / 1_000).toFixed(1)}s`;
  }
  if (unit === "usd" || unit === "USD" || unit === "$") {
    return `$${value.toFixed(4)}`;
  }
  return value % 1 === 0 ? String(value) : value.toFixed(2);
}
