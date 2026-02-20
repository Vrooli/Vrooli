import { useState, useCallback, useEffect, useRef } from "react";
import {
  X,
  GripHorizontal,
  ChevronUp,
  ChevronDown,
  Focus,
  Trash2,
  RotateCcw,
  Timer,
  Clock,
  AlertCircle,
} from "lucide-react";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import { useCountdown } from "../hooks/useCountdown";
import { HEADER_COLORS } from "../consts/config";
import { POLICY_OPTIONS, policyKey, parsePolicySelection } from "../consts/policy-options";
import type { SessionInfo, PolicyMode } from "../lib/api";
import { updateSessionPolicy, toErrorInfo } from "../lib/api";
import { Button } from "./ui/button";

interface SessionsModalProps {
  sessions: Array<{ session: SessionInfo }>;
  onDeleteSession: (id: string) => void;
}

function SessionPolicyControl({
  session,
  onPolicyChange,
}: {
  session: SessionInfo;
  onPolicyChange: (id: string, mode: PolicyMode, duration?: string) => void;
}) {
  const currentKey = policyKey(session.policy.mode, session.policy.duration);
  const countdown = useCountdown(
    session.created_at,
    session.policy.mode,
    session.policy.duration,
  );

  return (
    <div className="flex items-center gap-1.5">
      <Timer className="h-3 w-3 shrink-0 text-wc-text-faint" />
      <select
        data-testid={`sessions-policy-select-${session.id}`}
        className="h-5 rounded border border-wc-default bg-wc-surface px-1 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
        value={currentKey}
        onChange={(e) => {
          const parsed = parsePolicySelection(e.target.value);
          if (!parsed) return;
          onPolicyChange(session.id, parsed.mode, parsed.duration);
        }}
      >
        {POLICY_OPTIONS.map((opt) => (
          <option key={policyKey(opt.mode, opt.duration)} value={policyKey(opt.mode, opt.duration)}>
            {opt.label}
          </option>
        ))}
      </select>
      {countdown && (
        <span className="flex items-center gap-0.5 text-xs text-wc-text-faint">
          <Clock className="h-2.5 w-2.5" />
          {countdown}
        </span>
      )}
    </div>
  );
}

