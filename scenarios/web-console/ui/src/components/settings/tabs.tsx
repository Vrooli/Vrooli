/* eslint-disable react-refresh/only-export-components */
import { useMemo } from "react";
import {
  Blocks,
  Keyboard,
  PlugZap,
  LayoutTemplate,
  Send,
  Settings2,
  StickyNote,
  TerminalSquare,
  Volume2,
  Waves,
  CircleUserRound,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";

export const SETTINGS_TAB_IDS = [
  "account",
  "sessions",
  "workspace",
  "voice-input",
  "voice-output",
  "shortcuts",
  "new-pane-defaults",
  "templates",
  "snippets",
  "handoff-rules",
  "integrations",
] as const;

export type SettingsTabId = typeof SETTINGS_TAB_IDS[number];

export interface SettingsTabDefinition {
  id: SettingsTabId;
  label: string;
  shortLabel: string;
  description: string;
  icon: typeof TerminalSquare;
}

const TAB_ICONS: Record<SettingsTabId, typeof TerminalSquare> = {
  account: CircleUserRound,
  sessions: TerminalSquare,
  workspace: Settings2,
  "voice-input": Waves,
  "voice-output": Volume2,
  shortcuts: Keyboard,
  "new-pane-defaults": Blocks,
  templates: LayoutTemplate,
  snippets: StickyNote,
  "handoff-rules": Send,
  integrations: PlugZap,
};

const TAB_STRING_KEYS = {
  account: {
    label: "settings.tabs.account.label",
    shortLabel: "settings.tabs.account.shortLabel",
    description: "settings.tabs.account.description",
  },
  sessions: {
    label: strings.settings.tabs.sessions.label,
    shortLabel: strings.settings.tabs.sessions.shortLabel,
    description: strings.settings.tabs.sessions.description,
  },
  workspace: {
    label: strings.settings.tabs.workspace.label,
    shortLabel: strings.settings.tabs.workspace.shortLabel,
    description: strings.settings.tabs.workspace.description,
  },
  "voice-input": {
    label: strings.settings.tabs.voiceInput.label,
    shortLabel: strings.settings.tabs.voiceInput.shortLabel,
    description: strings.settings.tabs.voiceInput.description,
  },
  "voice-output": {
    label: strings.settings.tabs.voiceOutput.label,
    shortLabel: strings.settings.tabs.voiceOutput.shortLabel,
    description: strings.settings.tabs.voiceOutput.description,
  },
  shortcuts: {
    label: strings.settings.tabs.shortcuts.label,
    shortLabel: strings.settings.tabs.shortcuts.shortLabel,
    description: strings.settings.tabs.shortcuts.description,
  },
  "new-pane-defaults": {
    label: strings.settings.tabs.newPaneDefaults.label,
    shortLabel: strings.settings.tabs.newPaneDefaults.shortLabel,
    description: strings.settings.tabs.newPaneDefaults.description,
  },
  templates: {
    label: strings.settings.tabTemplates,
    shortLabel: strings.settings.tabTemplates,
    description: strings.groupTemplates.title,
  },
  snippets: {
    label: strings.settings.tabs.snippets.label,
    shortLabel: strings.settings.tabs.snippets.shortLabel,
    description: strings.settings.tabs.snippets.description,
  },
  "handoff-rules": {
    label: strings.settings.tabHandoffRules,
    shortLabel: strings.settings.tabHandoffRules,
    description: strings.handoffRules.footer,
  },
  integrations: {
    label: strings.settings.tabs.integrations.label,
    shortLabel: strings.settings.tabs.integrations.shortLabel,
    description: strings.settings.tabs.integrations.description,
  },
} as const;

export function useSettingsTabs(): SettingsTabDefinition[] {
  const { t } = useTranslation();
  return useMemo(
    () =>
      SETTINGS_TAB_IDS.map((id) => ({
        id,
        label: t(TAB_STRING_KEYS[id].label),
        shortLabel: t(TAB_STRING_KEYS[id].shortLabel),
        description: t(TAB_STRING_KEYS[id].description),
        icon: TAB_ICONS[id],
      })),
    [t],
  );
}

export const DEFAULT_SETTINGS_TAB: SettingsTabId = "workspace";
