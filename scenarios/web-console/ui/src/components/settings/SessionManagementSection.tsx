import { useCallback, useEffect, useRef, useState, type CSSProperties } from "react";
import {
  AlertCircle,
  Archive,
  ChevronDown,
  ChevronUp,
  Clock,
  Focus,
  FolderOpen,
  MoreVertical,
  RotateCcw,
  TerminalSquare,
  Timer,
  Trash2,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { strings } from "../../consts/strings";
import { HEADER_COLORS } from "../../consts/config";
import { BACKEND_OPTIONS } from "../../consts/backend-options";
import { POLICY_OPTIONS, parsePolicySelection, policyKey } from "../../consts/policy-options";
import { useCountdown } from "../../hooks/useCountdown";
import { useMediaQuery } from "../../hooks/useMediaQuery";
import { useWorkspaceSync } from "../../hooks/useWorkspaceSync";
import type { ArchivePrunePlan, ArchiveRetentionSnapshot, BackendID, PolicyMode, SessionInfo } from "../../api/sessions";
import { getArchiveRetention, pruneArchive, updateSessionPolicy } from "../../api/sessions";
import { toErrorInfo } from "../../lib/errors";
import { getSessionDefaults, updateSessionDefaults } from "../../api/settings";
import { fetchCapabilities } from "../../api/capabilities";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { cn } from "../../lib/classnames";
import { Button } from "../ui/button";

import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";

interface SelectOption {
  value: string;
  label: string;
}

/**
 * One styled select for every choice on the surface. A native control keeps the
 * platform picker on touch (the right affordance on a phone) while appearance,
 * border, radius, and focus ring are shared so it stops reading as an unstyled
 * browser widget next to the library buttons.
 */
function NativeSelect({
  id,
  testId,
  value,
  onChange,
  options,
  ariaLabel,
  disabled,
  size = "compact",
  className,
}: {
  id?: string;
  testId?: string;
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  ariaLabel: string;
  disabled?: boolean;
  size?: "compact" | "touch";
  className?: string;
}) {
  return (
    <span className={cn("relative inline-flex min-w-0 items-center", className)}>
      <select
        id={id}
        data-testid={testId}
        aria-label={ariaLabel}
        disabled={disabled}
        value={value}
        onChange={(event) => { onChange(event.target.value); }}
        className={cn(
          "w-full appearance-none truncate rounded-lg border border-wc-default bg-wc-surface-input ps-2.5 pe-7 font-medium text-wc-text-secondary outline-none transition-colors",
          "focus-visible:border-wc-accent focus-visible:ring-2 focus-visible:ring-wc-accent/40 disabled:opacity-50",
          size === "touch" ? "h-11 text-sm" : "h-7 text-xs",
        )}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      <ChevronDown aria-hidden className="pointer-events-none absolute end-2 h-3.5 w-3.5 text-wc-text-faint" />
    </span>
  );
}

/** The pane color palette, shared by the desktop hover popover and the touch sheet. */
function ColorSwatches({ onSelect, className }: { onSelect: (color: string) => void; className?: string }) {
  const { t } = useTranslation();
  return (
    <div className={cn("flex flex-wrap items-center gap-1.5", className)}>
      <button
        type="button"
        className="h-6 w-6 rounded-full border border-wc-default"
        style={{ backgroundColor: "rgb(var(--wc-surface-input))" }}
        onClick={() => { onSelect("transparent"); }}
        title={t(strings.settings.sessionsSection.noColor)}
      />
      {HEADER_COLORS.map((color) => (
        <button
          key={color}
          type="button"
          className="h-6 w-6 rounded-full border border-wc-default"
          style={{ backgroundColor: color }}
          onClick={() => { onSelect(color); }}
          title={color}
        />
      ))}
    </div>
  );
}

/**
 * The pane color swatch. Fine pointers get the hover popover; touch gets a
 * static swatch because a hover-only popover is unreachable with a finger —
 * the same choices live in the action sheet.
 */
function PaneColorDot({
  color,
  interactive,
  onSelect,
}: {
  color: string;
  interactive: boolean;
  onSelect: (color: string) => void;
}) {
  const swatch = (
    <span
      className="block h-4 w-4 rounded-full border border-wc-default"
      style={{ backgroundColor: color !== "transparent" ? color : "rgb(var(--wc-surface-input))" }}
    />
  );
  if (!interactive) {
    return <span className="shrink-0">{swatch}</span>;
  }
  return (
    <span className="group relative shrink-0">
      {swatch}
      <span className="absolute start-0 top-full z-wc-chrome mt-1 hidden gap-1 rounded-xl border border-wc-default bg-wc-surface-raised p-2 shadow-xl group-hover:flex">
        <ColorSwatches onSelect={onSelect} />
      </span>
    </span>
  );
}

function SessionPolicyControl({
  session,
  interactive,
  onPolicyChange,
}: {
  session: SessionInfo;
  interactive: boolean;
  onPolicyChange: (sessionId: string, mode: PolicyMode, duration?: string) => void;
}) {
  const { t } = useTranslation();
  const currentKey = policyKey(session.policy.mode, session.policy.duration);
  const countdown = useCountdown(session.created_at, session.policy.mode, session.policy.duration);
  const currentLabel = POLICY_OPTIONS.find(
    (option) => policyKey(option.mode, option.duration) === currentKey,
  )?.label;

  if (!interactive) {
    return (
      <span className="inline-flex items-center gap-1.5">
        <span className="inline-flex items-center gap-1 rounded-full bg-wc-surface-input px-2 py-0.5 text-[10px] font-medium text-wc-text-secondary">
          <Timer className="h-3 w-3 shrink-0 text-wc-text-faint" />
          {currentLabel}
        </span>
        {countdown && <span className="text-[10px] text-wc-text-faint">{countdown}</span>}
      </span>
    );
  }

  return (
    <span className="inline-flex items-center gap-1.5">
      <Timer aria-hidden className="h-3 w-3 shrink-0 text-wc-text-faint" />
      <NativeSelect
        testId={`sessions-policy-select-${session.id}`}
        ariaLabel={t(strings.settings.sessionsSection.policyLabel)}
        value={currentKey}
        onChange={(value) => {
          const parsed = parsePolicySelection(value);
          if (!parsed) return;
          onPolicyChange(session.id, parsed.mode, parsed.duration);
        }}
        options={POLICY_OPTIONS.map((option) => ({
          value: policyKey(option.mode, option.duration),
          label: option.label,
        }))}
      />
      {countdown && (
        <span className="flex items-center gap-1 text-xs text-wc-text-faint">
          <Clock className="h-2.5 w-2.5" />
          {countdown}
        </span>
      )}
    </span>
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
    getSessionDefaults().then((d) => {
      if (cancelled) return;
      setDefaultBackend(d.default_backend as BackendID);
      setDefaultPolicyKey(policyKey(d.default_policy.mode, d.default_policy.duration));
    }).catch(() => {});
    fetchCapabilities().then((caps) => {
      if (cancelled) return;
      if (caps.session_backends) {
        setAvailableBackends(
          caps.session_backends.filter((b) => b.available).map((b) => b.id),
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
      <div className="flex min-w-0 flex-col gap-2">
        <div className="flex flex-col gap-0.5">
          <div className="text-sm font-medium text-wc-text-secondary">
            {t(strings.settings.sessionsSection.defaultsTitle)}
          </div>
          <div className="text-[11px] text-wc-text-muted">
            {t(strings.settings.sessionsSection.defaultsHint)}
          </div>
        </div>

        <div className="grid grid-cols-[max-content_minmax(0,1fr)] items-center gap-x-3 gap-y-2">
          {showBackendSelector && (
            <>
              <label
                htmlFor="session-defaults-backend"
                className="text-xs text-wc-text-secondary"
              >
                {t(strings.settings.sessionsSection.defaultBackendLabel)}
              </label>
              <NativeSelect
                id="session-defaults-backend"
                testId="session-defaults-backend"
                ariaLabel={t(strings.settings.sessionsSection.defaultBackendLabel)}
                value={defaultBackend}
                disabled={saving}
                onChange={(value) => { void handleBackendChange(value as BackendID); }}
                options={backendOptions.map((b) => ({ value: b.id, label: b.label }))}
                className="w-full sm:w-56"
              />
            </>
          )}
          <label htmlFor="session-defaults-policy" className="text-xs text-wc-text-secondary">
            {t(strings.settings.sessionsSection.defaultTimeoutLabel)}
          </label>
          <NativeSelect
            id="session-defaults-policy"
            testId="session-defaults-policy"
            ariaLabel={t(strings.settings.sessionsSection.defaultTimeoutLabel)}
            value={defaultPolicyKey}
            disabled={saving}
            onChange={(value) => { void handlePolicyChange(value); }}
            options={POLICY_OPTIONS.map((option) => ({
              value: policyKey(option.mode, option.duration),
              label: option.label,
            }))}
            className="w-full sm:w-56"
          />
        </div>
      </div>
    </SettingsList.Group>
  );
}

function formatStorageBytes(bytes: number): string {
  if (bytes < 1024) return `${String(bytes)} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[index] ?? "B"}`;
}

/**
 * Archive size as a single quiet row, not a full section: skeleton while the
 * measurement is in flight, a retryable line when it fails, and the numbers
 * once they land. The row carries the one action the numbers imply: prune the
 * entries the retention policy no longer keeps. Pruning is always confirmed
 * from a dry-run plan, so the destructive call is never the first one.
 */
function ArchiveStorageSummary({ onRequestClose }: { onRequestClose: () => void }) {
  const { t } = useTranslation();
  const setSidebarView = useWorkspaceStore((state) => state.setSidebarView);
  const [state, setState] = useState<{ status: "loading" | "ready" | "error"; data: ArchiveRetentionSnapshot | null }>({
    status: "loading",
    data: null,
  });
  const [plan, setPlan] = useState<ArchivePrunePlan | null>(null);
  const [phase, setPhase] = useState<"idle" | "scanning" | "confirm" | "applying" | "done" | "error">("idle");

  const load = useCallback(() => {
    setState({ status: "loading", data: null });
    getArchiveRetention()
      .then((snapshot) => { setState({ status: "ready", data: snapshot }); })
      .catch(() => { setState({ status: "error", data: null }); });
  }, []);

  useEffect(() => { load(); }, [load]);

  const scan = useCallback(async () => {
    setPhase("scanning");
    try {
      const next = await pruneArchive(false);
      setPlan(next);
      setPhase(next.actions.length === 0 ? "done" : "confirm");
    } catch {
      setPlan(null);
      setPhase("error");
    }
  }, []);

  const apply = useCallback(async () => {
    setPhase("applying");
    try {
      const result = await pruneArchive(true);
      setPlan(result);
      setPhase("done");
      load();
    } catch {
      setPhase("error");
    }
  }, [load]);

  const reset = useCallback(() => {
    setPlan(null);
    setPhase("idle");
  }, []);

  const stats = state.data?.stats;
  const busy = phase === "scanning" || phase === "applying";
  const reclaimed = plan?.reclaimed_bytes ?? 0;

  return (
    <div
      data-testid="archive-storage-summary"
      data-entry-count={state.status === "ready" && stats ? stats.entry_count : ""}
      data-total-bytes={state.status === "ready" && stats ? stats.total_bytes : ""}
    >
      <SettingsList.Group>
        <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
          <div className="flex min-w-0 items-start gap-2.5">
            <Archive aria-hidden className="mt-0.5 h-4 w-4 shrink-0 text-wc-accent" />
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium text-wc-text-secondary">
                {t(strings.settings.sessionsSection.archiveStorageTitle)}
              </div>
              {state.status === "loading" && (
                <span
                  aria-hidden
                  className="mt-1 block h-3 w-2/3 max-w-[13rem] animate-pulse rounded-full bg-wc-surface-input"
                />
              )}
              {state.status === "error" && (
                <div className="flex flex-wrap items-center gap-2 text-[11px] text-wc-error-text">
                  <span>{t(strings.settings.sessionsSection.archiveStorageError)}</span>
                  <button
                    type="button"
                    data-testid="archive-storage-retry"
                    onClick={load}
                    className="rounded-md px-1.5 py-0.5 font-medium text-wc-accent underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent/40"
                  >
                    {t(strings.settings.sessionsSection.retry)}
                  </button>
                </div>
              )}
              {state.status === "ready" && stats && (
                <div className="text-[11px] text-wc-text-muted">
                  {t(strings.settings.sessionsSection.archiveStorageSummary, {
                    count: stats.entry_count,
                    size: formatStorageBytes(stats.total_bytes),
                  })}
                </div>
              )}
              {phase === "done" && (
                <div data-testid="archive-storage-prune-result" className="mt-1 text-[11px] text-wc-text-muted">
                  {reclaimed > 0
                    ? t(strings.settings.sessionsSection.archiveStoragePruneDone, { size: formatStorageBytes(reclaimed) })
                    : t(strings.settings.sessionsSection.archiveStorageNothingToPrune)}
                </div>
              )}
              {phase === "error" && (
                <div data-testid="archive-storage-prune-error" className="mt-1 text-[11px] text-wc-error-text">
                  {t(strings.settings.sessionsSection.archiveStoragePruneError)}
                </div>
              )}
            </div>
          </div>
          {state.status === "ready" && (
            <div className="flex shrink-0 flex-wrap items-center gap-1.5 sm:justify-end">
              <Button
                data-testid="archive-storage-prune"
                variant="outline"
                size="sm"
                className="h-8 px-3 text-xs"
                disabled={busy}
                onClick={() => { void scan(); }}
              >
                <Trash2 className="me-1 h-3 w-3" />
                {busy ? t(strings.settings.sessionsSection.archiveStoragePruneRunning) : t(strings.settings.sessionsSection.archiveStoragePrune)}
              </Button>
              <Button
                data-testid="archive-storage-manage"
                variant="ghost"
                size="sm"
                className="h-8 px-3 text-xs"
                onClick={() => { setSidebarView("archive"); onRequestClose(); }}
              >
                <FolderOpen className="me-1 h-3 w-3" />
                {t(strings.settings.sessionsSection.archiveStorageManage)}
              </Button>
            </div>
          )}
        </div>
      </SettingsList.Group>

      <BottomSheet
        open={phase === "confirm" || phase === "applying"}
        onOpenChange={(open) => { if (!open && phase !== "applying") reset(); }}
        title={t(strings.settings.sessionsSection.archiveStoragePruneTitle)}
        closeLabel={t(strings.settings.closeAriaLabel)}
        testId="archive-prune-sheet"
        avoidKeyboard
      >
        <div className="flex flex-col gap-4 pb-2">
          <p className="text-sm text-wc-text-secondary">
            {t(strings.settings.sessionsSection.archiveStoragePruneBody, {
              count: plan?.actions.length ?? 0,
              size: formatStorageBytes(reclaimed),
            })}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              className="flex-1"
              disabled={phase === "applying"}
              onClick={reset}
            >
              {t(strings.settings.sessionsSection.cancel)}
            </Button>
            <Button
              data-testid="archive-storage-prune-confirm"
              variant="danger"
              className="flex-1"
              disabled={phase === "applying"}
              onClick={() => { void apply(); }}
            >
              {phase === "applying"
                ? t(strings.settings.sessionsSection.archiveStoragePruneRunning)
                : t(strings.settings.sessionsSection.archiveStoragePruneConfirm, { size: formatStorageBytes(reclaimed) })}
            </Button>
          </div>
        </div>
      </BottomSheet>
    </div>
  );
}

function SheetAction({
  icon: Icon,
  label,
  testId,
  disabled,
  destructive,
  onSelect,
}: {
  icon: typeof ChevronUp;
  label: string;
  testId: string;
  disabled?: boolean;
  destructive?: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      data-testid={testId}
      disabled={disabled}
      onClick={onSelect}
      className={cn(
        "flex min-h-11 w-full items-center gap-3 rounded-lg px-3 text-start text-sm transition hover:bg-wc-surface-input focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent/40 disabled:opacity-40",
        destructive ? "text-wc-error-detail" : "text-wc-text-secondary",
      )}
    >
      <Icon className="h-4 w-4 shrink-0" />
      <span>{label}</span>
    </button>
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
  const { t } = useTranslation();
  const isMobile = useMediaQuery("(max-width: 767px)");
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
  const [menuPaneId, setMenuPaneId] = useState<string | null>(null);
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
      policyErrorTimer.current = setTimeout(() => { setPolicyError(null); }, 5000);
    }
  }, []);

  const sessionMap = new Map(sessions.map((item) => [item.session.id, item.session]));

  const movePane = useCallback((paneId: string, direction: -1 | 1) => {
    const index = panes.findIndex((entry) => entry.sessionId === paneId);
    const target = index + direction;
    if (index < 0 || target < 0 || target >= panes.length) return;
    movePaneToIndex(paneId, target);
    const reordered = [...panes];
    const removed = reordered.splice(index, 1);
    const moved = removed[0];
    if (moved) reordered.splice(target, 0, moved);
    syncPaneOrder(reordered.map((entry) => entry.sessionId), activePane);
  }, [panes, movePaneToIndex, syncPaneOrder, activePane]);

  const focusPane = useCallback((paneId: string) => {
    setActivePane(paneId);
    syncActivePane(panes.map((entry) => entry.sessionId), paneId);
    onRequestClose();
  }, [panes, setActivePane, syncActivePane, onRequestClose]);

  const renamePane = useCallback((paneId: string, name: string) => {
    const trimmed = name.trim();
    if (!trimmed) return;
    renamePaneById(paneId, trimmed);
    syncPaneUpdate(paneId, { name: trimmed });
  }, [renamePaneById, syncPaneUpdate]);

  const recolorPane = useCallback((paneId: string, color: string) => {
    setPaneColor(paneId, color);
    syncPaneUpdate(paneId, { header_color: color });
  }, [setPaneColor, syncPaneUpdate]);

  const menuPane = menuPaneId ? panes.find((entry) => entry.sessionId === menuPaneId) : undefined;
  const menuIndex = menuPane ? panes.indexOf(menuPane) : -1;
  const menuSession = menuPane ? sessionMap.get(menuPane.sessionId) : undefined;

  const closeMenu = useCallback(() => { setMenuPaneId(null); }, []);

  return (
    <>
      <SettingsList
        variant="auto"
        density={isMobile ? "compact" : "comfortable"}
        style={{
          "--rcl-settings-section-gap": isMobile ? "1.25rem" : undefined,
          "--rcl-settings-group-gap": isMobile ? "0.5rem" : undefined,
        } as CSSProperties}
      >
        {!isMobile && (
          <SettingsList.Intro
            eyebrow={t(strings.settings.sessionsSection.eyebrow)}
            title={t(strings.settings.sessionsSection.title)}
            description={t(strings.settings.sessionsSection.description)}
          />
        )}

        <SessionDefaultsControl />

        <ArchiveStorageSummary onRequestClose={onRequestClose} />

        <SettingsList.Group>
          <div className="flex min-w-0 flex-col gap-3">
            <div className="flex min-w-0 items-center justify-between gap-3">
              <div className="min-w-0">
                <div className="text-sm font-medium text-wc-text-secondary">
                  {t(strings.settings.sessionsSection.openTerminals)}
                </div>
                <div className="text-[11px] text-wc-text-muted">
                  {t(strings.settings.sessionsSection.openTerminalsHint)}
                </div>
              </div>
              <Button
                data-testid="sessions-reset-layout"
                variant="outline"
                size="sm"
                className="h-9 shrink-0 px-3 text-xs"
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
              <div className="flex flex-col items-center gap-2 py-6 text-center">
                <TerminalSquare aria-hidden className="h-5 w-5 text-wc-text-faint" />
                <div className="text-xs text-wc-text-faint">
                  {t(strings.settings.sessionsSection.noTerminalsOpen)}
                </div>
              </div>
            ) : (
              <div className="flex min-w-0 flex-col divide-y divide-wc-default/70 rounded-xl border border-wc-default bg-wc-surface-base/40">
                {panes.map((pane, index) => {
                  const session = sessionMap.get(pane.sessionId);
                  const isActive = pane.sessionId === activePane;
                  return (
                    <div
                      key={pane.sessionId}
                      data-testid={`sessions-pane-${pane.sessionId}`}
                      data-active={isActive ? "true" : undefined}
                      className={cn(
                        "relative flex min-w-0 items-start gap-2.5 p-3 first:rounded-t-xl last:rounded-b-xl",
                        isActive && "bg-wc-accent/5",
                      )}
                    >
                      {isActive && (
                        <span aria-hidden className="absolute inset-y-2 start-0 w-0.5 rounded-full bg-wc-accent" />
                      )}

                      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                        <div className="flex min-w-0 items-center gap-2">
                          <span
                            aria-hidden
                            className="w-3 shrink-0 text-center text-[10px] font-semibold tabular-nums text-wc-text-faint"
                          >
                            {index + 1}
                          </span>
                          <PaneColorDot
                            color={pane.headerColor}
                            interactive={!isMobile}
                            onSelect={(color) => { recolorPane(pane.sessionId, color); }}
                          />

                          {editingName === pane.sessionId ? (
                            <input
                              className="min-w-0 flex-1 rounded-lg border border-wc-accent bg-wc-surface-input px-2 py-1 text-sm text-wc-text-primary outline-none"
                              value={editValue}
                              autoFocus
                              onChange={(event) => { setEditValue(event.target.value); }}
                              onBlur={() => {
                                renamePane(pane.sessionId, editValue);
                                setEditingName(null);
                              }}
                              onKeyDown={(event) => {
                                if (event.key === "Enter") {
                                  renamePane(pane.sessionId, editValue);
                                  setEditingName(null);
                                } else if (event.key === "Escape") {
                                  setEditingName(null);
                                }
                              }}
                            />
                          ) : (
                            <button
                              className="min-w-0 flex-1 truncate text-start text-sm font-medium text-wc-text-secondary"
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
                          <div className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1.5 ps-5">
                            {isActive && (
                              <span className="inline-flex items-center rounded-full bg-wc-accent/15 px-2 py-0.5 text-[10px] font-medium text-wc-accent">
                                {t(strings.settings.sessionsSection.active)}
                              </span>
                            )}
                            {session.backend === "persistent" && (
                              <span className="inline-flex items-center rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] font-medium text-blue-300">
                                {t(strings.settings.sessionsSection.persistent)}
                              </span>
                            )}
                            <SessionPolicyControl
                              session={session}
                              interactive={!isMobile}
                              onPolicyChange={(id, mode, duration) => {
                                void handlePolicyChange(id, mode, duration);
                              }}
                            />
                          </div>
                        )}
                      </div>

                      <div className="flex shrink-0 items-center gap-0.5">
                        {isMobile ? (
                          <Button
                            data-testid={`sessions-pane-menu-${pane.sessionId}`}
                            variant="ghost"
                            size="icon"
                            shape="square"
                            className="h-9 w-9"
                            onClick={() => { setMenuPaneId(pane.sessionId); }}
                            aria-label={t(strings.settings.sessionsSection.actionsLabel)}
                            title={t(strings.settings.sessionsSection.actionsLabel)}
                          >
                            <MoreVertical className="h-4 w-4" />
                          </Button>
                        ) : (
                          <>
                            <Button
                              data-testid={`sessions-pane-up-${pane.sessionId}`}
                              variant="ghost"
                              size="icon"
                              shape="square"
                              className="h-8 w-8"
                              disabled={index === 0}
                              onClick={() => { movePane(pane.sessionId, -1); }}
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
                              onClick={() => { movePane(pane.sessionId, 1); }}
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
                              onClick={() => { focusPane(pane.sessionId); }}
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
                              onClick={() => { onDeleteSession(pane.sessionId); }}
                              title={t(strings.settings.sessionsSection.terminateSession)}
                            >
                              <Trash2 className="h-3.5 w-3.5 text-wc-error-detail" />
                            </Button>
                          </>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </SettingsList.Group>
      </SettingsList>

      {menuPane && (
        <BottomSheet
          open
          onOpenChange={(open) => { if (!open) closeMenu(); }}
          title={menuPane.name}
          closeLabel={t(strings.settings.closeAriaLabel)}
          testId={`sessions-pane-sheet-${menuPane.sessionId}`}
          avoidKeyboard
        >
          <div className="flex flex-col gap-1 pb-2">
            <div className="px-3 pb-2 pt-1">
              <div className="text-[11px] font-semibold uppercase tracking-[0.08em] text-wc-text-muted">
                {t(strings.settings.sessionsSection.color)}
              </div>
              <ColorSwatches
                className="mt-2"
                onSelect={(color) => { recolorPane(menuPane.sessionId, color); }}
              />
            </div>

            {menuSession && (
              <div className="px-3 pb-2">
                <div className="text-[11px] font-semibold uppercase tracking-[0.08em] text-wc-text-muted">
                  {t(strings.settings.sessionsSection.policyLabel)}
                </div>
                <NativeSelect
                  size="touch"
                  testId={`sessions-pane-sheet-policy-${menuPane.sessionId}`}
                  ariaLabel={t(strings.settings.sessionsSection.policyLabel)}
                  className="mt-2 w-full"
                  value={policyKey(menuSession.policy.mode, menuSession.policy.duration)}
                  onChange={(value) => {
                    const parsed = parsePolicySelection(value);
                    if (!parsed) return;
                    void handlePolicyChange(menuPane.sessionId, parsed.mode, parsed.duration);
                  }}
                  options={POLICY_OPTIONS.map((option) => ({
                    value: policyKey(option.mode, option.duration),
                    label: option.label,
                  }))}
                />
              </div>
            )}

            <SheetAction
              icon={ChevronUp}
              label={t(strings.settings.sessionsSection.moveUp)}
              testId={`sessions-pane-sheet-up-${menuPane.sessionId}`}
              disabled={menuIndex <= 0}
              onSelect={() => { movePane(menuPane.sessionId, -1); closeMenu(); }}
            />
            <SheetAction
              icon={ChevronDown}
              label={t(strings.settings.sessionsSection.moveDown)}
              testId={`sessions-pane-sheet-down-${menuPane.sessionId}`}
              disabled={menuIndex < 0 || menuIndex >= panes.length - 1}
              onSelect={() => { movePane(menuPane.sessionId, 1); closeMenu(); }}
            />
            <SheetAction
              icon={Focus}
              label={t(strings.settings.sessionsSection.focusPane)}
              testId={`sessions-pane-sheet-focus-${menuPane.sessionId}`}
              onSelect={() => { focusPane(menuPane.sessionId); closeMenu(); }}
            />
            <SheetAction
              icon={Trash2}
              label={t(strings.settings.sessionsSection.terminateSession)}
              testId={`sessions-pane-sheet-remove-${menuPane.sessionId}`}
              destructive
              onSelect={() => { onDeleteSession(menuPane.sessionId); closeMenu(); }}
            />
          </div>
        </BottomSheet>
      )}
    </>
  );
}
