/**
 * The settings you occasionally reach for, collapsed: where the session lands in
 * the workspace, how it will look, and how it runs. None competes with the
 * choice the drawer exists for.
 */
import { useState } from "react";
import { ChevronDown, ChevronRight, Info } from "lucide-react";
import { Input } from "@vrooli/react-component-library/Input";
import { Select } from "@vrooli/react-component-library/Select";
import { useTranslation } from "react-i18next";

import type { BackendID } from "../../api/sessions";
import { strings } from "../../consts/strings";
import { POLICY_OPTIONS, policyKey } from "../../consts/policy-options";
import { cn } from "../../lib/classnames";
import { parseDurationMs } from "../../lib/format";
import { formatLongDuration } from "../../lib/localeFormat";
import type { TabGroupMeta } from "../../stores/useWorkspaceStore";
import GroupDestinationTrigger from "./GroupPicker";
import { FOCUS_RING, SELECT_TEXT } from "./controls";

const DISCLOSURE = cn("flex min-h-11 w-full items-center justify-between gap-3 rounded text-start text-[13px] text-wc-text-muted transition hover:text-wc-text-secondary", FOCUS_RING);
const LABEL = "text-xs font-medium text-wc-text-secondary";

export interface LaunchExtrasProps {
  groups: TabGroupMeta[];
  destinationGroupId: string | null;
  onDestination: (groupId: string | null) => void;
  onCreateDestination: (name: string) => void;
  appearance?: { headerColor: string; themeId: string; fontSize: number };
  workingDir: string;
  onWorkingDir: (value: string) => void;
  backends: readonly { id: BackendID; label: string }[];
  backend: BackendID;
  onBackend: (backend: BackendID) => void;
  policyKey: string;
  onPolicyKey: (key: string) => void;
}

export function LaunchExtras({ groups, destinationGroupId, onDestination, onCreateDestination, appearance, workingDir, onWorkingDir, backends, backend, onBackend, policyKey: selectedPolicy, onPolicyKey }: LaunchExtrasProps) {
  const { t, i18n } = useTranslation();
  const [appearanceOpen, setAppearanceOpen] = useState(false);
  const [optionsOpen, setOptionsOpen] = useState(false);
  const destinationGroup = groups.find((group) => group.id === destinationGroupId) ?? null;
  const policyOptions = POLICY_OPTIONS.map((option) => ({
    value: policyKey(option.mode, option.duration),
    label: option.duration ? formatLongDuration(parseDurationMs(option.duration), i18n.resolvedLanguage) : t(strings.terminalLauncher.timeoutNever),
  }));
  return (
    <div className="grid px-4 pt-2 pb-2">
      {/* Always present, even with no group. A control that appeared only
          sometimes would be one the operator never learns to look for. */}
      <GroupDestinationTrigger groups={groups} value={destinationGroupId} onChange={onDestination} onCreate={onCreateDestination} />

      {appearance && (
        <>
          <button type="button" data-testid="launcher-appearance-toggle" className={DISCLOSURE} aria-expanded={appearanceOpen} onClick={() => { setAppearanceOpen((value) => !value); }}>
            <span className="font-medium">{t(strings.launcher.appearance)}</span>
            <span className="flex min-w-0 items-center gap-1.5">
              <span className="h-2.5 w-2.5 shrink-0 rounded-full border border-wc-default" style={{ backgroundColor: destinationGroup?.color ?? appearance.headerColor }} aria-hidden />
              <span className="truncate">{t(strings.launcher.appearanceSummary, {
                color: destinationGroup ? t(strings.launcher.appearanceFromGroup) : t(strings.launcher.appearanceDefault),
                theme: appearance.themeId,
                size: appearance.fontSize,
              })}</span>
              {appearanceOpen ? <ChevronDown className="h-3.5 w-3.5 shrink-0" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5 shrink-0 rtl:rotate-180" aria-hidden />}
            </span>
          </button>
          {appearanceOpen && (
            <dl data-testid="launcher-appearance" className="grid grid-cols-3 gap-2 rounded-lg border border-wc-default bg-wc-surface-base/40 px-3 py-2 text-[11px]">
              <div><dt className="text-wc-text-faint">{t(strings.appearance.headerColorHeading)}</dt><dd className="truncate text-wc-text-secondary">{destinationGroup ? destinationGroup.name : appearance.headerColor}</dd></div>
              <div><dt className="text-wc-text-faint">{t(strings.appearance.terminalThemeHeading)}</dt><dd className="truncate text-wc-text-secondary">{appearance.themeId}</dd></div>
              <div><dt className="text-wc-text-faint">{t(strings.appearance.fontSizeHeading)}</dt><dd className="text-wc-text-secondary">{appearance.fontSize}px</dd></div>
            </dl>
          )}
        </>
      )}

      <button type="button" data-testid="launcher-options-toggle" className={DISCLOSURE} aria-expanded={optionsOpen} onClick={() => { setOptionsOpen((value) => !value); }}>
        <span className="font-medium">{t(strings.terminalLauncher.sessionOptions)}</span>
        {optionsOpen ? <ChevronDown className="h-3.5 w-3.5 shrink-0" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5 shrink-0 rtl:rotate-180" aria-hidden />}
      </button>
      {optionsOpen && (
        <div className="grid gap-3 rounded-xl border border-wc-default bg-wc-surface-base/50 p-3">
          <div className="grid gap-1.5">
            <label htmlFor="launcher-working-dir" className={LABEL}>{t(strings.terminalLauncher.workingDirectory)}</label>
            <Input id="launcher-working-dir" data-testid="launcher-working-dir" value={workingDir} onChange={(event) => { onWorkingDir(event.target.value); }} placeholder={t(strings.terminalLauncher.workingDirectoryPlaceholder)} />
          </div>
          {backends.length > 1 && (
            <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-x-3">
              <span className={LABEL}>{t(strings.terminalLauncher.backendLabel)}</span>
              <Select className={SELECT_TEXT} aria-label={t(strings.terminalLauncher.backendLabel)} data-testid="launcher-backend-select" value={backend}
                options={backends.map((option) => ({ value: option.id, label: option.label }))} onValueChange={(value) => { onBackend(value as BackendID); }} />
            </div>
          )}
          <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-x-3">
            <span className={LABEL}>{t(strings.terminalLauncher.timeoutLabel)}</span>
            <Select className={SELECT_TEXT} aria-label={t(strings.terminalLauncher.timeoutLabel)} data-testid="launcher-timeout-select" value={selectedPolicy} options={policyOptions} onValueChange={onPolicyKey} />
          </div>
          <p className="flex items-start gap-2 text-xs text-wc-text-faint">
            <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
            <span>{backend === "persistent" ? t(strings.terminalLauncher.persistentHint) : t(strings.terminalLauncher.advancedHint)}</span>
          </p>
        </div>
      )}
    </div>
  );
}
