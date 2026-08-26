// DOC: docs/reference/configuration.md#launcher-shortcuts
// DOC: docs/internal/SEAMS.md#1-entry-presentation
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import {
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  Info,
  Loader2,
  Monitor,
  RefreshCw,
  Search,
  Server,
  Terminal,
  WifiOff,
  XCircle,
  Zap,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "./ui/button";
import { DrawerShell } from "@vrooli/react-component-library/DrawerShell/1.0.0";
import { strings } from "../consts/strings";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../consts/shortcuts";
import { shortcutsClient } from "../api/shortcuts";
import { BACKEND_OPTIONS } from "../consts/backend-options";
import { POLICY_OPTIONS, policyKey, parsePolicySelection } from "../consts/policy-options";
import { slugify } from "../lib/slugify";
import type { BackendID, BackendOption, ExpirationPolicy, PolicyMode } from "../api/sessions";
import type { TargetCatalog, TerminalTarget, TerminalTargetState } from "../api/targets";

export type { TerminalTarget } from "../api/targets";

// [REQ:P0-006a] Terminal Launch Flow UI
// [REQ:P0-006b] Configurable Shortcut Entries
// [REQ:P1-002b] Shortcut Profile Management UI
// [REQ:P1-013a] Remote target catalog and readiness

const optionCardClass =
  "flex min-h-[60px] w-full items-center gap-3 rounded-xl border border-wc-default bg-wc-surface-input px-4 py-3 text-start transition hover:border-wc-accent hover:bg-wc-surface-input/80 disabled:cursor-not-allowed disabled:opacity-50";

const codexDeviceAuthCommand = "codex login --device-auth";

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
  shortcuts?: ShortcutEntry[];
  isCreating?: boolean;
  defaultBackend?: BackendID;
  defaultPolicy?: ExpirationPolicy;
  availableBackends?: BackendOption[];
  availableTargets?: TerminalTarget[];
  targetCatalog?: TargetCatalog;
  targetsLoading?: boolean;
  onRefreshTargets?: () => void | Promise<void>;
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

function statusIcon(state: TerminalTargetState | undefined) {
  if (state === "dispatchable") return <CheckCircle2 className="h-4 w-4" aria-hidden />;
  if (state === "offline") return <WifiOff className="h-4 w-4" aria-hidden />;
  if (state === "needs-update") return <RefreshCw className="h-4 w-4" aria-hidden />;
  return <CircleAlert className="h-4 w-4" aria-hidden />;
}

function statusClass(state: TerminalTargetState | undefined): string {
  if (state === "dispatchable") return "border-emerald-400/30 bg-emerald-400/10 text-emerald-300";
  if (state === "offline") return "border-slate-400/20 bg-slate-400/10 text-slate-300";
  if (state === "needs-update") return "border-amber-400/30 bg-amber-400/10 text-amber-300";
  return "border-rose-400/30 bg-rose-400/10 text-rose-300";
}

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
  const [targetSearch, setTargetSearch] = useState("");
  const targetButtonRefs = useRef<Record<string, HTMLButtonElement | null>>({});

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
  const remoteTargets = targets.filter((target) => target.kind !== "local");
  const filteredRemoteTargets = remoteTargets.filter((target) => {
    const query = targetSearch.trim().toLowerCase();
    return !query || `${target.label} ${target.os ?? ""} ${target.arch ?? ""}`.toLowerCase().includes(query);
  });
  const visibleTargets = useMemo(() => [targets[0] ?? localFallback, ...filteredRemoteTargets], [filteredRemoteTargets, targets]);
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

  const handleTargetKeyDown = useCallback((event: KeyboardEvent, currentID: string) => {
    const isHome = event.key === "Home";
    const isEnd = event.key === "End";
    const direction = event.key === "ArrowDown" || event.key === "ArrowRight" ? 1
      : event.key === "ArrowUp" || event.key === "ArrowLeft" ? -1
        : 0;
    if (!isHome && !isEnd && direction === 0) return;
    event.preventDefault();
    const currentIndex = visibleTargets.findIndex((target) => target.id === currentID);
    const nextIndex = isHome ? 0 : isEnd ? visibleTargets.length - 1 : Math.min(Math.max(currentIndex + direction, 0), visibleTargets.length - 1);
    const next = visibleTargets[nextIndex];
    if (!next) return;
    setSelectedTarget(next.id);
    targetButtonRefs.current[next.id]?.focus();
  }, [visibleTargets]);

  const catalogMessage = targetCatalog?.status === "unconfigured"
    ? t(strings.terminalLauncher.unconfigured)
    : targetCatalog?.status === "configured-empty"
      ? t(strings.terminalLauncher.configuredEmpty)
      : targetCatalog?.status === "registry-error"
        ? t(strings.terminalLauncher.registryError)
        : targetCatalog?.message;

  return (
    <DrawerShell
      open={open}
      onClose={onClose}
      closeAriaLabel={t(strings.terminalLauncher.closeAriaLabel)}
      title={t(strings.terminalLauncher.newTerminal)}
      size="compact"
      avoidKeyboard
      panelTestId="terminal-launcher"
    >
      <div className="flex max-h-[min(760px,calc(100dvh-2rem))] min-h-0 flex-col">
        <div className="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
          <header>
            <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-wc-accent">{t(strings.terminalLauncher.eyebrow)}</div>
            <p className="mt-1 text-sm leading-5 text-wc-text-muted">{t(strings.terminalLauncher.description)}</p>
          </header>

          {noBackendAvailable && (
            <div data-testid="launcher-no-backend" role="alert" className="flex gap-3 rounded-xl border border-rose-400/25 bg-rose-400/10 p-3 text-sm text-rose-100">
              <XCircle className="mt-0.5 h-4 w-4 shrink-0 text-rose-300" aria-hidden />
              <span>{backendUnavailableReason}</span>
            </div>
          )}

          <section aria-labelledby="launcher-locations-heading" className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 id="launcher-locations-heading" className="text-sm font-semibold text-wc-text-primary">{t(strings.terminalLauncher.locations)}</h3>
                <p className="text-xs text-wc-text-faint">{launchOnCopy(selected.label)}</p>
              </div>
              {onRefreshTargets && (
                <button
                  type="button"
                  data-testid="launcher-target-refresh"
                  onClick={() => void onRefreshTargets()}
                  disabled={targetsLoading}
                  className="inline-flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-60"
                  aria-label={t(strings.terminalLauncher.refreshTargets)}
                >
                  <RefreshCw className={targetsLoading ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden />
                  <span className="hidden sm:inline">{targetsLoading ? t(strings.terminalLauncher.refreshing) : t(strings.terminalLauncher.refreshTargets)}</span>
                </button>
              )}
            </div>

            {targetsLoading && targets.length <= 1 ? (
              <div data-testid="launcher-target-loading" className="flex items-center gap-3 rounded-xl border border-wc-default bg-wc-surface-input/50 p-4 text-sm text-wc-text-muted">
                <Loader2 className="h-4 w-4 animate-spin text-wc-accent" aria-hidden />
                {t(strings.terminalLauncher.loadingTargets)}
              </div>
            ) : (
              <div className="space-y-2" role="listbox" aria-label={t(strings.terminalLauncher.locations)}>
                <TargetCard
                  target={targets[0] ?? localFallback}
                  selected={selected.id === (targets[0]?.id ?? "local")}
                  onSelect={setSelectedTarget}
                  statusLabel={statusLabelFor((targets[0] ?? localFallback).state)}
                  onKeyDown={handleTargetKeyDown}
                  buttonRef={(node) => { targetButtonRefs.current[(targets[0] ?? localFallback).id] = node; }}
                />
                {remoteTargets.length > 0 && (
                  <div className="space-y-2 pt-2">
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
                        <Server className="h-3.5 w-3.5" aria-hidden />
                        {t(strings.terminalLauncher.remoteNodes)}
                      </div>
                      <span className="text-xs text-wc-text-faint">{remoteTargets.length}</span>
                    </div>
                    {remoteTargets.length > 3 && (
                      <label className="relative block">
                        <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-wc-text-faint" aria-hidden />
                        <input
                          data-testid="launcher-target-search"
                          aria-label={t(strings.terminalLauncher.searchNodes)}
                          value={targetSearch}
                          onChange={(event) => { setTargetSearch(event.target.value); }}
                          placeholder={t(strings.terminalLauncher.searchNodes)}
                          className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input py-2 ps-9 pe-3 text-sm text-wc-text-primary outline-none transition placeholder:text-wc-text-faint focus:border-wc-accent"
                        />
                      </label>
                    )}
                    {filteredRemoteTargets.map((target) => (
                      <TargetCard key={target.id} target={target} selected={selected.id === target.id} onSelect={setSelectedTarget} statusLabel={statusLabelFor(target.state)} onKeyDown={handleTargetKeyDown} buttonRef={(node) => { targetButtonRefs.current[target.id] = node; }} />
                    ))}
                    {filteredRemoteTargets.length === 0 && <p className="px-1 text-sm text-wc-text-muted">{t(strings.terminalLauncher.noRemoteNodes)}</p>}
                  </div>
                )}
              </div>
            )}

            {catalogMessage && (
              <div data-testid="launcher-target-catalog-state" className="flex gap-3 rounded-xl border border-amber-400/25 bg-amber-400/10 p-3 text-sm text-amber-100">
                <CircleAlert className="mt-0.5 h-4 w-4 shrink-0 text-amber-300" aria-hidden />
                <div className="min-w-0">
                  <div>{catalogMessage}</div>
                  {targetCatalog?.recovery_action && <div className="mt-1 text-xs text-amber-200/80">{targetCatalog.recovery_action}</div>}
                </div>
              </div>
            )}
          </section>

          <section aria-labelledby="launcher-actions-heading" className="space-y-3">
            <div>
              <h3 id="launcher-actions-heading" className="text-sm font-semibold text-wc-text-primary">{t(strings.terminalLauncher.actions)}</h3>
              <p className="text-xs text-wc-text-faint">{selected.available ? launchOnCopy(selected.label) : t(strings.terminalLauncher.chooseReadyLocation)}</p>
            </div>
            {!selected.available && (
              <div className="flex gap-3 rounded-xl border border-rose-400/25 bg-rose-400/10 p-3 text-sm text-rose-100">
                <XCircle className="mt-0.5 h-4 w-4 shrink-0 text-rose-300" aria-hidden />
                <div><div>{t(strings.terminalLauncher.targetUnavailable)}</div><div className="mt-1 text-xs text-rose-200/80">{selected.recovery_action ?? selected.failure_rung}</div></div>
              </div>
            )}

            <button
              type="button"
              data-testid="launcher-empty-shell"
              onClick={() => { onLaunch(buildLaunchOptions()); }}
              disabled={isCreating || !selected.available || noBackendAvailable}
              className={optionCardClass}
            >
              <Terminal className="h-5 w-5 shrink-0 text-wc-accent" aria-hidden />
              <div className="min-w-0 flex-1"><div className="font-medium text-wc-text-primary">{t(strings.terminalLauncher.emptyShell)}</div><div className="text-sm text-wc-text-muted">{t(strings.terminalLauncher.emptyShellDescription)}</div></div>
              <ChevronRight className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />
            </button>

            <button
              type="button"
              data-testid="launcher-codex-sign-in"
              onClick={() => { onLaunch(buildLaunchOptions(codexDeviceAuthCommand)); }}
              disabled={isCreating || !selected.available || noBackendAvailable}
              className={optionCardClass}
            >
              <ShieldIcon />
              <div className="min-w-0 flex-1"><div className="font-medium text-wc-text-primary">{t(strings.terminalLauncher.codexSignIn)}</div><div className="text-sm text-wc-text-muted">{t(strings.terminalLauncher.codexSignInDescription)}</div></div>
              <ChevronRight className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />
            </button>

            {shortcuts.length > 0 && (
              <div className="space-y-2">
                <div className="px-1 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.terminalLauncher.shortcuts)}</div>
                {shortcuts.filter((shortcut) => shortcut.command !== codexDeviceAuthCommand).map((shortcut) => (
                  <button key={shortcut.command} type="button" data-testid={`launcher-shortcut-${slugify(shortcut.label)}`} onClick={() => { onLaunch(buildLaunchOptions(shortcut.command)); }} disabled={isCreating || !selected.available || noBackendAvailable} className={optionCardClass}>
                    <Zap className="h-5 w-5 shrink-0 text-yellow-400" aria-hidden />
                    <div className="min-w-0 flex-1"><div className="font-medium text-wc-text-primary">{shortcut.label}</div><div className="truncate text-sm text-wc-text-muted">{shortcut.description || shortcut.command}</div></div>
                  </button>
                ))}
              </div>
            )}
          </section>

          <section className="space-y-2">
            <div className="px-1 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.terminalLauncher.customCommand)}</div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input data-testid="launcher-custom-input" type="text" value={customCommand} onChange={(event) => { setCustomCommand(event.target.value); }} onKeyDown={(event) => { if (event.key === "Enter") handleLaunchCustom(); }} placeholder={t(strings.terminalLauncher.commandPlaceholder)} className="min-h-11 min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary outline-none placeholder:text-wc-text-faint focus:border-wc-accent" />
              <Button data-testid="launcher-custom-launch" size="sm" onClick={handleLaunchCustom} disabled={isCreating || !customCommand.trim() || !selected.available || noBackendAvailable}>{t(strings.terminalLauncher.launch)}</Button>
            </div>
          </section>

          <section className="space-y-2">
            <button type="button" data-testid="launcher-options-toggle" className="flex min-h-11 items-center gap-1 px-1 text-xs font-semibold uppercase tracking-wider text-wc-text-faint hover:text-wc-text-muted" onClick={() => { setOptionsOpen((value) => !value); }} aria-expanded={optionsOpen}>
              {optionsOpen ? <ChevronDown className="h-3.5 w-3.5" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5" aria-hidden />}
              {t(strings.terminalLauncher.sessionOptions)}
            </button>
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

          {selected.kind !== "local" && selected.readiness && selected.readiness.length > 0 && (
            <div className="rounded-xl border border-wc-default bg-wc-surface-base/40 p-4">
              <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-wc-text-faint"><Info className="h-3.5 w-3.5" aria-hidden />{t(strings.terminalLauncher.readiness)}</div>
              <div className="grid gap-2 sm:grid-cols-2">{selected.readiness.map((fact) => <div key={fact.key} className="flex items-center gap-2 text-xs text-wc-text-secondary"><span className={fact.passed ? "text-emerald-300" : "text-rose-300"}>{fact.passed ? <Check className="h-3.5 w-3.5" aria-hidden /> : <XCircle className="h-3.5 w-3.5" aria-hidden />}</span><span className="truncate" title={fact.detail}>{fact.label}</span></div>)}</div>
            </div>
          )}
        </div>
        <footer className="shrink-0 border-t border-wc-default bg-wc-surface-raised/95 px-5 py-3 text-xs text-wc-text-faint">
          {isCreating ? <div data-testid="launcher-creating" className="flex items-center justify-center gap-2 text-sm text-wc-text-muted"><Loader2 className="h-4 w-4 animate-spin" aria-hidden />{t(strings.terminalLauncher.creating)}</div> : selected.kind !== "local" && selected.available ? <div className="flex items-center gap-2"><Monitor className="h-3.5 w-3.5" aria-hidden /><span>{selected.os && selected.arch ? `${selected.os}/${selected.arch}` : selected.label}</span>{lastSeenCopy(selected, t(strings.terminalLauncher.neverSeen)) && <span className="ms-auto">{t(strings.terminalLauncher.lastSeen)}: {lastSeenCopy(selected, t(strings.terminalLauncher.neverSeen))}</span>}</div> : null}
        </footer>
      </div>
    </DrawerShell>
  );
}

function ShieldIcon() {
  return <Server className="h-5 w-5 shrink-0 text-violet-300" aria-hidden />;
}

function TargetCard({ target, selected, onSelect, statusLabel, onKeyDown, buttonRef }: { target: TerminalTarget; selected: boolean; onSelect: (id: string) => void; statusLabel: string; onKeyDown: (event: KeyboardEvent, id: string) => void; buttonRef: (node: HTMLButtonElement | null) => void }) {
  const isLocal = target.kind === "local";
  const state = target.state ?? (target.available ? "dispatchable" : "unavailable");
  const label = isLocal ? "This machine" : target.label;
  const metadata = [target.os, target.arch].filter(Boolean).join(" · ");
  const reason = target.recovery_action ?? target.failure_rung;
  return (
    <button ref={buttonRef} type="button" role="option" aria-selected={selected} aria-label={`${label}, ${statusLabel}${reason ? `, ${reason}` : ""}`} title={reason} data-testid={`launcher-target-card-${slugify(target.id)}`} onClick={() => { onSelect(target.id); }} onKeyDown={(event) => { onKeyDown(event, target.id); }} className={`flex min-h-[68px] w-full items-center gap-3 rounded-xl border px-4 py-3 text-start transition ${selected ? "border-wc-accent bg-wc-accent/10 shadow-[0_0_0_1px_rgba(129,140,248,0.16)]" : "border-wc-default bg-wc-surface-input hover:border-wc-accent/60"}`}>
      <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${isLocal ? "bg-wc-accent/15 text-wc-accent" : "bg-wc-surface-base text-wc-text-secondary"}`}>{isLocal ? <Monitor className="h-5 w-5" aria-hidden /> : <Server className="h-5 w-5" aria-hidden />}</span>
      <span className="min-w-0 flex-1"><span className="flex items-center gap-2"><span className="truncate font-medium text-wc-text-primary">{label}</span>{selected && <Check className="h-4 w-4 shrink-0 text-wc-accent" aria-hidden />}</span><span className="mt-0.5 block truncate text-xs text-wc-text-faint">{metadata || (isLocal ? "Web Console host" : target.node_id || target.id)}</span></span>
      <span className={`inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2 py-1 text-[11px] font-medium ${statusClass(state)}`}>{statusIcon(state)}<span className="hidden sm:inline">{statusLabel}</span></span>
    </button>
  );
}
