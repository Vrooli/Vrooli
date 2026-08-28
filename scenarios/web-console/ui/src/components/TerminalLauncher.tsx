// DOC: docs/reference/configuration.md#launcher-shortcuts
// DOC: docs/internal/SEAMS.md#1-entry-presentation
import { useCallback, useEffect, useMemo, useState, type CSSProperties } from "react";
import {
  ChevronDown,
  ChevronRight,
  Info,
  Loader2,
  Monitor,
  Play,
  Settings2,
  Terminal,
  TriangleAlert,
  XCircle,
  Zap,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Input } from "@vrooli/react-component-library/Input";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { strings } from "../consts/strings";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../consts/shortcuts";
import { shortcutsClient } from "../api/shortcuts";
import { BACKEND_OPTIONS } from "../consts/backend-options";
import { POLICY_OPTIONS, policyKey, parsePolicySelection } from "../consts/policy-options";
import { slugify } from "../lib/slugify";
import type { BackendID, BackendOption, ExpirationPolicy, PolicyMode } from "../api/sessions";
import type { TargetCatalog, TerminalTarget, TerminalTargetState } from "../api/targets";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";
import MachinePicker from "./launcher/MachinePicker";
import GroupDestinationTrigger from "./launcher/GroupPicker";
import GroupModePanel, { type GroupCreationRequest } from "./launcher/GroupModePanel";
import { cardSupportsAttribution, commandForCard, foldAttributedShortcuts } from "./launcher/agentGrid";
import { cn } from "../lib/classnames";

export type { TerminalTarget } from "../api/targets";

/** One session at a time, or a whole group in one trip through the dialog. */
type LauncherMode = "one-session" | "group";

// [REQ:P0-006a] Terminal Launch Flow UI
// [REQ:P0-006b] Configurable Shortcut Entries
// [REQ:P1-002b] Shortcut Profile Management UI
// [REQ:P1-013a] Remote target catalog and readiness

const optionCardClass =
  "flex min-h-[60px] w-full items-center gap-3 rounded-xl border border-wc-default bg-wc-surface-input px-4 py-3 text-start transition hover:border-wc-accent hover:bg-wc-surface-input/80 disabled:cursor-not-allowed disabled:opacity-50";

/** Denser than optionCardClass: these sit two-up at every width. */
const agentCardClass =
  "flex min-h-[56px] w-full items-center gap-2.5 rounded-xl border border-wc-default bg-wc-surface-input px-3 py-2.5 text-start transition hover:border-wc-accent hover:bg-wc-surface-input/80 disabled:cursor-not-allowed disabled:opacity-50";

export interface LaunchOptions {
  command?: string;
  backend?: BackendID;
  policy?: { mode: PolicyMode; duration?: string };
  target?: TerminalTarget;
  workingDir?: string;
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
  /** Opens the shortcut editor, which owns the agent list this grid renders. */
  onEditShortcuts?: () => void;
  /** Opens the template editor, which owns the saved group recipes. */
  onEditTemplates?: () => void;
}

