// DOC: docs/reference/configuration.md#launcher-shortcuts
// DOC: docs/internal/SEAMS.md#1-entry-presentation
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Info,
  Loader2,
  Monitor,
  Play,
  TriangleAlert,
  XCircle,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Input } from "@vrooli/react-component-library/Input";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { strings } from "../consts/strings";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../consts/shortcuts";
import { shortcutsClient } from "../api/shortcuts";
import type { InstallOutcome } from "../api/capabilities";
import { BACKEND_OPTIONS } from "../consts/backend-options";
import { POLICY_OPTIONS, policyKey, parsePolicySelection } from "../consts/policy-options";
import type { BackendID, BackendOption, ExpirationPolicy, PolicyMode } from "../api/sessions";
import type { TargetCatalog, TerminalTarget, TerminalTargetState } from "../api/targets";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";
import MachinePicker from "./launcher/MachinePicker";
import GroupDestinationTrigger from "./launcher/GroupPicker";
import GroupModePanel, { type GroupCreationRequest } from "./launcher/GroupModePanel";
import AgentGridSection from "./launcher/AgentGridSection";
import { useAgentPreferences } from "./launcher/useAgentPreferences";
import { mergeAgentOrder } from "./launcher/agentGrid";
import { useUsageMode } from "./launcher/useUsageMode";
import { CONTEXT_LIMIT_CHOICES, withContextLimit, type ContextLimitChoice } from "./launcher/contextLimit";
import { useContextCompactionStore } from "../stores/useContextCompactionStore";
import UsageModeSheet, { clockTime } from "./launcher/UsageModeSheet";
import { modelSelectionClient } from "../api/modelSelection";
import { getBusinessAccountUsage, getBusinessAccounts, type BusinessAccount, type BusinessAccountUsageSummary } from "../api/monetization";
import type { GetRoleModelChoicesResponse, RoleModelChoice } from "@vrooli/proto-types/web-console/v1/modelselection/model_selection_pb";
import { withModelArgs } from "./launcher/AgentGridSection";

export type { TerminalTarget } from "../api/targets";

/** One session at a time, or a whole group in one trip through the dialog. */
type LauncherMode = "one-session" | "group";
type SelectedLaunch = { kind: "agent"; agentId: string; label: string; command: string; fullPowerModel?: string } | { kind: "shell" };
const LAUNCHER_DEFAULT_CHOICE = "__launcher_default__";
const VROOLI_CREDITS_SOURCE = "vrooli-credits";

function formatCredits(value: number | undefined): string {
  if (value === undefined) return "Unknown";
  if (value < 0) return "Unlimited";
  return new Intl.NumberFormat().format(value);
}

function roleChoiceKey(choice: RoleModelChoice): string {
  return JSON.stringify([choice.source, choice.model]);
}

// [REQ:P0-006a] Terminal Launch Flow UI
// [REQ:P0-006b] Configurable Shortcut Entries
// [REQ:P1-002b] Shortcut Profile Management UI
// [REQ:P1-013a] Remote target catalog and readiness

export interface LaunchOptions {
  command?: string;
  backend?: BackendID;
  policy?: { mode: PolicyMode; duration?: string };
  target?: TerminalTarget;
  workingDir?: string;
  businessAccountId?: string;
}

interface TerminalLauncherProps {
  open: boolean;
  onClose: () => void;
  onLaunch: (options: LaunchOptions) => void;
  /**
   * Every group the operator has, so the destination control can offer them.
   * Passing the list (rather than reading the store here) keeps the dialog a
   * pure component the tests can drive.
   */
  groups?: TabGroupMeta[];
  /**
   * The group this launcher was opened for — set when it was opened from a
   * group header's add control. Before this existed the pending group lived
   * in a ref the dialog never saw, so the dialog was structurally incapable
   * of stating its own destination.
   */
  pendingGroupId?: string | null;
  /** Change the destination from inside the dialog. */
  onDestinationChange?: (groupId: string | null) => void;
  /** Create a group by typing a name that matches none. Resolves to its id. */
  onCreateGroup?: (name: string) => Promise<string | null>;
  /** The appearance a new session will receive, named before launch. */
  appearance?: { headerColor: string; themeId: string; fontSize: number };
  /** Select a machine when the launcher was opened from its fleet card. */
  initialTarget?: TerminalTarget;
  /**
   * Create a whole group and its roles in one action.
   *
   * When absent the dialog offers no group mode, so a caller that has not
   * wired the handler cannot show a control that would do nothing.
   */
  onCreateGroupFromRoles?: (request: GroupCreationRequest) => void;
  shortcuts?: ShortcutEntry[];
  isCreating?: boolean;
  defaultBackend?: BackendID;
  defaultPolicy?: ExpirationPolicy;
  availableBackends?: BackendOption[];
  availableTargets?: TerminalTarget[];
  targetCatalog?: TargetCatalog;
  targetsLoading?: boolean;
  onRefreshTargets?: () => void | Promise<void>;
  /** Opens the machines surface, which owns linking and permissions. */
  onOpenMachines?: () => void;
  /** Starts the governed install action for a missing capability. */
  /**
   * Starts the governed install action for a missing capability. Returning the
   * promise lets the card show progress for the whole run rather than flashing
   * a state the operator cannot read.
   */
  onInstallCapability?: (capabilityID: string, target: TerminalTarget) => Promise<InstallOutcome>;
  /** Opens the shortcut editor, which owns the agent list this grid renders. */
  onEditShortcuts?: () => void;
  /** Opens the template editor, which owns the saved group recipes. */
  onEditTemplates?: () => void;
}

/**
 * Wire shortcuts to the UI shape. Proto string fields are never undefined, so
 * blank optionals are normalized here rather than at every read site.
 */
function decodeShortcuts(wire: readonly { label: string; command: string; description: string; agentId: string }[]): ShortcutEntry[] {
  return wire.map((entry) => ({
    label: entry.label,
    command: entry.command,
    description: entry.description || undefined,
    agentId: entry.agentId || undefined,
  }));
}