export default function SessionsModal({
  sessions,
  onDeleteSession,
}: SessionsModalProps) {
  const sessionsModalOpen = useWorkspaceStore((s) => s.sessionsModalOpen);
  const setSessionsModalOpen = useWorkspaceStore((s) => s.setSessionsModalOpen);
  const panes = useWorkspaceStore((s) => s.panes);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const setPaneColor = useWorkspaceStore((s) => s.setPaneColor);
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const resetLayout = useWorkspaceStore((s) => s.resetLayout);

  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture } =
    useDraggablePosition({
      isActive: sessionsModalOpen,
      storageKey: "wc-sessions-pos",
      defaultPosition: () => {
        if (typeof window === "undefined") return { x: 100, y: 100 };
        return {
          x: Math.max(12, (window.innerWidth - 400) / 2),
          y: Math.max(12, window.innerHeight * 0.1),
        };
      },
    });

  const [editingName, setEditingName] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [policyError, setPolicyError] = useState<string | null>(null);
  const policyErrorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (policyErrorTimer.current) clearTimeout(policyErrorTimer.current);
    };
  }, []);

  const handlePolicyChange = useCallback(
    async (sessionId: string, mode: PolicyMode, duration?: string) => {
      try {
        await updateSessionPolicy(sessionId, { mode, duration });
        setPolicyError(null);
      } catch (err) {
        const info = toErrorInfo(err);
        setPolicyError(info.recovery || info.message);
        if (policyErrorTimer.current) clearTimeout(policyErrorTimer.current);
        policyErrorTimer.current = setTimeout(() => setPolicyError(null), 5000);
      }
    },
    [],
  );

  if (!sessionsModalOpen) return null;

  const close = () => setSessionsModalOpen(false);

  // Join store panes with session data by session ID
  const sessionMap = new Map(sessions.map((s) => [s.session.id, s.session]));

  return (
    <>
      <div
        data-testid="sessions-backdrop"
        className="fixed inset-0 z-40 bg-wc-backdrop"
        onClick={close}
      />

      <div
        ref={(node) => { elementRef.current = node; }}
        data-testid="sessions-modal"
        className="fixed left-0 top-0 z-50 w-[25rem] max-w-[calc(100vw-24px)] max-h-[80vh] overflow-hidden rounded-lg border border-wc-default bg-wc-surface-raised shadow-2xl flex flex-col"
        style={floatingStyle}
        onPointerDown={(e) => {
          const target = e.target as HTMLElement | null;
          const isOnHandle = Boolean(target?.closest("[data-drag-handle]"));
          const isOnControl = Boolean(target?.closest("button, a, input, textarea, select"));
          if (isOnHandle && !isOnControl) {
            pointerHandlers.onPointerDown(e);
          }
        }}
        onPointerMove={pointerHandlers.onPointerMove}
        onPointerUp={pointerHandlers.onPointerUp}
        onPointerCancel={pointerHandlers.onPointerCancel}
        onClickCapture={handleClickCapture}
      >
        {/* Drag handle header */}
        <div
          data-drag-handle
          className="flex items-center justify-between px-4 py-2 border-b border-wc-default cursor-grab active:cursor-grabbing select-none touch-none"
        >
          <div className="flex items-center gap-2">
            <GripHorizontal className="h-4 w-4 text-wc-text-faint" />
            <h2 className="text-sm font-semibold text-wc-text-primary">
              Sessions
            </h2>
          </div>
          <div className="flex items-center gap-1">
            <Button
              data-testid="sessions-reset-layout"
              variant="ghost"
              size="sm"
              className="text-xs text-wc-text-faint"
              onClick={resetLayout}
              title="Reset grid layout"
            >
              <RotateCcw className="mr-1 h-3 w-3" /> Reset Layout
            </Button>
            <Button
              data-testid="sessions-close"
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={close}
            >
              <X className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>

        {/* Policy error */}
        {policyError && (
          <div
            data-testid="sessions-policy-error"
            className="mx-4 mt-2 flex items-start gap-2 rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300"
          >
            <AlertCircle className="mt-0.5 h-3 w-3 shrink-0" />
            <span>{policyError}</span>
            <button
              className="ml-auto shrink-0 text-red-400 hover:text-red-200"
              onClick={() => setPolicyError(null)}
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        {/* Pane list */}
        <div className="flex-1 overflow-y-auto p-4 space-y-1.5">
          {panes.length === 0 ? (
            <div className="text-center text-xs text-wc-text-faint py-2">
              No terminals open
            </div>
          ) : (
            panes.map((pane, idx) => {
              const session = sessionMap.get(pane.sessionId);
              return (
                <div
                  key={pane.sessionId}
                  data-testid={`sessions-pane-${pane.sessionId}`}
                  className="rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 space-y-1"
                >
                  <div className="flex items-center gap-1.5">
                    {/* Color picker */}
                    <div className="relative group">
                      <div
                        className="h-3.5 w-3.5 rounded-full shrink-0 border border-wc-default cursor-pointer"
                        style={{
                          backgroundColor:
                            pane.headerColor !== "transparent"
                              ? pane.headerColor
                              : "rgb(var(--wc-surface-input))",
                        }}
                      />
                      <div className="hidden group-hover:flex absolute top-full left-0 mt-1 gap-0.5 rounded border border-wc-default bg-wc-surface-raised p-1 z-10">
                        <button
                          className="h-3.5 w-3.5 rounded-full border border-wc-default"
                          style={{ backgroundColor: "rgb(var(--wc-surface-input))" }}
                          onClick={() => setPaneColor(pane.sessionId, "transparent")}
                        />
                        {HEADER_COLORS.map((color) => (
                          <button
                            key={color}
                            className="h-3.5 w-3.5 rounded-full border border-wc-default"
                            style={{ backgroundColor: color }}
                            onClick={() => setPaneColor(pane.sessionId, color)}
                          />
                        ))}
                      </div>
                    </div>

                    {/* Editable name */}
                    {editingName === pane.sessionId ? (
                      <input
                        className="flex-1 rounded border border-wc-accent bg-wc-surface-base px-1 text-xs text-wc-text-primary outline-none"
                        value={editValue}
                        autoFocus
                        onChange={(e) => setEditValue(e.target.value)}
                        onBlur={() => {
                          if (editValue.trim()) renamePaneById(pane.sessionId, editValue.trim());
                          setEditingName(null);
                        }}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            if (editValue.trim()) renamePaneById(pane.sessionId, editValue.trim());
                            setEditingName(null);
                          } else if (e.key === "Escape") {
                            setEditingName(null);
                          }
                        }}
                      />
                    ) : (
                      <button
                        className="flex-1 text-left text-xs text-wc-text-secondary truncate hover:text-wc-text-primary"
                        onClick={() => {
                          setEditingName(pane.sessionId);
                          setEditValue(pane.name);
                        }}
                      >
                        {pane.name}
                      </button>
                    )}

                    {/* Reorder buttons */}
                    <Button
                      data-testid={`sessions-pane-up-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      disabled={idx === 0}
                      onClick={() => movePaneToIndex(pane.sessionId, idx - 1)}
                      title="Move up"
                    >
                      <ChevronUp className="h-3 w-3" />
                    </Button>
                    <Button
                      data-testid={`sessions-pane-down-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      disabled={idx === panes.length - 1}
                      onClick={() => movePaneToIndex(pane.sessionId, idx + 1)}
                      title="Move down"
                    >
                      <ChevronDown className="h-3 w-3" />
                    </Button>
                    <Button
                      data-testid={`sessions-pane-focus-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      onClick={() => {
                        setActivePane(pane.sessionId);
                        close();
                      }}
                      title="Focus terminal"
                    >
                      <Focus className="h-3 w-3" />
                    </Button>
                    <Button
                      data-testid={`sessions-pane-remove-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      onClick={() => onDeleteSession(pane.sessionId)}
                      title="Close terminal"
                    >
                      <Trash2 className="h-3 w-3 text-wc-text-faint" />
                    </Button>
                  </div>

                  {/* Policy control */}
                  {session && (
                    <SessionPolicyControl
                      session={session}
                      onPolicyChange={handlePolicyChange}
                    />
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>
    </>
  );
}