const localFallback: TerminalTarget = {
  id: "local",
  kind: "local",
  label: "This machine",
  available: true,
  state: "dispatchable",
  status: "LOCAL",
  online: true,
  readiness: [{ key: "local", label: "Web Console process", passed: true, detail: "Available on this machine" }],
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
  onEditShortcuts,
  onEditTemplates,
  groups = [],
  pendingGroupId = null,
  onDestinationChange,
  onCreateGroup,
  appearance,
  onCreateGroupFromRoles,
}: TerminalLauncherProps) {
  const { t } = useTranslation();
  const [customCommand, setCustomCommand] = useState("");
  const [workingDir, setWorkingDir] = useState("");
  const [apiShortcuts, setApiShortcuts] = useState<ShortcutEntry[] | null>(null);
  const [selectedBackend, setSelectedBackend] = useState<BackendID>(defaultBackend);
  const [selectedPolicyKey, setSelectedPolicyKey] = useState<string>(
    defaultPolicy ? policyKey(defaultPolicy.mode, defaultPolicy.duration) : "never",
  );
  const [optionsOpen, setOptionsOpen] = useState(false);
  const [selectedTarget, setSelectedTarget] = useState("local");
  // The destination is dialog state, seeded from the caller's pending group.
  // Seeding rather than mirroring is deliberate: once the operator changes it
  // here, a re-render from the caller must not silently move it back.
  const [destinationGroupId, setDestinationGroupId] = useState<string | null>(pendingGroupId);
  const [attributed, setAttributed] = useState(false);
  const [mode, setMode] = useState<LauncherMode>("one-session");
  const [appearanceOpen, setAppearanceOpen] = useState(false);

  // Reseed the destination each time the dialog opens, so opening it from a
  // different group header shows that group rather than the last one.
  useEffect(() => {
    if (open) {
      setDestinationGroupId(pendingGroupId);
      // Always reopen on the single-session mode: it is the common case, and
      // landing in group mode because the last trip used it would surprise.
      setMode("one-session");
    }
  }, [open, pendingGroupId]);

  useEffect(() => { setSelectedBackend(defaultBackend); }, [defaultBackend]);
  useEffect(() => {
    if (defaultPolicy) setSelectedPolicyKey(policyKey(defaultPolicy.mode, defaultPolicy.duration));
  }, [defaultPolicy]);

  useEffect(() => {
    if (!open || shortcutsProp) return;
    let cancelled = false;
    shortcutsClient.getEffective({})
      .then((resp) => {
        if (!cancelled) setApiShortcuts(resp.shortcuts.map((s) => ({
          label: s.label,
          command: s.command,
          description: s.description || undefined,
        })));
      })
      .catch(() => {
        if (!cancelled) setApiShortcuts(null);
      });
    return () => { cancelled = true; };
  }, [open, shortcutsProp]);

  const shortcuts = shortcutsProp ?? apiShortcuts ?? DEFAULT_SHORTCUTS;
  const catalogTargets = targetCatalog?.targets ?? availableTargets;
  const targets = useMemo(() => {
    const local = catalogTargets.find((target) => target.id === "local") ?? localFallback;
    return [local, ...catalogTargets.filter((target) => target.id !== "local")];
  }, [catalogTargets]);
  const selected = targets.find((target) => target.id === selectedTarget) ?? targets[0] ?? localFallback;
  const statusLabels: Record<TerminalTargetState, string> = {
    dispatchable: t(strings.terminalLauncher.dispatchable),
    offline: t(strings.terminalLauncher.offline),
    "needs-update": t(strings.terminalLauncher.needsUpdate),
    unconfigured: t(strings.terminalLauncher.unconfigured),
    unavailable: t(strings.terminalLauncher.unavailable),
  };
  const statusLabelFor = (state: TerminalTargetState | undefined) => statusLabels[state ?? "unavailable"];
  const launchOnCopy = (label: string) => t(strings.terminalLauncher.launchOn).replace("{{label}}", label);

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

  // Four agent cards, not eight rows: the "(attributed)" entries are the same
  // agents with one setting changed, and the toggle below says so.
  const agentCards = useMemo(() => foldAttributedShortcuts(shortcuts), [shortcuts]);

  const buildLaunchOptions = useCallback((command?: string): LaunchOptions => {
    const parsed = parsePolicySelection(selectedPolicyKey);
    return {
      command,
      target: selected,
      workingDir: workingDir.trim() || undefined,
      backend: selectedBackend !== defaultBackend ? selectedBackend : undefined,
      policy: parsed ?? undefined,
    };
  }, [defaultBackend, selected, selectedBackend, selectedPolicyKey, workingDir]);

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
            <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-wc-accent">{t(strings.terminalLauncher.eyebrow)}</div>
            <p className="mt-1 text-sm leading-5 text-wc-text-muted">{t(strings.terminalLauncher.description)}</p>
          </header>

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
          <section aria-labelledby="launcher-actions-heading" className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              <h3 id="launcher-actions-heading" className="text-sm font-semibold text-wc-text-primary">{t(strings.launcher.agents)}</h3>
              {/* One toggle replaces four duplicate rows. The choice was
                  always "which agent" plus "attributed or not"; the eight-row
                  list just spelled the second half four times. */}
              {agentCards.some(cardSupportsAttribution) && (
                <label className="flex min-h-11 items-center gap-2 text-xs text-wc-text-secondary" title={t(strings.launcher.attributedHint)}>
                  <input
                    type="checkbox"
                    data-testid="launcher-attributed-toggle"
                    checked={attributed}
                    onChange={(event) => { setAttributed(event.target.checked); }}
                    className="h-4 w-4 accent-[rgb(var(--wc-accent))]"
                  />
                  {t(strings.launcher.attributed)}
                </label>
              )}
            </div>

            {/* Two columns at every width. A single column of full-width rows is a
                list you read one entry at a time; the grid is one you scan. */}
            <div data-testid="launcher-agent-grid" className="grid grid-cols-2 gap-2">
              {agentCards.map((card) => {
                const command = commandForCard(card, attributed);
                if (!command) return null;
                return (
                  <button
                    key={card.label}
                    type="button"
                    data-testid={`launcher-agent-${slugify(card.label)}`}
                    onClick={() => { onLaunch(buildLaunchOptions(command)); }}
                    disabled={isCreating || !selected.available || noBackendAvailable}
                    className={agentCardClass}
                    title={command}
                  >
                    <Zap className="h-4 w-4 shrink-0 text-yellow-400" aria-hidden />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium text-wc-text-primary">{card.label}</span>
                      <span className="block truncate text-[11px] text-wc-text-muted">
                        {(attributed ? card.attributedDescription : card.description) ?? command}
                      </span>
                    </span>
                  </button>
                );
              })}

              <button
                type="button"
                data-testid="launcher-empty-shell"
                onClick={() => { onLaunch(buildLaunchOptions()); }}
                disabled={isCreating || !selected.available || noBackendAvailable}
                className={agentCardClass}
              >
                <Terminal className="h-4 w-4 shrink-0 text-wc-accent" aria-hidden />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium text-wc-text-primary">{t(strings.terminalLauncher.emptyShell)}</span>
                  <span className="block truncate text-[11px] text-wc-text-muted">{t(strings.terminalLauncher.emptyShellDescription)}</span>
                </span>
              </button>

              {/* The grid renders a fixed set; the shortcut list is where it
                  comes from. A dashed card says so in place, instead of
                  leaving the operator to guess which settings tab owns it. */}
              {onEditShortcuts && (
                <button
                  type="button"
                  data-testid="launcher-edit-shortcuts"
                  onClick={onEditShortcuts}
                  className={cn(agentCardClass, "border-dashed")}
                >
                  <Settings2 className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium text-wc-text-secondary">{t(strings.launcher.editShortcuts)}</span>
                    <span className="block truncate text-[11px] text-wc-text-muted">{t(strings.launcher.shortcutCount, { count: shortcuts.length })}</span>
                  </span>
                </button>
              )}
            </div>

            {/* Anything the operator added to their shortcut list that is not
                one of the agents above still gets a row, so this fold can
                never hide their own entries. */}
            {shortcuts.filter((shortcut) => !agentCards.some((card) => card.command === shortcut.command || card.attributedCommand === shortcut.command)).length > 0 && (
              <div className="space-y-2">
                <div className="px-1 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.terminalLauncher.shortcuts)}</div>
                {shortcuts
                  .filter((shortcut) => !agentCards.some((card) => card.command === shortcut.command || card.attributedCommand === shortcut.command))
                  .map((shortcut) => (
                    <button
                      key={shortcut.command}
                      type="button"
                      data-testid={`launcher-shortcut-${slugify(shortcut.label)}`}
                      onClick={() => { onLaunch(buildLaunchOptions(shortcut.command)); }}
                      disabled={isCreating || !selected.available || noBackendAvailable}
                      className={optionCardClass}
                    >
                      <Zap className="h-5 w-5 shrink-0 text-yellow-400" aria-hidden />
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-wc-text-primary">{shortcut.label}</div>
                        <div className="truncate text-sm text-wc-text-muted">{shortcut.description || shortcut.command}</div>
                      </div>
                    </button>
                  ))}
              </div>
            )}
          </section>
          )}

          {mode === "one-session" && (
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

