// DOC: docs/reference/configuration.md#launcher-shortcuts
// DOC: docs/internal/SEAMS.md#1-entry-presentation
/**
 * The New session drawer. The order is the operator's: where it runs, then what
 * kind of session, then the settings for that kind, then launch. One set of data
 * hooks feeds two compositions: a sheet in sequence on a phone, and settings
 * beside the choice on a desktop.
 */
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { Loader2, MoreHorizontal, TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Alert } from "@vrooli/react-component-library/Alert";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Menu, MenuContent, MenuItem, MenuTrigger } from "@vrooli/react-component-library/Menu";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { strings } from "../consts/strings";
import type { ShortcutEntry } from "../consts/shortcuts";
import type { InstallOutcome } from "../api/capabilities";
import { BACKEND_OPTIONS } from "../consts/backend-options";
import { policyKey, parsePolicySelection } from "../consts/policy-options";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { formatTokens } from "../lib/localeFormat";
import type { AgentLaunchIntent, BackendID, BackendOption, ExpirationPolicy, PolicyMode } from "../api/sessions";
import type { TargetCatalog, TerminalTarget } from "../api/targets";
import { useContextCompactionStore } from "../stores/useContextCompactionStore";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";
import { AgentRail, AgentRailHeader } from "./launcher/AgentRail";
import { buildAgentGrid, cardLaunches, cardSelectable, defaultAgentID, mergeAgentOrder, orderAgents } from "./launcher/agentGrid";
import { handleRadioKeys } from "./launcher/radioKeys";
import { harnessAction, harnessNeedsAction, harnessState } from "./launcher/agentStatus";
import GroupModePanel, { type GroupCreationRequest } from "./launcher/GroupModePanel";
import { HarnessSettings } from "./launcher/HarnessSettings";
import { LaunchExtras } from "./launcher/LaunchExtras";
import { LaunchFooter } from "./launcher/LaunchFooter";
import MachinePicker from "./launcher/MachinePicker";
import { ManageAgents, type ManagePage } from "./launcher/ManageAgents";
import { ShellPanel } from "./launcher/ShellPanel";
import { useAgentInstalls } from "./launcher/useAgentInstalls";
import { useAgentPreferences } from "./launcher/useAgentPreferences";
import { useAgentUpdates } from "./launcher/useAgentUpdates";
import { useAgentUsage } from "./launcher/useAgentUsage";
import { useBillingAccount } from "./launcher/useBillingAccount";
import { useEffectiveShortcuts } from "./launcher/useEffectiveShortcuts";
import { declaredWindowsFor, useHarnesses } from "./launcher/useHarnesses";
import { choiceBlocked, limitOptions, modelLabel, useModelPolicy } from "./launcher/useModelPolicy";
import UsageModeSheet from "./launcher/UsageModeSheet";
import { useUsageMode } from "./launcher/useUsageMode";
import { reservedUsageRows } from "./launcher/usageWindows";

export type { TerminalTarget } from "../api/targets";

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
  /** Settings for an agent launch. The server composes them onto `command`. */
  agentLaunch?: AgentLaunchIntent;
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
  /**
   * Starts the governed install action for a missing capability. Returning the
   * promise lets Manage show progress for the whole run rather than flashing a
   * state the operator cannot read.
   */
  onInstallCapability?: (capabilityID: string, target: TerminalTarget) => Promise<InstallOutcome>;
  /** Opens the shortcut editor, which owns the agent list this grid renders. */
  onEditShortcuts?: () => void;
  /** Opens the template editor, which owns the saved group recipes. */
  onEditTemplates?: () => void;
}

type SessionType = "agent" | "shell" | "group";

/** Says the panel continues under the footer. Over the padding at the end, it paints nothing. */
function ScrollFade() {
  return <div aria-hidden className="pointer-events-none sticky -bottom-px z-[1] -mb-px h-6 bg-gradient-to-t from-wc-surface-raised to-transparent" />;
}

