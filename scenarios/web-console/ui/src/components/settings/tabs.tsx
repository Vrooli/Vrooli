/* eslint-disable react-refresh/only-export-components */
import {
  Blocks,
  Keyboard,
  PlugZap,
  Settings2,
  TerminalSquare,
  Volume2,
  Waves,
} from "lucide-react";

export const SETTINGS_TAB_IDS = [
  "sessions",
  "workspace",
  "voice-input",
  "voice-output",
  "shortcuts",
  "new-pane-defaults",
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

export const SETTINGS_TABS: SettingsTabDefinition[] = [
  {
    id: "sessions",
    label: "Sessions",
    shortLabel: "Sessions",
    description: "Manage open terminals, policies, order, and focus.",
    icon: TerminalSquare,
  },
  {
    id: "workspace",
    label: "Workspace",
    shortLabel: "Workspace",
    description: "Layout and workspace behavior across desktop and mobile.",
    icon: Settings2,
  },
  {
    id: "voice-input",
    label: "Voice Input",
    shortLabel: "Input",
    description: "Speech recognition, microphone access, and streaming behavior.",
    icon: Waves,
  },
  {
    id: "voice-output",
    label: "Voice Output",
    shortLabel: "Output",
    description: "TTS backend selection, diagnostics, and auto-speak settings.",
    icon: Volume2,
  },
  {
    id: "shortcuts",
    label: "Shortcut Profiles",
    shortLabel: "Shortcuts",
    description: "Saved command presets for new terminal launches.",
    icon: Keyboard,
  },
  {
    id: "new-pane-defaults",
    label: "New Pane Defaults",
    shortLabel: "Defaults",
    description: "Appearance defaults applied when new panes are created.",
    icon: Blocks,
  },
  {
    id: "integrations",
    label: "Integrations",
    shortLabel: "Integrations",
    description: "Connected services and provider configuration.",
    icon: PlugZap,
  },
];

export const DEFAULT_SETTINGS_TAB: SettingsTabId = "workspace";
