// DOC: docs/reference/configuration.md#launcher-shortcuts
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback, useEffect } from "react";
import { Terminal, Zap, ChevronDown, ChevronRight, Info } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "./ui/button";
import { DrawerShell } from "./DrawerShell";
import { strings } from "../consts/strings";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../consts/shortcuts";
import { shortcutsClient } from "../api/shortcuts";
import { BACKEND_OPTIONS } from "../consts/backend-options";
import { POLICY_OPTIONS, policyKey, parsePolicySelection } from "../consts/policy-options";
import { slugify } from "../lib/slugify";
import type { BackendID, BackendOption, ExpirationPolicy, PolicyMode } from "../api/sessions";

// [REQ:P0-006a] Terminal Launch Flow UI
// [REQ:P0-006b] Configurable Shortcut Entries
// [REQ:P1-002b] Shortcut Profile Management UI

/** Shared class string for launcher option cards. */
const optionCardClass =
  "flex w-full items-center gap-3 rounded-md border border-wc-default bg-wc-surface-input px-4 py-3 text-start transition hover:border-wc-accent hover:bg-wc-surface-input/80 disabled:opacity-50";

export interface LaunchOptions {
	command?: string;
	backend?: BackendID;
	policy?: { mode: PolicyMode; duration?: string };
	target?: TerminalTarget;
}

