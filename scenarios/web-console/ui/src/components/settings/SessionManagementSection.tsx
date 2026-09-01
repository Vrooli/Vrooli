import { useCallback, useEffect, useRef, useState } from "react";
import {
  AlertCircle,
  Archive,
  ChevronDown,
  ChevronUp,
  Clock,
  Focus,
  RotateCcw,
  Timer,
  Trash2,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { HEADER_COLORS } from "../../consts/config";
import { BACKEND_OPTIONS } from "../../consts/backend-options";
import { POLICY_OPTIONS, parsePolicySelection, policyKey } from "../../consts/policy-options";
import { useCountdown } from "../../hooks/useCountdown";
import { useWorkspaceSync } from "../../hooks/useWorkspaceSync";
import type { ArchiveRetentionSnapshot, BackendID, PolicyMode, SessionInfo } from "../../api/sessions";
import { getArchiveRetention, updateSessionPolicy } from "../../api/sessions";
import { toErrorInfo } from "../../lib/errors";
import { getSessionDefaults, updateSessionDefaults } from "../../api/settings";
import { fetchCapabilities } from "../../api/capabilities";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { Button } from "../ui/button";

import { SettingsList } from "@vrooli/react-component-library/SettingsList/0";

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

function SessionDefaultsControl() {
  const { t } = useTranslation();
  const [defaultBackend, setDefaultBackend] = useState<BackendID>("standard");
  const [defaultPolicyKey, setDefaultPolicyKey] = useState<string>("never");
  const [availableBackends, setAvailableBackends] = useState<BackendID[]>(["standard"]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;
    // Load current defaults
    getSessionDefaults().then((d) => {
      if (cancelled) return;
      setDefaultBackend(d.default_backend as BackendID);
      if (d.default_policy) {
        setDefaultPolicyKey(policyKey(d.default_policy.mode as PolicyMode, d.default_policy.duration));
      }
    }).catch(() => {});
    // Load available backends
    fetchCapabilities().then((caps) => {
      if (cancelled) return;
      if (caps.session_backends) {
        setAvailableBackends(
          caps.session_backends.filter((b) => b.available).map((b) => b.id as BackendID),
        );
      }
    }).catch(() => {});
    return () => { cancelled = true; };
  }, []);

  const handleBackendChange = useCallback(async (backend: BackendID) => {
    setDefaultBackend(backend);
    setSaving(true);
    try {
      await updateSessionDefaults({ default_backend: backend });
    } catch {
      // Revert on failure would need previous value — best-effort for now
    } finally {
      setSaving(false);
    }
  }, []);

  const handlePolicyChange = useCallback(async (value: string) => {
    setDefaultPolicyKey(value);
    const parsed = parsePolicySelection(value);
    if (!parsed) return;
    setSaving(true);
    try {
      await updateSessionDefaults({
        default_policy: { mode: parsed.mode, duration: parsed.duration },
      });
    } catch {
      // Best-effort
    } finally {
      setSaving(false);
    }
  }, []);

  const showBackendSelector = availableBackends.length > 1;
  const backendOptions = BACKEND_OPTIONS.filter((b) => availableBackends.includes(b.id));

  return (
    <SettingsList.Group>
      <div>
        <div className="text-sm font-medium text-wc-text-secondary">{t(strings.settings.sessionsSection.defaultsTitle)}</div>
        <div className="text-[11px] text-wc-text-muted">
          {t(strings.settings.sessionsSection.defaultsHint)}
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-4">
        {showBackendSelector && (
          <div className="flex items-center gap-2">
            <label className="text-xs text-wc-text-secondary">{t(strings.settings.sessionsSection.defaultBackendLabel)}</label>
            <select
              data-testid="session-defaults-backend"
              className="h-7 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
              value={defaultBackend}
              disabled={saving}
              onChange={(e) => handleBackendChange(e.target.value as BackendID)}
            >
              {backendOptions.map((b) => (
                <option key={b.id} value={b.id}>{b.label}</option>
              ))}
            </select>
          </div>
        )}
        <div className="flex items-center gap-2">
          <label className="text-xs text-wc-text-secondary">{t(strings.settings.sessionsSection.defaultTimeoutLabel)}</label>
          <select
            data-testid="session-defaults-policy"
            className="h-7 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
            value={defaultPolicyKey}
            disabled={saving}
            onChange={(e) => handlePolicyChange(e.target.value)}
          >
            {POLICY_OPTIONS.map((opt) => (
              <option key={policyKey(opt.mode, opt.duration)} value={policyKey(opt.mode, opt.duration)}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>
    </SettingsList.Group>
  );
}

function formatStorageBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[index]}`;
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
  const { t } = useTranslation();
  const panes = useWorkspaceStore((state) => state.panes);
  const activePane = useWorkspaceStore((state) => state.activePane);
  const movePaneToIndex = useWorkspaceStore((state) => state.movePaneToIndex);
  const setActivePane = useWorkspaceStore((state) => state.setActivePane);
  const setPaneColor = useWorkspaceStore((state) => state.setPaneColor);
  const renamePaneById = useWorkspaceStore((state) => state.renamePaneById);
  const resetLayout = useWorkspaceStore((state) => state.resetLayout);
  const { syncActivePane, syncPaneOrder, syncPaneUpdate } = useWorkspaceSync();

  const [editingName, setEditingName] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [policyError, setPolicyError] = useState<string | null>(null);
  const [archiveRetention, setArchiveRetention] = useState<ArchiveRetentionSnapshot | null>(null);
  const policyErrorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (policyErrorTimer.current) {
        clearTimeout(policyErrorTimer.current);
      }
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    getArchiveRetention()
      .then((snapshot) => {
        if (!cancelled) setArchiveRetention(snapshot);
      })
      .catch(() => {
        if (!cancelled) setArchiveRetention(null);
      });
    return () => { cancelled = true; };
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
    <SettingsList>
      <SettingsList.Intro
        eyebrow={t(strings.settings.sessionsSection.eyebrow)}
        title={t(strings.settings.sessionsSection.title)}
        description={t(strings.settings.sessionsSection.description)}
      />

      <SessionDefaultsControl />

      <div
        data-testid="archive-storage-summary"
        data-entry-count={archiveRetention?.stats.entry_count ?? ""}
        data-total-bytes={archiveRetention?.stats.total_bytes ?? ""}
      >
        <SettingsList.Group>
          <div className="flex items-start gap-3">
            <Archive className="mt-0.5 h-4 w-4 text-wc-accent" />
            <div>
              <div className="text-sm font-medium text-wc-text-secondary">
                {t(strings.settings.sessionsSection.archiveStorageTitle)}
              </div>
              <div className="text-[11px] text-wc-text-muted">
                {archiveRetention
                  ? t(strings.settings.sessionsSection.archiveStorageSummary, {
                      count: archiveRetention.stats.entry_count,
                      size: formatStorageBytes(archiveRetention.stats.total_bytes),
                    })
                  : t(strings.settings.sessionsSection.archiveStorageLoading)}
              </div>
            </div>
          </div>
        </SettingsList.Group>
      </div>

      <SettingsList.Group>
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">{t(strings.settings.sessionsSection.openTerminals)}</div>
            <div className="text-[11px] text-wc-text-muted">
              {t(strings.settings.sessionsSection.openTerminalsHint)}
            </div>
          </div>
          <Button
            data-testid="sessions-reset-layout"
            variant="outline"
            size="sm"
            className="h-8 px-3 text-xs"
            onClick={resetLayout}
          >
            <RotateCcw className="me-1 h-3 w-3" />
            {t(strings.settings.sessionsSection.resetLayout)}
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
          <div className="py-6 text-center text-xs text-wc-text-faint">{t(strings.settings.sessionsSection.noTerminalsOpen)}</div>
        ) : (
          <div>
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
                          <div className="absolute left-0 top-full z-wc-chrome mt-1 hidden gap-1 rounded-xl border border-wc-default bg-wc-surface-raised p-2 shadow-xl group-hover:flex">
                            <button
                              className="h-4 w-4 rounded-full border border-wc-default"
                              style={{ backgroundColor: "rgb(var(--wc-surface-input))" }}
                              onClick={() => {
                                setPaneColor(pane.sessionId, "transparent");
                                syncPaneUpdate(pane.sessionId, { header_color: "transparent" });
                              }}
                              title={t(strings.settings.sessionsSection.noColor)}
                            />
                            {HEADER_COLORS.map((color) => (
                              <button
                                key={color}
                                className="h-4 w-4 rounded-full border border-wc-default"
                                style={{ backgroundColor: color }}
                                onClick={() => {
                                  setPaneColor(pane.sessionId, color);
                                  syncPaneUpdate(pane.sessionId, { header_color: color });
                                }}
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
                              if (editValue.trim()) {
                                const trimmed = editValue.trim();
                                renamePaneById(pane.sessionId, trimmed);
                                syncPaneUpdate(pane.sessionId, { name: trimmed });
                              }
                              setEditingName(null);
                            }}
                            onKeyDown={(event) => {
                              if (event.key === "Enter") {
                                if (editValue.trim()) {
                                  const trimmed = editValue.trim();
                                  renamePaneById(pane.sessionId, trimmed);
                                  syncPaneUpdate(pane.sessionId, { name: trimmed });
                                }
                                setEditingName(null);
                              } else if (event.key === "Escape") {
                                setEditingName(null);
                              }
                            }}
                          />
                        ) : (
                          <button
                            className="truncate text-start text-sm font-medium text-wc-text-secondary"
                            onClick={() => {
                              setEditingName(pane.sessionId);
                              setEditValue(pane.name);
                            }}
                            title={t(strings.settings.sessionsSection.renamePane)}
                          >
                            {pane.name}
                          </button>
                        )}
                      </div>

                      {session && (
                        <div className="flex items-center gap-3">
                          {session.backend === "persistent" && (
                            <span className="inline-flex items-center rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] font-medium text-blue-300">
                              {t(strings.settings.sessionsSection.persistent)}
                            </span>
                          )}
                          <SessionPolicyControl session={session} onPolicyChange={handlePolicyChange} />
                        </div>
                      )}
                    </div>

                    <div className="flex items-center gap-1">
                      <Button
                        data-testid={`sessions-pane-up-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        shape="square"
                        className="h-8 w-8"
                        disabled={index === 0}
                        onClick={() => {
                          movePaneToIndex(pane.sessionId, index - 1);
                          const reordered = [...panes];
                          const removed = reordered.splice(index, 1);
                          const moved = removed[0];
                          if (moved) reordered.splice(index - 1, 0, moved);
                          syncPaneOrder(reordered.map((entry) => entry.sessionId), activePane);
                        }}
                        title={t(strings.settings.sessionsSection.moveUp)}
                      >
                        <ChevronUp className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        data-testid={`sessions-pane-down-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        shape="square"
                        className="h-8 w-8"
                        disabled={index === panes.length - 1}
                        onClick={() => {
                          movePaneToIndex(pane.sessionId, index + 1);
                          const reordered = [...panes];
                          const removed = reordered.splice(index, 1);
                          const moved = removed[0];
                          if (moved) reordered.splice(index + 1, 0, moved);
                          syncPaneOrder(reordered.map((entry) => entry.sessionId), activePane);
                        }}
                        title={t(strings.settings.sessionsSection.moveDown)}
                      >
                        <ChevronDown className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        data-testid={`sessions-pane-focus-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        shape="square"
                        className="h-8 w-8"
                        onClick={() => {
                          setActivePane(pane.sessionId);
                          syncActivePane(panes.map((entry) => entry.sessionId), pane.sessionId);
                          onRequestClose();
                        }}
                        title={t(strings.settings.sessionsSection.focusPane)}
                      >
                        <Focus className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        data-testid={`sessions-pane-remove-${pane.sessionId}`}
                        variant="ghost"
                        size="icon"
                        shape="square"
                        className="h-8 w-8"
                        onClick={() => onDeleteSession(pane.sessionId)}
                        title={t(strings.settings.sessionsSection.terminateSession)}
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
      </SettingsList.Group>
    </SettingsList>
  );
}