const EMPTY_IDS: readonly string[] = [];

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
  const isDesktop = useMediaQuery("(min-width: 48rem)");
  const [sessionType, setSessionType] = useState<SessionType | null>(null);
  const [customCommand, setCustomCommand] = useState("");
  const [workingDir, setWorkingDir] = useState("");
  const [selectedBackend, setSelectedBackend] = useState<BackendID>(defaultBackend);
  const [selectedPolicyKey, setSelectedPolicyKey] = useState<string>(defaultPolicy ? policyKey(defaultPolicy.mode, defaultPolicy.duration) : "never");
  const [selectedTarget, setSelectedTarget] = useState(initialTarget?.id ?? "local");
  // The destination is dialog state, seeded from the caller's pending group.
  // Seeding rather than mirroring is deliberate: once the operator changes it
  // here, a re-render from the caller must not silently move it back.
  const [destinationGroupId, setDestinationGroupId] = useState<string | null>(pendingGroupId);
  const [pickedAgentId, setPickedAgentId] = useState<string | null>(null);
  // Per launch only: never stored, so neither can be left on for the next session.
  const [modelSettings, setModelSettings] = useState<{ identity: string; contextLimit: number | null; fullPower: boolean } | null>(null);
  const [usageSheet, setUsageSheet] = useState<{ agentId?: string } | null>(null);
  const [managePage, setManagePage] = useState<ManagePage>(null);
  const [footerHost, setFooterHost] = useState<HTMLElement | null>(null);
  const manageButton = useRef<HTMLButtonElement>(null);
  const drawerRoot = useRef<HTMLDivElement>(null);

  // Every visit waits for an explicit type choice. The destination follows
  // this open's caller without inheriting an agent or shell launch mode.
  useEffect(() => {
    if (!open) {
      setPickedAgentId(null);
      setModelSettings(null);
      setSessionType(null);
      setManagePage(null);
      return;
    }
    setDestinationGroupId(pendingGroupId);
    setSelectedTarget(initialTarget?.id ?? "local");
    setManagePage(null);
  }, [initialTarget?.id, open, pendingGroupId]);

  useEffect(() => { setSelectedBackend(defaultBackend); }, [defaultBackend]);
  useEffect(() => {
    if (defaultPolicy) setSelectedPolicyKey(policyKey(defaultPolicy.mode, defaultPolicy.duration));
  }, [defaultPolicy]);

  const agentModeOpen = open && sessionType === "agent";
  const { shortcuts, profileId, fetchEffective } = useEffectiveShortcuts(agentModeOpen, shortcutsProp);
  const agentPreferences = useAgentPreferences(agentModeOpen && !shortcutsProp && profileId !== null, profileId);
  const prefs = agentPreferences.preferences;
  const hiddenAgentIds = prefs?.hiddenAgentIds ?? EMPTY_IDS;

  const catalogTargets = targetCatalog?.targets ?? availableTargets;
  const targets = useMemo(() => {
    const catalogWithInitial = initialTarget && !catalogTargets.some((target) => target.id === initialTarget.id)
      ? [...catalogTargets, initialTarget]
      : catalogTargets;
    const local = catalogWithInitial.find((target) => target.id === "local") ?? {
      id: "local",
      kind: "local" as const,
      label: t(strings.terminalLauncher.thisMachine),
      available: true,
      state: "dispatchable" as const,
      status: "LOCAL",
      online: true,
    };
    return [local, ...catalogWithInitial.filter((target) => target.id !== "local")] as [TerminalTarget, ...TerminalTarget[]];
  }, [catalogTargets, initialTarget, t]);
  const selected = targets.find((target) => target.id === selectedTarget) ?? targets[0];

  useEffect(() => {
    if (!targets.some((target) => target.id === selectedTarget)) setSelectedTarget(targets[0].id);
  }, [selectedTarget, targets]);

  const backends = availableBackends
    ? BACKEND_OPTIONS.filter((backend) => availableBackends.some((available) => available.id === backend.id && available.available))
    : BACKEND_OPTIONS;
  const noBackendAvailable = backends.length === 0;
  const backendUnavailableReason = availableBackends?.find((backend) => !backend.available)?.reason || t(strings.terminalLauncher.noBackend);
  const launchBlocked = isCreating || !selected.available || noBackendAvailable;

  const installAgent = useCallback(async (agentID: string): Promise<InstallOutcome> => {
    if (!onInstallCapability) return { status: "unconfirmed" };
    return onInstallCapability(agentID, selected);
  }, [onInstallCapability, selected]);
  const readyAgentKey = useMemo(() => buildAgentGrid({ readiness: selected.readiness, shortcuts }).agents.filter((card) => card.state === "ready").map((card) => card.agentID).join(","), [selected.readiness, shortcuts]);
  const installs = useAgentInstalls(selected.id, onInstallCapability ? installAgent : undefined, readyAgentKey);
  const grid = useMemo(() => buildAgentGrid({ readiness: selected.readiness, shortcuts, installing: installs.installing }), [installs.installing, selected.readiness, shortcuts]);
  const agents = useMemo(() => orderAgents(grid.agents, { orderMode: prefs?.orderMode, orderedAgentIds: prefs?.orderedAgentIds, lastLaunchedAt: prefs?.lastLaunchedAt }), [grid.agents, prefs?.lastLaunchedAt, prefs?.orderMode, prefs?.orderedAgentIds]);
  const railCards = useMemo(() => agents.filter((card) => cardLaunches(card) && !hiddenAgentIds.includes(card.agentID)), [agents, hiddenAgentIds]);
  const offered = useMemo(() => railCards.filter((card) => cardSelectable(card, selected.kind)), [railCards, selected.kind]);

  // A harness is always selected. Nothing is stored for the default, so a hidden
  // pick falls back on its own, and last-launched applies once preferences arrive.
  const selectedCard = offered.find((card) => card.agentID === pickedAgentId)
    ?? offered.find((card) => card.agentID === defaultAgentID(offered, prefs?.lastLaunchedAt))
    ?? null;
  const selectedAgentId = selectedCard?.agentID ?? null;

  const harnesses = useHarnesses(agentModeOpen, selected.id);
  const harness = selectedAgentId ? harnesses.harnesses[selectedAgentId] : undefined;
  const declaredWindows = useCallback((agentId: string) => declaredWindowsFor(harnesses, agentId), [harnesses]);

  const usageState = useAgentUsage(agentModeOpen && selected.available, selected.id, railCards.map((card) => card.agentID).join(","));
  const updateState = useAgentUpdates(agentModeOpen && selected.available, selected.id, agents.filter((card) => card.state !== "not-applicable").map((card) => card.agentID).join(","));

  const reportedSlots = reservedUsageRows(railCards.map((card) => declaredWindows(card.agentID)), railCards.map((card) => usageState.usages[card.agentID]));
  // Resolving descriptors or hiding a card must not shrink the rail under a
  // thumb. A new drawer visit or machine can begin with its own row budget.
  const [usageSpace, setUsageSpace] = useState({ targetId: selected.id, open, slots: reportedSlots });
  const slots = usageSpace.targetId === selected.id && usageSpace.open === open ? Math.max(usageSpace.slots, reportedSlots) : reportedSlots;
  if (slots !== usageSpace.slots || usageSpace.targetId !== selected.id || usageSpace.open !== open) setUsageSpace({ targetId: selected.id, open, slots });
  const needsAction = useCallback((agentId: string) => {
    const card = agents.find((item) => item.agentID === agentId);
    const state = card && harnessState(card, updateState.updates[agentId], updateState.loading);
    return state !== undefined && harnessNeedsAction(state) && (state !== "update_ready" || harnessAction(state, updateState.updates[agentId]) === "update");
  }, [agents, updateState.loading, updateState.updates]);
  const actionCount = agents.filter((card) => needsAction(card.agentID)).length;

  const [billingAccountId, setBillingAccountId] = useState("");
  const model = useModelPolicy(agentModeOpen, selectedAgentId, selected.id, selectedCard?.command ?? "", billingAccountId);
  const billing = useBillingAccount(model.billingProbeReady, billingAccountId, setBillingAccountId);
  const currentModelSettings = modelSettings?.identity === model.identity ? modelSettings : undefined;
  const contextLimit = currentModelSettings?.contextLimit ?? null;
  const fullPower = currentModelSettings?.fullPower ?? false;
  const setContextLimit = (value: number | null) => { setModelSettings({ identity: model.identity, contextLimit: value, fullPower }); };
  const setFullPower = (value: boolean) => { setModelSettings({ identity: model.identity, contextLimit, fullPower: value }); };

  // The mode file belongs to this machine, so it is only shown for local launches.
  const usageMode = useUsageMode(open && selected.kind === "local");
  const saveAgent = selected.kind === "local" ? usageMode.state.agents.find((agent) => agent.agentId === selectedAgentId) : undefined;
  // Only a Save mode lowers the model; a Learn comparison has no settings and is shown in the sheet.
  const saveMode = usageMode.state.modes.find((mode) => mode.kind !== "learn" && mode.id === saveAgent?.modeId);

  // A usage mode reports the launch name; the list knows what to call it.
  const modeModel = model.loading || model.failed ? undefined : fullPower ? saveAgent?.usualModel : saveAgent?.model;
  const modeChoice = model.choices.find((choice) => choice.model === modeModel && !choice.metered)
    ?? model.choices.find((choice) => choice.model === modeModel);
  const effectiveChoice = model.explicitChoice ?? (modeModel ? modeChoice : model.selectedChoice);
  const modelBlocked = !model.literal && Boolean(effectiveChoice && choiceBlocked(effectiveChoice, model.stale));
  const effectiveModelName = model.explicitChoice?.displayName ?? modelLabel(model, modeModel) ?? model.defaultChoice?.displayName;
  // A limit is offered per model, so one chosen for another model no longer applies.
  const appliedLimit = contextLimit !== null && !model.literal && limitOptions(effectiveChoice).includes(contextLimit) ? contextLimit : null;

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

  const buildLaunchOptions = (command?: string, agentLaunch?: AgentLaunchIntent): LaunchOptions => ({
    command,
    target: selected,
    workingDir: workingDir.trim() || undefined,
    backend: selectedBackend !== defaultBackend ? selectedBackend : undefined,
    policy: parsePolicySelection(selectedPolicyKey) ?? undefined,
    businessAccountId: sessionType === "agent" && model.usesVrooliCredits ? billing.accountId || undefined : undefined,
    agentLaunch,
  });

  // The launch carries what was chosen and the server composes the command. A
  // command the operator wrote stays literal, so nothing is sent with it.
  const launchAgent = () => {
    if (!selectedCard || launchBlocked || (model.usesVrooliCredits && billing.blocked) || modelBlocked || model.loading) return;
    const chosenModel = model.failed ? undefined : fullPower ? saveAgent?.usualModel : model.explicitChoice?.model;
    const hasSettings = !model.literal && (chosenModel !== undefined || appliedLimit !== null);
    onLaunch(buildLaunchOptions(selectedCard.command, hasSettings ? {
      agent_id: selectedCard.agentID,
      model: chosenModel,
      provider: model.explicitChoice?.source,
      max_credits: model.usesVrooliCredits ? billing.maxCredits : undefined,
      context_limit_tokens: appliedLimit ?? undefined,
    } : undefined));
    agentPreferences.recordLaunch(selectedCard.agentID);
  };
  const launchShell = () => {
    if (launchBlocked) return;
    onLaunch(buildLaunchOptions(customCommand.trim() || undefined));
    setCustomCommand("");
  };

  // A harness is already selected, so Enter launches it unless focus is on a control.
  const onDrawerKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "Enter" || event.target !== event.currentTarget || launchBlocked) return;
    if (sessionType === "agent") launchAgent();
    else if (sessionType === "shell") launchShell();
  };

  const retryPreferences = shortcutsProp ? undefined : () => {
    void fetchEffective().then((id) => { if (id === profileId) void agentPreferences.reload(); }).catch(() => { void agentPreferences.reload(); });
  };

  const catalogMessage = targetCatalog?.status === "unconfigured"
    ? t(strings.terminalLauncher.unconfigured)
    : targetCatalog?.status === "configured-empty"
      ? t(strings.terminalLauncher.configuredEmpty)
      : targetCatalog?.status === "registry-error"
        ? t(strings.terminalLauncher.registryError)
        : targetCatalog?.message;

  const locale = i18n.resolvedLanguage;
  const summaryParts = [selected.label];
  if (sessionType === "shell") summaryParts.push(t(strings.terminalLauncher.summaryNoAgent));
  else {
    summaryParts.push(model.literal ? t(strings.terminalLauncher.modelSetByCommand) : effectiveModelName ?? t(strings.terminalLauncher.summaryDefaultModel));
    if (appliedLimit !== null) summaryParts.push(t(strings.launcher.summaryCompacts, { limit: formatTokens(appliedLimit, locale) }));
    if (saveAgent?.model && !model.literal && !fullPower && !model.explicitChoice) summaryParts.push(t(strings.launcher.summarySaving));
  }
  if (destinationGroup) summaryParts.push(destinationGroup.name);

  const headerActions = (
    <div className="flex items-center gap-1.5">
      {targetsLoading && targets.length <= 1 ? (
        <div data-testid="launcher-target-loading" className="flex min-h-11 items-center gap-2 rounded-[var(--radius-control)] border border-wc-default bg-wc-surface-input px-3 text-[13px] text-wc-text-muted">
          <Loader2 className="h-3.5 w-3.5 animate-spin text-wc-accent" aria-hidden />
          {t(strings.terminalLauncher.loadingTargets)}
        </div>
      ) : (
        <MachinePicker targets={targets} selectedId={selected.id} onSelect={setSelectedTarget} onOpenMachines={onOpenMachines} catalogMessage={catalogMessage}
          catalogRecovery={targetCatalog?.recovery_action} onRefresh={onRefreshTargets} refreshing={targetsLoading} />
      )}
      {sessionType === "agent" && selected.kind === "local" && (
        <Menu>
          {/* The trigger's own button styling is for labelled triggers; this one
              wears the library's icon button, shaped like the machine control beside it. */}
          <MenuTrigger {...{ asChild: true }}>
            <IconButton data-testid="launcher-overflow" aria-label={t(strings.terminalLauncher.moreActions)} shape="rounded" size="lg">
              <MoreHorizontal aria-hidden />
            </IconButton>
          </MenuTrigger>
          <MenuContent>
            <MenuItem onSelect={() => { useContextCompactionStore.getState().show(); }}>{t(strings.launcher.compactIdleOpen)}</MenuItem>
            {usageMode.state.agents.length > 0 && <MenuItem onSelect={() => { setUsageSheet({}); }}>{t(strings.launcher.usageModeSettings)}</MenuItem>}
          </MenuContent>
        </Menu>
      )}
    </div>
  );

  const sessionTypes: { id: SessionType; label: string }[] = [
    { id: "agent", label: t(strings.terminalLauncher.typeAgent) },
    { id: "shell", label: t(strings.terminalLauncher.typeShell) },
  ];
  if (onCreateGroupFromRoles) sessionTypes.push({ id: "group", label: t(strings.terminalLauncher.typeGroup) });
  const selectSessionType = (next: SessionType) => { setSessionType(next); };
  const typeTabs = (
    <div role="radiogroup" aria-label={t(strings.terminalLauncher.sessionType)} className="flex flex-wrap gap-2"
      onKeyDown={(event) => { handleRadioKeys(event, (index) => { const next = sessionTypes[index]; if (next) selectSessionType(next.id); }); }}>
      {sessionTypes.map((item, index) => {
        const checked = sessionType === item.id;
        return (
          <button key={item.id} type="button" role="radio" aria-checked={checked} tabIndex={checked || (sessionType === null && index === 0) ? 0 : -1}
            data-testid={`launcher-mode-${item.id}`}
            className={`rounded-md border px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-focus ${checked ? "border-wc-focus bg-wc-surface-raised text-wc-text" : "border-wc-default text-wc-text hover:bg-wc-surface-raised"}`}
            onClick={(event) => { selectSessionType(item.id); event.currentTarget.focus(); }}>
            {item.label}
          </button>
        );
      })}
    </div>
  );

  const alerts = ((sessionType === "agent" && noBackendAvailable) || (sessionType !== null && !selected.available)) && (
    <div className="grid gap-2 px-4 pt-2">
      {noBackendAvailable && (
        <div data-testid="launcher-no-backend"><Alert tone="danger" title={backendUnavailableReason} /></div>
      )}
      {!selected.available && (
        <p data-testid="launcher-target-unavailable" className="flex items-start gap-2 text-xs text-amber-300">
          <TriangleAlert className="mt-px h-3.5 w-3.5 shrink-0" aria-hidden />
          <span>
            {t(strings.terminalLauncher.targetUnavailable)}
            {(selected.recovery_action ?? selected.failure_rung) && <span className="text-amber-200/70"> · {selected.recovery_action ?? selected.failure_rung}</span>}
          </span>
        </p>
      )}
    </div>
  );

  const agentChoice = (
    <section aria-labelledby="launcher-agents-heading">
      <AgentRailHeader actionCount={actionCount} onManage={() => { setManagePage("list"); }} manageRef={manageButton} />
      {railCards.length > 0
        ? <AgentRail cards={railCards} targetKind={selected.kind} selectedAgentId={selectedAgentId} onSelect={setPickedAgentId} disabled={launchBlocked} usages={usageState.usages} usageLoading={usageState.loading} slots={slots} declaredWindows={declaredWindows} needsAction={needsAction} wrap={isDesktop} />
        : <p data-testid="launcher-no-agents" className="mx-4 rounded-xl border border-dashed border-wc-default px-3 py-6 text-center text-sm text-wc-text-muted">{t(strings.launcher.noAgentsShown)}</p>}
    </section>
  );

  const settings = selectedCard ? (
    <HarnessSettings card={selectedCard} harness={harness} harnessKnown={harnesses.known} model={model} effectiveChoice={effectiveChoice} effectiveModelName={effectiveModelName}
      limit={appliedLimit} onLimit={setContextLimit} saveAgent={saveAgent} saveMode={saveMode} canControlUsage={usageMode.state.canControl} usageBusy={usageMode.busy} usageError={usageMode.error}
      onStartSave={() => { if (selectedAgentId) setUsageSheet({ agentId: selectedAgentId }); }} onEndSave={(id) => { void usageMode.end(id); }}
      fullPower={fullPower} onFullPower={setFullPower} billing={billing} onOpenDetail={() => { if (selectedAgentId) setManagePage({ agentId: selectedAgentId }); }}
      refresh={{ run: selectedAgentId && updateState.runs[selectedAgentId]?.action === "refresh_catalog" ? updateState.runs[selectedAgentId] : undefined, start: () => { if (selectedAgentId) void updateState.run(selectedAgentId, "refresh_catalog").then(model.reload); } }} />
  ) : null;

  const shellPanel = <ShellPanel command={customCommand} onCommand={setCustomCommand} onSubmit={launchShell} shortcuts={grid.commands} disabled={launchBlocked}
    onShortcut={(command) => { onLaunch(buildLaunchOptions(command)); }} />;
  const groupPanel = onCreateGroupFromRoles ? (
    <div className="px-4 pt-4"><GroupModePanel open={open} onCreate={onCreateGroupFromRoles} isCreating={isCreating} disabled={!selected.available || noBackendAvailable} footerHost={footerHost} machineLabel={selected.label} onEditTemplates={onEditTemplates} /></div>
  ) : null;

  const extras = sessionType === null || sessionType === "group" ? null : (
    <LaunchExtras groups={groups} destinationGroupId={destinationGroupId} onDestination={changeDestination} onCreateDestination={createDestination} appearance={appearance}
      workingDir={workingDir} onWorkingDir={setWorkingDir} backends={backends} backend={selectedBackend} onBackend={setSelectedBackend} policyKey={selectedPolicyKey} onPolicyKey={setSelectedPolicyKey} />
  );

  const footer = sessionType === null ? null : sessionType === "group"
    ? <div ref={setFooterHost} className="w-full" />
    : (
      <LaunchFooter testId="launcher-submit-selection" summary={summaryParts.join(" · ")}
        label={sessionType === "shell" ? t(strings.terminalLauncher.launchSelectedShell) : selectedCard ? t(strings.terminalLauncher.launchSelectedAgent, { name: selectedCard.label }) : t(strings.terminalLauncher.launch)}
        disabled={launchBlocked || (sessionType === "agent" && (!selectedCard || (model.usesVrooliCredits && billing.blocked) || modelBlocked || model.loading))}
        busy={isCreating} busyLabel={t(strings.terminalLauncher.creating)}
        onLaunch={sessionType === "shell" ? launchShell : launchAgent} />
    );

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      closeLabel={t(strings.terminalLauncher.closeAriaLabel)}
      title={t(strings.terminalLauncher.newSession)}
      headerActions={headerActions}
      footer={isDesktop ? undefined : footer}
      size="lg"
      avoidKeyboard
      // The drawer itself takes focus, which paints no ring on any device, and
      // Enter launches from there. See onDrawerKeyDown.
      initialFocusRef={drawerRoot}
      panelClassName={`wc-rcl-theme wc-launcher-panel ${isDesktop ? "md:!w-[58rem] md:!max-w-full" : ""}`}
      contentClassName={isDesktop ? "!overflow-hidden" : undefined}
      // The dialog caps its own height against the host viewport and owns the
      // scroll region; a local max-height would be measured against the wrong
      // viewport and make a second scroller inside it.
      testId="terminal-launcher"
    >
      {isDesktop ? (
        <div ref={drawerRoot} tabIndex={-1} onKeyDown={onDrawerKeyDown} data-testid="launcher-desktop" className="grid h-[min(35rem,calc(100dvh-10rem))] grid-cols-[minmax(0,1fr)_380px] outline-none">
          <div className="min-w-0 overflow-y-auto border-e border-wc-default py-4">
            <div className="max-w-[22rem] px-4">{typeTabs}</div>
            {alerts}
            {sessionType === "agent" && agentChoice}
            {sessionType === "shell" && shellPanel}
            {sessionType === "group" && groupPanel}
            {extras}
          </div>
          <div className="flex min-h-0 flex-col">
            <div className="relative min-h-0 flex-1 overflow-y-auto p-4 pb-0">
              {sessionType === "agent" ? settings : sessionType === null ? null : <p className="px-1 text-xs text-wc-text-muted">{t(strings.terminalLauncher.nothingToConfigure)}</p>}
              <ScrollFade />
            </div>
            <div className="border-t border-wc-default p-4">{footer}</div>
          </div>
        </div>
      ) : (
        <div ref={drawerRoot} tabIndex={-1} onKeyDown={onDrawerKeyDown} data-testid="launcher-phone" className="min-h-[calc(100dvh-14rem)] pt-4 outline-none">
          <div className="px-4">{typeTabs}</div>
          {alerts}
          {sessionType === "agent" && <>{agentChoice}<div className="px-4 pt-3">{settings}</div></>}
          {sessionType === "shell" && shellPanel}
          {sessionType === "group" && groupPanel}
          {extras}
          <ScrollFade />
        </div>
      )}
      <UsageModeSheet open={usageSheet !== null} initialAgentId={usageSheet?.agentId} modelName={(id) => modelLabel(model, id) ?? id} onClose={() => { setUsageSheet(null); }} usageMode={usageMode} />
      <ManageAgents page={managePage} onPage={setManagePage} returnFocusRef={manageButton} target={selected} cards={agents}
        preferences={shortcutsProp ? null : agentPreferences} preferencesFailed={agentPreferences.error} onRetryPreferences={retryPreferences}
        onOrder={(ids) => { if (prefs) void agentPreferences.update({ orderedAgentIds: mergeAgentOrder(prefs.orderedAgentIds, ids) }); }}
        onVisibility={(id, hide) => {
          if (!prefs) return;
          const others = prefs.hiddenAgentIds.filter((item) => item !== id);
          void agentPreferences.update({ hiddenAgentIds: hide ? [...others, id] : others });
        }}
        updates={updateState} usages={usageState.usages} usageLoading={usageState.loading} declaredWindows={declaredWindows} installs={installs} canInstall={Boolean(onInstallCapability)} onEditShortcuts={onEditShortcuts} />
    </ResponsiveDialog>
  );
}
