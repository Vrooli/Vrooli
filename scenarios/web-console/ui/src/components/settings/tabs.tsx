/* eslint-disable react-refresh/only-export-components */
import { useMemo } from "react";
import {
  Bell,
  Blocks,
  Keyboard,
  PlugZap,
  LayoutTemplate,
  MonitorSmartphone,
  Send,
  Settings2,
  StickyNote,
  TerminalSquare,
  Volume2,
  Waves,
  CircleUserRound,
  CloudCog,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";

export const SETTINGS_TAB_IDS = [
  "account",
  "devices",
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
  "notifications",
  "deployments",
] as const;

export type SettingsTabId = typeof SETTINGS_TAB_IDS[number];

export interface SettingsTabDefinition {
  id: SettingsTabId;
  label: string;
  shortLabel: string;
  description: string;
  /** Extra words the settings search matches, beyond label and description. */
  keywords: string;
  icon: typeof TerminalSquare;
}

const TAB_ICONS: Record<SettingsTabId, typeof TerminalSquare> = {
  account: CircleUserRound,
  devices: MonitorSmartphone,
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
  notifications: Bell,
  deployments: CloudCog,
};

const TAB_STRING_KEYS = {
  account: {
    label: "settings.tabs.account.label",
    shortLabel: "settings.tabs.account.shortLabel",
    description: "settings.tabs.account.description",
  },
  devices: {
    label: strings.settings.tabs.devices.label,
    shortLabel: strings.settings.tabs.devices.shortLabel,
    description: strings.settings.tabs.devices.description,
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
  notifications: {
    label: strings.settings.tabs.notifications.label,
    shortLabel: strings.settings.tabs.notifications.shortLabel,
    description: strings.settings.tabs.notifications.description,
  },
  deployments: {
    label: "settings.tabs.deployments.label",
    shortLabel: "settings.tabs.deployments.shortLabel",
    description: "settings.tabs.deployments.description",
  },
} as const;

const TAB_KEYWORDS: Record<SettingsTabId, string> = {
  account: "billing credits subscription key provider identity login",
  devices: "machines computers screens mobile desktop remote access",
  sessions: "terminals panes close archive restore focus order",
  workspace: "theme appearance font size colors zoom layout sidebar grid",
  "voice-input": "microphone speech recognition dictation audio streaming",
  "voice-output": "tts speak read aloud voice audio backend",
  shortcuts: "keys presets commands launch profiles keyboard",
  "new-pane-defaults": "theme appearance font size colors terminal defaults new pane",
  templates: "group roles recipes saved starter",
  snippets: "text reusable saved insert composer handoff",
  "handoff-rules": "send share route capture rules",
  integrations: "services connect provider api configuration",
  notifications: "alerts events notify scenarios urgency",
  deployments: "staging release publish deploy operations",
};

export function useSettingsTabs(): SettingsTabDefinition[] {
  const { t } = useTranslation();
  return useMemo(
    () =>
      SETTINGS_TAB_IDS.map((id) => ({
        id,
        label: id === "deployments" ? "Deployments" : t(TAB_STRING_KEYS[id].label),
        shortLabel: id === "deployments" ? "Deployments" : t(TAB_STRING_KEYS[id].shortLabel),
        description: id === "deployments" ? "Review staging profile and deployment operation status." : t(TAB_STRING_KEYS[id].description),
        keywords: TAB_KEYWORDS[id],
        icon: TAB_ICONS[id],
      })),
    [t],
  );
}

export const DEFAULT_SETTINGS_TAB: SettingsTabId = "workspace";
