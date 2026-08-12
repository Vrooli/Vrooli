/* eslint-disable react-refresh/only-export-components */
import { useMemo } from "react";
import { Blocks, Keyboard, PlugZap, Settings2, TerminalSquare, Volume2, Waves, } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
export const SETTINGS_TAB_IDS = [
    "sessions",
    "workspace",
    "voice-input",
    "voice-output",
    "shortcuts",
    "new-pane-defaults",
    "integrations",
];
const TAB_ICONS = {
    sessions: TerminalSquare,
    workspace: Settings2,
    "voice-input": Waves,
    "voice-output": Volume2,
    shortcuts: Keyboard,
    "new-pane-defaults": Blocks,
    integrations: PlugZap,
};
const TAB_STRING_KEYS = {
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
    integrations: {
        label: strings.settings.tabs.integrations.label,
        shortLabel: strings.settings.tabs.integrations.shortLabel,
        description: strings.settings.tabs.integrations.description,
    },
};
export function useSettingsTabs() {
    const { t } = useTranslation();
    return useMemo(() => SETTINGS_TAB_IDS.map((id) => ({
        id,
        label: t(TAB_STRING_KEYS[id].label),
        shortLabel: t(TAB_STRING_KEYS[id].shortLabel),
        description: t(TAB_STRING_KEYS[id].description),
        icon: TAB_ICONS[id],
    })), [t]);
}
export const DEFAULT_SETTINGS_TAB = "workspace";
