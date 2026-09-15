import { useState, useCallback, useEffect } from "react";
import { useSuggestionsSettings } from "../../hooks/useSuggestionsSettings";
import { useModeHistory } from "../../hooks/useModeHistory";
import { useAgentSettings } from "../../hooks/useAgentSettings";
import {
  getAllTemplates,
  deleteTemplate as deleteTemplateFromAPI,
  resetTemplate as resetTemplateFromAPI,
} from "../../data/templates";
import {
  getAllSkills,
  deleteSkill as deleteSkillFromAPI,
  createSkill as createSkillFromAPI,
  updateSkill as updateSkillFromAPI,
  syncSkills as syncSkillsFromAPI,
} from "../../data/skills";
import type { TemplateWithSource, SkillWithSource, Skill } from "../../lib/types/templates";
import type { Theme, SettingsTab } from "./settingsTypes";
import { DEFAULT_MODEL } from "./settingsTypes";

export function useSettingsState(_open: boolean, activeTab: SettingsTab, onClose: () => void, onEditTemplate?: (template: TemplateWithSource, allTemplates: TemplateWithSource[]) => void) {
  const [theme, setTheme] = useState<Theme>(() => {
    if (typeof window !== "undefined") {
      const stored = localStorage.getItem("theme");
      if (stored === "light" || stored === "dark") return stored;
    }
    return "dark";
  });
  const [defaultModel, setDefaultModelState] = useState<string>(() => {
    if (typeof window !== "undefined") return localStorage.getItem("defaultModel") || DEFAULT_MODEL;
    return DEFAULT_MODEL;
  });
  // Suggestions
  const { visible: suggestionsVisible, setVisible: setSuggestionsVisible, mergeModel, setMergeModel, autoSuggest, autoSuggestLoading, autoSuggestError, updateAutoSuggest } = useSuggestionsSettings();
  const { history: modeHistory, clearHistory: clearModeHistory } = useModeHistory();

  // Agent
  const { settings: agentSettings, setSettings: setAgentSettings, resetSettings: resetAgentSettings } = useAgentSettings();

  // Templates
  const [templates, setTemplates] = useState<TemplateWithSource[]>([]);
  const [isLoadingTemplates, setIsLoadingTemplates] = useState(false);

  // Skills
  const [skills, setSkills] = useState<SkillWithSource[]>([]);
  const [isLoadingSkills, setIsLoadingSkills] = useState(false);
  const [isSyncingSkills, setIsSyncingSkills] = useState(false);
  const [editingSkill, setEditingSkill] = useState<SkillWithSource | null>(null);
  const [isCreatingSkill, setIsCreatingSkill] = useState(false);

  // Suggestions draft
  const [suggestionsDraft, setSuggestionsDraft] = useState(autoSuggest);
  const [isSavingSuggestions, setIsSavingSuggestions] = useState(false);
  const [suggestionsSaveError, setSuggestionsSaveError] = useState<string | null>(null);

  useEffect(() => {
    if (activeTab === "templates") {
      setIsLoadingTemplates(true);
      void getAllTemplates().then(setTemplates).finally(() => setIsLoadingTemplates(false));
    }
  }, [activeTab]);

  useEffect(() => {
    if (activeTab === "skills") {
      setIsLoadingSkills(true);
      void getAllSkills().then(setSkills).finally(() => setIsLoadingSkills(false));
    }
  }, [activeTab]);

  useEffect(() => { setSuggestionsDraft(autoSuggest); }, [autoSuggest]);

  // Theme effect
  useEffect(() => {
    const root = document.documentElement;
    if (theme === "light") root.classList.add("light-theme");
    else root.classList.remove("light-theme");
    localStorage.setItem("theme", theme);
  }, [theme]);

  const handleThemeChange = useCallback((newTheme: Theme) => setTheme(newTheme), []);

  const handleDefaultModelChange = useCallback((modelId: string) => {
    setDefaultModelState(modelId);
    if (typeof window !== "undefined") localStorage.setItem("defaultModel", modelId);
  }, []);

  const handleDeleteTemplate = useCallback(async (templateId: string) => {
    await deleteTemplateFromAPI(templateId);
    setTemplates(await getAllTemplates());
  }, []);

  const handleResetTemplate = useCallback(async (templateId: string) => {
    await resetTemplateFromAPI(templateId);
    setTemplates(await getAllTemplates());
  }, []);

  const handleEditTemplate = useCallback((template: TemplateWithSource) => {
    onEditTemplate?.(template, templates);
    onClose();
  }, [onEditTemplate, onClose, templates]);

  const handleDeleteSkill = useCallback(async (skillId: string) => {
    await deleteSkillFromAPI(skillId);
    setSkills(await getAllSkills());
  }, []);

  const handleSyncSkills = useCallback(async () => {
    setIsSyncingSkills(true);
    try { await syncSkillsFromAPI(); setSkills(await getAllSkills()); } finally { setIsSyncingSkills(false); }
  }, []);

  const handleEditSkill = useCallback((skill: SkillWithSource | null) => {
    if (skill === null) { setIsCreatingSkill(true); setEditingSkill(null); return; }
    setIsCreatingSkill(false);
    setEditingSkill(skill);
  }, []);

  const handleSaveSkill = useCallback(async (skillData: Omit<Skill, "id" | "createdAt" | "updatedAt">) => {
    if (isCreatingSkill) await createSkillFromAPI(skillData);
    else if (editingSkill) await updateSkillFromAPI(editingSkill.id, skillData);
    setSkills(await getAllSkills());
    setEditingSkill(null);
    setIsCreatingSkill(false);
  }, [isCreatingSkill, editingSkill]);

  const handleSaveSuggestions = useCallback(async () => {
    setIsSavingSuggestions(true);
    setSuggestionsSaveError(null);
    try { await updateAutoSuggest(suggestionsDraft); }
    catch (error) { setSuggestionsSaveError(error instanceof Error ? error.message : "Failed to save suggestions settings"); }
    finally { setIsSavingSuggestions(false); }
  }, [suggestionsDraft, updateAutoSuggest]);

  return {
    theme, defaultModel,
    suggestionsVisible, setSuggestionsVisible, mergeModel, setMergeModel,
    autoSuggestLoading, autoSuggestError,
    modeHistory, clearModeHistory,
    agentSettings, setAgentSettings, resetAgentSettings,
    templates, isLoadingTemplates,
    skills, setSkills, isLoadingSkills, isSyncingSkills,
    editingSkill, setEditingSkill, isCreatingSkill, setIsCreatingSkill,
    suggestionsDraft, setSuggestionsDraft, isSavingSuggestions, suggestionsSaveError,
    handleThemeChange, handleDefaultModelChange,
    handleDeleteTemplate, handleResetTemplate, handleEditTemplate,
    handleDeleteSkill, handleSyncSkills, handleEditSkill, handleSaveSkill,
    handleSaveSuggestions,
  };
}