const localFallback: TerminalTarget = {
  id: "local",
  kind: "local",
  label: "This machine",
  available: true,
  state: "dispatchable",
  status: "LOCAL",
  online: true,
  readiness: [{ key: "local", label: "Aquila process", passed: true, detail: "Available on this machine" }],
};

function lastSeenCopy(target: TerminalTarget, neverSeenLabel: string): string | null {
  if (target.kind === "local") return null;
  if (!target.last_seen_at) return neverSeenLabel;
  try {
    return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(target.last_seen_at));
  } catch {
    return target.last_seen_at;
  }
}

export default function TerminalLauncher({
  open,
  onClose,
  onLaunch,
  shortcuts: shortcutsProp,
  isCreating = false,
  defaultBackend = "standard",
  defaultPolicy,
  availableBackends,
  availableTargets = [],
  targetCatalog,
  targetsLoading = false,
  onRefreshTargets,
  onOpenMachines,
  onInstallCapability,
  onEditShortcuts,
  onEditTemplates,
  groups = [],
  pendingGroupId = null,
  onDestinationChange,
  onCreateGroup,
  appearance,
  onCreateGroupFromRoles,
  initialTarget,
}: TerminalLauncherProps) {
  const { t, i18n } = useTranslation();
  const [customCommand, setCustomCommand] = useState("");
  const [workingDir, setWorkingDir] = useState("");
  const [apiShortcuts, setApiShortcuts] = useState<ShortcutEntry[] | null>(null);
  const [effectiveProfileId, setEffectiveProfileId] = useState<string | null>(null);
  const agentPreferences = useAgentPreferences(open && !shortcutsProp && effectiveProfileId !== null, effectiveProfileId);
  const [selectedBackend, setSelectedBackend] = useState<BackendID>(defaultBackend);
  const [selectedPolicyKey, setSelectedPolicyKey] = useState<string>(
    defaultPolicy ? policyKey(defaultPolicy.mode, defaultPolicy.duration) : "never",
  );
  const [optionsOpen, setOptionsOpen] = useState(false);
  const [selectedTarget, setSelectedTarget] = useState(initialTarget?.id ?? "local");
  // The destination is dialog state, seeded from the caller's pending group.
  // Seeding rather than mirroring is deliberate: once the operator changes it
  // here, a re-render from the caller must not silently move it back.
  const [destinationGroupId, setDestinationGroupId] = useState<string | null>(pendingGroupId);
  const [mode, setMode] = useState<LauncherMode>("one-session");
  const [appearanceOpen, setAppearanceOpen] = useState(false);
  const [selectedLaunch, setSelectedLaunch] = useState<SelectedLaunch | null>(null);
  const [modelPolicy, setModelPolicy] = useState<GetRoleModelChoicesResponse | null>(null);
  const [modelPolicyLoading, setModelPolicyLoading] = useState(false);
  const [modelPolicyError, setModelPolicyError] = useState(false);
  const [selectedModelKey, setSelectedModelKey] = useState("");
  const [modelOverrideExplicit, setModelOverrideExplicit] = useState(false);
  const [businessAccounts, setBusinessAccounts] = useState<BusinessAccount[]>([]);
  const [businessAccountsLoading, setBusinessAccountsLoading] = useState(false);
  const [businessAccountsError, setBusinessAccountsError] = useState(false);
  const [selectedBusinessAccountId, setSelectedBusinessAccountId] = useState("");
  const [businessAccountUsage, setBusinessAccountUsage] = useState<BusinessAccountUsageSummary | null>(null);
  const [businessAccountUsageLoading, setBusinessAccountUsageLoading] = useState(false);
  const [businessAccountUsageError, setBusinessAccountUsageError] = useState(false);

  // Reseed the destination each time the dialog opens, so opening it from a
  // different group header shows that group rather than the last one.
  useEffect(() => {
    if (open) {
      setDestinationGroupId(pendingGroupId);
      setSelectedTarget(initialTarget?.id ?? "local");
      setSelectedLaunch(null);
      setModelPolicy(null);
      setSelectedModelKey("");
      setModelOverrideExplicit(false);
      setBusinessAccounts([]);
      setBusinessAccountsError(false);
      setSelectedBusinessAccountId("");
      setBusinessAccountUsage(null);
      setBusinessAccountUsageError(false);
      // Always reopen on the single-session mode: it is the common case, and
      // landing in group mode because the last trip used it would surprise.
      setMode("one-session");
    }
  }, [initialTarget?.id, open, pendingGroupId]);

  useEffect(() => { setSelectedBackend(defaultBackend); }, [defaultBackend]);
  useEffect(() => {
    if (defaultPolicy) setSelectedPolicyKey(policyKey(defaultPolicy.mode, defaultPolicy.duration));
  }, [defaultPolicy]);

  useEffect(() => {
    if (!open || shortcutsProp) return;
    let cancelled = false;
    setEffectiveProfileId(null);
    shortcutsClient.getEffective({})
      .then((resp) => {
        if (cancelled) return;
        setApiShortcuts(decodeShortcuts(resp.shortcuts));
        setEffectiveProfileId(resp.profileId);
      })
      .catch(() => {
        if (!cancelled) { setApiShortcuts(null); setEffectiveProfileId(null); }
      });
    return () => { cancelled = true; };
  }, [open, shortcutsProp, t]);

  const shortcuts = shortcutsProp ?? apiShortcuts ?? DEFAULT_SHORTCUTS;
  const catalogTargets = targetCatalog?.targets ?? availableTargets;
  const targets = useMemo(() => {
    const catalogWithInitial = initialTarget && !catalogTargets.some((target) => target.id === initialTarget.id)
      ? [...catalogTargets, initialTarget]
      : catalogTargets;
    const local = catalogWithInitial.find((target) => target.id === "local") ?? localFallback;
    return [local, ...catalogWithInitial.filter((target) => target.id !== "local")];
  }, [catalogTargets, initialTarget]);
  const selected = targets.find((target) => target.id === selectedTarget) ?? targets[0] ?? localFallback;

  useEffect(() => {
    if (!open || selectedLaunch?.kind !== "agent" || !selectedLaunch.agentId) {
      setModelPolicy(null);
      setModelPolicyLoading(false);
      setModelPolicyError(false);
      setSelectedModelKey("");
      setModelOverrideExplicit(false);
      return;
    }
    let cancelled = false;
    setModelPolicy(null);
    setModelPolicyLoading(true);
    setModelPolicyError(false);
    setSelectedModelKey("");
    setModelOverrideExplicit(false);
    modelSelectionClient.getRoleModelChoices({ agentId: selectedLaunch.agentId, targetId: selected.id })
      .then((response) => { if (!cancelled) setModelPolicy(response); })
      .catch(() => { if (!cancelled) setModelPolicyError(true); })
      .finally(() => { if (!cancelled) setModelPolicyLoading(false); });
    return () => { cancelled = true; };
  }, [open, selected.id, selectedLaunch?.kind, selectedLaunch?.kind === "agent" ? selectedLaunch.agentId : undefined]);

  const modelChoices = modelPolicy?.choices ?? [];
  const modelSources = [...new Set(modelChoices.map((choice) => choice.source))];
  const defaultModelChoice = modelChoices.find((choice) => choice.isDefault) ?? modelChoices[0];
  const selectedChoice = modelOverrideExplicit
    ? modelChoices.find((choice) => roleChoiceKey(choice) === selectedModelKey)
    : defaultModelChoice;
  const selectedSource = selectedChoice?.source ?? modelSources[0] ?? "";
  const sourceChoices = modelChoices.filter((choice) => choice.source === selectedSource);
  const usesVrooliCredits = selectedLaunch?.kind === "agent" && selectedChoice?.source === VROOLI_CREDITS_SOURCE;
  useEffect(() => {
    if (!open || !usesVrooliCredits) {
      setBusinessAccounts([]);
      setBusinessAccountsLoading(false);
      setBusinessAccountsError(false);
      setSelectedBusinessAccountId("");
      return;
    }
    let cancelled = false;
    setBusinessAccountsLoading(true);
    setBusinessAccountsError(false);
    setSelectedBusinessAccountId("");
    void getBusinessAccounts().then(({ accounts }) => {
      if (cancelled) return;
      setBusinessAccounts(accounts ?? []);
      setSelectedBusinessAccountId(accounts?.[0]?.id ?? "");
    }).catch(() => {
      if (!cancelled) {
        setBusinessAccounts([]);
        setBusinessAccountsError(true);
      }
    }).finally(() => { if (!cancelled) setBusinessAccountsLoading(false); });
    return () => { cancelled = true; };
  }, [open, usesVrooliCredits]);
  useEffect(() => {
    if (!open || !usesVrooliCredits || !selectedBusinessAccountId) {
      setBusinessAccountUsage(null);
      setBusinessAccountUsageLoading(false);
      setBusinessAccountUsageError(false);
      return;
    }
    let cancelled = false;
    setBusinessAccountUsage(null);
    setBusinessAccountUsageLoading(true);
    setBusinessAccountUsageError(false);
    void getBusinessAccountUsage(selectedBusinessAccountId).then((usage) => {
      if (cancelled) return;
      if (usage.business_account_id !== selectedBusinessAccountId) {
        setBusinessAccountUsageError(true);
        return;
      }
      setBusinessAccountUsage(usage);
    }).catch(() => {
      if (!cancelled) setBusinessAccountUsageError(true);
    }).finally(() => { if (!cancelled) setBusinessAccountUsageLoading(false); });
    return () => { cancelled = true; };
  }, [open, selectedBusinessAccountId, usesVrooliCredits]);
  useEffect(() => {
    if (modelPolicy?.policyStatus !== "ready" || modelChoices.length === 0) return;
    setSelectedModelKey((current) => modelChoices.some((choice) => roleChoiceKey(choice) === current)
      ? current
      : roleChoiceKey(defaultModelChoice!));
  }, [defaultModelChoice, modelChoices, modelPolicy?.policyStatus]);

  // The mode file belongs to this machine, so it is only shown for local launches.
  const usageMode = useUsageMode(open && selected.kind === "local");
  const [usageSheetOpen, setUsageSheetOpen] = useState(false);
  // Per launch only: never stored, so it cannot be left on for the next session.
  const [contextLimit, setContextLimit] = useState<ContextLimitChoice>("standard");
  // Only a Save mode lowers the model; a Learn comparison has no settings and is shown in the sheet.
  const activeMode = usageMode.state.modes.find((item) => item.kind !== "learn");
  const statusLabels: Record<TerminalTargetState, string> = {
    dispatchable: t(strings.terminalLauncher.dispatchable),
    offline: t(strings.terminalLauncher.offline),
    "needs-update": t(strings.terminalLauncher.needsUpdate),
    unconfigured: t(strings.terminalLauncher.unconfigured),
    unavailable: t(strings.terminalLauncher.unavailable),
  };
  const statusLabelFor = (state: TerminalTargetState | undefined) => statusLabels[state ?? "unavailable"];

  useEffect(() => {
    if (!targets.some((target) => target.id === selectedTarget)) setSelectedTarget(targets[0]?.id ?? "local");
  }, [selectedTarget, targets]);

  const backends = availableBackends
    ? BACKEND_OPTIONS.filter((backend) => availableBackends.some((available) => available.id === backend.id && available.available))
    : BACKEND_OPTIONS;
  const showBackendSelector = backends.length > 1;
  const noBackendAvailable = backends.length === 0;
  const backendUnavailableReason = availableBackends?.find((backend) => !backend.available)?.reason ?? "No terminal backend is available on this host.";

  const destinationGroup = groups.find((group) => group.id === destinationGroupId) ?? null;

  const changeDestination = useCallback((groupId: string | null) => {
    setDestinationGroupId(groupId);
    onDestinationChange?.(groupId);
  }, [onDestinationChange]);

  const createDestination = useCallback((name: string) => {
    if (!onCreateGroup) return;
    // Server-first: the backend mints the id, then the dialog adopts it.
    // Fabricating one here would produce a destination that does not exist.
    void onCreateGroup(name).then((groupId) => {
      if (groupId) changeDestination(groupId);
    });
  }, [changeDestination, onCreateGroup]);

  // The grid only renders an install affordance when this prop is passed at
  // all (see the onInstall={...} guard below), so the missing-handler branch
  // reports "unknown" rather than inventing a success it did not observe.
  const installAgent = useCallback(async (agentID: string): Promise<InstallOutcome> => {
    if (!onInstallCapability) return { status: "unconfirmed" };
    return onInstallCapability(agentID, selected);
  }, [onInstallCapability, selected]);

  const selectAgent = useCallback((command: string, agentId: string) => {
    const shortcut = shortcuts.find((entry) => entry.agentId === agentId);
    setModelOverrideExplicit(false);
    setSelectedLaunch({ kind: "agent", agentId, label: shortcut?.label ?? agentId, command });
  }, [shortcuts]);
  const selectAgentFullPower = useCallback((command: string, agentId: string, model: string, args: string) => {
    const shortcut = shortcuts.find((entry) => entry.agentId === agentId);
    setModelOverrideExplicit(false);
    setSelectedLaunch({ kind: "agent", agentId, label: shortcut?.label ?? agentId, command: withModelArgs(command, args), fullPowerModel: model });
  }, [shortcuts]);

  const buildLaunchOptions = useCallback((command?: string): LaunchOptions => {
    const parsed = parsePolicySelection(selectedPolicyKey);
    return {
      command,
      target: selected,
      workingDir: workingDir.trim() || undefined,
      backend: selectedBackend !== defaultBackend ? selectedBackend : undefined,
      policy: parsed ?? undefined,
      businessAccountId: usesVrooliCredits ? selectedBusinessAccountId || undefined : undefined,
    };
  }, [defaultBackend, selected, selectedBackend, selectedPolicyKey, selectedBusinessAccountId, usesVrooliCredits, workingDir]);

  const submitSelectedLaunch = useCallback(() => {
    if (!selectedLaunch) return;
    let command = selectedLaunch.kind === "agent" ? selectedLaunch.command : undefined;
    if (command && selectedLaunch.kind === "agent" && !selectedLaunch.fullPowerModel && modelOverrideExplicit && modelPolicy?.policyStatus === "ready" && selectedChoice && modelPolicy.modelFlag) {
      command = withModelArgs(command, `${modelPolicy.modelFlag} ${selectedChoice.launchModel}`);
    }
    onLaunch(buildLaunchOptions(command === undefined ? undefined : withContextLimit(command, contextLimit)));
  }, [buildLaunchOptions, contextLimit, modelOverrideExplicit, modelPolicy, onLaunch, selectedChoice, selectedLaunch]);

  const handleLaunchCustom = useCallback(() => {
    if (!customCommand.trim() || !selected.available || noBackendAvailable) return;
    onLaunch(buildLaunchOptions(customCommand.trim()));
    setCustomCommand("");
  }, [buildLaunchOptions, customCommand, noBackendAvailable, onLaunch, selected.available]);

  const catalogMessage = targetCatalog?.status === "unconfigured"
    ? t(strings.terminalLauncher.unconfigured)
    : targetCatalog?.status === "configured-empty"
      ? t(strings.terminalLauncher.configuredEmpty)
      : targetCatalog?.status === "registry-error"
        ? t(strings.terminalLauncher.registryError)
        : targetCatalog?.message;

  // Built once and placed by whichever mode is showing: one-session pairs it
  // with the destination, group mode pairs it with the template.
  const machineControl = targetsLoading && targets.length <= 1
    ? (
      <div data-testid="launcher-target-loading" className="flex min-h-11 flex-1 basis-[13rem] items-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input/50 px-3 text-sm text-wc-text-muted">
        <Loader2 className="h-4 w-4 animate-spin text-wc-accent" aria-hidden />
        {t(strings.terminalLauncher.loadingTargets)}
      </div>
    )
    : (
      <MachinePicker
        targets={targets}
        selectedId={selected.id}
        onSelect={setSelectedTarget}
        onOpenMachines={onOpenMachines}
        catalogMessage={catalogMessage}
        catalogRecovery={targetCatalog?.recovery_action}
        onRefresh={onRefreshTargets}
        refreshing={targetsLoading}
      />
    );

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      closeLabel={t(strings.terminalLauncher.closeAriaLabel)}
      title={t(strings.terminalLauncher.newTerminal)}
      size="lg"
      avoidKeyboard
      // The dialog caps its own height against the host viewport and owns the
      // scroll region. The local max-height was measured against 100dvh, which
      // is the layout viewport rather than the one this app actually has, and
      // the inner scroller made a second one inside it.
      testId="terminal-launcher"
    >
      <div className="flex min-h-0 flex-col">
        <div className="min-h-0 flex-1 space-y-5 p-5">
          <header>
            <div className="flex items-center justify-between gap-3">
              <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-wc-accent">{t(strings.terminalLauncher.eyebrow)}</div>
              {selected.kind === "local" && (activeMode || usageMode.state.agents.length > 0) && (
                <button type="button" data-testid={activeMode ? "launcher-usage-chip" : "launcher-usage-start"} onClick={() => { setUsageSheetOpen(true); }}
                  className={activeMode
                    ? "flex min-h-11 items-center gap-1.5 rounded-full border border-wc-accent/40 bg-wc-accent/10 px-3 text-xs font-semibold text-wc-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent"
                    : "min-h-11 rounded-lg px-2 text-xs font-semibold text-wc-text-faint hover:text-wc-text-secondary focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent"}>
                  {activeMode
                    ? t(strings.launcher.usageModeChip, { until: activeMode.untilKind === "off" || !activeMode.expiresAt ? t(strings.launcher.usageModeUntilOffShort) : t(strings.launcher.usageModeUntil, { time: clockTime(activeMode.expiresAt, i18n.resolvedLanguage) }) })
                    : t(strings.launcher.usageModeStartButton)}
                </button>
              )}
            </div>
            <p className="mt-1 text-sm leading-5 text-wc-text-muted">{t(strings.terminalLauncher.description)}</p>
          </header>
          {selected.kind === "local" && mode === "one-session" && (
            <section aria-label={t(strings.launcher.contextLimitLabel)} className="space-y-1.5" data-testid="launcher-context-limit">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">{t(strings.launcher.contextLimitLabel)}</h3>
                <button type="button" data-testid="launcher-compact-idle" onClick={() => { useContextCompactionStore.getState().show(); }}
                  className="min-h-11 rounded-lg px-2 text-xs font-semibold text-wc-text-faint hover:text-wc-text-secondary focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent">{t(strings.launcher.compactIdleOpen)}</button>
              </div>
              <div role="radiogroup" aria-label={t(strings.launcher.contextLimitLabel)} className="flex flex-wrap gap-2">
                {CONTEXT_LIMIT_CHOICES.map((choice) => (
                  <button key={choice} type="button" role="radio" aria-checked={contextLimit === choice} data-testid={`launcher-context-limit-${choice}`}
                    onClick={() => { setContextLimit(choice); }}
                    className={`min-h-11 rounded-lg border px-3 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent ${contextLimit === choice ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary" : "border-wc-default text-wc-text-secondary"}`}>
                    {t(choice === "keep_long" ? strings.launcher.contextLimitKeepLong : choice === "tight" ? strings.launcher.contextLimitTight : strings.launcher.contextLimitStandard)}
                  </button>))}
              </div>
              <p className="text-[11px] text-wc-text-faint">{t(strings.launcher.contextLimitHint)}</p>
            </section>
          )}
          <UsageModeSheet open={usageSheetOpen} onClose={() => { setUsageSheetOpen(false); }} usageMode={usageMode} />

          {/* One session, or a whole group. The switch is only offered when
              the caller can actually create a group, and it is the library's
              tab strip rather than a local imitation of one — roving focus,
              scroll-into-view and the selected-tab contract come with it. */}
          {onCreateGroupFromRoles && (
            <Tabs
              mode="controlled"
              ariaLabel={t(strings.terminalLauncher.newTerminal)}
              active={mode}
              onChange={(next) => { setMode(next as LauncherMode); }}
              items={[
                { id: "one-session", label: t(strings.launcher.modeOneSession) },
                { id: "group", label: t(strings.launcher.modeGroup) },
              ]}
              itemTestId={(id) => (id === "group" ? "launcher-mode-group" : "launcher-mode-one-session")}
            />
          )}

          {noBackendAvailable && (
            <div data-testid="launcher-no-backend" role="alert" className="flex gap-3 rounded-xl border border-rose-400/25 bg-rose-400/10 p-3 text-sm text-rose-100">
              <XCircle className="mt-0.5 h-4 w-4 shrink-0 text-rose-300" aria-hidden />
              <span>{backendUnavailableReason}</span>
            </div>
          )}

          {/* Where it runs, on ONE line. Both are one-line decisions, and the
              dialog's vertical space belongs to the choice the operator
              actually came here to make. There is no section heading and no
              refresh button beside the row: the fleet's state, its refresh
              control and the link/manage actions all live inside the machine
              menu, where the list they describe is. */}
          {mode === "one-session" && (
            <div className="flex flex-wrap items-center gap-2">
              {machineControl}
              {/* The destination renders on EVERY open, including with no
                  pending group, where it reads "No group". A control that
                  appeared only sometimes would be a control the operator
                  never learns to look for. */}
              <GroupDestinationTrigger
                groups={groups}
                value={destinationGroupId}
                onChange={changeDestination}
                onCreate={createDestination}
              />
            </div>
          )}

          {/* One line, not a card. An unreachable machine is worth saying;
              it is not worth a quarter of the dialog. */}
          {!selected.available && (
            <p data-testid="launcher-target-unavailable" className="flex items-start gap-2 text-xs text-amber-300">
              <TriangleAlert className="mt-px h-3.5 w-3.5 shrink-0" aria-hidden />
              <span>
                {t(strings.terminalLauncher.targetUnavailable)}
                {(selected.recovery_action ?? selected.failure_rung) && (
                  <span className="text-amber-200/70"> · {selected.recovery_action ?? selected.failure_rung}</span>
                )}
              </span>
            </p>
          )}

          {mode === "group" && onCreateGroupFromRoles && (
            <GroupModePanel
              open={open}
              onCreate={onCreateGroupFromRoles}
              isCreating={isCreating}
              disabled={!selected.available || noBackendAvailable}
              machineSlot={machineControl}
              onCancel={onClose}
              onEditTemplates={onEditTemplates}
            />
          )}

          {mode === "one-session" && (
            <AgentGridSection
              key={selected.id}
              shortcuts={shortcuts}
              target={selected}
              usageAgents={selected.kind === "local" ? usageMode.state.agents : undefined}
              usageModes={usageMode.state.modes}
              selectionMode
              selectedAgentId={selectedLaunch?.kind === "agent" ? selectedLaunch.agentId : null}
              selectedShell={selectedLaunch?.kind === "shell"}
              onSelect={selectAgent}
              onSelectFullPower={selectAgentFullPower}
              onSelectShell={() => { setSelectedLaunch({ kind: "shell" }); }}
              disabled={isCreating || !selected.available || noBackendAvailable}
              onLaunch={(command) => { onLaunch(buildLaunchOptions(command === undefined ? command : withContextLimit(command, contextLimit))); }}
              onInstall={onInstallCapability ? installAgent : undefined}
              onEditShortcuts={onEditShortcuts}
              updatesEnabled={open}
              orderedAgentIds={agentPreferences.preferences?.orderedAgentIds}
              hiddenAgentIds={agentPreferences.preferences?.hiddenAgentIds}
              preferencesBusy={agentPreferences.busy}
              preferencesError={agentPreferences.error ? t(strings.launcher.preferencesError) : undefined}
              onRetryPreferences={shortcutsProp ? undefined : () => {
                void shortcutsClient.getEffective({}).then((resp) => {
                  setApiShortcuts(decodeShortcuts(resp.shortcuts));
                  setEffectiveProfileId(resp.profileId);
                  if (resp.profileId === effectiveProfileId) void agentPreferences.reload();
                }).catch(() => { void agentPreferences.reload(); });
              }}
              onReorder={agentPreferences.preferences ? (ids) => agentPreferences.update(mergeAgentOrder(agentPreferences.preferences?.orderedAgentIds ?? [], ids), agentPreferences.preferences?.hiddenAgentIds ?? []) : undefined}
              onVisibilityChange={agentPreferences.preferences ? (id, hidden) => {
                const current = agentPreferences.preferences;
                if (!current) return;
                const next = hidden ? [...current.hiddenAgentIds.filter((item) => item !== id), id] : current.hiddenAgentIds.filter((item) => item !== id);
                void agentPreferences.update(current.orderedAgentIds, next);
              } : undefined}
            />
          )}

          {mode === "one-session" && selectedLaunch && (
            <section data-testid="launcher-selected-launch" aria-label={t(strings.terminalLauncher.selectedLaunch)} className="space-y-3 rounded-xl border border-wc-accent/40 bg-wc-accent/5 p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 className="text-sm font-semibold text-wc-text-primary">
                  {selectedLaunch.kind === "shell" ? t(strings.terminalLauncher.emptyShell) : selectedLaunch.label}
                </h3>
                <button type="button" data-testid="launcher-clear-selection" onClick={() => { setSelectedLaunch(null); }}
                  className="min-h-11 rounded-lg px-3 text-xs font-semibold text-wc-accent hover:bg-wc-surface-input focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent">
                  {t(strings.terminalLauncher.clearLaunchSelection)}
                </button>
              </div>
              {selectedLaunch.kind === "agent" && (
                <div data-testid="launcher-model-selection" className="space-y-2">
                  {modelPolicyLoading ? (
                    <p role="status" className="text-xs text-wc-text-muted">{t(strings.terminalLauncher.modelChoicesLoading)}</p>
                  ) : modelPolicyError || !modelPolicy || modelPolicy.policyStatus === "unknown" ? (
                    <p role="status" className="text-xs text-wc-text-muted">{t(strings.terminalLauncher.modelChoicesUnknown)}</p>
                  ) : modelPolicy.policyStatus === "unsupported" ? (
                    <p role="status" className="text-xs text-wc-text-muted">{modelPolicy.detail || t(strings.terminalLauncher.modelChoicesUnsupported)}</p>
                  ) : modelPolicy.policyStatus === "stale" ? (
                    <p role="status" className="text-xs text-amber-200">{t(strings.terminalLauncher.modelChoicesStale)}</p>
                  ) : (
                    <>
                      {selectedLaunch.fullPowerModel && <p role="status" className="text-xs text-wc-text-secondary">{t(strings.terminalLauncher.usageModeFullPowerSelected, { model: selectedLaunch.fullPowerModel })}</p>}
                      {!selectedLaunch.fullPowerModel && modelSources.length > 1 ? (
                        <label className="flex min-h-11 flex-col gap-1 text-xs font-medium text-wc-text-secondary">
                          {t(strings.terminalLauncher.modelSource)}
                          <select data-testid="launcher-model-source" aria-label={t(strings.terminalLauncher.modelSource)} value={modelOverrideExplicit ? selectedSource : LAUNCHER_DEFAULT_CHOICE}
                            onChange={(event) => {
                              if (event.target.value === LAUNCHER_DEFAULT_CHOICE) { setModelOverrideExplicit(false); return; }
                              const next = modelChoices.find((choice) => choice.source === event.target.value);
                              if (next) { setSelectedModelKey(roleChoiceKey(next)); setModelOverrideExplicit(true); }
                            }}
                            className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent">
                            <option value={LAUNCHER_DEFAULT_CHOICE}>{t(strings.terminalLauncher.useCurrentLauncherDefault)}</option>
                            {modelSources.map((source) => {
                              const choice = modelChoices.find((item) => item.source === source);
                              return <option key={source} value={source}>{choice?.sourceLabel || source} ({source})</option>;
                            })}
                          </select>
                        </label>
                      ) : !selectedLaunch.fullPowerModel && selectedChoice ? (
                        <p data-testid="launcher-model-source" className="text-xs text-wc-text-secondary">{t(strings.terminalLauncher.modelSource)}: {selectedChoice.sourceLabel || selectedChoice.source} ({selectedChoice.source})</p>
                      ) : null}
                      {!selectedLaunch.fullPowerModel && <label className="flex min-h-11 flex-col gap-1 text-xs font-medium text-wc-text-secondary">
                        {t(strings.terminalLauncher.modelChoice)}
                        <select data-testid="launcher-model-choice" aria-label={t(strings.terminalLauncher.modelChoice)} value={modelOverrideExplicit && selectedChoice ? roleChoiceKey(selectedChoice) : LAUNCHER_DEFAULT_CHOICE}
                          onChange={(event) => {
                            if (event.target.value === LAUNCHER_DEFAULT_CHOICE) { setModelOverrideExplicit(false); return; }
                            setSelectedModelKey(event.target.value); setModelOverrideExplicit(true);
                          }}
                          className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent">
                          <option value={LAUNCHER_DEFAULT_CHOICE}>{t(strings.terminalLauncher.useCurrentLauncherDefault)}</option>
                          {sourceChoices.map((choice) => <option key={roleChoiceKey(choice)} value={roleChoiceKey(choice)}>{choice.model}</option>)}
                        </select>
                      </label>}
                      {usesVrooliCredits && (
                        <label className="flex min-h-11 flex-col gap-1 text-xs font-medium text-wc-text-secondary">
                          Billing account
                          {businessAccountsLoading ? (
                            <span role="status" data-testid="launcher-business-account-loading" className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 py-3 text-sm text-wc-text-muted">Loading business accounts…</span>
                          ) : businessAccountsError ? (
                            <span role="alert" data-testid="launcher-business-account-error" className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 py-3 text-sm text-rose-200">Business accounts are unavailable. Try selecting the Vrooli credits source again.</span>
                          ) : businessAccounts.length === 0 ? (
                            <span role="status" data-testid="launcher-business-account-empty" className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 py-3 text-sm text-wc-text-muted">No business accounts are available for this subscription.</span>
                          ) : (
                            <select data-testid="launcher-business-account" aria-label="Billing account" value={selectedBusinessAccountId}
                              onChange={(event) => { setSelectedBusinessAccountId(event.target.value); }}
                              className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent">
                              {businessAccounts.map((account) => <option key={account.id} value={account.id}>{account.display_name}</option>)}
                            </select>
                          )}
                          {businessAccountsLoading || businessAccountsError || businessAccounts.length === 0 ? null : businessAccountUsageLoading ? (
                            <span role="status" data-testid="launcher-business-account-usage-loading" className="text-xs font-normal text-wc-text-muted">Loading current AI credit usage…</span>
                          ) : businessAccountUsageError || !businessAccountUsage || businessAccountUsage.business_account_id !== selectedBusinessAccountId ? (
                            <span role="alert" data-testid="launcher-business-account-usage-error" className="text-xs font-normal text-rose-200">Current account usage is unavailable. Paid launch is disabled until it can be checked.</span>
                          ) : (
                            <span data-testid="launcher-business-account-usage" className="text-xs font-normal text-wc-text-muted">
                              {formatCredits(businessAccountUsage.remaining.ai_credits)} AI credits remaining · {formatCredits(businessAccountUsage.reserved.ai_credits)} reserved
                            </span>
                          )}
                        </label>
                      )}
                      <p className="text-[11px] text-wc-text-muted">{t(strings.terminalLauncher.modelAvailabilityUnknown)}</p>
                      {modelPolicy.provenance && <p className="break-words text-[11px] text-wc-text-faint">{t(strings.terminalLauncher.modelPolicyProvenance, { date: modelPolicy.observedAt || t(strings.terminalLauncher.unavailable), source: modelPolicy.provenance })}</p>}
                    </>
                  )}
                </div>
              )}
              <button type="button" data-testid="launcher-submit-selection" disabled={isCreating || !selected.available || noBackendAvailable || usesVrooliCredits && (businessAccountsLoading || businessAccountsError || !selectedBusinessAccountId || businessAccountUsageLoading || businessAccountUsageError || !businessAccountUsage || businessAccountUsage.business_account_id !== selectedBusinessAccountId || typeof businessAccountUsage.remaining.ai_credits !== "number" || businessAccountUsage.remaining.ai_credits === 0)}
                onClick={submitSelectedLaunch}
                className="min-h-11 w-full rounded-lg bg-wc-accent px-4 text-sm font-semibold text-wc-surface-base hover:brightness-110 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-wc-accent disabled:opacity-50">
                {selectedLaunch.kind === "shell" ? t(strings.terminalLauncher.launchSelectedShell) : t(strings.terminalLauncher.launchSelectedAgent, { name: selectedLaunch.label })}
              </button>
            </section>
          )}

          {mode === "one-session" && !selectedLaunch && (
          <section className="space-y-2" aria-label={t(strings.terminalLauncher.customCommand)}>
            {/* The prompt marker and the launch control both live inside the
                field's own border, so the row reads as one command bar. The
                `$` used to be an absolutely-positioned span with a hand-tuned
                `ps-7` behind it to clear it; the adornment slot supplies its
                own gutter and collapses the input's padding on that side. */}
            <InputGroup
              testId="launcher-custom-group"
              size="lg"
            >
              <InputGroup.Adornment side="leading" className="font-mono text-sm">$</InputGroup.Adornment>
              <InputGroup.Field>
                <Input data-testid="launcher-custom-input" type="text" aria-label={t(strings.terminalLauncher.customCommand)} value={customCommand} onChange={(event) => { setCustomCommand(event.target.value); }} onKeyDown={(event) => { if (event.key === "Enter") handleLaunchCustom(); }} placeholder={t(strings.terminalLauncher.commandPlaceholder)} className="font-mono text-sm text-wc-text-primary placeholder:font-sans placeholder:text-wc-text-faint" />
              </InputGroup.Field>
              <InputGroup.Segment side="trailing" emphasis="solid" testId="launcher-custom-launch" aria-label={t(strings.terminalLauncher.launch)} title={t(strings.terminalLauncher.launch)} onClick={handleLaunchCustom} disabled={isCreating || !customCommand.trim() || !selected.available || noBackendAvailable}>
                <Play aria-hidden className="h-4 w-4" />
              </InputGroup.Segment>
            </InputGroup>
          </section>
          )}

          {/* Two disclosures on one line, because neither is a section: they
              are the settings you occasionally reach for, and giving each a
              bordered card of its own is what made the dialog feel padded. */}
          <section className="space-y-2">
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
              {appearance && (
                <button type="button" data-testid="launcher-appearance-toggle" className="flex min-h-11 items-center gap-1.5 text-xs text-wc-text-faint transition hover:text-wc-text-secondary" onClick={() => { setAppearanceOpen((value) => !value); }} aria-expanded={appearanceOpen}>
                  {appearanceOpen ? <ChevronDown className="h-3.5 w-3.5 shrink-0" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5 shrink-0" aria-hidden />}
                  <span className="font-medium text-wc-text-secondary">{t(strings.launcher.appearance)}</span>
                  <span className="h-2.5 w-2.5 shrink-0 rounded-full border border-wc-default" style={{ backgroundColor: destinationGroup?.color ?? appearance.headerColor }} aria-hidden />
                  <span className="truncate">
                    {t(strings.launcher.appearanceSummary, {
                      color: destinationGroup ? t(strings.launcher.appearanceFromGroup) : t(strings.launcher.appearanceDefault),
                      theme: appearance.themeId,
                      size: appearance.fontSize,
                    })}
                  </span>
                </button>
              )}
              <button type="button" data-testid="launcher-options-toggle" className="flex min-h-11 items-center gap-1.5 text-xs text-wc-text-faint transition hover:text-wc-text-secondary" onClick={() => { setOptionsOpen((value) => !value); }} aria-expanded={optionsOpen}>
                {optionsOpen ? <ChevronDown className="h-3.5 w-3.5 shrink-0" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5 shrink-0" aria-hidden />}
                <span className="font-medium text-wc-text-secondary">{t(strings.terminalLauncher.sessionOptions)}</span>
              </button>
            </div>

            {appearance && appearanceOpen && (
              <dl data-testid="launcher-appearance" className="grid grid-cols-3 gap-2 rounded-lg border border-wc-default bg-wc-surface-base/40 px-3 py-2 text-[11px]">
                <div>
                  <dt className="text-wc-text-faint">{t(strings.appearance.headerColorHeading)}</dt>
                  <dd className="truncate text-wc-text-secondary">{destinationGroup ? destinationGroup.name : appearance.headerColor}</dd>
                </div>
                <div>
                  <dt className="text-wc-text-faint">{t(strings.appearance.terminalThemeHeading)}</dt>
                  <dd className="truncate text-wc-text-secondary">{appearance.themeId}</dd>
                </div>
                <div>
                  <dt className="text-wc-text-faint">{t(strings.appearance.fontSizeHeading)}</dt>
                  <dd className="text-wc-text-secondary">{appearance.fontSize}px</dd>
                </div>
              </dl>
            )}

            {optionsOpen && (
              <div className="space-y-3 rounded-xl border border-wc-default bg-wc-surface-base/50 p-4">
                <div className="space-y-1.5"><label htmlFor="launcher-working-dir" className="text-xs font-medium text-wc-text-secondary">{t(strings.terminalLauncher.workingDirectory)}</label><input id="launcher-working-dir" data-testid="launcher-working-dir" value={workingDir} onChange={(event) => { setWorkingDir(event.target.value); }} placeholder={t(strings.terminalLauncher.workingDirectoryPlaceholder)} className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary outline-none placeholder:text-wc-text-faint focus:border-wc-accent" /></div>
                <div className="space-y-1.5"><label htmlFor="launcher-target-select" className="text-xs font-medium text-wc-text-secondary">{t(strings.terminalLauncher.locations)}</label><select id="launcher-target-select" data-testid="launcher-target-select" className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-secondary outline-none focus:border-wc-accent" value={selectedTarget} onChange={(event) => { setSelectedTarget(event.target.value); }}>{targets.map((target) => <option key={target.id} value={target.id}>{target.label}{target.available ? "" : ` — ${statusLabelFor(target.state)}`}</option>)}</select></div>
                {showBackendSelector && <div className="flex items-center gap-2"><label htmlFor="launcher-backend-select" className="text-xs text-wc-text-secondary">{t(strings.terminalLauncher.backendLabel)}</label><select id="launcher-backend-select" data-testid="launcher-backend-select" className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-secondary outline-none focus:border-wc-accent" value={selectedBackend} onChange={(event) => { setSelectedBackend(event.target.value as BackendID); }}>{backends.map((backend) => <option key={backend.id} value={backend.id}>{backend.label}</option>)}</select></div>}
                <div className="flex items-center gap-2"><label htmlFor="launcher-timeout-select" className="text-xs text-wc-text-secondary">{t(strings.terminalLauncher.timeoutLabel)}</label><select id="launcher-timeout-select" data-testid="launcher-timeout-select" className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-secondary outline-none focus:border-wc-accent" value={selectedPolicyKey} onChange={(event) => { setSelectedPolicyKey(event.target.value); }}>{POLICY_OPTIONS.map((option) => <option key={policyKey(option.mode, option.duration)} value={policyKey(option.mode, option.duration)}>{option.label}</option>)}</select></div>
                <div className="flex items-start gap-2 text-xs text-wc-text-faint"><Info className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden /><span>{selectedBackend === "persistent" ? t(strings.terminalLauncher.persistentHint) : t(strings.terminalLauncher.advancedHint)}</span></div>
              </div>
            )}
          </section>

        </div>
        <footer className="shrink-0 border-t border-wc-default bg-wc-surface-raised/95 px-5 py-3 text-xs text-wc-text-faint">
          {isCreating ? <div data-testid="launcher-creating" className="flex items-center justify-center gap-2 text-sm text-wc-text-muted"><Loader2 className="h-4 w-4 animate-spin" aria-hidden />{t(strings.terminalLauncher.creating)}</div> : selected.kind !== "local" && selected.available ? <div className="flex items-center gap-2"><Monitor className="h-3.5 w-3.5" aria-hidden /><span>{selected.os && selected.arch ? `${selected.os}/${selected.arch}` : selected.label}</span>{lastSeenCopy(selected, t(strings.terminalLauncher.neverSeen)) && <span className="ms-auto">{t(strings.terminalLauncher.lastSeen)}: {lastSeenCopy(selected, t(strings.terminalLauncher.neverSeen))}</span>}</div> : null}
        </footer>
      </div>
    </ResponsiveDialog>
  );
}