export interface TerminalTarget {
  id: string;
  kind: "local" | "bridge-node" | "ssh" | "attached";
	label: string;
	available: boolean;
	readiness?: string[];
	failureRung?: string;
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
}: TerminalLauncherProps) {
  const { t } = useTranslation();
  const [customCommand, setCustomCommand] = useState("");
  const [apiShortcuts, setApiShortcuts] = useState<ShortcutEntry[] | null>(null);
  const [selectedBackend, setSelectedBackend] = useState<BackendID>(defaultBackend);
  const [selectedPolicyKey, setSelectedPolicyKey] = useState<string>(
    defaultPolicy ? policyKey(defaultPolicy.mode, defaultPolicy.duration) : "never",
  );
	const [optionsOpen, setOptionsOpen] = useState(false);
	const localTarget: TerminalTarget = { id: "local", kind: "local", label: "This machine", available: true };
	const targets = [localTarget, ...availableTargets];
	const [selectedTarget, setSelectedTarget] = useState("local");

  // Reset selections when defaults change
  useEffect(() => {
    setSelectedBackend(defaultBackend);
  }, [defaultBackend]);

  useEffect(() => {
    if (defaultPolicy) {
      setSelectedPolicyKey(policyKey(defaultPolicy.mode, defaultPolicy.duration));
    }
  }, [defaultPolicy]);

  // [REQ:P1-002b] Fetch configuration-driven shortcuts from API on open.
  // Falls back to DEFAULT_SHORTCUTS if the API call fails or prop is provided.
  useEffect(() => {
    if (!open || shortcutsProp) return;
    let cancelled = false;
    shortcutsClient.getEffective({})
      .then((resp) => {
        if (!cancelled) {
          setApiShortcuts(resp.shortcuts.map((s) => ({
            label: s.label,
            command: s.command,
            description: s.description || undefined,
          })));
        }
      })
      .catch(() => {
        if (!cancelled) setApiShortcuts(null);
      });
    return () => { cancelled = true; };
  }, [open, shortcutsProp]);

  const shortcuts = shortcutsProp ?? apiShortcuts ?? DEFAULT_SHORTCUTS;

  // Determine available backends from props or fallback to all options
  const backends = availableBackends
    ? BACKEND_OPTIONS.filter((b) => availableBackends.some((ab) => ab.id === b.id && ab.available))
    : BACKEND_OPTIONS;

  const showBackendSelector = backends.length > 1;

  const buildLaunchOptions = useCallback(
    (command?: string): LaunchOptions => {
      const parsed = parsePolicySelection(selectedPolicyKey);
      // Only include backend when the user explicitly changed it from the
      // default. When omitted, the API uses its configured server default
      // (typically "persistent"). This avoids a race where the hardcoded
      // fallback ("standard") is sent before the async fetch of the real
      // server default completes — which previously caused all sessions to
      // be non-persistent and lost on restart.
      const userChangedBackend = selectedBackend !== defaultBackend;
      return {
			command,
			target: targets.find((target) => target.id === selectedTarget),
			backend: userChangedBackend ? selectedBackend : undefined,
        policy: parsed ?? undefined,
      };
    },
		[selectedBackend, selectedPolicyKey, defaultBackend, selectedTarget, targets],
  );

  // Custom command launch is separate because it validates non-empty input
  // and clears the text field after launching.
  const handleLaunchCustom = useCallback(() => {
    if (customCommand.trim()) {
      onLaunch(buildLaunchOptions(customCommand.trim()));
      setCustomCommand("");
    }
  }, [customCommand, onLaunch, buildLaunchOptions]);

  return (
    <DrawerShell
      open={open}
      onClose={onClose}
      closeAriaLabel={t(strings.terminalLauncher.closeAriaLabel)}
      title={t(strings.terminalLauncher.newTerminal)}
      panelTestId="terminal-launcher"
    >
      <div className="flex h-full flex-col">
        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
          {/* Empty shell option */}
          <button
            data-testid="launcher-empty-shell"
            onClick={() => onLaunch(buildLaunchOptions())}
            disabled={isCreating}
            className={optionCardClass}
          >
            <Terminal className="h-5 w-5 shrink-0 text-wc-accent" />
            <div>
              <div className="font-medium text-wc-text-primary">{t(strings.terminalLauncher.emptyShell)}</div>
              <div className="text-sm text-wc-text-muted">
                {t(strings.terminalLauncher.emptyShellDescription)}
              </div>
            </div>
          </button>

          {/* Shortcut entries */}
          {shortcuts.length > 0 && (
            <div className="space-y-2">
              <div className="px-1 text-xs font-medium uppercase tracking-wider text-wc-text-faint">
                {t(strings.terminalLauncher.shortcuts)}
              </div>
              {shortcuts.map((shortcut) => (
                <button
                  key={shortcut.command}
                  data-testid={`launcher-shortcut-${slugify(shortcut.label)}`}
                  onClick={() => onLaunch(buildLaunchOptions(shortcut.command))}
                  disabled={isCreating}
                  className={optionCardClass}
                >
                  <Zap className="h-5 w-5 shrink-0 text-yellow-400" />
                  <div className="min-w-0 flex-1">
                    <div className="font-medium text-wc-text-primary">
                      {shortcut.label}
                    </div>
                    <div className="truncate text-sm text-wc-text-muted">
                      {shortcut.description || shortcut.command}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}

          {/* Custom command */}
          <div className="space-y-2">
            <div className="px-1 text-xs font-medium uppercase tracking-wider text-wc-text-faint">
              {t(strings.terminalLauncher.customCommand)}
            </div>
            <div className="flex gap-2">
              <input
                data-testid="launcher-custom-input"
                type="text"
                value={customCommand}
                onChange={(e) => setCustomCommand(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleLaunchCustom();
                }}
                placeholder={t(strings.terminalLauncher.commandPlaceholder)}
                className="flex-1 rounded-md border border-wc-default bg-wc-surface-input px-3 py-2 text-sm text-wc-text-primary placeholder:text-wc-text-faint focus:border-wc-accent focus:outline-none"
              />
              <Button
                data-testid="launcher-custom-launch"
                size="sm"
                onClick={handleLaunchCustom}
                disabled={isCreating || !customCommand.trim()}
              >
                {t(strings.terminalLauncher.launch)}
              </Button>
            </div>
          </div>

          {/* Session Options */}
          <div className="space-y-2">
            <button
              data-testid="launcher-options-toggle"
              className="flex items-center gap-1 px-1 text-xs font-medium uppercase tracking-wider text-wc-text-faint hover:text-wc-text-muted"
              onClick={() => setOptionsOpen(!optionsOpen)}
            >
              {optionsOpen ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )}
              {t(strings.terminalLauncher.sessionOptions)}
            </button>
            {optionsOpen && (
              <div className="space-y-2 rounded-md border border-wc-default bg-wc-surface-base/50 p-3">
				<div className="space-y-1">
					<label htmlFor="launcher-target-select" className="text-xs text-wc-text-secondary">Run on</label>
					<select
						id="launcher-target-select"
						data-testid="launcher-target-select"
						className="h-7 w-full rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
						value={selectedTarget}
						onChange={(e) => setSelectedTarget(e.target.value)}
					>
						{targets.map((target) => (
							<option key={target.id} value={target.id} disabled={!target.available}>
								{target.label}{target.available ? "" : ` — ${target.failureRung || "not ready"}`}
							</option>
						))}
					</select>
					{targets.find((target) => target.id === selectedTarget)?.readiness?.map((fact) => (
						<div key={fact} className="text-[11px] text-wc-text-faint">✓ {fact}</div>
					))}
				</div>
                {showBackendSelector && (
                  <div className="flex items-center gap-2">
                    <label className="text-xs text-wc-text-secondary">{t(strings.terminalLauncher.backendLabel)}</label>
                    <select
                      data-testid="launcher-backend-select"
                      className="h-7 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
                      value={selectedBackend}
                      onChange={(e) => setSelectedBackend(e.target.value as BackendID)}
                    >
                      {backends.map((b) => (
                        <option key={b.id} value={b.id}>
                          {b.label}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <label className="text-xs text-wc-text-secondary">{t(strings.terminalLauncher.timeoutLabel)}</label>
                  <select
                    data-testid="launcher-timeout-select"
                    className="h-7 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
                    value={selectedPolicyKey}
                    onChange={(e) => setSelectedPolicyKey(e.target.value)}
                  >
                    {POLICY_OPTIONS.map((opt) => (
                      <option key={policyKey(opt.mode, opt.duration)} value={policyKey(opt.mode, opt.duration)}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>
                {selectedBackend === "persistent" && (
                  <div className="flex items-start gap-1.5 text-[11px] text-wc-text-faint">
                    <Info className="mt-0.5 h-3 w-3 shrink-0" />
                    <span>{t(strings.terminalLauncher.persistentHint)}</span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {isCreating && (
          <div className="shrink-0 border-t border-wc-default px-4 py-2 text-center text-sm text-wc-text-muted">
            {t(strings.terminalLauncher.creating)}
          </div>
        )}
      </div>
    </DrawerShell>
  );
}
