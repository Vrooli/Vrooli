import { useState, useCallback } from "react";
import type { SettingsTab } from "../components/settings/Settings";
import type { TemplateWithSource } from "../lib/types/templates";
import { updateTemplate as updateTemplateAPI, updateDefaultTemplate as updateDefaultTemplateAPI } from "../data/templates";

export interface AppModalsState {
  showLabelManager: boolean;
  setShowLabelManager: (show: boolean) => void;
  showSettings: boolean;
  setShowSettings: (show: boolean) => void;
  settingsInitialTab: SettingsTab;
  showKeyboardShortcuts: boolean;
  setShowKeyboardShortcuts: (show: boolean) => void;
  showUsageStats: boolean;
  setShowUsageStats: (show: boolean) => void;
  settingsEditingTemplate: TemplateWithSource | null;
  setSettingsEditingTemplate: (template: TemplateWithSource | null) => void;
  settingsAllTemplates: TemplateWithSource[];
  setSettingsAllTemplates: (templates: TemplateWithSource[]) => void;
  handleOpenSettings: (tab?: SettingsTab) => void;
  handleOpenAgentSettings: () => void;
  handleShowKeyboardShortcuts: () => void;
  handleShowUsageStats: () => void;
  handleEditTemplateFromSettings: (template: TemplateWithSource, allTemplates: TemplateWithSource[]) => void;
  handleSaveTemplateFromSettings: (
    templateData: Omit<TemplateWithSource, "id" | "createdAt" | "updatedAt" | "isBuiltIn" | "source" | "hasDefault">,
    options?: { applyToDefault?: boolean }
  ) => Promise<void>;
  anyModalOpen: boolean;
}

export function useAppModals(): AppModalsState {
  const [showLabelManager, setShowLabelManager] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [settingsInitialTab, setSettingsInitialTab] = useState<SettingsTab>("general");
  const [showKeyboardShortcuts, setShowKeyboardShortcuts] = useState(false);
  const [showUsageStats, setShowUsageStats] = useState(false);
  const [settingsEditingTemplate, setSettingsEditingTemplate] = useState<TemplateWithSource | null>(null);
  const [settingsAllTemplates, setSettingsAllTemplates] = useState<TemplateWithSource[]>([]);

  const handleOpenSettings = useCallback((tab: SettingsTab = "general") => {
    setSettingsInitialTab(tab);
    setShowSettings(true);
  }, []);

  const handleOpenAgentSettings = useCallback(() => {
    handleOpenSettings("agent");
  }, [handleOpenSettings]);

  const handleShowKeyboardShortcuts = useCallback(() => {
    setShowKeyboardShortcuts(true);
  }, []);

  const handleShowUsageStats = useCallback(() => {
    setShowUsageStats(true);
  }, []);

  const handleEditTemplateFromSettings = useCallback((template: TemplateWithSource, allTemplates: TemplateWithSource[]) => {
    setSettingsEditingTemplate(template);
    setSettingsAllTemplates(allTemplates);
  }, []);

  const handleSaveTemplateFromSettings = useCallback(async (
    templateData: Omit<TemplateWithSource, "id" | "createdAt" | "updatedAt" | "isBuiltIn" | "source" | "hasDefault">,
    options?: { applyToDefault?: boolean }
  ) => {
    if (!settingsEditingTemplate) return;
    if (options?.applyToDefault) {
      await updateDefaultTemplateAPI(settingsEditingTemplate.id, templateData);
    } else {
      await updateTemplateAPI(settingsEditingTemplate.id, templateData);
    }
    setSettingsEditingTemplate(null);
  }, [settingsEditingTemplate]);

  const anyModalOpen = showLabelManager || showSettings || showKeyboardShortcuts || showUsageStats || !!settingsEditingTemplate;

  return {
    showLabelManager,
    setShowLabelManager,
    showSettings,
    setShowSettings,
    settingsInitialTab,
    showKeyboardShortcuts,
    setShowKeyboardShortcuts,
    showUsageStats,
    setShowUsageStats,
    settingsEditingTemplate,
    setSettingsEditingTemplate,
    settingsAllTemplates,
    setSettingsAllTemplates,
    handleOpenSettings,
    handleOpenAgentSettings,
    handleShowKeyboardShortcuts,
    handleShowUsageStats,
    handleEditTemplateFromSettings,
    handleSaveTemplateFromSettings,
    anyModalOpen,
  };
}
