import { useCallback, useEffect, useRef, useState } from "react";
import {
  AlertCircle,
  ChevronDown,
  ChevronUp,
  Clock,
  Focus,
  RotateCcw,
  Timer,
  Trash2,
} from "lucide-react";
import { HEADER_COLORS } from "../../consts/config";
import { POLICY_OPTIONS, parsePolicySelection, policyKey } from "../../consts/policy-options";
import { useCountdown } from "../../hooks/useCountdown";
import { useWorkspaceSync } from "../../hooks/useWorkspaceSync";
import type { PolicyMode, SessionInfo } from "../../lib/api";
import { toErrorInfo, updateSessionPolicy } from "../../lib/api";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { Button } from "../ui/button";
import { SettingsCard, SettingsSectionIntro } from "./primitives";

function SessionPolicyControl({
  session,
  onPolicyChange,
}: {
  session: SessionInfo;
  onPolicyChange: (sessionId: string, mode: PolicyMode, duration?: string) => void;
}) {
  const currentKey = policyKey(session.policy.mode, session.policy.duration);
  const countdown = useCountdown(session.created_at, session.policy.mode, session.policy.duration);

  return (
    <div className="flex items-center gap-1.5">
      <Timer className="h-3 w-3 shrink-0 text-wc-text-faint" />
      <select
        data-testid={`sessions-policy-select-${session.id}`}
        className="h-6 rounded-lg border border-wc-default bg-wc-surface px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
        value={currentKey}
        onChange={(event) => {
          const parsed = parsePolicySelection(event.target.value);
          if (!parsed) return;
          onPolicyChange(session.id, parsed.mode, parsed.duration);
        }}
      >
        {POLICY_OPTIONS.map((option) => (
          <option key={policyKey(option.mode, option.duration)} value={policyKey(option.mode, option.duration)}>
            {option.label}
          </option>
        ))}
      </select>
      {countdown && (
        <span className="flex items-center gap-1 text-xs text-wc-text-faint">
          <Clock className="h-2.5 w-2.5" />
          {countdown}
        </span>
      )}
    </div>
  );
}

interface SessionManagementSectionProps {
  sessions: Array<{ session: SessionInfo }>;
  onDeleteSession: (id: string) => void;
  onRequestClose: () => void;
}

