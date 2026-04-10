/**
 * ChatStatusIcon -- compact status indicator for the chat header.
 *
 * In LLM mode: a MessageSquare icon with a popover showing model info.
 * In Agent mode: a Bot icon surrounded by a circular progress ring.
 */

import { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { Bot, MessageSquare, StopCircle } from "lucide-react";
import type { AgentModeStatus, Model } from "../../lib/api";
import type { AgentMetric } from "./agent/AgentEventList";
import {
  RING_SIZE,
  STROKE_WIDTH,
  RADIUS,
  CIRCUMFERENCE,
  RING_COLORS,
  statusTextColor,
  statusLabel,
  MetricsDisplay,
} from "./ChatStatusHelpers";

interface ChatStatusIconProps {
  chatMode: "llm" | "agent";
  model: string;
  models: Model[];
  isAgentActive: boolean;
  agentStatus?: AgentModeStatus | null;
  agentMetrics?: AgentMetric[];
  agentError?: { message: string; recovery?: string } | null;
  onStopAgent?: () => void;
}

export function ChatStatusIcon({
  chatMode, model, models, isAgentActive, agentStatus, agentMetrics, agentError, onStopAgent,
}: ChatStatusIconProps) {
  const [isOpen, setIsOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const [popoverPos, setPopoverPos] = useState<{ top: number; left: number } | null>(null);

  useEffect(() => {
    if (!isOpen || !triggerRef.current) { setPopoverPos(null); return; }
    const rect = triggerRef.current.getBoundingClientRect();
    const popoverWidth = chatMode === "agent" ? 288 : 256;
    const margin = 8;
    let left = rect.left;
    if (left + popoverWidth > window.innerWidth - margin) left = window.innerWidth - popoverWidth - margin;
    if (left < margin) left = margin;
    setPopoverPos({ top: rect.bottom + margin, left });
  }, [isOpen, chatMode]);

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

  useEffect(() => {
    if (!isOpen) return;
    function handleKey(e: KeyboardEvent) { if (e.key === "Escape") setIsOpen(false); }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [isOpen]);

  /* ── LLM mode ──────────────────────────────────────────────────── */
  if (chatMode === "llm") {
    const currentModel = models.find((m) => m.id === model);
    return (
      <div className="relative">
        <button ref={triggerRef} onClick={() => setIsOpen(!isOpen)} className="p-1 rounded-md text-slate-400 hover:text-white hover:bg-white/10 transition-colors" title="Chat info" data-testid="chat-status-icon">
          <MessageSquare className="h-4 w-4" />
        </button>
        {isOpen && popoverPos && createPortal(
          <div ref={popoverRef} className="fixed z-[9999] w-64 bg-slate-900 border border-white/10 rounded-lg shadow-xl p-3 animate-in fade-in-0 zoom-in-95 duration-100" style={{ top: popoverPos.top, left: popoverPos.left }}>
            <div className="text-xs text-slate-500 mb-2">LLM Mode</div>
            <div className="flex items-center gap-2">
              <Bot className="h-4 w-4 text-indigo-400 flex-shrink-0" />
              <div className="min-w-0">
                <div className="text-sm font-medium text-white truncate">{currentModel?.name || model}</div>
                {currentModel?.description && <div className="text-xs text-slate-400 mt-0.5 line-clamp-2">{currentModel.description}</div>}
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

  let dashOffset: number;
  if (!status || !isAgentActive) dashOffset = CIRCUMFERENCE;
  else if (isRunning) dashOffset = hasProgress ? CIRCUMFERENCE - (progress / 100) * CIRCUMFERENCE : CIRCUMFERENCE * 0.75;
  else dashOffset = 0;

  const hasError = !!agentError || (status === "failed" && !!agentStatus?.error_msg);
  const iconColor = isRunning ? "text-blue-400" : status === "complete" ? "text-green-400" : status === "failed" || status === "cancelled" ? "text-red-400" : status === "needs_review" ? "text-yellow-400" : "text-slate-400";

  return (
    <div className="relative">
      <button ref={triggerRef} onClick={() => setIsOpen(!isOpen)} className={`relative p-0.5 rounded-md hover:bg-white/10 transition-colors ${hasError ? "ring-1 ring-red-500/50" : ""}`} title={status ? `Agent: ${statusLabel(status)}` : "Agent mode"} data-testid="chat-status-icon">
        <div className="relative" style={{ width: RING_SIZE, height: RING_SIZE }}>
          {status && isAgentActive && (
            <svg width={RING_SIZE} height={RING_SIZE} className={`absolute inset-0 ${isRunning && !hasProgress ? "animate-spin" : ""}`} style={isRunning && !hasProgress ? { animationDuration: "1.5s" } : undefined}>
              <circle cx={RING_SIZE / 2} cy={RING_SIZE / 2} r={RADIUS} fill="none" strokeWidth={STROKE_WIDTH} className="stroke-white/10" />
              <circle cx={RING_SIZE / 2} cy={RING_SIZE / 2} r={RADIUS} fill="none" strokeWidth={STROKE_WIDTH} strokeLinecap="round" className={ringColor} style={{ strokeDasharray: CIRCUMFERENCE, strokeDashoffset: dashOffset, transform: "rotate(-90deg)", transformOrigin: "center", transition: hasProgress ? "stroke-dashoffset 0.3s ease" : undefined }} />
            </svg>
          )}
          <Bot className={`absolute inset-0 m-auto h-4 w-4 ${iconColor}`} />
        </div>
      </button>

      {isOpen && popoverPos && createPortal(
        <div ref={popoverRef} className="fixed z-[9999] w-72 bg-slate-900 border border-white/10 rounded-lg shadow-xl animate-in fade-in-0 zoom-in-95 duration-100" style={{ top: popoverPos.top, left: popoverPos.left }}>
          <div className="p-3 space-y-3">
            <div className="flex items-center justify-between">
              <div className="text-xs text-slate-500">Agent Mode</div>
              {status && <span className={`text-xs font-medium ${statusTextColor(status)}`}>{statusLabel(status)}</span>}
            </div>
            {agentStatus?.phase && isRunning && <div className="text-xs text-slate-400">{agentStatus.phase}</div>}
            {isRunning && hasProgress && (
              <div>
                <div className="h-1.5 bg-zinc-700 rounded-full overflow-hidden"><div className="h-full bg-blue-500 transition-all duration-300 rounded-full" style={{ width: `${Math.min(progress, 100)}%` }} /></div>
                <div className="text-xs text-slate-500 mt-1">{Math.round(progress)}%</div>
              </div>
            )}
            {agentMetrics && agentMetrics.length > 0 && <MetricsDisplay metrics={agentMetrics} />}
            {agentError && (
              <div className="text-xs text-red-400 bg-red-500/10 rounded p-2">
                {agentError.message}
                {agentError.recovery && <span className="text-slate-400 block mt-1">{agentError.recovery}</span>}
              </div>
            )}
            {status === "failed" && agentStatus?.error_msg && !agentError && <div className="text-xs text-red-400 bg-red-500/10 rounded p-2 break-words">{agentStatus.error_msg}</div>}
            {isRunning && onStopAgent && (
              <button onClick={() => { onStopAgent(); setIsOpen(false); }} className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs text-red-400 hover:text-red-300 bg-red-500/10 hover:bg-red-500/20 transition-colors">
                <StopCircle className="h-3.5 w-3.5" />Stop Agent
              </button>
            )}
          </div>
        </div>,
        document.body,
      )}
    </div>
  );
}
