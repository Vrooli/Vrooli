import type { LucideIcon } from "lucide-react";

export type Theme = "dark" | "light";
export type ViewMode = "bubble" | "compact";
export type SettingsTab = "general" | "ai" | "agent" | "templates" | "suggestions" | "skills" | "data";

// Default model ROLE used when none is set. The backend resolves this OpenRouter
// policy role to a concrete model via resource-openrouter (greenfield: no
// hard-coded provider slug here).
export const DEFAULT_MODEL = "chat.default";

// Default view mode
export const DEFAULT_VIEW_MODE: ViewMode = "bubble";

export interface SettingsTabDef {
  value: SettingsTab;
  label: string;
  icon: LucideIcon;
}

// Get/set default model from localStorage
export function getDefaultModel(): string {
  if (typeof window !== "undefined") {
    return localStorage.getItem("defaultModel") || DEFAULT_MODEL;
  }
  return DEFAULT_MODEL;
}

export function setDefaultModel(modelId: string): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("defaultModel", modelId);
  }
}

// Get/set view mode from localStorage
export function getViewMode(): ViewMode {
  if (typeof window !== "undefined") {
    const stored = localStorage.getItem("viewMode");
    if (stored === "bubble" || stored === "compact") {
      return stored;
    }
  }
  return DEFAULT_VIEW_MODE;
}

export function setViewMode(mode: ViewMode): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("viewMode", mode);
  }
}