export default function SessionManagementSection({
  sessions,
  onDeleteSession,
  onRequestClose,
}: SessionManagementSectionProps) {
  const panes = useWorkspaceStore((state) => state.panes);
  const movePaneToIndex = useWorkspaceStore((state) => state.movePaneToIndex);
  const setActivePane = useWorkspaceStore((state) => state.setActivePane);
  const setPaneColor = useWorkspaceStore((state) => state.setPaneColor);
  const renamePaneById = useWorkspaceStore((state) => state.renamePaneById);
  const resetLayout = useWorkspaceStore((state) => state.resetLayout);
  const { syncActivePane } = useWorkspaceSync();

  const [editingName, setEditingName] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [policyError, setPolicyError] = useState<string | null>(null);
  const policyErrorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (policyErrorTimer.current) {
        clearTimeout(policyErrorTimer.current);
      }
    };
  }, []);

  const handlePolicyChange = useCallback(async (sessionId: string, mode: PolicyMode, duration?: string) => {
    try {
      await updateSessionPolicy(sessionId, { mode, duration });
      setPolicyError(null);
    } catch (error) {
      const info = toErrorInfo(error);
      setPolicyError(info.recovery || info.message);
      if (policyErrorTimer.current) {
        clearTimeout(policyErrorTimer.current);
      }
      policyErrorTimer.current = setTimeout(() => setPolicyError(null), 5000);
    }
  }, []);

  const sessionMap = new Map(sessions.map((item) => [item.session.id, item.session]));

  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow="Terminal Runtime"
        title="Sessions"
        description="Inspect active terminals, change expiration policies, reorder panes, and jump directly to the pane you need."
      />

      <SettingsCard className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">Open terminals</div>
            <div className="text-[11px] text-wc-text-muted">
              Sessions are server-side processes. Panes are their workspace views.
            </div>
          </div>
          <Button
            data-testid="sessions-reset-layout"
            variant="outline"
            size="sm"
            className="h-8 px-3 text-xs"
            onClick={resetLayout}
          >
            <RotateCcw className="mr-1 h-3 w-3" />
            Reset layout
          </Button>
        </div>

        {policyError && (
          <div
            data-testid="sessions-policy-error"
            className="flex items-start gap-2 rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300"
          >
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>{policyError}</span>
          </div>
        )}

        {panes.length === 0 ? (
          <div className="py-6 text-center text-xs text-wc-text-faint">No terminals open</div>
        ) : (
          <div className="space-y-3">
            {panes.map((pane, index) => {
              const session = sessionMap.get(pane.sessionId);
              return (
                <div
                  key={pane.sessionId}
                  data-testid={`sessions-pane-${pane.sessionId}`}
                  className="rounded-xl border border-wc-default bg-wc-surface-base/70 p-3"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1 space-y-2">
                      <div className="flex items-center gap-2">
                        <div className="relative group">
                          <div
                            className="h-4 w-4 rounded-full border border-wc-default"
                            style={{
                              backgroundColor:
                                pane.headerColor !== "transparent"
                                  ? pane.headerColor
                                  : "rgb(var(--wc-surface-input))",
                            }}
                          />
                          <div className="absolute left-0 top-full z-10 mt-1 hidden gap-1 rounded-xl border border-wc-default bg-wc-surface-raised p-2 shadow-xl group-hover:flex">
                            <button
                              className="h-4 w-4 rounded-full border border-wc-default"
                              style={{ backgroundColor: "rgb(var(--wc-surface-input))" }}
                              onClick={() => setPaneColor(pane.sessionId, "transparent")}
                              title="No color"
                            />
                            {HEADER_COLORS.map((color) => (
                              <button
                                key={color}
                                className="h-4 w-4 rounded-full border border-wc-default"
                                style={{ backgroundColor: color }}
                                onClick={() => setPaneColor(pane.sessionId, color)}
                                title={color}
                              />
                            ))}
                          </div>
                        </div>

                        {editingName === pane.sessionId ? (
                          <input
                            className="min-w-0 flex-1 rounded-lg border border-wc-accent bg-wc-surface-input px-2 py-1 text-sm text-wc-text-primary outline-none"
                            value={editValue}
                            autoFocus
                            onChange={(event) => setEditValue(event.target.value)}
                            onBlur={() => {
                              if (editValue.trim()) renamePaneById(pane.sessionId, editValue.trim());
                              setEditingName(null);
                            }}
                            onKeyDown={(event) => {
                              if (event.key === "Enter") {
                                if (editValue.trim()) renamePaneById(pane.sessionId, editValue.trim());
                                setEditingName(null);
                              } else if (event.key === "Escape") {
                                setEditingName(null);
                              }
                            }}
                          />
                        ) : (
                          <button
                            className="truncate text-left text-sm font-medium text-wc-text-secondary"
                            onClick={() => {
                              setEditingName(pane.sessionId);
                              setEditValue(pane.name);
                            }}
                            title="Rename pane"
                          >
                            {pane.name}
                          </button>
                        )}
                      </div>

                      {session && (
                        <SessionPolicyControl session={session} onPolicyChange={handlePolicyChange} />
                      )}
                    </div>

                    <div className="flex items-center gap-1">
                      <Button
                        data-testid={`sessions-pane-up-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        disabled={index === 0}
                        onClick={() => movePaneToIndex(pane.sessionId, index - 1)}
                        title="Move up"
                      >
                        <ChevronUp className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        data-testid={`sessions-pane-down-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        disabled={index === panes.length - 1}
                        onClick={() => movePaneToIndex(pane.sessionId, index + 1)}
                        title="Move down"
                      >
                        <ChevronDown className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        data-testid={`sessions-pane-focus-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => {
                          setActivePane(pane.sessionId);
                          syncActivePane(useWorkspaceStore.getState().panes.map((entry) => entry.sessionId), pane.sessionId);
                          onRequestClose();
                        }}
                        title="Focus pane"
                      >
                        <Focus className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        data-testid={`sessions-pane-remove-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => onDeleteSession(pane.sessionId)}
                        title="Terminate session"
                      >
                        <Trash2 className="h-3.5 w-3.5 text-wc-error-detail" />
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </SettingsCard>
    </div>
  );
}
